package ai_mark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"time"
)

var chatModel *ChatModel

// TODO 使用上下文缓存

func initChatModel() error {
	if chatModel != nil {
		return nil
	}

	apiKey := viper.GetString("chatModel.apiKey")
	model := viper.GetString("chatModel.model")
	endPoint := viper.GetString("chatModel.endPoint")
	tokenizerEndPoint := viper.GetString("chatModel.tokenizer.endPoint")

	maxConcurrency := 50
	if viper.IsSet("chatModel.maxConcurrency") {
		maxConcurrency = viper.GetInt("chatModel.maxConcurrency")
	}
	if maxConcurrency <= 0 {
		maxConcurrency = 50
	}

	maxTokens := 16
	if viper.IsSet("chatModel.maxTokens") {
		maxTokens = viper.GetInt("chatModel.maxTokens")
	}
	if maxTokens <= 0 {
		maxTokens = 16
	}

	if apiKey == "" || model == "" || endPoint == "" || tokenizerEndPoint == "" {
		return fmt.Errorf("从配置文件获取chatModel.model, chatModel.apiKey, chatModel.endPoint, chatModel.tokenizer.endPoint 配置项失败")
	}

	chatModel = &ChatModel{
		model:             model,
		endPoint:          endPoint,
		apiKey:            apiKey,
		tokenizerEndPoint: tokenizerEndPoint,
		maxConcurrency:    maxConcurrency,
		maxTokens:         maxTokens * 1024,                    // 该模型最大可以是 32 k
		ResponseFormat:    ResponseFormat{Type: "json_object"}, // 意味着，ai 回复 json，需要在提示词包含 json 关键字
	}
	return nil
}

func GetChatModel() (*ChatModel, error) {
	if chatModel == nil {
		if err := initChatModel(); err != nil {
			return &ChatModel{}, fmt.Errorf("初始化大模型失败: %v", err)
		}
	}
	return chatModel, nil
}

func (c *ChatModel) GetChatMaxConcurrency() int {
	return c.maxConcurrency
}

func (c *ChatModel) AIMark(ctx context.Context, questions []*QuestionDetail, studentAnswers []*StudentAnswer, sendChatCompletions func(context.Context, []Message) (ChatResponse, error)) (ResponseContent, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	if forceErr == "AIMark" {
		return ResponseContent{}, ForceErr
	}

	if len(questions) == 0 {
		z.Info("所需问题为空，没有题目可以批改")
		return ResponseContent{}, nil
	}

	if len(studentAnswers) == 0 {
		z.Info("学生答案为空，没有学生可以批改")
		return ResponseContent{}, nil
	}

	questionsJson, err := json.Marshal(questions)
	if forceErr == "json.Marshal-questions" {
		err = ForceErr
	}
	if err != nil {
		return ResponseContent{}, fmt.Errorf("构造 questions json 失败: %v", err)
	}

	studentAnswersJson, err := json.Marshal(studentAnswers)
	if forceErr == "json.Marshal-studentAnswers" {
		err = ForceErr
	}
	if err != nil {
		return ResponseContent{}, fmt.Errorf("构造 student answers json 失败: %v", err)
	}

	messages := []Message{
		{

			Role:     "system", // 系统角色，告诉模型一些初始指令
			MContent: []MessageContent{{Content: fmt.Sprintf("%s\n%s\n", markPrompt, string(questionsJson)), Type: "text"}},
			//Content: sysPrompt + "\n",
		},
		{
			Role:     "user", // 用户角色，表示用户发送的消息
			MContent: []MessageContent{{Content: string(studentAnswersJson), Type: "text"}},
			//Content: studentAnswerJson,
		},
	}

	start := time.Now()

	chatResp, err := sendChatCompletions(ctx, messages)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("发送 http 请求失败: %v", err)
	}

	elapsed := time.Since(start)
	z.Sugar().Infof("%s 本次花费时间: %s", c.model, elapsed)

	if chatResp.Error.Code != "" || chatResp.Error.Message != "" {
		return ResponseContent{}, fmt.Errorf("大模型服务端出错：%+v", chatResp.Error)
	}

	var resp ResponseContent

	if len(chatResp.Choices) == 0 {
		return ResponseContent{}, errors.New("大模型没有回复任何结果")
	}

	err = json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &resp)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("解析大模型返回消息的 json 结构 失败: %v, 内容：%v", err, chatResp.Choices[0].Message.Content)
	}

	z.Sugar().Infof("AIMark 回复: %+v", resp.MarkResults)

	return resp, nil
}

// 可获取发送的 token 数
//func (c *ChatModel) Tokenizer(ctx context.Context, texts []string) (tokenResp TokenizerResponse, err error) {
//	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
//
//	reqBody := TokenizerRequest{
//		model: c.model,
//		Text:  texts,
//	}
//
//	jsonData, err := json.Marshal(reqBody)
//	if err != nil || forceErr == "Tokenizer-json.Marshal" {
//		err = fmt.Errorf("构造 请求体JSON 失败: %v", err)
//		z.Error(err.Error())
//		return
//	}
//
//	req := fasthttp.AcquireRequest()
//	defer fasthttp.ReleaseRequest(req)
//
//	resp := fasthttp.AcquireResponse()
//	defer fasthttp.ReleaseResponse(resp)
//
//	// 设置请求方法和URL
//	req.Header.SetMethod("POST")
//	req.SetRequestURI(c.tokenizerEndPoint)
//
//	// 设置请求头
//	req.Header.Set("Authorization", "Bearer "+c.apiKey)
//	req.Header.Set("Content-Type", "application/json")
//
//	// 设置请求体
//	req.SetBody(jsonData)
//
//	// 创建客户端并发送请求
//	client := &fasthttp.Client{}
//	// 使用DoTimeout来实现超时控制
//	timeoutDuration := 60 * time.Second
//	fastErr := client.DoTimeout(req, resp, timeoutDuration)
//	if fastErr != nil {
//		err = fmt.Errorf("发送请求失败: %v", fastErr)
//		z.Error(err.Error())
//		return
//	}
//
//	// 检查响应状态码
//	if resp.StatusCode() >= 400 {
//		err = fmt.Errorf("server returned non-2xx status: %d, content: %s", resp.StatusCode(), resp.String())
//		z.Error(err.Error())
//		return
//	}
//
//	// 读取响应体
//	bodyText := resp.Body()
//
//	err = json.Unmarshal(bodyText, &tokenResp)
//	if err != nil || forceErr == "Tokenizer-json.Unmarshal" {
//		err = fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
//		z.Error(err.Error())
//		return
//	}
//
//	return
//}

// 发送 http 请求, 获取 ai 回复
func (c *ChatModel) SendChatCompletions(ctx context.Context, messages []Message) (ChatResponse, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "SendChatCompletions" {
		return ChatResponse{}, ForceErr
	}

	// 构建请求体
	requestBody := RequestBody{
		Model:          c.model,
		ResponseFormat: c.ResponseFormat,
		MaxTokens:      c.maxTokens,
		Messages:       messages,
		Temperature:    0,
		//TopP:           1, // 两个值一般不一起改
		//Thinking:       Think{Type: "disabled"},
	}

	jsonData, err := json.Marshal(requestBody)
	if forceErr == "json.Marshal-requestBody" {
		err = ForceErr
	}
	if err != nil {
		return ChatResponse{}, fmt.Errorf("构造请求体失败: %v", err)
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 设置请求方法和 URL
	req.Header.SetMethod("POST")
	req.SetRequestURI(c.endPoint)

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.SetBody(jsonData)

	// 创建客户端并发送请求
	client := &fasthttp.Client{}

	// 使用DoTimeout来实现超时控制 （10 min）
	timeoutDuration := 10 * 60 * time.Second

	// 正式发送请求
	fastErr := client.DoTimeout(req, resp, timeoutDuration)
	if forceErr == "client.DoTimeout" {
		fastErr = ForceErr
	}
	if fastErr != nil {
		return ChatResponse{}, fmt.Errorf("发送请求失败: %v", fastErr)
	}

	// 检查响应状态码
	if resp.StatusCode() >= 400 || forceErr == "resp.StatusCode()" {
		return ChatResponse{}, fmt.Errorf("大模型服务端返回 非2xx 响应码: %d, 响应内容: %s", resp.StatusCode(), resp.String())
	}

	// 读取响应体
	bodyText := resp.Body()

	var chatResp ChatResponse

	err = json.Unmarshal(bodyText, &chatResp)
	if forceErr == "json.Unmarshal-bodyText" {
		err = ForceErr
	}
	if err != nil {
		return ChatResponse{}, fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
	}

	return chatResp, nil
}
