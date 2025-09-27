package ai_mark

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"w2w.io/cmn"
)

var defaultChatModel ChatModel
var defaultMessages []Message

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()

	err := initChatModel()
	if err != nil {
		panic(err)
	}

	defaultChatModel = *chatModel

	defaultMessages = []Message{
		{
			Role: "system",
			MContent: []MessageContent{
				{Content: "你是一个作词家，你可以根据所给的关键词作词，并以 JSON 格式返回结果，例如 {\"lyrics\": \"xxx\"}", Type: "text"},
			},
		},
		{
			Role: "user",
			MContent: []MessageContent{
				{Content: "请根据关键词 '鸟儿，晴天' 作词，并以 JSON 格式返回：{\"lyrics\": \"...\"}", Type: "text"},
			},
		},
	}

}

func TestChatModel_SendChatCompletions(t *testing.T) {
	tests := []struct {
		name           string
		messages       []Message
		forceErr       string
		expectedErrStr string
	}{
		{
			name:     "success",
			messages: defaultMessages,
		},
		{
			name:           "error: 构造请求体失败",
			forceErr:       "json.Marshal-requestBody",
			expectedErrStr: "构造请求体失败",
		},
		{
			name:           "error: 发送请求失败",
			messages:       defaultMessages,
			forceErr:       "client.DoTimeout",
			expectedErrStr: "发送请求失败",
		},
		{
			name:           "error: ai 服务端出错",
			messages:       defaultMessages,
			forceErr:       "resp.StatusCode()",
			expectedErrStr: "大模型服务端返回 非2xx 响应码",
		},
		{
			name:           "error: 解析返回的响应体失败",
			messages:       defaultMessages,
			forceErr:       "json.Unmarshal-bodyText",
			expectedErrStr: "解析返回的响应体失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			chatResp, err := defaultChatModel.SendChatCompletions(ctx, tt.messages)
			z.Sugar().Infof("chatResp: %+v", chatResp)

			if tt.expectedErrStr != "" {
				assert.Emptyf(t, chatResp, "期待获取空结构体，但获得：%v", chatResp)
				assert.Error(t, err, "期待错误，但是没有")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NotEmpty(t, chatResp, "期待获取非空结构体，但获得空结构体")
				assert.NoErrorf(t, err, "期待没错，但是获得错误: %v", err)
			}
		})
	}
}

func TestChatModel_AIMark(t *testing.T) {
	var tests = []struct {
		name                string
		questions           []*QuestionDetail
		studentAnswers      []*StudentAnswer
		sendChatCompletions func(context.Context, []Message) (ChatResponse, error)
		forceErr            string
		expectResp          ResponseContent
		expectedErrStr      string
	}{
		{
			name:           "success",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{
					Choices: []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
						FinishReason string `json:"finish_reason"`
					}{
						{
							Message: struct {
								Content string `json:"content"`
							}{
								Content: string(mustMarshal(TestedRespResults[0])),
							},
						},
					},
				}, nil
			},
			expectResp: TestedRespResults[0],
		},
		{
			name:           "success: 没有问题",
			questions:      []*QuestionDetail{},
			studentAnswers: TestedStudentAnswers[0],
			expectResp:     ResponseContent{},
		},
		{
			name:           "success: 没有学生",
			questions:      TestedQuestionDetails[0],
			studentAnswers: []*StudentAnswer{},
			expectResp:     ResponseContent{},
		},
		{
			name:           "error: 构造 questions json 失败",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			forceErr:       "json.Marshal-questions",
			expectedErrStr: "构造 questions json 失败",
		},
		{
			name:           "error: 构造 student answers json 失败",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			forceErr:       "json.Marshal-studentAnswers",
			expectedErrStr: "构造 student answers json 失败",
		},
		{
			name:           "error: 发送 http 请求失败",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{}, errors.New("mock http error")
			},
			expectedErrStr: "发送 http 请求失败",
		},
		{
			name:           "error: 大模型服务端出错-错误码存在",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{
					Error: struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					}{
						Code: "500",
					},
				}, nil
			},
			expectedErrStr: "大模型服务端出错",
		},
		{
			name:           "error: 大模型服务端出错-错误消息存在",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{
					Error: struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					}{
						Message: "mock server error",
					},
				}, nil
			},
			expectedErrStr: "大模型服务端出错",
		},
		{
			name:           "error: 大模型没有回复任何结果",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{}, nil
			},
			expectedErrStr: "大模型没有回复任何结果",
		},
		{
			name:           "error: 解析大模型返回消息的 json 结构 失败",
			questions:      TestedQuestionDetails[0],
			studentAnswers: TestedStudentAnswers[0],
			sendChatCompletions: func(ctx context.Context, messages []Message) (ChatResponse, error) {
				return ChatResponse{
					Choices: []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
						FinishReason string `json:"finish_reason"`
					}{
						{
							Message: struct {
								Content string `json:"content"`
							}{
								Content: "{",
							},
						},
					},
				}, nil
			},
			expectedErrStr: "解析大模型返回消息的 json 结构 失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			resp, err := defaultChatModel.AIMark(ctx, tt.questions, tt.studentAnswers, tt.sendChatCompletions)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待得到错误，但是没有")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但是获得错误: %v", err)
				assert.Equal(t, tt.expectResp, resp)
			}
		})
	}
}

//func TestChatModel_Tokenizer(t *testing.T) {
//	tests := []struct {
//		name           string
//		texts          []string
//		chatModel      ChatModel
//		forceErr       string
//		expectedErrStr string
//		expectedTokens []int
//		setup          func() ChatModel
//	}{
//		{
//			name:     "success",
//			forceErr: "",
//			texts: []string{
//				"天空为什么这么蓝",
//				"花儿为什么这么香",
//			},
//			chatModel:      *defaultChatModel,
//			expectedErrStr: "",
//		},
//		{
//			name: "构造请求体JSON失败",
//			texts: []string{
//				"天空为什么这么蓝",
//				"花儿为什么这么香",
//			},
//			chatModel:      *defaultChatModel,
//			forceErr:       "Tokenizer-json.Marshal",
//			expectedErrStr: "构造请求体JSON失败",
//		},
//		{
//			name: "发送请求失败",
//			texts: []string{
//				"天空为什么这么蓝",
//				"花儿为什么这么香",
//			},
//			chatModel:      *defaultChatModel,
//			expectedErrStr: "发送请求失败",
//			setup: func() ChatModel {
//				model := *defaultChatModel
//				model.tokenizerEndPoint = ""
//				return model
//			},
//		},
//		{
//			name:     "server returned non-2xx status",
//			forceErr: "",
//			//texts: []string{
//			//	"天空为什么这么蓝",
//			//	"花儿为什么这么香",
//			//},
//			chatModel:      *defaultChatModel,
//			expectedErrStr: "server returned non-2xx status",
//		},
//		{
//			name: "解析返回的响应体失败",
//			texts: []string{
//				"天空为什么这么蓝",
//				"花儿为什么这么香",
//			},
//			chatModel:      *defaultChatModel,
//			forceErr:       "Tokenizer-json.Unmarshal",
//			expectedErrStr: "解析返回的响应体失败",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if tt.setup != nil {
//				tt.chatModel = tt.setup()
//			}
//
//			ctx := context.Background()
//			if tt.forceErr != "" {
//				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//			}
//			tokenResp, err := tt.chatModel.Tokenizer(ctx, tt.texts)
//
//			z.Sugar().Infof("tokenResp: %+v", tokenResp)
//
//			if err != nil {
//				if tt.expectedErrStr == "" {
//					t.Errorf("expected success, but got error: %v", err.Error())
//				} else {
//					assert.Contains(t, err.Error(), tt.expectedErrStr)
//				}
//			} else if tt.expectedErrStr != "" {
//				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
//			}
//		})
//	}
//}

// 辅助函数，快速生成 JSON []byte
func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
