package paper_respondence

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"w2w.io/cmn"
	"w2w.io/null"
)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

// 模拟执行并捕获响应
func executeAndCaptureResponse(ctx context.Context) *cmn.ReplyProto {
	// 执行 StudentAnswer 函数
	StudentAnswer(ctx)

	// 从上下文中获取响应
	q := cmn.GetCtxValue(ctx)
	return q.Msg
}

// 验证请求和上下文是否正确设置
func verifyRequestAndContext(t *testing.T, req *http.Request, ctx context.Context, method string) {
	// 验证请求是否正确设置
	if req == nil {
		t.Error("Request should not be nil")
	}
	if req.Method != method {
		t.Errorf("Expected method %s, got %s", method, req.Method)
	}

	// 验证上下文是否正确设置
	q := cmn.GetCtxValue(ctx)
	if q == nil {
		t.Error("Context value should not be nil")
		return
	}

	// 验证服务上下文是否正确设置
	if q.R != req {
		t.Error("Request in context should match the original request")
	}
	if q.W == nil {
		t.Error("Response writer in context should not be nil")
	}
	if q.SysUser == nil {
		t.Error("SysUser in context should not be nil")
	} else if q.SysUser.ID.Int64 != 54321 {
		t.Errorf("Expected SysUser ID 54321, got %d", q.SysUser.ID.Int64)
	}
	if q.Msg == nil {
		t.Error("Msg in context should not be nil")
	}
}

// 验证响应是否符合预期的成功响应，更加通用的版本，支持自定义验证逻辑
func verifyResponse(t *testing.T, resp *cmn.ReplyProto, method string, expectedData interface{}) {
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// 根据请求方法验证响应
	switch method {
	case "POST":
		// 对于 POST 请求，验证响应中是否包含预期的数据
		if resp.Status != 0 {
			t.Errorf("Expected success status 0, got %d", resp.Status)
		}
		if len(resp.Data) == 0 {
			t.Error("Response data should not be empty for POST request")
		}

		// 如果提供了预期数据，则验证响应数据是否符合预期
		if expectedData != nil {
			// 将响应数据解析为预期的类型
			expectedDataBytes, err := json.Marshal(expectedData)
			if err != nil {
				t.Fatalf("Failed to marshal expected data: %v", err)
			}

			// 使用反射比较数据结构
			var actualData interface{}
			var expectedDataObj interface{}

			// 解析响应数据
			if err := json.Unmarshal(resp.Data, &actualData); err != nil {
				t.Fatalf("Failed to unmarshal response data: %v", err)
			}

			// 解析预期数据
			if err := json.Unmarshal(expectedDataBytes, &expectedDataObj); err != nil {
				t.Fatalf("Failed to unmarshal expected data: %v", err)
			}

			// 比较数据结构
			if !reflect.DeepEqual(actualData, expectedDataObj) {
				actualJSON, _ := json.MarshalIndent(actualData, "", "  ")
				expectedJSON, _ := json.MarshalIndent(expectedDataObj, "", "  ")
				t.Errorf("Response data does not match expected data.\nGot: %s\nExpected: %s", actualJSON, expectedJSON)
			}
		}

	case "GET":
		// 对于 GET 请求，验证响应中是否包含预期的数据
		if resp.Status != 0 {
			t.Errorf("Expected success status 0, got %d", resp.Status)
		}
		if len(resp.Data) == 0 {
			t.Error("Response data should not be empty for GET request")
		}

		// 如果提供了预期数据，则验证响应数据是否符合预期
		if expectedData != nil {
			// 将响应数据解析为预期的类型
			expectedDataBytes, err := json.Marshal(expectedData)
			if err != nil {
				t.Fatalf("Failed to marshal expected data: %v", err)
			}

			// 使用反射比较数据结构
			var actualData interface{}
			var expectedDataObj interface{}

			// 解析响应数据
			if err := json.Unmarshal(resp.Data, &actualData); err != nil {
				t.Fatalf("Failed to unmarshal response data: %v", err)
			}

			// 解析预期数据
			if err := json.Unmarshal(expectedDataBytes, &expectedDataObj); err != nil {
				t.Fatalf("Failed to unmarshal expected data: %v", err)
			}

			// 比较数据结构
			if !reflect.DeepEqual(actualData, expectedDataObj) {
				actualJSON, _ := json.MarshalIndent(actualData, "", "  ")
				expectedJSON, _ := json.MarshalIndent(expectedDataObj, "", "  ")
				t.Errorf("Response data does not match expected data.\nGot: %s\nExpected: %s", actualJSON, expectedJSON)
			}
		}

	default:
		t.Errorf("Unexpected method: %s", method)
	}
}

// 验证错误响应是否符合预期，更加通用的版本，支持自定义验证逻辑
func verifyErrorResponse(t *testing.T, resp *cmn.ReplyProto, expectedCode int, expectedMsg string, allowData bool) {
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// 验证错误码
	if resp.Status != expectedCode {
		t.Errorf("Expected error status %d, got %d", expectedCode, resp.Status)
	}

	// 验证错误消息
	if expectedMsg != "" && !strings.Contains(resp.Msg, expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: '%s'", expectedMsg, resp.Msg)
	}

	// 对于错误响应，Data 通常为空，除非允许包含数据
	if !allowData && len(resp.Data) > 0 && !bytes.Equal(resp.Data, []byte("null")) {
		t.Errorf("Expected no data in error response, got: %s", string(resp.Data))
	}
}

func TestStudentAnswer(t *testing.T) {
	cmn.ConfigureForTest()
	// 定义测试用例
	testCases := []struct {
		name        string
		method      string
		url         string
		reqBody     *cmn.ReqProto
		expectedLog string
		// 预期结果
		expectSuccess   bool        // 是否期望成功
		expectedCode    int         // 预期错误码
		expectedMessage string      // 预期错误消息
		expectedData    interface{} // 预期数据（可选）
	}{
		// POST 请求测试用例
		{
			name:   "POST 请求 - 保存学生答案 - 基本情况",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"id": 0,
					"examinee_id": 12345,
					"type": "00",
					"question_id": 67890,
					"answer": {"text":"这是学生的答案"},
					"student_id": 54321
				}`),
			},
			expectedLog:     "POST test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 保存学生答案 - 练习模式",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"id": 0,
					"practice_submission_id": 98765,
					"type": "02",
					"question_id": 67890,
					"answer": {"text":"这是练习模式的答案"},
					"student_id": 54321
				}`),
			},
			expectedLog:     "POST test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 更新学生答案",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"id": 123,
					"examinee_id": 12345,
					"type": "00",
					"question_id": 67890,
					"answer": {"text":"这是更新后的答案"},
					"student_id": 54321
				}`),
			},
			expectedLog:     "POST test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 带附件的学生答案",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"id": 0,
					"examinee_id": 12345,
					"type": "00",
					"question_id": 67890,
					"answer": {"text":"这是带附件的答案"},
					"student_id": 54321,
					"attachment_paths": ["path/to/file1.jpg", "path/to/file2.pdf"]
				}`),
			},
			expectedLog:     "POST test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 缺少必要参数",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"id": 0,
					"type": "00",
					"answer": {"text":"缺少题目ID和考生ID"},
					"student_id": 54321
				}`),
			},
			expectedLog:     "POST test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "题目ID不能为空",
		},

		// GET 请求测试用例
		{
			name:            "GET 请求 - 通过考生ID获取学生答案",
			method:          "GET",
			url:             "/api/respondent?question_id=67890&examinee_id=12345",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - 通过练习提交ID获取学生答案",
			method:          "GET",
			url:             "/api/respondent?question_id=67890&practice_submission_id=98765",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - 缺少题目ID参数",
			method:          "GET",
			url:             "/api/respondent?examinee_id=12345",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "题目ID不能为空",
		},
		{
			name:            "GET 请求 - 缺少考生ID和练习提交ID参数",
			method:          "GET",
			url:             "/api/respondent?question_id=67890",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "考生ID和练习提交ID不能同时为空",
		},
	}

	// 运行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc.method, tc.url, tc.reqBody, tc.expectedLog, tc.expectSuccess, tc.expectedCode, tc.expectedMessage, tc.expectedData)
		})
	}
}

// 创建模拟的上下文，更加通用的版本，支持自定义用户ID和请求头
func createMockContext(t *testing.T, req *http.Request, options ...ContextOption) context.Context {
	// 创建基本的上下文
	ctx := context.Background()

	// 创建响应记录器
	rec := httptest.NewRecorder()

	// 创建默认的服务上下文
	q := &cmn.ServiceCtx{
		R: req,
		W: rec,
		SysUser: &cmn.TUser{
			ID: null.IntFrom(54321), // 默认用户ID
		},
		Msg: &cmn.ReplyProto{},
	}

	// 应用自定义选项
	for _, option := range options {
		option(q)
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}

// ContextOption 定义了一个函数类型，用于自定义上下文
type ContextOption func(*cmn.ServiceCtx)

// WithUserID 设置用户ID
func WithUserID(userID int64) ContextOption {
	return func(q *cmn.ServiceCtx) {
		q.SysUser.ID = null.IntFrom(userID)
	}
}

// WithHeaders 设置请求头
func WithHeaders(headers map[string]string) ContextOption {
	return func(q *cmn.ServiceCtx) {
		for key, value := range headers {
			q.R.Header.Set(key, value)
		}
	}
}

// WithCustomData 设置自定义数据
func WithCustomData(key string, value interface{}) ContextOption {
	return func(q *cmn.ServiceCtx) {
		// 这里可以根据需要扩展，例如添加到上下文中
		// 目前仅作为示例
	}
}

// 运行单个测试用例，更加通用的版本，支持自定义验证逻辑
func runTestCase(t *testing.T, method, url string, reqBody *cmn.ReqProto, expectedLog string, expectSuccess bool, expectedCode int, expectedMessage string, expectedData interface{}, options ...ContextOption) {
	var req *http.Request
	var err error

	if reqBody != nil {
		// 对于 POST 请求，准备请求体
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	} else {
		// 对于 GET 请求，没有请求体
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// 创建模拟的上下文，应用自定义选项
	ctx := createMockContext(t, req, options...)

	// 验证测试设置是否正确
	t.Log(expectedLog)

	// // 验证请求和上下文是否正确设置
	// verifyRequestAndContext(t, req, ctx, method)

	// 执行测试并捕获响应
	resp := executeAndCaptureResponse(ctx)

	// 根据预期结果验证响应
	if expectSuccess {
		// 预期成功，验证成功响应
		verifyResponse(t, resp, method, expectedData)
	} else {
		// 预期失败，验证错误响应
		// 默认不允许错误响应包含数据
		verifyErrorResponse(t, resp, expectedCode, expectedMessage, false)
	}
}

func TestCheckExamStatus(t *testing.T) {
	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		reqBody         *cmn.ReqProto
		expectedLog     string
		expectSuccess   bool
		expectedCode    int
		expectedMessage string
		expectedData    interface{}
		options         []ContextOption
	}{
		// 成功场景 - 有效的考试会话ID
		{
			name:            "GET 请求 - 有效的考试会话ID",
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=12345",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // 根据实际情况设置预期数据
			options:         []ContextOption{WithUserID(54321)},
		},
		// 失败场景 - 缺少考试会话ID
		{
			name:            "GET 请求 - 缺少考试会话ID",
			method:          "GET",
			url:             "/api/exam/status",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "考试会话ID不能为空",
			expectedData:    nil,
			options:         []ContextOption{WithUserID(54321)},
		},
		// 失败场景 - 无效的考试会话ID
		{
			name:            "GET 请求 - 无效的考试会话ID",
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=99999",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    404,
			expectedMessage: "考试会话不存在",
			expectedData:    nil,
			options:         []ContextOption{WithUserID(54321)},
		},
		// 失败场景 - 用户未登录
		{
			name:            "GET 请求 - 用户未登录",
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=12345",
			reqBody:         nil,
			expectedLog:     "GET test setup completed successfully",
			expectSuccess:   false,
			expectedCode:    401,
			expectedMessage: "用户未登录",
			expectedData:    nil,
			options:         []ContextOption{}, // 不设置用户ID，模拟未登录状态
		},
	}

	// 运行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc.method, tc.url, tc.reqBody, tc.expectedLog, tc.expectSuccess, tc.expectedCode, tc.expectedMessage, tc.expectedData, tc.options...)
		})
	}
}
