package mark

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
		forceErr       string
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
			name:       "success with markMode:12（查询单个练习错题集学生）",
			answerType: "02",
			cond: QueryCondition{
				PracticeID:                21,
				TeacherID:                 testedTeacherID,
				PracticeSubmissionID:      2401,
				PracticeWrongSubmissionID: 2601,
			},
			markerInfo: MarkerInfo{
				MarkMode: "14",
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
		{
			name:       "invalid practice_submission_id with markMode:12（查询单个练习学生）",
			answerType: "02",
			cond: QueryCondition{
				PracticeID: 21,
				TeacherID:  testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "12",
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "invalid practice_submission_id",
		},
		{
			name:       "invalid practice_submission_id with markMode:14（查询单个练习错题集学生）",
			answerType: "02",
			cond: QueryCondition{
				PracticeID: 21,
				TeacherID:  testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkMode: "14",
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "invalid practice_submission_id",
		},
		{
			name:       "exec getStudentAnswersByMarkMode SQL error",
			answerType: "02",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			forceErr:       "QueryStudentAnswersByMarkMode-pgxConn.Query",
			expectedErrStr: "exec getStudentAnswersByMarkMode SQL error",
		},
		{
			name:       "unable to scan row data",
			answerType: "02",
			cond: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

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
		forceErr       string
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
		{
			name: "begin transaction error",
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
			forceErr:       "updateStudentAnswerScore-pgxConn.Begin",
			expectedErrStr: "begin transaction error",
		},
		{
			name: "begin transaction error",
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
			forceErr: "tx.Rollback",
		},
		{
			name: "exec updateStudentAnswerScore sql error",
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
			forceErr:       "tx.QueryRow",
			expectedErrStr: "exec updateStudentAnswerScore sql error",
		},
		{
			name: "exec updateStudentAnswerScore sql error",
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
			forceErr:       "tx.Commit",
			expectedErrStr: "commit tx error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

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

func TestUpdateExamSessionOrPracticeSubmissionState(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name                  string
		teacherID             int64
		examSessionIDs        []int64
		practiceSubmissionIDs []int64
		status                string
		forceErr              string
		expectedErrStr        string
	}{
		{
			name:           "success",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{101},
			status:         "10",
			expectedErrStr: "",
		},
		{
			name:                  "success （修改练习提交状态）",
			teacherID:             testedTeacherID,
			practiceSubmissionIDs: []int64{2401},
			status:                "08",
			expectedErrStr:        "",
		},
		{
			name:           "invalid params: exam_session_ids or practice_submission_ids is required",
			teacherID:      testedTeacherID,
			status:         "10",
			expectedErrStr: "invalid params: exam_session_ids or practice_submission_ids is required",
		},
		{
			name:                  "invalid params: exam_session_ids and practice_submission_ids cannot be both specified",
			teacherID:             testedTeacherID,
			examSessionIDs:        []int64{101},
			practiceSubmissionIDs: []int64{2401},
			status:                "10",
			expectedErrStr:        "invalid params: exam_session_ids and practice_submission_ids cannot be both specified",
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
		{
			name:           "exec updateExamSessionState sql error",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{101},
			status:         "10",
			forceErr:       "updateExamSessionOrPracticeSubmissionState-tx.Query",
			expectedErrStr: "exec updateExamSessionState sql error",
		},
		{
			name:           "unable to scan row data",
			teacherID:      testedTeacherID,
			examSessionIDs: []int64{101},
			status:         "10",
			forceErr:       "rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

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

			_, err = updateExamSessionOrPracticeSubmissionState(ctx, &tx, tt.teacherID, tt.examSessionIDs, tt.practiceSubmissionIDs, tt.status)
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
		forceErr       string
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
		{
			name:           "exec update query error",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			forceErr:       "tx.Query",
			expectedErrStr: "exec update query error",
		},
		{
			name:           "unable to scan row data",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			forceErr:       "rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
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
		forceErr       string
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
		{
			name: "exec QueryMarkerInfo SQL error",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryMarkerInfo-pgxConn.Query",
			expectedErrStr: "exec QueryMarkerInfo SQL error",
		},
		{
			name: "unable to scan row data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryMarkerInfo-rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

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

func TestQueryExamList(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name             string
		req              QueryMarkingListReq
		forceErr         string
		expectedList     []Exam
		expectedRowCount int
		expectedErrStr   string
	}{
		{
			name: "success",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
		},
		{
			name: "success with filter:exam name",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:    10,
				Offset:   0,
				ExamName: "exam 1",
			},
		},
		{
			name: "success with filter:exam name",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:    10,
				Offset:   0,
				ExamName: "exam 2",
			},
			expectedRowCount: 0,
		},
		{
			name: "success with filter:start_time && end_time",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:     10,
				Offset:    0,
				StartTime: time.Now().Add(-22 * time.Hour),
				EndTime:   time.Now().Add(-20 * time.Hour),
			},
		},
		{
			name: "invalid query condition(limit==0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  0,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(limit<0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  -3,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(limit>1000)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  1001,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(offset<0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: -3,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "getExamList count SQL error",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryExamList-pgxConn.QueryRow",
			expectedErrStr: "getExamList count SQL error",
		},
		{
			name: "getExamList SQL error",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryExamList-pgxConn.Query",
			expectedErrStr: "getExamList SQL error",
		},
		{
			name: "scan getExamList SQL result error",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryExamList-rows.Scan",
			expectedErrStr: "scan getExamList SQL result error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, rowCount, err := QueryExamList(ctx, tt.req)
			z.Sugar().Infof("-->(%d)list: %+v", rowCount, list)

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

func TestQueryPracticeList(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name             string
		req              QueryMarkingListReq
		forceErr         string
		expectedList     []Practice
		expectedRowCount int
		expectedErrStr   string
	}{
		{
			name: "success",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
		},
		{
			name: "success-admin",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: true,
				},
				Limit:  10,
				Offset: 0,
			},
		},
		{
			name: "success with filter:practice name",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:        10,
				Offset:       0,
				PracticeName: "practice 1",
			},
		},
		{
			name: "success with filter:practice name",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:    10,
				Offset:   0,
				ExamName: "test",
			},
			expectedRowCount: 0,
		},
		{
			name: "invalid query condition(limit==0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  0,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(limit<0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  -3,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(limit>1000)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  1001,
				Offset: 0,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid query condition(offset<0)",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: -3,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "QueryPracticeList row count SQL error",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryPracticeList-pgxConn.QueryRow",
			expectedErrStr: "QueryPracticeList row count SQL error",
		},
		{
			name: "QueryPracticeList SQL error",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryPracticeList-pgxConn.Query",
			expectedErrStr: "QueryPracticeList SQL error",
		},
		{
			name: "扫描解析练习数据失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "QueryPracticeList-rows.Scan",
			expectedErrStr: "扫描解析练习数据失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, rowCount, err := QueryPracticeList(ctx, tt.req)
			z.Sugar().Infof("-->(%d)list: %+v", rowCount, list)

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

func TestQueryExamineeInfo(t *testing.T) {
	cleanTestData()
	initTestData()
	//defer cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
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
			name: "success with filter:examinee_id",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
				ExamineeID:    2201,
			},
		},
		{
			name: "invalid query condition",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "exec query examinee info SQL error",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryExamineeInfo-pgxConn.Query",
			expectedErrStr: "exec query examinee info SQL error",
		},
		{
			name: "unable to scan row data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryExamineeInfo-rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, err := QueryExamineeInfo(ctx, tt.cond)
			z.Sugar().Infof("-->(%d)list: %+v", len(list), list)

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

func TestQueryMarkingResults(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
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
			name: "success with filter:examinee_id",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
		},
		{
			name: "success-practice",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 22,
			},
		},
		{
			name: "success-practice with filter: practice_submission_id",
			cond: QueryCondition{
				TeacherID:            testedTeacherID,
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
		},
		{
			name: "invalid params: 请求参数必须包含教师ID",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			expectedErrStr: "invalid params",
		},
		{
			name: "invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个",
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个",
		},
		{
			name: "invalid params: 请求参数不能同时包含练习ID和考试场次ID",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
				PracticeID:    21,
			},
			expectedErrStr: "invalid params: 请求参数不能同时包含练习ID和考试场次ID",
		},
		{
			name: "exec getMarkingResults SQL error",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryMarkingResults-pgxConn.Query",
			expectedErrStr: "exec getMarkingResults SQL error",
		},
		{
			name: "unable to scan row data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			forceErr:       "QueryMarkingResults-rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, err := QueryMarkingResults(ctx, tt.cond)

			data, e := json.Marshal(list)
			if e != nil {
				panic(e)
				return
			}

			z.Sugar().Infof("-->(%d)list: %+v", len(list), string(data))

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

func TestQueryQuestionsByMarkMode(t *testing.T) {
	cleanTestData()
	initTestData()
	//defer cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		markerInfo     MarkerInfo
		forceErr       string
		expectedErrStr string
	}{
		{
			name: "success",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
		},
		{
			name: "success-practice",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 22,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
		},
		{
			name: "success with markMode:06",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "06",
				MarkMethod: "00",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405]`),
					},
				},
			},
		},
		{
			name: "invalid query condition",
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name: "invalid params: 请求参数必须包含练习ID和考试ID中的一个",
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			expectedErrStr: "invalid params: 请求参数必须包含练习ID和考试ID中的一个",
		},
		{
			name: "invalid params: 请求参数不能同时包含练习ID和考试ID",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				PracticeID:    22,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			expectedErrStr: "invalid params: 请求参数不能同时包含练习ID和考试ID",
		},
		{
			name: "invalid mark mode",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID: testedTeacherID,
			},
			expectedErrStr: "invalid mark mode",
		},
		{
			name: "invalid mark mode",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "07",
				MarkMethod: "00",
			},
			expectedErrStr: "invalid mark mode",
		},
		{
			name: "invalid markInfos",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "06",
				MarkMethod: "00",
			},
			expectedErrStr: "invalid markInfos",
		},
		{
			name: "unable to unmarshal question ids",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "06",
				MarkMethod: "00",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405`),
					},
				},
			},
			expectedErrStr: "unable to unmarshal question ids",
		},
		{
			name: "exec getQuestionsQuery SQL error",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			forceErr:       "QueryQuestionsByMarkMode-pgxConn.Query",
			expectedErrStr: "exec getQuestionsQuery SQL error",
		},
		{
			name: "unable to scan row data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			forceErr:       "QueryQuestionsByMarkMode-rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
		{
			name: "unable to convert raw standard answer data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 103,
			},
			markerInfo: MarkerInfo{
				MarkerID:   testedTeacherID,
				MarkMode:   "10",
				MarkMethod: "00",
			},
			expectedErrStr: "unable to convert raw standard answer data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, err := QueryQuestionsByMarkMode(ctx, tt.cond, tt.markerInfo)
			data, e := json.Marshal(list)
			if e != nil {
				t.Errorf("unable to marshal list: %v", err.Error())
			}
			z.Sugar().Infof("-->(%d)list: %+v", len(list), string(data))

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

func TestQueryStudentsInfo(t *testing.T) {
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
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
			name: "success with filter:examinee_id",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
		},
		{
			name: "success-practice",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 22,
			},
		},
		{
			name: "success-practice with filter: practice_submission_id",
			cond: QueryCondition{
				TeacherID:            testedTeacherID,
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
		},
		{
			name: "invalid params: 请求参数必须包含教师ID",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			expectedErrStr: "invalid params",
		},
		{
			name: "invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个",
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个",
		},
		{
			name: "invalid params: 请求参数不能同时包含练习ID和考试场次ID",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
				PracticeID:    21,
			},
			expectedErrStr: "invalid params: 请求参数不能同时包含练习ID和考试场次ID",
		},
		{
			name: "exec getMarkingResults SQL error",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryStudentInfo-pgxConn.Query",
			expectedErrStr: "exec query student info SQL error",
		},
		{
			name: "unable to scan row data",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryStudentsInfo-rows.Scan",
			expectedErrStr: "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			list, err := QueryStudentsInfo(ctx, tt.cond)

			data, e := json.Marshal(list)
			if e != nil {
				panic(e)
				return
			}

			z.Sugar().Infof("-->(%d)list: %+v", len(list), string(data))

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

func TestUpdatePracticeWrongSubmissionsState(t *testing.T) {
	cleanTestData()
	initTestData()
	//defer cleanTestData()

	tests := []struct {
		name                       string
		teacherID                  int64
		practiceWrongSubmissionIDs []int64
		status                     string
		forceErr                   string
		expectedErrStr             string
	}{
		{
			name:                       "success",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{2601},
			status:                     "10",
			expectedErrStr:             "",
		},
		{
			name:           "invalid params: practice_wrong_submission_ids is required",
			teacherID:      testedTeacherID,
			status:         "06",
			expectedErrStr: "invalid params: practice_wrong_submission_ids is required",
		},
		{
			name:                       "invalid params: status is required",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{2601},
			status:                     "",
			expectedErrStr:             "invalid params: status is required",
		},
		{
			name:                       "invalid params: teacher_id is required",
			teacherID:                  -3,
			practiceWrongSubmissionIDs: []int64{2601},
			status:                     "06",
			expectedErrStr:             "invalid params: teacher_id is required",
		},
		{
			name:                       "exec updateExamSessionState sql error",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{2601},
			status:                     "06",
			forceErr:                   "updatePracticeWrongSubmissionState-tx.Query",
			expectedErrStr:             "exec updatePracticeWrongSubmissionState sql error",
		},
		{
			name:                       "unable to scan row data",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{2601},
			status:                     "06",
			forceErr:                   "updatePracticeWrongSubmissionState-rows.Scan",
			expectedErrStr:             "unable to scan row data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			pgxConn := cmn.GetPgxConn()
			var tx pgx.Tx
			var err error
			tx, err = pgxConn.Begin(ctx)
			if err != nil {
				err = fmt.Errorf("begin transaction error: %v", err)
				panic(err)
			}

			defer func() {
				if err != nil {
					err_ := tx.Rollback(ctx)
					if err_ != nil {
						z.Error(err_.Error())
						return
					}
				} else {
					err_ := tx.Commit(ctx)
					if err_ != nil {
						z.Error(err_.Error())
						return
					}
				}
			}()

			_, err = updatePracticeWrongSubmissionState(ctx, tx, tt.teacherID, tt.practiceWrongSubmissionIDs, tt.status)
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
