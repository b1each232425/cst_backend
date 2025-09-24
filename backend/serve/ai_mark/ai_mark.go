package ai_mark

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"time"
	"w2w.io/cmn"
)

// TODO 使用上下文缓存

type ChatModel struct {
	Model             string
	Endpoint          string
	ApiKey            string
	MaxConcurrency    int
	TokenizerEndPoint string
	MaxTokens         int            `json:"max_tokens"` // 最大思考内容Token长度为 32k
	ResponseFormat    ResponseFormat `json:"response_format"`
}

// ChatResponse
/*
 Error   struct { ... } `json:"error"`      ---- 用来承载错误信息（请求失败时返回）
 Choices []struct { ... } `json:"choices"`  ---- 模型的回复结果是一个列表（因为一次请求可以要求返回多个候选答案）
 Usage   struct { ... } `json:"usage"`      ---- Token 使用情况，用于计费或调试
 Id      string `json:"id"`                 ---- 这次请求的唯一 ID，方便日志和问题排查。
 Model   string `json:"model"`              ---- 返回时确认实际用的模型名
*/
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
	MarkResults []struct {
		QuestionID int64   `json:"question_id"`
		Score      float64 `json:"score"`
		Analyze    string  `json:"analyze"`
	} `json:"mark_results"`
}

type RequestBody struct {
	Model          string         `json:"model"`           // 指定要使用的模型 ID
	ResponseFormat ResponseFormat `json:"response_format"` // 指定模型返回结果的格式
	MaxTokens      int            `json:"max_tokens"`      // 限制模型回答的最大 token 数量
	Messages       []Message      `json:"messages"`        // 聊天上下文，存放一组消息
	Temperature    float32        `json:"temperature"`     // 低温度可迫使模型选择“最确定的下一个词”，避免模型在字段名、值的表达上随意变化。
	TopP           float32        `json:"top_p"`           // 对于稳定输出，通常直接设 1.0 或略低即可。
	//Thinking       Think          `json:"thinking"`
}

type Message struct {
	Role     string           `json:"role"` // 角色，表示消息来自谁。（system/user/assistant）
	Content  string           `json:"content1"`
	Type     string           `json:"type"`    // 消息类型
	MContent []MessageContent `json:"content"` // 发送给模型的消息（允许多模态、多段的复杂消息）
}

type MessageContent struct {
	Content string `json:"text"` // 文字内容
	Type    string `json:"type"` // 内容类型，如"text"、"image"
}

type Think struct {
	Type string `json:"type"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

// TokenizerRequest 表示一次分词请求的参数
type TokenizerRequest struct {
	Model string   `json:"model"` // 模型 ID，比如 "doubao-seed-1-6"
	Text  []string `json:"text"`  // 要进行分词的文本，可以传多个字符串
}

// TokenizerResponse 表示分词 API 的返回结果
type TokenizerResponse struct {
	Object  string             `json:"object"`  // 返回对象的类型，一般是 "tokenizer"
	ID      string             `json:"id"`      // 请求 ID，用于唯一标识一次请求
	Model   string             `json:"model"`   // 使用的模型 ID
	Data    []TokenizationInfo `json:"data"`    // 每个输入文本对应的分词结果数组
	Created int                `json:"created"` // 创建时间（Unix 时间戳）
}

// TokenizationInfo 表示单条文本的分词信息
type TokenizationInfo struct {
	Object        string  `json:"object"`         // 返回对象的类型，一般是 "tokenization"
	Index         int     `json:"index"`          // 输入文本的索引（对应 TokenizerRequest.Text 里的顺序）
	TotalTokens   int     `json:"total_tokens"`   // 总的 token 数量
	TokenIDs      []int   `json:"token_ids"`      // 分词后的 token ID 序列（每个 token 对应词表里的索引）
	OffsetMapping [][]int `json:"offset_mapping"` // 每个 token 在原始文本里的起始和结束位置。例如 [[0, 2], [2, 5]] 表示第一个 token 是 text[0:2]，第二个 token 是 text[2:5]
}

// SubjectiveAnswer 主观题答案结构
type SubjectiveAnswer struct {
	Index             int      `json:"index"`              // 答案索引
	Answer            string   `json:"answer"`             // 答案
	AlternativeAnswer []string `json:"alternative_answer"` // 备选答案
	Score             float64  `json:"score"`              // 分数
	GradingRule       string   `json:"grading_rule"`       // 评分规则
}

type QuestionDetail struct {
	ID     int64   `json:"id"`
	Score  float64 `json:"score"`
	Answer string  `json:"answer"`
	Rule   string  `json:"rule"`
}

type StudentAnswer struct {
	QuestionID int64  `json:"question_id"`
	Answer     string `json:"answer"`
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

func initChatModel() error {
	if chatModel != nil {
		return nil
	}

	apiKey := viper.GetString("chatModel.apiKey")
	model := viper.GetString("chatModel.model")
	endPoint := viper.GetString("chatModel.endpoint")
	tokenizerEndPoint := viper.GetString("chatModel.tokenizer.endpoint")

	maxConcurrency := 50
	if viper.IsSet("chatModel.maxConcurrency") {
		maxConcurrency = viper.GetInt("chatModel.maxConcurrency")
	}

	maxTokens := 16
	if viper.IsSet("chatModel.maxTokens") {
		maxTokens = viper.GetInt("chatModel.maxTokens")
	}

	if apiKey == "" || model == "" || endPoint == "" || tokenizerEndPoint == "" {
		return fmt.Errorf("从配置文件获取chatModel.model, chatModel.apiKey, chatModel.endpoint, chatModel.tokenizer.endpoint 配置项失败")
	}

	chatModel = NewChatModel(model, endPoint, apiKey, tokenizerEndPoint, maxConcurrency, maxTokens)
	return nil
}

func NewChatModel(model string, endpoint string, apiKey string, tokenizerEndPoint string, maxConcurrency, maxTokens int) *ChatModel {
	return &ChatModel{
		Model:             model,
		Endpoint:          endpoint,
		ApiKey:            apiKey,
		TokenizerEndPoint: tokenizerEndPoint,
		MaxConcurrency:    maxConcurrency,
		MaxTokens:         maxTokens * 1024, // 该模型最大可以是 32 k
		ResponseFormat:    ResponseFormat{Type: "json_object"},
	}
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
	return c.MaxConcurrency
}

func (c *ChatModel) AIMark(ctx context.Context, questions []*QuestionDetail, studentAnswers []*StudentAnswer) (ResponseContent, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码

	if len(studentAnswers) == 0 {
		z.Info("没有学生可以批改")
		return ResponseContent{}, nil
	}

	questionsJson, err := json.Marshal(questions)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("构造 questions json 失败: %v", err)
	}

	studentAnswersJson, err := json.Marshal(studentAnswers)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("构造 student answers json 失败: %v", err)
	}

	questionsStr := string(questionsJson)
	studentAnswersStr := string(studentAnswersJson)

	messages := []Message{
		{

			Role:     "system", // 系统角色，告诉模型一些初始指令
			MContent: []MessageContent{{Content: fmt.Sprintf("%s\n%s", markPrompt, questionsStr) + "\n", Type: "text"}},
			//Content: sysPrompt + "\n",
		},
		{
			Role:     "user", // 用户角色，表示用户发送的消息
			MContent: []MessageContent{{Content: studentAnswersStr, Type: "text"}},
			//Content: studentAnswerJson,
		},
	}

	start := time.Now()

	chatResp, err := c.sendChatCompletions(ctx, messages)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("发送 http 请求失败: %v", err)
	}

	elapsed := time.Since(start)
	z.Sugar().Infof("%s 本次花费时间: %s", c.Model, elapsed)

	if chatResp.Error.Code != "" || chatResp.Error.Message != "" || forceErr == "AIMark-大模型服务端出错" {
		return ResponseContent{}, fmt.Errorf("大模型服务端出错：%+v", chatResp.Error)
	}

	var resp ResponseContent

	err = json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &resp)
	if err != nil {
		return ResponseContent{}, fmt.Errorf("解析大模型返回消息的 json 结构 失败: %v, 内容：%v", err, chatResp.Choices[0].Message.Content)
	}

	z.Sugar().Infof("AIMark 回复: %+v", resp.MarkResults)

	return resp, nil
}

// 可获取发送的 token 数
func (c *ChatModel) Tokenizer(ctx context.Context, texts []string) (tokenResp TokenizerResponse, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码

	reqBody := TokenizerRequest{
		Model: c.Model,
		Text:  texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil || forceErr == "Tokenizer-json.Marshal" {
		err = fmt.Errorf("构造 请求体JSON 失败: %v", err)
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

// 发送 http 请求
func (c *ChatModel) sendChatCompletions(ctx context.Context, messages []Message) (ChatResponse, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	// 构建请求体
	requestBody := RequestBody{
		Model:          c.Model,
		ResponseFormat: c.ResponseFormat,
		MaxTokens:      c.MaxTokens,
		Messages:       messages,
		Temperature:    0,
		//TopP:           1, // TODO 两个值一般不一起改
		//Thinking:       Think{Type: "disabled"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil || forceErr == "sendChatCompletions-json.Marshal" {
		return ChatResponse{}, fmt.Errorf("构造请求体失败: %v", err)
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 设置请求方法和 URL
	req.Header.SetMethod("POST")
	req.SetRequestURI(c.Endpoint)

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.SetBody(jsonData)

	// 创建客户端并发送请求
	client := &fasthttp.Client{}

	// 使用DoTimeout来实现超时控制 （10 min）
	timeoutDuration := 10 * 60 * time.Second

	// 正式发送请求
	fastErr := client.DoTimeout(req, resp, timeoutDuration)
	if fastErr != nil {
		return ChatResponse{}, fmt.Errorf("发送请求失败: %v", fastErr)
	}

	// 检查响应状态码
	if resp.StatusCode() >= 400 {
		return ChatResponse{}, fmt.Errorf("大模型服务端返回 非2xx 响应码: %d, 响应内容: %s", resp.StatusCode(), resp.String())
	}

	// 读取响应体
	bodyText := resp.Body()

	var chatResp ChatResponse

	err = json.Unmarshal(bodyText, &chatResp)
	if err != nil || forceErr == "sendChatCompletions-json.Unmarshal" {
		return ChatResponse{}, fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
	}

	return chatResp, nil
}
