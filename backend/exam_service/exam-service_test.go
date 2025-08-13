package exam_service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
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

// 定义自定义类型用于 context key，避免键冲突
type contextKey string

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
			(id, name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
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
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testAcademicAffair, "sys^admin", "测试用户", "test_user", 2002)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 插入测试批阅员数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testGrader, "sys^admin", "测试批阅员", "test_grader", 2005)
	if err != nil {
		t.Fatalf("创建测试批阅员失败: %v", err)
	}

	// 插入测试学生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testStudent1, "sys^student", "测试学生", "test_student", 2008)
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
			INSERT INTO assessuser.t_question (id, type, difficulty, creator,status)
			VALUES ($1, $2, $3, $4, $5)
		`, q.id, q.qtype, q.difficulty, q.creator, q.status)
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
		INSERT INTO t_paper (id, name, category, creator, status) 
		VALUES ($1, '测试试卷', '00', $2, '00') `, testPaperID, testAcademicAffair)
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

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSetExamTimers(t *testing.T) {

	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer CleanTestExamData(t)

	tests := []struct {
		name          string
		examID        int64
		expectError   bool
		errorContains string
		checkTimers   bool
		forceError    string
	}{
		{
			name:        "正常设置考试定时器",
			examID:      testNormalExamID,
			expectError: false,
			checkTimers: true,
		},
		{
			name:        "设置不存在的考试定时器",
			examID:      999999,
			expectError: false,
			checkTimers: false,
		},
		{
			name:          "查询考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制查询考试场次信息错误",
			checkTimers:   false,
			forceError:    "queryExamSessions",
		},
		{
			name:          "扫描考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制获取考试场次信息错误",
			checkTimers:   false,
			forceError:    "scanExamSessionInfo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()
			if tt.forceError != "" {
				// 强制模拟错误
				ctx = context.WithValue(ctx, "SetExamTimers-force-error", tt.forceError)
			}

			// 初始化全局定时器管理器
			examTimerMgr = NewExamTimerManager(ctx)
			defer examTimerMgr.StopAll()

			// 执行测试
			err := SetExamTimers(ctx, tt.examID)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("期望错误包含 '%s'，但得到 '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到错误: %v", err)
				}
			}

			// 验证定时器设置
			if tt.checkTimers && !tt.expectError {
				examTimerMgr.mutex.Lock()
				startTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_START, testExamSessionID1)
				endTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_END, testExamSessionID1)

				if _, exists := examTimerMgr.timers[startTimerKey]; !exists {
					t.Error("考试场次开始定时器未设置")
				}

				if _, exists := examTimerMgr.timers[endTimerKey]; !exists {
					t.Error("考试场次结束定时器未设置")
				}
				examTimerMgr.mutex.Unlock()

				t.Logf("成功设置定时器，当前定时器数量: %d", len(examTimerMgr.timers))
			}
		})
	}
}

func TestCancelExamTimers(t *testing.T) {
	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer CleanTestExamData(t)

	tests := []struct {
		name          string
		examID        int64
		expectError   bool
		errorContains string
		setupTimers   bool
		forceError    string
	}{
		{
			name:        "正常取消考试定时器",
			examID:      testNormalExamID2,
			expectError: false,
			setupTimers: true,
		},
		{
			name:        "取消不存在的考试定时器",
			examID:      999999,
			expectError: false,
			setupTimers: false,
		},
		{
			name:          "查询考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制查询考试场次信息错误",
			setupTimers:   false,
			forceError:    "CancelExamTimers-force-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceError != "" {
				// 强制模拟错误
				ctx = context.WithValue(ctx, "CancelExamTimers-force-error", tt.forceError)
			}

			// 初始化全局定时器管理器
			examTimerMgr = NewExamTimerManager(ctx)
			defer examTimerMgr.StopAll()

			// 如果需要，先设置定时器
			if tt.setupTimers {
				err := SetExamTimers(context.Background(), tt.examID)
				if err != nil {
					t.Fatalf("设置考试定时器失败: %v", err)
				}

				// 验证定时器已设置
				examTimerMgr.mutex.Lock()
				initialTimerCount := len(examTimerMgr.timers)
				examTimerMgr.mutex.Unlock()

				if initialTimerCount == 0 {
					t.Fatal("定时器设置失败，无法进行取消测试")
				}
				t.Logf("设置了 %d 个定时器", initialTimerCount)
			}

			// 执行测试 - 取消定时器
			err := CancelExamTimers(ctx, tt.examID)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("期望错误包含 '%s'，但得到 '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到错误: %v", err)
				}

				// 验证定时器已取消
				if tt.setupTimers {
					examTimerMgr.mutex.Lock()
					startTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_START, testExamSessionID1)
					endTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_END, testExamSessionID1)

					if _, exists := examTimerMgr.timers[startTimerKey]; exists {
						t.Error("考试场次开始定时器未正确取消")
					}

					if _, exists := examTimerMgr.timers[endTimerKey]; exists {
						t.Error("考试场次结束定时器未正确取消")
					}

					finalTimerCount := len(examTimerMgr.timers)
					examTimerMgr.mutex.Unlock()

					t.Logf("取消定时器后，剩余定时器数量: %d", finalTimerCount)
				}
			}
		})
	}
}
