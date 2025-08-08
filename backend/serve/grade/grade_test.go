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
	//* @apiParam (200) {Number} [teacherID] 教师ID
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
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无效页码，页码不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=abc&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "无效每页数量，页数不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=abc&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "负页码",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=-1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "负每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=-10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无页码",
			method:          "GET",
			url:             "/api/grade/list?category=exam&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&teacherID=-1&submitted=-1",
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
			name:            "无效教师ID，教师ID不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=abc&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "有效教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=1622&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "非法教师ID，教师ID为负数",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-100&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		// --------------------------------------------------
		//               考试成绩列表 - 教师身份 - 考试ID校验     |
		// --------------------------------------------------
		{
			name:            "无效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=abc&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "非法考试ID，为负数",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=-100&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "有效考试ID",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&examID=108&submitted=-1",
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
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&name=math_test&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（未提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=0",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（已提交）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤（不筛选）",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤请求非法，数字非法",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-9",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "提交状态过滤请求非法，为非数字",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "考试类型过滤",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&type=02&submitted=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "空提交状态",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "最大页码和每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1000&pageSize=1000&teacherID=-1&submitted=-1",
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
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "无效页码，页码不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=abc&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "无效每页数量，页数不为数字",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=abc&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          1,
		},
		{
			name:            "负页码",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=-1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "负每页数量",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=-10&teacherID=-1",
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
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=abc",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "非法练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=-100",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "有效练习ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&practiceID=109",
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
			name:            "非法教师ID",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-100&practiceID=109",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
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
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&name=math_test",
			expectSuccess:   true,
			expectedStatus:  0,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "最大页码和每页数量",
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
			name:            "缺少category参数",
			method:          "GET",
			url:             "/api/grade/list?page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "",
			userID:          -1,
		},
		{
			name:            "不支持的类型",
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
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
			userID:          1,
		},
		{
			name:            "conn nil",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn nil",
			userID:          1,
		},
		{
			name:            "conn query fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn query fail",
			userID:          1,
		},
		{
			name:            "conn query fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn query fail",
			userID:          1,
		},
		{
			name:            "rows scan fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "rows scan fail",
			userID:          1,
		},
		{
			name:            "conn.QueryRow fail",
			method:          "GET",
			url:             "/api/grade/list?category=exam&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn.QueryRow fail",
			userID:          1,
		},
		{
			name:            "rows scan fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "rows scan fail",
			userID:          1,
		},
		{
			name:            "conn.QueryRow fail",
			method:          "GET",
			url:             "/api/grade/list?category=practice&page=1&pageSize=10&teacherID=-1&submitted=-1",
			expectSuccess:   false,
			expectedStatus:  -1,
			expectedMessage: "",
			forceError:      "conn.QueryRow fail",
			userID:          1,
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

