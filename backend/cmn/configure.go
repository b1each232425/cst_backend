package cmn

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"runtime"

	"github.com/asdine/storm/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"w2w.io/null"

	"github.com/gomodule/redigo/redis"

	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

type (
	PackageStarter func()

	DefaultLogger struct {
		fd *os.File
	}
)

var (
	PackageStarters []PackageStarter

	AppStartTimeLayout = "2006-01-02 15:04:05.000-0700"
	AppStartTime       time.Time

	rootDB *storm.DB
	userDB storm.Node

	logDB storm.Node

	AppLaunchPath string

	consoleBootLogger = log.New(os.Stdout, "", 0)

	D DefaultLogger

	//sqlxDB, pgxConn global dbms connection
	sqlxDB *sqlx.DB

	pgxConn *pgxpool.Pool

	redisPool *redis.Pool

	//CancelWaitDbNotify cancel waiting for pg db notify
	CancelWaitDbNotify context.CancelFunc

	InDebugMode bool

	//AttackDefence enable ddos defence
	AttackDefence bool

	//DisableAA disable authorize/authenticate
	DisableAA bool

	SerializationReq bool

	GRPCAddr = "localhost"
	GRPCPort = 6691

	BaseRepo string
)

func defaultLogger(level string, msg string) {
	_ = log.Output(3, fmt.Sprintf("%s %s", level, msg))
	_ = consoleBootLogger.Output(3, fmt.Sprintf("%s %s", level, msg))
}

func (*DefaultLogger) Debug(msg string) {
	defaultLogger("debug", msg)
}

func (*DefaultLogger) Info(msg string) {
	defaultLogger(" info", msg)
}

func (*DefaultLogger) Warn(msg string) {
	defaultLogger(" warn", msg)
}

func (*DefaultLogger) Error(msg string) {
	defaultLogger("error", msg)
}

func (v *DefaultLogger) Fatal(msg string) {
	defaultLogger("fatal", msg)
	_ = v.fd.Close()
	os.Exit(-1)
}

// InitDbByParams set the connection parameters
func InitDbByParams(db, dbHost, dbPort, dbUser, dbPwd string) {
	if db == "" || dbHost == "" || dbPort == "" || dbUser == "" || dbPwd == "" {
		D.Fatal("missing some/all parameters, please supply db, dbHost, dbPort, dbUser, dbPwd ")
	}
	//"postgres://pgx_md5:secret@localhost:5432/pgx_test")
	connInfo := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		dbUser, dbPwd, dbHost, dbPort, db)
	var err error

	sqlxDB, err = sqlx.Open("pgx", connInfo)
	if err != nil {
		D.Fatal(err.Error())
	}

	maxDBOpenConn := 10
	if viper.IsSet("dbms.postgresql.max-conn") {
		maxDBOpenConn = viper.GetInt("dbms.postgresql.max-conn")
	}
	sqlxDB.SetMaxOpenConns(maxDBOpenConn)
	sqlxDB.SetConnMaxIdleTime(60 * time.Second)

	//sqlxDB.
	pgxPoolSetup(connInfo)

	D.Info("begin ping db server " + dbHost + ":" + dbPort)
	_, err = sqlxDB.Exec(`set timezone='Asia/Chongqing'`)
	if err != nil {
		sqlxDB.Close()
		D.Fatal(err.Error())
	}

}

func afterConnect(ctx context.Context, conn *pgx.Conn) (err error) {
	if conn == nil {
		err = fmt.Errorf("conn is nil")
		fmt.Println(err.Error())
		return
	}
	_, err = conn.Exec(ctx, `set timezone='Asia/Chongqing'`)
	log.Println("pg connected")
	return
}

func pgxPoolSetup(conn string) {
	maxDBOpenConn := 10
	if viper.IsSet("dbms.postgresql.max-conn") {
		maxDBOpenConn = viper.GetInt("dbms.postgresql.max-conn")
	}
	poolCfg, err := pgxpool.ParseConfig(conn)
	if err != nil {
		D.Fatal(err.Error())
	}

	//poolCfg.ConnConfig.OnNotice = dbNoticeHandler
	//poolCfg.ConnConfig.OnNotification = dbNotificationHandler

	poolCfg.AfterConnect = afterConnect
	poolCfg.MaxConns = int32(maxDBOpenConn)
	poolCfg.MaxConnIdleTime = 60 * time.Second
	pgxConn, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		D.Fatal(err.Error())
	}

	if pgxConn == nil {
		D.Fatal("pgxConn is nil")
	}

	s := `set timezone = 'Asia/Chongqing'`
	_, err = pgxConn.Exec(context.Background(), s)
	if err != nil {
		D.Fatal(err.Error())
	}

	s = `listen qNearDbMsg`
	_, err = pgxConn.Exec(context.Background(), s)
	if err != nil {
		D.Fatal(err.Error())
	}

	go dbNotification()
}

func dbNotification() {
	ctx := context.Background()
	ctx, CancelWaitDbNotify = context.WithCancel(ctx)

	c, err := pgxConn.Acquire(ctx)
	if err != nil {
		D.Fatal(err.Error())
	}

	defer c.Release()

	//*** ATTENTION:
	//	select pg_notify($channel,'msg')
	// the $channel must be LOWERCASE ***
	// so, the statement should be select pg_notify('qneardbmsg','hello world')
	_, err = c.Exec(ctx, "listen qNearDbMsg")
	if err != nil {
		D.Fatal(err.Error())
		return
	}

	for {
		n, err := c.Conn().WaitForNotification(ctx)
		if err != nil {
			D.Error(err.Error())
			break
		}
		if n != nil {
			D.Info(n.Payload)
		}
	}
}

// configureDb initialize database connection
func configureDb() {
	if sqlxDB != nil {
		return
	}

	db := "kdb"
	dbHost := "cst.gzhu.edu.cn"
	dbPort := "16900"
	dbUser := "kuser"

	dbPwd := "ak47-Ever"

	var s string
	s = "dbms.postgresql.addr"
	if viper.IsSet(s) {
		dbHost = viper.GetString(s)
	}
	s = "dbms.postgresql.port"
	if viper.IsSet(s) {
		dbPort = fmt.Sprintf("%d", viper.GetInt(s))
	}
	s = "dbms.postgresql.db"
	if viper.IsSet(s) {
		db = viper.GetString(s)
	}
	s = "dbms.postgresql.user"
	if viper.IsSet(s) {
		dbUser = viper.GetString(s)
	}
	s = "dbms.postgresql.pwd"
	if viper.IsSet(s) {
		dbPwd = viper.GetString(s)
	}

	InitDbByParams(db, dbHost, dbPort, dbUser, dbPwd)
	D.Info(fmt.Sprintf("connected with db server %s:%s", dbHost, dbPort))

	redisConnInit()

	var isBBoltEnabled bool
	s = "dbms.bbolt.enable"
	if viper.IsSet(s) {
		isBBoltEnabled = viper.GetBool(s)
	}

	if isBBoltEnabled && initSysDB() != nil {
		Terminate(-1)
	}
}

// GetPgxConn return connected pgx.Conn object
func GetPgxConn() *pgxpool.Pool {
	if pgxConn == nil {
		configureDb()
	}

	if pgxConn == nil {
		D.Fatal("pgxConn is nil")
	}

	return pgxConn
}

// GetDbConn return connected *sqlx.DB
func GetDbConn() *sqlx.DB {
	if sqlxDB == nil {
		configureDb()
	}
	return sqlxDB
}

// GetRedisConn return redis.Conn
func GetRedisConn() redis.Conn {
	if redisPool == nil {
		redisConnInit()
	}
	poolStats := redisPool.Stats()

	D.Info(fmt.Sprintf("redisPool activeCount:%d, idleAcount: %d",
		poolStats.ActiveCount, poolStats.IdleCount))

	return redisPool.Get()
}

func redisConnInit() {
	if redisPool != nil {
		return
	}

	host := "cst.gzhu.edu.cn"
	port := 16910

	var s string
	s = "dbms.redis.addr"
	if viper.IsSet(s) {
		host = viper.GetString(s)
	}
	s = "dbms.redis.port"
	if viper.IsSet(s) {
		port = viper.GetInt(s)
	}
	serv := fmt.Sprintf("%s:%d", host, port)
	log.Printf("connecting redis to %s", serv)
	redisPool = &redis.Pool{
		MaxIdle:     32,
		IdleTimeout: 60 * time.Minute,
		Dial: func() (conn redis.Conn, err error) {
			for {
				conn, err = redis.Dial("tcp", serv)
				if err != nil {
					D.Error(err.Error())
					<-time.After(time.Second * 15)
					continue
				}
				pass := "x2Jc5K^5"
				if viper.IsSet("dbms.redis.cert") {
					pass = viper.GetString("dbms.redis.cert")
				}

				_, err = conn.Do("AUTH", pass)
				if err != nil {
					D.Error(err.Error())
					<-time.After(time.Second * 15)
					continue
				}
				log.Printf("redis connected with " + serv)

				if viper.IsSet("dbms.redis.init") {
					cleanCache := viper.GetBool("dbms.redis.init")
					if cleanCache {
						_, err = conn.Do("flushdb")
						if err != nil {
							D.Error(err.Error())
							return
						}
						D.Info("successfully cleanup redis db")
						defer disableNextFlushDB()
					}
				}
				break
			}
			return
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	conn := redisPool.Get()
	_, err := conn.Do("INFO")
	if err != nil {
		D.Fatal(err.Error())
	}
	D.Info(fmt.Sprintf("connected with redis: %s\n", serv))
}

// CleanRedis redis current db
func CleanRedis() {
	r := GetRedisConn()
	defer r.Close()

	_, err := r.Do("flushdb")
	if err != nil {
		D.Error(err.Error())
		return
	}
	fmt.Println("successfully cleanup redis db")
	defer disableNextFlushDB()
}

func disableNextFlushDB() {
	err := JsonWrite(viper.ConfigFileUsed(), "dbms.redis.init", false)
	if err != nil {
		D.Error(err.Error())
		return
	}
}

func initSysDB() (err error) {
	basePath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		D.Fatal(err.Error())
	}

	dbFN := basePath + string(os.PathSeparator) + "sys.db"
	fmt.Printf("boltdb: %s\n", dbFN)
	rootDB, err = storm.Open(dbFN,
		storm.BoltOptions(0600, &bolt.Options{Timeout: 1 * time.Second}))

	if err != nil {
		D.Fatal(fmt.Sprintf(`open sessionDB failed: %s, please using "lsof | grep sessonDB" to see who lock this file\n`,
			err.Error()))
	}

	if userDB == nil {
		userDB = rootDB.From("user")
	}
	D.Info("user db opened")

	if logDB == nil {
		logDB = rootDB.From("log")
	}
	D.Info("logs db opened")

	return
}

// packageSettle after configure parameters, database connector, logger settle then
// non cmn package can use it.
func nonCmnPackageSetup() {
	for _, v := range PackageStarters {
		v()
	}
}

func Configure() {
	gob.Register(map[string]string{})

	gob.Register(null.String{})
	gob.Register(null.Int{})
	gob.Register(null.Float{})
	gob.Register(null.Time{})
	gob.Register(null.Bool{})
	gob.Register(null.QNearTime{})

	var err error
	AppLaunchPath, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err.Error())
		Terminate(-1)
	}

	logDir := AppLaunchPath + "/logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.Mkdir(logDir, os.ModePerm)
	}

	bootLogFN := logDir + "/bootlog.txt"
	fd, err := os.OpenFile(bootLogFN, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal("open " + bootLogFN + " failed by " + err.Error())
	}
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	mf := io.MultiWriter(os.Stdout, fd)
	log.SetOutput(mf)
	D.fd = fd
	D.Info("=========================")
	D.Info("== boot logger started ==")

	// adding application startup directory as first search path.
	viper.AddConfigPath(AppLaunchPath)

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		D.Fatal(err.Error())
	}

	viper.AddConfigPath(userHomeDir)

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		D.Fatal(err.Error())
	}

	viper.AddConfigPath(userConfigDir)

	wd, err := os.Getwd()
	if err != nil {
		D.Fatal(err.Error())
	}

	viper.AddConfigPath(wd)

	cfgFileName := ".config_" + runtime.GOOS
	switch runtime.GOOS {
	case "darwin", "windows", "linux":
		break

	default:
		D.Fatal("unsupported os: " + runtime.GOOS)

	}

	configureFileName := AppLaunchPath + string(os.PathSeparator) + cfgFileName + ".json"
	if _, err := os.Stat(configureFileName); err != nil {
		templateFileName := AppLaunchPath + string(os.PathSeparator) +
			".config_" + runtime.GOOS + "_template.json"
		if _, err := os.Stat(templateFileName); err != nil {
			D.Fatal("can not find " + templateFileName + ", " +
				err.Error())
		}

		src, err := os.Open(templateFileName)
		if err != nil {
			D.Fatal("can not open " + templateFileName)
		}

		defer func() { _ = src.Close() }()

		dst, err := os.Create(configureFileName)
		if err != nil {
			D.Fatal("can not create " + configureFileName)
		}
		defer func() { _ = dst.Close() }()

		if _, err = io.Copy(dst, src); err != nil {
			D.Fatal(err.Error())
		}

		D.Info(fmt.Sprintf("can not find %s, recreate it by %s\n",
			configureFileName, templateFileName))
	}
	viper.SetConfigName(cfgFileName)

	viper.AutomaticEnv() // read in environment variables that match

	err = viper.ReadInConfig()
	if err != nil {
		D.Fatal(err.Error())
	}

	D.Info("configured with " + viper.ConfigFileUsed())

	configureDb()

	InitLogger()

	if viper.IsSet("debug.aa") {
		zjTEL := viper.GetInt64("debug.aa")
		if zjTEL == 13450464791 {
			DisableAA = true
		}
	}

	if viper.IsSet("debug.enable") {
		InDebugMode = viper.GetBool("debug.enable")
	}

	if InDebugMode {
		if viper.IsSet("debug.serializationReq") {
			SerializationReq = viper.GetBool("debug.serializationReq")
		}
	}

	if viper.IsSet("webServe.attackDefence") {
		AttackDefence = viper.GetBool("webServe.attackDefence")
	}

	if viper.IsSet("w2w.grpc.addr") {
		GRPCAddr = viper.GetString("w2w.grpc.addr")
	}

	if viper.IsSet("repo.base") {
		BaseRepo = viper.GetString("repo.base")
	}

	if viper.IsSet("w2w.grpc.port") {
		GRPCPort = viper.GetInt("w2w.grpc.port")
	}

	if viper.IsSet("webServe.attackDefence") {
		AttackDefence = viper.GetBool("webServe.attackDefence")
	}
	nonCmnPackageSetup()
}
