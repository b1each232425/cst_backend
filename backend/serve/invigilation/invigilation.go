package invigilation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
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
	CanUpdate        bool                   `json:"canUpdate"`
}

type InvigilationFile struct {
	FileID        int64    `json:"FileID"`
	ExamSessionID int64    `json:"ExamSessionID"`
	ExamRoomID    null.Int `json:"ExamRoomID"`
	CheckSum      string   `json:"CheckSum"`
	Name          string   `json:"Name"`
	Size          int64    `json:"Size"`
}

var needToUpdateExaminee map[string]bool = map[string]bool{
	"00": false, // 考场记录
	"02": true,  // 考生状态
	"04": true,  // 考生备注
	"06": true,  // 考生延时
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

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "监考管理.获取监考列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: invigilation,

		Path: "/invigilation",
		Name: "invigilation",

		Developer: developer,
		WhiteList: true,

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "监考管理.获取监考详情",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "监考管理.更新监考信息",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: invigilationFile,

		Path: "/invigilation/file",
		Name: "invigilationFile",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}

func checkInvigilationAuthority(ctx context.Context, examSessionID int64, examRoomID null.Int, userID int64, domain string) (hasAuth bool, err error) {
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

func canUpdateInvigilationInfo(ctx context.Context, examSessionID int64, examRoomID null.Int) (canUpdate bool, err error) {
	z.Info("---->" + cmn.FncName())

	var forceErr string
	if val := ctx.Value("canUpdateInvigilationInfo-force-error"); val != nil {
		forceErr = val.(string)
	}

	if examSessionID <= 0 {
		err = fmt.Errorf("无效的考试场次ID: %d", examSessionID)
		z.Error(err.Error())
		return
	}

	conn := cmn.GetPgxConn()

	// 检查当前是否还能更新监考信息
	centralServerUrl := viper.GetString("examSiteServerSync.centralServerUrl")
	now := time.Now().UnixMilli()

	if centralServerUrl != "" {
		syncTime := viper.GetInt64("examSiteServerSync.syncDelay")
		syncTime = syncTime * 1000
		checkInvigilationSQL := `
		SELECT EXISTS(
				SELECT 1
				FROM v_invigilation_info vi
				WHERE vi.exam_session_id = $1 AND vi.exam_room_id = $2
				AND (vi.es_actual_end_time IS NULL OR vi.es_actual_end_time + $3 > $4)
			)
		`
		err = conn.QueryRow(ctx, checkInvigilationSQL, examSessionID, examRoomID, syncTime, now).Scan(&canUpdate)
		if forceErr == "checkInvigilation" {
			err = fmt.Errorf("强制检查当前是否还能更新监考信息错误")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}

		return
	}

	// 如果是中心服务器，则需检查该考试是否为线下考试，如果是线下则不允许更新
	checkInvigilationSQL := `
	SELECT EXISTS(
			SELECT 1
			FROM v_invigilation_info vi
			WHERE vi.exam_session_id = $1 AND vi.exam_room_id = $2
			AND vi.exam_mode != '02'
			AND (vi.es_actual_end_time IS NULL OR vi.es_actual_end_time > $3)
		)
	`
	err = conn.QueryRow(ctx, checkInvigilationSQL, examSessionID, examRoomID, now).Scan(&canUpdate)
	if forceErr == "checkInvigilation" {
		err = fmt.Errorf("强制检查当前是否还能更新监考信息错误")
	}
	if err != nil {
		z.Error(err.Error())
		return
	}

	return
}

func afterCaret(s string) string {
	parts := strings.SplitN(s, "^", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
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

	var readable bool
	readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if forceErr == "auth_mgt.GetUserAuthority" {
		q.Err = fmt.Errorf("强制获取用户权限错误")
	}
	if q.Err != nil {
		q.RespErr()
		return
	}

	userRole := afterCaret(authority.Role.Domain)

	conn := cmn.GetPgxConn()
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		if !readable {
			q.Err = fmt.Errorf("用户无权访问监考列表，请联系管理员获取权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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

		if page <= 0 {
			page = 1
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
		if userID > 0 && userRole != "examSiteAdmin" {
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
				abnormal_examinee_num,
				exam_mode
			FROM v_invigilation_info
			WHERE 1=1
		` + filterSQL + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, pageSize, (page-1)*pageSize)

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
				&inv.ExamMode,
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

	var readable bool
	readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var editable bool
	editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionUpdate)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var authority *auth_mgt.Authority
	z.Info("获取用户权限")
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if forceErr == "auth_mgt.GetUserAuthority" {
		q.Err = fmt.Errorf("强制获取用户权限错误")
	}
	if q.Err != nil {
		q.RespErr()
		return
	}

	userRole := afterCaret(authority.Role.Domain)

	conn := cmn.GetPgxConn()
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		if !readable {
			q.Err = fmt.Errorf("用户无权访问监考详情，请联系管理员获取权限，请联系管理员获取权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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

		var examRoomID null.Int
		roomID := gjson.Get(qry, "Data.ExamRoomID").Int()
		if roomID <= 0 {
			examRoomID.Int64 = 0
			examRoomID.Valid = false
		} else {
			examRoomID.Int64 = roomID
			examRoomID.Valid = true
		}

		// 检查用户是否有权限获取监考信息
		if userRole != "examSiteAdmin" {
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
				q.Err = fmt.Errorf("用户(%d)无法获取该场考试的监考信息: %d - %d", userID, examSessionID, examRoomID.Int64)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 先获取考场监考的基本信息
		var invigilatorInfo cmn.TVInvigilationInfo
		q.Err = conn.QueryRow(ctx, `
			SELECT invigilator_ids, invigilator_names, invigilator_num, exam_session_id, start_time, end_time, status, exam_site_id, exam_site_name, exam_room_id, exam_room_name, exam_room_capacity, exam_session_name, record, basic_eval, examinee_num, absentee_num, cheater_num, abnormal_examinee_num, exam_mode, exam_type
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
			&invigilatorInfo.ExamMode,
			&invigilatorInfo.ExamType,
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
		if page <= 0 {
			page = 1
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
			(examinee_number ILIKE '%%' || $%d || '%%' OR
			id_card_no ILIKE '%%' || $%d || '%%' OR
			official_name ILIKE '%%' || $%d || '%%')
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
		args = append(args, pageSize, (page-1)*pageSize)
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

		// 如果是考点服务器，则允许考试结束后继续更新监考信息
		centralServerUrl := viper.GetString("examSiteServerSync.centralServerUrl")
		var canUpdateWhenExamEnded bool = false
		if centralServerUrl != "" && forceErr != "centralServer" {
			canUpdateWhenExamEnded = true
		} else {
			canUpdateWhenExamEnded = false
		}

		info.CanUpdate = canUpdateWhenExamEnded

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
		if !editable {
			q.Err = fmt.Errorf("用户无权更新监考信息")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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

		var examRoomID null.Int
		roomID := gjson.Get(qry, "Data.ExamRoomID").Int()
		if roomID <= 0 {
			examRoomID.Int64 = 0
			examRoomID.Valid = false
		} else {
			examRoomID.Int64 = roomID
			examRoomID.Valid = true
		}

		// 检查用户是否有权限获取监考信息
		if userRole != "examSiteAdmin" {
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
				q.Err = fmt.Errorf("用户(%d)无法获取该场考试的监考信息: %d - %d", userID, examSessionID, examRoomID.Int64)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 00：考场记录，02：考生状态， 04：考生备注， 06：考生延时
		updateType := gjson.Get(qry, "Data.UpdateType").Str
		if updateType != "00" && updateType != "02" && updateType != "04" && updateType != "06" {
			q.Err = fmt.Errorf("无效的更新类型: %s", updateType)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		record := gjson.Get(qry, "Data.Record").Str

		basicEval := gjson.Get(qry, "Data.BasicEval").Str
		if basicEval != "00" && basicEval != "02" && basicEval != "04" && !needToUpdateExaminee[updateType] {
			q.Err = fmt.Errorf("无效的考场情况: %s", basicEval)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examineeStatus := gjson.Get(qry, "Data.ExamineeStatus").Str

		examineeRemark := gjson.Get(qry, "Data.ExamineeRemark").Str

		extraTime := gjson.Get(qry, "Data.ExtraTime").Int()

		var examineeIDs []int64
		examinees := gjson.Get(qry, "Data.Examinees").Array()
		for _, v := range examinees {
			id := v.Int()
			if id <= 0 {
				q.Err = fmt.Errorf("无效的考生ID: %d", id)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examineeIDs = append(examineeIDs, id)
		}

		if len(examineeIDs) == 0 && needToUpdateExaminee[updateType] {
			q.Err = fmt.Errorf("请指定考生")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var canUpdate bool
		now := time.Now().UnixMilli()
		canUpdate, q.Err = canUpdateInvigilationInfo(ctx, examSessionID, examRoomID)
		if forceErr == "canUpdateInvigilationInfo" {
			q.Err = fmt.Errorf("强制检查是否还能更新监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "canUpdate" {
			canUpdate = false
		}
		if !canUpdate {
			q.Err = fmt.Errorf("当前无法更新该考试场次的监考信息")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查考生是否是该考场的考生
		if needToUpdateExaminee[updateType] {
			checkSQL := `
            SELECT (
                SELECT COUNT(id) 
                FROM v_examinee_info vei
                WHERE vei.exam_session_id = $1
                  AND vei.exam_room_id = $2
                  AND vei.id = ANY($3)
                  AND vei.examinee_status NOT IN ('08','16')
            ) = COALESCE(array_length($3,1), 0)
		`
			var allValid bool
			q.Err = conn.QueryRow(ctx, checkSQL, examSessionID, examRoomID, examineeIDs).Scan(&allValid)
			if forceErr == "checkExaminees" {
				q.Err = fmt.Errorf("强制检查考生是否是该考场的考生错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if !allValid {
				q.Err = fmt.Errorf("要更新的考生不在该考场考试")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 开启事务准备进行更新
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

		// 更新考场记录
		if updateType == "00" {
			updateSQL := `
				UPDATE t_exam_record
				SET content = $1,
					basic_eval = COALESCE(NULLIF($2, ''), basic_eval),
					updated_by = $3,
					update_time = $4
				WHERE exam_room = $5 AND exam_session = $6
			`
			_, q.Err = tx.Exec(ctx, updateSQL, record, basicEval, userID, now, examRoomID, examSessionID)
			if forceErr == "updateExamRecord" {
				q.Err = fmt.Errorf("强制更新考场记录错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Resp()
			return
		}

		// 更新指定的考生状态
		if updateType == "02" {
			updateExamineeStatusSQL := `
			UPDATE t_examinee
				SET status = COALESCE(
						NULLIF($1, ''),
						CASE 
							WHEN t_examinee.end_time IS NOT NULL THEN '10' -- 如果该考生有结束时间，则设置为已交卷
							WHEN t_examinee.start_time IS NULL AND vei.actual_end_time <= $2 THEN '02' -- 如果该考生没有开始时间，并且该考生的考试时间结束了，则设置为缺考，此处设置的目的是为了处理exam_maintenance中可能出现的同时更新，确保最后覆盖的结果是正确的
							ELSE '00'
						END
					),
					updated_by = $3,
					update_time = $4
				FROM v_examinee_info vei
				WHERE t_examinee.id = ANY($5)
				AND vei.id = t_examinee.id
			`
			_, q.Err = tx.Exec(ctx, updateExamineeStatusSQL, examineeStatus, now, userID, now, examineeIDs)
			if forceErr == "updateExamineeStatus" {
				q.Err = fmt.Errorf("强制更新考生状态错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Resp()
			return
		}

		// 更新指定的考生备注
		if updateType == "04" {
			updateExamineeRemarkSQL := `
			UPDATE t_examinee
				SET remark = $1,
					updated_by = $2,
					update_time = $3,
					status = '00'
				FROM v_examinee_info vei
				WHERE t_examinee.id = ANY($4)
				AND vei.id = t_examinee.id
			`
			_, q.Err = tx.Exec(ctx, updateExamineeRemarkSQL, examineeRemark, userID, now, examineeIDs)
			if forceErr == "updateExamineeRemark" {
				q.Err = fmt.Errorf("强制更新考生备注错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Resp()
			return
		}

		// 先检查延长的时间是否符合要求
		if extraTime <= 0 {
			q.Err = fmt.Errorf("无效的延长时间: %d", extraTime)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 锁定场次记录，防止并发修改
		lockSQL := `
			SELECT 1 FROM t_exam_session WHERE id = $1 FOR UPDATE
		`
		_, q.Err = tx.Exec(ctx, lockSQL, examSessionID)
		if forceErr == "lockExamSession" {
			q.Err = fmt.Errorf("强制锁定考试场次错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		checkExamSessionSQL := `
			SELECT status FROM t_exam_session WHERE id = $1
		`
		var examSessionStatus string
		q.Err = tx.QueryRow(ctx, checkExamSessionSQL, examSessionID).Scan(&examSessionStatus)
		if forceErr == "checkExamSession" {
			q.Err = fmt.Errorf("强制检查考试场次状态错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if examSessionStatus != "04" || forceErr == "examSessionNotOngoing" {
			q.Err = fmt.Errorf("只能在考试进行时延长考生时间")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		checkExamineeSQL := `
		SELECT EXISTS(
			SELECT 1
			FROM v_examinee_info vei
			WHERE vei.id = ANY($1)
			AND vei.extendable_time < $2
		)`
		var invalid bool
		q.Err = conn.QueryRow(ctx, checkExamineeSQL, examineeIDs, extraTime).Scan(&invalid)
		if forceErr == "checkExtendableTime" {
			q.Err = fmt.Errorf("强制检查考生是否允许延长时间错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if invalid || forceErr == "invalidExtendableTime" {
			q.Err = fmt.Errorf("部分考生不允许延长 %d 分钟时间", extraTime)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 更新延长时间
		updateExamineeExtraTimeSQL := `
			UPDATE t_examinee
			SET extra_time = t_examinee.extra_time + $1,
				updated_by = $2,
				update_time = $3
			FROM v_examinee_info vei
			WHERE t_examinee.id = vei.id
			AND t_examinee.id = ANY($4)
			AND t_examinee.extra_time + $1 <= vei.extendable_time
		`
		var commandTag pgconn.CommandTag
		commandTag, q.Err = tx.Exec(ctx, updateExamineeExtraTimeSQL, extraTime, userID, now, examineeIDs)
		if forceErr == "updateExamineeExtraTime" {
			q.Err = fmt.Errorf("强制更新考生延长时间错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if commandTag.RowsAffected() != int64(len(examineeIDs)) || forceErr == "notAllExtended" {
			q.Err = fmt.Errorf("部分考生延长时间失败：超过允许的最大延长时间")
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

		if invigilationFile.ExamRoomID.Int64 <= 0 {
			invigilationFile.ExamRoomID.Valid = false
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
			q.Err = fmt.Errorf("用户(%d)无法上传该场考试的监考附件: %d - %d", userID, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID.Int64)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var canUpdate bool
		canUpdate, q.Err = canUpdateInvigilationInfo(ctx, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
		if forceErr == "canUpdateInvigilationInfo" {
			q.Err = fmt.Errorf("强制检查是否还能更新监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "canUpdate" {
			canUpdate = false
		}
		if !canUpdate {
			q.Err = fmt.Errorf("当前无法更新该考试场次的监考信息")
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

		if invigilationFile.ExamRoomID.Int64 <= 0 {
			invigilationFile.ExamRoomID.Valid = false
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
			q.Err = fmt.Errorf("用户(%d)无法删除该场考试的监考附件: %d - %d", userID, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID.Int64)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var canUpdate bool
		canUpdate, q.Err = canUpdateInvigilationInfo(ctx, invigilationFile.ExamSessionID, invigilationFile.ExamRoomID)
		if forceErr == "canUpdateInvigilationInfo" {
			q.Err = fmt.Errorf("强制检查是否还能更新监考信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if forceErr == "canUpdate" {
			canUpdate = false
		}
		if !canUpdate {
			q.Err = fmt.Errorf("当前无法更新该考试场次的监考信息")
			z.Error(q.Err.Error())
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
				q.Err = fmt.Errorf("强制获取文件信息错误: %w", q.Err)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingFiles = append(existingFiles, file)
		}

		if len(existingFiles) == 0 || forceErr == "noFiles" {
			q.Err = fmt.Errorf("未找到监考附件记录: %d - %d", invigilationFile.ExamSessionID, invigilationFile.ExamRoomID.Int64)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 删除对应的文件ID
		var exists bool
		for i, file := range existingFiles {
			if file.FileID == invigilationFile.FileID {
				existingFiles = append(existingFiles[:i], existingFiles[i+1:]...)
				exists = true
				break
			}
		}

		if !exists || forceErr == "fileNotFound" {
			q.Err = fmt.Errorf("未找到要删除的文件: %d", invigilationFile.FileID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
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
