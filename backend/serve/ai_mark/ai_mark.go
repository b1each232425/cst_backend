package ai_mark

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"strings"
	"time"
	"w2w.io/cmn"
)

type Service interface {
	SendChatCompletions(ctx context.Context, messages []Message) (chatResp ChatResponse, err error)
	AIMark(ctx context.Context, question *QuestionDetails, studentAnswers []*StudentAnswer) (respContent *ResponseContent, err error)
	AIReview(ctx context.Context, rawContent *ResponseContent) (respContent *ResponseContent, err error)
	GenerateChatPrompt(question *QuestionDetails) (prompt string)
}

type ChatModel struct {
	Model             string
	Endpoint          string
	ApiKey            string
	Prompt            string
	ReviewPrompt      string
	MaxConcurrency    int
	TokenizerEndPoint string
	MaxTokens         int            `json:"max_tokens"`
	ResponseFormat    ResponseFormat `json:"response_format"`
}

type ChatResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Id    string `json:"id"`
	Model string `json:"model"`
}

type ResponseContent struct {
	MarkResult []struct {
		StudentID int64   `json:"student_id"`
		Score     float64 `json:"score"`
		Analyze   string  `json:"analyze"`
	} `json:"mark_result"`
}

type RequestBody struct {
	Model          string         `json:"model"`
	ResponseFormat ResponseFormat `json:"response_format"`
	MaxTokens      int            `json:"max_tokens"`
	Messages       []Message      `json:"messages"`
	Temperature    float32        `json:"temperature"`
	TopP           float32        `json:"top_p"`
	//Thinking       Think          `json:"thinking"`
}

type Message struct {
	Role     string           `json:"role"`
	Content  string           `json:"content1"`
	Type     string           `json:"type"`
	MContent []MessageContent `json:"content"`
}

type MessageContent struct {
	Content string `json:"text"`
	Type    string `json:"type"`
}

type Think struct {
	Type string `json:"type"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type TokenizerRequest struct {
	Model string   `json:"model"`
	Text  []string `json:"text"`
}

type TokenizerResponse struct {
	Object  string             `json:"object"`
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Data    []TokenizationInfo `json:"data"`
	Created int                `json:"created"`
}

type TokenizationInfo struct {
	Object        string  `json:"object"`
	Index         int     `json:"index"`
	TotalTokens   int     `json:"total_tokens"`
	TokenIDs      []int   `json:"token_ids"`
	OffsetMapping [][]int `json:"offset_mapping"`
}

// SubjectiveAnswer 主观题答案结构
type SubjectiveAnswer struct {
	Index             int      `json:"index"`              // 答案索引
	Answer            string   `json:"answer"`             // 答案
	AlternativeAnswer []string `json:"alternative_answer"` // 备选答案
	Score             float64  `json:"score"`              // 分数
	GradingRule       string   `json:"grading_rule"`       // 评分规则
}

type QuestionDetails struct {
	QuestionID int64   `json:"question_id"`
	Index      int     `json:"index"`
	Score      float64 `json:"score"`
	Content    string  `json:"question"`
	Answer     string  `json:"answer"`
	Rule       string  `json:"rule"`
	//Prompt       string `json:"prompt"`
	//ReviewPrompt string `json:"review_prompt"`
}

type StudentAnswer struct {
	StudentID int64  `json:"student_id"`
	Answer    string `json:"answer"`
	Index     int    `json:"index"`
}

var chatModel *ChatModel
var z *zap.Logger

const ForceErrKey = "force-err"

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func NewChatModel(model string, endpoint string, apiKey string, prompt string, reviewPrompt string, tokenizerEndPoint string, maxConcurrency int) *ChatModel {
	return &ChatModel{
		Model:             model,
		Endpoint:          endpoint,
		ApiKey:            apiKey,
		TokenizerEndPoint: tokenizerEndPoint,
		MaxConcurrency:    maxConcurrency,
		MaxTokens:         16 * 1024,
		ResponseFormat:    ResponseFormat{Type: "json_object"},
		Prompt:            prompt,
		ReviewPrompt:      reviewPrompt,
	}
}

func GetChatModel() (Service, error) {
	if chatModel == nil {
		err := Init()
		return chatModel, err
	}
	return chatModel, nil
}

func SetChatModel(c *ChatModel) {
	chatModel = c
}

func Init() error {
	if chatModel != nil {
		return nil
	}

	apiKey := viper.GetString("chatModel.apiKey")
	model := viper.GetString("chatModel.model")
	endPoint := viper.GetString("chatModel.endpoint")
	tokenizerEndPoint := viper.GetString("chatModel.tokenizer.endpoint")
	maxConcurrency := viper.GetInt("chatModel.maxConcurrency")

	if apiKey == "" || model == "" || endPoint == "" || tokenizerEndPoint == "" || maxConcurrency <= 0 {
		err := fmt.Errorf("从配置文件获取chatModel.model, chatModel.apiKey, chatModel.endpoint, chatModel.tokenizer.endpoint, chatModel.maxConcurrency 配置项失败")
		z.Error(err.Error())
		return err
	}

	chatModel = NewChatModel(model, endPoint, apiKey, promptTmpl, reviewPrompt, tokenizerEndPoint, maxConcurrency)
	return nil

}

func (c *ChatModel) GetChatMaxConcurrency() int {
	return c.MaxConcurrency
}

func (c *ChatModel) SendChatCompletions(ctx context.Context, messages []Message) (chatResp ChatResponse, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	// 构建请求体
	requestBody := RequestBody{
		Model:          c.Model,
		ResponseFormat: c.ResponseFormat,
		MaxTokens:      c.MaxTokens,
		Messages:       messages,
		Temperature:    0.5,
		TopP:           0.5,
		//Thinking:       Think{Type: "disabled"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil || forceErr == "SendChatCompletions-json.Marshal" {
		err = fmt.Errorf("构造请求体失败: %v", err)
		z.Error(err.Error())
		return
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 设置请求方法和URL
	req.Header.SetMethod("POST")
	req.SetRequestURI(c.Endpoint)

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.SetBody(jsonData)

	// 创建客户端并发送请求
	client := &fasthttp.Client{}
	// 使用DoTimeout来实现超时控制
	timeoutDuration := 10 * 60 * time.Second
	fastErr := client.DoTimeout(req, resp, timeoutDuration)
	if fastErr != nil {
		err = fmt.Errorf("发送请求失败: %v", fastErr)
		z.Error(err.Error())
		return
	}

	// 检查响应状态码
	if resp.StatusCode() >= 400 {
		err = fmt.Errorf("server returned non-2xx status: %d, content: %s", resp.StatusCode(), resp.String())
		z.Error(err.Error())
		return
	}

	// 读取响应体
	bodyText := resp.Body()

	err = json.Unmarshal(bodyText, &chatResp)
	if err != nil || forceErr == "SendChatCompletions-json.Unmarshal" {
		err = fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
		z.Error(err.Error())
		return
	}

	return
}

func (c *ChatModel) AIMark(ctx context.Context, question *QuestionDetails, studentAnswers []*StudentAnswer) (respContent *ResponseContent, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	if c.Prompt == "" || c.ReviewPrompt == "" {
		c.Prompt = promptTmpl
		c.ReviewPrompt = reviewPrompt
	}

	if len(studentAnswers) == 0 {
		err = fmt.Errorf("no student answers to mark")
		return
	}

	jsonData, err := json.Marshal(studentAnswers)
	if err != nil || forceErr == "AIMark-json.Marshal" {
		err = fmt.Errorf("failed to marshal student answers: %v", err)
		return
	}

	sysPrompt := c.GenerateChatPrompt(question)

	var builder strings.Builder
	builder.Write(jsonData)

	content := builder.String()

	//var messages = []Message{
	//	{
	//		Role:    "system",
	//		Content: sysPrompt + "\n",
	//	},
	//	{
	//		Role:    "user",
	//		Content: content,
	//	},
	//}

	var messages = []Message{
		{

			Role: "system",
			MContent: []MessageContent{
				{Content: sysPrompt + "\n",
					Type: "text"},
			},
			//Content: sysPrompt + "\n",
		},
		{
			Role: "user",
			MContent: []MessageContent{
				{Content: content,
					Type: "text"},
			},
			//Content: content,
		},
	}

	start := time.Now()

	z.Sugar().Infof("sending AIMark request, question_id:(%d), content: %s", question.QuestionID, content)

	chatResp, err := c.SendChatCompletions(ctx, messages)
	if err != nil {
		return
	}
	elapsed := time.Since(start)
	z.Sugar().Infof("%s Time taken: %s", c.Model, elapsed)

	if chatResp.Error.Code != "" || chatResp.Error.Message != "" || forceErr == "AIMark-大模型服务端出错" {
		err = fmt.Errorf("大模型服务端出错：%+v", chatResp.Error)
		return
	}

	//z.Sugar().Infof("Response: %s", chatResp.Choices[0].Message.Content)

	var resp ResponseContent
	err = json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &resp)
	if err != nil {
		err = fmt.Errorf("解析大模型返回消息的json结构失败: %v, 内容：%v", err, chatResp.Choices[0].Message.Content)
		z.Error(err.Error())
		return
	}

	respContent = &resp

	z.Sugar().Infof("AIMark response: %+v, question: (%d)", respContent.MarkResult, question.QuestionID)

	return c.AIReview(ctx, respContent)
}

func (c *ChatModel) AIReview(ctx context.Context, rawContent *ResponseContent) (respContent *ResponseContent, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	respJson, err := json.Marshal(rawContent)
	if err != nil || forceErr == "AIReview-json.Marshal" {
		err = fmt.Errorf("marshal response content error: %v", err)
		return
	}

	//z.Sugar().Infof("AIReview request content: %s", string(respJson))

	//messages := []Message{
	//	{
	//		Role:    "system",
	//		Content: c.ReviewPrompt,
	//	},
	//	{
	//		Role:    "user",
	//		Content: string(respJson),
	//	},
	//}

	var messages = []Message{
		{

			Role: "system",
			MContent: []MessageContent{
				{
					Content: c.ReviewPrompt,
					Type:    "text",
				},
			},
			//Content: sysPrompt + "\n",
		},
		{
			Role: "user",
			MContent: []MessageContent{
				{Content: string(respJson),
					Type: "text"},
			},
			//Content: content,
		},
	}

	start := time.Now()

	chatResp, err := c.SendChatCompletions(ctx, messages)
	if err != nil {
		return
	}
	elapsed := time.Since(start)
	z.Sugar().Infof("%s (review) Time taken: %s", c.Model, elapsed)

	if chatResp.Error.Code != "" || chatResp.Error.Message != "" || forceErr == "AIReview-大模型服务端出错" {
		err = fmt.Errorf("大模型服务端出错：%+v", chatResp.Error)
		z.Error(err.Error())
		return
	}

	var resp ResponseContent
	err = json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &resp)
	if err != nil {
		err = fmt.Errorf("解析大模型返回消息的json结构失败: %v, 返回结构：%+v", err, chatResp)
		z.Error(err.Error())
		return
	}

	respContent = &resp

	return
}

func (c *ChatModel) Tokenizer(ctx context.Context, texts []string) (tokenResp TokenizerResponse, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码

	reqBody := TokenizerRequest{
		Model: c.Model,
		Text:  texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil || forceErr == "Tokenizer-json.Marshal" {
		err = fmt.Errorf("构造请求体JSON失败: %v", err)
		z.Error(err.Error())
		return
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 设置请求方法和URL
	req.Header.SetMethod("POST")
	req.SetRequestURI(c.TokenizerEndPoint)

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.SetBody(jsonData)

	// 创建客户端并发送请求
	client := &fasthttp.Client{}
	// 使用DoTimeout来实现超时控制
	timeoutDuration := 60 * time.Second
	fastErr := client.DoTimeout(req, resp, timeoutDuration)
	if fastErr != nil {
		err = fmt.Errorf("发送请求失败: %v", fastErr)
		z.Error(err.Error())
		return
	}

	// 检查响应状态码
	if resp.StatusCode() >= 400 {
		err = fmt.Errorf("server returned non-2xx status: %d, content: %s", resp.StatusCode(), resp.String())
		z.Error(err.Error())
		return
	}

	// 读取响应体
	bodyText := resp.Body()

	err = json.Unmarshal(bodyText, &tokenResp)
	if err != nil || forceErr == "Tokenizer-json.Unmarshal" {
		err = fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
		z.Error(err.Error())
		return
	}

	return
}

func (c *ChatModel) GenerateChatPrompt(question *QuestionDetails) (prompt string) {
	scoreTmpl := "<question_data>\n**题目总分：**(score)分\n\n"
	standardAnswerTmpl := "- **题目标准答案：**\n    (standard_answer)\n\n </question_data>\n"
	markRoleTmpl := "\n<marking_rules> - **批改规则**：\n   (markRole)\n\n\n </marking_rules>\n"

	sysPrompt := c.Prompt + strings.Replace(scoreTmpl, "(score)", fmt.Sprintf("%f", question.Score), 1) + strings.Replace(standardAnswerTmpl, "(standard_answer)", question.Answer, 1) + strings.Replace(markRoleTmpl, "(markRole)", question.Rule, 1)
	return sysPrompt
}
