package paper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"

	"w2w.io/cmn"
	"w2w.io/null"
)

const (
	initQuestionUserID = 9999999
)

var (
	BankQuestionIDs = []int64{10000001, 10000002, 10000003, 10000004, 10000005}
	TestUserIDs     = []int64{90001, 90002, 90003, 90004, 90005} // 测试用户ID列表
)

// createMockContextWithBody 构造带body的context，仿照exam_mgt/exam-mgt_test.go
func createMockContextWithBody(method, path string, data any, forceError string, userID int64) context.Context {
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
			ID: null.NewInt(userID, true),
		},
	}
	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
	return context.WithValue(ctx, "force-error", forceError)
}

// createMockContextWithUnMarshalBody 构造带未序列化body的context，仿照exam_mgt/exam-mgt_test.go
func createMockContextWithUnMarshalBody(method, path string, data string, forceError string, userID int64) context.Context {
	var req *http.Request
	if data != "" {
		bodyBytes := json.RawMessage(data)
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
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
			ID: null.NewInt(userID, true),
		},
	}
	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
	return context.WithValue(ctx, "force-error", forceError)
}

// cleanupTestPaperData 清理测试插入的paper、group、question等数据
func cleanupTestPaperData(t *testing.T, paperIDs []int64) {
	if len(paperIDs) == 0 {
		return
	}
	db := cmn.GetPgxConn()
	ctx := context.Background()
	txn, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Logf("[cleanup] begin tx failed: %v", err)
		return
	}
	defer txn.Rollback(ctx)
	// 1. 删除 t_paper_question
	_, _ = txn.Exec(ctx, `DELETE FROM t_paper_question WHERE group_id IN (SELECT id FROM t_paper_group WHERE paper_id = ANY($1))`, paperIDs)
	// 2. 删除 t_paper_group
	_, _ = txn.Exec(ctx, `DELETE FROM t_paper_group WHERE paper_id = ANY($1)`, paperIDs)
	// 3. 删除 t_paper
	_, _ = txn.Exec(ctx, `DELETE FROM t_paper WHERE id = ANY($1)`, paperIDs)
	_ = txn.Commit(ctx)
}

// initTestQuestionBankData 初始化题库题目数据
func initTestQuestionBankData() {
	// 五道题
	questions := []struct {
		id         int64
		qtype      string
		difficulty string
		creator    int64
		status     string
	}{
		{BankQuestionIDs[0], "00", "1", initQuestionUserID, "00"},
		{BankQuestionIDs[1], "02", "1", initQuestionUserID, "00"},
		{BankQuestionIDs[2], "04", "1", initQuestionUserID, "00"},
		{BankQuestionIDs[3], "06", "1", initQuestionUserID, "00"},
		{BankQuestionIDs[4], "08", "1", initQuestionUserID, "00"},
	}
	db := cmn.GetPgxConn()
	ctx := context.Background()
	for _, q := range questions {
		mustExec(ctx, db, `
			INSERT INTO assessuser.t_question (id, type, difficulty, creator,status)
			VALUES ($1, $2, $3, $4, $5)
		`, q.id, q.qtype, q.difficulty, q.creator, q.status)
	}

}

func createTestPaperWithGroupsAndQuestions(ctx context.Context, bankQuestionIDs []int64, testUserID int64) (paperID int64, groupIDs []int64, questionIDs []int64, err error) {
	now := time.Now().UnixMilli()
	pgxConn := cmn.GetPgxConn()

	// 开始事务
	tx, err := pgxConn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, nil, nil, fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback(ctx)

	// 创建试卷
	paper := &cmn.TPaper{
		Name:              null.StringFrom("Test Paper"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("02"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("Test Description"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           null.IntFrom(testUserID),
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         null.IntFrom(testUserID),
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
		AccessMode:        null.StringFrom("00"), // 默认访问模式
	}

	//初始化一张空试卷
	err = tx.QueryRow(ctx, `
		INSERT INTO t_paper 
			(name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
		RETURNING id`,
		paper.Name.String,
		paper.AssemblyType.String,
		paper.Category.String,
		paper.Level.String,
		paper.SuggestedDuration.Int64,
		paper.Tags,
		paper.Creator.Int64,
		paper.CreateTime.Int64,
		paper.UpdatedBy.Int64,
		paper.UpdateTime.Int64,
		paper.Status.String,
		paper.AccessMode.String,
	).Scan(&paperID)

	if err != nil {
		return 0, nil, nil, fmt.Errorf("创建试卷失败: %v", err)
	}

	// 定义题组
	groupNames := []string{"Group A", "Group B"}
	groupIDMap := make(map[string]int64)

	// 创建题组
	for i, name := range groupNames {
		var groupID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO t_paper_group 
				(paper_id, name, "order", creator, create_time, updated_by, update_time, status) 
			VALUES 
				($1, $2, $3, $4, $5, $4, $5, $6) 
			RETURNING id`,
			paperID,
			name,
			i+1,
			testUserID,
			now,
			"00",
		).Scan(&groupID)

		if err != nil {
			return paperID, nil, nil, fmt.Errorf("创建题组失败: %v", err)
		}

		groupIDs = append(groupIDs, groupID)
		groupIDMap[fmt.Sprintf("g%d", i)] = groupID
	}

	// 为每个题组添加题目
	if len(bankQuestionIDs) > 0 {
		for _, groupID := range groupIDs {
			for j, bankQuestionID := range bankQuestionIDs {
				if j >= 2 { // 每个题组最多添加2道题
					break
				}
				var questionID int64
				err = tx.QueryRow(ctx, `
					INSERT INTO t_paper_question 
						(bank_question_id, group_id, "order", score, creator, create_time, updated_by, update_time, status, sub_score) 
					VALUES 
						($1, $2, $3, $4, $5, $6, $5, $6, $7, $8) 
					RETURNING id`,
					bankQuestionID,
					groupID,
					j+1,
					6.0, // 默认分数
					testUserID,
					now,
					"00",
					types.JSONText(`[1,2,3]`),
				).Scan(&questionID)

				if err != nil {
					return paperID, groupIDs, nil, fmt.Errorf("创建试题失败: %v", err)
				}
				questionIDs = append(questionIDs, questionID)
			}
		}
	}

	// 提交事务
	if err = tx.Commit(ctx); err != nil {
		return paperID, groupIDs, questionIDs, fmt.Errorf("提交事务失败: %v", err)
	}

	return paperID, groupIDs, questionIDs, nil
}

func mustExec(ctx context.Context, db *pgxpool.Pool, query string, args ...interface{}) {
	_, err := db.Exec(ctx, query, args...)
	if err != nil {
		panic(fmt.Sprintf("Failed to execute query: %s, args: %v, error: %v", query, args, err))
	}
}

func cleanupTestBankQuestions() {
	db := cmn.GetPgxConn()
	ctx := context.Background()
	txn, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		panic(fmt.Sprintf("Failed to begin transaction: %v", err)) // 如果开始事务失败，直接 panic
	}
	defer txn.Rollback(ctx)
	// 删除题目
	_, err = txn.Exec(ctx, `DELETE FROM assessuser.t_question WHERE creator = $1`, initQuestionUserID)
	if err != nil {
		fmt.Printf("Failed to delete test questions: %v\n", err)
		return
	}
	// 删除试卷
	_, err = txn.Exec(ctx, `DELETE FROM assessuser.t_paper WHERE creator = ANY($1)`, TestUserIDs)
	if err != nil {
		fmt.Printf("Failed to delete test papers: %v\n", err)
		return
	}
	_ = txn.Commit(ctx)
}

func TestMain(m *testing.M) {
	cmn.ConfigureForTest()
	initTestQuestionBankData()
	m.Run()
	cleanupTestBankQuestions()
}

func TestPaperListGetMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	ctx := context.Background()
	userID := int64(90002)
	otherUserID := int64(90099)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		wantError     bool
		userID        int64
		forceError    string
		expectedError string // 新增期望的错误信息字段
		setup         func() []int64
	}{
		{
			name:          "正常分页查询",
			query:         "page=1&pageSize=10",
			expectedCount: 3,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 3; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("测试试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:          "名称过滤",
			query:         "name=唯一名",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx,
					`INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ('唯一名试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "标签过滤",
			query:         "tags=vue",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx,
					`INSERT INTO t_paper (name, category,tags, creator, create_time, updated_by, update_time, status) 
	VALUES ('唯一名试卷', '00',$3, $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli(), types.JSONText(`["vue"]`)).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "分类过滤",
			query:         "category=02",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx,
					`INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ('分类试卷', '02', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "搜素不存在分类",
			query:         "category=03",
			expectedCount: 0,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx,
					`INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ('分类试卷', '03', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "分页边界-第二页无数据",
			query:         "page=2&pageSize=5",
			expectedCount: 0,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 5; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("分页试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:          "非法分页参数",
			query:         "page=abc&pageSize=xyz",
			expectedCount: 10, // 默认page=1,pageSize=10
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 10; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("非法分页试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:          "组合过滤-名称+分类+标签",
			query:         "name=组合试卷&category=02&tags=go",
			expectedCount: 1,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx,
					`INSERT INTO t_paper (name, category,tags, creator, create_time, updated_by, update_time, status) 
	VALUES ('组合试卷', '02',$3, $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli(), types.JSONText(`["go"]`)).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "空参数-默认分页",
			query:         "",
			expectedCount: 5,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 5; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("默认分页试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:          "极大页码",
			query:         "page=999&pageSize=10",
			expectedCount: 0,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 3; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("大页码试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:          "非创建者权限",
			query:         "",
			expectedCount: 0,
			wantError:     false,
			userID:        otherUserID,
			forceError:    "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('他人试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:          "无数据",
			query:         "name=不存在的试卷",
			expectedCount: 0,
			wantError:     false,
			userID:        userID,
			forceError:    "",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "无效用户ID",
			query:         "",
			wantError:     true,
			userID:        0,
			forceError:    "",
			expectedError: "Invalid UserID",
			setup:         func() []int64 { return nil },
		},
		{
			name:       "ForceErrorQueryCount",
			query:      "",
			wantError:  true,
			userID:     userID,
			forceError: "getPaperList-QueryRowCount-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "QueryRow",
			query:      "",
			wantError:  true,
			userID:     userID,
			forceError: "getPaperList-QueryRow-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "RowScan",
			query:      "",
			wantError:  true,
			userID:     userID,
			forceError: "getPaperList-RowScan-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 3; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("测试试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "RowScan",
			query:      "",
			wantError:  true,
			userID:     userID,
			forceError: "getPaperList-RowErr-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 3; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
	VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("测试试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paperIDs []int64
			if tt.setup != nil {
				paperIDs = tt.setup()
				t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			}
			ctxGet := createMockContextWithBody("GET", "/paper?"+tt.query, "", tt.forceError, tt.userID)
			q := cmn.GetCtxValue(ctxGet)
			q.R.URL.RawQuery = tt.query
			PaperList(ctxGet)
			if tt.wantError {
				if q.Msg.Status == 0 || !strings.Contains(q.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, q.Msg)
				}
			} else {
				if q.Msg.Status != 0 {
					t.Fatalf("期望成功, 实际: %+v", q.Msg)
				}
				var papers []struct{ ID int64 }
				_ = json.Unmarshal(q.Msg.Data, &papers)
				if len(papers) != tt.expectedCount {
					t.Errorf("返回数量不符, got %d, want %d", len(papers), tt.expectedCount)
				}
			}
		})
	}
}

func TestPaperListDeleteMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	ctx := context.Background()
	userID := int64(90003)

	tests := []struct {
		name          string
		deleteIDs     []int64
		wantError     bool
		userID        int64
		forceError    string
		expectedError string
		setup         func() []int64
	}{
		{
			name:       "正常批量删除",
			deleteIDs:  nil, // 由setup生成
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "部分ID无效",
			deleteIDs:  nil, // setup生成+1个不存在的ID
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 1; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, "部分无效试卷", userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				// 加入一个不存在的ID
				return append(ids, 99999999)
			},
		},
		{
			name:       "无效用户ID",
			deleteIDs:  nil,
			wantError:  true,
			userID:     0,
			forceError: "",
			setup: func() []int64 {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return []int64{id}
			},
		},
		{
			name:       "io.ReadAll-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "PaperList-delete-io.ReadAll-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "Body.Close-err",
			deleteIDs:  nil,
			wantError:  false,
			userID:     userID,
			forceError: "PaperList-delete-Body.Close-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "json.Unmarshal1-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "PaperList-delete-json.Unmarshal1-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "json.Unmarshal2-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "PaperList-delete-json.Unmarshal2-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "BeginTx-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "PaperList-delete-BeginTx-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "Commit-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "PaperList-delete-Commit-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "Rollback-err",
			deleteIDs:  nil,
			wantError:  false,
			userID:     userID,
			forceError: "PaperList-delete-Rollback-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "空试卷ID",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "",
			setup: func() []int64 {
				return nil
			},
		},
		{
			name:       "deletePapers-exec-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "deletePapers-exec-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "deletePapersgroups-exec-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "deletePapersgroups-exec-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
		{
			name:       "deletePapersquestions-exec-err",
			deleteIDs:  nil,
			wantError:  true,
			userID:     userID,
			forceError: "deletePapersquestions-exec-err",
			setup: func() []int64 {
				var ids []int64
				for i := 0; i < 2; i++ {
					var id int64
					_ = db.QueryRow(ctx, `INSERT INTO t_paper 
    (name, category, creator, create_time, updated_by, update_time, status) 
VALUES ($1, '00', $2, $3, $2, $3, '00') RETURNING id`, fmt.Sprintf("待删除试卷%d", i+1), userID, time.Now().UnixMilli()).Scan(&id)
					ids = append(ids, id)
				}
				return ids
			},
		},
	}

	t.Run("buf is nil", func(t *testing.T) {
		ctxDel := createMockContextWithBody("DELETE", "/paper", nil, "", userID)
		PaperList(ctxDel)
		q := cmn.GetCtxValue(ctxDel)
		if q.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", q.Msg)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paperIDs []int64
			if tt.setup != nil {
				paperIDs = tt.setup()
				t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			}
			deleteIDs := tt.deleteIDs
			if deleteIDs == nil {
				deleteIDs = paperIDs
			}
			ctxDel := createMockContextWithBody("DELETE", "/paper", deleteIDs, tt.forceError, tt.userID)
			PaperList(ctxDel)
			q := cmn.GetCtxValue(ctxDel)
			if tt.wantError {
				if q.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", q.Msg)
				}
			} else {
				if q.Msg.Status != 0 || q.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", q.Msg)
				}
				// 校验数据库status已变更
				for _, id := range paperIDs {
					var status string
					err := db.QueryRow(ctx, "SELECT status FROM t_paper WHERE id=$1", id).Scan(&status)
					if err != nil {
						if errors.Is(err, pgx.ErrNoRows) {
							continue
						}
					}
					if status != "02" { // StatusUnNormal
						t.Errorf("删除后status应为'02', got %v", status)
					}
				}
			}
		})
	}
}

func TestManualPaperPostMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	ctx := context.Background()
	userID := int64(91001)

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		forceError    string
		expectedError string
		setup         func() []int64
	}{
		{
			name:       "正常新建试卷",
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup:      func() []int64 { return nil },
		},
		{
			name:          "无效用户ID",
			wantError:     true,
			userID:        0,
			forceError:    "",
			expectedError: "无效用户ID",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "BeginTx-err",
			wantError:     true,
			userID:        userID,
			forceError:    "BeginTx-err",
			expectedError: "BeginTx-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "recover-err",
			wantError:     false,
			userID:        userID,
			forceError:    "recover-err",
			expectedError: "recover-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "Rollback-err",
			wantError:     false,
			userID:        userID,
			forceError:    "Rollback-err",
			expectedError: "Rollback-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "Commit-err",
			wantError:     true,
			userID:        userID,
			forceError:    "Commit-err",
			expectedError: "Commit-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "tx.QueryRow-err",
			wantError:     true,
			userID:        userID,
			forceError:    "tx.QueryRow-err",
			expectedError: "tx.QueryRow-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "tx.Query-err",
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Query-err",
			expectedError: "tx.Query-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "rows.Scan-err",
			wantError:     true,
			userID:        userID,
			forceError:    "rows.Scan-err",
			expectedError: "rows.Scan-err",
			setup:         func() []int64 { return nil },
		},
		{
			name:          "rows.Err-err",
			wantError:     true,
			userID:        userID,
			forceError:    "rows.Err-err",
			expectedError: "rows.Err-err",
			setup:         func() []int64 { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paperIDs []int64
			if tt.setup != nil {
				paperIDs = tt.setup()
				t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			}
			ctxPost := createMockContextWithBody("POST", "/paper/manual", "", tt.forceError, tt.userID)
			ManualPaper(ctxPost)
			q := cmn.GetCtxValue(ctxPost)
			if tt.wantError {
				if q.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", q.Msg)
				}
			} else {
				if q.Msg.Status != 0 {
					t.Fatalf("期望成功, 实际: %+v", q.Msg)
				}
				var resp struct {
					Paper struct{ ID null.Int }
				}
				if err := json.Unmarshal(q.Msg.Data, &resp); err != nil {
					t.Fatalf("POST返回数据解析失败: %v", err)
				}
				if !resp.Paper.ID.Valid || resp.Paper.ID.Int64 == 0 {
					t.Fatalf("POST未返回有效paperID: %+v", resp)
				}
				// 校验数据库确实有该paper
				var cnt int
				_ = db.QueryRow(ctx, "SELECT COUNT(*) FROM t_paper WHERE id=$1", resp.Paper.ID.Int64).Scan(&cnt)
				if cnt != 1 {
					t.Errorf("数据库未找到新建试卷记录")
				}
				// 清理
				cleanupTestPaperData(t, []int64{resp.Paper.ID.Int64})
			}
		})
	}
}

func TestManualPaperPutMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	ctx := context.Background()
	userID := int64(91002)

	tests1 := []struct {
		name          string
		reqBody       *UpdateManualPaperRequest
		wantError     bool
		userID        int64
		forceError    string
		expectedError string // 增加期望的错误信息字段
		setup         func() (int64, []int64)
		validate      func(*testing.T, context.Context, *cmn.ServiceCtx, int64)
	}{
		{
			name: "正常更新试卷",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
                                    VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') 
                                    RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category, level, desc string
				_ = db.QueryRow(ctx, "SELECT name, category, level, description FROM t_paper WHERE id=$1", paperID).Scan(&name, &category, &level, &desc)
				if name != "单元测试试卷" || category != "02" || level != "04" || desc != "desc" {
					t.Errorf("PUT后数据库字段未正确更新: got %s %s %s %s", name, category, level, desc)
				}
			},
		},
		{
			name: "无效用户ID",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        0,
			forceError:    "",
			expectedError: "无效用户ID",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
                                    VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') 
                                    RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "ParseInt-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "invalid syntax",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "io.ReadAll-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:  true,
			userID:     userID,
			forceError: "io.ReadAll-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "R.Body.Close-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     false,
			userID:        userID,
			forceError:    "R.Body.Close-err",
			expectedError: "R.Body.Close-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "json.Unmarshal-err",
			expectedError: "json.Unmarshal-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "validate-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Field validation for 'Actions' failed on the 'min' tag",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "无效的试卷ID",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "无效的试卷ID",
			setup: func() (int64, []int64) {
				return -1, []int64{}
			},
		},
		{
			name: "unexists paper",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "试卷不存在",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return 99999999, []int64{id}
			},
		},
		//测试updateManualPaper函数
		{
			name: "BeginTx-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "BeginTx-err",
			expectedError: "BeginTx-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "recover-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "recover-err",
			expectedError: "recover-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('单元测试试卷', '02', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "unsupported action type",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "rollback-err",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "unsupported action type",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("回滚错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		{
			name: "Rollback-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "rollback-err",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "Rollback-err",
			expectedError: "Rollback-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("回滚错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		{
			name: "Commit-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "Commit-err",
			expectedError: "Commit-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		//update_info
		{
			name: "update_info-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "update_info",
						Payload: json.RawMessage(`[]`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal array into Go value of type paper.UpdatePaperBasicInfoRequest",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		{
			name: "empty update_info",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "update_info",
						Payload: json.RawMessage(`{}`),
					},
				},
			},
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category, level, desc string
				var updateTime int64
				_ = db.QueryRow(ctx, "SELECT name, category, level, description, update_time FROM t_paper WHERE id=$1", paperID).Scan(&name, &category, &level, &desc, &updateTime)
				if name != "待更新试卷" || category != "00" || level != "" || desc != "" {
					t.Errorf("空更新后数据库字段被修改: name=%s category=%s level=%s desc=%s", name, category, level, desc)
				}
			},
		},
		{
			name: "update_info-tx.Exec-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "update_info",
						Payload: json.RawMessage(`{  
                        "Name": "单元测试试卷",
                        "category": "02",
                        "level": "04",
                        "duration": 60,
                        "description": "desc",
                        "tags": ["tag1", "tag2"]
                    }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("tx.Exec执行失败，事务应该回退，数据不应该更新: name=%s category=%s", name, category)
				}
			},
		},
		//add_group
		{
			name: "add_group-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "add_group",
						Payload: json.RawMessage(`[]`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			expectedError: "cannot unmarshal array into Go value of type paper.AddQuestionGroupRequest",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		{
			name: "正常添加题组",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "add_group",
						Payload: json.RawMessage(`{
                            "name": "一、单选题",
                            "order": 1
                        }`),
					},
				},
			},
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				// 从响应中获取新建的题组ID
				var results []ActionResult
				err := json.Unmarshal(q.Msg.Data, &results)
				if err != nil {
					t.Fatalf("解析响应失败: %v", err)
				}

				// 将 result.Result 从 float64 转换为 int64
				resultFloat, ok := results[0].Result.(float64)
				if !ok {
					t.Fatalf("无法将result转换为float64,实际类型=%T", results[0].Result)
				}
				GroupID := int64(resultFloat)

				// 验证题组是否被正确创建
				var name string
				var order int
				err = db.QueryRow(ctx, `SELECT name, "order" FROM t_paper_group WHERE id=$1 AND paper_id=$2`,
					GroupID, paperID).Scan(&name, &order)

				if err != nil {
					t.Fatalf("查询题组失败: %v", err)
				}

				// 验证字段值
				if name != "一、单选题" {
					t.Errorf("题组名称错误,期望='一、单选题',实际=%s", name)
				}
				if order != 1 {
					t.Errorf("题组顺序错误,期望=1,实际=%d", order)
				}
			},
		},
		{
			name: "add_group-tx.QueryRow-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "add_group",
						Payload: json.RawMessage(`{
                            "name": "一、单选题",
                            "order": 1
                        }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "tx.QueryRow-err",
			expectedError: "tx.QueryRow-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				// 校验响应是否包含错误信息
				if q.Msg.Status == 0 || !strings.Contains(q.Msg.Msg, "tx.QueryRow-err") {
					t.Errorf("期望QueryRow错误,实际响应:%+v", q.Msg)
				}

				// 验证没有创建题组
				var count int
				err := db.QueryRow(ctx, "SELECT COUNT(*) FROM t_paper_group WHERE paper_id=$1", paperID).Scan(&count)
				if err != nil {
					t.Fatalf("查询题组失败: %v", err)
				}
				if count != 0 {
					t.Errorf("期望题组数=0,实际=%d", count)
				}
			},
		},
		{
			name: "add_group-validate-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action: "add_group",
						Payload: json.RawMessage(`{
                            "name": "",
                            "order": -1
                        }`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Field validation for 'Name' failed on the 'required' tag",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		//delete_group
		{
			name: "delete_group-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "delete_group",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type int64",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//add_question
		{
			name: "add_question-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "add_question",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type []paper.AddQuestionsRequest",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//delete_question
		{
			name: "delete_question-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "delete_question",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type []int64",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//update_question
		{
			name: "update_question-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "update_question",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type []paper.UpdatePaperQuestionRequest",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//update_group
		{
			name: "update_group-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "update_group",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type paper.UpdateQuestionsGroupRequest",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//move_question
		{
			name: "move_question-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "move_question",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type []int64",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
		//move_group
		{
			name: "move_group-json.Unmarshal-err",
			reqBody: &UpdateManualPaperRequest{
				[]UpdateManualPaperAction{
					{
						Action:  "move_group",
						Payload: json.RawMessage(`""`),
					},
				},
			},
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "cannot unmarshal string into Go value of type []int64",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var name, category string
				_ = db.QueryRow(ctx, "SELECT name, category FROM t_paper WHERE id=$1", paperID).Scan(&name, &category)
				if name != "待更新试卷" || category != "00" {
					t.Errorf("非法JSON反序列化错误后数据被修改: name=%s category=%s", name, category)
				}
			},
		},
	}

	t.Run("UnmarshalJSON", func(t *testing.T) {
		var id int64
		_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
		t.Cleanup(func() { cleanupTestPaperData(t, []int64{id}) })

		ctxPut := createMockContextWithUnMarshalBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(id), `{`, "", userID)
		qPut := cmn.GetCtxValue(ctxPut)
		qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", id)
		ManualPaper(ctxPut)
		if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, "unexpected end of JSON input") {
			t.Errorf("期望错误信息包含'%s', 实际: %+v", "unexpected end of JSON input", qPut.Msg)
		}
	})

	t.Run("buf is nil", func(t *testing.T) {
		var paperID int64
		_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
                                    VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') 
                                    RETURNING id`, userID, time.Now().UnixMilli()).Scan(&paperID)
		t.Cleanup(func() { cleanupTestPaperData(t, []int64{paperID}) })
		ctxPut := createMockContextWithBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(paperID), nil, "", userID)
		qPut := cmn.GetCtxValue(ctxPut)
		qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
		ManualPaper(ctxPut)
		if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, "/api/paper/manual with empty body") {
			t.Errorf("期望错误信息包含'%s', 实际: %+v", "/api/paper/manual with empty body", qPut.Msg)
		}
	})

	for _, tt := range tests1 {
		t.Run(tt.name, func(t *testing.T) {
			paperID, paperIDs := tt.setup()
			t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			ctxPut := createMockContextWithBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(paperID), tt.reqBody, tt.forceError, tt.userID)
			qPut := cmn.GetCtxValue(ctxPut)
			qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
			if tt.name == "ParseInt-err" {
				qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%s", "123abc")
			}
			ManualPaper(ctxPut)
			if tt.wantError {
				if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, qPut.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
				if tt.validate != nil {
					tt.validate(t, ctx, qPut, paperID)
				}
			}
		})
	}

	test2 := []struct {
		name          string
		reqBody       any
		wantError     bool
		userID        int64
		forceError    string
		expectedError string // 增加期望的错误信息字段
		setup         func(t *testing.T) (int64, any)
		validate      func(*testing.T, context.Context, *cmn.ServiceCtx, int64)
	}{
		//delete_group
		{
			name:       "正常删除题组",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				// 创建一个题组以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3) RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				// 创建删除题组的结构体
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_group",
							Payload: json.RawMessage(fmt.Sprintf("%d", groupID)),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupCount int64
				_ = db.QueryRow(ctx, "SELECT group_count FROM v_paper WHERE id=$1", paperID).Scan(&groupCount)
				if groupCount != 0 {
					t.Errorf("非法JSON反序列化错误后数据被修改: group_count=%d", groupCount)
				}
			},
		},
		{
			name:          "不合规题组ID",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: ErrEmptyGroupID.Error(),
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				// 创建一个题组以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3) RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				// 创建删除题组的结构体
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_group",
							Payload: json.RawMessage(fmt.Sprintf("%d", -1)),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupCount int64
				_ = db.QueryRow(ctx, "SELECT group_count FROM v_paper WHERE id=$1", paperID).Scan(&groupCount)
				if groupCount != 0 {
					t.Errorf("非法JSON反序列化错误后数据被修改: group_count=%d", groupCount)
				}
			},
		},
		{
			name:          "handleDeleteGroup-tx.QueryRow-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "handleDeleteGroup-tx.QueryRow-err",
			expectedError: "handleDeleteGroup-tx.QueryRow-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				// 创建一个题组以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3) RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				// 创建删除题组的结构体
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_group",
							Payload: json.RawMessage(fmt.Sprintf("%d", groupID)),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupCount int64
				_ = db.QueryRow(ctx, "SELECT group_count FROM v_paper WHERE id=$1", paperID).Scan(&groupCount)
				if groupCount != 0 {
					t.Errorf("非法JSON反序列化错误后数据被修改: group_count=%d", groupCount)
				}
			},
		},
		{
			name:          "handleDeleteGroup-tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "handleDeleteGroup-tx.Exec-err",
			expectedError: "handleDeleteGroup-tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				// 创建一个题组以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3) RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				// 创建删除题组的结构体
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_group",
							Payload: json.RawMessage(fmt.Sprintf("%d", groupID)),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupCount int64
				_ = db.QueryRow(ctx, "SELECT group_count FROM v_paper WHERE id=$1", paperID).Scan(&groupCount)
				if groupCount != 0 {
					t.Errorf("非法JSON反序列化错误后数据被修改: group_count=%d", groupCount)
				}
			},
		},
		{
			name:          "题组不存在于当前试卷",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: ErrRecordNotFound.Error(),
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				// 创建一个题组以便删除
				_ = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3) RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				// 创建删除题组的结构体
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_group",
							Payload: json.RawMessage(fmt.Sprintf("%d", 99999999)),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupCount int64
				_ = db.QueryRow(ctx, "SELECT group_count FROM v_paper WHERE id=$1", paperID).Scan(&groupCount)
				if groupCount != 0 {
					t.Errorf("非法JSON反序列化错误后数据被修改: group_count=%d", groupCount)
				}
			},
		},
		//add_question
		{
			name:       "正常添加题目",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          1,
						BankQuestionID: BankQuestionIDs[0],
						Score:          2,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(1), questionCount)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				//解析q中DATA返回的题目ID
				var results []ActionResult
				err = json.Unmarshal(q.Msg.Data, &results)
				require.NoError(t, err)
				qIDmaps, ok := results[0].Result.(map[string]interface{})
				require.True(t, ok)
				qIDFloat64, ok := qIDmaps["temp_question1"].(float64)
				qID := int64(qIDFloat64)
				require.True(t, ok)
				require.Equal(t, len(qIDmaps), 1, "期望题目ID映射长度为1")
				require.Equal(t, len(groups), 1, "期望题组长度为1")
				require.Equal(t, groups[0].Questions[0].ID.Int64, qID, "期望题目ID与映射一致")
				require.Equal(t, groups[0].Questions[0].Order.Int64, int64(1), "期望题目顺序为1")
				require.Equal(t, groups[0].Questions[0].Score.Float64, 2.0, "期望题目分数为2")
				require.Equal(t, groups[0].Questions[0].SubScore, []float64{1, 2, 3}, "期望题目子分数为[1, 2, 3]")
			},
		},
		{
			name:          "添加的题目分值为0",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Field validation for 'Score' failed on the 'required' tag",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          1,
						BankQuestionID: BankQuestionIDs[0],
						Score:          0,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "添加的题目序号为0",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Field validation for 'Order' failed on the 'required' tag",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          0,
						BankQuestionID: BankQuestionIDs[0],
						Score:          2,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "tx.Query-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Query-err",
			expectedError: "tx.Query-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          1,
						BankQuestionID: BankQuestionIDs[0],
						Score:          2,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "rows.Scan-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "rows.Scan-err",
			expectedError: "rows.Scan-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          1,
						BankQuestionID: BankQuestionIDs[0],
						Score:          2,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "rows.Err-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "rows.Err-err",
			expectedError: "rows.Err-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建添加题目的结构体
				payload := []AddQuestionsRequest{
					{
						TempID:         "temp_question1",
						GroupID:        groupID,
						Order:          1,
						BankQuestionID: BankQuestionIDs[0],
						Score:          2,
						SubScore:       []float64{1, 2, 3},
					},
				}
				jsonPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "add_question",
							Payload: json.RawMessage(jsonPayload),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		//delete_question
		{
			name:       "正常删除题目",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2, $4,$2, $3, $2, $3,'00') RETURNING id`,
					groupID, userID, time.Now().UnixMilli(), BankQuestionIDs[0]).Scan(&questionID)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{questionID})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(0), questionCount)
			},
		},
		{
			name:          "删除题目数组为空",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "题目数组为空",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2, $4,$2, $3, $2, $3,'00') RETURNING id`,
					groupID, userID, time.Now().UnixMilli(), BankQuestionIDs[0]).Scan(&questionID)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(0), questionCount)
			},
		},
		{
			name:          "删除题目ID不存在",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "no rows in result",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2, $4,$2, $3, $2, $3,'00') RETURNING id`,
					groupID, userID, time.Now().UnixMilli(), BankQuestionIDs[0]).Scan(&questionID)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{9999999})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(0), questionCount)
			},
		},
		{
			name:          "tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2, $4,$2, $3, $2, $3,'00') RETURNING id`,
					groupID, userID, time.Now().UnixMilli(), BankQuestionIDs[0]).Scan(&questionID)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{9999999})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "delete_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(0), questionCount)
			},
		},
		//update_question
		{
			name:       "正常更新题目分数",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:    questionID,
						Score: 3,
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(1), questionCount)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 1, "期望题组长度为1")
				require.Equal(t, groups[0].Questions[0].Order.Int64, int64(1), "期望题目顺序为1")
				require.Equal(t, groups[0].Questions[0].Score.Float64, 3.0, "期望题目分数为3")
				require.Equal(t, groups[0].Questions[0].SubScore, []float64{1, 2, 3}, "期望题目子分数为[1, 2, 3]")
			},
		},
		{
			name:       "正常更新题目所在题组",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID, groupID2 int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组1', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:      questionID,
						GroupID: groupID2,
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(1), questionCount)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[1].Questions[0].Order.Int64, int64(1), "期望题目顺序为1")
				require.Equal(t, groups[1].Questions[0].Score.Float64, 2.0, "期望题目分数为2")
				require.Equal(t, groups[1].Questions[0].SubScore, []float64{1, 2, 3}, "期望题目子分数为[1, 2, 3]")
			},
		},
		{
			name:       "正常更新题目小题分",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:       questionID,
						SubScore: []float64{2, 2, 2},
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(1), questionCount)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 1, "期望题组长度为1")
				require.Equal(t, groups[0].Questions[0].Order.Int64, int64(1), "期望题目顺序为1")
				require.Equal(t, groups[0].Questions[0].Score.Float64, 2.0, "期望题目分数为2")
				require.Equal(t, groups[0].Questions[0].SubScore, []float64{2, 2, 2}, "期望题目子分数为[2, 2, 2]")
			},
		},
		{
			name:          "传的小题分为负值",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Error:Field validation for 'SubScore[0]' failed on the 'min' tag",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:       questionID,
						SubScore: []float64{-1, -2, -3},
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:       questionID,
						SubScore: []float64{1, 2, 3},
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "不传递更新字段",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "r",
			expectedError: "更新题目没有传入需要更新的字段或传入的字段为零值",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID: questionID,
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "传入不存在的题目ID",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "no rows in result",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID)
				require.NoError(t, err)
				updateReq := []UpdatePaperQuestionRequest{
					{
						ID:       9999999,
						SubScore: []float64{1, 2, 3},
					},
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		//update_group
		{
			name:       "正常更新题组名称",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				updateReq := UpdateQuestionsGroupRequest{
					ID:   groupID,
					Name: "名字已修改",
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 1, "期望题组长度为1")
				require.Equal(t, groups[0].Name.String, "名字已修改", "期望题组名字为名字已修改")
			},
		},
		{
			name:          "缺少题组ID",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Error:Field validation for 'ID' failed on the 'min' tag",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				updateReq := UpdateQuestionsGroupRequest{
					Name: "名字已修改",
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "缺少更新题组名称",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "Error:Field validation for 'Name' failed on the 'required' tag",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				updateReq := UpdateQuestionsGroupRequest{
					ID: groupID,
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				updateReq := UpdateQuestionsGroupRequest{
					ID:   groupID,
					Name: "11111",
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "题组不存在",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "no rows in result set",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				updateReq := UpdateQuestionsGroupRequest{
					ID:   9999999,
					Name: "11111",
				}
				jsondata, err := json.Marshal(updateReq)
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "update_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		//move_question
		{
			name:       "正常调整题目位置",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{questionID2, questionID1})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var questionCount int64
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT question_count,groups_data FROM v_paper WHERE id=$1", paperID).Scan(&questionCount, &groupData)
				require.NoError(t, err)
				require.Equal(t, int64(2), questionCount)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 1, "期望题组长度为1")
				require.Equal(t, len(groups[0].Questions), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Questions[0].Order.Int64, int64(1), "期望题目顺序为1")
				require.Equal(t, groups[0].Questions[0].BankQuestionID.Int64, BankQuestionIDs[1], fmt.Sprintf("期望题目1关联的题库题目是%d", BankQuestionIDs[1]))
				require.Equal(t, groups[0].Questions[1].Order.Int64, int64(2), "期望题目顺序为2")
				require.Equal(t, groups[0].Questions[1].BankQuestionID.Int64, BankQuestionIDs[0], fmt.Sprintf("期望题目2关联的题库题目是%d", BankQuestionIDs[0]))
			},
		},
		{
			name:          "传入的题目ID重复",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "存在重复ID",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{questionID2, questionID2})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "传入的题目ID存在负数",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "ID必须大于0",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{-100})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "传入的题目数组为空",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "题目数组为空",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{questionID1, questionID2})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "题目数组中的题目并不存在",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "no rows in result set",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{9999999, 888888})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		{
			name:          "请求的题目数组长度与试卷实际题数不符",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID int64
				var questionID1, questionID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '待删除题组', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[0], userID, time.Now().UnixMilli()).Scan(&questionID1)
				require.NoError(t, err)
				// 创建一个题目
				err = db.QueryRow(ctx, `INSERT INTO t_paper_question (group_id, "order",score,sub_score,bank_question_id, creator, create_time, updated_by, update_time,status) 
					VALUES ($1, 1, 2,$2, $3,$4, $5, $4, $5,'00') RETURNING id`,
					groupID, []float64{1, 2, 3}, BankQuestionIDs[1], userID, time.Now().UnixMilli()).Scan(&questionID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{9999999, 888888, 777777})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_question",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
			},
		},
		//move_group
		{
			name:       "正常调整题组位置",
			reqBody:    nil,
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{groupID2, groupID1})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "传入的题组ID存在负数",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "ID必须大于0",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{-99999})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "传入的题组ID重复",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "存在重复ID",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{groupID1, groupID1})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "传入的题目数组为空",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "题组数组为空",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "tx.QueryRow-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.QueryRow-err",
			expectedError: "tx.QueryRow-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{groupID1, groupID2})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "tx.Exec-err",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "tx.Exec-err",
			expectedError: "tx.Exec-err",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{groupID1, groupID2})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
		{
			name:          "题目数组在数据库并不存在",
			reqBody:       nil,
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "no rows in result set",
			setup: func(t *testing.T) (int64, any) {
				var id int64
				var groupID1, groupID2 int64
				// 创建一个试卷
				err := db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组1', 1, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID1)
				require.NoError(t, err)
				// 创建一个题组
				err = db.QueryRow(ctx, `INSERT INTO t_paper_group (paper_id, name, "order", creator, create_time, updated_by, update_time,status) 
					VALUES ($1, '移动题组2', 2, $2, $3, $2, $3,'00') RETURNING id`,
					id, userID, time.Now().UnixMilli()).Scan(&groupID2)
				require.NoError(t, err)
				jsondata, err := json.Marshal([]int64{999999, 888888})
				require.NoError(t, err)
				reqBody := UpdateManualPaperRequest{
					[]UpdateManualPaperAction{
						{
							Action:  "move_group",
							Payload: json.RawMessage(jsondata),
						},
					},
				}
				return id, reqBody
			},
			validate: func(t *testing.T, ctx context.Context, q *cmn.ServiceCtx, paperID int64) {
				var groupData types.JSONText
				err := db.QueryRow(ctx, "SELECT groups_data FROM v_paper WHERE id=$1", paperID).Scan(&groupData)
				require.NoError(t, err)
				var groups []Group
				err = json.Unmarshal(groupData, &groups)
				require.NoError(t, err)
				require.Equal(t, len(groups), 2, "期望题组长度为2")
				require.Equal(t, groups[0].Name.String, "移动题组2", "期望移动题组2排在第一位")
				require.Equal(t, groups[0].Order.Int64, int64(1), "期望移动题组2顺序为1")
				require.Equal(t, groups[1].Name.String, "移动题组1", "期望移动题组1排在第二位")
				require.Equal(t, groups[1].Order.Int64, int64(2), "期望移动题组1顺序为2")
			},
		},
	}

	for _, tt := range test2 {
		t.Run(tt.name, func(t *testing.T) {
			paperID, reqBody := tt.setup(t)
			t.Cleanup(func() { cleanupTestPaperData(t, []int64{paperID}) })
			ctxPut := createMockContextWithBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(paperID), reqBody, tt.forceError, tt.userID)
			qPut := cmn.GetCtxValue(ctxPut)
			qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
			ManualPaper(ctxPut)
			if tt.wantError {
				if qPut.Msg.Status == 0 || !strings.Contains(qPut.Msg.Msg, tt.expectedError) {
					t.Errorf("期望错误信息包含'%s', 实际: %+v", tt.expectedError, qPut.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
				if tt.validate != nil {
					tt.validate(t, ctx, qPut, paperID)
				}
			}
		})
	}
}

func TestManualPaperGetMethod(t *testing.T) {
	cmn.ConfigureForTest()
	db := cmn.GetPgxConn()
	ctx := context.Background()
	userID := int64(91003)

	tests := []struct {
		name          string
		wantError     bool
		userID        int64
		forceError    string
		expectedError string
		setup         func() (int64, []int64)
	}{
		{
			name:          "正常获取试卷详情",
			wantError:     false,
			userID:        userID,
			forceError:    "",
			expectedError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待查试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name:          "无效用户ID",
			wantError:     true,
			userID:        0,
			forceError:    "",
			expectedError: "无效用户ID",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name:          "试卷不存在",
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: ErrRecordNotFound.Error(),
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return 9999999, []int64{id}
			},
		},
		{
			name:          "试卷ID不合规",
			wantError:     true,
			userID:        userID,
			forceError:    "",
			expectedError: "试卷ID不合规",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return -1, []int64{id}
			},
		},
	}

	t.Run("ParseInt Error", func(t *testing.T) {
		ctxGet := createMockContextWithBody("GET", "/paper/manual?paper_id="+fmt.Sprint("str"), "", "", userID)
		qGet := cmn.GetCtxValue(ctxGet)
		qGet.R.URL.RawQuery = fmt.Sprintf("paper_id=%s", "str")
		ManualPaper(ctxGet)
		if qGet.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qGet.Msg)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paperID, paperIDs := tt.setup()
			t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			ctxGet := createMockContextWithBody("GET", "/paper/manual?paper_id="+fmt.Sprint(paperID), "", tt.forceError, tt.userID)
			qGet := cmn.GetCtxValue(ctxGet)
			qGet.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
			ManualPaper(ctxGet)
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
				var resp struct {
					ID   null.Int
					Name null.String
				}
				_ = json.Unmarshal(qGet.Msg.Data, &resp)
				if !resp.ID.Valid || resp.ID.Int64 != paperID {
					t.Errorf("GET返回ID不符: got %v, want %v", resp.ID, paperID)
				}
				var dbName string
				_ = db.QueryRow(ctx, "SELECT name FROM t_paper WHERE id=$1", paperID).Scan(&dbName)
				if resp.Name.String != dbName {
					t.Errorf("GET返回name与数据库不符: got %v, want %v", resp.Name.String, dbName)
				}
			}
		})
	}
}
