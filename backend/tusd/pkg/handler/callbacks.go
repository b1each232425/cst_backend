package handler

import (
	"errors"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"log/slog"
	"os"
	"strconv"
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
	for k, v := range evt.Upload.MetaData {
		slog.Info("--", k, v)
	}
	return
}
