package grade

import (
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
)

type Map map[string]interface{}
type JSONText = types.JSONText

// ********** 成绩列表接口 **********
type GradeListReq struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	ExamID     int64 `json:"examID"`
	PracticeID int64 `json:"practiceID"`
	Filter     struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Submitted int    `json:"submitted"`
	}
}

type ExamSessionInfo struct {
	ExamID             null.Int    `json:"exam_id"`             // 考试id
	ExamSessionID      null.Int    `json:"exam_session_id"`     // 考试场次id
	PaperName          null.String `json:"paper_name"`          // 考卷名称
	StartTime          null.Int    `json:"start_time"`          // 考试开始时间
	EndTime            null.Int    `json:"end_time"`            // 考试结束时间
	Status             null.String `json:"status"`              // 考试状态
	MarkMode           null.String `json:"mark_mode"`           // 阅卷方式
	TotalScore         null.Float  `json:"total_score"`         // 试卷总分
	AverageScore       null.Float  `json:"average_score"`       // 平均分
	ScheduledExaminees null.Int    `json:"scheduled_examinees"` // 考试应考人数
	ActualExaminees    null.Int    `json:"actual_examinees"`    // 实际参考人数
	PassExaminees      null.Int    `json:"pass_examinees"`      // 及格人数
}

type GradeExam struct {
	ID        null.Int          `json:"id"`        // 考试id
	Name      null.String       `json:"name"`      // 考试名称
	Type      null.String       `json:"type"`      // 考试类型
	Sessions  []ExamSessionInfo `json:"sessions"`  // 考试场次列表
	Submitted null.Bool         `json:"submitted"` // 是否已提交成绩
	Status    null.String       `json:"status"`    // 考试状态
}

type GradePractice struct {
	ID                null.Int    `json:"id"`                 // 练习id
	Name              null.String `json:"name"`               // 练习名称
	TotalScore        null.Float  `json:"total_score"`        // 练习总分
	AverageScore      null.Float  `json:"average_score"`      // 平均分
	CompletedStudents null.Int    `json:"completed_students"` // 完成练习人数
	PassedStudents    null.Int    `json:"passed_students"`    // 通过人数
}

// ********** 成绩分布接口 **********
type ExamSessionGradeDistribution struct {
	ExamID            null.Int    `json:"exam_id"`            // 考试ID
	ExamSessionID     null.Int    `json:"exam_session_id"`    // 考试场次ID
	ExamPaperID       null.Int    `json:"exam_paper_id"`      // 考试试卷ID
	ExamPaperName     null.String `json:"exam_paper_name"`    // 考试试卷名称
	PaperTotalScore   null.Float  `json:"total_score"`        // 试卷总分
	ScoreDistribution []null.Int  `json:"score_distribution"` // 分数分布
}

type ExamGradeDistribution struct {
	ExamID            null.Int                       `json:"exam_id"`            // 考试ID
	ExamName          null.String                    `json:"exam_name"`          // 考试名称
	GradeDistribution []ExamSessionGradeDistribution `json:"grade_distribution"` // 考试成绩分布
}

type PracticeGradeDistribution struct {
	PracticeID        null.Int    `json:"practice_id"`        // 练习ID
	PracticeName      null.String `json:"practice_name"`      // 练习名称
	TotalScore        null.Float  `json:"total_score"`        // 练习总分
	TotalStudents     null.Int    `json:"total_students"`     // 完成练习人数
	GradeDistribution []null.Int  `json:"grade_distribution"` // 练习成绩分布
}

// ********** 考生成绩列表接口 **********
type GradeExamineeListReq struct {
	ExamID     []int64 // 考试ID
	PracticeID []int64 // 练习ID
	Page       int     // 页码
	PageSize   int     // 每页数量
	Filter     struct {
		Keyword string // 关键字:姓名,昵称,电话
	}
}

// ********** 考试 **********
type ExamSessionScore struct {
	ExamID        int64       `json:"exam_id"`         // 考试ID
	ExamSessionID int64       `json:"exam_session_id"` // 考试场次ID
	PaperName     null.String `json:"paper_name"`      // 试卷名称
	Score         null.Float  `json:"score"`           // 学生得分
	SessionNum    int64       `json:"session_num"`     // 场次序号
}

type StudentExamScore struct {
	StudentID    int64              `json:"student_id"`    // 学生ID
	Phone        null.String        `json:"phone"`         // 学生手机号
	Name         null.String        `json:"name"`          // 学生姓名
	Nickname     null.String        `json:"nickname"`      // 学生昵称
	Remark       null.String        `json:"remark"`        // 备注
	TotalScore   float64            `json:"total_score"`   // 总得分
	ExamSessions []ExamSessionScore `json:"exam_sessions"` // 学生在各场次的成绩
}

type ExamineeScoreList struct {
	ExamID        int64              `json:"exam_id"`        // 考试ID
	ExamName      null.String        `json:"exam_name"`      // 考试名称
	StudentScores []StudentExamScore `json:"student_scores"` // 学生成绩列表
}

// ********** 练习 **********
type PracticeExamineeScoreInfo struct {
	StuID        null.Int    `json:"stu_id"`        // 学生ID
	Phone        null.String `json:"phone"`         // 学生手机号
	Name         null.String `json:"name"`          // 学生姓名
	Nickname     null.String `json:"nickname"`      // 学生昵称
	Remark       null.String `json:"remark"`        // 备注
	HighestScore null.Float  `json:"highest_score"` // 最高分
	SubmittedCnt null.Int    `json:"submitted_cnt"` // 提交次数
}

type PracticeScoreList struct {
	PracticeID    int64                       `json:"practice_id"`    // 练习ID
	PracticeName  null.String                 `json:"practice_name"`  // 练习名称
	StudentScores []PracticeExamineeScoreInfo `json:"student_scores"` // 学生成绩列表
}

// ********** 成绩接口 **********
type Analysis struct {
	ExamPaperID          int64                               `json:"exam_paper_id"`          // 考试试卷ID
	ExamPaper            *cmn.TVExamPaper                    `json:"exam_paper"`             // 考卷
	ExamPaperGroup       []*cmn.TExamPaperGroup              `json:"exam_paper_groups"`      // 考卷题组
	ExamPaperQuestion    map[int64][]*examPaper.ExamQuestion `json:"exam_paper_questions"`   // 考卷题目
	QuestionAnswersStats map[null.Int]map[string]int         `json:"question_answers_stats"` // 试题答案统计
	SubjectiveScores     map[null.Int]float64                `json:"subjective_scores"`      // 主观题评分列表
}

// ********** 学生成绩接口 **********
type ExamSessionScoreRank struct {
	StudentID    int64       `json:"student_id"`
	OfficialName null.String `json:"official_name"`
	TotalScore   null.Float  `json:"total_score"`
	Rank         null.Int    `json:"rank"`
}

type SessionInfo struct {
	ID         null.Int `json:"ID,omitempty"`
	PaperID    null.Int `json:"PaperID,omitempty"`
	Duration   null.Int `json:"Duration,omitempty"`
	SessionNum null.Int `json:"SessionNum,omitempty"`
	ExamineeID null.Int `json:"ExamineeID,omitempty"`
}
