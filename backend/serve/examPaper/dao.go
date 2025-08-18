/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷-答卷数据库层
 * @Date: 2025-07-21 13:14:34
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-17 09:45:50
 */
package examPaper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"math/rand"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

// LoadExamPaperDetailsById 获取完整考卷信息（包括试卷信息、题组信息、题目信息） 支持根据参数的传入决定是否需要查询题组题目、去除答案等 用于生成与绑定学生答卷
/*
关键参数说明：
	examPaperId 考卷Id
	withQuestions 是否查询题组及题目信息 对应返回体[]*cmn.TExamPaperGroup、map[int64][]*ExamQuestion
	withAnswers 题目是否需要携带答案
	withAnalysis 题目是否需要携带解析
返回参数说明：
	1、考卷基本信息
	2、考卷题组信息
	3、考卷题目信息
	4、可能返回的错误
*/
func LoadExamPaperDetailsById(ctx context.Context, tx pgx.Tx, examPaperId int64, withQuestions, withAnswers, withAnalysis bool) (*cmn.TVExamPaper, []*cmn.TExamPaperGroup, map[int64][]*ExamQuestion, error) {
	var err error
	if examPaperId <= 0 {
		err = fmt.Errorf("invalid examPaper ID param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	// 一张考卷卷拥有的题组map
	var examGroups []*cmn.TExamPaperGroup
	// 一个题组下拥有的题目数组
	examQuestions := make(map[int64][]*ExamQuestion)
	var ep cmn.TVExamPaper
	// 动态构建 SQL 字段
	fields := []string{"id", "exam_session_id", "practice_id", "name", "creator", "create_time", "updated_by",
		"update_time", "status", "total_score", "question_count", "group_count"}
	scanArgs := []interface{}{
		&ep.ID, &ep.ExamSessionID, &ep.PracticeID, &ep.Name,
		&ep.Creator, &ep.CreateTime, &ep.UpdatedBy, &ep.UpdateTime, &ep.Status,
		&ep.TotalScore, &ep.QuestionCount, &ep.GroupCount,
	}
	if withQuestions {
		fields = append(fields, "groups_data")
		scanArgs = append(scanArgs, &ep.GroupsData)
	}
	s := fmt.Sprintf("SELECT %s FROM v_exam_paper WHERE id=$1 AND status=$2", strings.Join(fields, ","))
	z.Sugar().Debugf("打印输出sql查询语句：%v", s)
	err = tx.QueryRow(ctx, s, examPaperId, PaperStatus.Normal).Scan(scanArgs...)
	if err != nil {
		err = fmt.Errorf("select examPaper failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if !withQuestions || len(ep.GroupsData) == 0 {
		return &ep, nil, nil, nil
	}
	var groupData []ExamGroup
	if forceErr == "json" {
		ep.GroupsData = types.JSONText(`invalid json: missing closing brace`)
	}
	if forceErr == "jsonA" {
		ep.GroupsData = types.JSONText(`[]`) // 空数组
	}
	err = json.Unmarshal(ep.GroupsData, &groupData)
	if err != nil {
		err = fmt.Errorf("unmarshal group data failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if len(groupData) == 0 {
		err = fmt.Errorf("empty examPaper groups data")
		z.Error(err.Error())
		return &ep, nil, nil, err
	}
	for _, v := range groupData {
		examGroups = append(examGroups, &cmn.TExamPaperGroup{
			ID:          v.ID,
			ExamPaperID: v.ExamPaperID,
			Name:        v.Name,
			Order:       v.Order,
			Creator:     v.Creator,
			CreateTime:  v.CreateTime,
			UpdatedBy:   v.UpdatedBy,
			UpdateTime:  v.UpdateTime,
			Addi:        v.Addi,
			Status:      v.Status,
		})
		if _, exists := examQuestions[v.ID.Int64]; !exists {
			examQuestions[v.ID.Int64] = make([]*ExamQuestion, 0)
		}
		// 保留答案与解析
		for idx := range v.Questions {
			q := v.Questions[idx]
			var answersSlice []interface{}
			if len(q.Answers) > 0 {
				if forceErr == "jsonB" {
					q.Answers = types.JSONText(`invalid json: missing closing brace`)
				}
				if err = json.Unmarshal(q.Answers, &answersSlice); err != nil {
					err = fmt.Errorf("failed to unmarshal Answers questionId:%v for:%v", q.ID.Int64, err)
					z.Error(err.Error())
					return &ep, nil, nil, err
				}
			}
			q.AnswerNum = len(answersSlice)
			if !withAnalysis {
				q.Analysis = null.String{}
			}
			if !withAnswers {
				q.Answers = nil
			}
			examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
		}
	}
	// 信息已经取出，直接清除
	ep.GroupsData = nil
	return &ep, examGroups, examQuestions, nil
}

// GenerateAnswerQuestion 根据一次考卷生成学生答卷信息 批量插入学生答卷
func GenerateAnswerQuestion(ctx context.Context, tx pgx.Tx, req GenerateAnswerQuestionsRequest, uid int64) error {
	var err error
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	now := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(now))
	err = cmn.Validate(req)
	if err != nil {
		return err
	}
	var studentIds []int64
	if req.Category == PaperCategory.Exam {
		studentIds = req.ExamineeIDs
	} else {
		studentIds = req.PracticeSubmissionID
	}

	// 获取考题
	_, pg, pq, err := LoadExamPaperDetailsById(ctx, tx, req.ExamPaperID, true, true, true)
	if err != nil {
		return err
	}
	var Answers []*cmn.TStudentAnswers
	for _, id := range studentIds {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			if req.IsQuestionRandom {
				// 这里会重复的影响到原数组 ， 每一次也是会打乱一次题组内的顺序的
				r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			}
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID: q.ID,
					GroupID:    g.ID,
					Order:      null.IntFrom(int64(order)),
				}
				// 区分答卷的用途
				if req.Category == PaperCategory.Exam {
					answer.ExamineeID = null.IntFrom(id)
				}
				if req.Category == PaperCategory.Practice {
					answer.PracticeSubmissionID = null.IntFrom(id)
					answer.WrongAttempt = null.IntFrom(0) // 赋予练习初值，用于学生初次查询某一次练习提交记录的错题集
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					if req.IsOptionRandom {
						answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
						if err != nil || forceErr == "Rand" {
							return err
						}
					} else {
						answer.ActualAnswers = q.Answers
						answer.ActualOptions = q.Options
					}
				}
				Answers = append(Answers, answer)
				order++
			}
		}
	}
	// 限制学生答卷一次性插入的数量
	_limit := 500
	for i := 0; i < len(Answers); i += _limit {
		end := i + _limit
		if end > len(Answers) {
			end = len(Answers)
		}
		batchStudentAnswers := Answers[i:end]
		// 执行单次操作语句
		query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers,wrong_attempt
		) VALUES %s
	`
		values := make([]string, 0, len(batchStudentAnswers))
		args := make([]interface{}, 0, len(batchStudentAnswers)*12) // 11个字段
		idx := 1
		for _, a := range batchStudentAnswers {
			placeholders := make([]string, 12)
			for i := 0; i < 12; i++ {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				idx++
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				req.Category, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers, a.WrongAttempt,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		z.Sugar().Debugf("打印输出一下插入学生作答SQL语句:%v", insertQuery)
		z.Sugar().Debugf("打印输出一下插入学生作答SQL参数:%v", args)
		_, err := tx.Exec(ctx, insertQuery, args...)
		if err != nil || forceErr == "query" {
			err = fmt.Errorf("failed batch insert student answer: %v", err)
			z.Error(err.Error())
			return err
		}
	}

	return nil

}

// LoadPaperTemplateById  一张标准的试卷内容 支持按需加载试卷模板（支持题组和题目按需解析）
/*
关键参数说明：
	paperId 试卷模版Id
	withQuestions 是否查询题组及题目信息
*/
func LoadPaperTemplateById(ctx context.Context, paperId int64, withQuestions bool) (*cmn.TVPaper, []*cmn.TPaperGroup, map[int64][]*Question, error) {
	if paperId <= 0 {
		err := fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	var err error
	// 一张试卷拥有的题组map
	var groups []*cmn.TPaperGroup
	// 一个题组下拥有的题目数组
	questions := make(map[int64][]*Question)
	var p cmn.TVPaper
	conn := cmn.GetPgxConn()
	fields := []string{
		"id", "name", "assembly_type", "category", "level",
		"suggested_duration", "description", "tags", "config",
		"creator", "create_time", "update_time", "status",
		"total_score", "question_count", "group_count",
	}
	scanArgs := []interface{}{
		&p.ID, &p.Name, &p.AssemblyType, &p.Category, &p.Level,
		&p.SuggestedDuration, &p.Description, &p.Tags, &p.Config,
		&p.Creator, &p.CreateTime, &p.UpdateTime, &p.Status,
		&p.TotalScore, &p.QuestionCount, &p.GroupCount,
	}
	if withQuestions {
		fields = append(fields, "groups_data")
		scanArgs = append(scanArgs, &p.GroupsData)
	}
	s := fmt.Sprintf("SELECT %s FROM assessuser.v_paper WHERE id=$1 AND status != $2", strings.Join(fields, ","))
	err = conn.QueryRow(ctx, s, paperId, PaperStatus.Invalid).Scan(scanArgs...)
	if err != nil {
		err = fmt.Errorf("select paper failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if !withQuestions || len(p.GroupsData) == 0 {
		return &p, nil, nil, nil
	}
	var groupData []Group
	if forceErr == "json" {
		p.GroupsData = types.JSONText(`invalid json: missing closing brace`)
	}
	if err := json.Unmarshal(p.GroupsData, &groupData); err != nil {
		err = fmt.Errorf("unmarshal group data failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	for _, v := range groupData {
		groups = append(groups, &cmn.TPaperGroup{
			ID:         v.ID,
			PaperID:    v.PaperID,
			Name:       v.Name,
			Order:      v.Order,
			Creator:    v.Creator,
			CreateTime: v.CreateTime,
			UpdatedBy:  v.UpdatedBy,
			UpdateTime: v.UpdateTime,
			Addi:       v.Addi,
			Status:     v.Status,
		})

		if _, exists := questions[v.ID.Int64]; !exists {
			questions[v.ID.Int64] = make([]*Question, 0)
		}

		// 安全追加题目（避免指针重用）
		for idx := range v.Questions {
			q := v.Questions[idx] // 创建副本
			answerNum := 0
			if q.SubScore != nil && len(q.SubScore) > 0 {
				answerNum = len(q.SubScore)
			}
			q.AnswerNum = answerNum
			questions[v.ID.Int64] = append(questions[v.ID.Int64], &q)
		}
	}
	// 释放题组JSON实体内存
	p.GroupsData = nil

	return &p, groups, questions, nil
}

// GenerateExamPaper 生成一张可用考卷（包括题组与题目） 占位符最多65535个，优化为同一个事务批量插入考题，避免试卷体量过大而无法一次插入的问题
/*
关键参数说明：
	category 考卷服务类型 00：考试 02：练习
	paperId 试卷模版Id
	practiceId 练习Id 若类型为练习必填 否则填0
	examSessionId 考试场次Id 若类型为考试必填 否则填0
	uid 操作人Id
	genMarkInfo 是否生成批改信息 用于考试创建时配置批改信息 考卷ID与考卷题组的结构
*/
func GenerateExamPaper(ctx context.Context, tx pgx.Tx, category string, paperId, practiceId, examSessionId int64, uid int64, genMarkInfo bool) (*int64, []SubjectiveQuestionGroup, error) {
	var err error
	var esid, pid interface{}
	esid = examSessionId
	pid = practiceId
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	if paperId <= 0 {
		err = fmt.Errorf("invalid paperId ID param")
		z.Error(err.Error())
		return nil, nil, err
	}
	if category != PaperCategory.Exam && category != PaperCategory.Practice {
		err = fmt.Errorf("invalid category param")
		z.Error(err.Error())
		return nil, nil, err
	}
	if practiceId <= 0 && examSessionId <= 0 {
		err = fmt.Errorf("invalid practice or examSession ID param")
		z.Error(err.Error())
		return nil, nil, err
	}
	if uid <= 0 {
		err = fmt.Errorf("invalid uid ID param")
		z.Error(err.Error())
		return nil, nil, err
	}
	if practiceId == 0 {
		pid = nil
	}
	if examSessionId == 0 {
		esid = nil
	}
	p, pg, pq, err := LoadPaperTemplateById(ctx, paperId, true)
	if err != nil {
		return nil, nil, err
	}
	if len(pg) == 0 || forceErr == "pg" {
		err = fmt.Errorf("invalid paper question group")
		z.Error(err.Error())
		return nil, nil, err
	}
	now := time.Now().UnixMilli()
	ep := cmn.TExamPaper{}
	insertPaperSQL := `
		INSERT INTO assessuser.t_exam_paper 
			(exam_session_id, practice_id, name, creator, create_time, update_time) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`
	err = tx.QueryRow(
		ctx,
		insertPaperSQL,
		esid, pid, p.Name, uid, now, now,
	).Scan(&ep.ID)
	if err != nil || forceErr == "query1" {
		err = fmt.Errorf("插入考卷失败: %w", err)
		z.Error(err.Error())
		return nil, nil, err
	}

	// 这里需要返回id + order的顺序
	// 修改题组插入：返回ID和顺序
	groupSQL := `INSERT INTO assessuser.t_exam_paper_group 
		(exam_paper_id, name, "order", creator, create_time, update_time) 
	VALUES %s 
	RETURNING id, "order"` // 只返回ID和顺序

	groupArgs := make([]interface{}, 0, len(pg)*6)
	groupPlaceholders := make([]string, 0, len(pg))

	for i, g := range pg {
		ph := fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)",
			i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6)
		groupPlaceholders = append(groupPlaceholders, ph)
		groupArgs = append(groupArgs,
			ep.ID, g.Name, g.Order, uid, now, now)
		z.Sugar().Debugf("打印一下原试卷模拟提供的题组信息ID:%v，名字：%v，顺序:%v,", g.ID, g.Name, g.Order)
	}

	fullGroupSQL := fmt.Sprintf(groupSQL, strings.Join(groupPlaceholders, ","))

	// 执行查询获取新题组信息
	rows, err := tx.Query(ctx, fullGroupSQL, groupArgs...)
	if err != nil || forceErr == "query2" {
		return nil, nil, fmt.Errorf("插入题组失败: %w", err)
	}
	defer rows.Close()

	// 创建题组映射：仅使用order作为键
	orderToNewIDMap := make(map[int64]int64)  // 顺序值到新ID的映射
	orderToOrigIDMap := make(map[int64]int64) // 顺序值到原始ID的映射

	// 收集所有返回的题组
	for rows.Next() {
		var newGroupID int64
		var order int64

		if err := rows.Scan(&newGroupID, &order); err != nil || forceErr == "scan" {
			err = fmt.Errorf("扫描题组信息失败:%v", err)
			z.Error(err.Error())
			return nil, nil, err
		}

		orderToNewIDMap[order] = newGroupID
	}
	if err := rows.Err(); err != nil || forceErr == "row" {
		err = fmt.Errorf("遍历扫描题组信息中出错：%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}

	// 创建顺序值到原始ID的映射
	for _, g := range pg {
		orderToOrigIDMap[g.Order.Int64] = g.ID.Int64
	}

	// 题目插入部分
	questionSQL := `INSERT INTO assessuser.t_exam_paper_question (
		score, type, content, options, answers, analysis, title, 
		answer_file_path, test_file_path, input, output, example, repo, 
		creator, create_time, update_time, addi, "order", group_id, question_attachments_path
	) VALUES %s RETURNING id`

	// 那这里就需要构建这个插入数据的数组
	var tqs []*cmn.TExamPaperQuestion
	for _, origGroup := range pg {
		newGroupID, exists := orderToNewIDMap[origGroup.Order.Int64]
		if forceErr == "notExists" {
			exists = false
		}
		if !exists {
			err = fmt.Errorf("无法找到顺序为:%v的题组", origGroup.Order.Int64)
			z.Error(err.Error())
			return nil, nil, err
		}
		// 这里根据旧的题组ID去获取到对应的题目
		// 获取该题组的题目

		qs, exists := pq[origGroup.ID.Int64]
		if forceErr == "qs" {
			exists = false
		}
		if !exists || len(qs) == 0 {
			continue
		}
		//这里构建这个数组
		for _, q := range qs {
			// 处理填空题和简答题
			if q.Type == QuestionCategory.FillInBlank || q.Type == QuestionCategory.ShortAnswer {
				var answers []NonSelectQuestionAnswer
				if forceErr == "jsonA" {
					q.Answers = types.JSONText(`invalid json: missing closing brace`)
				}
				if forceErr == "jsonB" {
					q.Answers = []byte(`[
        {"index":1,"answer":"ans1","alternative_answer":["alt1"],"score":5.0,"grading_rule":"rule1"},
        {"index":2,"answer":"ans2","alternative_answer":["alt2"],"score":5.0,"grading_rule":"rule2"}
    ]`)
					// 设置分数数量为 3（比答案数量多 1）
					q.SubScore = []float64{5.0, 5.0, 5.0}
				}
				if err := json.Unmarshal(q.Answers, &answers); err != nil {
					return nil, nil, fmt.Errorf("解析答案失败: %w", err)
				}
				if len(answers) != len(q.SubScore) {
					return nil, nil, fmt.Errorf("题目ID：%v中的答案数量(%d)与分数数量(%d)不匹配", q.ID, len(answers), len(q.SubScore))
				}
				for i := range answers {
					answers[i].Score = q.SubScore[i]
				}
				jsonData, _ := json.Marshal(answers)
				q.Answers = jsonData
			}

			tq := &cmn.TExamPaperQuestion{}
			tq.Score = q.Score
			tq.Type = null.StringFrom(q.Type)
			tq.Content = q.Content
			tq.Options = q.Options
			tq.Answers = q.Answers
			tq.Analysis = q.Analysis
			tq.Title = q.Title
			tq.AnswerFilePath = q.AnswerFilePath
			tq.TestFilePath = q.TestFilePath
			tq.Input = q.Input
			tq.Output = q.Output
			tq.Example = q.Example
			tq.Repo = q.Repo
			tq.Order = q.Order
			tq.Creator = null.IntFrom(uid)
			tq.CreateTime = null.IntFrom(now)
			tq.UpdateTime = null.IntFrom(now)
			tq.Addi = q.Addi
			tq.GroupID = null.IntFrom(newGroupID)
			tq.QuestionAttachmentsPath = q.QuestionAttachmentsPath

			tqs = append(tqs, tq)
		}
	}

	// 每批最多一次性插入1000考题 1000 * 20  = 20000 参数个数
	_limit := 1000

	// 批量插入考题，避免参数超出postgresql限制
	for start := 0; start < len(tqs); start += _limit {
		end := start + _limit
		if end > len(tqs) {
			end = len(tqs)
		}
		batch := tqs[start:end]
		var (
			placeholders []string
			questionArgs []interface{}
			paramIndex   = 1
		)
		for _, s := range batch {
			phFields := make([]string, 20)
			for i := 0; i < 20; i++ {
				phFields[i] = fmt.Sprintf("$%d", paramIndex)
				paramIndex++
			}
			placeholders = append(placeholders, "("+strings.Join(phFields, ",")+")")

			questionArgs = append(questionArgs,
				s.Score,
				s.Type,
				s.Content,
				s.Options,
				s.Answers,
				s.Analysis,
				s.Title,
				s.AnswerFilePath,
				s.TestFilePath,
				s.Input,
				s.Output,
				s.Example,
				s.Repo,
				s.Creator,
				s.CreateTime,
				s.UpdateTime,
				s.Addi,
				s.Order,
				s.GroupID,
				s.QuestionAttachmentsPath,
			)
		}
		fullQuestionSQL := fmt.Sprintf(questionSQL, strings.Join(placeholders, ","))
		//z.Sugar().Debugf("插入题目批次 [%d-%d] SQL: %s", start, end-1, fullQuestionSQL)
		//z.Sugar().Debugf("插入题目批次 [%d-%d] 参数: %v", start, end-1, questionArgs)

		batchRows, err := tx.Query(ctx, fullQuestionSQL, questionArgs...)
		if err != nil || forceErr == "query3" {
			err = fmt.Errorf("插入题目批次 [%d-%d] 失败: %v", start, end-1, err)
			z.Error(err.Error())
			return nil, nil, err
		}
		batchIndex := start
		for batchRows.Next() {
			var id int64
			if err := batchRows.Scan(&id); err != nil || forceErr == "scan1" {
				batchRows.Close()
				err = fmt.Errorf("扫描题目ID失败 [%d]: %v", batchIndex, err)
				z.Error(err.Error())
				return nil, nil, err
			}
			if batchIndex < len(tqs) {
				tqs[batchIndex].ID = null.IntFrom(id)
			}
			batchIndex++
		}
		// 立即关闭
		batchRows.Close()

		if err := batchRows.Err(); err != nil || forceErr == "row1" {
			err = fmt.Errorf("读取题目ID失败 [%d-%d]: %v", start, end-1, err)
			z.Error(err.Error())
			return nil, nil, err
		}
		if forceErr == "notMatch" {
			batchIndex = 1
			end = 2
		}
		if batchIndex != end {
			err = fmt.Errorf("题目ID数量不匹配: 预期%d, 实际%d", end-start, batchIndex-start)
			z.Error(err.Error())
			return nil, nil, err
		}
	}

	if !genMarkInfo {
		return &ep.ID.Int64, nil, nil
	}
	// 批改配置变量
	var groups []SubjectiveQuestionGroup
	groupQuestions := make(map[int64][]int64)
	for _, question := range tqs {
		if question.Type.String != QuestionCategory.FillInBlank &&
			question.Type.String != QuestionCategory.ShortAnswer {
			continue
		}

		groupID := question.GroupID.Int64
		groupQuestions[groupID] = append(groupQuestions[groupID], question.ID.Int64)
	}

	for groupID, questionIDs := range groupQuestions {
		groups = append(groups, SubjectiveQuestionGroup{
			GroupID:     groupID,
			QuestionIDs: questionIDs,
		})
	}
	return &ep.ID.Int64, groups, nil
}

// LoadExamPaperDetailByUserId 考试/练习 考生/练习生获取试卷 可用于获取考试时/练习时的试卷 也可用于查看学生答题情况时的获取试卷
/*
关键参数说明：
	tx 事务 由于有操作需要先写后读，且只有在同一个事务中操作写与读，才能被共享
	examPaperId 考卷Id
	pSubmissionId 练习生提交Id 大于0则查询练习学生试卷
	examineeId 考生Id 大于0则查询考试学生试卷
	withStudentAnswer 是否携带学生作答信息 如果为true 题目中会携带学生作答信息 反之ExamQuestion结构体中学生作答信息为空
	withAnswer 题目是否携带原答案 如果为true 题目中会携带题目 反之ExamQuestion结构体中题目答案为空
	withAnswerScore 是否携带学生作答分数与解析 如果为true 题目中会携带学生作答题目的得分与答案解析 反之ExamQuestion结构体中学生得分、解析为空
特别说明：若pSubmissionId与examineeId均大于0 默认取考试学生试卷
实现设计思路：
	先获取考卷本身拥有的题组与考题，再获取单一考生/练习生答卷 进行合并自定义过之后的数据（乱序的选项与答案等）
	根据传入参数，选择是否查询学生作答信息、题目原答案解析等
*/
func LoadExamPaperDetailByUserId(ctx context.Context, tx pgx.Tx, examPaperId, pSubmissionId, examineeId int64, withStudentAnswer, withAnswer, withAnswerScore bool) (*cmn.TVExamPaper, map[int64]*cmn.TExamPaperGroup, map[int64][]*ExamQuestion, error) {
	if examPaperId <= 0 {
		err := fmt.Errorf("invalid examPaperId ID param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if pSubmissionId <= 0 && examineeId <= 0 {
		err := fmt.Errorf("invalid pSubmissionId、examineeId  param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	TEST := "test"
	//查看是否需要返回mock的数据
	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			{
				now := time.Now().UnixMilli()
				var p cmn.TVExamPaper
				p.ID = null.IntFrom(101)
				p.ExamSessionID = null.IntFrom(201)
				p.PracticeID = null.IntFrom(201)
				p.Name = null.StringFrom("英语期末试卷")
				p.Creator = null.IntFrom(1)
				p.CreateTime = null.IntFrom(now)
				p.UpdateTime = null.IntFrom(now)
				p.Status = null.StringFrom("00")
				p.TotalScore = null.FloatFrom(6)
				p.QuestionCount = null.IntFrom(2)

				groupMap := make(map[int64]*cmn.TExamPaperGroup)
				groupMap[int64(200)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(200),
					Name:  null.StringFrom("一、单选题（共1题，共3分）"),
					Order: null.IntFrom(1),
				}
				groupMap[int64(201)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(201),
					Name:  null.StringFrom("二、填空题（共1题，共3分）"),
					Order: null.IntFrom(2),
				}

				questionMap := make(map[int64][]*ExamQuestion)
				qList1 := make([]*ExamQuestion, 0)
				var q1 ExamQuestion
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`[
                        {
                            "label": "A",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">瀑布模型</span></p>"
                        },
                        {
                            "label": "B",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">喷泉模型</span></p>"
                        },
                        {
                            "label": "C",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">螺旋模型</span></p>"
                        },
                        {
                            "label": "D",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">增量模型</span></p>"
                        }
                ]`)
				q1.Score = null.FloatFrom(3)
				q1.Order = null.IntFrom(1)
				qList1 = append(qList1, &q1)

				qList2 := make([]*ExamQuestion, 0)
				var q2 ExamQuestion
				q2.ID = null.IntFrom(2045)
				q2.Type = null.StringFrom("06")
				q2.Title = null.StringFrom("")
				q2.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">用例之间的关系主要有三种：</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(1)</span><span style=\"font-family: 等线; font-size: 12pt\">、</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(2) </span><span style=\"font-family: 等线; font-size: 12pt\">和</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\"> (3)</span></p>")
				q2.Options = JSONText(`[]`)
				q2.Score = null.FloatFrom(3)
				// 这里是给主观题使用的小题个数
				q2.AnswerNum = 3
				q2.Order = null.IntFrom(1)
				qList2 = append(qList2, &q2)

				questionMap[int64(200)] = qList1
				questionMap[int64(201)] = qList2

				return &p, groupMap, questionMap, nil

			}
		}
	}

	var err error
	forceErr, _ := ctx.Value("force-error").(string)
	// 构建查询语句变量
	var s, whereClause, query string
	// 选取根据考生ID还是练习提交记录ID
	var queryId int64
	var sAnswer []*cmn.TStudentAnswers
	orderBy := `ORDER BY sa."order"`
	s = `SELECT 
			sa.question_id,
			sa."order",
			CASE WHEN $2 THEN sa.answer ELSE NULL END AS answer,
        	CASE WHEN $3 THEN sa.answer_score ELSE NULL END AS answer_score,
            CASE WHEN $4 THEN 
            	CASE 
                	WHEN q.type IN ('00','02','04') THEN sa.actual_answers 
                	ELSE q.answers 
            	END 
        	END AS answers,
			CASE
				WHEN q.type IN ('00', '02', '04') THEN sa.actual_options
                ELSE q.options
            END AS options
		FROM assessuser.t_student_answers sa
		JOIN assessuser.t_exam_paper_question q ON sa.question_id = q.id
		`
	if pSubmissionId > 0 && examineeId <= 0 {
		whereClause = `WHERE sa.practice_submission_id = $1`
		query = fmt.Sprintf("%s %s %s", s, whereClause, orderBy)
		queryId = pSubmissionId
	} else {
		whereClause = `WHERE sa.examinee_id = $1`
		query = fmt.Sprintf("%s %s %s", s, whereClause, orderBy)
		queryId = examineeId
	}
	rows, err := tx.Query(ctx, query, queryId, withStudentAnswer, withAnswerScore, withAnswer)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("query statement failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a cmn.TStudentAnswers
		// 将属于这个学生的真正题目、学生作答等信息获取出来，解析在答卷结构体中，最后经过循环遍历，嵌入成一个完整的题目
		err = rows.Scan(&a.QuestionID, &a.Order, &a.Answer, &a.AnswerScore, &a.ActualAnswers, &a.ActualOptions)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("scan student answer failed:%v", err)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
		sAnswer = append(sAnswer, &a)
	}
	if len(sAnswer) == 0 || forceErr == "emptyAnswer" {
		err = fmt.Errorf("query empty student answer, please checkout generate exam paper or student answer")
		z.Error(err.Error())
		return nil, nil, nil, err
	}

	// 获取本次考卷基本信息、考卷题组、考题原本信息
	p, pg, pq, err := LoadExamPaperDetailsById(ctx, tx, examPaperId, true, withAnswer, withAnswerScore)
	if err != nil {
		return nil, nil, nil, err
	}

	// 构建指针操作哈希表 避免嵌套O（N*M）循环 无需重新构建题组map
	questionIndex := make(map[int64]*ExamQuestion)
	for _, questions := range pq {
		for _, q := range questions {
			questionIndex[q.ID.Int64] = q
		}
	}
	// 合并学生答卷记录的信息到考题中
	for _, sa := range sAnswer {
		tq, exists := questionIndex[sa.QuestionID.Int64]
		if forceErr == "exist" {
			exists = false
		}
		if !exists {
			err = fmt.Errorf("exam question id:%v not found in exam paper", sa.QuestionID.Int64)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
		// 赋值学生作答情况
		tq.StudentAnswer = sa.Answer
		tq.StudentScore = sa.AnswerScore
		// 赋值真实记录乱序之后的学生答卷
		tq.Order = sa.Order
		tq.Options = sa.ActualOptions
		tq.Answers = sa.ActualAnswers
		if !withAnswerScore {
			tq.Analysis = null.String{}
		}
	}

	// 构建前端需要的题组结构体
	groupMap := make(map[int64]*cmn.TExamPaperGroup)
	for _, g := range pg {
		groupMap[g.ID.Int64] = g
	}
	return p, groupMap, pq, nil
}

// DeleteExamPaperById 物理删除孤数据：考卷、考卷题组、考卷题目、学生答题情况 ： 使用级联删除 待测试
/*
 */
func DeleteExamPaperById(ctx context.Context, tx pgx.Tx, examSessionIDs, practiceIDs []int64) error {
	var err error
	forceErr, _ := ctx.Value("force-error").(string)
	if len(examSessionIDs) == 0 && len(practiceIDs) == 0 {
		err = fmt.Errorf("invalid examSessionIDs or practiceIDs param")
		z.Error(err.Error())
		return err
	}
	invalidIDs := []int64{}
	// 这里有一个非法的名单，然后需要返回给前端
	for _, s := range examSessionIDs {
		// 这里如果
		if s <= 0 {
			invalidIDs = append(invalidIDs, s)
		}
	}
	for _, p := range practiceIDs {
		// 这里如果
		if p <= 0 {
			invalidIDs = append(invalidIDs, p)
		}
	}

	if len(invalidIDs) != 0 {
		err = fmt.Errorf("要删除的考试场次或者练习考卷信息的ID非法 错误如下：%v", invalidIDs)
		z.Error(err.Error())
		return err
	}
	s := `DELETE FROM t_exam_paper WHERE exam_session_id = ANY($1) OR practice_id = ANY($2)`
	_, err = tx.Exec(ctx, s, examSessionIDs, practiceIDs)
	if err != nil || forceErr == "delete" {
		err = fmt.Errorf("级联删除对应考试场次与练习考卷、学生答卷数据失败：%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}
