package service

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"

	"w2w.io/mux"
	"w2w.io/tusd/filelocker"
	"w2w.io/tusd/pkg/filestore"
	tusd "w2w.io/tusd/pkg/handler"
)

type TraceHandler struct {
	Handler *tusd.Handler

	TusdBasePath string
	tusdApiRegex *regexp.Regexp

	TusdUploadStorePath string
}

func (h *TraceHandler) Middleware(handler http.Handler) http.Handler {
	if h.Handler == nil {
		z.Fatal("TraceHandler is nil")
	}
	if h.TusdBasePath == "" {
		z.Fatal("TusdBasePath is empty")
	}
	if h.tusdApiRegex == nil {
		z.Fatal("tusdApiRegex is nil")
	}
	if h.TusdUploadStorePath == "" {
		z.Fatal("TusdUploadStorePath is empty")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.tusdApiRegex.Match([]byte(r.URL.Path)) {
			handler.ServeHTTP(w, r)
			return
		}

		p := strings.TrimPrefix(r.URL.Path, h.TusdBasePath)
		if len(p) > 0 && p[0] == '/' {
			p = p[1:]
		}
		rp := strings.TrimPrefix(r.URL.RawPath, h.TusdBasePath)
		if len(rp) > 0 && rp[0] == '/' {
			rp = rp[1:]
		}

		r2 := new(http.Request)
		if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			//h.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
			return
		}

		method := strings.ToLower(r2.Method)
		if method == "post" {
			v := r2.Header.Get("Upload-Metadata")
			metadata := r2.URL.Query().Get("metadata")
			if v == "" && metadata != "" {
				r2.Header.Set("Upload-Metadata", metadata)
			}
		}

		h.Handler.ServeHTTP(w, r2)
	})
}

func tusdSetup(r *mux.Router) (err error) {
	key := "tusd.fileStorePath"
	uploadDir := "./uploads"
	if viper.IsSet(key) {
		uploadDir = viper.GetString(key)
	}
	uploadDir, err = filepath.Abs(uploadDir)
	if err != nil {
		z.Fatal(fmt.Sprintf("Unable to make absolute path: %s", err))
	}

	z.Info(fmt.Sprintf("Using '%s' as file store directory.\n", uploadDir))
	if err := os.MkdirAll(uploadDir, os.FileMode(0774)); err != nil {
		z.Fatal(fmt.Sprintf("Unable to ensure directory exists: %s", err))
	}

	store := filestore.New(uploadDir)
	locker := filelocker.New(uploadDir)

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	basePath := "/api/file"
	key = "tusd.basePath"
	if viper.IsSet(key) {
		basePath = viper.GetString(key)
	}

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath: basePath,

		DisableDownload:      false,
		DisableTermination:   false,
		DisableConcatenation: false,

		StoreComposer: composer,

		RespectForwardedHeaders:   true,
		PreFinishResponseCallback: tusd.PreFinishRespCB,
	})
	if err != nil {
		log.Fatalf("unable to create handler: %s", err)
	}

	traceHandler := &TraceHandler{Handler: handler}
	traceHandler.TusdBasePath = basePath
	traceHandler.TusdUploadStorePath = uploadDir
	traceHandler.tusdApiRegex = regexp.MustCompile("(?i)" + basePath + "(?:/.*)?$")

	r.Use(traceHandler.Middleware)
	return
}
