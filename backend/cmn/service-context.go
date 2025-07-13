package cmn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ModuleAuthor struct {
	Name  string `json:"name,omitempty"`
	Tel   string `json:"tel,omitempty"`
	Email string `json:"email,omitempty"`
	Addi  string `json:"addi,omitempty"`
}

type ctxKey string

func (v ctxKey) String() string {
	return string(v)
}

const (
	CUnknownCaller       = iota
	CPcBrowserCaller     = 1 << 0
	CAndroidWxCaller     = 1 << 1
	CIOSWxCaller         = 1 << 2
	CMobileBrowserCaller = 1 << 3
	CMacWxCaller         = 1 << 4
	CWinWxCaller         = 1 << 5

	CUnknownWxCaller = 1 << 7
)

func GetCallerTypeName(i int) string {
	switch i {
	case CUnknownCaller:
		return "unknown"

	case CUnknownWxCaller:
		return "unknownWx"

	case CAndroidWxCaller:
		return "androidWx"

	case CIOSWxCaller:
		return "iOSWx"

	case CMacWxCaller:
		return "macWx"

	case CWinWxCaller:
		return "winWx"

	case CPcBrowserCaller:
		return "pcBrowser"

	case CMobileBrowserCaller:
		return "mobileBrowser"

	default:
		return "unknown"
	}
}

type cServiceCtx struct {
	Err  error // error occurred during process
	Stop bool  // should run next process

	Attacker  bool // the requester is an attacker
	WhiteList bool // the request path in white list

	Ep *ServeEndPoint

	//stack *stack

	Responded bool // Dose response written

	Session *sessions.Session // gorilla cookie's

	Redis redis.Conn

	W http.ResponseWriter
	R *http.Request

	DomainList []string

	Domains []TDomain

	//角色中是否有管理员角色
	IsAdmin bool

	//是否在请求的URL中包含了admin=true
	ReqAdminFnc bool

	WxUser  *TWxUser
	SysUser *TUser
	Msg     *ReplyProto

	CallerType int

	UserAgent string

	WxLoginProcessed bool

	//xkb *xkbCtx
	//reqScope map[string]interface{} // session variables

	TouchTime time.Time

	Channel chan []byte

	RoutineID int
	BeginTime time.Time

	Tag map[string]interface{}

	//用户访问系统所使用的角色
	Role int64

	//用户访问的模块类型: 未知类型，函数，同未知类型，后台管理员模块，前台普通用户模块
	ReqFnType int
}

type ServiceCtx struct {
	Err  error // error occurred during process
	Stop bool  // should run next process

	Attacker  bool // the requester is an attacker
	WhiteList bool // the request path in white list

	Ep *ServeEndPoint

	//stack *stack

	Responded bool // Dose response written

	Session *sessions.Session // gorilla cookie's

	Redis redis.Conn

	W http.ResponseWriter
	R *http.Request

	DomainList []string

	Domains []TDomain

	//角色中是否有管理员角色
	IsAdmin bool

	//是否在请求的URL中包含了admin=true
	ReqAdminFnc bool

	WxUser  *TWxUser
	SysUser *TUser
	Msg     *ReplyProto

	CallerType int

	UserAgent string

	WxLoginProcessed bool

	//xkb *xkbCtx
	//reqScope map[string]interface{} // session variables

	TouchTime time.Time

	Channel chan []byte

	RoutineID int
	BeginTime time.Time

	Tag map[string]interface{}

	//用户访问系统所使用的角色
	Role int64

	//用户访问的模块类型: 未知类型，函数，同未知类型，后台管理员模块，前台普通用户模块
	ReqFnType int
}

func (v *ServiceCtx) RespErr() {
	if v.Responded {
		z.Error("responded")
		return
	}

	// for test session without http context
	if v.W == nil {
		return
	}

	v.Responded = true
	v.Stop = true
	if v.Err == nil {
		v.Err = fmt.Errorf("v.err is nil")
	}

	if v.Msg == nil {
		v.Msg = &ReplyProto{
			API:    v.R.URL.Path,
			Method: v.R.Method,
		}
	}

	v.Msg.Msg = v.Err.Error()
	if v.Msg.Status == 0 {
		v.Msg.Status = -1
	}

	//-410xx都是权限或账号错，需要清除session后重新与数据库同步
	if (v.Msg.Status / 100) == -410 {
		CleanSession(context.WithValue(context.Background(),
			QNearKey, v))
	}

	buf, err := json.Marshal(v.Msg)
	if err != nil {
		z.Error(err.Error())
		_, _ = fmt.Fprintf(v.W, err.Error())
		return
	}

	s := string(buf)
	if len(v.Msg.Data) > 0 {
		trial := fmt.Sprintf(`{"trial":%s}`, string(v.Msg.Data))
		t := make(map[string]interface{})

		v.Err = json.Unmarshal([]byte(trial), &t)
		if v.Err != nil {
			z.Error(trial)
			z.Error(v.Err.Error())
			v.RespErr()
			return
		}
		s = s[:len(buf)-1] + `,"data":` + string(v.Msg.Data) + "}"
	}

	v.W.Header().Add("Content-Type", "application/json")
	_, _ = fmt.Fprintf(v.W, s)
}

func (v *ServiceCtx) Resp() {
	if v.Err != nil {
		v.RespErr()
		return
	}

	if v.Responded {
		z.Error("responded")
		return
	}

	// for test session without http context
	if v.W == nil {
		return
	}

	if v.Msg == nil {
		v.Msg = &ReplyProto{
			API:    v.R.URL.Path,
			Method: v.R.Method,
		}
		v.Err = errors.New("v.Msg is nil")
		v.RespErr()
		return
	}

	buf, err := json.Marshal(v.Msg)
	if err != nil {
		z.Error(err.Error())
		_, _ = fmt.Fprintf(v.W, err.Error())
		return
	}

	s := string(buf)
	if len(v.Msg.Data) > 0 {
		trial := fmt.Sprintf(`{"trial":%s}`, string(v.Msg.Data))
		t := make(map[string]interface{})

		v.Err = json.Unmarshal([]byte(trial), &t)
		if v.Err != nil {
			v.Msg.Data = nil
			z.Error(trial)
			z.Error(v.Err.Error())
			v.RespErr()
			return
		}
		s = s[:len(buf)-1] + `,"data":` + string(v.Msg.Data) + "}"
	}

	v.W.Header().Add("Content-Type", "application/json")
	_, _ = fmt.Fprintf(v.W, "%s", s)

	v.Responded = true
}

const QNearKey = ctxKey("ServiceCtx")

func GetCtxValue(ctx context.Context) (q *ServiceCtx) {
	var err error
	f := ctx.Value(QNearKey)
	if f == nil {
		err = fmt.Errorf(`get nil from ctx.Value["%s"]`, QNearKey.String())
		z.Error(err.Error())
		panic(err.Error())
	}
	var ok bool
	q, ok = f.(*ServiceCtx)
	if !ok {
		err := fmt.Errorf("failed to type assertion for *ServiceCtx")
		z.Error(err.Error())
		panic(err.Error())
	}
	if q == nil {
		err := fmt.Errorf(`ctx.Value["%s"] should be non nil *ServiceCtx`, QNearKey.String())
		z.Error(err.Error())
		panic(err.Error())
	}
	return
}

func XCleanSession(w http.ResponseWriter, _ *http.Request) {
	cookie := http.Cookie{
		Name:   "qNearSession",
		Value:  "",
		Domain: viper.GetString("webServe.serverName"),
		Path:   "/",
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)
}

var (
	Services = make(map[string]*ServeEndPoint)

	serviceMutex sync.Mutex

	AttackerList = make(map[string]bool)
)

var rIsAPI = regexp.MustCompile(`(?i)^/api/(.*)?$`)

func ApiList() (data []byte, err error) {

	var a []*ServeEndPoint
	for _, v := range Services {
		a = append(a, v)
	}

	sort.Slice(a, func(i, j int) bool {
		return a[i].Path <= a[j].Path
	})

	data, err = json.Marshal(a)
	return
}

func AddService(ep *ServeEndPoint) (err error) {
	for {
		if ep == nil {
			err = errors.New("ep is nil")
			break
		}

		if ep.Path == "" {
			err = errors.New("ep.path empty")
			break
		}

		if ep.PathPattern == "" {
			ep.PathPattern = fmt.Sprintf(`(?i)^%s(/.*)?$`, ep.Path)
		}
		ep.PathMatcher = regexp.MustCompile(ep.PathPattern)

		if ep.IsFileServe {
			if ep.DocRoot == "" {
				err = errors.New("must specify docRoot when ep.isFileServe equal true")
				break
			}

			if ep.Fn == nil {
				ep.Fn = WebFS
			}
		} else {
			if ep.Fn == nil {
				err = errors.New("must specify fn when ep.isFileServe equal false")
				break
			}

			if !rIsAPI.MatchString(ep.Path) {
				ep.Path = strings.ReplaceAll("/api/"+ep.Path, "//", "/")
			}
		}

		if ep.Name == "" {
			err = errors.New("must specify apiName")
			break
		}

		if ep.DomainID == 0 {
			ep.DomainID = int64(CDomainSys)
		}
		if ep.DefaultDomain == 0 {
			ep.DefaultDomain = int64(CDomainSys)
		}

		if ep.AccessControlLevel == "" {
			ep.AccessControlLevel = "0"
		}
		_, ok := Services[ep.Path]
		if ok {
			err = errors.New(fmt.Sprintf("%s[%s] already exists", ep.Path, ep.Name))
		}
		break
	}

	if err != nil {
		z.Error(err.Error())
		return
	}

	z.Info(ep.Name + " added")

	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	Services[ep.Path] = ep
	return
}

func BuildURL(r *http.Request) (dst string) {
	if r == nil {
		return
	}

	scheme := "https:"
	host := viper.GetString("webServe.serverName")

	if r.URL.Scheme != "" {
		dst = r.URL.Scheme + "//"
	} else {
		dst = scheme + "//"
	}
	if r.URL.Host != "" {
		dst = dst + r.URL.Host
	} else {
		dst = dst + host
	}
	if r.URL.Path != "" {
		dst = dst + r.URL.Path
	}
	if r.URL.RawQuery != "" {
		dst = dst + "?" + r.URL.RawQuery
	}
	if r.URL.Fragment != "" {
		dst = dst + "#" + r.URL.Fragment
	}
	return
}

func guessIdxFile(f string) []string {
	f = filepath.Clean(f)

	var idxList []string
	for {
		f, _ = filepath.Split(f)
		idxList = append(idxList, f+"index.html")
		n := filepath.Clean(f)
		if f != "" && f != n {
			f = n
			continue
		}
		break
	}
	return idxList
}

const webFilePattern = `^(/*\S+)*/*\S+\.\S+$`

var rWebFilePattern = regexp.MustCompile(webFilePattern)

/*
WebFS static file serve
1 if found the request file then return it,
2 if we can not find the target and the q.Ep.PageRoute is true then return the guessed index.html,
3 else return 404 not found
*/
func WebFS(ctx context.Context) {
	q := GetCtxValue(ctx)
	q.Responded = true
	q.Stop = true
	z.Info("---->" + FncName())

	if q.Ep == nil {
		q.Err = fmt.Errorf("call jsFS with nil q.Ep")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if q.Ep.DocRoot == "" {
		q.Err = fmt.Errorf("call jsFS with empty q.Ep.docRoot")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if len(q.R.URL.Path) < len(q.Ep.Path) {
		q.Err = fmt.Errorf("len(q.R.URL.Path) < len(q.Ep.path), it shouldn't happen")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	//---------
	if origin := q.R.Header.Get("Origin"); origin != "" {
		q.W.Header().Set("Access-Control-Allow-Origin", origin)
		q.W.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		q.W.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		q.W.Header().Set("Vary", "Origin")
		q.W.Header().Set("Access-Control-Allow-Credentials", "true")

		// Stop here if its Preflighted OPTIONS request
		if q.R.Method == "OPTIONS" {
			return
		}
	}

	//---------
	// physical file root directory
	fsRoot := filepath.Clean(q.Ep.DocRoot)

	// API url path
	epPath := strings.TrimSuffix(q.Ep.Path, "/")

	// Request URL
	urlPath := strings.TrimSuffix(q.R.URL.Path, "/")

	// relative path include file name
	f := strings.TrimPrefix(urlPath[len(epPath):], "/")

	// physical path
	targetFileName := filepath.Clean(fsRoot + string(os.PathSeparator) + f)

	fileInfo, err := os.Stat(targetFileName)
	if os.IsNotExist(err) {
		z.Warn("InExistence: " + targetFileName)

		if rWebFilePattern.Match([]byte(f)) || !q.Ep.PageRoute {
			// missing the request specific file or non page route app.
			http.NotFound(q.W, q.R)
			return
		}

		// request is a path and q.Ep.PageRoute is true

		var idxHtmlFound bool
		idxList := guessIdxFile("/" + f)
		for _, v := range idxList {
			targetFileName = fsRoot + v
			_, err := os.Stat(targetFileName)
			if os.IsNotExist(err) {
				continue
			}

			if err != nil {
				q.Err = err
				z.Error(err.Error())
				q.Responded = false
				q.RespErr()
				return
			}
			idxHtmlFound = true
			break
		}

		if !idxHtmlFound {
			http.NotFound(q.W, q.R)
			return
		}
	}

	if err == nil && fileInfo.IsDir() && !q.Ep.AllowDirectoryList {
		idxHTML := filepath.Clean(targetFileName + string(os.PathSeparator) + "index.html")
		if _, err := os.Stat(idxHTML); os.IsNotExist(err) {
			q.W.WriteHeader(http.StatusForbidden)
			_, _ = q.W.Write([]byte("Access to the resource is forbidden!"))
			return
		}
	}

	http.ServeFile(q.W, q.R, targetFileName)
}

func EpByPath(path string) (ep *ServeEndPoint, isPathValid bool) {
	if path == "/" {
		ep, isPathValid = Services[path]
		return
	}

	var parts []string
	for _, n := range strings.Split(path, "/") {
		if n == "" {
			continue
		}
		parts = append(parts, n)
	}

	for len(parts) > 0 {
		p := "/" + strings.Join(parts, "/")
		ep, isPathValid = Services[p]

		if isPathValid {
			break
		}
		ep, isPathValid = Services[p+"/"]
		if isPathValid {
			break
		}
		parts = parts[:len(parts)-1]
	}
	return
}

func IsAttacker(path string, remoteAddr string) (ep *ServeEndPoint, isAttacker, whiteList, isPathValid bool) {
	if path == "" || remoteAddr == "" {
		z.Error("call isAttacker with empty path | remoteAddr")
		return
	}

	ep, isPathValid = EpByPath(path)
	// [::1]:54582, 127.0.0.1:54582
	addr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		z.Error(err.Error())
		isAttacker = true
		whiteList = false
		return
	}
	_, isAttackerAddr := AttackerList[addr]

	if !isPathValid {
		z.Info(fmt.Sprintf("request invalid path: %s, %s will be marked as attacker",
			path, addr))
		AttackerList[addr] = true
	}

	if isAttackerAddr {
		z.Info(addr + " is marked as attacker")
	}

	if isPathValid {
		whiteList = ep.WhiteList
	}

	isAttacker = (!isPathValid || isAttackerAddr) && AttackDefence
	return
}

func CleanCacheByUserID(userID int64) (err error) {
	if userID <= 0 {
		err = fmt.Errorf("zero/Invalid userID")
		z.Error(err.Error())
		return
	}

	r := GetRedisConn()
	var keys []interface{}
	for {
		key := fmt.Sprintf("%s:%d", SysUserByID, userID)
		var account string
		account, err = redis.String(r.Do("get", key))
		if err != nil {
			z.Error(err.Error())
			return
		}
		keys = append(keys, key)

		key = fmt.Sprintf("%s:%s", SysUserByName, account)
		var userData string
		userData, err = redis.String(r.Do("JSON.GET", key, "."))
		if err != nil {
			z.Error(err.Error())
			return
		}

		rX := gjson.Get(userData, "MobilePhone")
		if rX.Exists() && rX.Num > 0 {
			userData, _ = sjson.Set(userData, "MobilePhone",
				fmt.Sprintf("%d", int64(rX.Num)))
		}

		keys = append(keys, key)

		var u TUser
		err = json.Unmarshal([]byte(userData), &u)
		if err != nil {
			z.Error(err.Error())
			return
		}

		key = fmt.Sprintf("%s:%s", SysUserByTel, u.MobilePhone.String)

		account, err = redis.String(r.Do("get", key))
		if err != nil {
			z.Warn(err.Error())
		}
		if account != "" {
			keys = append(keys, key)
		}

		key = fmt.Sprintf("%s:%s", SysUserByEmail, u.Email.String)
		account, err = redis.String(r.Do("get", key))
		if err != nil {
			z.Warn(err.Error())
		}
		if account != "" {
			keys = append(keys, key)
		}

		key = fmt.Sprintf("%s:%d", WxUserByID, userID)
		var unionID string
		unionID, err = redis.String(r.Do("get", key))
		if err != nil {
			z.Error(err.Error())
			break
		}
		keys = append(keys, key)

		key = fmt.Sprintf("%s:%s", WxUserByUnionID, unionID)
		var wxUserData string
		wxUserData, err = redis.String(r.Do("JSON.GET", key, "."))
		if err != nil {
			z.Error(err.Error())
			break
		}
		keys = append(keys, key)

		var x TWxUser
		err = json.Unmarshal([]byte(wxUserData), &x)
		if err != nil {
			z.Error(err.Error())
			return
		}

		key = fmt.Sprintf("%s:%s", WxUserByOpenID, x.MpOpenID.String)
		account, err = redis.String(r.Do("get", key))
		if err != nil {
			z.Warn(err.Error())
		}
		if account != "" {
			keys = append(keys, key)
		}
		break
	}

	for k, v := range keys {
		z.Info(fmt.Sprintf("%d:%s", k, v))
	}

	var reply interface{}
	reply, err = r.Do("DEL", keys...)
	if err != nil {
		z.Error(err.Error())
		return
	}

	keysDropped, ok := reply.(int64)
	if !ok {
		err = fmt.Errorf("reply should be a int, %v", keysDropped)
		z.Error(err.Error())
		return
	}

	z.Info(fmt.Sprintf("user(%d) cache cleaned", userID))
	return
}

// EraseUser 抹除用户及其数据
// https://qnear.cn/api/dbStatus?xCleanSession=142857&erase=true
func EraseUser(userID int64) (err error) {
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
	tx, err := GetPgxConn().Begin(ctx)
	if err != nil {
		z.Error(err.Error())
		return
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil {
			z.Error(err.Error())
		}
	}()

	for _, v := range s {
		_, err = tx.Exec(ctx, v, userID)
		if err != nil {
			z.Error(err.Error())
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		z.Error(err.Error())
	}
	return
}

func CleanSession(ctx context.Context) {
	q := GetCtxValue(ctx)
	userID, _ := q.Session.Values["ID"].(int64)
	if userID <= 0 {
		q.Err = fmt.Errorf("invalid session")
		z.Error(q.Err.Error())
		return
	}
	defer func() {
		z.Warn(fmt.Sprintf("%d 's session has been cleaned", userID))
	}()

	q.Session.Options.MaxAge = -1
	for k := range q.Session.Values {
		delete(q.Session.Values, k)
	}

	q.Err = q.Session.Save(q.R, q.W)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	q.Err = CleanCacheByUserID(userID)
	if q.Err != nil {
		z.Error(q.Err.Error())
	}
	if strings.ToLower(q.R.URL.Query().Get("erase")) == "true" {
		q.Err = eraseUser(userID)
	}
}

func CacheSysUser(ctx context.Context, sysUser *TUser) {
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

	key := fmt.Sprintf("%s:%s", SysUserByName, sysUser.Account)
	_, q.Err = q.Redis.Do("JSON.SET", key, ".", string(buf))
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key = fmt.Sprintf("%s:%d", SysUserByID, sysUser.ID.Int64)
	_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	if sysUser.MobilePhone.Valid && sysUser.MobilePhone.String != "" {
		key = fmt.Sprintf("%s:%s", SysUserByTel, sysUser.MobilePhone.String)
		_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		z.Info("set cache by " + SysUserByTel)
	}

	if sysUser.Email.Valid && sysUser.Email.String != "" {
		key = fmt.Sprintf("%s:%s", SysUserByEmail, sysUser.Email.String)
		_, q.Err = q.Redis.Do("SET", key, sysUser.Account)
		if q.Err != nil {
			z.Error(q.Err.Error())
			return
		}
		z.Info("set cache by " + SysUserByEmail)
	}
}

func CacheWxUser(ctx context.Context, wxUser *TWxUser) {
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

	key := fmt.Sprintf("%s:%s", WxUserByUnionID, wxUser.UnionID.String)
	_, q.Err = q.Redis.Do("JSON.SET", key, ".", string(buf))
	if q.Err != nil {
		z.Error(q.Err.Error())
		return
	}

	key = fmt.Sprintf("%s:%d", WxUserByID, wxUser.ID.Int64)

	_, q.Err = q.Redis.Do("SET", key, wxUser.UnionID.String)
	if q.Err != nil {
		z.Error(q.Err.Error())
	}

	var haveValidOpenID bool
	if wxUser.MpOpenID.Valid && wxUser.MpOpenID.String != "" {
		key = fmt.Sprintf("%s:%s", WxUserByOpenID, wxUser.MpOpenID.String)
		_, q.Err = q.Redis.Do("SET", key, wxUser.ID.Int64)
		if q.Err != nil {
			z.Error(q.Err.Error())
		}
		haveValidOpenID = true
	}

	if wxUser.WxOpenID.Valid && wxUser.WxOpenID.String != "" {
		key = fmt.Sprintf("%s:%s", WxUserByOpenID, wxUser.WxOpenID.String)
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

func PrintSession(ctx context.Context) {
	q := GetCtxValue(ctx)
	kv := []string{"\n"}

	for k, v := range q.Session.Values {
		kv = append(kv, fmt.Sprintf("%16v: %v", k, v))
	}
	z.Info(strings.Join(kv, "\n"))

}

func ValidateCookie(ctx context.Context) (err error) {
	q := GetCtxValue(ctx)
	if q.Session.IsNew {
		return
	}

	if InDebugMode {
		PrintSession(ctx)
	}

	account, ok1 := q.Session.Values["Account"]

	accountStr, _ := account.(string)
	authenticated, ok2 := q.Session.Values["Authenticated"]

	authenticatedBool, _ := authenticated.(bool)

	if ok2 && authenticatedBool && (!ok1 || accountStr == "") {
		z.Error("authed but account is empty")
	}

	name, _ := q.Session.Values["Account"].(string)
	if name == "" {
		// it's from pc browser and access white list URL
		return
	}

	var key string
	var jsonStr string
	key = fmt.Sprintf("%s:%s", SysUserByName, name)
	jsonStr, err = redis.String(q.Redis.Do("JSON.GET", key, "."))

	if err != nil && err.Error() == RedisNil {
		CleanSession(ctx)
		q.Session.Options.MaxAge = 0
		_ = q.Session.Save(q.R, q.W)
		err = nil
		return
	}

	if err != nil {
		z.Error(err.Error())
		return
	}

	if jsonStr == "" {
		CleanSession(ctx)
		q.Session.Options.MaxAge = 0
		_ = q.Session.Save(q.R, q.W)
		err = nil
	}

	return
}

func BuildUserInfo(ctx context.Context) (authenticated bool, err error) {
	q := GetCtxValue(ctx)
	authenticated, _ = q.Session.Values["Authenticated"].(bool)
	if !authenticated {
		return
	}

	userDefaultRole, _ := q.Session.Values["Role"].(int)

	name, _ := (q.Session.Values["Account"]).(string)
	for {
		if name == "" {
			err = fmt.Errorf(`session.Values["Account"] is empty while authenticated`)
			z.Error(err.Error())
			break
		}

		var key string
		var jsonStr string
		key = fmt.Sprintf("%s:%s", SysUserByName, name)
		jsonStr, err = redis.String(q.Redis.Do("JSON.GET", key, "."))

		if err != nil && err.Error() == RedisNil {
			err = nil
		}
		if err != nil {
			z.Error(err.Error())
			break
		}

		if jsonStr == "" {
			CleanSession(ctx)
			authenticated = false
			break
		}

		var sysUser TUser
		r := gjson.Get(jsonStr, "MobilePhone")
		if r.Exists() && r.Num > 0 {
			jsonStr, q.Err = sjson.Set(jsonStr, "MobilePhone",
				fmt.Sprintf("%d", int64(r.Num)))
		}

		err = json.Unmarshal([]byte(jsonStr), &sysUser)
		if err != nil {
			z.Error(err.Error())
			q.Msg.Status = -200
			break
		}

		if userDefaultRole <= 0 {
			s := `select ` + UserTableSet + ` from t_user where id=$1`
			row := sqlxDB.QueryRowx(s, sysUser.ID.Int64)
			err = row.StructScan(&sysUser)
			if err != nil {
				z.Error(err.Error())
				break
			}

			if !sysUser.Role.Valid || sysUser.Role.Int64 <= 0 {
				err = fmt.Errorf("用户(%d).Role无效", sysUser.ID.Int64)
				z.Error(err.Error())
				break
			}
			q.Session.Values["Role"] = sysUser.Role.Int64
			_ = q.Session.Save(q.R, q.W)
		}

		err = InvalidEmptyNullValue(&sysUser)
		if err != nil {
			q.Msg.Status = -200
			break
		}

		if userDefaultRole <= 0 {
			_, err = q.Redis.Do("JSON.SET", key, ".Role", sysUser.Role.Int64)
			if err != nil {
				z.Error(err.Error())
				break
			}
			userDefaultRole = int(sysUser.Role.Int64)
		}

		q.SysUser = &sysUser
		q.Tag["callerID"] = -410100

		// ------ settle wxUser
		key = fmt.Sprintf("%s:%d", WxUserByID, sysUser.ID.Int64)
		var openID string
		openID, err = redis.String(q.Redis.Do("GET",
			fmt.Sprintf("%s:%d", WxUserByID, sysUser.ID.Int64)))

		if err != nil && err.Error() == RedisNil {
			err = nil
		}
		if err != nil {
			z.Error(err.Error())
			break
		}

		if openID == "" && (q.CallerType == CAndroidWxCaller ||
			q.CallerType == CIOSWxCaller) {

			err = fmt.Errorf("in WeiXin but missing wxUser at %s", key)
			z.Error(err.Error())
			q.Msg.Status = -5000
			break
		}

		if openID == "" {
			//it's pc browser user maybe using upLogin without weiXin login
			break
		}

		key = fmt.Sprintf("%s:%s", WxUserByUnionID, openID)
		jsonStr, err = redis.String(q.Redis.Do("JSON.GET",
			fmt.Sprintf("%s:%s", WxUserByUnionID, openID), "."))

		if err != nil && err.Error() == RedisNil {
			err = nil
		}
		if err != nil {
			z.Error(err.Error())
			break
		}

		if jsonStr == "" {
			err = fmt.Errorf("%s is empty in redis", key)
			break
		}

		var wxUser TWxUser
		err = json.Unmarshal([]byte(jsonStr), &wxUser)
		if err != nil {
			z.Error(err.Error())
			q.Msg.Status = -400
			break
		}

		err = InvalidEmptyNullValue(&wxUser)
		if err != nil {
			q.Msg.Status = -400
			break
		}

		switch q.CallerType {
		case CUnknownWxCaller, CWinWxCaller, CMacWxCaller,
			CIOSWxCaller, CAndroidWxCaller:
			if wxUser.MpOpenID.String == "" {
				q.Msg.Status = -888
				CleanSession(ctx)
				authenticated = false
				return
			}

		case CMobileBrowserCaller, CPcBrowserCaller:
			//q.session.Values["loginType"] = "upLogin"
			loginType := q.Session.Values["loginType"]

			if wxUser.WxOpenID.String == "" && loginType == "wxLogin" {
				q.Msg.Msg = "请使用微信扫码登录"
				q.Msg.Status = -888
				CleanSession(ctx)
				authenticated = false
				return
			}
		default:
			err = fmt.Errorf("不好意思，暂时不支持您所用的系统")
			authenticated = false
			return
		}

		q.WxUser = &wxUser
		break
	}
	return
}
