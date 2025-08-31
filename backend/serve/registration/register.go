package registration

//annotation:register-service
//author:{"name":"LilEYi","tel":"13535215794", "email":"3102128343@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"w2w.io/cmn"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
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
		Fn: register,

		Path: "/registration",
		Name: "registration",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: registerStudentH,

		Path: "/registrationStudent",
		Name: "registrationStudent",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// just a trial
func register(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	// 用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid UserID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var DomainID int64
	for _, domain := range q.Domains {
		if domain.ID.Int64 == RegisterDomainID.Student || domain.ID.Int64 == RegisterDomainID.Teacher || domain.ID.Int64 == RegisterDomainID.Admin || domain.ID.Int64 == RegisterDomainID.SuperAdmin {
			DomainID = domain.ID.Int64
			break
		}
	}
	if DomainID != 0 && DomainID < RegisterDomainID.Student {
		DomainID = RegisterDomainID.Teacher
	}
	switch DomainID {
	case RegisterDomainID.Student:
		{
			method := strings.ToLower(q.R.Method)
			switch method {
			case "get":
				{
					name := q.R.URL.Query().Get("name")
					course := q.R.URL.Query().Get("course")
					status := q.R.URL.Query().Get("status")
					pageStr := q.R.URL.Query().Get("page")
					if pageStr == "" {
						q.Err = fmt.Errorf("缺失分页查询页号")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					var page int
					page, q.Err = strconv.Atoi(pageStr)
					if forceErr == "pageParseInt" {
						q.Err = fmt.Errorf("将字符串转化为整形失败")
					}
					if q.Err != nil {
						q.Err = fmt.Errorf("分页查询的页号解析失败：%v", q.Err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					pageSizeStr := q.R.URL.Query().Get("pageSize")
					if pageSizeStr == "" {
						q.Err = fmt.Errorf("缺失分页查询页大小")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					var pageSize int
					pageSize, q.Err = strconv.Atoi(pageSizeStr)
					if forceErr == "pageSizeParseInt" {
						q.Err = fmt.Errorf("将字符串转化为整形失败")
					}
					if q.Err != nil {
						q.Err = fmt.Errorf("分页页大小解析失败：%v", q.Err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}

					//排序字段
					orderBy := []string{"r.create_time desc"}
					r, total, err := ListRegisterS(ctx, name, course, status, orderBy, page, pageSize, userID)
					if err != nil {
						q.Err = err
						q.RespErr()
						return
					}
					result := Map{
						"total":     total,
						"registers": r,
					}
					data, err := json.Marshal(result)
					if forceErr == "json" {
						err = fmt.Errorf("将要返回数据结构反序列化失败")
					}
					if err != nil {
						z.Error(err.Error())
						q.Err = err
						q.RespErr()
						return
					}
					q.Msg.Data = data
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "OK"
					q.Resp()
				}
			case "post":
				{
					var buf []byte
					buf, q.Err = io.ReadAll(q.R.Body)
					if forceErr == "ioReadAll" {
						q.Err = fmt.Errorf("尝试打开前端数据流失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					defer func() {
						q.Err = q.R.Body.Close()
						if forceErr == "Close" {
							q.Err = fmt.Errorf("关闭打开的二进制数据流失败")
						}
						if q.Err != nil {
							z.Error(q.Err.Error())
						}
					}()
					if len(buf) == 0 {
						q.Err = fmt.Errorf("Call /api/registration with  empty body")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//获取请求的结构体
					var qry cmn.ReqProto
					q.Err = json.Unmarshal(buf, &qry)
					if forceErr == "readReqProtoJson" {
						q.Err = fmt.Errorf("读取前端数据结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}

					var rs registerStudentOnce
					q.Err = json.Unmarshal(qry.Data, &rs)
					if forceErr == "readPJson" {
						q.Err = fmt.Errorf("读取前端数据practice字段结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if rs.RegisterID <= 0 {
						q.Err = fmt.Errorf("无效的报名计划ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if rs.Status == "" {
						q.Err = fmt.Errorf("无效的报名计划状态")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					var student registerStudentType
					var students []registerStudentType
					student.StudentID = userID
					student.ExamType = ExamType.Normal
					students = append(students, student)
					err := StudentRegister(ctx, rs.RegisterID, rs.Status, RegisterType.Once, students, userID)
					if err != nil {
						q.Err = err
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "OK"
					q.Resp()
					return
				}
			}
		}
	case RegisterDomainID.Teacher:
		{
			method := strings.ToLower(q.R.Method)
			switch method {
			case "get":
				{
					name := q.R.URL.Query().Get("name")
					course := q.R.URL.Query().Get("course")
					status := q.R.URL.Query().Get("status")
					idStr := q.R.URL.Query().Get("id")
					pageStr := q.R.URL.Query().Get("page")

					var page int
					page, q.Err = strconv.Atoi(pageStr)
					if forceErr == "pageParseInt" {
						q.Err = fmt.Errorf("将字符串转化为整形失败")
					}
					if q.Err != nil {
						q.Err = fmt.Errorf("分页查询的页号解析失败：%v", q.Err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					pageSizeStr := q.R.URL.Query().Get("pageSize")
					var pageSize int
					pageSize, q.Err = strconv.Atoi(pageSizeStr)
					if forceErr == "pageSizeParseInt" {
						q.Err = fmt.Errorf("将字符串转化为整形失败")
					}
					if q.Err != nil {
						q.Err = fmt.Errorf("分页页大小解析失败：%v", q.Err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//如果有id则只查询单个报名计划
					if idStr != "" && pageStr != "" && pageSizeStr != "" {
						message := q.R.URL.Query().Get("message")
						registerType := q.R.URL.Query().Get("register_type")
						var id int64
						id, q.Err = strconv.ParseInt(idStr, 10, 64)
						if forceErr == "idParseInt" {
							q.Err = fmt.Errorf("将字符串转化为整形失败")
						}
						if q.Err != nil {
							q.Err = fmt.Errorf("报名计划ID解析失败：%v", q.Err.Error())
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
						//设置排序字段
						orderBy := []string{"eps.create_time desc"}
						s, total, err := GetRegisterStudentById(ctx, id, message, registerType, status, orderBy, page, pageSize, userID)
						if err != nil {
							q.Err = err
							q.RespErr()
							return
						}
						result := Map{
							"student": s,
							"total":   total,
						}
						data, err := json.Marshal(result)
						if forceErr == "json" {
							err = fmt.Errorf("将要返回数据结构反序列化失败")
						}
						if err != nil {
							z.Error(err.Error())
							q.Err = err
							q.RespErr()
							return
						}
						q.Msg.Data = data
						z.Info("---->" + cmn.FncName())
						q.Msg.Msg = "OK"
						q.Resp()
						return
					} else if idStr != "" && pageStr == "" && pageSizeStr == "" {
						//查询单个报名计划详情
						var id int64
						id, q.Err = strconv.ParseInt(idStr, 10, 64)
						if q.Err != nil {
							q.Err = fmt.Errorf("报名计划ID解析失败：%v", q.Err.Error())
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}

						r, practiceIds, currentNumber, err := LoadRegisterById(ctx, id)
						if err != nil {
							q.Err = err
							q.RespErr()
							return
						}
						result := Map{
							"register":       r,
							"practice_ids":   practiceIds,
							"current_number": currentNumber,
						}
						data, err := json.Marshal(result)
						if forceErr == "json" {
							err = fmt.Errorf("将要返回数据结构反序列化失败")
						}
						if err != nil {
							z.Error(err.Error())
							q.Err = err
							q.RespErr()
							return
						}
						q.Msg.Data = data
						z.Info("---->" + cmn.FncName())
						q.Msg.Msg = "OK"
						q.Resp()
						return
					}
					//排序字段
					orderBy := []string{"r.create_time desc"}
					r, total, err := ListRegisterT(ctx, name, course, status, orderBy, page, pageSize, userID)
					if err != nil {
						q.Err = err
						q.RespErr()
						return
					}
					result := Map{
						"total":     total,
						"registers": r,
					}
					data, err := json.Marshal(result)
					if forceErr == "json" {
						err = fmt.Errorf("将要返回数据结构反序列化失败")
					}
					if err != nil {
						z.Error(err.Error())
						q.Err = err
						q.RespErr()
						return
					}
					q.Msg.Data = data
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "OK"
					q.Resp()
					return
				}
			case "post":
				{
					var buf []byte
					buf, q.Err = io.ReadAll(q.R.Body)
					if forceErr == "ioReadAll" {
						q.Err = fmt.Errorf("尝试打开前端数据流失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					defer func() {
						q.Err = q.R.Body.Close()
						if forceErr == "Close" {
							q.Err = fmt.Errorf("关闭打开的二进制数据流失败")
						}
						if q.Err != nil {
							z.Error(q.Err.Error())
						}
					}()
					if len(buf) == 0 {
						q.Err = fmt.Errorf("Call /api/registration with  empty body")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//获取请求的结构体
					var qry cmn.ReqProto
					q.Err = json.Unmarshal(buf, &qry)
					if forceErr == "readReqProtoJson" {
						q.Err = fmt.Errorf("读取前端数据结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					var action string
					action = qry.Action
					var r RegisterInfo
					q.Err = json.Unmarshal(qry.Data, &r)
					if forceErr == "readRJson" {
						q.Err = fmt.Errorf("读取前端数据Register字段结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//校验Register字段
					q.Err = ValidateRegisterInfo(r.Registration, r.PracticeIds)
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					err := UpsertRegister(ctx, r.Registration, r.PracticeIds, userID, action)
					if err != nil {
						q.Err = err
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "OK"
					q.Resp()
					return
				}
			case "patch":
				{
					idStr := q.R.URL.Query().Get("ids")
					status := q.R.URL.Query().Get("status")
					if idStr == "" {
						q.Err = fmt.Errorf("缺失报名计划ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					idArray := strings.Split(idStr, ",")

					var ids []int64
					var invalidValues []string
					for _, s := range idArray {
						c := strings.TrimSpace(s)
						if c == "" {
							continue
						}
						id, err := strconv.ParseInt(c, 10, 64)
						if err != nil {
							invalidValues = append(invalidValues, s)
							continue
						}
						ids = append(ids, id)
					}
					if len(ids) == 0 {
						q.Err = fmt.Errorf("请传入有效的需要操作的报名计划ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if len(invalidValues) > 0 {
						// 这里就要返回了
						q.Err = fmt.Errorf("传入的报名计划ID中存在非法值：%v", invalidValues)
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//批量操作报名状态
					q.Err = OperateRegisterStatus(ctx, ids, status, userID)
					if q.Err != nil {
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "ok"
					q.Msg.Status = 0
					q.Resp()
					return
				}
			default:
				q.Err = fmt.Errorf("please call /api/Practice with  http POST/GET/PATCH method")
				q.RespErr()
				return

			}
		}
	}
}
func registerStudentH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	//用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid UserID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var DomainID int64
	for _, domain := range q.Domains {
		if domain.ID.Int64 == RegisterDomainID.Student || domain.ID.Int64 == RegisterDomainID.Teacher || domain.ID.Int64 == RegisterDomainID.Admin || domain.ID.Int64 == RegisterDomainID.SuperAdmin {
			DomainID = domain.ID.Int64
			break
		}
	}
	//把不是学生权限的都转为教师权限
	if DomainID != 0 && DomainID < RegisterDomainID.Student {
		DomainID = RegisterDomainID.Teacher
	}
	switch DomainID {
	case RegisterDomainID.Teacher:
		{
			method := strings.ToLower(q.R.Method)
			switch method {
			case "patch":
				{
					idStr := q.R.URL.Query().Get("ids")
					status := q.R.URL.Query().Get("status")
					registerIDStr := q.R.URL.Query().Get("register_id")
					failReason := q.R.URL.Query().Get("fail_reason")
					if registerIDStr == "" {
						q.Err = fmt.Errorf("缺失报名计划ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					registerID, err := strconv.ParseInt(registerIDStr, 10, 64)
					if err != nil {
						q.Err = fmt.Errorf("报名计划ID转换失败")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if idStr == "" {
						q.Err = fmt.Errorf("缺失报名计划学生ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					idArray := strings.Split(idStr, ",")
					var ids []int64
					var invalidValues []string
					for _, s := range idArray {
						if s == "" {
							continue
						}
						id, err := strconv.ParseInt(s, 10, 64)
						if err != nil {
							invalidValues = append(invalidValues, s)
							continue
						}
						ids = append(ids, id)
					}
					if len(ids) == 0 {
						q.Err = fmt.Errorf("报名计划学生ID为空")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if len(invalidValues) > 0 {
						q.Err = fmt.Errorf("传入的报名计划学生ID中存在非法值：%v", invalidValues)
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					q.Err = OperateRegisterStudentStatus(ctx, nil, ids, status, userID, registerID, failReason)
					if q.Err != nil {
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "ok"
					q.Msg.Status = 0
					q.Resp()
					return
				}
			case "post":
				{
					var buf []byte
					buf, q.Err = io.ReadAll(q.R.Body)
					if forceErr == "ioReadAll" {
						q.Err = fmt.Errorf("尝试打开前端数据流失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					defer func() {
						q.Err = q.R.Body.Close()
						if forceErr == "Close" {
							q.Err = fmt.Errorf("关闭打开的二进制数据流失败")
						}
						if q.Err != nil {
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
					}()
					if len(buf) == 0 {
						q.Err = fmt.Errorf("Call /api/register with  empty body")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					var qry cmn.ReqProto
					q.Err = json.Unmarshal(buf, &qry)
					if forceErr == "readReqProtoJson" {
						q.Err = fmt.Errorf("读取前端数据结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					//当Action为“move的时候为迁移操作”
					if qry.Action == "move" {
						var ms moveStudent
						q.Err = json.Unmarshal(qry.Data, &ms)
						if forceErr == "readMoveStudentJson" {
							q.Err = fmt.Errorf("读取前端数据MoveStudent字段结构体失败")
						}
						if q.Err != nil {
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
						if ms.ToRegisterID <= 0 || ms.FromRegisterID <= 0 {
							q.Err = fmt.Errorf("请传入有效的迁移的报名计划ID")
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
						if len(ms.Student) == 0 {
							q.Err = fmt.Errorf("请传入有效的迁移的报名计划学生ID")
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
						q.Err = MoveStudent(ctx, ms.FromRegisterID, ms.ToRegisterID, ms.Student, ms.Status, userID)
						if q.Err != nil {
							q.RespErr()
							return
						}
						z.Info("---->" + cmn.FncName())
						q.Msg.Msg = "ok"
						q.Msg.Status = 0
						q.Resp()
						return

					}
					var rs registerStudent
					q.Err = json.Unmarshal(qry.Data, &rs)
					if forceErr == "readPJson" {
						q.Err = fmt.Errorf("读取前端数据Register字段结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if rs.RegisterID <= 0 {
						q.Err = fmt.Errorf("请传入有效的报名计划ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if len(rs.Student) == 0 {
						q.Err = fmt.Errorf("请传入有效的报名计划学生ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					q.Err = UpsertRegisterStudent(ctx, rs.RegisterID, rs.Student, userID)
					if q.Err != nil {
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Msg = "ok"
					q.Msg.Status = 0
					q.Resp()
					return
				}
			}
		}
	}
	z.Info("---->" + cmn.FncName())
	q.Msg.Msg = cmn.FncName()
	q.Resp()
}
