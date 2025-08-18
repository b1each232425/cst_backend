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
