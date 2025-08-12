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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/mux"
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

	/* 获取题库详细信息
	 */

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: questionBankDetail,

		Path: "/question-banks/{id}",
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
			status
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

// 题库接口
func questionBanks(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

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

		// 检查是否为教师身份，此身份只能获取自己的题库
		var userID int64
		if userDoamin == DomainTeacher {
			userID = q.SysUser.ID.Int64
			if userID <= 0 {
				q.Err = fmt.Errorf("invalid userID: %d", userID)
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
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 设置创建者
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("invalid userID: %d", userID)
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
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取返回ID
		bankID, ok := bank.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be int64, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		bank.ID = null.IntFrom(bankID)

		// 返回响应
		buf, q.Err = cmn.MarshalJSON(&bank)
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
		questionBankTags := gjson.GetBytes(buf, "data.tags").Array()
		if questionBankName == "" && len(questionBankTags) == 0 {
			q.Err = fmt.Errorf("call /api/question-banks with empty question bank name and tags")
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

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("invalid userID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 设置更新者以及更新时间（毫秒级时间戳）
		bank.UpdatedBy = null.IntFrom(userID)
		t := cmn.GetNowInMS()
		bank.UpdateTime = null.NewInt(t, true)

		filters := []map[string]interface{}{
			{"ID": map[string]interface{}{"EQ": questionBankID}},
		}

		qry.Action = "update"
		qry.Filter = filters
		q.Err = cmn.DML(&bank.Filter, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
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
		return

	case "delete":
		// 设置创建者
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("invalid userID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var deleteBankIDs []int64
		q.Err = json.Unmarshal(qry.Data, &deleteBankIDs)
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
		//首先先检测被删除的题库里面的题库题目是否被t_paper_question引用，引用的话则进行软删除，更新状态为已删除，没有引用的话则硬删除
		//然后题库本身也进行删除操作
		//以上操作需要在数据库事务中进行且需要确保原子性
		//如果其中任何一步失败，则整个事务回滚
		//删除的时候需要判断是否是题库创建者
		var tx pgx.Tx
		tx, q.Err = conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if p := recover(); p != nil {
				err := tx.Rollback(ctx)
				if err != nil {
					z.Error(err.Error())
				}
				q.Err = fmt.Errorf("panic occurred: %v", p)
				z.Error(q.Err.Error())
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if err != nil && errors.Is(q.Err, pgx.ErrTxClosed) {
					z.Error(err.Error())
				}
			} else {
				err := tx.Commit(ctx)
				if err != nil {
					z.Error(err.Error())
				}
			}
		}()
		// 这里是执行具体的删除操作
		// 1. 检查题库是否存在
		// 检查每个试卷的权限
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
		updatetime := cmn.GetNowInMS()
		// 2. 删除题库题目(存在引用软删除，不存在硬删除)
		// 2.1 首先更新题目状态
		updateQuestionSQL := `
			WITH question_ids AS (
				SELECT id FROM t_question 
				WHERE belong_to = ANY($1::bigint[])
			),
			referenced_questions AS (
				SELECT DISTINCT bank_question_id
				FROM t_paper_question
				WHERE bank_question_id IN (SELECT id FROM question_ids)
			)
			UPDATE t_question SET 
				status = CASE 
					WHEN id IN (SELECT bank_question_id FROM referenced_questions) THEN '02'  -- 软删除
					ELSE '06'  -- 硬删除标记
				END,
				updated_by = $2,
				update_time = $3
			WHERE belong_to = ANY($1::bigint[])
		`
		_, q.Err = tx.Exec(ctx, updateQuestionSQL, deleteBankIDs, userID, updatetime)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 2.2 然后删除标记为硬删除的题目
		deleteQuestionSQL := `
			DELETE FROM t_question 
			WHERE belong_to = ANY($1::bigint[])
			AND status = '06'  -- 删除被标记为硬删除的题目
		`
		_, q.Err = tx.Exec(ctx, deleteQuestionSQL, deleteBankIDs)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 3. 删除题库
		//直接硬删除题库，关联的题目在删除后belong会被设置为null，所以不用管
		deleteBankSQL := `
			DELETE FROM t_question_bank
			WHERE id = ANY($1::bigint[])
		`
		_, q.Err = tx.Exec(ctx, deleteBankSQL, deleteBankIDs)
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

/*
	 *
		 * @api {GET} /question-banks/{id} 获取题库详情
		 * @apiName questionBankDetail
		 * @apiVersion 1.0.0
		 * @apiDescription 获取题库详情，包括：题库名;题库标签;题库创建者(ID和名称);题库创建时间;题库更新者(ID和名称);题库更新时间;题库下所有题目涉及的难度、类型、标签;题库下题目数量
		 * @apiGroup QuestionBank

		 * @apiParam {Number} id 题库ID

		 * @apiSuccess {Number} status 响应状态码(0表示成功)
		 * @apiSuccess {Object} data 题库详情数据
		 * @apiSuccess {Number} data.id 题库ID
		 * @apiSuccess {String} data.name 题库名称
		 * @apiSuccess {String} data.type 题库类型代码
		 * @apiSuccess {String[]} data.tags 题库标签数组
		 * @apiSuccess {Number} data.creator 创建者ID
		 * @apiSuccess {String} data.creator_name 创建者名称
		 * @apiSuccess {String} data.create_time 创建时间(时间戳字符串)
		 * @apiSuccess {String} data.updated_time 更新时间(时间戳字符串)
		 * @apiSuccess {String[]} data.question_types 题目类型代码数组
		 * @apiSuccess {String[]} data.question_tags 题目标签数组
		 * @apiSuccess {Number[]} data.question_difficulties 题目难度数组(1-简单,2-中等,3-困难)
		 * @apiSuccess {String} msg 响应消息
		 * @apiSuccessExample {json} 成功响应:
		 *     HTTP/1.1 200 OK
		 *     {
		 *         "status": 0,
		 *         "data": {
		 *             "id": 84,
		 *             "name": "测试综合题库",
		 *             "type": "00",
		 *             "tags": [
		 *               "数据结构",
		 *               "算法",
		 *               "操作系统",
		 *               "计算机网络"
		 *             ],
		 *             "creator": 1626,
		 *             "creator_name": "超级管理员",
		 *             "create_time": "1754045506881",
		 *             "updated_time": "1754046073204",
		 *             "question_types": [
		 *               "06",
		 *               "04",
		 *               "00",
		 *               "08",
		 *               "02"
		 *             ],
		 *             "question_tags": [
		 *               "排序",
		 *               "OSI模型",
		 *               "传输层",
		 *               "数据结构",
		 *               "事务管理",
		 *               "进程调度",
		 *               "实时系统",
		 *               "编译原理",
		 *               "计算机网络",
		 *               "基础",
		 *               "软件设计",
		 *               "SQL",
		 *               "面向对象",
		 *               "数据库",
		 *               "网络协议",
		 *               "生命周期模型",
		 *               "测试",
		 *               "TCP/IP",
		 *               "算法",
		 *               "计算机基础",
		 *               "进程线程",
		 *               "主键",
		 *               "HTTP",
		 *               "软件工程",
		 *               "操作系统",
		 *               "时间复杂度",
		 *               "Java",
		 *               "病毒传播",
		 *               "信息安全"
		 *             ],
		 *             "question_difficulties": [1, 3, 2]
		 *         },
		 *         "msg": "success"
		 *     }

		 * @apiError {Number} status 错误状态码(-1表示失败)
		 * @apiError {String} msg 错误消息
		 * @apiErrorExample {json} 错误响应:
		 *     HTTP/1.1 400 Bad Request
		 *     {
		 *         "status": -1,
		 *         "msg": "bankID is required"
		 *     }
*/
func questionBankDetail(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

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

	if method != "get" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	vars := mux.Vars(q.R)
	bankIDStr := vars["id"]
	if bankIDStr == "" {
		q.Err = fmt.Errorf("bankID is required")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	bankID, err := strconv.ParseInt(bankIDStr, 10, 64)
	if err != nil {
		q.Err = fmt.Errorf("invalid bankID: %s", bankIDStr)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var BankDetail cmn.TVQuestionBank

	// 获取题库信息
	s1 := `
		SELECT
			id,
			name,
			type,
			tags,
			creator,
			official_name AS creator_name,
			create_time,
			update_time,
			question_count,
			question_types,
			question_difficulties,
			question_tags
		FROM v_question_bank
		WHERE id = $1
	`
	err = conn.QueryRow(ctx, s1, bankID).Scan(
		&BankDetail.ID,
		&BankDetail.Name,
		&BankDetail.Type,
		&BankDetail.Tags,
		&BankDetail.Creator,
		&BankDetail.OfficialName,
		&BankDetail.CreateTime,
		&BankDetail.UpdateTime,
		&BankDetail.QuestionCount,
		&BankDetail.QuestionTypes,
		&BankDetail.QuestionDifficulties,
		&BankDetail.QuestionTags,
	)
	if err != nil {
		q.Err = fmt.Errorf("获取题库信息失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(BankDetail)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Data = jsonData
	q.Msg.Msg = "success"
	q.Resp()
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

	switch question.Type {
	case "00", "02", "04":
		var options []QuestionOption
		err = json.Unmarshal(question.Options, &options)
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
		if len(options) < 2 {
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
				err = fmt.Errorf("invalid type: %s", t)
				z.Error(err.Error())
				return nil, 0, err
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
		}
		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("invalid pageSize: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
		}

		params := QueryQuestionsParams{
			BankID:     bankID,
			Content:    content,
			Tags:       tags,
			Type:       questionTypes,
			Difficulty: difficultyList,
			Page:       page,
			PageSize:   pageSize,
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
	case "put":
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
		var req cmn.ReqProto
		q.Err = json.Unmarshal(buf, &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var question cmn.TQuestion
		q.Err = json.Unmarshal(req.Data, &question)
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
		q.Msg.Status = 0
		q.Msg.Msg = "success"
	case "delete":
		// 设置创建者
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("invalid userID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var deleteQuestionIDs []int64
		q.Err = json.Unmarshal(qry.Data, &deleteQuestionIDs)
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
		//首先检测被删除的题目是否被t_paper_question引用，引用的话则进行软删除，没有引用的话则硬删除
		//以上操作需要在数据库事务中进行且需要确保原子性
		//如果其中任何一步失败，则整个事务回滚
		var tx pgx.Tx
		tx, q.Err = conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if p := recover(); p != nil {
				err := tx.Rollback(ctx)
				if err != nil {
					z.Error(err.Error())
				}
				q.Err = fmt.Errorf("panic occurred: %v", p)
				z.Error(q.Err.Error())
			}
			if q.Err != nil {
				err := tx.Rollback(ctx)
				if err != nil {
					z.Error(err.Error())
				}
			}
			err := tx.Commit(ctx)
			if err != nil {
				z.Error(err.Error())
			}
		}()
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
		updatetime := cmn.GetNowInMS()
		// 删除题目 - 如果被试卷引用则软删除，否则硬删除
		// 1. 首先标记题目状态
		updateSQL := `
			WITH referenced_questions AS (
				SELECT DISTINCT bank_question_id 
				FROM t_paper_question 
				WHERE bank_question_id = ANY($1::bigint[])
			)
			UPDATE t_question SET 
				status = CASE 
					WHEN id IN (SELECT bank_question_id FROM referenced_questions) THEN '02'  -- 软删除
					ELSE '06'  -- 硬删除标记
				END,
				updated_by = $2,
				update_time = $3
			WHERE id = ANY($1::bigint[])`
		_, q.Err = tx.Exec(ctx, updateSQL, deleteQuestionIDs, userID, updatetime)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 2. 然后删除标记为硬删除的题目
		deleteSQL := `
			DELETE FROM t_question 
			WHERE id = ANY($1::bigint[])
			AND status = '06'  -- 删除被标记为硬删除的题目
		`
		_, q.Err = tx.Exec(ctx, deleteSQL, deleteQuestionIDs)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}
