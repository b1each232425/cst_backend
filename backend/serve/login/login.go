// Package login management
package login

//annotation:upLogin-service
//author:{"name":"user","tel":"18928776452","email":"XUnion@GMail.com"}

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"w2w.io/cmn"
	"w2w.io/null"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("user.Enroll called")

	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: upLogin,

		Path: "/login",
		Name: "login",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: upLogout,

		Path: "/logout",
		Name: "logout",

		Developer: developer,
		WhiteList: false,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

}

type upInfo struct {
	Name string `json:"name,omitempty"`
	Cert string `json:"cert,omitempty"`
}

func upLoginCached(ctx context.Context, u *upInfo) (bHit bool) {
	q := cmn.GetCtxValue(ctx)
	if u == nil || u.Name == "" || u.Cert == "" {
		q.Err = fmt.Errorf("call cached with nil/empty parameter u")
		z.Error(q.Err.Error())
		return
	}

	var key, jsonStr, name string
	var err error

	for {
		key = fmt.Sprintf("%s:%s", cmn.SysUserByID, u.Name)
		name, err = redis.String(q.Redis.Do("GET", key))
		if err == nil {
			break
		}

		key = fmt.Sprintf("%s:%s", cmn.SysUserByTel, u.Name)
		name, err = redis.String(q.Redis.Do("GET", key))
		if err == nil {
			break
		}

		key = fmt.Sprintf("%s:%s", cmn.SysUserByEmail, u.Name)
		name, err = redis.String(q.Redis.Do("GET", key))
		if err == nil {
			break
		}
		name = u.Name
		break
	}

	key = fmt.Sprintf("%s:%s", cmn.SysUserByName, name)
	jsonStr, err = redis.String(q.Redis.Do("JSON.GET", key, "."))
	if err != nil {
		z.Info(err.Error())
		return
	}

	var sysUser cmn.TUser
	q.Err = json.Unmarshal([]byte(jsonStr), &sysUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.Err = cmn.InvalidEmptyNullValue(&sysUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	q.Err = bcrypt.CompareHashAndPassword([]byte(sysUser.UserToken.String), []byte(u.Cert))
	if q.Err != nil {
		q.Err = fmt.Errorf("%s: invalid user/token", u.Name)
		z.Error(q.Err.Error())
		return
	}

	q.SysUser = &sysUser

	openID, err := redis.String(q.Redis.Do("GET", fmt.Sprintf("%s:%d", cmn.WxUserByID, sysUser.ID.Int64)))
	if err != nil {
		z.Info(err.Error())
		return true
	}

	jsonStr, q.Err = redis.String(q.Redis.Do("JSON.GET", fmt.Sprintf("%s:%s", cmn.WxUserByUnionID, openID), "."))
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	var wxUser cmn.TWxUser
	q.Err = json.Unmarshal([]byte(jsonStr), &wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.Err = cmn.InvalidEmptyNullValue(&wxUser)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}
	q.WxUser = &wxUser
	return true
}

func upLogin(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http POST method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("Call /api/upLogin with  empty body")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var u upInfo
	q.Err = json.Unmarshal(buf, &u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if u.Name == "" || u.Cert == "" {
		q.Err = fmt.Errorf("call /api/upLogin with empty u/p")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if upLoginCached(ctx, &u) { // user enrolled
		q.Session.Values["loginType"] = "upLogin"
		q.Session.Values["ID"] = q.SysUser.ID.Int64
		q.Session.Values["Account"] = q.SysUser.Account
		q.Session.Values["Role"] = q.SysUser.Category
		q.Session.Values["Authenticated"] = true
		q.Err = q.Session.Save(q.R, q.W)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		t := q.SysUser.UserToken
		q.SysUser.UserToken = null.NewString("", false)
		buf, q.Err = cmn.MarshalJSON(q.SysUser)
		q.SysUser.UserToken = t
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = buf
		q.Resp()
		z.Info(fmt.Sprintf("%s hit cache", u.Name))

		return
	}
	//-------
	s := `
	    select ID,category,type,addr,id_card_no,
				mobile_phone,email,account,official_name,gender,nickname,
				avatar,avatar_type,user_token 
			from t_user 
			where (
				account=$1 
				or id=$2 
				or mobile_phone=$1 
				or email=$1
				or official_name=$1) 
			  and user_token=public.crypt($3,user_token)`
	sqlxDB := cmn.GetDbConn()
	var stmt *sqlx.Stmt
	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	defer func() {
		q.Err = stmt.Close()
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	// 13580452503: 0x03,29-75-42-97
	//  4294967295
	userID, err := strconv.ParseInt(u.Name, 10, 64)
	if err != nil {
		z.Info(err.Error())
	}
	if userID > 0xFFFFFFFF {
		userID = 0
	}

	row := stmt.QueryRowx(u.Name, userID, u.Cert)
	if errors.Is(q.Err, sql.ErrNoRows) {
		q.Err = fmt.Errorf("错误的用户名/口令")
		q.RespErr()
		return
	}

	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var sysUser cmn.TUser
	q.Err = row.StructScan(&sysUser)
	if errors.Is(q.Err, sql.ErrNoRows) {
		q.Err = fmt.Errorf("错误的用户名/口令")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.SysUser = &sysUser

	if !sysUser.ID.Valid || sysUser.ID.Int64 == 0 {
		q.Err = fmt.Errorf("sysUser.ID is invalid")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	cmn.CacheSysUser(ctx, &sysUser)

	// user enrolled
	q.Session.Values["loginType"] = "upLogin"
	q.Session.Values["ID"] = sysUser.ID.Int64
	q.Session.Values["Account"] = sysUser.Account
	q.Session.Values["Role"] = sysUser.Category
	q.Session.Values["Authenticated"] = true
	q.Err = q.Session.Save(q.R, q.W)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	t := q.SysUser.UserToken
	q.SysUser.UserToken = null.NewString("", false)
	buf, q.Err = cmn.MarshalJSON(q.SysUser)
	q.SysUser.UserToken = t
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Msg.Data = buf

	s = `select id, subscribe, subscribe_time, wx_open_id, mp_open_id, 
				pay_open_id, union_id, group_id, open_id, nickname, 
				sex, language, city, province, country, head_img_url,addi
			from t_wx_user
			where id=$1`

	stmt, q.Err = sqlxDB.Preparex(s)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = stmt.Close()
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	q.Tag["callerID"] = -410104

	row = stmt.QueryRowx(sysUser.ID.Int64)
	var wxUser cmn.TWxUser
	q.Err = row.StructScan(&wxUser)

	if q.Err != nil && !errors.Is(q.Err, sql.ErrNoRows) {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if q.Err == nil {
		cmn.CacheWxUser(ctx, &wxUser)
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Session.Values["wxUnionID"] = wxUser.UnionID
		q.Err = q.Session.Save(q.R, q.W)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}
	q.Err = nil

	q.Resp()
}

func upLogout(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())
	q.Stop = true

	q.W.Header().Add("Access-Control-Allow-Origin", q.R.Header.Get("Origin"))
	q.W.Header().Add("Access-Control-Allow-Credentials", "true")

	cmn.CleanSession(ctx)
	if q.Err != nil {
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Resp()
}
