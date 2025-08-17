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

	// 接口参数 需要测试
	//* @apiParam (200) {String} [category] 	成绩类型
	//* @apiParam (200) {Number} [page] 	页码
	//* @apiParam (200) {Number} [pageSize] 每页数量
	//* @apiParam (200) {Number} [examID] 考试ID
	//* @apiParam (200) {Number} [practiceID] 练习ID
	//* @apiParam (200) {String} [name] 考试名称
	//* @apiParam (200) {String} [type] 考试类型
	//* @apiParam (200) {Number} [submitted] 是否已提交

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
			name:            "管理员正常获取所有教师考试成绩列表",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "用户异常",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -2,
		},
		{
			name:            "无效页码，页码不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=abc&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "无效每页数量，页数不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=abc&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "负页码",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=-1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "负每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=-10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无页码",
			method:          "GET",
			url:             "/api/grade/list?category=exam&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               考试成绩列表 - 教师身份 - 身份校验       |
		// --------------------------------------------------

		// --------------------------------------------------
		//               考试成绩列表 - 教师身份 - 考试ID校验     |
		// --------------------------------------------------
		{
			name:            "无效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&examID=abc&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "非法考试ID，为负数",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&examID=-100&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "有效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&examID=15908&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               考试成绩列表 - 教师身份 - 筛选条件校验    |
		// --------------------------------------------------
		{
			name:            "考试名称过滤",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&name=math_test&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（未提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=0",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（已提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（不筛选）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤请求非法，数字非法",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-9",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤请求非法，为非数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "考试类型过滤",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&type=02&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "空提交状态",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "最大页码和每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1000&pageSize=1000&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//               练习成绩列表 - 管理员身份               |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "管理员正常获取所有教师练习成绩列表",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "用户异常",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -2,
		},
		{
			name:            "无效页码，页码不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=abc&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "无效每页数量，页数不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "负页码",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=-1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "负每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=-10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               练习成绩列表 - 练习ID校验               |
		// --------------------------------------------------
		{
			name:            "无效练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&practiceID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "非法练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&practiceID=-100",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "有效练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               练习成绩列表 - 教师ID校验               |
		// --------------------------------------------------
		{
			name:            "有效教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=1574&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               练习成绩列表 - 筛选过滤               |
		// --------------------------------------------------
		{
			name:            "练习名称过滤",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&name=math_test",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "最大页码和每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1000&pageSize=1000&practiceID=109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// 共同异常
		{
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade/list?page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "不支持的类型",
			method:          "GET",
			url:             "/api/grade/list?category=invalid&page=1&pageSize=10",
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
		// --------------------------------------------------
		// --------------------------------------------------
		//            debug环境 force-error测试               |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "用户为空",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "非法请求，鉴权用户失败",
			forceError:      "q.SysUser nil",
			userID:          1,
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
			userID:          1,
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
			userID:          1,
		},
		{
			name:            "conn query fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn query fail",
			userID:          1,
		},
		{
			name:            "conn query fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn query fail",
			userID:          1,
		},
		{
			name:            "rows scan fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "rows scan fail",
			userID:          -1,
		},
		{
			name:            "conn.QueryRow fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn.QueryRow fail",
			userID:          -1,
		},
		{
			name:            "rows scan fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "rows scan fail",
			userID:          -1,
		},
		{
			name:            "conn.QueryRow fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn.QueryRow fail",
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
			name:            "提交考试成绩",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "无效用户",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "examID长度为0",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少考试ID",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          168,
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
			userID:          168,
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
		},
		{
			name:            "缺少考试ID",
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
		{
			name:            "空考试ID数组",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无效考试ID（非整数）",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":["invalid",2,3]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "invalid exam_ids: non-integer value",
		},
		{
			name:            "无效考试ID（非整数）",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[-2,2,3]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "invalid exam_ids",
		},
		{
			name:            "缺少data字段",
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
			name:            "大量考试ID",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,4,5,6,7,8,9,10]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          1622,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//            debug环境 force-error测试               |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "io.ReadAll fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "io.ReadAll fail",
			userID:          168,
		},
		{
			name:            "q.R.Body.Close-fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "q.R.Body.Close-fail",
			userID:          168,
		},
		{
			name:            "conn query fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn query fail",
			userID:          168,
		},
		{
			name:            "rows scan fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "rows scan fail",
			userID:          168,
		},
		{
			name:            "io.ReadAll len(buf)==0",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "io.ReadAll len(buf)==0",
			userID:          168,
		},
		{
			name:            "请求用户空",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "q.SysUser nil",
			userID:          168,
		},
		{
			name:            "setExamGradeSubmitted fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "setExamGradeSubmitted fail",
			userID:          168,
		},
		{
			name:            "setExamGradeSubmitted fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "setExamGradeSubmitted fail",
			userID:          168,
		},
		//{
		//	name:            "exam has not ended yet",
		//	method:          "PATCH",
		//	url:             "/api/grade/submission",
		//	reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "",
		//	forceError:      "exam has not ended yet",
		//	userID:          168,
		//},
		{
			name:            "conn nil",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
			userID:          168,
		},
		{
			name:            "conn begin tx fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn begin tx fail",
			userID:          168,
		},
		{
			name:            "txSuccess must fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "txSuccess must fail",
			userID:          168,
		},
		{
			name:            "tx exec fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "tx exec fail",
			userID:          168,
		},
		{
			name:            "tx commit fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "tx commit fail",
			userID:          168,
		},
		{
			name:            "tx rollback fail",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "tx rollback fail",
			userID:          168,
		},
		{
			name:            "examSession endTime invalid",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "examSession endTime invalid",
			userID:          168,
		},
		{
			name:            "endTime after currTime",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "endTime after currTime",
			userID:          168,
		},
		{
			name:            "examIDs len <= 0",
			method:          "PATCH",
			url:             "/api/grade/submission",
			reqBody:         `{"data":{"exam_ids":[1,2,3,108,160.159,158,157]}}`,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "examIDs len <= 0",
			userID:          168,
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
	//ctx := context.Background()

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
	ctx := context.WithValue(context.Background(), cmn.QNearKey, q)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}
	//return ctx
	return context.WithValue(ctx, "force-error", forceError)
}

func TestGradeDistributionH(t *testing.T) {
	// 测试gradeDistributionH函数
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
		{
			name:            "正常测试考试",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=5",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "正常测试练习",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=2109&columnNum=5",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     列数校验                       |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "columnNum为1",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum为零",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=0",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数无效(columnNum=0)",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum为负数",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=-5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数无效(columnNum=-5)",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum为空",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum为非数字",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数无效",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少columnNum参数",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数为空",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     考试ID校验                     |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "缺少examID参数",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试ID为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examID负数",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=-159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examID无效",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examID为空字符串",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试ID为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examID为非数字",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=abc&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "无效考试ID",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     category校验                  |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "category为未支持类型",
			method:          "GET",
			url:             "/api/grade/distribution?category=unknown&examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade/distribution?examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "类型为空",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     练习ID校验                     |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "练习ID为空",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "练习ID为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习ID为负数",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=-1&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "练习ID无效",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习ID为非数字",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=abc&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "无效练习ID",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少practiceID参数",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "练习ID为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum参数为1",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=2109&columnNum=1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少columnNum参数",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=159",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "columnNum为负数",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=159&columnNum=-5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "列数无效",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     强制触发                       |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "请求方法错误",
			method:          "POST",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "unsupported method: post",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "查询考试成绩分布失败query row fail",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询考试成绩分布失败",
			forceError:      "query row fail",
			userID:          168,
		},
		{
			name:            "查询练习成绩分布失败query row fail",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询练习成绩分布失败",
			forceError:      "query row fail",
			userID:          168,
		},
		{
			name:            "查询练习成绩分布失败pgx.ErrNoRows",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=2109&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "该练习还不能统计分布",
			forceError:      "pgx.ErrNoRows",
			userID:          168,
		},
		{
			name:            "查询考试成绩分布失败pgx.ErrNoRows",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "该考试还不能统计分布",
			forceError:      "pgx.ErrNoRows",
			userID:          168,
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/distribution?category=exam&examID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接失败",
			forceError:      "conn nil",
			userID:          168,
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/distribution?category=practice&practiceID=159&columnNum=5",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接失败",
			forceError:      "conn nil",
			userID:          168,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			ctx := createMockContext(t, req, tc.forceError, tc.userID)
			gradeDistributionH(ctx)

			q := cmn.GetCtxValue(ctx)
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}

func TestGradeExamineeListH(t *testing.T) {
	// 测试gradeExamineeListH函数
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
		{
			name:            "有效考试ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "有效练习ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2109&page=1&pageSize=10",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "不分页有效考试ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=-1&pageSize=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "不分页有效练习ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2109&page=-1&pageSize=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "批量不分页有效考试ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=158,159,160,161,162&page=-1&pageSize=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "批量不分页有效练习ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2107,2109&page=-1&pageSize=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		//                     keyword                      |
		// --------------------------------------------------
		{
			name:            "keyword正常筛选",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10&keyword=张三",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "keyword正常筛选",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2109&page=1&pageSize=10&keyword=张三",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     分页校验                       |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "缺少page参数",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "页码为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "page为空",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "页码为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "page参数为非数字",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=abc&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入无效页码: abc",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "pageSize参数为非数字",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入无效每页数量: abc",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少pageSize参数",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "每页数量为空:",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "pageSize为空",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "每页数量为空:",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		// --------------------------------------------------
		//                     考试ID校验                     |
		// --------------------------------------------------
		// --------------------------------------------------
		{
			name:            "缺少examID参数",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试ID长度小于等于0",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "无效考试ID",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=-1&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入考试ID存在非正整数: -1",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "page为负数",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=-2&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "页码小于等于0(page=-2)",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "pageSize为0",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=0",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "每页数量小于等于0(pageSize=0)",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "page为极大值",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=999999999&page=1&pageSize=10",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examID为非数字字符串",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=abc&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "strconv.ParseInt",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习ID存在非数字",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&page=1&pageSize=10&practiceID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "strconv.ParseInt",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习ID为空",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&page=1&pageSize=10&practiceID=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "练习ID列表为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习page<0",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&page=-2&pageSize=10&practiceID=159",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "页码小于等于0(page=-2)",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "练习pagesize<0",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&page=1&pageSize=-2&practiceID=159",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "每页数量小于等于0",
			forceError:      "",
			userID:          168,
		},
		//	--------------------------------------------------
		//                   类别校验
		// ---------------------------------------------------
		{
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade/examinee/list?examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "类别为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "无效category参数",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=abc&examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "请求方法校验",
			method:          "POST",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "unsupported method",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "考试conn nil",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接为空",
			forceError:      "conn nil",
			userID:          168,
		},
		{
			name:            "练习conn nil",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接为空",
			forceError:      "conn nil",
			userID:          168,
		},
		{
			name:            "考试conn Query fail",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询考试考生成绩列表失败",
			forceError:      "conn Query fail",
			userID:          168,
		},
		{
			name:            "练习conn Query fail",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2109&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询练习考生成绩列表失败",
			forceError:      "conn Query fail",
			userID:          168,
		},
		{
			name:            "考试rows Scan fail",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=exam&examID=159&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "扫描考试考生成绩列表失败",
			forceError:      "rows Scan fail",
			userID:          168,
		},
		{
			name:            "练习rows Scan fail",
			method:          "GET",
			url:             "/api/grade/examinee/list?category=practice&practiceID=2109&page=1&pageSize=10",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询练习考生成绩列表失败",
			forceError:      "rows Scan fail",
			userID:          168,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			ctx := createMockContext(t, req, tc.forceError, tc.userID)

			gradeExamineeListH(ctx)

			q := cmn.GetCtxValue(ctx)
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}

func TestGradeH(t *testing.T) {
	// 测试gradeH函数
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
		{
			name:            "有效考试场次ID",
			method:          "GET",
			url:             "/api/grade?category=exam&examSessionID=201",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "有效练习ID",
			method:          "GET",
			url:             "/api/grade?category=practice&practiceID=2109",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		//                         类型校验
		// --------------------------------------------------
		{
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade?examSessionID=201",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入类别参数为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "无效category参数",
			method:          "GET",
			url:             "/api/grade?category=abc&examSessionID=201",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "category参数为空",
			method:          "GET",
			url:             "/api/grade?category=&examSessionID=201",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入类别参数为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "category参数为空格",
			method:          "GET",
			url:             "/api/grade?category= exam&examSessionID=201",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型",
			forceError:      "",
			userID:          168,
		},
		// --------------------------------------------------
		//                   考试场次ID校验
		// --------------------------------------------------
		{
			name:            "有效考试场次ID",
			method:          "GET",
			url:             "/api/grade?category=exam&examSessionID=201",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少examSessionID参数",
			method:          "GET",
			url:             "/api/grade?category=exam",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examSessionID为非数字",
			method:          "GET",
			url:             "/api/grade?category=exam&examSessionID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID无效, 传入:abc",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少examSessionID参数",
			method:          "GET",
			url:             "/api/grade?category=exam",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "examSessionID为空字符串",
			method:          "GET",
			url:             "/api/grade?category=exam&examSessionID=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "考试场次ID为负数",
			method:          "GET",
			url:             "/api/grade?category=exam&examSessionID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试场次ID 和 练习ID 不能同时为非正整数",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "无效练习ID",
			method:          "GET",
			url:             "/api/grade?category=practice&practiceID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "传入practiceID无效: abc",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "缺少practiceID参数",
			method:          "GET",
			url:             "/api/grade?category=practice",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "practiceID为空",
			forceError:      "",
			userID:          168,
		},
		{
			name:            "方法无效",
			method:          "POST",
			url:             "/api/grade?category=exam&examSessionID=201",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "unsupported method: post",
			forceError:      "",
			userID:          168,
		},
		//	--------------------------------------------------
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade?category=practice&practiceID=2109",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接为空",
			forceError:      "conn nil",
			userID:          168,
		},
		//{
		//	name:            "conn begin tx fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "开启事务失败",
		//	forceError:      "conn begin tx fail",
		//	userID:          168,
		//},
		//{
		//	name:            "txSuccess must fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "回滚事务失败",
		//	forceError:      "txSuccess must fail",
		//	userID:          168,
		//},
		//{
		//	name:            "tx commit fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "查询成绩分析失败",
		//	forceError:      "tx commit fail",
		//	userID:          168,
		//},
		//{
		//	name:            "conn tx rollback",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "success",
		//	forceError:      "conn tx rollback",
		//	userID:          168,
		//},
		{
			name:            "ep conn QueryRow fail",
			method:          "GET",
			url:             "/api/grade?category=practice&practiceID=2109",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取考卷ID失败",
			forceError:      "ep conn QueryRow fail",
			userID:          168,
		},
		{
			name:            "LoadExamPaperDetailsById fail",
			method:          "GET",
			url:             "/api/grade?category=practice&practiceID=2109",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "LoadExamPaperDetailsById fail",
			userID:          168,
		},
		//{
		//	name:            "sa conn Query fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生答案失败",
		//	forceError:      "sa conn Query fail",
		//	userID:          168,
		//},
		//{
		//	name:            "qas rows Scan fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=exam&examSessionID=201",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生答案失败",
		//	forceError:      "qas rows Scan fail",
		//	userID:          168,
		//},
		//{
		//	name:            "Unmarshal fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "反序列化学生答案失败",
		//	forceError:      "Unmarshal fail",
		//	userID:          168,
		//},
		//{
		//	name:            "ansType not match",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "success",
		//	forceError:      "ansType not match",
		//	userID:          168,
		//},
		//{
		//	name:            "ansJson nil",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "success",
		//	forceError:      "ansJson nil",
		//	userID:          168,
		//},
		//{
		//	name:            "sjs conn Query fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "获取考卷平均分失败",
		//	forceError:      "sjs conn Query fail",
		//	userID:          168,
		//},
		//{
		//	name:            "sjs rows Scan fail",
		//	method:          "GET",
		//	url:             "/api/grade?category=practice&practiceID=2109",
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "获取考卷平均分失败",
		//	forceError:      "sjs conn Query fail",
		//	userID:          168,
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			ctx := createMockContext(t, req, tc.forceError, tc.userID)

			gradeH(ctx)

			q := cmn.GetCtxValue(ctx)
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}

func TestGradeSH(t *testing.T) {
	testCases := []struct {
		name            string
		method          string
		url             string
		userID          int64
		expectSuccess   bool
		expectedStatus  int
		expectedMessage string
		forceError      string
	}{
		// ------------------ 正向场景 ------------------
		{
			name:            "有效考试场次ID",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
		},
		{
			name:            "有效练习ID",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109",
			userID:          1684,
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "success",
			forceError:      "",
		},
		// ------------------ 类型校验 ------------------
		{
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade/s?examSessionID=201",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型:",
			forceError:      "",
		},
		{
			name:            "无效category参数",
			method:          "GET",
			url:             "/api/grade/s?category=unknown&examSessionID=201",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "不支持的类型:",
			forceError:      "",
		},
		// ------------------ 考试场次ID校验 ------------------
		{
			name:            "缺少examSessionID参数",
			method:          "GET",
			url:             "/api/grade/s?category=exam",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID为空",
			forceError:      "",
		},
		{
			name:            "examSessionID为非数字",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=abc",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "examSessionID无效: 0",
			forceError:      "",
		},
		{
			name:            "examSessionID负数",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=-1",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试场次ID无效",
			forceError:      "",
		},
		{
			name:            "studentID负数",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=201&studentID=-1",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "学生ID无效",
			forceError:      "",
		},
		{
			name:            "studentID用户负数",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=201",
			userID:          -2,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "学生ID无效",
			forceError:      "",
		},
		// ------------------ 练习ID校验 ------------------
		{
			name:            "缺少practiceID参数",
			method:          "GET",
			url:             "/api/grade/s?category=practice",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "practiceID为空",
			forceError:      "",
		},
		{
			name:            "practiceID为非数字",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=abc",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "practiceID无效: abc",
			forceError:      "",
		},
		{
			name:            "practiceID负数",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=-1",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "练习ID无效",
			forceError:      "",
		},
		{
			name:            "studentID负数",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109&studentID=-1",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "学生ID无效",
			forceError:      "",
		},
		// ------------------ 方法校验 ------------------
		{
			name:            "POST方法",
			method:          "POST",
			url:             "/api/grade/s?category=exam&examSessionID=201",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
		},
		//	------ 强制校验 ------
		{
			name:            "q.SysUser nil",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=201",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "q.SysUser nil",
		},
		{
			name:            "studentID",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=201&studentID=1675",
			userID:          1,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
		},
		{
			name:            "studentID非数字",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=201&studentID=abc",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "无效学生ID:",
			forceError:      "",
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
		},
		{
			name:            "conn begin tx fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "开启事务失败",
			forceError:      "conn begin tx fail",
			userID:          1675,
		},
		//{
		//	name:            "txSuccess must fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=exam&examSessionID=198",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "回滚事务失败",
		//	forceError:      "txSuccess must fail",
		//	userID:          1675,
		//},
		//{
		//	name:            "tx commit fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=exam&examSessionID=198",
		//	expectSuccess:   true,
		//	expectedStatus:  0,
		//	expectedMessage: "查询学生成绩",
		//	forceError:      "tx commit fail",
		//	userID:          1675,
		//},
		//{
		//	name:            "conn begin tx fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=exam&examSessionID=198",
		//	userID:          1675,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "",
		//	forceError:      "conn begin tx fail",
		//},
		{
			name:            "get score exam fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "get score exam fail",
			forceError:      "get score exam fail",
		},
		{
			name:            "get score practice fail",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "get score practice fail",
			forceError:      "get score practice fail",
		},
		//	 ------ 强制校验 ------
		{
			name:            "dao conn nil",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接为空",
			forceError:      "dao conn nil",
		},
		{
			name:            "LoadExamPaperDetailByUserId fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "调用LoadExamPaperDetailByUserId失败",
			forceError:      "LoadExamPaperDetailByUserId fail",
		},
		{
			name:            "rks conn Query fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询考试场次成绩失败",
			forceError:      "rks conn Query fail",
		},
		{
			name:            "rks rows Scan fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "扫描考试场次成绩失败",
			forceError:      "rks rows Scan fail",
		},
		{
			name:            "ei conn Query fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询考试场次ID失败",
			forceError:      "ei conn Query fail",
		},
		{
			name:            "esi conn Query fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询考试场次失败",
			forceError:      "esi conn Query fail",
		},
		{
			name:            "esi rows Scan fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "扫描考试场次失败",
			forceError:      "esi rows Scan fail",
		},
		{
			name:            "ansNum conn Query fail",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "学生ID不能为空",
			forceError:      "ansNum conn Query fail",
		},
		{
			name:            "ansTime invalid",
			method:          "GET",
			url:             "/api/grade/s?category=exam&examSessionID=198",
			userID:          1675,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "考试场次时间为空",
			forceError:      "ansTime invalid",
		},
		//	---- practice ----
		{
			name:            "dao conn nil",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "获取数据库连接为空",
			forceError:      "dao conn nil",
		},
		{
			name:            "ep conn QueryRow fail",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "查询练习学生试卷ID失败",
			forceError:      "ep conn QueryRow fail",
		},
		{
			name:            "LoadExamPaperDetailByUserId fail",
			method:          "GET",
			url:             "/api/grade/s?category=practice&practiceID=2109",
			userID:          1684,
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "调用LoadExamPaperDetailByUserId失败",
			forceError:      "LoadExamPaperDetailByUserId fail",
		},
		//{
		//	name:            "ansNum conn QueryRow fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=practice&practiceID=2109",
		//	userID:          1684,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生答案失败",
		//	forceError:      "ansNum conn QueryRow fail",
		//},
		//{
		//	name:            "duration conn QueryRow fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=practice&practiceID=2109",
		//	userID:          1684,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生建议时长失败",
		//	forceError:      "duration conn QueryRow fail",
		//},
		//{
		//	name:            "usedTime conn QueryRow fail",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=practice&practiceID=2109",
		//	userID:          1684,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生使用时长失败",
		//	forceError:      "usedTime conn QueryRow fail",
		//},
		//{
		//	name:            "usedTime pgx.ErrNoRows",
		//	method:          "GET",
		//	url:             "/api/grade/s?category=practice&practiceID=2109",
		//	userID:          1684,
		//	expectSuccess:   false,
		//	expectedStatus:  -1,
		//	expectedMessage: "查询学生使用时长失败",
		//	forceError:      "usedTime pgx.ErrNoRows",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			ctx := createMockContext(t, req, tc.forceError, tc.userID)

			gradeSH(ctx)

			q := cmn.GetCtxValue(ctx)
			if tc.expectSuccess {
				assert.Equal(t, tc.expectedStatus, q.Msg.Status)
				assert.Equal(t, tc.expectedMessage, q.Msg.Msg)
			} else {
				assert.NotEqual(t, 0, q.Msg.Status)
				assert.Contains(t, q.Msg.Msg, tc.expectedMessage)
			}
		})
	}
}
