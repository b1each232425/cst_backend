package service

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"regexp"
	"w2w.io/mux"
	"w2w.io/tusd/pkg/filelocker"
	"w2w.io/tusd/pkg/filestore"
	"w2w.io/tusd/pkg/handler"
	"w2w.io/tusd/pkg/hooks"
)

var Flags = struct {
	HttpHost  string
	HttpPort  string
	HttpSock  string
	MaxSize   int64
	UploadDir string
	BasePath  string

	ShowGreeting bool

	DisableDownload    bool
	DisableTermination bool

	Timeout  int64
	S3Bucket string

	S3Endpoint string
	S3PartSize int64

	S3ObjectPrefix string
	S3DisableSSL   bool

	S3DisableContentHashes bool
	S3TransferAcceleration bool

	GCSBucket       string
	GCSObjectPrefix string

	AzStorage string

	AzContainerAccessType string

	AzBlobAccessTier   string
	AzObjectPrefix     string
	AzEndpoint         string
	EnabledHooksString string
	FileHooksDir       string
	HttpHooksEndpoint  string

	HttpHooksForwardHeaders string

	HttpHooksRetry    int
	HttpHooksBackoff  int
	GrpcHooksEndpoint string
	GrpcHooksRetry    int
	GrpcHooksBackoff  int

	HooksStopUploadCode int

	PluginHookPath string

	EnabledHooks []hooks.HookType

	ShowVersion   bool
	ExposeMetrics bool
	MetricsPath   string
	BehindProxy   bool
	VerboseOutput bool

	TLSCertFile string
	TLSKeyFile  string
	TLSMode     string

	CPUProfile string
}{
	HttpHost: "0.0.0.0", HttpPort: "1080", HttpSock: "",
	MaxSize: 0, UploadDir: "./data", BasePath: "/api/file/",
	ShowGreeting: true, DisableDownload: false, DisableTermination: false, Timeout: 6 * 1000,
	S3Bucket: "", S3ObjectPrefix: "", S3Endpoint: "", S3PartSize: 50 * 1024 * 1024,
	S3DisableContentHashes: false, S3DisableSSL: false, S3TransferAcceleration: false,
	GCSBucket: "", GCSObjectPrefix: "",
	AzStorage: "", AzContainerAccessType: "", AzBlobAccessTier: "", AzObjectPrefix: "", AzEndpoint: "",
	EnabledHooksString: "pre-create,post-create,post-receive,post-terminate,post-finish",
	FileHooksDir:       "",
	HttpHooksEndpoint:  "", HttpHooksForwardHeaders: "", HttpHooksRetry: 3, HttpHooksBackoff: 1,
	GrpcHooksEndpoint: "", GrpcHooksRetry: 3, GrpcHooksBackoff: 1,
	HooksStopUploadCode: 0, PluginHookPath: "", ShowVersion: false, ExposeMetrics: true, MetricsPath: "/metrics",
	BehindProxy: false, VerboseOutput: true, TLSCertFile: "", TLSKeyFile: "", TLSMode: "tls12", CPUProfile: "",
}

var Composer *handler.StoreComposer

func CreateComposer() {
	defer func() {
		z.Info(fmt.Sprintf("Using %.2fMB as maximum size.\n", float64(Flags.MaxSize)/1024/1024))
	}()

	key := "webServe.fileStorePath"
	if viper.IsSet(key) {
		Flags.UploadDir = viper.GetString(key)
	}

	uploadDir, err := filepath.Abs(Flags.UploadDir)
	if err != nil {
		z.Fatal(fmt.Sprintf("Unable to make absolute path: %s", err))
	}

	z.Info(fmt.Sprintf("Using '%s' as directory storage.\n", uploadDir))
	if err := os.MkdirAll(uploadDir, os.FileMode(0774)); err != nil {
		z.Fatal(fmt.Sprintf("Unable to ensure directory exists: %s", err))
	}

	Composer = handler.NewStoreComposer()
	store := filestore.New(uploadDir)
	store.UseIn(Composer)

	locker := filelocker.New(uploadDir)
	locker.UseIn(Composer)

}

func uploadEventProc(h handler.HookEvent, action string) {
	z.Info(action)
}

func setupFileServeHandler(r *mux.Router) (err error) {
	fileServeEP := "(?i)/api/file(?:/.*)?$"

	key := "webServe.fileServeEP"
	if viper.IsSet(key) {
		fileServeEP = viper.GetString(key)
	}

	var behindProxy bool
	key = "webServe.behindProxy"
	if viper.IsSet(key) {
		behindProxy = viper.GetBool(key)
	}

	if Composer == nil {
		CreateComposer()
	}

	config := handler.Config{
		RegExpFileServeEP: regexp.MustCompile(fileServeEP),

		MaxSize:  Flags.MaxSize,
		BasePath: Flags.BasePath,

		RespectForwardedHeaders: behindProxy,

		DisableDownload:    Flags.DisableDownload,
		DisableTermination: Flags.DisableTermination,

		StoreComposer: Composer,

		NotifyCompleteUploads:   true,
		NotifyTerminatedUploads: true,
		NotifyUploadProgress:    true,
		NotifyCreatedUploads:    true,

		PreUploadCreateCallback:   handler.PreUploadCreateCB,
		PreFinishResponseCallback: handler.PreFinishRespCB,
	}

	//if err = SetupPreHooks(&config); err != nil {
	//	z.Fatal(fmt.Sprintf("Unable to setup hooks for urHandler: %s", err))
	//}

	urHandler, err := handler.NewUnroutedHandler(config)
	if err != nil {
		z.Fatal(fmt.Sprintf("Unable to create urHandler: %s", err))
	}

	z.Info(fmt.Sprintf("Using %s as the base path.\n", fileServeEP))

	//SetupPostHooks(urHandler)

	r.Use(urHandler.Middleware)

	r.PathPrefix(fileServeEP).Methods("POST").HandlerFunc(urHandler.PostFile)
	r.PathPrefix(fileServeEP).Methods("HEAD").HandlerFunc(urHandler.HeadFile)
	r.PathPrefix(fileServeEP).Methods("PATCH").HandlerFunc(urHandler.PatchFile)
	r.PathPrefix(fileServeEP).Methods("GET").HandlerFunc(urHandler.GetFile)
	r.PathPrefix(fileServeEP).Methods("DELETE").HandlerFunc(urHandler.DelFile)

	go func() {
		for {
			ev := <-urHandler.CompleteUploads
			uploadEventProc(ev, "complete")
		}
	}()
	go func() {
		for {
			ev := <-urHandler.TerminatedUploads
			uploadEventProc(ev, "terminated")
		}
	}()

	go func() {
		for {
			ev := <-urHandler.UploadProgress
			uploadEventProc(ev, "progress")
		}
	}()

	go func() {
		for {
			ev := <-urHandler.CreatedUploads
			uploadEventProc(ev, "created")
		}
	}()

	z.Info(fmt.Sprintf("Supported tus extensions: %s\n",
		urHandler.SupportedExtensions()))
	return
}
