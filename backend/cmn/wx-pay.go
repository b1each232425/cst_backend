package cmn

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/clbanning/mxj"

	"github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"w2w.io/qrcode"
)

type wxJSPayParam struct {
	AppID     string `json:"appId,omitempty"`
	TimeStamp string `json:"timeStamp,omitempty"`
	NonceStr  string `json:"nonceStr,omitempty"`
	Pakcage   string `json:"package,omitempty"`
	Signtype  string `json:"signType,omitempty"`
	PaySign   string `json:"paySign,omitempty"`
}

func getRealIP() (ip string, err error) {
	status := 0
	var body []byte
	status, body, err = fasthttp.Get(nil, realIPServ)
	if err != nil {
		z.Error(err.Error())
		return
	}
	if status != 200 {
		err = fmt.Errorf("statusCode!=200")
		z.Error(err.Error())
		return
	}
	if len(body) == 0 {
		err = fmt.Errorf("len(body)==0")
		z.Error(err.Error())
		return
	}
	ip = strings.Split(string(body), ":")[0]
	return
}

func wxPayVerifySign(xmlData string, key string) (result bool) {
	xmlData = strings.Trim(xmlData, " ")
	key = strings.Trim(key, " ")
	if len(xmlData) == 0 {
		z.Warn("call wxPayVerifySign with empty xmlData")
		return
	}

	if len(key) == 0 {
		z.Warn("call wxPayVerifySign with empty key")
		return
	}

	root, err := mxj.NewMapXml([]byte(xmlData))
	if err != nil {
		z.Error(err.Error())
		return
	}

	v, ok := root["xml"]
	if !ok || v == nil {
		z.Error("missing root xml field in data: " + xmlData)
		return
	}
	kv, ok := v.(map[string]interface{})
	if !ok || kv == nil {
		z.Error("empty xml: " + xmlData)
		return
	}

	originalSign, ok := kv["sign"].(string)
	if !ok || originalSign == "" {
		z.Error("missing sign field in data: " + xmlData)
		return
	}

	delete(kv, "sign")

	var vl []string
	for k, v := range kv {
		vl = append(vl, fmt.Sprintf("%s=%v", k, v))
	}
	sort.Strings(vl)
	vl = append(vl, "key="+key)
	q := strings.Join(vl, "&")

	md5Ctx := md5.New()
	var n int
	n, err = md5Ctx.Write([]byte(q))
	if err != nil {
		z.Error(err.Error())
		return
	}
	if n != len([]byte(q)) {
		err = fmt.Errorf("writed length mismatc")
		z.Error(err.Error())
		return
	}
	cipherStr := md5Ctx.Sum(nil)
	sign := strings.ToUpper(hex.EncodeToString(cipherStr))
	result = sign == originalSign
	return
}

var wxV3Key = ``

func wxPayMD5Sign(reqDef interface{}, accountKey string) (sign string, err error) {
	// wxOrderAPIV3Cert := wxV3Key
	// if viper.IsSet("wxServe.wxOrderApiV3Cert") {
	// 	wxOrderAPIV3Cert = viper.GetString("wxServe.wxOrderApiV3Cert")
	// }

	if reflect.TypeOf(reqDef).Kind() != reflect.Ptr {
		err = fmt.Errorf("v should be a struct pointer")
		z.Error(err.Error())
		return
	}
	value := reflect.ValueOf(reflect.ValueOf(reqDef).Elem().Interface())
	if value.Kind() != reflect.Struct {
		err = fmt.Errorf("v should be a struct pointer")
		z.Error(err.Error())
		return
	}
	var keyList []string
	kv := make(map[string]interface{})
	for i := 0; i < value.NumField(); i++ {
		tag := value.Type().Field(i).Tag.Get("xml")
		if tag == "" {
			tag = value.Type().Field(i).Tag.Get("json")
		}
		if tag == "xml" || tag == "" {
			continue
		}

		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		v := value.Field(i).Interface()
		zero := reflect.Zero(reflect.TypeOf(v)).Interface()
		if v == zero {
			continue
		}
		keyList = append(keyList, name)
		kv[name] = v
	}

	sort.Strings(keyList)
	var qList []string
	for _, key := range keyList {
		qList = append(qList, fmt.Sprintf("%s=%v", key, kv[key]))
	}
	//qList = append(qList, "key="+wxOrderAPIV3Cert)
	qList = append(qList, "key="+accountKey)
	q := strings.Join(qList, "&")

	md5Ctx := md5.New()
	var n int
	n, err = md5Ctx.Write([]byte(q))
	if err != nil {
		z.Error(err.Error())
		return
	}
	if n != len([]byte(q)) {
		err = fmt.Errorf("writed length mismatc")
		z.Error(err.Error())
		return
	}
	cipherStr := md5Ctx.Sum(nil)
	sign = strings.ToUpper(hex.EncodeToString(cipherStr))
	return
}

type sceneInfo struct {
	XMLName   xml.Name `xml:"scene_info"`
	SceneInfo string   `xml:",cdata"`
}

// func (v sceneInfo) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
// 	tokens := []xml.Token{start}
// 	s := xml.StartElement{Name: xml.Name{Space: "", Local: "abc"}}
// 	tokens = append(tokens, s, xml.EndElement{Name: start.Name})
// 	return nil
// }

type wxPayReq struct {
	XMLName    xml.Name `xml:"xml" json:"-"`
	AppID      string   `xml:"appid,omitempty" json:"appid,omitempty"`
	MchID      string   `xml:"mch_id,omitempty" json:"mch_id,omitempty"`
	NonceStr   string   `xml:"nonce_str,omitempty" json:"nonce_str,omitempty"`
	Body       string   `xml:"body,omitempty" json:"body,omitempty"`
	Detail     string   `xml:"detail,omitempty" json:"detail,omitempty"`
	Attach     string   `xml:"attach,omitempty" json:"attach,omitempty"`
	OutTradeNO string   `xml:"out_trade_no,omitempty" json:"out_trade_no,omitempty"`
	FeeType    string   `xml:"fee_type,omitempty" json:"fee_type,omitempty"`
	TotalFee   int      `xml:"total_fee,omitempty" json:"total_fee,omitempty"`

	TimeStart  string `xml:"time_start,omitempty" json:"time_start,omitempty"`
	TimeExpire string `xml:"time_expire,omitempty" json:"time_expire,omitempty"`
	GoodsTag   string `xml:"goods_tag,omitempty" json:"goods_tag,omitempty"`
	NotifyURL  string `xml:"notify_url,omitempty" json:"notify_url,omitempty"`
	TradeType  string `xml:"trade_type,omitempty" json:"trade_type,omitempty"`
	ProductID  string `xml:"product_id,omitempty" json:"product_id,omitempty"`
	OpenID     string `xml:"openid,omitempty" json:"openid,omitempty"`
	DeviceInfo string `xml:"device_info,omitempty" json:"device_info,omitempty"`

	Sign     string `xml:"sign,omitempty" json:"sign,omitempty"`
	SignType string `xml:"sign_type,omitempty" json:"sign_type,omitempty"`

	SceneInfo string `xml:"scene_info,omitempty" json:"scene_info,omitempty"`

	SPBillCreateIP string `xml:"spbill_create_ip,omitempty" json:"spbill_create_ip,omitempty"`
	LimitPay       string `xml:"limit_pay,omitempty" json:"limit_pay,omitempty"`
	Receipt        string `xml:"receipt,omitempty" json:"receipt,omitempty"`

	//-------- refund
	TransactionID string `xml:"transaction_id,omitempty" json:"transaction_id,omitempty"`
	OutRefundNo   string `xml:"out_refund_no,omitempty" json:"out_refund_no,omitempty"`
	RefundFee     int64  `xml:"refund_fee,omitempty" json:"refund_fee,omitempty"`
	RefundFeeType string `xml:"refund_fee_type,omitempty" json:"refund_fee_type,omitempty"`
	RefundDesc    string `xml:"refund_desc,omitempty" json:"refund_desc,omitempty"`
	RefundAccount string `xml:"refund_account,omitempty" json:"refund_account,omitempty"`
}

func wxPayOrderReq(ctx context.Context, orderID int64) (wxPayURI string) {
	q := GetCtxValue(ctx)

	defaultFilter := map[string]interface{}{
		"ID": map[string]interface{}{"EQ": orderID},
	}
	var o TOrder
	o.TableMap = &o
	o.Action = "select"
	req := ReqProto{
		Action: o.Action,
		Filter: defaultFilter,
	}
	q.Err = DML(&o.Filter, &req)
	if q.Err != nil {
		return
	}
	if o.RowCount != 1 {
		q.Err = fmt.Errorf("inexistent of order id=%d", orderID)
		z.Error(q.Err.Error())
		return
	}
	existsOrder, ok := o.Result[0].(*TOrder)
	if !ok {
		q.Err = fmt.Errorf("o.result[0].(*TOrder) should be ok while it's not")
		z.Error(q.Err.Error())
		return
	}

	if !existsOrder.Status.Valid || existsOrder.Status.String != "0" {
		q.Err = fmt.Errorf("orderID=%d 的订单状态:%s 不允许支付", orderID, existsOrder.Status.String)
		z.Error(q.Err.Error())
		return
	}

	if !existsOrder.Amount.Valid || existsOrder.Amount.Float64 <= 0 {
		q.Err = fmt.Errorf("orderID=%d 的订单金额:%f 错误，不允许支付", orderID,
			existsOrder.Amount.Float64)
		z.Error(q.Err.Error())
		return
	}

	// -----------------

	if !existsOrder.InsuranceTypeID.Valid || existsOrder.InsuranceTypeID.Int64 <= 0 {
		q.Err = fmt.Errorf("无效TOrder(%d).InsuranceTypeID", orderID)
		z.Error(q.Err.Error())
		return
	}

	var payAccount *TPayAccount

	if !payAccount.Type.Valid || payAccount.Type.String != "wx_mp" {
		q.Err = fmt.Errorf("接口%s只支持微信支付，请核查%s(t_pay_account.name)的支付账号配置",
			q.R.URL.Path, payAccount.Type.String)
		z.Error(q.Err.Error())
		return
	}

	if !payAccount.Account.Valid || payAccount.Account.String == "" ||
		!payAccount.Key.Valid || payAccount.Key.String == "" ||
		!payAccount.AppID.Valid || payAccount.AppID.String == "" {
		q.Err = fmt.Errorf("%s(t_pay_account.name)，对应的account/key/app_id无效，请核查", payAccount.Type.String)
		z.Error(q.Err.Error())
		return
	}

	// -----------------------------------

	var r wxPayReq
	r.TotalFee = int(existsOrder.Amount.Float64)

	switch q.CallerType {
	case CAndroidWxCaller, CIOSWxCaller:
		r.TradeType = "JSAPI"
		if q.WxUser == nil {
			q.Err = fmt.Errorf("call wxPayOrderReq with JSAPI but q.WxUser is nil")
			z.Error(q.Err.Error())

			return
		}
		if !q.WxUser.MpOpenID.Valid || q.WxUser.MpOpenID.String == "" {
			q.Err = fmt.Errorf("call wxPayOrderReq with JSAPI but q.WxUser.MpOpenID is empty")
			z.Error(q.Err.Error())
			return
		}

		openIDsValue, ok := q.Session.Values["openIDs"]
		if !ok || openIDsValue == nil {
			openIDsValue = make(map[string]string)
		}
		openIDs, ok := openIDsValue.(map[string]string)
		if !ok {
			q.Err = fmt.Errorf("openIDs should be map[string]string")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		r.OpenID, _ = openIDs[payAccount.AppID.String]
		if r.OpenID == "" {
			q.Err = fmt.Errorf(`openID for appID: %s inexistence in q.session.Values["openIDs"]`,
				payAccount.AppID.String)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

	case CMobileBrowserCaller:
		r.TradeType = "MWEB"

	case CUnknownCaller, CMacWxCaller, CWinWxCaller,
		CUnknownWxCaller, CPcBrowserCaller:
		r.TradeType = "NATIVE"

	default:
		q.Err = fmt.Errorf("不支持此系统的微信支付")
		z.Error(q.Err.Error())
		return
	}

	r.AppID = payAccount.AppID.String
	r.MchID = payAccount.Account.String

	r.NonceStr = generateNonce()
	r.OutTradeNO = existsOrder.TradeNo.String //fmt.Sprintf("%d", (int64(rand.Int31())<<32)|orderID)
	r.Body = "校快保-订单费用支付"
	r.FeeType = "CNY"
	r.DeviceInfo = "WEB_11230433"
	//r.TotalFee = 1

	var serverName string
	if viper.IsSet("webServe.serverName") {
		serverName = viper.GetString("webServe.serverName")
	}
	r.NotifyURL = fmt.Sprintf("https://%s/api/wxPaid", serverName)

	h5Info := `{"h5_info":{"type":"Wap","wap_url":"https%3A%2F%2F` +
		serverName + `%2Fapi%2FwxPaid","wap_name":"校快保-支付"}}`
	r.SceneInfo = h5Info // sceneInfo{SceneInfo: h5Info}

	//r.SPBillCreateIP, q.Err = getRealIP()
	r.SPBillCreateIP = clnAddr(q.R)

	r.Attach = "142857"
	r.SignType = "MD5"

	r.Detail = "WXG"
	r.GoodsTag = "平平安安"
	r.ProductID = fmt.Sprintf("%d", (int64(rand.Int31())<<32)|orderID)
	r.LimitPay = "no_credit"
	r.Receipt = "N"

	t := time.Now()
	r.TimeStart = t.Format("20060102150405")
	tn := t.Add(time.Hour * 1)
	r.TimeExpire = tn.Format("20060102150405")
	z.Info(r.TimeStart + "," + r.TimeExpire)
	r.Sign, q.Err = wxPayMD5Sign(&r, payAccount.Key.String)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	var buf []byte
	buf, q.Err = xml.MarshalIndent(&r, "", "  ")
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	z.Info("\n" + string(buf))

	//---------
	//联保支付
	//if existsOrder.PayType.Valid && existsOrder.PayType.String == "联保" {
	//	wxPayURI, q.Err = uiWxPay(ctx, &r, existsOrder)
	//	return
	//}

	wxUnifiedOrderURL := `https://api.mch.weixin.qq.com/pay/unifiedorder`
	orderReq := &fasthttp.Request{}
	orderReq.SetRequestURI(wxUnifiedOrderURL)
	orderReq.SetBody(buf)

	//req.Header.Set("Accept", "application/xml")
	//req.Header.Set("Content-Type", "application/xml;charset=utf-8")

	// orderReq.Header.SetReferer("https://qnear.cn")
	orderReq.Header.SetContentType("application/xml;charset=UTF-8")
	orderReq.Header.SetMethod("POST")
	resp := &fasthttp.Response{}
	c := &fasthttp.Client{}
	q.Err = c.Do(orderReq, resp)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	body := resp.Body()
	z.Info("\n" + string(body))
	var reply wxOrderReply
	q.Err = xml.Unmarshal(body, &reply)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if reply.ReturnCode != "SUCCESS" {
		q.Err = fmt.Errorf("微信下单,通信失败: %s", reply.ReturnMsg)
		z.Error(q.Err.Error())
		return
	}

	if reply.ResultCode != "SUCCESS" {
		q.Err = fmt.Errorf("微信下单,交易失败: %s", reply.ReturnMsg)
		z.Error(q.Err.Error())
		if reply.ErrCode != "" {
			q.Err = fmt.Errorf("微信下单,交易失败: %s: %s", reply.ErrCode, reply.ErrCodeDes)
			z.Error(q.Err.Error())
		}
		return
	}

	switch r.TradeType {
	case "JSAPI":
		jsParam := wxJSPayParam{
			AppID:     reply.APPID,
			TimeStamp: fmt.Sprintf("%d", GetNowInMS()/1000),
			NonceStr:  generateNonce(),
			Pakcage:   fmt.Sprintf("prepay_id=%s", reply.PrepayID),
			Signtype:  "MD5",
		}

		jsParam.PaySign, q.Err = wxPayMD5Sign(&jsParam, payAccount.Key.String)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		var buf []byte
		buf, q.Err = json.MarshalIndent(&jsParam, "", "  ")
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		wxPayURI = string(buf)
		z.Info(wxPayURI)
	case "MWEB":
		if reply.MwebURL == "" {
			q.Err = fmt.Errorf("微信支付URL地址为空")
			z.Error(q.Err.Error())
			return
		}
		wxPayURI = reply.MwebURL
	case "NATIVE":
		if reply.CodeURL == "" {
			q.Err = fmt.Errorf("微信支付URL地址为空")
			z.Error(q.Err.Error())
			return
		}
		wxPayURI = reply.CodeURL
	default:
		q.Err = fmt.Errorf("unsupported trade_type: " + r.TradeType)
		z.Error(q.Err.Error())
		return
	}

	//pageRoute can't "redirect(303)" directly
	//http.Redirect(q.W, q.r, reply.MwebURL, http.StatusSeeOther)

	return
}

func routineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		z.Error(err.Error())
		return -1
	}

	return id
}

func timePass(ctx context.Context, msg string) {
	q := GetCtxValue(ctx)
	now := time.Now()
	z.Info(fmt.Sprintf("gid: %6d, %8dms, %s[_perf_]", q.RoutineID, now.Sub(q.BeginTime)/time.Millisecond, msg))
	q.BeginTime = now
}

func wxPaid(ctx context.Context) {
	q := GetCtxValue(ctx)

	q.Stop = true
	z.Info("---->" + FncName())
	if strings.ToLower(q.R.Method) != "post" {
		z.Error("call wxPaid must using post http method")
		return
	}

	var buf []byte
	buf, q.Err = ioutil.ReadAll(q.R.Body)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	defer q.R.Body.Close()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("call /api/order by post with empty body")
		z.Error(q.Err.Error())
		return
	}

	z.Info(string(buf))
	var reply wxOrderReply
	q.Err = xml.Unmarshal(buf, &reply)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	if reply.ReturnCode != "SUCCESS" {
		//将支付失败消息传到请求支付状态队列中
		z.Error(reply.ReturnMsg)
		//go wxReplyToSse(reply.OutTradeNO, false)
		return
	}
	//将支付失败消息传到请求支付状态队列中
	//go wxReplyToSse(reply.OutTradeNO, true)

	z.Info("submit order success")

	resp := `"<xml><return_code><![CDATA[SUCCESS]]></return_code><return_msg><![CDATA[OK]]></return_msg></xml>"`
	go fmt.Fprintf(q.W, resp)
	//-------------------------------------------

	s := `select id, type, name, app_id, account, 
	    key, cert, creator, domain_id, addi, remark, 
	    status, create_time, update_time 
	  from t_pay_account where account=$1 limit 1`

	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	defer stmt.Close()

	row := stmt.QueryRowx(reply.MchID)

	var payAccount TPayAccount
	q.Err = row.StructScan(&payAccount)
	if q.Err == sql.ErrNoRows {
		q.Err = fmt.Errorf("未找到%s(t_pay_account.name)的支付账号配置，请核查", payAccount.Type.String)
		z.Error(q.Err.Error())
		return
	}

	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if !payAccount.Account.Valid || payAccount.Account.String == "" ||
		!payAccount.Key.Valid || payAccount.Key.String == "" ||
		!payAccount.AppID.Valid || payAccount.AppID.String == "" {
		q.Err = fmt.Errorf("%s(t_pay_account.name)，对应的account/key/app_id无效，请核查", payAccount.Type.String)
		z.Error(q.Err.Error())
		return
	}

	// sign verify
	if !wxPayVerifySign(string(buf), payAccount.Key.String) {
		z.Error("微信支付签名校验错误: " + string(buf))
		return
	}

	z.Info("wxPaid sign verified")

	//createInsurancePolicy(ctx, reply.OutTradeNO, &reply)
}

func refund(ctx context.Context, tradeNo string) {

}

func setupWxUserByOpenID(ctx context.Context, openID string) (err error) {
	q := GetCtxValue(ctx)
	key := fmt.Sprintf("%s:%s", CWxUserByOpenID, openID)
	userID, err := redis.Int64(q.Redis.Do("GET", key))
	if err != nil {
		z.Error(err.Error())
		return
	}

	key = fmt.Sprintf("%s:%d", CWxUserByID, userID)
	unionID, err := redis.String(q.Redis.Do("GET", key))
	if err != nil {
		z.Error(err.Error())
		return
	}

	key = fmt.Sprintf("%s:%s", CWxUserByUnionID, unionID)
	jsonStr, err := redis.String(q.Redis.Do("JSON.GET", key, "."))
	if err != nil {
		z.Error(err.Error())
		return
	}

	var wxUser TWxUser
	err = json.Unmarshal([]byte(jsonStr), &wxUser)
	if err != nil {
		z.Error(err.Error())
		q.Msg.Status = -400
		return
	}

	q.Err = InvalidEmptyNullValue(&wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	q.WxUser = &wxUser
	return
}

// getKeyByAccount query key by associated account
func getPayAccount(account string, name string) (payAccount *TPayAccount, err error) {
	var f []string
	var v []interface{}
	var k []string
	if account != "" {
		f = append(f, "account=$1")
		v = append(v, account)
		k = append(k, account)

	}
	if name != "" {
		f = append(f, fmt.Sprintf("name=$%d", len(f)+1))
		v = append(v, name)
		k = append(k, account)
	}

	if len(f) <= 0 {
		err = fmt.Errorf("请指定搜索条件")
		z.Error(err.Error())
		return
	}
	w := strings.Join(f, " and ")

	s := fmt.Sprintf(`select 
	  id,name,app_id,type,account,key,cert 
	  from t_pay_account 
	  where %s limit 1`, w)

	var stmt *sqlx.Stmt
	stmt, err = sqlxDB.Preparex(s)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer stmt.Close()

	row := stmt.QueryRowx(v...)

	payAccount = &TPayAccount{}
	err = row.StructScan(payAccount)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("未找到%s(t_pay_account.account/name)的支付账号配置，请核查",
			strings.Join(k, "/"))
		z.Error(err.Error())
		return
	}

	if err != nil {
		z.Error(err.Error())
		return
	}

	if !payAccount.Account.Valid || payAccount.Account.String == "" ||
		!payAccount.Key.Valid || payAccount.Key.String == "" ||
		!payAccount.AppID.Valid || payAccount.AppID.String == "" {
		err = fmt.Errorf("%s(t_pay_account.name)，对应的account/key/app_id无效，请核查", payAccount.Type.String)
		z.Error(err.Error())
	}
	return
}

func wxPay(ctx context.Context) {
	q := GetCtxValue(ctx)

	z.Info("---->" + FncName())
	q.Stop = true
	var orderID int64

	strOrderID := q.R.URL.Query().Get("orderID")
	if strOrderID == "" {
		q.Err = fmt.Errorf("请提供订单号")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	orderID, q.Err = strconv.ParseInt(strOrderID, 10, 64)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if orderID <= 0 {
		q.Err = fmt.Errorf("订单号为0")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Err = setTradeNo(orderID)
	if q.Err != nil {
		q.RespErr()
		return
	}

	payURI := wxPayOrderReq(ctx, orderID)
	if q.Err != nil {
		q.RespErr()
		return
	}
	if payURI == "" {
		q.Err = fmt.Errorf("payURI is empty")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	switch q.CallerType {

	case CUnknownCaller, CMacWxCaller, CWinWxCaller,
		CUnknownWxCaller, CPcBrowserCaller:
		var png []byte
		png, q.Err = qrcode.Encode(payURI, qrcode.Highest, 256)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if len(png) == 0 {
			q.Err = fmt.Errorf("qr-code image is empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"payQrCode":"%s"}`,
			base64.StdEncoding.EncodeToString(png)))
		q.Msg.Status = 0

	case CAndroidWxCaller, CIOSWxCaller:
		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"payInfo":%s}`, payURI))
		q.Msg.Status = 0
		z.Info(string(q.Msg.Data))
	case CMobileBrowserCaller:
		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"payURL":"%s"}`, payURI))
		q.Msg.Status = 0

	default:
		q.Err = fmt.Errorf("unknown caller type:%d", q.CallerType)
		q.RespErr()
		return
	}
	q.Resp()

	return
}

func wxRefund(ctx context.Context) {
	q := GetCtxValue(ctx)

	z.Info("---->" + FncName())
	q.Stop = true

	var r wxPayReq

	r.AppID = "wx0fefb244eeef3422"
	r.MchID = "1538924421"
	r.NonceStr = generateNonce()
	r.SignType = "MD5"

	r.OutTradeNO = "MdJXhsAZp9uSn2dyiFrtFLa6WpGtBTwn"
	r.OutRefundNo = "MdJXhsAZp9uSn2dyiFrtFLa6WpGtBTwn"
	r.TotalFee = 3
	r.RefundFee = 3
	r.RefundFeeType = "CNY"
	r.RefundDesc = "日期/人员变更"
	r.Sign, q.Err = wxPayMD5Sign(&r, "z4fo7AEDLVdshbGWvTnNxOJvtI3nH8yr")
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var buf []byte
	buf, q.Err = xml.Marshal(&r)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	refundURL := "https://api.mch.weixin.qq.com/secapi/pay/refund"
	req := &fasthttp.Request{}
	req.SetRequestURI(refundURL)
	req.Header.SetContentType("application/json;charset=UTF-8")
	req.Header.SetMethod("POST")
	req.SetBody(buf)

	cert, err := tls.LoadX509KeyPair("private/qnear_cert.pem",
		"private/qnear_key.pem")
	if err != nil {
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}

	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	c := &fasthttp.Client{TLSConfig: &tlsCfg}

	resp := &fasthttp.Response{}
	q.Err = c.Do(req, resp)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	body := resp.Body()
	z.Info(string(body))

	var reply wxOrderReply
	q.Err = xml.Unmarshal(body, &reply)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if reply.ReturnCode != "SUCCESS" {
		q.Err = fmt.Errorf("微信支付通信失败: %s", reply.ReturnMsg)
		z.Error(q.Err.Error())
		return
	}

	if reply.ResultCode != "SUCCESS" {
		q.Err = fmt.Errorf("微信支付交易失败: %s", reply.ReturnMsg)
		z.Error(q.Err.Error())
		if reply.ErrCode != "" {
			q.Err = fmt.Errorf("微信支付交易失败: %s: %s", reply.ErrCode, reply.ErrCodeDes)
			z.Error(q.Err.Error())
		}
		return
	}
	z.Info(fmt.Sprintf("成功退了%d分钱", reply.RefundFee))
}

func setTradeNoWithTX(orderID int64, tx *sqlx.Tx) (err error) {
	var txCreated bool
	if tx == nil {
		txCreated = true
		tx, err = sqlxDB.Beginx()
		if err != nil {
			z.Error(err.Error())
			return
		}
		defer tx.Rollback()
	}

	s := `update t_order set trade_no=$1, order_status='18' where id=$2 and status='0'`
	var stmt *sql.Stmt
	var result sql.Result
	var d int64
	var tradeNo = tradeNoWithID(orderID)
	z.Info(fmt.Sprintf("支付订单：%d，设置tradeNo:%s", orderID, tradeNo))

	stmt, err = tx.Prepare(s)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer stmt.Close()
	result, err = stmt.Exec(tradeNo, orderID)
	if err != nil {
		z.Error(err.Error())
		return
	}
	d, err = result.RowsAffected()
	if err != nil {
		z.Error(err.Error())
		return
	}
	if d != 1 {
		err = fmt.Errorf("影响行数为0，更新订单%d trade_no失败，请检查当前订单是否未支付状态：%s", orderID, s)
		z.Error(err.Error())
		return
	}
	z.Info(fmt.Sprintf("set trade_no order=%d trade_no=%s", orderID, tradeNo))

	if !txCreated {
		return
	}

	err = tx.Commit()
	if err != nil {
		z.Error(err.Error())
	}
	return
}

//请求wxPay接口时为该未支付订单设置trade_no

func setTradeNo(orderID int64) (err error) {
	return setTradeNoWithTX(orderID, nil)
}

func simplePost(postData string, url string) {
	req := &fasthttp.Request{}
	req.SetRequestURI(url)
	req.Header.SetContentType("application/xml;charset=UTF-8")
	req.Header.SetMethod("POST")
	req.SetBody([]byte(postData))
	resp := &fasthttp.Response{}
	c := &fasthttp.Client{}

	err := c.Do(req, resp)
	if err != nil {
		z.Error(err.Error())
		return
	}
	body := resp.Body()
	z.Info(string(body))
}

// wxOrderReply 微信请求返回结果
type wxOrderReply struct {
	ReturnCode string `xml:"return_code,omitempty" json:"return_code,omitempty"`   //返回状态码
	ReturnMsg  string `xml:"return_msg,omitempty" json:"return_msg,omitempty"`     //返回信息
	APPID      string `xml:"appid,omitempty" json:"appid,omitempty"`               //公众账号ID
	MchID      string `xml:"mch_id,omitempty" json:"mch_id,omitempty"`             //商户号
	DeviceInfo string `xml:"device_info,omitempty" json:"device_info,omitempty"`   //设备号
	NonceStr   string `xml:"nonce_str,omitempty" json:"nonce_str,omitempty"`       //随机字符串
	Sign       string `xml:"sign,omitempty" json:"sign,omitempty"`                 //签名
	ResultCode string `xml:"result_code,omitempty" json:"result_code,omitempty"`   //业务结果
	ErrCode    string `xml:"err_code,omitempty" json:"err_code,omitempty"`         //错误代码
	ErrCodeDes string `xml:"err_code_des,omitempty" json:"err_code_des,omitempty"` //错误代码描述
	TradeType  string `xml:"trade_type,omitempty" json:"trade_type,omitempty"`     //交易类型
	PrepayID   string `xml:"prepay_id,omitempty" json:"prepay_id,omitempty"`       //预支付交易会话标识
	MwebURL    string `xml:"mweb_url,omitempty" json:"mweb_url,omitempty"`         //支付跳转链接
	CodeURL    string `xml:"code_url,omitempty" json:"code_url,omitempty"`         //支付二维码链接

	OpenID string `xml:"openid" json:"openid,omitempty"`

	Subscribed    string `xml:"is_subscribe" json:"is_subscribe,omitempty"`
	BankType      string `xml:"bank_type" json:"bank_type,omitempty"`
	TotalFee      int    `xml:"total_fee" json:"total_fee,omitempty"`
	FeeType       string `xml:"fee_type" json:"fee_type,omitempty"`
	CashFee       int    `xml:"cash_fee" json:"cash_fee,omitempty"`
	CashFeeType   string `xml:"cash_fee_type" json:"cash_fee_type,omitempty"`
	TransactionID string `xml:"transaction_id" json:"transaction_id,omitempty"`
	OutTradeNO    string `xml:"out_trade_no" json:"out_trade_no,omitempty"`
	Attach        string `xml:"attach" json:"attach,omitempty"`
	TimeEnd       string `xml:"time_end" json:"time_end,omitempty"`

	TradeState     string `xml:"trade_state" json:"trade_state,omitempty"`
	TradeStateDesc string `xml:"trade_state_desc" json:"trade_state_desc,omitempty"`

	SettlementTotalFee int `xml:"settlement_total_fee" json:"settlement_total_fee,omitempty"`

	CouponFee         int `xml:"coupon_fee" json:"coupon_fee,omitempty"`
	CouponRefundFee   int `xml:"coupon_refund_fee" json:"coupon_refund_fee,omitempty"`
	CouponRefundCount int `xml:"coupon_refund_count" json:"coupon_refund_count,omitempty"`

	CouponCount int `xml:"coupon_count" json:"coupon_count,omitempty"`

	// refund
	OutRefundNo   string `xml:"out_refund_no,omitempty" json:"out_refund_no,omitempty"`
	RefundFee     int64  `xml:"refund_fee,omitempty" json:"refund_fee,omitempty"`
	RefundFeeType string `xml:"refund_fee_type,omitempty" json:"refund_fee_type,omitempty"`
	RefundDesc    string `xml:"refund_desc,omitempty" json:"refund_desc,omitempty"`
	RefundAccount string `xml:"refund_account,omitempty" json:"refund_account,omitempty"`

	RefundID string `xml:"refund_id,omitempty" json:"refund_id,omitempty"`

	SettlementRefundFee int64 `xml:"settlement_refund_fee,omitempty" json:"settlement_refund_fee,omitempty"`

	CashRefundFee int64 `xml:"cash_refund_fee,omitempty" json:"cash_refund_fee,omitempty"`

	RefundCount      int64 `xml:"refund_count,omitempty" json:"refund_count,omitempty"`
	TotalRefundCount int64 `xml:"total_refund_count,omitempty" json:"total_refund_count,omitempty"`

	ReqInfo string `xml:"req_info,omitempty" json:"req_info,omitempty"`

	RefundStatus string `xml:"refund_status,omitempty" json:"refund_status,omitempty"`

	RefundRecvAccout  string `xml:"refund_recv_accout,omitempty" json:"refund_recv_accout,omitempty"`
	RefundRecvAccount string `xml:"refund_recv_account,omitempty" json:"refund_recv_account,omitempty"`

	RefundReqSrc string `xml:"refund_request_source,omitempty" json:"refund_request_source,omitempty"`

	SuccessTime string `xml:"success_time,omitempty" json:"success_time,omitempty"`

	// --------------
	OutRefundNO0 string `xml:"out_refund_no_$0" json:"out_refund_no_$0,omitempty"`
	OutRefundNO1 string `xml:"out_refund_no_$1" json:"out_refund_no_$1,omitempty"`
	OutRefundNO2 string `xml:"out_refund_no_$2" json:"out_refund_no_$2,omitempty"`
	OutRefundNO3 string `xml:"out_refund_no_$3" json:"out_refund_no_$3,omitempty"`
	OutRefundNO4 string `xml:"out_refund_no_$4" json:"out_refund_no_$4,omitempty"`
	OutRefundNO5 string `xml:"out_refund_no_$5" json:"out_refund_no_$5,omitempty"`
	OutRefundNO6 string `xml:"out_refund_no_$6" json:"out_refund_no_$6,omitempty"`
	OutRefundNO7 string `xml:"out_refund_no_$7" json:"out_refund_no_$7,omitempty"`
	OutRefundNO8 string `xml:"out_refund_no_$8" json:"out_refund_no_$8,omitempty"`
	OutRefundNO9 string `xml:"out_refund_no_$9" json:"out_refund_no_$9,omitempty"`

	RefundID0 string `xml:"refund_id_$0" json:"refund_id_$0,omitempty"`
	RefundID1 string `xml:"refund_id_$1" json:"refund_id_$1,omitempty"`
	RefundID2 string `xml:"refund_id_$2" json:"refund_id_$2,omitempty"`
	RefundID3 string `xml:"refund_id_$3" json:"refund_id_$3,omitempty"`
	RefundID4 string `xml:"refund_id_$4" json:"refund_id_$4,omitempty"`
	RefundID5 string `xml:"refund_id_$5" json:"refund_id_$5,omitempty"`
	RefundID6 string `xml:"refund_id_$6" json:"refund_id_$6,omitempty"`
	RefundID7 string `xml:"refund_id_$7" json:"refund_id_$7,omitempty"`
	RefundID8 string `xml:"refund_id_$8" json:"refund_id_$8,omitempty"`
	RefundID9 string `xml:"refund_id_$9" json:"refund_id_$9,omitempty"`

	RefundChannel0 string `xml:"refund_channel_$0" json:"refund_channel_$0,omitempty"`
	RefundChannel1 string `xml:"refund_channel_$1" json:"refund_channel_$1,omitempty"`
	RefundChannel2 string `xml:"refund_channel_$2" json:"refund_channel_$2,omitempty"`
	RefundChannel3 string `xml:"refund_channel_$3" json:"refund_channel_$3,omitempty"`
	RefundChannel4 string `xml:"refund_channel_$4" json:"refund_channel_$4,omitempty"`
	RefundChannel5 string `xml:"refund_channel_$5" json:"refund_channel_$5,omitempty"`
	RefundChannel6 string `xml:"refund_channel_$6" json:"refund_channel_$6,omitempty"`
	RefundChannel7 string `xml:"refund_channel_$7" json:"refund_channel_$7,omitempty"`
	RefundChannel8 string `xml:"refund_channel_$8" json:"refund_channel_$8,omitempty"`
	RefundChannel9 string `xml:"refund_channel_$9" json:"refund_channel_$9,omitempty"`

	RefundFee0 int64 `xml:"refund_fee_$0,omitempty" json:"refund_fee_$0,omitempty"`
	RefundFee1 int64 `xml:"refund_fee_$1,omitempty" json:"refund_fee_$1,omitempty"`
	RefundFee2 int64 `xml:"refund_fee_$2,omitempty" json:"refund_fee_$2,omitempty"`
	RefundFee3 int64 `xml:"refund_fee_$3,omitempty" json:"refund_fee_$3,omitempty"`
	RefundFee4 int64 `xml:"refund_fee_$4,omitempty" json:"refund_fee_$4,omitempty"`
	RefundFee5 int64 `xml:"refund_fee_$5,omitempty" json:"refund_fee_$5,omitempty"`
	RefundFee6 int64 `xml:"refund_fee_$6,omitempty" json:"refund_fee_$6,omitempty"`
	RefundFee7 int64 `xml:"refund_fee_$7,omitempty" json:"refund_fee_$7,omitempty"`
	RefundFee8 int64 `xml:"refund_fee_$8,omitempty" json:"refund_fee_$8,omitempty"`
	RefundFee9 int64 `xml:"refund_fee_$9,omitempty" json:"refund_fee_$9,omitempty"`

	SettlementRefundFee0 int64 `xml:"settlement_refund_fee_$0,omitempty" json:"settlement_refund_fee_$0,omitempty"`
	SettlementRefundFee1 int64 `xml:"settlement_refund_fee_$1,omitempty" json:"settlement_refund_fee_$1,omitempty"`
	SettlementRefundFee2 int64 `xml:"settlement_refund_fee_$2,omitempty" json:"settlement_refund_fee_$2,omitempty"`
	SettlementRefundFee3 int64 `xml:"settlement_refund_fee_$3,omitempty" json:"settlement_refund_fee_$3,omitempty"`
	SettlementRefundFee4 int64 `xml:"settlement_refund_fee_$4,omitempty" json:"settlement_refund_fee_$4,omitempty"`
	SettlementRefundFee5 int64 `xml:"settlement_refund_fee_$5,omitempty" json:"settlement_refund_fee_$5,omitempty"`
	SettlementRefundFee6 int64 `xml:"settlement_refund_fee_$6,omitempty" json:"settlement_refund_fee_$6,omitempty"`
	SettlementRefundFee7 int64 `xml:"settlement_refund_fee_$7,omitempty" json:"settlement_refund_fee_$7,omitempty"`
	SettlementRefundFee8 int64 `xml:"settlement_refund_fee_$8,omitempty" json:"settlement_refund_fee_$8,omitempty"`
	SettlementRefundFee9 int64 `xml:"settlement_refund_fee_$9,omitempty" json:"settlement_refund_fee_$9,omitempty"`

	CouponType00 string `xml:"coupon_type_$0_$0" json:"coupon_type_$0_$0,omitempty"`
	CouponType01 string `xml:"coupon_type_$0_$1" json:"coupon_type_$0_$1,omitempty"`
	CouponType02 string `xml:"coupon_type_$0_$2" json:"coupon_type_$0_$2,omitempty"`
	CouponType03 string `xml:"coupon_type_$0_$3" json:"coupon_type_$0_$3,omitempty"`
	CouponType04 string `xml:"coupon_type_$0_$4" json:"coupon_type_$0_$4,omitempty"`
	CouponType05 string `xml:"coupon_type_$0_$5" json:"coupon_type_$0_$5,omitempty"`
	CouponType06 string `xml:"coupon_type_$0_$6" json:"coupon_type_$0_$6,omitempty"`
	CouponType07 string `xml:"coupon_type_$0_$7" json:"coupon_type_$0_$7,omitempty"`
	CouponType08 string `xml:"coupon_type_$0_$8" json:"coupon_type_$0_$8,omitempty"`
	CouponType09 string `xml:"coupon_type_$0_$9" json:"coupon_type_$0_$9,omitempty"`

	CouponRefundCount0 int `xml:"coupon_refund_count_$0" json:"coupon_refund_count_$0,omitempty"`
	CouponRefundCount1 int `xml:"coupon_refund_count_$1" json:"coupon_refund_count_$1,omitempty"`
	CouponRefundCount2 int `xml:"coupon_refund_count_$2" json:"coupon_refund_count_$2,omitempty"`
	CouponRefundCount3 int `xml:"coupon_refund_count_$3" json:"coupon_refund_count_$3,omitempty"`
	CouponRefundCount4 int `xml:"coupon_refund_count_$4" json:"coupon_refund_count_$4,omitempty"`
	CouponRefundCount5 int `xml:"coupon_refund_count_$5" json:"coupon_refund_count_$5,omitempty"`
	CouponRefundCount6 int `xml:"coupon_refund_count_$6" json:"coupon_refund_count_$6,omitempty"`
	CouponRefundCount7 int `xml:"coupon_refund_count_$7" json:"coupon_refund_count_$7,omitempty"`
	CouponRefundCount8 int `xml:"coupon_refund_count_$8" json:"coupon_refund_count_$8,omitempty"`
	CouponRefundCount9 int `xml:"coupon_refund_count_$9" json:"coupon_refund_count_$9,omitempty"`

	CouponRefundID00 int `xml:"coupon_refund_id_$0_$0" json:"coupon_refund_id_$0_$0,omitempty"`
	CouponRefundID01 int `xml:"coupon_refund_id_$0_$1" json:"coupon_refund_id_$0_$1,omitempty"`
	CouponRefundID02 int `xml:"coupon_refund_id_$0_$2" json:"coupon_refund_id_$0_$2,omitempty"`
	CouponRefundID03 int `xml:"coupon_refund_id_$0_$3" json:"coupon_refund_id_$0_$3,omitempty"`
	CouponRefundID04 int `xml:"coupon_refund_id_$0_$4" json:"coupon_refund_id_$0_$4,omitempty"`
	CouponRefundID05 int `xml:"coupon_refund_id_$0_$5" json:"coupon_refund_id_$0_$5,omitempty"`
	CouponRefundID06 int `xml:"coupon_refund_id_$0_$6" json:"coupon_refund_id_$0_$6,omitempty"`
	CouponRefundID07 int `xml:"coupon_refund_id_$0_$7" json:"coupon_refund_id_$0_$7,omitempty"`
	CouponRefundID08 int `xml:"coupon_refund_id_$0_$8" json:"coupon_refund_id_$0_$8,omitempty"`
	CouponRefundID09 int `xml:"coupon_refund_id_$0_$9" json:"coupon_refund_id_$0_$9,omitempty"`

	CouponRefundFee00 int `xml:"coupon_refund_fee_$0_$0" json:"coupon_refund_fee_$0_$0,omitempty"`
	CouponRefundFee01 int `xml:"coupon_refund_fee_$0_$1" json:"coupon_refund_fee_$0_$1,omitempty"`
	CouponRefundFee02 int `xml:"coupon_refund_fee_$0_$2" json:"coupon_refund_fee_$0_$2,omitempty"`
	CouponRefundFee03 int `xml:"coupon_refund_fee_$0_$3" json:"coupon_refund_fee_$0_$3,omitempty"`
	CouponRefundFee04 int `xml:"coupon_refund_fee_$0_$4" json:"coupon_refund_fee_$0_$4,omitempty"`
	CouponRefundFee05 int `xml:"coupon_refund_fee_$0_$5" json:"coupon_refund_fee_$0_$5,omitempty"`
	CouponRefundFee06 int `xml:"coupon_refund_fee_$0_$6" json:"coupon_refund_fee_$0_$6,omitempty"`
	CouponRefundFee07 int `xml:"coupon_refund_fee_$0_$7" json:"coupon_refund_fee_$0_$7,omitempty"`
	CouponRefundFee08 int `xml:"coupon_refund_fee_$0_$8" json:"coupon_refund_fee_$0_$8,omitempty"`
	CouponRefundFee09 int `xml:"coupon_refund_fee_$0_$9" json:"coupon_refund_fee_$0_$9,omitempty"`

	RefundStatus0 string `xml:"refund_status_$0" json:"refund_status_$0,omitempty"`
	RefundStatus1 string `xml:"refund_status_$1" json:"refund_status_$1,omitempty"`
	RefundStatus2 string `xml:"refund_status_$2" json:"refund_status_$2,omitempty"`
	RefundStatus3 string `xml:"refund_status_$3" json:"refund_status_$3,omitempty"`
	RefundStatus4 string `xml:"refund_status_$4" json:"refund_status_$4,omitempty"`
	RefundStatus5 string `xml:"refund_status_$5" json:"refund_status_$5,omitempty"`
	RefundStatus6 string `xml:"refund_status_$6" json:"refund_status_$6,omitempty"`
	RefundStatus7 string `xml:"refund_status_$7" json:"refund_status_$7,omitempty"`
	RefundStatus8 string `xml:"refund_status_$8" json:"refund_status_$8,omitempty"`
	RefundStatus9 string `xml:"refund_status_$9" json:"refund_status_$9,omitempty"`

	RefundAccount0 string `xml:"refund_account_$0" json:"refund_account_$0,omitempty"`
	RefundAccount1 string `xml:"refund_account_$1" json:"refund_account_$1,omitempty"`
	RefundAccount2 string `xml:"refund_account_$2" json:"refund_account_$2,omitempty"`
	RefundAccount3 string `xml:"refund_account_$3" json:"refund_account_$3,omitempty"`
	RefundAccount4 string `xml:"refund_account_$4" json:"refund_account_$4,omitempty"`
	RefundAccount5 string `xml:"refund_account_$5" json:"refund_account_$5,omitempty"`
	RefundAccount6 string `xml:"refund_account_$6" json:"refund_account_$6,omitempty"`
	RefundAccount7 string `xml:"refund_account_$7" json:"refund_account_$7,omitempty"`
	RefundAccount8 string `xml:"refund_account_$8" json:"refund_account_$8,omitempty"`
	RefundAccount9 string `xml:"refund_account_$9" json:"refund_account_$9,omitempty"`

	RefundRecvAccount0 string `xml:"refund_recv_account_$0" json:"refund_recv_account_$0,omitempty"`
	RefundRecvAccount1 string `xml:"refund_recv_account_$1" json:"refund_recv_account_$1,omitempty"`
	RefundRecvAccount2 string `xml:"refund_recv_account_$2" json:"refund_recv_account_$2,omitempty"`
	RefundRecvAccount3 string `xml:"refund_recv_account_$3" json:"refund_recv_account_$3,omitempty"`
	RefundRecvAccount4 string `xml:"refund_recv_account_$4" json:"refund_recv_account_$4,omitempty"`
	RefundRecvAccount5 string `xml:"refund_recv_account_$5" json:"refund_recv_account_$5,omitempty"`
	RefundRecvAccount6 string `xml:"refund_recv_account_$6" json:"refund_recv_account_$6,omitempty"`
	RefundRecvAccount7 string `xml:"refund_recv_account_$7" json:"refund_recv_account_$7,omitempty"`
	RefundRecvAccount8 string `xml:"refund_recv_account_$8" json:"refund_recv_account_$8,omitempty"`
	RefundRecvAccount9 string `xml:"refund_recv_account_$9" json:"refund_recv_account_$9,omitempty"`

	RefundRecvAccout0 string `xml:"refund_recv_accout_$0" json:"refund_recv_accout_$0,omitempty"`
	RefundRecvAccout1 string `xml:"refund_recv_accout_$1" json:"refund_recv_accout_$1,omitempty"`
	RefundRecvAccout2 string `xml:"refund_recv_accout_$2" json:"refund_recv_accout_$2,omitempty"`
	RefundRecvAccout3 string `xml:"refund_recv_accout_$3" json:"refund_recv_accout_$3,omitempty"`
	RefundRecvAccout4 string `xml:"refund_recv_accout_$4" json:"refund_recv_accout_$4,omitempty"`
	RefundRecvAccout5 string `xml:"refund_recv_accout_$5" json:"refund_recv_accout_$5,omitempty"`
	RefundRecvAccout6 string `xml:"refund_recv_accout_$6" json:"refund_recv_accout_$6,omitempty"`
	RefundRecvAccout7 string `xml:"refund_recv_accout_$7" json:"refund_recv_accout_$7,omitempty"`
	RefundRecvAccout8 string `xml:"refund_recv_accout_$8" json:"refund_recv_accout_$8,omitempty"`
	RefundRecvAccout9 string `xml:"refund_recv_accout_$9" json:"refund_recv_accout_$9,omitempty"`

	RefundSuccessTime0 string `xml:"refund_success_time_$0" json:"refund_success_time_$0,omitempty"`
	RefundSuccessTime1 string `xml:"refund_success_time_$1" json:"refund_success_time_$1,omitempty"`
	RefundSuccessTime2 string `xml:"refund_success_time_$2" json:"refund_success_time_$2,omitempty"`
	RefundSuccessTime3 string `xml:"refund_success_time_$3" json:"refund_success_time_$3,omitempty"`
	RefundSuccessTime4 string `xml:"refund_success_time_$4" json:"refund_success_time_$4,omitempty"`
	RefundSuccessTime5 string `xml:"refund_success_time_$5" json:"refund_success_time_$5,omitempty"`
	RefundSuccessTime6 string `xml:"refund_success_time_$6" json:"refund_success_time_$6,omitempty"`
	RefundSuccessTime7 string `xml:"refund_success_time_$7" json:"refund_success_time_$7,omitempty"`
	RefundSuccessTime8 string `xml:"refund_success_time_$8" json:"refund_success_time_$8,omitempty"`
	RefundSuccessTime9 string `xml:"refund_success_time_$9" json:"refund_success_time_$9,omitempty"`

	// -------------------
	CouponType0 string `xml:"coupon_type_$0" json:"CouponType_$0,omitempty"`
	CouponType1 string `xml:"coupon_type_$1" json:"CouponType_$1,omitempty"`
	CouponType2 string `xml:"coupon_type_$2" json:"CouponType_$2,omitempty"`
	CouponType3 string `xml:"coupon_type_$3" json:"CouponType_$3,omitempty"`
	CouponType4 string `xml:"coupon_type_$4" json:"CouponType_$4,omitempty"`
	CouponType5 string `xml:"coupon_type_$5" json:"CouponType_$5,omitempty"`
	CouponType6 string `xml:"coupon_type_$6" json:"CouponType_$6,omitempty"`
	CouponType7 string `xml:"coupon_type_$7" json:"CouponType_$7,omitempty"`
	CouponType8 string `xml:"coupon_type_$8" json:"CouponType_$8,omitempty"`
	CouponType9 string `xml:"coupon_type_$9" json:"CouponType_$9,omitempty"`

	CouponIDN0 string `xml:"coupon_id_0" json:"coupon_id_0,omitempty"`
	CouponIDN1 string `xml:"coupon_id_1" json:"coupon_id_1,omitempty"`
	CouponIDN2 string `xml:"coupon_id_2" json:"coupon_id_2,omitempty"`
	CouponIDN3 string `xml:"coupon_id_3" json:"coupon_id_3,omitempty"`
	CouponIDN4 string `xml:"coupon_id_4" json:"coupon_id_4,omitempty"`
	CouponIDN5 string `xml:"coupon_id_5" json:"coupon_id_5,omitempty"`
	CouponIDN6 string `xml:"coupon_id_6" json:"coupon_id_6,omitempty"`
	CouponIDN7 string `xml:"coupon_id_7" json:"coupon_id_7,omitempty"`
	CouponIDN8 string `xml:"coupon_id_8" json:"coupon_id_8,omitempty"`
	CouponIDN9 string `xml:"coupon_id_9" json:"coupon_id_9,omitempty"`

	CouponFeeN0 string `xml:"coupon_fee_0" json:"coupon_fee_0,omitempty"`
	CouponFeeN1 string `xml:"coupon_fee_1" json:"coupon_fee_1,omitempty"`
	CouponFeeN2 string `xml:"coupon_fee_2" json:"coupon_fee_2,omitempty"`
	CouponFeeN3 string `xml:"coupon_fee_3" json:"coupon_fee_3,omitempty"`
	CouponFeeN4 string `xml:"coupon_fee_4" json:"coupon_fee_4,omitempty"`
	CouponFeeN5 string `xml:"coupon_fee_5" json:"coupon_fee_5,omitempty"`
	CouponFeeN6 string `xml:"coupon_fee_6" json:"coupon_fee_6,omitempty"`
	CouponFeeN7 string `xml:"coupon_fee_7" json:"coupon_fee_7,omitempty"`
	CouponFeeN8 string `xml:"coupon_fee_8" json:"coupon_fee_8,omitempty"`
	CouponFeeN9 string `xml:"coupon_fee_9" json:"coupon_fee_9,omitempty"`

	CouponID0 string `xml:"coupon_id_$0" json:"coupon_id_$0,omitempty"`
	CouponID1 string `xml:"coupon_id_$1" json:"coupon_id_$1,omitempty"`
	CouponID2 string `xml:"coupon_id_$2" json:"coupon_id_$2,omitempty"`
	CouponID3 string `xml:"coupon_id_$3" json:"coupon_id_$3,omitempty"`
	CouponID4 string `xml:"coupon_id_$4" json:"coupon_id_$4,omitempty"`
	CouponID5 string `xml:"coupon_id_$5" json:"coupon_id_$5,omitempty"`
	CouponID6 string `xml:"coupon_id_$6" json:"coupon_id_$6,omitempty"`
	CouponID7 string `xml:"coupon_id_$7" json:"coupon_id_$7,omitempty"`
	CouponID8 string `xml:"coupon_id_$8" json:"coupon_id_$8,omitempty"`
	CouponID9 string `xml:"coupon_id_$9" json:"coupon_id_$9,omitempty"`

	CouponFee0 string `xml:"coupon_fee_$0" json:"coupon_fee_$0,omitempty"`
	CouponFee1 string `xml:"coupon_fee_$1" json:"coupon_fee_$1,omitempty"`
	CouponFee2 string `xml:"coupon_fee_$2" json:"coupon_fee_$2,omitempty"`
	CouponFee3 string `xml:"coupon_fee_$3" json:"coupon_fee_$3,omitempty"`
	CouponFee4 string `xml:"coupon_fee_$4" json:"coupon_fee_$4,omitempty"`
	CouponFee5 string `xml:"coupon_fee_$5" json:"coupon_fee_$5,omitempty"`
	CouponFee6 string `xml:"coupon_fee_$6" json:"coupon_fee_$6,omitempty"`
	CouponFee7 string `xml:"coupon_fee_$7" json:"coupon_fee_$7,omitempty"`
	CouponFee8 string `xml:"coupon_fee_$8" json:"coupon_fee_$8,omitempty"`
	CouponFee9 string `xml:"coupon_fee_$9" json:"coupon_fee_$9,omitempty"`

	CouponRefundFee0 int `xml:"coupon_refund_fee_$0" json:"coupon_refund_fee_$0,omitempty"`
	CouponRefundFee1 int `xml:"coupon_refund_fee_$1" json:"coupon_refund_fee_$1,omitempty"`
	CouponRefundFee2 int `xml:"coupon_refund_fee_$2" json:"coupon_refund_fee_$2,omitempty"`
	CouponRefundFee3 int `xml:"coupon_refund_fee_$3" json:"coupon_refund_fee_$3,omitempty"`
	CouponRefundFee4 int `xml:"coupon_refund_fee_$4" json:"coupon_refund_fee_$4,omitempty"`
	CouponRefundFee5 int `xml:"coupon_refund_fee_$5" json:"coupon_refund_fee_$5,omitempty"`
	CouponRefundFee6 int `xml:"coupon_refund_fee_$6" json:"coupon_refund_fee_$6,omitempty"`
	CouponRefundFee7 int `xml:"coupon_refund_fee_$7" json:"coupon_refund_fee_$7,omitempty"`
	CouponRefundFee8 int `xml:"coupon_refund_fee_$8" json:"coupon_refund_fee_$8,omitempty"`
	CouponRefundFee9 int `xml:"coupon_refund_fee_$9" json:"coupon_refund_fee_$9,omitempty"`

	CouponRefundID0 int `xml:"coupon_refund_id_$0" json:"coupon_refund_id_$0,omitempty"`
	CouponRefundID1 int `xml:"coupon_refund_id_$1" json:"coupon_refund_id_$1,omitempty"`
	CouponRefundID2 int `xml:"coupon_refund_id_$2" json:"coupon_refund_id_$2,omitempty"`
	CouponRefundID3 int `xml:"coupon_refund_id_$3" json:"coupon_refund_id_$3,omitempty"`
	CouponRefundID4 int `xml:"coupon_refund_id_$4" json:"coupon_refund_id_$4,omitempty"`
	CouponRefundID5 int `xml:"coupon_refund_id_$5" json:"coupon_refund_id_$5,omitempty"`
	CouponRefundID6 int `xml:"coupon_refund_id_$6" json:"coupon_refund_id_$6,omitempty"`
	CouponRefundID7 int `xml:"coupon_refund_id_$7" json:"coupon_refund_id_$7,omitempty"`
	CouponRefundID8 int `xml:"coupon_refund_id_$8" json:"coupon_refund_id_$8,omitempty"`
	CouponRefundID9 int `xml:"coupon_refund_id_$9" json:"coupon_refund_id_$9,omitempty"`
}
