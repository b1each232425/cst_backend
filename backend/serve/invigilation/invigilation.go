package invigilation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
)

//annotation:invigilation-service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

var (
	z *zap.Logger
)

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("invigilation zLogger settled")
	})
}

type ExamineeInfo struct {
	StudentID      null.Int    `json:"StudentID"`      // 学生ID
	ExamineeID     null.Int    `json:"ExamineeID"`     // 考生ID
	ExamSessionID  null.Int    `json:"ExamSessionID"`  // 考试场次ID
	ExamRoomID     null.Int    `json:"ExamRoomID"`     // 考场ID
	ExamCard       null.String `json:"ExamCard"`       // 考生准考证号
	IDCardNo       null.String `json:"IDCardNo"`       // 考生身份证号
	Name           null.String `json:"Name"`           // 考生姓名
	Status         null.String `json:"Status"`         // 考试状态
	Remark         null.String `json:"Remark"`         // 备注
	ExtraTime      null.String `json:"ExtraTime"`      // 延长时间, 例如 "00:30:00" 表示延长30分钟
	ExtendableTime null.String `json:"ExtendableTime"` // 可延长时间, 例如 "00:30:00" 表示可延长30分钟
}

type InvigilationDetails struct {
	InvigilationInfo cmn.TVInvigilationInfo `json:"info"`
	Examinees        []ExamineeInfo         `json:"examinees"`
	Files            []InvigilationFile     `json:"files"`
}

type InvigilationFile struct {
	FileID        int64  `json:"FileID"`
	ExamSessionID int64  `json:"ExamSessionID"`
	ExamRoomID    int64  `json:"ExamRoomID"`
	CheckSum      string `json:"CheckSum"`
	Name          string `json:"Name"`
	Size          int64  `json:"Size"`
}

func Enroll(author string) {
	z.Info("exam_mgt.Enroll called")

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
		Fn: invigilationList,

		Path: "/invigilation/list",
		Name: "invigilationList",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: invigilation,

		Path: "/invigilation",
		Name: "invigilation",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}

func checkInvigilationAuthority(ctx context.Context, examSessionID, examRoomID, userID int64, domain string) (hasAuth bool, err error) {
	z.Info("---->" + cmn.FncName())

	var forceErr string
	if val := ctx.Value("checkInvigilationAuthority-force-error"); val != nil {
		forceErr = val.(string)
	}

	if examSessionID <= 0 {
		err = fmt.Errorf("无效的考试场次ID: %d", examSessionID)
		z.Error(err.Error())
		return
	}

	if examRoomID <= 0 {
		err = fmt.Errorf("无效的考场ID: %d", examRoomID)
		z.Error(err.Error())
		return
	}

	if userID <= 0 {
		err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(err.Error())
		return
	}

	conn := cmn.GetPgxConn()
	checkSQL := `
		SELECT EXISTS (
			SELECT 1
			FROM v_invigilation_info
			WHERE exam_session_id = $1 AND exam_room_id = $2 
			AND $3 = ANY(invigilator_ids)
		)
	`
	err = conn.QueryRow(context.Background(), checkSQL, examSessionID, examRoomID, userID).Scan(&hasAuth)
	if forceErr == "QueryRow" {
		err = fmt.Errorf("强制查询错误")
	}
	if err != nil {
		z.Error(err.Error())
		return
	}

	return
}

func invigilationList(ctx context.Context) {
	z.Info("---->" + cmn.FncName())
	q := cmn.GetCtxValue(ctx)

	// 用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("invigilationList-force-error"); val != nil {
		forceErr = val.(string)
	}

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	conn := cmn.GetPgxConn()
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			qry = `{
				"Action": "select",
				"OrderBy": [{"ID": "DESC"}],
				"Filter": {
					"ExamSessionName": "",
					"ExamRoomName": "",
					"StartTime": -1,
					"EndTime": -1,
					"ExamSessionStatus": ""
				},
				"Page": 0,
				"PageSize": 10
			}`
		}

		page := gjson.Get(qry, "Page").Int()
		pageSize := gjson.Get(qry, "PageSize").Int()

		if page < 0 {
			page = 0
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		var filterStr []string
		var args []interface{}
		argIndex := 1

		examSessionNameFilter := gjson.Get(qry, "Filter.ExamSessionName").Str
		if examSessionNameFilter != "" {
			filterStr = append(filterStr, fmt.Sprintf("exam_session_name ILIKE '%%' || $%d || '%%'", argIndex))
			args = append(args, examSessionNameFilter)
			argIndex++
		}

		examRoomNameFilter := gjson.Get(qry, "Filter.ExamRoomName").Str
		if examRoomNameFilter != "" {
			filterStr = append(filterStr, fmt.Sprintf("exam_room_name ILIKE '%%' || $%d || '%%'", argIndex))
			args = append(args, examRoomNameFilter)
			argIndex++
		}

		startTimeFilter := gjson.Get(qry, "Filter.StartTime").Int()
		if startTimeFilter >= 0 {
			filterStr = append(filterStr, fmt.Sprintf("start_time >= $%d", argIndex))
			args = append(args, startTimeFilter)
			argIndex++
		}

		endTimeFilter := gjson.Get(qry, "Filter.EndTime").Int()
		if endTimeFilter >= 0 {
			filterStr = append(filterStr, fmt.Sprintf("end_time <= $%d", argIndex))
			args = append(args, endTimeFilter)
			argIndex++
		}

		examSessionStatusFilter := gjson.Get(qry, "Filter.ExamSessionStatus").Str
		if examSessionStatusFilter != "" {
			filterStr = append(filterStr, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, examSessionStatusFilter)
			argIndex++
		}

		// 只查询自己要监考的考试
		if userID > 0 {
			filterStr = append(filterStr, fmt.Sprintf("$%d = ANY(invigilator_ids)", argIndex))
			args = append(args, userID)
			argIndex++
		}

		// 只查询特定状态的考试
		var statusList = []string{"02", "04", "06", "08", "10", "12"}
		filterStr = append(filterStr, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, statusList)
		argIndex++

		var filterSQL string
		if len(filterStr) > 0 {
			filterSQL = " AND " + strings.Join(filterStr, " AND ")
		}

		var countSQL string
		countSQL = "SELECT COUNT(exam_session_id) FROM v_invigilation_info WHERE 1=1" + filterSQL
		q.Err = conn.QueryRow(ctx, countSQL, args...).Scan(&q.Msg.RowCount)
		if forceErr == "QueryRow" {
			q.Err = fmt.Errorf("强制执行错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var querySQL string
		querySQL = `
			SELECT
				invigilator_ids,
				invigilator_names,
				invigilator_num,
				exam_session_id,
				start_time,
				end_time,
				status,
				exam_site_id,
				exam_site_name,
				exam_room_id,
				exam_room_name,
				exam_room_capacity,
				exam_session_name,
				record,
				basic_eval,
				examinee_num,
				absentee_num,
				cheater_num,
				abnormal_examinee_num
			FROM v_invigilation_info
			WHERE 1=1
		` + filterSQL + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, pageSize, page*pageSize)

		var rows pgx.Rows
		rows, q.Err = conn.Query(ctx, querySQL, args...)
		if rows != nil {
			defer rows.Close()
		}
		if forceErr == "conn.Query" {
			q.Err = fmt.Errorf("强制执行错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var invigilations []cmn.TVInvigilationInfo
		for rows.Next() {
			var inv cmn.TVInvigilationInfo
			q.Err = rows.Scan(
				&inv.InvigilatorIds,
				&inv.InvigilatorNames,
				&inv.InvigilatorNum,
				&inv.ExamSessionID,
				&inv.StartTime,
				&inv.EndTime,
				&inv.Status,
				&inv.ExamSiteID,
				&inv.ExamSiteName,
				&inv.ExamRoomID,
				&inv.ExamRoomName,
				&inv.ExamRoomCapacity,
				&inv.ExamSessionName,
				&inv.Record,
				&inv.BasicEval,
				&inv.ExamineeNum,
				&inv.AbsenteeNum,
				&inv.CheaterNum,
				&inv.AbnormalExamineeNum,
			)
			if forceErr == "rows.Scan" {
				q.Err = fmt.Errorf("强制执行错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			invigilations = append(invigilations, inv)
		}
		q.Msg.Data, q.Err = json.Marshal(invigilations)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
	return
}

// 监考详情
func invigilation(ctx context.Context) {
	z.Info("---->" + cmn.FncName())
	q := cmn.GetCtxValue(ctx)

	// 用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("invigilationList-force-error"); val != nil {
		forceErr = val.(string)
	}

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	conn := cmn.GetPgxConn()
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 获取考试详情
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("参数 q 不能为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examSessionID := gjson.Get(qry, "Data.ExamSessionID").Int()
		if examSessionID <= 0 {
			q.Err = fmt.Errorf("无效的考试场次ID: %d", examSessionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examRoomID := gjson.Get(qry, "Data.ExamRoomID").Int()
		if examRoomID <= 0 {
			q.Err = fmt.Errorf("无效的考场ID: %d", examRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查用户是否有权限获取监考信息
		var hasAuth bool
		hasAuth, q.Err = checkInvigilationAuthority(ctx, examSessionID, examRoomID, userID, "")
		if forceErr == "checkInvigilationAuthority" {
			q.Err = fmt.Errorf("强制检查用户是否有权限获取监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "noAuth" {
			hasAuth = false
		}
		if !hasAuth {
			q.Err = fmt.Errorf("用户(%d)无法获取该场考试的监考信息: %d - %d", userID, examSessionID, examRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 先获取考场监考的基本信息
		var invigilatorInfo cmn.TVInvigilationInfo
		q.Err = conn.QueryRow(ctx, `
			SELECT invigilator_ids, invigilator_names, invigilator_num, exam_session_id, start_time, end_time, status, exam_site_id, exam_site_name, exam_room_id, exam_room_name, exam_room_capacity, exam_session_name, record, basic_eval, examinee_num, absentee_num, cheater_num, abnormal_examinee_num
			FROM v_invigilation_info
			WHERE exam_session_id = $1 AND exam_room_id = $2
		`, examSessionID, examRoomID).Scan(
			&invigilatorInfo.InvigilatorIds,
			&invigilatorInfo.InvigilatorNames,
			&invigilatorInfo.InvigilatorNum,
			&invigilatorInfo.ExamSessionID,
			&invigilatorInfo.StartTime,
			&invigilatorInfo.EndTime,
			&invigilatorInfo.Status,
			&invigilatorInfo.ExamSiteID,
			&invigilatorInfo.ExamSiteName,
			&invigilatorInfo.ExamRoomID,
			&invigilatorInfo.ExamRoomName,
			&invigilatorInfo.ExamRoomCapacity,
			&invigilatorInfo.ExamSessionName,
			&invigilatorInfo.Record,
			&invigilatorInfo.BasicEval,
			&invigilatorInfo.ExamineeNum,
			&invigilatorInfo.AbsenteeNum,
			&invigilatorInfo.CheaterNum,
			&invigilatorInfo.AbnormalExamineeNum,
		)
		if forceErr == "queryInvigilationInfo" {
			q.Err = fmt.Errorf("强制查询监考信息错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		page := gjson.Get(qry, "Page").Int()
		if page < 0 {
			page = 0
		}
		pageSize := gjson.Get(qry, "PageSize").Int()
		if pageSize <= 0 {
			pageSize = 10
		}

		var filterStr []string
		var args []interface{}
		argIndex := 1

		// 添加搜索过滤条件（如果有的话）
		searchTextFilter := gjson.Get(qry, "Filter.SearchText").Str
		if searchTextFilter != "" {
			filterStr = append(filterStr, fmt.Sprintf(`
			examinee_number ILIKE '%%' || $%d || '%%' OR
			id_card_no ILIKE '%%' || $%d || '%%' OR
			official_name ILIKE '%%' || $%d || '%%' 
			`, argIndex, argIndex, argIndex))
			args = append(args, searchTextFilter)
			argIndex++
		}

		// 加上考场过滤条件
		filterStr = append(filterStr, fmt.Sprintf("exam_session_id = $%d", argIndex))
		args = append(args, examSessionID)
		argIndex++

		filterStr = append(filterStr, fmt.Sprintf("exam_room_id = $%d", argIndex))
		args = append(args, examRoomID)
		argIndex++

		var filterSQL string
		if len(filterStr) > 0 {
			filterSQL = " AND " + strings.Join(filterStr, " AND ")
		}

		var countSQL string
		countSQL = "SELECT COUNT(id) FROM v_examinee_info WHERE 1=1 AND examinee_status != '08'" + filterSQL
		q.Err = conn.QueryRow(ctx, countSQL, args...).Scan(&q.Msg.RowCount)
		if forceErr == "queryExamineeCount" {
			q.Err = fmt.Errorf("强制执行错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examineeSQL := `
			SELECT 
				student_id,
				id AS examinee_id,
				exam_session_id,
				exam_room_id,
				examinee_number,
				id_card_no,
				official_name AS student_name,
				examinee_status,
				remark,
				extra_time,
				extendable_time
			FROM v_examinee_info
			WHERE 1=1 AND examinee_status != '08'
		` + filterSQL + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, pageSize, page*pageSize)
		var rows pgx.Rows
		rows, q.Err = conn.Query(ctx, examineeSQL, args...)
		if rows != nil {
			defer rows.Close()
		}
		if forceErr == "queryExamineeInfos" {
			q.Err = fmt.Errorf("强制执行错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examinees []ExamineeInfo
		for rows.Next() {
			var e ExamineeInfo
			q.Err = rows.Scan(
				&e.StudentID,
				&e.ExamineeID,
				&e.ExamSessionID,
				&e.ExamRoomID,
				&e.ExamCard,
				&e.IDCardNo,
				&e.Name,
				&e.Status,
				&e.Remark,
				&e.ExtraTime,
				&e.ExtendableTime,
			)

			z.Sugar().Infof(forceErr)
			if forceErr == "rows.Scan" {
				q.Err = fmt.Errorf("强制执行错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examinees = append(examinees, e)
		}

		var info InvigilationDetails
		info.InvigilationInfo = invigilatorInfo
		info.Examinees = examinees

		// 获取监考附件信息
		querySQL := `
		SELECT er.exam_room, er.exam_session, f.id, f.digest, f.file_name
		FROM t_exam_record er
		CROSS JOIN LATERAL jsonb_array_elements_text(er.files) AS fid(file_id)
		JOIN t_file f ON f.id = fid.file_id::bigint
		WHERE er.exam_room = $1 AND er.exam_session = $2;
		`
		var existingFiles []InvigilationFile
		var fileRows pgx.Rows
		fileRows, q.Err = conn.Query(ctx, querySQL, examRoomID, examSessionID)
		if fileRows != nil {
			defer fileRows.Close()
		}
		if forceErr == "queryFiles" {
			q.Err = fmt.Errorf("强制查询文件错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for fileRows.Next() {
			var file InvigilationFile
			q.Err = fileRows.Scan(&file.ExamRoomID, &file.ExamSessionID, &file.FileID, &file.CheckSum, &file.Name)
			if forceErr == "scanFiles" {
				q.Err = fmt.Errorf("强制获取文件信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingFiles = append(existingFiles, file)
		}

		info.Files = existingFiles

		q.Msg.Data, q.Err = json.Marshal(info)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

	case "patch":
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
	return
}

func invigilationFile(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 当前用户登录选择的域
	userRole := q.SysUser.Role.Int64

	// 开启事务
	var tx pgx.Tx
	tx, q.Err = conn.Begin(ctx)
	if forceErr == "tx.Begin" {
		q.Err = fmt.Errorf("强制开始事务错误")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil || forceErr == "tx.Rollback" {
			rollbackErr := tx.Rollback(context.Background())
			if forceErr == "tx.Rollback" {
				rollbackErr = fmt.Errorf("强制回滚事务错误")
			}
			if rollbackErr != nil {
				z.Error(fmt.Sprintf("failed to rollback transaction: %s", rollbackErr.Error()))
			}
			return
		}

		commitErr := tx.Commit(context.Background())
		if forceErr == "tx.Commit" {
			commitErr = fmt.Errorf("强制提交事务错误")
		}
		if commitErr != nil {
			z.Error(fmt.Sprintf("failed to commit transaction: %s", commitErr.Error()))
		}
	}()

	if forceErr == "tx.Commit" || forceErr == "tx.Rollback" {
		return
	}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if forceErr == "io.Close" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var invigilationFile InvigilationFile
		q.Err = json.Unmarshal(qry.Data, &invigilationFile)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invigilationFile.ExamSessionID <= 0 {
			q.Err = fmt.Errorf("无效的考试场次ID: %d", invigilationFile.ExamSessionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invigilationFile.ExamRoomID <= 0 {
			q.Err = fmt.Errorf("无效的考场ID: %d", invigilationFile.ExamRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var hasAuth bool
		hasAuth, q.Err = checkInvigilationAuthority(ctx, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID, userID, "")
		if forceErr == "checkInvigilationAuthority" {
			q.Err = fmt.Errorf("强制检查用户是否有权限获取监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "noAuth" {
			hasAuth = false
		}
		if !hasAuth {
			q.Err = fmt.Errorf("用户(%d)无法上传该场考试的监考附件: %d - %d", userID, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var newFileID int64
		newFileID, q.Err = cmn.NewFileRecord(ctx, tx, invigilationFile.CheckSum, invigilationFile.Name, invigilationFile.Size, userRole, userID)
		if forceErr == "NewFileRecord" {
			q.Err = fmt.Errorf("强制创建文件记录错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 获取监考附件信息
		querySQL := `
		SELECT er.exam_room, er.exam_session, f.id, f.digest, f.file_name
		FROM t_exam_record er
		CROSS JOIN LATERAL jsonb_array_elements_text(er.files) AS fid(file_id)
		JOIN t_file f ON f.id = fid.file_id::bigint
		WHERE er.exam_room = $1 AND er.exam_session = $2;
		`
		var existingFiles []InvigilationFile
		var fileRows pgx.Rows
		fileRows, q.Err = tx.Query(ctx, querySQL, invigilationFile.ExamRoomID, invigilationFile.ExamSessionID)
		if fileRows != nil {
			defer fileRows.Close()
		}
		if forceErr == "queryFiles" {
			q.Err = fmt.Errorf("强制查询文件错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for fileRows.Next() {
			var file InvigilationFile
			q.Err = fileRows.Scan(&file.ExamRoomID, &file.ExamSessionID, &file.FileID, &file.CheckSum, &file.Name)
			if forceErr == "scanFiles" {
				q.Err = fmt.Errorf("强制获取文件信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingFiles = append(existingFiles, file)
		}

		invigilationFile.FileID = newFileID
		existingFiles = append(existingFiles, invigilationFile)

		// 组成新的文件ID数组
		var fileIDs []int64
		for _, file := range existingFiles {
			fileIDs = append(fileIDs, file.FileID)
		}

		// 更新考试监考记录中的附件列表
		var filesJSON []byte
		filesJSON, q.Err = json.Marshal(fileIDs)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		_, q.Err = tx.Exec(ctx, `UPDATE t_exam_record SET files = $1 WHERE exam_session = $2 AND exam_room = $3`, filesJSON, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
		if forceErr == "tx.Exec" {
			q.Err = fmt.Errorf("强制更新监考记录错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data, q.Err = json.Marshal(existingFiles)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Resp()
		return

	case "delete":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if forceErr == "io.Close" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var invigilationFile InvigilationFile
		q.Err = json.Unmarshal(qry.Data, &invigilationFile)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invigilationFile.ExamSessionID <= 0 {
			q.Err = fmt.Errorf("无效的考试场次ID: %d", invigilationFile.ExamSessionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invigilationFile.ExamRoomID <= 0 {
			q.Err = fmt.Errorf("无效的考场ID: %d", invigilationFile.ExamRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invigilationFile.FileID <= 0 {
			q.Err = fmt.Errorf("无效的文件ID: %d", invigilationFile.FileID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var hasAuth bool
		hasAuth, q.Err = checkInvigilationAuthority(ctx, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID, userID, "")
		if forceErr == "checkInvigilationAuthority" {
			q.Err = fmt.Errorf("强制检查用户是否有权限获取监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "noAuth" {
			hasAuth = false
		}
		if !hasAuth {
			q.Err = fmt.Errorf("用户(%d)无法上传该场考试的监考附件: %d - %d", userID, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取监考附件记录
		// 获取监考附件信息
		querySQL := `
		SELECT er.exam_room, er.exam_session, f.id, f.digest, f.file_name
		FROM t_exam_record er
		CROSS JOIN LATERAL jsonb_array_elements_text(er.files) AS fid(file_id)
		JOIN t_file f ON f.id = fid.file_id::bigint
		WHERE er.exam_room = $1 AND er.exam_session = $2;
		`
		var existingFiles []InvigilationFile
		var fileRows pgx.Rows
		fileRows, q.Err = tx.Query(ctx, querySQL, invigilationFile.ExamRoomID, invigilationFile.ExamSessionID)
		if fileRows != nil {
			defer fileRows.Close()
		}
		if forceErr == "queryFiles" {
			q.Err = fmt.Errorf("强制查询文件错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for fileRows.Next() {
			var file InvigilationFile
			q.Err = fileRows.Scan(&file.ExamRoomID, &file.ExamSessionID, &file.FileID, &file.CheckSum, &file.Name)
			if forceErr == "scanFiles" {
				q.Err = fmt.Errorf("强制获取文件信息错误: %w", q.Err)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingFiles = append(existingFiles, file)
		}

		if len(existingFiles) == 0 {
			q.Err = fmt.Errorf("未找到监考附件记录: %d - %d", invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 删除对应的文件ID
		for i, file := range existingFiles {
			if file.FileID == invigilationFile.FileID {
				existingFiles = append(existingFiles[:i], existingFiles[i+1:]...)
				break
			}
		}

		var fileIDs []int64
		for _, file := range existingFiles {
			fileIDs = append(fileIDs, file.FileID)
		}

		// 更新考试监考记录中的附件列表
		var filesJSON []byte
		filesJSON, q.Err = json.Marshal(fileIDs)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		_, q.Err = tx.Exec(ctx, `UPDATE t_exam_record SET files = $1 WHERE exam_session = $2 AND exam_room = $3`, filesJSON, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
		if forceErr == "tx.Exec" {
			q.Err = fmt.Errorf("强制更新监考记录错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 序列化要返回给前端的数据
		q.Msg.Data, q.Err = json.Marshal(existingFiles)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制序列化文件信息错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 最后删除对应的文件记录
		q.Err = cmn.DeleteFileRecord(ctx, tx, invigilationFile.FileID)
		if forceErr == "DeleteFileRecord" {
			q.Err = fmt.Errorf("强制删除文件记录错误")
		}
		if q.Err != nil {
			q.Msg.Data = nil
			q.RespErr()
			return
		}

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
	return
}
