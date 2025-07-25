package grade

import (
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
	"w2w.io/null"
)

// ExamGrade 是考试成绩数据
type GradeListArgs struct {
	Category      string  `json:"category"`
	Page          int     `json:"page"`
	PageSize      int     `json:"pageSize"`
	PassScoreRate float64 `json:"passScoreRate"`
	TeacherID     int     `json:"teacherID"`
	ExamID        int     `json:"examID"`
	PracticeID    int     `json:"practiceID"`
	Filter        struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Submitted int    `json:"submitted"`
	}
}

// ExamSession 是考试场次数据
type ExamSessionInfo struct {
	ExamID             null.Int    `json:"exam_id"`             // 考试id
	ExamSessionID      null.Int    `json:"exam_session_id"`     // 考试场次id
	PaperName          null.String `json:"paper_name"`          // 考卷名称
	StartTime          null.Time   `json:"start_time"`          // 考试开始时间
	EndTime            null.Time   `json:"end_time"`            // 考试结束时间
	MarkMode           null.String `json:"mark_mode"`           // 阅卷方式
	TotalScore         null.Float  `json:"total_score"`         // 试卷总分
	AverageScore       null.Float  `json:"average_score"`       // 平均分
	ScheduledExaminees null.Int    `json:"scheduled_examinees"` // 考试应考人数
	ActualExaminees    null.Int    `json:"actual_examinees"`    // 实际参考人数
	PassExaminees      null.Int    `json:"pass_examinees"`      // 及格人数
}

// GradeExam 是考试成绩数据
type GradeExam struct {
	ID        null.Int          `json:"id"`        // 考试id
	Name      null.String       `json:"name"`      // 考试名称
	Type      null.String       `json:"type"`      // 考试类型
	Sessions  []ExamSessionInfo `json:"sessions"`  // 考试场次列表
	Submitted null.Bool         `json:"submitted"` // 是否已提交成绩
}

// GradePractice 是练习成绩数据
type GradePractice struct {
	ID                null.Int    `json:"id"`                 // 练习id
	Name              null.String `json:"name"`               // 练习名称
	TotalScore        null.Float  `json:"total_score"`        // 练习总分
	AverageScore      null.Float  `json:"average_score"`      // 平均分
	CompletedStudents null.Int    `json:"completed_students"` // 完成练习人数
	PassedStudents    null.Int    `json:"passed_students"`    // 通过人数
}

type GradeSubmitArgs struct {
	TeacherID int64 `json:"teacherID"`
	ExamIDs   []int `json:"examIDs"`
}

type GradeDistributionExamArgs struct {
	ExamID    int // 考试ID
	ColumnNum int // 分布列数
}

type GradeDistributionPracticeArgs struct {
	PracticeID int // 练习ID
	ColumnNum  int // 分布列数
}

type GradeListExamineeExamArgs struct {
	ExamID    []int // 考试ID
	TeacherID int   // 教师ID
	ClassID   int   // 班级ID
	Page      int   // 页码
	PageSize  int   // 每页数量
	Filter    struct {
		Keyword string `json:"keyword"`
	}
}

type GradeListExamineePracticeArgs struct {
	PracticeID []int // 练习ID
	TeacherID  int   // 教师ID
	ClassID    int   // 班级ID
	Page       int   // 页码
	PageSize   int   // 每页数量
	Filter     struct {
		Keyword string `json:"keyword"`
	}
}

// ExamSessionGradeDistribution 是考试场次成绩分布数据
type ExamSessionGradeDistribution struct {
	ExamID            null.Int    `json:"exam_id"`            // 考试ID
	ExamSessionID     null.Int    `json:"exam_session_id"`    // 考试场次ID
	ExamPaperID       null.Int    `json:"exam_paper_id"`      // 考试试卷ID
	ExamPaperName     null.String `json:"exam_paper_name"`    // 考试试卷名称
	PaperTotalScore   null.Float  `json:"total_score"`        // 试卷总分
	ScoreDistribution []null.Int  `json:"score_distribution"` // 分数分布
}

// ExamGradeDistribution 是考试成绩分布数据
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

type ExamExamineeScoreInfo struct {
	StuID         null.Int    `json:"stu_id"`          // 学生ID
	Phone         null.String `json:"phone"`           // 学生手机号
	Name          null.String `json:"name"`            // 学生姓名
	Nickname      null.String `json:"nickname"`        // 学生昵称
	Score         null.Float  `json:"score"`           // 成绩
	TotalScore    null.Float  `json:"total_score"`     // 总分
	Remark        null.String `json:"remark"`          // 备注
	ExamID        null.Int    `json:"exam_id"`         // 考试ID
	ExamSessionID null.Int    `json:"exam_session_id"` // 考试场次ID
}

type PracticeExamineeScoreInfo struct {
	StuID        null.Int    `json:"stu_id"`        // 学生ID
	Phone        null.String `json:"phone"`         // 学生手机号
	Name         null.String `json:"name"`          // 学生姓名
	Nickname     null.String `json:"nickname"`      // 学生昵称
	Remark       null.String `json:"remark"`        // 备注
	PracticeID   null.Int    `json:"practice_id"`   // 练习ID
	HighestScore null.Float  `json:"highest_score"` // 最高分
	SubmittedCnt null.Int    `json:"submitted_cnt"` // 提交次数
}

type QuestionGroups struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type JSONText = types.JSONText

type GradeAnalysisExamArgs struct {
	ExamSessionID int // 考试场次ID
	TeacherID     int // 教师ID
	ClassID       int // 班级ID
}

type GradeAnalysisPracticeArgs struct {
	PracticeID int // 练习ID
	PaperID    int // 试卷ID
	TeacherID  int // 教师ID
	ClassID    int // 班级ID
}

type ExamAnalysis struct {
	ExamSessionID        null.Int                    `json:"exam_session_id"`        // 考试场次ID
	ExamPaperID          int64                       `json:"exam_paper_id"`          // 考试试卷ID
	QuestionGroups       []JSONText                  `json:"question_groups"`        // 题目分组
	Questions            []cmn.TExamPaperQuestion    `json:"questions"`              // 试题分析列表
	ExamPaper            *cmn.TVPaper                `json:"exam_paper"`             // 试卷详情
	QuestionAnswersStats map[null.Int]map[string]int `json:"question_answers_stats"` // 试题答案统计
	SubjectiveScores     map[null.Int]float64        `json:"subjective_scores"`      // 主观题评分列表
}

type PracticeAnalysis struct {
	ExamPaperID          null.Int                    `json:"exam_paper_id"`          // 考试试卷ID
	Questions            []cmn.TExamPaperQuestion    `json:"questions"`              // 试题分析列表
	QuestionAnswersStats map[null.Int]map[string]int `json:"question_answers_stats"` // 试题答案统计
	SubjectiveScores     map[null.Int]float64        `json:"subjective_scores"`      // 主观题评分列表
	QuestionGroups       []JSONText                  `json:"question_groups"`        // 题目分组
}
