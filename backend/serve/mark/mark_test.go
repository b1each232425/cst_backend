package mark

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
	"w2w.io/null"
	"w2w.io/serve/examPaper"

	"w2w.io/cmn"
)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

const testedTeacherID = 1101

type mockResponseWriter struct {
	headers http.Header
	body    *bytes.Buffer
	status  int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}
func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}
func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func newMockServiceCtx(method string, params map[string]string, bodyBytes []byte) *cmn.ServiceCtx {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	r := &http.Request{
		Method: method,
		URL:    &url.URL{RawQuery: form.Encode()},
		Body:   io.NopCloser(bytes.NewReader(bodyBytes)),
	}

	w := &mockResponseWriter{headers: http.Header{}, body: &bytes.Buffer{}}
	return &cmn.ServiceCtx{
		W: w,
		R: r,
		Msg: &cmn.ReplyProto{
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.IntFrom(testedTeacherID), // 请求用户ID
		},
	}
}

func newReqProtoData(v interface{}) []byte {
	if v == nil {
		return nil
	}

	jsonData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	req := &cmn.ReqProto{
		Data: jsonData,
	}

	bytes, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	return bytes
}

// RepeatStringSlice 返回一个包含 `count` 个 `value` 的字符串切片
func RepeatStringSlice(value string, count int) []string {
	slice := make([]string, count)
	for i := 0; i < count; i++ {
		slice[i] = value
	}
	return slice
}

func initTestData() {
	var queries []string
	var args [][][]interface{}
	var tempArgs [][]interface{}

	queries = append(queries, `INSERT INTO t_user(id, account, category) VALUES($1, $2, $3)`)
	args = append(args, [][]interface{}{
		{1101, "test account 1", "test user"},
		{1102, "test account 2", "test user"},
		{1103, "test account 3", "test user"},
		{1201, "test account 4", "test student"},
		{1202, "test account 5", "test student"},
		{1203, "test account 6", "test student"},
	})

	queries = append(queries, `INSERT INTO t_exam_info(id, name, type, creator, create_time, status) VALUES($1, $2, $3, $4, $5, $6)`)
	tempArgs = [][]interface{}{
		{11, "test exam 1", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{12, "test exam 2", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{13, "test exam 3", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
	}
	args = append(args, tempArgs)

	insertExamSessionSQL := `INSERT INTO t_exam_session(id, exam_id, paper_id, mark_method, mark_mode, start_time, end_time, session_num, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	queries = append(queries, insertExamSessionSQL)
	now := time.Now()
	tempArgs = [][]interface{}{
		{101, 11, 101, "00", "10", now.Add(-25 * time.Hour).UnixMilli(), now.Add(-24 * time.Hour).UnixMilli(), 1, "00", 1101},
		{102, 11, 102, "00", "10", now.Add(-23 * time.Hour).UnixMilli(), now.Add(-22 * time.Hour).UnixMilli(), 2, "00", 1101},
		{103, 11, 103, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper(id, exam_session_id, name, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{101, 101, "test exam paper 1", "00", 1101},
		{102, 102, "test exam paper 2", "00", 1101},
		{103, 103, "test exam paper 3", "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper_group(id, exam_paper_id, name, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{301, 101, "test group 1", "00", 1101},
		{302, 102, "test group 2", "00", 1101},
		{303, 103, "test group 3", "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper_question(id, score, type, content, answers, group_id, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`)
	tempArgs = [][]interface{}{
		{401, 3, "00", "单选1", `["C"]`, 301, "00", 1101},
		{402, 3, "02", "多选1", `["B", "D"]`, 301, "00", 1101},
		{403, 3, "04", "判断1", `["A"]`, 301, "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_examinee(id, student_id, exam_session_id, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{2201, 1201, 101, "10", 1101},
		{2202, 1202, 101, "10", 1101},
		{2203, 1203, 101, "10", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_student_answers(id, examinee_id, question_id, answer, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7)`)
	tempArgs = [][]interface{}{
		{21, 2201, 401, `{"answer": ["C"]}`, `["C"]`, "00", 1101},
		{22, 2201, 402, `{"answer": ["B", "D"]}`, `["B", "D"]`, "00", 1101},
		{23, 2201, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{24, 2202, 401, `{"answer": ["B"]}`, `["C"]`, "00", 1101},
		{25, 2202, 402, `{"answer": ["B"]}`, `["B", "D"]`, "00", 1101},
		{26, 2202, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
	}
	args = append(args, tempArgs)

	insertMarkInfoSQL := `INSERT INTO t_mark_info(id, exam_session_id, mark_teacher_id, creator, status) VALUES($1, $2, $3, $4, $5)`
	queries = append(queries, insertMarkInfoSQL)
	tempArgs = [][]interface{}{
		{201, 101, 1101, 1101, "00"},
		{202, 102, 1101, 1101, "00"},
		{203, 103, 1101, 1101, "00"},
		{204, 108, 1101, 1101, "00"}, // 测试软删除接口
	}
	args = append(args, tempArgs)

	if len(queries) != len(args) {
		panic("queries and args length not equal")
		return
	}

	var tx *sql.Tx
	var err error
	sqlxDB := cmn.GetDbConn()
	tx, err = sqlxDB.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		panic(err)
		return
	}

	defer func() {
		if err != nil {
			err = tx.Rollback()
			panic(err)
		}
	}()

	for i, query := range queries {
		for _, arg := range args[i] {
			_, err = tx.Exec(query, arg...)
			if err != nil {
				err = fmt.Errorf("insert test data SQL error: %s", err.Error())
				panic(err)
				return
			}
		}

		//_, err = tx.Exec(query, args[i]...)
		//if err != nil {
		//	err = fmt.Errorf("insert test data SQL error: %s", err.Error())
		//	panic(err)
		//	return
		//}

	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		panic(err)
	}

}

func cleanTestData() {
	var queries []string
	var args [][]interface{}

	queries = append(queries, `DELETE FROM t_student_answers WHERE id = $1`)
	args = append(args, []interface{}{21, 22, 23, 24, 25, 26})

	queries = append(queries, `DELETE FROM t_examinee WHERE id = $1`)
	args = append(args, []interface{}{2201, 2202, 2203})

	queries = append(queries, `DELETE FROM t_exam_paper_question WHERE id = $1`)
	args = append(args, []interface{}{401, 402, 403})

	queries = append(queries, `DELETE FROM t_exam_paper_group WHERE id = $1`)
	args = append(args, []interface{}{301, 302, 303})

	queries = append(queries, `DELETE FROM t_user WHERE id = $1`)
	args = append(args, []interface{}{1101, 1102, 1103, 1201, 1202, 1203})

	queries = append(queries, `DELETE FROM t_mark_info WHERE id = $1`)
	args = append(args, []interface{}{201, 202, 203, 204})

	queries = append(queries, `DELETE FROM t_exam_paper WHERE id = $1`)
	args = append(args, []interface{}{101, 102, 103})

	queries = append(queries, `DELETE FROM t_exam_session WHERE id = $1`)
	args = append(args, []interface{}{101, 102, 103})

	queries = append(queries, `DELETE FROM t_exam_info WHERE id = $1`)
	args = append(args, []interface{}{11, 12, 13})

	pgxConn := cmn.GetPgxConn()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		panic(err)
		return
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			panic(err)
		}
	}()

	if len(queries) != len(args) {
		err = fmt.Errorf("queries and args length not equal")
		panic(err)
		return
	}

	for i, query := range queries {
		for _, arg := range args[i] {
			_, err = tx.Exec(ctx, query, arg)
			if err != nil {
				err = fmt.Errorf("cleanTestData exec SQL (%d) error: %v", i, err)
				panic(err)
				return
			}
		}

		//_, err = tx.Exec(ctx, query, args[i])
		//if err != nil {
		//	err = fmt.Errorf("cleanTestData exec SQL (%d) error: %v", i, err)
		//	panic(err)
		//	return
		//}
	}

	err = tx.Commit(ctx)
}

func TestCleanTestData(t *testing.T) {
	t.Run("cleanTestData", func(t *testing.T) {
		cleanTestData()
	})
}

func TestGetExamList(t *testing.T) {
	z := cmn.GetLogger()
	cleanTestData()
	z.Info("starting mark Test")
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		expectedErrStr   string
		expectedMsg      *cmn.ReplyProto
		expectedExams    []Exam
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
		},
		{
			name: "success with default params",
			params: map[string]string{
				"page_index": "",
				"page_size":  "",
				"exam_name":  "",
				"status":     "",
			},
		},
		{
			name: "success with time filter",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
				"start_time": fmt.Sprintf("%d", time.Now().Add(-22*time.Hour).UnixMilli()),
				"end_time":   fmt.Sprintf("%d", time.Now().Add(-20*time.Hour).UnixMilli()),
			},
		},
		{
			name: "method not get",
			params: map[string]string{
				"page_index": "2",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
			requestMethod:  "POST",
			expectedErrStr: "please call /api/mark/getExamList with http GET method",
		},
		{
			name: "invalid_params(start_time)",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
				"start_time": "abc",
				"end_time":   fmt.Sprintf("%d", time.Now().Add(-22*time.Hour).UnixMilli()),
			},
			expectedErrStr: "error parsing start time",
		},
		{
			name: "invalid_params(end_time)",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
				"start_time": fmt.Sprintf("%d", time.Now().Add(-22*time.Hour).UnixMilli()),
				"end_time":   "a",
			},
			expectedErrStr: "error parsing end time",
		},
		{
			name: "invalid_params(page_index)",
			params: map[string]string{
				"page_index": "abc",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
			expectedErrStr: "error parsing page index",
		},
		{
			name: "invalid_params(page_index)",
			params: map[string]string{
				"page_index": "-1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
			expectedErrStr: "page index must be greater than 0",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page_index": "2",
				"page_size":  "-2",
				"exam_name":  "",
				"status":     "",
			},
			expectedErrStr: "page size must be between 1 and 1000",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page_index": "2",
				"page_size":  "abc",
				"exam_name":  "",
				"status":     "",
			},
			expectedErrStr: "error parsing page size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.requestMethod != "" {
				method = tt.requestMethod
			} else {
				method = "GET"
			}
			ctx := context.WithValue(context.Background(), cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
			GetExamList(ctx)
			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("GetExamList: %+v", q.Msg)
			if q.Err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", q.Err.Error())
				} else {
					assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}

			if q.Msg.Status != 0 {
				t.Logf("unexpected status: %d", q.Msg.Status)
			}

			//assert.Equal(t, tt.expectedMsg, q.Msg)
		})
	}
}

func TestGetMarkingDetails(t *testing.T) {

	z := cmn.GetLogger()
	z.Info("starting mark Test")
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		expectedErrStr   string
		expectedMsg      *cmn.ReplyProto
		expectedExams    []Exam
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.requestMethod != "" {
				method = tt.requestMethod
			} else {
				method = "GET"
			}
			ctx := context.WithValue(context.Background(), cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
			GetMarkingDetails(ctx)
			q := cmn.GetCtxValue(ctx)
			if q.Err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", q.Err.Error())
				} else {
					assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}

			if q.Msg.Status != 0 {
				t.Logf("unexpected status: %d", q.Msg.Status)
			}
		})
	}

}

func TestHandleMarkerInfo(t *testing.T) {

	z := cmn.GetLogger()
	z.Info("starting mark Test")
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		requestParams  HandleMarkerInfoReq
		expectedErrStr string
	}{
		{
			name:      "success-update mark info state",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID:  101,
				MarkMode:       "00",
				Markers:        []int64{1001},
				ExamSessionIDs: []int64{108},
				Status:         "02",
			},
		},
		{
			name:      "success-markMode:00-自动批改",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "00",
				Markers:       []int64{1001},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:02-全卷多评",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "02",
				Markers:       []int64{101, 102},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:04-试卷分配",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				ExamineeIDs:   []int64{801, 802, 803, 804},
				MarkMode:      "04",
				Markers:       []int64{1001, 1002},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:06-题组专评",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "06",
				Markers:       []int64{1001, 1002},
				//QuestionIDGroups: [][]int64{{1, 2, 3}, {4, 5}, {6, 7}},
				QuestionGroups: []examPaper.SubjectiveQuestionGroup{
					{
						GroupID:     1,
						QuestionIDs: []int64{1, 2, 3},
					},
					{
						GroupID:     2,
						QuestionIDs: []int64{4, 5},
					},
					{
						GroupID:     3,
						QuestionIDs: []int64{6, 7},
					},
				},
				Status: "00",
			},
		},
		{
			name:      "success-markMode:10-单人批改",
			teacherID: 1001,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Markers:       []int64{1001},
				Status:        "00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			pgxConn := cmn.GetPgxConn()
			tx, err := pgxConn.Begin(ctx)
			if err != nil {
				panic(err)
				return
			}

			defer func() {
				if err != nil {
					err_ := tx.Rollback(ctx)
					if err_ != nil {
						panic(err_)
						return
					}
				}
			}()

			err = HandleMarkerInfo(ctx, &tx, tt.teacherID, tt.requestParams)
			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}

			err = tx.Commit(ctx)
			if err != nil {
				panic(err)
				return
			}
		})
	}
}

func TestMarkObjectiveQuestionAnswers(t *testing.T) {

	z := cmn.GetLogger()
	z.Info("starting mark Test")
	cleanTestData()
	initTestData()
	//defer cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		requestParams  QueryCondition
		expectedErrStr string
	}{
		{
			name:      "success",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
		},
		{
			name:      "invalid params(teacher id)",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
			},
			expectedErrStr: "invalid query condition",
		},
		{
			name:      "invalid params(exam_session_id or practice_submission_id < 0)",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "invalid params: exam session id or practice id must be greater than zero",
		},

		{
			name:      "invalid params(both exam_session_id and practice_submission_id > 0)",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
				PracticeID:    10001,
			},
			expectedErrStr: "invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := MarkObjectiveQuestionAnswers(ctx, tt.requestParams)
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

func TestSaveMarkingResults(t *testing.T) {
	z := cmn.GetLogger()
	cleanTestData()
	z.Info("starting mark Test")
	initTestData()
	//defer cleanTestData()

	tests := []struct {
		name              string
		params            map[string]string
		requestMethod     string
		requestData       interface{}
		isInvalidReqProto bool
		bodyStr           string
		expectedErrStr    string
		expectedMsg       *cmn.ReplyProto
		expectedRowCount  int
	}{
		{
			name:          "success-insert",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(103),
					ExamineeID:    null.IntFrom(2203),
					QuestionID:    null.IntFrom(10001),
					Score:         null.FloatFrom(3),
					MarkDetails:   make(types.JSONText, 0),
					Creator:       null.IntFrom(testedTeacherID),
				},
			},
		},
		{
			name:          "success-insert with auto creator",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(103),
					ExamineeID:    null.IntFrom(2203),
					QuestionID:    null.IntFrom(10001),
					Score:         null.FloatFrom(3),
					MarkDetails:   make(types.JSONText, 0),
				},
			},
		},
		{
			name:           "method not post",
			requestMethod:  "GET",
			expectedErrStr: "please call /api/mark/getMarkingDetails with http post method",
		},
		{
			name:           "no requestBody",
			requestMethod:  "POST",
			expectedErrStr: "call /mark/marking-results with empty body",
		},
		{
			name:           "invalid request body(len == 0)",
			requestMethod:  "POST",
			requestData:    []*cmn.TMark{},
			expectedErrStr: "invalid request body",
		},
		{
			name:           "invalid request body data struct",
			requestMethod:  "POST",
			requestData:    "{",
			expectedErrStr: "failed to unmarshal marking results",
		},
		{
			name:              "invalid request body data struct 2",
			requestMethod:     "POST",
			requestData:       "{",
			isInvalidReqProto: true,
			bodyStr:           "{",
			expectedErrStr:    "failed to unmarshal request body",
		},
		{
			name:          "invalid insert(no teacher id)",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					ExamSessionID: null.IntFrom(103),
					ExamineeID:    null.IntFrom(2203),
					QuestionID:    null.IntFrom(10001),
					Score:         null.FloatFrom(3),
					MarkDetails:   make(types.JSONText, 0),
					Creator:       null.IntFrom(testedTeacherID),
				},
			},
			expectedErrStr: "teacherID is required",
		},
		{
			name:          "invalid insert(no question id)",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(103),
					ExamineeID:    null.IntFrom(2203),
					Score:         null.FloatFrom(3),
					MarkDetails:   make(types.JSONText, 0),
					Creator:       null.IntFrom(testedTeacherID),
				},
			},
			expectedErrStr: "questionID is required",
		},
		{
			name:          "invalid insert exam mark(no exam session id)",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					TeacherID:   null.IntFrom(testedTeacherID),
					ExamineeID:  null.IntFrom(2203),
					QuestionID:  null.IntFrom(10001),
					Score:       null.FloatFrom(3),
					MarkDetails: make(types.JSONText, 0),
					Creator:     null.IntFrom(testedTeacherID),
				},
			},
			expectedErrStr: "invalid insert params",
		},
		{
			name:          "invalid insert exam mark(no examinee id)",
			requestMethod: "POST",
			requestData: []*cmn.TMark{
				{
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(103),
					QuestionID:    null.IntFrom(10001),
					Score:         null.FloatFrom(3),
					MarkDetails:   make(types.JSONText, 0),
					Creator:       null.IntFrom(testedTeacherID),
				},
			},
			expectedErrStr: "invalid insert params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctxVal []byte
			if tt.isInvalidReqProto {
				ctxVal = json.RawMessage(tt.bodyStr)
			} else {
				ctxVal = newReqProtoData(tt.requestData)
			}
			ctx := context.WithValue(context.Background(), cmn.QNearKey, newMockServiceCtx(tt.requestMethod, tt.params, ctxVal))
			SaveMarkingResults(ctx)
			q := cmn.GetCtxValue(ctx)
			if q.Err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", q.Err.Error())
				} else {
					assert.Contains(t, q.Msg.Msg, tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}
