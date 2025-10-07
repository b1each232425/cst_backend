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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
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
		Name: "题库管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "题库管理.题库.查询题库",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "题库管理.题库.创建题库",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "题库管理.题库.更新题库",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
			{
				Name:         "题库管理.题库.删除题库",
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

	/* 题目相关接口
	 */

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: questions,

		Path: "/questions",
		Name: "题目管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "题库管理.题目.查询题目",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "题库管理.题目.创建题目",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "题库管理.题目.更新题目",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
			{
				Name:         "题库管理.题目.删除题目",
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
		Fn: QuestionLock,

		Path: "/question/lock",
		Name: "question_lock",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	/* 题目附件相关接口
	 */

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: questionFiles,

		Path: "/question-files",
		Name: "题目附件管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "题库管理.题目附件.上传附件",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "题库管理.题目附件.删除附件",
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

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	conn := cmn.GetPgxConn()

	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-banks", auth_mgt.CAPIAccessActionRead)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有查询题库的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取查询参数
		keyword := q.R.URL.Query().Get("keyword")
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")
		bankIDStr := q.R.URL.Query().Get("bankID")
		typeStr := q.R.URL.Query().Get("type")

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

		// 根据用户角色决定是否添加创建者过滤条件
		var creatorFilter int64 = 0
		if authority.Role.Priority.Valid && authority.Role.Priority.Int64 == auth_mgt.CDomainPriorityUser {
			// 只有普通用户才需要按创建者过滤，管理员和超级管理员可以看到域内所有题库
			creatorFilter = userID
		}

		params := QueryQuestionBankParams{
			BankID:   bankID,
			Keyword:  keyword,
			Page:     page,
			PageSize: pageSize,
			Creator:  creatorFilter,
			Type:     typeStr,
		}

		var rowCount int64
		var conditions []string
		var args []interface{}
		argIndex := 1

		// 基础状态过滤（必须条件）
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, "00")
		argIndex++

		if forceError == "EmptyDomain" {
			authority.AccessibleDomains = []int64{}
		}

		// 拼接资源范围 - 用户可访问的所有 domain_id
		if len(authority.AccessibleDomains) > 0 {
			conditions = append(conditions, fmt.Sprintf("domain_id = ANY($%d)", argIndex))
			args = append(args, authority.AccessibleDomains)
			argIndex++
		} else {
			// 如果用户没有可访问的域，则返回空结果
			q.Err = errors.New("用户没有可访问的域")
			q.RespErr()
			return
		}

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

		// 题库类型过滤
		if params.Type != "" {
			// 验证类型参数
			if params.Type != QuestionBankTypeNormal && params.Type != QuestionBankTypeKnowledge {
				q.Err = fmt.Errorf("invalid type parameter: %s", params.Type)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, params.Type)
			argIndex++
		} else {
			// 如果type为空，则只返回普通题库（type=00）
			conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, QuestionBankTypeNormal)
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
			status,
			knowledge_bank_id
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

		var list []TVQuestionBankWithStats
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
				&bank.KnowledgeBankID,
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

			// 获取题库统计信息
			stats, err := getQuestionBankStats(ctx, conn, bank.ID.Int64)
			if err != nil {
				z.Error(fmt.Sprintf("获取题库统计信息失败: %v", err))
				// 如果统计信息获取失败，使用空的统计信息
				stats = QuestionBankStats{}
			}

			// 创建包含统计信息的题库结构体
			bankWithStats := TVQuestionBankWithStats{
				TVQuestionBank: bank,
				Stats:          stats,
			}
			list = append(list, bankWithStats)
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-banks", auth_mgt.CAPIAccessActionCreate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有创建题库的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		if questionBankType != QuestionBankTypeNormal && questionBankType != QuestionBankTypeKnowledge {
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

		// 处理知识点库ID
		knowledgeBankId := gjson.GetBytes(qry.Data, "knowledgeBankId")
		if knowledgeBankId.Exists() && knowledgeBankId.Int() > 0 {
			bank.KnowledgeBankID = null.IntFrom(knowledgeBankId.Int())
		}

		bank.Creator = null.IntFrom(userID)

		//设置所属域
		bank.DomainID = authority.Domain.ID
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-banks", auth_mgt.CAPIAccessActionUpdate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有更新题库的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		questionBankType := gjson.GetBytes(buf, "data.type").String()
		var questionBankTags []gjson.Result
		if questionBankTagsJson.Exists() {
			questionBankTags = questionBankTagsJson.Array()
		}

		if questionBankName == "" && len(questionBankTags) == 0 && questionBankType == "" {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name, tags and type")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 验证题库类型
		if questionBankType != "" && questionBankType != QuestionBankTypeNormal && questionBankType != QuestionBankTypeKnowledge {
			q.Err = fmt.Errorf("call /api/question-banks with unsupported question bank type: %s", questionBankType)
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

		// 如果提供了题库类型，则更新
		if questionBankType != "" {
			setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, questionBankType)
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-banks", auth_mgt.CAPIAccessActionDelete)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有删除题库的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		//题库进行软删除操作（status: 00->02），题库题目级联软删除，相关的试卷题目也级联软删除,但要把关联试卷的版本号加1
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
		// 1. 检查题库是否存在且未被删除
		// 检查每个题库的权限
		var checkSQL string
		var errorMessages []string
		checkSQL = `
			SELECT COALESCE(array_agg(
				CASE
					WHEN tqb.id IS NULL THEN '题库(' || ids.id || ')不存在'
					WHEN tqb.status = '02' THEN '题库(' || COALESCE(tqb.name, '未知') || ')已被删除'
					WHEN tqb.creator != $2 THEN '题库(' || COALESCE(tqb.name, '未知') || ')非题库创建者，无删除权限'
					ELSE NULL
				END
			) FILTER (WHERE CASE
					WHEN tqb.id IS NULL THEN '题库(' || ids.id || ')不存在'
					WHEN tqb.status = '02' THEN '题库(' || COALESCE(tqb.name, '未知') || ')已被删除'
					WHEN tqb.creator != $2 THEN '题库(' || COALESCE(tqb.name, '未知') || ')非题库创建者，无删除权限'
					ELSE NULL
				END IS NOT NULL), ARRAY[]::text[]) as error_messages
			FROM unnest($1::bigint[]) AS ids(id)
			LEFT JOIN t_question_bank tqb ON tqb.id = ids.id
			WHERE tqb.id IS NULL OR tqb.status = '02' OR tqb.creator != $2`
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
		// 3. 软删除题库 - 更新status为02
		softDeleteBankSQL := `
			UPDATE t_question_bank
			SET status = '02', update_time = $2, updated_by = $3
			WHERE id = ANY($1::bigint[]) AND status = '00'
		`
		_, q.Err = tx.Exec(ctx, softDeleteBankSQL, deleteBankIDs, cmn.GetNowInMS(), userID)
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
	case QuestionTypeComprehensive: // 综合应用题
		return validateComprehensiveQuestion(question)
	case QuestionTypeExercise: // 综合演练题
		return validateExerciseQuestion(question)
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

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if q.Err != nil {
		z.Error("获取用户权限失败: " + q.Err.Error())
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/questions", auth_mgt.CAPIAccessActionRead)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有查询题目的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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

		var difficultyList []string
		// 只有当传入difficulty参数时才进行解析
		if len(difficultyStrs) > 0 {
			for _, d := range difficultyStrs {
				if d != "" { // 跳过空值
					// 验证难度值是否有效
					if _, ok := QuestionDifficulty[d]; !ok {
						q.Err = fmt.Errorf("invalid difficulty: %v", d)
						q.RespErr()
						return
					}
					difficultyList = append(difficultyList, d)
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

		// 如果传了questionID，直接查询单道题目
		if params.QuestionID > 0 {
			var question cmn.TQuestion
			s := `
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
					files,
					access_mode,
					belong_to,
					knowledges
				FROM t_question
				WHERE status = '00' AND belong_to = $1 AND id = $2`

			q.Err = conn.QueryRow(ctx, s, params.BankID, params.QuestionID).Scan(
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
				&question.Knowledges,
			)
			if forceError == "questions.single.QueryRow" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				if q.Err == pgx.ErrNoRows {
					q.Err = fmt.Errorf("题目不存在或已删除")
				}
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 为单道题目添加allKnowledges字段
			questionsWithKnowledges, err := enrichQuestionsWithAllKnowledges(ctx, []cmn.TQuestion{question}, params.BankID)
			if err != nil {
				z.Error("获取知识点库失败: " + err.Error())
				q.Err = err
				q.RespErr()
				return
			}

			var jsonData []byte
			if len(questionsWithKnowledges) > 0 {
				jsonData, q.Err = json.Marshal(questionsWithKnowledges[0])
			} else {
				jsonData, q.Err = json.Marshal(question)
			}
			if forceError == "questions.single.json.Marshal" {
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
			q.Msg.RowCount = 1
			q.Resp()
			return
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
			validDifficulties := make([]string, 0)
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
			files,
			access_mode,
			belong_to,
			knowledges
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
				&question.Knowledges,
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

		// 为题目列表添加allKnowledges字段
		questionsWithKnowledges, err := enrichQuestionsWithAllKnowledges(ctx, list, params.BankID)
		if err != nil {
			z.Error("获取知识点库失败: " + err.Error())
			q.Err = err
			q.RespErr()
			return
		}

		var jsonData []byte
		jsonData, q.Err = json.Marshal(questionsWithKnowledges)
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/questions", auth_mgt.CAPIAccessActionCreate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有创建题目的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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

		// 开启事务
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
				panicErr := fmt.Errorf("panic occurred: %v", p)
				z.Error(panicErr.Error())
				err := tx.Rollback(ctx)
				if forceError == "tx.Rollback.panic" {
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
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
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

		if forceError == "tx.Rollback.panic" {
			panic(errors.New(forceError))
		}
		if forceError == "tx.Commit" {
			return
		}
		if forceError == "tx.Rollback" {
			q.Err = errors.New(forceError)
			return
		}

		// 准备批量插入
		batch := &pgx.Batch{}
		var validQuestions []cmn.TQuestion
		now := cmn.GetNowInMS()

		for _, question := range questions {
			valid, err := validateQuestion(&question)
			if !valid && err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			// 设置基础字段
			question.Creator = null.IntFrom(userID)
			question.CreateTime = null.IntFrom(now)
			question.UpdateTime = null.IntFrom(now)
			question.Status = StatusNormal

			q.Err = cmn.InvalidEmptyNullValue(&question)
			if forceError == "cmn.InvalidEmptyNullValue" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 准备插入SQL
			insertSQL := `
				INSERT INTO t_question (
					type, content, options, answers, score, difficulty, tags, analysis,
					title, answer_file_path, test_file_path, input, output, example,
					repo, "order", creator, create_time, updated_by, update_time,
					addi, status, files, access_mode, belong_to
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
					$17, $18, $19, $20, $21, $22, $23, $24, $25
				) RETURNING id`

			batch.Queue(insertSQL,
				question.Type,
				question.Content,
				question.Options,
				question.Answers,
				question.Score,
				question.Difficulty,
				question.Tags,
				question.Analysis,
				question.Title,
				question.AnswerFilePath,
				question.TestFilePath,
				question.Input,
				question.Output,
				question.Example,
				question.Repo,
				question.Order,
				question.Creator,
				question.CreateTime,
				question.UpdatedBy,
				question.UpdateTime,
				question.Addi,
				question.Status,
				question.QuestionAttachmentsPath,
				question.AccessMode,
				question.BelongTo,
			)

			validQuestions = append(validQuestions, question)
		}

		// 执行批量插入
		var br pgx.BatchResults
		br = tx.SendBatch(ctx, batch)

		var insertQuestions []cmn.TQuestion
		for i := range validQuestions {
			var questionID int64
			q.Err = br.QueryRow().Scan(&questionID)
			if forceError == "br.QueryRow" {
				q.Err = errors.New(forceError)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			validQuestions[i].ID = null.IntFrom(questionID)
			insertQuestions = append(insertQuestions, validQuestions[i])
		}

		q.Err = br.Close()
		if forceError == "br.Close" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/questions", auth_mgt.CAPIAccessActionUpdate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有更新题目的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		//尝试获取题目锁
		_, q.Err = cmn.TryLock(ctx, question.ID.Int64, userID, QuestionLockPrefix, QuestionLockExpiration)
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
		}
		//释放题目锁
		if forceError == "cmn.ReleaseLock" {
			_ = cmn.ReleaseLock(ctx, question.ID.Int64, userID, QuestionLockPrefix)
		}
		q.Err = cmn.ReleaseLock(ctx, question.ID.Int64, userID, QuestionLockPrefix)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Status = 0
		q.Msg.Msg = "success"
	case "delete":
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/questions", auth_mgt.CAPIAccessActionDelete)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有删除题目的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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
		//直接软删除需要删除的题目（status: 00->02），关联的试卷题目会被级联软删除，且把关联试卷的版本号加一 TODO
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
					WHEN tq.status = '02' THEN '题目(' || ids.id || ')已被删除'
					WHEN tq.creator != $2 THEN '题目(' || ids.id || ')非题目创建者，无删除权限'
					ELSE NULL
				END
			) FILTER (WHERE CASE
					WHEN tq.id IS NULL THEN '题目(' || ids.id || ')不存在'
					WHEN tq.status = '02' THEN '题目(' || ids.id || ')已被删除'
					WHEN tq.creator != $2 THEN '题目(' || ids.id || ')非题目创建者，无删除权限'
					ELSE NULL
				END IS NOT NULL), ARRAY[]::text[]) as error_messages
			FROM unnest($1::bigint[]) AS ids(id)
			LEFT JOIN t_question tq ON tq.id = ids.id
			WHERE tq.id IS NULL OR tq.status = '02' OR tq.creator != $2`
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

		// 2. 软删除题目 - 更新status为02
		softDeleteSQL := `
			UPDATE t_question
			SET status = '02', update_time = $2, updated_by = $3
			WHERE id = ANY($1::bigint[]) AND status = '00'
		`
		_, q.Err = tx.Exec(ctx, softDeleteSQL, deleteQuestionIDs, cmn.GetNowInMS(), userID)
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

// 题目锁 获取锁\延长锁\释放锁
// QuestionLock 处理题目编辑锁相关的HTTP请求
func QuestionLock(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceError := ""
	if val, ok := ctx.Value("force-error").(string); ok {
		forceError = val
	}

	//获取用户ID
	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid userID: %d", userID)
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
		//accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question/lock", auth_mgt.CDataAccessModeEdit)
		//if q.Err != nil {
		//	fmt.Printf("检查API访问权限失败: %v\n", q.Err)
		//	return
		//}
		//if !accessible {
		//	fmt.Println("用户没有访问权限")
		//	return
		//}
		// 解析并验证题目ID
		questionIDStr := q.R.URL.Query().Get("question_id")
		var questionID int64
		questionID, q.Err = strconv.ParseInt(questionIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if questionID <= 0 {
			q.Err = fmt.Errorf("invalid questionID: %d", questionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//尝试获取题目锁
		_, q.Err = cmn.TryLock(ctx, questionID, userID, QuestionLockPrefix, QuestionLockExpiration)
		if forceError == "cmn.TryLock" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	case "put":
		// 解析并验证题目ID
		// 2. 检查API访问权限
		//var accessible bool
		//accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question/lock", auth_mgt.CDataAccessModeEdit)
		//if q.Err != nil {
		//	fmt.Printf("检查API访问权限失败: %v\n", q.Err)
		//	return
		//}
		//if !accessible {
		//	fmt.Println("用户没有访问权限")
		//	return
		//}
		questionIDStr := q.R.URL.Query().Get("question_id")
		var questionID int64
		questionID, q.Err = strconv.ParseInt(questionIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if questionID <= 0 {
			q.Err = fmt.Errorf("invalid questionID: %d", questionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//延长题目锁
		q.Err = cmn.RefreshLock(ctx, questionID, userID, QuestionLockPrefix, QuestionLockExpiration)
		if forceError == "cmn.RefreshLock" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
	case "delete":
		// 解析并验证题目ID
		questionIDStr := q.R.URL.Query().Get("question_id")
		var questionID int64
		questionID, q.Err = strconv.ParseInt(questionIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if questionID <= 0 {
			q.Err = fmt.Errorf("invalid questionID: %d", questionID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//释放题目锁
		q.Err = cmn.ReleaseLock(ctx, questionID, userID, QuestionLockPrefix)
		if forceError == "cmn.ReleaseLock" {
			q.Err = errors.New(forceError)
		}
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

// 处理题目相关的附件上传逻辑
func questionFiles(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if forceErr == "auth_mgt.GetUserAuthority" {
		q.Err = fmt.Errorf("强制获取用户权限错误")
	}
	if q.Err != nil {
		q.RespErr()
		return
	}

	// 当前用户登录选择的域
	userRole := authority.Role.ID.Int64

	// 开启事务
	var tx pgx.Tx
	tx, q.Err = conn.Begin(ctx)
	if forceErr == "tx.Begin" {
		q.Err = fmt.Errorf("强制开始事务错误")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil || forceErr == "tx.Rollback" {
			rollbackErr := tx.Rollback(context.Background())
			if forceErr == "tx.Rollback" {
				rollbackErr = fmt.Errorf("强制回滚事务错误")
			}
			if rollbackErr != nil {
				z.Error(fmt.Sprintf("failed to rollback transaction: %s", rollbackErr.Error()))
			}
			return
		}

		commitErr := tx.Commit(context.Background())
		if forceErr == "tx.Commit" {
			commitErr = fmt.Errorf("强制提交事务错误")
		}
		if commitErr != nil {
			z.Error(fmt.Sprintf("failed to commit transaction: %s", commitErr.Error()))
		}
	}()

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		// 检查用户是否有权限上传题目附件
		accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-files", auth_mgt.CAPIAccessActionCreate)
		if err != nil || forceErr == "CheckUserAPIAccessible" {
			q.Err = fmt.Errorf("failed to check user API access: %w", err)
			q.RespErr()
			return
		}
		if !accessible || forceErr == "no-access" {
			q.Err = fmt.Errorf("用户没有权限上传题目附件")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 更新题目附件
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var questionFiles []QuestionFile
		q.Err = json.Unmarshal(qry.Data, &questionFiles)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(questionFiles) == 0 {
			q.Err = fmt.Errorf("题目附件列表为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查所有题目是否存在
		questionIDs := make([]int64, 0, len(questionFiles))
		questionIDMap := make(map[int64]bool)
		for _, qf := range questionFiles {
			if !questionIDMap[qf.QuestionID] {
				questionIDs = append(questionIDs, qf.QuestionID)
				questionIDMap[qf.QuestionID] = true
			}
		}

		checkExistsSQL := `
			SELECT id
			FROM t_question
			WHERE id = ANY($1) AND status != '02'
		`
		var existingQuestionRows pgx.Rows
		existingQuestionRows, q.Err = tx.Query(ctx, checkExistsSQL, questionIDs)
		if forceErr == "checkQuestionExists" {
			q.Err = fmt.Errorf("强制检查题目存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer existingQuestionRows.Close()

		existingQuestionIDs := make(map[int64]bool)
		for existingQuestionRows.Next() {
			var questionID int64
			q.Err = existingQuestionRows.Scan(&questionID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingQuestionIDs[questionID] = true
		}

		// 检查是否有不存在的题目
		for _, qf := range questionFiles {
			if !existingQuestionIDs[qf.QuestionID] {
				q.Err = fmt.Errorf("题目ID %d 不存在", qf.QuestionID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 按题目ID分组处理附件
		questionFileMap := make(map[int64][]QuestionFile)
		for _, qf := range questionFiles {
			questionFileMap[qf.QuestionID] = append(questionFileMap[qf.QuestionID], qf)
		}

		// 存储所有题目的最终附件信息
		allQuestionFiles := make(map[int64][]cmn.TFile)

		// 处理每个题目的附件
		for questionID, files := range questionFileMap {
			// 获取该题目的所有现有附件
			getQuestionFilesSQL := `
				SELECT f.id, f.digest, f.file_name, f.size, f.domain_id, f.creator
				FROM t_question q
				CROSS JOIN LATERAL jsonb_array_elements_text(q.files) file_id_text(value)
				JOIN t_file f ON f.id = file_id_text.value::bigint
				WHERE q.id = $1
				  AND q.files IS NOT NULL
				  AND jsonb_typeof(q.files) = 'array'
				  AND q.status != '02'
				  AND f.status != '2'
			`
			var questionFileRows pgx.Rows
			questionFileRows, q.Err = tx.Query(ctx, getQuestionFilesSQL, questionID)
			if forceErr == "questionFiles.tx.Query" {
				q.Err = fmt.Errorf("强制查询题目文件错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer questionFileRows.Close()

			var existingFiles []cmn.TFile
			for questionFileRows.Next() {
				var tFile cmn.TFile
				q.Err = questionFileRows.Scan(&tFile.ID, &tFile.Digest, &tFile.FileName, &tFile.Size, &tFile.DomainID, &tFile.Creator)
				if forceErr == "questionFiles.rows.Scan" {
					q.Err = fmt.Errorf("强制扫描题目文件行错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				existingFiles = append(existingFiles, tFile)
			}

			// 处理每个要上传的文件
			for _, newFile := range files {
				var needUpdate bool = true
				var replacedFileID int64 = -1
				var fileID int64

				// 检查是否有相同digest和文件名的文件
				for i := 0; i < len(existingFiles); i++ {
					// 如果digest和文件名不同，则算作新增文件
					if existingFiles[i].Digest != newFile.CheckSum || existingFiles[i].FileName != newFile.Name {
						continue
					}

					// 文件名、创建者、domainID都一致，则不用做任何处理
					if existingFiles[i].Creator.Int64 == userID && existingFiles[i].DomainID.Int64 == userRole {
						needUpdate = false
						break
					}

					// 如果创建者、domainID有一个不一致，则替换该ID，删除原来的文件记录
					replacedFileID = existingFiles[i].ID.Int64
					break
				}

				// 如果附件有更新，则创建新文件记录
				if needUpdate {
					fileID, q.Err = cmn.NewFileRecord(ctx, tx, newFile.CheckSum, newFile.Name, newFile.Size, userRole, userID)
					if q.Err != nil {
						q.RespErr()
						return
					}

					// 如果是替换，删除旧文件
					if replacedFileID != -1 {
						q.Err = cmn.DeleteFileRecord(ctx, tx, replacedFileID)
						if q.Err != nil {
							q.RespErr()
							return
						}
						// 更新existingFiles数组中对应的fileID
						for i := range existingFiles {
							if existingFiles[i].ID.Int64 == replacedFileID {
								existingFiles[i].ID = null.IntFrom(fileID)
								break
							}
						}
					} else {
						// 新增文件，添加到数组
						newTFile := cmn.TFile{
							ID:       null.IntFrom(fileID),
							Digest:   newFile.CheckSum,
							FileName: newFile.Name,
							Size:     null.IntFrom(newFile.Size),
						}
						existingFiles = append(existingFiles, newTFile)
					}
				}
			}

			// 更新题目的files字段
			var questionFileIDs []int64
			for _, qf := range existingFiles {
				questionFileIDs = append(questionFileIDs, qf.ID.Int64)
			}

			var filesJSON []byte
			filesJSON, q.Err = json.Marshal(questionFileIDs)
			if forceErr == "json.Marshal" {
				q.Err = fmt.Errorf("强制JSON序列化错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			_, q.Err = tx.Exec(ctx, `UPDATE t_question SET files = $1 WHERE id = $2`, filesJSON, questionID)
			if forceErr == "tx.Exec" {
				q.Err = fmt.Errorf("强制更新题目信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 保存该题目的最终附件信息
			allQuestionFiles[questionID] = existingFiles
		}

		// 构造返回数据
		var questionFileReply []QuestionFile
		for questionID, files := range allQuestionFiles {
			for _, qf := range files {
				questionFileReply = append(questionFileReply, QuestionFile{
					QuestionID: questionID,
					CheckSum:   qf.Digest,
					Name:       qf.FileName,
					Size:       qf.Size.Int64,
				})
			}
		}

		q.Msg.Data, q.Err = json.Marshal(questionFileReply)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Resp()
		return

	case "delete":
		// 检查用户是否有权限删除题目附件
		accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/question-files", auth_mgt.CAPIAccessActionDelete)
		if err != nil || forceErr == "CheckUserAPIAccessible" {
			q.Err = fmt.Errorf("failed to check user API access: %w", err)
			q.RespErr()
			return
		}
		if !accessible || forceErr == "no-access" {
			q.Err = fmt.Errorf("用户没有权限删除题目附件")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if forceErr == "io.Close" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var questionFiles []QuestionFile
		q.Err = json.Unmarshal(qry.Data, &questionFiles)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(questionFiles) == 0 {
			q.Err = fmt.Errorf("题目附件列表为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查所有题目是否存在
		questionIDs := make([]int64, 0, len(questionFiles))
		questionIDMap := make(map[int64]bool)
		for _, qf := range questionFiles {
			if !questionIDMap[qf.QuestionID] {
				questionIDs = append(questionIDs, qf.QuestionID)
				questionIDMap[qf.QuestionID] = true
			}
		}

		checkExistsSQL := `
			SELECT id
			FROM t_question
			WHERE id = ANY($1) AND status != '02'
		`
		var existingQuestionRows pgx.Rows
		existingQuestionRows, q.Err = tx.Query(ctx, checkExistsSQL, questionIDs)
		if forceErr == "checkQuestionExists" {
			q.Err = fmt.Errorf("强制检查题目存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer existingQuestionRows.Close()

		existingQuestionIDs := make(map[int64]bool)
		for existingQuestionRows.Next() {
			var questionID int64
			q.Err = existingQuestionRows.Scan(&questionID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			existingQuestionIDs[questionID] = true
		}

		// 检查是否有不存在的题目
		for _, qf := range questionFiles {
			if !existingQuestionIDs[qf.QuestionID] {
				q.Err = fmt.Errorf("题目ID %d 不存在", qf.QuestionID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 按题目ID分组处理要删除的附件
		questionFileMap := make(map[int64][]QuestionFile)
		for _, qf := range questionFiles {
			questionFileMap[qf.QuestionID] = append(questionFileMap[qf.QuestionID], qf)
		}

		// 存储所有题目的最终附件信息
		allQuestionFiles := make(map[int64][]cmn.TFile)

		// 处理每个题目的附件删除
		for questionID, filesToDelete := range questionFileMap {
			// 获取该题目的所有现有附件
			getQuestionFilesSQL := `
				SELECT f.id, f.digest, f.file_name, f.size
				FROM t_question q
				CROSS JOIN LATERAL jsonb_array_elements_text(q.files) file_id_text(value)
				JOIN t_file f ON f.id = file_id_text.value::bigint
				WHERE q.id = $1
				  AND q.files IS NOT NULL
				  AND jsonb_typeof(q.files) = 'array'
				  AND q.status != '02'
				  AND f.status != '2'
			`
			var questionFileRows pgx.Rows
			questionFileRows, q.Err = tx.Query(ctx, getQuestionFilesSQL, questionID)
			if forceErr == "getQuestionFiles" {
				q.Err = fmt.Errorf("强制获取题目附件信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer questionFileRows.Close()

			var existingFiles []cmn.TFile
			for questionFileRows.Next() {
				var tFile cmn.TFile
				q.Err = questionFileRows.Scan(&tFile.ID, &tFile.Digest, &tFile.FileName, &tFile.Size)
				if forceErr == "scanQuestionFile" {
					q.Err = fmt.Errorf("强制获取题目附件信息错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				existingFiles = append(existingFiles, tFile)
			}

			// 删除指定的附件
			var deleteFileIDs []int64
			for _, fileToDelete := range filesToDelete {
				for i, existingFile := range existingFiles {
					if existingFile.Digest == fileToDelete.CheckSum && existingFile.FileName == fileToDelete.Name {
						// 找到匹配的题目附件，记录要删除的文件ID
						deleteFileIDs = append(deleteFileIDs, existingFile.ID.Int64)
						// 从现有文件中移除
						existingFiles = append(existingFiles[:i], existingFiles[i+1:]...)
						break
					}
				}
			}

			// 删除文件记录
			for _, deleteFileID := range deleteFileIDs {
				q.Err = cmn.DeleteFileRecord(ctx, tx, deleteFileID)
				if forceErr == "handleDeleteQuestionFile" {
					q.Err = fmt.Errorf("强制删除题目附件错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			// 更新题目的files字段
			var questionFileIDs []int64
			for _, qf := range existingFiles {
				questionFileIDs = append(questionFileIDs, qf.ID.Int64)
			}

			var filesJSON []byte
			filesJSON, q.Err = json.Marshal(questionFileIDs)
			if forceErr == "questionFiles.json.Marshal.Delete" {
				q.Err = fmt.Errorf("强制序列化题目附件ID数组错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			_, q.Err = tx.Exec(ctx, `
				UPDATE t_question
				SET files = $1
				WHERE id = $2
			`, filesJSON, questionID)
			if forceErr == "questionInfo.tx.UpdateFiles.Delete" {
				q.Err = fmt.Errorf("强制更新题目附件字段错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 保存该题目的最终附件信息
			allQuestionFiles[questionID] = existingFiles
		}

		// 组装返回给前端的文件信息
		var questionFileReply []QuestionFile
		for questionID, files := range allQuestionFiles {
			for _, qf := range files {
				questionFileReply = append(questionFileReply, QuestionFile{
					QuestionID: questionID,
					CheckSum:   qf.Digest,
					Name:       qf.FileName,
					Size:       qf.Size.Int64,
				})
			}
		}

		q.Msg.Data, q.Err = json.Marshal(questionFileReply)
		if forceErr == "questionFiles.json.Marshal" {
			q.Err = fmt.Errorf("强制序列化题目文件错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// getQuestionBankStats 获取题库统计信息
func getQuestionBankStats(ctx context.Context, conn *pgxpool.Pool, bankID int64) (QuestionBankStats, error) {
	var stats QuestionBankStats

	// 1. 获取题目总数
	totalCountQuery := `
		SELECT COUNT(*)
		FROM t_question
		WHERE belong_to = $1 AND status = '00'
	`
	err := conn.QueryRow(ctx, totalCountQuery, bankID).Scan(&stats.TotalCount)
	if err != nil {
		return stats, fmt.Errorf("获取题目总数失败: %v", err)
	}

	// 2. 获取各题型各难度统计
	typeDifficultyStatsQuery := `
		SELECT
			type,
			difficulty,
			COUNT(*) as count
		FROM t_question
		WHERE belong_to = $1 AND status = '00'
		GROUP BY type, difficulty
		ORDER BY type, difficulty
	`
	rows, err := conn.Query(ctx, typeDifficultyStatsQuery, bankID)
	if err != nil {
		return stats, fmt.Errorf("获取题型难度统计失败: %v", err)
	}
	defer rows.Close()

	// 按题型分组统计
	typeDifficultyMap := make(map[string][]QuestionDifficultyCount)
	typeCountMap := make(map[string]int64) // 记录每个题型的总数

	for rows.Next() {
		var typeCode, difficultyCode string
		var count int64
		err := rows.Scan(&typeCode, &difficultyCode, &count)
		if err != nil {
			return stats, fmt.Errorf("扫描题型难度统计失败: %v", err)
		}

		difficultyName, exists := QuestionDifficulty[difficultyCode]
		if !exists {
			difficultyName = "未知难度"
		}

		difficultyStats := QuestionDifficultyCount{
			DifficultyName: difficultyName,
			DifficultyCode: difficultyCode,
			Count:          count,
		}

		// 添加到题型难度映射
		typeDifficultyMap[typeCode] = append(typeDifficultyMap[typeCode], difficultyStats)
		// 累加题型总数
		typeCountMap[typeCode] += count
	}

	// 构建题型统计结果
	var types []QuestionTypeCount
	for typeCode, difficulties := range typeDifficultyMap {
		typeName, exists := QuestionTypes[typeCode]
		if !exists {
			typeName = "未知题型"
		}

		types = append(types, QuestionTypeCount{
			TypeName:     typeName,
			TypeCode:     typeCode,
			Count:        typeCountMap[typeCode],
			Difficulties: difficulties,
		})
	}
	stats.Types = types

	return stats, nil
}
