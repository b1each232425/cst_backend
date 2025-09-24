/*
 * @Author: wusaber33
 * @Date: 2025-08-03 21:39:33
 * @LastEditors: wusaber33
 * @LastEditTime: 2025-09-04 02:41:22
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

	"w2w.io/serve/auth_mgt"

	"github.com/jackc/pgx/v5"

	"w2w.io/serve/examPaper"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jmoiron/sqlx/types"
	"w2w.io/null"

	"go.uber.org/zap"
	"w2w.io/cmn"
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
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷管理.获得试卷详情",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "试卷管理.自定义组卷",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "试卷管理.保存试卷",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

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
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷管理.获取试卷列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "试卷管理.发布试卷",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "试卷管理.删除试卷",
				AccessAction: auth_mgt.CAPIAccessActionDelete,
				Configurable: true,
			},
		},

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

}

// 创建试卷\更新试卷\获取试卷详情
// ManualPaper 处理手动组卷相关的HTTP请求
// 支持以下操作:
// - POST: 创建新的空白试卷
// - PUT: 更新试卷内容和结构
// - GET: 获取试卷详细信息/预览
func ManualPaper(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)

	// 1. 验证用户身份和权限
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = ErrInvalidUserID
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

	// 获取数据库连接
	db := cmn.GetPgxConn()
	// 创建带超时的上下文
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

	//强制错误，用于测试
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	switch method {
	case "post":
		// 创建新试卷
		// 获取用户是否有写权限
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper/manual", auth_mgt.CAPIAccessActionCreate)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
			return
		}
		// 开启事务，插入试卷和默认分组
		var tx pgx.Tx
		tx, q.Err = db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		// 强制错误，用于测试
		if forceError == "BeginTx-err" {
			q.Err = errors.New(forceError)
			_ = tx.Rollback(ctx)
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
				panicErr := fmt.Errorf("panic occurred: %v", p)
				z.Error(panicErr.Error())
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
					q.RespErr()
				}
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					z.Error(err.Error())
				}
			}
			// 提交事务
			err := tx.Commit(ctx)
			if forceError == "Commit-err" {
				err = errors.New(forceError)
				q.Err = err
				q.RespErr()
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()
		if forceError == "recover-err" {
			panic(errors.New(forceError))
		}
		if forceError == "Commit-err" {
			return
		}
		if forceError == "Rollback-err" {
			q.Err = errors.New(forceError)
			return
		}

		//初始化一张空试卷SQL
		now := time.Now().UnixMilli()
		initPaperSql := `
INSERT INTO t_paper 
    (name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status,domain_id) 
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,$12) 
RETURNING id`

		paper := cmn.TPaper{
			Name:              null.NewString(DefaultPaperName, true),
			AssemblyType:      null.NewString(ManualAssemblyType, true),
			Category:          null.NewString(PaperCategoryExam, true),
			Level:             null.NewString(Easy, true),
			SuggestedDuration: null.NewInt(DefaultSuggestedDuration, true),
			Tags:              types.JSONText("[]"),
			Creator:           null.IntFrom(userID),
			CreateTime:        null.IntFrom(now),
			UpdatedBy:         null.IntFrom(userID),
			UpdateTime:        null.IntFrom(now),
			Status:            null.NewString(StatusUnPublished, true),
			DomainID:          null.IntFrom(authority.Domain.ID.Int64),
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
		defer rows.Close()
		// 强制错误，用于测试
		if forceError == "tx.Query-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper/manual", auth_mgt.CAPIAccessActionUpdate)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
			return
		}
		// 获取并验证试卷ID
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//尝试获取试卷锁REDIS_LOCK_PREFIX
		_, q.Err = cmn.TryLock(ctx, paperID, userID, REDIS_LOCK_PREFIX, REDIS_LOCK_EXPRIATION)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取试卷状态和创建者信息
		var status string
		var creatorID int64
		status, creatorID, q.Err = getPaperStatusAndCreator(ctx, paperID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 如果试卷不是未发布状态，则不能更新试卷
		if status != StatusUnPublished {
			q.Err = errors.New("试卷已发布或归档，不能更新")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var hasPermission bool
		// 如果是超级管理员，则直接拥有权限
		if authority.Role.Priority.Int64 == 0 {
			hasPermission = true
		} else {
			hasPermission = creatorID == userID
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
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper/manual", auth_mgt.CAPIAccessActionRead)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
			return
		}
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

		//解析获取试卷详情后模式（编辑或预览）
		mode := q.R.URL.Query().Get("mode")
		if mode == "" {
			mode = "edit"
		}

		//var hasPermission bool
		//// 如果是超级管理员，则直接拥有权限
		//if role == "superAdmin" {
		//	hasPermission = true
		//} else {
		//	hasPermission, q.Err = isPaperCreator(ctx, paperID, userID)
		//	if q.Err != nil {
		//		z.Error(q.Err.Error())
		//		q.RespErr()
		//		return
		//	}
		//}
		//
		//if !hasPermission {
		//	q.Err = ErrWithoutPermission
		//	z.Error(q.Err.Error())
		//	q.RespErr()
		//	return
		//}
		switch mode {
		case "edit":
			// 获取数据库连接
			db := cmn.GetPgxConn()

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
		case "preview":
			var paper *cmn.TVPaper
			var groups []*cmn.TPaperGroup
			var questions map[int64][]*examPaper.Question
			paper, groups, questions, q.Err = examPaper.LoadPaperTemplateById(dmlCtx, paperID, true)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			// 构建前端需要的题组结构体
			groupMap := make(map[int64]*cmn.TPaperGroup)
			for _, g := range groups {
				groupMap[g.ID.Int64] = g
			}
			//定义结构体用于整合数据发送给前端
			type Msg struct {
				Paper             *cmn.TVPaper
				QuestionGroupInfo map[int64]*cmn.TPaperGroup
				Questions         map[int64][]*examPaper.Question
			}

			msg := Msg{
				Paper:             paper,
				QuestionGroupInfo: groupMap,
				Questions:         questions,
			}
			var data []byte
			data, q.Err = json.Marshal(&msg)
			if forceError == "json.Marshal" {
				q.Err = errors.New("marshal err")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Msg.Data = data
			q.Msg.Msg = "success"
			q.Msg.Status = 0
		default:
			// 默认操作，返回错误信息
			q.Err = fmt.Errorf("不支持当前mode: %s", mode)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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
		_ = tx.Rollback(ctx)
	}
	if err != nil {
		z.Error("start transaction failed: " + err.Error())
		return
	}

	defer func() {
		// 如果发生panic或错误，尝试回滚事务
		if p := recover(); p != nil {
			panicErr := fmt.Errorf("panic occurred: %v", p)
			z.Error(panicErr.Error())
			rollbackErr := tx.Rollback(ctx)
			if forceError == "recover-err" {
				rollbackErr = errors.New(forceError)
				err = rollbackErr
			}
			if rollbackErr != nil {
				z.Error(rollbackErr.Error())
			}
			return
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
	// 使用一个map来存储题目ID，之后可以通过临时ID来映射真实ID，然后再执行移动题目
	var tempIDMap = make(map[int64]int64)
	now := cmn.GetNowInMS()
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
			if basicInfo.Level != "" && basicInfo.Level != Easy && basicInfo.Level != Medium && basicInfo.Level != Hard && basicInfo.Level != FairlyEasy && basicInfo.Level != FairlyHard {
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
			addField(true, "update_time", now)

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
			// 验证请求体和准备批量插入
			// 使用 pgx.Batch 进行高性能批量插入
			batch := &pgx.Batch{}
			const insertSQL = `INSERT INTO t_paper_question 
    (bank_question_id, group_id, "order", score, sub_score, creator, create_time, updated_by, update_time, status) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`

			// 验证每个题目数据并添加到批处理中
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
				if len(q.SubScore) > 0 && q.Type != QuestionTypeFillInBlank && q.Type != QuestionTypeComprehensive && q.Type != QuestionTypeExercise {
					err = fmt.Errorf("题目小题分数只能在填空题、综合应用题或综合演练题中使用: %s", q.Type)
					z.Error(err.Error())
					return
				}
				if len(q.SubScore) > 0 && (q.Type == QuestionTypeFillInBlank || q.Type == QuestionTypeComprehensive || q.Type == QuestionTypeExercise) {
					for _, sub := range q.SubScore {
						if sub <= 0 {
							err = fmt.Errorf("题目小题分数不能小于等于0: %f", sub)
							z.Error(err.Error())
							return
						}
					}
				}

				// 将插入语句添加到批处理中
				batch.Queue(insertSQL,
					q.BankQuestionID, q.GroupID, q.Order, q.Score, q.SubScore,
					userID, now, userID, now, StatusNormal)
			}

			// 执行批量操作
			batchResults := tx.SendBatch(ctx, batch)
			if forceError == "batch.SendBatch-err" {
				err = errors.New(forceError)
				batchResults.Close()
				return nil, fmt.Errorf("批量插入题目失败 [错误:%v]", err)
			}
			defer batchResults.Close()

			// 直接构建临时ID到数据库ID的映射
			idMapping := make(map[int64]int64, len(req))

			// 处理数据库返回的自增ID并构建映射关系
			for i := 0; i < len(req); i++ {
				var id int64
				err = batchResults.QueryRow().Scan(&id)
				if forceError == "batchResults.Scan-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error("error occurred while scanning returned ID", zap.Error(err), zap.Int("index", i))
					return nil, fmt.Errorf("扫描返回ID失败 [错误:%v]", err)
				}

				// 直接将数据库返回的ID与请求中的临时ID对应
				idMapping[req[i].TempID] = id
			}

			// 检查批量操作是否有错误
			err = batchResults.Close()
			if forceError == "batchResults.Close-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error("error occurred while closing batch results", zap.Error(err))
				return nil, fmt.Errorf("关闭批量结果时出错 [错误:%v]", err)
			}

			// 将ID映射结果存储到结果中
			result = idMapping
			// 清理临时ID映射
			tempIDMap = idMapping
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
			commandTag, err = tx.Exec(ctx, sql, req.Name, userID, now, req.ID)
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
			// 替换题目ID数组中的临时ID（有些题目可能刚刚才创建）
			for i, id := range orders {
				if newID, ok := tempIDMap[id]; ok {
					orders[i] = newID
				}
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
// - POST: 发布试卷
func PaperList(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}

	method := strings.ToLower(q.R.Method)

	//获取用户ID
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = ErrInvalidUserID
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

	// 获取数据库连接
	db := cmn.GetPgxConn()
	// 创建带超时的上下文
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

	switch method {
	case "get":
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper", auth_mgt.CAPIAccessActionRead)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
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
		req.Published = false
		if published := queryParams.Get("published"); published != "" {
			if p, err := strconv.ParseBool(published); err == nil {
				req.Published = p
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

		// 可以查看未作废试卷或只查看已发布试卷
		if req.Published {
			whereClauses = append(whereClauses, "p.status = '06' ")
		} else {
			whereClauses = append(whereClauses, "p.status IN ('00', '06')")
		}

		if forceError == "EmptyDomain" {
			authority.AccessibleDomains = []int64{}
		}

		// 拼接资源范围 - 用户可访问的所有 domain_id
		if len(authority.AccessibleDomains) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("p.domain_id = ANY($%d)", paramCount))
			params = append(params, authority.AccessibleDomains)
			paramCount++
		} else {
			// 如果用户没有可访问的域，则返回空结果
			q.Err = errors.New("用户没有可访问的域")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//// 如果设置了self，则只查询当前用户创建的试卷
		//if req.Self {
		//	var creatorClause strings.Builder
		//	creatorClause.WriteString("p.creator = $")
		//	creatorClause.WriteString(strconv.Itoa(paramCount))
		//	whereClauses = append(whereClauses, creatorClause.String())
		//	params = append(params, userID)
		//	paramCount++
		//}

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
		countSQLBuilder.WriteString("SELECT COUNT(*) FROM t_paper p ")
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

		// 查询分页数据：先 top 取 id，再统计并拼装列表
		var listSQLBuilder strings.Builder
		listSQLBuilder.WriteString(`
WITH top_ids AS (
	SELECT p.id, p.exampaper_id, p.update_time
	FROM t_paper p
	`)
		// 将动态 where 条件拼入 filtered CTE
		listSQLBuilder.WriteString(whereClause)
		listSQLBuilder.WriteString(`
	ORDER BY p.update_time DESC, p.id DESC
	LIMIT $`)
		// LIMIT / OFFSET 使用动态参数位
		listSQLBuilder.WriteString(strconv.Itoa(paramCount))
		listSQLBuilder.WriteString(" OFFSET $")
		listSQLBuilder.WriteString(strconv.Itoa(paramCount + 1))
		listSQLBuilder.WriteString(`
),
paper_stats AS (
	SELECT pg.paper_id,
				 COALESCE(SUM(pq.score), 0)::float8 AS total_score,
				 COALESCE(COUNT(pq.id), 0)::numeric AS question_count
	FROM t_paper_group pg
	LEFT JOIN t_paper_question pq
		ON pq.group_id = pg.id AND pq.status <> '02'
	WHERE pg.status <> '02'
		AND pg.paper_id IN (SELECT id FROM top_ids)
	GROUP BY pg.paper_id
),
exam_stats AS (
	SELECT eg.exam_paper_id,
				 COALESCE(SUM(eq.score), 0)::float8 AS total_score,
				 COALESCE(COUNT(eq.id), 0)::numeric AS question_count
	FROM t_exam_paper_group eg
	LEFT JOIN t_exam_paper_question eq
		ON eq.group_id = eg.id AND eq.status <> '02'
	WHERE eg.status <> '02'
		AND eg.exam_paper_id IN (
			SELECT exampaper_id FROM top_ids WHERE exampaper_id IS NOT NULL
		)
	GROUP BY eg.exam_paper_id
)
SELECT 
	p.id,
	p.exampaper_id,
	p.name,
	p.assembly_type,
	p.category,
	p.level,
	p.suggested_duration,
	COALESCE(
		CASE WHEN p.exampaper_id IS NOT NULL AND p.status <> '00' THEN es.total_score END,
		CASE WHEN p.exampaper_id IS NULL OR p.status = '00' THEN ps.total_score END,
		0
	)::float8 AS total_score,
	COALESCE(
		CASE WHEN p.exampaper_id IS NOT NULL AND p.status <> '00' THEN es.question_count END,
		CASE WHEN p.exampaper_id IS NULL OR p.status = '00' THEN ps.question_count END,
		0
	)::numeric AS question_count,
	p.tags,
	p.create_time,
	p.update_time,
	p.status,
	p.creator
FROM top_ids ti
JOIN t_paper p ON p.id = ti.id
LEFT JOIN paper_stats ps ON ps.paper_id = p.id
LEFT JOIN exam_stats es  ON es.exam_paper_id = p.exampaper_id
LEFT JOIN t_user u       ON u.id = p.creator
ORDER BY p.update_time DESC, p.id DESC`)

		dataParams := append(params, req.PageSize, offset)
		var rows pgx.Rows

		rows, q.Err = db.Query(dmlCtx, listSQLBuilder.String(), dataParams...)
		defer rows.Close()
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRow-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var papers []cmn.TVPaper
		for rows.Next() {
			var paper cmn.TVPaper
			q.Err = rows.Scan(&paper.ID, &paper.ExampaperID, &paper.Name, &paper.AssemblyType, &paper.Category, &paper.Level, &paper.SuggestedDuration, &paper.TotalScore, &paper.QuestionCount, &paper.Tags, &paper.CreateTime, &paper.UpdateTime, &paper.Status, &paper.Creator)
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
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper", auth_mgt.CAPIAccessActionDelete)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
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

		// 开启事务
		var tx pgx.Tx
		tx, q.Err = db.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.RepeatableRead,
		})
		if forceError == "PaperList-delete-BeginTx-err" {
			q.Err = errors.New(forceError)
			_ = tx.Rollback(ctx)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 确保事务结束时回滚
		defer func() {
			if p := recover(); p != nil {
				panicErr := fmt.Errorf("panic occurred: %v", p)
				z.Error(panicErr.Error())
				err := tx.Rollback(ctx)
				if forceError == "PaperList-delete-Rollback-panic" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
				}
				return
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

		if forceError == "EmptyDomain" {
			authority.AccessibleDomains = []int64{}
		}

		// 检查每个试卷的权限
		var checkSQL string
		var errorMessages []string
		// 超级管理员或管理员检查试卷存在性和域权限
		if authority.Role.Priority.Int64 == auth_mgt.CDomainPriorityAdmin || authority.Role.Priority.Int64 == auth_mgt.CDomainPrioritySuperAdmin {
			if len(authority.AccessibleDomains) > 0 {
				checkSQL = `
					SELECT COALESCE(array_agg(
						CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							ELSE NULL 
						END
					) FILTER (WHERE CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							ELSE NULL 
						END IS NOT NULL), ARRAY[]::text[]) as error_messages
					FROM unnest($1::bigint[]) AS ids(id)
					LEFT JOIN t_paper p ON p.id = ids.id
					WHERE p.id IS NULL OR NOT (p.domain_id = ANY($2))`
				q.Err = tx.QueryRow(ctx, checkSQL, paperIDs, authority.AccessibleDomains).Scan(&errorMessages)
			} else {
				// 如果没有可访问的域，直接返回错误
				errorMessages = []string{"用户没有可访问的域权限"}
			}
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
			if len(authority.AccessibleDomains) > 0 {
				checkSQL = `
					SELECT COALESCE(array_agg(
						CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							WHEN p.creator != $3 THEN '试卷（' || COALESCE(p.name, '未知') || '）非试卷创建者，无删除权限'
							ELSE NULL 
						END
					) FILTER (WHERE CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							WHEN p.creator != $3 THEN '试卷（' || COALESCE(p.name, '未知') || '）非试卷创建者，无删除权限'
							ELSE NULL 
						END IS NOT NULL), ARRAY[]::text[]) as error_messages
					FROM unnest($1::bigint[]) AS ids(id)
					LEFT JOIN t_paper p ON p.id = ids.id
					WHERE p.id IS NULL OR NOT (p.domain_id = ANY($2)) OR p.creator != $3`
				q.Err = tx.QueryRow(ctx, checkSQL, paperIDs, authority.AccessibleDomains, userID).Scan(&errorMessages)
			} else {
				// 如果没有可访问的域，直接返回错误
				errorMessages = []string{"用户没有可访问的域权限"}
			}
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
			q.Msg.Status = -1
			q.Err = errors.New(validErrors.String())
			q.RespErr()
			return
		}

		now := cmn.GetNowInMS()
		//1. 软删除 t_paper
		paperSQL := `UPDATE t_paper SET status = $2, updated_by = $3, update_time = $4 WHERE id = ANY($1)`
		_, q.Err = tx.Exec(dmlCtx, paperSQL, paperIDs, StatusDeleted, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapers-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//2. 软删除 t_paper_group
		groupSQL := `UPDATE t_paper_group SET status = $2, updated_by = $3, update_time = $4 WHERE paper_id = ANY($1)`
		_, q.Err = tx.Exec(ctx, groupSQL, paperIDs, StatusDeleted, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersgroups-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//3. 软删除 t_paper_question
		questionSQL := `UPDATE t_paper_question SET status = $2, updated_by = $3, update_time = $4 WHERE group_id IN (SELECT id FROM t_paper_group WHERE paper_id = ANY($1))`
		_, q.Err = tx.Exec(ctx, questionSQL, paperIDs, StatusDeleted, userID, now)
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
	case "post":
		// 发布试卷
		// 2. 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper", auth_mgt.CAPIAccessActionCreate)
		if q.Err != nil {
			fmt.Printf("检查API访问权限失败: %v\n", q.Err)
			return
		}
		if !accessible {
			fmt.Println("用户没有访问权限")
			return
		}
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
		_, q.Err = cmn.TryLock(ctx, paperID, userID, REDIS_LOCK_PREFIX, REDIS_LOCK_EXPRIATION)
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
		if forceError == "BeginTx" {
			q.Err = errors.New(forceError)
			_ = tx.Rollback(ctx)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 确保事务结束时回滚
		defer func() {
			if p := recover(); p != nil {
				panicErr := fmt.Errorf("panic occurred: %v", p)
				z.Error(panicErr.Error())
				err := tx.Rollback(ctx)
				if forceError == "Rollback-panic" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
				}
				return
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if forceError == "Rollback" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					z.Error(err.Error())
					return
				}
			}
			// 提交事务
			err := tx.Commit(ctx)
			if forceError == "Commit" {
				err = errors.New(forceError)
				q.Err = err
				q.RespErr()
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()
		if forceError == "Rollback-panic" {
			panic(errors.New(forceError))
		}
		if forceError == "Commit" {
			return
		}
		if forceError == "Rollback" {
			q.Err = errors.New(forceError)
			return
		}

		// 检测试卷是否已发布
		var paper cmn.TPaper
		q.Err = tx.QueryRow(dmlCtx, `SELECT category,status FROM t_paper WHERE id = $1`, paperID).Scan(&paper.Category, &paper.Status)
		if forceError == "tx.QueryRow" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if paper.Status.String != StatusUnPublished {
			q.Err = errors.New("试卷已发布或已归档")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查每个试卷的权限
		var checkSQL string
		var errorMessages []string
		// 超级管理员或管理员检查试卷存在性和域权限
		if authority.Role.Priority.Int64 == auth_mgt.CDomainPriorityAdmin || authority.Role.Priority.Int64 == auth_mgt.CDomainPrioritySuperAdmin {
			if len(authority.AccessibleDomains) > 0 {
				checkSQL = `
					SELECT COALESCE(array_agg(
						CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							ELSE NULL 
						END
					) FILTER (WHERE CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							ELSE NULL 
						END IS NOT NULL), ARRAY[]::text[]) as error_messages
					FROM unnest($1::bigint[]) AS ids(id)
					LEFT JOIN t_paper p ON p.id = ids.id
					WHERE p.id IS NULL OR NOT (p.domain_id = ANY($2))`
				q.Err = tx.QueryRow(ctx, checkSQL, []int64{paperID}, authority.AccessibleDomains).Scan(&errorMessages)
			} else {
				// 如果没有可访问的域，直接返回错误
				errorMessages = []string{"用户没有可访问的域权限"}
			}
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
			if len(authority.AccessibleDomains) > 0 {
				checkSQL = `
					SELECT COALESCE(array_agg(
						CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							WHEN p.creator != $3 THEN '试卷（' || COALESCE(p.name, '未知') || '）非试卷创建者，无发布权限'
							ELSE NULL 
						END
					) FILTER (WHERE CASE 
							WHEN p.id IS NULL THEN '试卷（' || ids.id || '）不存在'
							WHEN NOT (p.domain_id = ANY($2)) THEN '试卷（' || COALESCE(p.name, '未知') || '）不在当前域范围内'
							WHEN p.creator != $3 THEN '试卷（' || COALESCE(p.name, '未知') || '）非试卷创建者，无发布权限'
							ELSE NULL 
						END IS NOT NULL), ARRAY[]::text[]) as error_messages
					FROM unnest($1::bigint[]) AS ids(id)
					LEFT JOIN t_paper p ON p.id = ids.id
					WHERE p.id IS NULL OR NOT (p.domain_id = ANY($2)) OR p.creator != $3`
				q.Err = tx.QueryRow(ctx, checkSQL, []int64{paperID}, authority.AccessibleDomains, userID).Scan(&errorMessages)
			} else {
				// 如果没有可访问的域，直接返回错误
				errorMessages = []string{"用户没有可访问的域权限"}
			}
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
			q.Msg.Status = -1
			q.Err = errors.New(validErrors.String())
			q.RespErr()
			return
		}

		// 生成考卷并修改试卷状态为已发布
		var examPaperID *int64
		examPaperID, q.Err = examPaper.GenerateExamPaper(dmlCtx, tx, paperID, userID)
		if forceError == "examPaper.GenerateExamPaper" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 修改试卷状态为已发布并存储exampaperID到试卷表，便于练习或考试获取
		now := cmn.GetNowInMS()
		_, q.Err = tx.Exec(dmlCtx, `UPDATE t_paper SET status = $1, exampaper_id = $2, update_time = $3, updated_by = $4 WHERE id = $5`, StatusPublished, examPaperID, now, userID, paperID)
		if forceError == "UPDATE-tx.Exec" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 删除试卷题组和试卷题目，后续用不到了，直接级联删除题组即可,题目会跟着级联删除
		_, q.Err = tx.Exec(dmlCtx, `DELETE FROM t_paper_group WHERE paper_id = $1`, paperID)
		if forceError == "DELETE-tx.Exec" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//尝试获取试卷锁REDIS_LOCK_PREFIX
		q.Err = cmn.ReleaseLock(ctx, paperID, userID, REDIS_LOCK_PREFIX)
		if forceError == "cmn.ReleaseLock" {
			q.Err = errors.New(forceError)
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
	//var authority *auth_mgt.Authority
	//authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	//if q.Err != nil {
	//	fmt.Printf("获取用户权限失败: %v\n", q.Err)
	//	return
	//}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 2. 检查API访问权限
		//var accessible bool
		//accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper/lock", auth_mgt.CDataAccessModeEdit)
		//if q.Err != nil {
		//	fmt.Printf("检查API访问权限失败: %v\n", q.Err)
		//	return
		//}
		//if !accessible {
		//	fmt.Println("用户没有访问权限")
		//	return
		//}
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
		_, q.Err = cmn.TryLock(ctx, paperID, userID, REDIS_LOCK_PREFIX, REDIS_LOCK_EXPRIATION)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	case "put":
		// 解析并验证试卷ID
		// 2. 检查API访问权限
		//var accessible bool
		//accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/paper/lock", auth_mgt.CDataAccessModeEdit)
		//if q.Err != nil {
		//	fmt.Printf("检查API访问权限失败: %v\n", q.Err)
		//	return
		//}
		//if !accessible {
		//	fmt.Println("用户没有访问权限")
		//	return
		//}
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

// getPaperStatusAndCreator 获取试卷状态和创建者信息
func getPaperStatusAndCreator(ctx context.Context, paperID int64) (string, int64, error) {
	// 获取数据库连接
	db := cmn.GetPgxConn()
	var status string
	var creatorID int64
	err := db.QueryRow(ctx, `
	SELECT status, creator FROM t_paper 
	WHERE id = $1
	`, paperID).Scan(&status, &creatorID)
	if err != nil {
		z.Error("failed to get paper status and creator: " + err.Error())
		return "", 0, err
	}
	return status, creatorID, nil
}
