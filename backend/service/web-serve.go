package service

//go:generate go run service-enroll-generate.go -a=annotation:(?P<name>.*)-service

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/klauspost/compress/gzhttp"
	"github.com/klauspost/compress/gzip"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
	"w2w.io/cmn"
	"w2w.io/mux"
)

var (
	z *zap.Logger

	pgxConn *pgxpool.Pool
	sqlxDB  *sqlx.DB
	rConn   redis.Conn
)

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		pgxConn = cmn.GetPgxConn()
		sqlxDB = cmn.GetDbConn()
		rConn = cmn.GetRedisConn()

		z.Info("service zLogger settled")
	})
}

var store = sessions.NewCookieStore([]byte("aLongStory"))

func crashed(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	q.Stop = true
	r := recover()
	if r == nil {
		return
	}

	reader := bufio.NewReader(strings.NewReader(string(debug.Stack())))

	n := 7
	var panicStack []string
	for i := 0; ; i++ {
		line, _, err := reader.ReadLine()
		if i == n || i == n+1 {
			panicStack = append(panicStack, string(line))
		}
		if err != nil || i > n+1 {
			break
		}
	}

	templatePanicString := fmt.Sprintf("_CRLF_%s_CRLF_%s",
		strings.ReplaceAll(strings.Join(panicStack, "_CRLF_"), "\t", ""), r)

	s := strings.ReplaceAll(templatePanicString, "_CRLF_", "\n\t")
	webString := strings.ReplaceAll(templatePanicString, "_CRLF_", ", ")
	q.Err = fmt.Errorf(webString)
	z.Error(s)
	q.RespErr()
}

func restoreLog() {
	logLevel := 0
	if viper.IsSet("zLogLevel") {
		logLevel = viper.GetInt("zLogLevel")
	}
	cmn.SetLogLevel(int8(logLevel))
}

func disableLog(r *http.Request) bool {
	if r.URL.Query().Get("token") != "858f8dd898b75fe86926" {
		return false
	}

	//100无实际意义，仅表示一个足够大的数，使任何日志的级别也达不到，从而抑制日志输出
	cmn.SetLogLevel(100)
	return true
}

var rIsAPI = regexp.MustCompile(`(?i)^/api/(.*)?$`)

var (
	rWxIOS     = regexp.MustCompile(`(iPhone)(.*)(MicroMessenger)`)
	rWxAndroid = regexp.MustCompile(`(Android)(.*)(MicroMessenger)`)
	rMacWx     = regexp.MustCompile(`\(Macintosh; .*(?P<osVer> \d*_\d*_\d*\)).* MicroMessenger/(?P<wxVer>\d*\.\d*\.\d*)\((?P<wxVerHex>.*)\) MacWechat`)
	rWinWx     = regexp.MustCompile(`\(Windows \S* (?P<osVer>\d*\.\d*)(; )?(?P<subSys>\S*)\).* MicroMessenger/(?P<wxVer>\d*\.\d*\.\d*)`)
	rIsWx      = regexp.MustCompile(`MicroMessenger`)
	rMobile    = regexp.MustCompile(`(Android|iPhone)`)
)

var shortLiveMutex sync.Mutex
var door sync.Mutex

func WebServe(_ *cobra.Command, _ []string) {
	router := mux.NewRouter()

	// MUST before enroll
	cmn.OnlineUsers.Mux = router
	//-----
	Enroll()
	cmn.LoadPayAccount()
	err := cmn.InitAuth()
	if err != nil {
		z.Fatal(err.Error())
	}

	err = setupFileServeHandler(router)
	if err != nil {
		z.Fatal(err.Error())
	}

	var rootExists bool
	var pathList []string
	for k := range cmn.Services {
		if k == "/" {
			rootExists = true
			continue
		}

		pathList = append(pathList, k)
	}
	// ----------------------------
	sort.Strings(pathList)

	if rootExists {
		pathList = append(pathList, "/")
	}

	for i, k := range pathList {
		// Would be fixed in golang-1.22
		k := k

		if cmn.Services[k].IsFileServe {
			router.PathPrefix(k).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqProc(k, w, r)
			})

			z.Info(fmt.Sprintf("[%2d] file serve: %s, name: %s, physical path: %s",
				i, k, cmn.Services[k].Name, cmn.Services[k].DocRoot))
			continue
		}

		router.HandleFunc(k, func(w http.ResponseWriter, r *http.Request) {
			reqProc(k, w, r)
		})
		z.Info(fmt.Sprintf("[%2d] api serve: %s, name: %s", i, cmn.Services[k].Name, k))
	}

	host := "qnear.cn"
	if viper.IsSet("webServe.serverName") {
		host = viper.GetString("webServe.serverName")
	}

	appLaunchPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		z.Fatal(err.Error())
		return
	}

	certPath := appLaunchPath + "/certs"
	var hostWhiteList string
	if viper.IsSet("webServe.hostWhiteList") {
		hostWhiteList = viper.GetString("webServe.hostWhiteList")
		names := strings.Split(hostWhiteList, ",")
		host := "qnear.cn"
		if viper.IsSet("webServe.serverName") {
			host = viper.GetString("webServe.serverName")
		}
		var exists bool
		for _, v := range names {
			if v != host {
				continue
			}

			exists = true
			break
		}

		if !exists {
			log.Fatal(fmt.Sprintf("webServe.serverName:%s must exists in webServe.hostWhiteList: %s",
				host, hostWhiteList))
		}
	}

	if hostWhiteList == "" {
		hostWhiteList = host
	}

	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,

		HostPolicy: autocert.HostWhitelist(
			strings.Split(hostWhiteList, ",")...), //Your domain here

		Cache: autocert.DirCache(certPath), //Folder for storing certificates
	}

	//getWxAccessToken(2)

	httpListenPort := 8080
	if viper.IsSet("webServe.httpListenPort") {
		httpListenPort = viper.GetInt("webServe.httpListenPort")
	}

	httpsListenPort := 8443
	if viper.IsSet("webServe.httpsListenPort") {
		httpsListenPort = viper.GetInt("webServe.httpsListenPort")
	}

	var autoCert bool
	if viper.IsSet("webServe.autoCert") {
		autoCert = viper.GetBool("webServe.autoCert")
	}

	var ep string
	if autoCert {
		ep = fmt.Sprintf(":%v", httpsListenPort)
	} else {
		ep = fmt.Sprintf(":%v", httpListenPort)
	}

	s1 := "***********************************************************"
	s2 := "   ************ app started ****************************"
	s3 := fmt.Sprintf("                  db: %s@%s:%d/%s", viper.GetString("dbms.postgresql.user"),
		viper.GetString("dbms.postgresql.addr"),
		viper.GetInt32("dbms.postgresql.port"),
		viper.GetString("dbms.postgresql.db"))
	s8 := fmt.Sprintf("             version: %s", cmn.GetBuildVer())
	s4 := fmt.Sprintf("               redis: %s:%d", viper.GetString("dbms.redis.addr"),
		viper.GetInt32("dbms.redis.port"))

	s5 := "      web serve on *" + ep

	s6 := "   *****************************************************"
	s7 := "***********************************************************"

	z.Info(fmt.Sprintf("\n\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n",
		s1, s2, s3, s4, s5, s8, s6, s7))

	wrapper, err := gzhttp.NewWrapper(gzhttp.MinSize(2000),
		gzhttp.CompressionLevel(gzip.BestSpeed))

	if err != nil {
		log.Fatalln(err)
	}

	serv := &http.Server{
		Addr: ep,
		//Handler: GzipHandler(router),
		Handler: wrapper(router),
		//MaxHeaderBytes: 1024 * 1024 * 4,
	}

	if autoCert {
		serv.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
		go func() { _ = http.ListenAndServe(":http", certManager.HTTPHandler(nil)) }()
		_ = serv.ListenAndServeTLS("", "")
		return
	}

	cmn.AppStartTime = time.Now()
	z.Info(cmn.AppStartTime.Format(cmn.AppStartTimeLayout))
	_ = serv.ListenAndServe()
}

// reqProc main API entry
func reqProc(reqPath string, w http.ResponseWriter, r *http.Request) {
	//以单例运行

	runningSessionID := time.Now().UnixNano()
	isApi := rIsAPI.MatchString(r.URL.Path)
	if isApi {
		start := cmn.GetNowInMS()
		shortLiveMutex.Lock()
		cmn.RunningSessions[runningSessionID] =
			cmn.RunningSession{Api: r.URL.Path, BeginTime: start, RemoteAddr: r.RemoteAddr}
		shortLiveMutex.Unlock()

		defer func() {
			z.Info(fmt.Sprintf("%s: %dms pprof", r.URL.Path, cmn.GetNowInMS()-start))
			z.Info("------------ end ---------------")
			shortLiveMutex.Lock()
			delete(cmn.RunningSessions, runningSessionID)
			shortLiveMutex.Unlock()
		}()
	}

	pgxInUse := pgxConn.Stat().AcquiredConns()
	sqlxInUse := sqlxDB.Stats().InUse

	defer func() {
		pgxInUseNow := pgxConn.Stat().AcquiredConns()
		sqlxInUseNow := sqlxDB.Stats().InUse

		if d := pgxInUseNow - pgxInUse; d > 0 {
			z.Warn(fmt.Sprintf("%s: pgx connection leaked: %d", r.URL.Path, d))
		}

		if d := sqlxInUseNow - sqlxInUse; d > 0 {
			z.Warn(fmt.Sprintf("%s: sqlx connection leaked: %d", r.URL.Path, d))
		}

		cmn.DbState(pgxConn)
		cmn.DbState(sqlxDB)
	}()

	cmn.DebugMode(w)

	if disableLog(r) {
		defer restoreLog()
	}

	if cmn.SerializationReq {
		z.Warn("serializationReq")
		z.Info("try lock")
		door.Lock()
		z.Info("got lock")
		defer func() {
			door.Unlock()
			z.Info("release lock")
		}()
	}

	userAgent := r.Header.Get("User-Agent")
	var clnType = cmn.CPcBrowserCaller

	if userAgent != "" {
		switch {
		case rWxAndroid.MatchString(userAgent):
			clnType = cmn.CAndroidWxCaller

		case rWxIOS.MatchString(userAgent):
			clnType = cmn.CIOSWxCaller

		case rWinWx.MatchString(userAgent):
			clnType = cmn.CWinWxCaller

		case rMacWx.MatchString(userAgent):
			clnType = cmn.CMacWxCaller

		case !rIsWx.MatchString(userAgent) && rMobile.MatchString(userAgent):
			clnType = cmn.CMobileBrowserCaller

		case rIsWx.MatchString(userAgent):
			clnType = cmn.CUnknownWxCaller

		default:
			clnType = cmn.CPcBrowserCaller
		}
	} else {
		z.Warn("userAgent is empty")
	}

	ep, isAttacker, whiteList, isPathValid := cmn.IsAttacker(r.URL.Path, r.RemoteAddr)
	if !isPathValid {
		z.Info(fmt.Sprintf("invalid path: %s", r.URL.Path))
		return
	}

	if isAttacker {
		z.Info(fmt.Sprintf("judge as attacker path: %s", r.URL.Path))
		return
	}

	z.Info(fmt.Sprintf("%s(%s): %s", ep.Name, ep.Path, r.URL.Path))
	// ---------------------------
	q := &cmn.ServiceCtx{
		WhiteList:  whiteList,
		CallerType: clnType,

		Redis: cmn.GetRedisConn(),

		R:  r,
		W:  w,
		Ep: ep,

		ReqAdminFnc: r.URL.Query().Get("admin") == "true",
		Msg: &cmn.ReplyProto{
			API:    r.URL.Path,
			Method: r.Method,
		},
		BeginTime: time.Now(),

		Tag: make(map[string]interface{}),
	}

	var err error
	q.Session, err = store.Get(r, "qNearSessions")
	if err != nil {
		z.Error(err.Error())
		cmn.XCleanSession(w, r)
		http.Redirect(w, r, cmn.BuildURL(r), http.StatusFound)
		return
	}
	q.Session.Options.SameSite = http.SameSiteStrictMode
	ctx := context.WithValue(context.Background(), cmn.QNearKey, q)
	defer crashed(ctx)

	xCleanSession := q.R.URL.Query().Get("xCleanSession")
	if xCleanSession == "142857" {
		cmn.CleanSession(ctx)
		if q.Err != nil {
			q.RespErr()
		} else {
			q.Resp()
		}
		return
	}

	q.Err = cmn.ValidateCookie(ctx)
	if q.Err != nil {
		q.RespErr()
		return
	}

	cmn.WxLogin(ctx)
	if q.Stop || q.Err != nil {
		return
	}

	_, q.Err = cmn.BuildUserInfo(ctx)
	if q.Err != nil {
		q.Stop = true
		q.RespErr()
		return
	}

	if rIsAPI.MatchString(q.R.URL.Path) {
		if q.R.URL.Query().Get("admin") == "true" {
			q.ReqFnType = cmn.CFuncAdminFileServe
		} else {
			q.ReqFnType = cmn.CFuncNonAdminFileServe
		}
		q.Session.Values["reqFnType"] = q.ReqFnType
		q.Err = q.Session.Save(r, w)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	var ok bool
	q.ReqFnType, ok = q.Session.Values["reqFnType"].(int)
	if !ok {
		q.ReqFnType = cmn.CFuncNonAdminFileServe
	}

	//authenticate
	if !cmn.DisableAA {
		_ = cmn.Authenticate(ctx)
		if q.Err != nil {
			return
		}
	}

	// request nonexistence api
	if isApi && reqPath == "/" {
		q.Err = fmt.Errorf("request nonexistence api: %s", r.URL.Path)
		q.RespErr()
		z.Warn(q.Err.Error())
		return
	}

	cmn.Services[reqPath].Fn(ctx)
}
