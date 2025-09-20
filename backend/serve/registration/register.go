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
	"w2w.io/serve/auth_mgt"
)

const (
	//事件类型
	EVENT_TYPE_REGISTER_START      = "register_start"
	EVENT_TYPE_REGISTER_END        = "register_end"
	EVENT_TYPE_REGISTER_REVIEW_END = "register_review_end"

	//默认最大并发数
	DEFAULT_MAX_WORKERS = 5
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
		pgxConn = cmn.GetPgxConn()
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
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.查询报名计划",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "报名管理.创建报名计划",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "报名管理.编辑报名计划",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
			{
				Name:         "报名管理.删除报名计划",
				AccessAction: auth_mgt.CAPIAccessActionDelete,
				Configurable: true,
			},
		},

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
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.查询报名学生",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "报名管理.导入报名学生",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "报名管理.通过或不通过报名学生",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

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

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.查询报名审核人",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn:   registerStatus,
		Path: "/register/Status",
		Name: "registrationStatus",
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.更新报名状态",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},
		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn:   registerStudentL,
		Path: "/register/student",
		Name: "registrationStudentL",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.查询报名列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "报名管理.学生报名",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
		},
		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: registerDetail,

		Path: "/registration/detail",
		Name: "registrationDetail",
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.查询报名计划详情",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},
		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: registerStudentMove,

		Path: "/registerStudent/move",
		Name: "registrationStudentMove",
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "报名管理.移动学生",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	read, create, update, deleteble, err := GetAuthAPIAccessible(ctx, authority, "/api/registration")
	z.Error(fmt.Sprintf("获取用户的可执行权限的路径: %v", q.Ep.Path))
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		{
			if !read {
				q.Err = fmt.Errorf("该用户没有读数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			name := q.R.URL.Query().Get("name")
			course := q.R.URL.Query().Get("course")
			status := q.R.URL.Query().Get("status")
			pageStr := q.R.URL.Query().Get("page")
			pageSizeStr := q.R.URL.Query().Get("pageSize")

			searchType := q.R.URL.Query().Get("search_type")
			if searchType != "00" && searchType != "02" {
				q.Err = fmt.Errorf("搜索类型错误")
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
			r, total, err := ListRegisterT(ctx, name, course, status, orderBy, page, pageSize, userID, searchType)
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
			if !r.Registration.ID.Valid {
				if !create {
					q.Err = fmt.Errorf("该用户没有创建数据权限")
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			} else {
				if !update {
					q.Err = fmt.Errorf("该用户没有更新数据权限")
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

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
			if !deleteble {
				q.Err = fmt.Errorf("该用户没有删除数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	read, create, update, _, err := GetAuthAPIAccessible(ctx, authority, "/api/registrationStudent")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		{
			if !read {
				q.Err = fmt.Errorf("该用户没有读数据的权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			idStr := q.R.URL.Query().Get("id")
			pageStr := q.R.URL.Query().Get("page")
			pageSizeStr := q.R.URL.Query().Get("pageSize")
			status := q.R.URL.Query().Get("status")
			//如果有id则只查询单个报名计划
			if idStr != "" && pageStr != "" && pageSizeStr != "" {
				searchType := q.R.URL.Query().Get("search_type")
				if searchType != "00" && searchType != "02" {
					q.Err = fmt.Errorf("搜索类型错误")
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
				s, total, err := GetRegisterStudentById(ctx, id, message, registerType, status, orderBy, page, pageSize, userID, searchType)
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
			}
		}
	case "patch":
		{
			if !update {
				q.Err = fmt.Errorf("该用户没有更新数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
			if !create {
				q.Err = fmt.Errorf("该用户没有创建数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	read, _, _, _, err := GetAuthAPIAccessible(ctx, authority, "/api/registrationReviewer")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	method := q.R.Method
	method = strings.ToLower(method)
	if method != "get" {
		q.Err = fmt.Errorf("请使用get方法调用/api/registrationReviewer")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if !read {
		q.Err = fmt.Errorf("该用户没有读数据权限")
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
func registerStatus(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid UserID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	_, _, update, _, err := GetAuthAPIAccessible(ctx, authority, "/api/register/Status")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}

	method := q.R.Method
	method = strings.ToLower(method)
	switch method {
	case "patch":
		{
			if !update {
				q.Err = fmt.Errorf("该用户没有更新数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
	}

}
func registerDetail(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	read, _, _, _, err := GetAuthAPIAccessible(ctx, authority, "/api/registration/detail")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	method := q.R.Method
	method = strings.ToLower(method)
	switch method {
	case "get":
		{
			if !read {
				q.Err = fmt.Errorf("该用户没有读数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			idStr := q.R.URL.Query().Get("id")
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

	}

}
func registerStudentMove(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	_, create, _, _, err := GetAuthAPIAccessible(ctx, authority, "/api/registerStudent/move")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	method := q.R.Method
	method = strings.ToLower(method)
	switch method {
	case "post":
		{
			if !create {
				q.Err = fmt.Errorf("该用户没有创建数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var buf []byte
			buf, q.Err = io.ReadAll(q.R.Body)
			if q.Err != nil || forceErr == "ioRead" {
				q.Err = fmt.Errorf("获取前端数据流失败,%v", q.Err)
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
				q.Err = fmt.Errorf("数据流为空")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var qry cmn.ReqProto
			q.Err = json.Unmarshal(buf, &qry)
			if q.Err != nil || forceErr == "Unmarshal" {
				q.Err = fmt.Errorf("读取前端数据结构体失败")
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
		}
	}
}

func registerStudentL(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
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
	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	read, create, _, _, err := GetAuthAPIAccessible(ctx, authority, "/api/register/student")
	if err != nil {
		q.Err = fmt.Errorf("获取用户的可执行权限失败: %v", err)
		q.RespErr()
		return
	}
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		{
			if !read {
				q.Err = fmt.Errorf("该用户没有读数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
			if !create {
				q.Err = fmt.Errorf("该用户没有创建数据权限")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
	case "register_review_end":
		{
			err := handleRegisterReviewEndEvent(tm.ctx, event)
			if err != nil {
				z.Error("处理报名审核结束事件失败",
					zap.Error(err),
					zap.Int64("register_id", event.RegisterID))
			}
		}
	case "register_end":
		{
			err := handleRegisterEndEvent(tm.ctx, event)
			if err != nil {
				z.Error("处理报名结束事件失败",
					zap.Error(err),
					zap.Int64("register_id", event.RegisterID))
			}

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
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	//查询报名计划信息
	s := `
	SELECT 
		r.id,
		r.start_time,
		r.end_time,
		r.review_end_time
	FROM assessuser.t_register_plan r
	WHERE r.id = $1 AND r.status NOT IN ($2,$3,$4)
`
	rows, err := pgxConn.Query(ctx, s, registerID, RegisterStatus.Cancel, RegisterStatus.Deleted, RegisterStatus.Disabled)
	defer rows.Close()
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
		err = rows.Scan(&registerId, &startTime, &endTime, &reviewEndTime)
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
func CancelRegisterTimers(ctx context.Context, registerID int64) error {
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	s := `SELECT id FROM assessuser.t_register_plan WHERE id = $1 AND status NOT IN ($2 ,$3 ,$4 )`

	//查询报名计划信息
	rows, err := pgxConn.Query(ctx, s, registerID, RegisterStatus.Cancel, RegisterStatus.Deleted, RegisterStatus.Disabled)
	defer rows.Close()
	if err != nil || forceErr == "queryRegisterPlan" {
		err = fmt.Errorf("查询报名计划信息错误")
		z.Error("查询报名计划信息失败",
			zap.Int64("register_id", registerID),
			zap.Error(err))
		return err
	}
	//收集所有场次ID并取消对应的定时器
	cancelCount := 0
	for rows.Next() {
		var registerID int64
		if err := rows.Scan(&registerID); err == nil {
			registerTimerManager.CancelTimer(EVENT_TYPE_REGISTER_END, registerID)
			registerTimerManager.CancelTimer(EVENT_TYPE_REGISTER_REVIEW_END, registerID)
			cancelCount++
		}
	}
	z.Info("成功取消报名计划定时器",
		zap.Int64("register_id", registerID),
		zap.Int("cancelled_sessions", cancelCount))
	return nil
}

// 停止所有定时器
func (tm *RegistrationTimerManager) StopAll() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	for key, timer := range tm.timers {
		timer.Stop()
		delete(tm.timers, key)
	}
	tm.cancel()
}

// 初始化现有的报名计划定时器
func InitializeRegisterTimers(ctx context.Context) error {
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	val := ctx.Value("InitializeRegisterTimers-force-error")
	if val != nil {
		forceErr = val.(string)
	}
	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("强制触动的panic错误")
		}
		if r != nil {
			z.Error("Panic recovered in InitializeRegisterTimers",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	//查询报名计划信息
	query := `
	SELECT r.id ,
	r.start_time,
	r.end_time,
	r.review_end_time
	FROM assessuser.t_register_plan r
	WHERE r. status  IN ($1 , $2 , $3)
	ORDER BY  r.id
`
	rows, err := pgxConn.Query(ctx, query, RegisterStatus.PendingRelease, RegisterStatus.Released, RegisterStatus.Ending)
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
			zap.Error(err))
		return err
	}
	//设置报名计划定时器
	for rows.Next() {
		var registerID, startTime, endTime, reviewEndTime int64
		err := rows.Scan(&registerID, &startTime, &endTime, &reviewEndTime)
		if forceErr == "scanRegisterPlan" {
			err = fmt.Errorf("扫描报名计划信息错误")
		}
		if err != nil {
			z.Error("扫描报名计划信息失败",
				zap.Error(err))
			return err
		}
		registerTimerManager.SetTimer(registerID, endTime, RegisterEvent{
			RegisterID: registerID,
			Type:       EVENT_TYPE_REGISTER_END,
		})
		registerTimerManager.SetTimer(registerID, reviewEndTime, RegisterEvent{
			RegisterID: registerID,
			Type:       EVENT_TYPE_REGISTER_REVIEW_END,
		})

	}
	return nil
}

//// 处理报名计划开始事件
//func HandleRegisterStartEvent(ctx context.Context, event RegisterEvent) error {
//	z.Info("---->" + cmn.FncName())
//	forceErr := ""
//	if val := ctx.Value("handleRegisterStartEvent-force-error"); val != nil {
//		forceErr = val.(string)
//	}
//	tx, err := pgxConn.Begin(ctx)
//	if forceErr == "beginTx" {
//		err = fmt.Errorf("强制开启事务错误")
//	}
//	if err != nil {
//		z.Error("开始事务失败", zap.Error(err))
//		return err
//	}
//	defer func() {
//		r := recover()
//		if forceErr == "panic" {
//			r = fmt.Errorf("强制panic触发")
//		}
//		if r != nil {
//			z.Error("Panic recovered in handleExamSessionStart",
//				zap.Any("panic", r),
//				zap.Stack("stack"))
//			return
//		}
//		if err != nil || forceErr == "rollback" {
//			rbErr := tx.Rollback(ctx)
//			if forceErr == "rollback" {
//				rbErr = fmt.Errorf("强制回滚错误")
//			}
//			if rbErr != nil {
//				z.Error("事务回滚失败", zap.Error(rbErr))
//			}
//			return
//		}
//		err = tx.Commit(ctx)
//		if forceErr == "commit" {
//			err = fmt.Errorf("强制提交错误")
//		}
//		if err != nil {
//			z.Error("事务提交失败", zap.Error(err))
//		}
//	}()
//	now := time.Now().UnixMilli()
//	//到开始时间的报名计划正式开始
//	var result pgconn.CommandTag
//	result, err = tx.Exec(ctx, `
//		UPDATE t_register_plan
//		SET status = '01'
//		WHERE id = $1 AND status = '02'
//	`, registerID)
//
//}

// 处理报名计划结束事件
func handleRegisterEndEvent(ctx context.Context, event RegisterEvent) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleRegisterReviewEnd-force-error"); val != nil {
		forceErr = val.(string)
	}

	tx, err := pgxConn.Begin(ctx)
	if forceErr == "beginTx" {
		err = fmt.Errorf("强制开启事务错误")
	}
	if err != nil {
		z.Error("开启事务失败", zap.Error(err))
		return err
	}

	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("强制触发的panic错误")
		}
		if r != nil {
			z.Error("Panic recovered in handleRegisterReviewEnd",
				zap.Any("panic", r),
				zap.Stack("stack"))

			if rbErr := tx.Rollback(ctx); rbErr != nil || forceErr == "panic" {
				z.Error("panic后事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		if err != nil || forceErr == "rollback" {
			rbErr := tx.Rollback(ctx)
			if forceErr == "rollback" {
				rbErr = fmt.Errorf("强制回滚错误")
			}
			if rbErr != nil {
				z.Error("事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		cmErr := tx.Commit(ctx)
		if forceErr == "commit" {
			cmErr = fmt.Errorf("强制提交错误")
		}
		if cmErr != nil {
			z.Error("事务提交失败", zap.Error(cmErr))
		}
	}()

	now := time.Now().UnixMilli()
	//更新报名计划状态为已结束
	_, err = tx.Exec(
		ctx,
		`UPDATE assessuser.t_register_plan SET status = $1,update_time= $2 WHERE id = $3 AND status IN ($4 ,$5) AND end_time <= $6`,
		RegisterStatus.Ending,
		now,
		event.RegisterID,
		RegisterStatus.Released,
		RegisterStatus.PendingRelease,
		now,
	)
	if err != nil || forceErr == "exec" {
		err = fmt.Errorf("exec failed:%v", err)
		z.Error(err.Error())
		return err
	}
	z.Info("报名计划结束事件处理",
		zap.Int64("register_id", event.RegisterID))
	return nil
}
func handleRegisterReviewEndEvent(ctx context.Context, event RegisterEvent) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleRegisterReviewEnd-force-error"); val != nil {
		forceErr = val.(string)
	}

	tx, err := pgxConn.Begin(ctx)
	if forceErr == "beginTx" {
		err = fmt.Errorf("强制开启事务错误")
	}
	if err != nil {
		z.Error("开启事务失败", zap.Error(err))
		return err
	}

	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("强制触发的panic错误")
		}
		if r != nil {
			z.Error("Panic recovered in handleRegisterReviewEnd",
				zap.Any("panic", r),
				zap.Stack("stack"))

			if rbErr := tx.Rollback(ctx); rbErr != nil || forceErr == "panic" {
				z.Error("panic后事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		if err != nil || forceErr == "rollback" {
			rbErr := tx.Rollback(ctx)
			if forceErr == "rollback" {
				rbErr = fmt.Errorf("强制回滚错误")
			}
			if rbErr != nil {
				z.Error("事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		cmErr := tx.Commit(ctx)
		if forceErr == "commit" {
			cmErr = fmt.Errorf("强制提交错误")
		}
		if cmErr != nil {
			z.Error("事务提交失败", zap.Error(cmErr))
		}
	}()

	now := time.Now().UnixMilli()
	//更新报名计划状态为审核截止
	_, err = tx.Exec(
		ctx,
		`UPDATE assessuser.t_register_plan SET status = $1,update_time= $2 WHERE id = $3 AND status IN ($4 ,$5) AND review_end_time <= $6`,
		RegisterStatus.ReviewEnding,
		now,
		event.RegisterID,
		RegisterStatus.Released,
		RegisterStatus.Ending,
		now,
	)
	if err != nil || forceErr == "exec" {
		err = fmt.Errorf("exec failed:%v", err)
		z.Error(err.Error())
		return err
	}
	z.Info("报名计划审核结束事件处理",
		zap.Int64("register_id", event.RegisterID))
	return nil
}
func RegisterMaintainService() {
	ctx, cancel := context.WithCancel(context.Background())

	registerTimerManager = NewRegistrationTimerManager(ctx, cancel)
	//初始化定时器
	err := InitializeRegisterTimers(ctx)
	if err != nil {
		z.Error("初始化定时器失败", zap.Error(err))
		cancel()
		return
	}
}
