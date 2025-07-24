package question_bank

//annotation:template-service
//author:{"name":"cpf","tel":"15817621370", "email":"3410304292@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
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
		Fn: questionBanks,

		Path: "/question-banks",
		Name: "question-banks",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: questions,

		Path: "/questions",
		Name: "questions",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// 获取题库列表
func getQuestionBankList(ctx context.Context, conn *pgxpool.Pool, params QueryQuestionBankParams) (list []cmn.TQuestionBank, rowCount int64, err error) {
	if ctx == nil {
		err := fmt.Errorf("ctx is nil")
		z.Error(err.Error())
		return nil, 0, err
	}
	if conn == nil {
		err := fmt.Errorf("conn is nil")
		z.Error(err.Error())
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	// 基础状态过滤（必须条件）
	conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
	args = append(args, "00")
	argIndex++

	// 关键词过滤
	if params.Keyword != "" {
		keywordCondition := fmt.Sprintf("(name LIKE $%d OR tags @> $%d)", argIndex, argIndex+1)
		conditions = append(conditions, keywordCondition)
		args = append(args, "%"+params.Keyword+"%")
		args = append(args, fmt.Sprintf(`["%s"]`, params.Keyword))
		argIndex += 2
	}

	// bankID过滤
	if params.BankID > 0 {
		conditions = append(conditions, fmt.Sprintf("id = $%d", argIndex))
		args = append(args, params.BankID)
		argIndex++
	}

	// 权限过滤TODO

	// 构建完整的WHERE子句
	var whereClause string
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// 总数查询
	s1 := "SELECT COUNT(*) FROM t_question_bank" + whereClause
	err = conn.QueryRow(ctx, s1, args...).Scan(&rowCount)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}

	// 数据查询
	s2 := fmt.Sprintf(`
		SELECT
			id,
			type,
			name,
			tags,
			repos,
			default_repo,
			creator,
			create_time,
			updated_by,
			update_time,
			remark,
			status,
			question_count,
			access_mode
		FROM t_question_bank
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)
	rows, err := conn.Query(ctx, s2, args...)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var q cmn.TQuestionBank
		err = rows.Scan(
			&q.ID,
			&q.Type,
			&q.Name,
			&q.Tags,
			&q.Repos,
			&q.DefaultRepo,
			&q.Creator,
			&q.CreateTime,
			&q.UpdatedBy,
			&q.UpdateTime,
			&q.Remark,
			&q.Status,
			&q.QuestionCount,
			&q.AccessMode,
		)
		if err != nil {
			err = fmt.Errorf("rows.Scan error: %s", err.Error())
			z.Error(err.Error())
			return nil, 0, err
		}
		list = append(list, q)
	}

	if rows.Err() != nil {
		err = fmt.Errorf("rows.Err error: %s", rows.Err().Error())
		z.Error(err.Error())
		return nil, 0, err
	}

	return list, rowCount, nil
}

// 是否有权限修改题库
func hasPermissionToEditBank(bankID int64, userID int64) (exists bool, err error) {
	if bankID <= 0 || userID <= 0 {
		return false, fmt.Errorf("invalid bankID or userID: %d, %d", bankID, userID)
	}

	params := []interface{}{bankID, userID}
	conn := cmn.GetPgxConn()
	s := "select count(id) from t_question_bank where id = $1 and creator = $2"
	var count int
	err = conn.QueryRow(context.Background(), s, params...).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 题库接口
func questionBanks(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	conn := cmn.GetPgxConn()

	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 获取查询参数
		keyword := q.R.URL.Query().Get("keyword")
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")
		bankIDStr := q.R.URL.Query().Get("bankID")

		// 设置默认分页参数
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "99"
		}
		page, _ := strconv.ParseInt(pageStr, 10, 64)
		pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)

		userID := null.IntFrom(1000)
		if q.SysUser != nil {
			userID = q.SysUser.ID
		}
		//level := "00" // 权限等级，"00" 表示全局权限（TODO: 替换为真实来源）

		var bankID int64
		if bankIDStr != "" {
			bankID, q.Err = strconv.ParseInt(bankIDStr, 10, 64)
			if q.Err != nil || bankID <= 0 {
				q.Err = fmt.Errorf("invalid bankID: %s", bankIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		params := QueryQuestionBankParams{
			BankID:   bankID,
			Keyword:  keyword,
			Page:     page,
			PageSize: pageSize,
			UserID:   userID.Int64,
		}

		list, rowCount, err := getQuestionBankList(ctx, conn, params)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		jsonData, err := json.Marshal(list)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Data = jsonData
		q.Msg.Msg = "success"
		q.Msg.RowCount = rowCount
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
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		questionBankName := gjson.GetBytes(buf, "data.name").String()
		if questionBankName == "" {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		questionBankType := gjson.GetBytes(buf, "data.type").String()
		if questionBankType == "" {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank type")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if questionBankType != QuestionBankTypeTheory && questionBankType != QuestionBankTypeCoding {
			q.Err = fmt.Errorf("call /api/question-banks with unsupported question bank type: %s", questionBankType)
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

		var bank cmn.TQuestionBank
		bank.TableMap = &bank
		q.Err = json.Unmarshal(qry.Data, &bank)
		if q.Err != nil {
			z.Info(string(qry.Data))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 设置创建者
		userID := null.IntFrom(1000)
		if q.SysUser != nil {
			userID = q.SysUser.ID
		}
		bank.Creator = userID

		// 校验字段
		q.Err = cmn.InvalidEmptyNullValue(&bank)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 写库
		qry.Action = "insert"
		q.Err = cmn.DML(&bank.Filter, &qry)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 获取返回ID
		bankID, ok := bank.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be int64, but it isn't")
			q.RespErr()
			return
		}
		bank.ID = null.IntFrom(bankID)

		// 返回响应
		buf, q.Err = cmn.MarshalJSON(&bank)
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Msg.Data = buf
		q.Msg.Msg = "success"
		q.Resp()

	case "put":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		questionBankID := gjson.GetBytes(buf, "data.id").Int()
		if questionBankID <= 0 {
			q.Err = fmt.Errorf("call /api/question-banks with invalid question bank ID: %d", questionBankID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		questionBankName := gjson.GetBytes(buf, "data.name").String()
		questionBankTags := gjson.GetBytes(buf, "data.tags").Array()
		if questionBankName == "" && len(questionBankTags) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name and tags")
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

		var bank cmn.TQuestionBank
		bank.TableMap = &bank
		q.Err = json.Unmarshal(qry.Data, &bank)
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

		if CurrentLevelSign != GlobalLevelSign {
			// 不具备全局权限时，检查是否有权限修改题库
			var exists bool
			exists, q.Err = hasPermissionToEditBank(questionBankID, userID.Int64)
			if q.Err != nil {
				q.RespErr()
				return
			}
			if !exists {
				q.Err = fmt.Errorf("call /api/question-banks with invalid question bank ID: %d", questionBankID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 设置更新者以及更新时间（毫秒级时间戳）
		bank.UpdatedBy = userID
		t := cmn.GetNowInMS()
		bank.UpdateTime = null.NewInt(t, true)

		filters := []map[string]interface{}{
			{"ID": map[string]interface{}{"EQ": questionBankID}},
		}

		qry.Action = "update"
		qry.Filter = filters
		q.Err = cmn.DML(&bank.Filter, &qry)
		if q.Err != nil {
			q.RespErr()
			return
		}

		rowAffected, ok := bank.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("_, ok = c.filter.qryResult.(string) should be ok while it's not")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if rowAffected == 0 {
			q.Err = fmt.Errorf("no rows affected, maybe the question bank does not exist or you do not have permission to edit it")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"RowAffected":%d}`, rowAffected))
		q.Msg.Msg = "success"
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

func validateQuestion(question *cmn.TQuestion) (valid bool, err error) {
	if question == nil {
		err = fmt.Errorf("question cannot be nil")
		z.Error(err.Error())
		return false, err
	}
	_, ok := QuestionTypes[question.Type]
	if !ok {
		err = fmt.Errorf("unsupported question type: %s", question.Type)
		z.Error(err.Error())
		return false, err
	}
	_, ok = QuestionDifficulty[question.Difficulty]
	if !ok {
		err = fmt.Errorf("unsupported question difficulty: %v", question.Difficulty)
		z.Error(err.Error())
		return false, err
	}
	if question.Score.Float64 <= 0 {
		err = fmt.Errorf("question score must be greater than zero")
		z.Error(err.Error())
		return false, err
	}
	if question.BelongTo.Int64 <= 0 {
		err = fmt.Errorf("question belongTo must be greater than zero")
		z.Error(err.Error())
		return false, err
	}

	switch question.Type {
	case "00", "02", "04":
		var options []QuestionOption
		err = json.Unmarshal(question.Options, &options)
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
		if len(options) <= 2 {
			err = fmt.Errorf("question options must have at least two options")
			z.Error(err.Error())
			return false, err
		}
		for _, option := range options {
			if option.Label == "" || option.Value == "" {
				err = fmt.Errorf("question options must have non-empty label and value")
				z.Error(err.Error())
				return false, err
			}
		}
	case "06", "08":
		var answers []SubjectiveAnswer
		err = json.Unmarshal(question.Options, &answers)
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
		for _, answer := range answers {
			if answer.Index < 1 || answer.Score <= 0 || answer.Answer == "" || answer.GradingRule == "" {
				err = fmt.Errorf("subjective question must have non-empty index, score, answer and grading rule")
				z.Error(err.Error())
				return false, err
			}
		}
	}

	return true, nil
}

func updateBankQuestionCount(ctx context.Context, conn *pgxpool.Pool, questionBankID int64, count int64, updatedBy int64) error {
	if ctx == nil {
		err := fmt.Errorf("ctx is nil")
		z.Error(err.Error())
		return err
	}
	if conn == nil {
		err := fmt.Errorf("conn is nil")
		z.Error(err.Error())
		return err
	}
	if questionBankID <= 0 {
		err := fmt.Errorf("questionBankID must be greater than zero")
		z.Error(err.Error())
		return err
	}
	if count <= 0 {
		err := fmt.Errorf("count must be greater than zero")
		z.Error(err.Error())
		return err
	}
	if updatedBy <= 0 {
		err := fmt.Errorf("updatedBy must be greater than zero")
		z.Error(err.Error())
		return err
	}
	t := cmn.GetNowInMS()
	UpdateTime := null.NewInt(t, true)

	s := `
	UPDATE t_question_bank
	SET question_count = $1, updated_by = $2, update_time = $3
	WHERE id = $4
	`
	_, err := conn.Exec(ctx, s, count, updatedBy, UpdateTime, questionBankID)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	return nil
}

func getQuestionList(ctx context.Context, conn *pgxpool.Pool, params QueryQuestionsParams) (list []cmn.TQuestion, rowCount int64, err error) {
	if ctx == nil {
		err := fmt.Errorf("ctx is nil")
		z.Error(err.Error())
		return nil, 0, err
	}
	if conn == nil {
		err := fmt.Errorf("conn is nil")
		z.Error(err.Error())
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	// 基础状态过滤（必须条件）
	conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
	args = append(args, "00")
	argIndex++

	// 名称过滤
	if params.Name != "" {
		keywordCondition := fmt.Sprintf("(name LIKE $%d)", argIndex)
		conditions = append(conditions, keywordCondition)
		args = append(args, "%"+params.Name+"%")
		argIndex += 1
	}

	// 标签过滤
	if params.Tags != "" {
		keywordCondition := fmt.Sprintf("(tags @> $%d)", argIndex)
		conditions = append(conditions, keywordCondition)
		args = append(args, fmt.Sprintf(`["%s"]`, params.Tags))
		argIndex += 1
	}

	// 类型过滤
	if params.Type != "" {
		// 类型判断
		_, ok := QuestionTypes[params.Type]
		if !ok {
			err = fmt.Errorf("invalid type: %s", params.Type)
			z.Error(err.Error())
			return nil, 0, err
		}
		keywordCondition := fmt.Sprintf("(type = $%d)", argIndex)
		conditions = append(conditions, keywordCondition)
		args = append(args, params.Type)
		argIndex += 1
	}

	// 难度过滤
	if params.Difficulty.Valid && params.Difficulty.Int64 != 0 {
		// 难度判断
		_, ok := QuestionDifficulty[params.Difficulty]
		if !ok {
			err = fmt.Errorf("invalid difficulty: %d", params.Difficulty.Int64)
			z.Error(err.Error())
			return nil, 0, err
		}
		keywordCondition := fmt.Sprintf("(difficulty = $%d)", argIndex)
		conditions = append(conditions, keywordCondition)
		args = append(args, params.Difficulty.Int64)
		argIndex += 1
	}

	// 权限过滤TODO

	// 构建完整的WHERE子句
	var whereClause string
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// 总数查询
	s1 := "SELECT COUNT(*) FROM t_question" + whereClause
	err = conn.QueryRow(ctx, s1, args...).Scan(&rowCount)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}

	// 数据查询
	s2 := fmt.Sprintf(`
		SELECT
			id,
			type,
			content,
			options,
			answers,
			score,
			difficulty,
			tags,
			analysis,
			title,
			answer_file_path,
			test_file_path,
			input,
			output,
			example,
			repo,
			"order",
			creator,
			create_time,
			updated_by,
			update_time,
			addi,
			status,
			question_attachments_path,
			access_mode,
			belong_to
		FROM t_question
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)
	rows, err := conn.Query(ctx, s2, args...)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var q cmn.TQuestion
		err = rows.Scan(
			&q.ID,
			&q.Type,
			&q.Content,
			&q.Options,
			&q.Answers,
			&q.Score,
			&q.Difficulty,
			&q.Tags,
			&q.Analysis,
			&q.Title,
			&q.AnswerFilePath,
			&q.TestFilePath,
			&q.Input,
			&q.Output,
			&q.Example,
			&q.Repo,
			&q.Order,
			&q.Creator,
			&q.CreateTime,
			&q.UpdatedBy,
			&q.UpdateTime,
			&q.Addi,
			&q.Status,
			&q.QuestionAttachmentsPath,
			&q.AccessMode,
			&q.BelongTo,
		)
		if err != nil {
			err = fmt.Errorf("rows.Scan error: %s", err.Error())
			z.Error(err.Error())
			return nil, 0, err
		}
		list = append(list, q)
	}

	if rows.Err() != nil {
		err = fmt.Errorf("rows.Err error: %s", rows.Err().Error())
		z.Error(err.Error())
		return nil, 0, err
	}

	return list, rowCount, nil
}

// Questions 接口
func questions(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	conn := cmn.GetPgxConn()

	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 获取查询参数
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")
		bankIDStr := q.R.URL.Query().Get("bankID")
		name := q.R.URL.Query().Get("name")
		tags := q.R.URL.Query().Get("tags")
		questionType := q.R.URL.Query().Get("type")
		difficulty := q.R.URL.Query().Get("difficulty")

		// 设置默认分页参数
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "10"
		}
		page, _ := strconv.ParseInt(pageStr, 10, 64)
		pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)

		userID := null.IntFrom(1000)
		if q.SysUser != nil {
			userID = q.SysUser.ID
		}
		//level := "00" // 权限等级，"00" 表示全局权限（TODO: 替换为真实来源）

		if bankIDStr == "" {
			q.Err = fmt.Errorf("bankID is empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		bankID, _ := strconv.ParseInt(bankIDStr, 10, 64)
		difficultyInt, _ := strconv.ParseInt(difficulty, 10, 64)

		params := QueryQuestionsParams{
			BankID:     bankID,
			Name:       name,
			Tags:       tags,
			Type:       questionType,
			Difficulty: null.IntFrom(difficultyInt),
			Page:       page,
			PageSize:   pageSize,
			UserID:     userID.Int64,
		}

		list, rowCount, err := getQuestionList(ctx, conn, params)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		jsonData, err := json.Marshal(list)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Data = jsonData
		q.Msg.Msg = "success"
		q.Msg.RowCount = rowCount
		q.Resp()

	case "post":
		// 处理 POST 请求
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/questions with empty body")
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

		var questions []cmn.TQuestion
		q.Err = json.Unmarshal(qry.Data, &questions)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userID := null.IntFrom(1000)
		if q.SysUser != nil {
			userID = q.SysUser.ID
		}

		var insertQuestions []cmn.TQuestion
		for _, question := range questions {
			valid, err := validateQuestion(&question)
			if !valid && err != nil {
				q.Err = err
				q.RespErr()
				return
			}
			question.TableMap = &question
			question.Creator = userID

			q.Err = cmn.InvalidEmptyNullValue(&question)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 写库
			qry.Action = "insert"
			qry.Data, _ = json.Marshal(question)
			q.Err = cmn.DML(&question.Filter, &qry)
			if q.Err != nil {
				q.RespErr()
				return
			}

			ID, ok := question.QryResult.(int64)
			if !ok {
				q.Err = fmt.Errorf("qryResult should be int64")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			question.ID = null.IntFrom(ID)

			insertQuestions = append(insertQuestions, question)
		}

		// 同步更新对于题库
		count := int64(len(insertQuestions))
		bankID := insertQuestions[0].BelongTo.Int64
		q.Err = updateBankQuestionCount(ctx, conn, bankID, count, userID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var result []json.RawMessage
		for _, Q := range insertQuestions {
			b, err := cmn.MarshalJSON(&Q)
			if err != nil {
				z.Error(err.Error())
				q.Err = err
				q.RespErr()
				return
			}
			result = append(result, b)
		}
		// 返回插入后的所有记录
		buf, q.Err = json.Marshal(result)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Data = buf
		q.Msg.Msg = "success"
		q.Resp()

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}
