package ai_mark

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"w2w.io/cmn"
)

var defaultChatModel *ChatModel

func init() {
	//cmn.ConfigureForTest()
	InitViper()
	z = cmn.GetLogger()
	var err error
	defaultChatModel, err = GetChatModel()
	if err != nil {
		panic(err)
	}
}

func InitViper() {

	appLaunchPath, err := cmn.GetProjectRoot()
	if err != nil {
		panic(err)
	}

	viper.AddConfigPath(appLaunchPath)
	viper.SetConfigType("json")

	cfgFileName := ".config_" + runtime.GOOS
	viper.SetConfigName(cfgFileName)

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

}

func TestChatModel_SendChatCompletions(t *testing.T) {

	tests := []struct {
		name           string
		messages       []Message
		chatModel      ChatModel
		forceErr       string
		expectedErrStr string
		expectedResp   ChatResponse
		setup          func() ChatModel
	}{
		{
			name:     "success",
			forceErr: "",
			messages: []Message{
				{
					Role:    "system",
					Content: defaultChatModel.GenerateChatPrompt(testedQuestionDetails[0]),
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			expectedErrStr: "",
		},
		{
			name:     "success with default prompt",
			forceErr: "",
			messages: []Message{
				{
					Role:    "system",
					Content: defaultChatModel.GenerateChatPrompt(testedQuestionDetails[0]),
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			expectedErrStr: "",
			setup: func() ChatModel {
				model := *defaultChatModel
				model.Prompt = ""
				return model
			},
		},
		{
			name: "构造请求体失败",
			messages: []Message{
				{
					Role:    "system",
					Content: defaultChatModel.GenerateChatPrompt(testedQuestionDetails[0]),
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			forceErr:       "SendChatCompletions-json.Marshal",
			expectedErrStr: "构造请求体失败",
		},
		{
			name: "发送请求失败",
			messages: []Message{
				{
					Role:    "system",
					Content: defaultChatModel.GenerateChatPrompt(testedQuestionDetails[0]),
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			expectedErrStr: "发送请求失败",
			setup: func() ChatModel {
				model := *defaultChatModel
				model.Endpoint = ""
				return model
			},
		},
		{
			name:     "server returned non-2xx status",
			forceErr: "",
			messages: []Message{
				{
					Role:    "system",
					Content: "假设你是一个负责考试评卷的专家，你需要根据学生回答和标准答案，生成一个仅包含得分点说明的解析。",
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			expectedErrStr: "server returned non-2xx status",
		},
		{
			name: "解析返回的响应体失败",
			messages: []Message{
				{
					Role:    "system",
					Content: defaultChatModel.GenerateChatPrompt(testedQuestionDetails[0]),
				},
				{
					Role: "user",
					Content: `
						[{"student_id":"101","answer":"1. 独立式（Fat AP）组网。 - 优点：部署简单。 - 缺点：管理复杂。"},{"student_id":"102","answer":"1. 独立式组网：每个无线接入点单独配置和管理。 - 优点：简单，成本低。 - 缺点：管理复杂。 2. 控制器集中式（Fit AP + AC）组网：AP受控于无线控制器。 - 优点：集中管理，适合小型网络。 - 缺点：成本高。"}]
						`,
				},
			},
			chatModel:      *defaultChatModel,
			forceErr:       "SendChatCompletions-json.Unmarshal",
			expectedErrStr: "解析返回的响应体失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.chatModel = tt.setup()
			}

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
			chatResp, err := tt.chatModel.SendChatCompletions(ctx, tt.messages)

			z.Sugar().Infof("chatResp: %+v", chatResp)

			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}

func TestChatModel_AIReview(t *testing.T) {
	tests := []struct {
		name           string
		rawContent     *ResponseContent
		chatModel      ChatModel
		forceErr       string
		expectedErrStr string
		expectedResp   ChatResponse
		setup          func() ChatModel
		checkedFunc    func(chatResp ResponseContent) (string, bool)
	}{
		{
			name:       "success",
			rawContent: testedRespResults[0],
			chatModel:  *defaultChatModel,
			checkedFunc: func(chatResp ResponseContent) (string, bool) {
				var msg string
				if len(chatResp.MarkResult) != 1 {
					msg = fmt.Sprintf("长度不匹配，期望1，实际%d", len(chatResp.MarkResult))
					return msg, false
				}

				if chatResp.MarkResult[0].StudentID != 102 {
					msg = fmt.Sprintf("学生ID不匹配，期望102，实际%d", chatResp.MarkResult[0].StudentID)
					return msg, false
				}

				if chatResp.MarkResult[0].Score != 6 {
					msg = fmt.Sprintf("得分不匹配，期望6，实际%f", chatResp.MarkResult[0].Score)
					return msg, false
				}

				return "", true
			},
		},
		{
			name:           "marshal response content error",
			rawContent:     testedRespResults[0],
			chatModel:      *defaultChatModel,
			forceErr:       "AIReview-json.Marshal",
			expectedErrStr: "marshal response content error",
		},
		{
			name:           "SendChatCompletions error",
			rawContent:     testedRespResults[0],
			chatModel:      *defaultChatModel,
			forceErr:       "SendChatCompletions-json.Marshal",
			expectedErrStr: "构造请求体失败",
		},
		{
			name:           "AIReview-大模型服务端出错",
			rawContent:     testedRespResults[0],
			chatModel:      *defaultChatModel,
			forceErr:       "AIReview-大模型服务端出错",
			expectedErrStr: "大模型服务端出错",
		},
		{
			name:           "解析大模型返回消息的json结构失败",
			rawContent:     testedRespResults[0],
			chatModel:      *defaultChatModel,
			expectedErrStr: "解析大模型返回消息的json结构失败",
			setup: func() ChatModel {
				model := *defaultChatModel
				model.ReviewPrompt = "无论输入什么，请你返回一个空数组json结构"
				return model
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.chatModel = tt.setup()
			}

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
			chatResp, err := tt.chatModel.AIReview(ctx, tt.rawContent)

			z.Sugar().Infof("chatResp: %+v", chatResp)

			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}

			if tt.checkedFunc != nil {
				msg, ok := tt.checkedFunc(*chatResp)
				if !ok {
					t.Errorf(msg)
				}
				assert.True(t, ok)
			}
		})
	}
}

func TestChatModel_AIMark(t *testing.T) {
	tests := []struct {
		name           string
		question       *QuestionDetails
		studentAnswers []*StudentAnswer
		chatModel      ChatModel
		forceErr       string
		expectedErrStr string
		expectedResp   ChatResponse
		setup          func() ChatModel
	}{
		{
			name:           "success",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
		},
		{
			name:           "success with default prompt",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
			setup: func() ChatModel {
				model := *defaultChatModel
				model.Prompt = ""
				return model
			},
		},
		{
			name:           "no student answers to mark",
			question:       testedQuestionDetails[0],
			studentAnswers: []*StudentAnswer{},
			chatModel:      *defaultChatModel,
			expectedErrStr: "no student answers to mark",
		},
		{
			name:           "failed to marshal student answers",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
			forceErr:       "AIMark-json.Marshal",
			expectedErrStr: "failed to marshal student answers",
		},
		{
			name:           "SendChatCompletions error",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
			forceErr:       "SendChatCompletions-json.Marshal",
			expectedErrStr: "构造请求体失败",
		},
		{
			name:           "AIMark-大模型服务端出错",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
			forceErr:       "AIMark-大模型服务端出错",
			expectedErrStr: "大模型服务端出错",
		},
		{
			name:           "解析大模型返回消息的json结构失败",
			question:       testedQuestionDetails[0],
			studentAnswers: testedStudentAnswers[0],
			chatModel:      *defaultChatModel,
			expectedErrStr: "解析大模型返回消息的json结构失败",
			setup: func() ChatModel {
				model := *defaultChatModel
				model.Prompt = "无论输入什么，请你返回一个空数组json结构"
				return model
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.chatModel = tt.setup()
			}

			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			}
			chatResp, err := tt.chatModel.AIMark(ctx, tt.question, tt.studentAnswers)

			z.Sugar().Infof("chatResp: %+v", chatResp)

			if err != nil {
				if tt.expectedErrStr == "" {
					t.Errorf("expected success, but got error: %v", err.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				}
			} else if tt.expectedErrStr != "" {
				t.Errorf("expected error: %s, but got success", tt.expectedErrStr)
			}
		})
	}
}
