/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷-答卷数据库层
 * @Date: 2025-07-21 13:14:34
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-21 15:45:07
 */
package examPaper

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

// LoadExamPaperDetailsById 获取完整考卷信息（包括试卷信息、题组信息、题目信息） 支持根据参数的传入决定是否需要查询题组题目、去除答案等
// 无乱序前提下，仅查看考卷信息即可 无需查看学生答卷信息 无需考生信息（因为考卷没有进行乱序处理）
/*
关键参数说明：
	epid 考卷Id
	withQuestions 是否查询题组及题目信息
	withAnswers 题目是否需要去掉答案
*/
// TODO 增加乱序处理 ：需要查询t_student_answer表，对题目数据进行处理
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
			// 这里是创建题组 如果是不存在这个map中，避免重复创建题组
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
			// 初始化题目切片,以免被覆盖
			if _, exists := examQuestions[v.ID.Int64]; !exists {
				examQuestions[v.ID.Int64] = make([]*ExamQuestion, 0)
			}
			// 去除答案
			if !withAnswers {
				// 安全追加题目（避免指针重用）
				for idx := range v.Questions {
					q := v.Questions[idx] // 创建副本
					examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
				}
			} else {
				// 安全追加题目（避免指针重用）
				for idx := range v.Questions {
					q := v.Questions[idx] // 创建副本
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
					examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
				}
			}

		}
	}

	return &ep, examGroups, examQuestions, nil
}

// GenerateAnswerQuestion 生成学生答题信息
// TODO 重构调用时机与传入参数
//
//	也就是根据每一个考题，去创建属于一次练习或者一次考试场次的这个学生的答题情况
func GenerateAnswerQuestion(ctx context.Context, tx *sqlx.Tx, req GenerateAnswerQuestionsRequest, uid int64) error {
	var err error
	now := time.Now().UnixMilli()
	err = cmn.Validate(req)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	var AnswererIDs []int64
	if req.Category == PaperCategory.Exam {
		AnswererIDs = req.ExamineeIDs
	} else if req.Category == PaperCategory.Practice {
		AnswererIDs = req.PracticeSubmissionID
	} else {
		err = fmt.Errorf("invalid param:category:%v", req.Category)
		z.Error(err.Error())
		return err
	}

	_, pg, pq, err := LoadExamPaperDetailsById(ctx, req.ExamPaperID, true, true)
	if err != nil {
		err = fmt.Errorf("LoadExamPaperById call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	var Answers []cmn.TStudentAnswers
	// 每个学生，都需要创建每一个试卷中的题目为答题，并且是一次性创建
	for _, id := range AnswererIDs {
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					Type:       null.StringFrom(req.Category),
					QuestionID: q.ID,
					Creator:    null.IntFrom(uid),
					GroupID:    g.ID,
					Order:      q.Order,
				}
				if req.Category == PaperCategory.Exam {
					answer.ExamineeID = null.IntFrom(id)
				}
				if req.Category == PaperCategory.Practice {
					answer.PracticeSubmissionID = null.IntFrom(id)
				}
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					answer.ActualAnswers = q.Answers
					answer.ActualOptions = q.Options
				}
				Answers = append(Answers, *answer)
			}
		}
	}

	t := `INSERT INTO t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			   creator, create_time,
			 update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
	// 插入学生作答数据
	tstr := strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?),", len(Answers)-1) + "(?,?,?,?,?,?,?,?,?,?,?)"
	tArgs := make([]interface{}, 0, len(Answers)*11)

	for _, a := range Answers {
		tArgs = append(tArgs, a.Type, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID, a.Creator, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers)
	}
	s1 := fmt.Sprintf(t, tstr)
	insertQuery, args, err := sqlx.In(s1, tArgs...)
	if err != nil {
		err = fmt.Errorf("prepare student answer insert failed:%v", err)
		z.Error(err.Error())
		return err
	}
	insertQuery = sqlx.Rebind(sqlx.DOLLAR, insertQuery)
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
