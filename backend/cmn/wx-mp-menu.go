package cmn

import (
	"fmt"
	"github.com/spf13/viper"
	"net/url"
	"strings"
)

func createMxMenuGetCode(redirectURI string) (dstURL string) {
	//https://qnear.cn/xkb/fd
	if redirectURI == "" {
		redirectURI = "/xkb/fd"
	}
	role := "anonymous"

	wxMpAppID := "wx0fefb244eeef3422"
	if viper.IsSet("wxServe.wxMpAppID") {
		wxMpAppID = viper.GetString("wxServe.wxMpAppID")
	}

	host := "qnear.cn"
	if viper.IsSet("webServe.serverName") {
		host = viper.GetString("webServe.serverName")
	}

	// dstURL = fmt.Sprintf("%s://%s/api/wxLogin?role=%s&goto=%s",
	// 	"https", host, role, redirectURI)

	dstURL = fmt.Sprintf("%s://%s%s?role=%s&goto=%s",
		"https", host, redirectURI, role, redirectURI)

	dstURL = strings.ReplaceAll(url.PathEscape(dstURL), "&", "%26")

	base := "https://open.weixin.qq.com/connect/"
	dstURL = base + "oauth2/authorize?appid=" +
		wxMpAppID +
		"&redirect_uri=" + dstURL +
		"&response_type=code&scope=snsapi_userinfo&state=" + wxMpAppID + "#wechat_redirect"

	z.Info(dstURL)
	return
}

const (
	wxMpMenuDefXkbPolicyholer = `{
		"button":[{
			"name":"我要投保",
			"type":"view",
			"url":"%s"
		},{
			"name":"优惠活动",
			"type":"click",
			"key":"promotion"
		},{
			"name":"试用体验",
			"type":"view",
			"url":"%s"
		}]
	}`

	wxMpMenuDefXkbSales = ``

	wxMpMenuDefXkbSchoolAdmin = ``
)

//https://qnear.cn/t/orders
func createWxMenu() {
	if wxMainMpAccessToken.Token == "" {
		z.Error("weChatAccessToken.Token is empty")
		return
	}

	ptn := fmt.Sprintf(wxMpMenuDefXkbPolicyholer,
		createMxMenuGetCode("/xkb/fd"),
		createMxMenuGetCode("/t/orders"))

	z.Info(ptn)
	//urlPtn := `https://api.weixin.qq.com/cgi-bin/menu/create?access_token=%s`

	// orderReq := &fasthttp.Request{}
	// orderReq.SetRequestURI(fmt.Sprintf(urlPtn, wxMpAccessToken.Token))

	// orderReq.SetBody([]byte(ptn))
	// orderReq.Header.SetContentType("application/json;charset=UTF-8")
	// orderReq.Header.SetMethod("POST")
	// resp := &fasthttp.Response{}
	// c := &fasthttp.Client{}
	// err := c.Do(orderReq, resp)
	// if err != nil {
	// 	z.Error(err.Error())
	// 	return
	// }
	// body := resp.Body()
	// type errMsg struct {
	// 	Code int    `json:"errcode,omitempty"`
	// 	Msg  string `json:"errmsg,omitempty"`
	// }
	// var r errMsg
	// err = json.Unmarshal(body, &r)
	// if err != nil {
	// 	z.Error(err.Error())
	// 	return
	// }
	// if r.Code != 0 {
	// 	z.Error(fmt.Sprintf("create wxMpMenu failed: code=%d, msg=%s", r.Code, r.Msg))
	// 	return
	// }
	// z.Info("create wxMpMenu success")
}
