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

	"github.com/lib/pq"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
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

	/* 题库相关接口
	 */

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

	/* 题目相关接口
	 */

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

// 题库接口
func questionBanks(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}

	// 设置创建者
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid userID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 检查权限
	userDomains := q.Domains     //用户权限域
	role := q.SysUser.Role.Int64 //用户角色编号
	var userDoamin string
	for _, d := range userDomains {
		if d.ID.Valid && d.ID.Int64 == role {
			userDoamin = d.Domain
			break
		}
	}
	// 判断是否在允许范围内
	isAllowed := isAllowedDomain(userDoamin)

	if !isAllowed {
		q.Err = fmt.Errorf("domain %s is not allowed", userDoamin)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
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
		page, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing page: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing pageSize: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var bankID int64
		if bankIDStr != "" {
			bankID, err = strconv.ParseInt(bankIDStr, 10, 64)
			if err != nil {
				q.Err = fmt.Errorf("error parsing bankID: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if bankID <= 0 {
				q.Err = fmt.Errorf("invalid bankID: %d", bankID)
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
			Creator:  userID,
		}

		var rowCount int64
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

		// 检查是否有creator
		if params.Creator > 0 {
			conditions = append(conditions, fmt.Sprintf("creator = $%d", argIndex))
			args = append(args, params.Creator)
			argIndex++
		}

		// 构建完整的WHERE子句
		var whereClause string
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " AND ")
		}

		// 总数查询
		s1 := "SELECT COUNT(*) FROM t_question_bank" + whereClause
		q.Err = conn.QueryRow(ctx, s1, args...).Scan(&rowCount)
		if forceError == "conn.QueryRow" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 数据查询
		s2 := fmt.Sprintf(`
		SELECT
			id,
			name,
			type,
			tags,
			creator,
			official_name,
			create_time,
			update_time,
			question_count,
			question_types,
			question_difficulties,
			question_tags,
			status
		FROM v_question_bank
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
		offset := (params.Page - 1) * params.PageSize
		args = append(args, params.PageSize, offset)
		var rows pgx.Rows
		rows, q.Err = conn.Query(ctx, s2, args...)
		defer rows.Close()
		if forceError == "conn.Query" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var list []cmn.TVQuestionBank
		for rows.Next() {
			var bank cmn.TVQuestionBank
			err = rows.Scan(
				&bank.ID,
				&bank.Name,
				&bank.Type,
				&bank.Tags,
				&bank.Creator,
				&bank.OfficialName,
				&bank.CreateTime,
				&bank.UpdateTime,
				&bank.QuestionCount,
				&bank.QuestionTypes,
				&bank.QuestionDifficulties,
				&bank.QuestionTags,
				&bank.Status,
			)
			if forceError == "rows.Scan" {
				err = errors.New(forceError)
			}
			if err != nil {
				q.Err = fmt.Errorf("rows.Scan error: %s", err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			list = append(list, bank)
		}

		q.Err = rows.Err()
		if forceError == "rows.Err()" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var jsonData []byte
		jsonData, q.Err = json.Marshal(list)
		if forceError == "json.Marshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Data = jsonData
		q.Msg.Msg = "success"
		q.Msg.RowCount = rowCount
	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io-ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
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

		questionBankName := gjson.GetBytes(qry.Data, "name").String()
		if questionBankName == "" {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		questionBankType := gjson.GetBytes(qry.Data, "type").String()
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

		var bank cmn.TQuestionBank
		bank.TableMap = &bank
		q.Err = json.Unmarshal(qry.Data, &bank)
		if forceError == "json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		bank.Creator = null.IntFrom(userID)

		//设置所属域
		bank.DomainID = null.IntFrom(1999)

		// 写库
		qry.Action = "insert"
		q.Err = cmn.DML(&bank.Filter, &qry)
		if forceError == "cmn.DML" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取返回ID
		bankID, ok := bank.QryResult.(int64)
		if forceError == "bank.QryResult.bankID" {
			ok = false
		}
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be int64, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		bank.ID = null.IntFrom(bankID)

		// 返回响应
		buf, q.Err = cmn.MarshalJSON(&bank)
		if forceError == "cmn.MarshalJSON" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = buf
		q.Msg.Msg = "success"
	case "put":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
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

		questionBankID := gjson.GetBytes(buf, "data.id").Int()
		if questionBankID <= 0 {
			q.Err = fmt.Errorf("call /api/question-banks with invalid question bank ID: %d", questionBankID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		questionBankName := gjson.GetBytes(buf, "data.name").String()
		questionBankTagsJson := gjson.GetBytes(buf, "data.tags")
		var questionBankTags []gjson.Result
		if questionBankTagsJson.Exists() {
			questionBankTags = questionBankTagsJson.Array()
		}

		if questionBankName == "" && len(questionBankTags) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name and tags")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开始构建更新SQL语句
		updateSQL := "UPDATE t_question_bank SET "
		var args []interface{}
		var setClauses []string
		argIndex := 1

		// 添加更新时间
		t := cmn.GetNowInMS()
		setClauses = append(setClauses, fmt.Sprintf("update_time = $%d", argIndex))
		args = append(args, t)
		argIndex++

		// 添加更新用户
		setClauses = append(setClauses, fmt.Sprintf("updated_by = $%d", argIndex))
		args = append(args, userID)
		argIndex++

		// 如果提供了题库名称，则更新
		if questionBankName != "" {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, questionBankName)
			argIndex++
		}

		// 如果提供了题库标签，则更新
		if len(questionBankTags) > 0 {
			// 将gjson.Result数组转换为字符串数组
			var strTags []string
			for _, tag := range questionBankTags {
				strTags = append(strTags, tag.String())
			}

			// 将标签数组序列化为JSON
			tagsJSON, err := json.Marshal(strTags)
			if forceError == "json.Marshal" {
				err = errors.New(forceError)
			}
			if err != nil {
				q.Err = fmt.Errorf("无法序列化标签: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIndex))
			args = append(args, tagsJSON)
			argIndex++
		}

		// 完成SQL语句
		updateSQL += strings.Join(setClauses, ", ")
		updateSQL += fmt.Sprintf(" WHERE id = $%d", argIndex)
		args = append(args, questionBankID)

		// 执行更新操作
		var commandTag pgconn.CommandTag
		commandTag, q.Err = conn.Exec(ctx, updateSQL, args...)
		if forceError == "conn.Exec" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			q.Err = fmt.Errorf("更新题库失败: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if commandTag.RowsAffected() == 0 {
			q.Err = fmt.Errorf("更新题库失败：没有记录被更新")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"RowAffected":%d}`, commandTag.RowsAffected()))
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
		return
	case "delete":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
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
		var deleteBankIDs []int64
		q.Err = json.Unmarshal(qry.Data, &deleteBankIDs)
		if forceError == "json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//检测ID数组
		q.Err = validateIDs(deleteBankIDs)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//执行删除操作
		//题库直接进行删除操作，题库题目级联删除，相关的试卷题目也级联删除,但要把关联试卷的版本号加1
		//以上操作需要在数据库事务中进行且需要确保原子性
		//如果其中任何一步失败，则整个事务回滚
		//删除的时候需要判断是否是题库创建者
		var tx pgx.Tx
		tx, q.Err = conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if forceError == "conn.BeginTx" {
			q.Err = errors.New(forceError)
			_ = tx.Rollback(ctx)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if p := recover(); p != nil {
				panicErr := fmt.Errorf("panic occurred: %v", p)
				z.Error(panicErr.Error())
				err := tx.Rollback(ctx)
				// 强制错误，用于测试
				if forceError == "recover" {
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
				if forceError == "tx.Rollback" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil && !errors.Is(q.Err, pgx.ErrTxClosed) {
					z.Error(err.Error())
				}
				return
			}
			err := tx.Commit(ctx)
			if forceError == "tx.Commit" {
				err = errors.New(forceError)
				q.Err = err
				q.RespErr()
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()
		if forceError == "recover" {
			panic(errors.New(forceError))
		}
		if forceError == "tx.Commit" {
			return
		}
		if forceError == "tx.Rollback" {
			q.Err = errors.New(forceError)
			return
		}
		// 这里是执行具体的删除操作
		// 1. 检查题库是否存在
		// 检查每个题库的权限
		var checkSQL string
		var errorMessages []string
		checkSQL = `
			SELECT COALESCE(array_agg(
				CASE 
					WHEN tqb.id IS NULL THEN '题库(' || ids.id || ')不存在'
					WHEN tqb.creator != $2 THEN '题库(' || COALESCE(tqb.name, '未知') || ')非题库创建者，无删除权限'
					ELSE NULL 
				END
			) FILTER (WHERE CASE 
					WHEN tqb.id IS NULL THEN '题库(' || ids.id || ')不存在'
					WHEN tqb.creator != $2 THEN '题库(' || COALESCE(tqb.name, '未知') || ')非题库创建者，无删除权限'
					ELSE NULL 
				END IS NOT NULL), ARRAY[]::text[]) as error_messages
			FROM unnest($1::bigint[]) AS ids(id)
			LEFT JOIN t_question_bank tqb ON tqb.id = ids.id
			WHERE tqb.id IS NULL OR tqb.creator != $2`
		q.Err = tx.QueryRow(ctx, checkSQL, deleteBankIDs, userID).Scan(&errorMessages)
		if forceError == "tx.QueryRow" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
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

		// 如果有任何不能删除的题库，返回错误
		if validErrors.Len() > 0 {
			q.Msg.Status = -1
			q.Err = errors.New(validErrors.String())
			q.RespErr()
			return
		}
		// 3. 删除题库
		//直接硬删除题库，关联的题库题目以及关联题库题目的试卷题目会被级联删除
		deleteBankSQL := `
			DELETE FROM t_question_bank
			WHERE id = ANY($1::bigint[])
		`
		_, q.Err = tx.Exec(ctx, deleteBankSQL, deleteBankIDs)
		if forceError == "tx.Exec" {
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
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func validateQuestion(question *cmn.TQuestion) (valid bool, err error) {
	if question == nil {
		err = fmt.Errorf("question cannot be nil")
		z.Error(err.Error())
		return false, err
	}

	// 基础字段验证
	_, ok := QuestionTypes[question.Type]
	if !ok {
		err = fmt.Errorf("unsupported question type: %s", question.Type)
		z.Error(err.Error())
		return false, err
	}

	_, ok = QuestionDifficulty[question.Difficulty.Int64]
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

	// 验证题目内容不能为空
	if !question.Content.Valid || strings.TrimSpace(question.Content.String) == "" {
		err = fmt.Errorf("question content cannot be empty")
		z.Error(err.Error())
		return false, err
	}

	if len(question.Tags) > 0 {
		var tags []string
		err = json.Unmarshal(question.Tags, &tags)
		if err != nil {
			err = fmt.Errorf("question tags format invalid: %v", err)
			z.Error(err.Error())
			return false, err
		}
		if len(tags) > 0 {
			for _, tag := range tags {
				if strings.TrimSpace(tag) == "" {
					err = fmt.Errorf("question tags cannot contain empty values")
					z.Error(err.Error())
					return false, err
				}
			}
		}

	}

	// 根据题型进行细化验证
	switch question.Type {
	case QuestionTypeSingleChoice: // 单选题
		return validateSingleChoiceQuestion(question)
	case QuestionTypeMultipleChoice: // 多选题
		return validateMultipleChoiceQuestion(question)
	case QuestionTypeTrueFalse: // 判断题
		return validateTrueFalseQuestion(question)
	case QuestionTypeFillInBlank: // 填空题
		return validateFillInBlankQuestion(question)
	case QuestionTypeEssay: // 简答题
		return validateEssayQuestion(question)
	default:
		err = fmt.Errorf("unsupported question type for validation: %s", question.Type)
		z.Error(err.Error())
		return false, err
	}
}

// Questions 接口
func questions(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}

	// 检查权限
	userDomains := q.Domains     //用户权限域
	role := q.SysUser.Role.Int64 //用户角色编号
	var userDoamin string
	for _, d := range userDomains {
		if d.ID.Valid && d.ID.Int64 == role {
			userDoamin = d.Domain
			break
		}
	}
	// 判断是否在允许范围内
	isAllowed := isAllowedDomain(userDoamin)

	if !isAllowed {
		q.Err = fmt.Errorf("domain %s is not allowed", userDoamin)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	conn := cmn.GetPgxConn()

	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 获取查询参数
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")
		bankIDStr := q.R.URL.Query().Get("bankID")
		content := q.R.URL.Query().Get("content")
		tagsParams := q.R.URL.Query()["tags"] // 允许获取多个tag参数
		questionIDStr := q.R.URL.Query().Get("questionID")
		var tags []string
		for _, t := range tagsParams {
			if t != "" {
				tags = append(tags, t)
			}
		}
		// 获取type参数并过滤空值
		typeParams := q.R.URL.Query()["type"]
		var questionTypes []string
		for _, t := range typeParams {
			if t != "" {
				questionTypes = append(questionTypes, t)
			}
		}
		difficultyStrs := q.R.URL.Query()["difficulty"] // 允许获取多个difficulty参数

		var difficultyList []int64
		// 只有当传入difficulty参数时才进行解析
		if len(difficultyStrs) > 0 {
			for _, d := range difficultyStrs {
				if d != "" { // 跳过空值
					val, err := strconv.ParseInt(d, 10, 64)
					if err != nil {
						q.Err = fmt.Errorf("invalid difficulty: %v", d)
						q.RespErr()
						return
					}
					difficultyList = append(difficultyList, val)
				}
			}
		}

		if bankIDStr == "" {
			q.Err = fmt.Errorf("bankID is empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		bankID, err := strconv.ParseInt(bankIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("invalid bankID: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var questionID int64
		if questionIDStr != "" {
			questionID, err = strconv.ParseInt(questionIDStr, 10, 64)
			if err != nil {
				q.Err = fmt.Errorf("invalid questionID: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 设置默认分页参数
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "10"
		}
		page, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("invalid page: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("invalid pageSize: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		params := QueryQuestionsParams{
			BankID:     bankID,
			QuestionID: questionID,
			Content:    content,
			Tags:       tags,
			Type:       questionTypes,
			Difficulty: difficultyList,
			Page:       page,
			PageSize:   pageSize,
		}

		var rowCount int64
		var conditions []string
		var args []interface{}
		argIndex := 1

		// 基础状态过滤（必须条件）
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, "00")
		argIndex++

		// 题库过滤(必须)
		c := fmt.Sprintf("(belong_to = $%d)", argIndex)
		conditions = append(conditions, c)
		args = append(args, params.BankID)
		argIndex += 1

		// 题目ID过滤
		if params.QuestionID > 0 {
			conditions = append(conditions, fmt.Sprintf("id = $%d", argIndex))
			args = append(args, params.QuestionID)
			argIndex += 1
		}

		// 题目内容过滤
		if params.Content != "" {
			c := fmt.Sprintf("(content ILIKE $%d)", argIndex)
			conditions = append(conditions, c)
			args = append(args, "%"+params.Content+"%")
			argIndex += 1
		}

		// 标签过滤
		if len(params.Tags) > 0 {
			c := fmt.Sprintf("tags ?| $%d", argIndex)
			args = append(args, pq.Array(params.Tags))
			conditions = append(conditions, c)
			argIndex++
		}

		// 类型过滤
		if len(params.Type) > 0 {
			// 校验合法性
			for _, t := range params.Type {
				if _, ok := QuestionTypes[t]; !ok {
					q.Err = fmt.Errorf("invalid type: %s", t)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			placeholders := make([]string, len(params.Type))
			for i := range params.Type {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, params.Type[i])
				argIndex++
			}
			c := fmt.Sprintf("(type IN (%s))", strings.Join(placeholders, ","))
			conditions = append(conditions, c)
		}

		// 难度过滤
		if len(params.Difficulty) > 0 {
			validDifficulties := make([]int64, 0)
			// 校验合法性并只添加有效的难度值
			for _, d := range params.Difficulty {
				if _, ok := QuestionDifficulty[d]; ok {
					validDifficulties = append(validDifficulties, d)
				}
			}

			// 只有在有有效难度值时才添加条件
			if len(validDifficulties) > 0 {
				placeholders := make([]string, len(validDifficulties))
				for i := range validDifficulties {
					placeholders[i] = fmt.Sprintf("$%d", argIndex)
					args = append(args, validDifficulties[i])
					argIndex++
				}
				c := fmt.Sprintf("(difficulty IN (%s))", strings.Join(placeholders, ","))
				conditions = append(conditions, c)
			}
		}

		// 构建完整的WHERE子句
		var whereClause string
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " AND ")
		}

		// 总数查询
		s1 := "SELECT COUNT(*) FROM t_question" + whereClause
		q.Err = conn.QueryRow(ctx, s1, args...).Scan(&rowCount)
		if forceError == "questions.conn.QueryRow" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
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
		var rows pgx.Rows
		rows, q.Err = conn.Query(ctx, s2, args...)
		defer rows.Close()
		if forceError == "questions.conn.Query" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var list []cmn.TQuestion
		for rows.Next() {
			var question cmn.TQuestion
			q.Err = rows.Scan(
				&question.ID,
				&question.Type,
				&question.Content,
				&question.Options,
				&question.Answers,
				&question.Score,
				&question.Difficulty,
				&question.Tags,
				&question.Analysis,
				&question.Title,
				&question.AnswerFilePath,
				&question.TestFilePath,
				&question.Input,
				&question.Output,
				&question.Example,
				&question.Repo,
				&question.Order,
				&question.Creator,
				&question.CreateTime,
				&question.UpdatedBy,
				&question.UpdateTime,
				&question.Addi,
				&question.Status,
				&question.QuestionAttachmentsPath,
				&question.AccessMode,
				&question.BelongTo,
			)
			if forceError == "questions.rows.Scan" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			list = append(list, question)
		}

		q.Err = rows.Err()
		if forceError == "questions.rows.Err()" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var jsonData []byte
		jsonData, q.Err = json.Marshal(list)
		if forceError == "questions.json.Marshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Data = jsonData
		q.Msg.Msg = "success"
		q.Msg.RowCount = rowCount
	case "post":
		// 处理 POST 请求
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

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
		if forceError == "json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
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
			question.Creator = null.IntFrom(userID)

			q.Err = cmn.InvalidEmptyNullValue(&question)
			if forceError == "cmn.InvalidEmptyNullValue" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 写库
			qry.Action = "insert"
			qry.Data, _ = json.Marshal(question)
			q.Err = cmn.DML(&question.Filter, &qry)
			if forceError == "cmn.DML" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			ID, ok := question.QryResult.(int64)
			if forceError == "QryResult.(int64)" {
				ok = false
			}
			if !ok {
				q.Err = fmt.Errorf("qryResult should be int64")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			question.ID = null.IntFrom(ID)

			insertQuestions = append(insertQuestions, question)
		}

		var result []json.RawMessage
		for _, Q := range insertQuestions {
			var b json.RawMessage
			b, q.Err = cmn.MarshalJSON(&Q)
			if forceError == "cmn.MarshalJSON" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			result = append(result, b)
		}
		// 返回插入后的所有记录
		buf, q.Err = json.Marshal(result)
		if forceError == "json.Marshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Data = buf
		q.Msg.Msg = "success"
	case "put":
		// 处理 PUT 请求
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/questions with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var req cmn.ReqProto
		q.Err = json.Unmarshal(buf, &req)
		if forceError == "reqproto-json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var question cmn.TQuestion
		q.Err = json.Unmarshal(req.Data, &question)
		if forceError == "cmn.TQuestion-json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		valid, err := validateQuestion(&question)
		if !valid && err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		now := cmn.GetNowInMS()
		updateSQL := `
			UPDATE t_question
			SET content = $1,
			options = $2,
			answers = $3,
			score = $4,
			difficulty = $5,
			tags = $6,
			analysis = $7,
			updated_by = $8,
			update_time = $9
			WHERE id = $10
		`
		// 执行更新操作
		var commandTag pgconn.CommandTag
		commandTag, q.Err = conn.Exec(ctx, updateSQL, question.Content, question.Options, question.Answers, question.Score, question.Difficulty, question.Tags, question.Analysis, now, now, question.ID)
		if forceError == "conn.Exec" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if commandTag.RowsAffected() == 0 {
			q.Err = fmt.Errorf("no rows updated")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		} //检测当前题目是否被试卷引用，如果被引用，则把引用试卷的版本号加一 TODO

		q.Msg.Status = 0
		q.Msg.Msg = "success"
	case "delete":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceError == "io.ReadAll" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceError == "q.R.Body.Close()" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		if forceError == "q.R.Body.Close()" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty body")
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
		var deleteQuestionIDs []int64
		q.Err = json.Unmarshal(qry.Data, &deleteQuestionIDs)
		if forceError == "json.Unmarshal" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//检测ID数组
		q.Err = validateIDs(deleteQuestionIDs)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//执行删除操作
		//直接硬删除需要删除的题目，关联的试卷题目会被级联删除，且把关联试卷的版本号加一 TODO
		//以上操作需要在数据库事务中进行且需要确保原子性
		//如果其中任何一步失败，则整个事务回滚
		var tx pgx.Tx
		tx, q.Err = conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if forceError == "conn.BeginTx" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if p := recover(); p != nil {
				err := tx.Rollback(ctx)
				if forceError == "recover" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
				}
				err = fmt.Errorf("panic occurred: %v", p)
				z.Error(err.Error())
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if forceError == "tx.Rollback" {
					err = errors.New(forceError)
					q.Err = err
					q.RespErr()
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
			err := tx.Commit(ctx)
			if forceError == "tx.Commit" {
				err = errors.New(forceError)
				q.Err = err
				q.RespErr()
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()
		if forceError == "recover" {
			panic(errors.New(forceError))
		}
		if forceError == "tx.Commit" {
			return
		}
		if forceError == "tx.Rollback" {
			q.Err = errors.New(forceError)
			return
		}
		// 这里是执行具体的删除操作
		var checkSQL string
		var errorMessages []string
		//检查题目是否存在且是否有权限删除
		checkSQL = `
			SELECT COALESCE(array_agg(
				CASE 
					WHEN tq.id IS NULL THEN '题目(' || ids.id || ')不存在'
					ELSE NULL 
				END
			) FILTER (WHERE CASE 
					WHEN tq.id IS NULL THEN '题目(' || ids.id || ')不存在'
					ELSE NULL 
				END IS NOT NULL), ARRAY[]::text[]) as error_messages
			FROM unnest($1::bigint[]) AS ids(id)
			LEFT JOIN t_question tq ON tq.id = ids.id
			WHERE tq.id IS NULL OR tq.creator != $2`
		q.Err = tx.QueryRow(ctx, checkSQL, deleteQuestionIDs, userID).Scan(&errorMessages)
		if forceError == "tx.QueryRow" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
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

		// 如果有任何不能删除的题目，返回错误
		if validErrors.Len() > 0 {
			q.Msg.Status = -1
			q.Err = errors.New(validErrors.String())
			q.RespErr()
			return
		}

		// 2. 然后删除标记为硬删除的题目
		deleteSQL := `
			DELETE FROM t_question 
			WHERE id = ANY($1::bigint[])
		`
		_, q.Err = tx.Exec(ctx, deleteSQL, deleteQuestionIDs)
		if forceError == "tx.Exec" {
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
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}
