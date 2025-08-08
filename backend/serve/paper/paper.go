/*
 * @Author: wusaber33
 * @Date: 2025-08-03 21:39:33
 * @LastEditors: wusaber33
 * @LastEditTime: 2025-08-07 17:02:57
 * @FilePath: \assess\backend\serve\paper\paper.go
 * @Description:
 * Copyright (c) 2025 by wusaber33, All Rights Reserved.
 */
package paper

//annotation:template-service
//author:{"name":"wuzhen","tel":"13424074477", "email":"3117398733@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jmoiron/sqlx/types"
	"w2w.io/null"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

// actionsWithResult 定义需要返回结果的操作类型集合
var actionsWithResult = map[string]bool{
	"add_question": true, // 添加试题操作需返回新试题ID
	"add_group":    true, // 添加分组操作需返回新分组ID
}

// Constants 定义HTTP请求超时时间
const (
	TIMEOUT = 5 * time.Second // HTTP请求处理超时时间
)

// Constants 定义试卷相关的业务常量
const (
	// 默认分组名称
	DefaultGroup1Name = "一、单选题"
	DefaultGroup2Name = "二、多选题"
	DefaultGroup3Name = "三、判断题"
	DefaultGroup4Name = "四、填空题"
	DefaultGroup5Name = "五、简答题"

	// 记录状态定义
	StatusNormal   = "00" // 正常状态
	StatusUnNormal = "02" // 已删除(软删除)
	StatusDeleted  = "04" // 彻底删除

	// 试卷分类
	PaperCategoryExam     = "00" // 考试试卷
	PaperCategoryPractice = "02" // 练习试卷

	// 题目类型定义
	QuestionTypeMultiChoice  = "00" // 多选题
	QuestionTypeSingleChoice = "02" // 单选题
	QuestionTypeJudgement    = "04" // 判断题
	QuestionTypeFillBlank    = "06" // 填空题
	QuestionTypeShortAnswer  = "08" // 简答题

	// 试卷难度等级
	Simple = "00" // 简单
	Medium = "02" // 中等
	Hard   = "04" // 困难

	// 默认配置项
	DefaultSuggestedDuration                                                = 120    // 默认答题时长(分钟)
	DefaultPaperName                                                        = "新建试卷" // 默认试卷名称
	PaperShareStatusPrivate, PaperShareStatusShared, PaperShareStatusPublic = "00", "02", "04"
	ManualAssemblyType                                                      = "00"

	//试卷长度限制
	MaxDescription = 500
	MaxPaperName   = 50

	//试卷编辑锁前缀
	REDIS_LOCK_PREFIX     = "paper_lock:"
	REDIS_LOCK_EXPRIATION = 5 * time.Minute
)

// 全局日志对象
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
		Fn: ManualPaper,

		Path: "/paper/manual",
		Name: "paper_manual",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: PaperList,

		Path: "/paper",
		Name: "paper",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: PaperLock,

		Path: "/paper/lock",
		Name: "paper_lock",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	//_ = cmn.AddService(&cmn.ServeEndPoint{
	//	Fn: PaperShareUsers,
	//
	//	Path: "/paper/share/users",
	//	Name: "paper_share",
	//
	//	Developer: developer,
	//	WhiteList: true,
	//
	//	//DomainID 创建该API的账号归属的domain
	//	DomainID: int64(cmn.CDomainSys),
	//
	//	//DefaultDomain 该API将默认授权给的用户
	//	DefaultDomain: int64(cmn.CDomainSys),
	//})
	//
	//_ = cmn.AddService(&cmn.ServeEndPoint{
	//	Fn: PaperShareStatus,
	//
	//	Path: "/paper/share/status",
	//	Name: "paper_status",
	//
	//	Developer: developer,
	//	WhiteList: true,
	//
	//	//DomainID 创建该API的账号归属的domain
	//	DomainID: int64(cmn.CDomainSys),
	//
	//	//DefaultDomain 该API将默认授权给的用户
	//	DefaultDomain: int64(cmn.CDomainSys),
	//})
}

// 创建试卷\更新试卷\获取试卷详情
// ManualPaper 处理手动组卷相关的HTTP请求
// 支持以下操作:
// - POST: 创建新的空白试卷
// - PUT: 更新试卷内容和结构
// - GET: 获取试卷详细信息
func ManualPaper(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)

	//强制错误，用于测试
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	switch method {
	case "post":
		// 创建新试卷
		// 1. 验证用户身份和权限
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = ErrInvalidUserID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取用户角色
		roleID := q.SysUser.Role.Int64
		if roleID <= 0 {
			q.Err = ErrInvalidRoleID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//判断用户是否有权限获取试卷列表
		var role string
		var resourceDomain string
		// 从q.Domains找到当前用户角色的角色名称
		for _, domain := range q.Domains {
			if domain.ID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				//获取资源子域
				resourceDomain = resources[0]
				//按.拆分子域，只获取.前两截，如cst.school.affair,只获取cst.school
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				}
				//获取角色，按后缀区分角色权限，如superAdmin
				role = resources[1]
				break
			}
		}

		//判断用户角色是否有权限
		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取数据库连接
		db := cmn.GetPgxConn()
		// 创建带超时的上下文
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		// 获取资源域ID
		resourceIDSql := `SELECT id FROM t_domain WHERE domain = $1`
		var resourceID int64
		q.Err = db.QueryRow(ctx, resourceIDSql, resourceDomain).Scan(&resourceID)
		// 强制错误，用于测试
		if forceError == "tx.QueryRow1-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开启事务，插入试卷和默认分组
		var tx pgx.Tx
		tx, q.Err = db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		// 强制错误，用于测试
		if forceError == "BeginTx-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 确保事务结束时正确回滚或提交
		defer func() {
			// 如果发生panic或错误，尝试回滚事务
			p := recover()
			if p != nil {
				err := tx.Rollback(ctx)
				// 强制错误，用于测试
				if forceError == "recover-err" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
					q.Err = err
				}
				return
			}
			// 强制错误，用于测试
			if forceError == "Rollback-err" {
				q.Err = errors.New(forceError)
			}
			// 如果有错误，则回滚事务
			if q.Err != nil {
				err := tx.Rollback(ctx)
				// 强制错误，用于测试
				if forceError == "Rollback-err" {
					err = errors.New(forceError)
				}
				if err != nil && err != pgx.ErrTxClosed {
					z.Error(err.Error())
				}
			}
		}()
		if forceError == "recover-err" {
			panic(errors.New(forceError))
		}

		//初始化一张空试卷SQL
		now := time.Now().UnixMilli()
		initPaperSql := `
INSERT INTO t_paper 
    (name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode,domain_id) 
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,$12,$13) 
RETURNING id`

		paper := cmn.TPaper{
			Name:              null.NewString(DefaultPaperName, true),
			AssemblyType:      null.NewString(ManualAssemblyType, true),
			Category:          null.NewString(PaperCategoryExam, true),
			Level:             null.NewString(Simple, true),
			SuggestedDuration: null.NewInt(DefaultSuggestedDuration, true),
			Tags:              types.JSONText("[]"),
			Creator:           null.IntFrom(userID),
			CreateTime:        null.IntFrom(now),
			UpdatedBy:         null.IntFrom(userID),
			UpdateTime:        null.IntFrom(now),
			Status:            null.NewString(StatusNormal, true),
			AccessMode:        null.NewString(PaperShareStatusPrivate, true),
			DomainID:          null.IntFrom(resourceID),
		}
		q.Err = tx.QueryRow(ctx, initPaperSql,
			paper.Name.String,
			paper.AssemblyType.String,
			paper.Category.String,
			paper.Level.String,
			paper.SuggestedDuration.Int64,
			paper.Tags,
			paper.Creator.Int64,
			paper.CreateTime.Int64,
			paper.UpdatedBy.Int64,
			paper.UpdateTime.Int64,
			paper.Status.String,
			paper.AccessMode.String,
			paper.DomainID.Int64,
		).Scan(&paper.ID)
		// 强制错误，用于测试
		if forceError == "tx.QueryRow2-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 准备创建默认题型分组
		groupNames := []string{
			DefaultGroup1Name, // 单选题分组
			DefaultGroup2Name, // 多选题分组
			DefaultGroup3Name, // 判断题分组
			DefaultGroup4Name, // 填空题分组
			DefaultGroup5Name, // 简答题分组
		}

		groupSql := `INSERT INTO t_paper_group 
    (paper_id, name, "order", creator, create_time, updated_by, update_time, status)
VALUES
    ($1, $2, 1, $3, $4, $3, $4, $5),
    ($1, $6, 2, $3, $4, $3, $4, $5),
    ($1, $7, 3, $3, $4, $3, $4, $5),
    ($1, $8, 4, $3, $4, $3, $4, $5),
    ($1, $9, 5, $3, $4, $3, $4, $5)
RETURNING id`
		args := []any{
			paper.ID,
			groupNames[0],
			userID,
			now,
			StatusNormal,
			groupNames[1],
			groupNames[2],
			groupNames[3],
			groupNames[4],
		}
		var rows pgx.Rows
		rows, q.Err = tx.Query(ctx, groupSql, args...)
		// 强制错误，用于测试
		if forceError == "tx.Query-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer rows.Close()
		// 扫描返回的分组ID
		groups := make([]cmn.TPaperGroup, 0, 5)
		for i := 0; rows.Next(); i++ {
			var groupID int64
			q.Err = rows.Scan(&groupID)
			if forceError == "rows.Scan-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			group := cmn.TPaperGroup{
				ID:         null.IntFrom(groupID),
				PaperID:    paper.ID,
				Name:       null.NewString(groupNames[i], true),
				Order:      null.NewInt(int64(i+1), true),
				Creator:    null.IntFrom(userID),
				CreateTime: null.IntFrom(now),
				UpdatedBy:  null.IntFrom(userID),
				UpdateTime: null.IntFrom(now),
				Status:     null.NewString(StatusNormal, true),
			}
			groups = append(groups, group)
		}
		// 检查是否有行错误
		q.Err = rows.Err()
		if forceError == "rows.Err-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error("rows error", zap.Error(q.Err))
			q.RespErr()
			return
		}
		// 提交事务
		q.Err = tx.Commit(ctx)
		if forceError == "Commit-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 返回创建结果
		result := InitialManualPaperResponse{
			Paper:  paper,
			Groups: groups,
		}
		var buf []byte
		buf, q.Err = json.Marshal(result)
		q.Msg.RowCount = 1
		q.Msg.Data = buf
		q.Resp()
	case "put":
		// 更新试卷内容
		// 获取并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取并验证用户ID
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = ErrInvalidUserID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取用户角色
		roleID := q.SysUser.Role.Int64
		if roleID <= 0 {
			q.Err = ErrInvalidRoleID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//判断用户是否有权限获取试卷列表
		var role string
		var resourceDomain string
		// 从q.Domains找到当前用户角色的角色名称
		for _, domain := range q.Domains {
			if domain.ID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				}
				role = resources[1]
				break
			}
		}

		// 检查用户角色
		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var hasPermission bool
		// 如果是超级管理员，则直接拥有权限
		if role == "superAdmin" {
			hasPermission = true
		} else {
			// 检查用户是否是试卷创建者
			hasPermission, q.Err = isPaperCreator(ctx, paperID, userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}
		// 如果不是创建者，则返回无权限错误
		if !hasPermission {
			q.Err = fmt.Errorf("无权更新试卷[ID:%d], 当前用户[ID:%d]不是试卷创建者", paperID, userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 读取请求体内容
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			err := q.R.Body.Close()
			if forceError == "R.Body.Close-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()

		// 检查请求体是否为空
		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/paper/manual with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取请求的结构体
		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取需要保存到数据库的数据
		var u UpdateManualPaperRequest
		q.Err = json.Unmarshal(qry.Data, &u)
		if forceError == "json.Unmarshal-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(u.Actions) <= 0 {
			q.Err = fmt.Errorf("更新试卷[ID:%d]时，未提供任何操作", paperID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		var results []ActionResult
		results, q.Err = updateManualPaper(dmlCtx, paperID, userID, u)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if results != nil {
			data, _ := json.Marshal(results)
			q.Msg.Data = data
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	case "get":
		// 获取试卷详情
		// 解析并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if paperID <= 0 {
			q.Err = ErrInvalidPaperID
			z.Error(q.Err.Error())
			q.RespErr()
		}

		// 获取并验证用户ID
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = ErrInvalidUserID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取用户角色
		roleID := q.SysUser.Role.Int64
		if roleID <= 0 {
			q.Err = ErrInvalidRoleID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//判断用户是否有权限获取试卷列表
		var role string
		var resourceDomain string
		// 从q.Domains找到当前用户角色的角色名称
		for _, domain := range q.Domains {
			if domain.ID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				}
				role = resources[1]
				break
			}
		}

		// 检查用户角色
		// 只有教师、超级管理员和管理员可以获取试卷详情
		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var hasPermission bool
		// 如果是超级管理员，则直接拥有权限
		if role == "superAdmin" {
			hasPermission = true
		} else {
			hasPermission, q.Err = isPaperCreator(ctx, paperID, userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		if !hasPermission {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取数据库连接
		db := cmn.GetPgxConn()
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		query := `SELECT 
    id,name,assembly_type,category,level,suggested_duration,description,tags,creator,create_time,update_time,status,total_score,question_count,groups_data
	FROM v_paper
	WHERE id = $1
	LIMIT 1`

		// 执行查询
		var paper cmn.TVPaper
		q.Err = db.QueryRow(dmlCtx, query, paperID).Scan(
			&paper.ID,
			&paper.Name,
			&paper.AssemblyType,
			&paper.Category,
			&paper.Level,
			&paper.SuggestedDuration,
			&paper.Description,
			&paper.Tags,
			&paper.Creator,
			&paper.CreateTime,
			&paper.UpdateTime,
			&paper.Status,
			&paper.TotalScore,
			&paper.QuestionCount,
			&paper.GroupsData,
		)
		if forceError == "tx.QueryRow-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 包装响应体
		data, _ := json.Marshal(paper)
		q.Msg.Data = data
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
	default:
		// 默认操作，返回错误信息
		q.Err = fmt.Errorf("不支持该方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()

}

// 更新试卷流程
func updateManualPaper(ctx context.Context, paperID, userID int64, req UpdateManualPaperRequest) (results []ActionResult, err error) {
	// 获取日志记录器
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	// 获取日志记录器
	sqlxDB := cmn.GetPgxConn()
	// 启动事务
	var tx pgx.Tx
	tx, err = sqlxDB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	// 如果强制错误为"BeginTx-err"，则模拟事务启动错误
	if forceError == "BeginTx-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("start transaction failed: " + err.Error())
		return
	}

	defer func() {
		// 如果发生panic或错误，尝试回滚事务
		if p := recover(); p != nil {
			err = errors.New(fmt.Sprint(p))
			rollbackErr := tx.Rollback(ctx)
			if forceError == "recover-err" {
				rollbackErr = errors.New(forceError)
				err = rollbackErr
			}
			if rollbackErr != nil {
				z.Error(rollbackErr.Error())
				return
			}
		}
		// 如果有错误，尝试回滚事务，否则提交事务
		if err != nil {
			// 回滚事务
			rollbackErr := tx.Rollback(ctx)
			if forceError == "Rollback-err" {
				rollbackErr = errors.New(forceError)
				err = rollbackErr
			}
			// 如果回滚失败，记录错误
			if rollbackErr != nil {
				z.Error(err.Error())
			}
		}
		// 提交事务
		commitErr := tx.Commit(ctx)
		if forceError == "Commit-err" {
			commitErr = errors.New(forceError)
			err = commitErr
		}
		// 如果提交失败，记录错误
		if commitErr != nil {
			z.Error(commitErr.Error())
		}
	}()
	if forceError == "recover-err" {
		panic(errors.New(forceError))
	}
	// 执行请求的操作
	for _, act := range req.Actions {
		// 存储操作结果
		var result interface{}

		// 检查操作类型
		switch act.Action {
		// 更新试卷基本信息
		case "update_info":
			// 解析基本信息
			var basicInfo UpdatePaperBasicInfoRequest
			err = json.Unmarshal(act.Payload, &basicInfo)
			if err != nil {
				z.Error("failed to unmarshal basic info payload: " + err.Error())
				return
			}
			// 验证基本信息
			if basicInfo.Name != "" && len(basicInfo.Name) > MaxPaperName {
				err = fmt.Errorf("试卷名称长度超出限制: %d", len(basicInfo.Name))
				z.Error(err.Error())
				return
			}
			if basicInfo.Category != "" && basicInfo.Category != PaperCategoryExam && basicInfo.Category != PaperCategoryPractice {
				err = fmt.Errorf("试卷分类不合法: %s", basicInfo.Category)
				z.Error(err.Error())
				return
			}
			if basicInfo.Level != "" && basicInfo.Level != Simple && basicInfo.Level != Medium && basicInfo.Level != Hard {
				err = fmt.Errorf("试卷难度不合法: %s", basicInfo.Level)
				z.Error(err.Error())
				return
			}
			if basicInfo.Duration != nil && *basicInfo.Duration < 0 {
				err = fmt.Errorf("试卷建议时长不能小于0: %d", *basicInfo.Duration)
				z.Error(err.Error())
				return
			}
			if basicInfo.Description != nil && len(*basicInfo.Description) > MaxDescription {
				err = fmt.Errorf("试卷描述长度超出限制: %d", len(*basicInfo.Description))
				z.Error(err.Error())
				return
			}

			// 构建更新语句
			var (
				setClauses []string
				params     []interface{}
				paramIndex = 1
			)

			// 添加基本信息字段到更新语句
			// 使用闭包函数来简化添加字段的逻辑
			addField := func(condition bool, field string, value interface{}) {
				if condition {
					setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
					params = append(params, value)
					paramIndex++
				}
			}
			// 添加条件和字段到更新语句
			addField(basicInfo.Name != "", "name", basicInfo.Name)
			addField(basicInfo.Category != "", "category", basicInfo.Category)
			addField(basicInfo.Level != "", "level", basicInfo.Level)
			addField(basicInfo.Duration != nil, "suggested_duration", basicInfo.Duration)
			addField(basicInfo.Description != nil, "description", basicInfo.Description)

			// 添加Tags字段
			if basicInfo.Tags != nil {
				// 将Tags转换为JSON格式
				var jsonTags []byte
				jsonTags, err = json.Marshal(basicInfo.Tags)
				if forceError == "tag-json.Marshal-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error("failed to marshal tags: " + err.Error())
					return
				}
				addField(true, "tags", jsonTags)
			}

			//更新更新者与更新时间
			addField(true, "updated_by", userID)
			addField(true, "update_time", time.Now().UnixMilli())

			// 如果只有系统字段被更新，说明用户什么都没改，直接返回
			if len(setClauses) <= 2 {
				return
			}

			//添加where条件
			whereClause := fmt.Sprintf("WHERE id = $%d", paramIndex)
			params = append(params, paperID)

			sqlStr := fmt.Sprintf("UPDATE t_paper SET %s %s", strings.Join(setClauses, ", "), whereClause)
			_, err = tx.Exec(ctx, sqlStr, params...)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error("failed to update paper info: " + err.Error())
				return
			}
		case "add_group":
			// 解析请求体
			var req AddQuestionGroupRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			// 验证请求体
			if req.Name != "" && len(req.Name) > 100 {
				err = fmt.Errorf("题组名称长度超出限制: %d", len(req.Name))
				z.Error(err.Error())
				return
			}
			if req.Order < 0 {
				err = fmt.Errorf("题组顺序不能小于0: %d", req.Order)
				z.Error(err.Error())
				return
			}
			// 检查题组名称是否重复
			var exists bool
			err = tx.QueryRow(ctx, `SELECT EXISTS (
				SELECT 1 FROM t_paper_group 
				WHERE paper_id = $1 AND name = $2
			)`, paperID, req.Name).Scan(&exists)
			if forceError == "handleAddGroup-tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			if exists {
				err = fmt.Errorf("题组名称已存在: %s", req.Name)
				z.Error(err.Error())
				return
			}
			now := time.Now().UnixMilli()
			const batchInsertPaperQuestionGroupsSQL = `INSERT INTO t_paper_group 
    (paper_id, name, "order", creator, create_time, updated_by, update_time, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

			err = tx.QueryRow(ctx, batchInsertPaperQuestionGroupsSQL,
				paperID,
				req.Name,
				req.Order,
				userID,
				now,
				userID,
				now,
				StatusNormal,
			).Scan(&result)
			if forceError == "tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		case "delete_group":
			//解析结构体
			var groupID int64
			err = json.Unmarshal(act.Payload, &groupID)
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 验证题组ID
			if groupID <= 0 {
				err = ErrEmptyGroupID
				z.Error(ErrEmptyGroupID.Error())
				return
			}
			// 检查题组是否存在且属于该试卷
			var exists bool
			err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM t_paper_group 
			WHERE id = $1 AND paper_id = $2
		)
	`, groupID, paperID).Scan(&exists)
			if forceError == "handleDeleteGroup-tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			if !exists {
				err = fmt.Errorf("question group not found in current paper: %s", ErrRecordNotFound.Error())
				z.Error(err.Error())
				return
			}
			// 删除题组
			_, err = tx.Exec(ctx, `
		DELETE FROM t_paper_group WHERE id = $1 AND paper_id = $2
	`, groupID, paperID)
			if forceError == "handleDeleteGroup-tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		case "add_question":
			// 解析请求体
			var req []AddQuestionsRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			// 验证请求体
			const batchInsertPaperQuestionsSQL = `INSERT INTO t_paper_question 
    (bank_question_id, group_id, "order", score,sub_score, creator, create_time, updated_by, update_time, status) 
VALUES %s RETURNING id`
			// 生成占位符和参数
			var placeholders []string
			var args []interface{}
			paramIndex := 1
			now := time.Now().UnixMilli()
			// 生成插入语句的占位符和参数
			for _, q := range req {
				// 验证题目数据
				if q.BankQuestionID <= 0 {
					err = fmt.Errorf("题库题目ID不能为空或小于等于0: %d", q.BankQuestionID)
					z.Error(err.Error())
					return
				}
				if q.GroupID <= 0 {
					err = fmt.Errorf("题组ID不能小于等于0: %d", q.GroupID)
					z.Error(err.Error())
					return
				}
				if q.Order <= 0 {
					err = fmt.Errorf("题目顺序不能小于等于0: %d", q.Order)
					z.Error(err.Error())
					return
				}
				if q.Score <= 0 {
					err = fmt.Errorf("题目分数不能小于等于0: %f", q.Score)
					z.Error(err.Error())
					return
				}
				if len(q.SubScore) > 0 && q.Type != QuestionTypeShortAnswer && q.Type != QuestionTypeFillBlank {
					err = fmt.Errorf("题目小题分数只能在简答题和填空题中使用: %s", q.Type)
					z.Error(err.Error())
					return
				}
				if len(q.SubScore) > 0 && (q.Type == QuestionTypeShortAnswer || q.Type == QuestionTypeFillBlank) {
					for _, sub := range q.SubScore {
						if sub <= 0 {
							err = fmt.Errorf("题目小题分数不能小于等于0: %f", sub)
							z.Error(err.Error())
							return
						}
					}
				}
				placeholders = append(placeholders, fmt.Sprintf(
					"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
					paramIndex, paramIndex+1, paramIndex+2, paramIndex+3, paramIndex+4,
					paramIndex+5, paramIndex+6, paramIndex+7, paramIndex+8, paramIndex+9,
				))
				args = append(args,
					q.BankQuestionID, q.GroupID, q.Order, q.Score, q.SubScore,
					userID, now, userID, now, StatusNormal,
				)
				paramIndex += 10
			}

			// 使用参数化查询
			query := fmt.Sprintf(batchInsertPaperQuestionsSQL,
				strings.Join(placeholders, ","))
			var rows pgx.Rows
			rows, err = tx.Query(ctx, query, args...)
			defer rows.Close()
			if forceError == "tx.Query-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			// 直接构建临时ID到数据库ID的映射
			idMapping := make(map[string]int64, len(req))
			i := 0
			// 处理数据库返回的自增ID并构建映射关系
			for rows.Next() {
				var id int64
				err = rows.Scan(&id)
				if forceError == "rows.Scan-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error("error occurred while scanning returned ID", zap.Error(err))
					return nil, fmt.Errorf("扫描返回ID失败 [错误:%v]", err)
				}

				// 直接将数据库返回的ID与请求中的临时ID对应
				idMapping[req[i].TempID] = id
				i++
			}

			// 检查迭代过程中的错误
			err = rows.Err()
			if forceError == "rows.Err-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error("error occurred while getting returned ID", zap.Error(err))
				return nil, fmt.Errorf("获取返回ID过程出错 [错误:%v]", err)
			}
			// 将ID映射结果存储到结果中
			result = idMapping
		case "delete_question":
			//解析结构体
			var questionIDs []int64
			err = json.Unmarshal(act.Payload, &questionIDs)
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 验证题目ID
			err = validateIDs(questionIDs)
			if err != nil {
				z.Error(err.Error())
				return
			}
			const sql = `
DELETE FROM t_paper_question tpq 
	WHERE tpq.id = ANY($1::bigint[])`
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sql, questionIDs)
			// 如果强制错误为"tx.Exec-err"，则模拟执行错误
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 检查是否有行被删除
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "update_question":
			//解析结构体
			var reqs []UpdatePaperQuestionRequest
			if err = json.Unmarshal(act.Payload, &reqs); err != nil {
				z.Error(err.Error())
				return
			}
			// 验证请求体并构建更新语句
			for _, update := range reqs {
				//检查结构体
				if update.ID <= 0 {
					err = fmt.Errorf("题目ID不能为空或小于等于0: %d", update.ID)
					z.Error(err.Error())
					return
				}
				if len(update.SubScore) > 0 {
					// 验证小题分数是否为正数
					for _, sub := range update.SubScore {
						if sub <= 0 {
							err = fmt.Errorf("题目小题分数不能小于0: %f", sub)
							z.Error(err.Error())
							return
						}
					}
				}
				// 验证题目ID
				var setClauses []string
				var args []interface{}
				argIndex := 1 // 参数索引从1开始（公共字段从第3个参数开始）

				// 动态构建 SET 子句：仅包含非空字段
				// 题组ID
				if update.GroupID > 0 {
					setClauses = append(setClauses, "group_id = $"+strconv.Itoa(argIndex))
					args = append(args, update.GroupID)
					argIndex++
				}
				// 题目分数
				if update.Score > 0 {
					setClauses = append(setClauses, "score = $"+strconv.Itoa(argIndex))
					args = append(args, update.Score)
					argIndex++
				}

				// 题目小题分数
				if len(update.SubScore) > 0 {
					subScore, _ := json.Marshal(update.SubScore)
					setClauses = append(setClauses, "sub_score = $"+strconv.Itoa(argIndex))
					args = append(args, subScore)
					argIndex++
				}

				// 公共字段：必须更新（即使其他字段为空）
				setClauses = append(setClauses, "updated_by = $"+strconv.Itoa(argIndex))
				args = append(args, userID)
				argIndex++
				setClauses = append(setClauses, "update_time = $"+strconv.Itoa(argIndex))
				args = append(args, time.Now().UnixMilli())
				argIndex++

				// 如果只有系统字段被更新，说明用户什么都没改，直接返回
				if len(setClauses) <= 2 {
					err = fmt.Errorf("更新题目失败: 没有需要更新的字段或所有字段都为零值")
					z.Error(err.Error())
					return
				}

				// 构建完整 SQL
				query := fmt.Sprintf(
					"UPDATE t_paper_question SET %s WHERE id = $%d",
					strings.Join(setClauses, ", "),
					argIndex,
				)
				args = append(args, update.ID)

				// 执行更新
				var commandTag pgconn.CommandTag
				commandTag, err = tx.Exec(ctx, query, args...)
				if forceError == "tx.Exec-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error(err.Error())
					return
				}
				// 检查是否有行被更新
				rowsAffected := commandTag.RowsAffected()
				if rowsAffected == 0 {
					z.Error(ErrRecordNotFound.Error())
					return nil, ErrRecordNotFound
				}
			}
		case "update_group":
			//解析结构体
			var req UpdateQuestionsGroupRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			// 验证请求体
			if req.ID <= 0 {
				err = fmt.Errorf("题组ID不能为空或小于等于0: %d", req.ID)
				z.Error(err.Error())
				return
			}
			if req.Name == "" {
				err = fmt.Errorf("题组名称不能为空")
				z.Error(err.Error())
				return
			}
			if len(req.Name) > 100 {
				err = fmt.Errorf("题组名称长度超出限制: %d", len(req.Name))
				z.Error(err.Error())
				return
			}
			// 验证题组名称是否重复
			var exists bool
			err = tx.QueryRow(ctx, `SELECT EXISTS (
				SELECT 1 FROM t_paper_group 
				WHERE paper_id = $1 AND name = $2 AND id != $3
			)`, paperID, req.Name, req.ID).Scan(&exists)
			if forceError == "handleUpdateGroup-tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			if exists {
				err = fmt.Errorf("题组名称已存在: %s", req.Name)
				z.Error(err.Error())
				return
			}
			// 构建更新语句
			sql := `UPDATE t_paper_group
SET name = $1, updated_by = $2, update_time = $3
WHERE id = $4`
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sql, req.Name, userID, time.Now().UnixMilli(), req.ID)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 检查是否有行被更新
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "move_question":
			//解析结构体
			var orders []int64
			err = json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 验证题目ID
			if err = validateIDs(orders); err != nil {
				return nil, err
			}
			// 查询实际题组数量
			var actualCount int
			err = tx.QueryRow(ctx, `
        SELECT question_count FROM v_paper 
        WHERE id = $1 AND status != '02'`,
				paperID).Scan(&actualCount)
			if forceError == "tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 验证数量
			if len(orders) != actualCount {
				err = fmt.Errorf("题目数量不匹配: 输入了%d个ID, 但实际只找到%d个题目", len(orders), actualCount)
				z.Error(err.Error())
				return
			}
			now := time.Now().UnixMilli()
			// 构建批量更新SQL
			sqlStr := `
	    UPDATE t_paper_question pq
		SET 
			"order" = o.new_order,
			updated_by = $2,
			update_time = $3
		FROM (
		    SELECT id,ordinality as new_order
			FROM unnest($1::bigint[]) WITH ORDINALITY as arr(id, ordinality)
			)o
		WHERE pq.id = o.id;`
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sqlStr, orders, userID, now)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 检查是否有行被更新
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "move_group":
			//解析结构体
			var orders []int64
			err = json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 验证题组ID
			if err = validateIDs(orders); err != nil {
				return
			}
			// 1. 查询实际题组数量
			var actualCount int
			err = tx.QueryRow(ctx, `
        SELECT group_count FROM v_paper 
        WHERE id = $1 AND status != '02'`,
				paperID).Scan(&actualCount)
			if forceError == "tx.QueryRow-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 2. 验证数量
			if len(orders) != actualCount {
				err = fmt.Errorf("题组数量不匹配: 输入了%d个ID, 但实际只找到%d个题组", len(orders), actualCount)
				z.Error(err.Error())
				return
			}
			now := time.Now().UnixMilli()
			// 构建批量更新SQL
			sqlStr := `
	    UPDATE t_paper_group pg
		SET 
			"order" = o.new_order,
			updated_by = $2,
			update_time = $3
		FROM (
		    SELECT id,ordinality as new_order
			FROM unnest($1::bigint[]) WITH ORDINALITY as arr(id, ordinality)
			)o
		WHERE pg.id = o.id;`
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sqlStr, orders, userID, now)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			// 检查是否有行被更新
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		default:
			err = fmt.Errorf("unsupported action type: %s", act.Action)
			z.Error(err.Error())
			return
		}

		// 只有指定的操作需要返回结果
		if actionsWithResult[act.Action] {
			results = append(results, ActionResult{
				Action: act.Action,
				Result: result,
			})
		}
	}

	return
}

// 试卷首页  列表获取\删除试卷
// PaperList 处理试卷列表相关的HTTP请求
// 支持以下操作:
// - GET: 分页获取试卷列表,支持按名称、状态等条件筛选
// - DELETE: 批量删除试卷
func PaperList(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		//获取用户ID
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = ErrInvalidUserID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取用户角色
		roleID := q.SysUser.Role.Int64
		if roleID <= 0 {
			q.Err = fmt.Errorf("invalid role: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//判断用户是否有权限获取试卷列表
		var role string
		var resourceDomain string
		// 从q.Domains找到当前用户角色的角色名称
		for _, domain := range q.Domains {
			if domain.ID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				}
				role = resources[1]
				break
			}
		}

		// 检查用户角色
		// 只有教师、超级管理员和管理员可以获取试卷列表
		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取数据库连接
		db := cmn.GetPgxConn()
		// 创建带超时的上下文
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		resourceIDSql := `SELECT id FROM t_domain WHERE domain = $1`
		var resourceID int64
		q.Err = db.QueryRow(dmlCtx, resourceIDSql, resourceDomain).Scan(&resourceID)
		if forceError == "tx.QueryRow-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//创建请求体并绑定参数
		var req PaperListRequest
		queryParams := q.R.URL.Query()

		// 解析查询参数
		if name := queryParams.Get("name"); name != "" {
			req.Name = name
		}
		if tags := queryParams.Get("tags"); tags != "" {
			req.Tags = tags
		}
		if category := queryParams.Get("category"); category != "" {
			req.Category = category
		}
		req.Page = 1
		if page := queryParams.Get("page"); page != "" {
			if p, err := strconv.Atoi(page); err == nil {
				req.Page = p
			}
		}
		req.PageSize = 10
		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
			if p, err := strconv.Atoi(pageSize); err == nil {
				req.PageSize = p
			}
		}

		//req.Self = false
		//if self := queryParams.Get("self"); self != "" {
		//	if s, err := strconv.ParseBool(self); err == nil {
		//		req.Self = s
		//	}
		//}

		// 参数校验
		if req.Page <= 0 {
			q.Err = fmt.Errorf("页数小于等于0: %d", req.Page)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if req.PageSize != 5 && req.PageSize != 10 && req.PageSize != 20 {
			q.Err = fmt.Errorf("无效的页大小: %d", req.PageSize)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if req.Category != PaperCategoryExam && req.Category != PaperCategoryPractice && req.Category != "" {
			q.Err = fmt.Errorf("无效的试卷分类: %s", req.Category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if req.Name != "" && len(req.Name) > MaxPaperName {
			q.Err = fmt.Errorf("查询试卷名称过长，最大长度为: %d", MaxPaperName)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 构建查询条件
		var totalCount int64
		offset := (req.Page - 1) * req.PageSize
		// 构建动态查询条件
		var whereClauses []string
		var params []interface{}
		paramCount := 1

		// 基础条件：状态为有效
		whereClauses = append(whereClauses, "p.status = '00'")

		// 资源范围
		var resourceClause strings.Builder
		resourceClause.WriteString("p.domain_id = $")
		resourceClause.WriteString(strconv.Itoa(paramCount))
		whereClauses = append(whereClauses, resourceClause.String())
		params = append(params, resourceID)
		paramCount++
		//// 如果设置了self，则只查询当前用户创建的试卷
		//if req.Self {
		//	var creatorClause strings.Builder
		//	creatorClause.WriteString("p.creator = $")
		//	creatorClause.WriteString(strconv.Itoa(paramCount))
		//	whereClauses = append(whereClauses, creatorClause.String())
		//	params = append(params, userID)
		//	paramCount++
		//}

		//// 权限控制
		//accessControlClause := fmt.Sprintf(`(
		//	p.creator = $%d
		//	OR p.access_mode = '04'
		//	OR (
		//		p.access_mode = '02'
		//		AND EXISTS (
		//			SELECT 1 FROM t_resource_share s
		//			WHERE s.type = $%d AND s.resource_id = p.id AND s.user_id = $%d AND s.status = '00'
		//		)
		//	)
		//)`, paramCount, paramCount+1, paramCount+2)
		//whereClauses = append(whereClauses, accessControlClause)
		//params = append(params, userID, PaperResourceShareType, userID)
		//paramCount += 3

		// 用途精确查询
		if req.Category != "" {
			var categoryClause strings.Builder
			categoryClause.WriteString("p.category = $")
			categoryClause.WriteString(strconv.Itoa(paramCount))
			whereClauses = append(whereClauses, categoryClause.String())
			params = append(params, req.Category)
			paramCount++
		}

		// 名称模糊查询
		if req.Name != "" {
			var nameClause strings.Builder
			nameClause.WriteString("p.name ILIKE $")
			nameClause.WriteString(strconv.Itoa(paramCount))
			whereClauses = append(whereClauses, nameClause.String())
			params = append(params, "%"+req.Name+"%")
			paramCount++
		}

		// 标签过滤
		var tags []string
		if req.Tags != "" {
			tags = strings.Split(req.Tags, ",")
			var cleanedTags []string
			for _, tag := range tags {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					cleanedTags = append(cleanedTags, trimmedTag)
				}
			}
			tags = cleanedTags
		}
		// 如果tags不为空，则添加到查询条件
		if len(tags) > 0 {
			var tagsClause strings.Builder
			tagsClause.WriteString("p.tags @> $")
			tagsClause.WriteString(strconv.Itoa(paramCount))
			whereClauses = append(whereClauses, tagsClause.String())
			tagsJSON, _ := json.Marshal(tags)
			params = append(params, tagsJSON)
			paramCount++
		}

		// 构建WHERE子句
		var whereClause string
		if len(whereClauses) > 0 {
			whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
		}

		// 查询总数
		var countSQLBuilder strings.Builder
		countSQLBuilder.WriteString("SELECT COUNT(*) FROM v_paper p ")
		countSQLBuilder.WriteString(whereClause)
		q.Err = db.QueryRow(ctx, countSQLBuilder.String(), params...).Scan(&totalCount)
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRowCount-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 查询分页数据
		var listSQLBuilder strings.Builder
		listSQLBuilder.WriteString(`
		SELECT p.id, p.name, p.assembly_type, p.category, p.level, p.suggested_duration, p.total_score, p.question_count, p.tags, p.create_time, p.update_time, p.status, p.creator, p.creator_info, p.access_mode
		FROM v_paper p
		`)
		listSQLBuilder.WriteString(whereClause)
		listSQLBuilder.WriteString(`
		ORDER BY p.update_time DESC
		LIMIT $`)
		listSQLBuilder.WriteString(strconv.Itoa(paramCount))
		listSQLBuilder.WriteString(" OFFSET $")
		listSQLBuilder.WriteString(strconv.Itoa(paramCount + 1))
		dataParams := append(params, req.PageSize, offset)
		var rows pgx.Rows

		rows, q.Err = db.Query(dmlCtx, listSQLBuilder.String(), dataParams...)
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRow-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer rows.Close()
		var papers []cmn.TVPaper
		for rows.Next() {
			var paper cmn.TVPaper
			q.Err = rows.Scan(&paper.ID, &paper.Name, &paper.AssemblyType, &paper.Category, &paper.Level, &paper.SuggestedDuration, &paper.TotalScore, &paper.QuestionCount, &paper.Tags, &paper.CreateTime, &paper.UpdateTime, &paper.Status, &paper.Creator, &paper.CreatorInfo, &paper.AccessMode)
			if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-RowScan-err" {
				q.Err = errors.New(val)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			papers = append(papers, paper)
		}
		q.Err = rows.Err()
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-RowErr-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 返回结果
		data, _ := json.Marshal(papers)
		q.Msg.Data = data
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.RowCount = totalCount
		q.Msg.Msg = "success"
		q.Resp()
	case "delete":
		// 获取并验证用户ID
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = ErrInvalidUserID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取用户角色
		roleID := q.SysUser.Role.Int64
		if roleID <= 0 {
			q.Err = ErrInvalidRoleID
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//判断用户是否有权限获取试卷列表
		var role string
		var resourceDomain string
		// 从q.Domains找到当前用户角色的角色名称
		for _, domain := range q.Domains {
			if domain.ID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				}
				role = resources[1]
				break
			}
		}

		// 检查用户角色
		// 只有教师、超级管理员和管理员可以删除试卷
		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = ErrWithoutPermission
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 读取请求体
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "PaperList-delete-io.ReadAll-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			err := q.R.Body.Close()
			if forceError == "PaperList-delete-Body.Close-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()

		// 检查请求体是否为空
		// 如果请求体为空，则返回错误
		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/paper/manual by delete with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取请求的结构体
		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceError == "PaperList-delete-json.Unmarshal1-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取需要保存到数据库的数据
		var paperIDs []int64
		q.Err = json.Unmarshal(qry.Data, &paperIDs)
		if forceError == "PaperList-delete-json.Unmarshal2-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//参数校验
		q.Err = validateIDs(paperIDs)
		if q.Err != nil {
			q.Err = fmt.Errorf("invalid paper IDs: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查试卷ID是否为空
		db := cmn.GetPgxConn()
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		// 获取资源ID
		resourceIDSql := `SELECT id FROM t_domain WHERE domain = $1`
		var resourceID int64
		q.Err = db.QueryRow(ctx, resourceIDSql, resourceDomain).Scan(&resourceID)
		if forceError == "tx.QueryRow-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开启事务
		var tx pgx.Tx
		tx, q.Err = db.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.RepeatableRead,
		})
		if forceError == "PaperList-delete-BeginTx-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 确保事务结束时回滚
		defer func() {
			if p := recover(); p != nil {
				err := tx.Rollback(ctx)
				if forceError == "PaperList-delete-Rollback-panic" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
					return
				}
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if forceError == "PaperList-delete-Rollback-err" {
					err = errors.New(forceError)
				}
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					z.Error(err.Error())
					return
				}
			}
			// 提交事务
			err := tx.Commit(ctx)
			if forceError == "PaperList-delete-Commit-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()
		if forceError == "PaperList-delete-Rollback-panic" {
			panic(errors.New(forceError))
		}

		// 检查每个试卷的权限
		var checkSQL string
		var errorMessages []string
		if role == "superAdmin" {
			// 管理员检查试卷存在性和域
			checkSQL = `
				SELECT array_agg(
					CASE 
						WHEN p.id IS NULL THEN '试卷 "' || ids.id || '" 不存在'
						WHEN p.domain_id != $2 THEN '试卷 "' || COALESCE(p.name, '未知') || '" 不在当前域范围内'
						ELSE NULL 
					END
				) as error_messages
				FROM unnest($1::bigint[]) AS ids(id)
				LEFT JOIN t_paper p ON p.id = ids.id
				WHERE p.id IS NULL OR p.domain_id != $2`
			q.Err = tx.QueryRow(ctx, checkSQL, paperIDs, resourceID).Scan(&errorMessages)
			if forceError == "superAdmin-tx.QueryRow-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		} else {
			// 普通用户检查试卷存在性、域和创建者
			checkSQL = `
				SELECT array_agg(
					CASE 
						WHEN p.id IS NULL THEN '试卷 "' || ids.id || '" 不存在'
						WHEN p.domain_id != $2 THEN '试卷 "' || COALESCE(p.name, '未知') || '" 不在当前域范围内'
						WHEN p.creator != $3 THEN '试卷 "' || COALESCE(p.name, '未知') || '" 非试卷创建者，无删除权限'
						ELSE NULL 
					END
				) as error_messages
				FROM unnest($1::bigint[]) AS ids(id)
				LEFT JOIN t_paper p ON p.id = ids.id
				WHERE p.id IS NULL OR p.domain_id != $2 OR p.creator != $3`
			q.Err = tx.QueryRow(ctx, checkSQL, paperIDs, resourceID, userID).Scan(&errorMessages)
			if forceError == "normaluser-tx.QueryRow-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 移除空错误消息并在每个错误前添加换行符
		var validErrors strings.Builder
		for i, msg := range errorMessages {
			if msg != "" {
				// 不是第一个错误时，先添加换行符
				if i > 0 {
					validErrors.WriteString("\n")
				}
				validErrors.WriteString(msg)
			}
		}

		// 如果有任何不能删除的试卷，返回错误
		if validErrors.Len() > 0 {
			q.Msg.Msg = validErrors.String()
			q.Msg.Status = -1
			q.Err = errors.New("部分试卷无法删除")
			return
		}
		now := time.Now().UnixMilli()
		// 1. 软删除 t_paper
		paperSQL := `UPDATE t_paper SET status = $2, updated_by = $3, update_time = $4 WHERE id = ANY($1)`
		_, q.Err = tx.Exec(dmlCtx, paperSQL, paperIDs, StatusUnNormal, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapers-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 2. 软删除 t_paper_group
		groupSQL := `UPDATE t_paper_group SET status = $2, updated_by = $3, update_time = $4 WHERE paper_id = ANY($1)`
		_, q.Err = tx.Exec(ctx, groupSQL, paperIDs, StatusUnNormal, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersgroups-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 3. 软删除 t_paper_question
		questionSQL := `UPDATE t_paper_question SET status = $2, updated_by = $3, update_time = $4 WHERE group_id IN (SELECT id FROM t_paper_group WHERE paper_id = ANY($1))`
		_, q.Err = tx.Exec(ctx, questionSQL, paperIDs, StatusUnNormal, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersquestions-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Status = 0
		q.Msg.Msg = "success"
	default:
		// 处理其他方法
		q.Err = fmt.Errorf("不支持该方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

// 试卷锁 获取锁\延长锁\释放锁
// PaperLock 处理试卷编辑锁相关的HTTP请求
func PaperLock(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	//forceError := ""
	//if val, ok := ctx.Value("force-error").(string); ok {
	//	forceError = val
	//}

	//获取用户ID
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = ErrInvalidUserID
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 解析并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if paperID <= 0 {
			q.Err = ErrInvalidPaperID
			z.Error(q.Err.Error())
			q.RespErr()
		}

		//尝试获取试卷锁REDIS_LOCK_PREFIX
		var success bool
		success, q.Err = cmn.TryLock(ctx, paperID, userID, REDIS_LOCK_PREFIX, REDIS_LOCK_EXPRIATION)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if !success {
			q.Err = fmt.Errorf("当前试卷正在被其他用户编辑")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	case "put":
		// 解析并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if paperID <= 0 {
			q.Err = ErrInvalidPaperID
			z.Error(q.Err.Error())
			q.RespErr()
		}

		//尝试获取试卷锁REDIS_LOCK_PREFIX
		q.Err = cmn.RefreshLock(ctx, paperID, userID, REDIS_LOCK_PREFIX, REDIS_LOCK_EXPRIATION)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	case "delete":
		// 解析并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if paperID <= 0 {
			q.Err = ErrInvalidPaperID
			z.Error(q.Err.Error())
			q.RespErr()
		}

		//尝试获取试卷锁REDIS_LOCK_PREFIX
		q.Err = cmn.ReleaseLock(ctx, paperID, userID, REDIS_LOCK_PREFIX)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	default:
		// 处理其他方法
		q.Err = fmt.Errorf("不支持该方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

// isPaperCreator 检查用户是否为试卷的创建者
func isPaperCreator(ctx context.Context, paperID, userID int64) (bool, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	// 获取数据库连接
	db := cmn.GetPgxConn()
	var isCreator bool
	err := db.QueryRow(ctx, `
	SELECT EXISTS (
		SELECT 1 FROM t_paper 
		WHERE id = $1 AND creator = $2 AND status != '02'
	)`, paperID, userID).Scan(&isCreator)
	if forceError == "isPaperCreator-QueryRow-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("failed to check if user is paper creator: " + err.Error())
		return false, err
	}
	return isCreator, nil
}

//// 试卷共享
//func PaperShareUsers(ctx context.Context) {
//	q := cmn.GetCtxValue(ctx)
//	z.Info("---->" + cmn.FncName())
//
//	method := strings.ToLower(q.R.Method)
//	switch method {
//	case "get":
//		//创建请求体并绑定参数
//		var req GetSharedUserListRequest
//		queryParams := q.R.URL.Query()
//		req.Page = 1
//		if page := queryParams.Get("page"); page != "" {
//			if p, err := strconv.Atoi(page); err == nil {
//				req.Page = p
//			}
//		}
//		req.PageSize = 10
//		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
//			if p, err := strconv.Atoi(pageSize); err == nil {
//				req.PageSize = p
//			}
//		}
//		//获取试卷ID
//		paperIDStr := q.R.URL.Query().Get("paper_id")
//		var paperID int64
//		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取过滤值
//		if filter := q.R.URL.Query().Get("filter"); filter != "" {
//			req.Filter = filter
//		}
//		var userID int64 =1574
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//
//		//获取用户ID
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, paperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//		var shared_users []cmn.TVPaperShare
//		var rouCount int64
//		shared_users, rouCount, q.Err = getPaperShareInfo(dmlCtx, tx, paperID, req)
//		if q.Err != nil {
//			q.RespErr()
//			return
//		}
//		if len(shared_users) != 0 {
//			data, _ := json.Marshal(shared_users)
//			q.Msg.Data = data
//			q.Msg.RowCount = rouCount
//		}
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	case "post":
//		var buf []byte
//		buf, q.Err = io.ReadAll(q.R.Body)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		defer func() {
//			q.Err = q.R.Body.Close()
//			if q.Err != nil {
//				z.Error(q.Err.Error())
//				q.RespErr()
//				return
//			}
//		}()
//
//		if len(buf) == 0 {
//			q.Err = fmt.Errorf("Call /api/paper/share/users with empty body")
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取请求的结构体
//		var qry cmn.ReqProto
//		q.Err = json.Unmarshal(buf, &qry)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取需要保存到数据库的数据
//		var u ManagePaperShareRequest
//		q.Err = json.Unmarshal(qry.Data, &u)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//		var userID int64 =1574
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, u.PaperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//
//		q.Err = managePaperShareUsers(dmlCtx, tx, u, userID)
//		if q.Err != nil {
//			tx.Rollback()
//			q.RespErr()
//			return
//		}
//		if q.Err = tx.Commit(); q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		committed = true
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	}
//}
//
//// 设置试卷共享状态
//func PaperShareStatus(ctx context.Context) {
//	q := cmn.GetCtxValue(ctx)
//	z.Info("---->" + cmn.FncName())
//	method := strings.ToLower(q.R.Method)
//	switch method {
//	case "put":
//		var buf []byte
//		buf, q.Err = io.ReadAll(q.R.Body)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		defer func() {
//			q.Err = q.R.Body.Close()
//			if q.Err != nil {
//				z.Error(q.Err.Error())
//				q.RespErr()
//				return
//			}
//		}()
//
//		if len(buf) == 0 {
//			q.Err = fmt.Errorf("Call /api/paper/share/status with empty body")
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取请求的结构体
//		var qry cmn.ReqProto
//		q.Err = json.Unmarshal(buf, &qry)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取需要保存到数据库的数据
//		var u UpdatePaperAccessModeRequest
//		q.Err = json.Unmarshal(qry.Data, &u)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var userID int64 =1574
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, u.PaperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//		q.Err = updatePaperShareStatus(dmlCtx, tx, u, userID)
//		if q.Err != nil {
//			tx.Rollback()
//			q.RespErr()
//			return
//		}
//		if q.Err = tx.Commit(); q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		committed = true
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	}
//}
