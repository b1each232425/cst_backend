/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷-答卷数据库层
 * @Date: 2025-07-21 13:14:34
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-24 12:00:30
 */
package examPaper

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"math/rand"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

// LoadExamPaperDetailsById 获取完整考卷信息（包括试卷信息、题组信息、题目信息） 支持根据参数的传入决定是否需要查询题组题目、去除答案等 用于生成与绑定学生答卷
/*
关键参数说明：
	epid 考卷Id
	withQuestions 是否查询题组及题目信息 对应返回体[]*cmn.TExamPaperGroup、map[int64][]*ExamQuestion
	withAnswers 题目是否需要答案
*/
func LoadExamPaperDetailsById(ctx context.Context, epid int64, withQuestions, withAnswers bool) (*cmn.TVExamPaper, []*cmn.TExamPaperGroup, map[int64][]*ExamQuestion, error) {
	var err error
	if epid <= 0 {
		err = fmt.Errorf("invalid examPaper ID param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	// 一张考卷卷拥有的题组map
	var examGroups []*cmn.TExamPaperGroup
	// 一个题组下拥有的题目数组
	examQuestions := make(map[int64][]*ExamQuestion)
	sqlxDB := cmn.GetDbConn()
	var ep cmn.TVExamPaper
	// 动态构建 SQL 字段
	fields := []string{"id", "exam_session_id", "practice_id", "name", "creator", "create_time",
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
	err = sqlxDB.QueryRowxContext(ctx, s, epid, PaperStatus.Normal).Scan(scanArgs...)
	if err != nil {
		err = fmt.Errorf("LoadExamPaperById call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if withQuestions && len(ep.GroupsData) > 0 {
		var groupData []ExamGroup
		err = json.Unmarshal(ep.GroupsData, &groupData)
		if err != nil {
			z.Error("unmarshal group data failed", zap.Error(err))
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
			if withAnswers {
				for idx := range v.Questions {
					q := v.Questions[idx]
					examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
				}
			} else {
				// 去除答案并设置题目中小题数量
				for idx := range v.Questions {
					q := v.Questions[idx]
					var answersSlice []interface{}
					if len(q.Answers) > 0 {
						if err = json.Unmarshal([]byte(q.Answers), &answersSlice); err != nil {
							err = fmt.Errorf("failed to unmarshal Answers questionId:%v for:%v", q.ID.Int64, err)
							z.Error(err.Error())
							return &ep, nil, nil, err
						}
					}
					q.AnswerNum = len(answersSlice)
					q.Answers = nil
					q.Analysis = null.String{}
					examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
				}
			}
		}
	}
	// 信息已经取出，直接清除
	ep.GroupsData = nil
	return &ep, examGroups, examQuestions, nil
}

// GenerateAnswerQuestion 根据一次考卷生成学生答卷信息 批量插入学生答卷
func GenerateAnswerQuestion(ctx context.Context, tx *sqlx.Tx, req GenerateAnswerQuestionsRequest, uid int64) error {
	var err error
	now := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(now))
	err = cmn.Validate(req)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	var studentIds []int64
	if req.Category == PaperCategory.Exam {
		studentIds = req.ExamineeIDs
	} else if req.Category == PaperCategory.Practice {
		studentIds = req.PracticeSubmissionID
	} else {
		err = fmt.Errorf("invalid param:category:%v", req.Category)
		z.Error(err.Error())
		return err
	}

	// 获取考题
	_, pg, pq, err := LoadExamPaperDetailsById(ctx, req.ExamPaperID, true, true)
	if err != nil {
		err = fmt.Errorf("LoadExamPaperById call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	var Answers []*cmn.TStudentAnswers
	for _, id := range studentIds {
		for _, g := range pg {
			order := 1
			pqList := pq[g.ID.Int64]
			if req.IsQuestionRandom {
				// 这里会重复的影响到原数组 ， 每一次也是会打乱一次题组内的顺序的
				r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			}
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					Type:       null.StringFrom(req.Category),
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
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					if req.IsOptionRandom {
						answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, actualOptions, actualAnswers)
						if err != nil {
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
		err = BatchInsertStudentAnswer(ctx, tx, batchStudentAnswers, uid)
		if err != nil {
			err = fmt.Errorf("BatchInsertStudentAnswer call failed:%v", err)
			z.Error(err.Error())
			return err
		}
	}

	return nil

}

// BatchInsertStudentAnswer  按照批次批量生成学生试卷每道题的答题（支持题组和题目按需解析）
/*
关键参数说明：
	sa 需要新增的学生答题数组
	uid 操作者
*/
func BatchInsertStudentAnswer(ctx context.Context, tx *sqlx.Tx, sa []*cmn.TStudentAnswers, uid int64) error {
	var err error
	if len(sa) == 0 {
		err = fmt.Errorf("empty params student answer")
		z.Error(err.Error())
		return nil
	}
	now := time.Now().UnixMilli()
	tArgs := make([]interface{}, 0, len(sa)*11)
	for _, a := range sa {
		tArgs = append(tArgs, a.Type, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID, uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers)
	}

	t := `INSERT INTO t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			   creator, create_time,
			 update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
	tStr := strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?)", len(sa)-1) + "(?,?,?,?,?,?,?,?,?,?,?)"

	s := fmt.Sprintf(t, tStr)
	q, args, err := sqlx.In(s, tArgs...)
	if err != nil {
		err = fmt.Errorf("prepare student answer insert failed:%v", err)
		z.Error(err.Error())
		return err
	}
	insertQuery := sqlx.Rebind(sqlx.DOLLAR, q)
	z.Sugar().Debugf("打印输出一下插入学生作答SQL语句:%v", insertQuery)
	z.Sugar().Debugf("打印输出一下插入学生作答SQL参数:%v", args)
	_, err = tx.ExecContext(ctx, insertQuery, args...)
	if err != nil {
		err = fmt.Errorf("create student answer insert failed call failed:%v", err)
		z.Error(err.Error())
		return err
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
	var err error
	// 一张试卷拥有的题组map
	var groups []*cmn.TPaperGroup
	// 一个题组下拥有的题目数组
	questions := make(map[int64][]*Question)
	var p cmn.TVPaper
	sqlxDB := cmn.GetDbConn()
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
	s := fmt.Sprintf("SELECT %s FROM v_paper WHERE id=$1 AND status != $2", strings.Join(fields, ","))
	err = sqlxDB.QueryRowxContext(ctx, s, paperId, PaperStatus.Invalid).Scan(scanArgs...)
	if err != nil {
		err = fmt.Errorf("LoadExamPaperById call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if withQuestions && len(p.GroupsData) > 0 {
		var groupData []Group
		if err := json.Unmarshal(p.GroupsData, &groupData); err != nil {
			z.Error("unmarshal group data failed", zap.Error(err))
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
				questions[v.ID.Int64] = append(questions[v.ID.Int64], &q)
			}
		}
	}

	return &p, groups, questions, nil
}

// GenerateExamPaper 生成一张可用考卷（包括题组与题目）
/*
关键参数说明：
	category 考卷服务类型 00：考试 02：练习
	paperId 试卷模版Id
	practiceId 练习Id 若类型为练习必填 否则填0
	examSessionId 考试场次Id 若类型为考试必填 否则填0
	uid 操作人Id
	genMarkInfo 是否生成批改信息 用于考试创建时配置批改信息
*/
func GenerateExamPaper(ctx context.Context, tx *sql.Tx, category string, paperId, practiceId, examSessionId int64, uid int64, genMarkInfo bool) (*int64, []SubjectiveQuestionGroup, error) {
	var err error
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
	p, pg, pq, err := LoadPaperTemplateById(ctx, paperId, true)
	if err != nil {
		err = fmt.Errorf("LoadPaperTemplateById call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}
	if len(pg) == 0 {
		err = fmt.Errorf("invalid paper question group")
		z.Error(err.Error())
		return nil, nil, err
	}
	var ep cmn.TExamPaper
	if category == PaperCategory.Exam && examSessionId > 0 {
		ep.ExamSessionID = null.IntFrom(examSessionId)
	} else if category == PaperCategory.Practice && practiceId > 0 {
		ep.PracticeID = null.IntFrom(practiceId)
	} else {
		err = fmt.Errorf("invalid mapping between category and request id")
		z.Error(err.Error())
		return nil, nil, err
	}
	ep.Name = p.Name
	ep.Creator = null.IntFrom(uid)
	ep.CreateTime = null.IntFrom(time.Now().UnixMilli())
	ep.UpdateTime = null.IntFrom(time.Now().UnixMilli())

	// 插入一张考卷
	s := `INSERT INTO t_exam_paper (exam_session_id,practice_id,name,creator,create_time,update_time) VALUES ($1,$2,$3,$4,$5,$6)`
	err = tx.QueryRowContext(ctx, s, &ep.ExamSessionID, &ep.PracticeID, ep.Name, ep.Creator, ep.CreateTime, ep.UpdateTime).Scan(&ep.ID)
	if err != nil {
		err = fmt.Errorf("create exam paper call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}

	// 插入考卷题组数据
	t := `INSERT INTO t_exam_paper_group (exam_paper_id,name,“order”,creator,create_time,update_time) VALUES %s`
	tstr := strings.Repeat("(?, ?, ?, ?, ?, ?),", len(pg)-1) + "(?, ?, ?, ?, ?, ?)"
	tGroupArgs := make([]interface{}, 0, len(pg)*6)

	for _, g := range pg {
		tGroupArgs = append(tGroupArgs, ep.ID, g.Name, g.Order, uid, time.Now().UnixMilli(), time.Now().UnixMilli())
	}
	s1 := fmt.Sprintf(t, tstr)
	groupQuery, groupArgs, err := sqlx.In(s1, tGroupArgs...)
	if err != nil {
		err = fmt.Errorf("prepare group query failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}
	groupQuery = sqlx.Rebind(sqlx.DOLLAR, groupQuery)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", groupQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", groupArgs)
	_, err = tx.ExecContext(ctx, groupQuery, groupArgs...)
	if err != nil {
		err = fmt.Errorf("create exam paper group call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}

	var tQuestionArgs []interface{}
	now := time.Now().UnixMilli()
	var questionCount int
	// 批改配置变量
	var groups []SubjectiveQuestionGroup

	// 遍历这个题组题目的group
	for gid, qs := range pq {
		var group SubjectiveQuestionGroup
		if genMarkInfo {
			group.GroupID = gid
			group.QuestionIDs = make([]int64, 0)
		}
		for _, q := range qs {
			if genMarkInfo {
				group.QuestionIDs = append(group.QuestionIDs, q.ID.Int64)
			}
			questionCount++
			// 若是填空题、简答题 动态进行赋予分值
			if q.Type == QuestionCategory.FillInBlank || q.Type == QuestionCategory.ShortAnswer {
				var a []NonSelectQuestionAnswer
				if err = json.Unmarshal(q.Answers, &a); err != nil {
					err = fmt.Errorf("unmarshal answer failed:%v", err)
					z.Error(err.Error())
					return nil, nil, err
				}
				// 校验分数配置
				if len(a) != len(q.SubScore) {
					z.Sugar().Errorw("分数配置不匹配", "question_id", q.ID,
						"expected", len(q.SubScore),
						"actual", len(a))
					return nil, nil, fmt.Errorf("答案分数配置不匹配")
				}
				// 设置答案分数
				for i, ans := range a {
					if i < len(q.SubScore) {
						// 设置分数
						ans.Score = q.SubScore[i]
						a[i] = ans
					}
				}
				jsonAnswer, _ := json.Marshal(a)
				q.Answers = jsonAnswer
			}
			tQuestionArgs = append(tQuestionArgs, q.Score, q.Type, q.Content, q.Options,
				q.Answers, q.Analysis, q.Title, q.AnswerFilePath, q.TestFilePath, q.Input,
				q.Output, q.Example, q.Repo, uid, now, now, q.Addi, q.Order, gid, q.QuestionAttachmentsPath,
			)
		}
		if genMarkInfo {
			groups = append(groups, group)
		}
	}
	// 批量生成考题
	t1 := `INSERT INTO t_exam_paper_question (score,type,content,options,answer,analysis,title,answer_file_path,test_file_path,input,output,example,repo,creator,create_time,update_time,addi,"order",group_id,question_attachments_path) VALUES %s`
	tstr1 := strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?),", questionCount-1) +
		"(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	s2 := fmt.Sprintf(t1, tstr1)
	questionQuery, questionArgs, err := sqlx.In(s2, tQuestionArgs...)
	if err != nil {
		err = fmt.Errorf("prepare question query failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}
	questionQuery = sqlx.Rebind(sqlx.DOLLAR, questionQuery)
	z.Sugar().Debugf("打印输出一下题目SQL语句:%v", questionQuery)
	z.Sugar().Debugf("打印输出一下题目SQL参数:%v", questionArgs)
	_, err = tx.ExecContext(ctx, questionQuery, questionArgs...)
	if err != nil {
		err = fmt.Errorf("create exam paper question call failed:%v", err)
		z.Error(err.Error())
		return nil, nil, err
	}

	return &ep.ID.Int64, groups, nil

}

// LoadExamPaperDetailByUserId 考试/练习 考生/练习生获取试卷 可用于获取考试时/练习时的试卷 也可用于查看学生答题情况时的试卷
/*
关键参数说明：
	examPaperId 考卷Id
	pSubmissionId 练习生提交Id
	examineeId 考生Id
	withStudentAnswer 是否携带学生作答信息 如果为true 题目中会携带学生作答信息 反之ExamQuestion结构体中学生作答信息为空
	withAnswer 题目是否携带原答案、解析 如果为true 题目中会携带题目 反之ExamQuestion结构体中题目答案、解析为空
	withAnswerScore 是否携带学生作答分数 如果为true 题目中会携带学生作答题目的得分 反之ExamQuestion结构体中学生得分为空
实现设计思路：
	先获取考卷本身拥有的题组与考题，再获取单一考生/练习生答卷 进行合并自定义过之后的数据（乱序的选项与答案等）
	根据传入参数，选择是否查询学生作答信息、题目原答案解析等
*/
func LoadExamPaperDetailByUserId(ctx context.Context, examPaperId, pSubmissionId, examineeId int64, withStudentAnswer, withAnswer, withAnswerScore bool) (*cmn.TVExamPaper, map[int64]*cmn.TExamPaperGroup, map[int64][]*ExamQuestion, error) {
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
				var p *cmn.TVExamPaper
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
				var q1 *ExamQuestion
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`[
                    [
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
                    ]
                ]`)
				q1.Score = null.FloatFrom(3)
				q1.Order = null.IntFrom(1)
				qList1 = append(qList1, q1)

				qList2 := make([]*ExamQuestion, 0)
				var q2 *ExamQuestion
				q2.ID = null.IntFrom(2045)
				q2.Type = null.StringFrom("06")
				q2.Title = null.StringFrom("")
				q2.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">用例之间的关系主要有三种：</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(1)</span><span style=\"font-family: 等线; font-size: 12pt\">、</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(2) </span><span style=\"font-family: 等线; font-size: 12pt\">和</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\"> (3)</span></p>")
				q2.Options = JSONText(`[]`)
				q2.Score = null.FloatFrom(3)
				// 这里是给主观题使用的小题个数
				q2.AnswerNum = 3
				q2.Order = null.IntFrom(1)
				qList2 = append(qList2, q2)

				questionMap[int64(200)] = qList1
				questionMap[int64(201)] = qList2

				return p, groupMap, questionMap, nil

			}
		default:
			{
				err := fmt.Errorf("不成功都是失败")
				return nil, nil, nil, err
			}
		}
	}

	var err error
	// 构建查询语句变量
	var s, whereClause, query string
	// 选取根据考生ID还是练习提交记录ID
	var queryId int64
	var stmt *sqlx.Stmt
	sqlxDB := cmn.GetDbConn()
	var sAnswer []*cmn.TStudentAnswers
	orderBy := `ORDER BY sa."order"`
	s = `SELECT 
			sa.question_id,sa."order",
			CASE WHEN $2 THEN sa.answer ELSE NULL END AS answer,
        	CASE WHEN $3 THEN sa.answer_score ELSE NULL END AS answer_score,
            CASE WHEN $4 THEN 
            	CASE 
                	WHEN q.type IN ('00','02','04') THEN sa.actual_answers 
                	ELSE q.answers 
            	END 
        	END AS answers
			CASE
				WHEN q.type IN ('00', '02', '04') THEN sa.actual_options
                ELSE q.options
            END AS options,
		FROM t_student_answers sa
		JOIN t_exam_paper_question q ON sa.question_id = q.id
		`
	if pSubmissionId > 0 && examineeId <= 0 {
		whereClause = `WHERE sa.practice_submission_id = $1`
		query = fmt.Sprintf("%s %s %s", s, whereClause, orderBy)
		queryId = pSubmissionId
	} else if pSubmissionId <= 0 {
		whereClause = `WHERE sa.examinee_id = $1`
		query = fmt.Sprintf("%s %s %s", s, whereClause, orderBy)
		queryId = examineeId
	} else {
		err = fmt.Errorf("invalid pSubmissionId、examineeId  param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	stmt, err = sqlxDB.Preparex(query)
	if err != nil {
		err = fmt.Errorf("prepare statement failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	defer func() {
		err = stmt.Close()
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	rows, err := stmt.QueryxContext(ctx, queryId, withStudentAnswer, withAnswerScore, withAnswer)
	if err != nil {
		err = fmt.Errorf("query statement failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()

	for rows.Next() {
		var a cmn.TStudentAnswers
		// 将属于这个学生的真正题目、学生作答等信息获取出来，解析在答卷结构体中，最后经过循环遍历，嵌入成一个完整的题目
		err = rows.Scan(&a.QuestionID, &a.Order, &a.Answer, &a.AnswerScore, &a.ActualAnswers, &a.ActualOptions)
		if err != nil {
			err = fmt.Errorf("解析学生答卷错误：%v", err)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
		sAnswer = append(sAnswer, &a)
	}
	if len(sAnswer) == 0 {
		err = fmt.Errorf("查询出学生答卷为空，请检查查询答卷语句或生成答卷逻辑")
		z.Error(err.Error())
		return nil, nil, nil, err
	}

	// 获取本次考卷基本信息、考卷题组、考题原本信息
	p, pg, pq, err := LoadExamPaperDetailsById(ctx, examPaperId, true, withAnswer)
	if err != nil {
		err = fmt.Errorf("LoadExamPaperDetailsById call failed:%v", err)
		z.Error(err.Error())
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
		if !exists {
			z.Sugar().Warnf("exam question id:%v not found in examPaper:", sa.QuestionID.Int64)
			continue
		}
		// 赋值学生作答情况
		tq.StudentAnswer = sa.Answer
		tq.StudentScore = sa.AnswerScore
		// 赋值真实记录乱序之后的学生答卷
		tq.Order = sa.Order
		tq.Options = sa.ActualOptions
		tq.Answers = sa.ActualAnswers
	}

	// 构建前端需要的题组结构体
	groupMap := make(map[int64]*cmn.TExamPaperGroup)
	for _, g := range pg {
		groupMap[g.ID.Int64] = g
	}
	return p, groupMap, pq, nil
}
