package mark

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"w2w.io/cmn"
	"w2w.io/null"
)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func TestQueryStudentAnswersByMarkMode(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		answerType     string
		cond           QueryCondition
		markerInfo     MarkerInfo
		expectedErrStr string
	}{
		{
			name:       "success with 客观题",
			answerType: "02",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "",
		},
		{
			name:       "success with 练习客观题",
			answerType: "02",
			cond: QueryCondition{
				PracticeID: 21,
				TeacherID:  testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "",
		},
		{
			name:       "success with 主观题",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "",
		},
		{
			name:       "success with markMode:04",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "04",
				MarkerID: testedTeacherID,
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkExamineeIds: types.JSONText(`[2201]`),
					},
				},
			},
			expectedErrStr: "",
		},
		{
			name:       "success with markMode:04（单个考生）",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
				ExamineeID:    2201,
			},
			markerInfo: MarkerInfo{
				MarkMode: "04",
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "",
		},
		{
			name:       "success with markMode:06",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 102,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkerID: testedTeacherID,
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405]`),
					},
				},
			},
			expectedErrStr: "",
		},
		{
			name:       "success with markMode:12（查询单个练习学生）",
			answerType: "02",
			cond: QueryCondition{
				PracticeID:           21,
				TeacherID:            testedTeacherID,
				PracticeSubmissionID: 2401,
			},
			markerInfo: MarkerInfo{
				MarkMode: "12",
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "",
		},
		{
			name:       "invalid answer type",
			answerType: "",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "invalid answer_type",
		},
		{
			name:       "invalid query condition",
			answerType: "02",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name:       "markMode:04 - invalid examinee ids in mark_infos",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "04",
				MarkerID: testedTeacherID,
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkExamineeIds: types.JSONText(`]`),
					},
				},
			},
			expectedErrStr: "unmarshal mark examinee ids error",
		},
		{
			name:       "markMode:06 - invalid question ids in mark_infos",
			answerType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkerID: testedTeacherID,
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`]`),
					},
				},
			},
			expectedErrStr: "unable to unmarshal question_ids",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := QueryStudentAnswersByMarkMode(ctx, tt.answerType, tt.cond, tt.markerInfo)
			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

func TestUpdateStudentAnswerScore(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		markingResults []*cmn.TMark
		cond           QueryCondition
		expectedErrStr string
	}{
		{
			name: "success",
			markingResults: []*cmn.TMark{
				{
					QuestionID: null.IntFrom(401),
					TeacherID:  null.IntFrom(testedTeacherID),
					Score:      null.FloatFrom(2.5),
					ExamineeID: null.IntFrom(2201),
				},
			},
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedErrStr: "",
		},
		{
			name: "success with practice",
			markingResults: []*cmn.TMark{
				{
					QuestionID:           null.IntFrom(424),
					TeacherID:            null.IntFrom(testedTeacherID),
					Score:                null.FloatFrom(2.5),
					PracticeSubmissionID: null.IntFrom(2401),
				},
			},
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 21,
			},
			expectedErrStr: "",
		},
		{
			name: "no marking results for update",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedErrStr: "no marking results for update",
		},
		{
			name: "invalid params",
			markingResults: []*cmn.TMark{
				{
					QuestionID: null.IntFrom(401),
					TeacherID:  null.IntFrom(testedTeacherID),
					Score:      null.FloatFrom(2.5),
					ExamineeID: null.IntFrom(2201),
				},
			},
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "invalid params: exam_session_id or practice_id is required",
		},
		{
			name: "question_id is required in mark",
			markingResults: []*cmn.TMark{
				{
					TeacherID:  null.IntFrom(testedTeacherID),
					Score:      null.FloatFrom(2.5),
					ExamineeID: null.IntFrom(2201),
				},
			},
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedErrStr: "question_id is required in mark",
		},
		{
			name: "teacher id is required in mark",
			markingResults: []*cmn.TMark{
				{
					QuestionID: null.IntFrom(401),
					Score:      null.FloatFrom(2.5),
					ExamineeID: null.IntFrom(2201),
				},
			},
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedErrStr: "teacher id is required in mark",
		},
		{
			name: "examinee_id is required where exam_session_id is specified",
			markingResults: []*cmn.TMark{
				{
					QuestionID:           null.IntFrom(401),
					TeacherID:            null.IntFrom(testedTeacherID),
					Score:                null.FloatFrom(2.5),
					PracticeSubmissionID: null.IntFrom(2401),
				},
			},
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedErrStr: "examinee_id is required where exam_session_id is specified",
		},
		{
			name: "practice_submission_id is required where practice_id is specified",
			markingResults: []*cmn.TMark{
				{
					QuestionID: null.IntFrom(401),
					TeacherID:  null.IntFrom(testedTeacherID),
					Score:      null.FloatFrom(2.5),
					ExamineeID: null.IntFrom(2201),
				},
			},
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 21,
			},
			expectedErrStr: "practice_submission_id is required where practice_id is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := updateStudentAnswerScore(ctx, tt.markingResults, tt.cond)
			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

func TestUpdateExamSessionState(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		examSessionIDs []int64
		status         string
		expectedErrStr string
	}{
		{
			name:           "success",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{101},
			status:         "10",
			expectedErrStr: "",
		},
		{
			name:           "no examSessionIDs to update",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{},
			status:         "10",
			expectedErrStr: "no examSessionIDs to update",
		},
		{
			name:           "invalid params: status is required",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{101},
			status:         "",
			expectedErrStr: "invalid params: status is required",
		},
		{
			name:           "invalid params: teacher_id is required",
			teacherID:      -3,
			examSessionIDs: []int64{101},
			status:         "10",
			expectedErrStr: "invalid params: teacher_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pgxConn := cmn.GetPgxConn()
			var tx pgx.Tx
			var err error
			tx, err = pgxConn.Begin(ctx)
			if err != nil {
				err = fmt.Errorf("begin transaction error: %v", err)
				panic(err)
			}

			defer func() {
				if err == nil {
					err = tx.Commit(ctx)
				}
				if err != nil {
					err_ := tx.Rollback(ctx)
					if err_ != nil {
						z.Error(err_.Error())
						return
					}
				}
			}()

			_, err = updateExamSessionState(ctx, &tx, tt.teacherID, tt.examSessionIDs, tt.status)
			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

func TestUpdateMarkerInfoState(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		ids            []int64
		mode           string
		expectedErrStr string
	}{
		{
			name:      "success",
			teacherID: testedTeacherID,
			ids:       []int64{101},
			mode:      "00",
		},
		{
			name:      "success with practice",
			teacherID: testedTeacherID,
			ids:       []int64{21},
			mode:      "02",
		},
		{
			name:           "invalid teacher ID param",
			ids:            []int64{101},
			mode:           "00",
			expectedErrStr: "invalid teacher ID param",
		},
		{
			name:           "invalid teacher ID param",
			teacherID:      testedTeacherID,
			ids:            []int64{},
			mode:           "00",
			expectedErrStr: "no exam_session_ids or practice_ids to update",
		},
		{
			name:           "invalid update mode",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "abc",
			expectedErrStr: "invalid update mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pgxConn := cmn.GetPgxConn()
			var tx pgx.Tx
			var err error
			tx, err = pgxConn.Begin(ctx)
			if err != nil {
				err = fmt.Errorf("begin transaction error: %v", err)
				panic(err)
			}

			defer func() {
				if err == nil {
					err = tx.Commit(ctx)
				}
				if err != nil {
					err_ := tx.Rollback(ctx)
					if err_ != nil {
						z.Error(err_.Error())
						return
					}
				}
			}()

			_, err = UpdateMarkerInfoState(ctx, &tx, tt.teacherID, tt.ids, tt.mode)
			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

func TestQueryMarkerInfo(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		expectedErrStr string
	}{
		{
			name: "success",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
		},
		{
			name: "success with practice",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 21,
			},
		},
		{
			name: "invalid params",
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "invalid params: either exam session id or practice id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			result, err := QueryMarkerInfo(ctx, tt.cond)
			z.Sugar().Infof("result: %+v", result.MarkInfos)

			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

//func TestInsertMarkerInfo(t *testing.T) {
//	cleanTestData()
//	initTestData()
//	defer cleanTestData()
//
//	tests := []struct {
//		name           string
//		markInfo       []cmn.TMarkInfo
//		expectedErrStr string
//	}{
//		{
//			name:     "success",
//			markInfo: []cmn.TMarkInfo{
//				{
//					MarkTeacherID: 1101,
//					MarkCount:     1,
//					MarkQuestionGroups: []examPaper.SubjectiveQuestionGroup{
//						{
//							GroupID:     1,
//							QuestionIDs: []int64{1},
//						},
//					},
//					MarkExamineeIds: []int64{1},
//				},
//			},
//		},
//	}
//}
