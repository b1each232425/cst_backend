package invigilation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
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

		var req cmn.ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if forceErr == "json.Unmarshal1" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if req.Page < 0 {
			req.Page = 0
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
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
		args = append(args, req.PageSize, req.Page*req.PageSize)

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
