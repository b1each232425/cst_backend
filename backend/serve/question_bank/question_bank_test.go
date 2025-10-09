package question_bank

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/require"
	"w2w.io/cmn"
	"w2w.io/null"
)

const (
	teacherRoleID    = int64(2003)
	studentRoleID    = int64(2008)
	resourceDomainID = int64(1999)
	superAdminRoleID = int64(2000)
)

var (
	TestUserIDs = []int64{} // 测试用户ID列表
)

// TestGetKnowledgeBankKnowledges 测试获取知识点库knowledges功能
func TestGetKnowledgeBankKnowledges(t *testing.T) {
	ctx := context.Background()

	// 测试获取不存在的题库的知识点库
	knowledges, err := getKnowledgeBankKnowledges(ctx, 99999)
	require.NoError(t, err)
	require.Equal(t, "[]", string(knowledges))

	// 测试获取存在的题库但无关联知识点库的情况
	// 这里需要先创建一个测试题库，然后测试
	// 由于这是单元测试，我们主要测试函数逻辑
}

// TestEnrichQuestionsWithAllKnowledges 测试为题目添加allKnowledges字段功能
func TestEnrichQuestionsWithAllKnowledges(t *testing.T) {
	ctx := context.Background()

	// 创建测试题目
	testQuestion := cmn.TQuestion{
		ID:         null.IntFrom(1),
		Type:       "00",
		Content:    null.StringFrom("测试题目"),
		BelongTo:   null.IntFrom(1),
		Knowledges: types.JSONText(`{"linkedKnowledges":[],"customKnowledges":["测试知识点"]}`),
	}

	questions := []cmn.TQuestion{testQuestion}

	// 测试为题目添加allKnowledges字段
	result, err := enrichQuestionsWithAllKnowledges(ctx, questions, 1)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].AllKnowledges)

	// 验证返回的结构体包含原始题目信息
	require.Equal(t, testQuestion.ID, result[0].TQuestion.ID)
	require.Equal(t, testQuestion.Type, result[0].TQuestion.Type)
	require.Equal(t, testQuestion.Content, result[0].TQuestion.Content)
}

func TestMain(m *testing.M) {
	cmn.ConfigureForTest()
	// 读取测试数据
	testDataFile := "test-user.json"
	data, err := os.ReadFile(testDataFile)
	if err != nil {
		e := fmt.Sprintf("Failed to read test data file %s: %v", testDataFile, err)
		z.Fatal(e)
	}

	var testData struct {
		Users       []cmn.TUser `json:"users"`
		UserDomains []struct {
			Account string   `json:"Account"`
			Domains []string `json:"Domains"`
		} `json:"user_domains"`
	}

	err = json.Unmarshal(data, &testData)
	if err != nil {
		e := fmt.Sprintf("Failed to unmarshal test data from %s: %v", testDataFile, err)
		z.Fatal(e)
	}
	db := cmn.GetDbConn()
	//转换并插入测试数据到数据库
	for _, userData := range testData.Users {
		err = userData.Create(cmn.GetDbConn())
		if err != nil {
			e := fmt.Sprintf("Failed to create user %v: %v", userData.ID.Int64, err)
			z.Warn(e)
		}
	}

	// 处理用户域关系数据
	pgxConn := cmn.GetPgxConn()
	for _, userDomain := range testData.UserDomains {
		// 根据Account查询用户ID
		var userID int64
		err = pgxConn.QueryRow(context.Background(), "SELECT id FROM t_user WHERE account = $1", userDomain.Account).Scan(&userID)
		if err != nil {
			e := fmt.Sprintf("Failed to find user with account %s: %v", userDomain.Account, err)
			z.Warn(e)
			continue
		}
		TestUserIDs = append(TestUserIDs, userID)

		// 为每个域创建用户域关系
		for _, domainStr := range userDomain.Domains {
			// 根据Domain字符串查询域ID
			var domainID int64
			err = pgxConn.QueryRow(context.Background(), "SELECT id FROM t_domain WHERE domain = $1", domainStr).Scan(&domainID)
			if err != nil {
				e := fmt.Sprintf("Failed to find domain with domain string %s: %v", domainStr, err)
				z.Warn(e)
				continue
			}

			// 创建用户域关系记录
			userDomainRecord := cmn.TUserDomain{
				SysUser: null.IntFrom(userID),
				Domain:  null.IntFrom(domainID),
			}

			err = userDomainRecord.Create(db)
			if err != nil {
				e := fmt.Sprintf("Failed to create user domain relation for user %d and domain %d: %v", userID, domainID, err)
				z.Warn(e)
			}
		}
	}
	m.Run()
	// 清理测试数据
	clearSqlTUserDomain := "DELETE FROM t_user_domain"
	_, err = pgxConn.Exec(context.Background(), clearSqlTUserDomain)
	if err != nil {
		e := fmt.Sprintf("Failed to clear user domain data: %v", err)
		z.Warn(e)
	}
}

// createMockContextWithBody 构造带body的context，仿照exam_mgt/exam-mgt_test.go
func createMockContextWithBody(method, path string, data any, forceError string, userID int64, userRole int64) context.Context {
	var req *http.Request
	if data != nil { // 修改：检查 data 是否为 nil，而不是空字符串
		// 1. 将 data 序列化为 JSON
		bodyBytes, err := json.Marshal(data)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal request data: %v", err)) // 如果序列化失败，直接 panic
		}

		// 2. 构造 ReqProto 结构体，Data 字段是 json.RawMessage
		body := &cmn.ReqProto{
			Data: bodyBytes, // 直接使用序列化后的 JSON
		}

		// 3. 重新序列化 ReqProto（因为 ReqProto 可能包含其他字段）
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal request data: %v", err)) // 如果再次序列化失败，直接 panic
		}

		// 4. 构造 http.Request
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		z.Sugar().Info(data)
		// 如果 data 是 nil，构造一个空的请求体
		req = httptest.NewRequest(method, path, nil)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2000),
		Domain: "cst.school^superAdmin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2002),
		Domain: "cst.school.academicAffair^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2003),
		Domain: "cst.school^teacher",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2008),
		Domain: "cst.school^student",
	})
	// 构造 ServiceCtx 和 Context
	w := httptest.NewRecorder()
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
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}
	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
	return context.WithValue(ctx, "force-error", forceError)
}

// createMockContextWithUnMarshalBody 构造带未序列化body的context，仿照exam_mgt/exam-mgt_test.go
func createMockContextWithUnMarshalBody(method, path string, data string, forceError string, userID int64, userRole int64) context.Context {
	var req *http.Request
	if data != "" {
		bodyBytes := json.RawMessage(data)
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2000),
		Domain: "cst.school^superAdmin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2002),
		Domain: "cst.school.academicAffair^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2003),
		Domain: "cst.school^teacher",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2008),
		Domain: "cst.school^student",
	})
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
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}
	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
	return context.WithValue(ctx, "force-error", forceError)
}

func cleanupTestBankQuestions(t *testing.T) {
	db := cmn.GetPgxConn()
	ctx := context.Background()
	txn, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	require.NoError(t, err)
	defer txn.Rollback(ctx)
	// 删除题目
	_, err = txn.Exec(ctx, `DELETE FROM assessuser.t_question WHERE creator = Any($1)`, TestUserIDs)
	require.NoError(t, err)
	// 删除题目
	_, err = txn.Exec(ctx, `DELETE FROM assessuser.t_question_bank WHERE creator = Any($1)`, TestUserIDs)
	require.NoError(t, err)
	err = txn.Commit(ctx)
	require.NoError(t, err)
}

func initTestQuestionBankAndQuestion(t *testing.T, userID int64, withoutQuestion bool) (int64, []int64) {
	// 提前准备好测试数据
	testBankFilePath := "test-bank.json"
	testQuestionFilePath := "test-question.json"

	bankBytes, err := os.ReadFile(testBankFilePath)
	require.NoError(t, err)
	questionBytes, err := os.ReadFile(testQuestionFilePath)
	require.NoError(t, err)

	var testBankData cmn.TQuestionBank
	var testQuestionData []cmn.TQuestion

	err = json.Unmarshal(bankBytes, &testBankData)
	require.NoError(t, err)
	err = json.Unmarshal(questionBytes, &testQuestionData)
	require.NoError(t, err)

	// 数据库连接
	db := cmn.GetDbConn()

	// 插入题库并记录映射
	testBankData.Creator = null.NewInt(userID, true)
	err = testBankData.Create(db)
	require.NoError(t, err)
	testBankID := testBankData.ID.Int64
	fmt.Printf("Created question bank with ID: %v\n", testBankID)

	// 如果不需要题目，则直接返回
	if withoutQuestion {
		return testBankID, []int64{}
	}

	// 插入该题库下的所有题目
	var questionIDs []int64
	for _, question := range testQuestionData {
		// 设置题目id归属
		question.BelongTo = null.NewInt(testBankID, true)
		question.Creator = null.NewInt(userID, true)

		// 将 Tags 序列化为 JSON
		tagsJSON, err := json.Marshal(question.Tags)
		require.NoError(t, err)

		// 直接执行 SQL 插入
		err = db.QueryRowx(`
			INSERT INTO t_question (
				type, content, options, answers, score, difficulty, tags, analysis,
				title, answer_file_path, test_file_path, input, output, example,
				repo, "order", creator, create_time, status, access_mode, belong_to
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13, $14,
				$15, $16, $17, $18, $19, $20, $21
			) RETURNING id`,
			question.Type, question.Content, question.Options, question.Answers,
			question.Score, question.Difficulty, tagsJSON, question.Analysis,
			question.Title, question.AnswerFilePath, question.TestFilePath,
			question.Input, question.Output, question.Example,
			question.Repo, question.Order, question.Creator, time.Now().UnixMilli(),
			"00", "00", question.BelongTo,
		).Scan(&question.ID)
		require.NoError(t, err)
		questionIDs = append(questionIDs, question.ID.Int64)
	}

	return testBankID, questionIDs
}

func TestQuestoinBankPostMethod(t *testing.T) {
	type CreateBankReq struct {
		Name string   `json:"name"`
		Type string   `json:"type"`
		Tags []string `json:"tags"`
	}
	cmn.ConfigureForTest()
	testUserID := TestUserIDs[0]
	test2 := []struct {
		name          string
		reqBody       any
		wantError     bool
		userID        int64
		roleID        int64 // 添加角色ID字段
		forceError    string
		expectedError string // 增加期望的错误信息字段
		validate      func(*testing.T, context.Context, *cmn.ServiceCtx, cmn.TQuestionBank)
	}{
		{
			name: "正常创建题库",
			reqBody: CreateBankReq{
				Name: "正常创建题库",
				Type: QuestionBankTypeNormal,
				Tags: []string{"go", "vue"},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "无法获取用户权限",
			reqBody: CreateBankReq{
				Name: "failed to get user role: no rows in result set",
				Type: QuestionBankTypeNormal,
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        -1, // 使用教师角色ID
			forceError:    "",
			expectedError: "failed to get user role: no rows in result set",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "io-ReadAll",
			reqBody: CreateBankReq{
				Name: "正常创建题库",
				Type: QuestionBankTypeNormal,
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "io-ReadAll",
			expectedError: "io-ReadAll",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "q.R.Body.Close()",
			reqBody: CreateBankReq{
				Name: "正常创建题库",
				Type: QuestionBankTypeNormal,
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "题库名称为空",
			reqBody: CreateBankReq{
				Name: "",
				Type: QuestionBankTypeNormal,
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "call /api/question-banks with empty question bank name",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "题库类型为空",
			reqBody: CreateBankReq{
				Name: "题库类型为空",
				Type: "",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "call /api/question-banks with empty question bank type",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "不支持的题库类型",
			reqBody: CreateBankReq{
				Name: "不支持的题库类型",
				Type: "03",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "call /api/question-banks with unsupported question bank type: 03",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "json.Unmarshal",
			reqBody: CreateBankReq{
				Name: "不支持的题库类型",
				Type: "00",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "json.Unmarshal",
			expectedError: "json.Unmarshal",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "无效用户ID",
			reqBody: CreateBankReq{
				Name: "无效用户ID",
				Type: "02",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        -1,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "invalid userID: -1",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "cmn.DML",
			reqBody: CreateBankReq{
				Name: "不支持的题库类型",
				Type: "00",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "cmn.DML",
			expectedError: "cmn.DML",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "bank.QryResult.bankID",
			reqBody: CreateBankReq{
				Name: "不支持的题库类型",
				Type: "00",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "bank.QryResult.bankID",
			expectedError: "s.qryResult should be int64, but it isn't",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
		{
			name: "cmn.MarshalJSON",
			reqBody: CreateBankReq{
				Name: "不支持的题库类型",
				Type: "00",
				Tags: []string{"go", "vue"},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "cmn.MarshalJSON",
			expectedError: "cmn.MarshalJSON",
			validate: func(t *testing.T, ctx context.Context, qPut *cmn.ServiceCtx, bank cmn.TQuestionBank) {
				require.NotNil(t, qPut)
				require.Equal(t, bank.Name.String, "正常创建题库")
				require.Equal(t, bank.Type.String, QuestionBankTypeNormal)
				var tags []string
				err := json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Equal(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, bank.Creator.Int64)
			},
		},
	}

	t.Run("buf is nil", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxPut := createMockContextWithUnMarshalBody("POST", "/question-banks", "", "", testUserID, teacherRoleID)
		qPut := cmn.GetCtxValue(ctxPut)
		questionBanks(ctxPut)
		require.Equal(t, -1, qPut.Msg.Status)
		require.Equal(t, "call /api/question-banks with empty body", qPut.Msg.Msg)
	})

	t.Run("ReqProto Unmarshal Fail", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxPut := createMockContextWithUnMarshalBody("POST", "/question-banks", "{", "", testUserID, teacherRoleID)
		qPut := cmn.GetCtxValue(ctxPut)
		questionBanks(ctxPut)
		require.Equal(t, -1, qPut.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", qPut.Msg.Msg)
	})

	for _, tt := range test2 {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			t.Cleanup(func() { cleanupTestBankQuestions(t) })
			ctxPut := createMockContextWithBody("POST", "/question-banks", tt.reqBody, tt.forceError, tt.userID, tt.roleID)
			qPut := cmn.GetCtxValue(ctxPut)
			questionBanks(ctxPut)
			if tt.wantError {
				if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, qPut.Msg.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
				if tt.validate != nil {
					var bank cmn.TQuestionBank
					err := json.Unmarshal(qPut.Msg.Data, &bank)
					require.NoError(t, err)
					tt.validate(t, ctx, qPut, bank)
				}
			}
		})
	}
}

func TestQuestoinBankPutMethod(t *testing.T) {
	type UpdateBankReq struct {
		ID   int64    `json:"id"`
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	cmn.ConfigureForTest()
	testUserID := TestUserIDs[0]
	now := cmn.GetNowInMS()
	test2 := []struct {
		name          string
		reqBodyFn     func(int64) any // 改成函数，可以设置ID
		wantError     bool
		userID        int64
		roleID        int64 // 添加角色ID字段
		forceError    string
		expectedError string // 增加期望的错误信息字段
		setup         func(*testing.T, context.Context) int64
		validate      func(*testing.T, context.Context, int64)
	}{
		{
			name: "正常更新题库",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "正常更新题库",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "只更新题库名称",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "只更新题库名称",
				}
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "只更新题库名称", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"数据结构", "算法"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "只更新题库标签",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "test-综合题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "不传任何更新参数",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID: id,
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "call /api/question-banks with empty question bank name and tags",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
			},
		},
		{
			name: "ID在数据库不存在",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   99999999,
					Name: "更新题库失败：没有记录被更新",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "更新题库失败：没有记录被更新",
			expectedError: "更新题库失败：没有记录被更新",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "题库ID小于等于0",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID: 0,
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "",
			expectedError: "call /api/question-banks with invalid question bank ID: 0",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
			},
		},
		{
			name: "io.ReadAll",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "io.ReadAll",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "io.ReadAll",
			expectedError: "io.ReadAll",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "q.R.Body.Close()",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "io.ReadAll",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "json.Marshal",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "json.Marshal",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "json.Marshal",
			expectedError: "json.Marshal",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
		{
			name: "conn.Exec",
			reqBodyFn: func(id int64) any {
				return UpdateBankReq{
					ID:   id,
					Name: "conn.Exec",
					Tags: []string{"go", "vue"},
				}
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID, // 使用教师角色ID
			forceError:    "conn.Exec",
			expectedError: "conn.Exec",
			setup: func(t *testing.T, ctx context.Context) int64 {
				t.Helper()
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, true) // 无需创建题目
				return bankID
			},
			validate: func(t *testing.T, ctx context.Context, bankID int64) {
				//数据查询题库
				db := cmn.GetPgxConn()
				var name string
				var tagsBytes []byte
				var updateTime int64
				var updatedBy int64

				err := db.QueryRow(ctx, `
					SELECT name, tags, update_time,updated_by
					FROM t_question_bank
					WHERE id = $1
				`, bankID).Scan(&name, &tagsBytes, &updateTime, &updatedBy)

				require.NoError(t, err)
				require.Equal(t, "正常更新题库", name)

				var tags []string
				err = json.Unmarshal(tagsBytes, &tags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"go", "vue"}, tags)
				require.Equal(t, testUserID, updatedBy, "更新者应该是当前用户")
				//更新时间应该大于now
				require.Greater(t, updateTime, now, "更新时间应该被设置")
			},
		},
	}

	t.Run("buf is nil", func(t *testing.T) {

		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxPut := createMockContextWithUnMarshalBody("PUT", "/question-banks", "", "", testUserID, teacherRoleID)
		qPut := cmn.GetCtxValue(ctxPut)
		questionBanks(ctxPut)
		require.Equal(t, -1, qPut.Msg.Status)
		require.Equal(t, "call /api/question-banks with empty body", qPut.Msg.Msg)
	})

	t.Run("ReqProto Unmarshal Fail", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxPut := createMockContextWithUnMarshalBody("PUT", "/question-banks", "{", "", testUserID, teacherRoleID)
		qPut := cmn.GetCtxValue(ctxPut)
		questionBanks(ctxPut)
		require.Equal(t, -1, qPut.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", qPut.Msg.Msg)
	})

	for _, tt := range test2 {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// 先设置好题库ID
			bankID := tt.setup(t, ctx)

			// 使用reqBodyFn生成请求体
			var reqBody any
			if tt.reqBodyFn != nil {
				reqBody = tt.reqBodyFn(bankID)
			}

			t.Cleanup(func() { cleanupTestBankQuestions(t) })
			ctxPut := createMockContextWithBody("PUT", "/question-banks", reqBody, tt.forceError, tt.userID, tt.roleID)
			qPut := cmn.GetCtxValue(ctxPut)
			questionBanks(ctxPut)
			if tt.wantError {
				if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, qPut.Msg.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
				if tt.validate != nil {
					tt.validate(t, ctx, bankID)
				}
			}
		})
	}
}

func TestQuestionBankGetMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	userID := TestUserIDs[0]

	// 清理测试环境
	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	// 额外创建几个简单题库用于分页和过滤测试
	insertTestBanks := func(t *testing.T, creatorID int64, count int) []int64 {
		var bankIDs []int64
		ctx := context.Background()
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		// 构建基础题库数据
		for i := 0; i < count; i++ {
			// 使用不同的标签组合
			tags := []string{fmt.Sprintf("tag%d", i+1)}
			if i%2 == 0 {
				tags = append(tags, "golang")
			}
			if i%3 == 0 {
				tags = append(tags, "vue")
			}

			tagsJSON, err := json.Marshal(tags)
			require.NoError(t, err)

			now := time.Now().UnixMilli()
			var bankID int64
			err = tx.QueryRow(ctx, `
				INSERT INTO t_question_bank
				(name, type, tags, creator, create_time, updated_by, update_time, status, domain_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				RETURNING id
			`, fmt.Sprintf("测试题库-%d", i+1), "00", tagsJSON, creatorID, now, creatorID, now, "00", resourceDomainID).Scan(&bankID)
			require.NoError(t, err)
			bankIDs = append(bankIDs, bankID)
		}

		err = tx.Commit(ctx)
		require.NoError(t, err)
		return bankIDs
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		wantError     bool
		userID        int64
		roleID        int64 // 用户角色ID
		forceError    string
		expectedError string
		setup         func(t *testing.T) []int64
		validate      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64)
	}{
		{
			name:          "无条件获取全部题库",
			query:         "/question-banks",
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(banks), 6) // 应包含预先创建的带题目题库 + 5个额外题库

				// 验证返回的题库中包含了我们创建的题库
				idMap := make(map[int64]bool)
				bankMap := make(map[int64]cmn.TVQuestionBank)
				for _, bank := range banks {
					idMap[bank.ID.Int64] = true
					bankMap[bank.ID.Int64] = bank
				}

				// 检查初始创建的带有题目的题库
				require.True(t, idMap[bankID], "返回结果应包含带题目的题库ID: %d", bankID)
				if detailBank, ok := bankMap[bankID]; ok {
					require.Equal(t, "test-综合题库", detailBank.Name.String, "题库名称应匹配")
					require.NotZero(t, detailBank.QuestionCount.Int64, "题目数量应大于0")
					require.Equal(t, userID, detailBank.Creator.Int64, "创建者应匹配")

					// 测试有题目的题库应该有题目类型、难度和标签
					var tags []string
					err := json.Unmarshal(detailBank.Tags, &tags)
					require.NoError(t, err)
					require.Contains(t, tags, "数据结构", "题库应有预期标签")
					require.Contains(t, tags, "算法", "题库应有预期标签")
				}

				// 检查其他创建的题库
				for _, id := range bankIDs {
					require.True(t, idMap[id], "返回结果应包含创建的题库ID: %d", id)
				}
			},
		},
		{
			name:          "用户没有可访问的域",
			query:         "/question-banks",
			expectedCount: 5,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "EmptyDomain",
			expectedError: "用户没有可访问的域",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "按关键词过滤题库名称",
			query:         "/question-banks?keyword=测试题库-3",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(banks), 1)

				// 验证返回的题库名称包含关键词
				for _, bank := range banks {
					require.Contains(t, bank.Name.String, "测试题库-3")
				}
			},
		},
		{
			name:          "按标签过滤",
			query:         "/question-banks?keyword=vue",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5) // 创建5个题库，其中序号被3整除的带有vue标签
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)

				// 检查包含vue标签的题库(0号和3号)
				vueCount := 0
				for _, bank := range banks {
					var tags []string
					err := json.Unmarshal(bank.Tags, &tags)
					require.NoError(t, err)

					for _, tag := range tags {
						if tag == "vue" {
							vueCount++
							break
						}
					}
				}
				require.GreaterOrEqual(t, vueCount, 2)
			},
		},
		{
			name:          "分页查询-第1页",
			query:         "/question-banks?page=1&pageSize=2",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)
				require.LessOrEqual(t, len(banks), 2, "应返回不超过2个题库")
			},
		},
		{
			name:          "分页查询-第2页",
			query:         "/question-banks?page=2&pageSize=2",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 5)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)
				require.LessOrEqual(t, len(banks), 2, "应返回不超过2个题库")

				// 确保第2页与第1页不同
				ctxPage1 := createMockContextWithBody("GET", "/question-banks?page=1&pageSize=2", nil, "", userID, teacherRoleID)
				qPage1 := cmn.GetCtxValue(ctxPage1)
				questionBanks(ctxPage1)
				var banksPage1 []cmn.TVQuestionBank
				err = json.Unmarshal(qPage1.Msg.Data, &banksPage1)
				require.NoError(t, err)

				// 检查两页返回的ID不重复
				page1IDs := make(map[int64]bool)
				for _, bank := range banksPage1 {
					page1IDs[bank.ID.Int64] = true
				}

				for _, bank := range banks {
					require.False(t, page1IDs[bank.ID.Int64], "第2页不应包含第1页的题库")
				}
			},
		},
		{
			name:          "按ID查询特定题库",
			query:         "/question-banks?bankID=%d", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, 0, q.Msg.Status)
				var banks []cmn.TVQuestionBank
				err := json.Unmarshal(q.Msg.Data, &banks)
				require.NoError(t, err)
				require.Equal(t, 1, len(banks), "应只返回1个题库")
				require.Equal(t, bankID, banks[0].ID.Int64, "返回的题库ID应匹配")

				// 详细检查返回的题库信息
				bank := banks[0]
				require.Equal(t, "test-综合题库", bank.Name.String, "题库名称应匹配")
				require.Equal(t, "00", bank.Type.String, "题库类型应匹配")
				require.Equal(t, userID, bank.Creator.Int64, "创建者应匹配")
				require.Equal(t, int64(len(questionIDs)), bank.QuestionCount.Int64, "题目数量应匹配")

				var tags []string
				err = json.Unmarshal(bank.Tags, &tags)
				require.NoError(t, err)
				require.Contains(t, tags, "数据结构", "应包含预期标签")
				require.Contains(t, tags, "算法", "应包含预期标签")
			},
		},
		//{
		//	name:          "非教师用户无法查询题库",
		//	query:         "/question-banks",
		//	expectedCount: 0,
		//	wantError:     true,
		//	userID:        userID,
		//	roleID:        studentRoleID, // 使用学生角色
		//	forceError:    "",
		//	expectedError: "domain cst.school^student is not allowed",
		//	setup: func(t *testing.T) []int64 {
		//		return insertTestBanks(t, userID, 3)
		//	},
		//	validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
		//		require.Equal(t, -1, q.Msg.Status)
		//		require.Contains(t, q.Msg.Msg, "not allowed")
		//	},
		//},
		{
			name:          "page不是数字",
			query:         "/question-banks?page=abc&pageSize=2",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID, // 使用学生角色
			forceError:    "",
			expectedError: "error parsing page",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 3)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, -1, q.Msg.Status)
				require.Contains(t, q.Msg.Msg, "not allowed")
			},
		},
		{
			name:          "pageSize不是数字",
			query:         "/question-banks?page=1&pageSize=abc",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID, // 使用学生角色
			forceError:    "",
			expectedError: "error parsing pageSize",
			setup: func(t *testing.T) []int64 {
				return insertTestBanks(t, userID, 3)
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
				require.Equal(t, -1, q.Msg.Status)
				require.Contains(t, q.Msg.Msg, "not allowed")
			},
		},
		{
			name:          "题库ID不是整数",
			query:         "/question-banks?bankID=abc", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "error parsing bankID",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "题库ID小于0",
			query:         "/question-banks?bankID=0", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid bankID: 0",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "conn.QueryRow",
			query:         "/question-banks", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "conn.QueryRow",
			expectedError: "conn.QueryRow",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "conn.Query",
			query:         "/question-banks", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "conn.Query",
			expectedError: "conn.Query",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "rows.Scan",
			query:         "/question-banks", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "rows.Scan",
			expectedError: "rows.Scan",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "rows.Err()",
			query:         "/question-banks", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "rows.Err()",
			expectedError: "rows.Err()",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
		{
			name:          "json.Marshal",
			query:         "/question-banks", // 将在运行时填充预先创建的带题目的题库ID
			expectedCount: 1,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "json.Marshal",
			expectedError: "json.Marshal",
			setup: func(t *testing.T) []int64 {
				// 直接使用预先创建的题库ID
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, bankIDs []int64) {
			},
		},
	}

	// 执行测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试环境
			bankIDs := tt.setup(t)

			// 根据是否需要特定ID来构造查询字符串
			query := tt.query
			if strings.Contains(query, "%d") && len(bankIDs) > 0 {
				query = fmt.Sprintf(query, bankIDs[0])
			}

			// 创建测试上下文
			ctx := createMockContextWithBody("GET", query, nil, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctx)

			// 调用被测函数
			questionBanks(ctx)

			// 验证结果
			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status)
				if tt.expectedError != "" {
					require.Contains(t, q.Msg.Msg, tt.expectedError)
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "应返回成功状态码")
				require.Equal(t, "success", q.Msg.Msg, "应返回成功信息")

				// 执行测试用例特定的验证
				if tt.validate != nil {
					tt.validate(t, ctx, q, bankIDs)
				}
			}
		})
	}
}

// 检查多个题库是否存在
func getQuestionBanksByIDs(ctx context.Context, bankIDs []int64) ([]cmn.TQuestionBank, error) {
	conn := cmn.GetPgxConn()

	query := `SELECT id, name, tags, type, creator FROM t_question_bank WHERE id = ANY($1)`
	rows, err := conn.Query(ctx, query, bankIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []cmn.TQuestionBank
	for rows.Next() {
		var bank cmn.TQuestionBank
		err := rows.Scan(&bank.ID, &bank.Name, &bank.Tags, &bank.Type, &bank.Creator)
		if err != nil {
			return nil, err
		}
		banks = append(banks, bank)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return banks, nil
}

// 检查题库中的题目是否存在
func getQuestionsByBankID(ctx context.Context, bankID int64) ([]cmn.TQuestion, error) {
	conn := cmn.GetPgxConn()

	query := `SELECT id, content, type FROM t_question WHERE belong_to = $1`
	rows, err := conn.Query(ctx, query, bankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []cmn.TQuestion
	for rows.Next() {
		var q cmn.TQuestion
		err := rows.Scan(&q.ID, &q.Content, &q.Type)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}

func TestQuestionBankDeleteMethod(t *testing.T) {
	cmn.ConfigureForTest()
	testUserID := TestUserIDs[0]
	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		roleID        int64 // 添加角色ID字段
		forceError    string
		expectedError string // 增加期望的错误信息字段
		setup         func(*testing.T, context.Context) []int64
		validate      func(*testing.T, context.Context, []int64)
	}{
		{
			name:          "正常删除单个题库",
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个测试题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotEmpty(t, questionIDs, "应该成功创建题目")
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 验证题库是否已删除
				banks, err := getQuestionBanksByIDs(ctx, bankIDs)
				require.NoError(t, err, "查询应该成功执行")
				require.Empty(t, banks, "题库列表应该为空")

				// 验证题库中的题目是否也被级联删除
				questions, err := getQuestionsByBankID(ctx, bankIDs[0])
				require.NoError(t, err, "查询应该成功执行")
				require.Empty(t, questions, "题库中的题目应该已被级联删除")
			},
		},
		{
			name:          "批量删除多个题库",
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建多个测试题库
				bankID1, questionIDs1 := initTestQuestionBankAndQuestion(t, testUserID, false)
				bankID2, questionIDs2 := initTestQuestionBankAndQuestion(t, testUserID, false)
				bankID3, questionIDs3 := initTestQuestionBankAndQuestion(t, testUserID, false)

				// 确保每个题库都有题目
				require.NotEmpty(t, questionIDs1, "第一个题库应该成功创建题目")
				require.NotEmpty(t, questionIDs2, "第二个题库应该成功创建题目")
				require.NotEmpty(t, questionIDs3, "第三个题库应该成功创建题目")

				return []int64{bankID1, bankID2, bankID3}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 验证所有题库是否已删除
				banks, err := getQuestionBanksByIDs(ctx, bankIDs)
				require.NoError(t, err, "查询应该成功执行")
				require.Empty(t, banks, "所有题库应该已被删除")

				// 验证所有题目都被级联删除
				for _, bankID := range bankIDs {
					questions, err := getQuestionsByBankID(ctx, bankID)
					require.NoError(t, err, "查询应该成功执行")
					require.Empty(t, questions, fmt.Sprintf("题库 %d 中的题目应该已被级联删除", bankID))
				}
			},
		},
		{
			name:          "非题库创建者无法删除",
			wantError:     true,
			userID:        TestUserIDs[1], // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题库",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 验证题库未被删除
				banks, err := getQuestionBanksByIDs(ctx, bankIDs)
				require.NoError(t, err, "题库应该仍然存在")
				require.Len(t, banks, 1, "题库列表应该包含一个题库")
				require.Equal(t, testUserID, banks[0].Creator.Int64)
			},
		},
		{
			name:          "题库不存在",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题库",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{999999, 888888} // 使用不存在的ID
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 不需要验证，因为题库本来就不存在
			},
		},
		{
			name:          "部分题库非创建者无法删除",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题库",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建两个题库，一个是当前用户创建的，另一个是另一用户创建的
				bankID1, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				bankID2, _ := initTestQuestionBankAndQuestion(t, TestUserIDs[1], false)
				return []int64{bankID1, bankID2}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 验证两个题库都未被删除（批量删除是事务性的，一个失败全部回滚）
				banks, err := getQuestionBanksByIDs(ctx, bankIDs)
				require.NoError(t, err, "查询应该成功执行")
				require.Len(t, banks, 2, "两个题库都应该仍然存在")
			},
		},
		//{
		//	name:          "非教师角色无法删除题库",
		//	wantError:     true,
		//	userID:        testUserID,
		//	roleID:        studentRoleID, // 使用学生角色
		//	forceError:    "",
		//	expectedError: "domain cst.school^student is not allowed",
		//	setup: func(t *testing.T, ctx context.Context) []int64 {
		//		bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
		//		return []int64{bankID}
		//	},
		//	validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
		//		// 验证题库未被删除
		//		banks, err := getQuestionBanksByIDs(ctx, bankIDs)
		//		require.NoError(t, err, "题库应该仍然存在")
		//		require.Len(t, banks, 1, "题库列表应该包含一个题库")
		//		require.Equal(t, testUserID, banks[0].Creator.Int64)
		//	},
		//},
		{
			name:          "io.ReadAll",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "io.ReadAll",
			expectedError: "io.ReadAll",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "q.R.Body.Close()",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "json.Unmarshal",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "json.Unmarshal",
			expectedError: "json.Unmarshal",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "conn.BeginTx",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "conn.BeginTx",
			expectedError: "conn.BeginTx",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "conn.BeginTx",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "conn.BeginTx",
			expectedError: "conn.BeginTx",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "recover",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "recover",
			expectedError: "recover",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "tx.Rollback",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Rollback",
			expectedError: "tx.Rollback",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{999999} // 使用不存在的ID
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
				// 不需要验证，因为题库本来就不存在
			},
		},
		{
			name:          "tx.Exec",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "tx.Exec",
			expectedError: "tx.Exec",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 使用testUserID创建题库，但用TestUserIDs[1]尝试删除
				bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "tx.Commit",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Commit",
			expectedError: "tx.Commit",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个测试题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotEmpty(t, questionIDs, "应该成功创建题目")
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "tx.QueryRow",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.QueryRow",
			expectedError: "tx.QueryRow",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个测试题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotEmpty(t, questionIDs, "应该成功创建题目")
				return []int64{bankID}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "空请求体",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "ID List cannot be empty",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "题库ID为负数",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "ID must be greater than 0: -1",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{-1}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "题库ID为0",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "ID must be greater than 0: 0",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{0}
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
		{
			name:          "题库ID数组重复",
			wantError:     true,
			userID:        testUserID, // 使用不同的用户ID
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "Duplicate ID found: 1",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{1, 1} // 重复的ID
			},
			validate: func(t *testing.T, ctx context.Context, bankIDs []int64) {
			},
		},
	}

	t.Run("空请求体", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxDelete := createMockContextWithUnMarshalBody("DELETE", "/question-banks", "", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questionBanks(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "call /api/question-banks with empty body", q.Msg.Msg)
	})

	t.Run("不支持方法", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxDelete := createMockContextWithUnMarshalBody("PATCH", "/question-banks", "", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questionBanks(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Contains(t, q.Msg.Msg, "unsupported method")
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		t.Cleanup(func() { cleanupTestBankQuestions(t) })
		ctxDelete := createMockContextWithUnMarshalBody("DELETE", "/question-banks", "{", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questionBanks(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", q.Msg.Msg)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// 先设置好题库ID
			bankIDs := tt.setup(t, ctx)

			t.Cleanup(func() { cleanupTestBankQuestions(t) })
			ctxDelete := createMockContextWithBody("DELETE", "/question-banks", bankIDs, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctxDelete)
			questionBanks(ctxDelete)

			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status, "期望错误状态码为-1")
				if !strings.Contains(q.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, q.Msg.Msg)
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "期望成功状态码为0")
				require.Equal(t, "success", q.Msg.Msg, "期望成功消息为'success'")

				// 执行测试用例特定的验证
				if tt.validate != nil {
					tt.validate(t, ctx, bankIDs)
				}
			}
		})
	}
}

func TestQuestionGetMethod(t *testing.T) {
	cmn.ConfigureForTest()
	userID := TestUserIDs[0]

	// 清理测试环境
	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	tests := []struct {
		name          string
		query         string
		expectedCount int
		wantError     bool
		userID        int64
		roleID        int64 // 用户角色ID
		forceError    string
		expectedError string
		validate      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx)
	}{
		{
			name:          "正常获取题库的题目列表",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.NotEmpty(t, questions, "题目列表不应为空")

				// 验证题目属于指定题库
				for _, question := range questions {
					require.Equal(t, bankID, question.BelongTo.Int64, "题目应属于指定题库")
					require.NotEmpty(t, question.Content.String, "题目内容不应为空")
					require.True(t, question.Score.Float64 > 0, "题目分数应大于0")
				}
			},
		},
		{
			name:          "根据题目ID查询单个题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&questionID=" + fmt.Sprintf("%d", questionIDs[0]),
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var question cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &question)
				require.NoError(t, err, "应能正确解析单个题目")
				require.Equal(t, questionIDs[0], question.ID.Int64, "题目ID应匹配")
				require.Equal(t, bankID, question.BelongTo.Int64, "题目应属于指定题库")
				require.NotEmpty(t, question.Content.String, "题目内容不应为空")
				require.True(t, question.Score.Float64 > 0, "题目分数应大于0")
				require.Equal(t, int64(1), q.Msg.RowCount, "应返回1条记录")
			},
		},
		{
			name:          "查询不存在的题目ID",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&questionID=99999999",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题目不存在或已删除",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "查询其他题库的题目ID",
			query:         "/questions?bankID=99999999&questionID=" + fmt.Sprintf("%d", questionIDs[0]),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题目不存在或已删除",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "无效的题目ID参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&questionID=invalid_id",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "按内容过滤题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&content=网络",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都包含关键词
				for _, question := range questions {
					require.Contains(t, question.Content.String, "网络", "题目内容应包含关键词")
				}
			},
		},
		{
			name:          "按题目类型过滤",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=00",
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是指定类型
				for _, question := range questions {
					require.Equal(t, "00", question.Type, "题目类型应匹配")
				}
			},
		},
		{
			name:          "按填空题类型过滤",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=06",
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是填空题
				for _, question := range questions {
					require.Equal(t, "06", question.Type, "题目类型应为填空题")
					// 填空题通常具有特定的内容特征，比如包含括号
					require.NotEmpty(t, question.Content.String, "填空题内容不应为空")
				}
			},
		},
		{
			name:          "按简答题类型过滤",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=08",
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是简答题
				for _, question := range questions {
					require.Equal(t, "08", question.Type, "题目类型应为简答题")
					// 简答题通常分数较高
					require.True(t, question.Score.Float64 >= 3, "简答题分数应较高")
				}
			},
		},
		{
			name:          "按难度过滤题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&difficulty=2",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是指定难度
				for _, question := range questions {
					require.Equal(t, "02", question.Difficulty, "题目难度应匹配")
				}
			},
		},
		{
			name:          "分页查询题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&page=1&pageSize=3",
			expectedCount: 3,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.LessOrEqual(t, len(questions), 3, "分页查询应返回不超过3个题目")
			},
		},
		{
			name:          "题库ID为空",
			query:         "/questions",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "bankID is empty",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "题库ID无效",
			query:         "/questions?bankID=abc",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid bankID",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		//{
		//	name:          "非教师角色无法查询题目",
		//	query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
		//	expectedCount: 0,
		//	wantError:     true,
		//	userID:        userID,
		//	roleID:        studentRoleID, // 使用学生角色
		//	forceError:    "",
		//	expectedError: "domain cst.school^student is not allowed",
		//	validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
		//		// 不需要验证，因为应该返回错误
		//	},
		//},
		{
			name:          "按单个标签过滤题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&tags=数据结构",
			expectedCount: 3,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都包含指定标签
				for _, question := range questions {
					var tags []string
					err := json.Unmarshal(question.Tags, &tags)
					require.NoError(t, err, "应能正确解析题目标签")
					require.Contains(t, tags, "数据结构", "题目标签应包含'数据结构'")
				}
			},
		},
		{
			name:          "按多个标签过滤题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&tags=数据结构&tags=算法",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都包含指定标签中的至少一个
				for _, question := range questions {
					var tags []string
					err := json.Unmarshal(question.Tags, &tags)
					require.NoError(t, err, "应能正确解析题目标签")

					hasTargetTag := false
					for _, tag := range tags {
						if tag == "数据结构" || tag == "算法" {
							hasTargetTag = true
							break
						}
					}
					require.True(t, hasTargetTag, "题目应包含'数据结构'或'算法'标签")
				}
			},
		},
		{
			name:          "按多个题目类型过滤",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=00&type=02",
			expectedCount: 10,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是指定类型之一
				for _, question := range questions {
					require.True(t, question.Type == "00" || question.Type == "02",
						"题目类型应为单选题(00)或多选题(02)")
				}
			},
		},
		{
			name:          "按多个难度过滤题目",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&difficulty=1&difficulty=3",
			expectedCount: 4,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都是指定难度之一
				for _, question := range questions {
					require.True(t, question.Difficulty == "00" || question.Difficulty == "04",
						"题目难度应为易(00)或中(04)")
				}
			},
		},
		{
			name:          "多条件组合过滤-类型+难度",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=00&difficulty=2",
			expectedCount: 3,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都满足组合条件
				for _, question := range questions {
					require.Equal(t, "00", question.Type, "题目类型应为单选题")
					require.Equal(t, "02", question.Difficulty, "题目难度应为较易")
				}
			},
		},
		{
			name:          "多条件组合过滤-类型+标签+难度",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=00&tags=数据结构&difficulty=1&difficulty=2",
			expectedCount: 2,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都满足组合条件
				for _, question := range questions {
					require.Equal(t, "00", question.Type, "题目类型应为单选题")
					require.True(t, question.Difficulty == "00" || question.Difficulty == "02",
						"题目难度应为易或较易")

					var tags []string
					err := json.Unmarshal(question.Tags, &tags)
					require.NoError(t, err, "应能正确解析题目标签")
					require.Contains(t, tags, "数据结构", "题目应包含'数据结构'标签")
				}
			},
		},
		{
			name:          "多条件组合过滤-内容+类型+标签",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&content=数据&type=00&type=02&tags=数据结构",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都满足组合条件
				for _, question := range questions {
					require.Contains(t, question.Content.String, "数据", "题目内容应包含'数据'")
					require.True(t, question.Type == "00" || question.Type == "02",
						"题目类型应为单选题或多选题")

					var tags []string
					err := json.Unmarshal(question.Tags, &tags)
					require.NoError(t, err, "应能正确解析题目标签")
					require.Contains(t, tags, "数据结构", "题目应包含'数据结构'标签")
				}
			},
		},
		{
			name:          "不支持的题目类型",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=99",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid type: 99",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "混合支持和不支持的题目类型",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=00&type=99",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid type: 99",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "无效难度值会被忽略",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&difficulty=1&difficulty=99",
			expectedCount: 7, // 只返回难度为1的题目
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目难度都为00（无效难度值被忽略）
				for _, question := range questions {
					require.Equal(t, "00", question.Difficulty, "题目难度应为易(00)")
				}
			},
		},
		{
			name:          "按不存在的标签过滤",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&tags=不存在的标签",
			expectedCount: 0,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.Empty(t, questions, "不存在的标签应返回空结果")
			},
		},
		{
			name:          "复杂多条件过滤-所有参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&content=网络&type=00&tags=计算机网络&difficulty=2&page=1&pageSize=5",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")

				// 验证所有返回的题目都满足复杂组合条件
				for _, question := range questions {
					require.Contains(t, question.Content.String, "网络", "题目内容应包含'网络'")
					require.Equal(t, "00", question.Type, "题目类型应为单选题")
					require.Equal(t, "02", question.Difficulty, "题目难度应为较易")

					var tags []string
					err := json.Unmarshal(question.Tags, &tags)
					require.NoError(t, err, "应能正确解析题目标签")
					require.Contains(t, tags, "计算机网络", "题目应包含'计算机网络'标签")
				}
				require.LessOrEqual(t, len(questions), 5, "分页应限制结果数量")
			},
		},
		{
			name:          "空标签参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&tags=",
			expectedCount: 25, // 应返回所有题目
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.NotEmpty(t, questions, "空标签参数应返回所有题目")
			},
		},
		{
			name:          "空类型参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&type=",
			expectedCount: 25, // 应返回所有题目
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.NotEmpty(t, questions, "空类型参数应返回所有题目")
			},
		},
		{
			name:          "只有无效难度值",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&difficulty=99&difficulty=100",
			expectedCount: 25, // 无效难度被忽略，应返回所有题目
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析题目列表")
				require.NotEmpty(t, questions, "无效难度值被忽略，应返回所有题目")
			},
		},
		// 错误覆盖测试用例 - 用于提高代码覆盖率
		{
			name:          "强制总数查询错误-questions.conn.QueryRow",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.conn.QueryRow",
			expectedError: "questions.conn.QueryRow",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制数据查询错误-questions.conn.Query",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.conn.Query",
			expectedError: "questions.conn.Query",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制行扫描错误-questions.rows.Scan",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.rows.Scan",
			expectedError: "questions.rows.Scan",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制行错误检查-questions.rows.Err()",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.rows.Err()",
			expectedError: "questions.rows.Err()",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制JSON序列化错误-questions.json.Marshal",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.json.Marshal",
			expectedError: "questions.json.Marshal",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制单个题目查询错误-questions.single.QueryRow",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&questionID=" + fmt.Sprintf("%d", questionIDs[0]),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.single.QueryRow",
			expectedError: "questions.single.QueryRow",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "强制单个题目JSON序列化错误-questions.single.json.Marshal",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&questionID=" + fmt.Sprintf("%d", questionIDs[0]),
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "questions.single.json.Marshal",
			expectedError: "questions.single.json.Marshal",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		// 参数解析错误覆盖测试用例
		{
			name:          "无效的bankID参数",
			query:         "/questions?bankID=invalid_bank_id",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid bankID",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "无效的page参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&page=invalid_page",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid page",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "无效的pageSize参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&pageSize=invalid_pagesize",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid pageSize",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name:          "无效的difficulty参数",
			query:         "/questions?bankID=" + fmt.Sprintf("%d", bankID) + "&difficulty=invalid_difficulty",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid difficulty",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
	}

	// 执行测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 根据是否需要特定ID来构造查询字符串
			query := tt.query

			// 创建测试上下文
			ctx := createMockContextWithBody("GET", query, nil, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctx)

			// 调用被测函数
			questions(ctx)

			// 验证结果
			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status)
				if tt.expectedError != "" {
					require.Contains(t, q.Msg.Msg, tt.expectedError)
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "应返回成功状态码")
				require.Equal(t, "success", q.Msg.Msg, "应返回成功信息")

				// 执行测试用例特定的验证
				if tt.validate != nil {
					tt.validate(t, ctx, q)
				}
			}
		})
	}
}

func TestQuestionPostMethod(t *testing.T) {
	cmn.ConfigureForTest()
	testUserID := TestUserIDs[0]

	// 先创建一个测试题库用于添加题目
	t.Cleanup(func() { cleanupTestBankQuestions(t) })
	bankID, _ := initTestQuestionBankAndQuestion(t, testUserID, false)

	tests := []struct {
		name          string
		reqBody       any
		wantError     bool
		userID        int64
		roleID        int64 // 添加角色ID字段
		forceError    string
		expectedError string // 增加期望的错误信息字段
		validate      func(*testing.T, context.Context, *cmn.ServiceCtx)
	}{
		{
			name: "正常创建单选题",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Len(t, questions, 1, "应返回1个题目")

				question := questions[0]
				require.NotZero(t, question.ID.Int64, "题目ID应不为0")
				require.Equal(t, "00", question.Type, "题目类型应匹配")
				require.Equal(t, bankID, question.BelongTo.Int64, "题目应属于指定题库")
				require.Equal(t, testUserID, question.Creator.Int64, "创建者应匹配")
			},
		},
		{
			name: "conn.BeginTx",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "conn.BeginTx",
			expectedError: "conn.BeginTx",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "tx.Rollback",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Rollback",
			expectedError: "tx.Rollback",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "tx.Commit",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Commit",
			expectedError: "tx.Commit",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "tx.Rollback.panic",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Rollback.panic",
			expectedError: "tx.Rollback.panic",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "br.QueryRow",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "br.QueryRow",
			expectedError: "br.QueryRow",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "br.Close",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "br.Close",
			expectedError: "br.Close",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "正常创建多选题",
			reqBody: []cmn.TQuestion{
				{
					Type:       "02",
					Difficulty: "02",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道多选题测试</span></p>"),
					Tags:       types.JSONText(`["测试", "多选"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A", "C"]`),
					Score:                   null.FloatFrom(3),
					Analysis:                null.StringFrom("<p>多选题解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Equal(t, "02", questions[0].Type, "题目类型应为多选题")
			},
		},
		{
			name: "正常创建判断题",
			reqBody: []cmn.TQuestion{
				{
					Type:       "04",
					Difficulty: "00",
					Content:    null.StringFrom("<p>这是一道判断题</p>"),
					Tags:       types.JSONText(`["判断"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "正确" },
						{ "label": "B", "value": "错误" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(1),
					Analysis:                null.StringFrom("<p>判断题解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Equal(t, "04", questions[0].Type, "题目类型应为判断题")
			},
		},
		{
			name: "批量创建题目",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p>批量题目1</p>"),
					Tags:       types.JSONText(`["批量"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "选项A" },
						{ "label": "B", "value": "选项B" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>解析1</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
				{
					Type:       "00",
					Difficulty: "02",
					Content:    null.StringFrom("<p>批量题目2</p>"),
					Tags:       types.JSONText(`["批量"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "选项A" },
						{ "label": "B", "value": "选项B" }
					]`),
					Answers:                 types.JSONText(`["B"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>解析2</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Len(t, questions, 2, "应返回2个题目")

				for _, question := range questions {
					require.NotZero(t, question.ID.Int64, "每个题目ID应不为0")
					require.Equal(t, bankID, question.BelongTo.Int64, "题目应属于指定题库")
				}
			},
		},
		{
			name: "题目类型无效",
			reqBody: []cmn.TQuestion{
				{
					Type:       "99", // 无效类型
					Difficulty: "00",
					Content:    null.StringFrom("<p>无效类型题目</p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options:    types.JSONText(`[]`),
					Answers:    types.JSONText(`["A"]`),
					Score:      null.FloatFrom(2),
					BelongTo:   null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "unsupported question type",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "题目分数无效",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p>分数无效题目</p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "选项A" },
						{ "label": "B", "value": "选项B" }
					]`),
					Answers:  types.JSONText(`["A"]`),
					Score:    null.FloatFrom(0), // 无效分数
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question score must be greater than zero",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "选择题选项不足",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p>选项不足题目</p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "只有一个选项" }
					]`), // 选择题至少需要2个选项
					Answers:  types.JSONText(`["A"]`),
					Score:    null.FloatFrom(2),
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "single choice question must have at least 2 options",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "填空题缺少必需字段",
			reqBody: []cmn.TQuestion{
				{
					Type:       "06",
					Difficulty: "00",
					Content:    null.StringFrom(`<p>这是一道填空题，<span class="blank-item">____</span>是答案</p>`),
					Tags:       types.JSONText(`["测试"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "",
							"score": 2,
							"grading_rule": ""
						}
					]`), // 缺少必需的字段值
					Score:    null.FloatFrom(2),
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "fill-in-blank question answer content cannot be empty for index 1",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "简答题分数为零",
			reqBody: []cmn.TQuestion{
				{
					Type:       "08",
					Difficulty: "00",
					Content:    null.StringFrom("<p>这是一道简答题</p>"),
					Tags:       types.JSONText(`["测试"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "答案内容",
							"score": 0,
							"grading_rule": "keyword_match"
						}
					]`), // 分数为0
					Score:    null.FloatFrom(5),
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "essay question answer score must be greater than 0, got: 0.000000",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "题目内容为空",
			reqBody: []cmn.TQuestion{
				{
					Type:       "06",
					Difficulty: "00",
					Content:    null.StringFrom(""), // 空内容
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{
							"index": 1,
							"answer": "答案",
							"score": 2,
							"grading_rule": "exact_match"
						}
					]`),
					Answers:  types.JSONText(`["答案"]`),
					Score:    null.FloatFrom(2),
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question content cannot be empty",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "题目内容只有空格",
			reqBody: []cmn.TQuestion{
				{
					Type:       "08",
					Difficulty: "00",
					Content:    null.StringFrom("   "), // 只有空格
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{
							"index": 1,
							"answer": "答案内容",
							"score": 5,
							"grading_rule": "keyword_match"
						}
					]`),
					Answers:  types.JSONText(`["答案内容"]`),
					Score:    null.FloatFrom(5),
					BelongTo: null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question content cannot be empty",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 不需要验证，因为应该返回错误
			},
		},
		{
			name: "正常创建填空题",
			reqBody: []cmn.TQuestion{
				{
					Type:       "06",
					Difficulty: "02",
					Content:    null.StringFrom(`<p>请填写Go语言的关键字：<span class="blank-item">____</span>是用来声明变量的关键字，<span class="blank-item">____</span>是用来声明常量的关键字。</p>`),
					Tags:       types.JSONText(`["填空", "Go语言"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "var",
							"alternative_answer": ["VAR"],
							"score": 2,
							"grading_rule": "exact_match"
						},
						{
							"index": 2,
							"answer": "const",
							"alternative_answer": ["CONST"],
							"score": 2,
							"grading_rule": "exact_match"
						}
					]`), // 填空题使用SubjectiveAnswer格式
					Score:                   null.FloatFrom(4),
					Analysis:                null.StringFrom("<p>var用于声明变量，const用于声明常量</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Equal(t, "06", questions[0].Type, "题目类型应为填空题")

				// 验证填空题的答案格式
				var answers []SubjectiveAnswer
				err = json.Unmarshal(questions[0].Answers, &answers)
				require.NoError(t, err, "应能正确解析填空题答案")
				require.Len(t, answers, 2, "填空题应有2个答案")
				require.Equal(t, "var", answers[0].Answer, "第一个答案应为var")
				require.Equal(t, "const", answers[1].Answer, "第二个答案应为const")
			},
		},
		{
			name: "正常创建简答题",
			reqBody: []cmn.TQuestion{
				{
					Type:       "08",
					Difficulty: "04",
					Content:    null.StringFrom("<p>请简述Go语言中goroutine的工作原理以及它与传统线程的区别。</p>"),
					Tags:       types.JSONText(`["简答", "Go语言", "并发"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "goroutine是Go语言中的轻量级线程，由Go运行时管理。与传统操作系统线程相比，goroutine具有更小的内存占用、更快的创建和销毁速度、以及由Go调度器管理的协作式调度特性。",
							"alternative_answer": [],
							"score": 10,
							"grading_rule": "keyword_match"
						}
					]`), // 简答题使用SubjectiveAnswer格式
					Score:                   null.FloatFrom(10),
					Analysis:                null.StringFrom("<p>考查对Go语言并发机制的理解，重点是goroutine的特点和优势</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Equal(t, "08", questions[0].Type, "题目类型应为简答题")
				require.Equal(t, "04", questions[0].Difficulty, "简答题难度应为中")
				require.Equal(t, float64(10), questions[0].Score.Float64, "简答题分数应为10分")
			},
		},
		{
			name: "批量创建不同类型题目",
			reqBody: []cmn.TQuestion{
				{
					Type:       "06", // 填空题
					Difficulty: "00",
					Content:    null.StringFrom(`<p>Go语言的包管理工具是<span class="blank-item">____</span></p>`),
					Tags:       types.JSONText(`["填空", "基础"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "go mod",
							"alternative_answer": ["go modules"],
							"score": 2,
							"grading_rule": "exact_match"
						}
					]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>go mod是Go语言的官方包管理工具</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
				{
					Type:       "08", // 简答题
					Difficulty: "02",
					Content:    null.StringFrom("<p>请解释Go语言中interface{}的作用</p>"),
					Tags:       types.JSONText(`["简答", "接口"]`),
					Answers: types.JSONText(`[
						{
							"index": 1,
							"answer": "interface{}是Go语言中的空接口，可以接受任何类型的值，常用于需要处理未知类型数据的场景。",
							"alternative_answer": [],
							"score": 5,
							"grading_rule": "keyword_match"
						}
					]`),
					Score:                   null.FloatFrom(5),
					Analysis:                null.StringFrom("<p>考查对Go语言接口机制的理解</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				var questions []cmn.TQuestion
				err := json.Unmarshal(q.Msg.Data, &questions)
				require.NoError(t, err, "应能正确解析返回的题目")
				require.Len(t, questions, 2, "应返回2个题目")

				// 验证填空题
				fillQuestion := questions[0]
				require.Equal(t, "06", fillQuestion.Type, "第一个题目应为填空题")
				require.Equal(t, bankID, fillQuestion.BelongTo.Int64, "题目应属于指定题库")

				// 验证简答题
				essayQuestion := questions[1]
				require.Equal(t, "08", essayQuestion.Type, "第二个题目应为简答题")
				require.Equal(t, bankID, essayQuestion.BelongTo.Int64, "题目应属于指定题库")
			},
		},
		//{
		//	name: "非教师角色无法创建题目",
		//	reqBody: []cmn.TQuestion{
		//		{
		//			Type:       "00",
		//			Difficulty: null.IntFrom(1),
		//			Content:    null.StringFrom("<p>学生创建题目</p>"),
		//			Tags:       types.JSONText(`["测试"]`),
		//			Options: types.JSONText(`[
		//				{ "label": "A", "value": "选项A" },
		//				{ "label": "B", "value": "选项B" }
		//			]`),
		//			Answers:  types.JSONText(`["A"]`),
		//			Score:    null.FloatFrom(2),
		//			BelongTo: null.IntFrom(bankID),
		//		},
		//	},
		//	wantError:     true,
		//	userID:        testUserID,
		//	roleID:        studentRoleID, // 使用学生角色
		//	forceError:    "",
		//	expectedError: "domain cst.school^student is not allowed",
		//	validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
		//		// 不需要验证，因为应该返回错误
		//	},
		//},
		{
			name: "用户ID无效",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        -1,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "无效的用户ID: -1",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "io.ReadAll",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "io.ReadAll",
			expectedError: "io.ReadAll",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "q.R.Body.Close()",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "json.Unmarshal",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "json.Unmarshal",
			expectedError: "json.Unmarshal",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "cmn.InvalidEmptyNullValue",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "cmn.InvalidEmptyNullValue",
			expectedError: "cmn.InvalidEmptyNullValue",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "cmn.MarshalJSON",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "cmn.MarshalJSON",
			expectedError: "cmn.MarshalJSON",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "json.Marshal",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "json.Marshal",
			expectedError: "json.Marshal",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "json.Marshal",
			reqBody: []cmn.TQuestion{
				{
					Type:       "00",
					Difficulty: "00",
					Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是一道单选题测试</span></p>"),
					Tags:       types.JSONText(`["测试"]`),
					Options: types.JSONText(`[
						{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">选项A</span></p>" },
						{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">选项B</span></p>" },
						{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">选项C</span></p>" },
						{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">选项D</span></p>" }
					]`),
					Answers:                 types.JSONText(`["A"]`),
					Score:                   null.FloatFrom(2),
					Analysis:                null.StringFrom("<p>这是解析</p>"),
					QuestionAttachmentsPath: types.JSONText(`[]`),
					BelongTo:                null.IntFrom(bankID),
				},
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "json.Marshal",
			expectedError: "json.Marshal",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
	}

	t.Run("空请求体", func(t *testing.T) {
		ctxPost := createMockContextWithUnMarshalBody("POST", "/questions", "", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxPost)
		questions(ctxPost)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "call /api/questions with empty body", q.Msg.Msg)
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		ctxPost := createMockContextWithUnMarshalBody("POST", "/questions", "{", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxPost)
		questions(ctxPost)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", q.Msg.Msg)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctxPost := createMockContextWithBody("POST", "/questions", tt.reqBody, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctxPost)
			questions(ctxPost)

			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status, "期望错误状态码为-1")
				if tt.expectedError != "" {
					require.Contains(t, q.Msg.Msg, tt.expectedError, "错误信息应包含期望内容")
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "期望成功状态码为0")
				require.Equal(t, "success", q.Msg.Msg, "期望成功消息为'success'")

				// 执行测试用例特定的验证
				if tt.validate != nil {
					tt.validate(t, ctx, q)
				}
			}
		})
	}
}

// TestQuestionPutMethod 测试题目更新方法 (PUT方法)
func TestQuestionPutMethod(t *testing.T) {
	cmn.ConfigureForTest()
	ctx := context.Background()
	testUserID := TestUserIDs[0]

	// 清理测试环境
	t.Cleanup(func() {
		cleanupTestBankQuestions(t)
	})

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	// 使用第一个题目进行测试
	testQuestionID := questionIDs[0]

	tests := []struct {
		name          string
		reqBody       any
		wantError     bool
		userID        int64
		roleID        int64
		forceError    string
		expectedError string
		validate      func(*testing.T, context.Context, *cmn.ServiceCtx)
	}{
		{
			name: "正常更新题目",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00", // 单选题
				Difficulty: "02",
				Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是更新后的单选题</span></p>"),
				Tags:       types.JSONText(`["更新测试", "单选题"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">更新选项A</span></p>" },
					{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">更新选项B</span></p>" },
					{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">更新选项C</span></p>" },
					{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">更新选项D</span></p>" }
				]`),
				Answers:                 types.JSONText(`["B"]`),
				Score:                   null.FloatFrom(3),
				Analysis:                null.StringFrom("<p>这是更新后的解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
				// 验证数据库中的题目已被更新
				db := cmn.GetPgxConn()
				var content, options, answers, analysis string
				var score float64
				var difficulty int64
				var tags []byte

				err := db.QueryRow(ctx, `
					SELECT content, options, answers, score, difficulty, tags, analysis
					FROM t_question
					WHERE id = $1
				`, testQuestionID).Scan(&content, &options, &answers, &score, &difficulty, &tags, &analysis)

				require.NoError(t, err)
				require.Contains(t, content, "这是更新后的单选题")
				require.Contains(t, options, "更新选项A")
				require.Equal(t, `["B"]`, answers)
				require.Equal(t, float64(3), score)
				require.Equal(t, int64(2), difficulty)
				require.Contains(t, analysis, "这是更新后的解析")

				var questionTags []string
				err = json.Unmarshal(tags, &questionTags)
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"更新测试", "单选题"}, questionTags)
			},
		},
		{
			name: "释放锁失败",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00", // 单选题
				Difficulty: "02",
				Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是更新后的单选题</span></p>"),
				Tags:       types.JSONText(`["更新测试", "单选题"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">更新选项A</span></p>" },
					{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">更新选项B</span></p>" },
					{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">更新选项C</span></p>" },
					{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">更新选项D</span></p>" }
				]`),
				Answers:                 types.JSONText(`["B"]`),
				Score:                   null.FloatFrom(3),
				Analysis:                null.StringFrom("<p>这是更新后的解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "cmn.ReleaseLock",
			expectedError: "lock not held by current client",
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {
			},
		},
		{
			name: "更新不存在的题目",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(999999), // 不存在的题目ID
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>不存在的题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "no rows updated",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "题目内容为空",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom(""), // 空内容
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question content cannot be empty",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "无效的题目类型",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "99", // 无效类型
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "unsupported question type: 99",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "无效的难度等级",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "99", // 无效难度
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "unsupported question difficulty",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "分数小于等于0",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(0), // 分数为0
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question score must be greater than zero",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "所属题库ID无效",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(0), // 无效的题库ID
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "question belongTo must be greater than zero",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "io.ReadAll错误",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "io.ReadAll",
			expectedError: "io.ReadAll",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "q.R.Body.Close()错误",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "json.Unmarshal错误(ReqProto)",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "reqproto-json.Unmarshal",
			expectedError: "reqproto-json.Unmarshal",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "json.Unmarshal错误(cmn.TQuestion)",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "cmn.TQuestion-json.Unmarshal",
			expectedError: "cmn.TQuestion-json.Unmarshal",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
		{
			name: "conn.Exec错误",
			reqBody: cmn.TQuestion{
				ID:         null.IntFrom(testQuestionID),
				Type:       "00",
				Difficulty: "00",
				Content:    null.StringFrom("<p>测试题目</p>"),
				Tags:       types.JSONText(`["测试"]`),
				Options: types.JSONText(`[
					{ "label": "A", "value": "<p>选项A</p>" },
					{ "label": "B", "value": "<p>选项B</p>" }
				]`),
				Answers:                 types.JSONText(`["A"]`),
				Score:                   null.FloatFrom(2),
				Analysis:                null.StringFrom("<p>解析</p>"),
				QuestionAttachmentsPath: types.JSONText(`[]`),
				BelongTo:                null.IntFrom(bankID),
			},
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "conn.Exec",
			expectedError: "conn.Exec",
			validate:      func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx) {},
		},
	}

	t.Run("空请求体", func(t *testing.T) {
		ctxPut := createMockContextWithUnMarshalBody("PUT", "/questions", "", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxPut)
		questions(ctxPut)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "call /api/questions with empty body", q.Msg.Msg)
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		reqBody := cmn.TQuestion{
			ID:         null.IntFrom(testQuestionID),
			Type:       "00", // 单选题
			Difficulty: "02",
			Content:    null.StringFrom("<p><span style=\"font-size: 12pt\">这是更新后的单选题</span></p>"),
			Tags:       types.JSONText(`["更新测试", "单选题"]`),
			Options: types.JSONText(`[
					{ "label": "A", "value": "<p><span style=\"font-size: 12pt\">更新选项A</span></p>" },
					{ "label": "B", "value": "<p><span style=\"font-size: 12pt\">更新选项B</span></p>" },
					{ "label": "C", "value": "<p><span style=\"font-size: 12pt\">更新选项C</span></p>" },
					{ "label": "D", "value": "<p><span style=\"font-size: 12pt\">更新选项D</span></p>" }
				]`),
			Answers:                 types.JSONText(`["B"]`),
			Score:                   null.FloatFrom(3),
			Analysis:                null.StringFrom("<p>这是更新后的解析</p>"),
			QuestionAttachmentsPath: types.JSONText(`[]`),
			BelongTo:                null.IntFrom(bankID),
		}
		ctxPut := createMockContextWithBody("PUT", "/questions", reqBody, "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxPut)
		_, err := cmn.TryLock(ctxPut, reqBody.ID.Int64, TestUserIDs[1], QuestionLockPrefix, QuestionLockExpiration)
		require.NoError(t, err)
		defer func() {
			err = cmn.ReleaseLock(ctxPut, reqBody.ID.Int64, TestUserIDs[1], QuestionLockPrefix)
			require.NoError(t, err)
		}()
		questions(ctxPut)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, fmt.Sprintf("key question_lock:%d is locked", reqBody.ID.Int64), q.Msg.Msg)
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		ctxPut := createMockContextWithUnMarshalBody("PUT", "/questions", "{", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxPut)
		questions(ctxPut)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", q.Msg.Msg)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxPut := createMockContextWithBody("PUT", "/questions", tt.reqBody, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctxPut)
			questions(ctxPut)

			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status, "期望错误状态码为-1")
				if tt.expectedError != "" {
					require.Contains(t, q.Msg.Msg, tt.expectedError, "错误信息应包含期望内容")
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "期望成功状态码为0")
				require.Equal(t, "success", q.Msg.Msg, "期望成功消息为'success'")

				// 执行测试用例特定的验证
				if tt.validate != nil {
					tt.validate(t, ctx, q)
				}
			}
		})
	}
}

// TestQuestionDeleteMethod 测试题目删除方法 (DELETE方法)
func TestQuestionDeleteMethod(t *testing.T) {
	cmn.ConfigureForTest()
	ctx := context.Background()
	testUserID := TestUserIDs[0]

	// 清理测试环境
	t.Cleanup(func() {
		cleanupTestBankQuestions(t)
	})

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		roleID        int64
		forceError    string
		expectedError string
		setup         func(*testing.T, context.Context) []int64  // 返回要删除的题目ID列表
		validate      func(*testing.T, context.Context, []int64) // 验证删除结果
	}{
		{
			name:          "正常删除单个题目",
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 验证题目是否已删除
				questions, err := getQuestionsByIDs(ctx, questionIDs)
				require.NoError(t, err, "查询应该成功执行")
				require.Empty(t, questions, "题目应该已被删除")
			},
		},
		{
			name:          "批量删除多个题目",
			wantError:     false,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建多个题目
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false) // 创建多个题目
				require.NotZero(t, bankID, "题库ID不应为0")
				require.GreaterOrEqual(t, len(questionIDs), 3, "应创建至少3个题目")

				// 返回前3个题目用于删除测试
				return questionIDs[:3]
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 验证所有题目是否已删除
				questions, err := getQuestionsByIDs(ctx, questionIDs)
				require.NoError(t, err, "查询应该成功执行")
				require.Empty(t, questions, "所有题目应该已被删除")
			},
		},
		{
			name:          "删除不存在的题目",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题目",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 返回一个不存在的题目ID
				return []int64{999999}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 无需验证，因为删除操作应该失败
			},
		},
		{
			name:          "删除混合存在和不存在的题目",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "题目",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个题目，然后混合存在和不存在的ID
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0], 999999, 888888} // 混合存在和不存在的ID
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 验证存在的题目仍然存在（删除操作应该失败）
				questions, err := getQuestionsByIDs(ctx, []int64{questionIDs[0]})
				require.NoError(t, err, "查询应该成功执行")
				require.NotEmpty(t, questions, "存在的题目应该仍然存在")
			},
		},
		{
			name:          "空的题目ID数组",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "ID List cannot be empty",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{} // 空数组
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 无需验证
			},
		},
		{
			name:          "包含无效ID的数组",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "ID must be greater than 0: 0",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				return []int64{0, -1} // 无效ID
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
				// 无需验证
			},
		},
		{
			name:          "io.ReadAll",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "io.ReadAll",
			expectedError: "io.ReadAll",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "q.R.Body.Close()",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "q.R.Body.Close()",
			expectedError: "q.R.Body.Close()",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "json.Unmarshal",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "json.Unmarshal",
			expectedError: "json.Unmarshal",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "recover",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "recover",
			expectedError: "recover",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "tx.Rollback",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Rollback",
			expectedError: "tx.Rollback",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "tx.Commit",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Commit",
			expectedError: "tx.Commit",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "conn.BeginTx",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "conn.BeginTx",
			expectedError: "conn.BeginTx",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "tx.QueryRow",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.QueryRow",
			expectedError: "tx.QueryRow",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
		{
			name:          "tx.Exec",
			wantError:     true,
			userID:        testUserID,
			roleID:        teacherRoleID,
			forceError:    "tx.Exec",
			expectedError: "tx.Exec",
			setup: func(t *testing.T, ctx context.Context) []int64 {
				// 创建一个包含题目的题库
				bankID, questionIDs := initTestQuestionBankAndQuestion(t, testUserID, false)
				require.NotZero(t, bankID, "题库ID不应为0")
				require.NotEmpty(t, questionIDs, "应创建至少一个题目")
				return []int64{questionIDs[0]}
			},
			validate: func(t *testing.T, ctx context.Context, questionIDs []int64) {
			},
		},
	}

	// 测试边界情况
	t.Run("用户ID无效", func(t *testing.T) {
		ctxDelete := createMockContextWithBody("DELETE", "/questions", []int64{1}, "", 0, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questions(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Contains(t, q.Msg.Msg, "无效的用户ID: 0")
	})

	t.Run("空请求体", func(t *testing.T) {
		ctxDelete := createMockContextWithUnMarshalBody("DELETE", "/questions", "", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questions(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "call /api/question-banks with empty body", q.Msg.Msg)
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		ctxDelete := createMockContextWithUnMarshalBody("DELETE", "/questions", "{", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questions(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "unexpected end of JSON input", q.Msg.Msg)
	})

	t.Run("不支持的请求方法", func(t *testing.T) {
		ctxDelete := createMockContextWithUnMarshalBody("PATCH", "/questions", "{", "", testUserID, teacherRoleID)
		q := cmn.GetCtxValue(ctxDelete)
		questions(ctxDelete)
		require.Equal(t, -1, q.Msg.Status)
		require.Equal(t, "unsupported method: patch", q.Msg.Msg)
	})

	// 运行主要测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试数据
			questionIDs := tt.setup(t, ctx)

			// 执行删除操作
			ctxDelete := createMockContextWithBody("DELETE", "/questions", questionIDs, tt.forceError, tt.userID, tt.roleID)
			q := cmn.GetCtxValue(ctxDelete)
			questions(ctxDelete)

			if tt.wantError {
				require.Equal(t, -1, q.Msg.Status, "期望错误状态码为-1")
				if tt.expectedError != "" {
					require.Contains(t, q.Msg.Msg, tt.expectedError, "错误信息应包含期望内容")
				}
			} else {
				require.Equal(t, 0, q.Msg.Status, "期望成功状态码为0")
				require.Equal(t, "success", q.Msg.Msg, "期望成功消息为'success'")
			}

			// 执行测试用例特定的验证
			if tt.validate != nil {
				tt.validate(t, ctx, questionIDs)
			}
		})
	}
}

// getQuestionsByIDs 根据题目ID列表查询题目
func getQuestionsByIDs(ctx context.Context, questionIDs []int64) ([]cmn.TQuestion, error) {
	if len(questionIDs) == 0 {
		return []cmn.TQuestion{}, nil
	}

	db := cmn.GetPgxConn()
	query := `
		SELECT id, type, content, options, answers, score, difficulty, tags,
		       analysis, title, answer_file_path, test_file_path, input, output,
		       example, repo, "order", creator, create_time, updated_by,
		       update_time, addi, status, question_attachments_path,
		       access_mode, belong_to
		FROM t_question
		WHERE id = ANY($1) AND status = '00'
	`

	rows, err := db.Query(ctx, query, questionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []cmn.TQuestion
	for rows.Next() {
		var question cmn.TQuestion
		err = rows.Scan(
			&question.ID,
			&question.Type,
			&question.Content,
			&question.Options,
			&question.Answers,
			&question.Score,
			&question.Difficulty,
			&question.Tags,
			&question.Analysis,
			&question.Title,
			&question.AnswerFilePath,
			&question.TestFilePath,
			&question.Input,
			&question.Output,
			&question.Example,
			&question.Repo,
			&question.Order,
			&question.Creator,
			&question.CreateTime,
			&question.UpdatedBy,
			&question.UpdateTime,
			&question.Addi,
			&question.Status,
			&question.QuestionAttachmentsPath,
			&question.AccessMode,
			&question.BelongTo,
		)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// TestQuestionLockGetMethod 测试获取锁的功能 (GET方法)
func TestQuestionLockGetMethod(t *testing.T) {
	cmn.ConfigureForTest()
	userID := int64(90005) // 测试用户ID

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		roleID        int64
		forceError    string
		expectedError string
		setup         func(t *testing.T) (int64, []int64)
	}{
		{
			name:          "正常获取题目锁",
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "无效题目ID",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			setup: func(t *testing.T) (int64, []int64) {
				return -1, nil
			},
		},
		{
			name:          "题目ID为0",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			setup: func(t *testing.T) (int64, []int64) {
				return 0, nil
			},
		},
		{
			name:          "无效用户ID",
			wantError:     true,
			userID:        0,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid userID",
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "强制TryLock错误",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "cmn.TryLock",
			expectedError: "cmn.TryLock",
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
	}

	// 测试无效questionID的字符串解析错误
	t.Run("ParseInt Error", func(t *testing.T) {
		ctxGet := createMockContextWithBody("GET", "/question/lock?question_id=str", "", "", userID, teacherRoleID)
		qGet := cmn.GetCtxValue(ctxGet)
		qGet.R.URL.RawQuery = "question_id=str"
		QuestionLock(ctxGet)
		if qGet.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qGet.Msg)
		}
	})

	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questionID, _ := tt.setup(t)

			ctxGet := createMockContextWithBody("GET", "/question/lock?question_id="+fmt.Sprint(questionID), "", tt.forceError, tt.userID, tt.roleID)
			qGet := cmn.GetCtxValue(ctxGet)
			qGet.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
			QuestionLock(ctxGet)
			t.Cleanup(func() {
				// 清理可能的锁
				if questionID > 0 {
					_ = cmn.ReleaseLock(ctxGet, questionID, tt.userID, QuestionLockPrefix)
				}
			})

			if tt.wantError {
				if qGet.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", qGet.Msg)
				}
				if tt.expectedError != "" && !strings.Contains(qGet.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误消息包含 %q, 实际为: %q", tt.expectedError, qGet.Msg.Msg)
				}
			} else {
				if qGet.Msg.Status != 0 || qGet.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qGet.Msg)
				}
			}
		})
	}
}

// TestQuestionLockPutMethod 测试刷新锁的功能 (PUT方法)
func TestQuestionLockPutMethod(t *testing.T) {
	cmn.ConfigureForTest()
	userID := int64(90005) // 测试用户ID

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		roleID        int64
		forceError    string
		expectedError string
		setup         func(t *testing.T) (int64, []int64)
		needLock      bool // 是否需要先获取锁
	}{
		{
			name:          "正常刷新题目锁",
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			needLock:      true,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "无效题目ID",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return -1, nil
			},
		},
		{
			name:          "题目ID为0",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return 0, nil
			},
		},
		{
			name:          "无效用户ID",
			wantError:     true,
			userID:        0,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid userID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "强制RefreshLock错误",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "cmn.RefreshLock",
			expectedError: "cmn.RefreshLock",
			needLock:      true,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
	}

	// 测试无效questionID的字符串解析错误
	t.Run("ParseInt Error", func(t *testing.T) {
		ctxPut := createMockContextWithBody("PUT", "/question/lock?question_id=str", "", "", userID, teacherRoleID)
		qPut := cmn.GetCtxValue(ctxPut)
		qPut.R.URL.RawQuery = "question_id=str"
		QuestionLock(ctxPut)
		if qPut.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qPut.Msg)
		}
	})

	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questionID, _ := tt.setup(t)
			ctxPut := createMockContextWithBody("PUT", "/question/lock?question_id="+fmt.Sprint(questionID), "", tt.forceError, tt.userID, tt.roleID)
			// 如果需要先获取锁
			if tt.needLock && questionID > 0 {
				_, _ = cmn.TryLock(ctxPut, questionID, tt.userID, QuestionLockPrefix, QuestionLockExpiration)
			}

			qPut := cmn.GetCtxValue(ctxPut)
			qPut.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
			QuestionLock(ctxPut)
			t.Cleanup(func() {
				// 清理可能的锁
				if questionID > 0 {
					_ = cmn.ReleaseLock(ctxPut, questionID, tt.userID, QuestionLockPrefix)
				}
			})
			if tt.wantError {
				if qPut.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", qPut.Msg)
				}
				if tt.expectedError != "" && !strings.Contains(qPut.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误消息包含 %q, 实际为: %q", tt.expectedError, qPut.Msg.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
			}
		})
	}
}

// TestQuestionLockDeleteMethod 测试释放锁的功能 (DELETE方法)
func TestQuestionLockDeleteMethod(t *testing.T) {
	cmn.ConfigureForTest()
	userID := int64(90005) // 测试用户ID

	// 先创建一个包含题目的题库，供所有测试用例共享
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		roleID        int64
		forceError    string
		expectedError string
		setup         func(t *testing.T) (int64, []int64)
		needLock      bool // 是否需要先获取锁
	}{
		{
			name:          "正常释放题目锁",
			wantError:     false,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "",
			needLock:      true,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "释放不存在的锁",
			wantError:     true, // 释放不存在的锁通常会报错
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "lock not held by current client",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "无效题目ID",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return -1, nil
			},
		},
		{
			name:          "题目ID为0",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid questionID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return 0, nil
			},
		},
		{
			name:          "无效用户ID",
			wantError:     true,
			userID:        0,
			roleID:        teacherRoleID,
			forceError:    "",
			expectedError: "invalid userID",
			needLock:      false,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
		{
			name:          "强制ReleaseLock错误",
			wantError:     true,
			userID:        userID,
			roleID:        teacherRoleID,
			forceError:    "cmn.ReleaseLock",
			expectedError: "cmn.ReleaseLock",
			needLock:      true,
			setup: func(t *testing.T) (int64, []int64) {
				return questionIDs[0], questionIDs
			},
		},
	}

	// 测试无效questionID的字符串解析错误
	t.Run("ParseInt Error", func(t *testing.T) {
		ctxDelete := createMockContextWithBody("DELETE", "/question/lock?question_id=str", "", "", userID, teacherRoleID)
		qDelete := cmn.GetCtxValue(ctxDelete)
		qDelete.R.URL.RawQuery = "question_id=str"
		QuestionLock(ctxDelete)
		if qDelete.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qDelete.Msg)
		}
	})

	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questionID, _ := tt.setup(t)
			ctxDelete := createMockContextWithBody("DELETE", "/question/lock?question_id="+fmt.Sprint(questionID), "", tt.forceError, tt.userID, tt.roleID)
			t.Cleanup(func() {
				// 清理可能的锁
				if questionID > 0 {
					_ = cmn.ReleaseLock(ctxDelete, questionID, tt.userID, QuestionLockPrefix)
				}
			})

			// 如果需要先获取锁
			if tt.needLock && questionID > 0 {
				_, _ = cmn.TryLock(ctxDelete, questionID, tt.userID, QuestionLockPrefix, QuestionLockExpiration)
			}

			qDelete := cmn.GetCtxValue(ctxDelete)
			qDelete.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
			QuestionLock(ctxDelete)

			if tt.wantError {
				if qDelete.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", qDelete.Msg)
				}
				if tt.expectedError != "" && !strings.Contains(qDelete.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误消息包含 %q, 实际为: %q", tt.expectedError, qDelete.Msg.Msg)
				}
			} else {
				if qDelete.Msg.Status != 0 || qDelete.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qDelete.Msg)
				}
			}
		})
	}
}

// TestQuestionLockUnsupportedMethod 测试不支持的HTTP方法
func TestQuestionLockUnsupportedMethod(t *testing.T) {
	cmn.ConfigureForTest()
	userID := int64(90005)

	// 先创建一个包含题目的题库
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	questionID := questionIDs[0]
	t.Cleanup(func() { cleanupTestBankQuestions(t) })

	ctxPost := createMockContextWithBody("POST", "/question/lock?question_id="+fmt.Sprint(questionID), "", "", userID, teacherRoleID)
	qPost := cmn.GetCtxValue(ctxPost)
	qPost.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
	QuestionLock(ctxPost)

	if qPost.Msg.Status == 0 {
		t.Errorf("期望错误, 实际无错: %+v", qPost.Msg)
	}
	if !strings.Contains(qPost.Msg.Msg, "不支持该方法") {
		t.Errorf("期望错误消息包含'不支持该方法', 实际为: %q", qPost.Msg.Msg)
	}
}

// TestQuestionLockLifecycle 测试锁的完整生命周期
func TestQuestionLockLifecycle(t *testing.T) {
	cmn.ConfigureForTest()
	userID := int64(90005)

	// 先创建一个包含题目的题库
	bankID, questionIDs := initTestQuestionBankAndQuestion(t, userID, false)
	require.NotZero(t, bankID, "题库ID不应为0")
	require.NotEmpty(t, questionIDs, "应创建至少一个题目")

	questionID := questionIDs[0]
	t.Cleanup(func() {
		cleanupTestBankQuestions(t)
	})

	// 1. 获取锁
	ctxGet := createMockContextWithBody("GET", "/question/lock?question_id="+fmt.Sprint(questionID), "", "", userID, teacherRoleID)
	qGet := cmn.GetCtxValue(ctxGet)
	qGet.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
	QuestionLock(ctxGet)

	if qGet.Msg.Status != 0 || qGet.Msg.Msg != "success" {
		t.Fatalf("获取锁失败: %+v", qGet.Msg)
	}

	// 2. 刷新锁
	ctxPut := createMockContextWithBody("PUT", "/question/lock?question_id="+fmt.Sprint(questionID), "", "", userID, teacherRoleID)
	qPut := cmn.GetCtxValue(ctxPut)
	qPut.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
	QuestionLock(ctxPut)

	if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
		t.Fatalf("刷新锁失败: %+v", qPut.Msg)
	}

	// 3. 释放锁
	ctxDelete := createMockContextWithBody("DELETE", "/question/lock?question_id="+fmt.Sprint(questionID), "", "", userID, teacherRoleID)
	qDelete := cmn.GetCtxValue(ctxDelete)
	qDelete.R.URL.RawQuery = fmt.Sprintf("question_id=%d", questionID)
	QuestionLock(ctxDelete)

	if qDelete.Msg.Status != 0 || qDelete.Msg.Msg != "success" {
		t.Fatalf("释放锁失败: %+v", qDelete.Msg)
	}
}

// TestGetQuestionBankStats 测试获取题库统计信息功能
func TestGetQuestionBankStats(t *testing.T) {
	ctx := context.Background()

	// 获取数据库连接
	conn := cmn.GetPgxConn()
	if conn == nil {
		t.Skip("数据库连接不可用，跳过测试")
		return
	}

	// 测试获取不存在的题库统计信息
	stats, err := getQuestionBankStats(ctx, conn, 99999)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalCount)
	require.Empty(t, stats.Types)

	// 测试获取存在的题库统计信息
	// 这里需要先创建一个测试题库和题目，然后测试统计功能
	// 由于这是单元测试，我们主要测试函数逻辑和错误处理
}
