/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-25 23:20:51
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-25 23:46:30
 */
package practice_mgt

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
