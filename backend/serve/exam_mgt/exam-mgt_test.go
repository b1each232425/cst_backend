/*
 * @Author: Mayux dbs45412@163.com
 * @Date: 2025-08-07 10:55:33
 * @LastEditors: Mayux dbs45412@163.com
 * @LastEditTime: 2025-08-15 12:08:16
 * @FilePath: \assess\backend\serve\exam_mgt\exam-mgt_test.go
 * @Description:
 * Copyright (c) 2025 by ${git_name}, All Rights Reserved.
 */
package exam_mgt

//annotation:exam_mgt
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"w2w.io/cmn"
	"w2w.io/exam_service"
	"w2w.io/null"
)

var (
	testPaperID                     = int64(99901)
	testPaperToPublishID            = int64(99902) // 用于测试发布考试的试卷
	testNormalExamID                = int64(99901)
	testDeleteExamID                = int64(99902)
	testNormalExamID2               = int64(99903)
	testExamToPublishID             = int64(99904) // 用于测试考试发布
	testErrorExamToPublishID1       = int64(99905) // 用于测试考试发布错误 - 时间不符合要求
	testEndExamID                   = int64(99906) // 已结束的考试
	testPublishedExamID             = int64(99907) // 已发布的考试
	testExamSessionID1              = int64(99901)
	testExamSessionID2              = int64(99902)
	testDeleteExamSessionID         = int64(99903)
	testExamSessionID3              = int64(99904)
	testExamSessionToPublishID1     = int64(99905) // 用于测试考试发布
	testExamSessionToPublishID2     = int64(99906) // 用于测试考试发布
	testExamSessionToPublishID3     = int64(99907) // 用于测试考试发布
	testExamSessionToPublishID4     = int64(99908) // 用于测试考试发布
	testExamSessionToPublishID5     = int64(99909) // 用于测试考试发布
	testErrorExamSessionToPublishID = int64(99910) // 用于测试考试发布错误 - 时间不符合要求
	testPublishedExamSessionID      = int64(99911) // 已发布的考试场次

	testAcademicAffair                       = int64(99901)
	testStudent1                             = int64(99902)
	testGrader                               = int64(99903) // 用于考试批阅员
	testExamSession1StartTime                = time.Now().Add(10 * time.Minute).UnixMilli()
	testExamSession1EndTime                  = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSession2StartTime                = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSession2EndTime                  = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionStartTime           = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionEndTime             = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID1StartTime     = time.Now().Add(10 * time.Minute).UnixMilli()
	testExamSessionToPublishID1EndTime       = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2StartTime     = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2EndTime       = time.Now().Add(30 * time.Minute).UnixMilli()
	testExamSessionToPublishID3StartTime     = time.Now().Add(30 * time.Minute).UnixMilli()
	testExamSessionToPublishID3EndTime       = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4StartTime     = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4EndTime       = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5StartTime     = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5EndTime       = time.Now().Add(60 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDStartTime = time.Now().Add(-10 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDEndTime   = time.Now().UnixMilli()
	BankQuestionIDs                          = []int64{10000001, 10000002, 10000003, 10000004, 10000005}
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
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
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
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
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
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
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
			ID:   null.NewInt(userID, true),   // 请求用户ID
			Role: null.NewInt(userRole, true), // 用户角色ID
		},
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 辅助函数：JSON序列化
func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}
	return data
}

// 辅助函数：生成指定长度的字符串
func generateLongString(length int) string {
	if length <= 0 {
		return ""
	}

	// 使用中文字符测试 rune 计算
	baseString := "这是一个测试字符串用于验证长度限制功能"
	baseLength := len([]rune(baseString))

	if length <= baseLength {
		return string([]rune(baseString)[:length])
	}

	// 如果需要更长的字符串，重复基础字符串
	var result strings.Builder
	result.Grow(length * 3) // 预分配空间，考虑中文字符的字节数

	for result.Len() < length*3 { // 估算字节数
		result.WriteString(baseString)
	}

	// 截取到确切的字符数
	resultRunes := []rune(result.String())
	if len(resultRunes) > length {
		resultRunes = resultRunes[:length]
	}

	return string(resultRunes)
}

// 生成一张测试试卷
func CreateTestPaperWithGroupsAndQuestions(ctx context.Context, tx pgx.Tx, bankQuestionIDs []int64, testUserID int64) (groupIDs []int64, questionIDs []int64, err error) {
	now := time.Now().UnixMilli()

	paperID := testPaperToPublishID

	// 创建试卷
	paper := &cmn.TPaper{
		Name:              null.StringFrom("Test Paper"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("00"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("Test Description"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           null.IntFrom(testUserID),
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         null.IntFrom(testUserID),
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
	}

	//初始化一张空试卷
	err = tx.QueryRow(ctx, `
		INSERT INTO t_paper 
			(id, name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, domain_id) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
		RETURNING id`,
		testPaperToPublishID,
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
		2000,
	).Scan(&paperID)

	if err != nil {
		return nil, nil, fmt.Errorf("创建试卷失败: %v", err)
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
			return nil, nil, fmt.Errorf("创建题组失败: %v", err)
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
					return groupIDs, nil, fmt.Errorf("创建试题失败: %v", err)
				}
				questionIDs = append(questionIDs, questionID)
			}
		}
	}

	return groupIDs, questionIDs, nil
}

func CreateTestExamData(t *testing.T) {

	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				err = tx.Commit(ctx)
			}
		}
	}()

	// 插入测试教务员数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role, status) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, testAcademicAffair, "sys^admin", "测试用户", "test_user", 2002, "00")
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 插入测试批阅员数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role, status) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, testGrader, "sys^admin", "测试批阅员", "test_grader", 2005, "00")
	if err != nil {
		t.Fatalf("创建测试批阅员失败: %v", err)
	}

	// 插入测试学生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role,status) 
		VALUES ($1, $2, $3, $4, $5,$6)
		ON CONFLICT (id) DO NOTHING`, testStudent1, "sys^student", "测试学生", "test_student", 2008, "00")
	if err != nil {
		t.Fatalf("创建测试学生失败: %v", err)
	}

	// 创建题库数据

	questions := []struct {
		id         int64
		qtype      string
		difficulty string
		creator    int64
		status     string
	}{
		{BankQuestionIDs[0], "00", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[1], "02", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[2], "04", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[3], "06", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[4], "08", "1", testAcademicAffair, "00"},
	}
	for _, q := range questions {
		_, err = tx.Exec(ctx, `
			INSERT INTO assessuser.t_question (id, type, difficulty, creator,create_time,updated_by,update_time,status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, q.id, q.qtype, q.difficulty, q.creator, time.Now().UnixMilli(), q.creator, time.Now().UnixMilli(), q.status)
		if err != nil {
			t.Fatalf("插入测试题目数据失败: %v", err)
		}
	}

	// 创建用于测试发布的试卷
	_, _, err = CreateTestPaperWithGroupsAndQuestions(ctx, tx, BankQuestionIDs, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("创建试卷失败: %v", err)
	}

	// 创建测试试卷
	_, err = tx.Exec(ctx, `
		INSERT INTO t_paper (id, name, category, creator, status, domain_id) 
		VALUES ($1, '测试试卷', '00', $2, '00', 2000) `, testPaperID, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("创建测试试卷失败: %v", err)
	}

	// 插入考试信息
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time, domain_id)
		VALUES ($1, '测试正常考试', '00', '00', '00', $2, $3, $2, $3, $4), 
		($5, '测试已删除的考试', '00', '00', '12', $2, $3, $2, $3, $4),
		($6, '测试正常考试2', '00', '00', '02', $2, $3, $2, $3, $4),
		($7, '测试发布考试', '00', '00', '00', $2, $3, $2, $3, $4),
		($8, '测试发布错误考试', '00', '00', '00', $2, $3, $2, $3, $4),
		($9, '测试已结束的考试', '00', '00', '06', $2, $3, $2, $3, $4),
		($10, '测试已发布的考试', '00', '00', '02', $2, $3, $2, $3, $4)
	`, testNormalExamID, testAcademicAffair, time.Now().UnixMilli(), 2002, testDeleteExamID,
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	var reviewerIDs []int64
	if testGrader > 0 {
		reviewerIDs = []int64{testGrader}
	}

	var nilReviewerIDs []int64
	nilReviewerIDs = make([]int64, 0)

	// 插入考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $19, '00', '00', 1, '02', $4, $5, $4, $5, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $19, '00', '00', 2, '02', $4, $5, $4, $5, $9, $10, '00', 10, '00'), 
		($11, $12, $3, $19, '00', '00', 3, '12', $3, $4, $3, $4, $13, $14, '00', 10, '00'),
		($15, $16, $3, $20, '00', '00', 4, '02', $3, $4, $3, $4, $17, $18, '00', 10, '00')
	`, testExamSessionID1, testNormalExamID, testPaperID, testAcademicAffair, time.Now().UnixMilli(),
		testExamSession1StartTime, testExamSession1EndTime, testExamSessionID2, testExamSession2StartTime, testExamSession2EndTime,
		testDeleteExamSessionID, testDeleteExamID, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime,
		testExamSessionID3, testNormalExamID2, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime, reviewerIDs, nilReviewerIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试场次数据失败: %v", err)
	}

	// 插入要发布的考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $23, '00', '00', 1, '00', $4, $5, $4, $5, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $23, '00', '00', 2, '00', $4, $5, $4, $5, $9, $10, '00', 10, '02'),
		($11, $2, $3, $23, '00', '00', 3, '00', $4, $5, $4, $5, $12, $13, '00', 10, '04'),
		($14, $2, $3, $24, '00', '00', 4, '00', $4, $5, $4, $5, $15, $16, '00', 10, '06'),
		($17, $2, $3, $23, '00', '00', 5, '00', $4, $5, $4, $5, $18, $19, '00', 10, '08'),
		($20, $25, $3, $23, '00', '00', 6, '00', $4, $5, $4, $5, $21, $22, '00', 10, '10'),
		($26, $27, $3, $23, '00', '00', 7, '02', $4, $5, $4, $5, $6, $7, '00', 10, '12')
	`, testExamSessionToPublishID1, testExamToPublishID, testPaperToPublishID, testAcademicAffair, time.Now().UnixMilli(), testExamSessionToPublishID1StartTime, testExamSessionToPublishID1EndTime,
		testExamSessionToPublishID2, testExamSessionToPublishID2StartTime, testExamSessionToPublishID2EndTime,
		testExamSessionToPublishID3, testExamSessionToPublishID3StartTime, testExamSessionToPublishID3EndTime,
		testExamSessionToPublishID4, testExamSessionToPublishID4StartTime, testExamSessionToPublishID4EndTime,
		testExamSessionToPublishID5, testExamSessionToPublishID5StartTime, testExamSessionToPublishID5EndTime,
		testErrorExamSessionToPublishID, testErrorExamSessionToPublishIDStartTime, testErrorExamSessionToPublishIDEndTime,
		reviewerIDs, nilReviewerIDs, testErrorExamToPublishID1, testPublishedExamSessionID, testPublishedExamID)

	// 插入考生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_examinee (exam_session_id, student_id, serial_number, status, creator, create_time)
		VALUES ($1, $2, 1, '00', $3, $4), 
		($5, $2, 2, '00', $3, $4),
		($6, $2, 3, '00', $3, $4),
		($7, $2, 1, '00', $3, $4),
		($8, $2, 1, '00', $3, $4),
		($9, $2, 1, '00', $3, $4),
		($10, $2, 1, '00', $3, $4)
	`, testExamSessionID1, testStudent1, testAcademicAffair, time.Now().UnixMilli(),
		testExamSessionID2, testExamSessionID3, testExamSessionToPublishID1, testExamSessionToPublishID2,
		testExamSessionToPublishID3, testExamSessionToPublishID4)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考生数据失败: %v", err)
	}

	return
}

func CleanTestExamData(t *testing.T) {
	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				tx.Commit(ctx)
			}
		}
	}()

	// 删除批改相关数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_mark_info WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试批改数据失败: %v", err)
	}

	// 删除答卷相关数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_student_answers WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试答卷数据失败: %v", err)
	}

	// 删除生成的考卷数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper_question WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷题目数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper_group WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷题组数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷数据失败: %v", err)
	}

	// 删除试卷1数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper WHERE id = $1
	`, testPaperID)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷数据失败: %v", err)
	}

	// 删除试卷数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper_question WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷题目数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper_group WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷题组数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷数据失败: %v", err)
	}

	// 删除题库数据
	_, err = tx.Exec(ctx, `
		DELETE FROM assessuser.t_question WHERE creator = $1
	`, testAcademicAffair)

	// 删除测试考生数据
	var testSessionIDs []int64
	testSessionIDs = append(testSessionIDs, testExamSessionID1, testExamSessionID2,
		testDeleteExamSessionID, testExamSessionID3, testExamSessionToPublishID1,
		testExamSessionToPublishID2, testExamSessionToPublishID3, testExamSessionToPublishID4, testExamSessionToPublishID5,
		testErrorExamSessionToPublishID, testPublishedExamSessionID)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_examinee WHERE exam_session_id = ANY($1)
	`, testSessionIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考生数据失败: %v", err)
	}

	// 删除测试考生数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_examinee WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考生数据失败: %v", err)
	}

	// 删除测试考试场次数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_session WHERE id = ANY($1)
	`, testSessionIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试场次数据失败: %v", err)
	}

	// 删除测试考试场次数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_session WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试场次数据失败: %v", err)
	}

	// 删除测试考试信息
	var testExamIDs []int64
	testExamIDs = append(testExamIDs, testNormalExamID, testDeleteExamID,
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_info WHERE id = ANY($1)
	`, testExamIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试信息失败: %v", err)
	}

	// 删除测试用户数据
	var testUserIDs []int64
	testUserIDs = append(testUserIDs, testAcademicAffair, testStudent1, testGrader)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_user WHERE id = ANY($1)
	`, testUserIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试用户数据失败: %v", err)
	}

}

func TestGenerateExamineeNumber(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber int64
		examInfo     cmn.TExamInfo
		examSessions []cmn.TExamSession
		expected     string
		description  string
	}{
		{
			name:         "线上考试模式-返回空字符串",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("00"), // 线上考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "",
			description: "线上考试不生成准考证号",
		},
		{
			name:         "线下考试模式-正常生成准考证号",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("02"), // 线下考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "24123000001",
			description: "24(年份) + 123(考试ID) + 000001(序号)",
		},
		{
			name:         "大序号测试-超过6位数",
			serialNumber: 1234567,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(789),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "257891234567",
			description: "序号超过6位时正常显示",
		},
		{
			name:         "小序号测试-需要补零",
			serialNumber: 5,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(1),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2023, 7, 20, 14, 30, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "231000005",
			description: "序号5 -> 000005",
		},
		{
			name:         "序号为0的情况",
			serialNumber: 0,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "25999000000",
			description: "序号0 -> 000000",
		},
		{
			name:         "大考试ID测试",
			serialNumber: 100,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2026, 3, 15, 16, 45, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "26999999000100",
			description: "大考试ID正常处理",
		},
		{
			name:         "空考试场次-使用当前时间",
			serialNumber: 50,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(777),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{},
			expected:     "",
			description:  "空场次时使用当前年份",
		},
		{
			name:         "无效的开始时间-使用当前时间",
			serialNumber: 25,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(888),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(0), // 无效时间戳
				},
			},
			expected:    "",
			description: "无效时间戳时使用当前年份",
		},
		{
			name:         "StartTime不Valid的情况",
			serialNumber: 10,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(555),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.Int{}, // StartTime不Valid
				},
			},
			expected:    "",
			description: "StartTime不Valid时使用当前年份",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExamineeNumber(tt.serialNumber, tt.examInfo, tt.examSessions)

			if tt.expected == "" {
				// 对于使用当前时间的情况，我们需要动态验证
				if tt.examInfo.Mode.String == "00" {
					// 线上考试应该返回空字符串
					if result != "" {
						t.Errorf("generateExamineeNumber() = %v, 期望空字符串", result)
					}
				} else if len(tt.examSessions) == 0 || !tt.examSessions[0].StartTime.Valid || tt.examSessions[0].StartTime.Int64 <= 0 {
					// 使用当前时间的情况，验证格式是否正确
					currentYear := time.Now().Year() % 100
					expectedPrefix := fmt.Sprintf("%02d%d", currentYear, tt.examInfo.ID.Int64)
					expectedSuffix := fmt.Sprintf("%06d", tt.serialNumber)
					expectedResult := expectedPrefix + expectedSuffix

					if result != expectedResult {
						t.Errorf("generateExamineeNumber() = %v, 期望 %v (使用当前年份 %d)", result, expectedResult, currentYear)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("generateExamineeNumber() = %v, 期望 %v (%s)", result, tt.expected, tt.description)
				}
			}
		})
	}
}

func TestValidateExamData(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name      string
		examData  ExamData
		isUpdate  bool
		wantError bool
		errorMsg  string
	}{
		{
			name: "有效的新建考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("期末考试"),
					Type: null.StringFrom("02"), // 期末成绩考试
					Mode: null.StringFrom("00"), // 线上考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",                  // 人工批卷
						PeriodMode:           null.StringFrom("00"), // 固定时段
						Duration:             null.IntFrom(120),     // 120分钟
						QuestionShuffledMode: null.StringFrom("00"), // 既有试题乱序也有选项乱序
						MarkMode:             null.StringFrom("00"), // 不需要手动批改
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(3 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "有效的更新考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(123),
					Name: null.StringFrom("期中考试"),
					Type: null.StringFrom("00"), // 平时考试
					Mode: null.StringFrom("02"), // 线下考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(2),
						MarkMethod:           "02",                  // 自动批卷
						PeriodMode:           null.StringFrom("02"), // 灵活时段
						Duration:             null.IntFrom(90),      // 90分钟
						QuestionShuffledMode: null.StringFrom("02"), // 选项乱序
						MarkMode:             null.StringFrom("02"), // 全卷多评
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(150 * time.Minute).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: false,
		},
		{
			name: "更新时考试ID无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(0), // 无效ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name: "考试名称为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom(""), // 空名称
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试名称不能为空",
		},
		{
			name: "无效的考试类型",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("99"), // 无效类型
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "无效的考试方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("99"), // 无效方式
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "考试场次为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{}, // 空场次
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次不能为空",
		},
		{
			name: "无效的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(0), // 无效试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "无效的批卷方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "99", // 无效批卷方式
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "考试时长无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(0), // 无效时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
		{
			name: "开始时间晚于当前时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(-2 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于当前时间",
		},
		{
			name: "开始时间晚于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "开始时间等于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()), // 开始时间等于结束时间
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "考试类型为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom(""), // 空字符串
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "考试方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom(""), // 空字符串
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "无效的时段模式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("99"), // 无效时段模式
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "时段模式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom(""), // 空字符串
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "无效的乱序方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("99"), // 无效乱序方式
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "乱序方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom(""), // 空字符串
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "无效的批改配置",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("99"), // 无效批改配置
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批改配置为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom(""), // 空字符串
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批卷方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "", // 空字符串
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "设定的考试时长大于总时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(180), // 180分钟，但总时长只有60分钟
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "设定的考试时长",
		},
		{
			name: "负数的考试ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(-1), // 负数ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name: "负数的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(-1), // 负数试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "负数的考试时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(-10), // 负数时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
		{
			name: "考试名称超过50个字符",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的考试名称1"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试名称过长",
		},
		{
			name: "考试名称正好50个字符（边界值测试）",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("这是一个测试考试名称这是一个测试考试名称这是一个测试考试名称这是一个测试考试名称这是十个字"), // 正好50个字符
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "考试规则超过5000个字符",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name:  null.StringFrom("考试"),
					Type:  null.StringFrom("00"),
					Mode:  null.StringFrom("00"),
					Rules: null.StringFrom(generateLongString(5001)), // 超过5000个字符
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试规则过长",
		},
		{
			name: "考试规则正好5000个字符（边界值测试）",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name:  null.StringFrom("考试"),
					Type:  null.StringFrom("00"),
					Mode:  null.StringFrom("00"),
					Rules: null.StringFrom(generateLongString(5000)), // 正好5000个字符
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "考试名称包含中英文混合（测试rune计算）",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("Final Exam期末考试2024年度测试Hello World你好世界"), // 中英文混合，正好50个字符
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "考试名称包含emoji字符（测试rune计算）",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("📚数学考试📝测试用例🎯"), // 包含emoji，测试Unicode字符计算
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "空的考试规则（Valid=false）",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name:  null.StringFrom("考试"),
					Type:  null.StringFrom("00"),
					Mode:  null.StringFrom("00"),
					Rules: null.String{}, // 空的规则，Valid=false
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(60),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false, // 空的规则应该是允许的
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExamData(tt.examData, tt.isUpdate)

			if tt.wantError {
				if err == nil {
					t.Errorf("validateExamData() 期望返回错误，但实际返回 nil")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateExamData() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateExamData() 期望返回 nil，但实际返回错误 = %v", err)
				}
			}
		})
	}
}

// cleanupTestData 清理测试过程中插入的数据
func cleanupTestExamData(t *testing.T, creators []int64) {
	if len(creators) == 0 {
		return
	}

	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// 删除考生记录
	for _, creator := range creators {
		_, err = tx.Exec(ctx, `
			DELETE FROM t_examinee 
			WHERE exam_session_id IN (
				SELECT id FROM t_exam_session WHERE creator = $1
			)`, creator)
		if err != nil {
			t.Logf("删除考生记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试场次记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试场次记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试信息记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试信息记录失败 (creator=%d): %v", creator, err)
		}
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Logf("提交清理事务失败: %v", err)
	} else {
		t.Logf("成功清理 %d 个测试考试的数据", len(creators))
	}
}

func createTestUser(t *testing.T, userID int64, role int64) {
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 创建用户
	_, err := conn.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, userID, "sys^admin", "测试用户", "test_user", role)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
}

func cleanTestUser(t *testing.T, userID int64) {
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 删除用户
	_, err := conn.Exec(ctx, `DELETE FROM t_user WHERE id = $1`, userID)
	if err != nil {
		t.Fatalf("删除测试用户失败: %v", err)
	}
}

// TestExamPostMethod 测试 exam 函数的 POST 方法（创建临时考试）
func TestExamPostMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	userID := int64(99999)
	var creators []int64
	creators = append(creators, userID)

	t.Cleanup(func() {
		cleanupTestExamData(t, creators)
		cleanTestUser(t, userID)
	})
	createTestUser(t, userID, 2002) // 创建一个测试用户，角色为2002（考试管理员）

	tests := []struct {
		name          string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
		userRole      int64
		checkResult   bool // 是否检查返回结果
	}{
		{
			name:          "成功创建临时考试-教务员权限",
			forceError:    "",
			expectedError: false,
			description:   "教务员角色成功创建临时考试",
			userID:        userID,
			userRole:      2002,
			checkResult:   true,
		},
		{
			name:          "成功创建临时考试-超级管理员权限",
			forceError:    "",
			expectedError: false,
			description:   "超级管理员角色成功创建临时考试",
			userID:        userID,
			userRole:      2001,
			checkResult:   true,
		},
		{
			name:          "成功创建临时考试-教师权限",
			forceError:    "",
			expectedError: false,
			description:   "教师角色成功创建临时考试",
			userID:        userID,
			userRole:      2003,
			checkResult:   true,
		},
		{
			name:          "权限不足-学生角色",
			forceError:    "",
			expectedError: true,
			errorContains: "用户没有创建考试的权限",
			description:   "学生角色不能创建考试",
			userID:        userID,
			userRole:      2008,
			checkResult:   false,
		},
		{
			name:          "权限不足-无权限角色",
			forceError:    "",
			expectedError: true,
			errorContains: "未找到角色ID",
			description:   "无权限角色不能创建考试",
			userID:        userID,
			userRole:      9999,
			checkResult:   false,
		},
		{
			name:          "无效用户ID-零值",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0时应返回错误",
			userID:        0,
			userRole:      2002,
			checkResult:   false,
		},
		{
			name:          "无效用户ID-负值",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为负值时应返回错误",
			userID:        -1,
			userRole:      2002,
			checkResult:   false,
		},
		{
			name:          "获取用户域失败",
			forceError:    "",
			expectedError: true,
			description:   "无法获取用户域时应返回错误",
			userID:        userID,
			userRole:      0, // 无效角色
			checkResult:   false,
		},
		{
			name:          "强制数据库插入错误",
			forceError:    "tx.QueryRow1",
			expectedError: true,
			errorContains: "强制查询错误",
			description:   "模拟数据库插入失败",
			userID:        userID,
			userRole:      2002,
			checkResult:   false,
		},
		{
			name:          "强制JSON序列化错误",
			forceError:    "json.Marshal",
			expectedError: true,
			errorContains: "强制json.Marshal错误",
			description:   "模拟JSON序列化失败",
			userID:        userID,
			userRole:      2002,
			checkResult:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文 - POST 方法不需要请求体
			queryParams := url.Values{}
			ctx := createMockContextWithRole("POST", "/api/exam", queryParams, tt.forceError, tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("exam() 意外panic: %v", r)
						}
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 检查返回结果
				if tt.checkResult {
					if serviceCtx.Msg.Data == nil {
						t.Errorf("exam() 期望返回数据，但数据为空")
						return
					}

					// 解析返回的考试ID
					var result struct {
						ID int64 `json:"id"`
					}
					if err := json.Unmarshal(serviceCtx.Msg.Data, &result); err != nil {
						t.Errorf("exam() 返回数据格式错误: %v", err)
						return
					}

					// 验证返回的ID有效
					if result.ID <= 0 {
						t.Errorf("exam() 返回的考试ID无效: %d", result.ID)
						return
					}

					// 验证数据库中确实创建了考试
					conn := cmn.GetPgxConn()
					var status string
					err := conn.QueryRow(context.Background(),
						"SELECT status FROM t_exam_info WHERE id = $1", result.ID).Scan(&status)
					if err != nil {
						t.Errorf("验证创建的考试失败: %v", err)
						return
					}

					// 验证状态为临时状态
					if status != "14" {
						t.Errorf("创建的考试状态错误，期望 '14'，实际 '%s'", status)
					}

					t.Logf("成功创建临时考试，ID: %d, 状态: %s", result.ID, status)
				}
			}
		})
	}
}

func TestExamGetMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 设置测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)

	// 清理函数
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		queryParams   string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
		userRole      int64
		mockValues    map[string]string
	}{
		{
			name:          "正常获取考试信息-教务员角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: false,
			description:   "教务员角色正常获取考试信息",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "正常获取考试信息-教务员角色 Query错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "conn.Query",
			expectedError: true,
			description:   "教务员角色正常获取考试信息 Query错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "正常获取考试信息-教务员角色 Scan错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "examinee_rows.Scan",
			expectedError: true,
			description:   "教务员角色正常获取考试信息 Scan错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "正常获取考试信息-学生角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: false,
			description:   "学生角色正常获取考试信息",
			userID:        testStudent1,
			userRole:      2008, // 学生角色
		},
		{
			name:          "获取考试信息-学生角色 JSON序列化错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "json.Marshal",
			expectedError: true,
			description:   "学生角色获取考试信息时JSON序列化错误",
			userID:        testStudent1,
			userRole:      2008, // 学生角色
		},
		{
			name:          "无效的考试ID-非数字",
			queryParams:   "exam_id=invalid",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为非数字时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无权限访问",
			queryParams:   "exam_id=99901",
			forceError:    "",
			expectedError: true,
			errorContains: "无权限访问",
			description:   "考试ID为99901时应返回无权限访问错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "validateUserExamPermission-false"},
		},
		{
			name:          "无效的考试ID-零值",
			queryParams:   "exam_id=0",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为0时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的考试ID-负值",
			queryParams:   "exam_id=-1",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为负值时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "缺少考试ID参数",
			queryParams:   "",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "缺少exam_id参数时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的用户ID",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID无效时应返回错误",
			userID:        0, // 无效用户ID
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的用户角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "未找到角色ID",
			description:   "用户角色无效时应返回错误",
			userID:        testAcademicAffair,
			userRole:      0, // 无效角色
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "不存在的考试ID",
			queryParams:   "exam_id=999999",
			forceError:    "",
			expectedError: true,
			errorContains: "考试不存在",
			description:   "查询不存在的考试ID时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "模拟GetExamInfo错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟GetExamInfo函数返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "GetExamInfo-error"},
		},
		{
			name:          "模拟GetExamSessions错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟GetExamSessions函数返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "GetExamSessions-error"},
		},
		{
			name:          "模拟JSON编码错误1",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "json.Marshal",
			expectedError: true,
			description:   "模拟JSON编码失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "模拟JSON编码错误2",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "json.Marshal",
			expectedError: true,
			description:   "模拟JSON编码失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "模拟权限验证失败",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟用户无权限访问考试",
			userID:        999,
			userRole:      2002,
			mockValues:    map[string]string{"test": "validateUserExamPermission-error"},
		},
		{
			name:          "验证考试是否存在失败",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟验证考试是否存在失败",
			userID:        999,
			userRole:      2002,
			mockValues:    map[string]string{"test": "examExists-error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析查询参数
			queryParams := url.Values{}
			if tt.queryParams != "" {
				parts := strings.Split(tt.queryParams, "&")
				for _, part := range parts {
					if kv := strings.Split(part, "="); len(kv) == 2 {
						queryParams.Set(kv[0], kv[1])
					}
				}
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole("GET", "/api/exam", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 设置mock值
			for key, value := range tt.mockValues {
				ctx = context.WithValue(ctx, key, value)
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("exam() 意外panic: %v", r)
						}
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 验证返回的数据
				if serviceCtx.Msg.Data == nil {
					t.Errorf("exam() 期望返回数据，但数据为空")
					return
				}

				// 尝试解析返回的JSON数据
				var examData ExamData
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examData); err != nil {
					t.Errorf("exam() 返回数据格式错误: %v", err)
					return
				}

				// 验证考试信息
				if examData.ExamInfo.ID.Int64 != testNormalExamID {
					t.Logf("exam() 返回的信息 %v", examData.ExamInfo)
					t.Errorf("exam() 返回的考试ID错误，期望 %d, 实际 %d", testNormalExamID, examData.ExamInfo.ID.Int64)
				}

				// 验证场次信息
				if len(examData.ExamSessions) == 0 {
					t.Errorf("exam() 期望返回场次信息，但为空")
				}
			}
		})
	}
}

// TestExamPutMethod 测试 exam 函数的 PUT 方法（更新考试信息）
func TestExamPutMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	go exam_service.ExamMaintainService()

	// 创建有效的考试数据模板
	validExamData := ExamData{
		ExamInfo: cmn.TExamInfo{
			ID:        null.IntFrom(testNormalExamID),
			Name:      null.StringFrom("更新的考试名称"),
			Rules:     null.StringFrom("更新的考试规则"),
			Type:      null.StringFrom("00"),
			Mode:      null.StringFrom("00"),
			Files:     types.JSONText(`{}`),
			Submitted: null.BoolFrom(false),
			Status:    null.StringFrom("00"),
			Addi:      types.JSONText(`{}`),
		},
		ExamSessions: []cmn.TExamSession{
			{
				SessionNum:           null.IntFrom(1),
				PaperID:              null.IntFrom(testPaperToPublishID),
				StartTime:            null.IntFrom(time.Now().Add(24 * time.Hour).UnixMilli()),
				EndTime:              null.IntFrom(time.Now().Add(25 * time.Hour).UnixMilli()),
				Duration:             null.IntFrom(60),
				QuestionShuffledMode: null.StringFrom("06"),
				NameVisibilityIn:     null.BoolFrom(true),
				MarkMethod:           "00",
				MarkMode:             null.StringFrom("10"),
				PeriodMode:           null.StringFrom("00"),
				LateEntryTime:        null.IntFrom(10),
				EarlySubmissionTime:  null.IntFrom(5),
				ReviewerIds:          []int64{testGrader},
			},
		},
		ExamineeIDs: []int64{testStudent1},
	}

	tests := []struct {
		name           string
		description    string
		userID         int64
		userRole       int64
		requestBodyGen func() interface{} // 生成请求体的函数
		forceError     string
		expectedError  bool
		errorContains  string
		checkResult    bool // 是否检查更新结果
	}{
		{
			name:        "成功更新未发布考试",
			description: "成功更新处于未发布状态的考试",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			expectedError: false,
			checkResult:   true,
		},
		{
			name:        "成功更新临时考试状态为14",
			description: "成功更新处于临时状态(14)的考试，应同时更新creator和create_time",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				// 先创建一个临时考试
				conn := cmn.GetPgxConn()
				ctx := context.Background()
				var tempExamID int64

				err := conn.QueryRow(ctx, `
					INSERT INTO t_exam_info (
						creator, create_time, updated_by, update_time, status, domain_id
					) VALUES (
						$1, $2, $3, $4, $5, $6
					) RETURNING id
				`, int64(99999), time.Now().UnixMilli(), int64(99999), time.Now().UnixMilli(), "14", int64(2000)).Scan(&tempExamID)

				if err != nil {
					t.Fatalf("创建临时考试失败: %v", err)
				}

				data := validExamData
				data.ExamInfo.ID = null.IntFrom(tempExamID)
				return data
			},
			expectedError: false,
			checkResult:   true,
		},
		{
			name:        "成功更新已发布考试",
			description: "成功更新处于已发布状态的考试",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				data.ExamSessions[0].PaperID = null.IntFrom(testPaperToPublishID)
				return data
			},
			expectedError: false,
			checkResult:   false, // 已发布考试逻辑复杂，暂不检查详细结果
		},
		{
			name:        "权限验证失败-学生角色",
			description: "学生角色不能更新考试",
			userID:      testStudent1,
			userRole:    2008,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			expectedError: true,
			errorContains: "用户没有考试相关的权限",
			checkResult:   false,
		},
		{
			name:        "考试数据验证失败",
			description: "考试数据不符合验证规则",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(0) // 无效的考试ID
				return data
			},
			expectedError: true,
			errorContains: "",
			checkResult:   false,
		},
		{
			name:        "考试不存在",
			description: "更新不存在的考试",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(999999)
				return data
			},
			expectedError: true,
			errorContains: "考试不存在",
			checkResult:   false,
		},
		{
			name:        "考试状态不允许更新",
			description: "已结束的考试不能更新",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testEndExamID)
				return data
			},
			expectedError: true,
			errorContains: "当前考试状态不允许更新",
			checkResult:   false,
		},
		{
			name:           "空请求体",
			description:    "请求体为空",
			userID:         testAcademicAffair,
			userRole:       2002,
			requestBodyGen: func() interface{} { return "" },
			expectedError:  true,
			errorContains:  "请求体为空",
			checkResult:    false,
		},
		{
			name:        "无效用户ID",
			description: "用户ID为0",
			userID:      0,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			expectedError: true,
			errorContains: "无效的用户ID",
			checkResult:   false,
		},
		{
			name:        "无效用户角色",
			description: "用户角色无效",
			userID:      testAcademicAffair,
			userRole:    0,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			expectedError: true,
			checkResult:   false,
		},
		// 强制错误测试用例
		{
			name:        "强制IO读取错误",
			description: "模拟IO读取错误",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			forceError:    "io.ReadAll",
			expectedError: true,
			errorContains: "强制读取请求体错误",
			checkResult:   false,
		},
		{
			name:        "强制IO关闭错误",
			description: "模拟IO关闭错误",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "io.Close",
			expectedError: false,
			checkResult:   false,
		},
		{
			name:        "强制JSON解析错误1",
			description: "模拟第一次JSON解析失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			forceError:    "json.Unmarshal",
			expectedError: true,
			errorContains: "强制JSON解析错误",
			checkResult:   false,
		},
		{
			name:        "强制JSON解析错误2",
			description: "模拟第二次JSON解析失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				return validExamData
			},
			forceError:    "json.Unmarshal2",
			expectedError: true,
			errorContains: "强制第二次JSON解析错误",
			checkResult:   false,
		},
		{
			name:        "强制考试存在检查错误",
			description: "模拟考试存在检查失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "examExists",
			expectedError: true,
			errorContains: "强制检查考试存在错误",
			checkResult:   false,
		},
		{
			name:        "强制查询当前状态错误",
			description: "模拟查询当前考试状态失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "conn.QueryRow",
			expectedError: true,
			errorContains: "强制查询当前考试状态错误",
			checkResult:   false,
		},
		{
			name:        "强制获取旧考试场次ID错误",
			description: "模拟获取旧考试场次ID失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "getExamSessionIDs",
			expectedError: true,
			errorContains: "强制获取旧考试场次ID错误",
			checkResult:   false,
		},
		{
			name:        "强制事务开启错误",
			description: "模拟事务开启失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.Begin",
			expectedError: true,
			errorContains: "强制开启事务错误",
			checkResult:   false,
		},
		{
			name:        "强制事务回滚错误",
			description: "模拟事务回滚失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.Rollback",
			expectedError: false,
			checkResult:   false,
		},
		{
			name:        "强制事务提交错误",
			description: "模拟事务提交失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.Commit",
			expectedError: false,
			checkResult:   false,
		},
		{
			name:        "强制更新考试信息错误",
			description: "模拟更新考试信息失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.UpdateExamInfo",
			expectedError: true,
			errorContains: "强制更新考试信息错误",
			checkResult:   false,
		},
		{
			name:        "强制软删除考试场次错误",
			description: "模拟软删除考试场次失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.SoftDeleteExamSessions",
			expectedError: true,
			errorContains: "强制删除考试场次错误",
			checkResult:   false,
		},
		{
			name:        "强制软删除考生错误",
			description: "模拟软删除考生失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.SoftDeleteExaminee",
			expectedError: true,
			errorContains: "强制删除考生错误",
			checkResult:   false,
		},
		{
			name:        "强制插入考试场次错误",
			description: "模拟插入考试场次失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.QueryExamSession",
			expectedError: true,
			errorContains: "强制查询错误",
			checkResult:   false,
		},
		{
			name:        "强制批量插入考生错误",
			description: "模拟批量插入考生失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testExamToPublishID)
				return data
			},
			forceError:    "tx.InsertExaminees",
			expectedError: true,
			errorContains: "强制执行批量插入考生错误",
			checkResult:   false,
		},
		// 已发布考试的特殊错误测试
		{
			name:        "已发布考试-强制删除考卷错误",
			description: "更新已发布考试时删除考卷失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "examPaper.DeleteExamPaperById",
			expectedError: true,
			errorContains: "强制删除考卷和答卷错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制处理批改信息错误",
			description: "更新已发布考试时处理批改信息失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "mark.HandleMarkerInfo",
			expectedError: true,
			errorContains: "强制处理批改信息错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制查询考生错误",
			description: "更新已发布考试时查询考生失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "tx.SearchExaminee",
			expectedError: true,
			errorContains: "查询考生失败",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制生成考卷错误",
			description: "更新已发布考试时生成考卷失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "examPaper.GenerateExamPaper",
			expectedError: true,
			errorContains: "强制生成考卷错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制生成答卷错误",
			description: "更新已发布考试时生成答卷失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "examPaper.GenerateAnswerQuestion",
			expectedError: true,
			errorContains: "强制生成答卷错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制更新考生考卷ID错误",
			description: "更新已发布考试时更新考生考卷ID失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "tx.UpdateExamineeExamPaperID",
			expectedError: true,
			errorContains: "强制更新考生考卷ID错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制转换批改员ID错误",
			description: "更新已发布考试时转换批改员ID失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "convertToInt64Array",
			expectedError: true,
			errorContains: "转换批改员ID失败",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制查询考试创建者错误",
			description: "更新已发布考试时查询创建者失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "tx.SearchExamCreator",
			expectedError: true,
			errorContains: "强制查询考试创建者错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制处理批改员信息错误2",
			description: "更新已发布考试时处理批改员信息失败2",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "mark.HandleMarkerInfo2",
			expectedError: true,
			errorContains: "强制处理批改员信息错误",
			checkResult:   false,
		},
		{
			name:        "已发布考试-强制设置考试计时器错误",
			description: "更新已发布考试时设置计时器失败",
			userID:      testAcademicAffair,
			userRole:    2002,
			requestBodyGen: func() interface{} {
				data := validExamData
				data.ExamInfo.ID = null.IntFrom(testPublishedExamID)
				return data
			},
			forceError:    "exam_service.SetExamTimers",
			expectedError: true,
			errorContains: "强制设置考试计时器错误",
			checkResult:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成请求体
			requestBody := tt.requestBodyGen()
			var requestBodyStr string

			switch body := requestBody.(type) {
			case string:
				requestBodyStr = body
			case ExamData:
				requestBodyStr = string(mustMarshal(t, body))
			default:
				t.Fatalf("不支持的请求体类型: %T", requestBody)
			}

			// 创建模拟上下文
			ctx := createMockContextWithBody("PUT", "/api/exam", requestBodyStr, tt.forceError, tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("exam() 意外panic: %v", r)
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 检查更新结果
				if tt.checkResult {
					// 验证考试信息确实被更新
					conn := cmn.GetPgxConn()
					var examInfo struct {
						Name    string
						Status  string
						Creator int64
					}

					examID := requestBody.(ExamData).ExamInfo.ID.Int64
					err := conn.QueryRow(context.Background(),
						"SELECT name, status, creator FROM t_exam_info WHERE id = $1", examID).Scan(
						&examInfo.Name, &examInfo.Status, &examInfo.Creator)

					if err != nil {
						t.Errorf("验证更新的考试失败: %v", err)
						return
					}

					t.Logf("考试更新成功，ID: %d, 名称: %s, 状态: %s, 创建者: %d",
						examID, examInfo.Name, examInfo.Status, examInfo.Creator)
				}
			}
		})
	}
}

// TestExamUnsupportedMethod 测试 exam 函数的不支持方法（default case）
func TestExamUnsupportedMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name          string
		method        string
		description   string
		userID        int64
		userRole      int64
		expectedError bool
		errorContains string
	}{
		{
			name:          "OPTIONS方法-不支持",
			method:        "OPTIONS",
			description:   "OPTIONS方法应该返回不支持的错误",
			userID:        1,
			userRole:      2002,
			expectedError: true,
			errorContains: "unsupported method: options",
		},
		{
			name:          "HEAD方法-不支持",
			method:        "HEAD",
			description:   "HEAD方法应该返回不支持的错误",
			userID:        1,
			userRole:      2002,
			expectedError: true,
			errorContains: "unsupported method: head",
		},
		{
			name:          "自定义方法-不支持",
			method:        "CUSTOM",
			description:   "自定义方法应该返回不支持的错误",
			userID:        1,
			userRole:      2002,
			expectedError: true,
			errorContains: "unsupported method: custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文
			queryParams := url.Values{}
			ctx := createMockContextWithRole(tt.method, "/api/exam", queryParams, "", tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("exam() 意外panic: %v", r)
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}
		})
	}
}

func TestValidateUserExamPermission(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据 - 需要先创建测试用的考试和考生数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的考试ID列表
	var testExamIDs []int64
	var testUserIDs []int64
	var testSessionIDs []int64

	// 清理函数
	defer func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Logf("开始清理事务失败: %v", err)
			return
		}
		defer tx.Rollback(ctx)

		// 清理考生记录
		for _, sessionID := range testSessionIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_examinee WHERE exam_session_id = $1`, sessionID)
			if err != nil {
				t.Logf("清理考生记录失败: %v", err)
			}
		}

		// 清理考试场次
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE exam_id = $1`, examID)
			if err != nil {
				t.Logf("清理考试场次失败: %v", err)
			}
		}

		// 清理考试记录
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败: %v", err)
			}
		}

		// 清理用户记录
		for _, userID := range testUserIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_user WHERE id = $1`, userID)
			if err != nil {
				t.Logf("清理用户记录失败: %v", err)
			}
		}

		err = tx.Commit(ctx)
		if err != nil {
			t.Logf("提交清理事务失败: %v", err)
		}
	}()

	// 创建测试用户
	testTeacherID := int64(99001)
	testAcademicAffairID := int64(99004)
	testStudentID := int64(99002)
	testAdminID := int64(99003)
	testUserIDs = append(testUserIDs, testTeacherID, testStudentID, testAdminID, testAcademicAffairID)

	currentTime := time.Now().UnixMilli()

	// 插入测试用户
	_, err := conn.Exec(ctx, `
		INSERT INTO t_user (id, role, account, category, official_name, create_time, update_time, status) 
		VALUES 
			($1, 2003, 'test_teacher', '00', '测试教师', $5, $5, '00'),
			($2, 2008, 'test_student', '00', '测试学生', $5, $5, '00'),
			($3, 2001, 'test_admin', '00', '测试管理员', $5, $5, '00'),
			($4, 2002, 'test_academic_affair', '00', '测试教务', $5, $5, '00')
		ON CONFLICT (id) DO NOTHING`,
		testTeacherID, testStudentID, testAdminID, testAcademicAffairID, currentTime)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 创建测试考试（由testTeacherID创建）
	var testExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, type, mode, creator, create_time, updated_by, update_time, status, domain_id
		) VALUES (
			'测试权限考试', '00', '00', $1, $2, $1, $2, '02', 2002
		) RETURNING id`,
		testTeacherID, currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建测试考试场次
	var testSessionID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 1, 123, $2, $3, 120, '00', '00', '00', '00', '02', $4, $2, $4, $2
		) RETURNING id`,
		testExamID, currentTime, currentTime+7200000, testTeacherID).Scan(&testSessionID)
	if err != nil {
		t.Fatalf("创建测试考试场次失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID)

	// 创建考生记录
	_, err = conn.Exec(ctx, `
		INSERT INTO t_examinee (
			student_id, exam_session_id, creator, create_time, updated_by, update_time, 
			status, addi, serial_number, examinee_number
		) VALUES (
			$1, $2, $3, $4, $3, $4, '00', '{}', 1, '24001000001'
		)`,
		testStudentID, testSessionID, testTeacherID, currentTime)
	if err != nil {
		t.Fatalf("创建考生记录失败: %v", err)
	}

	tests := []struct {
		name        string
		userID      int64
		examID      int64
		domain      string
		wantResult  bool
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		mockValue   string
	}{
		{
			name:        "管理员权限-应该有权限",
			userID:      testAdminID,
			examID:      testExamID,
			domain:      "cst.school^admin", // 管理员角色
			wantResult:  true,
			wantError:   false,
			description: "管理员对所有考试都有权限",
			forceError:  "",
		},
		{
			name:        "教务员权限QueryRow 错误",
			userID:      testAcademicAffairID,
			examID:      testExamID,
			domain:      "cst.school^academicAffair^admin", // 教务员角色
			wantResult:  false,
			wantError:   true,
			description: "教务员对自己创建的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "教务员权限",
			userID:      testAcademicAffairID,
			examID:      testExamID,
			domain:      "cst.school^academicAffair^admin",
			wantResult:  true,
			wantError:   false,
			description: "教务员对自己创建的考试有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试",
			userID:      testStudentID,
			examID:      testExamID,
			domain:      "cst.school^student", // 学生角色
			wantResult:  true,
			wantError:   false,
			description: "学生对自己参加的考试有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试 QueryRow 错误",
			userID:      testStudentID,
			examID:      testExamID,
			domain:      "cst.school^student", // 学生角色
			wantResult:  false,
			wantError:   true,
			description: "学生对自己参加的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "无效的用户ID",
			userID:      0,
			examID:      testExamID,
			domain:      "cst.school^academicAffair^admin",
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "用户ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数用户ID",
			userID:      -1,
			examID:      testExamID,
			domain:      "cst.school^academicAffair^admin",
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数用户ID应该返回错误",
			forceError:  "",
		},
		{
			name:        "无效的考试ID",
			userID:      testStudentID,
			examID:      0,
			domain:      "cst.school^student",
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "考试ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数考试ID",
			userID:      testStudentID,
			examID:      -1,
			domain:      "cst.school^student",
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数考试ID应该返回错误",
			forceError:  "",
		},
		// Mock测试用例
		{
			name:        "Mock-normal-resp",
			userID:      testStudentID,
			examID:      testExamID,
			domain:      "cst.school^student",
			wantResult:  true,
			wantError:   false,
			description: "Mock normal-resp应该返回true, nil",
			forceError:  "",
			mockValue:   "normal-resp",
		},
		{
			name:        "Mock-validateUserExamPermission-false",
			userID:      testStudentID,
			examID:      testExamID,
			domain:      "cst.school^student",
			wantResult:  false,
			wantError:   false,
			description: "Mock validateUserExamPermission-false应该返回false, nil",
			forceError:  "",
			mockValue:   "validateUserExamPermission-false",
		},
		{
			name:        "Mock-validateUserExamPermission-error",
			userID:      testStudentID,
			examID:      testExamID,
			domain:      "cst.school^student",
			wantResult:  false,
			wantError:   true,
			errorMsg:    "validateUserExamPermission error",
			description: "Mock validateUserExamPermission-error应该返回false, error",
			forceError:  "",
			mockValue:   "validateUserExamPermission-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// 设置错误注入
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.forceError)
			}
			// 设置mock值
			if tt.mockValue != "" {
				ctx = context.WithValue(ctx, "test", tt.mockValue)
			}

			result, err := validateUserExamPermission(ctx, tt.userID, tt.examID, tt.domain)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("validateUserExamPermission() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateUserExamPermission() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateUserExamPermission() 期望没有错误，但返回错误: %v", err)
					return
				}
			}

			// 检查结果
			if result != tt.wantResult {
				t.Errorf("validateUserExamPermission() = %v, 期望 %v (%s)", result, tt.wantResult, tt.description)
			}
		})
	}
}

func TestGetExamInfo(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	ctx := context.Background()

	CleanTestExamData(t)
	CreateTestExamData(t)

	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name        string
		examID      int64
		domain      string
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, cmn.TExamInfo, error)
	}{
		{
			name:        "管理员角色-正常获取考试信息",
			examID:      testNormalExamID,
			domain:      "cst.school^admin", // 管理员
			wantError:   false,
			description: "管理员获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testNormalExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testNormalExamID)
				}
				if !examInfo.Name.Valid || examInfo.Name.String != "测试正常考试" {
					t.Errorf("考试名称不匹配: got %v, want '测试正常考试'", examInfo.Name)
				}
				// 管理员应该能看到完整信息包括creator等字段
				if !examInfo.Creator.Valid {
					t.Errorf("管理员应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教务员角色-正常获取考试信息",
			examID:      testNormalExamID,
			domain:      "cst.school.academicAffair^admin", // 教务员
			wantError:   false,
			description: "教务员获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testNormalExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testNormalExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教务员角色-正常获取考试信息 Scan错误",
			examID:      testNormalExamID,
			domain:      "cst.school.academicAffair^admin",
			wantError:   true,
			description: "教务员角色-正常获取考试信息 Scan错误",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testNormalExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testNormalExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-只获取部分考试信息",
			examID:      testNormalExamID,
			domain:      "cst.school^student", // 学生
			wantError:   false,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
		},
		{
			name:        "学生角色-只获取部分考试信息 Scan错误",
			examID:      testNormalExamID,
			domain:      "cst.school^student", // 学生
			wantError:   true,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			domain:      "cst.school^student",
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			domain:      "cst.school^student",
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			domain:      "cst.school^student",
			wantError:   true,
			description: "不存在的考试ID应该返回sql.ErrNoRows",
		},
		{
			name:        "已删除的考试",
			examID:      testDeleteExamID,
			domain:      "cst.school.academicAffair^admin",
			wantError:   true,
			description: "已删除的考试(status='12')应该查询不到",
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != 1 {
					t.Errorf("Mock考试ID不匹配: got %v, want 1", examInfo.ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "GetExamInfo-error",
			wantError:   true,
			errorMsg:    "GetExamInfo error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamInfo(testCtx, tt.examID, tt.domain)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamInfo() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamInfo() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamInfo() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestGetExamSessions(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	ctx := context.Background()

	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name        string
		examID      int64
		domain      string
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, []cmn.TExamSession, error)
	}{
		{
			name:        "教务员角色-获取完整场次信息",
			examID:      testNormalExamID,
			domain:      "cst.school.academicAffair^admin",
			wantError:   false,
			description: "教务员获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 教师也应该能看到完整信息
				session1 := sessions[0]
				if !session1.PaperName.Valid {
					t.Errorf("教师应该能看到试卷名称")
				}
			},
			forceError: "",
		},
		{
			name:        "教务员角色-获取完整场次信息 Query错误",
			examID:      testNormalExamID,
			domain:      "cst.school.academicAffair^admin",
			wantError:   true,
			description: "教务员获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "教务员角色-获取完整场次信息 Scan错误",
			examID:      testNormalExamID,
			domain:      "cst.school.academicAffair^admin",
			wantError:   true,
			description: "教务员获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-获取基本场次信息",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			wantError:   false,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 学生不应该看到试卷名称等敏感信息
				session1 := sessions[0]
				if session1.PaperName.Valid {
					t.Errorf("学生不应该看到试卷名称，但获取到了: %v", session1.PaperName)
				}
				// 但应该能看到基本信息
				if !session1.StartTime.Valid {
					t.Errorf("学生应该能看到开始时间")
				}
			},
			forceError: "",
		},
		{
			name:        "学生角色-获取基本场次信息 Query错误",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "学生角色-获取基本场次信息 Scan错误",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			domain:      "cst.school^student",
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			domain:      "cst.school^student",
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			domain:      "cst.school^student",
			wantError:   false,
			description: "不存在的考试ID应该返回空数组",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 0 {
					t.Errorf("不存在的考试应该返回空数组: got %d sessions", len(sessions))
				}
			},
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 1 {
					t.Errorf("Mock应该返回1个场次: got %d", len(sessions))
					return
				}
				if !sessions[0].ID.Valid || sessions[0].ID.Int64 != 10001 {
					t.Errorf("Mock场次ID不匹配: got %v, want 10001", sessions[0].ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "GetExamSessions-error",
			wantError:   true,
			errorMsg:    "GetExamSessions error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testNormalExamID,
			domain:      "cst.school^student",
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamSessions(testCtx, tt.domain, tt.examID)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamSessions() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamSessions() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamSessions() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestUpdateExamStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	// 创建基础上下文
	ctx := context.Background()

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			// 测试结束后回滚事务
			tx.Rollback(ctx)
		}
	}()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)

	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name         string
		examIDs      []int64
		newStatus    string
		userID       int64
		forceError   string
		wantError    bool
		errorMsg     string
		shouldVerify bool
	}{
		{
			name:         "正常更新单个考试状态-草稿到发布",
			examIDs:      []int64{testNormalExamID},
			newStatus:    "02",
			userID:       testAcademicAffair,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "正常更新多个考试状态-发布到进行中",
			examIDs:      []int64{testNormalExamID, testNormalExamID2},
			newStatus:    "04",
			userID:       testAcademicAffair,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:      "空的考试ID数组",
			examIDs:   []int64{},
			newStatus: "02",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "考试ID数组不能为空",
		},
		{
			name:      "包含无效考试ID的数组-零值",
			examIDs:   []int64{testNormalExamID, 0},
			newStatus: "02",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "包含无效考试ID的数组-负值",
			examIDs:   []int64{testNormalExamID, -1},
			newStatus: "02",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的用户ID-零值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "02",
			userID:    0,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "无效的用户ID-负值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "02",
			userID:    -1,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "空的状态值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "更新状态不能为空",
		},
		{
			name:       "数据库执行错误",
			examIDs:    []int64{testNormalExamID},
			newStatus:  "01",
			userID:     testAcademicAffair,
			forceError: "tx.Exec",
			wantError:  true,
			errorMsg:   "force error",
		},
		{
			name:      "不存在的考试ID",
			examIDs:   []int64{999999},
			newStatus: "01",
			userID:    testAcademicAffair,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前的状态
			var originalStatuses map[int64]string
			if tt.shouldVerify && !tt.wantError && len(tt.examIDs) > 0 {
				originalStatuses = make(map[int64]string)
				for _, examID := range tt.examIDs {
					var originalStatus string
					err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", examID).Scan(&originalStatus)
					if err != nil {
						t.Fatalf("获取更新前状态失败: %v", err)
					}
					originalStatuses[examID] = originalStatus
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamStatus(testCtx, tx, tt.newStatus, tt.userID, tt.examIDs...)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify && len(tt.examIDs) > 0 {
					for _, examID := range tt.examIDs {
						var currentStatus string
						err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", examID).Scan(&currentStatus)
						if err != nil {
							// 对于不存在的考试ID，忽略错误
							if examID == 999999 {
								continue
							}
							t.Errorf("验证更新结果失败: %v", err)
							return
						}

						if currentStatus != tt.newStatus {
							t.Errorf("updateExamStatus() 考试ID %d 状态更新失败，期望状态 = %v, 实际状态 = %v", examID, tt.newStatus, currentStatus)
						}

						// 验证 update_time 和 updated_by 字段
						var updatedBy int64
						var updateTime int64
						err = tx.QueryRow(ctx, "SELECT updated_by, update_time FROM t_exam_info WHERE id = $1", examID).Scan(&updatedBy, &updateTime)
						if err != nil {
							t.Errorf("验证更新字段失败: %v", err)
						} else {
							if updatedBy != tt.userID {
								t.Errorf("updateExamStatus() 考试ID %d updated_by 字段错误，期望 = %v, 实际 = %v", examID, tt.userID, updatedBy)
							}
							// 验证 update_time 是最近更新的（容忍1分钟误差）
							if time.Since(time.UnixMilli(updateTime)) > time.Minute {
								t.Errorf("updateExamStatus() 考试ID %d update_time 字段未正确更新，时间 = %v", examID, updateTime)
							}
						}
					}
				}
			}
		})
	}
}

// TestUpdateExamSessionStatus 测试 updateExamSessionStatus 函数
func TestUpdateExamSessionStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	ctx := context.Background()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			// 测试结束后回滚事务
			tx.Rollback(ctx)
		}
	}()

	tests := []struct {
		name         string
		examIDs      []int64
		newStatus    string
		userID       int64
		forceError   string
		wantError    bool
		errorMsg     string
		shouldVerify bool
	}{
		{
			name:         "正常更新单个考试场次状态-待开始到进行中",
			examIDs:      []int64{testNormalExamID},
			newStatus:    "04",
			userID:       testAcademicAffair,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "正常更新多个考试场次状态-进行中到已结束",
			examIDs:      []int64{testNormalExamID, testNormalExamID2},
			newStatus:    "06",
			userID:       testAcademicAffair,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:      "空的考试ID数组",
			examIDs:   []int64{},
			newStatus: "02",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "考试ID数组不能为空",
		},
		{
			name:      "包含无效考试ID的数组-零值",
			examIDs:   []int64{testNormalExamID, 0},
			newStatus: "02",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "包含无效考试ID的数组-负值",
			examIDs:   []int64{testNormalExamID, -1},
			newStatus: "01",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的用户ID-零值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "01",
			userID:    0,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "无效的用户ID-负值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "01",
			userID:    -1,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "空的状态值",
			examIDs:   []int64{testNormalExamID},
			newStatus: "",
			userID:    testAcademicAffair,
			wantError: true,
			errorMsg:  "更新状态不能为空",
		},
		{
			name:       "数据库执行错误",
			examIDs:    []int64{testNormalExamID},
			newStatus:  "01",
			userID:     testAcademicAffair,
			forceError: "tx.Exec",
			wantError:  true,
			errorMsg:   "force error",
		},
		{
			name:      "不存在的考试场次ID",
			examIDs:   []int64{999999},
			newStatus: "01",
			userID:    testAcademicAffair,
			wantError: false, // SQL执行成功但影响行数为0
		},
		{
			name:         "更新为作废状态",
			examIDs:      []int64{testNormalExamID},
			newStatus:    "00",
			userID:       testAcademicAffair,
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "批量更新包含存在和不存在的考试ID",
			examIDs:      []int64{testNormalExamID, 999999},
			newStatus:    "02",
			userID:       testAcademicAffair,
			wantError:    false,
			shouldVerify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前所有场次的状态
			var originalStatusesMap map[int64][]string
			if tt.shouldVerify && !tt.wantError && len(tt.examIDs) > 0 {
				originalStatusesMap = make(map[int64][]string)
				for _, examID := range tt.examIDs {
					var originalStatuses []string
					rows, err := tx.Query(ctx, "SELECT status FROM t_exam_session WHERE exam_id = $1 ORDER BY session_num", examID)
					if err != nil {
						// 对于不存在的考试ID，忽略错误
						if examID == 999999 {
							continue
						}
						t.Fatalf("获取更新前状态失败: %v", err)
					}
					defer rows.Close()

					for rows.Next() {
						var status string
						if err := rows.Scan(&status); err != nil {
							t.Fatalf("扫描原始状态失败: %v", err)
						}
						originalStatuses = append(originalStatuses, status)
					}
					originalStatusesMap[examID] = originalStatuses
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamSessionStatus(testCtx, tx, tt.newStatus, tt.userID, tt.examIDs...)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamSessionStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamSessionStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamSessionStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify && len(tt.examIDs) > 0 {
					for _, examID := range tt.examIDs {
						// 检查该考试ID的所有场次状态是否都被更新
						rows, err := tx.Query(ctx, "SELECT status, updated_by, update_time FROM t_exam_session WHERE exam_id = $1 ORDER BY session_num", examID)
						if err != nil {
							// 对于不存在的考试ID，忽略错误
							if examID == 999999 {
								continue
							}
							t.Errorf("验证更新结果查询失败: %v", err)
							return
						}
						defer rows.Close()

						var sessionCount int
						originalStatuses := originalStatusesMap[examID]
						for rows.Next() {
							var currentStatus string
							var updatedBy int64
							var updateTime int64

							if err := rows.Scan(&currentStatus, &updatedBy, &updateTime); err != nil {
								t.Errorf("验证更新结果扫描失败: %v", err)
								return
							}

							sessionCount++

							// 验证状态是否正确更新
							if currentStatus != tt.newStatus {
								t.Errorf("updateExamSessionStatus() 考试ID %d 场次%d状态更新失败，期望状态 = %v, 实际状态 = %v", examID, sessionCount, tt.newStatus, currentStatus)
							}

							// 验证 updated_by 字段
							if updatedBy != tt.userID {
								t.Errorf("updateExamSessionStatus() 考试ID %d 场次%d updated_by 字段错误，期望 = %v, 实际 = %v", examID, sessionCount, tt.userID, updatedBy)
							}

							// 验证 update_time 是最近更新的（容忍1分钟误差）
							if time.Since(time.UnixMilli(updateTime)) > time.Minute {
								t.Errorf("updateExamSessionStatus() 考试ID %d 场次%d update_time 字段未正确更新，时间 = %v", examID, sessionCount, updateTime)
							}
						}

						// 验证是否更新了所有场次
						if len(originalStatuses) > 0 && sessionCount != len(originalStatuses) {
							t.Errorf("updateExamSessionStatus() 考试ID %d 场次数量不匹配，期望更新 %d 个场次，实际更新了 %d 个", examID, len(originalStatuses), sessionCount)
						}
					}
				}
			}
		})
	}
}

// TestExamList 测试 examList 函数
func TestExamList(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		method        string
		queryParams   string
		userID        int64
		userRole      int64
		forceError    string
		expectedError bool
		errorContains string
		description   string
		checkResult   func(t *testing.T, serviceCtx *cmn.ServiceCtx)
	}{
		{
			name:          "教务员角色-默认查询",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "教务员角色使用默认查询参数获取考试列表",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(examList) == 0 {
					t.Error("期望返回考试列表，但为空")
				}
				if serviceCtx.Msg.RowCount <= 0 {
					t.Error("期望返回行数大于0")
				}
			},
		},
		{
			name:          "教务员角色-默认查询 Query错误",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			forceError:    "conn.Query",
			description:   "教务员角色-默认查询 Query错误",
		},
		{
			name:          "教师角色-默认查询 Scan错误",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			forceError:    "rows.Scan",
			description:   "教务员角色-默认查询 Scan错误",
		},
		{
			name:          "学生角色-查询自己的考试",
			method:        "GET",
			queryParams:   "",
			userID:        testStudent1,
			userRole:      2008, // 学生角色
			expectedError: false,
			description:   "学生角色查询自己参与的考试",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 学生应该至少能看到一个自己参与的考试
				if len(examList) == 0 {
					t.Error("学生应该能看到自己参与的考试")
				}
			},
		},
		{
			name:          "学生角色-查询自己的考试 Query错误",
			method:        "GET",
			queryParams:   "",
			userID:        testStudent1,
			userRole:      2008, // 学生角色
			expectedError: true,
			forceError:    "conn.Query",
			description:   "学生角色-查询自己的考试 Query错误",
		},
		{
			name:          "学生角色-查询自己的考试 Scan错误",
			method:        "GET",
			queryParams:   "",
			userID:        testStudent1,
			userRole:      2008, // 学生角色
			expectedError: true,
			forceError:    "rows.Scan",
			description:   "学生角色-查询自己的考试 Scan错误",
		},
		{
			name:          "自定义查询-按名称过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"测试考试","Status":"","StartTime":0,"EndTime":0},"Page":1,"PageSize":10}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "按考试名称过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 验证返回的考试名称包含过滤条件
				for _, exam := range examList {
					if !strings.Contains(exam.ExamName, "测试考试1") {
						t.Errorf("返回的考试名称不符合过滤条件: %s", exam.ExamName)
					}
				}
			},
		},
		{
			name:          "自定义查询-按时间过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":` + strconv.FormatInt(testExamSession1StartTime, 10) + `,"EndTime":` + strconv.FormatInt(testExamSession1EndTime, 10) + `},"Page":1,"PageSize":10}`,
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: false,
			description:   "按考试时间过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
			},
		},
		{
			name:          "自定义查询-按状态过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"02","StartTime":0,"EndTime":0},"Page":1,"PageSize":10}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "按考试状态过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 验证返回的考试状态符合过滤条件
				for _, exam := range examList {
					if exam.Status != "02" {
						t.Errorf("返回的考试状态不符合过滤条件: %s", exam.Status)
					}
				}
			},
		},
		{
			name:          "自定义查询-分页测试1",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":1,"PageSize":1}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该不超过1
				if len(examList) != 1 {
					t.Errorf("分页测试失败，期望返回1个考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "自定义查询-分页测试2",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":-1,"PageSize":1}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该不超过1
				if len(examList) > 1 {
					t.Errorf("分页测试失败，期望最多返回1个考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "自定义查询-分页测试3",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":1,"PageSize":-1}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该为2
				if len(examList) <= 0 {
					t.Errorf("分页测试失败，期望返回考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "无效JSON查询参数",
			method:        "GET",
			queryParams:   `q={"invalid json}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			description:   "无效的JSON查询参数应返回错误",
		},
		{
			name:          "无效的用户ID-零值",
			method:        "GET",
			queryParams:   "",
			userID:        0,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0应返回错误",
		},
		{
			name:          "无效的用户ID-负值",
			method:        "GET",
			queryParams:   "",
			userID:        -1,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为负值应返回错误",
		},
		{
			name:          "无效的用户角色-零值",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      0,
			expectedError: true,
			errorContains: "未找到角色ID",
			description:   "用户角色为0应返回错误",
		},
		{
			name:          "时间范围错误-开始时间晚于结束时间",
			method:        "GET",
			queryParams:   `q={"Action":"select","Filter":{"Name":"","Status":"","StartTime":2000000000000,"EndTime":1000000000000},"Page":1,"PageSize":10}`,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "开始时间不能晚于结束时间",
			description:   "开始时间晚于结束时间应返回错误",
		},
		{
			name:          "不支持的HTTP方法",
			method:        "POST",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "unsupported method",
			description:   "POST方法应返回不支持的错误",
		},
		{
			name:          "模拟数据库查询错误",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			forceError:    "conn.QueryRow",
			expectedError: true,
			description:   "模拟数据库查询错误",
		},
		{
			name:          "模拟JSON序列化错误1",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			forceError:    "json.Marshal1",
			expectedError: true,
			description:   "模拟JSON序列化错误1",
		},
		{
			name:          "模拟JSON序列化错误2",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			forceError:    "json.Marshal2",
			expectedError: true,
			description:   "模拟JSON序列化错误2",
		},
		{
			name:          "模拟强制JSON解析错误1",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			forceError:    "json.Unmarshal1",
			expectedError: true,
			description:   "模拟强制JSON解析错误1",
		},
		{
			name:          "模拟强制JSON解析错误2",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			forceError:    "json.Unmarshal2",
			expectedError: true,
			description:   "模拟强制JSON解析错误1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析查询参数
			queryParams := url.Values{}
			if tt.queryParams != "" {
				if strings.HasPrefix(tt.queryParams, "q=") {
					queryParams.Set("q", tt.queryParams[2:])
				} else {
					parts := strings.Split(tt.queryParams, "&")
					for _, part := range parts {
						if kv := strings.Split(part, "="); len(kv) == 2 {
							queryParams.Set(kv[0], kv[1])
						}
					}
				}
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/api/examList", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 调用被测试的函数
			examList(ctx)

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("examList() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("examList() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("examList() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, serviceCtx)
				}
			}
		})
	}
}

// TestExaminee 测试examinee函数
func TestExaminee(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		method        string
		queryParams   map[string]string
		userID        int64
		userRole      int64
		forceError    string
		expectedError bool
		errorContains string
		checkResult   func(t *testing.T, serviceCtx *cmn.ServiceCtx)
		description   string
		mockValues    map[string]string
	}{
		{
			name:   "GET方法-成功获取考生列表",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: false,
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回考生数据，但Data为空")
					return
				}

				var examinees []Examinee
				err := json.Unmarshal(serviceCtx.Msg.Data, &examinees)
				if err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}

				// 验证数据结构
				if len(examinees) == 0 {
					t.Log("返回空的考生列表（这可能是正常的，如果没有考生数据）")
				} else {
					for _, examinee := range examinees {
						if examinee.StudentID <= 0 {
							t.Errorf("无效的学生ID: %d", examinee.StudentID)
						}
						if examinee.OfficialName == "" {
							t.Error("学生姓名不能为空")
						}
						if examinee.Account == "" {
							t.Error("学生账号不能为空")
						}
					}
				}
			},
			description: "教师角色成功获取考生列表",
		},
		{
			name:   "GET方法-获取权限失败",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2003, // 教师角色
			expectedError: true,
			mockValues:    map[string]string{"test": "validateUserExamPermission-error"},
			description:   "获取权限失败",
		},
		{
			name:        "GET方法-缺少exam_id参数",
			method:      "GET",
			queryParams: map[string]string{
				// 不提供exam_id
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "缺少exam_id参数应返回错误",
		},
		{
			name:   "GET方法-无效的exam_id参数",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "invalid",
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "无效的exam_id参数应返回错误",
		},
		{
			name:   "GET方法-exam_id为0",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "0",
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "exam_id为0应返回错误",
		},
		{
			name:   "GET方法-exam_id为负数",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "-1",
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "exam_id为负数应返回错误",
		},
		{
			name:   "GET方法-无效的用户ID",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        0, // 无效的用户ID
			userRole:      2002,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "无效的用户ID应返回错误",
		},
		{
			name:   "GET方法-无效的用户角色",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      0,
			expectedError: true,
			errorContains: "未找到角色ID",
			description:   "无效的用户角色应返回错误",
		},
		{
			name:   "GET方法-不存在的考试ID",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "999999", // 不存在的考试ID
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "无权限访问该考试",
			description:   "不存在的考试ID应返回权限错误",
		},
		{
			name:   "POST方法-不支持的方法",
			method: "POST",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "PUT方法-不支持的方法",
			method: "PUT",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "DELETE方法-不支持的方法",
			method: "DELETE",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "GET方法-模拟数据库查询错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "conn.Query",
			expectedError: true,
			errorContains: "force error",
			description:   "模拟数据库查询错误",
		},
		{
			name:   "GET方法-模拟数据库扫描错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "conn.Scan",
			expectedError: true,
			description:   "模拟数据库扫描错误",
		},
		{
			name:   "GET方法-模拟数据库Rows错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "rows.Err",
			expectedError: true,
			description:   "模拟数据库扫描错误",
		},
		{
			name:   "GET方法-模拟JSON序列化错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testNormalExamID),
			},
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "json.Marshal",
			expectedError: true,
			errorContains: "强制JSON序列化错误",
			description:   "模拟JSON序列化错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("测试用例: %s", tt.description)

			// 构建查询参数
			queryParams := url.Values{}
			for key, value := range tt.queryParams {
				queryParams.Set(key, value)
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/api/examinee", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 如果有模拟的错误，设置mockValues
			for key, value := range tt.mockValues {
				ctx = context.WithValue(ctx, key, value)
			}

			// 调用被测试的函数
			examinee(ctx)

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("examinee() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("examinee() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("examinee() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, serviceCtx)
				}
			}
		})
	}
}

// TestValidateUserForExamCreate 测试用户域权限验证
func TestValidateUserForExamCreateOrUpdate(t *testing.T) {
	cmn.ConfigureForTest()
	tests := []struct {
		name          string
		domain        string
		expectValid   bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "有效的cst管理员域",
			domain:      "cst.school.academicAffair^admin",
			expectValid: true,
		},
		{
			name:        "学生角色",
			domain:      "cst.school^student",
			expectValid: false,
		},
		{
			name:        "另一个有效的cst管理员域",
			domain:      "cst.university.department^admin",
			expectValid: true,
		},
		{
			name:        "cst前缀的admin域",
			domain:      "cst.admin.system^admin",
			expectValid: true,
		},
		{
			name:        "有cst但无admin权限的域",
			domain:      "cst.school.academicAffair^user",
			expectValid: false,
		},
		{
			name:        "有cst但无权限标识的域",
			domain:      "cst.school.student",
			expectValid: false,
		},
		{
			name:        "只有admin但无cst前缀",
			domain:      "admin",
			expectValid: false,
		},
		{
			name:        "包含admin但无cst前缀且位置错误",
			domain:      "admin.school.department^user",
			expectValid: false,
		},
		{
			name:        "无cst前缀且无admin权限",
			domain:      "school.department^user",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateUserForExamCreateOrUpdate(tt.domain)

			if tt.expectValid {
				assert.True(t, valid, "期望域验证通过")
			} else {
				assert.False(t, valid, "期望域验证失败")
			}
		})
	}
}

// TestExamStatus 测试考试状态更改
func TestExamStatus(t *testing.T) {

	cmn.ConfigureForTest()

	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	go exam_service.ExamMaintainService()

	tests := []struct {
		name           string
		description    string
		examID         int64
		userID         int64
		userRole       int64
		queryParams    string
		expectSuccess  bool
		errorContains  string
		forceError     string
		method         string
		expectedStatus string
	}{
		{
			name:          "非00状态考试发布",
			description:   "已发布的考试不能再次发布",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testPublishedExamID),
			expectSuccess: false,
			errorContains: "尝试发布不属于未发布状态的考试",
		},
		{
			name:          "作废考试时删除考卷失败",
			description:   "作废考试时删除考卷失败",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			forceError:    "examPaper.DeleteExamPaperById",
			errorContains: "强制删除考卷和答卷错误",
		},
		{
			name:          "作废考试时更新考生状态失败",
			description:   "作废考试时更新考生状态失败",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			forceError:    "updateExamineeStatus",
			errorContains: "强制更新考生状态错误",
		},
		{
			name:          "作废发布时强制查询错误",
			description:   "强制查询错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "QueryRow.CheckStatus",
			errorContains: "强制查询错误",
		},
		{
			name:           "正常的作废请求",
			description:    "正常的作废请求",
			examID:         testPublishedExamID,
			userID:         testAcademicAffair,
			userRole:       2002,
			queryParams:    fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess:  true,
			method:         "PUT",
			expectedStatus: "16",
		},
		{
			name:          "强制获取考试场次ID错误",
			description:   "强制获取考试场次ID错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "GetExamSessionIDs",
			errorContains: "强制获取考试场次ID错误",
		},
		{
			name:          "强制处理批改信息错误",
			description:   "强制处理批改信息错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "mark.HandleMarkerInfo",
			errorContains: "强制处理批改信息错误",
		},
		{
			name:          "强制更新考试状态错误",
			description:   "强制更新考试状态错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "updateExamStatus",
			errorContains: "强制更新考试状态错误",
		},
		{
			name:          "强制更新考试场次状态错误",
			description:   "强制更新考试场次状态错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "updateExamSessionStatus",
			errorContains: "强制更新考试场次状态错误",
		},
		{
			name:          "强制作废考试定时器错误",
			description:   "强制作废考试定时器错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "exam_service.CancelExamTimers",
			errorContains: "强制作废考试定时器错误",
		},
		{
			name:          "强制处理批改信息错误",
			description:   "强制处理批改信息错误",
			examID:        testPublishedExamID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testPublishedExamID),
			expectSuccess: false,
			method:        "PUT",
			forceError:    "mark.HandleMarkerInfo",
			errorContains: "强制处理批改信息错误",
		},
		{
			name:          "尝试作废不属于待开始状态的考试，无法执行作废操作",
			description:   "尝试作废不属于待开始状态的考试，无法执行作废操作",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"16"}}`, testExamToPublishID),
			expectSuccess: false,
			method:        "PUT",
			errorContains: "尝试作废不属于待开始状态的考试，无法执行作废操作",
		},
		{
			name:          "无效的请求方法",
			description:   "无效的请求方法",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"11"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "unsupported method",
			method:        "POST",
		},
		{
			name:          "未知状态",
			description:   "未知状态",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"11"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "不支持更新的考试状态",
		},
		{
			name:          "无权限访问",
			description:   "无权限访问",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2008,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "无权限访问",
		},
		{
			name:          "强制查询当前考试状态错误",
			description:   "强制查询当前考试状态错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "强制查询错误",
			forceError:    "QueryRow.CheckStatus",
		},
		{
			name:          "强制检查考试存在错误",
			description:   "强制检查考试存在错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "强制检查考试存在错误",
			forceError:    "checkExamExists",
		},
		{
			name:          "事务开始错误",
			description:   "事务开始错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "tx.Begin",
			errorContains: "强制开始事务错误",
		},
		{
			name:          "强制查询考生错误",
			description:   "强制查询考生错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "tx.Query",
			errorContains: "强制查询考生错误",
		},
		{
			name:          "强制获取考生ID错误",
			description:   "强制获取考生ID错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "rows.Scan",
			errorContains: "强制获取考生ID错误",
		},
		{
			name:          "强制生成考卷错误",
			description:   "强制生成考卷错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "examPaper.GenerateExamPaper",
			errorContains: "强制生成考卷错误",
		},
		{
			name:          "强制生成答卷错误",
			description:   "强制生成答卷错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "examPaper.GenerateAnswerQuestion",
			errorContains: "强制生成答卷错误",
		},
		{
			name:          "强制更新考生考卷ID错误",
			description:   "强制更新考生考卷ID错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "tx.Exec",
			errorContains: "强制更新考生考卷ID错误",
		},
		{
			name:          "强制查询考试创建者错误",
			description:   "强制查询考试创建者错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "tx.QueryRow",
			errorContains: "强制查询考试创建者错误",
		},
		{
			name:          "强制处理批改员信息错误",
			description:   "强制处理批改员信息错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "mark.HandleMarkerInfo",
			errorContains: "强制处理批改员信息错误",
		},
		{
			name:          "强制处理批改员ID转换错误",
			description:   "强制处理批改员ID转换错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "convertToInt64Array",
			errorContains: "强制转换批改员ID失败",
		},
		{
			name:          "强制更新考试状态错误",
			description:   "强制更新考试状态错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "updateExamStatus",
			errorContains: "强制更新考试状态错误",
		},
		{
			name:          "强制更新考试场次状态错误",
			description:   "强制更新考试场次状态错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "updateExamSessionStatus",
			errorContains: "强制更新考试场次状态错误",
		},
		{
			name:          "强制设置考试计时器错误",
			description:   "强制设置考试计时器错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "exam_service.SetExamTimers",
			errorContains: "强制设置考试计时器错误",
		},
		{
			name:          "检查考试是否存在时失败",
			description:   "检查考试是否存在时失败",
			examID:        99999,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   `q={"data":{"IDs":[99999],"Status":"02"}}`,
			expectSuccess: false,
			errorContains: "考试不存在或已被删除",
			forceError:    "examExists",
		},
		{
			name:          "检查考试是否存在时失败2",
			description:   "检查考试是否存在时失败2",
			examID:        99999,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   `q={"data":{"IDs":[99999,99999999],"Status":"02"}}`,
			expectSuccess: false,
			errorContains: "部分考试不存在或已被删除",
			forceError:    "examExists",
		},
		{
			name:          "强制获取考试场次错误",
			description:   "强制获取考试场次错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "GetExamSessions",
			errorContains: "强制获取考试场次错误",
		},
		{
			name:           "成功发布考试",
			description:    "教务员成功发布考试，状态从00变为02",
			examID:         testExamToPublishID,
			userID:         testAcademicAffair,
			userRole:       2002, // 教务员角色
			queryParams:    fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess:  true,
			errorContains:  "",
			expectedStatus: "02",
		},
		{
			name:           "事务提交错误",
			description:    "事务提交错误",
			examID:         testExamToPublishID,
			userID:         testAcademicAffair,
			userRole:       2002, // 教务员角色
			queryParams:    fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess:  true,
			forceError:     "tx.Commit",
			expectedStatus: "02",
		},
		{
			name:          "事务回滚错误",
			description:   "事务回滚错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			forceError:    "tx.Rollback",
		},
		{
			name:          "考试的开始时间已过",
			description:   "考试的开始时间已过",
			examID:        testErrorExamToPublishID1,
			userID:        testAcademicAffair,
			userRole:      2002, // 教务员角色
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testErrorExamToPublishID1),
			expectSuccess: false,
			errorContains: "考试的开始时间已过",
		},
		{
			name:          "强制尝试获取考试锁错误",
			description:   "强制尝试获取考试锁错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "强制尝试获取考试锁错误",
			forceError:    "cmn.TryLock",
		},
		{
			name:          "考试正在被其他用户编辑",
			description:   "考试正在被其他用户编辑",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "考试正在被其他用户编辑",
			forceError:    "cmn.TryLockFailed",
		},
		{
			name:           "强制释放考试锁错误",
			description:    "强制释放考试锁错误",
			examID:         testExamToPublishID,
			userID:         testAcademicAffair,
			userRole:       2002,
			queryParams:    fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess:  true,
			errorContains:  "强制释放考试锁错误",
			forceError:     "cmn.ReleaseLock",
			expectedStatus: "02",
		},
		{
			name:          "无效的RoleID",
			description:   "无效的RoleID",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      0,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "未找到角色ID",
		},
		{
			name:          "无效的考试ID",
			description:   "使用不存在的考试ID应该返回错误",
			examID:        -1,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   `q={"data":{"IDs":[-1],"Status":"02"}}`,
			expectSuccess: false,
			errorContains: "无效的考试ID",
		},
		{
			name:          "缺少q参数",
			description:   "缺少q参数应该返回错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   "",
			expectSuccess: false,
			errorContains: "请指定更新参数q",
		},
		{
			name:          "缺少考试ID",
			description:   "数据中没有包含考试编号应该返回错误",
			examID:        0,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   `q={"data":{"Status":"02"}}`,
			expectSuccess: false,
			errorContains: "data.IDs必须是数组格式",
		},
		{
			name:          "缺少状态参数",
			description:   "缺少Status参数应该返回错误",
			examID:        testExamToPublishID,
			userID:        testAcademicAffair,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "请指定更新参数data.Status",
		},
		{
			name:          "无效用户ID",
			description:   "无效的用户ID应该返回错误",
			examID:        testExamToPublishID,
			userID:        0,
			userRole:      2002,
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess: false,
			errorContains: "无效的用户ID",
		},
		{
			name:           "提交错误",
			description:    "提交事务时发生错误",
			examID:         testExamToPublishID,
			userID:         testAcademicAffair,
			userRole:       2002,
			queryParams:    fmt.Sprintf(`q={"data":{"IDs":[%d],"Status":"02"}}`, testExamToPublishID),
			expectSuccess:  true,
			forceError:     "tx.Commit",
			expectedStatus: "02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建查询参数
			queryParams := url.Values{}
			if tt.queryParams != "" {
				queryParams.Add("q", strings.TrimPrefix(tt.queryParams, "q="))
			}

			var ctx context.Context

			// 创建mock上下文
			if tt.method == "" {
				ctx = createMockContextWithRole("PUT", "/api/exam/status", queryParams, "", tt.userID, tt.userRole)
			} else {
				ctx = createMockContextWithRole(tt.method, "/api/exam/status", queryParams, tt.forceError, tt.userID, tt.userRole)
			}

			// 保考试状态为00
			if tt.examID > 0 && tt.examID != testNormalExamID && tt.examID != testPublishedExamID {
				conn := cmn.GetPgxConn()
				// 重置考试状态为00
				_, err := conn.Exec(context.Background(),
					`UPDATE t_exam_info SET status = '00' WHERE id = $1`, tt.examID)
				if err != nil {
					t.Fatalf("重置考试状态失败: %v", err)
				}
			}

			if tt.examID == testPublishedExamID {
				conn := cmn.GetPgxConn()
				// 重置考试状态为02
				_, err := conn.Exec(context.Background(),
					`UPDATE t_exam_info SET status = '02' WHERE id = $1`, tt.examID)
				if err != nil {
					t.Fatalf("重置考试状态失败: %v", err)
				}
			}

			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "examStatus-force-error", tt.forceError)
			}

			// 执行测试
			examStatus(ctx)

			// 获取响应上下文
			q := cmn.GetCtxValue(ctx)

			// 验证结果
			if tt.expectSuccess {
				// 期望成功的情况
				if q.Err != nil {
					t.Errorf("%s: 期望成功，但返回错误: %v", tt.description, q.Err)
					return
				}

				// 验证考试状态是否更新为02
				if tt.examID > 0 {
					conn := cmn.GetPgxConn()
					var currentStatus string
					err := conn.QueryRow(context.Background(),
						`SELECT status FROM t_exam_info WHERE id = $1`, tt.examID).Scan(&currentStatus)
					if err != nil {
						t.Errorf("%s: 查询考试状态失败: %v", tt.description, err)
						return
					}

					if currentStatus != tt.expectedStatus {
						t.Errorf("%s: 考试状态未正确更新，期望: %s, 实际: %s", tt.description, tt.expectedStatus, currentStatus)
					}

					// 验证考试场次状态也更新为02
					var sessionStatus string
					err = conn.QueryRow(context.Background(),
						`SELECT status FROM t_exam_session WHERE exam_id = $1 LIMIT 1`, tt.examID).Scan(&sessionStatus)
					if err == nil && sessionStatus != tt.expectedStatus {
						t.Errorf("%s: 考试场次状态未正确更新，期望: %s, 实际: %s", tt.description, tt.expectedStatus, sessionStatus)
					}
				}

				// 验证响应状态
				if q.Msg.Status != 0 {
					t.Errorf("%s: 期望响应状态为0，实际为: %d", tt.description, q.Msg.Status)
				}
			} else {
				// 期望失败的情况
				if q.Err == nil {
					t.Errorf("%s: 期望返回错误，但实际成功", tt.description)
					return
				}

				if tt.errorContains != "" && !strings.Contains(q.Err.Error(), tt.errorContains) {
					t.Errorf("%s: 错误信息不匹配，期望包含: %s, 实际: %s",
						tt.description, tt.errorContains, q.Err.Error())
				}

				// 验证响应状态不为0
				if q.Msg.Status == 0 {
					t.Errorf("%s: 期望响应状态不为0，但实际为0", tt.description)
				}
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

// TestExamExists 测试 examExists 函数
func TestExamExists(t *testing.T) {
	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		examID        int64
		testMode      string
		forceError    string
		expectExists  bool
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name:         "正常情况-考试存在",
			examID:       testNormalExamID,
			expectExists: true,
			expectError:  false,
			description:  "查询存在的考试应该返回true",
		},
		{
			name:         "正常情况-考试不存在",
			examID:       999999,
			expectExists: false,
			expectError:  false,
			description:  "查询不存在的考试应该返回false",
		},
		{
			name:          "无效的考试ID-负数",
			examID:        -1,
			expectExists:  false,
			expectError:   true,
			errorContains: "无效的考试ID",
			description:   "负数考试ID应该返回错误",
		},
		{
			name:          "无效的考试ID-零",
			examID:        0,
			expectExists:  false,
			expectError:   true,
			errorContains: "无效的考试ID",
			description:   "零考试ID应该返回错误",
		},
		{
			name:          "数据库查询错误",
			examID:        testNormalExamID,
			forceError:    "conn.QueryRow",
			expectExists:  false,
			expectError:   true,
			errorContains: "force error",
			description:   "强制数据库查询错误",
		},
		{
			name:         "测试模式-正常响应",
			examID:       testNormalExamID,
			testMode:     "normal-resp",
			expectExists: true,
			expectError:  false,
			description:  "测试模式下正常响应",
		},
		{
			name:          "测试模式-examExists错误",
			examID:        testNormalExamID,
			testMode:      "examExists-error",
			expectExists:  false,
			expectError:   true,
			errorContains: "examExists error",
			description:   "测试模式下examExists错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建上下文
			ctx := context.Background()

			// 设置测试模式
			if tt.testMode != "" {
				ctx = context.WithValue(ctx, TEST, tt.testMode)
			}

			// 设置强制错误
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "examExists-force-error", tt.forceError)
			}

			// 执行测试
			exists, err := examExists(ctx, tt.examID)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("%s: 期望返回错误，但没有错误", tt.description)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("%s: 错误信息不匹配，期望包含: %s, 实际: %s", tt.description, tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("%s: 期望无错误，但返回错误: %v", tt.description, err)
					return
				}
			}

			// 验证存在性结果
			if !tt.expectError && exists != tt.expectExists {
				t.Errorf("%s: 存在性结果不匹配，期望: %v, 实际: %v", tt.description, tt.expectExists, exists)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

func TestGetExamSessionIDs(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name        string
		examIDs     []int64
		forceError  string
		wantError   bool
		errorMsg    string
		description string
		checkResult func(*testing.T, []int64, error)
	}{
		{
			name:        "获取单个考试的场次ID",
			examIDs:     []int64{testNormalExamID},
			forceError:  "",
			wantError:   false,
			description: "正常获取单个考试的所有活跃场次ID",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessionIDs) != 2 {
					t.Errorf("场次ID数量不匹配: got %d, want 2", len(sessionIDs))
					return
				}
				// 验证返回的场次ID包含预期的ID
				expectedIDs := map[int64]bool{testExamSessionID1: false, testExamSessionID2: false}
				for _, id := range sessionIDs {
					if _, exists := expectedIDs[id]; exists {
						expectedIDs[id] = true
					}
				}
				for expectedID, found := range expectedIDs {
					if !found {
						t.Errorf("缺少预期的场次ID: %d", expectedID)
					}
				}
			},
		},
		{
			name:        "获取多个考试的场次ID",
			examIDs:     []int64{testNormalExamID, testNormalExamID2},
			forceError:  "",
			wantError:   false,
			description: "正常获取多个考试的所有活跃场次ID",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessionIDs) != 3 {
					t.Errorf("场次ID数量不匹配: got %d, want 3", len(sessionIDs))
					return
				}
				// 验证返回的场次ID包含预期的ID
				expectedIDs := map[int64]bool{
					testExamSessionID1: false,
					testExamSessionID2: false,
					testExamSessionID3: false,
				}
				for _, id := range sessionIDs {
					if _, exists := expectedIDs[id]; exists {
						expectedIDs[id] = true
					}
				}
				for expectedID, found := range expectedIDs {
					if !found {
						t.Errorf("缺少预期的场次ID: %d", expectedID)
					}
				}
			},
		},
		{
			name:        "获取不存在考试的场次ID",
			examIDs:     []int64{999999},
			forceError:  "",
			wantError:   false,
			description: "获取不存在考试的场次ID应返回空数组",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessionIDs) != 0 {
					t.Errorf("场次ID数量不匹配: got %d, want 0", len(sessionIDs))
				}
			},
		},
		{
			name:        "空考试ID列表",
			examIDs:     []int64{},
			forceError:  "",
			wantError:   false,
			description: "空考试ID列表应返回空数组",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessionIDs) != 0 {
					t.Errorf("场次ID数量不匹配: got %d, want 0", len(sessionIDs))
				}
			},
		},
		{
			name:        "数据库查询错误",
			examIDs:     []int64{testNormalExamID},
			forceError:  "conn.QueryOldExamSessionRows",
			wantError:   true,
			errorMsg:    "强制查询旧考试场次错误",
			description: "模拟数据库查询错误",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err == nil {
					t.Errorf("期望有错误，但没有收到")
					return
				}
				if !strings.Contains(err.Error(), "强制查询旧考试场次错误") {
					t.Errorf("错误消息不匹配: got %s, want contains '强制查询旧考试场次错误'", err.Error())
				}
			},
		},
		{
			name:        "Scan错误",
			examIDs:     []int64{testNormalExamID},
			forceError:  "oldExamSessionRows.Scan",
			wantError:   true,
			errorMsg:    "强制扫描旧考试场次ID错误",
			description: "模拟Scan错误",
			checkResult: func(t *testing.T, sessionIDs []int64, err error) {
				if err == nil {
					t.Errorf("期望有错误，但没有收到")
					return
				}
				if !strings.Contains(err.Error(), "强制扫描旧考试场次ID错误") {
					t.Errorf("错误消息不匹配: got %s, want contains '强制扫描旧考试场次ID错误'", err.Error())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// 设置强制错误
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			// 调用被测试的函数
			sessionIDs, err := getExamSessionIDs(ctx, tt.examIDs...)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("%s: 期望错误但没有收到", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("%s: 错误消息不匹配，期望包含: %s, 实际: %s", tt.description, tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("%s: 意外错误: %v", tt.description, err)
					return
				}
			}

			// 使用自定义检查函数验证结果
			if tt.checkResult != nil {
				tt.checkResult(t, sessionIDs, err)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

func TestConvertToInt64Array(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		forceError  string
		wantError   bool
		errorMsg    string
		expected    []int64
		description string
	}{
		{
			name:        "空值输入",
			data:        nil,
			forceError:  "",
			wantError:   false,
			expected:    []int64{},
			description: "nil输入应返回空数组",
		},
		{
			name:        "已经是int64数组",
			data:        []int64{1, 2, 3, 4, 5},
			forceError:  "",
			wantError:   false,
			expected:    []int64{1, 2, 3, 4, 5},
			description: "直接返回int64数组",
		},
		{
			name:        "空int64数组",
			data:        []int64{},
			forceError:  "",
			wantError:   false,
			expected:    []int64{},
			description: "返回空的int64数组",
		},
		{
			name:        "interface{}数组包含int64",
			data:        []interface{}{int64(10), int64(20), int64(30)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{10, 20, 30},
			description: "转换interface{}数组中的int64元素",
		},
		{
			name:        "interface{}数组包含float64",
			data:        []interface{}{float64(10.0), float64(20.0), float64(30.0)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{10, 20, 30},
			description: "转换interface{}数组中的float64元素（JSON常见类型）",
		},
		{
			name:        "interface{}数组包含int",
			data:        []interface{}{int(15), int(25), int(35)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{15, 25, 35},
			description: "转换interface{}数组中的int元素",
		},
		{
			name:        "interface{}数组包含int32",
			data:        []interface{}{int32(100), int32(200), int32(300)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{100, 200, 300},
			description: "转换interface{}数组中的int32元素",
		},
		{
			name:        "interface{}数组包含混合数值类型",
			data:        []interface{}{int64(1), float64(2.0), int(3), int32(4)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{1, 2, 3, 4},
			description: "转换interface{}数组中的混合数值类型",
		},
		{
			name:        "interface{}数组包含不支持的类型",
			data:        []interface{}{int64(1), "invalid", int64(3)},
			forceError:  "",
			wantError:   true,
			errorMsg:    "unsupported type in array: string",
			description: "interface{}数组中包含不支持的字符串类型应返回错误",
		},
		{
			name:        "interface{}数组包含布尔类型",
			data:        []interface{}{int64(1), true, int64(3)},
			forceError:  "",
			wantError:   true,
			errorMsg:    "unsupported type in array: bool",
			description: "interface{}数组中包含不支持的布尔类型应返回错误",
		},
		{
			name:        "不支持的数据类型 - 字符串",
			data:        "not an array",
			forceError:  "",
			wantError:   true,
			errorMsg:    "unsupported data type: string",
			description: "字符串类型应返回错误",
		},
		{
			name:        "不支持的数据类型 - map",
			data:        map[string]int{"a": 1, "b": 2},
			forceError:  "",
			wantError:   true,
			errorMsg:    "unsupported data type: map[string]int",
			description: "map类型应返回错误",
		},
		{
			name:        "空interface{}数组",
			data:        []interface{}{},
			forceError:  "",
			wantError:   false,
			expected:    []int64{},
			description: "空interface{}数组应返回空int64数组",
		},
		{
			name:        "大数字float64转换",
			data:        []interface{}{float64(1234567890123456)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{1234567890123456},
			description: "大数字float64应正确转换为int64",
		},
		{
			name:        "负数转换",
			data:        []interface{}{int64(-1), float64(-2.0), int(-3), int32(-4)},
			forceError:  "",
			wantError:   false,
			expected:    []int64{-1, -2, -3, -4},
			description: "负数应正确转换",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// 设置强制错误（如果有）
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "examStatus-force-error", tt.forceError)
			}

			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			// 调用被测试的函数
			result, err := convertToInt64Array(ctx, tt.data)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("%s: 期望错误但没有收到", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("%s: 错误消息不匹配，期望包含: %s, 实际: %s", tt.description, tt.errorMsg, err.Error())
				}
				t.Logf("%s: 正确收到期望的错误: %v", tt.name, err)
			} else {
				if err != nil {
					t.Errorf("%s: 意外错误: %v", tt.description, err)
					return
				}

				// 检查结果长度
				if len(result) != len(tt.expected) {
					t.Errorf("%s: 结果长度不匹配，期望: %d, 实际: %d", tt.description, len(tt.expected), len(result))
					return
				}

				// 检查每个元素
				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("%s: 索引%d的值不匹配，期望: %d, 实际: %d", tt.description, i, expected, result[i])
					}
				}
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

func TestConvertToInt64ArrayEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() interface{}
		wantError   bool
		errorMsg    string
		description string
	}{
		{
			name: "interface{}数组包含nil",
			setupFunc: func() interface{} {
				return []interface{}{int64(1), nil, int64(3)}
			},
			wantError:   true,
			errorMsg:    "unsupported type in array: <nil>",
			description: "interface{}数组中包含nil应返回错误",
		},
		{
			name: "复杂嵌套类型",
			setupFunc: func() interface{} {
				return []interface{}{int64(1), []int{2, 3}, int64(4)}
			},
			wantError:   true,
			errorMsg:    "unsupported type in array: []int",
			description: "interface{}数组中包含嵌套数组应返回错误",
		},
		{
			name: "interface{}类型但不是数组",
			setupFunc: func() interface{} {
				var x interface{} = int64(42)
				return x
			},
			wantError:   true,
			errorMsg:    "unsupported data type: int64",
			description: "单个interface{}值（非数组）应返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			data := tt.setupFunc()

			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			result, err := convertToInt64Array(ctx, data)

			if tt.wantError {
				if err == nil {
					t.Errorf("%s: 期望错误但没有收到，结果: %v", tt.description, result)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("%s: 错误消息不匹配，期望包含: %s, 实际: %s", tt.description, tt.errorMsg, err.Error())
				}
				t.Logf("%s: 正确收到期望的错误: %v", tt.name, err)
			} else {
				if err != nil {
					t.Errorf("%s: 意外错误: %v", tt.description, err)
				}
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

func TestExamDeleteMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		description   string
		userID        int64
		userRole      int64
		requestBody   interface{}
		forceError    string
		expectedError bool
		errorContains string
		setupFunc     func(t *testing.T) // 测试前的数据准备
		cleanupFunc   func(t *testing.T) // 测试后的清理
		verifyFunc    func(t *testing.T) // 验证数据库状态
	}{
		{
			name:          "请求体为空",
			description:   "删除考试时请求体为空应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   "",
			expectedError: true,
			errorContains: "请求体为空",
		},
		{
			name:          "学生用户不能删除考试",
			description:   "学生角色用户不应该有删除考试的权限",
			userID:        testStudent1,
			userRole:      2008,
			requestBody:   []int64{testNormalExamID},
			expectedError: true,
			errorContains: "学生用户不能删除考试",
		},
		{
			name:          "没有提供要删除的考试ID",
			description:   "考试ID列表为空时应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{},
			expectedError: true,
			errorContains: "没有提供要删除的考试ID",
		},
		{
			name:          "无效的考试ID",
			description:   "提供无效的考试ID（负数或零）应返回错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{-1, 0},
			expectedError: true,
			errorContains: "无效的考试ID",
		},
		{
			name:          "强制JSON解析错误1",
			description:   "模拟第一次JSON解析失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "exam-delete-json.Unmarshal1-err",
			expectedError: true,
			errorContains: "exam-delete-json.Unmarshal1-err",
		},
		{
			name:          "强制JSON解析错误2",
			description:   "模拟第二次JSON解析失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "exam-delete-json.Unmarshal2-err",
			expectedError: true,
			errorContains: "exam-delete-json.Unmarshal2-err",
		},
		{
			name:          "强制检查考试是否能删除错误",
			description:   "强制检查考试是否能删除错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "checkExam",
			expectedError: true,
			errorContains: "强制检查考试存在错误",
		},
		{
			name:          "无法删除考试",
			description:   "无法删除考试",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testPublishedExamID},
			expectedError: true,
			errorContains: "考试无法删除",
		},
		{
			name:          "无法删除考试2",
			description:   "无法删除考试2",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testPublishedExamID, testEndExamID},
			expectedError: true,
			errorContains: "部分考试无法删除",
		},

		{
			name:          "强制读取请求体错误",
			description:   "模拟读取请求体失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "exam-delete-io.ReadAll-err",
			expectedError: true,
			errorContains: "exam-delete-io.ReadAll-err",
		},
		{
			name:          "强制获取考试场次ID错误",
			description:   "模拟获取考试场次ID失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "getExamSessionIDs",
			expectedError: true,
			errorContains: "强制获取考试场次ID错误",
		},
		{
			name:          "强制获取考试编辑锁错误",
			description:   "模拟获取考试编辑锁错误",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "cmn.TryLock",
			expectedError: true,
			errorContains: "强制获取考试锁错误",
		},
		{
			name:          "强制获取考试编辑锁失败",
			description:   "模拟获取考试编辑锁失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "cmn.TryLockFailed",
			expectedError: true,
			errorContains: "考试正在被其他用户编辑",
		},
		{
			name:          "强制事务开始错误",
			description:   "模拟事务开始失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.Begin",
			expectedError: true,
			errorContains: "强制开启事务错误",
		},
		{
			name:          "强制删除批改信息错误",
			description:   "模拟删除批改信息失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "mark.HandleMarkerInfo",
			expectedError: true,
			errorContains: "强制删除批改信息错误",
		},
		{
			name:          "强制删除考生错误",
			description:   "模拟软删除考生失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.SoftDeleteExaminee",
			expectedError: true,
			errorContains: "强制删除考生错误",
		},
		{
			name:          "强制删除考试场次错误",
			description:   "模拟软删除考试场次失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.SoftDeleteExamSessions",
			expectedError: true,
			errorContains: "强制删除考试场次错误",
		},
		{
			name:          "强制删除考卷错误",
			description:   "模拟删除考卷失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "examPaper.DeleteExamPaperById",
			expectedError: true,
			errorContains: "强制删除考卷和答卷错误",
		},
		{
			name:          "强制删除考试信息错误",
			description:   "模拟软删除考试信息失败",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.SoftDeleteExamInfo",
			expectedError: true,
			errorContains: "强制删除考试信息错误",
		},
		{
			name:          "成功删除考试",
			description:   "成功删除单个考试及其相关数据",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			expectedError: false,
			verifyFunc: func(t *testing.T) {
				conn := cmn.GetPgxConn()
				ctx := context.Background()

				// 检查考试状态已更新为12（删除状态）
				var examStatus string
				err := conn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id=$1", testNormalExamID).Scan(&examStatus)
				assert.Nil(t, err)
				assert.Equal(t, "12", examStatus)

				// 检查考试场次状态已更新为14（删除状态）
				var sessionStatus string
				err = conn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE exam_id=$1", testNormalExamID).Scan(&sessionStatus)
				assert.Nil(t, err)
				assert.Equal(t, "14", sessionStatus)

				// 检查考生状态已更新为08（删除状态）
				var examineeStatus string
				err = conn.QueryRow(ctx, "SELECT status FROM t_examinee WHERE exam_session_id=$1", testExamSessionID1).Scan(&examineeStatus)
				assert.Nil(t, err)
				assert.Equal(t, "08", examineeStatus)
			},
		},
		{
			name:          "成功删除多个考试",
			description:   "成功删除多个考试及其相关数据",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testDeleteExamID, testExamToPublishID},
			expectedError: false,
			verifyFunc: func(t *testing.T) {
				conn := cmn.GetPgxConn()
				ctx := context.Background()

				// 检查所有考试状态已更新为12（删除状态）
				for _, examID := range []int64{testDeleteExamID, testExamToPublishID} {
					var examStatus string
					err := conn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id=$1", examID).Scan(&examStatus)
					assert.Nil(t, err)
					assert.Equal(t, "12", examStatus)
				}
			},
		},
		{
			name:          "强制关闭请求体错误",
			description:   "模拟关闭请求体失败（仅记录日志，不影响主流程）",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "exam-delete-io.Close-err",
			expectedError: false, // 这个错误只记录日志，不影响主流程
		},
		{
			name:          "强制事务回滚错误",
			description:   "模拟事务回滚失败（仅记录日志）",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.Rollback",
			expectedError: false, // 回滚错误只记录日志
		},
		{
			name:          "强制事务提交错误",
			description:   "模拟事务提交失败（仅记录日志）",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "tx.Commit",
			expectedError: false, // 提交错误只记录日志
		},
		{
			name:          "强制释放锁错误",
			description:   "模拟释放考试锁失败（仅记录日志）",
			userID:        testAcademicAffair,
			userRole:      2002,
			requestBody:   []int64{testNormalExamID},
			forceError:    "cmn.ReleaseLock",
			expectedError: false, // 释放锁错误只记录日志
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			// 执行测试前的准备工作
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			// 执行测试后的清理工作
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc(t)
			}

			// 构建请求体
			var requestBody string
			if tt.requestBody == "" {
				requestBody = ""
			} else {
				data, err := json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				requestBody = string(data)
			}

			// 创建模拟上下文
			ctx := createMockContextWithBody("DELETE", "/exam", requestBody, tt.forceError, tt.userID, tt.userRole)

			// 执行被测试的函数
			exam(ctx)

			// 获取响应
			q := cmn.GetCtxValue(ctx)

			// 验证结果
			if tt.expectedError {
				assert.NotNil(t, q.Err, "%s: 期望有错误但没有收到", tt.description)
				if tt.errorContains != "" {
					assert.Contains(t, q.Err.Error(), tt.errorContains, "%s: 错误消息不匹配", tt.description)
				}
				t.Logf("%s: 收到期望的错误: %v", tt.name, q.Err)
			} else {
				if q.Err != nil {
					t.Errorf("%s: 意外错误: %v", tt.description, q.Err)
				} else {
					assert.NotNil(t, q.Msg, "%s: 期望有响应消息", tt.description)
					t.Logf("%s: 操作成功完成", tt.name)
				}
			}

			// 执行数据库状态验证
			if tt.verifyFunc != nil && !tt.expectedError {
				tt.verifyFunc(t)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

// TestExamLock 测试考试锁功能
func TestExamLock(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 设置测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)

	// 清理函数
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	tests := []struct {
		name          string
		method        string
		queryParams   string
		forceError    string
		expectedError bool
		errorContains string
		expectedMsg   string
		description   string
		userID        int64
		userRole      int64
	}{
		// GET 方法测试（获取考试锁）
		{
			name:          "GET-正常获取考试锁-教务员角色",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: false,
			expectedMsg:   "成功获取考试锁",
			description:   "教务员角色正常获取考试锁",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-无效考试ID-空参数",
			method:        "GET",
			queryParams:   "exam_id=",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "空考试ID参数",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-无效考试ID-非数字",
			method:        "GET",
			queryParams:   "exam_id=abc",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "非数字考试ID",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-无效考试ID-零值",
			method:        "GET",
			queryParams:   "exam_id=0",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "零值考试ID",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-无效考试ID-负数",
			method:        "GET",
			queryParams:   "exam_id=-1",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "负数考试ID",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-无效用户ID",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0",
			userID:        0,
			userRole:      2002,
		},
		{
			name:          "GET-无效用户域",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "未找到角色ID",
			description:   "无效用户域",
			userID:        testAcademicAffair,
			userRole:      9999,
		},
		{
			name:          "GET-用户权限验证失败",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermission",
			expectedError: true,
			errorContains: "强制验证用户考试权限错误",
			description:   "强制用户权限验证错误",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-用户无权限访问考试",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermissionFailed",
			expectedError: true,
			errorContains: "无权访问该考试",
			description:   "用户无权限访问该考试",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-尝试获取锁失败",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "cmn.TryLock",
			expectedError: true,
			errorContains: "强制尝试获取考试锁错误",
			description:   "强制尝试获取锁失败",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "GET-考试正在被其他用户编辑",
			method:        "GET",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "cmn.TryLockFailed",
			expectedError: true,
			errorContains: "考试正在被其他用户编辑",
			description:   "考试正在被其他用户编辑",
			userID:        testAcademicAffair,
			userRole:      2002,
		},

		// PUT 方法测试（刷新考试锁）
		{
			name:          "PUT-正常刷新考试锁-教务员角色",
			method:        "PUT",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: false,
			expectedMsg:   "成功刷新考试锁",
			description:   "教务员角色正常刷新考试锁",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "PUT-无效考试ID-空参数",
			method:        "PUT",
			queryParams:   "exam_id=",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "空考试ID参数",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "PUT-无效用户ID",
			method:        "PUT",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0",
			userID:        0,
			userRole:      2002,
		},
		{
			name:          "PUT-用户权限验证失败",
			method:        "PUT",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermission",
			expectedError: true,
			errorContains: "强制验证用户考试权限错误",
			description:   "强制用户权限验证错误",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "PUT-用户无权限访问考试",
			method:        "PUT",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermissionFailed",
			expectedError: true,
			errorContains: "无权访问该考试",
			description:   "用户无权限访问该考试",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "PUT-刷新锁失败",
			method:        "PUT",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "cmn.RefreshLock",
			expectedError: true,
			errorContains: "强制刷新考试锁错误",
			description:   "强制刷新锁失败",
			userID:        testAcademicAffair,
			userRole:      2002,
		},

		// DELETE 方法测试（释放考试锁）
		{
			name:          "DELETE-正常释放考试锁-教务员角色",
			method:        "DELETE",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: false,
			expectedMsg:   "成功清除考试锁",
			description:   "教务员角色正常释放考试锁",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "DELETE-无效考试ID-空参数",
			method:        "DELETE",
			queryParams:   "exam_id=",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "空考试ID参数",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "DELETE-无效用户ID",
			method:        "DELETE",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0",
			userID:        0,
			userRole:      2002,
		},
		{
			name:          "DELETE-用户权限验证失败",
			method:        "DELETE",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermission",
			expectedError: true,
			errorContains: "强制验证用户考试权限错误",
			description:   "强制用户权限验证错误",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "DELETE-用户无权限访问考试",
			method:        "DELETE",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "validateUserExamPermissionFailed",
			expectedError: true,
			errorContains: "无权访问该考试",
			description:   "用户无权限访问该考试",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "DELETE-释放锁失败",
			method:        "DELETE",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "cmn.ReleaseLock",
			expectedError: true,
			errorContains: "强制释放考试锁错误",
			description:   "强制释放锁失败",
			userID:        testAcademicAffair,
			userRole:      2002,
		},

		// 不支持的方法测试
		{
			name:          "PATCH-不支持的方法",
			method:        "PATCH",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "unsupported method: patch",
			description:   "不支持的HTTP方法",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
		{
			name:          "POST-不支持的方法",
			method:        "POST",
			queryParams:   fmt.Sprintf("exam_id=%d", testNormalExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "unsupported method: post",
			description:   "不支持的HTTP方法",
			userID:        testAcademicAffair,
			userRole:      2002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			// 创建查询参数
			queryParams, _ := url.ParseQuery(tt.queryParams)

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/exam/lock", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 调用函数
			examLock(ctx)

			// 获取响应
			q := cmn.GetCtxValue(ctx)

			// 验证结果
			if tt.expectedError {
				assert.Error(t, q.Err, tt.description)
				if tt.errorContains != "" {
					assert.Contains(t, q.Err.Error(), tt.errorContains, tt.description)
				}
				t.Logf("%s: 正确收到期望的错误: %v", tt.name, q.Err)
			} else {
				assert.NoError(t, q.Err, tt.description)
				if tt.expectedMsg != "" {
					assert.Contains(t, q.Msg.Msg, tt.expectedMsg, tt.description)
				}
				t.Logf("%s: 操作成功完成，响应消息: %s", tt.name, q.Msg.Msg)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

// TestUpdateExamineeStatus 测试更新考生状态功能
func TestUpdateExamineeStatus(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 设置测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)

	// 清理函数
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	// 获取数据库连接
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	tests := []struct {
		name          string
		newStatus     string
		userID        int64
		examIDs       []int64
		forceError    string
		expectedError bool
		errorContains string
		description   string
		verifyFunc    func(t *testing.T)
	}{
		{
			name:          "正常更新考生状态-单个考试",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID},
			forceError:    "",
			expectedError: false,
			description:   "正常更新单个考试的考生状态",
			verifyFunc: func(t *testing.T) {
				// 验证状态是否正确更新
				var count int
				err := conn.QueryRow(ctx, `
					SELECT COUNT(*) FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id = $1 AND e.status = '02' AND e.updated_by = $2
				`, testNormalExamID, testAcademicAffair).Scan(&count)
				assert.NoError(t, err)
				assert.Greater(t, count, 0, "应该有考生状态被更新")
			},
		},
		{
			name:          "正常更新考生状态-多个考试",
			newStatus:     "03",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID, testNormalExamID2},
			forceError:    "",
			expectedError: false,
			description:   "正常更新多个考试的考生状态",
			verifyFunc: func(t *testing.T) {
				// 验证状态是否正确更新
				var count int
				err := conn.QueryRow(ctx, `
					SELECT COUNT(*) FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id IN ($1, $2) AND e.status = '03' AND e.updated_by = $3
				`, testNormalExamID, testNormalExamID2, testAcademicAffair).Scan(&count)
				assert.NoError(t, err)
				assert.Greater(t, count, 0, "应该有考生状态被更新")
			},
		},
		{
			name:          "考试ID数组为空",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{},
			forceError:    "",
			expectedError: true,
			errorContains: "考试ID数组不能为空",
			description:   "考试ID数组为空应该返回错误",
		},
		{
			name:          "无效的考试ID-零值",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{0},
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID: 0",
			description:   "零值考试ID应该返回错误",
		},
		{
			name:          "无效的考试ID-负数",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{-1},
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID: -1",
			description:   "负数考试ID应该返回错误",
		},
		{
			name:          "无效的考试ID-混合有效无效",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID, 0},
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID: 0",
			description:   "包含无效考试ID的数组应该返回错误",
		},
		{
			name:          "无效的用户ID-零值",
			newStatus:     "02",
			userID:        0,
			examIDs:       []int64{testNormalExamID},
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID: 0",
			description:   "零值用户ID应该返回错误",
		},
		{
			name:          "无效的用户ID-负数",
			newStatus:     "02",
			userID:        -1,
			examIDs:       []int64{testNormalExamID},
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID: -1",
			description:   "负数用户ID应该返回错误",
		},
		{
			name:          "更新状态为空",
			newStatus:     "",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID},
			forceError:    "",
			expectedError: true,
			errorContains: "更新状态不能为空",
			description:   "空状态应该返回错误",
		},
		{
			name:          "数据库执行错误",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID},
			forceError:    "updateExamineeStatus.Exec",
			expectedError: true,
			errorContains: "force error: updateExamineeStatus.Exec",
			description:   "强制数据库执行错误",
		},
		{
			name:          "不更新状态为08的考生",
			newStatus:     "04",
			userID:        testAcademicAffair,
			examIDs:       []int64{testNormalExamID},
			forceError:    "",
			expectedError: false,
			description:   "不应该更新状态为08的考生",
			verifyFunc: func(t *testing.T) {
				// 首先设置一个考生状态为08
				tx, err := conn.Begin(ctx)
				if err != nil {
					t.Fatalf("开始事务失败: %v", err)
				}

				// 更新一个考生状态为08
				_, err = tx.Exec(ctx, `
					UPDATE t_examinee 
					SET status = '08'
					WHERE exam_session_id IN (
						SELECT id FROM t_exam_session WHERE exam_id = $1
					)
				`, testNormalExamID)
				if err != nil {
					tx.Rollback(ctx)
					t.Fatalf("设置考生状态为08失败: %v", err)
				}

				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("提交事务失败: %v", err)
				}

				// 验证状态为08的考生数量
				var count08Before int
				err = conn.QueryRow(ctx, `
					SELECT COUNT(*) FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id = $1 AND e.status = '08'
				`, testNormalExamID).Scan(&count08Before)
				assert.NoError(t, err)
				assert.Greater(t, count08Before, 0, "应该有状态为08的考生")

				// 执行更新操作
				tx2, err := conn.Begin(ctx)
				if err != nil {
					t.Fatalf("开始事务失败: %v", err)
				}

				err = updateExamineeStatus(ctx, tx2, "04", testAcademicAffair, testNormalExamID)
				assert.NoError(t, err)

				err = tx2.Commit(ctx)
				assert.NoError(t, err)

				// 验证状态为08的考生数量没有变化
				var count08After int
				err = conn.QueryRow(ctx, `
					SELECT COUNT(*) FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id = $1 AND e.status = '08'
				`, testNormalExamID).Scan(&count08After)
				assert.NoError(t, err)
				assert.Equal(t, count08Before, count08After, "状态为08的考生数量不应该改变")
			},
		},
		{
			name:          "更新不存在场次的考试",
			newStatus:     "02",
			userID:        testAcademicAffair,
			examIDs:       []int64{99999},
			forceError:    "",
			expectedError: false,
			description:   "更新不存在场次的考试不应该报错，但也不会更新任何记录",
			verifyFunc: func(t *testing.T) {
				// 验证没有记录被更新
				var count int
				err := conn.QueryRow(ctx, `
					SELECT COUNT(*) FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id = 99999 AND e.updated_by = $1
				`, testAcademicAffair).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count, "不应该有任何记录被更新")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%s: 开始测试 - %s", tt.name, tt.description)

			// 创建带强制错误的上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 开始事务
			tx, err := conn.Begin(testCtx)
			if err != nil {
				t.Fatalf("开始事务失败: %v", err)
			}

			// 调用被测试的函数
			err = updateExamineeStatus(testCtx, tx, tt.newStatus, tt.userID, tt.examIDs...)

			// 验证结果
			if tt.expectedError {
				assert.Error(t, err, tt.description)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, tt.description)
				}
				t.Logf("%s: 正确收到期望的错误: %v", tt.name, err)
				tx.Rollback(testCtx)
			} else {
				assert.NoError(t, err, tt.description)

				// 提交事务以便验证函数
				err = tx.Commit(testCtx)
				assert.NoError(t, err, "提交事务应该成功")

				// 执行验证函数
				if tt.verifyFunc != nil {
					tt.verifyFunc(t)
				}

				t.Logf("%s: 操作成功完成", tt.name)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

// TestExamUser 测试examUser函数
func TestExamUser(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer func() {
		CleanTestExamData(t)
	}()

	tests := []struct {
		name          string
		method        string
		queryParams   string
		userID        int64
		userRole      int64
		forceError    string
		expectedError bool
		errorContains string
		checkResult   func(t *testing.T, serviceCtx *cmn.ServiceCtx)
		description   string
	}{
		{
			name:          "成功查询单个用户",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99901),
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: false,
			description:   "查询单个用户的基本信息",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var userInfos []ExamUserInfo
				if err := json.Unmarshal(serviceCtx.Msg.Data, &userInfos); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(userInfos) != 1 {
					t.Errorf("期望返回1个用户信息，实际返回%d个", len(userInfos))
					return
				}
				user := userInfos[0]
				if user.ID != 99901 {
					t.Errorf("用户ID不匹配: got %d, want 99901", user.ID)
				}
				if user.Name != "测试用户" {
					t.Errorf("用户名称不匹配: got %s, want 测试用户", user.Name)
				}

			},
		},
		{
			name:          "成功查询多个用户",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d,%d]}}`, 99901, 99902),
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: false,
			description:   "查询多个用户的基本信息",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var userInfos []ExamUserInfo
				if err := json.Unmarshal(serviceCtx.Msg.Data, &userInfos); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(userInfos) != 2 {
					t.Errorf("期望返回2个用户信息，实际返回%d个", len(userInfos))
					return
				}
				// 检查用户信息
				userMap := make(map[int64]ExamUserInfo)
				for _, user := range userInfos {
					userMap[user.ID] = user
				}
				if user, ok := userMap[99901]; ok {
					if user.Name != "测试用户" {
						t.Errorf("用户99901名称不匹配: got %s, want 测试用户", user.Name)
					}
				} else {
					t.Error("未找到用户99901")
				}
				if user, ok := userMap[99902]; ok {
					if user.Name != "测试学生" {
						t.Errorf("用户99902名称不匹配: got %s, want 测试学生", user.Name)
					}
				} else {
					t.Error("未找到用户99902")
				}
			},
		},
		{
			name:          "缺少查询参数q",
			method:        "GET",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: true,
			errorContains: "请指定参数q",
			description:   "缺少查询参数q应返回错误",
		},
		{
			name:          "不支持的HTTP方法-POST",
			method:        "POST",
			queryParams:   "",
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: true,
			errorContains: "unsupported method: post",
			description:   "POST方法应返回不支持的错误",
		},
		{
			name:          "查询不存在的用户",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99999),
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: false,
			description:   "查询不存在的用户应返回空列表",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var userInfos []ExamUserInfo
				if err := json.Unmarshal(serviceCtx.Msg.Data, &userInfos); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(userInfos) != 0 {
					t.Errorf("期望返回0个用户信息，实际返回%d个", len(userInfos))
				}
			},
		},
		{
			name:          "空的用户ID列表",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[]}}`),
			userID:        testAcademicAffair,
			userRole:      2002,
			expectedError: false,
			description:   "空的用户ID列表应返回空结果",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var userInfos []ExamUserInfo
				if err := json.Unmarshal(serviceCtx.Msg.Data, &userInfos); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(userInfos) != 0 {
					t.Errorf("期望返回0个用户信息，实际返回%d个", len(userInfos))
				}
			},
		},
		{
			name:          "json.Marshal1",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[]}}`),
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "json.Marshal1",
			expectedError: true,
			errorContains: "强制JSON序列化错误",
			description:   "强制JSON序列化错误",
		},
		{
			name:          "json.Marshal2",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99901),
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "json.Marshal2",
			expectedError: true,
			errorContains: "强制JSON序列化错误",
			description:   "强制JSON序列化错误",
		},
		{
			name:          "notArray",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99901),
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "notArray",
			expectedError: true,
			errorContains: "data.IDs必须是数组格式",
			description:   "data.IDs必须是数组格式",
		},
		{
			name:          "模拟数据库查询错误",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99901),
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "conn.Query",
			expectedError: true,
			errorContains: "强制查询用户信息错误",
			description:   "模拟数据库查询错误",
		},
		{
			name:          "模拟数据库查询错误2",
			method:        "GET",
			queryParams:   fmt.Sprintf(`q={"data":{"IDs":[%d]}}`, 99901),
			userID:        testAcademicAffair,
			userRole:      2002,
			forceError:    "rows.Scan",
			expectedError: true,
			errorContains: "强制获取用户信息错误",
			description:   "模拟数据库查询错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			queryParams := url.Values{}
			if tt.queryParams != "" {
				queryParams.Add("q", strings.TrimPrefix(tt.queryParams, "q="))
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/api/examuser", queryParams, tt.forceError, tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("examUser() 意外panic: %v", r)
						}
					}
				}()

				examUser(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("examUser() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("examUser() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("examUser() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 检查结果
				if tt.checkResult != nil {
					tt.checkResult(t, serviceCtx)
				}
			}
		})
	}
}
