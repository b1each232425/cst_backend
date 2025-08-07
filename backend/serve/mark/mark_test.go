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
	"net/http/httptest"
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
	testedIDGroups = make([][]interface{}, 0)
}

const testedTeacherID = 1101

var testedIDGroups [][]interface{}

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

	w := httptest.NewRecorder()
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
		{1204, "test account 7", "test student"},
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
		{101, 11, 101, "02", "00", now.Add(-25 * time.Hour).UnixMilli(), now.Add(-24 * time.Hour).UnixMilli(), 1, "00", 1101}, // 纯理论
		{102, 11, 102, "00", "10", now.Add(-23 * time.Hour).UnixMilli(), now.Add(-22 * time.Hour).UnixMilli(), 2, "00", 1101}, // 理论+主观
		{103, 11, 103, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 题目答案缺失
		{104, 11, 104, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
		{105, 11, 105, "00", "", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 批改模式缺失
		{106, 11, 106, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_practice(id, name, correct_mode, type, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	tempArgs = [][]interface{}{
		{21, "test practice 1", "00", "00", "02", 1101},
		{22, "test practice 2", "02", "00", "02", 1101},
		{23, "test practice 3", "00", "00", "02", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper(id, exam_session_id, practice_id, name, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	tempArgs = [][]interface{}{
		{101, 101, null.NewInt(0, false), "test exam paper 1", "00", 1101},
		{102, 102, null.NewInt(0, false), "test exam paper 2", "00", 1101},
		{103, 103, null.NewInt(0, false), "test exam paper 3", "00", 1101},
		{104, 104, null.NewInt(0, false), "test exam paper 4", "00", 1101},
		{124, null.NewInt(0, false), 21, "test practice paper 4", "00", 1101},
		{125, null.NewInt(0, false), 22, "test practice paper 5", "00", 1101},
		{126, null.NewInt(0, false), 23, "test practice paper 6", "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper_group(id, exam_paper_id, name, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{301, 101, "test group 1", "00", 1101},
		{302, 102, "test group 2", "00", 1101},
		{303, 103, "test group 3", "00", 1101},
		{304, 104, "test group 4", "00", 1101},
		{324, 124, "test group 24", "00", 1101},
		{325, 125, "test group 25", "00", 1101},
		{326, 126, "test group 26", "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_exam_paper_question(id, score, type, content, answers, group_id, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`)
	tempArgs = [][]interface{}{
		{401, 3, "00", "单选1", `["C"]`, 301, "00", 1101},
		{402, 3, "02", "多选1", `["B", "D"]`, 301, "00", 1101},
		{403, 3, "04", "判断1", `["A"]`, 301, "00", 1101},
		{404, 3, "00", "单选1", `["C"]`, 302, "00", 1101},
		{405, 3, "06", "填空1", `[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`, 302, "00", 1101},
		{411, 3, "00", "单选2", `[]`, 303, "00", 1101}, // 异常数据
		{412, 3, "02", "多选2", `["B", "D"]`, 304, "00", 1101},
		{424, 3, "00", "单选1", `["C"]`, 324, "00", 1101}, // 练习
		{425, 3, "02", "多选1", `["B", "D"]`, 324, "00", 1101},
		{426, 3, "04", "判断1", `["A"]`, 324, "00", 1101},
		{421, 3, "00", "单选2", `[]`, 326, "00", 1101}, // 异常数据
		{422, 3, "06", "填空2", `[{"index": 1, "score": 1, "answer": "填空1第一空答案", "grading_rule": "1"},{"index": 2, "score": 2, "answer": "填空2第二空答案", "grading_rule": "2"}]`, 302, "00", 1101},
		{423, 3, "06", "异常答案填空1", `{}`, 303, "00", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_examinee(id, student_id, exam_session_id, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{2201, 1201, 101, "10", 1101},
		{2202, 1202, 101, "10", 1101},
		{2203, 1203, 103, "10", 1101},
		{2204, 1204, 103, "10", 1101},
		{2205, 1201, 102, "10", 1101},
		{2206, 1201, 103, "10", 1101},
		{2207, 1202, 103, "10", 1101},
	}
	args = append(args, tempArgs)

	queries = append(queries, `INSERT INTO t_practice_submissions(id, student_id, practice_id, status, creator) VALUES($1, $2, $3, $4, $5)`)
	tempArgs = [][]interface{}{
		{2401, 1201, 21, "06", 1101},
		{2402, 1202, 21, "06", 1101},
		{2403, 1203, 21, "06", 1101},
	}
	args = append(args, tempArgs)

	// 考试作答
	queries = append(queries, `INSERT INTO t_student_answers(id, examinee_id, question_id, answer, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7)`)
	tempArgs = [][]interface{}{
		{21, 2201, 401, `{"answer": ["C"]}`, `["C"]`, "00", 1101},
		{22, 2201, 402, `{"answer": ["B", "D"]}`, `["B", "D"]`, "00", 1101},
		{23, 2201, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{24, 2202, 401, `{"answer": ["B"]}`, `["C"]`, "00", 1101},
		{25, 2202, 402, `{"answer": ["B"]}`, `["B", "D"]`, "00", 1101},
		{26, 2202, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{27, 2203, 411, `{"answer": ["B"]}`, `[]`, "00", 1101},
		{28, 2205, 404, `{"answer": ["B"]}`, `["B"]`, "00", 1101},
		{29, 2205, 405, `{"answer": ["填空学生作答1"]}`, `[]`, "00", 1101},
		{30, 2204, 411, `{"answer": {}}`, `["B"]`, "00", 1101},
		{31, 2206, 411, `{"answer": ["B"]}`, `{}`, "00", 1101},
		{32, 2207, 411, `{"answer": []}`, `["B"]`, "00", 1101},
	}
	args = append(args, tempArgs)

	// 练习作答
	queries = append(queries, `INSERT INTO t_student_answers(id, practice_submission_id, question_id, answer, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7)`)
	tempArgs = [][]interface{}{
		{41, 2401, 424, `{"answer": ["C"]}`, `["C"]`, "00", 1101},
		{42, 2401, 425, `{"answer": ["B", "D"]}`, `["B", "D"]`, "00", 1101},
		{43, 2401, 426, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{44, 2402, 424, `{"answer": ["B"]}`, `["C"]`, "00", 1101},
		{45, 2402, 425, `{"answer": ["B"]}`, `["B", "D"]`, "00", 1101},
		{46, 2402, 426, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
	}
	args = append(args, tempArgs)

	insertMarkInfoSQL := `INSERT INTO t_mark_info(id, exam_session_id, practice_id, mark_teacher_id, creator, status) VALUES($1, $2, $3, $4, $5, $6)`
	queries = append(queries, insertMarkInfoSQL)
	tempArgs = [][]interface{}{
		{201, 101, null.NewInt(0, false), 1101, 1101, "00"},
		{202, 102, null.NewInt(0, false), 1101, 1101, "00"},
		{203, 103, null.NewInt(0, false), 1101, 1101, "00"},
		{204, 108, null.NewInt(0, false), 1101, 1101, "00"}, // 测试软删除接口
		{205, null.NewInt(0, false), 21, 1101, 1101, "00"},
		{206, 106, null.NewInt(0, false), null.NewInt(0, false), 1101, "00"},
		{207, 105, null.NewInt(0, false), 1101, 1101, "00"},
	}
	args = append(args, tempArgs)

	query := `INSERT INTO t_mark(id, teacher_id, exam_session_id, examinee_id, question_id, mark_details, score, creator, status) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	queries = append(queries, query)
	tempArgs = [][]interface{}{
		{9901, testedTeacherID, 102, 2205, 405, `{}`, 2, testedTeacherID, "00"},
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
		var ids []interface{}
		for _, arg := range args[i] {
			ids = append(ids, arg[0])
			_, err = tx.Exec(query, arg...)
			if err != nil {
				err = fmt.Errorf("insert test data SQL error: %v", err)
				panic(err)
				return
			}
		}
		testedIDGroups = append(testedIDGroups, ids)

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

	queries = append(queries, `DELETE FROM t_mark WHERE id = $1`)
	args = append(args, []interface{}{9901})

	queries = append(queries, `DELETE FROM t_student_answers WHERE id = $1`)
	args = append(args, []interface{}{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 41, 42, 43, 44, 45, 46})

	queries = append(queries, `DELETE FROM t_examinee WHERE id = $1`)
	args = append(args, []interface{}{2201, 2202, 2203, 2204, 2205, 2206, 2207})

	queries = append(queries, `DELETE FROM t_exam_paper_question WHERE id = $1`)
	args = append(args, []interface{}{401, 402, 403, 404, 405, 411, 412, 424, 425, 426, 421, 422, 423})

	queries = append(queries, `DELETE FROM t_exam_paper_group WHERE id = $1`)
	args = append(args, []interface{}{301, 302, 303, 304, 324, 325, 326})

	queries = append(queries, `DELETE FROM t_user WHERE id = $1`)
	args = append(args, []interface{}{1101, 1102, 1103, 1201, 1202, 1203, 1204})

	queries = append(queries, `DELETE FROM t_mark_info WHERE id = $1`)
	args = append(args, []interface{}{201, 202, 203, 204, 205, 206, 207})

	queries = append(queries, `DELETE FROM t_exam_paper WHERE id = $1`)
	args = append(args, []interface{}{101, 102, 103, 104, 105, 106, 124, 125, 126})

	queries = append(queries, `DELETE FROM t_exam_session WHERE id = $1`)
	args = append(args, []interface{}{101, 102, 103, 104, 105, 106})

	queries = append(queries, `DELETE FROM t_practice_submissions WHERE id = $1`)
	args = append(args, []interface{}{2401, 2402, 2403})

	queries = append(queries, `DELETE FROM t_practice WHERE id = $1`)
	args = append(args, []interface{}{21, 22, 23})

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

	}

	err = tx.Commit(ctx)
}

//func cleanTestDataWithID(queries []string, args [][]interface{}) {
//	pgxConn := cmn.GetPgxConn()
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	tx, err := pgxConn.Begin(ctx)
//	if err != nil {
//		panic(err)
//		return
//	}
//
//	defer func() {
//		if err != nil {
//			err = tx.Rollback(ctx)
//			panic(err)
//		}
//	}()
//
//	if len(queries) != len(args) {
//		err = fmt.Errorf("queries and args length not equal")
//		panic(err)
//		return
//	}
//
//	for i, query := range queries {
//		for _, arg := range args[i] {
//			_, err = tx.Exec(ctx, query, arg)
//			if err != nil {
//				err = fmt.Errorf("cleanTestData exec SQL (%d) error: %v, query: %v, arg: %v", i, err, query, arg)
//
//				panic(err)
//				return
//			}
//		}
//
//	}
//
//	err = tx.Commit(ctx)
//}

//func TestMarkObjectiveQuestionAnswersBusiness(t *testing.T) {
//	t.Run("MarkObjectiveQuestionAnswersBusiness", func(t *testing.T) {
//		var ctx context.Context
//		var err error
//		ctx = context.Background()
//
//		//pgxConn := cmn.GetPgxConn()
//		//tx, err := pgxConn.Begin(ctx)
//		//if err != nil {
//		//	panic(err)
//		//	return
//		//}
//		//
//		//defer func() {
//		//	if err != nil {
//		//		err_ := tx.Rollback(ctx)
//		//		if err_ != nil {
//		//			panic(err_)
//		//			return
//		//		}
//		//	}
//		//}()
//		//
//		//err = HandleMarkerInfo(ctx, &tx, 1574, HandleMarkerInfoReq{
//		//	MarkMode:      "00",
//		//	Markers:       []int64{1574},
//		//	ExamSessionID: 152,
//		//	Status:        "00",
//		//})
//		//if err != nil {
//		//	t.Errorf("HandleMarkerInfo error: %v", err)
//		//}
//		//
//		//err = tx.Commit(ctx)
//		//if err != nil {
//		//	panic(err)
//		//	return
//		//}
//
//		ctx = context.Background()
//		err = MarkObjectiveQuestionAnswers(ctx, QueryCondition{
//			ExamSessionID: 152,
//			TeacherID:     1574,
//		})
//		if err != nil {
//			t.Errorf("MarkObjectiveQuestionAnswers error: %v", err)
//		}
//	})
//}

func TestCleanTestData(t *testing.T) {
	t.Run("cleanTestData", func(t *testing.T) {
		cleanTestData()
		initTestData()
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
		forceErr         string
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
		{
			name: "unable to marshal response data",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
			forceErr:       "HandleExamList-json.Marshal",
			expectedErrStr: "unable to marshal response data",
		},
		{
			name: "getExamList count SQL error",
			params: map[string]string{
				"page_index": "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
			},
			forceErr:       "QueryExamList-pgxConn.QueryRow",
			expectedErrStr: "getExamList count SQL error",
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

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
			HandleExamList(ctx)
			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("HandleExamList: %+v", q.Msg)
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
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
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
		{
			name: "method post",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			requestMethod:  "POST",
			expectedErrStr: "please call /api/mark/getMarkingDetails with http GET method",
		},
		{
			name: "exam_session_id is required",
			params: map[string]string{
				"examinee_id": "",
			},
			expectedErrStr: "exam_session_id is required",
		},
		{
			name: "error parsing exam_session_id",
			params: map[string]string{
				"exam_session_id": "abc",
				"examinee_id":     "",
			},
			expectedErrStr: "error parsing exam_session_id",
		},
		{
			name: "error parsing examinee_id",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "abc",
			},
			expectedErrStr: "error parsing examinee_id",
		},
		{
			name: "no marker info",
			params: map[string]string{
				"exam_session_id": "104",
				"examinee_id":     "",
			},
			expectedErrStr: "no marker info",
		},
		{
			name: "failed to marshal response data",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "HandleMarkingDetails-json.Marshal",
			expectedErrStr: "failed to marshal response data",
		},
		{
			name: "QueryExamineeInfo-error",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "QueryExamineeInfo-pgxConn.Query",
			expectedErrStr: "exec query examinee info SQL error",
		},
		{
			name: "QueryMarkingResults-error",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "QueryMarkingResults-pgxConn.Query",
			expectedErrStr: "exec getMarkingResults SQL error",
		},
		{
			name: "QueryMarkerInfo-error",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "QueryMarkerInfo-pgxConn.Query",
			expectedErrStr: "QueryMarkerInfo",
		},
		{
			name: "QueryExamQuestionsByMarkMode-error",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "QueryExamQuestionsByMarkMode-pgxConn.Query",
			expectedErrStr: "exec getQuestionsQuery SQL error",
		},
		{
			name: "QueryStudentAnswersByMarkMode-error",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "QueryStudentAnswersByMarkMode-pgxConn.Query",
			expectedErrStr: "exec getStudentAnswersByMarkMode SQL error",
		},
		{
			name: "failed to marshal response data",
			params: map[string]string{
				"exam_session_id": "101",
				"examinee_id":     "",
			},
			forceErr:       "HandleMarkingDetails-json.Marshal",
			expectedErrStr: "failed to marshal response data",
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

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
			HandleMarkingDetails(ctx)
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
		forceErr       string
		expectedErrStr string
	}{
		{
			name:      "success-update mark info state",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionIDs: []int64{108},
				Status:         "02",
			},
		},
		{
			name:      "success-update mark info state（练习）",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				PracticeIDs: []int64{901},
				Status:      "02",
			},
		},
		{
			name:      "no examSessionIDs to update mark info state",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				Markers: []int64{testedTeacherID},
				Status:  "02",
			},
			expectedErrStr: "no examSessionIDs to update mark info state",
		},
		{
			name:      "success-markMode:00-自动批改（练习）",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				PracticeID: 901,
				MarkMode:   "00",
				Markers:    []int64{testedTeacherID},
				Status:     "00",
			},
		},
		{
			name:      "success-markMode:10-单人批改（练习）",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				PracticeID: 902,
				MarkMode:   "10",
				Markers:    []int64{testedTeacherID},
				Status:     "00",
			},
		},

		{
			name:      "success-markMode:00-自动批改",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "00",
				Markers:       []int64{testedTeacherID},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:02-全卷多评",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "02",
				Markers:       []int64{101, 102},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:04-试卷分配",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				ExamineeIDs:   []int64{801, 802, 803, 804},
				MarkMode:      "04",
				Markers:       []int64{testedTeacherID, 1002},
				Status:        "00",
			},
		},
		{
			name:      "success-markMode:06-题组专评",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "06",
				Markers:       []int64{testedTeacherID, 1002},
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
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Markers:       []int64{testedTeacherID},
				Status:        "00",
			},
		},
		{
			name:      "invalid teacher id",
			teacherID: 0,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Markers:       []int64{testedTeacherID},
				Status:        "00",
			},
			expectedErrStr: "invalid teacherID",
		},
		{
			name:      "invalid teacher id",
			teacherID: -3,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Markers:       []int64{testedTeacherID},
				Status:        "00",
			},
			expectedErrStr: "invalid teacherID",
		},
		{
			name:      "invalid exam_session_id && practice_id",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				MarkMode: "10",
				Markers:  []int64{testedTeacherID},
				Status:   "00",
			},
			expectedErrStr: "invalid exam_session_id && practice_id",
		},
		{
			name:      "invalid status",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Markers:       []int64{testedTeacherID},
				Status:        "",
			},
			expectedErrStr: "invalid status",
		},
		{
			name:      "markMode:02 markers is empty",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "02",
				Status:        "00",
			},
			expectedErrStr: "markers is empty",
		},
		{
			name:      "markMode:04 examinee_ids is empty",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "04",
				Markers:       []int64{testedTeacherID, 1002},
				Status:        "00",
			},
			expectedErrStr: "examinee_ids is empty",
		},
		{
			name:      "markMode:06 invalid question groups",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "06",
				Markers:       []int64{testedTeacherID, 1002},
				Status:        "00",
			},
			expectedErrStr: "invalid question groups",
		},
		{
			name:      "markMode:10 invalid markers",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "10",
				Status:        "00",
			},
			expectedErrStr: "invalid markers",
		},
		{
			name:      "invalid mark mode",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				Status:        "00",
			},
			expectedErrStr: "no markInfos to insert",
		},
		{
			name:      "failed to marshal MarkExamineeIds",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				ExamineeIDs:   []int64{801, 802, 803, 804},
				MarkMode:      "04",
				Markers:       []int64{testedTeacherID, 1002},
				Status:        "00",
			},
			forceErr:       "json.Marshal-1",
			expectedErrStr: "failed to marshal MarkExamineeIds",
		},
		{
			name:      "failed to marshal markQuestionIDs",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "06",
				Markers:       []int64{testedTeacherID, 1002},
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
			forceErr:       "json.Marshal-2",
			expectedErrStr: "failed to marshal markQuestionIDs",
		},
		{
			name:      "exec insert query error",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionID: 101,
				MarkMode:      "00",
				Markers:       []int64{testedTeacherID},
				Status:        "00",
			},
			forceErr:       "HandleMarkerInfo-tx.QueryRow",
			expectedErrStr: "exec insert query error",
		},
		{
			name:      "exec update query error",
			teacherID: testedTeacherID,
			requestParams: HandleMarkerInfoReq{
				ExamSessionIDs: []int64{108},
				Status:         "02",
			},
			forceErr:       "tx.Query",
			expectedErrStr: "exec update query error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			pgxConn := cmn.GetPgxConn()
			tx, err := pgxConn.Begin(ctx)
			if err != nil {
				panic(err)
				return
			}

			defer func() {
				if err != nil {
					_ = tx.Rollback(ctx)
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
			}
		})
	}
}

func TestMarkObjectiveQuestionAnswers(t *testing.T) {
	z := cmn.GetLogger()
	z.Info("starting mark Test")
	cleanTestData()
	initTestData()
	defer cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		requestParams  QueryCondition
		forceErr       string
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
			name:      "success (批改单个考试学生)",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2201,
				TeacherID:     testedTeacherID,
			},
		},
		{
			name:      "success-practice (批改单个练习学生)",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				PracticeID:           21,
				PracticeSubmissionID: 2401,
				TeacherID:            testedTeacherID,
			},
		},
		{
			name:      "success-（学生作答结果字段为空数组，得分为0）",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 103,
				ExamineeID:    2207,
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
		{
			name:      "invalid actual answers",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 103,
				ExamineeID:    2203,
				TeacherID:     testedTeacherID,
			},
			expectedErrStr: "invalid actual answers",
		},
		{
			name:      "failed to unmarshal actual answers",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 103,
				ExamineeID:    2206,
				TeacherID:     testedTeacherID,
			},
			expectedErrStr: "failed to unmarshal actual answers",
		},
		{
			name:      "failed to unmarshal answer",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 103,
				ExamineeID:    2204,
				TeacherID:     testedTeacherID,
			},
			expectedErrStr: "failed to unmarshal answer",
		},
		{
			name:      "success-dont need to auto submit",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 102,
				TeacherID:     testedTeacherID,
			},
		},
		{
			name:      "failed to marshal mark details",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "MarkObjectiveQuestionAnswers-json.Marshal",
			expectedErrStr: "failed to marshal mark details",
		},
		{
			name:      "failed to query subjective question counts",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "MarkObjectiveQuestionAnswers-pgxConn.QueryRow",
			expectedErrStr: "failed to query subjective question counts",
		},
		{
			name:      "begin transaction error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "MarkObjectiveQuestionAnswers-pgxConn.Begin",
			expectedErrStr: "begin transaction error",
		},
		{
			name:      "tx.Rollback error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr: "MarkObjectiveQuestionAnswers-tx.Rollback",
		},
		{
			name:      "commit tx error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "MarkObjectiveQuestionAnswers-tx.Commit",
			expectedErrStr: "commit tx error",
		},
		{
			name:      "exec getStudentAnswersByMarkMode SQL error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "QueryStudentAnswersByMarkMode-pgxConn.Query",
			expectedErrStr: "exec getStudentAnswersByMarkMode SQL error",
		},
		{
			name:      "begin transaction error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "updateStudentAnswerScore-pgxConn.Begin",
			expectedErrStr: "begin transaction error",
		},
		{
			name:      "exec updateExamSessionState sql error",
			teacherID: testedTeacherID,
			requestParams: QueryCondition{
				ExamSessionID: 101,
				TeacherID:     testedTeacherID,
			},
			forceErr:       "updateExamSessionOrPracticeSubmissionState-tx.Query",
			expectedErrStr: "exec updateExamSessionState sql error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
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
	defer cleanTestData()

	tests := []struct {
		name              string
		params            map[string]string
		requestMethod     string
		requestData       interface{}
		isInvalidReqProto bool
		bodyStr           string
		forceErr          string
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
			expectedErrStr: "please call /api/mark/marking-results with http post method",
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
		{
			name:          "failed to read request body",
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
			forceErr:       "io.ReadAll",
			expectedErrStr: "failed to read request body",
		},
		{
			name:          "failed to close request body",
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
			forceErr: "io.Close",
			//expectedErrStr: "failed to read request body",
		},
		{
			name:          "begin transaction error",
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
			forceErr:       "pgxConn.Begin",
			expectedErrStr: "begin transaction error",
		},
		{
			name:          "rollback tx error",
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
			forceErr: "tx.Rollback",
			//expectedErrStr: "rollback tx error",
		},
		{
			name:          "exec upsertMark query error",
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
			forceErr:       "tx.QueryRow",
			expectedErrStr: "exec upsertMark query error",
		},
		{
			name:          "commit transaction error",
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
			forceErr:       "tx.Commit",
			expectedErrStr: "commit transaction error",
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

			ctx := context.Background()

			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(tt.requestMethod, tt.params, ctxVal))
			HandleMarkingResults(ctx)
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

func TestAutoMark(t *testing.T) {
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
				ExamSessionID: 101,
				ExamineeID:    2201,
			},
		},
		{
			name: "success-practice",
			cond: QueryCondition{
				PracticeID:           21,
				PracticeSubmissionID: 2401,
			},
		},
		{
			name: "success - 不需要自动批改",
			cond: QueryCondition{
				ExamSessionID: 102,
			},
		},
		{
			name:           "invalid params(exam_session_id or practice_submission_id < 0)",
			expectedErrStr: "invalid params: exam session id or practice id must be greater than zero",
		},
		{
			name: "invalid params(both exam_session_id and practice_submission_id > 0)",
			cond: QueryCondition{
				ExamSessionID:        101,
				PracticeID:           21,
				PracticeSubmissionID: 2401,
				ExamineeID:           2201,
			},
			expectedErrStr: "invalid params: exam session id && practice id cannot be both greater than zero",
		},
		{
			name: "no marker info found",
			cond: QueryCondition{
				ExamSessionID: 104,
			},
			expectedErrStr: "no marker info found",
		},
		{
			name: "invalid mark mode in marker info",
			cond: QueryCondition{
				ExamSessionID: 105,
			},
			expectedErrStr: "invalid mark mode in marker info",
		},
		{
			name: "exec QueryMarkerInfo SQL error",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2201,
			},
			forceErr:       "QueryMarkerInfo-pgxConn.Query",
			expectedErrStr: "exec QueryMarkerInfo SQL error",
		},
		{
			name: "failed to query subjective question counts",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2201,
			},
			forceErr:       "MarkObjectiveQuestionAnswers-pgxConn.QueryRow",
			expectedErrStr: "failed to query subjective question counts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
			err := AutoMark(ctx, tt.cond)
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
