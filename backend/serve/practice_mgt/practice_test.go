/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-25 23:20:51
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-01 15:51:21
 */
package practice_mgt

import (
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	"testing"
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

func TestUpsertPracticeH(t *testing.T) {

	if z == nil {
		cmn.ConfigureForTest()
	}
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
		{},
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
