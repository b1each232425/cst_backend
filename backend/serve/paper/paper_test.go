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

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"

	"w2w.io/cmn"
	"w2w.io/null"
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
		req = httptest.NewRequest(method, path, strings.NewReader(string([]byte{})))
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

func TestMain(m *testing.M) {
	cmn.ConfigureForTest()
	m.Run()
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
			name:       "无效用户ID",
			query:      "",
			wantError:  true,
			userID:     0,
			forceError: "",
			setup:      func() []int64 { return nil },
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
				if q.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", q.Msg)
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
		name       string
		deleteIDs  []int64
		wantError  bool
		userID     int64
		forceError string
		setup      func() []int64
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
		name       string
		wantError  bool
		userID     int64
		forceError string
		setup      func() []int64
	}{
		{
			name:       "正常新建试卷",
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "无效用户ID",
			wantError:  true,
			userID:     0,
			forceError: "",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "BeginTx-err",
			wantError:  true,
			userID:     userID,
			forceError: "BeginTx-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "recover-err",
			wantError:  false,
			userID:     userID,
			forceError: "recover-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "Rollback-err",
			wantError:  false,
			userID:     userID,
			forceError: "Rollback-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "createManualPaperTx-err",
			wantError:  true,
			userID:     userID,
			forceError: "createManualPaperTx-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "initialManualPaperGroupsTx-err",
			wantError:  true,
			userID:     userID,
			forceError: "initialManualPaperGroupsTx-err",
			setup:      func() []int64 { return nil },
		},
		{
			name:       "Commit-err",
			wantError:  true,
			userID:     userID,
			forceError: "Commit-err",
			setup:      func() []int64 { return nil },
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

	tests := []struct {
		name       string
		reqBody    *UpdateManualPaperRequest
		wantError  bool
		userID     int64
		forceError string
		setup      func() (int64, []int64)
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
			wantError:  true,
			userID:     0,
			forceError: "",
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
			wantError:  true,
			userID:     userID,
			forceError: "ParseInt-err",
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
			wantError:  false,
			userID:     userID,
			forceError: "R.Body.Close-err",
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
			wantError:  true,
			userID:     userID,
			forceError: "json.Unmarshal-err",
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
			wantError:  true,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name: "paperExists-err",
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
			forceError: "paperExists-err",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
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
			wantError:  true,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return 99999999, []int64{id}
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
		if qPut.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qPut.Msg)
		}
	})

	t.Run("buf is nil", func(t *testing.T) {
		var paperID int64
		_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
                                    VALUES ('待更新试卷', '00', $1, $2, $1, $2, '00') 
                                    RETURNING id`, userID, time.Now().UnixMilli()).Scan(&paperID)
		ctxPut := createMockContextWithBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(paperID), nil, "", userID)
		qPut := cmn.GetCtxValue(ctxPut)
		qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
		ManualPaper(ctxPut)
		if qPut.Msg.Status == 0 {
			t.Errorf("期望错误, 实际无错: %+v", qPut.Msg)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paperID, paperIDs := tt.setup()
			t.Cleanup(func() { cleanupTestPaperData(t, paperIDs) })
			ctxPut := createMockContextWithBody("PUT", "/paper/manual?paper_id="+fmt.Sprint(paperID), tt.reqBody, tt.forceError, tt.userID)
			qPut := cmn.GetCtxValue(ctxPut)
			qPut.R.URL.RawQuery = fmt.Sprintf("paper_id=%d", paperID)
			ManualPaper(ctxPut)
			if tt.wantError {
				if qPut.Msg.Status == 0 {
					t.Errorf("期望错误, 实际无错: %+v", qPut.Msg)
				}
			} else {
				if qPut.Msg.Status != 0 || qPut.Msg.Msg != "success" {
					t.Fatalf("期望成功, 实际: %+v", qPut.Msg)
				}
				// 校验数据库字段已变更
				var name, category, level, desc string
				_ = db.QueryRow(ctx, "SELECT name, category, level, description FROM t_paper WHERE id=$1", paperID).Scan(&name, &category, &level, &desc)
				if name != "单元测试试卷" || category != "02" || level != "04" || desc != "desc" {
					t.Errorf("PUT后数据库字段未正确更新: got %s %s %s %s", name, category, level, desc)
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
		name       string
		wantError  bool
		userID     int64
		forceError string
		setup      func() (int64, []int64)
	}{
		{
			name:       "正常获取试卷详情",
			wantError:  false,
			userID:     userID,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('待查试卷', '00', $1, $2, $1, $2, '00') RETURNING id`, userID, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
		{
			name:       "无效用户ID",
			wantError:  true,
			userID:     0,
			forceError: "",
			setup: func() (int64, []int64) {
				var id int64
				_ = db.QueryRow(ctx, `INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) VALUES ('无效用户试卷', '00', 1, $1, 1, $1, '00') RETURNING id`, time.Now().UnixMilli()).Scan(&id)
				return id, []int64{id}
			},
		},
	}

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
