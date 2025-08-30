package practice_mgt

//annotation:practice_mgt-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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
		Fn: practiceH,

		Path: "/practice",
		Name: "practice",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceStudentListH,

		Path: "/practiceStudentList",
		Name: "practiceStudentList",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// 整合为一个接口
func practiceH(ctx context.Context) {
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
	var domainID int64
	for _, domain := range q.Domains {
		if domain.ID.Int64 == PracticeDomainID.Student || domain.ID.Int64 == PracticeDomainID.Teacher || domain.ID.Int64 == PracticeDomainID.Admin || domain.ID.Int64 == PracticeDomainID.SuperAdmin {
			domainID = domain.ID.Int64
			break
		}
	}
	// 将所有非学生权限，都转换为教师权限
	if domainID != 0 && domainID < PracticeDomainID.Student {
		domainID = PracticeDomainID.Teacher
	}
	switch domainID {
	case PracticeDomainID.Student:
		{
			method := strings.ToLower(q.R.Method)
			if method != "get" {
				q.Err = fmt.Errorf("please call /api/practice with  http GET method")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			name := q.R.URL.Query().Get("name")
			d := q.R.URL.Query().Get("difficulty")
			t := q.R.URL.Query().Get("type")
			if t == "" {
				q.Err = fmt.Errorf("缺失练习类型")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
				q.Err = fmt.Errorf("将页面大小转化为整形失败")
			}
			if q.Err != nil {
				q.Err = fmt.Errorf("分页查询的页号解析失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			pageSizeStr := q.R.URL.Query().Get("page_size")
			if pageSizeStr == "" {
				q.Err = fmt.Errorf("缺失分页查询页大小")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var pageSize int
			pageSize, q.Err = strconv.Atoi(pageSizeStr)
			if forceErr == "pageSizeParseInt" {
				q.Err = fmt.Errorf("将分页大小转化为整形失败")
			}
			if q.Err != nil {
				q.Err = fmt.Errorf("分页页大小解析失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 排序字段
			orderBy := []string{"create_time desc"}
			p, total, err := ListPracticeS(ctx, t, name, d, orderBy, page, pageSize, userID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
			result := Map{
				"total":     total,
				"practices": p,
			}
			data, err := json.Marshal(result)
			if forceErr == "json" {
				err = fmt.Errorf("构建前端返回数据反序列化失败")
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

	case PracticeDomainID.Teacher:
		{
			method := strings.ToLower(q.R.Method)

			switch method {
			case "post":
				{
					var buf []byte
					buf, q.Err = io.ReadAll(q.R.Body)
					if forceErr == "io.ReadAll" {
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
						q.Err = fmt.Errorf("Call /api/practice with  empty body")
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
					var isClear bool
					if qry.Action == "clear" {
						isClear = true
					}

					var p practiceInfo
					q.Err = json.Unmarshal(qry.Data, &p)
					if forceErr == "readPJson" {
						q.Err = fmt.Errorf("读取前端数据practice字段结构体失败")
					}
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					q.Err = ValidatePractice(&p.Practice, p.Student)
					if q.Err != nil {
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					err := UpsertPractice(ctx, &p.Practice, p.Student, userID, isClear)
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
			case "get":
				{
					name := q.R.URL.Query().Get("name")
					pType := q.R.URL.Query().Get("type")
					status := q.R.URL.Query().Get("status")
					id := q.R.URL.Query().Get("id")
					// 此时需要获取练习具体信息
					if id != "" {
						var pid int64
						pid, q.Err = strconv.ParseInt(id, 10, 64)
						if forceErr == "idParseInt" {
							q.Err = fmt.Errorf("将字符串转化为整形失败")
						}
						if q.Err != nil {
							q.Err = fmt.Errorf("练习ID获取失败：%v", q.Err.Error())
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
						p, pName, sCount, err := LoadPracticeById(ctx, pid)
						if err != nil {
							q.Err = err
							q.RespErr()
							return
						}
						result := Map{
							"practice":      &p,
							"student_count": sCount,
							"paper_name":    pName,
						}
						data, err := json.Marshal(result)
						if forceErr == "json" {
							err = fmt.Errorf("将返回数据结构反序列化失败")
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
					pageSizeStr := q.R.URL.Query().Get("page_size")
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
					if pageSize >= 999 {
						q.Err = fmt.Errorf("页数量过大，不允许访问")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					// 排序字段
					orderBy := []string{"create_time desc"}
					p, total, err := ListPracticeT(ctx, name, pType, status, orderBy, page, pageSize, userID)
					if err != nil {
						q.Err = err
						q.RespErr()
						return
					}
					result := Map{
						"total":     total,
						"practices": p,
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
			case "patch":
				{
					// 更新练习状态，对练习进行发布、取消、删除操作
					idStr := q.R.URL.Query().Get("id")
					status := q.R.URL.Query().Get("status")

					// 这里去获取ID数组
					if idStr == "" {
						q.Err = fmt.Errorf("请传入要操作的练习ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}

					// 以逗号分隔
					idArray := strings.Split(idStr, ",")

					var ids []int64
					var invalidValues []string

					for _, s := range idArray {
						// 这里就取出每一个以逗号分隔的个体
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

					if len(invalidValues) > 0 {
						// 这里就要返回了
						q.Err = fmt.Errorf("传入的练习ID中存在非法值：%v", invalidValues)
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}

					if len(ids) == 0 {
						q.Err = fmt.Errorf("请传入有效的需要操作的练习ID")
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}

					// 批量操作练习状态V2
					q.Err = OperatePracticeStatusV2(ctx, ids, status, userID)
					if q.Err != nil {
						q.RespErr()
						return
					}
					z.Info("---->" + cmn.FncName())
					q.Msg.Status = 0
					q.Msg.Msg = "OK"
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

func practiceStudentListH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	var domainID int64
	for _, domain := range q.Domains {
		if domain.ID.Int64 == PracticeDomainID.Student || domain.ID.Int64 == PracticeDomainID.Teacher || domain.ID.Int64 == PracticeDomainID.Admin || domain.ID.Int64 == PracticeDomainID.SuperAdmin {
			domainID = domain.ID.Int64
			break
		}
	}
	if domainID != 0 && domainID == PracticeDomainID.Student {
		warn := "当前权限无法操作学生名单"
		q.Err = errors.New(warn)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	userID := q.SysUser.ID.Int64
	// 这里要进行域的处理，就是这个学生能查看谁的
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid UserID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		{
			id := q.R.URL.Query().Get("id")
			if id == "" {
				q.Err = fmt.Errorf("缺失练习ID")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var pid int64
			tResult := make([]int64, 0)
			result := make([]StudentInfo, 0)

			pid, q.Err = strconv.ParseInt(id, 10, 64)
			if q.Err != nil {
				q.Err = fmt.Errorf("练习ID解析为整形失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			s := `SELECT student_id FROM t_practice_student WHERE practice_id = $1 AND status = $2`
			sqlxDB := cmn.GetDbConn()
			rows, err := sqlxDB.QueryContext(ctx, s, pid, PracticeStudentStatus.Normal)
			if err != nil || forceErr == "query" {
				err = fmt.Errorf("query studentList in practice failed:%v", err)
				z.Error(err.Error())
				q.Err = err
				q.RespErr()
				return
			}
			defer func() {
				err = rows.Close()
				if forceErr == "Close" {
					err = fmt.Errorf("关闭数据库数据连接失败")
				}
				if err != nil {
					q.Err = err
					z.Error(err.Error())
					q.RespErr()
					return
				}
			}()
			for rows.Next() {
				var id int64
				q.Err = rows.Scan(&id)
				if forceErr == "Scan" {
					q.Err = fmt.Errorf("解析数据库数据失败")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				tResult = append(tResult, id)
			}
			// 这里已经有这个student的id数组之后，就需要获取这个数据
			if len(tResult) == 0 {
				data, _ := json.Marshal(tResult)
				q.Msg.Data = data
				z.Info("---->" + cmn.FncName())
				q.Msg.Msg = "OK but empty result"
				q.Resp()
				return
			}
			s1 := `SELECT id, account,official_name,id_card_no,mobile_phone FROM t_user WHERE id IN (?)`
			query, args, err := sqlx.In(s1, tResult)
			if forceErr == "sqlx-error" {
				err = fmt.Errorf("准备数据库语句失败")
			}
			if err != nil {
				q.Err = fmt.Errorf("prepare sqlx.In sql query failed:%v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			query = sqlx.Rebind(sqlx.DOLLAR, query)
			z.Sugar().Debugf("打印一下构建的查询用户信息语句：%v", query)
			z.Sugar().Debugf("打印一下构建的查询用户信息注入参数：%v", args)
			rows, err = sqlxDB.QueryContext(ctx, query, args...)
			if forceErr == "query1" {
				err = fmt.Errorf("执行数据库查询语句失败")
			}
			if err != nil {
				err = fmt.Errorf("查询练习中学生账号信息失败:%v", err)
				q.Err = err
				z.Error(err.Error())
				q.RespErr()
				return
			}
			defer func() {
				err = rows.Close()
				if forceErr == "Close1" {
					err = fmt.Errorf("关闭数据库数据连接失败")
				}
				if err != nil {
					q.Err = err
					z.Error(err.Error())
					q.RespErr()
					return
				}
			}()
			for rows.Next() {
				var s StudentInfo
				q.Err = rows.Scan(&s.ID, &s.Account, &s.OfficialName, &s.IdCardNo, &s.Phone)
				if forceErr == "Scan1" {
					q.Err = fmt.Errorf("解析数据库数据失败")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				result = append(result, s)
			}
			data, err := json.Marshal(result)
			if forceErr == "json" {
				err = fmt.Errorf("返回数据反序列失败")
			}
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
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
			if forceErr == "io.ReadAll" {
				q.Err = fmt.Errorf("读取连接字节流失败")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer func() {
				q.Err = q.R.Body.Close()
				if forceErr == "Close" {
					q.Err = fmt.Errorf("关闭连接字节流失败")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}()
			if len(buf) == 0 {
				q.Err = fmt.Errorf("Call /api/practiceStudentList with  empty body")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			//获取请求的结构体
			var qry cmn.ReqProto
			q.Err = json.Unmarshal(buf, &qry)
			if forceErr == "json" {
				q.Err = fmt.Errorf("反序列前端传入的json数据失败")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var p practiceStudent
			q.Err = json.Unmarshal(qry.Data, &p)
			if forceErr == "json1" {
				q.Err = fmt.Errorf("反序列前端data字段传入的json数据失败")
			}
			if q.Err != nil {
				q.Err = fmt.Errorf("反序列前端data字段传入的json数据失败:%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			//参数校验
			q.Err = cmn.Validate(p)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			rp, _, _, err := LoadPracticeById(ctx, p.Pid)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
			if rp.Status.String != PracticeStatus.Released {
				q.Err = fmt.Errorf("此时练习状态不为已发布状态，实际为：%v,无法成功调用更新学生名单", rp.Status.String)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Err = UpsertPracticeStudentV2(ctx, p.Pid, userID, p.Student)
			if q.Err != nil {
				q.RespErr()
				return
			}
			z.Info("---->" + cmn.FncName())
			q.Msg.Status = 0
			q.Msg.Msg = "OK"
			q.Resp()
			return
		}
	default:
		{
			q.Err = fmt.Errorf("please call /api/practiceStudentList with  http POST/GET method")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}
}
