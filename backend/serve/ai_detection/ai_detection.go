package ai_detection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"w2w.io/cmn"
	"w2w.io/serve/auth_mgt"
)

//annotation:ai_detection-service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

var (
	z               *zap.Logger
	AIModelInstance *AIModel
)

type ResponseFormat struct {
	Type string `json:"type"`
}

type AIModel struct {
	Model             string
	Endpoint          string
	ApiKey            string
	DetectionPrompt   string
	ReviewPrompt      string
	Limiter           *rate.Limiter
	TokenizerEndPoint string
	MaxTokens         int            `json:"max_tokens"`
	ResponseFormat    ResponseFormat `json:"response_format"`
	TimeOut           time.Duration
}

type Message struct {
	Role     string           `json:"role"`
	MContent []MessageContent `json:"content"`
}

type MessageContent struct {
	Content string `json:"text"`
	Type    string `json:"type"`
}

type RequestBody struct {
	Model          string         `json:"model"`
	ResponseFormat ResponseFormat `json:"response_format"`
	MaxTokens      int            `json:"max_tokens"`
	Messages       []Message      `json:"messages"`
	Temperature    float32        `json:"temperature"`
	TopP           float32        `json:"top_p"`
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

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("ai_detection zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("exam_mgt.Enroll called")

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
		Fn: aiDetection,

		Path: "/ai_detection",
		Name: "ai_detection",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}

func NewAIModel(model string, endpoint string, apiKey string, detectionPrompt string, tokenizerEndPoint string, maxConcurrency int, timeout time.Duration) *AIModel {
	return &AIModel{
		Model:             model,
		Endpoint:          endpoint,
		ApiKey:            apiKey,
		TokenizerEndPoint: tokenizerEndPoint,
		Limiter:           rate.NewLimiter(rate.Limit(maxConcurrency), maxConcurrency),
		MaxTokens:         16 * 1024,
		ResponseFormat:    ResponseFormat{Type: "json_object"},
		DetectionPrompt:   detectionPrompt,
		TimeOut:           timeout,
	}
}

func InitAIModel() error {
	if AIModelInstance != nil {
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

	AIModelInstance = NewAIModel(model, endPoint, apiKey, detectionPrompt, tokenizerEndPoint, maxConcurrency, 30*time.Second)
	return nil

}

func GetAIModel() (*AIModel, error) {
	if AIModelInstance == nil {
		err := InitAIModel()
		return AIModelInstance, err
	}
	return AIModelInstance, nil
}

func (a *AIModel) SendDetectionRequest(ctx context.Context, messages []Message) (response cmn.TQuestion, err error) {
	z.Info("---->" + cmn.FncName())
	forceErr, _ := ctx.Value("send-detection-request-forceErr").(string)

	if forceErr == "normal-resp" {
		return cmn.TQuestion{}, nil
	}

	if forceErr == "bad-resp" {
		return cmn.TQuestion{}, fmt.Errorf("这是一个测试错误")
	}

	// 限流
	ctx, cancel := context.WithTimeout(ctx, a.TimeOut)
	defer cancel()
	err = a.Limiter.Wait(ctx)
	if err != nil || forceErr == "Limiter.Wait" {
		reqErr := fmt.Errorf("当前使用人数过多，请稍后再试")
		z.Error(reqErr.Error() + ", " + err.Error())
		return
	}

	// 构建请求体
	requestBody := RequestBody{
		Model:          a.Model,
		ResponseFormat: a.ResponseFormat,
		MaxTokens:      a.MaxTokens,
		Messages:       messages,
		Temperature:    0.5,
		TopP:           0.5,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil || forceErr == "json.Marshal" {
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
	req.SetRequestURI(a.Endpoint)

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+a.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.SetBody(jsonData)

	// 创建客户端并发送请求
	client := &fasthttp.Client{}

	// 超时控制
	timeoutDuration := 1 * 60 * time.Second

	// 当不处于强制触发错误的情况下才发送请求，避免测试超时
	if forceErr == "" || forceErr == "DoTimeout" {
		z.Info("发送请求到大模型，endpoint: " + a.Endpoint)
		fastErr := client.DoTimeout(req, resp, timeoutDuration)
		if fastErr != nil || forceErr == "DoTimeout" {
			err = fmt.Errorf("发送请求失败: %v", fastErr)
			z.Error(err.Error())
			return
		}
	}

	// 检查响应状态码
	if resp.StatusCode() >= 400 || forceErr == "StatusCode" {
		err = fmt.Errorf("无法正常响应，响应码: %d, 内容: %s", resp.StatusCode(), resp.String())
		z.Error(err.Error())
		return
	}

	// 读取响应体
	bodyText := resp.Body()

	if forceErr != "" {
		bodyText = []byte(`{
			"error": {"code": "test_error", "message": "这是一个测试错误"},
			"choices": [{"message": {"content": "这是一个测试错误"}}]
		}`)
	}

	var chatResp ChatResponse
	err = json.Unmarshal(bodyText, &chatResp)
	if err != nil || forceErr == "json.Unmarshal" {
		err = fmt.Errorf("解析返回的响应体失败: %v, body: %s", err, string(bodyText))
		z.Error(err.Error())
		return
	}

	err = json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &response)
	if err != nil || forceErr == "json.Unmarshal2" {
		err = fmt.Errorf("解析大模型返回消息的json结构失败: %v, 内容：%v", err, chatResp.Choices[0].Message.Content)
		z.Error(err.Error())
		return
	}

	return
}

func aiDetection(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	// 用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("aiDetection-force-error"); val != nil {
		forceErr = val.(string)
	}

	var authority *auth_mgt.Authority
	authority, q.Err = auth_mgt.GetUserAuthority(ctx)
	if forceErr == "auth_mgt.GetUserAuthority" {
		q.Err = fmt.Errorf("强制获取用户权限错误")
	}
	if q.Err != nil {
		q.RespErr()
		return
	}
	_ = authority
	z.Info("用户: " + fmt.Sprintf("%d", q.SysUser.ID.Int64) + " 调用AI智能题目检测")

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if forceErr == "io.Close" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var question cmn.TQuestion
		q.Err = json.Unmarshal(qry.Data, &question)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if AIModelInstance == nil {
			AIModelInstance, q.Err = GetAIModel()
			if forceErr == "GetAIModel" {
				q.Err = fmt.Errorf("强制获取AIModel错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 准备prompt
		var questionData []byte
		questionData, q.Err = json.Marshal(question)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var messages []Message
		messages = append(messages, Message{
			Role:     "system",
			MContent: []MessageContent{{Content: AIModelInstance.DetectionPrompt, Type: "text"}},
		})
		messages = append(messages, Message{
			Role:     "user",
			MContent: []MessageContent{{Content: "请根据以下内容进行检测并返回结果: " + string(questionData), Type: "text"}},
		})

		var respQuestion cmn.TQuestion
		respQuestion, q.Err = AIModelInstance.SendDetectionRequest(ctx, messages)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var jsonData []byte
		jsonData, q.Err = json.Marshal(respQuestion)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制JSON序列化错误2")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = jsonData

	default:
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
	return

}
