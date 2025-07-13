package uid

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"w2w.io/tusd/pkg/handler"
)

// uid returns a unique id. These ids consist of 128 bits from a
// cryptographically strong pseudo-random generator and are like uuids, but
// without the dashes and significant bits.
//
// See: http://en.wikipedia.org/wiki/UUID#Random_UUID_probability_of_duplicates
func Uid() string {
	id := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		// This is probably an appropriate way to handle errors from our source
		// for random bits.
		panic(err)
	}
	return hex.EncodeToString(id)
}

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
