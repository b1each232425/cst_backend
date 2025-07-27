package paper_respondence

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"

	"testing"

	"w2w.io/cmn"
	"w2w.io/null"
)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func TestStudentAnswer(t *testing.T) {
	cmn.ConfigureForTest()
	// 定义测试用例
	testCases := []struct {
		name     string
		method   string
		url      string
		reqBody  *cmn.ReqProto
		userId   int64
		forceErr string
		ctxKey   string
		ctxValue string
		// 预期结果
		expectSuccess   bool            // 是否期望成功
		expectedMessage string          // 预期错误消息
		expectedData    json.RawMessage // 预期数据（可选）
	}{
		// POST 请求测试用例
		{
			name:   "POST 请求 - 保存学生答案 - 基本情况",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3112,
					"type": "00",
					"question_id": 3684,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1574,
			expectSuccess:   true,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer":["B"]
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1580,
			  "ID": 34795,
			  "ExamineeID":3112,
			  "QuestionID": 3684,
			  "Status": "00",
			  "Type": "00",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1580
			}`),
		},
		{
			name:   "POST 请求 - 保存学生答案 - 练习模式",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["B"]}
				}`),
			},
			expectSuccess:   true,
			userId:          1580,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "北京市朝阳区"
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1580,
			  "ID": 34795,
			  "PracticeSubmissionID": 159,
			  "QuestionID": 3624,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1580
			}`),
		},
		{
			name:   "POST 请求 - 更新学生答案",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			expectSuccess:   true,
			expectedMessage: "",
			userId:          1580,
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "广州市白云区"
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1580,
			  "ID": 34795,
			  "PracticeSubmissionID": 159,
			  "QuestionID": 3624,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1580
			}`),
		},
		{
			name:   "POST 请求 - 带附件的学生答案",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]},
					"attachment_paths": ["path/to/file1.jpg", "path/to/file2.pdf"]
				}`),
			},
			expectSuccess:   true,
			userId:          1580,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "广州市白云区"
			  },
			  "AnswerAttachmentsPath": ["path/to/file1.jpg", "path/to/file2.pdf"],
			  "CreateTime": 1753577351944,
			  "Creator": 1580,
			  "ID": 34795,
			  "PracticeSubmissionID": 159,
			  "QuestionID": 3624,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1580
			}`),
		},
		{
			name:   "POST 请求 - 练习缺少必要参数PracticeSubmissionID",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 练习缺少必要参数question_id",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_submission_id": 159,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 练习缺少必要参数answer",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 练习缺少必要参数answer",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - io read报错",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{

				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "io-readAll",
		},
		{
			name:   "POST 请求 - body close报错",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{

				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "body-close",
		},
		{
			name:   "POST 请求 - data unmarshal error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(``),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:            "POST 请求 - req unmarshal error",
			method:          "POST",
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:            "POST 请求 - buf-zero error",
			method:          "POST",
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "buf-zero",
		},
		{
			name:   "POST 请求 - begin-tx error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "begin-tx",
		},
		{
			name:   "POST 请求 - commit-tx error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "commit-tx",
		},
		{
			name:   "POST 请求 - marshal error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "marshal-Err",
		},
		{
			name:   "POST 请求 - empty-buf error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1580,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - rollback-tx error",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 159,
					"type": "02",
					"question_id": 3624,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:        1580,
			expectSuccess: false,
			forceErr:      "rollback-tx",
		},
		{
			name:   "POST 请求 - err type",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3112,
					"type": "04",
					"question_id": 3684,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1574,
			expectSuccess:   false,
			expectedMessage: "",
		},
		// GET 请求测试用例
		{
			name:            "GET 请求 - 通过考生ID获取学生答案",
			method:          "GET",
			url:             "/api/respondent?question_id=3684&examinee_id=3112",
			reqBody:         nil,
			expectSuccess:   true,
			userId:          1574,
			expectedMessage: "",
			expectedData: json.RawMessage(
				` {"ID":34895,"Type":"00","ExamineeID":3112,"QuestionID":3684,"Answer":{"answer":["B"]},"Creator":1574,"UpdatedBy":1574,"Status":"00","AnswerAttachmentsPath":[],"CreateTime":1753580552008,"UpdateTime":1753600525670}`,
			),
		},
		{
			name:            "GET 请求 - 通过练习提交ID获取学生答案",
			method:          "GET",
			url:             "/api/respondent?question_id=3624&practice_submission_id=159",
			reqBody:         nil,
			expectSuccess:   true,
			userId:          1580,
			expectedMessage: "",
			expectedData: json.RawMessage(
				` {"ID":34795,"Type":"02","QuestionID":3624,"Answer":{"answer":["广州市白云区"]},"Creator":1580,"UpdatedBy":1580,"Status":"00","CreateTime":1753551715472,"UpdateTime":1753583181560,"AnswerAttachmentsPath":[]}`,
			),
		},
		{
			name:            "GET 请求 - 缺少题目ID参数",
			method:          "GET",
			url:             "/api/respondent?examinee_id=12345",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1580,
			expectedMessage: "题目ID不能为空",
		},
		{
			name:            "GET 请求 - 缺少考生ID和练习提交ID参数",
			method:          "GET",
			url:             "/api/respondent?question_id=67890",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1580,
			expectedMessage: "考生ID和练习提交ID不能同时为空",
		},
		{
			name:            "GET 请求 - 考生ID不是数字",
			method:          "GET",
			url:             "/api/respondent?question_id=3684&examinee_id=yes",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1574,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - practiceSubmissionID不是数字",
			method:          "GET",
			url:             "/api/respondent?question_id=3624&practice_submission_id=yes",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1580,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - questionId不是数字",
			method:          "GET",
			url:             "/api/respondent?question_id=yes&examinee_id=3112",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1574,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - no row",
			method:          "GET",
			url:             "/api/respondent?question_id=1&examinee_id=3112",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1574,
			expectedMessage: "",
		},
		{
			name:            "GET 请求 - marshal err",
			method:          "GET",
			url:             "/api/respondent?question_id=3684&examinee_id=3112",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1574,
			expectedMessage: "",

			forceErr: "marshal-err",
		},
		{
			name:            "GET 请求 - error method",
			method:          "PUT",
			url:             "/api/respondent?question_id=3684&examinee_id=3112",
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1574,
			expectedMessage: "",
		},
	}

	// 运行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tc.reqBody != nil {

				// 对于 POST 请求，准备请求体
				bodyBytes, err := json.Marshal(tc.reqBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(bodyBytes))
				if tc.name == "buf-zero" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(nil))
				} else if tc.reqBody.Data == nil {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer([]byte("{")))
				} else if tc.name == "POST 请求 - empty-buf error" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer([]byte("")))
				}
			} else {
				// 对于 GET 请求，没有请求体
				req, err = http.NewRequest(tc.method, tc.url, nil)
			}

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			// 创建模拟的上下文，应用自定义选项
			ctx := createMockContext(t, req, tc.userId)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}

			// // 验证请求和上下文是否正确设置
			// verifyRequestAndContext(t, req, ctx, method)

			// 执行 StudentAnswer 函数
			StudentAnswer(ctx)

			// 从上下文中获取响应
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)
			// 根据预期结果验证响应
			if tc.expectSuccess {
				switch tc.method {
				case "POST":

					var result cmn.TStudentAnswers
					err := json.Unmarshal(resp.Data, &result)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					var expected cmn.TStudentAnswers
					err = json.Unmarshal(tc.expectedData, &expected)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					assert.Equal(t, expected.PracticeSubmissionID, result.PracticeSubmissionID)
					assert.Equal(t, expected.ExamineeID, result.ExamineeID)
					assert.JSONEq(t, expected.Answer.String(), expected.Answer.String())
					assert.Equal(t, expected.QuestionID, result.QuestionID)
					assert.JSONEq(t, expected.AnswerAttachmentsPath.String(), result.AnswerAttachmentsPath.String())
					assert.Equal(t, resp.Status, 0)

				case "GET":
					var result cmn.TStudentAnswers
					err := json.Unmarshal(resp.Data, &result)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					var expected cmn.TStudentAnswers
					err = json.Unmarshal(tc.expectedData, &expected)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					assert.Equal(t, expected.PracticeSubmissionID, result.PracticeSubmissionID)
					assert.JSONEq(t, expected.Answer.String(), expected.Answer.String())
					assert.Equal(t, expected.QuestionID, result.QuestionID)

					assert.JSONEq(t, expected.AnswerAttachmentsPath.String(), result.AnswerAttachmentsPath.String())
					assert.Equal(t, resp.Status, 0)
				}
			} else {

				assert.NotEmpty(t, resp.Msg)
				assert.Empty(t, resp.Data)
			}
		})
	}
}

// 创建模拟的上下文，更加通用的版本，支持自定义用户ID和请求头
func createMockContext(t *testing.T, req *http.Request, userId int64) context.Context {
	// 创建基本的上下文
	ctx := context.Background()

	// 创建响应记录器
	rec := httptest.NewRecorder()

	// 创建默认的服务上下文
	q := &cmn.ServiceCtx{
		R: req,
		W: rec,
		SysUser: &cmn.TUser{
			ID: null.IntFrom(userId), // 默认用户ID
		},
		Msg: &cmn.ReplyProto{},
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}

//func TestCheckExamStatus(t *testing.T) {
//	// 定义测试用例
//	testCases := []struct {
//		name            string
//		method          string
//		url             string
//		reqBody         *cmn.ReqProto
//		expectedLog     string
//		expectSuccess   bool
//		expectedCode    int
//		expectedMessage string
//		expectedData    interface{}
//		options         []ContextOption
//	}{
//		// 成功场景 - 有效的考试会话ID
//		{
//			name:            "GET 请求 - 有效的考试会话ID",
//			method:          "GET",
//			url:             "/api/exam/status?exam_session_id=12345",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   true,
//			expectedCode:    0,
//			expectedMessage: "",
//			expectedData:    nil, // 根据实际情况设置预期数据
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 缺少考试会话ID
//		{
//			name:            "GET 请求 - 缺少考试会话ID",
//			method:          "GET",
//			url:             "/api/exam/status",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "考试会话ID不能为空",
//			expectedData:    nil,
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 无效的考试会话ID
//		{
//			name:            "GET 请求 - 无效的考试会话ID",
//			method:          "GET",
//			url:             "/api/exam/status?exam_session_id=99999",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    404,
//			expectedMessage: "考试会话不存在",
//			expectedData:    nil,
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 用户未登录
//		{
//			name:            "GET 请求 - 用户未登录",
//			method:          "GET",
//			url:             "/api/exam/status?exam_session_id=12345",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    401,
//			expectedMessage: "用户未登录",
//			expectedData:    nil,
//			options:         []ContextOption{}, // 不设置用户ID，模拟未登录状态
//		},
//	}
//
//	// 运行所有测试用例
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			runTestCase(t, tc.method, tc.url, tc.reqBody, tc.expectedLog, tc.expectSuccess, tc.expectedCode, tc.expectedMessage, tc.expectedData, tc.options...)
//		})
//	}
//}
//
//func TestInitRespondent(t *testing.T) {
//	// 定义测试用例
//	testCases := []struct {
//		name            string
//		method          string
//		url             string
//		reqBody         *cmn.ReqProto
//		expectedLog     string
//		expectSuccess   bool
//		expectedCode    int
//		expectedMessage string
//		expectedData    interface{}
//		options         []ContextOption
//	}{
//		// 成功场景 - 考试类型初始化
//		{
//			name:   "POST 请求 - 考试类型初始化 - 成功",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_id": 12345,
//					"exam_session_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   true,
//			expectedCode:    0,
//			expectedMessage: "success",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 成功场景 - 练习类型初始化
//		{
//			name:   "POST 请求 - 练习类型初始化 - 成功",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "01",
//					"practice_id": 12345,
//					"practice_submission_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   true,
//			expectedCode:    0,
//			expectedMessage: "success",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 未登录用户
//		{
//			name:   "POST 请求 - 未登录用户",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_id": 12345,
//					"exam_session_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "student id is smaller than 0",
//			options:         []ContextOption{}, // 不设置用户ID，模拟未登录状态
//		},
//		// 失败场景 - 考试类型但缺少考试ID
//		{
//			name:   "POST 请求 - 考试类型但缺少考试ID",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_session_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "当前是考试，请输入大于0的考试id大于0的考试场次id",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 练习类型但缺少练习ID
//		{
//			name:   "POST 请求 - 练习类型但缺少练习ID",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "01",
//					"practice_submission_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "practice id is smaller than 0",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 未知类型
//		{
//			name:   "POST 请求 - 未知类型",
//			method: "POST",
//			url:    "/api/respondent/init",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "99",
//					"exam_id": 12345,
//					"exam_session_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "unknown respondence type",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 使用GET方法
//		{
//			name:            "GET 请求 - 方法不支持",
//			method:          "GET",
//			url:             "/api/respondent/init",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "please call /api/upLogin with  http POST method",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//	}
//
//	// 运行所有测试用例
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			runTestCase(t, tc.method, tc.url, tc.reqBody, tc.expectedLog, tc.expectSuccess, tc.expectedCode, tc.expectedMessage, tc.expectedData, tc.options...)
//		})
//	}
//}
//
//func TestSubmit(t *testing.T) {
//	// 定义测试用例
//	testCases := []struct {
//		name            string
//		method          string
//		url             string
//		reqBody         *cmn.ReqProto
//		expectedLog     string
//		expectSuccess   bool
//		expectedCode    int
//		expectedMessage string
//		expectedData    interface{}
//		options         []ContextOption
//	}{
//		// 成功场景 - 考试类型提交
//		{
//			name:   "POST 请求 - 考试类型提交 - 成功",
//			method: "POST",
//			url:    "/api/respondent/submit",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_id": 12345,
//					"exam_session_id": 67890,
//					"examinee_id": 54321
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   true,
//			expectedCode:    0,
//			expectedMessage: "success",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 成功场景 - 练习类型提交
//		{
//			name:   "POST 请求 - 练习类型提交 - 成功",
//			method: "POST",
//			url:    "/api/respondent/submit",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "01",
//					"practice_submission_id": 67890
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   true,
//			expectedCode:    0,
//			expectedMessage: "success",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 未登录用户
//		{
//			name:   "POST 请求 - 未登录用户",
//			method: "POST",
//			url:    "/api/respondent/submit",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_id": 12345,
//					"exam_session_id": 67890,
//					"examinee_id": 54321
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "validation failed", // 假设验证会失败，因为studentId为0
//			options:         []ContextOption{},   // 不设置用户ID，模拟未登录状态
//		},
//		// 失败场景 - 考试类型但缺少考试ID
//		{
//			name:   "POST 请求 - 考试类型但缺少考试ID",
//			method: "POST",
//			url:    "/api/respondent/submit",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "00",
//					"exam_session_id": 67890,
//					"examinee_id": 54321
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "当前是考试，请输入大于0的考试id大于0的考生id",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 练习类型但缺少提交ID
//		{
//			name:   "POST 请求 - 练习类型但缺少提交ID",
//			method: "POST",
//			url:    "/api/respondent/submit",
//			reqBody: &cmn.ReqProto{
//				Data: json.RawMessage(`{
//					"type": "01"
//				}`),
//			},
//			expectedLog:     "POST test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "当前是练习，请输入大于0的PracticeSubmissionID",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//		// 失败场景 - 使用GET方法
//		{
//			name:            "GET 请求 - 方法不支持",
//			method:          "GET",
//			url:             "/api/respondent/submit",
//			reqBody:         nil,
//			expectedLog:     "GET test setup completed successfully",
//			expectSuccess:   false,
//			expectedCode:    400,
//			expectedMessage: "please call /api/upLogin with  http POST method",
//			options:         []ContextOption{WithUserID(54321)},
//		},
//	}
//
//	// 运行所有测试用例
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			runTestCase(t, tc.method, tc.url, tc.reqBody, tc.expectedLog, tc.expectSuccess, tc.expectedCode, tc.expectedMessage, tc.expectedData, tc.options...)
//		})
//	}
//}
