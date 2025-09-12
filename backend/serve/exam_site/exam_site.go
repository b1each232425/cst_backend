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
	"regexp"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"database/sql"

	"github.com/jmoiron/sqlx/types"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"

	// "github.com/tidwall/sjson"

	"go.uber.org/zap"

	"w2w.io/cmn"
	"w2w.io/exam_service"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
	"w2w.io/serve/mark"
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
	QNearSessionsKey   = "qNearSessions"
)

var (
	z *zap.Logger

	syncLock   sync.Mutex
	syncStatus string

	pullChan chan int
	pushChan chan int

	centralServerUrl = ""
	sysUser          = "" // 登录账号/邮箱/手机号等唯一身份标识
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
	uploadDir        = "./uploads" // tusd upload dir
)

type sysUserInfo struct {
	Name      string
	ID        int64
	Session   string
	UserToken string
}

type (
	txCtxKeyStr  string
	dbConnKeyStr string
)

type examSiteInfo struct {
	ID          null.Int    `json:"id"`
	Name        string      `json:"name" validate:"required"`
	Address     string      `json:"address" validate:"required"`
	ServerHost  null.String `json:"serverHost"`
	Admin       int64       `json:"admin" validate:"required"`
	AdminName   null.String `json:"adminName"`
	RoomCount   null.Int    `json:"roomCount"`
	AccessToken null.String `json:"accessToken"`
}

type examRoomInfo struct {
	ID         null.Int       `json:"id"`
	ExamSiteID int            `json:"examSiteID" validate:"required"`
	Name       string         `json:"name" validate:"required"`
	Capacity   int            `json:"capacity" validate:"required"`
	Available  null.Bool      `json:"available"`
	RecentExam types.JSONText `json:"recentExam"`
}

type syncInfo struct {
	RecentExamID    int64    `json:"recentExamID"`                      // 最近一场考试的ID
	Path            string   `json:"path" validate:"required"`          // 数据存放路径
	TableFileList   []string `json:"tableFileList" validate:"required"` // 表格数据文件列表
	UploadFilesPath string   `json:"uploadFilesPath"`                   // 上传的文件存放路径
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
		Msg:         &cmn.ReplyProto{},
	})

	examSiteSyncInit(ctx)

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSite,

		Path: "/exam-site",
		Name: "exam-site",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "考点管理.获取考点信息",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "考点管理.创建考点",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "考点管理.编辑考点",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
			{
				Name:         "考点管理.删除考点",
				AccessAction: auth_mgt.CAPIAccessActionDelete,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSiteList,

		Path: "/exam-site/list",
		Name: "exam-site-list",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "考点管理.获取考点列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examRoom,

		Path: "/exam-room",
		Name: "exam-room",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "考点管理.创建考场",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examRoomList,

		Path: "/exam-room/list",
		Name: "exam-room-list",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "考点管理.获取考场列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainAssessExamSite),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainAssessExamSiteAdmin),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSiteSync,

		Path: "/exam-site/sync",
		Name: "exam-site-sync",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "考点管理.同步拉取",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
			{
				Name:         "考点管理.同步推送",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

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

	info.ID, info.UserToken, q.Err = parseAccessToken(accessToken)

	cli := fasthttp.Client{}

	reqBody := []byte(fmt.Sprintf(`{ "name": "%d", "cert": "%s" }`,
		info.ID,
		info.UserToken,
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
	if q.Err != nil || (cmn.InDebugMode && q.Tag["loginRespDataUnmarshalErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["loginRespDataUnmarshalErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	info.ID = data.ID.Int64

	// 只获取 "qNearSessions" 的cookie值
	re := regexp.MustCompile(fmt.Sprintf(`%s=([^;]+)`, QNearSessionsKey))
	cookies := resp.Header.Peek("Set-Cookie")
	matches := re.FindAllSubmatch(cookies, -1)
	for _, m := range matches {
		if len(m) > 1 {
			info.Session = string(m[1])
		}
	}

	return
}

// Pull 从中心服务器拉取数据并同步, 中间如果发生任何错误都会进行重试
//
// ctx 中必须包含 QNearKey 上下文
//
// retryCount 重试次数, 若值小于或等于0, 则不进行重试
func Pull(ctx context.Context, retryCount int, forceRenew bool) {

	q := cmn.GetCtxValue(ctx)

	dbConn := cmn.GetDbConn()

	defer func() {

		if (q.Err == nil || retryCount <= 0) {
			return
		}

		n := 999
		if retryCount > 0 {
			retryCount--
			n = retryCount
		}

		z.Sugar().Warnf("Pull operation failed, remaining retry times: %d", n)

		// retry when occurred err
		Pull(ctx, retryCount, forceRenew)
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
			return
		}

		// 开启考试计时器
		q.Err = exam_service.InitializeExamTimers(ctx)

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

	req.Header.SetCookie(QNearSessionsKey, info.Session)

	req.SetRequestURI(fmt.Sprintf("%s/api/exam-site/sync?forceRenew=%v", centralServerUrl, forceRenew))
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

	var sInfo syncInfo
	q.Err = json.Unmarshal(q.Msg.Data, &sInfo)
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
		sInfo.Path,
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

	cmd := fmt.Sprintf(`rsync -avz --mkpath -e "ssh -p %d" --delete %s/* %s`,
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

	q.Err = generateImportScript(sInfo.TableFileList, dest, "import_script.sql", true)
	if q.Err != nil {
		return
	}

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	_, q.Err = dbConn.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, dbUser))
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullCreateSchemaErr"] != nil) {

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
	cmd = fmt.Sprintf("PGPASSFILE=%s psql -v ON_ERROR_STOP=1 -h %s -p %d -U %s -d %s -f %s",
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

		// 如果执行导入脚本失败, 那么很大可能是中心服务器上缓存数据有问题, 那么就需要强制下次重新拉取
		forceRenew = true

		q.Err = fmt.Errorf("COMMAND: %s\t ERR: %w\t DETAIL: %s", cmd, q.Err, string(o))
		z.Error(q.Err.Error())
		return
	}

	z.Info(string(o))

	// 同步附件

	var fileList []string
	var rows *sql.Rows
	rows, q.Err = dbConn.Query(`SELECT digest FROM t_file`)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	
	defer rows.Close()

	for rows.Next() {

		var digest string
		q.Err = rows.Scan(&digest)
		if q.Err != nil {
			z.Error(q.Err.Error())
			continue
		}

		fileList = append(fileList, digest)
		fileList = append(fileList, digest+".info")

	}

	fileListPath := filepath.Join(dest, "file_list.txt")
	q.Err = os.WriteFile(fileListPath, []byte(strings.Join(fileList, "\n")), 0644)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["pullWriteFileListErr"] != nil) {
		if q.Err == nil {
			q.Err = q.Tag["pullWriteFileListErr"].(error)
		}

		z.Error(q.Err.Error())
		return
	}

	source = fmt.Sprintf("%s@%s:%s",
		sshUser,
		sshHost,
		sInfo.UploadFilesPath,
	)

	cmd = fmt.Sprintf(`rsync -avz --mkpath -e "ssh -p %d" --files-from=%s %s %s/`,
		sshPort,
		fileListPath,
		source,
		uploadDir,
	)

	o, q.Err = exec.CommandContext(ctx, "bash", "-c", cmd).CombinedOutput()
	if q.Err != nil || (cmn.InDebugMode && q.Tag["rsyncUploadFilesErrInPull"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["rsyncUploadFilesErrInPull"].(error)
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
	if (q.Err != nil && !errors.Is(q.Err, redis.Nil)) || (cmn.InDebugMode && q.Tag["getSyncStatusErrInPush"] != nil) {

		if q.Err == nil || errors.Is(q.Err, redis.Nil) {
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

	sInfo.RecentExamID, sInfo.TableFileList, q.Err = generateExportScript(uInfo.ID, source, "export_script.sql", true)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	// 执行导出脚本
	cmd := fmt.Sprintf("PGPASSFILE=%s psql -v ON_ERROR_STOP=1 -h %s -p %d -U %s -d %s -f %s",
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

	z.Info(string(o))

	// 将数据同步回中心服务器
	cmd = fmt.Sprintf(`rsync -avz --mkpath -e "ssh -p %d" --delete %s/* %s`,
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
	req.Header.SetCookie(QNearSessionsKey, uInfo.Session)
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

// getApiPermissions 获取当前用户在使用指定接口时是否可读/可写
func getApiPermissions(ctx context.Context, apiPath string) (readable, creatable, editable, deletable bool) {

	q := cmn.GetCtxValue(ctx)

	readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionRead)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiReadableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiReadableErr"].(error)
		}

		return
	}

	creatable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionCreate)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiCreatableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiCreatableErr"].(error)
		}

		return
	}

	editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionUpdate)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiEditableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiEditableErr"].(error)
		}

		return
	}

	deletable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionDelete)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiDeletableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiDeletableErr"].(error)
		}

		return
	}

	return
}

// createSysUser 创建考点服务器系统账号
func createSysUser(ctx context.Context, tx *sql.Tx, siteID int64) (sysUserID int64, accessToken string, err error) {

	q := cmn.GetCtxValue(ctx)

	officialName := fmt.Sprintf("考点%d", siteID)

	var account, userToken string

	b1 := make([]byte, 4)

	b2 := make([]byte, 32)

	// 该Read从不返回错误，并且始终填充 b, 一旦发生错误就直接Panic， 所以这里就不需要接收err
	rand.Read(b1)

	rand.Read(b2)

	account = fmt.Sprintf("exam-site-%x", b1)

	userToken = fmt.Sprintf("%x", b2)

	// 注册考点服务器系统账号，用于给考点服务器与中心服务器进行http通信验证
	sqlStr := fmt.Sprintf(`INSERT INTO t_user (category, type, official_name, account, user_token, creator, role)
	VALUES ('sys^admin', '08', $1, $2, crypt($3,gen_salt('bf')), 1000, %d)
	RETURNING 
		id`, cmn.CDomainAssessExamSite)

	var stmt1 *sql.Stmt
	stmt1, err = tx.Prepare(sqlStr)
	if err != nil || (cmn.InDebugMode && q.Tag["prepareCreateSysUserErr"] != nil) {

		if err == nil {
			err = q.Tag["prepareCreateSysUserErr"].(error)
		}

		z.Error(err.Error())
		return
	}

	defer stmt1.Close()

	err = stmt1.QueryRowContext(ctx, officialName, account, userToken).Scan(&sysUserID)
	if err != nil || (cmn.InDebugMode && q.Tag["sqlExecCreateSysUserErr"] != nil) {

		if err == nil {
			err = q.Tag["sqlExecCreateSysUserErr"].(error)
		}

		z.Error(err.Error())
		return
	}

	accessToken = generateAccessToken(sysUserID, userToken)

	return
}

/* 考点基础业务 */
// oooooooooo.
// `888'   `Y8b
//  888     888  .oooo.    .oooo.o  .ooooo.
//  888oooo888' `P  )88b  d88(  "8 d88' `88b
//  888    `88b  .oP"888  `"Y88b.  888ooo888
//  888    .88P d8(  888  o.  )88b 888    .o
// o888bood8P'  `Y888""8o 8""888P' `Y8bod8P'
//

// examSite 处理考点相关请求
func examSite(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	readable, creatable, editable, deletable := getApiPermissions(ctx, q.Ep.Path)

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

MethodSwitch:
	switch q.R.Method {

	case "GET":

		if !readable {
			q.Err = fmt.Errorf("当前用户没有权限获取该数据")
			z.Error(q.Err.Error())
			break
		}

	case "POST":

		if !creatable {
			q.Err = fmt.Errorf("当前用户没有权限创建该数据")
			z.Error(q.Err.Error())
			break
		}

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

		var sysUserID int64
		var accessToken string
		sysUserID, accessToken, q.Err = createSysUser(ctx, tx, info.ID.Int64)
		if q.Err != nil {
			break
		}

		// 添加考点信息
		sqlStr := `INSERT INTO t_exam_site (name, address, server_host, admin, sys_user, creator, domain_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		var stmt2 *sql.Stmt
		stmt2, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt2.Close()

		_, q.Err = stmt2.ExecContext(ctx, info.Name, info.Address, info.ServerHost, info.Admin, sysUserID, userID, authority.Domain.ID.Int64)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["sqlExecErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		info.AccessToken = null.StringFrom(accessToken)

		q.Msg.Data, q.Err = json.Marshal(info)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshalErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["jsonMarshalErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

	case "PATCH":

		if !editable {
			q.Err = fmt.Errorf("当前用户没有权限修改该数据")
			z.Error(q.Err.Error())
			break
		}

		var editInfo struct {
			examSiteInfo
			ResetAccessToken bool `json:"resetAccessToken"`
		}

		q.Err = json.Unmarshal(req.Data, &editInfo)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		if editInfo.ID.Int64 <= 0 {
			q.Err = fmt.Errorf("考点ID必须大于0, 当前传入: %d", editInfo.ID.Int64)
			z.Error(q.Err.Error())
			break
		}

		values := []interface{}{
			userID,
			editInfo.ID.Int64,
		}

		sets := []string{
			"updated_by=$1",
		}

		filters := []string{
			"id=$2",
		}

		domains := []string{
			"creator=$1",
		}
		for _, d := range authority.AccessibleDomains {
			domains = append(domains, fmt.Sprintf("domain_id=%d", d))
		}

		filters = append(filters, fmt.Sprintf("(%s)", strings.Join(domains, " OR ")))

		fieldMap := map[string]interface{}{
			"name":        editInfo.Name,
			"address":     editInfo.Address,
			"server_host": editInfo.ServerHost.String,
			"admin":       editInfo.Admin,
		}

		for field, value := range fieldMap {
			if v, ok := value.(string); ok && v == "" {
				continue
			}

			if v, ok := value.(int64); ok && v <= 0 {
				continue
			}

			sets = append(sets, fmt.Sprintf("%s=$%d", field, len(values)+1))
			values = append(values, value)
		}

		sqlStr := fmt.Sprintf(`UPDATE t_exam_site SET %s WHERE %s`, strings.Join(sets, ", "), strings.Join(filters, " AND "))

		var stmt *sql.Stmt
		stmt, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt.Close()

		var r sql.Result
		r, q.Err = stmt.ExecContext(ctx, values...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["sqlExecErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		var ra int64
		ra, q.Err = r.RowsAffected()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["rowsAffectedErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["rowsAffectedErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		if ra == 0 {
			q.Err = fmt.Errorf(`无权编辑该考点或该考点不存在(id: %d)`, editInfo.ID.Int64)
			z.Error(q.Err.Error())
			break
		}

		newToken := ""

		for editInfo.ResetAccessToken {
			b1 := make([]byte, 32)
			rand.Read(b1)
			newToken = fmt.Sprintf("%x", b1)

			sqlStr = `UPDATE t_user SET user_token=crypt($1,gen_salt('bf')), updated_by=$2 WHERE id=(SELECT sys_user FROM t_exam_site WHERE id=$3)`

			var stmt *sql.Stmt
			stmt, q.Err = tx.Prepare(sqlStr)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr2"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["prepareErr2"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			defer stmt.Close()

			r, q.Err = stmt.ExecContext(ctx, newToken, userID, editInfo.ID.Int64)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr2"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["sqlExecErr2"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			ra, q.Err = r.RowsAffected()
			if q.Err != nil || (cmn.InDebugMode && q.Tag["rowsAffectedErr2"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["rowsAffectedErr2"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			newToken = generateAccessToken(editInfo.ID.Int64, newToken)

			if ra != 0 {
				break
			}

			// 考点原有的系统账号被删除, 重新创建一个
			var sysUserID int64
			sysUserID, newToken, q.Err = createSysUser(ctx, tx, editInfo.ID.Int64)
			if q.Err != nil {
				break MethodSwitch
			}

			sqlStr = `UPDATE t_exam_site SET sys_user=$1, updated_by=$2 WHERE id=$3`

			stmt, q.Err = tx.Prepare(sqlStr)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr3"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["prepareErr3"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			defer stmt.Close()

			r, q.Err = stmt.ExecContext(ctx, sysUserID, userID, editInfo.ID.Int64)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["sqlExecErr3"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["sqlExecErr3"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			ra, q.Err = r.RowsAffected()
			if q.Err != nil || (cmn.InDebugMode && q.Tag["rowsAffectedErr3"] != nil) {
				if q.Err == nil {
					q.Err = q.Tag["rowsAffectedErr3"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			break
		}

		editInfo.AccessToken = null.StringFrom(newToken)

		q.Msg.Data, q.Err = json.Marshal(editInfo)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshalErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["jsonMarshalErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

	case "DELETE":

		if !deletable {
			q.Err = fmt.Errorf("当前用户没有权限删除该数据")
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

// examSiteList 处理考点列表相关请求
func examSiteList(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	readable, _, _, _ := getApiPermissions(ctx, q.Ep.Path)

	dbConn := cmn.GetDbConn()

MethodSwitch:
	switch q.R.Method {

	case "GET":

		if !readable {
			q.Err = fmt.Errorf("当前用户没有权限获取数据")
			z.Error(q.Err.Error())
			break
		}

		keys := []string{}

		values := []interface{}{
			userID,
		}

		// 获取拥有访问权限的域上的数据
		dks := []string{
			"t_exam_site.creator=$1",
		}

		l := len(values)

		for i, d := range authority.AccessibleDomains {
			dks = append(dks, fmt.Sprintf("t_exam_site.domain_id=$%d", i+l+1))
			values = append(values, d)
		}

		if len(dks) > 0 {
			keys = append(keys, fmt.Sprintf("(%s)", strings.Join(dks, " OR ")))
		}

		param := q.R.URL.Query().Get("q")

		var req cmn.ReqProto

		q.Err = json.Unmarshal([]byte(param), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		if req.Page < 1 {
			q.Err = fmt.Errorf("页码不能小于1")
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
			i := len(values) + 1
			keys = append(keys, fmt.Sprintf(`(t_exam_site.name ILIKE $%d OR t_exam_site.address ILIKE $%d OR t_exam_site.server_host ILIKE $%d)`, i, i, i))
			values = append(values, fmt.Sprintf("%%%s%%", nameFilter))
		}

		orderBy := "t_exam_site.id"

		orderByList := []string{}

		for _, o := range req.OrderBy {
			for k, v := range o {

				if k == "" || v == "" {
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
		LIMIT %d OFFSET %d`, s, req.PageSize, (req.Page-1)*req.PageSize)

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

// examRoom 处理考场相关请求
func examRoom(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	readable, creatable, editable, deletable := getApiPermissions(ctx, q.Ep.Path)

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

		if !readable {
			q.Err = fmt.Errorf("当前用户没有权限获取该数据")
			z.Error(q.Err.Error())
			break
		}

	case "POST":

		if !creatable {
			q.Err = fmt.Errorf("当前用户没有权限创建该数据")
			z.Error(q.Err.Error())
			break
		}

		var info examRoomInfo

		q.Err = json.Unmarshal(req.Data, &info)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		q.Err = cmn.Validate(&info)
		if q.Err != nil {
			break
		}

		if info.Capacity <= 0 {
			q.Err = fmt.Errorf("考场容量必须大于0")
			z.Error(q.Err.Error())
			break
		}

		var stmt1 *sql.Stmt

		// 检查当前是否有权限访问该考点
		ss := []string{}
		v := []interface{}{
			info.ExamSiteID,
			userID,
		}

		l := len(v)
		for i, d := range authority.AccessibleDomains {
			ss = append(ss, fmt.Sprintf("domain_id = $%d", i+l+1))
			v = append(v, d)
		}

		sqlStr := fmt.Sprintf(`SELECT id FROM t_exam_site WHERE id = $1 AND  (creator = $2 OR %s)`, strings.Join(ss, " OR "))
		stmt1, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareCheckAccessSqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareCheckAccessSqlErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt1.Close()

		var r sql.Result
		r, q.Err = stmt1.ExecContext(ctx, v...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["execCheckAccessSqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["execCheckAccessSqlErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		var c int64
		c, q.Err = r.RowsAffected()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["getCheckAccessResultErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["getCheckAccessResultErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		if c == 0 {
			q.Err = fmt.Errorf("当前用户无权获取该考点数据, id: %d", info.ExamSiteID)
			z.Error(q.Err.Error())
			break
		}

		// 添加考场

		sqlStr = `INSERT INTO t_exam_room (exam_site, name, capacity, creator, updated_by, domain_id)
		VALUES ($1, $2, $3, $4, $5, $6)`

		stmt1, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareStmtErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareStmtErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt1.Close()

		_, q.Err = stmt1.ExecContext(ctx, info.ExamSiteID, info.Name, info.Capacity, userID, userID, authority.Domain.ID.Int64)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["execSQLErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["execSQLErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

	case "PATCH":

		if !editable {
			q.Err = fmt.Errorf("当前用户没有权限修改该数据")
			z.Error(q.Err.Error())
			break
		}

	case "DELETE":

		if !deletable {
			q.Err = fmt.Errorf("当前用户没有权限删除该数据")
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

func examRoomList(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	userID := q.SysUser.ID.Int64

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	readable, _, _, _ := getApiPermissions(ctx, q.Ep.Path)

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

MethodSwitch:
	switch q.R.Method {

	case "GET":

		if !readable {
			q.Err = fmt.Errorf("当前无权获取考场列表数据")
			z.Error(q.Err.Error())
			break
		}

		param := q.R.URL.Query().Get("q")

		var req cmn.ReqProto

		q.Err = json.Unmarshal([]byte(param), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}

		examSiteID := gjson.Get(param, "data.examSiteID").Int()

		var stmt1 *sql.Stmt

		// 检查当前是否有权限访问该考点
		ss := []string{}
		v := []interface{}{
			examSiteID,
			userID,
		}

		l := len(v)
		for i, d := range authority.AccessibleDomains {
			ss = append(ss, fmt.Sprintf("domain_id = $%d", i+l+1))
			v = append(v, d)
		}

		sqlStr := fmt.Sprintf(`SELECT id FROM t_exam_site WHERE id = $1 AND  (creator = $2 OR %s)`, strings.Join(ss, " OR "))
		stmt1, q.Err = tx.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareCheckAccessSqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareCheckAccessSqlErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt1.Close()

		var r sql.Result
		r, q.Err = stmt1.ExecContext(ctx, v...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["execCheckAccessSqlErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["execCheckAccessSqlErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		var c int64
		c, q.Err = r.RowsAffected()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["getCheckAccessResultErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["getCheckAccessResultErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		if examSiteID != 0 && c == 0 {
			q.Err = fmt.Errorf("当前用户无权获取该考点数据, id: %d", examSiteID)
			z.Error(q.Err.Error())
			break
		}

		// 获取考场列表数据

		keys := []string{
			"t_exam_room.exam_site=$1",
		}

		values := []interface{}{
			examSiteID,
		}

		if examSiteID == 0 {
			keys = []string{}
			values = []interface{}{}
		}

		if req.Page < 1 {
			q.Err = fmt.Errorf("页码不能小于1")
			z.Error(q.Err.Error())
			break
		}

		if req.PageSize < 1 {
			q.Err = fmt.Errorf("每页条数不能小于1")
			z.Error(q.Err.Error())
			break
		}

		ss = []string{}
		ss = append(ss, fmt.Sprintf("creator = $%d", len(values)+1))
		values = append(values, userID)
		l = len(values)
		for i, d := range authority.AccessibleDomains {
			ss = append(ss, fmt.Sprintf("domain_id = $%d", i+l+1))
			values = append(values, d)
		}

		keys = append(keys, fmt.Sprintf("(%s)", strings.Join(ss, " OR ")))

		nameFilter := gjson.Get(param, "filter.name").Str

		if nameFilter != "" {
			i := len(values) + 1
			keys = append(keys, fmt.Sprintf(`t_exam_room.name ILIKE $%d`, i))
			values = append(values, fmt.Sprintf("%%%s%%", nameFilter))
		}

		startTime := gjson.Get(param, "filter.startTime").Int()
		endTime := gjson.Get(param, "filter.endTime").Int()

		if startTime > endTime {
			q.Err = fmt.Errorf(`开始时间不能大于结束时间, 当前开始时间: %d, 结束时间: %d`, startTime, endTime)
			z.Error(q.Err.Error())
			break
		}

		if gjson.Get(param, "filter.available").IsBool() {
			i := len(values) + 1
			v := gjson.Get(param, "filter.available").Bool()
			keys = append(keys, fmt.Sprintf(`available = $%d`, i))
			values = append(values, v)

			if startTime == 0 || endTime == 0 {
				q.Err = fmt.Errorf(`当按可用状态获取时, 必须指定开始时间与结束时间, 当前开始时间: %d, 结束时间: %d`, startTime, endTime)
				z.Error(q.Err.Error())
				break
			}
		}

		orderBy := "t_exam_room.id"

		orderByList := []string{}

		for _, o := range req.OrderBy {
			for k, v := range o {

				if k == "" || v == "" {
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

		// 考场下不一定会安排考生考试，自然也就不一定会有相应的数据，
		// 所以采用"左外连接"的方式去获取考场下的考试数据，确保考场数据可以正常获取
		sqlStr = fmt.Sprintf(`WITH related_exams AS (
			SELECT
				t_exam_room.id AS room_id,
				t_exam_info.id AS exam_id,
				t_exam_info.name AS exam_name,
				t_exam_info.status AS exam_status,
				t_exam_session.id AS session_id,
				t_exam_session.start_time,
				t_exam_session.end_time,
				COALESCE((t_exam_session.end_time < %d OR t_exam_session.start_time > %d ), true) AS available,
				ROW_NUMBER() OVER (
					PARTITION BY t_exam_room.id 
					ORDER BY t_exam_session.start_time DESC
				) AS rn
			FROM t_exam_room
				LEFT JOIN t_examinee t_examinee ON t_examinee.exam_room = t_exam_room.id
				LEFT JOIN t_exam_session ON t_exam_session.id = t_examinee.exam_session_id
				LEFT JOIN t_exam_info t_exam_info ON t_exam_info.id = t_exam_session.exam_id
			GROUP BY
				t_exam_room.id,
				t_exam_info.id,
				t_exam_session.id
		)
		SELECT 
			t_exam_room.id,
			t_exam_room.exam_site,
			t_exam_room.name,
			t_exam_room.capacity,
			BOOL_AND(related_exams.available) AS available,
			json_agg(
				json_build_object(
					'exam_id', exam_id,
					'exam_name', exam_name,
					'exam_status', exam_status,
					'session_id', session_id,
					'start_time', start_time,
					'end_time', end_time
				)
			) FILTER (WHERE rn = 1) ::jsonb AS recent_exam
		FROM t_exam_room
			JOIN related_exams ON related_exams.room_id = t_exam_room.id
		WHERE %s
		GROUP BY 
			t_exam_room.id, 
			t_exam_room.exam_site, 
			t_exam_room.name
		ORDER BY
			%s
			`, startTime, endTime, strings.Join(keys, " AND "), orderBy)

		stmt1, q.Err = dbConn.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareSqlErr1"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareSqlErr1"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		r, q.Err = stmt1.ExecContext(ctx, values...)
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

		sqlStr = fmt.Sprintf(`%s
		LIMIT %d OFFSET %d`, sqlStr, req.PageSize, (req.Page-1)*req.PageSize)

		stmt1, q.Err = dbConn.Prepare(sqlStr)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["prepareErr2"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["prepareErr2"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer stmt1.Close()

		var rows *sql.Rows
		rows, q.Err = stmt1.QueryContext(ctx, values...)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["queryErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["queryErr"].(error)
			}

			z.Error(q.Err.Error())
			break
		}

		defer rows.Close()

		result := []examRoomInfo{}

		for rows.Next() {
			var item examRoomInfo
			q.Err = rows.Scan(
				&item.ID,
				&item.ExamSiteID,
				&item.Name,
				&item.Capacity,
				&item.Available,
				&item.RecentExam,
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

/* 考点同步服务 */
//  .oooooo..o
// d8P'    `Y8
// Y88bo.      oooo    ooo ooo. .oo.    .ooooo.
//  `"Y8888o.   `88.  .8'  `888P"Y88b  d88' `"Y8
//      `"Y88b   `88..8'    888   888  888
// oo     .d8P    `888'     888   888  888   .o8
// 8""88888P'      .8'     o888o o888o `Y8bod8P'
//             .o..P'
//             `Y8P'
//

// examSiteSyncInit 考点同步初始化
//
// 如果中心服务器Url配置(centralServerUrl)不为空, 则以考点服务器运行, 否则以中心服务器运行
func examSiteSyncInit(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	sshUserKey := "examSiteServerSync.centralServerSSH.user"
	sshHostKey := "examSiteServerSync.centralServerSSH.host"
	sshPortKey := "examSiteServerSync.centralServerSSH.port"
	if viper.IsSet(sshUserKey) {
		sshUser = viper.GetString(sshUserKey)
	}

	if viper.IsSet(sshHostKey) {
		sshHost = viper.GetString(sshHostKey)
	}

	if viper.IsSet(sshPortKey) {
		sshPort = viper.GetInt(sshPortKey)
	}

	uploadDirKey := "tusd.uploadDir"
	if viper.IsSet(uploadDirKey) {
		uploadDir = viper.GetString(uploadDirKey)
	}

	uploadDir, q.Err = filepath.Abs(uploadDir)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["absUploadDirErr"] != nil) {
		if q.Err == nil {
			q.Err = q.Tag["absUploadDirErr"].(error)
		}
		z.Error(q.Err.Error())
		return
	}

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

		if q.Err == nil || errors.Is(q.Err, redis.Nil) {
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

	switch v {

	case PULLING:

		// 如果上一次没有完成拉取, 则重置为未拉取的状态
		_, q.Err = q.RedisClient.Set(ctx, SyncStatusKey, PUSHED, 0).Result()
		if q.Err != nil || (cmn.InDebugMode && q.Tag["setPushedStatusErr"] != nil) {

			if q.Err == nil {
				q.Err = q.Tag["setPushedStatusErr"].(error)
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

	Pull(ctx, maxRetry, false)
	if q.Err != nil {
		z.Error("初始化拉取同步数据失败")
	}

	pullChan = make(chan int)
	pushChan = make(chan int)

	pullChanOK := true

	pushChanOK := true

	go func() {

		interval := 3 * time.Minute
		if viper.GetInt("examSiteServerSync.syncInterval") > 0 {
			interval = time.Duration(viper.GetInt("examSiteServerSync.syncInterval")) * time.Second
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		syncDelay := 5 * time.Minute
		if viper.GetInt("examSiteServerSync.syncDelay") > 0 {
			syncDelay = time.Duration(viper.GetInt("examSiteServerSync.syncDelay")) * time.Second
		}

		for {

			select {

			case _, ok := <-pullChan:

				pullChanOK = ok

				if !ok {
					pullChan = nil
					break
				}

				Pull(ctx, maxRetry, false)

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

			case <-ticker.C:

				// 查询当前是否有尚未结束的考试或者临近开始的考试
				// 如果有，则不进行同步操作
				// 如果没有，则进行同步

				dbConn := cmn.GetDbConn()

				// 时间计算方式: 当前时间 + 延迟执行Push的时间 + 同步间隔时间 + 30分钟的缓冲时间
				dontSyncBefore := time.Now().Add(syncDelay).Add(interval).Add(30*time.Minute).Unix() * 1000

				c := 0
				q.Err = dbConn.QueryRow(`SELECT COUNT(id) FROM t_exam_session WHERE status = '04' OR (status = '02' AND start_time < $1)`, dontSyncBefore).Scan(&c)
				if q.Err != nil || (cmn.InDebugMode && q.Tag["queryOngoingExamErr"] != nil) {

					if q.Err == nil {
						q.Err = q.Tag["queryOngoingExamErr"].(error)
					}

					z.Error(q.Err.Error())
					break
				}

				if c > 0 {

					if cmn.InDebugMode && q.Tag["haveOngoingExam"] != nil {
						q.Tag["haveOngoingExam"].(chan int) <- 1
					}

					break
				}

				redisConn := cmn.GetRedisConn()

				// 获取当前同步状态
				var syncStatus string
				syncStatus, q.Err = redisConn.Get(ctx, SyncStatusKey).Result()
				if q.Err != nil || (cmn.InDebugMode && q.Tag["getSyncStatusErrInTimer"] != nil) {

					if q.Err == nil {
						q.Err = q.Tag["getSyncStatusErrInTimer"].(error)
					}

					z.Error(q.Err.Error())
					break
				}

				ticker.Stop()

				switch syncStatus {

				// 待推送
				case PULLED:

					z.Info(fmt.Sprintf("server will push data in %d seconds", syncDelay/time.Second))

					time.Sleep(syncDelay)

					Push(ctx, maxRetry)

					if cmn.InDebugMode && q.Tag["pushDone"] != nil {
						q.Tag["pushDone"].(chan int) <- 1
					}

				// 待拉取
				case PUSHED:
				}

				// 待拉取或推送成功后，立即进行拉取操作
				// 之所以要在推送后立即拉取，是为了避免在考试如果在一次拉取后等待进入待推送状态时开始进行，
				// 导致考试期间的数据会因为下一次的拉取操作而被覆盖，同时，也为了确保考点服务器上的考试是最近一次的数据
				Pull(ctx, maxRetry, false)

				if cmn.InDebugMode && q.Tag["pullDone"] != nil {
					q.Tag["pullDone"].(chan int) <- 1
				}

				ticker.Reset(interval)
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

// examSiteSync 处理考点同步相关请求
//
// 当Http Method 为 GET 时, 为考点服务器向中心服务器获取同步数据
//
// 当Http Method 为 POST 时, 为考点服务器向中心服务器推送同步数据
func examSiteSync(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	dbConn := cmn.GetDbConn()

	// 考点服务器系统账号ID
	userID := q.SysUser.ID.Int64

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

		forceRenew := q.R.URL.Query().Get("forceRenew") == "true"

		if centralServerUrl != "" {

			action := q.R.URL.Query().Get("action")

			switch action {

			case "pull":
				Pull(ctx, maxRetry, forceRenew)
			case "push":
				Push(ctx, maxRetry)
			default:
				q.Err = fmt.Errorf("不支持的同步操作: %s", action)
				z.Error(q.Err.Error())
			}

			break MethodSwitch
		}

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

		for s != "" {

			// 如果同步数据信息快照存在，则直接返回

			// 验证数据有效性
			q.Err = json.Unmarshal([]byte(s), &info)
			if q.Err != nil {
				z.Error(q.Err.Error())
				break MethodSwitch
			}

			if forceRenew {

				z.Info("强制重新准备考点数据")

				// 直接删除已有缓存目录
				err := os.RemoveAll(info.Path)
				if err != nil || (cmn.InDebugMode && q.Tag["removeOldCacheDirErr"] != nil) {
					z.Error(err.Error())
				}

				break
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

			break
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

		var recentExamID int64
		var tableFileList []string
		fileName := "export_script.sql"

		if cmn.InDebugMode && q.Tag["createExportScriptFileErr"] != nil {
			fileName = q.Tag["createExportScriptFileErr"].(error).Error()
		}

		if cmn.InDebugMode && q.Tag["writeExportScriptFileErr"] != nil {
			fileName = q.Tag["writeExportScriptFileErr"].(error).Error()
		}

		recentExamID, tableFileList, q.Err = generateExportScript(userID, folderFullPath, fileName, false)
		if q.Err != nil {
			break MethodSwitch
		}

		// 执行导出脚本
		cmd = fmt.Sprintf("PGPASSFILE=%s psql -v ON_ERROR_STOP=1 -h %s -p %d -U %s -d %s -f %s",
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

		z.Info(string(o))

		var dir string
		dir, q.Err = filepath.Abs(uploadDir)
		if q.Err != nil || (cmn.InDebugMode && q.Tag["getAbsUploadDirErr"] != nil) {
			if q.Err == nil {
				q.Err = q.Tag["getAbsUploadDirErr"].(error)
			}

			z.Error(q.Err.Error())
			break MethodSwitch
		}

		info = syncInfo{
			RecentExamID:    recentExamID,
			Path:            folderFullPath,
			TableFileList:   tableFileList,
			UploadFilesPath: dir,
		}

		var data []byte

		// 返回同步数据信息
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

		// 执行导入脚本
		cmd := fmt.Sprintf(`PGPASSFILE=%s psql -v ON_ERROR_STOP=1 -h %s -p %d -U %s -d %s -f %s`,
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

		z.Info(string(o))

		defer func() {

			// 清理已同步的数据
			_, err := q.RedisClient.Del(ctx, syncInfoSnapshotKey).Result()
			if err != nil || (cmn.InDebugMode && q.Tag["deleteKeyErr"] != nil) {

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
		}()

		var s string
		var err error
		s, err = q.RedisClient.Get(ctx, syncInfoSnapshotKey).Result()
		if (err != nil && !errors.Is(err, redis.Nil)) || (cmn.InDebugMode && q.Tag["getSyncInfoSnapshotErr"] != nil) {
			if q.Err == nil {
				q.Err = q.Tag["getSyncInfoSnapshotErr"].(error)
			}

			z.Error(err.Error())
			break MethodSwitch
		}

		err = json.Unmarshal([]byte(s), &sInfo)
		if err != nil {
			z.Error(err.Error())
			break MethodSwitch
		}

		// 触发自动批改
		var marks []mark.QueryCondition
		var rows *sql.Rows
		rows, err = dbConn.Query(`SELECT id FROM t_exam_session WHERE exam_id = $1`, sInfo.RecentExamID)
		if err != nil || (cmn.InDebugMode && q.Tag["queryExamSessionsErr"] != nil) {
			if err == nil {
				err = q.Tag["queryExamSessionsErr"].(error)
			}

			z.Error(err.Error())

			break MethodSwitch
		}

		defer rows.Close()

		for rows.Next() {
			var id int64
			err = rows.Scan(&id)
			if err != nil || (cmn.InDebugMode && q.Tag["scanErr"] != nil) {
				if err == nil {
					err = q.Tag["scanErr"].(error)
				}

				z.Error(err.Error())

				break MethodSwitch
			}

			marks = append(marks, mark.QueryCondition{
				ExamSessionID: id,
			})
		}

		for _, m := range marks {
			z.Info(fmt.Sprintf("执行自动批改, 考试场次ID: %d", m.ExamSessionID))
			mark.AutoMark(ctx, m)
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
