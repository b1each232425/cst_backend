package ai_detection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"w2w.io/cmn"
	"w2w.io/null"
)

const sampleQuestion = `{
	"type": "00",
	"content": "下列关于计算机网络的说法、错误的是（）。",
	"options": [
		{"label": "A", "value": "计算机网络是由计算机?和通信设备通过通信线路互联而成的系统"},
		{"label": "B", "value": "计算机网络的主要功能是实现资源共享和信息传递"},
		{"label": "C", "value": "计算机网络按覆盖范围可分为局域往，城域和网广域网"},
		{"label": "D", "value": "计算机网络的传输介质主要有双绞线、同轴电缆和光纤等"}
	],
	"answers": "A"
}`

func TestAIModel_SendDetectionRequest(t *testing.T) {
	cmn.ConfigureForTest()
	AIModelInstance, err := GetAIModel()
	if err != nil {
		t.Fatalf("GetAIModel failed: %v", err)
	}
	tests := []struct {
		name          string
		wantErr       bool
		forceErr      string
		wantResponse  cmn.TQuestion
		errorContains string
		messages      []Message
		checkResponse func(got, want cmn.TQuestion) bool
	}{
		{
			name: "正常请求",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "",
			wantErr:       false,
			errorContains: "",
		},
		{
			name: "构造请求体失败",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "json.Marshal",
			wantErr:       true,
			errorContains: "构造请求体失败",
		},
		{
			name: "发送请求失败",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "DoTimeout",
			wantErr:       true,
			errorContains: "发送请求失败",
		},
		{
			name: "无法正常响应",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "StatusCode",
			wantErr:       true,
			errorContains: "无法正常响应",
		},
		{
			name: "解析返回的响应体失败",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "json.Unmarshal",
			wantErr:       true,
			errorContains: "解析返回的响应体失败",
		},
		{
			name: "解析大模型返回消息的json结构失败",
			messages: []Message{
				{
					Role: "system", MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
				},
				{
					Role: "user", MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + sampleQuestion, Type: "text"}},
				},
			},
			forceErr:      "json.Unmarshal2",
			wantErr:       true,
			errorContains: "解析大模型返回消息的json结构失败",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, "send-detection-request-forceErr", tt.forceErr)
			}
			_, err := AIModelInstance.SendDetectionRequest(ctx, tt.messages)
			if (err != nil) != tt.wantErr {
				t.Errorf("AIModel.SendDetectionRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("AIModel.SendDetectionRequest() error = %v, want contains %v", err, tt.errorContains)
				return
			}
		})
	}
}

func createMockContextWithBody(method, path string, data string, forceError string, userID int64, userRole int64) context.Context {
	var req *http.Request

	if data != "" {
		// 创建ReqProto结构体，Data字段使用json.RawMessage类型
		body := &cmn.ReqProto{
			Data: json.RawMessage(data),
		}

		// 将请求体转换为JSON字符串
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			e := fmt.Sprintf("Failed to marshal request data: %v", err)
			z.Fatal(e)
		}

		// 创建mock HTTP请求
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.Header.Set("Content-Type", "application/json")

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2001),
		Domain: "assess^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2002),
		Domain: "assess.academicAffair^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2003),
		Domain: "assess^teacher",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2008),
		Domain: "assess^student",
	})

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID:   null.NewInt(userID, true),   // 请求用户ID
			Role: null.NewInt(userRole, true), // 用户角色ID
		},
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

func Test_aiDetection(t *testing.T) {
	cmn.ConfigureForTest()
	tests := []struct {
		name          string
		wantErr       bool
		forceErr      string
		errorContains string
		qusetion      cmn.TQuestion
		isBadResp     bool
		method        string
	}{
		{
			name:          "强制获取AIModel错误",
			forceErr:      "GetAIModel",
			qusetion:      cmn.TQuestion{},
			isBadResp:     false,
			wantErr:       true,
			errorContains: "强制获取AIModel错误",
			method:        "POST",
		},
		{
			name:          "正常流程",
			forceErr:      "",
			qusetion:      cmn.TQuestion{},
			isBadResp:     false,
			wantErr:       false,
			errorContains: "",
			method:        "POST",
		},
		{
			name:          "解析请求体错误",
			forceErr:      "json.Unmarshal",
			qusetion:      cmn.TQuestion{},
			isBadResp:     false,
			wantErr:       true,
			errorContains: "强制JSON解析错误",
			method:        "POST",
		},
		{
			name:          "解析请求体错误2",
			forceErr:      "json.Unmarshal2",
			qusetion:      cmn.TQuestion{},
			isBadResp:     false,
			wantErr:       true,
			errorContains: "强制第二次JSON解析错误",
			method:        "POST",
		},
		{
			name:          "SendDetectionRequest 返回错误",
			qusetion:      cmn.TQuestion{},
			forceErr:      "",
			isBadResp:     true,
			wantErr:       true,
			errorContains: "这是一个测试错误",
			method:        "POST",
		},
		{
			name:          "强制JSON序列化错误",
			qusetion:      cmn.TQuestion{},
			forceErr:      "json.Marshal",
			isBadResp:     false,
			wantErr:       true,
			errorContains: "强制JSON序列化错误",
			method:        "POST",
		},
		{
			name:          "强制JSON序列化错误2",
			qusetion:      cmn.TQuestion{},
			forceErr:      "json.Marshal2",
			isBadResp:     false,
			wantErr:       true,
			errorContains: "强制JSON序列化错误2",
			method:        "POST",
		},
		{
			name:          "请求体为空",
			qusetion:      cmn.TQuestion{},
			forceErr:      "",
			isBadResp:     false,
			wantErr:       true,
			errorContains: "请求体为空",
			method:        "POST",
		},
		{
			name:          "不支持的请求方法",
			qusetion:      cmn.TQuestion{},
			forceErr:      "",
			isBadResp:     false,
			wantErr:       true,
			errorContains: "不支持的请求方法",
			method:        "GET",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.qusetion)
			if tt.name == "请求体为空" {
				reqBody = []byte{}
			}
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			ctx := createMockContextWithBody(tt.method, "/ai-detection", string(reqBody), tt.forceErr, 1626, 2001)
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, "aiDetection-force-error", tt.forceErr)
			}
			if tt.isBadResp {
				ctx = context.WithValue(ctx, "send-detection-request-forceErr", "bad-resp")
			} else {
				ctx = context.WithValue(ctx, "send-detection-request-forceErr", "normal-resp")
			}

			aiDetection(ctx)
			q := cmn.GetCtxValue(ctx)

			// 验证结果
			if tt.wantErr != (q.Err != nil) {
				t.Errorf("%s: aiDetection() error = %v, wantErr %v", tt.name, q.Err, tt.wantErr)
			}
			if tt.errorContains != "" {
				assert.NotNil(t, q.Err, "%s: 期望有错误但没有收到", tt.name)
				if tt.errorContains != "" {
					assert.Contains(t, q.Err.Error(), tt.errorContains, "%s: 错误消息不匹配", tt.name)
				}
				t.Logf("%s: 收到期望的错误: %v", tt.name, q.Err)
			} else {
				if q.Err != nil {
					t.Errorf("%s: 意外错误: %v", tt.name, q.Err)
				} else {
					assert.NotNil(t, q.Msg, "%s: 期望有响应消息", tt.name)
					t.Logf("%s: 操作成功完成", tt.name)
				}
			}
		})
	}
}
