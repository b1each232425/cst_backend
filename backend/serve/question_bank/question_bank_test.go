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

var (
	RoleAdmin               = int64(2001)
	RoleAcademicAffairAdmin = int64(2002)
	RoleTeacher             = int64(2003)
	RoleStudent             = int64(2008)
	testUserID1             = int64(11) // 用户ID1,创建了两个题库
	testUserID2             = int64(12) // 用户ID2,未创建题库
	testBankID              = int64(1001)
)

// createMockContextWithRole 创建带用户角色的模拟上下文
func createMockContextWithRole(method, path string, queryParams url.Values, forceError string, userID, userRole int64) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleAdmin),
		Domain: DomainAdmin,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleAcademicAffairAdmin),
		Domain: DomainAcademicAffairAdmin,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleTeacher),
		Domain: DomainTeacher,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleStudent),
		Domain: DomainStudent,
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
			ID:   null.NewInt(userID, true),
			Role: null.NewInt(userRole, true),
		},
		Domains: domains,
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}

	return ctx
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
		ID:     null.IntFrom(RoleAdmin),
		Domain: DomainAdmin,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleAcademicAffairAdmin),
		Domain: DomainAcademicAffairAdmin,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleTeacher),
		Domain: DomainTeacher,
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(RoleStudent),
		Domain: DomainStudent,
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
			ID:   null.NewInt(userID, true),
			Role: null.NewInt(userRole, true),
		},
		Domains: domains,
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
	testBankData.Creator = null.NewInt(testUserID1, true)
	err = testBankData.Create(db)
	if err != nil {
		e := fmt.Sprintf("Failed to create question bank: %v", err)
		z.Warn(e)
	}
	testBankID = testBankData.ID.Int64
	fmt.Printf("Created question bank with ID: %v\n", testBankID)
	// 插入该题库下的所有题目
	for _, question := range testQuestionData {
		// 设置题目id归属
		question.BelongTo = null.NewInt(testBankID, true)
		question.Creator = null.NewInt(testUserID1, true)

		err = question.Create(db)
		if err != nil {
			e := fmt.Sprintf("Failed to create question (BelongTo: %v): %v", testBankID, err)
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
	_, err = pgxConn.Exec(context.Background(), clearSql2, testUserID1)
	if err != nil {
		e := fmt.Sprintf("Failed to clear test data: %v", err)
		z.Warn(e)
	}
}

// 测试questionBanks接口
func TestQuestionBanks(t *testing.T) {}

// 查询题库测试
func TestBankGetMethod(t *testing.T) {
	z.Info("TestBankGetMethod is running...")
	testCases := []struct {
		name          string
		description   string
		query         string
		forceError    string
		expectedError bool
		expectedRow   null.Int
		userID        int64
		userRole      int64
	}{
		{
			name:          "无权限访问-学生角色",
			description:   "使用学生身份访问时,应该返回无权限的错误",
			query:         "",
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleStudent,
		},
		{
			name:          "有访问权限-教师角色1",
			description:   "使用用户1教师身份访问时,返回1条数据记录",
			query:         "",
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "有访问权限-教师角色2",
			description:   "使用用户2教师身份访问时,返回0条数据记录",
			query:         "",
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(0),
			userID:        testUserID2,
			userRole:      RoleTeacher,
		},
		{
			name:          "有访问权限-教务员角色",
			description:   "使用用户2教务员身份访问时,返回1条数据记录",
			query:         "",
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleAdmin,
		},
		{
			name:          "有访问权限-教师角色3",
			description:   "使用非法ID用户教师身份访问,结果错误",
			query:         "",
			forceError:    "",
			expectedError: true,
			userID:        -1,
			userRole:      RoleTeacher,
		},
		{
			name:          "非法的bankID",
			description:   "使用非法ID的bankID访问,结果错误",
			query:         fmt.Sprintf("bankID=%d", -1),
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "合法且存在的bankID",
			description:   "使用合法且存在的bankID访问,结果正确,返回1条数据记录",
			query:         fmt.Sprintf("bankID=%d", testBankID),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "关键词搜索-针对题库名称1",
			description:   "输入题库名称部分内容搜索,记录存在,返回1条数据记录",
			query:         fmt.Sprintf("keyword=%s", "test"),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "关键词搜索-针对题库名称2",
			description:   "输入题库名称部分内容搜索,记录不不存在,返回0条数据记录",
			query:         fmt.Sprintf("keyword=%s", "error"),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(0),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "关键词搜索-针对题库标签1",
			description:   "输入标签部分内容搜索,记录存在,返回1条数据记录",
			query:         fmt.Sprintf("keyword=%s", "算法"),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "关键词搜索-针对题库标签2",
			description:   "输入标签部分内容搜索,记录不存在,返回1条数据记录",
			query:         fmt.Sprintf("keyword=%s", "error"),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(0),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "非法的页数",
			description:   "使用非法页数访问,结果错误",
			query:         fmt.Sprintf("page=%d", -1),
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "合法的页数",
			description:   "使用合法页数访问,结果正确,返回1条数据记录",
			query:         fmt.Sprintf("page=%d", 1),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "非法的每页记录数",
			description:   "使用非法每页记录数访问,结果错误",
			query:         fmt.Sprintf("pageSize=%d", -1),
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "合法的每页记录数",
			description:   "使用合法每页记录数访问,结果正确,返回1条数据记录",
			query:         fmt.Sprintf("pageSize=%d", 10),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:          "合法的页数和每页记录数",
			description:   "使用合法页数和每页记录数访问,结果正确,返回1条数据记录",
			query:         fmt.Sprintf("page=%d&pageSize=%d", 1, 10),
			forceError:    "",
			expectedError: false,
			expectedRow:   null.IntFrom(1),
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			queryParams := url.Values{}
			ctx := createMockContextWithRole("GET", "/api/exam", queryParams, "", tc.userID, tc.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("questionBanks 意外panic: %v", r)
					}
				}()

				questionBanks(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tc.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("questionBanks测试期望返回错误，但实际成功")
					return
				}

			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("questionBanks测试期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}

			if tc.expectedRow.Valid {
				if serviceCtx.Msg.RowCount != tc.expectedRow.Int64 {
					t.Errorf("questionBanks测试期望返回%d条数据，实际返回%d条数据", tc.expectedRow.Int64, serviceCtx.Msg.RowCount)
				}
			}
		})
	}
}

// 添加题库测试
func TestBankPostMethod(t *testing.T) {
	z.Info("TestBankPostMethod is running...")
	testCases := []struct {
		name             string
		description      string
		requestBody      string
		forceError       string
		expectedError    bool
		expectedErrorMsg string
		userID           int64
		userRole         int64
	}{
		{
			name:        "有权限-管理员角色",
			description: "正常添加题库-管理员角色",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"type": "00",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:    "",
			expectedError: false,
			userID:        testUserID1,
			userRole:      RoleAdmin,
		},
		{
			name:        "有权限-教师角色",
			description: "正常添加题库-教师角色",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"type": "00",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:    "",
			expectedError: false,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:        "无权限-学生角色",
			description: "正常添加题库-学生角色，期望失败",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"type": "00",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:       "",
			expectedError:    true,
			expectedErrorMsg: fmt.Errorf("domain %d is not allowed", RoleStudent).Error(),
			userID:           testUserID1,
			userRole:         RoleStudent,
		},
		{
			name:             "body为空",
			description:      "body为空，期望失败",
			requestBody:      "",
			forceError:       "",
			expectedError:    true,
			expectedErrorMsg: fmt.Errorf("call /api/question-banks with empty body").Error(),
			userID:           testUserID1,
			userRole:         RoleTeacher,
		},
		{
			name:          "json格式错误",
			description:   "json格式错误，期望失败",
			requestBody:   `not json`,
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:        "data.name为空",
			description: "data.name为空，期望失败",
			requestBody: `{
				"data": {
					"type": "00",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:       "",
			expectedError:    true,
			expectedErrorMsg: fmt.Errorf("call /api/question-banks with empty question bank name").Error(),
			userID:           testUserID1,
			userRole:         RoleTeacher,
		},
		{
			name:        "data.type为空",
			description: "data.type为空，期望失败",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:       "",
			expectedError:    true,
			expectedErrorMsg: fmt.Errorf("call /api/question-banks with empty question bank type").Error(),
			userID:           testUserID1,
			userRole:         RoleTeacher,
		},
		{
			name:        "data结构转换为题库格式失败",
			description: "body中的data结构，在转换为题库格式失败，原因为tags类型错误，期望失败",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"type": "00",
					"tags": "数据结构"
				}
			}`,
			forceError:    "",
			expectedError: true,
			userID:        testUserID1,
			userRole:      RoleTeacher,
		},
		{
			name:        "创建者为空",
			description: "创建者的值小于0，是不合法的，期望失败",
			requestBody: `{
				"data": {
					"name": "测试综合题库",
					"type": "00",
					"tags": [
						"数据结构",
						"算法",
						"操作系统",
						"计算机网络"
					]
				}
			}`,
			forceError:       "",
			expectedError:    true,
			expectedErrorMsg: fmt.Errorf("invalid userID").Error(),
			userID:           -1,
			userRole:         RoleTeacher,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := createMockContextWithBody("POST", "/api/exam", tc.requestBody, tc.forceError, tc.userID, tc.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("questionBanks 意外panic: %v", r)
					}
				}()

				questionBanks(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tc.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("questionBanks测试期望返回错误，但实际成功")
					return
				}

			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("questionBanks测试期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}

			// 错误信息检查
			if serviceCtx.Err != nil && tc.expectedErrorMsg != serviceCtx.Err.Error() {
				t.Errorf("questionBanks测试期望错误信息为:%s，实际错误信息为:%s", tc.expectedErrorMsg, serviceCtx.Err.Error())
			}
		})
	}
}

// // 测试查询题目
//
//	func TestQuestionGetMethod(t *testing.T) {
//		testCases := []struct {
//			name             string
//			description      string
//			query            string
//			forceError       string
//			expectedError    bool
//			expectedErrorMsg string
//			expectedRow      null.Int
//			userID           int64
//			userRole         int64
//		}{
//			{
//				name:             "无权限访问-学生角色",
//				description:      "使用学生身份访问时，应该返回无权限的错误",
//				query:            "",
//				forceError:       "",
//				expectedError:    true,
//				expectedErrorMsg: fmt.Errorf("domain %s is not allowed", RoleStudent).Error(),
//				userID:           testUserID1,
//				userRole:         RoleStudent,
//			},
//			{
//				name:          "有权限访问-教务员角色",
//				description:   "使用教务员身份访问时，返回正确结果",
//				query:         fmt.Sprintf("bankID=%d", testBankID),
//				forceError:    "",
//				expectedError: false,
//				expectedRow:   null.IntFrom(25),
//				userID:        testUserID1,
//				userRole:      RoleAdmin,
//			},
//			{
//				name:          "有权限访问-教师角色",
//				description:   "使用教师身份访问时，返回正确结果",
//				query:         fmt.Sprintf("bankID=%d", testBankID),
//				forceError:    "",
//				expectedError: false,
//				expectedRow:   null.IntFrom(25),
//				userID:        testUserID1,
//				userRole:      RoleTeacher,
//			},
//			{
//				name:             "缺少bankID",
//				query:            "",
//				forceError:       "",
//				expectedError:    true,
//				expectedErrorMsg: fmt.Errorf("bankID is empty").Error(),
//				userID:           testUserID1,
//				userRole:         RoleTeacher,
//			},
//			{
//				name:             "无效的bankID",
//				query:            "bankID=-1",
//				forceError:       "",
//				expectedError:    true,
//				expectedErrorMsg: fmt.Errorf("invalid bankID").Error(),
//				userID:           testUserID1,
//				userRole:         RoleTeacher,
//			},
//			{},
//		}
//	}
func TestMain(m *testing.M) {
	cmn.ConfigureForTest()
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
