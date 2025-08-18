// Package exam site management
package exam_site

//annotation:exam-site-service
//author:{"name":"Zhong Peiqi","tel":"13068178042","email":"zpekii3156@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"

	// "github.com/tidwall/sjson"

	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
)

const (
	IP_ADDR_REGEXP     = `^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])(:(\d{1,5}))?$`
	ExamSitePrefix     = "examSite"
	ExamSiteSyncPrefix = "examSiteSync"
	SyncStatusKey      = "examSiteSync:SyncStatus"
	PULLING            = "Pulling"
	PULLED             = "Pulled"
	PUSHING            = "Pushing"
	PUSHED             = "Pushed"
)

var (
	z *zap.Logger

	syncLock   sync.Mutex
	syncStatus string

	pullChan chan int
	pushChan chan int

	centralServerUrl = "http://localhost:6610"
	sysUser          = "" // 登录账号/邮箱/手机号等
	accessToken      = ""
	sshUser          = "root"
	sshHost          = "localhost"
	sshPort          = 22
	dbAddr           = "postgres"
	dbPort           = 6900
	dbName           = "assessdb"
	dbUser           = "assessuser"
	dbPwd            = "postgres"
	maxRetry         = 3
)

type sysUserInfo struct {
	Name        string
	ID          int64
	Session     string
	AccessToken string
}

type (
	txCtxKeyStr  string
	dbConnKeyStr string
)

type examSiteInfo struct {
	ID          null.Int    `json:"id"`
	Name        string      `json:"name" validate:"required"`
	Address     string      `json:"address" validate:"required"`
	ServerHost  null.String `json:"server_host"`
	Admin       int64       `json:"admin" validate:"required"`
	AdminName   null.String `json:"admin_name"`
	RoomCount   null.Int    `json:"room_count"`
	AccessToken null.String `json:"access_token"`
}

type syncInfo struct {
	Path          string   `json:"path" validate:"required"`
	TableFileList []string `json:"tableFileList" validate:"required"`
}

const (
	txCtxKey  txCtxKeyStr  = "exam_site_tx_ctx_key"
	dbConnKey dbConnKeyStr = "exam_site_db_conn_key"
)

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("exam_site zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("message.Enroll called")
	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, &cmn.ServiceCtx{
		RedisClient: cmn.GetRedisConn(),
		Msg: &cmn.ReplyProto{},
	})

	examSiteSyncInit(ctx)

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSite,

		Path: "/exam-site",
		Name: "exam-site",

		Developer: developer,
		WhiteList: false,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSiteList,

		Path: "/exam-site/list",
		Name: "exam-site-list",

		Developer: developer,
		WhiteList: false,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSiteSync,

		Path: "/exam-site/sync",
		Name: "exam-site-sync",

		Developer: developer,
		WhiteList: false,

		DomainID: int64(cmn.CDomainAssessExamSite),

		DefaultDomain: int64(cmn.CDomainAssessExamSite),
	})
}

// login 登录中心服务器进行身份验证
func login(ctx context.Context) (info sysUserInfo) {

	q := cmn.GetCtxValue(ctx)

	if viper.IsSet("examSiteServerSync.sysUser") {
		sysUser = viper.GetString("examSiteServerSync.sysUser")
	}

	if viper.IsSet("examSiteServerSync.accessToken") {
		accessToken = viper.GetString("examSiteServerSync.accessToken")
	}

	info.Name = sysUser

	info.AccessToken = accessToken

	cli := fasthttp.Client{}

	reqBody := []byte(fmt.Sprintf(`{ "name": "%s", "cert": "%s" }`,
		info.Name,
		info.AccessToken,
	))

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(fmt.Sprintf("%s/api/login", centralServerUrl))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(reqBody)

	q.Err = cli.Do(req, resp)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["loginReqErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["loginReqErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	var msg cmn.ReplyProto

	q.Err = json.Unmarshal(resp.Body(), &msg)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["loginBodyUnmarshalErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["loginBodyUnmarshalErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	if msg.Status != 0 || (cmn.InDebugMode && q.Tag["loginRespStatusErr"] != nil) {

		if msg.Status == 0 {
			msg.Msg = q.Tag["loginRespStatusErr"].(error).Error()
		}

		q.Err = fmt.Errorf("%s", msg.Msg)
		z.Error(q.Err.Error())
		return
	}

	var data cmn.TUser
	q.Err = json.Unmarshal(msg.Data, &data)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["loginRespDataUnmarshalErr"] != nil){

		if q.Err == nil {
			q.Err = q.Tag["loginRespDataUnmarshalErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	info.ID = data.ID.Int64

	// 只获取 "qNearSessions" 的cookie值
	for _, c := range resp.Header.PeekAll("Set-Cookie") {
		cookieStr := string(c)
		if strings.HasPrefix(cookieStr, "qNearSessions=") {
			info.Session = strings.TrimPrefix(cookieStr, "qNearSessions=")
			break
		}
	}

	return
}

// Pull 从中心服务器拉取数据并同步, 中间如果发生任何错误都会进行重试
//
// ctx 中必须包含 QNearKey 上下文
//
// retryCount 重试次数, 若值小于或等于0, 则不进行重试
func Pull(ctx context.Context, retryCount int) {

	q := cmn.GetCtxValue(ctx)

	dbConn := cmn.GetPgxConn()

	defer func() {

		if q.Err == nil || retryCount <= 0 {
			return
		}

		n := 999
		if retryCount > 0 {
			retryCount--
			n = retryCount
		}

		z.Sugar().Warnf("Pull operation failed, remaining retry times: %d", n)

		// retry when occurred err
		Pull(ctx, retryCount)
	}()

	z.Info("try lock")
	syncLock.Lock()
	z.Info("got lock")
	defer func() {
		syncLock.Unlock()
		z.Info("release lock")
	}()

	syncStatus, q.Err = q.RedisClient.Get(ctx, SyncStatusKey).Result()
	if (q.Err != nil && !errors.Is(q.Err, redis.Nil)) || (cmn.InDebugMode && q.Tag["getSyncStatusErrInPull"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["getSyncStatusErrInPull"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	switch syncStatus {

	case PULLING:
		q.Err = fmt.Errorf("当前正在拉取数据中, 不允许重复拉取")
		z.Error(q.Err.Error())
		retryCount = 0
		return

	case PULLED:
		q.Err = fmt.Errorf("当前数据尚未推送, 请先进行推送")
		z.Error(q.Err.Error())
		retryCount = 0
		return

	case PUSHING:
		q.Err = fmt.Errorf("当前正在推送数据中, 不允许进行拉取")
		z.Error(q.Err.Error())
		retryCount = 0
		return
	}

	_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PULLING, 0).Result()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["setPullingStatusErrInPull"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["setPullingStatusErrInPull"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	defer func() {
		if q.Err != nil {
			_, err := q.RedisClient.Set(ctx, SyncStatusKey, PUSHED, 0).Result()
			if err != nil || (cmn.InDebugMode && q.Tag["setPushedStatusErrInPull"] != nil) {

				if err == nil {
					err = q.Tag["setPushedStatusErrInPull"].(error)
					q.Err = err
				}

				z.Error(err.Error())
			}

			return
		}

		_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PULLED, 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["setPulledStatusErrInPull"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["setPulledStatusErrInPull"].(error)
			}

			z.Error(q.Err.Error())
		}

	}()

	// 登录验证
	info := login(ctx)
	if q.Err != nil {
		return
	}

	// 发送同步通知
	cli := fasthttp.Client{}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetCookie("qNearSessions", info.Session)

	req.SetRequestURI(fmt.Sprintf("%s/api/exam-site/sync", centralServerUrl))
	req.Header.SetMethod("GET")

	q.Err = cli.DoTimeout(req, resp, 30*time.Minute)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	q.Err = json.Unmarshal(resp.Body(), &q.Msg)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullRespBodyUnmarshalErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["pullRespBodyUnmarshalErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	if q.Msg.Status != 0 || (cmn.InDebugMode && q.Tag["pullRespStatusErr"] != nil) {

		if q.Msg.Status == 0 {
			q.Msg.Msg = q.Tag["pullRespStatusErr"].(error).Error()
		}

		q.Err = fmt.Errorf("%s", q.Msg.Msg)
		z.Error(q.Err.Error())
		return
	}

	var i syncInfo
	q.Err = json.Unmarshal(q.Msg.Data, &i)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullRespDataUnmarshalErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["pullRespDataUnmarshalErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	k := fmt.Sprintf("%s:%d", ExamSiteSyncPrefix, info.ID)
	_, q.Err = q.RedisClient.Set(ctx, k, []byte(q.Msg.Data), 0).Result()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullSetSyncInfoSnapShotErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["pullSetSyncInfoSnapShotErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	// rsync 拉取考点数据

	source := fmt.Sprintf("%s@%s:%s",
		sshUser,
		sshHost,
		i.Path,
	)

	b := make([]byte, 16)
	rand.Read(b)

	dest := filepath.Join(os.Getenv("PWD"), fmt.Sprintf("data/tmp/exam-site-%d/%x",
		info.ID, b))

	q.Err = os.MkdirAll(dest, 0755)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullMkdirAllDestDirErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["pullMkdirAllDestDirErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}
	defer func() {
		err := os.RemoveAll(dest)
		if err != nil || (cmn.InDebugMode && q.Tag["pullRemoveAllDestDirErr"] != nil) {

			if err == nil {
				err = q.Tag["pullRemoveAllDestDirErr"].(error)
				q.Msg.Msg = err.Error()
			}

			z.Error(err.Error())
		}
	}()

	cmd := fmt.Sprintf(`rsync -avz -e "ssh -p %d" --delete %s/* %s`,
		sshPort,
		source, dest)

	var o []byte
	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["rsyncErrInPull"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["rsyncErrInPull"].(error)
			cmd = ""
			o = []byte("")
		}

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))

		z.Error(q.Err.Error())
		return
	}

	q.Err = generateImportScript(i.TableFileList, dest, "import_script.sql", true)
	if q.Err != nil {
		return
	}

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	_, q.Err = dbConn.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, dbUser))
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullCreateSchemaErr"] != nil){

		if q.Err == nil {
			q.Err = q.Tag["pullCreateSchemaErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	// pg_restore
	cmd = fmt.Sprintf(`PGPASSFILE=%s pg_restore --host "%s" --port "%d" --username "%s" -w --dbname "%s" --clean --if-exists --single-transaction --schema-only "%s"`,
		pgpassFullPath,
		dbAddr,
		dbPort,
		dbUser,
		dbName,
		filepath.Join(dest, "schema.sql"),
	)

	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pgRestoreErrInPull"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["pgRestoreErrInPull"].(error)
			cmd = ""
			o = []byte("")
		}

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
		z.Error(q.Err.Error())
		return
	}

	// 执行导入脚本
	cmd = fmt.Sprintf("PGPASSFILE=%s psql -h %s -p %d -U %s -d %s -f %s",
		pgpassFullPath,
		dbAddr,
		dbPort,
		dbUser,
		dbName,
		filepath.Join(dest, "import_script.sql"),
	)

	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["psqlImportScriptInPullErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["psqlImportScriptInPullErr"].(error)
			cmd = ""
			o = []byte("")
		}

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
		z.Error(q.Err.Error())
		return
	}

}

// Push 将数据推送到中心服务器, 中间如果发生任何错误都会进行重试
//
// ctx 必须包含 QNearKey 上下文
//
// retryCount 重试次数, 若值小于或等于0, 则不进行重试
func Push(ctx context.Context, retryCount int) {

	q := cmn.GetCtxValue(ctx)

	defer func() {

		if q.Err == nil || retryCount <= 0 {
			return
		}

		n := 999
		if retryCount > 0 {
			retryCount--
			n = retryCount
		}

		z.Sugar().Warnf("Push operation failed, remaining retry times: %d", n)

		// retry when occurred err
		Push(ctx, retryCount)
	}()

	z.Info("try lock")
	syncLock.Lock()
	z.Info("got lock")
	defer func() {
		syncLock.Unlock()
		z.Info("release lock")
	}()

	syncStatus, q.Err = q.RedisClient.Get(ctx, SyncStatusKey).Result()
	if (q.Err != nil && !errors.Is(q.Err, redis.Nil) ) || (cmn.InDebugMode && q.Tag["getSyncStatusErrInPush"] != nil) {

		if q.Err == nil || errors.Is(q.Err, redis.Nil){
			q.Err = q.Tag["getSyncStatusErrInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	switch syncStatus {

	case PULLING:
		q.Err = fmt.Errorf("当前正在拉取数据中, 不允许进行推送")
		z.Error(q.Err.Error())
		retryCount = 0
		return

	case PUSHING:
		q.Err = fmt.Errorf("当前正在推送数据中，不允许重复推送")
		z.Error(q.Err.Error())
		retryCount = 0
		return

	case PUSHED:
		q.Err = fmt.Errorf("当前不允许推送数据,请先进行拉取同步数据")
		z.Error(q.Err.Error())
		retryCount = 0
		return

	}

	_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PUSHING, 0).Result()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["SetPushingStatusErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["SetPushingStatusErrInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	defer func() {

		// 这里排除 redis.Nil 的错误是为了防止因后续从 redis 获取同步信息快照为空从而进入 PULLED 状态, 在 PULLED状态下无法进行拉取,
		// 然而如果 q.Err 为 redis.Nil 错误, 则说明没有正常执行 Pull 进行拉取数据并设置同步信息快照, 理应允许重新拉取, 
		// 此时设置成 PUSHED 状态就允许进行重新进行 Pull 拉取, 从而避免死循环的发生
		if q.Err != nil && !errors.Is(q.Err, redis.Nil) {
			_, err := q.RedisClient.Set(ctx, SyncStatusKey, PULLED, 0).Result()
			if err != nil || (cmn.InDebugMode && q.Tag["setPulledStatusErrInPush"] != nil) {

				if err == nil {
					err = q.Tag["setPulledStatusErrInPush"].(error)
					q.Err = err
				}

				z.Error(err.Error())
			}

			return
		}

		_, err := q.RedisClient.Set(ctx, SyncStatusKey, PUSHED, 0).Result()
		if err != nil || (cmn.InDebugMode && q.Tag["setPushedStatusErrInPush"] != nil) {

			if err == nil {
				err = q.Tag["setPushedStatusErrInPush"].(error)
				q.Err = err
			}

			z.Error(err.Error())
		}

	}()

	var sInfo syncInfo
	var s string

	// 登录验证
	uInfo := login(ctx)
	if q.Err != nil {
		return
	}

	k := fmt.Sprintf("%s:%d", ExamSiteSyncPrefix, uInfo.ID)
	s, q.Err = q.RedisClient.Get(ctx, k).Result()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["getSyncInfoErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["getSyncInfoErrInPush"].(error)
		}

		if errors.Is(q.Err, redis.Nil) {
			q.Err = fmt.Errorf("没有找到同步信息, 请先进行拉取操作, err: %w", q.Err)
			retryCount = 0
		}

		z.Error(q.Err.Error())
		return
	}

	q.Err = json.Unmarshal([]byte(s), &sInfo)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["unmarshalSyncInfoErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["unmarshalSyncInfoErrInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	b := make([]byte, 16)

	rand.Read(b)

	source := filepath.Join(os.Getenv("PWD"), fmt.Sprintf("data/tmp/exam-site-%d/%x",
		uInfo.ID, b))

	dest := fmt.Sprintf("%s@%s:%s",
		sshUser,
		sshHost,
		sInfo.Path,
	)

	q.Err = os.MkdirAll(source, 0755)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["mkdirAllSourceInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["mkdirAllSourceInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}
	defer func() {
		err := os.RemoveAll(source)
		if err != nil || (cmn.InDebugMode && q.Tag["removeAllSourceInPush"] != nil) {

			if err == nil {
				err = q.Tag["removeAllSourceInPush"].(error)
				q.Err = err
			}

			z.Error(err.Error())
		}
	}()

	sInfo.TableFileList, q.Err = generateExportScript(uInfo.ID, source, "export_script.sql", true)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	// 执行导出脚本
	cmd := fmt.Sprintf("PGPASSFILE=%s psql -h %s -p %d -U %s -d %s -f %s",
		pgpassFullPath,
		dbAddr,
		dbPort,
		dbUser,
		dbName,
		filepath.Join(source, "export_script.sql"),
	)

	var o []byte
	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["psqlExportScriptErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["psqlExportScriptErrInPush"].(error)
			cmd = ""
			o = []byte("")
		}

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
		z.Error(q.Err.Error())
		return

	}

	cmd = fmt.Sprintf(`rsync -avz -e "ssh -p %d" --delete %s/* %s`,
		sshPort,
		source, dest)

	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["rsyncErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["rsyncErrInPush"].(error)
			cmd = ""
			o = []byte("")
		}

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
		z.Error(q.Err.Error())
		return
	}

	// 发送反向同步通知

	cli := fasthttp.Client{}

	var reqBody []byte
	var reqData []byte

	reqData, q.Err = json.Marshal(&sInfo)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	reqBody, q.Err = json.Marshal(&cmn.ReqProto{
		Data: reqData,
	})
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(fmt.Sprintf("%s/api/exam-site/sync", centralServerUrl))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(reqBody)

	q.Err = cli.Do(req, resp)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["syncReqErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["syncReqErrInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	var msg cmn.ReplyProto

	q.Err = json.Unmarshal(resp.Body(), &msg)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["syncReqBodyUnmarshalErrInPush"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["syncReqBodyUnmarshalErrInPush"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	if msg.Status != 0 || (cmn.InDebugMode && q.Tag["syncReqStatusErrInPush"] != nil) {

		if msg.Status == 0 {
			msg.Msg = q.Tag["syncReqStatusErrInPush"].(error).Error()
		}

		q.Err = fmt.Errorf("%s", msg.Msg)
		z.Error(q.Err.Error())
		return
	}

	// 清理已同步的数据
	retryCount = 0
	err := q.RedisClient.Del(ctx, k).Err()
	if err != nil || (cmn.InDebugMode && q.Tag["delSyncInfoSnapShotErrInPush"] != nil) {

		if err == nil {
			err = q.Tag["delSyncInfoSnapShotErrInPush"].(error)
			q.Err = err
		}

		z.Error(err.Error())
	}

}

// GetMaxRetry 获取同步重试最大次数
func GetMaxRetry() int {

	if viper.IsSet("examSiteServerSync.maxRetry") {
		maxRetry = viper.GetInt("examSiteServerSync.maxRetry")
	}

	return maxRetry
}

// SendPullMsg 发送拉取同步消息
func SendPullMsg() {

	if pullChan == nil {
		return
	}

	pullChan <- 1
}

// SendPushMsg 发送推送同步消息
func SendPushMsg() {

	if pushChan == nil {
		return
	}

	pushChan <- 1
}

func examSite(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	// [ ] 在角色可以进行切换后取消注释
	// role := q.SysUser.Role.Int64

	// if cmn.CDomain(role) != cmn.CDomainAssessExamSiteAdmin {
	// 	q.Err = fmt.Errorf("当前登录的用户角色没有权限访问该API")
	// 	z.Error(q.Err.Error())
	// 	q.Msg.Msg = q.Err.Error()
	// 	q.Msg.Status = -1
	// 	q.Resp()
	// 	return
	// }

	dbConn := cmn.GetDbConn()

	var tx *sql.Tx

	tx, q.Err = dbConn.Begin()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["txBeginErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["txBeginErr"].(error)
		}

		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	defer func() {
		if q.Err != nil {
			err := tx.Rollback()
			if err != nil || (cmn.InDebugMode && q.Tag["rollbackErr"] != nil) {

				if err == nil {
					q.Err = q.Tag["rollbackErr"].(error)
					err = q.Err
				}

				z.Error(err.Error())
			}
			return
		}

		err := tx.Commit()
		if err != nil || (cmn.InDebugMode && q.Tag["commitErr"] != nil) {

			if err == nil {
				q.Err = q.Tag["commitErr"].(error)
				err = q.Err
			}

			z.Error(err.Error())
		}

	}()

	ctx = context.WithValue(ctx, dbConnKey, dbConn)

	ctx = context.WithValue(ctx, txCtxKey, tx)

	var bodyBuf []byte
	bodyBuf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["readBodyErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["readBodyErr"].(error)
		}

		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var req cmn.ReqProto

	q.Err = json.Unmarshal(bodyBuf, &req)
	if q.Err != nil {
		z.Warn(q.Err.Error())
	}

	switch q.R.Method {

	case "GET":

	case "POST":

		var info examSiteInfo

		q.Err = json.Unmarshal(req.Data, &info)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		// 必要参数不为空校验
		q.Err = cmn.Validate(&info)
		if q.Err != nil {
			break
		}

		officialName := fmt.Sprintf("%s考点", info.Name)

		var account, userToken string

		b1 := make([]byte, 4)

		b2 := make([]byte, 32)

		// 该Read从不返回错误，并且始终填充 b, 一旦发生错误就直接Panic， 所以这里就不需要接收err
		rand.Read(b1)

		rand.Read(b2)

		account = fmt.Sprintf("exam-site-%x", b1)

		userToken = fmt.Sprintf("%x", b2)

		// 注册考点服务器系统账号，用于给考点服务器与中心服务器进行http通信验证
		sqlStr := `INSERT INTO t_user (category, type, official_name, account, user_token, creator)
		VALUES ('sys^admin', '08', $1, $2, crypt($3,gen_salt('bf')), 1000)
		RETURNING 
			id`

		var stmt1 *sql.Stmt
		stmt1, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr1"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr1"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt1.Close()

		var sysUserID int64

		q.Err = stmt1.QueryRowContext(ctx, officialName, account, userToken).Scan(&sysUserID)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr1"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["sqlExecErr1"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		// 添加考点服务器信息
		sqlStr = `INSERT INTO t_exam_site (name, address, server_host, admin, sys_user, creator)
		VALUES ($1, $2, $3, $4, $5, $6)
		`

		var stmt2 *sql.Stmt
		stmt2, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr2"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr2"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt2.Close()

		_, q.Err = stmt2.ExecContext(ctx, info.Name, info.Address, info.ServerHost, info.Admin, sysUserID, userID)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr2"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["sqlExecErr2"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		info.AccessToken = null.StringFrom(userToken)

		q.Msg.Data, q.Err = json.Marshal(info)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshalErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["jsonMarshalErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

	case "PATCH":

	case "DELETE":

	default:
		q.Err = fmt.Errorf("不支持的HTTP方法: %s", q.R.Method)
		z.Error(q.Err.Error())
	}

	if q.Err != nil {
		q.RespErr()
		return
	}

	q.Resp()
}

func examSiteList(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	role := q.SysUser.Role.Int64

	// [ ] 在角色可以进行切换后取消注释
	// if cmn.CDomain(role) != cmn.CDomainAssessExamSiteAdmin || cmn.CDomain(role) != cmn.CDomainAssessAdmin {
	// 	q.Err = fmt.Errorf("当前登录的用户角色没有权限访问该API")
	// 	z.Error(q.Err.Error())
	// 	q.Msg.Msg = q.Err.Error()
	// 	q.Msg.Status = -1
	// 	q.Resp()
	// 	return
	// }

	dbConn := cmn.GetDbConn()

MethodSwitch:
	switch q.R.Method {

	case "GET":

		// 默认获取当前账号创建的考点
		keys := []string{
			"t_exam_site.creator=$1",
		}

		values := []interface{}{
			userID,
		}

		// 如果是管理员角色，则获取所有考点
		if cmn.CDomain(role) == cmn.CDomainAssessAdmin {
			keys = []string{
				"1=$1",
			}
			values = []interface{}{
				1,
			}
		}

		param := q.R.URL.Query().Get("q")

		var req cmn.ReqProto

		q.Err = json.Unmarshal([]byte(param), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		if req.Page < 0 {
			q.Err = fmt.Errorf("页码不能小于0")
			z.Error(q.Err.Error())
			break
		}

		if req.PageSize < 1 {
			q.Err = fmt.Errorf("每页条数不能小于1")
			z.Error(q.Err.Error())
			break
		}

		nameFilter := gjson.Get(param, "filter.name").Str

		if nameFilter != "" {
			i := len(keys) + 1
			keys = append(keys, fmt.Sprintf(`(t_exam_site.name ILIKE $%d OR t_exam_site.address ILIKE $%d OR t_exam_site.server_host ILIKE $%d)`, i, i, i))
			values = append(values, fmt.Sprintf("%%%s%%", nameFilter))
		}

		orderBy := "t_exam_site.id"

		orderByList := []string{}

		for _, o := range req.OrderBy {
			for k, v := range o {

				if v == "" {
					continue
				}

				v = strings.ToUpper(v)

				if v != "ASC" && v != "DESC" {
					q.Err = fmt.Errorf("不支持的排序方式: %s key: %s", v, k)
					z.Error(q.Err.Error())
					break MethodSwitch
				}

				orderByList = append(orderByList, fmt.Sprintf("%s %s", k, v))
			}
		}

		if len(orderByList) > 0 {
			orderBy = strings.Join(orderByList, ", ")
		}

		s := fmt.Sprintf(`SELECT 
			t_exam_site.id,
			t_exam_site.name,
			t_exam_site.address,
			t_exam_site.server_host AS serverHost,
			t_exam_site.admin,
			t_user.official_name AS adminName,
			COUNT(t_exam_room.id) AS roomCount
		FROM t_exam_site
			LEFT JOIN t_exam_room ON t_exam_room.exam_site = t_exam_site.id
			LEFT JOIN t_user ON t_user.id = t_exam_site.admin
		WHERE %s
		GROUP BY
			t_exam_site.id,
			t_user.id
		ORDER BY
			%s
			`, strings.Join(keys, " AND "), orderBy)

		var stmt *sql.Stmt

		stmt, q.Err = dbConn.Prepare(s)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr1"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr1"].(error)
			}

			z.Error(q.Err.Error())
			break

		}

		defer stmt.Close()

		var r sql.Result
		r, q.Err = stmt.ExecContext(ctx, values...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr1"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["sqlExecErr1"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		q.Msg.RowCount, q.Err = r.RowsAffected()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["rowsAffectedErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["rowsAffectedErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		s = fmt.Sprintf(`%s
		LIMIT %d OFFSET %d`, s, req.PageSize, req.Page*req.PageSize)

		stmt, q.Err = dbConn.Prepare(s)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr2"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr2"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt.Close()

		var rows *sql.Rows
		rows, q.Err = stmt.QueryContext(ctx, values...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["queryErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["queryErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer rows.Close()

		result := []examSiteInfo{}

		for rows.Next() {
			var item examSiteInfo
			q.Err = rows.Scan(
				&item.ID,
				&item.Name,
				&item.Address,
				&item.ServerHost,
				&item.Admin,
				&item.AdminName,
				&item.RoomCount,
			)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["scanErr"] != nil) {

				if q.Err == nil {
					q.Err = q.Tag["scanErr"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			result = append(result, item)
		}

		q.Msg.Data, q.Err = json.Marshal(result)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshal"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["jsonMarshal"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

	default:
		q.Err = fmt.Errorf("不支持的HTTP方法: %s", q.R.Method)
		z.Error(q.Err.Error())
	}

	if q.Err != nil {
		q.RespErr()
		return
	}

	q.Resp()

}

func examSiteSyncInit(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	dbConn := cmn.GetPgxConn()

	config := dbConn.Config().ConnConfig

	dbAddr = config.Host

	dbPort = int(config.Port)

	dbName = config.Database

	dbUser = config.User

	dbPwd = config.Password

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	// 创建.pgpass文件
	var f *os.File
	f, q.Err = os.Create(pgpassFullPath)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["createPgpassErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["createPgpassErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	defer func() {
		err := f.Close()
		if err != nil || (cmn.InDebugMode && q.Tag["closePgpassErr"] != nil) {

			q.Err = err

			if q.Err == nil {
				q.Err = q.Tag["closePgpassErr"].(error)
			}

			z.Error(q.Err.Error())
			return
		}
	}()

	_, q.Err = f.WriteString(fmt.Sprintf("%s:%d:%s:%s:%s",
		dbAddr,
		dbPort,
		dbName,
		dbUser,
		dbPwd,
	))
	if q.Err != nil || (cmn.InDebugMode && q.Tag["writePgpassErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["writePgpassErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	q.Err = f.Chmod(0600)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["chmodPgpassErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["chmodPgpassErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	centralServerUrl = viper.GetString("examSiteServerSync.centralServerUrl")

	if centralServerUrl == "" {
		return
	}

	// 如果中心服务器地址不为空，则当前以考点服务器运行和同步数据

	maxRetry = GetMaxRetry()

	var v string
	v, q.Err = q.RedisClient.Get(ctx, SyncStatusKey).Result()
	if (q.Err != nil && !errors.Is(q.Err, redis.Nil)) || (cmn.InDebugMode && q.Tag["getSyncStatusErr"] != nil) {

		if q.Err == nil || errors.Is(q.Err, redis.Nil){
			q.Err = q.Tag["getSyncStatusErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	if v == "" || errors.Is(q.Err, redis.Nil) {
		_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, "", 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["setSyncStatusErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["setSyncStatusErr"].(error)
			}

			z.Error(q.Err.Error())
			return
		}
	}

	if q.Tag == nil {
		q.Tag = make(map[string]interface{})
	}

	switch v {

	case PULLING:

		// 如果上一次没有完成拉取, 则重置为未拉取的状态
		_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PUSHED, 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["setPushedStatusErr"] != nil) {

			if q.Err == nil {
				q.Err =q.Tag["setPushedStatusErr"].(error)
			}

			z.Error(q.Err.Error())
			return
		}

	case PUSHING:

		// 如果上一次没有完成推送, 则重置为未推送的状态, 然后重新推送
		_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PULLED, 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["setPulledStatusErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["setPulledStatusErr"].(error)
			}

			z.Error(q.Err.Error())
			return
		}

		Push(ctx, maxRetry)
		return
	}

	Pull(ctx, maxRetry)
	if q.Err != nil {
		z.Error("初始化拉取同步数据失败")
	}

	pullChan = make(chan int)
	pushChan = make(chan int)

	pullChanOK := true

	pushChanOK := true

	go func() {
		for {

			select {

			case _, ok := <-pullChan:

				pullChanOK = ok

				if !ok {
					pullChan = nil
					break
				}

				Pull(ctx, maxRetry)

				if cmn.InDebugMode && q.Tag["pullDone"] != nil {
					q.Tag["pullDone"].(chan int) <- 1
					continue
				}

			case _, ok := <-pushChan:

				pushChanOK = ok

				if !ok {
					pushChan = nil
					break
				}

				Push(ctx, maxRetry)

				if cmn.InDebugMode && q.Tag["pushDone"] != nil {
					q.Tag["pushDone"].(chan int) <- 1
					continue
				}

			}

			if !pullChanOK && !pushChanOK {

				if cmn.InDebugMode && q.Tag["closeChan"] != nil {
					q.Tag["closeChan"].(chan int) <- 1
				}

				return
			} 

			if cmn.InDebugMode && q.Tag["endMsgListen"] != nil {
				q.Tag["endMsgListen"].(chan int) <- 1
				return
			}
			
		}
	}()

}

func examSiteSync(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	// 考点服务器系统账号ID
	userID := q.SysUser.ID.Int64

	if viper.IsSet("examSiteServerSync.centralServerSSH.user") {
		sshUser = viper.GetString("examSiteServerSync.centralServerSSH.user")
	}

	if viper.IsSet("examSiteServerSync.centralServerSSH.host") {
		sshHost = viper.GetString("examSiteServerSync.centralServerSSH.host")
	}

	if viper.IsSet("examSiteServerSync.centralServerSSH.port") {
		sshPort = viper.GetInt("examSiteServerSync.centralServerSSH.port")
	}

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	var bodyBuf []byte
	bodyBuf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["readBodyErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["readBodyErr"].(error)
		}

		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var req cmn.ReqProto

	q.Err = json.Unmarshal(bodyBuf, &req)
	if q.Err != nil {
		z.Warn(q.Err.Error())
	}

	syncInfoSnapshotKey := fmt.Sprintf("%s:%d", ExamSiteSyncPrefix, userID)

MethodSwitch:
	switch q.R.Method {

	case "GET":

		// 返回考点数据

		var s string
		var info syncInfo

		s, q.Err = q.RedisClient.Get(ctx, syncInfoSnapshotKey).Result()
		if (q.Err != nil && !errors.Is(q.Err, redis.Nil)) || (cmn.InDebugMode && q.Tag["getSyncInfoSnapshotErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["getSyncInfoSnapshotErr"].(error)
			}

			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if s != "" {

			// 如果同步数据信息快照存在，则直接返回

			// 验证数据有效性
			q.Err = json.Unmarshal([]byte(s), &info)
			if q.Err != nil {
				z.Error(q.Err.Error())
				break MethodSwitch
			}

			for _, f := range info.TableFileList {

				_, q.Err = os.Stat(filepath.Join(info.Path, f))
				if q.Err != nil {
					z.Error(q.Err.Error())
					break
				}

			}

			if q.Err == nil {
				q.Msg.Data = []byte(s)
				break MethodSwitch
			}

			q.Err = fmt.Errorf("过往已准备的数据已失效, Path: %s, TabelFileList: %s", info.Path, strings.Join(info.TableFileList, ", "))

			z.Warn(q.Err.Error())
		}

		// 准备考点数据

		b := make([]byte, 16)

		rand.Read(b)

		folderFullPath := filepath.Join(os.Getenv("PWD"), fmt.Sprintf("/data/tmp/exam-site-%d/%x", userID, b))

		q.Err = os.MkdirAll(folderFullPath, 0755)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["mkdirAllErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["mkdirAllErr"].(error)
			}

			z.Error(q.Err.Error())
			break MethodSwitch
		}

		defer func() {
			if q.Err == nil {
				return
			}

			// 当发生错误时，清理临时目录
			err := os.RemoveAll(folderFullPath)
			if err != nil || (cmn.InDebugMode && q.Tag["removeTmpDirErr"] != nil) {

				q.Err = err

				if q.Err == nil {
					q.Err = q.Tag["removeTmpDirErr"].(error)
				}

				z.Error(q.Err.Error())
				return
			}
		}()

		if cmn.InDebugMode && q.Tag["removeTmpDirErr"] != nil {
			q.Err = q.Tag["removeTmpDirErr"].(error)
			return
		}

		// 导出数据库模式结构
		cmd := fmt.Sprintf(`PGPASSFILE=%s pg_dump --file "%s" --host "%s" --port "%d" --username "%s" -w --format=c --large-objects --encoding "UTF8" --schema-only --clean --if-exists --verbose --schema "%s" "%s"`,
			pgpassFullPath,
			filepath.Join(folderFullPath, "schema.sql"),
			dbAddr,
			dbPort,
			dbUser,
			dbUser,
			dbName,
		)

		var o []byte
		o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["pgDumpErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["pgDumpErr"].(error)
				cmd = ""
				o = []byte("")
			}

			q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
			z.Error(q.Err.Error())
			break MethodSwitch
		}

		var tableFileList []string
		fileName := "export_script.sql"

		if cmn.InDebugMode && q.Tag["createExportScriptFileErr"] != nil {
			fileName = q.Tag["createExportScriptFileErr"].(error).Error()
		}

		if cmn.InDebugMode && q.Tag["writeExportScriptFileErr"] != nil {
			fileName = q.Tag["writeExportScriptFileErr"].(error).Error()
		}

		tableFileList, q.Err = generateExportScript(userID, folderFullPath, fileName, false)
		if q.Err != nil {
			break MethodSwitch
		}

		cmd = fmt.Sprintf("PGPASSFILE=%s psql -h %s -p %d -U %s -d %s -f %s",
			pgpassFullPath,
			dbAddr,
			dbPort,
			dbUser,
			dbName,
			filepath.Join(folderFullPath, fileName),
		)
		o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["psqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["psqlErr"].(error)
				cmd = ""
				o = []byte("")
			}

			q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
			z.Error(q.Err.Error())
			break MethodSwitch
		}

		info = syncInfo{
			Path:          folderFullPath,
			TableFileList: tableFileList,
		}

		var data []byte

		data, q.Err = json.Marshal(info)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshalErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["jsonMarshalErr"].(error)
			}

			z.Error(q.Err.Error())
			break MethodSwitch
		}

		// 保存同步数据信息快照
		_, q.Err = q.RedisClient.Set(ctx, syncInfoSnapshotKey, data, 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["saveSyncInfoSnapshotErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["saveSyncInfoSnapshotErr"].(error)
			}

			z.Error(q.Err.Error())
			break MethodSwitch
		}

		q.Msg.Data = data
	
	case "POST":

		// 同步考点数据

		var sInfo syncInfo

		q.Err = json.Unmarshal(req.Data, &sInfo)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break MethodSwitch
		}

		q.Err = cmn.Validate(&sInfo)
		if q.Err != nil {
			break MethodSwitch
		}

		q.Err = generateImportScript(sInfo.TableFileList, sInfo.Path, "import_script.sql", false)
		if q.Err != nil {
			break MethodSwitch
		}

		cmd := fmt.Sprintf(`PGPASSFILE=%s psql -h %s -p %d -U %s -d %s -f %s`,
			pgpassFullPath,
			dbAddr,
			dbPort,
			dbUser,
			dbName,
			filepath.Join(sInfo.Path, "import_script.sql"),
		)

		var o []byte
		o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["psqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["psqlErr"].(error)
				cmd = ""
				o = []byte("")
			}

			q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
			z.Error(q.Err.Error())
			break MethodSwitch
		}

		// 清理已同步的数据
		_, err := q.RedisClient.Del(ctx, syncInfoSnapshotKey).Result()
		if err != nil || (cmn.InDebugMode && q.Tag["deleteKeyErr"] != nil){
			
			if err == nil {
				err = q.Tag["deleteKeyErr"].(error)
				q.Msg.Msg = err.Error()
			}
			
			z.Error(err.Error())
		}

		err = os.RemoveAll(sInfo.Path)
		if err != nil || (cmn.InDebugMode && q.Tag["removeAllErr"] != nil) {

			if err == nil {
				err = q.Tag["removeAllErr"].(error)
				q.Msg.Msg = err.Error()
			}

			z.Error(err.Error())
		}

	default:
		q.Err = fmt.Errorf("不支持的HTTP方法: %s", q.R.Method)
		z.Error(q.Err.Error())
	}

	if q.Err != nil {
		q.RespErr()
		return
	}

	q.Resp()

}
