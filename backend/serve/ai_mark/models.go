package ai_mark

import (
	"errors"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

type ChatModel struct {
	model             string
	endPoint          string
	apiKey            string
	maxConcurrency    int
	tokenizerEndPoint string
	maxTokens         int            // 最大思考内容Token长度为 32k
	ResponseFormat    ResponseFormat `json:"response_format"`
}

// ChatResponse
/*
 Error   struct { ... } `json:"error"`      ---- 用来承载错误信息（请求失败时返回）
 Choices []struct { ... } `json:"choices"`  ---- 模型的回复结果是一个列表（因为一次请求可以要求返回多个候选答案）
 Usage   struct { ... } `json:"usage"`      ---- Token 使用情况，用于计费或调试
 Id      string `json:"id"`                 ---- 这次请求的唯一 ID，方便日志和问题排查。
 model   string `json:"model"`              ---- 返回时确认实际用的模型名
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
	Role string `json:"role"` // 角色，表示消息来自谁。（system/user/assistant）
	//Content  string           `json:"content1"`
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

var z *zap.Logger

type forceErrKey string

const ForceErrKey = forceErrKey("force-err")

var ForceErr = errors.New("")

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}
