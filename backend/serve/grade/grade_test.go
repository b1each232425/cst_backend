package grade

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"w2w.io/cmn"
	"w2w.io/null"
)

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func TestGradeListH(t *testing.T) {
	cmn.ConfigureForTest()

	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		expectSuccess   bool
		expectedStatus  int
		expectedMessage string
		forceError      string
		userID          int64
	}{
		// --------------------------------------------------
		//               考试成绩列表 - 管理员身份               |
		// --------------------------------------------------
		{
			name:            "GET 请求 - 管理员正常获取所有教师考试成绩列表",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "GET 请求 - 无效页码，页码不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=abc&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "GET 请求 - 无效每页数量，页数不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=abc&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "GET 请求 - 负页码",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=-1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 负每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=-10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               考试成绩列表 - 教师身份 - 身份校验       |
		// --------------------------------------------------
		{
			name:            "GET 请求 - 无效教师ID，教师ID不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 有效教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 非法教师ID，教师ID为负数",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-100",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// 考试ID校验
		{
			name:            "GET 请求 - 无效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 非法考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=-100",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 有效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=108",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// 筛选条件校验
		{
			name:            "GET 请求 - 考试名称过滤",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&name=math_test",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 提交状态过滤（未提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=0",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 提交状态过滤（已提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 考试类型过滤",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&type=midterm",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 空提交状态",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 最大页码和每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1000&pageSize=1000&teacherID=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// 练习列表
		{
			name:            "GET 请求 - 获取练习成绩列表",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 无效练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 非法练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=-100",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 有效练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 非法教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-100&practiceID=109",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 有效教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=1574&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 练习名称过滤",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&name=math_test",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 最大页码和每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1000&pageSize=1000&teacherID=-1&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// 共同异常
		{
			name:            "GET 请求 - 缺少category参数",
			method:          "GET",
			url:             "/api/grade/list?page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "GET 请求 - 不支持的类型",
			method:          "GET",
			url:             "/api/grade/list?category=invalid&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "不支持的请求方法",
			method:          "POST",
			url:             "/api/grade/list",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
	}

	// 运行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建请求
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			// 创建模拟上下文
			ctx := createMockContext(t, req, tc.forceError, tc.userID)

			// 执行处理函数
			gradeListH(ctx)

			// 获取响应
			q := cmn.GetCtxValue(ctx)

			t.Logf("q.Msg: %+v", q.Msg)

			// 验证结果
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				//assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
				// assert.NotEmpty(t, q.Msg.Data)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				//assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}

func TestGradeSubmissionH(t *testing.T) {
	cmn.ConfigureForTest()

	// 定义测试用例
	testCases := []struct {
		name            string
		method          string
		url             string
		reqBody         string
		expectSuccess   bool
		expectedStatus  int
		expectedMessage string
		forceError      string
		userID          int64
	}{
		{
			name:            "PATCH 请求 - 提交考试成绩",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "PATCH 请求 - 缺少考试ID",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		}, {
			name:            "不支持的请求方法",
			method:          "GET",
			url:             "/api/grade/submission",
			reqBody:         "",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		}, {
			name:            "无效请求体",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `invalid json`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		}, {
			name:            "PATCH 请求 - 提交考试成绩",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "PATCH 请求 - 缺少考试ID",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "不支持的请求方法",
			method:          "GET",
			url:             "/api/grade/submission",
			reqBody:         "",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无效请求体",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `invalid json`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// New test cases
		{
			name:            "PATCH 请求 - 空考试ID数组",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		//{
		//	name:            "PATCH 请求 - 无效考试ID（非整数）",
		//	method:          "PATCH",
		//	url:             "/api/grade/submission",
		//	reqBody:         `{"data":{"exam_ids":["invalid",2,3]}}`,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "invalid exam_ids: non-integer value",
		//},
		{
			name:            "PATCH 请求 - 缺少data字段",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "PATCH 请求 - 大量考试ID",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,4,5,6,7,8,9,10]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
	}

	// 运行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建请求
			var req *http.Request
			var err error

			if tc.reqBody != "" {
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBufferString(tc.reqBody))
			} else {
				req, err = http.NewRequest(tc.method, tc.url, nil)
			}

			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			// 创建模拟上下文
			ctx := createMockContext(t, req, tc.forceError, tc.userID)

			// 执行处理函数
			gradeSubmissionH(ctx)

			// 获取响应
			q := cmn.GetCtxValue(ctx)

			// 验证结果
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				//assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				//assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}

// 创建模拟上下文
func createMockContext(t *testing.T, req *http.Request, forceError string, userId int64) context.Context {
	ctx := context.Background()

	// 创建响应记录器
	rec := httptest.NewRecorder()

	// 创建服务上下文2
	q := &cmn.ServiceCtx{
		R:         req,
		W:         rec,
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.IntFrom(userId),
		},
		Msg: &cmn.ReplyProto{},
	}

	// 将服务上下文存储到上下文中
	//return context.WithValue(ctx, cmn.QNearKey, q)
	ctx = context.WithValue(context.Background(), cmn.QNearKey, q)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}

	return ctx
}
