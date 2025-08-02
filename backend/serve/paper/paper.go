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

var actionsWithResult = map[string]bool{
	"add_question": true,
	"add_group":    true,
}

const (
	TIMEOUT = 5 * time.Second
)

const (
	DefaultGroup1Name, DefaultGroup2Name, DefaultGroup3Name, DefaultGroup4Name, DefaultGroup5Name                            = "一、单选题", "二、多选题", "三、判断题", "四、填空题", "五、简答题"
	StatusNormal, StatusUnNormal, StatusDeleted                                                                              = "00", "02", "04"
	PaperCategoryExam, PaperCategoryPractice                                                                                 = "00", "02"
	QuestionTypeMultiChoice, QuestionTypeSingleChoice, QuestionTypeJudgement, QuestionTypeFillBlank, QuestionTypeShortAnswer = "00", "02", "04", "06", "08"
	Simple, Medium, Hard, AllLevels                                                                                          = "00", "02", "04", "06"
	DefaultSuggestedDuration                                                                                                 = 120
	DefaultPaperName                                                                                                         = "新建试卷"
	PaperShareStatusPrivate, PaperShareStatusShared, PaperShareStatusPublic                                                  = "00", "02", "04"
	ManualAssemblyType                                                                                                       = "00"
	PaperResourceShareType                                                                                                   = "12"
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
func ManualPaper(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	switch method {
	case "post":
		//获取用户ID
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效用户ID: %d", userID)
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
			if domain.DomainID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				} else {
				}
				role = resources[1]
				break
			}
		}

		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = fmt.Errorf("没有权限创建试卷: %s", role)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		db := cmn.GetPgxConn()
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

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

		var tx pgx.Tx
		tx, q.Err = db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if forceError == "BeginTx-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			p := recover()
			if forceError == "recover-err" {
				p = errors.New(forceError)
			}
			if p != nil {
				err := tx.Rollback(ctx)
				if forceError == "recover-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
			if forceError == "Rollback-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if forceError == "Rollback-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
		}()

		//初始化一张空试卷SQL
		initPaperSql := `
INSERT INTO t_paper 
    (name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode,domain_id) 
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,$12) 
RETURNING id`

		paper := cmn.TPaper{
			Name:              null.NewString(DefaultPaperName, true),
			AssemblyType:      null.NewString(ManualAssemblyType, true),
			Category:          null.NewString(PaperCategoryExam, true),
			Level:             null.NewString(Simple, true),
			SuggestedDuration: null.NewInt(DefaultSuggestedDuration, true),
			Tags:              types.JSONText("[]"),
			Creator:           null.IntFrom(userID),
			CreateTime:        null.IntFrom(time.Now().UnixMilli()),
			UpdatedBy:         null.IntFrom(userID),
			UpdateTime:        null.IntFrom(time.Now().UnixMilli()),
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
		if forceError == "tx.QueryRow-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
		}

		groupNames := []string{
			DefaultGroup1Name,
			DefaultGroup2Name,
			DefaultGroup3Name,
			DefaultGroup4Name,
			DefaultGroup5Name,
		}
		now := time.Now().UnixMilli()
		addi := types.JSONText([]byte("{}"))
		status := StatusNormal

		groupSql := `INSERT INTO t_paper_group 
    (paper_id, name, "order", creator, create_time, updated_by, update_time, addi, status)
VALUES
    ($1, $2, 1, $3, $4, $3, $4, $5, $6),
    ($1, $7, 2, $3, $4, $3, $4, $5, $6),
    ($1, $8, 3, $3, $4, $3, $4, $5, $6),
    ($1, $9, 4, $3, $4, $3, $4, $5, $6),
    ($1, $10, 5, $3, $4, $3, $4, $5, $6)
RETURNING id`
		args := []any{
			paper.ID,
			groupNames[0],
			userID,
			now,
			addi,
			status,
			groupNames[1],
			groupNames[2],
			groupNames[3],
			groupNames[4],
		}
		var rows pgx.Rows
		rows, q.Err = tx.Query(ctx, groupSql, args...)
		if forceError == "tx.Query-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error("insert paper groups failed", zap.Error(q.Err))
			q.RespErr()
		}
		defer rows.Close()
		groups := make([]cmn.TPaperGroup, 0, 5)
		for i := 0; rows.Next(); i++ {
			var groupID int64
			q.Err = rows.Scan(&groupID)
			if forceError == "rows.Scan-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error("scan group id failed", zap.Error(q.Err))

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
				Addi:       addi,
				Status:     null.NewString(status, true),
			}
			groups = append(groups, group)
		}
		q.Err = rows.Err()
		if forceError == "rows.Err-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error("rows error", zap.Error(q.Err))
			q.RespErr()
		}
		q.Err = tx.Commit(ctx)
		if forceError == "Commit-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var isCreator bool
		// 如果是超级管理员，则直接拥有权限
		if q.IsAdmin {
			isCreator = true
		}
		isCreator, q.Err = isPaperCreator(ctx, paperID, userID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if !isCreator {
			q.Err = fmt.Errorf("无权限更新试卷: %d", paperID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
			q.Err = q.R.Body.Close()
			if forceError == "R.Body.Close-err" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("Call /api/paper/manual with empty body")
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

		//参数校验
		q.Err = cmn.Validate(u)
		if q.Err != nil {
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
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var isCreator bool
		isCreator, q.Err = isPaperCreator(ctx, paperID, userID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if !isCreator {
			q.Err = fmt.Errorf("无权限更新试卷: %d", paperID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		if paperID <= 0 {
			q.Err = ErrInvalidPaperID
			z.Error(q.Err.Error())
			q.RespErr()
		}

		query := `SELECT 
    id,name,assembly_type,category,level,suggested_duration,description,tags,creator,create_time,update_time,status,total_score,question_count,groups_data
	FROM v_paper
	WHERE id = $1
	LIMIT 1`

		db := cmn.GetPgxConn()
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
			if errors.Is(q.Err, ErrRecordNotFound) {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		data, _ := json.Marshal(paper)
		q.Msg.Data = data
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	}

}

// 试卷首页  列表获取\删除试卷
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
			q.Err = fmt.Errorf("invalid UserID: %d", userID)
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
			if domain.DomainID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				} else {
				}
				role = resources[1]
				break
			}
		}

		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = fmt.Errorf("没有权限创建试卷: %s", role)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		db := cmn.GetPgxConn()

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

		req.Self = false
		if self := queryParams.Get("self"); self != "" {
			if s, err := strconv.ParseBool(self); err == nil {
				req.Self = s
			}
		}

		q.Err = cmn.Validate(req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		// 如果设置了self，则只查询当前用户创建的试卷
		if req.Self {
			var creatorClause strings.Builder
			creatorClause.WriteString("p.creator = $")
			creatorClause.WriteString(strconv.Itoa(paramCount))
			whereClauses = append(whereClauses, creatorClause.String())
			params = append(params, userID)
			paramCount++
		}

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

		// 1. 查询总数
		var countSQLBuilder strings.Builder
		countSQLBuilder.WriteString("SELECT COUNT(*) FROM v_paper p ")
		countSQLBuilder.WriteString(whereClause)
		q.Err = db.QueryRow(ctx, countSQLBuilder.String(), params...).Scan(&totalCount)
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRowCount-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error("failed to count paper list", zap.Error(q.Err))
			q.RespErr()
			return
		}

		// 2. 查询分页数据
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
			z.Error("failed to query paper list", zap.Error(q.Err))
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
				z.Error("failed to scan paper basic info", zap.Error(q.Err))
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
			z.Error("rows iteration error", zap.Error(q.Err))
			q.RespErr()
			return
		}

		data, _ := json.Marshal(papers)
		q.Msg.Data = data
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.RowCount = totalCount
		q.Msg.Msg = "success"
		q.Resp()
	case "delete":
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("invalid UserID: %d", userID)
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
			if domain.DomainID.Int64 == roleID {
				//拆出domain的名称前缀，确认资源范围
				resources := strings.Split(domain.Domain, "^")
				resourceDomain = resources[0]
				parts := strings.Split(resourceDomain, ".")
				if len(parts) >= 2 {
					resourceDomain = strings.Join(parts[:2], ".")
				} else {
				}
				role = resources[1]
				break
			}
		}

		if role != "teacher" && role != "superAdmin" && role != "admin" {
			q.Err = fmt.Errorf("没有权限创建试卷: %s", role)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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

		if paperIDs == nil && len(paperIDs) == 0 {
			q.Err = ErrEmptyPaperIDs
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		db := cmn.GetPgxConn()
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

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

		var tx pgx.Tx
		tx, q.Err = db.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})
		if forceError == "PaperList-delete-BeginTx-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := tx.Rollback(ctx)
			if forceError == "PaperList-delete-Rollback-err" {
				err = errors.New(forceError)
			}
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				z.Error("事务回滚失败", zap.Error(err))
			}
		}()

		// 检查每个试卷的权限
		var checkSQL string
		if q.IsAdmin {
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
		}

		var errorMessages []string
		q.Err = tx.QueryRow(ctx, checkSQL, paperIDs, resourceID, userID).Scan(&errorMessages)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 移除空错误消息
		var validErrors []string
		for _, msg := range errorMessages {
			if msg != "" {
				validErrors = append(validErrors, msg)
			}
		}

		// 如果有任何不能删除的试卷，返回错误
		if len(validErrors) > 0 {
			data, _ := json.Marshal(validErrors)
			q.Msg.Data = data
			q.Msg.Status = 1
			q.Msg.Msg = "部分试卷无法删除"
			q.RespErr()
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
			z.Error("failed to soft delete t_paper", zap.Error(q.Err))
			return
		}

		// 2. 软删除 t_paper_group
		groupSQL := `UPDATE t_paper_group SET status = $2, updated_by = $3, update_time = $4 WHERE paper_id = ANY($1)`
		_, q.Err = tx.Exec(ctx, groupSQL, paperIDs, StatusUnNormal, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersgroups-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error("failed to soft delete t_paper_group", zap.Error(q.Err))
			return
		}

		// 3. 软删除 t_paper_question
		questionSQL := `UPDATE t_paper_question SET status = $2, updated_by = $3, update_time = $4 WHERE group_id IN (SELECT id FROM t_paper_group WHERE paper_id = ANY($1))`
		_, q.Err = tx.Exec(ctx, questionSQL, paperIDs, StatusUnNormal, userID, now)
		if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersquestions-exec-err" {
			q.Err = errors.New(val)
		}
		if q.Err != nil {
			z.Error("failed to soft delete t_paper_question", zap.Error(q.Err))
			return
		}
		q.Err = tx.Commit(ctx)
		if forceError == "PaperList-delete-Commit-err" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	}
}

// 更新试卷流程
func updateManualPaper(ctx context.Context, paperID, userID int64, req UpdateManualPaperRequest) (results []ActionResult, err error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	sqlxDB := cmn.GetPgxConn()
	var tx pgx.Tx
	tx, err = sqlxDB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if forceError == "BeginTx-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("updateManualPaper启动事务失败:" + err.Error())
		return
	}

	defer func() {
		if p := recover(); p != nil {
			rollbackErr := tx.Rollback(ctx)
			if forceError == "recover-err" {
				rollbackErr = errors.New(forceError)
			}
			if rollbackErr != nil {
				z.Error(rollbackErr.Error())
				err = rollbackErr
				return
			}
		}
		if err != nil {
			rollbackErr := tx.Rollback(ctx)
			if forceError == "Rollback-err" {
				rollbackErr = errors.New(forceError)
			}
			if rollbackErr != nil {
				z.Error(err.Error())
				err = rollbackErr
			}
		} else {
			commitErr := tx.Commit(ctx)
			if forceError == "Commit-err" {
				commitErr = errors.New(forceError)
			}
			if commitErr != nil {
				z.Error(commitErr.Error())
				err = commitErr
			}
		}
	}()
	if forceError == "recover-err" {
		panic(errors.New(forceError))
	}
	for _, act := range req.Actions {
		var result interface{}

		switch act.Action {
		case "update_info":
			var basicInfo UpdatePaperBasicInfoRequest
			err = json.Unmarshal(act.Payload, &basicInfo)
			if err != nil {
				z.Error("failed to unmarshal basic info payload: " + err.Error())
				return
			}
			err = cmn.Validate(&basicInfo)
			var (
				setClauses []string
				params     []interface{}
				paramIndex = 1
			)

			addField := func(condition bool, field string, value interface{}) {
				if condition {
					setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
					params = append(params, value)
					paramIndex++
				}
			}

			addField(basicInfo.Name != "", "name", basicInfo.Name)
			addField(basicInfo.Category != "", "category", basicInfo.Category)
			addField(basicInfo.Level != "", "level", basicInfo.Level)
			addField(basicInfo.Duration > 0, "suggested_duration", basicInfo.Duration)
			addField(true, "description", basicInfo.Description)

			if basicInfo.Tags != nil {
				jsonTags, _ := json.Marshal(basicInfo.Tags)
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
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sqlStr, params...)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error("failed to update paper info: " + err.Error())
				return
			}

			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "add_group":
			var req AddQuestionGroupRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			err = cmn.Validate(req)
			if err != nil {
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
				err = fmt.Errorf("题组不存在于当前试卷:" + ErrRecordNotFound.Error())
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
			var req []AddQuestionsRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			const batchInsertPaperQuestionsSQL = `INSERT INTO t_paper_question 
    (bank_question_id, group_id, "order", score,sub_score, creator, create_time, updated_by, update_time, status) 
VALUES %s RETURNING id`
			// 生成占位符和参数
			var placeholders []string
			var args []interface{}
			var ids []int64
			paramIndex := 1
			now := time.Now().UnixMilli()

			for _, q := range req {
				err = cmn.Validate(q)
				if err != nil {
					z.Error(err.Error())
					return nil, err
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

			// 修改点3：使用参数化查询（防止SQL注入）
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
			for rows.Next() {
				var id int64
				err = rows.Scan(&id)
				if forceError == "rows.Scan-err" {
					err = errors.New(forceError)
				}
				if err != nil {
					z.Error(err.Error())
					return nil, err
				}
				ids = append(ids, id)
			}
			err = rows.Err()
			if forceError == "rows.Err-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			idMapping := make(map[string]int64)
			for i, id := range ids {
				idMapping[req[i].TempID] = id
			}
			result = idMapping
		case "delete_question":
			var questionIDs []int64
			err = json.Unmarshal(act.Payload, &questionIDs)
			if err != nil {
				z.Error(err.Error())
				return
			}
			if len(questionIDs) == 0 {
				z.Error(ErrEmptyQuestionIDs.Error())
				return nil, ErrEmptyQuestionIDs
			}
			const sql = `
DELETE FROM t_paper_question tpq 
	WHERE tpq.id = ANY($1::bigint[])`
			var commandTag pgconn.CommandTag
			commandTag, err = tx.Exec(ctx, sql, questionIDs)
			if forceError == "tx.Exec-err" {
				err = errors.New(forceError)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "update_question":
			var reqs []UpdatePaperQuestionRequest
			if err = json.Unmarshal(act.Payload, &reqs); err != nil {
				z.Error(err.Error())
				return
			}
			for _, update := range reqs {
				//检查结构体
				err = cmn.Validate(update)
				if err != nil {
					z.Error(err.Error())
					return
				}
				var setClauses []string
				var args []interface{}
				argIndex := 1 // 参数索引从1开始（公共字段从第3个参数开始）

				// 动态构建 SET 子句：仅包含非空字段
				if update.GroupID > 0 {
					setClauses = append(setClauses, "group_id = $"+strconv.Itoa(argIndex))
					args = append(args, update.GroupID)
					argIndex++
				}

				if update.Score > 0 {
					setClauses = append(setClauses, "score = $"+strconv.Itoa(argIndex))
					args = append(args, update.Score)
					argIndex++
				}

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

				if len(setClauses) <= 2 {
					err = fmt.Errorf("更新题目没有传入需要更新的字段或传入的字段为零值")
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
				rowsAffected := commandTag.RowsAffected()
				if rowsAffected == 0 {
					z.Error(ErrRecordNotFound.Error())
					return nil, ErrRecordNotFound
				}
			}
		case "update_group":
			var req UpdateQuestionsGroupRequest
			if err = json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return
			}
			err = cmn.Validate(req)
			if err != nil {
				z.Error(err.Error())
				return
			}
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
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "move_question":
			var orders []int64
			err = json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return
			}
			if err = validateIDs(orders); err != nil {
				return nil, err
			}
			if len(orders) == 0 {
				z.Error(ErrEmptyQuestionIDs.Error())
				return nil, ErrEmptyQuestionIDs
			}
			// 1. 查询实际题组数量
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
			// 2. 验证数量
			if len(orders) != actualCount {
				err = fmt.Errorf("数量不匹配，输入%d个ID，实际有%d个题目", len(orders), actualCount)
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
			rowsAffected := commandTag.RowsAffected()
			if rowsAffected == 0 {
				z.Error(ErrRecordNotFound.Error())
				return nil, ErrRecordNotFound
			}
		case "move_group":
			var orders []int64
			err = json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return
			}
			if err = validateIDs(orders); err != nil {
				return
			}
			if len(orders) == 0 {
				z.Error(ErrEmptyGroupID.Error())
				return nil, ErrEmptyGroupID
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
				err = fmt.Errorf("数量不匹配，输入%d个ID，实际有%d个题组", len(orders), actualCount)
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

func isPaperCreator(ctx context.Context, paperID, userID int64) (bool, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
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
