package registration

//annotation:register-service
//author:{"name":"LilEYi","tel":"13535215794", "email":"3102128343@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
	"w2w.io/cmn"
)

const (
	//事件类型
	EVENT_TYPE_REGISTER_START      = "register_start"
	EVENT_TYPE_REGISTER_END        = "register_end"
	EVENT_TYPE_REGISTER_REVIEW_END = "register_reviewer_end"

	//默认最大并发数
	DEFAULT_MAX_WORKERS = 10
)

var (
	z                    *zap.Logger
	pgxConn              *pgxpool.Pool
	registerTimerManager *RegistrationTimerManager
)

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
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: registerReviewer,

		Path: "/registrationReviewer",
		Name: "registrationReviewer",

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
					pageSizeStr := q.R.URL.Query().Get("pageSize")
					//如果有id则只查询单个报名计划
					if idStr != "" && pageStr != "" && pageSizeStr != "" {
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

						r, practices, reviewers, currentNumber, err := LoadRegisterById(ctx, id)
						if err != nil {
							q.Err = err
							q.RespErr()
							return
						}
						result := Map{
							"register":       r,
							"practices":      practices,
							"reviewers":      reviewers,
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
					//校验reviewer_ids字段
					var reviewers []int64

					reviewers, q.Err = CheckReviewerIDs(r.Registration.ReviewerIds)
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					err := UpsertRegister(ctx, r.Registration, r.PracticeIds, userID, action, reviewers)
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
func registerReviewer(ctx context.Context) {
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
	if DomainID != 0 && DomainID < RegisterDomainID.Student {
		DomainID = RegisterDomainID.Teacher
	}
	switch DomainID {
	case RegisterDomainID.Teacher:
		{
			method := q.R.Method
			method = strings.ToLower(method)
			if method != "get" {
				q.Err = fmt.Errorf("请使用get方法调用/api/registrationReviewer")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			idStr := q.R.URL.Query().Get("id")
			if idStr == "" {
				q.Err = fmt.Errorf("请传入有效的报名计划ID")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var registerID int64
			registerID, q.Err = strconv.ParseInt(idStr, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			name := q.R.URL.Query().Get("name")
			pageStr := q.R.URL.Query().Get("page")
			if pageStr == "" {
				q.Err = fmt.Errorf("缺失分页查询页号")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var page int
			page, q.Err = strconv.Atoi(pageStr)
			if q.Err != nil {
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
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			orderBy := []string{"u.create_time desc"}

			reviewer, total, err := ListReviewers(ctx, userID, registerID, name, page, pageSize, orderBy)
			if err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var result Map
			result = Map{
				"total":     total,
				"reviewers": reviewer,
			}
			data, err := json.Marshal(result)
			if forceErr == "marshal" {
				q.Err = fmt.Errorf("marshal json失败")
			}
			if err != nil {
				z.Error(err.Error())
				q.RespErr()
				return
			}
			q.Msg.Data = data
			q.Msg.Msg = "OK"
			q.Msg.Status = 0
			q.Resp()
			return

		}

	}

}

// 创建定时器管理器
func NewRegistrationTimerManager(ctx context.Context, cancel context.CancelFunc) *RegistrationTimerManager {
	z.Info("---->" + cmn.FncName())
	maxWorkers := DEFAULT_MAX_WORKERS

	tm := &RegistrationTimerManager{
		timers:     make(map[string]*time.Timer),
		ctx:        ctx,
		cancel:     cancel,
		eventQueue: make(chan RegisterEvent, maxWorkers*15),
		maxWorkers: maxWorkers,
	}

	//启动固定数量的worker协程
	tm.startEventWorkers()
	return tm
}

// 启动固定数量的事件处理协程
func (tm *RegistrationTimerManager) startEventWorkers() {
	z.Info("---->" + cmn.FncName())
	for i := 0; i < tm.maxWorkers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-tm.ctx.Done():
					return
				case event := <-tm.eventQueue:
					tm.processEvent(event, workerID)
				}
			}
		}(i)
	}
}
func (tm *RegistrationTimerManager) processEvent(event RegisterEvent, workerID int) error {
	z.Info("---->" + cmn.FncName())
	switch event.Type {
	case "register_start":
		{
			//handleRegisterStart(tm.ctx, event)
		}
	case "register_end":
		{
			//handleRegisterEnd(tm.ctx, event)
		}
	default:
		err := fmt.Errorf("Invalid event type: %s", event.Type)
		z.Error(err.Error())
	}
	return nil
}

// 设置定时器
func (tm *RegistrationTimerManager) SetTimer(registerID int64, triggerTime int64, event RegisterEvent) {
	z.Info("---->" + cmn.FncName())
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	timerKey := fmt.Sprintf("%s_%d", event.Type, registerID)
	//如果已存在同类型的定时器，先停止它
	if existingTimer, exists := tm.timers[timerKey]; exists {
		existingTimer.Stop()
		delete(tm.timers, timerKey)
	}

	//计算延迟事件
	delay := time.Duration(triggerTime-time.Now().UnixMilli()) * time.Millisecond
	//如果时间已过，立即将时间添加到处理队列
	if delay <= 0 {
		select {
		case tm.eventQueue <- event:
		case <-tm.ctx.Done():
			return
		}
		return
	}
	//创建新的定时器
	timer := time.AfterFunc(delay, func() {
		//将事件添加到队列
		select {
		case tm.eventQueue <- event:
		case <-tm.ctx.Done():
			return
		}
		//从map中移除定时器
		tm.mutex.Lock()
		delete(tm.timers, timerKey)
		tm.mutex.Unlock()
	})
	tm.timers[timerKey] = timer
	z.Info("设置定时器",
		zap.String("event_type", event.Type),
		zap.Int64("register_id", registerID),
		zap.Duration("delay", delay))
}

// 取消定时器
func (tm *RegistrationTimerManager) CancelTimer(eventType string, registerID int64) {
	z.Info("---->" + cmn.FncName())
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	timerKey := fmt.Sprintf("%s_%d", eventType, registerID)
	if timer, exists := tm.timers[timerKey]; exists {
		timer.Stop()
		delete(tm.timers, timerKey)
		z.Info("取消定时器",
			zap.String("event_type", eventType),
			zap.Int64("register_id", registerID))
	}
}

// 设置报名计划定时器
func SetRegisterTimers(ctx context.Context, registerID int64) error {
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("SetRegisterTimers-force-error"); val != nil {
		forceErr = val.(string)
	}
	//查询报名计划信息
	s := `
	SELECT 
		r.id,
		r.start_time,
		r.end_time,
		r.reviewe_end_time
	FROM assessuser.t_register_plan r
	WHERE r.id = $1 AND r.status NOT IN ($2,$3,$4)
`
	rows, err := pgxConn.Query(ctx, s, registerID, RegisterStatus.Cancel, RegisterStatus.Deleted, RegisterStatus.Disabled)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if forceErr == "queryRegisterPlan" {
		err = fmt.Errorf("查询报名计划信息错误")
	}
	if err != nil {
		z.Error("查询报名计划信息失败",
			zap.Int64("register_id", registerID),
			zap.Error(err))
		return err
	}

	var registerPlanInfo []struct {
		RegisterID    int64 `json:"register_id"`
		StartTime     int64 `json:"start_time"`
		ReviewEndTime int64 `json:"review_end_time"`
		EndTime       int64 `json:"end_time"`
	}
	for rows.Next() {
		var registerId, startTime, endTime, reviewEndTime int64
		err := rows.Scan(&registerId, &startTime, &endTime, &reviewEndTime)
		if forceErr == "scanRegisterPlanInfo" {
			err = fmt.Errorf("获取报名计划信息错误")
		}
		if err != nil {
			z.Error("获取报名计划信息失败", zap.Error(err))
			return err
		}
		registerPlanInfo = append(registerPlanInfo, struct {
			RegisterID    int64 `json:"register_id"`
			StartTime     int64 `json:"start_time"`
			ReviewEndTime int64 `json:"review_end_time"`
			EndTime       int64 `json:"end_time"`
		}{
			RegisterID:    registerId,
			StartTime:     startTime,
			ReviewEndTime: reviewEndTime,
			EndTime:       endTime,
		})
		z.Info("查询到报名计划信息",
			zap.Int64("register_id", registerId),
			zap.Int64("start_time", startTime),
			zap.Int64("review_end_time", reviewEndTime),
			zap.Int64("end_time", endTime))
	}
	for _, registerPlan := range registerPlanInfo {
		//设置报名计划开始定时器
		registerTimerManager.SetTimer(registerPlan.RegisterID, registerPlan.StartTime, RegisterEvent{
			Type:       EVENT_TYPE_REGISTER_START,
			RegisterID: registerPlan.RegisterID,
		})
		//设置报名计划结束
		registerTimerManager.SetTimer(registerPlan.RegisterID, registerPlan.EndTime, RegisterEvent{
			Type:       EVENT_TYPE_REGISTER_END,
			RegisterID: registerPlan.RegisterID,
		})
		//设置报名计划审核结束
		registerTimerManager.SetTimer(registerPlan.RegisterID, registerPlan.ReviewEndTime, RegisterEvent{
			Type:       EVENT_TYPE_REGISTER_REVIEW_END,
			RegisterID: registerPlan.RegisterID,
		})
	}
	return nil
}

// 取消报名计划的所有的计时器
//func CancelRegisterTimers(ctx context.Context, registerID int64) error {
//	forceErr := ""
//	if val := ctx.Value("CancelRegisterTimers-force-error"); val != nil {
//		forceErr = val.(string)
//	}
//	s := `SELECT id FROM assessuser.t_register_plan WHERE id = $1 AND status NOT IN ($1 ,$2 ,$3 )`
//
//	//查询报名计划信息
//	rows, err := pgxConn.Query(ctx, s, registerID, RegisterStatus.Cancel, RegisterStatus.Deleted, RegisterStatus.Disabled)
//}
