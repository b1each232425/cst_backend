/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-25 23:20:51
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-02 22:00:09
 */
package practice_mgt

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

//func newMockServiceCtx(method string, params map[string]string, bodyBytes []byte) *cmn.ServiceCtx {
//	form := url.Values{}
//	for k, v := range params {
//		form.Set(k, v)
//	}
//
//	r := &http.Request{
//		Method: method,
//		URL:    &url.URL{RawQuery: form.Encode()},
//		Body:   io.NopCloser(bytes.NewReader(bodyBytes)),
//	}
//
//	w := &mockResponseWriter{headers: http.Header{}, body: &bytes.Buffer{}}
//	return &cmn.ServiceCtx{
//		W: w,
//		R: r,
//		Msg: &cmn.ReplyProto{
//			Method: method,
//		},
//		BeginTime: time.Now(),
//		Tag:       make(map[string]interface{}),
//		SysUser: &cmn.TUser{
//			ID: null.IntFrom(testedTeacherID), // 请求用户ID
//		},
//	}
//}

func TestPracticeH(t *testing.T) {

	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()
	var uid int64
	uid = 10086
	now := time.Now().UnixMilli()
	paperID := 22

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
		expectSuccess       bool            // 是否期望成功
		expectedMessage     string          // 预期错误消息
		expectFailedMessage string          // 预期成功消息
		expectedData        json.RawMessage // 预期数据（可选）
		setup               func() error
		Domain              []cmn.TDomain
	}{
		{
			//POST 创建/更新练习数据
			name:   "POST 教师创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Teacher)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 管理员创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Admin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 超级管理员创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 学生创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "please call /api/practice with  http GET method",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 管理员更新练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice":{
						"ID":10086,
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				}
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "please call /api/practice with  http GET method",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", paperID)
				return err
			},
		},
		{
			name: "POST 请求 - empty-buf error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice":{
						"ID":10086,
        				"PaperID": 22,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				}
				}`),
			},
			userId:          1634,
			expectSuccess:   false,
			expectedMessage: "Call /api/practice with  empty body",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.setup != nil {
				err := tc.setup()
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
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
				ctx = context.WithValue(ctx, "test", tc.forceErr)
			}

			practiceH(ctx)
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			if tc.expectSuccess {
				switch tc.method {
				case "POST":
					assert.Equal(t, resp.Msg, "OK")
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
				assert.Equal(t, resp.Status, -1)
				assert.NotEmpty(t, resp.Msg)
				assert.Equal(t, resp.Msg, tc.expectedMessage)
				assert.Empty(t, resp.Data)
			}
			_, err = conn.Exec(context.Background(), `DELETE FROM t_practice WHERE id = 10086;`)
			if err != nil {
				t.Fatal(err)
			}

			_, err = conn.Exec(context.Background(), `DELETE FROM t_practice WHERE name = '练习模拟数据期末考试';`)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
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
			ID: null.IntFrom(userId), // 默认用户ID
		},
		Msg:     &cmn.ReplyProto{},
		Domains: domain,
		Role:    r,
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}
