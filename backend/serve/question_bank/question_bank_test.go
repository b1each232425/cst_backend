package question_bank

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"
)

const testUserID = 1
const creator = 11

// createMockContextWithRole 创建带用户角色的模拟上下文
func createMockContextWithRole(method, path string, queryParams url.Values, forceError string, userID, userRole int64) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

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
			ID:   null.NewInt(userID, true),
			Role: null.NewInt(userRole, true),
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}

	return ctx
}

func createMockContextWithBody(method, path string, data string, forceError string, userID int64) context.Context {
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
			ID: null.NewInt(userID, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

func setup() {
	// 提前准备好测试数据
	testBankFilePath := "test-bank.json"
	testQuestionFilePath := "test-question.json"

	bankBytes, err := os.ReadFile(testBankFilePath)
	if err != nil {
		e := fmt.Sprintf("Failed to read test bank file %s: %v", testBankFilePath, err)
		z.Fatal(e)
	}
	questionBytes, err := os.ReadFile(testQuestionFilePath)
	if err != nil {
		e := fmt.Sprintf("Failed to read test question file %s: %v", testQuestionFilePath, err)
		z.Fatal(e)
	}

	var testBankData cmn.TQuestionBank
	var testQuestionData []cmn.TQuestion

	err = json.Unmarshal(bankBytes, &testBankData)
	if err != nil {
		e := fmt.Sprintf("Failed to unmarshal test bank data from %s: %v", testBankFilePath, err)
		z.Fatal(e)
	}
	err = json.Unmarshal(questionBytes, &testQuestionData)
	if err != nil {
		e := fmt.Sprintf("Failed to unmarshal test question data from %s: %v", testQuestionFilePath, err)
		z.Fatal(e)
	}

	// 数据库连接
	db := cmn.GetDbConn()

	// 插入题库并记录映射
	err = testBankData.Create(db)
	if err != nil {
		e := fmt.Sprintf("Failed to create question bank: %v", err)
		z.Warn(e)
	}
	bankID := testBankData.ID.Int64
	fmt.Printf("Created question bank with ID: %v\n", bankID)
	// 插入该题库下的所有题目
	for _, question := range testQuestionData {
		// 设置题目id归属
		question.BelongTo = null.NewInt(bankID, true)
		question.Creator = null.NewInt(creator, true)

		err = question.Create(db)
		if err != nil {
			e := fmt.Sprintf("Failed to create question (BelongTo: %v): %v", bankID, err)
			z.Warn(e)
		}
	}
}

func teardown() {
	// 清理测试数据
	clearSql1 := "DELETE FROM t_question_bank WHERE remark = 'test'"
	clearSql2 := `DELETE FROM t_question WHERE creator=$1`
	pgxConn := cmn.GetPgxConn()
	_, err := pgxConn.Exec(context.Background(), clearSql1)
	if err != nil {
		e := fmt.Sprintf("Failed to clear test data: %v", err)
		z.Warn(e)
	}
	_, err = pgxConn.Exec(context.Background(), clearSql2, creator)
	if err != nil {
		e := fmt.Sprintf("Failed to clear test data: %v", err)
		z.Warn(e)
	}
}

// 添加题库测试
func TestAddQuestionBank(t *testing.T) {
	z.Info("TestAddQuestionBank is running...")

}

func Test2(t *testing.T) {
	fmt.Println("I'm test2")
}

func TestMain(m *testing.M) {
	cmn.ConfigureForTest()
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
