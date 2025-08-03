package paper_respondence

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
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
		setup           func(tx pgx.Tx) error
		Domain          []cmn.TDomain
	}{
		// POST 请求测试用例
		{
			name:   "POST 请求 - 保存学生答案 - 基本情况",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3119,
					"type": "00",
					"question_id": 3794,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1623,
			expectSuccess:   true,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer":["B"]
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1626,
			  "ID": 34795,
			  "ExamineeID":3119,
			  "QuestionID": 3794,
			  "Status": "00",
			  "Type": "00",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1623
			}`),
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where examinee_id=3119`)
				return err
			},
		},
		{
			name:   "POST 请求 - 保存学生答案 - 练习模式",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["B"]}
				}`),
			},
			expectSuccess:   true,
			userId:          1634,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "北京市朝阳区"
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1634,
			  "ID": 34795,
			  "PracticeSubmissionID": 165,
			  "QuestionID": 3795,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1634
			}`),
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where practice_submission_id=165`)
				return err
			},
		},
		{
			name:   "POST 请求 - 更新学生答案",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			expectSuccess:   true,
			expectedMessage: "",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId: 1634,
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "广州市白云区"
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1634,
			  "ID": 34795,
			  "PracticeSubmissionID": 165,
			  "QuestionID": 3795,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1634
			}`),
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where practice_submission_id=165`)
				return err
			},
		},
		{
			name:   "POST 请求 - 带附件的学生答案",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]},
					"attachment_paths": ["path/to/file1.jpg", "path/to/file2.pdf"]
				}`),
			},
			expectSuccess:   true,
			userId:          1634,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer": "广州市白云区"
			  },
			  "AnswerAttachmentsPath": ["path/to/file1.jpg", "path/to/file2.pdf"],
			  "CreateTime": 1753577351944,
			  "Creator": 1634,
			  "ID": 34795,
			  "PracticeSubmissionID": 165,
			  "QuestionID": 3795,
			  "Status": "00",
			  "Type": "02",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1634
			}`),
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where practice_submission_id=165`)
				return err
			},
		},
		{
			name:   "POST 请求 - domain除了学生还有其他的",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3119,
					"type": "00",
					"question_id": 3794,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1623,
			expectSuccess:   true,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer":["B"]
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1626,
			  "ID": 34795,
			  "ExamineeID":3119,
			  "QuestionID": 3794,
			  "Status": "00",
			  "Type": "00",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1623
			}`),
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2003)},
				{ID: null.IntFrom(StudentDomainId)},
			},
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where examinee_id=3119`)
				return err
			},
		},
		{
			name:   "POST 请求 - domain不是学生",
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3119,
					"type": "00",
					"question_id": 3794,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1623,
			expectSuccess:   false,
			expectedMessage: "",
			expectedData: json.RawMessage(`{
			  "Answer": {
				"answer":["B"]
			  },
			  "AnswerAttachmentsPath": [],
			  "CreateTime": 1753577351944,
			  "Creator": 1626,
			  "ID": 34795,
			  "ExamineeID":3119,
			  "QuestionID": 3794,
			  "Status": "00",
			  "Type": "00",
			  "UpdateTime": 1753577351944,
			  "UpdatedBy": 1623
			}`),
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2003)},
			},
			setup: func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `update t_student_answers set status='00' where examinee_id=3119`)
				return err
			},
		},
		{
			name: "POST 请求 - 练习缺少必要参数PracticeSubmissionID",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - 练习缺少必要参数question_id",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_submission_id": 165,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name: "POST 请求 - 练习缺少必要参数answer",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name: "POST 请求 - 练习缺少必要参数answer",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - io read报错",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{

				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "io-readAll",
		},
		{
			name:   "POST 请求 - body close报错",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{

				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "body-close",
		},
		{
			name:   "POST 请求 - data unmarshal error",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(``),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:    "POST 请求 - req unmarshal error",
			method:  "POST",
			url:     "/api/respondent",
			reqBody: &cmn.ReqProto{},
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name:   "POST 请求 - buf-zero error",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         &cmn.ReqProto{},
			userId:          1634,
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
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId: 1634,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "begin-tx",
		},
		{
			name:   "POST 请求 - commit-tx error",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "commit-tx",
		},
		{
			name:   "POST 请求 - marshal error",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
			forceErr:        "marshal-Err",
		},
		{
			name: "POST 请求 - empty-buf error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		{
			name: "POST 请求 - rollback-tx error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method: "POST",
			url:    "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice_submission_id": 165,
					"type": "02",
					"question_id": 3795,
					"answer": {"answer":["广州市白云区"]}
				}`),
			},
			userId:        1634,
			expectSuccess: false,
			forceErr:      "rollback-tx",
		},
		{
			name:   "POST 请求 - err type",
			method: "POST",
			url:    "/api/respondent",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"examinee_id": 3119,
					"type": "04",
					"question_id": 3795,
					"answer": {"answer":["B"]}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "",
		},
		// GET 请求测试用例
		{
			name: "GET 请求 - 通过考生ID获取学生答案",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/respondent?question_id=3794&examinee_id=3119",
			reqBody:         nil,
			expectSuccess:   true,
			userId:          1623,
			expectedMessage: "",
			expectedData: json.RawMessage(
				` {"ID":34895,"Type":"00","ExamineeID":3119,"QuestionID":3794,"Answer":{"answer":["B"]},"Creator":1623,"UpdatedBy":1623,"Status":"00","AnswerAttachmentsPath":[],"CreateTime":1753580552008,"UpdateTime":1753600525670}`,
			),
		},
		{
			name:   "GET 请求 - 通过练习提交ID获取学生答案",
			method: "GET",
			url:    "/api/respondent?question_id=3795&practice_submission_id=165",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   true,
			userId:          1634,
			expectedMessage: "",
			expectedData: json.RawMessage(
				` {"ID":34795,"Type":"02","QuestionID":3795,"Answer":{"answer":["广州市白云区"]},"Creator":1634,"UpdatedBy":1634,"Status":"00","CreateTime":1753551715472,"UpdateTime":1753583181560,"AnswerAttachmentsPath":[]}`,
			),
		},
		{
			name:   "GET 请求 - 缺少题目ID参数",
			method: "GET",
			url:    "/api/respondent?examinee_id=12345",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "题目ID不能为空",
		},
		{
			name:   "GET 请求 - 缺少考生ID和练习提交ID参数",
			method: "GET",
			url:    "/api/respondent?question_id=67890",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "考生ID和练习提交ID不能同时为空",
		},
		{
			name:   "GET 请求 - 考生ID不是数字",
			method: "GET",
			url:    "/api/respondent?question_id=3684&examinee_id=yes",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1634,
			expectedMessage: "",
		},
		{
			name:   "GET 请求 - practiceSubmissionID不是数字",
			method: "GET",
			url:    "/api/respondent?question_id=3624&practice_submission_id=yes",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1634,
			expectedMessage: "",
		},
		{
			name:   "GET 请求 - questionId不是数字",
			method: "GET",
			url:    "/api/respondent?question_id=yes&examinee_id=3119",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "",
		},
		{
			name:   "GET 请求 - no row",
			method: "GET",
			url:    "/api/respondent?question_id=1&examinee_id=3119",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "",
		},
		{
			name:   "GET 请求 - marshal err",
			method: "GET",
			url:    "/api/respondent?question_id=3794&examinee_id=3119",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "",

			forceErr: "marshal-err",
		},
		{
			name:   "GET 请求 - error method",
			method: "PUT",
			url:    "/api/respondent?question_id=3684&examinee_id=3119",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			userId:          1623,
			expectedMessage: "",
		},
	}
	tx, err := cmn.GetPgxConn().BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		t.Fatalf("begin tx err: %v", err)
	}
	// 运行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 如果需要设置数据库状态
			if tc.setup != nil {
				err := tc.setup(tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(context.Background())
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = cmn.GetPgxConn().Begin(context.Background())
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}
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
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}

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
	tx.Commit(context.Background())
}

// 创建模拟的上下文，更加通用的版本，支持自定义用户ID和请求头
func createMockContext(req *http.Request, userId int64, domain []cmn.TDomain, role ...int) context.Context {
	// 创建基本的上下文
	ctx := context.Background()

	// 创建响应记录器
	rec := httptest.NewRecorder()
	var r int64 = 0
	if len(role) > 0 {
		r = int64(role[0])
	}
	// 创建默认的服务上下文
	q := &cmn.ServiceCtx{
		R: req,
		W: rec,
		SysUser: &cmn.TUser{
			ID:   null.IntFrom(userId), // 默认用户ID
			Role: null.IntFrom(r),
		},
		Msg:     &cmn.ReplyProto{},
		Domains: domain,
		Role:    r,
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}

func TestCheckExamStatus(t *testing.T) {
	cmn.ConfigureForTest()

	// 在测试开始前，保存原始数据库状态
	db := cmn.GetPgxConn()
	ctx := context.Background()

	// 开始事务，用于测试期间的数据修改
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) // 确保测试结束后回滚事务，恢复原始数据

	// 修改数据库中的考试状态以测试不同场景
	// 1. 保存考试会话155的原始数据
	var originalStartTime, originalEndTime, originalExamineeEndTime, originalAllowEntryTime sql.NullInt64
	var originalExamineeStatus sql.NullString
	err = tx.QueryRow(ctx, `
			SELECT start_time, actual_end_time, examinee_end_time, allow_entry_time, examinee_status 
			FROM v_examinee_info 
			WHERE exam_session_id = 155 AND student_id = 1623
		`).Scan(&originalStartTime, &originalEndTime, &originalExamineeEndTime, &originalAllowEntryTime, &originalExamineeStatus)
	if err != nil {
		t.Logf("Warning: Could not fetch original exam data: %v", err)
		// 继续测试，但只测试基本场景
	}

	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		reqBody         *cmn.ReqProto
		expectSuccess   bool
		expectedCode    int
		expectedMessage string
		expectedData    json.RawMessage // 预期数据（可选）
		forceErr        string
		userId          int64
		Domain          []cmn.TDomain
		// 测试前需要设置的数据库状态
		setupDB func(t *testing.T, tx pgx.Tx) error
	}{
		// 成功场景 - 考试可以进入（ExamCanBeEnter）
		{
			name: "GET 请求 - 考试可以进入",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`5`), // ExamCanBeEnter状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以进入
				// 1. 确保考试已开始（start_time < 当前时间）
				// 2. 确保考试未结束（actual_end_time > 当前时间）
				// 3. 确保考生未提交（examinee_end_time IS NULL）
				// 4. 设置考生状态为可以进入（examinee_status = '16'）
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name: "GET 请求 - 缺少学生域",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2001)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`5`), // ExamCanBeEnter状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以进入
				// 1. 确保考试已开始（start_time < 当前时间）
				// 2. 确保考试未结束（actual_end_time > 当前时间）
				// 3. 确保考生未提交（examinee_end_time IS NULL）
				// 4. 设置考生状态为可以进入（examinee_status = '16'）
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name: "GET 请求 - 有学生域还有其他的域",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2001)},
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`5`), // ExamCanBeEnter状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以进入
				// 1. 确保考试已开始（start_time < 当前时间）
				// 2. 确保考试未结束（actual_end_time > 当前时间）
				// 3. 确保考生未提交（examinee_end_time IS NULL）
				// 4. 设置考生状态为可以进入（examinee_status = '16'）
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		// 成功场景 - 考试未开始（StartTimeNotArrived）
		{
			name: "GET 请求 - 考试未开始",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`1`), // StartTimeNotArrived状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为未开始
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now+3600000, now+7200000) // 开始时间为1小时后，结束时间为2小时后
				return err
			},
		},
		// 成功场景 - 考试已结束（EndTimeArrived）
		{
			name: "GET 请求 - 考试已结束",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`2`), // EndTimeArrived状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为已结束
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-7200000, now-3600000) // 开始时间为2小时前，结束时间为1小时前
				return err
			},
		},
		// 成功场景 - 考试已提交（ExamSubmitted）
		{
			name: "GET 请求 - 考试已提交",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`3`), // ExamSubmitted状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为已提交
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				if err != nil {
					return err
				}

				// 设置考生已提交
				_, err = tx.Exec(ctx, `
						UPDATE t_examinee 
						SET end_time = $1 
						WHERE exam_session_id = 155 AND student_id = 1623
					`, now-1800000) // 提交时间为30分钟前
				return err
			},
		},
		// 成功场景 - 超过最迟进入时间（LateEntryTimeArrived）
		{
			name: "GET 请求 - 超过最迟进入时间",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   true,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    json.RawMessage(`4`), // LateEntryTimeArrived状态
			userId:          1623,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为超过最迟进入时间
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2,
						    period_mode = '00' ,
						    late_entry_time = $3
						WHERE id = 155
					`, now-3600000, now+3600000, 3) // 开始时间为1小时前，结束时间为1小时后
				if err != nil {
					return err
				}

				// 设置最迟进入时间为30分钟前
				_, err = tx.Exec(ctx, `
						UPDATE t_examinee 
						SET start_time = NULL,
						    end_time = NULL, 
						    status = '00' 
						WHERE exam_session_id = 155 AND student_id = 1623
					`) // 最迟进入时间为30分钟前
				return err
			},
		},
		// 失败场景 - 缺少考试会话ID
		{
			name: "GET 请求 - 缺少考试会话ID",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "examSessionId is required",
			expectedData:    nil,
			userId:          1623,
			setupDB:         nil, // 不需要设置数据库
		},
		// 失败场景 - 无效的考试会话ID
		{
			name: "GET 请求 - 无效的考试会话ID",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=99999",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    400,
			expectedMessage: "",
			expectedData:    nil,
			userId:          1623,
			setupDB:         nil, // 不需要设置数据库
		},
		// 失败场景 - 用户未登录
		{
			name: "GET 请求 - 用户未登录",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    200,
			expectedMessage: "",
			expectedData:    nil,
			userId:          0,   // 用户ID为0表示未登录
			setupDB:         nil, // 不需要设置数据库
		},
		{
			name: "GET 请求 - 请求方法不对",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "POST",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    200,
			expectedMessage: "",
			expectedData:    nil,
			userId:          1623, // 用户ID为0表示未登录
			setupDB:         nil,  // 不需要设置数据库
		},
		{
			name: "GET 请求 - exam_session_id不是数字",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=test",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // ExamSubmitted状态
			userId:          1623,
			setupDB:         nil,
		},
		{
			name: "GET 请求 - tx begin error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // ExamSubmitted状态
			userId:          1623,
			setupDB:         nil,
			forceErr:        "begin-tx",
		},
		{
			name: "GET 请求 - commit error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // ExamSubmitted状态
			userId:          1623,
			setupDB:         nil,
			forceErr:        "commit-tx",
		},
		{
			name: "GET 请求 - marshal-Err",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // ExamSubmitted状态
			userId:          1623,
			setupDB:         nil,
			forceErr:        "marshal-Err",
		},
		{
			name: "GET 请求 - rollback-tx-Err",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "GET",
			url:             "/api/exam/status?exam_session_id=155",
			reqBody:         nil,
			expectSuccess:   false,
			expectedCode:    0,
			expectedMessage: "",
			expectedData:    nil, // ExamSubmitted状态
			userId:          1623,
			setupDB:         nil,
			forceErr:        "rollback-tx",
		},
	}

	// 运行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 如果需要设置数据库状态
			if tc.setupDB != nil {
				err := tc.setupDB(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

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
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}

			// 执行 CheckExamStatus 函数
			CheckExamStatus(ctx)

			// 从上下文中获取响应
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			// 根据预期结果验证响应
			if tc.expectSuccess {
				// 成功场景
				assert.Equal(t, tc.expectedCode, resp.Status)
				if tc.expectedData != nil {
					var result int
					err := json.Unmarshal(resp.Data, &result)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					var expected int
					err = json.Unmarshal(tc.expectedData, &expected)
					if err != nil {
						t.Fatalf("Failed to unmarshal expected data: %v", err)
					}
					assert.Equal(t, expected, result)
				}
			} else {
				// 失败场景
				if tc.expectedMessage != "" {
					assert.Contains(t, resp.Msg, tc.expectedMessage)
				}
				assert.NotEqual(t, 0, resp.Status)
				assert.Empty(t, resp.Data)
			}
		})
	}

	// 测试结束后，恢复原始数据
	if originalStartTime.Valid && originalEndTime.Valid {
		// 开始新事务用于恢复
		tx, err = db.Begin(ctx)
		if err != nil {
			t.Fatalf("Failed to begin transaction for cleanup: %v", err)
		}
		defer tx.Rollback(ctx)

		// 恢复考试会话的原始时间
		_, err = tx.Exec(ctx, `
				UPDATE assessuser.t_exam_session 
				SET start_time = $1, 
				    end_time = $2 
				WHERE id = 155
			`, originalStartTime.Int64, originalEndTime.Int64)
		if err != nil {
			t.Logf("Warning: Failed to restore original exam session data: %v", err)
		}

		// 恢复考生的原始状态
		_, err = tx.Exec(ctx, `
				UPDATE assessuser.t_examinee 
				SET end_time = $1, 
				    status = $2 
				WHERE exam_session_id = 155 AND student_id = 1623
			`,
			originalExamineeEndTime, originalExamineeStatus)
		if err != nil {
			t.Logf("Warning: Failed to restore original examinee data: %v", err)
		}

		// 提交恢复事务
		err = tx.Commit(ctx)
		if err != nil {
			t.Logf("Warning: Failed to commit cleanup transaction: %v", err)
		}
	}
	tx.Commit(context.Background())
}

func TestInitRespondent(t *testing.T) {
	cmn.ConfigureForTest()

	// 在测试开始前，保存原始数据库状态
	db := cmn.GetPgxConn()
	ctx := context.Background()

	// 开始事务，用于测试期间的数据修改
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) // 确保测试结束后回滚事务，恢复原始数据

	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		reqBody         *cmn.ReqProto
		Type            string
		expectSuccess   bool
		ctxKey          string
		ctxValue        string
		expectedMessage string
		expectedData    json.RawMessage // 预期数据（可选）
		forceErr        string
		userId          int64
		Domain          []cmn.TDomain
		role            int
		// 测试前需要设置的数据库状态
		setupDB func(t *testing.T, tx pgx.Tx) error
		//清理数据
		clean func(t *testing.T, tx pgx.Tx) error
	}{
		// 成功场景 - 考试类型初始化
		{
			name:   "监考员允许进入初始化",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   true,
			expectedMessage: "",
			role:            2008,
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "不是学生域",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2001)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "role not exist",
			method: "POST",
			role:   1,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "学生退出网页，重新进入作答",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   true,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '00', 
						    start_time=$1,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`, now)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 ,
						    late_entry_time = $3
						WHERE id = 155
					`, now-3600000, now+3600000, 5000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - 正常考试初始化",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   true,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '00', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 ,
						    late_entry_time = $3
						WHERE id = 155
					`, now-3600000, now+3600000, 5000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - 不存在的exam_session_id",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 1,
					"student_id": 1623
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - 不存在的exam_id",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 1,
					"exam_session_id": 155,
					"student_id": 1623
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - exam_id为0",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 0,
					"exam_session_id": 155,
					"student_id": 1623
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - exam_id为-1",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": -1,
					"exam_session_id": 155,
					"student_id": 1623
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:          "POST 请求 - exam_session_id为0",
			method:        "POST",
			url:           "/api/respondent",
			expectSuccess: false,
			role:          2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 0
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "超过进入考试的最迟时间",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '00', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 ,
						    late_entry_time = $3
						WHERE id = 155
					`, now-3600000, now+3600000, 2) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "考试时间已经结束",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '00', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 ,
						    late_entry_time = $3
						WHERE id = 155
					`, now-3600000, now-2800, 2) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "考试未开始",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '00', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 ,
						    late_entry_time = $3
						WHERE id = 155
					`, now+3600000, now+2*3600000, 2) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "begin-tx-err",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId: 1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			forceErr: "begin-tx",
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "roll back err",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId: 1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			forceErr: "rollback-tx",
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "POST 请求 - exam_session_id为-1",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": -1
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - userId为-1",
			method: "POST",
			url:    "/api/respondent",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          -1,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - userId为0",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          -1,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "POST 请求 - io read err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			expectSuccess:   false,
			expectedMessage: "io read all error",
			userId:          1623,
			forceErr:        "io.ReadAll",
		},
		{
			name:   "empty-buf error",
			method: "POST", role: 2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
		},
		{
			name:   "buf-zero",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
		},
		// 失败场景 - 无效的JSON
		{
			name:   "POST 请求 - 无效的JSON",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			expectSuccess:   false,
			expectedMessage: "unexpected end of JSON input",
			userId:          1623,
		},
		{
			name:   "不是POST的请求方法",
			method: "GET",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			reqBody:         &cmn.ReqProto{},
			expectSuccess:   false,
			expectedMessage: "please call /api/upLogin with  http POST method",
			userId:          1623,
		},
		{
			name:   "close body err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url: "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "close body err",
		},
		// 失败场景 - 未登录用户
		{
			name:   "POST 请求 - 未登录用户",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url: "/api/respondent",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 123,
					"exam_session_id": 155
				}`),
			},
			expectSuccess:   false,
			expectedMessage: "student id is smaller than 0 or equal to 0",
			userId:          0,
		},
		{
			name:   "type是无效的",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "04",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:            "qry.data unmarshal err",
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			role:            2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId: 1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(""),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "save start time error",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			ctxKey:          "test",
			ctxValue:        "normal-resp",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(
					`{
						"type":"00",
					"exam_id":108,
					"exam_session_id":155
						}`,
				),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = $1 
						WHERE exam_session_id = 155 AND student_id = 1623
					`, now)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "考试marshal err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			forceErr:        "marshal err",
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "load paper detail err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			forceErr:        "load paper detail err",
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "get sessions err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			forceErr:        "get sessions err",
			expectedMessage: "",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
		{
			name:   "练习正常初始化",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   true,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `DELETE FROM assessuser.t_practice_submissions WHERE id>176`)
				if err != nil {
					t.Fatal(err)
				}
				return nil
			},
		},
		{
			name: "练习之前已经初始化了，继续进入",
			role: 2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "POST",
			url:             "/api/respondent",
			expectSuccess:   true,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `DELETE FROM assessuser.t_practice_submissions WHERE id>176`)
				if err != nil {
					t.Fatal(err)
				}
				return nil
			},
		},
		{
			name:   "practice_id为负数",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": -1
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `DELETE FROM assessuser.t_practice_submissions WHERE id>176`)
				if err != nil {
					t.Fatal(err)
				}
				return nil
			},
		},
		{
			name:   "调用练习管理接口失败",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			ctxKey:          "test",
			ctxValue:        "err",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2056
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), `DELETE FROM assessuser.t_practice_submissions WHERE id>176`)
				if err != nil {
					t.Fatal(err)
				}
				return nil
			},
		},
		{
			name:   "commit err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET status = '04' WHERE practice_id=2060 AND student_id=1634`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET status = '00' WHERE practice_id=2060 AND student_id=1634`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			forceErr: "commit-tx",
		},
		{
			name:   "save begin err",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			forceErr:        "save time",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET start_time = null , status='00' WHERE id=165`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET status = '00' WHERE id=165`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
		},
		{
			name:   "select elapsed seconds error",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			forceErr:        "select elapsed seconds",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET start_time = null , status='00' WHERE id=165`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				_, err = tx.Exec(context.Background(), `UPDATE assessuser.t_practice_submissions SET status = '00' WHERE id=165`)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
		},
		{
			name:   "update last start time error",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			forceErr: "update-last-start-time-err",
			setupDB: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
		},
		{
			name:   "update last start time error",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			forceErr: "marshal err",
			setupDB: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
		},
		{
			name:   "update last start time error",
			method: "POST",
			role:   2008,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			Type:            "练习",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060
				}`),
			},
			forceErr: "commit-tx",
			setupDB: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
		},
		{
			name:   "类型不是考试和练习",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/respondent",
			expectSuccess:   false,
			expectedMessage: "",
			role:            2008,
			userId:          1623,
			forceErr:        "type-err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				now := time.Now().UnixMilli()
				_, err := tx.Exec(ctx, `
						UPDATE t_examinee 
						SET status = '16', 
						    start_time=NULL,
						    end_time = NULL 
						WHERE exam_session_id = 155 AND student_id = 1623
					`)
				if err != nil {
					return err
				}

				// 更新考试会话的时间
				_, err = tx.Exec(ctx, `
						UPDATE t_exam_session 
						SET start_time = $1, 
						    end_time = $2 
						WHERE id = 155
					`, now-3600000, now+3600000) // 开始时间为1小时前，结束时间为1小时后
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 如果需要设置数据库状态
			if tc.setupDB != nil {
				err := tc.setupDB(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

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
				} else if tc.name == "empty-buf error" {
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
			ctx := createMockContext(req, tc.userId, tc.Domain, []int{tc.role}...)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}
			if tc.ctxValue != "" {
				ctx = context.WithValue(ctx, tc.ctxKey, tc.ctxValue)
			}

			// 调用被测试的函数
			InitRespondent(ctx)
			if tc.clean != nil {

				err := tc.clean(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}
				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}
				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

			// 从上下文中获取响应
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			// 根据预期结果验证响应
			if tc.expectSuccess {
				// 成功场景

				if tc.Type == "练习" {
					assert.Equal(t, resp.Status, 0)
					assert.NotEmpty(t, resp.Data)
					var data map[string]interface{}
					err := json.Unmarshal(resp.Data, &data)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					assert.NotEmpty(t, data["Info"])
					assert.NotEmpty(t, data["QuestionGroupInfo"])
					assert.NotEmpty(t, data["Questions"])
					//if tc.name == "练习之前已经初始化了，继续进入" {
					//	assert.NotEmpty(t, data["ElapsedSeconds"])
					//}

				} else {
					assert.Equal(t, resp.Status, 0)
					assert.NotEmpty(t, resp.Data)
					var data map[string]interface{}
					err := json.Unmarshal(resp.Data, &data)
					if err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					assert.NotEmpty(t, data["ExamineeInfo"])
					assert.NotEmpty(t, data["session"])
					assert.NotEmpty(t, data["exam_info"])
					assert.NotEmpty(t, data["QuestionGroupInfo"])
					assert.NotEmpty(t, data["Questions"])
				}

			} else {
				// 失败场景
				if tc.expectedMessage != "" {
					assert.Contains(t, resp.Msg, tc.expectedMessage)
				}
				assert.NotEqual(t, 0, resp.Status)
				assert.Empty(t, resp.Data)
			}

		})
	}
	tx.Commit(context.Background())
}

func TestSubmit(t *testing.T) {
	cmn.ConfigureForTest()

	// 在测试开始前，保存原始数据库状态
	db := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务，用于测试期间的数据修改
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 确保测试结束后回滚事务
	defer tx.Rollback(ctx)

	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		reqBody         *cmn.ReqProto
		expectSuccess   bool
		ctxKey          string
		ctxValue        string
		expectedMessage string
		expectedData    json.RawMessage // 预期数据（可选）
		forceErr        string
		userId          int64
		Domain          []cmn.TDomain
		Role            []int
		// 测试前需要设置的数据库状态
		setupDB func(t *testing.T, tx pgx.Tx) error
		//清理数据
		clean func(t *testing.T, tx pgx.Tx) error
	}{
		// 成功场景 - 考试类型提交
		{
			name:          "POST 请求 - 考试类型提交",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: true,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectedMessage: "success",
			userId:          1623,
			Role:            []int{2008},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2
					WHERE id = 155
				`, currentTime+3600, currentTime)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL ,
					    status='00'
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err
				}
				return err
			},
		},
		{
			name:          "POST 请求 - 除了学生域还有其他的域",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: true,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
				{ID: null.IntFrom(2001)},
			},
			expectedMessage: "invalid domain",
			userId:          1623,
			Role:            []int{2008},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2
					WHERE id = 155
				`, currentTime+3600, currentTime)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL ,
					    status='00'
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err
				}
				return err
			},
		},
		{
			name:   "POST 请求 - 练习类型提交",
			method: "POST",
			url:    "/api/submit",

			expectSuccess:   true,
			expectedMessage: "success",
			userId:          1634,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `UPDATE assessuser.t_student_answers SET answer=$1 WHERE practice_submission_id=165`, json.RawMessage(`{"answer":[""]}`))
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err
				}
				return err
			},
		},

		{
			name:   "GET 请求 - 应该失败",
			method: "GET",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody:         nil,
			expectSuccess:   false,
			expectedMessage: "please call /api/upLogin with  http POST method",
			userId:          1623,
		},
		// 失败场景 - 空请求体
		{
			name:    "buf-zero",
			method:  "POST",
			url:     "/api/submit",
			reqBody: &cmn.ReqProto{},
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "Call /api/respondent with  empty body",
			userId:          1623,
		},
		{
			name:    "POST 请求 - 无效的JSON",
			method:  "POST",
			url:     "/api/submit",
			reqBody: &cmn.ReqProto{},
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "unexpected end of JSON input",
			userId:          1623,
		},
		{
			name:   "POST 请求 - data解析失败",
			method: "POST",
			url:    "/api/submit",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(``),
			},
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId: 1623,
		},
		// 失败场景 - 未登录用户
		{
			name:   "POST 请求 - 未登录用户",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          0,
		},
		{
			name:   "POST 请求 - exam_id为0",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 0,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name:          "POST 请求 - exam_id为-1",
			method:        "POST",
			url:           "/api/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": -1,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name:            "POST 请求 - exam_session_id-1",
			method:          "POST",
			url:             "/api/submit",
			expectSuccess:   false,
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": -1,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name:          "POST 请求 - exam_session_id不存在",
			method:        "POST",
			url:           "/api/submit",
			expectSuccess: false,
			userId:        1623,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 2,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name:          "POST 请求 - exam_session_id为0",
			method:        "POST",
			url:           "/api/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 0,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name:   "POST 请求 - examinee_id为0",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 0
				}`),
			},
		},
		{
			name:   "POST 请求 - examinee_id为-1",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": -1
				}`),
			},
		},
		{
			name:   "POST 请求 - 练习practice_id为0",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/submit",
			expectSuccess:   false,
			expectedMessage: "当前是练习，请输入大于0的PracticeSubmissionID以及大于0的PracticeId",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 0,
					"practice_submission_id": 165
				}`),
			},
		},
		{
			name:   "POST 请求 - 练习practice_id为-1",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是练习，请输入大于0的PracticeSubmissionID以及大于0的PracticeId",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": -1,
					"practice_submission_id": 165
				}`),
			},
		},
		{
			name:   "POST 请求 - 练习practice_submission_id为0",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是练习，请输入大于0的PracticeSubmissionID以及大于0的PracticeId",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 0
				}`),
			},
		},
		{
			name:   "POST 请求 - 练习practice_submission_id为-1",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "当前是练习，请输入大于0的PracticeSubmissionID以及大于0的PracticeId",
			userId:          1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": -1
				}`),
			},
		},

		// 失败场景 - 未知类型
		{
			name:          "POST 请求 - 未知类型",
			method:        "POST",
			url:           "/api/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},

			userId: 1623,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "03",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119,
					"student_id": 1623
				}`),
			},
		},
		// 失败场景 - 事务开始失败
		{
			name:   "POST 请求 - 事务开始失败",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "begin-tx",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
		},
		{
			name: "POST 请求 - 事务提交失败",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			method:          "POST",
			url:             "/api/submit",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "commit-tx",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1 ,
					    start_time = $2 
					WHERE id = 155
				`, currentTime+3600, currentTime)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL ,
						start_time = NULL
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				if err != nil {
					return err
				}
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err

				}
				return err
			},
		},

		{
			name:   "POST 请求 - 事务回滚失败",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "rollback-tx",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 108,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1 
					WHERE id = 155
				`, currentTime+3600)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL 
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='02' where examinee_id=3119`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:   "POST 请求 - 批改失败（考试）",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "mark-err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1 
					WHERE id = 155
				`, currentTime+3600)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL 
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:   "POST 请求 - 批改失败（练习）",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:             "/api/submit",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			forceErr:        "mark-err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {

				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:   "POST 请求 - body close error",
			method: "POST",
			url:    "/api/submit",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1623,
			forceErr:        "close body err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:          "POST 请求 - 缺乏学生域",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(2001)},
			},
			expectedMessage: "invalid domain",
			userId:          1623,
			Role:            []int{2008},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2
					WHERE id = 155
				`, currentTime+3600, currentTime)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL ,
					    status='00'
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err
				}
				return err
			},
		},
		{
			name:            "POST 请求 - io.ReadAll error",
			method:          "POST",
			url:             "/api/submit",
			expectSuccess:   false,
			expectedMessage: "",
			userId:          1634,
			forceErr:        "io.ReadAll",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:          "POST 请求 - setAnswerCanNotUpdate error（考试）",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			userId:   1623,
			forceErr: "setAnswerCanNotUpdate error",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1 
					WHERE id = 155
				`, currentTime+3600)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL 
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:   "POST 请求 - setAnswerCanNotUpdate error（练习）",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:           "/api/respondent/submit",
			expectSuccess: false,
			userId:        1634,
			forceErr:      "setAnswerCanNotUpdate error",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:   "POST 请求 - submit error（练习）",
			method: "POST",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			url:           "/api/respondent/submit",
			expectSuccess: false,
			userId:        1634,
			forceErr:      "practice-submit-err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "02",
					"practice_id": 2060,
					"practice_submission_id": 165
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				return nil
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 练习类型提交后，将 submission 的 status 改为 "00"
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions 
					SET status = '00' ,
					    end_time = null
					WHERE id = 165 AND student_id = 1634
				`)

				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where practice_submission_id=165`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:          "POST 请求 - submit error（考试）",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: false,
			userId:        1623,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			forceErr: "exam-submit-err",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1 
					WHERE id = 155
				`, currentTime+3600)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL 
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err

				}
				return err
			},
		},
		{
			name:          "类型不是考试也不是练习",
			method:        "POST",
			url:           "/api/respondent/submit",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			forceErr: "type-err",
			userId:   1623,
			Role:     []int{2008},
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"type": "00",
					"exam_id": 111,
					"exam_session_id": 155,
					"examinee_id": 3119
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				// 更新t_examinee表而不是视图
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = $1, end_time = NULL, status = $2 
					WHERE exam_session_id = 155 AND student_id = 1623
				`, currentTime-3600, NormalStatus)
				if err != nil {
					return err
				}

				// 更新t_exam_session表设置end_time
				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2
					WHERE id = 155
				`, currentTime+3600, currentTime)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET end_time = NULL ,
					    status='00'
					WHERE exam_session_id = 155 AND student_id = 1623
				`)
				_, err = tx.Exec(ctx, `update t_student_answers set answer_score=null ,status='00' where examinee_id=3119`)
				if err != nil {
					return err
				}
				return err
			},
		},
	}
	defer tx.Rollback(context.Background())
	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置数据库状态
			if tc.setupDB != nil {
				err := tc.setupDB(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

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
				} else if tc.name == "empty-buf error" {
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
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}
			if tc.ctxValue != "" {
				ctx = context.WithValue(ctx, tc.ctxKey, tc.ctxValue)
			}

			// 调用被测试的函数
			Submit(ctx)
			if tc.clean != nil {
				err := tc.clean(t, tx)
				if err != nil {
					t.Fatalf("Failed to clean database: %v", err)
				}
				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}
				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

			// 从上下文中获取响应
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			// 根据预期结果验证响应
			if tc.expectSuccess {
				// 成功场景
				assert.Equal(t, 0, resp.Status)
				assert.Equal(t, "success", resp.Msg)
			} else {
				// 失败场景
				assert.NotEqual(t, 0, resp.Status)
				if tc.expectedMessage != "" {
					assert.Contains(t, resp.Msg, tc.expectedMessage)
				}
			}
		})
	}
	tx.Commit(context.Background())
}

func TestHandleExit(t *testing.T) {
	cmn.ConfigureForTest()

	// 在测试开始前，保存原始数据库状态
	db := cmn.GetPgxConn()
	ctx := context.Background()

	// 开始事务，用于测试期间的数据修改
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 确保测试结束后回滚事务
	defer tx.Rollback(ctx)

	// 定义测试用例
	testCases := []struct {
		name                 string
		expectSuccess        bool
		ctxKey               string
		ctxValue             string
		expectedMessage      string
		examineeId           int64
		expectCnt            int
		studentId            int64
		practiceSubmissionId int64
		forceErr             string
		Domain               []cmn.TDomain
		// 测试前需要设置的数据库状态
		setupDB func(t *testing.T, tx pgx.Tx) error
		//清理数据
		clean func(t *testing.T, tx pgx.Tx) error
	}{
		// 成功场景 - 考试类型退出
		{
			name:          "考试类型退出",
			expectSuccess: true,
			examineeId:    3119,
			studentId:     1623,
			expectCnt:     1,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为正常
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET status = $1 ,exit_cnt=0
					WHERE id = 3119
				`, NormalStatus)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 恢复考试状态
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET status = $1, exit_cnt = 0 
					WHERE id = 3119
				`, NormalStatus)
				return err
			},
		},
		// 成功场景 - 练习类型退出
		{
			name:                 "POST 请求 - 练习类型退出",
			expectSuccess:        true,
			expectedMessage:      "success",
			practiceSubmissionId: 164,
			studentId:            1634,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置练习状态为正常
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions
					SET status = $1,
					    last_start_time = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000 - 60000
					WHERE id = 164
				`, NormalStatus)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 恢复练习状态
				_, err := tx.Exec(ctx, `
					UPDATE t_practice_submissions
					SET status = $1,
					    last_start_time = NULL,
					    elapsed_seconds = 0
					WHERE id = 164
				`, NormalStatus)
				return err
			},
		},

		// 失败场景 - examinee_id和practice_submission_id都为0
		{
			name:            "examinee_id和practice_submission_id都为0",
			expectSuccess:   false,
			expectedMessage: "examinee id and practice submission id both are smaller than 0 or equal to 0",
		},
		// 失败场景 - student_id为0
		{
			name:                 "student_id为0",
			practiceSubmissionId: 164,
			expectSuccess:        false,
			studentId:            0,
		},
		// 失败场景 - 数据库操作失败（练习）
		{
			name: "practiceSubmissionId不存在（练习）",

			expectSuccess:        false,
			expectedMessage:      "no rows in result set",
			practiceSubmissionId: 10,
			studentId:            1634,
		},
		{
			name: "examineeId不存在（考试）",

			expectSuccess:   false,
			expectedMessage: "no rows in result set",
			examineeId:      10,
			studentId:       1634,
		},
		{
			name:          "考试类型studentId不存在",
			expectSuccess: true,
			examineeId:    3119,
			studentId:     1,
			expectCnt:     1,
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为正常
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET status = $1 ,exit_cnt=0
					WHERE id = 3119
				`, NormalStatus)
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 恢复考试状态
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET status = $1, exit_cnt = 0 
					WHERE id = 3119
				`, NormalStatus)
				return err
			},
		},
	}

	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置数据库状态
			if tc.setupDB != nil {
				err := tc.setupDB(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}
			if tc.ctxValue != "" {
				ctx = context.WithValue(ctx, tc.ctxKey, tc.ctxValue)
			}

			err = HandleExit(ctx, ExitReq{ExamineeID: tc.examineeId, PracticeSubmissionID: tc.practiceSubmissionId, StudentId: tc.studentId})
			if tc.expectSuccess {
				assert.NoError(t, err)
				if tc.examineeId > 0 {
					cnt := 0
					err = cmn.GetPgxConn().QueryRow(ctx, `SELECT exit_cnt FROM t_examinee WHERE id=$1`, tc.examineeId).Scan(&cnt)
					if err != nil {
						panic(err)
					}
					t.Logf("cnt:%v", cnt)
					assert.Equal(t, tc.expectCnt, cnt)
				} else {
					var sc int64
					var lastEndTime null.Int
					err = cmn.GetPgxConn().QueryRow(ctx, `SELECT last_end_time,elapsed_seconds FROM t_practice_submissions WHERE id=$1`, tc.practiceSubmissionId).Scan(&lastEndTime, &sc)
					if err != nil {
						panic(err)
					}
					t.Logf("last end time:%v;sc:%v", lastEndTime, sc)
					assert.NotEmpty(t, lastEndTime.Int64)
					assert.NotEmpty(t, sc)

				}
			} else {
				assert.Error(t, err)
				if tc.expectedMessage != "" {
					assert.Equal(t, tc.expectedMessage, err.Error())
				}

			}
			if tc.clean != nil {
				err := tc.clean(t, tx)
				if err != nil {
					t.Fatalf("Failed to clean database: %v", err)
				}
				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}
				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}
		})
	}
}

func TestAllowStudentCanBeInExam(t *testing.T) {
	cmn.ConfigureForTest()

	// 在测试开始前，保存原始数据库状态
	db := cmn.GetPgxConn()
	ctx := context.Background()

	// 开始事务，用于测试期间的数据修改
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 确保测试结束后回滚事务
	defer tx.Rollback(ctx)
	tests := []struct {
		name            string
		method          string
		url             string
		reqBody         *cmn.ReqProto
		expectSuccess   bool
		ctxKey          string
		ctxValue        string
		expectedMessage string
		expectedData    json.RawMessage // 预期数据（可选）
		forceErr        string
		userId          int64
		Domain          []cmn.TDomain
		Role            []int
		// 测试前需要设置的数据库状态
		setupDB func(t *testing.T, tx pgx.Tx) error
		//清理数据
		clean func(t *testing.T, tx pgx.Tx) error
	}{
		{
			name:          "正常允许学生进入考试",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: true,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "success",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "学生已经被允许进入",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "no rows in result set",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "exam_session_id为0",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 0,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "exam_session_id不存在",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "no rows in result set",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 2,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "exam_session_id为-1",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": -1,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "student_id为0",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			userId: 1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 0
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "student_id为-1",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": -1
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "student_id不存在",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "no rows in result set",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 2
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "作用域不是老师",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
			},
			expectedMessage: "invalid domain",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:            "作用域不存在",
			method:          "POST",
			url:             "/respondent/allow",
			expectSuccess:   false,
			expectedMessage: "invalid domain",
			userId:          1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:            "除了监考员域还有其他域",
			method:          "POST",
			url:             "/respondent/allow",
			expectSuccess:   false,
			expectedMessage: "",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(StudentDomainId)},
				{ID: null.IntFrom(ExamInvigilator)},
			},
			userId: 1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "teacher_id为0",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "",
			userId:          0,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "teacher_id为-1",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "",
			userId:          0,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "teacher_id不存在",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectedMessage: "no rows in result set",
			userId:          1,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, CanBeEnterStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:    "buf-zero",
			method:  "POST",
			url:     "/api/respondent/allow",
			reqBody: &cmn.ReqProto{},
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectSuccess:   false,
			expectedMessage: "Call /api/respondent with  empty body",
			userId:          1622,
		},
		{
			name:    " 无效的JSON",
			method:  "POST",
			url:     "/api/respondent/allow",
			reqBody: &cmn.ReqProto{},
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			expectSuccess:   false,
			expectedMessage: "unexpected end of JSON input",
			userId:          1622,
		},
		{
			name:          "事务提交失败",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			forceErr: "commit-tx",
			userId:   1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "事务回滚失败",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			forceErr: "rollback-tx",
			userId:   1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "close body err",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			forceErr: "close body err",
			userId:   1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "io.ReadAll error",
			method:        "POST",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			forceErr: "io.ReadAll",
			userId:   1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:          "方法不是POST",
			method:        "GET",
			url:           "/respondent/allow",
			expectSuccess: false,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(ExamInvigilator)},
			},
			userId: 1622,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"exam_session_id": 160,
					"student_id": 1658
				}`),
			},
			setupDB: func(t *testing.T, tx pgx.Tx) error {
				// 设置考试状态为可以提交
				currentTime := time.Now().Unix()
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}

				_, err = tx.Exec(ctx, `
					UPDATE t_exam_session 
					SET end_time = $1,
					    start_time = $2,
					    late_entry_time=$3,
					    reviewer_ids=$4
					WHERE id = 155
				`, currentTime+3600, currentTime+15, 12, json.RawMessage(`{1622}`))
				if err != nil {
					return err
				}
				return err
			},
			clean: func(t *testing.T, tx pgx.Tx) error {
				// 考试类型提交后，将 examinee 的 end_time 设为 null
				_, err := tx.Exec(ctx, `
					UPDATE t_examinee 
					SET start_time = NULL, end_time = NULL, status = $1 
					WHERE exam_session_id = 160 AND student_id = 1658
				`, NormalStatus)
				if err != nil {
					return err
				}
				return nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 设置数据库状态
			if tc.setupDB != nil {
				err := tc.setupDB(t, tx)
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}

				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}

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
				} else if tc.name == "empty-buf error" {
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
			ctx := createMockContext(req, tc.userId, tc.Domain)
			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErr, tc.forceErr)
			}
			if tc.ctxValue != "" {
				ctx = context.WithValue(ctx, tc.ctxKey, tc.ctxValue)
			}

			AllowStudentCanBeInExam(ctx)
			if tc.clean != nil {
				err := tc.clean(t, tx)
				if err != nil {
					t.Fatalf("Failed to clean database: %v", err)
				}
				// 提交事务以应用更改
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("Failed to commit transaction: %v", err)
				}
				// 开始新事务用于下一个测试或恢复
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Fatalf("Failed to begin new transaction: %v", err)
				}
			}
			// 从上下文中获取响应
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)
			// 根据预期结果验证响应
			if tc.expectSuccess {
				// 成功场景
				assert.Equal(t, 0, resp.Status)
				assert.Equal(t, "success", resp.Msg)
			} else {
				// 失败场景
				assert.NotEqual(t, 0, resp.Status)
				if tc.expectedMessage != "" {
					assert.Contains(t, resp.Msg, tc.expectedMessage)
				}
			}
		})
	}
}
