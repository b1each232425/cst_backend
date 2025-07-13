package cmn

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jmoiron/sqlx/types"
	"github.com/spf13/viper"
	"github.com/tidwall/sjson"
	"regexp"
	"strconv"
	"strings"
	"w2w.io/null"
)

//var RoleList = []string{
//	"anonymous",
//	"developer",
//	"investor",
//	"follower",
//	"proprietor",
//	"witness",
//	"xkb.user",
//	"xkb.sale.admin",
//	"xkb.school.admin",
//	"xkb.admin",
//}

//aa: Authenticate&Authorize

// AutoRole 自动选择角色
// 根据最近一次访问的前端页面类型来决定是前台角色还是后台角色
//
//			前端角色: 如果最近访问的是/xkb/web, 角色选择为sys^user
//	   后端角色: 如果最近访问的是/xkb/web_admin, 角色选择为sys^user
func AutoRole(ctx context.Context) (err error) {
	q := GetCtxValue(ctx)

	//set the default row
	q.Role = int64(CDomainSysUser) //sys^user
	roleInSession, ok := q.Session.Values["Role"].(int64)
	if !ok {
		z.Warn("can't find Role in session")
	}
	var roleInCache int64
	if q.SysUser != nil && q.SysUser.Role.Valid && q.SysUser.Role.Int64 > 0 {
		roleInCache = q.SysUser.Role.Int64
	}

	if roleInSession > 0 {
		q.Role = roleInSession
	} else if roleInCache > 0 {
		q.Role = roleInCache
	}

	var determined bool
	switch q.ReqFnType {
	case CFuncAdminFileServe:
		if q.Role != int64(CDomainSysUser) && q.Role != int64(CDomainXKBUser) {
			//已经是后台角色了
			determined = true
			break
		}

		for _, v := range q.Domains {
			if v.ID.Int64 != int64(CDomainSysUser) && v.ID.Int64 != int64(CDomainXKBUser) {
				//找到第一个不是前端的角色
				q.Role = v.ID.Int64
				determined = true
				break
			}
		}
	case CFuncNonAdminFileServe:
		fallthrough
	default:
		for _, v := range q.Domains {
			if v.ID.Int64 == int64(CDomainSysUser) {
				q.Role = v.ID.Int64
				determined = true
				break
			}
		}
	}
	if !determined {
		err = fmt.Errorf("用户(%d):无权限访问%s",
			q.SysUser.ID.Int64, q.Ep.Path)
		//q.Msg.Status = -888
		z.Warn(err.Error())

	}
	err = nil
	return
}

func setUserDomain(ctx context.Context) (err error) {
	q := GetCtxValue(ctx)
	var callerID int
	callerID, _ = (q.Tag["callerID"]).(int)
	if q.SysUser == nil || !q.SysUser.ID.Valid || q.SysUser.ID.Int64 <= 0 {
		err = fmt.Errorf("invalid q.SysUser | ID | Int64, callerID: %d", callerID)
		z.Error(err.Error())
		return
	}

	if q.DomainList != nil || q.Domains != nil {
		return
	}

	s := `select jsonb_agg(distinct jsonb_build_object(
			'Name',domain_name,'Priority',priority,
			'Domain',domain,
			'ID',auth_domain_id,
   	'Addi',addi)),jsonb_agg(distinct domain)
		from v_user_domain where user_id=$1`
	r := sqlxDB.QueryRow(s, q.SysUser.ID.Int64)

	var domains, domainList null.String

	err = r.Scan(&domains, &domainList)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("user(%d) didn't have any domain, callerID: %d",
			q.SysUser.ID.Int64, callerID)
		z.Error(err.Error())
		return
	}

	if err != nil {
		z.Error(err.Error())
		return
	}
	if !domains.Valid || domains.String == "" {
		CleanSession(ctx)
		err = fmt.Errorf("user(%d) domain list is empty, callerID: %d", q.SysUser.ID.Int64, callerID)
		z.Error(err.Error())
		return
	}

	err = json.Unmarshal([]byte(domains.String), &q.Domains)
	if err != nil {
		z.Error(fmt.Sprintf("%s, callerID: %d", err.Error(), callerID))
		return
	}
	err = json.Unmarshal([]byte(domainList.String), &q.DomainList)
	if err != nil {
		z.Error(fmt.Sprintf("%s, callerID: %d", err.Error(), callerID))
		return
	}

	re := regexp.MustCompile(`(?i:xkb\^sale|xkb\^admin|xkb\.school\^(admin|statistics)|sys\^admin)`)
	q.IsAdmin = re.Match([]byte(domainList.String))

	//自动设置用户的角色
	err = AutoRole(ctx)
	return
}

// Authenticate
// 一、授权判定算法
//  1. 获取用户的角色，用户可能有多个角色，但同一时刻只能使用一个角色，这个角色存放于t_user.role中
//     如果t_user.role为nil/0/则拒绝用户访问，所以，如果用户有多个角色，则系统应该提供用户一个选择角色的机会；
//  2. 以用户角色与访问的API为条件搜索v_user_domain_api，存在则允许访问，否则拒绝
//  3. 任何方式赋予用户角色时都会将该角色置为置为用户默认角色,保存到t_user.role中, t_user.role保存的值为
//     t_domain.id
//
// 二、授权判定流程
// 1. 用户访问
// 2. 显示开放页面
// 3. 请求非开放页面或接口
// 4. 后端返回
// 4.1  status:-1000,
// 4.2  msg.data.appID
// 4.3  要求前端登录
// 5. 前端
// 5.1  组装微信登录URL
// 5.2  记录当前地址为redirect_url
// 6. 用户点击微信登录
// 7. 后端收到code
// 7.1  获取用户信息，设置session为已登录
// 7.2  返回redirect_url，形式为angular_root/route_path
// 8. 前端请求非开放页面或接口
func Authenticate(ctx context.Context) (err error) {
	q := GetCtxValue(ctx)
	authenticated, _ := q.Session.Values["Authenticated"].(bool)

	defer func() {
		err = q.Err
	}()

	if authenticated {
		q.Err = setUserDomain(ctx)
		if q.Err != nil {
			q.Stop = true
			q.RespErr()
			return
		}
	}

	//白名单则无须登录
	if q.Ep.WhiteList {
		z.Info(q.R.URL.Path + " is whiteList")
		return
	}

	if !authenticated {
		//未登录，则要求重新登录
		q.Stop = true
		if rIsAPI.MatchString(q.R.URL.Path) {
			q.Msg.Status = -888
			q.Err = fmt.Errorf("请登录")
			//q.Msg.Data = types.JSONText(fmt.Sprintf(`{"wxLoginURL":"%s"}`, GetWxLoginURL(ctx)))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		z.Warn("重定向用户到微信登录")
		//http.Redirect(q.W, q.R, GetWxLoginURL(ctx), http.StatusSeeOther)
		return
	}

	if q.SysUser == nil || !q.SysUser.ID.Valid || q.SysUser.ID.Int64 < 0 {
		q.Stop = true
		q.Err = fmt.Errorf("无效的q.SysUser|sysUser.ID")
		z.Error(q.Err.Error())
		return
	}

	//判断用户是否有权限访问相应的api
	q.Err = AutoRole(ctx)
	if q.Err != nil {
		q.Stop = true
		q.RespErr()
		return
	}

	s := `select user_domain_id,domain_api_id,user_name,official_name,
			api_name,api_expose_path,domain_id,domain_name,domain
			from v_user_domain_api where user_id=$1 and api_expose_path=$2`

	var uda []TVUserDomainAPI

	q.Err = pgxscan.Select(ctx, pgxConn, &uda, s, q.SysUser.ID.Int64, q.Ep.Path)
	if pgxscan.NotFound(q.Err) || len(uda) == 0 {
		q.Stop = true
		q.Msg.Status = -1000

		q.Err = fmt.Errorf("%d无权访问%s", q.SysUser.ID.Int64, q.Ep.Path)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if q.Err != nil {
		q.Stop = true
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	//在此要处理一个用户两个角色都拥有同一个接口，但不同接口的数据权限不同
	//		_cDomainSysUser(377): 只能访问自己创建的数据
	//		_cDomainXKBSchoolAdmin(10008): 可以由t_relation获取数据授权, 此时
	//只能通过q.ReqFnType来确定角色的选取
	if len(uda) == 1 {
		if !uda[0].DomainID.Valid || uda[0].DomainID.Int64 <= 0 {
			q.Err = fmt.Errorf("invalid domainID on ud=%d,da=%d",
				uda[0].UserDomainID.Int64, uda[0].DomainAPIID.Int64)
			z.Error(q.Err.Error())
			return
		}

		//该角色有权限
		q.Role = uda[0].DomainID.Int64
		return
	}

	var determined bool
outer:
	switch q.ReqFnType {
	case CFuncNonAdminFileServe:
		//前台普通用户模块
		if q.Role == int64(CDomainSysUser) {
			determined = true
			break
		}
		for _, v := range q.Domains {
			if v.ID.Int64 == int64(CDomainSysUser) {
				q.Role = v.ID.Int64
				determined = true
				break outer
			}
		}
	case CFuncAdminFileServe:
		//后台管理员模块
		if q.Role != int64(CDomainSysUser) {
			determined = true
			break
		}
		for _, v := range q.Domains {
			if v.ID.Int64 != int64(CDomainSysUser) && v.ID.Int64 != int64(CDomainXKBUser) {
				//找到第一个不是前端的角色
				q.Role = v.ID.Int64
				determined = true
				break
			}
		}
	default:
		panic("unhandled default case")
	}

	if !determined {
		q.Err = fmt.Errorf("不能确定用户(%d)对功能(%s)的访问权限,q.ReqFnType=%d, 请核查",
			q.SysUser.ID.Int64, q.Ep.Path, q.ReqFnType)
		z.Error(q.Err.Error())
		q.Stop = true
		return
	}

	z.Debug(fmt.Sprintf("auth:user(%d), %s, %s",
		q.SysUser.ID.Int64, RoleName(CDomain(q.Role)), q.Ep.Path))

	return
}

// ACS Authentication/Authorization Control System
func ACS(ctx context.Context) {
	q := GetCtxValue(ctx)

	z.Info("---->" + FncName())
	q.Stop = true
	switch strings.ToUpper(q.R.Method) {
	case "GET":
		qryType := q.R.URL.Query().Get("t")
		if qryType == "" {
			q.Err = fmt.Errorf("请提供查询类型")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		switch qryType {

		case "u": // user, must supply name for search
			name := q.R.URL.Query().Get("n")
			if name == "" {
				q.Resp()
				return
			}

			s := `select u.id,id_card_no,id_card_type,email,nickname,mobile_phone,account,u.status,official_name,
       coalesce( official_name,nickname,mobile_phone,email,account,u.id::text) as fuse_name
				from t_user u`

			w := `official_name ilike $1
				or nickname ilike $1
				or mobile_phone ilike $1
				or account ilike $1
				or u.id::text ilike $1
				or id_card_no ilike $1`

			all := q.R.URL.Query().Get("all") == "true"
			if all {
				//全部用户
				s = s + " where " + w
			} else {
				//非客户, 需要把普遍用户*^{user,anonymous}过滤掉
				s = s + ` join t_domain_asset da on da.r_type='ud' and asset_id=u.id
  				join t_domain d on d.id=da.domain_id where (` + w + `) and (
    			d.domain not like '%^anonymous'
    			and  d.domain not like '%^user'
    			and d.domain  like '%^%'
				)`
			}

			var users []TUser
			q.Err = pgxscan.Select(ctx, pgxConn, &users, s, "%"+name+"%")
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var r []string
			for _, v := range users {
				var buf []byte
				buf, q.Err = MarshalJSON(&v)
				r = append(r, string(buf))
			}
			q.Msg.RowCount = int64(len(r))
			q.Msg.Data = []byte("[" + strings.Join(r, ",") + "]")
			q.Resp()
			return

		case "d": // domain
			s := `select id,name,domain	from t_domain	order by name`
			var d []TDomain
			q.Err = pgxscan.Select(ctx, pgxConn, &d, s)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var r []string
			for _, v := range d {
				var buf []byte
				buf, q.Err = MarshalJSON(&v)
				r = append(r, string(buf))
			}
			q.Msg.RowCount = int64(len(r))
			q.Msg.Data = []byte("[" + strings.Join(r, ",") + "]")
			q.Resp()
			return

		case "a": // api
			var a []TAPI
			s := `select id,name,expose_path,access_control_level,status,update_time from t_api	order by name`
			q.Err = pgxscan.Select(ctx, pgxConn, &a, s)

			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var r []string
			for _, v := range a {
				var buf []byte
				buf, q.Err = MarshalJSON(&v)
				r = append(r, string(buf))
			}
			q.Msg.RowCount = int64(len(r))
			q.Msg.Data = []byte("[" + strings.Join(r, ",") + "]")
			q.Resp()
			return

		}

		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("请提供查询条件")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var req ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		f := filter{Action: "select"}
		switch qryType {
		case "domain.asset":
			f.TableMap = &TVDomainAsset{Filter: f}

		case "domain.user":
			f.TableMap = &TVDomainUser{Filter: f}

		case "user.domain":
			f.TableMap = &TVUserDomain{Filter: f}

		case "domain.api":
			f.TableMap = &TVDomainAPI{Filter: f}

		case "api.domain":
			f.TableMap = &TVAPIDomain{Filter: f}

		case "api":
			f.TableMap = &TAPI{Filter: f}

		case "user":
			f.TableMap = &TUser{Filter: f}

		case "domain":
			f.TableMap = &TDomain{Filter: f}

		case "user.domain.api":
			f.TableMap = &TVUserDomainAPI{Filter: f}

		case "domain.insure":
			f.TableMap = &TRelation{Filter: f}

		default:
			q.Err = fmt.Errorf("unknown target: %s", qryType)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Err = DML(&f, &req)
		if q.Err != nil {
			q.RespErr()
			return
		}
		v, ok := f.QryResult.(string)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be string, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.RowCount = f.RowCount
		q.Msg.Data = types.JSONText(v)
		q.Resp()
		return

	case "PUT":

		//qry := q.R.URL.Query().Get("q")
		//if qry == "" {
		//	q.Err = fmt.Errorf("这是个啥子意思")
		//	z.Error(q.Err.Error())
		//	q.RespErr()
		//	return
		//}
		//z.Info(qry)
		//
		qryType := q.R.URL.Query().Get("t")
		if qryType == "" {
			q.Err = fmt.Errorf("请提供动作类型")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//var req ReqProto
		//q.Err = json.Unmarshal([]byte(qry), &req)
		//if q.Err != nil {
		//	z.Error(q.Err.Error())
		//	q.RespErr()
		//	return
		//}
		//
		//f := filter{Action: "update"}
		relationType := ""
		assetID := int64(-1)
		domainID := int64(-1)

		switch qryType {
		case "domain.user":
			s := q.R.URL.Query().Get("d")
			if s == "" {
				q.Err = fmt.Errorf("invalid domain")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			domainID, q.Err = strconv.ParseInt(s, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			s = q.R.URL.Query().Get("u")
			if s == "" {
				q.Err = fmt.Errorf("invalid user")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			assetID, q.Err = strconv.ParseInt(s, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			relationType = "ud"
			fallthrough
		case "domain.api":
			if relationType == "" {
				relationType = "da"
			}
			s := q.R.URL.Query().Get("d")
			if s == "" {
				q.Err = fmt.Errorf("invalid domain")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			domainID, q.Err = strconv.ParseInt(s, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if assetID < 0 {
				s = q.R.URL.Query().Get("a")
				if s == "" {
					q.Err = fmt.Errorf("invalid api")
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				assetID, q.Err = strconv.ParseInt(s, 10, 64)
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			s = `insert into t_domain_asset(domain_id,r_type,asset_id,creator) values($1,$2,$3,$4)
            on conflict (r_type, domain_id,asset_id) do update
            set domain_id=excluded.domain_id 
            where t_domain_asset.domain_id=excluded.domain_id
            returning id`

			r := pgxConn.QueryRow(context.Background(), s, domainID, relationType, assetID, q.SysUser.ID.Int64)

			var id sql.NullInt64
			q.Err = r.Scan(&id)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			s = fmt.Sprintf(`{"ID":%d}`, id.Int64)
			z.Info(s)
			q.Msg.Data = []byte(s)
			q.Resp()
			return

		case "api":
			//f.TableMap = &TAPI{}

		case "domain":
			//f.TableMap = &TDomain{}

		case "auth.data":
			//f.TableMap = &TRelation{}

		default:
			q.Err = fmt.Errorf("unknown/unsupport target: %s", qryType)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//q.Err = DML(&f, &req)
		//if q.Err != nil {
		//	q.RespErr()
		//	return
		//}
		//v, ok := f.QryResult.(int64)
		//if !ok {
		//	q.Err = fmt.Errorf("s.qryResult should be int64, but it isn't")
		//	z.Error(q.Err.Error())
		//	q.RespErr()
		//	return
		//}
		//
		//q.Msg.RowCount = v

		q.Resp()
		return

	case "POST":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("这是个啥子意思")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		z.Info(qry)

		qryType := q.R.URL.Query().Get("t")
		if qryType == "" {
			q.Err = fmt.Errorf("请提供动作类型")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		qry, q.Err = sjson.Delete(qry, "data.ID")
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var req ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		f := filter{Action: "insert"}
		switch qryType {
		case "user.domain":
			f.TableMap = &TUserDomain{Creator: q.SysUser.ID}

		case "domain.api":
			f.TableMap = &TDomainAPI{Creator: q.SysUser.ID}

		case "api":
			f.TableMap = &TAPI{Creator: q.SysUser.ID}

		case "domain":
			f.TableMap = &TDomain{Creator: q.SysUser.ID}

		case "auth.data", "domain.insure":
			f.TableMap = &TRelation{Creator: q.SysUser.ID}

		default:
			q.Err = fmt.Errorf("unknown/unsupport target: %s", qryType)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Err = DML(&f, &req)
		if q.Err != nil {
			q.RespErr()
			return
		}
		v, ok := f.QryResult.(int64)
		if !ok {
			q.Err = fmt.Errorf("s.qryResult should be int64, but it isn't")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.RowCount = 1
		q.Msg.Data = []byte(fmt.Sprintf(`{"ID":%d}`, v))
		q.Resp()
		return

	case "DELETE":
		qryType := q.R.URL.Query().Get("t")
		if qryType == "" {
			q.Err = fmt.Errorf("请提供目标类型")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		s := q.R.URL.Query().Get("id")
		if s == "" {
			q.Err = fmt.Errorf("请提供目标编号")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var objID int64
		objID, q.Err = strconv.ParseInt(s, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if objID <= 0 {
			q.Err = fmt.Errorf("请提供目标编号")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var values []interface{}
		values = append(values, objID)

		var ownerID int64

		s = q.R.URL.Query().Get("p")
		if s != "" {
			ownerID, q.Err = strconv.ParseInt(s, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if ownerID <= 0 {
				q.Err = fmt.Errorf("请提供目标编号")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			values = append(values, ownerID)
		}

		var actions []string
		switch qryType {
		case "domain.user":
			// ownerID: domain , objID: user
			actions = append(actions, "delete from t_user_domain where domain=$2 and sys_user=$1")

		case "user.domain":
			if ownerID > 0 {
				// ownerID: user , objID: domain
				actions = append(actions, "delete from t_user_domain where domain=$1 and sys_user=$2")
			} else {
				// objID: t_user_domain.id
				actions = append(actions, "delete from t_user_domain where id=$1")
			}

		case "domain.api":
			if ownerID > 0 {
				// ownerID: domain , objID: api
				actions = append(actions, "delete from t_domain_api where domain=$2 and api=$1")
			} else {
				// objID: t_domain_api.id
				actions = append(actions, "delete from t_domain_api where id=$1")
			}

		case "api.domain":
			// ownerID: api , objID: domain
			actions = append(actions, "delete from t_domain_api where domain=$1 and api=$2")

		case "api":
			actions = append(actions,
				"delete from t_domain_api where api=$1",
				"delete from t_api where id=$1",
			)

		case "domain":
			actions = append(actions,
				"delete from t_domain_api where domain=$1",
				"delete from t_user_domain where domain=$1",
				"delete from t_domain where id=$1",
			)

		case "user":
			actions = append(actions,
				"delete from t_user_domain where sys_user=$1",
				`update t_user set status='02' where id=$1`,
			)

		case "domain.insure":
			actions = append(actions,
				"delete from t_relation where id=$1",
			)

		default:
			q.Err = fmt.Errorf("unknown target: %s", qryType)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var rowAffected int64
		for i := 0; i < len(actions); i++ {
			var r sql.Result
			r, q.Err = sqlxDB.Exec(actions[i], values...)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			rowAffected, q.Err = r.RowsAffected()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if rowAffected <= 0 {
				z.Warn(fmt.Sprintf("could not find obj/relation:%s with id=%d, ownerID=%d", qryType, objID, ownerID))
			}
		}

		q.Msg.RowCount = rowAffected
		q.Msg.Data = types.JSONText(fmt.Sprintf(`{"stuff":"%s","ownerID":%d,"id":%d}`,
			qryType, ownerID, objID))
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unknown method: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

}

/*
	****************** *

一、概览
t_user定义用户基本属性
t_domain定义用户域、角色
t_user_domain定义用户归属域
t_api定义功能
t_access_control定义用户域(domain)可访问的功能(api)
t_relation定义数据关系

二、授权类型
1 不同用户u，同一被访问数据对象d

	1.1 d.e(exclusive)排他性授权，同一数据对象只能有一个同类型授权，如同一学校只能有一个销售(sale)、一个管理员(admin)
	1.2 d.c(coexist)可共存授权，同一数据对象可以有多个不同用户的同类型授权，如同一学校可有多个统计员(statistics)

2 同一用户u，同一被访问数据对象d，不允许多个授权。要通过创建“包含性授权”来表达数据范围"交叉"、"包含"问题。

3 同一用户，与被访问数据对象d无关

	1）s.e(exclusive)排他性授权，几种互斥授权类型中用户只能选一个，如管理员(^admin)、(^user)、(xkb^user)
	2）s.c(coexist)可共存授权，几种授权类型中用户能多选，如xkb^admin、abilityIdx^admin、ddxt^user

4 用户学校授权说明

	xkb^sale是d.e
	xkb.school^admin是d.e
	xkb.school^statistics是d.c

5 用户学校授权流程

	1）xkb^sale授权u,d
	A）删除u自己之前对d的授权
	B）删除其它u对d的授权
	C）授权

	2）xkb.school^admin的授权u,d
	A）删除u自己之前对d的授权
	B）删除其它u对d的授权
	C）授权

	3）xkb.school^statistics的授权u,d
	A）删除u自己之前对d的授权
	B）授权

三、说明
1）t_user.category只代表用户最近使用的授权域，不代表用户的所有授权域；
2）用户的所有授权域用 t_user_domain.{user,domain) 表示
3）用户可以基于

	微信公众号: t_wx_user.wx_open_ID（无密码）
	微信开放平台: t_wx_user.mp_open_ID（无密码）,
	系统编号: t_user.ID(t_user.cert)
	系统帐号: t_user.account(t_user.cert)
	电话: t_user.mobile_phone(t_user.cert)
	邮箱: t_user.email(t_user.cert)
	机构工号: t_user_domain.ID_on_domain(t_user.cert)
	登录系统

访问控制实现层级, t_api.access_control_level

	level 0: 无组/角色/数据限制
	level 2: 机构#角色级别, 实现了不同角色授权，但不控制数据范围
	level 4: 机构#角色$ID, 实现了不同角色授权，可控制 creator || all
	level 8: 机构.DEPT#角色$ID, 实现了不同角色授权，可控制 creator || GRPs

* ********************/

func InitAuth() (err error) {
	if len(Services) == 0 {
		err = fmt.Errorf("call initAuth with empty services")
		z.Error(err.Error())
		return
	}

	initAuth := false
	if viper.IsSet("aa.init") {
		initAuth = viper.GetBool("aa.init")
	}
	if !initAuth {
		return
	}

	// 把API信息存储到t_api中
	s := `insert into t_api(name,expose_path,creator,
        access_control_level,maintainer,domain_id,addi) 
			 values($1,$2,$3,$4,$5,$6,'{"init":true}') 
			 on conflict(name) do update
				set expose_path=excluded.expose_path
			 RETURNING ID`

	tx, err := sqlxDB.Begin()
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()

	createApiStmt, err := tx.Prepare(s)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer func() { _ = createApiStmt.Close() }()

	// 授权每一个平台API给sys^admin
	s = `with cte as(select id from t_domain where domain='sys^admin')
		insert into t_domain_asset(r_type,status,domain_id,asset_id,creator)
		select 'da','01',cte.id,$1,$2 from cte
		on conflict (r_type,domain_id,asset_id) do update
      set creator=excluded.creator,
			domain_id=excluded.domain_id,status=excluded.status
		returning id`
	createDomainAssetStmt, err := tx.Prepare(s)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer func() { _ = createDomainAssetStmt.Close() }()

	for _, v := range Services {
		//v := Services[k]
		if v.AccessControlLevel == "" {
			// level 0: 无组/角色/数据限制
			// level 2: 机构#角色级别, 实现了不同角色授权，但不控制数据范围
			// level 4: 机构#角色$ID, 实现了不同角色授权，可控制 creator || all
			// level 8: 机构.DEPT#角色$ID, 实现了不同角色授权，可控制 creator || GRPs
			v.AccessControlLevel = "0"
		}
		//insert into t_api(name,expose_path,creator,access_control_level,maintainer,domain_id) RETURNING ID
		r := createApiStmt.QueryRow(v.Name, v.Path, v.MaintainerID,
			v.AccessControlLevel, v.MaintainerID, v.DomainID)

		var id int64
		err = r.Scan(&id)
		if err != nil {
			z.Error(err.Error())
			return
		}

		//`insert into t_domain_asset(r_type,status,domain_id,asset_id,creator)
		//		values('da','01',cte.id,$1,$2)
		r = createDomainAssetStmt.QueryRow(id, v.MaintainerID)
		err = r.Scan(&id)
		if err != nil {
			z.Error(err.Error())
			return
		}
		fmt.Println(fmt.Sprintf("%s: %s", v.Name, v.Path))
	}

	err = tx.Commit()
	if err != nil {
		z.Error(err.Error())
		return
	}
	//sqlxDB.MustExec(aaInitFile)
	return
}
