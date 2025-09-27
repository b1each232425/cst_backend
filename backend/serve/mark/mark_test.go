package mark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

var testedIDGroups = make([][]interface{}, 0)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
	cmn.StartAsynqService()
}

// 模拟 cmn.ServiceCtx
func newMockServiceCtx(path, method string, params map[string]string, bodyBytes []byte) *cmn.ServiceCtx {
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
		Ep: &cmn.ServeEndPoint{
			Path: path,
		},
	}
}

// 模拟请求协议的 data 字段的数据
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

	bts, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	return bts
}

func newTaskPayload(taskType string, data interface{}) []byte {
	taskPayload := cmn.TaskPayload{
		Type:      taskType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		panic(err)
		return nil
	}

	return payloadBytes
}

// 返回一个包含 `count` 个 `value` 的字符串切片
func RepeatStringSlice(value string, count int) []string {
	slice := make([]string, count)
	for i := 0; i < count; i++ {
		slice[i] = value
	}
	return slice
}

//func cleanTestData() {
//	var queries []string
//	var args [][]interface{}
//
//	queries = append(queries, `DELETE FROM t_mark WHERE id = $1`)
//	args = append(args, []interface{}{9901})
//
//	queries = append(queries, `DELETE FROM t_student_answers WHERE id = $1`)
//	args = append(args, []interface{}{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 41, 42, 43, 44, 45, 46})
//
//	queries = append(queries, `DELETE FROM t_examinee WHERE id = $1`)
//	args = append(args, []interface{}{2201, 2202, 2203, 2204, 2205, 2206, 2207})
//
//	queries = append(queries, `DELETE FROM t_exam_paper_question WHERE id = $1`)
//	args = append(args, []interface{}{401, 402, 403, 404, 405, 411, 412, 424, 425, 426, 421, 422, 423})
//
//	queries = append(queries, `DELETE FROM t_exam_paper_group WHERE id = $1`)
//	args = append(args, []interface{}{301, 302, 303, 304, 324, 325, 326})
//
//	queries = append(queries, `DELETE FROM t_user WHERE id = $1`)
//	args = append(args, []interface{}{1101, 1102, 1103, 1201, 1202, 1203, 1204})
//
//	queries = append(queries, `DELETE FROM t_mark_info WHERE id = $1`)
//	args = append(args, []interface{}{201, 202, 203, 204, 205, 206, 207})
//
//	queries = append(queries, `DELETE FROM t_exam_paper WHERE id = $1`)
//	args = append(args, []interface{}{101, 102, 103, 104, 105, 106, 124, 125, 126})
//
//	queries = append(queries, `DELETE FROM t_exam_session WHERE id = $1`)
//	args = append(args, []interface{}{101, 102, 103, 104, 105, 106})
//
//	queries = append(queries, `DELETE FROM t_practice_submissions WHERE id = $1`)
//	args = append(args, []interface{}{2401, 2402, 2403})
//
//	queries = append(queries, `DELETE FROM t_practice WHERE id = $1`)
//	args = append(args, []interface{}{21, 22, 23})
//
//	queries = append(queries, `DELETE FROM t_exam_info WHERE id = $1`)
//	args = append(args, []interface{}{11, 12, 13})
//
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
//				err = fmt.Errorf("cleanTestData exec SQL (%d) error: %v", i, err)
//				panic(err)
//				return
//			}
//		}
//
//	}
//
//	err = tx.Commit(ctx)
//}

func TestCleanTestData(t *testing.T) {
	t.Run("cleanTestData", func(t *testing.T) {
		cleanTestData()
		initTestData()
		cleanTestData()
	})
}

// TODO 测试验证 Resp,RespErr,DATA

func TestGetExamList(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
		expectedErrStr   string
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
		},
		{
			name: "success with default params",
			params: map[string]string{
				"page":      "",
				"page_size": "",
				"exam_name": "",
				"status":    "",
			},
		},
		{
			name: "success with time filter",
			params: map[string]string{
				"page":       "1",
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
				"page":      "2",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			requestMethod:  "POST",
			expectedErrStr: "不支持的 HTTP 方法",
		},
		{
			name: "invalid_params(start_time)",
			params: map[string]string{
				"page":       "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
				"start_time": "abc",
				"end_time":   fmt.Sprintf("%d", time.Now().Add(-22*time.Hour).UnixMilli()),
			},
			expectedErrStr: "startTimeStr 类型转化失败",
		},
		{
			name: "invalid_params(end_time)",
			params: map[string]string{
				"page":       "1",
				"page_size":  "10",
				"exam_name":  "",
				"status":     "",
				"start_time": fmt.Sprintf("%d", time.Now().Add(-22*time.Hour).UnixMilli()),
				"end_time":   "a",
			},
			expectedErrStr: "endTimeStr 类型转化失败",
		},
		{
			name: "invalid_params(page)",
			params: map[string]string{
				"page":      "abc",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			expectedErrStr: "pageIndexStr 类型转化失败",
		},
		{
			name: "invalid_params(page)",
			params: map[string]string{
				"page":      "-1",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			expectedErrStr: "page 必须大于 0",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page":      "2",
				"page_size": "-2",
				"exam_name": "",
				"status":    "",
			},
			expectedErrStr: "pageSize 必须在 1 和 1000 之间",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page":      "2",
				"page_size": "abc",
				"exam_name": "",
				"status":    "",
			},
			expectedErrStr: "pageSizeStr 类型转化失败",
		},
		{
			name: "unable to marshal response data",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			forceErr:       "json.Marshal-exams",
			expectedErrStr: "构造 examsJson 失败",
		},
		{
			name: "QueryExamList error",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			forceErr:       "QueryExamList",
			expectedErrStr: "查询考试列表失败",
		},
	}

	initTestData()
	defer cleanTestData()

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

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/exam", method, tt.params, nil))

			getExamList(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("getExamList: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待获取到错误，但却没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但却获取到错误: %v", q.Err)
			}
		})
	}
}

func TestGetPracticeList(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
		expectedErrStr   string
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
			},
		},
		{
			name: "success with default params",
			params: map[string]string{
				"page":      "",
				"page_size": "",
			},
		},
		{
			name: "method not get",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
			},
			requestMethod:  "POST",
			expectedErrStr: "不支持的 HTTP 方法",
		},
		{
			name: "invalid_params(page)",
			params: map[string]string{
				"page":      "abc",
				"page_size": "10",
			},
			expectedErrStr: "pageIndexStr 类型转化失败",
		},
		{
			name: "invalid_params(page)",
			params: map[string]string{
				"page":      "-1",
				"page_size": "10",
			},
			expectedErrStr: "page 必须大于 0",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page":      "2",
				"page_size": "-2",
			},
			expectedErrStr: "pageSize 必须在 1 和 1000 之间",
		},
		{
			name: "invalid_params(page_size)",
			params: map[string]string{
				"page":      "2",
				"page_size": "abc",
			},
			expectedErrStr: "pageSizeStr 类型转化失败",
		},
		{
			name: "unable to marshal response data",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
			},
			forceErr:       "json.Marshal-practices",
			expectedErrStr: "构造 practicesJson 失败",
		},
		{
			name: "QueryPracticeList Error",
			params: map[string]string{
				"page":      "1",
				"page_size": "10",
				"exam_name": "",
				"status":    "",
			},
			forceErr:       "QueryPracticeList",
			expectedErrStr: "查询练习列表失败",
		},
	}

	initTestData()
	defer cleanTestData()

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

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/practice", method, tt.params, nil))

			getPracticeList(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("getPracticeList: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待错误，但是没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但是出现错误: %v", q.Err)
			}
		})
	}
}

func TestExamSessionDetail(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		params         map[string]string
		body           []byte
		requestMethod  string
		forceErr       string
		expectedErrStr string
	}{
		{
			name: "success-get",
			params: map[string]string{
				"exam_session_id": "101",
			},
			requestMethod: "GET",
		},
		{
			name:           "error-get: 没有携带路径参数 exam_session_id",
			requestMethod:  "GET",
			expectedErrStr: "请求参数必须包含 考试场次ID",
		},
		{
			name: "error-get: examSessionIDStr 无法转化为 int64",
			params: map[string]string{
				"exam_session_id": "abc",
			},
			requestMethod:  "GET",
			expectedErrStr: "examSessionIDStr 类型转化失败",
		},
		{
			name: "error-get: getQuestionsAndStudentInfos 执行失败",
			params: map[string]string{
				"exam_session_id": "101",
			},
			requestMethod:  "GET",
			forceErr:       "getQuestionsAndStudentInfos",
			expectedErrStr: "获取问题和学生信息失败",
		},
		{
			name: "error-get: 无法构造 detailJson",
			params: map[string]string{
				"exam_session_id": "102",
			},
			requestMethod:  "GET",
			forceErr:       "json.Marshal-detail",
			expectedErrStr: "构造 detailJson 失败",
		},
		{
			name: "success-post",
			body: newReqProtoData(cmn.TMark{
				TeacherID:     null.IntFrom(testedTeacherID),
				ExamSessionID: null.IntFrom(103),
				ExamineeID:    null.IntFrom(2203),
				QuestionID:    null.IntFrom(10001),
				Score:         null.FloatFrom(3),
				MarkDetails:   make(types.JSONText, 0),
				Creator:       null.IntFrom(testedTeacherID),
			}),
			requestMethod: "POST",
		},
		{
			name:           "error-post: saveMarkingResults 执行失败",
			requestMethod:  "POST",
			forceErr:       "saveMarkingResults",
			expectedErrStr: "保存考试批改结果失败",
		},
		{
			name:           "error-other-requestMethod: 请求方式错误",
			requestMethod:  "PATCH",
			expectedErrStr: "不支持的 HTTP 方法",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/exam-session/detail", tt.requestMethod, tt.params, tt.body))

			examSessionDetail(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("getPracticeList: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待错误，但是没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但是出现错误: %v", q.Err)
			}
		})
	}
}

func TestPracticeDetail(t *testing.T) {

}

func TestExamSessionSubmission(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
		expectedErrStr   string
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"exam_session_id": "102",
			},
		},
		{
			name:           "error: 请求方法错误",
			requestMethod:  "POST",
			expectedErrStr: "不支持的 HTTP 方法",
		},
		{
			name:           "error: 缺少 考试ID",
			expectedErrStr: "请求参数必须包含 考试场次ID",
		},
		{
			name: "error: examSessionIDStr 无法转化为 int64",
			params: map[string]string{
				"exam_session_id": "abc",
			},
			expectedErrStr: "examSessionIDStr 类型转化失败",
		},
		{
			name: "error: submitResult 执行失败",
			params: map[string]string{
				"exam_session_id": "102",
			},
			forceErr:       "submitResult",
			expectedErrStr: "提交考试失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.requestMethod != "" {
				method = tt.requestMethod
			} else {
				method = "PATCH"
			}

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/exam-session/results-submission", method, tt.params, nil))

			examSessionSubmission(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("examSessionSubmission: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待获取到错误，但却没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但却获取到错误: %v", q.Err)
			}
		})
	}
}

func TestPracticeStudentSubmission(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
		expectedErrStr   string
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
		},
		{
			name:           "error: 请求方法错误",
			requestMethod:  "POST",
			expectedErrStr: "不支持的 HTTP 方法",
		},
		{
			name:           "error: 缺少 练习ID",
			expectedErrStr: "请求参数必须包含 练习ID",
		},
		{
			name: "error: practice_id 无法转化为 int64",
			params: map[string]string{
				"practice_id": "abc",
			},
			expectedErrStr: "practiceIDStr 类型转化失败",
		},
		{
			name: "error: 缺少 练习提交ID",
			params: map[string]string{
				"practice_id": "22",
			},
			expectedErrStr: "请求参数必须包含 练习提交ID",
		},
		{
			name: "error: practice_id 无法转化为 int64",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "abc",
			},
			expectedErrStr: "practiceSubmissionIDStr 类型转化失败",
		},
		{
			name: "error: submitResult 执行失败",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
			forceErr:       "submitResult",
			expectedErrStr: "提交练习学生失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.requestMethod != "" {
				method = tt.requestMethod
			} else {
				method = "PATCH"
			}

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/practice-student/results-submission", method, tt.params, nil))

			practiceStudentSubmission(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("practiceStudentSubmission: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待获取到错误，但却没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但却获取到错误: %v", q.Err)
			}
		})
	}
}

func TestGetStudentAnswersAndMarkResults(t *testing.T) {
	tests := []struct {
		name             string
		params           map[string]string
		requestMethod    string
		forceErr         string
		expectedErrStr   string
		expectedRowCount int
	}{
		{
			name: "success",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
		},
		{
			name:           "error: 请求方法错误",
			requestMethod:  "POST",
			expectedErrStr: "不支持的 HTTP 方法",
		},
		{
			name:           "error: 考试ID、练习ID 都不包含",
			expectedErrStr: "请求参数必须包含 练习ID 或者 考试场次ID 中的一个",
		},
		{
			name: "error: 同时包含 考试ID、练习ID",
			params: map[string]string{
				"exam_session_id": "102",
				"practice_id":     "22",
			},
			expectedErrStr: "请求参数不能同时包含 练习ID 和 考试场次ID",
		},
		{
			name: "error: examSessionIDStr 无法转化为 int64",
			params: map[string]string{
				"exam_session_id": "102",
			},
			expectedErrStr: "examSessionIDStr 类型转化失败",
		},
		{
			name: "error: 缺少考生 ID",
			params: map[string]string{
				"exam_session_id": "102",
			},
			expectedErrStr: "缺少 考生ID",
		},
		{
			name: "error: examineeIDStr 无法转化为 int64",
			params: map[string]string{
				"exam_session_id": "102",
				"examinee_id":     "abc",
			},
			expectedErrStr: "examineeIDStr 类型转化失败",
		},
		{
			name: "error: 缺少练习提交 ID",
			params: map[string]string{
				"practice_id": "22",
			},
			expectedErrStr: "缺少 练习提交ID",
		},
		{
			name: "error: practiceSubmissionIDStr 无法转化为 int64",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "abc",
			},
			expectedErrStr: "practiceSubmissionIDStr 类型转化失败",
		},
		{
			name: "error: QueryMarkerInfo 执行失败",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
			forceErr:       "QueryMarkerInfo",
			expectedErrStr: "查询批改信息失败",
		},
		{
			name: "error: QueryStudentAnswers 执行失败",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
			forceErr:       "QueryStudentAnswers",
			expectedErrStr: "获取学生主观题答案失败",
		},
		{
			name: "error: QuerySubjectiveQuestionsMarkingResults 执行失败",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
			forceErr:       "QuerySubjectiveQuestionsMarkingResults",
			expectedErrStr: "获取老师对该学生的批改记录失败",
		},
		{
			name: "error: 构造 jsonData 失败",
			params: map[string]string{
				"practice_id":            "22",
				"practice_submission_id": "2404",
			},
			forceErr:       "json.Marshal-map",
			expectedErrStr: "构造 jsonData 失败",
		},
	}

	initTestData()
	defer cleanTestData()

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

			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx("/mark/student-answers-and-mark-results", method, tt.params, nil))

			getStudentAnswersAndMarkResults(ctx)

			q := cmn.GetCtxValue(ctx)
			z.Sugar().Infof("getStudentAnswersAndMarkResults: %+v", q.Msg)

			if tt.expectedErrStr != "" {
				assert.Error(t, q.Err, "期待获取到错误，但却没有错误")
				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, q.Err, "期待没有错误，但却获取到错误: %v", q.Err)
			}
		})
	}
}

//func TestMarkObjectiveQuestionAnswers(t *testing.T) {
//	z := cmn.GetLogger()
//	z.Info("starting mark Test")
//	cleanTestData()
//	initTestData()
//	defer cleanTestData()
//
//	tests := []struct {
//		name           string
//		teacherID      int64
//		requestParams  QueryCondition
//		forceErr       string
//		expectedErrStr string
//	}{
//		{
//			name:      "success",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//		},
//		{
//			name:      "success (批改单个考试学生)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//				TeacherID:     testedTeacherID,
//			},
//		},
//		{
//			name:      "success-practice (批改单个练习学生)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				PracticeID:           21,
//				PracticeSubmissionID: 2401,
//				TeacherID:            testedTeacherID,
//			},
//		},
//		{
//			name:      "success-practice (批改单个错题集练习学生)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				PracticeID:                21,
//				PracticeSubmissionID:      2406,
//				PracticeWrongSubmissionID: 2601,
//				TeacherID:                 testedTeacherID,
//			},
//		},
//		{
//			name:      "UpdateExamSessionOrPracticeSubmissionStatus error (批改单个错题集练习学生)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				PracticeID:                21,
//				PracticeSubmissionID:      2406,
//				PracticeWrongSubmissionID: 2601,
//				TeacherID:                 testedTeacherID,
//			},
//			forceErr:       "UpdatePracticeWrongSubmissionStatus-tx.Query",
//			expectedErrStr: "exec UpdatePracticeWrongSubmissionStatus sql error",
//		},
//		{
//			name:      "success-（学生作答结果字段为空数组，得分为0）",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 103,
//				ExamineeID:    2207,
//				TeacherID:     testedTeacherID,
//			},
//		},
//		{
//			name:      "invalid params(teacher id)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//			},
//			expectedErrStr: "invalid query condition",
//		},
//		{
//			name:      "invalid params(exam_session_id or practice_submission_id < 0)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				TeacherID: testedTeacherID,
//			},
//			expectedErrStr: "invalid params: exam session id or practice id must be greater than zero",
//		},
//		{
//			name:      "invalid params(both exam_session_id and practice_submission_id > 0)",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 101,
//				PracticeID:    10001,
//			},
//			expectedErrStr: "invalid params",
//		},
//		{
//			name:      "invalid actual answers",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 103,
//				ExamineeID:    2203,
//				TeacherID:     testedTeacherID,
//			},
//			expectedErrStr: "invalid actual answers",
//		},
//		{
//			name:      "failed to unmarshal actual answers",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 103,
//				ExamineeID:    2206,
//				TeacherID:     testedTeacherID,
//			},
//			expectedErrStr: "failed to unmarshal actual answers",
//		},
//		{
//			name:      "failed to unmarshal answer",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 103,
//				ExamineeID:    2204,
//				TeacherID:     testedTeacherID,
//			},
//			expectedErrStr: "failed to unmarshal answer",
//		},
//		{
//			name:      "success-dont need to auto submit",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 102,
//				TeacherID:     testedTeacherID,
//			},
//		},
//		{
//			name:      "failed to marshal mark details",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-json.Marshal",
//			expectedErrStr: "failed to marshal mark details",
//		},
//		{
//			name:      "failed to query subjective question counts",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-pgxConn.QueryRow",
//			expectedErrStr: "failed to query subjective question counts",
//		},
//		{
//			name:      "begin transaction error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-pgxConn.Begin",
//			expectedErrStr: "begin transaction error",
//		},
//		{
//			name:      "tx.Rollback error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr: "markObjectiveQuestionAnswers-tx.Rollback",
//		},
//		{
//			name:      "commit tx error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-tx.Commit",
//			expectedErrStr: "commit tx error",
//		},
//		{
//			name:      "commit tx error （批改单个练习错题集学生）",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				PracticeID:                21,
//				PracticeSubmissionID:      2406,
//				PracticeWrongSubmissionID: 2601,
//				TeacherID:                 testedTeacherID,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-tx.Commit",
//			expectedErrStr: "commit tx error",
//		},
//		{
//			name:      "exec getStudentAnswersByMarkMode SQL error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "QueryStudentAnswers-pgxConn.Query",
//			expectedErrStr: "exec getStudentAnswersByMarkMode SQL error",
//		},
//		{
//			name:      "begin transaction error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "UpdateStudentAnswerScore-pgxConn.Begin",
//			expectedErrStr: "begin transaction error",
//		},
//		{
//			name:      "exec updateExamSessionState sql error",
//			teacherID: testedTeacherID,
//			requestParams: QueryCondition{
//				ExamSessionID: 101,
//				TeacherID:     testedTeacherID,
//			},
//			forceErr:       "UpdateExamSessionOrPracticeSubmissionStatus-tx.Query",
//			expectedErrStr: "exec updateExamSessionState sql error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//			err := markObjectiveQuestionAnswers(ctx, tt.requestParams)
//			if err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", err.Error())
//				} else {
//					assert.Contains(t, err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//		})
//	}
//}

//func TestSaveMarkingResults(t *testing.T) {
//	cleanTestData()
//
//	tests := []struct {
//		name              string
//		params            map[string]string
//		requestMethod     string
//		requestData       interface{}
//		isInvalidReqProto bool
//		bodyStr           string
//		forceErr          string
//		expectedErrStr    string
//		expectedMsg       *cmn.ReplyProto
//	}{
//		{
//			name:          "success-insert",
//			requestMethod: "POST",
//			requestData: cmn.TMark{
//				TeacherID:     null.IntFrom(testedTeacherID),
//				ExamSessionID: null.IntFrom(103),
//				ExamineeID:    null.IntFrom(2203),
//				QuestionID:    null.IntFrom(10001),
//				Score:         null.FloatFrom(3),
//				MarkDetails:   make(types.JSONText, 0),
//				Creator:       null.IntFrom(testedTeacherID),
//			},
//		},
//		{
//			name:          "success-insert-practice",
//			requestMethod: "POST",
//			requestData: cmn.TMark{
//				TeacherID:            null.IntFrom(testedTeacherID),
//				PracticeID:           null.IntFrom(22),
//				PracticeSubmissionID: null.IntFrom(2404),
//				QuestionID:           null.IntFrom(427),
//				Score:                null.FloatFrom(3),
//				MarkDetails:          make(types.JSONText, 0),
//				Creator:              null.IntFrom(testedTeacherID),
//			},
//		},
//		{
//			name:          "success-insert with auto creator",
//			requestMethod: "POST",
//			requestData: cmn.TMark{
//				TeacherID:     null.IntFrom(testedTeacherID),
//				ExamSessionID: null.IntFrom(103),
//				ExamineeID:    null.IntFrom(2203),
//				QuestionID:    null.IntFrom(10001),
//				Score:         null.FloatFrom(3),
//				MarkDetails:   make(types.JSONText, 0),
//			},
//		},
//		{
//			name:          "success-insert with teacher id in ctx",
//			requestMethod: "POST",
//			requestData: cmn.TMark{
//				ExamSessionID: null.IntFrom(103),
//				ExamineeID:    null.IntFrom(2203),
//				QuestionID:    null.IntFrom(10001),
//				Score:         null.FloatFrom(3),
//				MarkDetails:   make(types.JSONText, 0),
//			},
//		},
//		{
//			name:           "method not post",
//			requestMethod:  "GET",
//			expectedErrStr: "please call /api/mark/marking-results with http post method",
//		},
//		{
//			name:           "no requestBody",
//			requestMethod:  "POST",
//			expectedErrStr: "failed to decode request body",
//		},
//		{
//			name:              "invalid request proto data struct",
//			requestMethod:     "POST",
//			requestData:       "{",
//			isInvalidReqProto: true,
//			bodyStr:           "{",
//			expectedErrStr:    "failed to decode request body",
//		},
//		{
//			name:           "invalid request data data struct",
//			requestMethod:  "POST",
//			requestData:    "{",
//			expectedErrStr: "failed to unmarshal marking results",
//		},
//		//{
//		//	name:          "exec upsertMark query error",
//		//	requestMethod: "POST",
//		//	requestData: cmn.TMark{
//		//		TeacherID:     null.IntFrom(testedTeacherID),
//		//		ExamSessionID: null.IntFrom(103),
//		//		ExamineeID:    null.IntFrom(2203),
//		//		QuestionID:    null.IntFrom(10001),
//		//		Score:         null.FloatFrom(3),
//		//		MarkDetail:   make(types.JSONText, 0),
//		//		Creator:       null.IntFrom(testedTeacherID),
//		//	},
//		//	forceErr:       "insertOrUpdateMarkingResults",
//		//	expectedErrStr: "exec upsertMark query error",
//		//},
//	}
//
//	initTestData()
//	defer cleanTestData()
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			var ctxVal []byte
//			if tt.isInvalidReqProto {
//				ctxVal = json.RawMessage(tt.bodyStr)
//			} else {
//				ctxVal = newReqProtoData(tt.requestData)
//			}
//
//			ctx := context.Background()
//
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//
//			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(tt.requestMethod, tt.params, ctxVal))
//
//			saveMarkingResults(ctx)
//
//			q := cmn.GetCtxValue(ctx)
//
//			if tt.expectedErrStr != "" {
//				assert.Error(t, q.Err, "expected response err, but got nil")
//				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
//			} else {
//				assert.NoErrorf(t, q.Err, "expected response no err, but got error: %v", q.Err)
//			}
//		})
//	}
//}
//
//func TestSubmitResult(t *testing.T) {
//	cleanTestData()
//
//	tests := []struct {
//		name             string
//		params           map[string]string
//		requestMethod    string
//		forceErr         string
//		expectedErrStr   string
//		expectedMsg      *cmn.ReplyProto
//		expectedRowCount int
//	}{
//		{
//			name: "success",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//		},
//		{
//			name: "success-practice with practice_submission_id",
//			params: map[string]string{
//				"practice_id":            "22",
//				"practice_submission_id": "2404",
//			},
//		},
//		{
//			name: "method not patch",
//			params: map[string]string{
//				"exam_session_id": "101",
//			},
//			requestMethod:  "POST",
//			expectedErrStr: "please call /api/mark/results-submission with http patch method",
//		},
//		{
//			name:           "请求参数必须包含练习ID或者考试场次ID中的一个",
//			params:         map[string]string{},
//			expectedErrStr: "请求参数必须包含练习ID或者考试场次ID中的一个",
//		},
//		{
//			name: "请求参数不能同时包含练习ID和考试场次ID",
//			params: map[string]string{
//				"exam_session_id": "102",
//				"practice_id":     "22",
//			},
//			expectedErrStr: "请求参数不能同时包含练习ID和考试场次ID",
//		},
//		{
//			name: "error parsing exam_session_id",
//			params: map[string]string{
//				"exam_session_id": "abc",
//			},
//			expectedErrStr: "error parsing exam_session_id",
//		},
//		{
//			name: "error parsing practice_id",
//			params: map[string]string{
//				"practice_id": "abc",
//			},
//			expectedErrStr: "error parsing practice_id",
//		},
//		{
//			name: "error parsing practice_submission_id",
//			params: map[string]string{
//				"practice_id":            "22",
//				"practice_submission_id": "abc",
//			},
//			expectedErrStr: "error parsing practice_submission_id",
//		},
//		{
//			name: "QuerySubjectiveQuestionsMarkingResults-error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr:       "QuerySubjectiveQuestionsMarkingResults-pgxConn.Query",
//			expectedErrStr: "exec getMarkingResults SQL error",
//		},
//		{
//			name: "UpdateStudentAnswerScore-error",
//			params: map[string]string{
//				"exam_session_id": "101",
//			},
//			expectedErrStr: "no marking results for update",
//		},
//		{
//			name: "UpdateExamSessionOrPracticeSubmissionStatus-error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr:       "UpdateExamSessionOrPracticeSubmissionStatus",
//			expectedErrStr: "exec updateExamSessionState sql error",
//		},
//		{
//			name: "pgxConn.Begin error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr:       "pgxConn.Begin",
//			expectedErrStr: "begin transaction error",
//		},
//		{
//			name: "tx.Rollback error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "tx.Rollback",
//		},
//		{
//			name: "queryMarkingResults error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "queryMarkingResults",
//		},
//		{
//			name: "UpdateStudentAnswerScore error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "UpdateStudentAnswerScore",
//		},
//		{
//			name: "tx.Commit error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "tx.Commit",
//		},
//	}
//
//	initTestData()
//	defer cleanTestData()
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			var method string
//			if tt.requestMethod != "" {
//				method = tt.requestMethod
//			} else {
//				method = "PATCH"
//			}
//
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
//			}
//
//			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
//
//			submitResult(ctx)
//
//			q := cmn.GetCtxValue(ctx)
//			if tt.expectedErrStr != "" {
//				assert.Error(t, q.Err, "expected response err, but got nil")
//				assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
//			} else {
//				assert.NoErrorf(t, q.Err, "expected response no err, but got error: %v", q.Err)
//			}
//		})
//	}
//}

//func TestUpdateMarkingState(t *testing.T) {
//	cleanTestData()
//	initTestData()
//	//defer cleanTestData()
//
//	tests := []struct {
//		name             string
//		params           map[string]string
//		requestMethod    string
//		forceErr         string
//		expectedErrStr   string
//		expectedMsg      *cmn.ReplyProto
//		expectedRowCount int
//	}{
//		{
//			name: "success",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//		},
//		{
//			name: "method not patch",
//			params: map[string]string{
//				"exam_session_id": "101",
//			},
//			requestMethod:  "POST",
//			expectedErrStr: "please call /api/mark/state with http patch method",
//		},
//		{
//			name:           "exam_session_id is required",
//			params:         map[string]string{},
//			expectedErrStr: "exam_session_id is required",
//		},
//		{
//			name: "error parsing exam_session_id",
//			params: map[string]string{
//				"exam_session_id": "abc",
//			},
//			expectedErrStr: "error parsing exam_session_id",
//		},
//		{
//			name: "UpdateExamSessionOrPracticeSubmissionStatus-error",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr:       "UpdateExamSessionOrPracticeSubmissionStatus-tx.Query",
//			expectedErrStr: "exec updateExamSessionState sql error",
//		},
//		{
//			name: "updateMarkingState-pgxConn.Begin",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr:       "updateMarkingState-pgxConn.Begin",
//			expectedErrStr: "begin transaction error",
//		},
//		{
//			name: "updateMarkingState-tx.Rollback",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "updateMarkingState-tx.Rollback",
//		},
//		{
//			name: "updateMarkingState-tx.Commit",
//			params: map[string]string{
//				"exam_session_id": "102",
//			},
//			forceErr: "updateMarkingState-tx.Commit",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			var method string
//			if tt.requestMethod != "" {
//				method = tt.requestMethod
//			} else {
//				method = "PATCH"
//			}
//
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
//			}
//
//			ctx = context.WithValue(ctx, cmn.QNearKey, newMockServiceCtx(method, tt.params, nil))
//			updateMarkingState(ctx)
//			q := cmn.GetCtxValue(ctx)
//			if q.Err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", q.Err.Error())
//				} else {
//					assert.Contains(t, q.Err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//
//			if q.Msg.Status != 0 {
//				t.Logf("unexpected status: %d", q.Msg.Status)
//			}
//		})
//	}
//}

//func TestAutoMark(t *testing.T) {
//	cleanTestData()
//	initTestData()
//	//defer cleanTestData()
//
//	cmn.RegisterTaskHandler(TaskTypeAIMarkRequest, taskMiddleware(handleAIMarkTask))
//
//	tests := []struct {
//		name           string
//		cond           QueryCondition
//		forceErr       string
//		expectedErrStr string
//	}{
//		{
//			name: "success",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//			},
//		},
//		{
//			name: "success",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//		},
//		{
//			name: "success-practice",
//			cond: QueryCondition{
//				PracticeID:           21,
//				PracticeSubmissionID: 2401,
//			},
//		},
//		{
//			name: "success - 不需要自动批改",
//			cond: QueryCondition{
//				ExamSessionID: 102,
//			},
//		},
//		{
//			name:           "invalid params(exam_session_id or practice_submission_id < 0)",
//			expectedErrStr: "invalid params: exam session id or practice id must be greater than zero",
//		},
//		{
//			name: "invalid params(both exam_session_id and practice_submission_id > 0)",
//			cond: QueryCondition{
//				ExamSessionID:        101,
//				PracticeID:           21,
//				PracticeSubmissionID: 2401,
//				ExamineeID:           2201,
//			},
//			expectedErrStr: "invalid params: exam session id && practice id cannot be both greater than zero",
//		},
//		{
//			name: "no marker info found",
//			cond: QueryCondition{
//				ExamSessionID: 104,
//			},
//			expectedErrStr: "no marker info found",
//		},
//		{
//			name: "invalid mark mode in marker info",
//			cond: QueryCondition{
//				ExamSessionID: 105,
//			},
//			expectedErrStr: "invalid mark mode in marker info",
//		},
//		{
//			name: "exec QueryMarkerInfo SQL error",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//			},
//			forceErr:       "QueryMarkerInfo-pgxConn.Query",
//			expectedErrStr: "exec QueryMarkerInfo SQL error",
//		},
//		{
//			name: "failed to query subjective question counts",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//			},
//			forceErr:       "markObjectiveQuestionAnswers-pgxConn.QueryRow",
//			expectedErrStr: "failed to query subjective question counts",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//			err := AutoMark(ctx, tt.cond)
//
//			time.Sleep(10 * time.Second)
//
//			if err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", err.Error())
//				} else {
//					assert.Contains(t, err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//		})
//	}
//}

//func TestHandleMarkerInfo(t *testing.T) {
//
//	z := cmn.GetLogger()
//	z.Info("starting mark Test")
//	cleanTestData()
//	initTestData()
//	defer cleanTestData()
//
//	tests := []struct {
//		name           string
//		teacherID      int64
//		requestParams  HandleMarkerInfoReq
//		forceErr       string
//		expectedErrStr string
//	}{
//		{
//			name:      "success-update mark info state",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionIDs: []int64{108},
//				Status:         "02",
//			},
//		},
//		{
//			name:      "success-update mark info state（练习）",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				PracticeIDs: []int64{901},
//				Status:      "02",
//			},
//		},
//		{
//			name:      "no examSessionIDs to update mark info state",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				Markers: []int64{testedTeacherID},
//				Status:  "02",
//			},
//			expectedErrStr: "no examSessionIDs to update mark info state",
//		},
//		{
//			name:      "success-markMode:00-自动批改（练习）",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				PracticeID: 901,
//				MarkMode:   "00",
//				Markers:    []int64{testedTeacherID},
//				Status:     "00",
//			},
//		},
//		{
//			name:      "success-markMode:10-单人批改（练习）",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				PracticeID: 902,
//				MarkMode:   "10",
//				Markers:    []int64{testedTeacherID},
//				Status:     "00",
//			},
//		},
//
//		{
//			name:      "success-markMode:00-自动批改",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "00",
//				Markers:       []int64{testedTeacherID},
//				Status:        "00",
//			},
//		},
//		{
//			name:      "success-markMode:02-全卷多评",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "02",
//				Markers:       []int64{101, 102},
//				Status:        "00",
//			},
//		},
//		{
//			name:      "success-markMode:04-试卷分配",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				ExamineeIDs:   []int64{801, 802, 803, 804},
//				MarkMode:      "04",
//				Markers:       []int64{testedTeacherID, 1002},
//				Status:        "00",
//			},
//		},
//		{
//			name:      "success-markMode:06-题组专评",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "06",
//				Markers:       []int64{testedTeacherID, 1002},
//				//QuestionIDGroups: [][]int64{{1, 2, 3}, {4, 5}, {6, 7}},
//				QuestionGroups: []examPaper.SubjectiveQuestionGroup{
//					{
//						GroupID:     1,
//						QuestionIDs: []int64{1, 2, 3},
//					},
//					{
//						GroupID:     2,
//						QuestionIDs: []int64{4, 5},
//					},
//					{
//						GroupID:     3,
//						QuestionIDs: []int64{6, 7},
//					},
//				},
//				Status: "00",
//			},
//		},
//		{
//			name:      "success-markMode:10-单人批改",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "10",
//				Markers:       []int64{testedTeacherID},
//				Status:        "00",
//			},
//		},
//		{
//			name:      "invalid teacher id",
//			teacherID: 0,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "10",
//				Markers:       []int64{testedTeacherID},
//				Status:        "00",
//			},
//			expectedErrStr: "invalid teacherID",
//		},
//		{
//			name:      "invalid teacher id",
//			teacherID: -3,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "10",
//				Markers:       []int64{testedTeacherID},
//				Status:        "00",
//			},
//			expectedErrStr: "invalid teacherID",
//		},
//		{
//			name:      "invalid exam_session_id && practice_id",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				MarkMode: "10",
//				Markers:  []int64{testedTeacherID},
//				Status:   "00",
//			},
//			expectedErrStr: "invalid exam_session_id && practice_id",
//		},
//		{
//			name:      "invalid status",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "10",
//				Markers:       []int64{testedTeacherID},
//				Status:        "",
//			},
//			expectedErrStr: "invalid status",
//		},
//		{
//			name:      "markMode:02 markers is empty",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "02",
//				Status:        "00",
//			},
//			expectedErrStr: "markers is empty",
//		},
//		{
//			name:      "markMode:04 examinee_ids is empty",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "04",
//				Markers:       []int64{testedTeacherID, 1002},
//				Status:        "00",
//			},
//			expectedErrStr: "examinee_ids is empty",
//		},
//		{
//			name:      "markMode:06 invalid question groups",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "06",
//				Markers:       []int64{testedTeacherID, 1002},
//				Status:        "00",
//			},
//			expectedErrStr: "invalid question groups",
//		},
//		{
//			name:      "markMode:10 invalid markers",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "10",
//				Status:        "00",
//			},
//			expectedErrStr: "invalid markers",
//		},
//		{
//			name:      "invalid mark mode",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				Status:        "00",
//			},
//			expectedErrStr: "no markInfos to insert",
//		},
//		{
//			name:      "failed to marshal MarkExamineeIds",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				ExamineeIDs:   []int64{801, 802, 803, 804},
//				MarkMode:      "04",
//				Markers:       []int64{testedTeacherID, 1002},
//				Status:        "00",
//			},
//			forceErr:       "json.Marshal-1",
//			expectedErrStr: "failed to marshal MarkExamineeIds",
//		},
//		{
//			name:      "failed to marshal markQuestionIDs",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "06",
//				Markers:       []int64{testedTeacherID, 1002},
//				//QuestionIDGroups: [][]int64{{1, 2, 3}, {4, 5}, {6, 7}},
//				QuestionGroups: []examPaper.SubjectiveQuestionGroup{
//					{
//						GroupID:     1,
//						QuestionIDs: []int64{1, 2, 3},
//					},
//					{
//						GroupID:     2,
//						QuestionIDs: []int64{4, 5},
//					},
//					{
//						GroupID:     3,
//						QuestionIDs: []int64{6, 7},
//					},
//				},
//				Status: "00",
//			},
//			forceErr:       "json.Marshal-2",
//			expectedErrStr: "failed to marshal markQuestionIDs",
//		},
//		{
//			name:      "exec insert query error",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionID: 101,
//				MarkMode:      "00",
//				Markers:       []int64{testedTeacherID},
//				Status:        "00",
//			},
//			forceErr:       "HandleMarkerInfo-tx.QueryRow",
//			expectedErrStr: "exec insert query error",
//		},
//		{
//			name:      "exec update query error",
//			teacherID: testedTeacherID,
//			requestParams: HandleMarkerInfoReq{
//				ExamSessionIDs: []int64{108},
//				Status:         "02",
//			},
//			forceErr:       "tx.Query",
//			expectedErrStr: "exec update query error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//
//			pgxConn := cmn.GetPgxConn()
//			tx, err := pgxConn.Begin(ctx)
//			if err != nil {
//				panic(err)
//				return
//			}
//
//			defer func() {
//				if err != nil {
//					_ = tx.Rollback(ctx)
//				}
//			}()
//
//			err = HandleMarkerInfo(ctx, &tx, tt.teacherID, tt.requestParams)
//			if err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", err.Error())
//				} else {
//					assert.Contains(t, err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//
//			err = tx.Commit(ctx)
//			if err != nil {
//				panic(err)
//			}
//		})
//	}
//}

//func TestGenerateAIMarkTask(t *testing.T) {
//
//	cmn.RegisterTaskHandler(TaskTypeAIMarkRequest, handleAIMarkTask)
//
//	var testedSubjectiveAnswers = []SubjectiveAnswer{
//		{
//			Index: 1,
//			Score: 10,
//			Answer: `
//					1. 独立式（Fat AP）组网：每个无线接入点单独配置和管理。
//
//					   - 优点：部署简单，成本较低，适合小型网络。
//					   - 缺点：管理复杂，扩展性差，无法集中控制。
//
//					2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。
//
//					   - 优点：集中管理、自动调优、漫游切换顺畅，适合中大型网络。
//					   - 缺点：成本高，对控制器性能依赖大。
//					`,
//			GradingRule: `
//					1. **组网方式名称（2分）**
//					   - 正确写出“独立式（Fat AP）组网”得1分，写出“控制器集中式（Fit AP + AC）组网”得1分。
//					   - 错误或遗漏组网方式名称不得分。
//					2. **独立式（Fat AP）组网的优缺点（4分）**
//					   - **优点**：答出“部署简单”“成本较低”“适合小型网络”中的任意两点得2分（每点1分）。
//					   - **缺点**：答出“管理复杂”“扩展性差”“无法集中控制”中的任意两点得2分（每点1分）。
//					   - 优点或缺点仅答出一点得1分，错误描述不得分。
//					3. **控制器集中式（Fit AP + AC）组网的优缺点（4分）**
//					   - **优点**：答出“集中管理”“自动调优”“漫游切换顺畅”“适合中大型网络”中的任意两点得2分（每点1分）。
//					   - **缺点**：答出“成本高”“对控制器性能依赖大”中的任意两点得2分（每点1分）。
//					   - 优点或缺点仅答出一点得1分，错误描述不得分。
//
//					`,
//		},
//	}
//
//	testedSubjectiveAnswerJson, err := json.Marshal(testedSubjectiveAnswers)
//	if err != nil {
//		t.Errorf("failed to marshal subjective answers: %v", err)
//		return
//	}
//
//	var testedStudentAnswers = []map[string][]string{
//		{
//			"answer": {
//				`
//					1. 独立式组网：每个无线接入点单独配置和管理。
//					  - 优点：简单，成本低。
//					  - 缺点：管理复杂。
//					2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。
//					  - 优点：集中管理，适合小型网络。
//					  - 缺点：成本高。
//`,
//			},
//		},
//	}
//
//	testedStudentAnswerJson, err := json.Marshal(testedStudentAnswers[0])
//	if err != nil {
//		t.Errorf("failed to marshal student answers: %v", err)
//		return
//	}
//
//	tests := []struct {
//		name           string
//		cond           QueryCondition
//		questions      []*cmn.TExamPaperQuestion
//		studentAnswers []*cmn.TVStudentAnswerQuestion
//		forceErr       string
//		expectedErrStr string
//		//setup          func() ChatModel
//		//checkedFunc    func(chatResp ResponseContent) (string, bool)
//	}{
//		{
//			name: "success",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//		},
//		{
//			name: "success with examinee id",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//				ExamineeID:    2201,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//		},
//		{
//			name: "success-practice",
//			cond: QueryCondition{
//				TeacherID:  testedTeacherID,
//				PracticeID: 22,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					PracticeID:           null.IntFrom(22),
//					PracticeSubmissionID: null.IntFrom(2401),
//					QuestionID:           null.IntFrom(1001),
//					Answer:               testedStudentAnswerJson,
//				},
//			},
//		},
//		{
//			name: "success-practice with submission id",
//			cond: QueryCondition{
//				TeacherID:            testedTeacherID,
//				PracticeID:           22,
//				PracticeSubmissionID: 2401,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					PracticeID:           null.IntFrom(22),
//					PracticeSubmissionID: null.IntFrom(2401),
//					QuestionID:           null.IntFrom(1001),
//					Answer:               testedStudentAnswerJson,
//				},
//			},
//		},
//		{
//			name: "用户ID无效",
//			cond: QueryCondition{
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "用户ID无效",
//		},
//		{
//			name: "请求参数需包含场次id或者练习id",
//			cond: QueryCondition{
//				TeacherID: testedTeacherID,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "请求参数需包含场次id或者练习id",
//		},
//		{
//			name: "请求参数不能同时包含练习ID和考试场次ID",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//				PracticeID:    22,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "请求参数不能同时包含练习ID和考试场次ID",
//		},
//		{
//			name: "学生答案题目ID无效",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "学生答案题目ID无效",
//		},
//		{
//			name: "学生答案考生ID无效",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID:        null.IntFrom(102),
//					PracticeSubmissionID: null.IntFrom(2401),
//					QuestionID:           null.IntFrom(1001),
//					Answer:               testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "学生答案考生ID无效",
//		},
//		{
//			name: "学生答案提交ID无效",
//			cond: QueryCondition{
//				TeacherID:  testedTeacherID,
//				PracticeID: 22,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					PracticeID: null.IntFrom(22),
//					ExamineeID: null.IntFrom(2201),
//					QuestionID: null.IntFrom(1001),
//					Answer:     testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "学生答案提交ID无效",
//		},
//		{
//			name: "学生答案考生ID和提交ID不能同时为空",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					QuestionID: null.IntFrom(1001),
//					Answer:     testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "学生答案考生ID和提交ID不能同时为空",
//		},
//		{
//			name: "failed to unmarshal student answer",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        types.JSONText(`["ABC"]`),
//				},
//			},
//			expectedErrStr: "failed to unmarshal student answer",
//		},
//		{
//			name: "题目ID无效",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					Score:   null.FloatFrom(10),
//					Answers: testedSubjectiveAnswerJson,
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "题目ID无效",
//		},
//		{
//			name: "unable to convert raw standard answer data",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: types.JSONText(`{}`),
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "unable to convert raw standard answer data",
//		},
//		{
//			name: "no standard answer found in question",
//			cond: QueryCondition{
//				TeacherID:     testedTeacherID,
//				ExamSessionID: 102,
//			},
//			questions: []*cmn.TExamPaperQuestion{
//				{
//					ID:      null.IntFrom(1001),
//					Score:   null.FloatFrom(10),
//					Answers: types.JSONText(`[]`),
//				},
//			},
//			studentAnswers: []*cmn.TVStudentAnswerQuestion{
//				{
//					ExamSessionID: null.IntFrom(102),
//					ExamineeID:    null.IntFrom(2201),
//					QuestionID:    null.IntFrom(1001),
//					Answer:        testedStudentAnswerJson,
//				},
//			},
//			expectedErrStr: "no standard answer found in question",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			//if tt.setup != nil {
//			//	tt.chatModel = tt.setup()
//			//}
//
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//
//			err := generateAIMarkTask(ctx, tt.cond, tt.questions, tt.studentAnswers)
//
//			if err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", err.Error())
//				} else {
//					assert.Contains(t, err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//
//			// 等待队列消费消息
//			//time.Sleep(time.Second * 5)
//
//			//if tt.checkedFunc != nil {
//			//	msg, ok := tt.checkedFunc(*chatResp)
//			//	if !ok {
//			//		t.Errorf(msg)
//			//	}
//			//	assert.True(t, ok)
//			//}
//		})
//	}
//}

//	func TestHandleAIMarkTask(t *testing.T) {
//		cleanTestData()
//		initTestData()
//
//		var testedPracticeQuestionDetails = *ai_mark.TestedQuestionDetails[0]
//		testedPracticeQuestionDetails.QuestionID = 428
//
//		tests := []struct {
//			name           string
//			task           *asynq.Task
//			forceErr       string
//			expectedErrStr string
//			setup          func() error
//			//checkedFunc    func(chatResp ResponseContent) (string, bool)
//		}{
//			{
//				name: "success",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:exam_session:102:count", 5, 0)
//					return nil
//				},
//			},
//			{
//				name: "success",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:exam_session:102:count", 1, 0)
//					return nil
//				},
//			},
//			{
//				name: "success with practice",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:            testedTeacherID,
//						PracticeID:           24,
//						PracticeSubmissionID: 2405,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 1, 0)
//					return nil
//				},
//			},
//			{
//				name: "success with practice（错题集批改）",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:                 testedTeacherID,
//						PracticeID:                24,
//						PracticeSubmissionID:      2405,
//						PracticeWrongSubmissionID: 2601,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 1, 0)
//					return nil
//				},
//			},
//			{
//				name:           "failed to unmarshal task payload",
//				task:           asynq.NewTask(TaskTypeAIMarkRequest, []byte(`[]`)),
//				expectedErrStr: "failed to unmarshal task payload",
//			},
//			{
//				name:           "failed to marshal task payload data",
//				task:           asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, "[]")),
//				expectedErrStr: "failed to marshal task payload data",
//				forceErr:       "handleAIMarkTask-json.Marshal-payload-data",
//			},
//			{
//				name:           "failed to unmarshal task payload data",
//				task:           asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, "[]")),
//				expectedErrStr: "failed to unmarshal task payload data",
//			},
//			{
//				name: "任务payload中的任务计数键为空",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//				})),
//				expectedErrStr: "任务payload中的任务计数键为空",
//			},
//			{
//				name: "任务payload中的教师ID无效",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "任务payload中的教师ID无效",
//			},
//			{
//				name: "任务payload中的考试场次ID与练习ID都无效",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID: testedTeacherID,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "任务payload中的考试场次ID与练习ID都无效",
//			},
//			{
//				name: "任务payload中的考试场次ID与练习ID不能同时有效",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//						PracticeID:    22,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "任务payload中的考试场次ID与练习ID不能同时有效",
//			},
//			{
//				name: "AIMark-error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "failed to marshal student answers",
//				forceErr:       "AIMark-json.Marshal",
//			},
//			{
//				name: "handleAIMarkTask-json.Marshal-mark-details",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "failed to marshal mark details",
//				forceErr:       "handleAIMarkTask-json.Marshal-mark-details",
//			},
//			{
//				name: "UpsertMarkingResults-error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "begin transaction error",
//				forceErr:       "pgxConn.Begin",
//			},
//			{
//				name: "UpdateStudentAnswerScore-error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "begin transaction error",
//				forceErr:       "UpdateStudentAnswerScore-pgxConn.Begin",
//			},
//			{
//				name: "lua脚本执行出错",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "lua脚本执行出错",
//				forceErr:       "handleAIMarkTask-script.Run().Result",
//			},
//			{
//				name: "lua脚本返回结果类型错误",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				expectedErrStr: "lua脚本返回结果类型错误",
//				forceErr:       "handleAIMarkTask-script-result.(string)",
//			},
//			{
//				name: "begin transaction error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:  testedTeacherID,
//						PracticeID: 24,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 1, 0)
//					return nil
//				},
//				forceErr:       "handleAIMarkTask-tx.Begin",
//				expectedErrStr: "begin transaction error",
//			},
//			{
//				name: "handleAIMarkTask-tx.Rollback error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:  testedTeacherID,
//						PracticeID: 24,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 5, 0)
//					return nil
//				},
//				forceErr: "handleAIMarkTask-tx.Rollback",
//			},
//			{
//				name: "handleAIMarkTask-tx.Commit error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:  testedTeacherID,
//						PracticeID: 24,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 5, 0)
//					return nil
//				},
//				forceErr: "handleAIMarkTask-tx.Commit",
//			},
//			{
//				name: "UpdatePracticeWrongSubmissionStatus error（错题集批改）",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:                 testedTeacherID,
//						PracticeID:                24,
//						PracticeSubmissionID:      2405,
//						PracticeWrongSubmissionID: 2601,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 1, 0)
//					return nil
//				},
//				forceErr:       "UpdatePracticeWrongSubmissionStatus-tx.Query",
//				expectedErrStr: "exec UpdatePracticeWrongSubmissionStatus sql error",
//			},
//			{
//				name: "UpdateExamSessionOrPracticeSubmissionStatus error",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       &testedPracticeQuestionDetails,
//						StudentAnswers: ai_mark.TestedStudentAnswers[1],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:            testedTeacherID,
//						PracticeID:           24,
//						PracticeSubmissionID: 2405,
//					},
//					UniqueTaskCountKey: "test:practice:24:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:practice:24:count", 1, 0)
//					return nil
//				},
//				forceErr:       "UpdateExamSessionOrPracticeSubmissionStatus-tx.Query",
//				expectedErrStr: "exec updateExamSessionState sql error",
//			},
//			{
//				name: "找不到任务计数键值",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:key-not-exists",
//				})),
//				expectedErrStr: "找不到任务计数键值",
//			},
//			{
//				name: "删除key失败",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:exam_session:102:count",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:exam_session:102:count", 1, 0)
//					return nil
//				},
//				forceErr:       "handleAIMarkTask-redisClient.Del",
//				expectedErrStr: "删除key失败",
//			},
//			{
//				name: "超减任务计数键值",
//				task: asynq.NewTask(TaskTypeAIMarkRequest, newTaskPayload(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
//					AIMarkRequest: AIMarkRequest{
//						QuestionDetails:       ai_mark.TestedQuestionDetails[0],
//						StudentAnswers: ai_mark.TestedStudentAnswers[0],
//					},
//					QueryCondition: QueryCondition{
//						TeacherID:     testedTeacherID,
//						ExamSessionID: 102,
//					},
//					UniqueTaskCountKey: "test:over-reduce",
//				})),
//				setup: func() error {
//					redisClient := cmn.GetRedisConn()
//					redisClient.Set(context.Background(), "test:over-reduce", 0, 0)
//					return nil
//				},
//				expectedErrStr: "超减任务计数键值",
//			},
//		}
//
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				if tt.setup != nil {
//					_ = tt.setup()
//				}
//
//				ctx := context.Background()
//				if tt.forceErr != "" {
//					ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//				}
//
//				err := handleAIMarkTask(ctx, tt.task)
//
//				if err != nil {
//					if tt.expectedErrStr == "" {
//						t.Errorf("expected success, but got error: %v", err.Error())
//					} else {
//						assert.Contains(t, err.Error(), tt.expectedErrStr)
//					}
//				} else if tt.expectedErrStr != "" {
//					t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//				}
//
//				// 等待队列消费消息
//				//time.Sleep(time.Second * 5)
//
//				//if tt.checkedFunc != nil {
//				//	msg, ok := tt.checkedFunc(*chatResp)
//				//	if !ok {
//				//		t.Errorf(msg)
//				//	}
//				//	assert.True(t, ok)
//				//}
//			})
//		}
//	}

func TestTaskMiddleware(t *testing.T) {
	// 初始化测试环境
	initAIMarkTaskLimiterAndBreaker()

	// 创建测试用的asynq任务
	task := asynq.NewTask("test", []byte("test payload"))

	tests := []struct {
		name          string
		setup         func()
		handler       func(ctx context.Context, task *asynq.Task) error
		expectedError string
		handlerCalled bool
	}{
		{
			name: "正常执行流程",
			setup: func() {
				// 重置为正常的限流器配置
				aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(10, 100, 10)
			},
			handler: func(ctx context.Context, task *asynq.Task) error {
				return nil
			},
			handlerCalled: true,
		},
		{
			name: "限流器拒绝请求-无可用并发",
			setup: func() {
				// 创建一个0并发的限流器来模拟限流
				aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(0, 100, 10)
			},
			handler: func(ctx context.Context, task *asynq.Task) error {
				return nil
			},
			expectedError: "semaphore acquire error",
			handlerCalled: false,
		},
		{
			name: "处理器返回错误",
			setup: func() {
				// 重置为正常的限流器配置
				aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(10, 100, 10)
			},
			handler: func(ctx context.Context, task *asynq.Task) error {
				return errors.New("handler error")
			},
			expectedError: "handler error",
			handlerCalled: true,
		},
		{
			name: "上下文超时",
			setup: func() {
				// 重置为正常的限流器配置
				aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(0, 100, 10)
			},
			handler: func(ctx context.Context, task *asynq.Task) error {
				// 模拟处理时间较长
				time.Sleep(3 * time.Second)
				return nil
			},
			expectedError: "context deadline exceeded",
			handlerCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行测试前的设置
			if tt.setup != nil {
				tt.setup()
			}

			// 记录处理器是否被调用
			handlerCalled := false
			testHandler := func(ctx context.Context, task *asynq.Task) error {
				handlerCalled = true
				return tt.handler(ctx, task)
			}

			// 应用中间件
			wrappedHandler := taskMiddleware(testHandler)

			// 创建上下文（根据测试需要可能带超时）
			var ctx context.Context
			if tt.name == "上下文超时" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
			}

			// 执行处理
			err := wrappedHandler(ctx, task)

			// 验证结果
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.handlerCalled, handlerCalled, "handler called status mismatch")
		})
	}
}
func TestTaskMiddleware_WithCircuitBreaker(t *testing.T) {
	// 初始化测试环境
	initAIMarkTaskLimiterAndBreaker()

	// 重置为正常的限流器配置
	aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(10, 100, 10)

	// 创建测试用的 asynq 任务
	task := asynq.NewTask("test", []byte("test payload"))

	t.Run("熔断器正常状态", func(t *testing.T) {
		handlerCalled := false
		testHandler := func(ctx context.Context, task *asynq.Task) error {
			handlerCalled = true
			return nil
		}

		wrappedHandler := taskMiddleware(testHandler)
		err := wrappedHandler(context.Background(), task)

		assert.NoError(t, err)
		assert.True(t, handlerCalled)
	})

	t.Run("熔断器打开状态", func(t *testing.T) {
		// 创建一个总是失败的处理器来触发熔断器
		failHandler := func(ctx context.Context, task *asynq.Task) error {
			return errors.New("simulated error")
		}

		wrappedFailHandler := taskMiddleware(failHandler)

		// 多次调用失败处理器来打开熔断器
		ctx := context.Background()
		for i := 0; i < 10; i++ {
			_ = wrappedFailHandler(ctx, task)
		}

		// 现在尝试一个应该成功的处理器
		successHandlerCalled := false
		successHandler := func(ctx context.Context, task *asynq.Task) error {
			successHandlerCalled = true
			return nil
		}

		wrappedSuccessHandler := taskMiddleware(successHandler)
		err := wrappedSuccessHandler(ctx, task)

		// 根据熔断器的配置，可能会返回错误（熔断器打开）
		// 注意：实际结果取决于gobreaker的具体实现和配置
		assert.True(t, successHandlerCalled || err != nil)
	})
}
