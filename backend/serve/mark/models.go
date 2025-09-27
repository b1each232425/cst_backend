package mark

import (
	"errors"
	"go.uber.org/zap"
	"sync"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/ai_mark"
	"w2w.io/serve/examPaper"
)

var z *zap.Logger

type forceErrKey string

const ForceErrKey = forceErrKey("force-err")

var ForceErr = errors.New("")

const (
	TaskTypeAIMarkRequest = "ai_mark:grade"
)

const (
	ExamSessionStatusSubmitted             = "10" // 这里其实是“已批改”
	PracticeSubmissionStatusSubmitted      = "08"
	PracticeWrongSubmissionStatusSubmitted = "08"
)

type User struct {
	ID      int64  `json:"id" validate:"required,gt=0"`
	Level   string `json:"level"`
	IsAdmin bool   `json:"is_admin"`
}

type QueryMarkingListReq struct {
	User         *User     `json:"user" validate:"required"`
	ExamName     string    `json:"exam_name" validate:"max=999"`
	PracticeName string    `json:"practice_name" validate:"max=999"`
	Limit        int       `json:"limit" validate:"required,gt=0,lte=1000"`
	Offset       int       `json:"offset" validate:"gte=0"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `json:"status"`
}

type MarkedInfo struct {
	MarkedPerson    int64 `json:"marked_person"`
	MarkedQuestions int64 `json:"marked_questions"`
}

type Detail struct {
	TeacherID       int64         `json:"teacher_id"`
	QuestionSets    []QuestionSet `json:"question_sets"`
	StudentInfos    []StudentInfo `json:"student_infos"`
	MarkedPerson    int64         `json:"marked_person"`
	MarkedQuestions int64         `json:"marked_questions"`
}

type MarkDetail struct {
	Analyze string  `json:"analyze"`
	Index   int     `json:"index"`
	Score   float64 `json:"score"`
}

type HandleMarkerInfoReq struct {
	Markers        []int64                             `json:"markers"`          // *批改员id数组
	QuestionGroups []examPaper.SubjectiveQuestionGroup `json:"question_groups"`  // TODO 题组（已废弃，不再外部传入，而是内部自己查）
	QuestionIDs    []int64                             `json:"question_ids"`     // 题目id数组
	ExamineeIDs    []int64                             `json:"examinee_ids"`     // 考生id数组
	MarkMode       string                              `json:"mark_mode"`        // *批卷模式 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改
	ExamSessionID  int64                               `json:"exam_session_id"`  // 考试场次id
	PracticeID     int64                               `json:"practice_id"`      // 练习id
	Status         string                              `json:"status"`           // *00 插入批改配置 02 删除批改配置
	ExamSessionIDs []int64                             `json:"exam_session_ids"` // 要删除的考试场次id数组
	PracticeIDs    []int64                             `json:"practice_ids"`     // 要删除的练习id数组
}

type AIMarkRequest struct {
	QuestionDetails []*ai_mark.QuestionDetail
	StudentAnswers  []*ai_mark.StudentAnswer
}

type AIMarkTaskPayLoad struct {
	AIMarkRequest  AIMarkRequest  `json:"ai_mark_request"`
	QueryCondition QueryCondition `json:"query_condition"`
	TaskTotalCount *int           `json:"task_total_count"` // 所属任务的批次总个数
	CountMu        *sync.Mutex
}

type Exam struct {
	Id           int64         `json:"id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Class        string        `json:"class"`
	ExamSessions []ExamSession `json:"exam_sessions"`
}

type ExamSession struct {
	Id                   int64  `json:"id"`
	Name                 string `json:"name"`
	PaperName            string `json:"paper_name"`
	ExamSessionType      string `json:"session_type"`
	MarkMethod           string `json:"mark_method"`
	MarkMode             string `json:"mark_mode"`
	RespondentCount      int    `json:"respondent_count"`
	UnMarkedStudentCount int    `json:"unmarked_student_count"`
	StartTime            int64  `json:"start_time"`
	EndTime              int64  `json:"end_time"`
	Status               string `json:"status"`
	MarkStatus           string `json:"mark_status"`
}

type Practice struct {
	ID                   int64  `json:"id"`                     //练习id
	Name                 string `json:"name"`                   // 练习名称
	MarkMode             string `json:"mark_mode"`              // 批阅方式
	Type                 string `json:"type"`                   // 练习类型
	RespondentCount      int    `json:"respondent_count"`       // 作答人数
	UnMarkedStudentCount int    `json:"unmarked_student_count"` // 未批阅人数
}

type QueryCondition struct {
	TeacherID                 int64 `json:"teacher_id" validate:"required,gt=0"`
	ExamSessionID             int64 `json:"exam_session_id" validate:"gte=0"`
	ExamineeID                int64 `json:"examinee_id" validate:"gte=0"`
	PracticeID                int64 `json:"practice_id" validate:"gte=0"`
	PracticeSubmissionID      int64 `json:"practice_submission_id" validate:"gte=0"`
	PracticeWrongSubmissionID int64 `json:"practice_wrong_submission_id"`
	QuestionID                int64 `json:"question_id"`
}

type MarkerInfo struct {
	MarkerID           int64           `json:"marker_id"`
	MarkInfos          []cmn.TMarkInfo `json:"mark_infos"`
	MarkMode           string          `json:"mark_mode"` // 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改 12：批改单个练习学生 14：批改练习错题
	MarkMethod         string          `json:"mark_method"`
	MarkedStudentNames []string        `json:"marked_student_names"`
}

type QuestionSet struct {
	cmn.TExamPaperGroup
	Score     float64                   `json:"Score"`     // 题组分数
	Questions []*cmn.TExamPaperQuestion `json:"Questions"` // 题目
}

// SubjectiveAnswer 主观题答案结构
type SubjectiveAnswer struct {
	Index             int      `json:"index"`              // 答案索引
	Answer            string   `json:"answer"`             // 答案
	AlternativeAnswer []string `json:"alternative_answer"` // 备选答案
	Score             float64  `json:"score"`              // 分数
	GradingRule       string   `json:"grading_rule"`       // 评分规则
}

type StudentInfo struct {
	OfficialName         null.String `json:"OfficialName"`
	SerialNumber         null.Int    `json:"SerialNumber"`
	ExamineeID           null.Int    `json:"ExamineeID"`
	PracticeSubmissionID null.Int    `json:"PracticeSubmissionID"`
}
