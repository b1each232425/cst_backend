package cmn

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"w2w.io/null"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

/*
	有code
	微信登录
	判断currentAppID类型
	获取openID、保存openID
	是否需要获取用户详细信息
		是，获取用户详细信息，保存用户详细信息
	nextAppID是否为空
		是，转到用户需要的URL
		否，生成获取openID的URL,redirect用户到相应URL

	//------------------------
	一、分层与openID存储
	第一层，用户关注的公众号openID, t_wx_user.mp_open_id
	第二层，用户用于支付的公众号openID, t_wx_user.pay_open_id
	第三+N(N >= 0)层，其它关联公众号,t_wx_user.addi{"openIDs":[
		{"公众号3":"openID3"},
		{"公众号4":"openID4"},
		...
	]}

	二、获取openID规则
	下一公众号appID由nextAppID指定，当前公众号由currAppID指定,如果
		nextAppID为空，则表示获取微信公众号openID结束，返回用户goto所指定的url

	三、goto指向的URL参数格式定义
	1、goto string，表示最终用户访问的页面;
	2、currAppID string,当前code对应的微信公众号appID，用于结合code获取openID;
	3、nextAppID string，下一次redirect获取openID的微信公众号appID;
	4、role string, 用户角色;
	5、qryUserInfo bool, 1: 尝试获取完整用户信息, 0/null/empty: 仅openID;
	6、userID int, 系统用户编号, TUser.ID;
	7、step int, 当获取用户标识的次数;
	8、appType string, wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保;
	9、state string与currAppID相同

	四、说明
	第一次获取openID对应的公众号是用户已经关注的公众号，会获取用户呢称等详细信息，其它
		微信公众号都只是获取openID

	"https://open.weixin.qq.com/connect/oauth2/authorize?
		appid=wx9bf2de6adcc2a356
		&redirect_uri=https%3A%2F%2Fqnear.cn%2Fapi%2FpWxLogin%3Fgoto%3D%2Fxkb%2Ffd%2Fpay
			%26role%3Dxkb%23user
		&response_type=code&scope=snsapi_base&state=wx9bf2de6adcc2a356#wechat_redirect";

	https://open.weixin.qq.com/connect/oauth2/authorize?appid=wx9bf2de6adcc2a356
	&redirect_uri=https://qnear.cn/api/wxLoginLayered?goto=/xkb/fd/pay&role=xkb#user

	&response_type=code&scope=snsapi_base&state=wx9bf2de6adcc2a356#wechat_redirect
	//重定向后会带上state参数，开发者可以填写a-zA-Z0-9的参数值，最多128字节 */

func wxLoginLayeredRepeal(ctx context.Context) {
	q := GetCtxValue(ctx)

	z.Info("---->" + FncName())
	if q.WxLoginProcessed {
		return
	}

	code := q.R.URL.Query().Get("code")
	if code == "" {
		q.Err = fmt.Errorf("wxLoginLayered-1: 没有code,不能用微信登录")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("wxLoginLayered-2: 有code")
	if usedCode(code) {
		z.Error(fmt.Sprintf("wxLoginLayered-3: 用过的code: %s", code))
		return
	}

	currAppID := q.R.URL.Query().Get("currAppID")
	if currAppID == "" {
		q.Err = fmt.Errorf("wxLoginLayered-4: 没有currAppID,不知道获取哪个openID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	appType := q.R.URL.Query().Get("appType")
	if appType == "" {
		q.Err = fmt.Errorf("wxLoginLayered-5: 没有appType,不知道是" +
			"mp.weixin.qq.com还是open.weixin.qq.com的openID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var s string
	switch appType {
	case "wx_mp":
		s = `select tokens->>'wxMpAppID',tokens->>'wxMpAppSecret'
		from t_external_domain_conf
		where app_id=$1 and app_type=$2`

	case "wx_open":
		s = `select tokens->>'wxOpenAppID',tokens->>'wxOpenAppSecret'
		from t_external_domain_conf
		where app_id=$1 and app_type=$2`

	default:
		q.Err = fmt.Errorf("wxLoginLayered-6: 当前还不支持%s平台,appID=%s", appType, currAppID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer stmt.Close()
	var row *sqlx.Row
	row = stmt.QueryRowx(currAppID, appType)
	var appID, appSecret null.String
	q.Err = row.Scan(&appID, &appSecret)
	if q.Err == sql.ErrNoRows {
		q.Err = fmt.Errorf("在表t_external_domain_conf中找不到appID=%s,appType=%s的配置信息，请运维人员修正",
			currAppID, appType)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	z.Info("wxLoginLayered-7: 找到currAppID对应的cert")

	if !appID.Valid || appID.String != currAppID {
		q.Err = fmt.Errorf("wxLoginLayered-8: invalid t_external_domain_conf.tokens.appID for %s", currAppID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !appSecret.Valid || appSecret.String == "" {
		q.Err = fmt.Errorf("wxLoginLayered-9: invalid t_external_domain_conf.tokens.appSecret for %s", currAppID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info(fmt.Sprintf("currAppID: %s, apptype: %s,appID: %s", currAppID, appType, appID.String))
	switch appType {
	case "wx_mp":
		fallthrough
	case "wx_open":
		getMxOpenID(ctx, code, appID.String, appSecret.String)
		if q.Err != nil {
			q.RespErr()
			return
		}
		z.Info("wxLoginLayered-9: 获取了对应currAppID对应的openID")

	default:
		q.Err = fmt.Errorf("wxLoginLayered-10: 当前还不支持%s平台,appID=%s", appType, currAppID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var getInsurancePayOpenID bool
	if viper.IsSet("insuranceOnWxPay.enable") {
		getInsurancePayOpenID = viper.GetBool("insuranceOnWxPay.enable")
	}

	nextAppID := q.R.URL.Query().Get("nextAppID")
	if nextAppID == "" || nextAppID == "null" || !getInsurancePayOpenID {
		z.Info(fmt.Sprintf("wxLoginLayered-11: nextAppID为空，重定向用户到目标页面:%s", q.R.URL.Query().Get("goto")))
		byParamGoTo(ctx)
		return
	}

	z.Info("wxLoginLayered-12: 有nextAppID,redirect用户去获取下一个对应的openID")

	// 此时session/q.SysUser/q.WxUser应该已经建立好了
	if q.SysUser == nil || !q.SysUser.ID.Valid || q.SysUser.ID.Int64 <= 0 {
		q.Err = fmt.Errorf("wxLoginLayered-13: 无效的q.SysUser|q.SysUser.ID")
		q.RespErr()
		return
	}

	host := "qnear.cn"
	if viper.IsSet("webServe.serverName") {
		host = viper.GetString("webServe.serverName")
	}
	role := q.R.URL.Query().Get("role")
	gotoParam := q.R.URL.Query().Get("goto")

	if role == "" || gotoParam == "" {
		q.Err = fmt.Errorf("wxLoginLayered-14: 无效的goto/role参数")
		q.RespErr()
		return
	}
	gotoURL := fmt.Sprintf(`https://%s/api/wxLoginLayered?goto=%s&userID=%d&role=%s&currAppID=%s&step=1&appType=wx_mp`,
		host, gotoParam, q.SysUser.ID.Int64, role, nextAppID)

	z.Info(gotoURL)
	redirectURL := strings.ReplaceAll(strings.ReplaceAll(url.PathEscape(gotoURL), "&", "%26"), "=", "%3D")

	tailURL := fmt.Sprintf(`&response_type=code&scope=snsapi_base&state=%s#wechat_redirect`, nextAppID)
	preURL := fmt.Sprintf(`https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s`, nextAppID)
	dstURL := fmt.Sprintf(`%s&redirect_uri=%s%s`, preURL, redirectURL, tailURL)

	z.Info(dstURL)
	z.Info("wxLoginLayered-15: 获取下一个openID")
	q.Stop = true
	http.Redirect(q.W, q.R, dstURL, http.StatusSeeOther)
}

func getMxOpenID(ctx context.Context, wxCode string, wxAppID string, wxAppSeceret string) (openID string) {
	q := GetCtxValue(ctx)
	q.Stop = true
	z.Info("---->" + FncName())

	tokenURLPtn := `https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code`
	tokenURL := fmt.Sprintf(tokenURLPtn, wxAppID, wxAppSeceret, wxCode)

	var acsToken wxPageAccessToken
	var receiver wxCallbackStatus
	receiver = &acsToken
	q.Err = callWxOAuthAPI(tokenURL, receiver)
	if q.Err != nil {
		return
	}

	if acsToken.OpenID == "" {
		q.Err = fmt.Errorf("getMxOpenID 1: empty acsToekn.OpenID")
		z.Error(q.Err.Error())
		return
	}
	openID = acsToken.OpenID

	var sysUserID int64
	userID := q.R.URL.Query().Get("userID")
	if userID != "" {
		sysUserID, q.Err = strconv.ParseInt(userID, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		if sysUserID <= 0 {
			q.Err = fmt.Errorf("getMxOpenID 5: 无效的userID: %s", userID)
			z.Error(q.Err.Error())
			return
		}

		s := `insert into t_external_domain_user(user_id,business_domain_id,user_domain_id)
			values($1,$2,$3)
			on conflict (business_domain_id,user_domain_id) do nothing
			returning id`

		var stmt *sqlx.Stmt
		stmt, q.Err = sqlxDB.Preparex(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer stmt.Close()
		r := stmt.QueryRow(sysUserID, wxAppID, acsToken.OpenID)
		var id int64
		q.Err = r.Scan(&id)
		if q.Err != nil && q.Err != sql.ErrNoRows {
			z.Error(q.Err.Error())
			return
		}
		if q.Err == sql.ErrNoRows {
			q.Err = nil
		}

		if id > 0 {
			z.Info(fmt.Sprintf("getMxOpenID 2: %s被保存, id=%d", acsToken.OpenID, id))
		} else {
			z.Info(fmt.Sprintf("getMxOpenID 3: %s已存在", acsToken.OpenID))
		}
	}

	// 是为了订单支付获取openID ?-----
	strOrderID := q.R.URL.Query().Get("o")
	payChannelName := q.R.URL.Query().Get("n")
	if strOrderID != "" && payChannelName != "" {
		var orderID int64
		orderID, q.Err = strconv.ParseInt(strOrderID, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		var payAccount *TPayAccount
		payAccount, q.Err = getPayAccount("", payChannelName)
		if q.Err != nil {
			return
		}

		if payAccount == nil || !payAccount.Account.Valid || payAccount.Account.String == "" ||
			!payAccount.Key.Valid || payAccount.Key.String == "" ||
			!payAccount.AppID.Valid || payAccount.AppID.String == "" {
			q.Err = fmt.Errorf("%s(t_pay_account.name)，对应的account/key/app_id无效，请核查", payChannelName)
			z.Error(q.Err.Error())
			return
		}

		payInfo := fmt.Sprintf(`{"payChannel":{"name":"%s",type":"%s","appID":"%s","openID":"%s","key":"%s"}}`,
			payAccount.Name.String, payAccount.Type.String, payAccount.AppID.String,
			acsToken.OpenID, payAccount.Key.String)
		s := `update t_order set addi=$1 where id=$2`
		var stmt *sql.Stmt
		stmt, q.Err = sqlxDB.Prepare(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		var r sql.Result
		r, q.Err = stmt.Exec(payInfo, orderID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		var rowsAffected int64
		rowsAffected, q.Err = r.RowsAffected()
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		if rowsAffected <= 0 {
			z.Warn(fmt.Sprintf("update t_order(id=%d).addi failed", orderID))
		}
	}
	//-------

	if q.R.URL.Query().Get("qryUserInfo") != "true" {
		if userID == "" || sysUserID <= 0 {
			q.Err = fmt.Errorf("getMxOpenID 4: 没有userID,不知道这个openID对应谁")
			z.Error(q.Err.Error())
			return
		}

		// 只为了获取用于微信支付的openID
		s := `update t_wx_user set pay_open_id=$1 where id=$2 returning union_id`
		var stmt *sql.Stmt
		stmt, q.Err = sqlxDB.Prepare(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer stmt.Close()

		var row *sql.Row
		row = stmt.QueryRow(acsToken.OpenID, sysUserID)
		var unionID null.String
		q.Err = row.Scan(&unionID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		if !unionID.Valid || unionID.String == "" {
			q.Err = fmt.Errorf("getMxOpenID 6: 获取t_wx_user.union_id失败, id=%s", userID)
			z.Error(q.Err.Error())
			return
		}

		//在此更新redis.TWxUser.PayOpenID
		key := fmt.Sprintf("%s:%s", CWxUserByUnionID, unionID.String)
		_, q.Err = q.Redis.Do("JSON.SET", key, ".PayOpenID", fmt.Sprintf(`"%s"`, openID))
		if q.Err != nil {
			z.Error(q.Err.Error())
		}
		return
	}

	//获取用户的详细信息
	appType := q.R.URL.Query().Get("appType")
	if appType == "" {
		q.Err = fmt.Errorf("getMxOpenID 3: 没有appType,不知道咋个获取openID")
		z.Error(q.Err.Error())
		return
	}

	//--------------------
	// Get userinfo by ACCESS_TOKEN
	z.Info("getMxOpenID 1: 去获取currAppID对应的用户详细信息")
	userURLPtn := `https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN`
	userURL := fmt.Sprintf(userURLPtn, acsToken.AcccessToken, acsToken.OpenID)

	var u wxUser
	receiver = &u
	q.Err = callWxOAuthAPI(userURL, receiver)
	if q.Err != nil {
		return
	}
	z.Info(fmt.Sprintf("getMxOpenID 2: 得到了currAppID对应的用户%s详细信息", u.Nickname))
	switch appType {
	case "wx_mp":
		u.origin = "mp"
	case "wx_open":
		u.origin = "open"
	default:
		q.Err = fmt.Errorf("getMxOpenID 3: 当前还不支持%s平台,appID=%s", appType, wxAppID)
		z.Error(q.Err.Error())
		return
	}
	category := q.R.URL.Query().Get("role")
	if category == "" {
		category = "xkb^user"
	}
	wxUserUpd(ctx, &u, category)

	return
}
