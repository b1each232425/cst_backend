package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"testing"

	"w2w.io/cmn"
)

// 定义测试中使用的错误
var (
	errorspkg = struct {
		ErrFileHeaderNil error
	}{
		ErrFileHeaderNil: errors.New("file header is nil"),
	}
)

func Test_ocr(t *testing.T) {
	// 创建测试用例
	tests := []struct {
		name         string
		filePath     string
		expectedErr  error
		expectedResp map[string]interface{}
	}{
		{"common1", "./test_photo/img_1.png", nil,

			map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			},
		},
		{"common2", "./test_photo/img_2.jpg", nil,
			map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			},
		},
		{"common3", "./test_photo/img_7.jpeg", nil,
			map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			},
		},
		{"sendHttpRequest-error", "./test_photo/img_7.jpeg", errors.New(""), nil},
		{"marshal_error", "./test_photo/img_7.jpeg", errors.New(""), nil},
		{"txt_format", "./test_photo/test.txt", errors.New(""), nil},
		{"nil_fileHeader", "", errorspkg.ErrFileHeaderNil, nil},
		// 添加额外的测试用例以提高覆盖率
		{"empty_fileHeader", "./test_photo/empty.jpg", errors.New("multipart form file header is empty"), nil},
		{"nil_multipartForm", "./test_photo/nil_form.jpg", errors.New("MultipartForm is nil while it shouldn't be"), nil},
		{"method_not_post", "./test_photo/method.jpg", errors.New("please call /api/upLogin with  http POST method"), nil},
		// 添加测试 ParseMultipartForm 错误的测试用例
		{"parse_multipart_form_error", "./test_photo/img_1.png", errors.New("parse multipart form error"), nil},
	}

	// 遍历测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP请求和上下文
			ctx := createMockContext(t, tt.name, tt.filePath, tt.expectedErr, tt.expectedResp)

			// 调用被测试的函数
			ocr(ctx)

			// 验证结果
			q := cmn.GetCtxValue(ctx)
			if tt.expectedErr != nil {
				if q.Err == nil {
					t.Errorf("Expected error %v, but got nil", tt.expectedErr)
				}
			} else if tt.expectedResp != nil {
				// 验证响应数据
				if q.Msg.Data == nil {
					t.Errorf("Expected response data, but got nil")
					return
				}
				var result map[string]interface{}
				err := json.Unmarshal(q.Msg.Data, &result)
				if err != nil {
					t.Errorf("Expected no error, but got %v", err)
					return
				}

				if !reflect.DeepEqual(result["gender"], tt.expectedResp["gender"]) {
					t.Errorf("Expected response %v, but got %v", tt.expectedResp, result)
				}
				if !reflect.DeepEqual(result["name"], tt.expectedResp["name"]) {
					t.Errorf("Expected response %v, but got %v", tt.expectedResp, result)
				}
				if !reflect.DeepEqual(result["id_number"], tt.expectedResp["id_number"]) {
					t.Errorf("Expected response %v, but got %v", tt.expectedResp, result)
				}

				// 这里可以添加更详细的响应验证逻辑
			}
		})
	}
}

// handleFile 函数用于处理文件上传，返回一个包含文件信息的multipart.FileHeader切片
func handleFile(file *os.File) []*multipart.FileHeader {
	// 创建一个 buffer 来存储 multipart 数据
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// 添加文件到 multipart 中
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		panic(err)
	}

	// 将文件内容复制到 multipart 中
	_, err = io.Copy(part, file)
	if err != nil {
		panic(err)
	}

	// 关闭 multipart writer
	err = writer.Close()
	if err != nil {
		panic(err)
	}
	reader := multipart.NewReader(&buffer, writer.Boundary())
	form, err := reader.ReadForm(50 << 20) // 50MB 限制
	if err != nil {
		panic(err)
	}

	fileHeaders := form.File["file"]

	return fileHeaders

}

// 创建模拟的HTTP上下文
func createMockContext(t *testing.T, testName, filePath string, expectedErr error, expectedResp map[string]interface{}) context.Context {
	// 创建基本的上下文
	ctx := context.Background()

	// 创建模拟的HTTP请求
	req, _ := http.NewRequest("POST", "/api/ocr", nil)

	// 根据测试用例设置不同的请求属性
	switch testName {
	case "method_not_post":
		req, _ = http.NewRequest("GET", "/ocr", nil)
	case "nil_multipartForm":
		// 不设置MultipartForm，保持为nil
	case "nil_fileHeader":
		// 设置空的MultipartForm但不添加文件
		req.MultipartForm = &multipart.Form{
			File: make(map[string][]*multipart.FileHeader),
		}
	case "empty_fileHeader":
		// 设置空的文件头数组
		req.MultipartForm = &multipart.Form{
			File: map[string][]*multipart.FileHeader{
				"file": {},
			},
		}
	case "parse_multipart_form_error":
		// 创建一个自定义的请求，模拟 ParseMultipartForm 返回错误
		req = &http.Request{
			Method: "POST",
			URL:    req.URL,
			Header: make(http.Header),
		}
		// 设置一个特殊的 Content-Type，使 ParseMultipartForm 失败
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")

	default:
		if filePath != "" {
			var fileHeaders []*multipart.FileHeader
			var file *os.File
			file, err := os.Open(filePath)
			if err != nil {
				t.Errorf("failed to open file: %v", err)
				return nil
			}

			fileHeaders = handleFile(file)
			// 为其他测试用例设置有效的MultipartForm和FileHeader
			req.MultipartForm = &multipart.Form{
				File: map[string][]*multipart.FileHeader{
					"file": fileHeaders,
				},
			}
		}
	}

	// 创建模拟的服务上下文
	q := &cmn.ServiceCtx{
		R: req,
		Msg: &cmn.ReplyProto{
			Data: nil,
		},
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)
	if testName != "common1" {
		testValue := "normal-resp"
		//存入测试字段
		switch testName {
		case "sendHttpRequest-error":
			testValue = "sendHttpRequest-error"
		case "marshal_error":
			testValue = "bad-resp"
		default:
			testValue = "normal-resp"
		}
		ctx = context.WithValue(ctx, TEST, testValue)
	}

	// 确保测试目录存在
	ensureTestDirectoryExists(t)

	return ctx
}

// 确保测试目录存在
func ensureTestDirectoryExists(t *testing.T) {
	dirPath := "./test_photo"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}
}

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func Test_sendHttpRequest(t *testing.T) {
	// 创建一个测试文件
	testFilePath := "./test_photo/img_1.png"

	// 打开测试文件
	testFile, err := os.Open(testFilePath)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer testFile.Close()

	// 创建文件头
	fileHeaders := handleFile(testFile)
	var fileHeader *multipart.FileHeader
	if len(fileHeaders) > 0 {
		fileHeader = fileHeaders[0]
	}

	type args struct {
		ctx        context.Context
		fileHeader *multipart.FileHeader
		url        string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "normal response test",
			args: args{
				ctx:        context.WithValue(context.Background(), TEST, "normal-resp"),
				fileHeader: fileHeader,
				url:        "http://example.com",
			},
			want: map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			},
			wantErr: false,
		},
		{
			name: "error response test",
			args: args{
				ctx:        context.WithValue(context.Background(), TEST, "sendHttpRequest-error"),
				fileHeader: fileHeader,
				url:        "http://example.com",
			},
			want:    nil,
			wantErr: true,
		},
		//{
		//	name: "nil file header test",
		//	args: args{
		//		ctx:        context.Background(),
		//		fileHeader: nil,
		//		url:        "http://example.com",
		//	},
		//	want:    nil,
		//	wantErr: true,
		//},
		{
			name: "empty url test",
			args: args{
				ctx:        context.Background(),
				fileHeader: fileHeader,
				url:        "",
			},
			want: map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			},
			wantErr: false,
		},
		{
			name: "error url test",
			args: args{
				ctx:        context.Background(),
				fileHeader: fileHeader,
				url:        "http://127.0.0.1:6268/api/error",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error detail has no msg test",
			args: args{
				ctx:        context.WithValue(context.Background(), TESTRESULT, "no_msg"),
				fileHeader: fileHeader,
				url:        "http://127.0.0.1:6268/api/error",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error result has no data  test",
			args: args{
				ctx:        context.WithValue(context.Background(), TESTRESULT, "no_data"),
				fileHeader: fileHeader,
				url:        "http://127.0.0.1:6268/api/error",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sendHttpRequest(tt.args.ctx, tt.args.fileHeader, tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("sendHttpRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got["name"], tt.want["name"]) {
				t.Errorf("sendHttpRequest() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got["gender"], tt.want["gender"]) {
				t.Errorf("sendHttpRequest() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got["id_number"], tt.want["id_number"]) {
				t.Errorf("sendHttpRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
