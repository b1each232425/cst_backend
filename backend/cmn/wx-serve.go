package cmn

//https://mp.weixin.qq.com/debug/

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"w2w.io/null"
)

type cdata struct {
	D string `xml:",cdata"`
}

/*
	****************************** *

微信用户

	登录回调(扫码后): /api/wxLogin
	支付后回调: /api/wxPaid

* ******************************
*/
type wxMsg struct {
	XMLName      xml.Name `json:"-" xml:"xml"`
	ToUserName   *cdata   `json:"ToUserName,omitempty" xml:"ToUserName,omitempty"`
	FromUserName *cdata   `json:"FromUserName,omitempty" xml:"FromUserName,omitempty"`
	CreateTime   int64    `json:"CreateTime,omitempty" xml:"CreateTime,omitempty"`

	MsgType *cdata `json:"MsgType,omitempty" xml:"MsgType,omitempty"` //link,location,shortvideo,video,voice,image,text

	// MsgType=="text"
	Content *cdata `json:"Content,omitempty" xml:"Content,omitempty"`
	PicURL  *cdata `json:"PicUrl,omitempty" xml:"PicUrl,omitempty"`

	// MsgType=="image"
	MediaID *cdata `json:"MediaId,omitempty" xml:"MediaId,omitempty"`

	Format *cdata `json:"Format,omitempty" xml:"Format,omitempty"` //voice: amr

	Recognition  *cdata `json:"Recognition,omitempty" xml:"Recognition,omitempty"`
	ThumbMediaID *cdata `json:"ThumbMediaId,omitempty" xml:"ThumbMediaId,omitempty"`

	LocationX string `json:"Location_X,omitempty" xml:"Location_X,omitempty"`
	LocationY string `json:"Location_Y,omitempty" xml:"Location_Y,omitempty"`

	Scale int    `json:"Scale,omitempty" xml:"Scale,omitempty"`
	Label *cdata `json:"Label,omitempty" xml:"Label,omitempty"`

	Event    *cdata `json:"Event,omitempty" xml:"Event,omitempty"`
	EventKey *cdata `json:"EventKey,omitempty" xml:"EventKey,omitempty"`

	Latitude  string `json:"Latitude,omitempty" xml:"Latitude,omitempty"`
	Longitude string `json:"Longitude,omitempty" xml:"Longitude,omitempty"`
	Precision string `json:"Precision,omitempty" xml:"Precision,omitempty"`

	// MsgType=="music"
	MusicURL   *cdata `json:"MusicURL,omitempty" xml:"MusicURL,omitempty"`
	HQMusicURL *cdata `json:"HQMusicUrl,omitempty" xml:"HQMusicUrl,omitempty"`

	Ticket *cdata `json:"Ticket,omitempty" xml:"Ticket,omitempty"`

	// MsgType=="news"
	ArticleCount int    `json:"ArticleCount,omitempty" xml:"ArticleCount,omitempty"`
	Articles     *cdata `json:"Articles,omitempty" xml:"Articles,omitempty"`

	Title       *cdata `json:"Title,omitempty" xml:"Title,omitempty"`
	Description *cdata `json:"Description,omitempty" xml:"Description,omitempty"`
	URL         *cdata `json:"Url,omitempty" xml:"Url,omitempty"`
	MsgID       int64  `json:"MsgId,omitempty" xml:"MsgId,omitempty"`
}

type weChatMsg struct {
	XMLName      xml.Name `json:"-" xml:"xml"`
	ToUserName   string   `json:"ToUserName,omitempty" xml:"ToUserName,CDATA,omitempty"`
	FromUserName string   `json:"FromUserName,omitempty" xml:"FromUserName,CDATA,omitempty"`
	CreateTime   int64    `json:"CreateTime,omitempty" xml:"CreateTime,omitempty"`

	MsgType string `json:"MsgType,omitempty" xml:"MsgType,CDATA,omitempty"` //link,location,shortvideo,video,voice,image,text

	// MsgType=="text"
	Content string `json:"Content,omitempty" xml:"Content,CDATA,omitempty"`
	PicURL  string `json:"PicUrl,omitempty" xml:"PicUrl,CDATA,omitempty"`

	// MsgType=="image"
	MediaID string `json:"MediaId,omitempty" xml:"MediaId,CDATA,omitempty"`

	Format string `json:"Format,omitempty" xml:"Format,CDATA,omitempty"` //voice: amr

	Recognition  string `json:"Recognition,omitempty" xml:"Recognition,CDATA,omitempty"`
	ThumbMediaID string `json:"ThumbMediaId,omitempty" xml:"ThumbMediaId,CDATA,omitempty"`

	LocationX string `json:"Location_X,omitempty" xml:"Location_X,omitempty"`
	LocationY string `json:"Location_Y,omitempty" xml:"Location_Y,omitempty"`

	Scale int    `json:"Scale,omitempty" xml:"Scale,omitempty"`
	Label string `json:"Label,omitempty" xml:"Label,CDATA,omitempty"`

	Event    string `json:"Event,omitempty" xml:"Event,CDATA,omitempty"`
	EventKey string `json:"EventKey,omitempty" xml:"EventKey,CDATA,omitempty"`

	Latitude  string `json:"Latitude,omitempty" xml:"Latitude,omitempty"`
	Longitude string `json:"Longitude,omitempty" xml:"Longitude,omitempty"`
	Precision string `json:"Precision,omitempty" xml:"Precision,omitempty"`

	// MsgType=="music"
	MusicURL   string `json:"MusicURL,omitempty" xml:"MusicURL,CDATA,omitempty"`
	HQMusicURL string `json:"HQMusicUrl,omitempty" xml:"HQMusicUrl,CDATA,omitempty"`

	Ticket string `json:"Ticket,omitempty" xml:"Ticket,CDATA,omitempty"`

	// MsgType=="news"
	ArticleCount int    `json:"ArticleCount,omitempty" xml:"ArticleCount,omitempty"`
	Articles     string `json:"Articles,omitempty" xml:"Articles,CDATA,omitempty"`

	Title       string `json:"Title,omitempty" xml:"Title,CDATA,omitempty"`
	Description string `json:"Description,omitempty" xml:"Description,CDATA,omitempty"`
	URL         string `json:"Url,omitempty" xml:"Url,CDATA,omitempty"`
	MsgID       int64  `json:"MsgId,omitempty" xml:"MsgId,omitempty"`
}

var wxOpenPageToken wxPageAccessToken
var wxMxPageToken wxPageAccessToken

var wxMpServeKeyToken = ""

func verifyWxSign(ctx context.Context) bool {
	q := GetCtxValue(ctx)
	servSign := q.R.URL.Query().Get("signature")
	timestamp := q.R.URL.Query().Get("timestamp")
	nonce := q.R.URL.Query().Get("nonce")
	if servSign == "" || timestamp == "" || nonce == "" {
		z.Warn("missing signature/timestamp/nonce from wxServe server push")
		return false
	}
	c := []string{wxMpServeKeyToken, timestamp, nonce}
	sort.Slice(c, func(i, j int) bool {
		return c[i] < c[j]
	})

	var b strings.Builder
	for i := 0; i < len(c); i++ {
		b.WriteString(c[i])
	}

	s := sha1.New()
	io.WriteString(s, b.String())
	sign := fmt.Sprintf("%x", s.Sum(nil))
	if sign == servSign {
		z.Info("signature verified")
		return true
	}

	z.Error("signature mismatch")
	return false
}

func showMsg(w http.ResponseWriter, r *http.Request, buf []byte) {
	z.Info(string(buf))
	var m weChatMsg
	err := xml.Unmarshal(buf, &m)
	if err != nil {
		z.Warn(err.Error())
		return
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		z.Info(err.Error())
		return
	}
	z.Info("received msg:\n" + string(b))

	switch m.MsgType {
	case "text", "video", "voice", "news", "image", "music", "shortvideo", "location", "link", "event":
		replyWxMsg(w, r, &m)
	default:
		fmt.Fprintf(w, "success")
		return
	}
}

func replyWxMsg(w http.ResponseWriter, r *http.Request, msg *weChatMsg) {
	var m wxMsg
	m.FromUserName = &cdata{D: "gh_b3803b26634b"}
	m.ToUserName = &cdata{D: msg.FromUserName}
	m.CreateTime = GetNowInMS()
	m.MsgType = &cdata{D: "text"}
	m.Content = &cdata{D: "这是一个美好的开始"}
	buf, err := xml.Marshal(m)
	if err != nil {
		z.Info(err.Error())
		return
	}
	z.Info("\n" + string(buf))

	fmt.Fprintf(w, "%s", string(buf))
}

type wxCallbackStatus interface {
	getErrCode() int
	getErrMsg() string
}
type wxPageAccessToken struct {
	AcccessToken string `json:"access_token,omitempty"`
	ExpireIn     int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	OpenID       string `json:"openid,omitempty"`
	Scope        string `json:"scope,omitempty"`

	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

func (s *wxPageAccessToken) getErrCode() int {
	return s.ErrCode
}
func (s *wxPageAccessToken) getErrMsg() string {
	return s.ErrMsg
}

type wxUser struct {
	origin string // 应用来源, mp: 公众号, open: 开放平台

	Subscribe     int   `json:"subscribe,omitempty"`
	SubscribeTime int64 `json:"subscribe_time,omitempty"`

	OpenID    string `json:"openid,omitempty" storm:"id"`
	UnionID   string `json:"unionid,omitempty"`
	GroupID   int    `json:"groupid,omitempty"`
	TagIDList []int  `json:"tagid_list,omitempty"`

	Nickname   string `json:"nickname,omitempty"`
	Sex        int    `json:"sex,omitempty"` //值为1时是男性，值为2时是女性，值为0时是未知
	Language   string `json:"language,omitempty"`
	City       string `json:"city,omitempty"`
	Province   string `json:"province,omitempty"`
	Country    string `json:"country,omitempty"`
	HeadimgURL string `json:"headimgurl,omitempty"`

	Privilege []string `json:"privilege,omitempty"`
	Remark    string   `json:"remark,omitempty"`

	SubscribeScene string `json:"subscribe_scene,omitempty"` /* 用户关注的渠道来源，
	ADD_SCENE_SEARCH 公众号搜索，
	ADD_SCENE_ACCOUNT_MIGRATION 公众号迁移，
	ADD_SCENE_PROFILE_CARD 名片分享，
	ADD_SCENE_QR_CODE 扫描二维码，
	ADD_SCENEPROFILE LINK 图文页内名称点击，
	ADD_SCENE_PROFILE_ITEM 图文页右上角菜单，
	ADD_SCENE_PAID 支付后关注，
	ADD_SCENE_OTHERS 其他 */

	QRScene    int    `json:"qr_scene,omitempty"`
	QRSceneStr string `json:"qr_scene_str,omitempty"`

	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

func (s *wxUser) getErrCode() int {
	return s.ErrCode
}
func (s *wxUser) getErrMsg() string {
	return s.ErrMsg
}

var wxMainMpAccessToken wxMpServeToken

type wxMpServeToken struct {
	Token    string `json:"access_token,omitempty"`
	ExpireIn int64  `json:"expires_in,omitempty"`
	ErrCode  int    `json:"errcode,omitempty"`
	ErrMsg   string `json:"errmsg,omitempty"`
}

// Transports should be reused instead of created as needed. 大量重复创建可能导致内存泄漏
var httpTransport *http.Transport

func getWxAccessToken(seconds int) bool {

	//https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=APPSECRET
	wxMpAppID := "wx0fefb244eeef3422"
	wxMpAppSecret := ""

	if viper.IsSet("wxServe.wxMpAppID") {
		wxMpAppID = viper.GetString("wxServe.wxMpAppID")
		z.Info("wxServe.wxMpAppID settle")
	}
	if viper.IsSet("wxServe.wxMpAppSecret") {
		wxMpAppSecret = viper.GetString("wxServe.wxMpAppSecret")
		z.Info("wxServe.wxMpAppSecret settle")
	}

	tokenURL := fmt.Sprintf(`https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%v&secret=%v`,
		wxMpAppID, wxMpAppSecret)

	var (
		proxyPort int
		proxyServ string
		proxyCert string
		proxyUser string

		wxTokenWithProxy bool

		mustGetWxMpAcsToken bool
	)

	if viper.IsSet("wxServe.mustGetWxMpAcsToken") {
		mustGetWxMpAcsToken = viper.GetBool("wxServe.mustGetWxMpAcsToken")
	}
	if viper.IsSet("proxy.port") {
		proxyPort = viper.GetInt("proxy.port")
	}

	if viper.IsSet("proxy.server") {
		proxyServ = viper.GetString("proxy.server")
	}
	if viper.IsSet("proxy.pwd") {
		proxyCert = viper.GetString("proxy.pwd")
	}

	if viper.IsSet("proxy.user") {
		proxyUser = viper.GetString("proxy.user")
	}
	if viper.IsSet("wxServe.wxTokenWithProxy") {
		wxTokenWithProxy = viper.GetBool("wxServe.wxTokenWithProxy")
		if wxTokenWithProxy && (proxyPort == 0 || proxyServ == "" ||
			proxyCert == "" || proxyUser == "") {
			z.Error("invalid proxy.server/port/user/pwd")
			os.Exit(-1)
		}
	}

	var data []byte
	var err error
	if wxTokenWithProxy {
		proxyEP := fmt.Sprintf("%s:%d", proxyServ, proxyPort)
		proxyURL := &url.URL{Host: proxyEP, Scheme: "http"}

		if httpTransport == nil {
			httpTransport = &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}

		client := &http.Client{Transport: httpTransport}

		req, err := http.NewRequest("GET", tokenURL, nil)
		if err != nil {
			z.Error(err.Error())
			return false
		}

		auth := fmt.Sprintf("%s:%s", proxyUser, proxyCert) //"mickey:142857"
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

		req.Header.Add("Proxy-Authorization", basicAuth)
		httpTransport.ProxyConnectHeader = req.Header

		resp, err := client.Do(req)
		if err != nil {
			z.Error(err.Error())
			return false
		}
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			z.Error(err.Error())
			return false
		}
		defer resp.Body.Close()

	} else {
		var status int
		status, data, err = fasthttp.Get(nil, tokenURL)
		if err != nil {
			z.Warn(err.Error())
			return false
		}
		if status != 200 {
			z.Warn(fmt.Sprintf("status is %v", status))
			return false
		}
	}

	err = json.Unmarshal(data, &wxMainMpAccessToken)
	if err != nil {
		z.Warn(err.Error())
		return false
	}

	if wxMainMpAccessToken.ErrCode != 0 || wxMainMpAccessToken.Token == "" {
		z.Error(fmt.Sprintf("appID: %s, errcode: %d, %s", wxMpAppID,
			wxMainMpAccessToken.ErrCode, wxMainMpAccessToken.ErrMsg))
		m := "failed to get wxAccessToken, please add this host IP to mp.weixin.qq.com's whiteIPList"
		if mustGetWxMpAcsToken {
			z.Error(m)
			os.Exit(-1)
		}
		z.Warn(m)

		t := time.NewTimer(time.Duration(2 * seconds * int(time.Second)))
		go func() {
			<-t.C
			getWxAccessToken(2 * seconds) // 120 seconds
		}()
		return false
	}

	if wxMainMpAccessToken.ExpireIn <= 60 {
		t := fmt.Sprintf("weChatAccessToken.ExpireIn is %d seconds are too small then 60", wxMainMpAccessToken.ExpireIn)
		z.Info(t)
		return true
	}

	t := time.NewTimer(time.Second * time.Duration(wxMainMpAccessToken.ExpireIn/2))
	go func() {
		<-t.C
		getWxAccessToken(2) // 120 seconds
	}()

	//z.Info(fmt.Sprintf("%s", wxMainMpAccessToken.Token))
	//createWxMenu()

	tokenURL = fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/getcallbackip?access_token=%s",
		wxMainMpAccessToken.Token)

	status, body, err := fasthttp.Get(nil, tokenURL)
	if err != nil {
		z.Warn(err.Error())
		return false
	}
	if status != 200 {
		err = fmt.Errorf(fmt.Sprintf("status code is %v", status))
		z.Warn(err.Error())
		return false
	}
	if len(body) < 32 {
		err = fmt.Errorf(fmt.Sprintf("body's length is %v which too short", len(body)))
		z.Warn(err.Error())
		return false
	}

	//z.Info(string(body))
	err = json.Unmarshal(body, &wxServIPs)
	if err != nil {
		z.Warn(err.Error())
		return false
	}
	//fmt.Println(wxServIPs)

	return true
}

type wxServIPList struct {
	IPList []string `json:"ip_list,omitempty"`
}

var wxServIPs wxServIPList

func wxServeAPICall(object, issue, openID string) bool {
	//user/info?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
	const base = "https://api.weixin.qq.com/cgi-bin"
	url := fmt.Sprintf(`%v/%v/%v?access_token=%v&openid=%v&lang=zh_CN`,
		base, object, issue, wxMainMpAccessToken.Token, openID)

	status, body, err := fasthttp.Get(nil, url)
	if err != nil {
		z.Warn(err.Error())
		return false
	}
	if status != 200 {
		z.Warn(fmt.Sprintf("status is %v", status))
		return false
	}
	switch {
	case object == "user" && issue == "info":
		var u wxUser
		err = json.Unmarshal(body, &u)
		if err != nil {
			z.Warn(err.Error())
			return false
		}

		z.Info(u.Nickname)
	}

	return true
}

var verifyBusinessDomainFileName = "/MP_verify_NoNLb44EuoLJ7ybT.txt"
var verifyBusinessDomainFileContent = "NoNLb44EuoLJ7ybT"

func wxVerifyDomain(ctx context.Context) {

	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	n := 0
	n, q.Err = fmt.Fprintf(q.W, "%s", verifyBusinessDomainFileContent)
	q.Responded = true
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if n != len(verifyBusinessDomainFileContent) {
		q.Err = fmt.Errorf("message length mismatch while send verify Bussiness Domain File Content")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("successfully reply to mp.weixin.qq.com for verify bussiness domain valid query")
	q.Responded = true
	q.Stop = true
	return
}

func wxVerifyXkbDomain(ctx context.Context) {

	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	n := 0
	n, q.Err = fmt.Fprintf(q.W, "%s", "87DVhsMdnS64dC0K")
	q.Responded = true
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if n != len(verifyBusinessDomainFileContent) {
		q.Err = fmt.Errorf("message length mismatch while send verify Bussiness Domain File Content")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("successfully reply to mp.weixin.qq.com for verify bussiness domain valid query")
	q.Responded = true
	q.Stop = true
	return
}
func wxVerifyTaiHeDomain(ctx context.Context) {

	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	n := 0
	n, q.Err = fmt.Fprintf(q.W, "%s", "yUDgQ4aLb7y2Orxw")
	q.Responded = true
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if n != len(verifyBusinessDomainFileContent) {
		q.Err = fmt.Errorf("message length mismatch while send verify Bussiness Domain File Content")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("successfully reply to mp.weixin.qq.com for verify bussiness domain valid query")
	q.Responded = true
	q.Stop = true
	return
}

// var wxServeURI = `(?i)^/api/wxServe(/.*)?$`
// var rWxServeURI = regexp.MustCompile(wxServeURI)
// var wxServeURILen = len(wxServeURI)

func wxServe(ctx context.Context) {

	q := GetCtxValue(ctx)

	// if (len(q.R.URL.Path) + 12) < wxServeURILen {
	// 	return
	// }

	z.Info("---->" + FncName())
	q.Stop = true

	// http://127.0.0.1:1080/wxServe?signature=050f9cfc8f71c0791e60492f1663185833e558d3&timestamp=1548569535&nonce=1891115665&echostr=4407590243533708287

	if !verifyWxSign(ctx) {
		q.Err = fmt.Errorf("invalid open.weixin.qq.com signature")
		return
	}

	echostr := q.R.URL.Query().Get("echostr")
	if echostr != "" {
		// 微信校验咱们的服务器
		fmt.Fprintf(q.W, "%s", echostr)
		return
	}

	openID := q.R.URL.Query().Get("openid")
	if openID != "" {
		z.Info("we have openID: " + openID)
	}

	switch strings.ToLower(q.R.Method) {
	case "post":
		// Message from mp.weixin.qq.com
		var buf []byte
		buf, q.Err = ioutil.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			break
		}
		defer q.R.Body.Close()
		showMsg(q.W, q.R, buf)
	}

	return
}

/*
Status
获取当前授权状态
接口: /api/status
参数: 无
授权: 非授权
返回结果样例
1)未登录返回

	{
	  "status": 0,
	  "API": "/api/status",
	  "method": "GET",
	  "data": {
	    "wxUserIsValid": false,
	    "sysUserIsValid": false,
	    "wxOpenAppID": "wxbbcdc7faf43cecec",
	    "wxMpAppID": "wx0fefb244eeef3422",
	    "serverName": "qnear.cn",
	    "appType": {
	      "name": "pcBrowser",
	      "id": 1
	    },
	    "authed": false,
	    "user": {}
	  }
	}

2.登录返回

	{
	  "data": {
	    "wxUserIsValid": false,
	    "sysUserIsValid": true,
	    "wxOpenAppID": "wxbbcdc7faf43cecec",
	    "wxMpAppID": "wx0fefb244eeef3422",
	    "serverName": "qnear.cn",
	    "appType": {
	      "name": "pcBrowser",
	      "id": 1
	    },
	    "authed": true,
	    "user": {
	      "ID": 1002,
	      "Category": "system",
	      "Type": "02",
	      "MobilePhone": "13710503433",
	      "Email": "dawnfire@126.com",
	      "Account": "mickey"
	    }
	  }
	}

说明:
authed: 是否登录
user:如果已登录，则是用户信息，未登录是{}
wxUserIsValid: 是否有微信帐号相关信息（微信登录后才有）
*/
func Status(ctx context.Context) {
	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())
	q.Stop = true

	q.W.Header().Add("Access-Control-Allow-Origin", q.R.Header.Get("Origin"))
	q.W.Header().Add("Access-Control-Allow-Credentials", "true")

	wxMpAppID := "wx0fefb244eeef3422"
	if viper.IsSet("wxServe.wxMpAppID") {
		wxMpAppID = viper.GetString("wxServe.wxMpAppID")
	}

	wxOpenAppID := "wxbbcdc7faf43cecec"
	if viper.IsSet("wxServe.wxOpenAppID") {
		wxOpenAppID = viper.GetString("wxServe.wxOpenAppID")
	}
	serverName := "qnear.cn"
	if viper.IsSet("webServe.serverName") {
		serverName = viper.GetString("webServe.serverName")
	}
	var wxUserValid, sysUserValid bool
	var headimgURL string
	if q.WxUser != nil {
		wxUserValid = true
		headimgURL = q.WxUser.HeadImgURL.String
	}
	var buf []byte
	if q.SysUser != nil {
		sysUserValid = true

		t := q.SysUser.UserToken
		q.SysUser.UserToken = null.NewString("", false)
		buf, q.Err = MarshalJSON(q.SysUser)
		q.SysUser.UserToken = t
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}
	if len(buf) == 0 {
		buf = []byte("{}")
	}

	//---------
	var domain []byte
	domain, q.Err = json.Marshal(q.DomainList)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	authenticated, _ := q.Session.Values["Authenticated"].(bool)
	url := getWxLoginURL(ctx)
	s := fmt.Sprintf(
		`{
			"wxLoginURL":"%s",
			"wxHeadImgURL":"%s",
			"wxUserIsValid":%v,
			"sysUserIsValid":%v,
			"wxOpenAppID":"%s",
			"wxMpAppID":"%s",
			"serverName":"%s",
			"serverTime":%d,
			"appType":{
				"name":"%s",
				"id":%d
			},
			"dbStats":%v,
			"authed":%v,
			"user":%s,
			"domain":%s,
			"runtime":"%s"
			}`,
		url, headimgURL,
		wxUserValid, sysUserValid,
		wxOpenAppID, wxMpAppID, serverName, time.Now().Unix()*1000,
		GetCallerTypeName(q.CallerType),
		q.CallerType,
		DbState(nil),
		authenticated, string(buf), string(domain), runtime.GOOS,
	)

	q.Msg.Data = types.JSONText(s)
	q.Resp()
}
