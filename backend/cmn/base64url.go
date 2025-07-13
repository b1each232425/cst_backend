package cmn

import (
	"encoding/base64"
	"strings"
)

// B64UDecode decodes base64url string to byte array
func B64UDecode(data string) ([]byte, error) {
	data = strings.Replace(data, "-", "+", -1) // 62nd char of encoding
	data = strings.Replace(data, "_", "/", -1) // 63rd char of encoding

	// Pad with trailing '='s
	switch len(data) % 4 {
	// no padding
	case 0:
	case 1:
	// 2 pad chars
	case 2:
		data += "=="
	// 1 pad char
	case 3:
		data += "="
	}

	return base64.StdEncoding.DecodeString(data)
}

// B64UEncode encodes given byte array to base64url string
func B64UEncode(data []byte) string {
	result := base64.StdEncoding.EncodeToString(data)
	// 62nd char of encoding
	result = strings.Replace(result, "+", "-", -1)
	// 63rd char of encoding
	result = strings.Replace(result, "/", "_", -1)
	// Remove any trailing '='s
	result = strings.Replace(result, "=", "", -1)

	return result
}
