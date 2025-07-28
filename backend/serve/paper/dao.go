package paper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx/types"
	"w2w.io/null"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"w2w.io/cmn"
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

// -------------------------------------------------试卷基础部分---------------------------------------------------
// 创建空卷
func createManualPaperTx(ctx context.Context, tx pgx.Tx, userID int64) (cmn.TPaper, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	//初始化一张空试卷SQL
	sql := `
INSERT INTO t_paper 
    (name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode) 
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
	}
	err := tx.QueryRow(ctx, sql,
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
	).Scan(&paper.ID)
	if forceError == "tx.QueryRow-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return cmn.TPaper{}, err
	}
	return paper, nil
}

// 初始化题组
func initialManualPaperGroupsTx(ctx context.Context, tx pgx.Tx, paperID, userID int64) ([]cmn.TPaperGroup, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
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

	sql := `INSERT INTO t_paper_group 
    (paper_id, name, "order", creator, create_time, updated_by, update_time, addi, status)
VALUES
    ($1, $2, 1, $3, $4, $3, $4, $5, $6),
    ($1, $7, 2, $3, $4, $3, $4, $5, $6),
    ($1, $8, 3, $3, $4, $3, $4, $5, $6),
    ($1, $9, 4, $3, $4, $3, $4, $5, $6),
    ($1, $10, 5, $3, $4, $3, $4, $5, $6)
RETURNING id`
	args := []any{
		paperID,
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
	rows, err := tx.Query(ctx, sql, args...)
	if forceError == "tx.Query-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("insert paper groups failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	groups := make([]cmn.TPaperGroup, 0, 5)
	for i := 0; rows.Next(); i++ {
		var groupID int64
		err := rows.Scan(&groupID)
		if forceError == "rows.Scan-err" {
			err = errors.New(forceError)
		}
		if err != nil {
			z.Error("scan group id failed", zap.Error(err))
			return nil, err
		}
		group := cmn.TPaperGroup{
			ID:         null.IntFrom(groupID),
			PaperID:    null.IntFrom(paperID),
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
	err = rows.Err()
	if forceError == "rows.Err-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("rows error", zap.Error(err))
		return nil, err
	}
	return groups, nil
}

func handleUpdateInfo(ctx context.Context, tx pgx.Tx, paperID int64, userID int64, basicInfo UpdatePaperBasicInfoRequest) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
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
	addField(basicInfo.Description != "", "description", basicInfo.Description)

	if basicInfo.Tags != nil {
		jsonTags, _ := json.Marshal(basicInfo.Tags)
		addField(true, "tags", jsonTags)
	}

	//更新更新者与更新时间
	addField(true, "updated_by", userID)
	addField(true, "update_time", time.Now().UnixMilli())

	// 如果只有系统字段被更新，说明用户什么都没改，直接返回
	if len(setClauses) <= 2 {
		return nil
	}

	//添加where条件
	whereClause := fmt.Sprintf("WHERE id = $%d", paramIndex)
	params = append(params, paperID)

	sqlStr := fmt.Sprintf("UPDATE t_paper SET %s %s", strings.Join(setClauses, ", "), whereClause)

	result, err := tx.Exec(ctx, sqlStr, params...)
	if forceError == "tx.Exec-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error("failed to update paper info: " + err.Error())
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		z.Error(ErrRecordNotFound.Error())
		return ErrRecordNotFound
	}

	return nil
}

func handleDeleteGroup(ctx context.Context, tx pgx.Tx, paperID int64, userID int64, groupID int64) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	// 检查题组是否存在且属于该试卷
	var exists bool
	err := tx.QueryRow(ctx, `
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
		return err
	}
	if !exists {
		err = fmt.Errorf("题组不存在于当前试卷:" + ErrRecordNotFound.Error())
		z.Error(err.Error())
		return err
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
		return err
	}

	return nil
}

func handleAddGroup(ctx context.Context, tx pgx.Tx, paperID int64, userID int64, req AddQuestionGroupRequest) (int64, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	// 插入题组
	var groupID int64
	now := time.Now().UnixMilli()
	const batchInsertPaperQuestionGroupsSQL = `INSERT INTO t_paper_group 
    (paper_id, name, "order", creator, create_time, updated_by, update_time, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := tx.QueryRow(ctx, batchInsertPaperQuestionGroupsSQL,
		paperID,
		req.Name,
		req.Order,
		userID,
		now,
		userID,
		now,
		StatusNormal,
	).Scan(&groupID)
	if forceError == "tx.QueryRow-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return 0, err
	}
	return groupID, nil
}

func handleUpdateGroup(ctx context.Context, tx pgx.Tx, userID int64, req UpdateQuestionsGroupRequest) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	sql := `UPDATE t_paper_group
SET name = $1, updated_by = $2, update_time = $3
WHERE id = $4`
	result, err := tx.Exec(ctx, sql, req.Name, userID, time.Now().UnixMilli(), req.ID)
	if forceError == "tx.Exec-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		z.Error(ErrRecordNotFound.Error())
		return ErrRecordNotFound
	}
	return nil
}

func handleAddQuestions(ctx context.Context, tx pgx.Tx, userID int64, req []AddQuestionsRequest) (map[string]int64, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
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
		err := cmn.Validate(q)
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

	rows, err := tx.Query(ctx, query, args...)
	defer rows.Close()
	if forceError == "tx.Query-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
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
		return map[string]int64{}, err
	}
	idMapping := make(map[string]int64)
	for i, id := range ids {
		idMapping[req[i].TempID] = id
	}
	return idMapping, nil
}

func handleDeleteQuestions(ctx context.Context, tx pgx.Tx, userID int64, questionIDs []int64) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	const sql = `
DELETE FROM t_paper_question tpq 
	WHERE tpq.id = ANY($1::bigint[])`

	result, err := tx.Exec(ctx, sql, questionIDs)
	if forceError == "tx.Exec-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		z.Error(ErrRecordNotFound.Error())
		return ErrRecordNotFound
	}
	return nil
}

func handleUpdateQuestions(ctx context.Context, tx pgx.Tx, userID int64, updates []UpdatePaperQuestionRequest) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	for _, update := range updates {
		//检查结构体
		err := cmn.Validate(update)
		if err != nil {
			z.Error(err.Error())
			return err
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
			err := fmt.Errorf("更新题目没有传入需要更新的字段或传入的字段为零值")
			z.Error(err.Error())
			return err
		}

		// 构建完整 SQL
		query := fmt.Sprintf(
			"UPDATE t_paper_question SET %s WHERE id = $%d",
			strings.Join(setClauses, ", "),
			argIndex,
		)
		args = append(args, update.ID)

		// 执行更新
		result, err := tx.Exec(ctx, query, args...)
		if forceError == "tx.Exec-err" {
			err = errors.New(forceError)
		}
		if err != nil {
			z.Error(err.Error())
			return err
		}
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			z.Error(ErrRecordNotFound.Error())
			return ErrRecordNotFound
		}
	}

	return nil
}

func handleMoveQuestion(ctx context.Context, tx pgx.Tx, paperID, userID int64, orders []int64) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	if len(orders) == 0 {
		z.Error(ErrEmptyQuestionIDs.Error())
		return ErrEmptyQuestionIDs
	}
	// 1. 查询实际题组数量
	var actualCount int
	err := tx.QueryRow(ctx, `
        SELECT question_count FROM v_paper 
        WHERE id = $1 AND status != '02'`,
		paperID).Scan(&actualCount)
	if forceError == "tx.QueryRow-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	// 2. 验证数量
	if len(orders) != actualCount {
		err := fmt.Errorf("数量不匹配，输入%d个ID，实际有%d个题目", len(orders), actualCount)
		z.Error(err.Error())
		return err
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
	result, err := tx.Exec(ctx, sqlStr, orders, userID, now)
	if forceError == "tx.Exec-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		z.Error(ErrRecordNotFound.Error())
		return ErrRecordNotFound
	}
	return nil
}

func handleMoveQuestionGroup(ctx context.Context, tx pgx.Tx, paperID, userID int64, orders []int64) error {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	if len(orders) == 0 {
		z.Error(ErrEmptyGroupID.Error())
		return ErrEmptyGroupID
	}
	// 1. 查询实际题组数量
	var actualCount int
	err := tx.QueryRow(ctx, `
        SELECT group_count FROM v_paper 
        WHERE id = $1 AND status != '02'`,
		paperID).Scan(&actualCount)
	if forceError == "tx.QueryRow-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	// 2. 验证数量
	if len(orders) != actualCount {
		err := fmt.Errorf("数量不匹配，输入%d个ID，实际有%d个题组", len(orders), actualCount)
		z.Error(err.Error())
		return err
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
	result, err := tx.Exec(ctx, sqlStr, orders, userID, now)
	if forceError == "tx.Exec-err" {
		err = errors.New(forceError)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		z.Error(ErrRecordNotFound.Error())
		return ErrRecordNotFound
	}
	return nil
}

func GetManualPaperDetailByPaperID(ctx context.Context, paperID int64) (*cmn.TVPaper, error) {
	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}
	if paperID <= 0 {
		z.Error(ErrInvalidPaperID.Error())
		return nil, ErrInvalidPaperID
	}

	query := `SELECT 
    id,name,assembly_type,category,level,suggested_duration,description,tags,creator,create_time,update_time,status,total_score,question_count,groups_data
	FROM v_paper
	WHERE id = $1
	LIMIT 1`

	db := cmn.GetPgxConn()
	var paper cmn.TVPaper
	err := db.QueryRow(ctx, query, paperID).Scan(
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
		err = errors.New(forceError)
	}
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			z.Error(err.Error())
			return nil, ErrRecordNotFound
		}
		z.Error(err.Error())
		return nil, err
	}
	return &paper, nil
}

func getPaperList(ctx context.Context, req PaperListRequest, userID int64) ([]cmn.TVPaper, int64, error) {
	err := cmn.Validate(req)
	if err != nil {
		return []cmn.TVPaper{}, 0, err
	}
	offset := (req.Page - 1) * req.PageSize
	// 构建动态查询条件
	var whereClauses []string
	var params []interface{}
	paramCount := 1

	// 基础条件：状态为有效
	whereClauses = append(whereClauses, "p.status = '00'")

	//查询当前用户试卷
	whereClauses = append(whereClauses, fmt.Sprintf("p.creator = $%d", paramCount))
	params = append(params, userID)
	paramCount++

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
		whereClauses = append(whereClauses, fmt.Sprintf("p.category = $%d", paramCount))
		params = append(params, req.Category)
		paramCount++
	}

	// 名称模糊查询
	if req.Name != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.name ILIKE $%d", paramCount))
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
		whereClauses = append(whereClauses, fmt.Sprintf("p.tags @> $%d", paramCount))
		tagsJSON, _ := json.Marshal(tags)
		params = append(params, tagsJSON)
		paramCount++
	}

	// 构建WHERE子句
	var whereClause string
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	db := cmn.GetPgxConn()

	// 1. 查询总数
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM v_paper p %s", whereClause)
	var totalCount int64
	err = db.QueryRow(ctx, countSQL, params...).Scan(&totalCount)
	if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRowCount-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("failed to count paper list", zap.Error(err))
		return []cmn.TVPaper{}, 0, err
	}

	// 2. 查询分页数据
	listSQL := fmt.Sprintf(`
		SELECT p.id, p.name, p.assembly_type, p.category, p.level, p.suggested_duration, p.total_score, p.question_count, p.tags, p.create_time, p.update_time, p.status, p.creator, p.creator_info, p.access_mode
		FROM v_paper p
		%s
		ORDER BY p.update_time DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, paramCount, paramCount+1)
	dataParams := append(params, req.PageSize, offset)
	row, err := db.Query(ctx, listSQL, dataParams...)
	if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-QueryRow-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("failed to query paper list", zap.Error(err))
		return []cmn.TVPaper{}, 0, err
	}
	defer row.Close()
	var papers []cmn.TVPaper
	for row.Next() {
		var paper cmn.TVPaper
		err := row.Scan(&paper.ID, &paper.Name, &paper.AssemblyType, &paper.Category, &paper.Level, &paper.SuggestedDuration, &paper.TotalScore, &paper.QuestionCount, &paper.Tags, &paper.CreateTime, &paper.UpdateTime, &paper.Status, &paper.Creator, &paper.CreatorInfo, &paper.AccessMode)
		if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-RowScan-err" {
			err = errors.New(val)
		}
		if err != nil {
			z.Error("failed to scan paper basic info", zap.Error(err))
			return []cmn.TVPaper{}, 0, err
		}
		papers = append(papers, paper)
	}
	err = row.Err()
	if val, ok := ctx.Value("force-error").(string); ok && val == "getPaperList-RowErr-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("rows iteration error", zap.Error(err))
		return []cmn.TVPaper{}, 0, err
	}
	return papers, totalCount, nil
}

func deletePapers(ctx context.Context, tx pgx.Tx, paperIDs []int64, userID int64) error {
	if paperIDs == nil && len(paperIDs) == 0 {
		z.Error(ErrEmptyPaperIDs.Error())
		return ErrEmptyPaperIDs
	}
	now := time.Now().UnixMilli()

	// 1. 软删除 t_paper
	paperSQL := `UPDATE t_paper SET status = $2, updated_by = $3, update_time = $4 WHERE id = ANY($1)`
	_, err := tx.Exec(ctx, paperSQL, paperIDs, StatusUnNormal, userID, now)
	if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapers-exec-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("failed to soft delete t_paper", zap.Error(err))
		return err
	}

	// 2. 软删除 t_paper_group
	groupSQL := `UPDATE t_paper_group SET status = $2, updated_by = $3, update_time = $4 WHERE paper_id = ANY($1)`
	_, err = tx.Exec(ctx, groupSQL, paperIDs, StatusUnNormal, userID, now)
	if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersgroups-exec-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("failed to soft delete t_paper_group", zap.Error(err))
		return err
	}

	// 3. 软删除 t_paper_question
	questionSQL := `UPDATE t_paper_question SET status = $2, updated_by = $3, update_time = $4 WHERE group_id IN (SELECT id FROM t_paper_group WHERE paper_id = ANY($1))`
	_, err = tx.Exec(ctx, questionSQL, paperIDs, StatusUnNormal, userID, now)
	if val, ok := ctx.Value("force-error").(string); ok && val == "deletePapersquestions-exec-err" {
		err = errors.New(val)
	}
	if err != nil {
		z.Error("failed to soft delete t_paper_question", zap.Error(err))
		return err
	}

	return nil
}

//func ValidateExistingPapers(ctx context.Context, tx pgx.Tx, paperIDs []int64) ([]int64, error) {
//	if len(paperIDs) == 0 {
//		z.Error(ErrEmptyPaperIDs.Error())
//		return []int64{}, nil
//	}
//	const sqlString = `SELECT id FROM t_paper WHERE id = ANY($1) AND status = '00'`
//	rows, err := tx.Query(ctx, sqlString, paperIDs)
//	if err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			z.Error(ErrPaperNotFound.Error())
//			return []int64{}, ErrPaperNotFound
//		}
//		z.Error(err.Error())
//		return []int64{}, err
//	}
//	defer rows.Close()
//
//	var validIDs []int64
//	for rows.Next() {
//		var id int64
//		if err := rows.Scan(&id); err != nil {
//			z.Error(err.Error())
//			return []int64{}, err
//		}
//		validIDs = append(validIDs, id)
//	}
//	if err := rows.Err(); err != nil {
//		z.Error(err.Error())
//		return []int64{}, err
//	}
//	return validIDs, nil
//}

//// 检查试卷是否存在
//func paperExists(ctx context.Context, paperID int64) (bool, error) {
//	z.Info("---->" + cmn.FncName())
//
//	if paperID <= 0 {
//		err := fmt.Errorf("无效的试卷ID: %d", paperID)
//		z.Error(err.Error())
//		return false, err
//	}
//
//	conn := cmn.GetPgxConn()
//	var exists bool
//	err := conn.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM t_paper WHERE id=$1 AND status!= '02')", paperID).Scan(&exists)
//	if val, ok := ctx.Value("force-error").(string); ok && val == "paperExists-QueryRow-err" {
//		err = errors.New(val)
//	}
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//
//	return exists, nil
//}

//// ---------------------------------------共享试卷----------------------------------------------
//func getPaperShareInfo(ctx context.Context, tx *pgx.Tx, paperID int64, req GetSharedUserListRequest) ([]cmn.TVPaperShare, int64, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return []cmn.TVPaperShare{}, 0, ErrInvalidPaperID
//	}
//	err := cmn.Validate(req)
//	if err != nil {
//		z.Error(err.Error())
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	offset := (req.Page - 1) * req.PageSize
//
//	// 构建动态 where 条件
//	var whereClauses []string
//	var params []interface{}
//	paramCount := 1
//
//	whereClauses = append(whereClauses, fmt.Sprintf("paper_id = $%d", paramCount))
//	params = append(params, paperID)
//	paramCount++
//
//	if req.Filter != "" {
//		whereClauses = append(whereClauses, fmt.Sprintf("(official_name ILIKE $%d OR mobile_phone ILIKE $%d OR account ILIKE $%d)", paramCount, paramCount, paramCount))
//		params = append(params, "%"+req.Filter+"%")
//		paramCount++
//	}
//
//	var whereClause string
//	if len(whereClauses) > 0 {
//		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
//	}
//
//	// 1. 查询总数
//	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM v_paper_share %s", whereClause)
//	var totalCount int64
//	err = tx.QueryRowContext(ctx, countSQL, params...).Scan(&totalCount)
//	if err != nil {
//		z.Error("failed to count paper share list", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//
//	// 2. 查询分页数据
//	listSQL := fmt.Sprintf(`
//		SELECT user_id, official_name, account, mobile_phone, shared_time
//		FROM v_paper_share
//		%s
//		ORDER BY shared_time DESC
//		LIMIT $%d OFFSET $%d`,
//		whereClause, paramCount, paramCount+1)
//	dataParams := append(params, req.PageSize, offset)
//	rows, err := tx.QueryContext(ctx, listSQL, dataParams...)
//	if err != nil {
//		z.Error("failed to query paper share list", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	defer rows.Close()
//	var shares []cmn.TVPaperShare
//	for rows.Next() {
//		var share cmn.TVPaperShare
//		err := rows.Scan(&share.UserID, &share.OfficialName, &share.Account, &share.MobilePhone, &share.SharedTime)
//		if err != nil {
//			z.Error("failed to scan paper share info", zap.Error(err))
//			return []cmn.TVPaperShare{}, 0, err
//		}
//		shares = append(shares, share)
//	}
//	if err := rows.Err(); err != nil {
//		z.Error("rows iteration error", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	return shares, totalCount, nil
//}
//
//func managePaperShareUsers(ctx context.Context, tx *pgx.Tx, req ManagePaperShareRequest, currentUserID int64) error {
//	// 参数校验
//	if req.PaperID <= 0 || currentUserID <= 0 {
//		return fmt.Errorf("invalid paper id or user id")
//	}
//	err := validateIDs(req.UserIDs)
//	if err != nil {
//		return err
//	}
//
//	now := time.Now().UnixMilli()
//
//	switch req.Action {
//	case "add":
//		valueStrings := make([]string, 0, len(req.UserIDs))
//		valueArgs := make([]interface{}, 0, len(req.UserIDs)*8)
//		for i, userID := range req.UserIDs {
//			base := i * 8
//			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
//				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8))
//			valueArgs = append(valueArgs,
//				PaperResourceShareType, // type
//				req.PaperID,            // resource_id
//				userID,                 // user_id
//				currentUserID,          // creator
//				now,                    // create_time
//				currentUserID,          // updated_by
//				now,                    // update_time
//				StatusNormal,           // status
//			)
//		}
//		query := fmt.Sprintf(`
//			INSERT INTO t_resource_share
//				(type, resource_id, user_id, creator, create_time, updated_by, update_time, status)
//			VALUES %s
//			ON CONFLICT (type, resource_id, user_id) DO UPDATE
//			SET
//				status = EXCLUDED.status,
//				updated_by = EXCLUDED.updated_by,
//				update_time = EXCLUDED.update_time
//		`, strings.Join(valueStrings, ","))
//		result, err := tx.ExecContext(ctx, query, valueArgs...)
//		if err != nil {
//			z.Error(fmt.Sprintf("failed to manage paper share users (add): %v", err))
//			return err
//		}
//		rowsAffected, err := result.RowsAffected()
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		if rowsAffected == 0 {
//			err := fmt.Errorf("no users were shared (paperID: %d)", req.PaperID)
//			z.Error(err.Error())
//			return err
//		}
//
//	case "remove":
//		result, err := tx.ExecContext(ctx,
//			`UPDATE t_resource_share
//			 SET status = $1,
//			     updated_by = $2,
//			     update_time = $3
//			 WHERE type = $4
//			   AND resource_id = $5
//			   AND user_id = ANY($6)
//			   AND status = $7`,
//			StatusUnNormal,         // $1
//			currentUserID,          // $2
//			now,                    // $3
//			PaperResourceShareType, // $4
//			req.PaperID,            // $5
//			pq.Array(req.UserIDs),  // $6
//			StatusNormal,           // $7
//		)
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		rowsAffected, err := result.RowsAffected()
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		if rowsAffected != int64(len(req.UserIDs)) {
//			err := fmt.Errorf("only %d of %d users removed (paperID: %d)",
//				rowsAffected, len(req.UserIDs), req.PaperID)
//			z.Error(err.Error())
//			return err
//		}
//	default:
//		err := fmt.Errorf("invalid action")
//		z.Error(err.Error())
//		return err
//	}
//	return nil
//}
//
//func updatePaperShareStatus(ctx context.Context, tx *pgx.Tx, req UpdatePaperAccessModeRequest, currentUserID int64) error {
//	now := time.Now().UnixMilli()
//	sql := `UPDATE t_resource_share
//	SET status = $1,
//		updated_by = $2,
//		update_time = $3
//		WHERE type = $4
//		AND resource_id = $5
//		`
//	result, err := tx.ExecContext(ctx, sql, req.AccessMode, currentUserID, now, PaperResourceShareType, req.PaperID)
//	if err != nil {
//		z.Error(err.Error())
//		return err
//	}
//	rowsAffected, err := result.RowsAffected()
//	if err != nil {
//		z.Error(err.Error())
//		return err
//	}
//	if rowsAffected == 0 {
//		err := fmt.Errorf("no users were shared (paperID: %d)", req.PaperID)
//		z.Error(err.Error())
//		return err
//	}
//	return nil
//}
//
//func validateUserPermissions(ctx context.Context, tx *pgx.Tx, paperID, userID int64) (bool, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return false, ErrInvalidPaperID
//	}
//	if userID <= 0 {
//		z.Error(ErrInvalidUserID.Error())
//		return false, ErrInvalidUserID
//	}
//	sqlString := `
//	SELECT EXISTS(
//	SELECT 1
//	FROM t_paper
//	WHERE id = $1 AND (
//		creator = $2
//		OR access_mode = '04'
//		OR (
//			access_mode = '02'
//			AND EXISTS(
//				SELECT 1 FROM t_paper_share WHERE paper_id = $1 AND user_id = $2 AND status = '00'
//			)
//		)
//	)
//	)
//	`
//	var result bool
//	err := tx.QueryRowContext(ctx, sqlString, paperID, userID).Scan(&result)
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//	return result, nil
//}
//
//func validateUserIsPaperCreator(ctx context.Context, tx *pgx.Tx, paperID, userID int64) (bool, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return false, ErrInvalidPaperID
//
//	}
//	if userID <= 0 {
//		z.Error(ErrInvalidUserID.Error())
//		return false, ErrInvalidUserID
//	}
//	sql := `
//SELECT EXISTS(
//SELECT 1
//FROM t_paper WHERE id = $1 AND creator = $2)`
//	var result bool
//	err := tx.QueryRowContext(ctx, sql, paperID, userID).Scan(&result)
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//	return result, nil
//}
