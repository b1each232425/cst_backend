// Package course management
package course

//annotation:course-service
//author:{"name":"user","tel":"18928776452","email":"XUnion@GMail.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"w2w.io/null"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("user.Enroll called")

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
		Fn: course,

		Path: "/course",
		Name: "course",

		Developer: developer,
		WhiteList: false,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: courseList,

		Path: "/course/list",
		Name: "course/list",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}

func nameExists(name string, courseID int64) (exists bool, err error) {
	if name == "" {
		return
	}

	params := []interface{}{name}
	conn := cmn.GetPgxConn()
	s := "select count(id) from t_course where name = $1"
	if courseID > 0 {
		params = append(params, courseID)
		s = "select count(id) from t_course where name = $1 and id != $2"
	}
	r := conn.QueryRow(context.Background(), s, params...)

	var count int64
	err = r.Scan(&count)
	if err != nil {
		z.Error(err.Error())
		return
	}

	if count > 0 {
		exists = true
	}

	return
}

func course(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	conn := cmn.GetPgxConn()

	z.Info("---->" + cmn.FncName())
	isPublish := q.R.URL.Query().Get("publish") != ""

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			qry = `{
				"Action": "select",
				"OrderBy": [{"ID": "DESC"}],
				"Filter": [
					{"Status" : {"IN": ["00","02","04","06"]}}
				],
				"Sets": [
					"ID",
					"Type",
					"Category",
					"Name",
					"Issuer",
					"IssueTime",
          "Sections",
          "Tags",
					"Data",
					"DefaultRepo",
					"Creator",
					"CreateTime",
					"UpdatedBy",
					"UpdateTime",
					"Addi",
					"Remark",
					"Status"
				]				
			}`
		}

		if qry == "nameExists" {
			courseName := q.R.URL.Query().Get("name")
			if courseName == "" {
				q.Err = fmt.Errorf("请提供课程名称")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var courseID int64
			s := q.R.URL.Query().Get("ID")
			if s != "" {
				courseID, q.Err = strconv.ParseInt(s, 10, 64)
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				if courseID <= 0 {
					courseID = -1
				}
			}

			var exists bool
			exists, q.Err = nameExists(courseName, courseID)
			if q.Err != nil {
				q.RespErr()
				return
			}

			q.Msg.RowCount = 0
			if exists {
				q.Msg.RowCount = 1
			}
			q.Resp()
			return
		}

		var req cmn.ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var c cmn.TCourse
		c.TableMap = &c

		q.Err = cmn.DML(&c.Filter, &req)
		if q.Err != nil {
			q.RespErr()
			return
		}

		v, ok := c.QryResult.(string)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be string, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.RowCount = c.RowCount
		q.Msg.Data = types.JSONText(v)
		q.Resp()

	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/course by post with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		courseID := int64(gjson.Get(string(buf), "data.ID").Num)
		if courseID <= 0 {
			var refinedBuf string
			refinedBuf, q.Err = sjson.Delete(string(buf), "data.ID")
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			buf = []byte(refinedBuf)
		}

		isUpdate := q.R.URL.Query().Get("isUpdate") == "true"
		courseName := gjson.Get(string(buf), "data.Name").String()
		if courseName == "" && !isUpdate {
			q.Err = fmt.Errorf("数据要包含课程名称")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var exists bool
		exists, q.Err = nameExists(courseName, 0)
		if q.Err != nil {
			q.RespErr()
			return
		}

		if exists && courseID <= 0 {
			q.Err = fmt.Errorf("【%s】已经存在，请创造一个更好的课程名称", courseName)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var c cmn.TCourse
		c.TableMap = &c
		//z.Info(string(qry.Data))
		q.Err = json.Unmarshal(qry.Data, &c)
		if q.Err != nil {
			z.Info(string(qry.Data))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userID := null.IntFrom(1000)
		if q.SysUser != nil {
			userID = q.SysUser.ID
		}
		c.Creator = userID

		q.Err = cmn.InvalidEmptyNullValue(&c)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 00：草稿
		// 02：已发布/上架
		// 04：已撤消/下架
		// 06：已禁用
		if !isUpdate {
			c.Status = null.NewString("00", true)
		}
		if isPublish {
			c.Status = null.NewString("02", true)
			c.IssueTime = null.NewInt(cmn.GetNowInMS(), true)
		}

		c.Action = "insert"
		if isUpdate {
			c.Action = "update"
		}
		//req := cmn.ReqProto{
		//	Action: c.Action,
		//}
		q.Err = cmn.DML(&c.Filter, &qry)
		if q.Err != nil {
			q.RespErr()
			return
		}

		courseID, ok := c.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("courseID, ok := c.QryResult.(int64) should be ok while it's not")
			q.RespErr()
			return
		}
		if !isUpdate {
			c.ID = null.IntFrom(courseID)
		}

		buf, q.Err = cmn.MarshalJSON(&c)
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Msg.Data = buf
		q.Resp()

	case "put":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("请指定更新参数q，不然咋办")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		courseID := gjson.Get(qry, "data.ID").String()
		if courseID == "" {
			q.Err = fmt.Errorf("数据中没有包含课程编号")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		courseName := gjson.Get(qry, "data.Name").String()
		if courseName != "" {
			var cid int64
			cid, q.Err = strconv.ParseInt(courseID, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var exists bool
			exists, q.Err = nameExists(courseName, cid)
			if q.Err != nil {
				q.RespErr()
				return
			}

			if exists {
				q.Err = fmt.Errorf("【%s】已经存在，请创造一个更好的课程名称", courseName)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		qry, q.Err = sjson.Delete(qry, "data.CreateTime")
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		qry, q.Err = sjson.Delete(qry, "data.UpdateTime")
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		qry, q.Err = sjson.Delete(qry, "data.Creator")
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var req cmn.ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if strings.ToLower(req.Action) != "update" {
			q.Err = fmt.Errorf("please specify update as action")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(req.Data) == 0 {
			q.Err = fmt.Errorf("不指定data，你想干啥子哦？")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var c cmn.TCourse
		c.TableMap = &c
		c.Action = req.Action
		if isPublish {
			c.Status = null.NewString("02", true)
			c.IssueTime = null.NewInt(cmn.GetNowInMS(), true)
		}
		t := cmn.GetNowInMS()
		z.Info(fmt.Sprintf("%d", t))
		c.UpdateTime = null.NewInt(t, true)
		c.CreateTime = null.NewInt(0, false)

		q.Err = cmn.DML(&c.Filter, &req)
		if q.Err != nil {
			q.RespErr()
			return
		}

		rowAffected, ok := c.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("_, ok = c.filter.qryResult.(string) should be ok while it's not")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if rowAffected == 0 {
			q.Err = fmt.Errorf("找不到的编号为%s的课程", courseID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"RowAffected":%d}`, rowAffected))
		q.Resp()
		return

	case "delete":
		courseID := q.R.URL.Query().Get("id")
		if courseID == "" {
			q.Err = fmt.Errorf("must provide course criteria")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var id int64
		id, q.Err = strconv.ParseInt(courseID, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		s := "delete from t_course where id=$1"

		var cmdTag pgconn.CommandTag
		cmdTag, q.Err = conn.Exec(ctx, s, id)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.RowCount = cmdTag.RowsAffected()
		if q.Msg.RowCount == 0 {
			q.Err = fmt.Errorf("不存在编号为%d的课程，删除失败", id)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

func courseList(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			qry = `{
				"Action": "select",
				"OrderBy": [{"ID": "DESC"}],
				"Filter": [
					{"Status" : {"IN": ["00","02","04","06"]}}
				],
				"Sets": [
					"ID",
					"Type",
					"Category",
					"Name",
					"Issuer",
					"IssueTime",
          "Sections",
          "Tags",
					"Data",
					"DefaultRepo",
					"Creator",
					"CreateTime",
					"UpdatedBy",
					"UpdateTime",
					"Addi",
					"Remark",
					"Status"
				]				
			}`
		}

		var req cmn.ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var c cmn.TCourse
		c.TableMap = &c

		q.Err = cmn.DML(&c.Filter, &req)
		if q.Err != nil {
			q.RespErr()
			return
		}

		v, ok := c.QryResult.(string)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be string, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.RowCount = c.RowCount
		q.Msg.Data = types.JSONText(v)
		q.Resp()
		break

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}
