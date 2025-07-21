package ocr

//annotation:ocr-service
//author:{"name":"OuYangHaoBin","tel":"13712562121","email":"1242968386@qq.com"}

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"w2w.io/cmn"
)

const (
	TIMEOUT    = 10 * time.Second
	TEST       = "test"
	TESTRESULT = "test_result"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("user.Enroll called")

	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: ocr,

		Path: "/ocr",
		Name: "ocr",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// isAllowedFileType 判断文件是否为允许的文件类型
func isAllowedFileType(fileHeader *multipart.FileHeader) bool {
	ext := filepath.Ext(fileHeader.Filename)
	ext = strings.ToLower(ext)
	return ext == ".jpg" || ext == ".png" || ext == ".jpeg"
}

func ocr(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http POST method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	r := q.R

	err := r.ParseMultipartForm(1024 * 1024 * 10)
	if err != nil {
		q.Err = err
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	fileHeaders := r.MultipartForm.File["file"]
	// 检查文件头是否为空
	if fileHeaders == nil {
		err := fmt.Errorf("multipart form file header is nil")
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	if len(fileHeaders) == 0 {
		err := fmt.Errorf("multipart form file header is empty")
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

	apiHost := viper.GetString("ocr.apiHost")
	if apiHost == "" {
		apiHost = "http://127.0.0.1:6268"
	}

	apiPath := viper.GetString("ocr.ocrApiPath")
	if apiPath == "" {
		apiPath = "/api/idCardRecognition"
	}
	url := apiHost + apiPath
	//只拿第一个文件
	fileHeader := fileHeaders[0]
	//检查文件后缀
	if !isAllowedFileType(fileHeader) {
		err := fmt.Errorf("file of name:%s has not allowed file type", fileHeader.Filename)
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}

	var result map[string]interface{}
	result, q.Err = sendHttpRequest(ctx, fileHeader, url, TIMEOUT)
	if q.Err != nil {
		q.RespErr()
		return
	}

	var buf []byte
	buf, q.Err = json.Marshal(&result)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.RowCount = 1
	q.Msg.Data = buf
	q.Resp()
}

func sendHttpRequest(ctx context.Context, fileHeader *multipart.FileHeader, url string, timeout time.Duration) (map[string]interface{}, error) {

	//查看是否需要返回mock的数据
	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			return map[string]interface{}{
				"gender":    "男",
				"id_number": "130927200201210915",
				"name":      "李书腾",
			}, nil
		case "bad-resp":
			return map[string]interface{}{
				"FN": func() {},
			}, nil
		case "sendHttpRequest-error":
			return nil, errors.New("sendHttpRequest error")
		}
	}

	if url == "" {
		url = "http://127.0.0.1:6268/api/idCardRecognition"
	}

	file, err := fileHeader.Open()
	if err != nil {
		z.Error("open file error", zap.Error(err))
		return nil, err
	}
	defer file.Close()

	// 创建缓冲区和 multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建 form file 字段
	part, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		z.Error("create form file error", zap.Error(err))
		return nil, err
	}

	// 将文件内容写入 part
	_, err = io.Copy(part, file)
	if err != nil {
		z.Error("copy file content error", zap.Error(err))
		return nil, err
	}

	// 关闭 writer 以写入结尾 boundary
	err = writer.Close()
	if err != nil {
		z.Error("close writer error", zap.Error(err))
		return nil, err
	}
	// 创建 fasthttp 请求
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType(writer.FormDataContentType()) // 设置 multipart/form-data 和 boundary
	req.SetBody(body.Bytes())

	if timeout <= 0 {
		timeout = TIMEOUT
	}

	// 创建客户端并发送请求
	client := &fasthttp.Client{}
	if err := client.DoTimeout(req, resp, timeout); err != nil {
		z.Error("do request error", zap.Error(err))
		return nil, err
	}

	var result map[string]interface{}
	testResultType, ok := ctx.Value(TESTRESULT).(string)
	if ok && testResultType != "" {
		switch testResultType {
		case "no_msg":
			resp.SetStatusCode(http.StatusInternalServerError)
			result = map[string]interface{}{
				"detail": map[string]interface{}{},
			}
		case "no_data":
			resp.SetStatusCode(http.StatusOK)
			result = map[string]interface{}{}
		}
	} else {
		err = json.Unmarshal(resp.Body(), &result)
		if err != nil {
			z.Error("unmarshal json error", zap.Error(err))
			return nil, err
		}
	}

	if resp.StatusCode() != http.StatusOK {
		detail, ok := result["detail"].(map[string]interface{})
		if !ok {
			err := errors.New("key of detail is not in map[string]interface{}")
			z.Error(err.Error())
			return nil, err
		}
		msg, ok := detail["msg"].(string)
		if !ok {
			err := errors.New("key of msg is not in map[string]interface{}")
			z.Error(err.Error())
			return nil, err
		}
		return nil, errors.New(msg)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		err := errors.New("key of data is not in map[string]interface{}")
		z.Error(err.Error())
		return nil, err
	}
	return data, nil
}
