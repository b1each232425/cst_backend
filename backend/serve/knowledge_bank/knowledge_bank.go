package knowledge_bank

//annotation:template-service
//author:{"name":"xw","tel":"18925051453", "email":"1062051028@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// 查询参数结构体
type QueryKnowledgeBankParams struct {
	BankID   int64  `json:"bankID"`
	Keyword  string `json:"keyword"`
	Page     int64  `json:"page"`
	PageSize int64  `json:"pageSize"`
	Creator  int64  `json:"creator"`
}

// 知识点库常量
const (
	KnowledgeBankStatusCreated = "00" // 已创建
	KnowledgeBankStatusDeleted = "02" // 已删除
)

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

	/* 知识点库相关接口
	 */

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: knowledgeBanks,

		Path: "/knowledge-bank",
		Name: "知识点库管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "知识点库管理.知识点库.查询知识点库",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "知识点库管理.知识点库.创建知识点库",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
			{
				Name:         "知识点库管理.知识点库.更新知识点库",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
			{
				Name:         "知识点库管理.知识点库.删除知识点库",
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

// 知识点库接口
func knowledgeBanks(ctx context.Context) {
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
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/knowledge-banks", auth_mgt.CAPIAccessActionRead)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有查询知识点库的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

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

		// 根据用户角色决定是否添加创建者过滤条件
		var creatorFilter int64 = 0
		if authority.Role.Priority.Valid && authority.Role.Priority.Int64 == auth_mgt.CDomainPriorityUser {
			// 只有普通用户才需要按创建者过滤，管理员和超级管理员可以看到域内所有知识点库
			creatorFilter = userID
		}

		params := QueryKnowledgeBankParams{
			BankID:   bankID,
			Keyword:  keyword,
			Page:     page,
			PageSize: pageSize,
			Creator:  creatorFilter,
		}

		var rowCount int64
		var conditions []string
		var args []interface{}
		argIndex := 1

		// 基础状态过滤（必须条件）
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, KnowledgeBankStatusCreated)
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

		// 构建完整的WHERE子句
		var whereClause string
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " AND ")
		}

		// 总数查询
		s1 := "SELECT COUNT(*) FROM t_knowledge_bank" + whereClause
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
			domain_id,
			name,
			tags,
			creator,
			create_time,
			update_time,
			knowledges,
			addi,
			status
		FROM t_knowledge_bank
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

		var list []cmn.TKnowledgeBank
		for rows.Next() {
			var bank cmn.TKnowledgeBank
			err = rows.Scan(
				&bank.ID,
				&bank.DomainID,
				&bank.Name,
				&bank.Tags,
				&bank.Creator,
				&bank.CreateTime,
				&bank.UpdateTime,
				&bank.Knowledges,
				&bank.Addi,
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
		// 检查API访问权限
		var accessible bool
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/knowledge-banks", auth_mgt.CAPIAccessActionCreate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有创建知识点库的权限")
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
			q.Err = fmt.Errorf("call /api/knowledge-banks with empty body")
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

		knowledgeBankName := gjson.GetBytes(qry.Data, "name").String()
		if knowledgeBankName == "" {
			q.Err = fmt.Errorf("call /api/knowledge-banks with empty knowledge bank name")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var bank cmn.TKnowledgeBank
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
		bank.Status = null.StringFrom(KnowledgeBankStatusCreated)

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
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/knowledge-banks", auth_mgt.CAPIAccessActionUpdate)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有更新知识点库的权限")
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
			q.Err = fmt.Errorf("call /api/knowledge-banks with empty body")
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

		knowledgeBankID := gjson.GetBytes(buf, "data.id").Int()
		if knowledgeBankID <= 0 {
			q.Err = fmt.Errorf("call /api/knowledge-banks with invalid knowledge bank ID: %d", knowledgeBankID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		knowledgeBankName := gjson.GetBytes(buf, "data.name").String()
		knowledgeBankTagsJson := gjson.GetBytes(buf, "data.tags")
		var knowledgeBankTags []gjson.Result
		if knowledgeBankTagsJson.Exists() {
			knowledgeBankTags = knowledgeBankTagsJson.Array()
		}

		if knowledgeBankName == "" && len(knowledgeBankTags) == 0 {
			q.Err = fmt.Errorf("call /api/knowledge-banks with empty knowledge bank name and tags")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开始构建更新SQL语句
		updateSQL := "UPDATE t_knowledge_bank SET "
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

		// 如果提供了知识点库名称，则更新
		if knowledgeBankName != "" {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, knowledgeBankName)
			argIndex++
		}

		// 如果提供了知识点库标签，则更新
		if len(knowledgeBankTags) > 0 {
			// 将gjson.Result数组转换为字符串数组
			var strTags []string
			for _, tag := range knowledgeBankTags {
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
		args = append(args, knowledgeBankID)

		// 执行更新操作
		var commandTag pgconn.CommandTag
		commandTag, q.Err = conn.Exec(ctx, updateSQL, args...)
		if forceError == "conn.Exec" {
			q.Err = errors.New(forceError)
		}
		if q.Err != nil {
			q.Err = fmt.Errorf("更新知识点库失败: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if commandTag.RowsAffected() == 0 {
			q.Err = fmt.Errorf("更新知识点库失败：没有记录被更新")
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
		accessible, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/knowledge-banks", auth_mgt.CAPIAccessActionDelete)
		if q.Err != nil {
			z.Error("检查API访问权限失败: " + q.Err.Error())
			q.RespErr()
			return
		}
		if !accessible {
			q.Err = fmt.Errorf("用户没有删除知识点库的权限")
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
			q.Err = fmt.Errorf("call /api/knowledge-banks with empty body")
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
		//知识点库直接进行删除操作，但要把关联试卷的版本号加1
		//以上操作需要在数据库事务中进行且需要确保原子性
		//如果其中任何一步失败，则整个事务回滚
		//删除的时候需要判断是否是知识点库创建者
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
		// 1. 检查知识点库是否存在且未被删除
		// 检查每个知识点库的权限
		var checkSQL string
		var errorMessages []string
		checkSQL = `
			SELECT COALESCE(array_agg(
				CASE
					WHEN tkb.id IS NULL THEN '知识点库(' || ids.id || ')不存在'
					WHEN tkb.status = '02' THEN '知识点库(' || COALESCE(tkb.name, '未知') || ')已被删除'
					WHEN tkb.creator != $2 THEN '知识点库(' || COALESCE(tkb.name, '未知') || ')非知识点库创建者，无删除权限'
					ELSE NULL
				END
			) FILTER (WHERE CASE
					WHEN tkb.id IS NULL THEN '知识点库(' || ids.id || ')不存在'
					WHEN tkb.status = '02' THEN '知识点库(' || COALESCE(tkb.name, '未知') || ')已被删除'
					WHEN tkb.creator != $2 THEN '知识点库(' || COALESCE(tkb.name, '未知') || ')非知识点库创建者，无删除权限'
					ELSE NULL
				END IS NOT NULL), ARRAY[]::text[]) as error_messages
			FROM unnest($1::bigint[]) AS ids(id)
			LEFT JOIN t_knowledge_bank tkb ON tkb.id = ids.id
			WHERE tkb.id IS NULL OR tkb.status = '02' OR tkb.creator != $2`
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

		// 如果有任何不能删除的知识点库，返回错误
		if validErrors.Len() > 0 {
			q.Msg.Status = -1
			q.Err = errors.New(validErrors.String())
			q.RespErr()
			return
		}
		// 3. 软删除知识点库 - 更新status为02
		softDeleteBankSQL := `
			UPDATE t_knowledge_bank
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

// validateIDs 验证ID数组的有效性
func validateIDs(ids []int64) error {
	// 检测数组是否为空
	if len(ids) == 0 {
		err := errors.New("ID List cannot be empty")
		z.Error(err.Error())
		return err
	}
	// 检查ID是否大于0并且没有重复
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			err := errors.Errorf("ID must be greater than 0: %d", id)
			z.Error(err.Error())
			return err
		}
		if seen[id] {
			err := errors.Errorf("Duplicate ID found: %d", id)
			z.Error(err.Error())
			return err
		}
		seen[id] = true
	}
	return nil
}
