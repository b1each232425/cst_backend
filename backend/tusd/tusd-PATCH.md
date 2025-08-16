# tusd-2.8.0 patch adaptive tus-js-client-4.3.1

demo usage please reference repository: dawnfire/tusdpatch.git#v1.0.0


## 1. internal/uid/uid.go
***add***

```go
func FileID(info handler.FileInfo) string {
	data := info.MetaData

	var checksum, errMsg string
	for {
		if data == nil {
			errMsg = "invalid/empty meta-data"
			break
		}

		v, ok := data["filename"]
		if !ok || v == "" {
			errMsg = "invalid/empty filename"
			break
		}

		v, ok = data["filesize"]
		if !ok || v == "" {
			errMsg = "invalid/empty filesize"
			break
		}

		checksum, ok = data["checksum"]
		if !ok || checksum == "" {
			errMsg = "invalid/empty checksum"
			break
		}

		v, ok = data["filetype"]
		if !ok || v == "" {
			info.MetaData["filetype"] = "application/octet-stream"
		}
		break
	}

	if errMsg != "" {
		//err := errors.New(errMsg)
		//panic(err)
		return Uid()
	}

	return checksum
}
```

## 2 pkg/azurestore/azurestore.go

***add ***

```go
func (store AzureStore) Query(ctx context.Context, criteria string) (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}
```

## 3 pkg/filestore/filestore.go

***modify***

```go
// UseIn sets this store as the core data store in the passed composer and adds
// all possible extension to it.
func (store FileStore) UseIn(composer *handler.StoreComposer) {
	composer.UseCore(store)
	composer.UseTerminater(store)
	composer.UseConcater(store)
	composer.UseLengthDeferrer(store)
	+composer.UseChecksum(store)
}


func (store FileStore) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	if info.ID == "" {
-		info.ID = uid.Uid()
+		info.ID = uid.FileID(info)
	}
```

*** add ***

```go
func (store FileStore) AsChecksumableUpload(upload handler.Upload) handler.ChecksumableUpload {
	return upload.(*fileUpload)
}

func (upload *fileUpload) Checksum(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	v := r.Header.Get("Upload-Checksum")
	fmt.Println(v)
	//Upload-Checksum
	//upload.
	//if err := os.Remove(upload.infoPath); err != nil {
	//	return err
	//}
	//if err := os.Remove(upload.binPath); err != nil {
	//	return err
	//}
	return nil
}





type fileDesc struct {
	ID   null.String
	Size null.Int

	SizeIsDeferred null.Bool

	Offset    null.Int
	IsPartial null.Bool
	IsFinal   null.Bool

	PartialUploads []null.String

	MetaData *struct {
		Checksum null.String `json:"checksum,omitempty"`
		Filename null.String `json:"filename,omitempty"`
		Filesize null.Int    `json:"filesize,omitempty"`
		Filetype null.String `json:"filetype,omitempty"`

		LastModified null.String `json:"lastModified,omitempty"`
	}

	Storage struct {
		Path null.String
		Type null.String
	}
}

// Query file by criteria
func (store FileStore) Query(ctx context.Context, criteria string) (result []byte, err error) {
	rFileNamePattern, err := regexp.Compile(criteria)
	if err != nil {
		return
	}

	returnAll := false
	if criteria == ".*" {
		returnAll = true
	}

	var fileList []string
	infoRe := regexp.MustCompile("\\.info$")
	_ = filepath.Walk(store.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !infoRe.MatchString(info.Name()) {
			return nil
		}

		data, err := os.ReadFile(fmt.Sprintf("%s/%s", store.Path, info.Name()))
		if err != nil {
			return err
		}

		var desc fileDesc
		err = json.Unmarshal(data, &desc)
		if err != nil {
			return err
		}

		if returnAll {
			fileList = append(fileList, string(data))
			return nil
		}

		if desc.MetaData == nil || !desc.MetaData.Filename.Valid || desc.MetaData.Filename.String == "" {
			return nil
		}

		if !rFileNamePattern.Match([]byte(desc.MetaData.Filename.String)) {
			return nil
		}

		fileList = append(fileList, string(data))
		return nil
	})

	if len(fileList) == 0 {
		return nil, nil
	}

	return []byte("[" + strings.Join(fileList, ",") + "]"), nil
}
```

## 4 pkg/gcsstore/gcsstore.go

*** add ***

```go
func (store GCSStore) Query(ctx context.Context, criteria string) (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}
```

## 5 pkg/handler/composer.go

*** add ***

```go
type StoreComposer struct {
	Core DataStore

	UsesTerminater     bool
	Terminater         TerminaterDataStore
	UsesLocker         bool
	Locker             Locker
	UsesConcater       bool
	Concater           ConcaterDataStore
	UsesLengthDeferrer bool
	LengthDeferrer     LengthDeferrerDataStore

	+UsesChecksum bool
	+Checksum     ChecksumDataStore
}


func (store *StoreComposer) UseChecksum(ext ChecksumDataStore) {
	store.UsesChecksum = ext != nil
	store.Checksum = ext
}
```


## 6 pkg/handler/config.go

*** add ***

```go
type Config struct {
	+RegExpFileServeEP *regexp.Regexp

```

## 7 pkg/handler/datastore.go

```go
type DataStore interface {
	// Create a new upload using the size as the file's length. The method must
	// return an unique id which is used to identify the upload. If no backend
	// (e.g. Riak) specifes the id you may want to use the uid package to
	// generate one. The properties Size and MetaData will be filled.
	NewUpload(ctx context.Context, info FileInfo) (upload Upload, err error)

	// GetUpload fetches the upload with a given ID. If no such upload can be found,
	// ErrNotFound must be returned.
	GetUpload(ctx context.Context, id string) (upload Upload, err error)

	+Query(ctx context.Context, criteria string) (result []byte, err error)
}
```
*** add ***

```go
type ChecksumableUpload interface {
	// Checksum .
	Checksum(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

// ChecksumDataStore .
type ChecksumDataStore interface {
	AsChecksumableUpload(upload Upload) ChecksumableUpload
}
```

## 8 pkg/handler/unrouted_handler.go

*** modify ***

```go
L388
		if changes.MetaData != nil {
			info.MetaData = changes.MetaData
		}

		if changes.Storage != nil {
			info.Storage = changes.Storage
		}
	}
L397
+	// -------------------------------------------
+	//----- ignore duplicated file upload ---
+	duplicated, err := handler.IsDuplicate(w, r)
+	if err != nil {
+		handler.sendError(c, err)
+		return
+	}
+
+	if duplicated {
+		if info.MetaData != nil && info.MetaData["checksum"] != "" {
+			info.ID = info.MetaData["checksum"]
+		}
+
+		resp.StatusCode = http.StatusAlreadyReported
+		url := handler.absFileURL(r, info.ID)
+		resp.Header["Location"] = url
+		msg := fmt.Sprintf("file '%s' already exists,checksum=%s,size=%s",
+			info.MetaData["filename"], info.MetaData["checksum"], info.MetaData["filesize"])
+		resp.Body = msg
+		handler.sendResp(c, resp)
+		return
+	}
+	// -------------------------------------------

	upload, err := handler.composer.Core.NewUpload(c, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}


L575
		if changes.MetaData != nil {
			info.MetaData = changes.MetaData
		}

		if changes.Storage != nil {
			info.Storage = changes.Storage
		}
	}


	// -------------------------------------------
	//----- ignore duplicated file upload ---
	duplicated, err := handler.IsDuplicate(w, r)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	if duplicated {
		if info.MetaData != nil && info.MetaData["checksum"] != "" {
			info.ID = info.MetaData["checksum"]
		}

		resp.StatusCode = http.StatusAlreadyReported
		url := handler.absFileURL(r, info.ID)
		resp.Header["Location"] = url
		msg := fmt.Sprintf("file '%s' already exists,checksum=%s,size=%s",
			info.MetaData["filename"], info.MetaData["checksum"], info.MetaData["filesize"])
		resp.Body = msg
		handler.sendResp(c, resp)
		return
	}
	// -------------------------------------------

L608
	upload, err := handler.composer.Core.NewUpload(c, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}
+	// -------------------------------------------
+	//----- ignore duplicated file upload ---
+	duplicated, err := handler.IsDuplicate(w, r)
+	if err != nil {
+		handler.sendError(c, err)
+		return
+	}
+
+	if duplicated {
+		if info.MetaData != nil && info.MetaData["checksum"] != "" {
+			info.ID = info.MetaData["checksum"]
+		}
+
+		resp.StatusCode = http.StatusAlreadyReported
+		url := handler.absFileURL(r, info.ID)
+		resp.Header["Location"] = url
+		msg := fmt.Sprintf("file '%s' already exists,checksum=%s,size=%s",
+			info.MetaData["filename"], info.MetaData["checksum"], info.MetaData["filesize"])
+		resp.Body = msg
+		handler.sendResp(c, resp)
+		return
+	}
+	// -------------------------------------------

```


*** modify ***

```go
func (handler *UnroutedHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	c := handler.getContext(w, r)

	+q := strings.TrimSpace(r.URL.Query().Get("q"))
	+if q != "" {
	+	handler.Query(w, r)
	+	return
	+}
```

*** add ***

```go
func (handler *UnroutedHandler) Query(w http.ResponseWriter, r *http.Request) {
	c := handler.newContext(w, r)
	q := r.URL.Query().Get("q")
	if q == "" {
		handler.sendError(c, errors.New("invalid/empty param q"))
		return
	}
	ctx := context.Background()

	fileList, err := handler.config.StoreComposer.Core.Query(ctx, q)

	if err != nil {
		handler.sendError(c, err)
		return
	}

	_, _ = w.Write(fileList)
}
```

## 9 pkg/s3store/s3store.go

*** add ***

```go
func (store S3Store) Query(ctx context.Context, criteria string) (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}
```

## 10 add file pkg/handler/callbacks.go

package handler

```go
package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

func PreFinishRespCB(evt HookEvent) (resp HTTPResponse, err error) {
	fileName := evt.Upload.Storage["Path"]
	defer func() {
		if err == nil {
			return
		}

		_ = os.RemoveAll(fileName)
		_ = os.RemoveAll(fileName + ".info")
		//resp.StatusCode = 400
		resp.Body = err.Error()
		slog.Info("file" + evt.Upload.MetaData["filename"] + "  deleted by " + err.Error())
		err = nil
	}()

	if fileName == "" {
		err = errors.New("can't find 'Path' from MetaData")
		slog.Error(err.Error())
		return
	}

	fileInfo, err := os.Stat(fileName)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	checkSum := evt.Upload.MetaData["checksum"]
	if checkSum == "" {
		err = errors.New("can't find 'checksum' from MetaData")
		slog.Error(err.Error())
		return
	}
	var n int
	s := evt.Upload.MetaData["filesize"]
	if s == "" || s == "0" {
		err = errors.New("invalid/empty 'filesize' from MetaData")
		slog.Error(err.Error())
		return
	}
	var fileSize uint64
	fileSize, err = strconv.ParseUint(s, 10, 64)
	fd, err := os.Open(fileName)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer func() { _ = fd.Close() }()

	const bufLen = 1024 * 1024 * 4
	buf := make([]byte, bufLen)
	var read int
	var totalRead uint64
	var beCheckSum string

	_, err = fd.Seek(0, 0)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	xxh := xxhash.New()
	totalRead = 0
	read = 0
	for {
		read, err = fd.Read(buf)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		n, err = xxh.Write([]byte(buf[:read]))
		if err != nil {
			slog.Error(err.Error())
			return
		}

		if n != read {
			slog.Error("n!=read")
			return
		}

		totalRead += uint64(read)
		if totalRead == uint64(fileInfo.Size()) {
			break
		}
	}

	beCheckSum = fmt.Sprintf("%02x", xxh.Sum64())
	if beCheckSum != checkSum || fileSize != totalRead {
		err = errors.New("checkSum mismatch, expected: " + beCheckSum + ", got: " + checkSum)
		slog.Error(err.Error())
	}

	return
}

func PreUploadCreateCB(evt HookEvent) (resp HTTPResponse, fic FileInfoChanges, err error) {
	// while fileInfo.Size() < evt.Upload.Size continue to upload
	return
}

func (handler *UnroutedHandler) IsDuplicate(w http.ResponseWriter, r *http.Request) (duplicated bool, err error) {
	metadata := ParseMetadataHeader(r.Header.Get("Upload-Metadata"))
	if metadata == nil {
		return
	}
	checksum, hasChecksum := metadata["checksum"]

	if !hasChecksum || checksum == "" {
		// 如果没有提供 checksum，按正常流程处理
		return
	}

	v := reflect.ValueOf(handler.composer.Core)
	storePath := v.FieldByName("Path").String()
	if storePath == "" {
		err = errors.New("fileStore.Path")
		slog.Error(err.Error())
		return
	}

	fileInfo, err := os.Stat(filepath.Join(storePath, checksum))
	if os.IsNotExist(err) {
		err = nil
		return
	}
	
	if err != nil {
		slog.Error(err.Error())
		return
	}

	s, ok := metadata["filesize"]
	if !ok || s == "" {
		err = errors.New(`metadata["filesize"] isn't set`)
		slog.Error(err.Error())
		return
	}
	expectFileSize, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	if fileInfo.Size() == expectFileSize {
		duplicated = true
		return
	}

	if fileInfo.Size() > expectFileSize {
		err = fmt.Errorf("警告: 文件 %s(%s) 存在但大小不匹配 (已存在: %d, 上传大小: %d), 但checksum竟然相同，难道是sha64不靠谱？",
			metadata["filename"], checksum, fileInfo.Size(), expectFileSize)
		slog.Error(err.Error())
		return
	}

	// while fileInfo.Size() < evt.Upload.Size continue to upload
	return
}

```



## 11 replace all

```github.com/tus/tusd/v2 ```
with
```w2w.io/tusd```

## 12 MUST preprocess before route to tus

```go
type TraceHandler struct {
	Handler *tusd.Handler
}

func (h *TraceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := strings.ToLower(r.Method)
	if method == "post" {
		v := r.Header.Get("Upload-Metadata")
		metadata := r.URL.Query().Get("metadata")
		if v == "" && metadata != "" {
			r.Header.Set("Upload-Metadata", metadata)
		}
	}

	h.Handler.ServeHTTP(w, r)
}
```

### 12.1

if set tus bellow options

```go
		NotifyCompleteUploads:   true,
		NotifyTerminatedUploads: true,
		NotifyUploadProgress:    true,
		NotifyCreatedUploads:    true,
```

then must consume the chan message else tud will be stalled/blocked

```go
	hooks := map[string]chan tusd.HookEvent{
		"CompleteUploads":   handler.CompleteUploads,
		"TerminatedUploads": handler.TerminatedUploads,
		"UploadProgress":    handler.UploadProgress,
		"CreatedUploads":    handler.CreatedUploads,
	}

	for n, c := range hooks {
		go func(n string, c chan tusd.HookEvent) {
			for v := range c {
				fn, ok := v.Upload.MetaData["filename"]
				if ok {
					fn = ", " + fn
				}
				log.Printf("----- %s, %s, %s %s",
					n, v.HTTPRequest.Method, v.Upload.ID, fn)
			}
		}(n, c)
	}
```

