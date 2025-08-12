// Package exam site management
package exam_site

//annotation:exam-site-service
//author:{"name":"Zhong Peiqi","tel":"13068178042","email":"zpekii3156@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"crypto/rand"
	"database/sql"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"

	// "github.com/tidwall/sjson"

	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
)

var (
	z                *zap.Logger
	createPgpassOnce sync.Once
)

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

	switch q.R.Method {

	case "GET":

	case "POST":

		var info examSiteInfo

		q.Err = json.Unmarshal(req.Data, &info)
		if q.Err != nil {
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

		// 该Read从不返回错误，一旦发生错误就直接Panic， 所以这里就不需要接收err
		// 并且始终填充 b
		_, _ = rand.Read(b1)

		_, _ = rand.Read(b2)

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

func examSiteSync(ctx context.Context) {

	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	q.Msg.Msg = cmn.FncName()

	// 考点服务器系统账号ID
	userID := q.SysUser.ID.Int64

	dbAddr := viper.GetString("dbms.postgresql.addr")

	dbPort := viper.GetInt("dbms.postgresql.port")

	dbName := viper.GetString("dbms.postgresql.db")

	dbUser := viper.GetString("dbms.postgresql.user")

	dbPwd := viper.GetString("dbms.postgresql.pwd")

	pgpassFullPath := filepath.Join(os.Getenv("HOME"), ".pgpass")

	// 创建.pgpass文件
	createPgpassOnce.Do(func() {

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
	})

	if q.Err != nil {
		q.RespErr()
		return
	}

MethodSwitch:
	switch q.R.Method {

	case "GET":

		action := q.R.URL.Query().Get("action")

		switch action {

		case "0":
			// 准备考点数据

			b := make([]byte, 16)

			_, q.Err = rand.Read(b)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["generateRandByteErr"] != nil){

				if q.Err == nil {
					q.Err = q.Tag["generateRandByteErr"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

			folderFullPath := filepath.Join(os.Getenv("PWD"), fmt.Sprintf("/data/tmp/%x", b))

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

			tableFileList, q.Err = generateExportScriptsCentralSide(userID, folderFullPath, fileName)
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

			d := struct {
				Path     string   `json:"path"`
				TableFileList []string `json:"tableFileList"`
			}{
				Path:     folderFullPath,
				TableFileList: tableFileList,
			}

			q.Msg.Data, q.Err = json.Marshal(d)
			if q.Err != nil || (cmn.InDebugMode && q.Tag["jsonMarshalErr"] != nil) {

				if q.Err == nil {
					q.Err = q.Tag["jsonMarshalErr"].(error)
				}

				z.Error(q.Err.Error())
				break MethodSwitch
			}

		case "1":
			// 同步考点数据

		default:
			q.Err = fmt.Errorf("不支持的同步操作: %s", action)
			z.Error(q.Err.Error())
			break MethodSwitch
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
