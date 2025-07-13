package cmn

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"golang.org/x/crypto/bcrypt"

	"github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"w2w.io/null"
)

func getWxServerIP() {
	/*
		forwarded := q.R.Header.Get("X-Forwarded-For")
		if forwarded == "" {
			forwarded = q.R.RemoteAddr
		}
		wxIP := strings.Split(forwarded, ":")[0]
		var isWxIP bool
		for _, v := range wxServIPs.IPList {
			if v == wxIP {
				isWxIP = true
				break
			}
		}
		if !isWxIP {
			q.Err = fmt.Errorf("%s不属于%v, 是攻击者", wxIP, wxServIPs.IPList)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}*/
}

func getUserCategory(ctx context.Context) (category string) {
	q := GetCtxValue(ctx)
	queries, ok := q.R.URL.Query()["role"]
	if !ok || len(queries) == 0 || queries[0] == "" {
		category = "anonymous"
		return
	}

	for _, v := range roleList {
		if strings.ToLower(v) != strings.ToLower(queries[0]) {
			continue
		}
		category = strings.ToLower(v)
		break
	}

	if category == "" {
		category = "anonymous"
	}
	return
}

func wxLoginCached(ctx context.Context, u *wxUser) (bHit bool) {
	q := GetCtxValue(ctx)
	if u == nil {
		q.Err = fmt.Errorf("call cached with nil parameter u")
		z.Error(q.Err.Error())
		return
	}

	if u.UnionID == "" {
		wxMpAppID := "wx0fefb244eeef3422"
		if viper.IsSet("wxServe.wxMpAppID") {
			wxMpAppID = viper.GetString("wxServe.wxMpAppID")
		}
		q.Err = fmt.Errorf("狗日的腾迅，您得搞一个微信开发平台帐号，再把%s这个公众帐号绑上，不然就没有UnionID，而咱们需要这个", wxMpAppID)
		z.Error(q.Err.Error())
		return
	}

	key := fmt.Sprintf("%s:%s", CWxUserByUnionID, u.UnionID)
	z.Info(key)
	jsonStr, err := redis.String(q.Redis.Do("JSON.GET", key, "."))
	if err != nil {
		z.Info(fmt.Sprintf("missing: %s", key))
		return
	}

	var wxUser TWxUser
	q.Err = json.Unmarshal([]byte(jsonStr), &wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.Err = InvalidEmptyNullValue(&wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if !wxUser.ID.Valid || wxUser.ID.Int64 == 0 {
		err = fmt.Errorf("wxUser.ID in redis is invalid")
		z.Error(err.Error())
		return
	}

	key = fmt.Sprintf("%s:%s", CWxUserByOpenID, u.OpenID)
	jsonStr, err = redis.String(q.Redis.Do("GET", key))
	if err != nil {
		z.Info(fmt.Sprintf("missing: %s", key))
		field := ""
		switch u.origin {
		case "mp":
			field = "mp_open_id"
		case "open":
			field = "wx_open_id"
		default:
			panic("what the fuck:" + u.origin)
		}

		s := fmt.Sprintf(`update t_wx_user set %s=$1 where id=$2`, field)
		var stmt *sql.Stmt
		stmt, q.Err = sqlxDB.Prepare(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				z.Error(err.Error())
			}
		}()

		var result sql.Result
		result, q.Err = stmt.Exec(u.OpenID, wxUser.ID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		var d int64
		if d, q.Err = result.RowsAffected(); q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		if d != 1 {
			q.Err = fmt.Errorf("更新userID=%d的openID:%s失败, 这说明数据库被重置过，让用户重新登录即可", wxUser.ID.Int64, u.OpenID)
			z.Error(q.Err.Error())
			return
		}
		_, q.Err = q.Redis.Do("SET", key, wxUser.ID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
	}

	var touched bool
	switch u.origin {
	case "mp":
		if !wxUser.MpOpenID.Valid {
			wxUser.MpOpenID = null.StringFrom(u.OpenID)
			touched = true
		}
	case "open":
		if !wxUser.WxOpenID.Valid {
			wxUser.WxOpenID = null.StringFrom(u.OpenID)
			touched = true
		}
	}
	if touched {
		var buf []byte
		buf, q.Err = MarshalJSON(&wxUser)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		key = fmt.Sprintf("%s:%s", CWxUserByUnionID, u.UnionID)
		_, q.Err = q.Redis.Do("JSON.SET", key, ".", string(buf))
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
	}

	key = fmt.Sprintf("%s:%d", CSysUserByID, wxUser.ID.Int64)
	var userName string
	userName, q.Err = redis.String(q.Redis.Do("GET", key))
	if q.Err != nil {
		q.Err = fmt.Errorf("user: %d not exists in cache", wxUser.ID.Int64)
		z.Error(q.Err.Error())
		q.Err = nil
		return
	}

	key = fmt.Sprintf("%s:%s", CSysUserByName, userName)
	jsonStr, q.Err = redis.String(q.Redis.Do("JSON.GET", key, "."))
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.Err = fmt.Errorf("user: %s not exists in cache", userName)
		return
	}

	r := gjson.Get(jsonStr, "MobilePhone")
	if r.Exists() && r.Num > 0 {
		jsonStr, q.Err = sjson.Set(jsonStr, "MobilePhone",
			fmt.Sprintf("%d", int64(r.Num)))
	}

	var sysUser TUser
	q.Err = json.Unmarshal([]byte(jsonStr), &sysUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.Err = InvalidEmptyNullValue(&sysUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if !sysUser.ID.Valid || sysUser.ID.Int64 == 0 {
		err = fmt.Errorf("sysUser.ID in redis is invalid")
		z.Error(err.Error())
		return
	}

	q.WxUser = &wxUser
	q.SysUser = &sysUser

	return true
}

var usedCodeList = make(map[string]time.Time)

func usedCode(code string) (used bool) {
	if _, ok := usedCodeList[code]; ok {
		return true
	}
	usedCodeList[code] = time.Now()
	for k, v := range usedCodeList {
		if -v.Sub(time.Now()) > time.Minute*6 {
			delete(usedCodeList, k)
		}
	}
	return false
}

// 作废
func wxLoginRepeal(ctx context.Context) {
	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	if q.WxLoginProcessed {
		return
	}

	code := q.R.URL.Query().Get("code")
	if code == "" {
		q.Err = fmt.Errorf("w2:没有code,不能用微信登录")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("w3:有code")
	if usedCode(code) {
		z.Error(fmt.Sprintf("w4:用过的code: %s", code))
		return
	}
	state := q.R.URL.Query().Get("state")
	if state == "" {
		q.Err = fmt.Errorf("w5:没有state,搞不清是微信公众号平台还是微信开放平台")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	category := getUserCategory(ctx)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//-----------------------------
	wxOpenAppID := "wxbbcdc7faf43cecec"
	if viper.IsSet("wxServe.wxOpenAppID") {
		wxOpenAppID = viper.GetString("wxServe.wxOpenAppID")
	}
	wxOpenAppSecret := "f4ce6525dbfd7e53376cc15dea624ce8"
	if viper.IsSet("wxServe.wxOpenAppSecret") {
		wxOpenAppSecret = viper.GetString("wxServe.wxOpenAppSecret")
	}
	wxMpAppID := "wx0fefb244eeef3422"
	if viper.IsSet("wxServe.wxMpAppID") {
		wxMpAppID = viper.GetString("wxServe.wxMpAppID")
	}
	wxMpAppSecret := "9e4a4dfd0da11c0cb473c08b758a2582"
	if viper.IsSet("wxServe.wxMpAppSecret") {
		wxMpAppSecret = viper.GetString("wxServe.wxMpAppSecret")
	}

	// User already approve and login through weChat
	// get ACCESS_TOKEN by CODE
	if state == wxOpenAppID {

		tokenURLPtn := `https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code`
		tokenURL := fmt.Sprintf(tokenURLPtn, wxOpenAppID, wxOpenAppSecret, code)

		var receiver wxCallbackStatus
		receiver = &wxOpenPageToken
		q.Err = callWxOAuthAPI(tokenURL, receiver)
		if q.Err != nil {
			q.RespErr()
			return
		}

		//--------------------
		// Get userinfo by ACCESS_TOKEN
		userURLPtn := `https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN`
		userURL := fmt.Sprintf(userURLPtn, wxOpenPageToken.AcccessToken, wxOpenPageToken.OpenID)

		var u wxUser
		receiver = &u
		q.Err = callWxOAuthAPI(userURL, receiver)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		u.origin = "open"
		wxUserUpd(ctx, &u, category)
		if q.Err != nil {
			q.RespErr()
			return
		}

	} else if state == wxMpAppID {
		tokenURLPtn := `https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code`
		tokenURL := fmt.Sprintf(tokenURLPtn, wxMpAppID, wxMpAppSecret, code)

		var receiver wxCallbackStatus
		receiver = &wxMxPageToken
		q.Err = callWxOAuthAPI(tokenURL, receiver)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//--------------------
		// Get userinfo by ACCESS_TOKEN
		userURLPtn := `https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN`
		userURL := fmt.Sprintf(userURLPtn, wxMxPageToken.AcccessToken, wxMxPageToken.OpenID)

		var u wxUser
		receiver = &u
		q.Err = callWxOAuthAPI(userURL, receiver)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		u.origin = "mp"
		wxUserUpd(ctx, &u, category)
		if q.Err != nil {
			q.RespErr()
			return
		}
	} else {
		q.Err = fmt.Errorf("unknown state type, it's must a attacker")
		z.Error(q.Err.Error())
		q.Stop = true
		q.RespErr()
		return
	}

	byParamGoTo(ctx)
}

func enrollWxUser(ctx context.Context, openID string) {
	q := GetCtxValue(ctx)

	if openID == "" {
		q.Err = fmt.Errorf("call getWxUser with empty openID")
		z.Error(q.Err.Error())
		return
	}

	//--------------------
	// Get wxMP linked user info by ACCESS_TOKEN
	wxUserInfoURLPattern := `https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s`
	wxUserInfoURL := fmt.Sprintf(wxUserInfoURLPattern, wxMainMpAccessToken.Token, openID)

	var u wxUser
	receiver := &u
	q.Err = callWxOAuthAPI(wxUserInfoURL, receiver)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	var buf []byte
	buf, q.Err = json.Marshal(&u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	z.Info(string(buf))

	wxUserUpd(ctx, &u, "xkb.user")
}

// --------------------------------------
func wxUserUpd(ctx context.Context, u *wxUser, category string) {
	q := GetCtxValue(ctx)

	if u.UnionID == "" {
		u.UnionID = u.OpenID
	}

	// user enrolled
	if wxLoginCached(ctx, u) {
		if !q.SysUser.ID.Valid || q.SysUser.ID.Int64 == 0 {
			q.Err = fmt.Errorf("q.SysUser.ID is invalid")
			z.Error(q.Err.Error())
			return
		}

		q.Session.Values["ID"] = q.SysUser.ID.Int64
		q.Session.Values["Account"] = q.SysUser.Account
		q.Session.Values["Role"] = q.SysUser.Category
		q.Session.Values["wxUnionID"] = q.WxUser.UnionID.String
		q.Session.Values["Authenticated"] = true
		q.Session.Values["loginType"] = "wxLogin"
		q.Err = q.Session.Save(q.R, q.W)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		z.Info(fmt.Sprintf("%s hit cache", q.SysUser.Account))
		return
	}

	if q.Err != nil {
		return
	}

	// ---- save to dbms
	var wxUser TWxUser

	wxUser.Subscribe = null.NewInt(int64(u.Subscribe), true)
	wxUser.SubscribeTime = null.NewInt(int64(u.SubscribeTime), true)
	switch u.origin {
	case "mp":
		wxUser.MpOpenID = null.StringFrom(u.OpenID)
	case "open":
		wxUser.WxOpenID = null.StringFrom(u.OpenID)
	}

	wxUser.UnionID = null.NewString(u.UnionID, true)
	wxUser.GroupID = null.NewInt(int64(u.GroupID), true)
	var tagIDList string
	for _, v := range u.TagIDList {
		tagIDList = tagIDList + fmt.Sprintf("%d,", v)
	}
	if len(tagIDList) > 0 {
		tagIDList = tagIDList[:len(tagIDList)-1]
	}
	wxUser.TagIDList = null.NewString(tagIDList, true)
	wxUser.Nickname = null.NewString(u.Nickname, true)
	wxUser.Sex = null.NewInt(int64(u.Sex), true)
	wxUser.Language = null.NewString(u.Language, true)
	wxUser.City = null.NewString(u.City, true)
	wxUser.Province = null.NewString(u.Province, true)
	wxUser.Country = null.NewString(u.Country, true)
	wxUser.HeadImgURL = null.NewString(u.HeadimgURL, true)

	wxUser.Privilege = null.NewString(strings.Join(u.Privilege, ","), true)
	wxUser.Remark = null.NewString(u.Remark, true)
	wxUser.QrScene = null.NewInt(int64(u.QRScene), true)
	wxUser.QrSceneStr = null.NewString(u.QRSceneStr, true)
	wxUser.SubscribeScene = null.NewString(u.SubscribeScene, true)

	wxUser.Filter.TableMap = &wxUser
	s := "select id,wx_open_id,mp_open_id from t_wx_user where union_id=$1"
	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	defer func() {
		err := stmt.Close()
		if err != nil {
			z.Error(err.Error())
		}
	}()

	row := stmt.QueryRowx(wxUser.UnionID)
	var userID int64
	var wxOpenID, mpOpenID null.String
	q.Err = row.Scan(&userID, &wxOpenID, &mpOpenID)
	if q.Err != nil && q.Err != sql.ErrNoRows {
		z.Error(q.Err.Error())
		return
	}

	var exists bool
	if q.Err != sql.ErrNoRows {
		exists = true
	}
	var wxFilter map[string]interface{}
	var sysFilter map[string]interface{}
	var sysUser TUser
	if exists {
		wxUser.Action = "update"
		wxFilter = map[string]interface{}{
			"ID": map[string]interface{}{"EQ": userID},
		}

		sysFilter = map[string]interface{}{
			"ID": map[string]interface{}{"EQ": userID},
		}

		sysUser.UpdateTime = null.NewInt(GetNowInMS(), true)
	} else {
		wxUser.Action = "insert"
		sysUser.CreateTime = null.NewInt(GetNowInMS(), true)
	}

	sysUser.Action = wxUser.Action

	//--------
	sysUser.Category = category
	sysUser.ExternalID = null.NewString(wxUser.UnionID.String, true)
	sysUser.Nickname = wxUser.Nickname

	var gender string
	switch wxUser.Sex.Int64 {
	case 0:
		gender = "未知"
	case 1:
		gender = "男"
	case 2:
		gender = "女"
	default:
		gender = "未声明"

	}
	sysUser.Gender = null.StringFrom(gender)

	sysUser.Avatar = []byte(wxUser.HeadImgURL.NullString.String)
	sysUser.LogonTime = null.NewInt(GetNowInMS(), true)
	sysUser.Role = null.IntFrom(377) //domain: sys^user
	sysUser.Filter.TableMap = &sysUser
	req := ReqProto{
		Action: sysUser.Action,
		Filter: sysFilter,
	}

	q.Err = DML(&sysUser.Filter, &req)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	newUserID, ok := sysUser.QryResult.(int64)
	if sysUser.Action == "insert" && (!ok || newUserID <= 0) {
		q.Err = fmt.Errorf("insert return 0 for RETURNING ID when insert into t_user")
		return
	}

	if exists {
		sysUser.ID = null.NewInt(userID, true)
		s := `select account,id,category,type,addr,id_card_no,
				mobile_phone,email,official_name,gender,nickname,
				avatar,avatar_type,user_token from t_user where id=$1`
		var stmt *sqlx.Stmt
		stmt, q.Err = sqlxDB.Preparex(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				z.Error(err.Error())
			}
		}()

		row := stmt.QueryRowx(userID)
		q.Err = row.StructScan(&sysUser)
		if q.Err != nil && q.Err != sql.ErrNoRows {
			z.Error(q.Err.Error())
			return
		}

		if q.Err == sql.ErrNoRows {
			q.Err = fmt.Errorf("can't find use with id=%d", userID)
			z.Error(q.Err.Error())
			return
		}
	} else {
		sysUser.ID = null.NewInt(newUserID, true)
		var buf []byte
		buf, q.Err = bcrypt.GenerateFromPassword([]byte("cSc^6z9B"), 6)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		sysUser.UserToken = null.NewString(string(buf), true)
		s := `update t_user set account=$1,user_token=public.crypt('vNb7_!529',public.gen_salt('bf')) where id=$2`
		var stmt *sqlx.Stmt
		stmt, q.Err = sqlxDB.Preparex(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				z.Error(err.Error())
			}
		}()

		sysUser.Account = fmt.Sprintf("%d", newUserID)
		var result sql.Result
		result, q.Err = stmt.Exec(sysUser.Account, newUserID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		var rowAffected int64
		rowAffected, q.Err = result.RowsAffected()
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}

		if rowAffected != 1 {
			q.Err = fmt.Errorf("update TUser.Name failed with id=%d", newUserID)
			z.Error(q.Err.Error())
			return
		}
		// setup authenticate
		s = `insert into t_user_domain(sys_user,domain,domain_id,creator,addi,grant_source)
				select $1,(select id from t_domain where domain=$2),
				(select id from t_domain where domain=$3),$4,
				'{"source":"wx"}','self'
			on conflict(sys_user,domain) do update set
				domain_id=excluded.domain_id,
				creator=excluded.creator
			returning id
			`

		stmt, q.Err = sqlxDB.Preparex(s)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				z.Error(err.Error())
			}
		}()

		r := stmt.QueryRow(newUserID, "sys^user", "sys", newUserID)
		var userDomainID null.Int
		q.Err = r.Scan(&userDomainID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		if !userDomainID.Valid || userDomainID.Int64 <= 0 {
			q.Err = fmt.Errorf("id is invalid on insert to TUserDomain")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}
	wxUser.ID = sysUser.ID

	req = ReqProto{
		Action: wxUser.Action,
		Filter: wxFilter,
	}

	q.Err = DML(&wxUser.Filter, &req)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if !sysUser.ID.Valid || sysUser.ID.Int64 == 0 {
		q.Err = fmt.Errorf("sysUser.ID is invalid")
		z.Error(q.Err.Error())
		return
	}
	// --------------------
	cacheSysUser(ctx, &sysUser)
	if q.Err != nil {
		return
	}
	cacheWxUser(ctx, &wxUser)
	if q.Err != nil {
		return
	}
	// user enrolled
	q.Session.Values["ID"] = sysUser.ID.Int64
	q.Session.Values["Account"] = sysUser.Account
	q.Session.Values["Role"] = 377 // sys^user, user's default domain
	q.Session.Values["wxUnionID"] = wxUser.UnionID.String
	q.Session.Values["Authenticated"] = true
	q.Session.Values["loginType"] = "wxLogin"

	q.Err = q.Session.Save(q.R, q.W)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.SysUser = &sysUser
	q.WxUser = &wxUser
	return
}

func cacheUser(ctx context.Context, userID int64, wxUnionID string) {
	q := GetCtxValue(ctx)

	var values []interface{}
	var qryFilter []string
	if userID > 0 {
		qryFilter = append(qryFilter, fmt.Sprintf("id=$%d", len(qryFilter)+1))
		values = append(values, userID)
	}
	if wxUnionID != "" {
		qryFilter = append(qryFilter, fmt.Sprintf("union_id=$%d", len(qryFilter)+1))
		values = append(values, wxUnionID)
	}
	s := fmt.Sprintf(`select id from v_xkb_user where %s`, strings.Join(qryFilter, " or "))

	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			z.Error(err.Error())
		}
	}()

	row := stmt.QueryRowx(values...)
	var existsID null.Int
	q.Err = row.Scan(&existsID)
	if q.Err != nil && q.Err != sql.ErrNoRows {
		z.Error(q.Err.Error())
		return
	}

	if q.Err == sql.ErrNoRows {
		q.Err = fmt.Errorf("can't find use with id=%d", userID)
		z.Error(q.Err.Error())
		return
	}
	var sysUser TUser
	sysUser.TableMap = &sysUser
	sysUser.Action = "select"

	req := ReqProto{
		Action: sysUser.Action,
		Filter: map[string]interface{}{
			"ID": map[string]interface{}{"EQ": existsID.Int64},
		},
	}
	q.Err = DML(&sysUser.Filter, &req)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	if sysUser.RowCount == 0 {
		q.Err = fmt.Errorf("can't find user with openID=%s or id=%d", wxUnionID, userID)
		z.Error(q.Err.Error())
		return
	}
	v, ok := sysUser.Result[0].(*TUser)
	if !ok {
		q.Err = fmt.Errorf("sysUser.result[0].(*TUser) should be ok while it not")
		z.Error(q.Err.Error())
		return
	}
	q.SysUser = v
	if !q.SysUser.ID.Valid || q.SysUser.ID.Int64 == 0 {
		q.Err = fmt.Errorf("sysUser.ID is invalid")
		z.Error(q.Err.Error())
		return
	}
	cacheSysUser(ctx, q.SysUser)
	if q.Err != nil {
		return
	}

	var wxUser TWxUser
	wxUser.TableMap = &wxUser
	wxUser.Action = "select"

	var err error
	req = ReqProto{
		Action: wxUser.Action,
		Filter: map[string]interface{}{
			"ID": map[string]interface{}{"EQ": existsID.Int64},
		},
	}
	err = DML(&wxUser.Filter, &req)
	if err != nil {
		z.Error(err.Error())
		return
	}
	if wxUser.RowCount == 0 {
		err = fmt.Errorf("can't find user with openID=%s or id=%d", wxUnionID, userID)
		z.Error(err.Error())
		return
	}
	w, ok := wxUser.Result[0].(*TWxUser)
	if !ok {
		err = fmt.Errorf("wxUser.result[0].(*TWxUser) should be ok while it not")
		z.Error(err.Error())
		return
	}
	q.WxUser = w

	cacheWxUser(ctx, q.WxUser)
}

func cacheWxUser(ctx context.Context, wxUser *TWxUser) {
	q := GetCtxValue(ctx)

	if wxUser == nil {
		q.Err = fmt.Errorf("wxUser is nil")
		z.Error(q.Err.Error())
		return
	}

	if !wxUser.ID.Valid || wxUser.ID.Int64 == 0 {
		q.Err = fmt.Errorf("wxUser.ID is invalid")
		z.Error(q.Err.Error())
		return
	}
	if !wxUser.UnionID.Valid || wxUser.UnionID.String == "" {
		q.Err = fmt.Errorf("wxUser.UnionID is invalid")
		z.Error(q.Err.Error())
		return
	}

	var buf []byte

	buf, q.Err = MarshalJSON(wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key := fmt.Sprintf("%s:%s", CWxUserByUnionID, wxUser.UnionID.String)
	_, q.Err = q.Redis.Do("JSON.SET", key, ".", string(buf))
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key = fmt.Sprintf("%s:%d", CWxUserByID, wxUser.ID.Int64)

	_, q.Err = q.Redis.Do("SET", key, wxUser.UnionID.String)
	if q.Err != nil {
		z.Error(q.Err.Error())
	}

	var haveValidOpenID bool
	if wxUser.MpOpenID.Valid && wxUser.MpOpenID.String != "" {
		key = fmt.Sprintf("%s:%s", CWxUserByOpenID, wxUser.MpOpenID.String)
		_, q.Err = q.Redis.Do("SET", key, wxUser.ID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
		}
		haveValidOpenID = true
	}

	if wxUser.WxOpenID.Valid && wxUser.WxOpenID.String != "" {
		key = fmt.Sprintf("%s:%s", CWxUserByOpenID, wxUser.WxOpenID.String)
		_, q.Err = q.Redis.Do("SET", key, wxUser.ID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
		}
		haveValidOpenID = true
	}

	if !haveValidOpenID {
		q.Err = fmt.Errorf("没有有效的openID")
		z.Error(q.Err.Error())
	}
}

func cacheSysUser(ctx context.Context, sysUser *TUser) {
	q := GetCtxValue(ctx)

	if sysUser == nil {
		q.Err = fmt.Errorf("sysUser is nil")
		z.Error(q.Err.Error())
		return
	}

	if !sysUser.ID.Valid || sysUser.ID.Int64 == 0 {
		q.Err = fmt.Errorf("sysUser.ID is invalid")
		z.Error(q.Err.Error())
		return
	}

	if sysUser.Account == "" {
		q.Err = fmt.Errorf("account is empty with id: %d", sysUser.ID.Int64)
		z.Error(q.Err.Error())
		return
	}

	var buf []byte
	buf, q.Err = MarshalJSON(sysUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key := fmt.Sprintf("%s:%s", CSysUserByName, sysUser.Account)
	_, q.Err = q.Redis.Do("JSON.SET", key, ".", string(buf))
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key = fmt.Sprintf("%s:%d", CSysUserByID, sysUser.ID.Int64)
	_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if sysUser.MobilePhone.Valid && sysUser.MobilePhone.String != "" {
		key = fmt.Sprintf("%s:%s", CSysUserByTel, sysUser.MobilePhone.String)
		_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		z.Info("set cache by " + CSysUserByTel)
	}

	if sysUser.Email.Valid && sysUser.Email.String != "" {
		key = fmt.Sprintf("%s:%s", CSysUserByEmail, sysUser.Email.String)
		_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		z.Info("set cache by " + CSysUserByEmail)
	}
}

func callWxOAuthAPI(dstURL string, v wxCallbackStatus) error {
	z.Info("---->" + FncName())
	if dstURL == "" || v == nil {
		return fmt.Errorf("dstURL or body receiver is empty/nil")
	}
	z.Info(dstURL)
	status, body, err := fasthttp.Get(nil, dstURL)
	if err != nil {
		z.Warn(err.Error())
		return err
	}
	if status != 200 {
		err = fmt.Errorf("%s", fmt.Sprintf("status code is %v", status))
		z.Warn(err.Error())
		return err
	}
	if len(body) < 32 {
		err = fmt.Errorf("%s", fmt.Sprintf("body's length is %v which too short", len(body)))
		z.Warn(err.Error())
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		z.Error(err.Error())
		return err
	}

	if v.getErrCode() != 0 {
		err = fmt.Errorf("%s", fmt.Sprintf("errCode=%v, %s", v.getErrCode(), v.getErrMsg()))
		z.Error(err.Error())
		return err
	}

	return nil
}

func reloadCacheBySession(ctx context.Context) {

	q := GetCtxValue(ctx)
	id, ok := (q.Session.Values["ID"]).(int64)
	if !ok || id <= 0 {
		q.Err = fmt.Errorf("ID should be int64 in session while it's not")
		z.Error(q.Err.Error())
		return
	}
	openID, ok := (q.Session.Values["wxUnionID"]).(string)
	if !ok {
		q.Err = fmt.Errorf("OpenID should be string in session while it's not")
		z.Error(q.Err.Error())
		return
	}
	cacheUser(ctx, id, openID)
}

func byParamGoTo(ctx context.Context) {
	q := GetCtxValue(ctx)
	dst := q.R.URL.Query().Get("goto")
	originalDST := dst
	if dst == "" {
		q.Stop = true
		q.Err = fmt.Errorf("goto param is empty")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if dst[0] != '/' {
		dst = "/" + dst
	}

	z.Info(">>>>>>>>>>>>>>>")
	z.Info(fmt.Sprintf("dst: %s", dst))
	z.Info(fmt.Sprintf("r.URL.Path: %s", q.R.URL.Path))
	z.Info("<<<<<<<<<<<<<<<<")
	if dst == q.R.URL.Path {
		q.WxLoginProcessed = true
		return
	}

	qry := q.R.URL.Query()
	var params []string
	for k := range qry {
		paramValue := qry.Get(k)
		if k == "goto" || k == "code" || paramValue == "" {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%v", k, paramValue))
	}
	var userID int64
	if q.SysUser != nil && q.SysUser.ID.Valid && q.SysUser.ID.Int64 > 0 {
		userID = q.SysUser.ID.Int64
	} else {
		z.Warn("q.SysUser.ID.int64 is invalid/nil")
	}
	params = append(params, fmt.Sprintf("userID=%d", userID))

	var isExternal bool
	re := regexp.MustCompile(`(?i)^(?P<addr>http(s)?://\w+\.\w+(\.\w+)*(/.*)*)`)
	match := re.FindStringSubmatch(originalDST)
	var externalAddr string
	if len(match) > 0 {
		matchResult := make(map[string]string)
		for k, v := range re.SubexpNames() {
			if k == 0 || v == "" {
				continue
			}
			matchResult[v] = match[k]
		}
		externalAddr = matchResult["addr"]
		isExternal = true
	}

	var dstURL string
	if isExternal {
		z.Info(originalDST + " is external URL")
		if len(params) > 0 {
			dstURL = fmt.Sprintf("%s/?%s", externalAddr, strings.Join(params, "&"))
		} else {
			dstURL = externalAddr
		}
	} else {
		z.Info(originalDST + " is non external URL")
		if len(params) > 0 {
			dstURL = fmt.Sprintf("%s://%s%s?%s",
				"https", q.R.Host, dst, strings.Join(params, "&"))
		} else {
			dstURL = fmt.Sprintf("%s://%s%s",
				"https", q.R.Host, dst)
		}
	}

	dstURL = strings.ReplaceAll(dstURL, "/?", "?")

	z.Info(dstURL)
	q.Stop = true
	http.Redirect(q.W, q.R, dstURL, http.StatusSeeOther)
}

func getWxOpenID(ctx context.Context,
	wxCode string,
	wxAppID string,
	wxAppSeceret string,
	appType string,
	isUserInfo bool, sysUserID int64) (openID string, userInfo *wxUser) {
	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	if !isUserInfo && sysUserID <= 0 {
		q.Err = fmt.Errorf("(!isUserInfo && sysUserID) is false")
		z.Error(q.Err.Error())
		return
	}

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

	if isUserInfo {
		// 需要获得用户详细信息
		userURLPtn := `https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN`
		userURL := fmt.Sprintf(userURLPtn, acsToken.AcccessToken, acsToken.OpenID)

		var u wxUser
		userInfo = &u
		receiver = &u
		q.Err = callWxOAuthAPI(userURL, receiver)
		if q.Err != nil {
			return
		}

		switch appType {
		case "wx_mp":
			u.origin = "mp"
		case "wx_open":
			u.origin = "open"
		default:
			q.Err = fmt.Errorf("当前还不支持%s平台,appID=%s", appType, wxAppID)
			z.Error(q.Err.Error())
			return
		}
		category := q.R.URL.Query().Get("role")
		if category == "" {
			category = "xkb^user"
		}
		wxUserUpd(ctx, &u, category)
		if q.Err != nil {
			return
		}

		if q.SysUser == nil {
			q.Err = fmt.Errorf("创建用户出错,openID= %s", u.OpenID)
			z.Error(q.Err.Error())
			return
		}

		if q.SysUser.ID.Valid && q.SysUser.ID.Int64 > 0 && sysUserID <= 0 {
			sysUserID = q.SysUser.ID.Int64
		}
	}
	if sysUserID <= 0 {
		q.Err = fmt.Errorf("未获得/生成有效的用户ID, 不能保存openID: %s", openID)
		z.Error(q.Err.Error())
		return
	}

	s := `insert into t_external_domain_user(user_id,business_domain_id,user_domain_id,domain_type,creator)
			values($1,$2,$3,$4,$1)
			on conflict do nothing
			returning id`

	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			z.Error(err.Error())
		}
	}()
	r := stmt.QueryRow(sysUserID, wxAppID, acsToken.OpenID, appType)
	var id int64
	q.Err = r.Scan(&id)
	if q.Err == sql.ErrNoRows {
		q.Err = nil
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if id > 0 {
		z.Info(fmt.Sprintf("%s被保存, id=%d", acsToken.OpenID, id))
	} else {
		z.Info(fmt.Sprintf("%s已存在", acsToken.OpenID))
	}
	return
}

var mainWxOpenAppID = "wxbbcdc7faf43cecec"
var mainWxOpenAppSecret = "f4ce6525dbfd7e53376cc15dea624ce8"

var mainWxMpAppID = "wx0fefb244eeef3422"
var mainWxMpAppSecret = "9e4a4dfd0da11c0cb473c08b758a2582"

// WxLogin  WxLogin ***************************************************************
func WxLogin(ctx context.Context) {
	q := GetCtxValue(ctx)
	z.Info("---->" + FncName())

	code := q.R.URL.Query().Get("code")

	if code == "" { //其它业务流程,没有code
		return
	}

	z.Info(fmt.Sprintf("有code: %s", code))
	// silent continue process request
	// 微信授权服务器可能会回调两次
	if usedCode(code) {
		z.Warn(fmt.Sprintf("用过的无效code: %s", code))
		return
	}

	state := q.R.URL.Query().Get("state")
	if state == "" {
		q.Err = fmt.Errorf("没有state, 是哪个appID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// isUserInfo = true, 表示获取的是主公众号的openID,
	// isUserInfo = false, 表示获取的是用于微信支付的公众号的openID, 此时url中必须包含userID
	isUserInfo := false

	if state == mainWxMpAppID || state == mainWxOpenAppID {
		isUserInfo = true
		z.Info(fmt.Sprintf("收到主公众号: %s的code", state))
	} else {
		z.Info(fmt.Sprintf("收到公众号: %s的code", state))
	}

	var sysUserID int64
	userID := q.R.URL.Query().Get("userID")
	if userID != "" {
		sysUserID, q.Err = strconv.ParseInt(userID, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if sysUserID <= 0 {
			q.Err = fmt.Errorf("无效的userID: %s", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if !isUserInfo && sysUserID == 0 {
		q.Err = fmt.Errorf("请在获取openID的URL中设置有效的userID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	mpConf, isWxMp := externalDomainsConf[state+"#wx_mp"]
	openConf, isWxOpen := externalDomainsConf[state+"#wx_open"]
	if !isWxMp && !isWxOpen {
		q.Err = fmt.Errorf("external_domain_conf中找不到state存储的appID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	appType := "wx_mp"
	secretType := "wxMpAppSecret"
	var tokens []byte
	if mpConf != nil {
		tokens = mpConf.Tokens
	} else {
		tokens = openConf.Tokens
		secretType = "wxOpenAppSecret"
		appType = "wx_open"
	}

	if len(tokens) == 0 {
		q.Err = fmt.Errorf("external_domain_conf(app_id=%s).tokens is nil/empty", state)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var appCfg map[string]interface{}
	q.Err = json.Unmarshal(tokens, &appCfg)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	secret, ok := appCfg[secretType]
	if !ok || secret == "" {
		q.Err = fmt.Errorf("external_domain_conf(app_id=%s).tokens.%s is nil/empty", state, secretType)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if mpConf == nil && openConf == nil {
		q.Err = fmt.Errorf("external_domain_conf中找不到state存储的appID")
		z.Error(q.Err.Error())
		q.RespErr()
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

	openID, ok := openIDs[state]
	if !ok {
		var userInfo *wxUser
		openID, userInfo = getWxOpenID(ctx, code,
			state, secret.(string), appType,
			isUserInfo, sysUserID)
		if q.Err != nil {
			q.RespErr()
			return
		}

		if isUserInfo && userInfo == nil {
			q.Err = fmt.Errorf("failed to get userInfo")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//save openID to session
		openIDs[state] = openID
		q.Session.Values["openIDs"] = openIDs
		q.Err = q.Session.Save(q.R, q.W)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if (sysUserID <= 0) && (q.SysUser == nil || !q.SysUser.ID.Valid ||
			q.SysUser.ID.Int64 <= 0) {

			q.Err = fmt.Errorf("无效的q.SysUser.ID,请研发后端核查")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if sysUserID <= 0 {
			sysUserID = q.SysUser.ID.Int64
		}
	}

	if sysUserID <= 0 {
		sysUserID, _ = q.Session.Values["ID"].(int64)
	}

	if sysUserID <= 0 {
		q.Err = fmt.Errorf("无效的sysUserID,请研发后端核查")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 只有在公众号里才能直接静默获取openID, 在pc browser 里会反复让用户扫码,体验很差
	if q.CallerType != CIOSWxCaller && q.CallerType != CAndroidWxCaller {
		return
	}

	//下一个需要获取openID的appID
	nextAppID := ""
	for _, v := range externalDomainsConf {
		if v.Status.String != "02" || v.AppType != "wx_mp" {
			continue
		}

		if v.AppID == "" {
			q.Err = fmt.Errorf("external_domain_conf(id=%d).app_id 无效,请管理员核查", v.ID.Int64)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		nextOpenID := openIDs[v.AppID]
		if nextOpenID != "" {
			continue
		}
		nextAppID = v.AppID
		break
	}

	//说明没有需要获取openID的第三方平台appID
	if nextAppID == "" {
		return
	}
	z.Info("nextAppID: " + nextAppID)

	host := "qnear.cn"
	if viper.IsSet("webServe.serverName") {
		host = viper.GetString("webServe.serverName")
	}

	// 1、appID string,当前code对应的微信公众号appID，用于结合code获取openID, 必须有;
	// 2、qryUserInfo bool, 1: 尝试获取完整用户信息, 0/null/empty: 仅openID, 必须有;
	// 3、userID int, 系统用户编号, TUser.ID, qryUserInfo=true则必须有,qryUserInfo=false则可以没有;
	// 4、step int, 第几次获取用户openID, 可以没有;
	// 5、appType string, wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保, 必须有;
	// 6、state string与appID相同, 必须有
	// 7、goto string微信登录后跳转的URL, 没有则默认跳转到当前请求的路径

	jumpTo := q.R.URL.Query().Get("goto")
	if jumpTo == "" {
		jumpTo = q.R.URL.Path
	}

	step := q.R.URL.Query().Get("step")
	if step == "" {
		step = "0"
	}

	var nextStep int64
	nextStep, q.Err = strconv.ParseInt(step, 10, 32)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	nextStep++

	redirect := fmt.Sprintf(`https://%s%s?appID=%s&qryUserInfo=%s&step=%d&appType=wx_mp&userID=%d`,
		host, jumpTo, nextAppID, "false", nextStep, sysUserID)

	z.Info(fmt.Sprintf("生成查询下一公众号: %s 的redirect URL:\n\t%s", nextAppID, redirect))
	redirectURL := strings.ReplaceAll(strings.ReplaceAll(url.PathEscape(redirect), "&", "%26"), "=", "%3D")

	scope := "snsapi_base"
	tailURL := fmt.Sprintf(`&response_type=code&scope=%s&state=%s#wechat_redirect`, scope, nextAppID)
	preURL := fmt.Sprintf(`https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s`, nextAppID)
	dstURL := fmt.Sprintf(`%s&redirect_uri=%s%s`, preURL, redirectURL, tailURL)

	z.Info(fmt.Sprintf("生成查询下一公众号: %s 的最终URL:\n\t%s", nextAppID, dstURL))
	q.Stop = true
	http.Redirect(q.W, q.R, dstURL, http.StatusSeeOther)
	return
}

func getWxLoginURL(ctx context.Context) (target string) {
	q := GetCtxValue(ctx)

	//default jumpTo value
	jumpTo := "/xkb/web"

	//configure jumpTo value
	if viper.IsSet("wxServe.goto") {
		jumpTo = viper.GetString("wxServe.goto")
	}

	//url inferred
	if !rIsAPI.MatchString(q.R.URL.Path) {
		jumpTo = q.R.URL.Path
	}

	//the goto parameter have highest priority
	if gotoDst := q.R.URL.Query().Get("goto"); gotoDst != "" {
		jumpTo = gotoDst
	}

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

	// 1、appID string,当前code对应的微信公众号appID，用于结合code获取openID, 必须有;
	// 2、qryUserInfo bool, 1: 尝试获取完整用户信息, 0/null/empty: 仅openID, 必须有;
	// 3、userID int, 系统用户编号, TUser.ID, qryUserInfo=true则必须有,qryUserInfo=false则可以没有;
	// 4、step int, 第几次获取用户openID, 可以没有;
	// 5、appType string, wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保, 必须有;
	// 6、state string与appID相同, 必须有

	base := "https://open.weixin.qq.com/connect/"
	redirectURI := `https://%s%s?appID=%s&qryUserInfo=true&step=0&appType=%s`

	switch q.CallerType {
	case CWinWxCaller:
		fallthrough
	case CMacWxCaller:
		fallthrough
	case CUnknownWxCaller:
		fallthrough
	case CIOSWxCaller:
		fallthrough
	case CAndroidWxCaller:
		//微信公众号登录
		z.Info("微信,公众号登录")
		s := fmt.Sprintf(redirectURI, serverName, jumpTo, wxMpAppID, "wx_mp")
		s = strings.ReplaceAll(strings.ReplaceAll(url.PathEscape(s), "&", "%26"), "=", "%3D")
		target = fmt.Sprintf(
			`%soauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect`,
			base, wxMpAppID, s, wxMpAppID)

	case CUnknownCaller:
		fallthrough
	case CMobileBrowserCaller:
		fallthrough
	case CPcBrowserCaller:
		z.Info("微信,扫码登录")
		//微信的扫码登录
		s := fmt.Sprintf(redirectURI, serverName, jumpTo, wxOpenAppID, "wx_open")
		s = strings.ReplaceAll(strings.ReplaceAll(url.PathEscape(s), "&", "%26"), "=", "%3D")
		target = fmt.Sprintf(
			`%sqrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect`,
			base, wxOpenAppID, s, wxOpenAppID)

	default:
		q.Err = fmt.Errorf("unsupported plateform")
		z.Error(q.Err.Error())
	}
	z.Info(target)
	return
}

// eraseUser 抹除用户及其数据
// https://qnear.cn/api/dbStatus?xCleanSession=142857&erase=true
func eraseUser(userID int64) (err error) {
	s := []string{
		`delete from t_external_domain_user where user_id = $1`,
		`delete from t_insurance_policy where creator = $1`,
		`delete from t_order where creator = $1`,
		`delete from t_user_domain where sys_user = $1`,
		`delete from t_relation where left_id = $1 and left_type = 't_user.id'`,
		`delete from t_xkb_user where id = $1`,
		`delete from t_wx_user where id = $1`,
		`delete from t_user where id = $1`,
	}
	ctx := context.Background()
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer tx.Rollback(ctx)

	for _, v := range s {
		_, err = tx.Exec(ctx, v, userID)
		if err != nil {
			z.Error(err.Error())
			return
		}
	}

	tx.Commit(ctx)
	return
}
