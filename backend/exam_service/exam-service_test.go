package exam_service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"w2w.io/cmn"
)

const (
	NORMAL_EXAM_ID       = 99901
	ABNORMAL_EXAM_ID     = 99902
	MULTISESSION_EXAM_ID = 99903

	NORMAL_SESSION_ID     = 88801
	ABNORMAL_SESSION_ID   = 88802
	ENDED_SESSION_ID      = 88803
	UNFINISHED_SESSION_ID = 88804
	MULTI_SESSION_1_ID    = 88805
	MULTI_SESSION_2_ID    = 88806

	NORMAL_EXAMINEE_ID          = 77701
	FINISHED_EXAMINEE_ID        = 77702
	UNFINISHED_EXAMINEE_ID      = 77703
	ABSENT_EXAMINEE_ID          = 77704
	MULTI_SESSION_1_EXAMINEE_ID = 77705
	MULTI_SESSION_2_EXAMINEE_ID = 77706

	NORMAL_USER_ID          = 77701
	FINISHED_USER_ID        = 77702
	UNFINISHED_USER_ID      = 77703
	ABSENT_USER_ID          = 77704
	MULTI_SESSION_1_USER_ID = 77705
	MULTI_SESSION_2_USER_ID = 77706

	NORMAL_EXAM_PAPER_ID          = 66601
	ABNORMAL_EXAM_PAPER_ID        = 66602
	ENDED_EXAM_PAPER_ID           = 66603
	UNFINISHED_EXAM_PAPER_ID      = 66604
	MULTI_SESSION_1_EXAM_PAPER_ID = 66605
	MULTI_SESSION_2_EXAM_PAPER_ID = 66606
)

type TestData struct {
	// 时间数据
	PastTime   int64 // 过去时间
	FutureTime int64 // 未来时间
	Now        int64 // 当前时间

	// 考试ID
	NormalExamID       int64
	AbnormalExamID     int64
	MultiSessionExamID int64

	// 场次ID
	NormalSessionID     int64
	AbnormalSessionID   int64
	EndedSessionID      int64
	UnfinishedSessionID int64
	MultiSession1ID     int64
	MultiSession2ID     int64

	// 试卷ID
	NormalExamPaperID        int64
	AbnormalExamPaperID      int64
	EndedExamPaperID         int64
	UnfinishedExamPaperID    int64
	MultiSession1ExamPaperID int64
	MultiSession2ExamPaperID int64

	// 考生ID
	NormalExamineeID        int64
	FinishedExamineeID      int64
	UnfinishedExamineeID    int64
	AbsentExamineeID        int64
	MultiSession1ExamineeID int64
	MultiSession2ExamineeID int64

	// 用户ID
	NormalUserID        int64
	FinishedUserID      int64
	UnfinishedUserID    int64
	AbsentUserID        int64
	MultiSession1UserID int64
	MultiSession2UserID int64
}

var globalTestData *TestData

// 测试辅助函数
func setupTestEnvironment(t *testing.T) {
	conn := cmn.GetRedisConn()
	ctx := context.Background()

	// 清理Redis测试数据
	err := conn.Del(ctx, EXAM_TIMER_SET_KEY).Err()
	require.NoError(t, err)

	// 初始化数据库连接
	if pgxConn == nil {
		pgxConn = cmn.GetPgxConn()
	}
}

func cleanupTestEnvironment(t *testing.T) {

	// 清理测试数据库数据 - 按依赖关系顺序删除
	ctx := context.Background()

	// 删除测试考生数据
	pgxConn.Exec(ctx, "DELETE FROM t_examinee WHERE remark = 'test_data'")

	// 删除测试考试试卷数据
	pgxConn.Exec(ctx, "DELETE FROM t_exam_paper WHERE id IN ($1, $2, $3, $4, $5, $6)", NORMAL_EXAM_PAPER_ID, ABNORMAL_EXAM_PAPER_ID, ENDED_EXAM_PAPER_ID, UNFINISHED_EXAM_PAPER_ID, MULTI_SESSION_1_EXAM_PAPER_ID, MULTI_SESSION_2_EXAM_PAPER_ID)

	// 删除测试考试场次数据
	pgxConn.Exec(ctx, "DELETE FROM t_exam_session WHERE session_num = 999999")

	// 删除测试考试数据
	pgxConn.Exec(ctx, "DELETE FROM t_exam_info WHERE name LIKE 'TEST_EXAM_%'")

	// 删除测试用户数据
	pgxConn.Exec(ctx, "DELETE FROM t_user WHERE remark LIKE 'test_data%'")

	// 重置全局测试数据
	globalTestData = nil
}

// 创建统一的测试数据
func createUnifiedTestData(t *testing.T, ctx context.Context) *TestData {
	if globalTestData != nil {
		return globalTestData
	}

	now := time.Now().UnixMilli()
	data := &TestData{
		PastTime:   now - 300000, // 5分钟前
		FutureTime: now + 300000, // 5分钟后
		Now:        now,

		// 使用常量定义的考试ID
		NormalExamID:       NORMAL_EXAM_ID,
		AbnormalExamID:     ABNORMAL_EXAM_ID,
		MultiSessionExamID: MULTISESSION_EXAM_ID,

		// 使用常量定义的场次ID
		NormalSessionID:     NORMAL_SESSION_ID,
		AbnormalSessionID:   ABNORMAL_SESSION_ID,
		EndedSessionID:      ENDED_SESSION_ID,
		UnfinishedSessionID: UNFINISHED_SESSION_ID,
		MultiSession1ID:     MULTI_SESSION_1_ID,
		MultiSession2ID:     MULTI_SESSION_2_ID,

		// 使用常量定义的试卷ID
		NormalExamPaperID:        NORMAL_EXAM_PAPER_ID,
		AbnormalExamPaperID:      ABNORMAL_EXAM_PAPER_ID,
		MultiSession1ExamPaperID: MULTI_SESSION_1_EXAM_PAPER_ID,
		MultiSession2ExamPaperID: MULTI_SESSION_2_EXAM_PAPER_ID,
		EndedExamPaperID:         ENDED_EXAM_PAPER_ID,
		UnfinishedExamPaperID:    UNFINISHED_EXAM_PAPER_ID,

		// 使用常量定义的考生ID
		NormalExamineeID:        NORMAL_EXAMINEE_ID,
		FinishedExamineeID:      FINISHED_EXAMINEE_ID,
		UnfinishedExamineeID:    UNFINISHED_EXAMINEE_ID,
		AbsentExamineeID:        ABSENT_EXAMINEE_ID,
		MultiSession1ExamineeID: MULTI_SESSION_1_EXAMINEE_ID,
		MultiSession2ExamineeID: MULTI_SESSION_2_EXAMINEE_ID,

		NormalUserID:        NORMAL_EXAMINEE_ID,
		FinishedUserID:      FINISHED_EXAMINEE_ID,
		UnfinishedUserID:    UNFINISHED_EXAMINEE_ID,
		AbsentUserID:        ABSENT_EXAMINEE_ID,
		MultiSession1UserID: MULTI_SESSION_1_EXAMINEE_ID,
		MultiSession2UserID: MULTI_SESSION_2_EXAMINEE_ID,
	}

	// 创建测试用户数据
	createTestUser(t, ctx, data.NormalUserID)
	createTestUser(t, ctx, data.FinishedUserID)
	createTestUser(t, ctx, data.UnfinishedUserID)
	createTestUser(t, ctx, data.AbsentUserID)
	createTestUser(t, ctx, data.MultiSession1UserID)
	createTestUser(t, ctx, data.MultiSession2UserID)

	// 创建考试数据
	createTestExamWithID(t, ctx, data.NormalExamID)
	createTestExamWithID(t, ctx, data.AbnormalExamID)
	createTestExamWithID(t, ctx, data.MultiSessionExamID)

	// 创建场次数据
	createTestExamSessionWithID(t, ctx, data.NormalSessionID, data.NormalExamID, data.PastTime, data.FutureTime)
	createTestExamSessionWithID(t, ctx, data.AbnormalSessionID, data.AbnormalExamID, data.PastTime, data.FutureTime)
	createTestExamSessionWithID(t, ctx, data.EndedSessionID, data.NormalExamID, data.PastTime-600000, data.PastTime)
	createTestExamSessionWithID(t, ctx, data.UnfinishedSessionID, data.NormalExamID, data.PastTime-600000, data.PastTime)
	createTestExamSessionWithID(t, ctx, data.MultiSession1ID, data.MultiSessionExamID, data.Now+5000, data.FutureTime)
	createTestExamSessionWithID(t, ctx, data.MultiSession2ID, data.MultiSessionExamID, data.Now+10000, data.FutureTime+300000)

	// 创建考试试卷数据
	createExamPaper(t, ctx, data.NormalExamPaperID, data.NormalSessionID)
	createExamPaper(t, ctx, data.AbnormalExamPaperID, data.AbnormalSessionID)
	createExamPaper(t, ctx, data.MultiSession1ExamPaperID, data.MultiSession1ID)
	createExamPaper(t, ctx, data.MultiSession2ExamPaperID, data.MultiSession2ID)
	createExamPaper(t, ctx, data.EndedExamPaperID, data.EndedSessionID)
	createExamPaper(t, ctx, data.UnfinishedExamPaperID, data.UnfinishedSessionID)

	// 创建考生数据
	// 为正常场次创建正常考生
	createTestExamineeWithIDs(t, ctx, data.NormalExamineeID, data.NormalUserID, data.NormalSessionID, "00")

	// 为已结束场次创建已完成考生
	createTestExamineeWithIDs(t, ctx, data.FinishedExamineeID, data.FinishedUserID, data.EndedSessionID, "10")

	// 为未完成场次创建未完成考生
	createTestExamineeWithIDs(t, ctx, data.UnfinishedExamineeID, data.UnfinishedUserID, data.UnfinishedSessionID, "00")

	// 为多场次考试创建考生
	createTestExamineeWithIDs(t, ctx, MULTI_SESSION_1_EXAMINEE_ID, data.MultiSession1UserID, data.MultiSession1ID, "00")
	createTestExamineeWithIDs(t, ctx, MULTI_SESSION_2_EXAMINEE_ID, data.MultiSession2UserID, data.MultiSession2ID, "00")

	globalTestData = data
	return data
}

func createTestUser(t *testing.T, ctx context.Context, userID int64) {
	// 创建指定ID的测试用户数据
	now := time.Now().UnixMilli()
	userName := fmt.Sprintf("TEST_USER_%d", userID)

	query := `
		INSERT INTO t_user (
			id, official_name, category, account, create_time, update_time, remark
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err := pgxConn.Exec(ctx, query, userID, userName, "00", fmt.Sprintf("%d", userID), now, now, "test_data")
	require.NoError(t, err, "创建指定ID的测试用户数据失败")
}

func createTestExamWithID(t *testing.T, ctx context.Context, examID int64) int64 {
	// 创建指定ID的测试考试数据
	now := time.Now().UnixMilli()
	examName := fmt.Sprintf("TEST_EXAM_%d", examID)

	query := `
		INSERT INTO t_exam_info (
			id, name, type, mode, status, creator, create_time, update_time
		) VALUES (
			$1, $2, '00', '00', '02', 1, $3, $3
		)
	`

	_, err := pgxConn.Exec(ctx, query, examID, examName, now)
	require.NoError(t, err, "创建指定ID的测试考试数据失败")

	return examID
}

func createTestExamSessionWithID(t *testing.T, ctx context.Context, sessionID, examID int64, startTime, endTime int64) int64 {
	// 创建指定ID的测试考试场次数据
	now := time.Now().UnixMilli()

	query := `
		INSERT INTO t_exam_session (
			id, exam_id, paper_id, mark_method, period_mode, duration, 
			mark_mode, creator, status, session_num, 
			start_time, end_time, create_time, update_time
		) VALUES (
			$1, $2, 1, '02', '00', 120, '00', 1, '02', 999999,
			$3, $4, $5, $5
		)
	`

	_, err := pgxConn.Exec(ctx, query, sessionID, examID, startTime, endTime, now)
	require.NoError(t, err, "创建指定ID的测试考试场次数据失败")

	return sessionID
}

func createTestExaminee(t *testing.T, ctx context.Context, sessionID int64) {
	// 生成固定的测试考生ID
	now := time.Now().UnixMilli()
	studentID := 700000 + (now % 99999) // 确保ID在测试范围内且唯一

	query := `
		INSERT INTO t_examinee (
			student_id, exam_room, exam_session_id, exam_paper_id,
			remark, creator, status, serial_number, examinee_number,
			create_time, update_time
		) VALUES (
			$1, 1, $2, 1, 'test_data', 1, '00', 1, $3, $4, $4
		)
	`

	examineeNumber := fmt.Sprintf("TEST%06d", studentID)
	_, err := pgxConn.Exec(ctx, query, studentID, sessionID, examineeNumber, now)
	require.NoError(t, err, "创建测试考生数据失败")
}

func createTestExamineeWithIDs(t *testing.T, ctx context.Context, id, studentID, examSessionID int64, status string) {
	// 创建指定ID和状态的测试考生数据
	now := time.Now().UnixMilli()

	query := `
		INSERT INTO t_examinee (
			id, student_id, exam_session_id, exam_paper_id,
			remark, creator, status, serial_number, examinee_number,
			create_time, update_time
		) VALUES (
			$1, $2, $3, $4, 'test_data', 1, $5, 1, $6, $7, $7
		)
	`

	examineeNumber := fmt.Sprintf("TEST%06d", studentID)
	_, err := pgxConn.Exec(ctx, query, id, studentID, examSessionID, 1, status, examineeNumber, now)
	require.NoError(t, err, "创建指定ID的测试考生数据失败")
}

func createExamPaper(t *testing.T, ctx context.Context, examPaperID, examSessionID int64) {
	query := `
		INSERT INTO t_exam_paper (
			id, exam_session_id, creator
		) VALUES (
			$1, $2, 1
		)
	`
	_, err := pgxConn.Exec(ctx, query, examPaperID, examSessionID)
	require.NoError(t, err, "创建考试试卷数据失败")
}

// TestExamTimerIntegration 集成测试：测试完整的定时器流程
func TestExamTimerIntegration(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 定义测试用例数组
	testCases := []struct {
		name           string
		examID         int64
		expectedTimers int
		description    string
	}{
		{
			name:           "正常考试定时器设置",
			examID:         testData.NormalExamID,
			expectedTimers: 6,
			description:    "正常考试应该设置对应数量的定时器事件",
		},
		{
			name:           "异常考试定时器设置",
			examID:         testData.AbnormalExamID,
			expectedTimers: 2,
			description:    "异常考试也应该设置定时器事件",
		},
		{
			name:           "多场次考试定时器设置",
			examID:         testData.MultiSessionExamID,
			expectedTimers: 4,
			description:    "多场次考试应该设置4个定时器事件",
		},
	}

	// 遍历测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 清理Redis状态
			conn := cmn.GetRedisConn()
			ctx := context.Background()
			conn.Del(ctx, EXAM_TIMER_SET_KEY)

			// 设置定时器
			err := SetExamTimers(ctx, tc.examID)
			require.NoError(t, err, "设置考试定时器应该成功")

			// 验证Redis中是否正确设置了定时器
			count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTimers, int(count), tc.description)

			// 验证事件内容
			events, err := conn.ZRange(ctx, EXAM_TIMER_SET_KEY, 0, -1).Result()
			require.NoError(t, err)

			startEventCount := 0
			endEventCount := 0

			for _, eventStr := range events {
				var event ExamEvent
				err := json.Unmarshal([]byte(eventStr), &event)
				require.NoError(t, err)

				assert.Equal(t, tc.examID, event.ExamID, "事件应该包含正确的考试ID")

				if event.Type == EVENT_TYPE_EXAM_SESSION_START {
					startEventCount++
				} else if event.Type == EVENT_TYPE_EXAM_SESSION_END {
					endEventCount++
				}
			}

			expectedSessionCount := tc.expectedTimers / 2
			assert.Equal(t, expectedSessionCount, startEventCount, "开始事件数量应该等于场次数量")
			assert.Equal(t, expectedSessionCount, endEventCount, "结束事件数量应该等于场次数量")
		})
	}
}

// TestTimerCancellation 测试定时器取消功能
func TestTimerCancellation(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 定义测试用例数组
	testCases := []struct {
		name           string
		examID         int64
		expectedTimers int
		description    string
	}{
		{
			name:           "正常考试定时器取消",
			examID:         testData.NormalExamID,
			expectedTimers: 6, // 正常考试有3个场次
			description:    "正常考试应该有6个定时器事件",
		},
		{
			name:           "异常考试定时器取消",
			examID:         testData.AbnormalExamID,
			expectedTimers: 2, // 异常考试只有1个场次
			description:    "异常考试应该有2个定时器事件",
		},
		{
			name:           "多场次考试定时器取消",
			examID:         testData.MultiSessionExamID,
			expectedTimers: 4, // 多场次考试有2个场次
			description:    "多场次考试应该有4个定时器事件",
		},
	}

	// 遍历测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 清理Redis状态
			conn := cmn.GetRedisConn()
			ctx := context.Background()
			conn.Del(ctx, EXAM_TIMER_SET_KEY)

			// 设置定时器
			err := SetExamTimers(ctx, tc.examID)
			require.NoError(t, err)

			// 验证定时器已设置
			count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTimers, int(count), tc.description)

			// 取消定时器
			err = CancelExamTimers(ctx, tc.examID)
			require.NoError(t, err)

			// 验证定时器已取消
			count, err = conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
			require.NoError(t, err)
			assert.Equal(t, 0, int(count), "所有定时器应该已被取消")
		})
	}
}

// TestHandleTimerEvent 测试定时器事件处理函数
func TestHandleTimerEvent(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 定义测试用例数组
	testCases := []struct {
		name           string
		eventType      string
		sessionID      int64
		examID         int64
		expectedStatus string
		statusField    string // "exam" 或 "session"
		description    string
		setupFunc      func()
	}{
		{
			name:           "处理开始事件-有考生",
			eventType:      EVENT_TYPE_EXAM_SESSION_START,
			sessionID:      testData.NormalSessionID,
			examID:         testData.NormalExamID,
			expectedStatus: "04",
			statusField:    "exam",
			description:    "开始事件应该将考试状态更新为进行中",
			setupFunc: func() {
				// 保持默认的待开始状态
			},
		},
		{
			name:           "处理开始事件-无考生",
			eventType:      EVENT_TYPE_EXAM_SESSION_START,
			sessionID:      testData.AbnormalSessionID,
			examID:         testData.AbnormalExamID,
			expectedStatus: "10",
			statusField:    "exam",
			description:    "无考生的开始事件应该将考试状态更新为异常",
			setupFunc: func() {
				// 保持默认的待开始状态
			},
		},
		{
			name:           "处理结束事件-考生已完成",
			eventType:      EVENT_TYPE_EXAM_SESSION_END,
			sessionID:      testData.EndedSessionID,
			examID:         testData.NormalExamID,
			expectedStatus: "06",
			statusField:    "session",
			description:    "结束事件应该将场次状态更新为已结束",
			setupFunc: func() {
				// 设置为进行中状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
		},
		{
			name:           "处理结束事件-考生未完成",
			eventType:      EVENT_TYPE_EXAM_SESSION_END,
			sessionID:      testData.UnfinishedSessionID,
			examID:         testData.NormalExamID,
			expectedStatus: "06",
			statusField:    "session",
			description:    "结束事件应该自动处理未完成考生并将场次状态更新为已结束",
			setupFunc: func() {
				// 设置为进行中状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.UnfinishedSessionID)
			},
		},
	}

	// 遍历测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行setup函数
			tc.setupFunc()

			// 创建事件数组（批量处理需要数组）
			events := []ExamEvent{
				{
					Type:          tc.eventType,
					ExamID:        tc.examID,
					ExamSessionID: tc.sessionID,
				},
			}

			// 调用批量事件处理函数
			if tc.eventType == EVENT_TYPE_EXAM_SESSION_START {
				handleExamSessionStartBatch(ctx, events)
			} else if tc.eventType == EVENT_TYPE_EXAM_SESSION_END {
				handleExamSessionEndBatch(ctx, events)
			}

			// 验证状态更新
			var actualStatus string
			var err error

			if tc.statusField == "exam" {
				err = pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", tc.examID).Scan(&actualStatus)
			} else {
				err = pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", tc.sessionID).Scan(&actualStatus)
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, actualStatus, tc.description)

			t.Logf("✅ %s 测试通过", tc.name)
		})
	}

	// 测试未知事件类型
	t.Run("处理未知事件类型", func(t *testing.T) {
		events := []ExamEvent{
			{
				Type:          "unknown_event",
				ExamID:        testData.NormalExamID,
				ExamSessionID: testData.NormalSessionID,
			},
		}

		// 调用批量处理函数
		require.NotPanics(t, func() {
			handleExamSessionStartBatch(ctx, events)
		}, "未知事件类型不应该导致panic")

		require.NotPanics(t, func() {
			handleExamSessionEndBatch(ctx, events)
		}, "未知事件类型不应该导致panic")

		t.Log("✅ 未知事件类型处理测试通过")
	})
}

// TestHandleExamSessionStartBatch 测试批量处理考试场次开始事件
func TestHandleExamSessionStartBatch(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 定义测试用例数组
	testCases := []struct {
		name        string
		events      []ExamEvent
		description string
		forceError  string
		expectError bool
		checkLogs   bool
	}{
		{
			name: "批量处理-混合场景",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.AbnormalSessionID,
					ExamID:        testData.AbnormalExamID,
				},
			},
			description: "批量处理混合场景：有考生的正常开始，无考生的设为异常",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
		},
		{
			name: "批量处理-多场次正常场景",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.MultiSession1ID,
					ExamID:        testData.MultiSessionExamID,
				},
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.MultiSession2ID,
					ExamID:        testData.MultiSessionExamID,
				},
			},
			description: "批量处理多场次正常场景：所有场次都有考生，应该正常开始",
			forceError:  "",
			expectError: false,
		},
		{
			name: "强制错误-查询考试场次信息失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			description: "测试查询考试场次信息失败的错误处理",
			forceError:  "queryExamSessions",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-扫描场次信息失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			description: "测试扫描场次信息失败的错误处理",
			forceError:  "scanExamSessionInfo",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-更新异常考试失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.AbnormalSessionID,
					ExamID:        testData.AbnormalExamID,
				},
			},
			description: "测试更新异常考试状态失败的错误处理",
			forceError:  "updateAbnormalExams",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-取消异常考试定时器失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.AbnormalSessionID,
					ExamID:        testData.AbnormalExamID,
				},
			},
			description: "测试取消异常考试定时器失败的错误处理",
			forceError:  "cancelAbnormalExamTimers",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-更新正常考试场次失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			description: "测试更新正常考试场次状态失败的错误处理",
			forceError:  "updateNormalExamSessions",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-更新正常考试失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			description: "测试更新正常考试状态失败的错误处理",
			forceError:  "updateNormalExams",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-panic",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_START,
					ExamSessionID: testData.NormalSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			description: "强制错误-panic",
			forceError:  "panic",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 重置测试数据状态
			pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id IN ($1, $2, $3)",
				testData.NormalExamID, testData.AbnormalExamID, testData.MultiSessionExamID)
			pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id IN ($1, $2, $3, $4)",
				testData.NormalSessionID, testData.AbnormalSessionID, testData.MultiSession1ID, testData.MultiSession2ID)

			// 为异常场景清理考生数据
			if tc.forceError == "updateAbnormalExams" || tc.forceError == "cancelAbnormalExamTimers" {
				pgxConn.Exec(ctx, "DELETE FROM t_examinee WHERE exam_session_id = $1", testData.AbnormalSessionID)
			}

			// 日志捕获
			var observedZapCore zapcore.Core
			var observedLogs *observer.ObservedLogs
			var originalLogger *zap.Logger

			if tc.checkLogs {
				observedZapCore, observedLogs = observer.New(zap.ErrorLevel)
				observedLogger := zap.New(observedZapCore)
				originalLogger = z
				z = observedLogger
				defer func() {
					z = originalLogger
				}()
			}

			// 设置错误注入
			testCtx := ctx
			if tc.forceError != "" {
				testCtx = context.WithValue(ctx, "handleExamSessionStartBatch-force-error", tc.forceError)
			}

			// 执行批量处理 - 使用defer recover来捕获panic
			handleExamSessionStartBatch(testCtx, tc.events)

			// 如果期望错误，则检查日志中的错误信息
			if tc.expectError && tc.checkLogs {
				allLogs := observedLogs.All()
				foundExpectedError := false

				for _, logEntry := range allLogs {
					switch tc.forceError {
					case "queryExamSessions":
						if strings.Contains(logEntry.Message, "批量查询场次信息失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "scanExamSessionInfo":
						if strings.Contains(logEntry.Message, "扫描场次信息失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "updateAbnormalExams":
						if strings.Contains(logEntry.Message, "批量更新考试为异常状态失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "cancelAbnormalExamTimers":
						if strings.Contains(logEntry.Message, "取消考试定时器失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "updateNormalExamSessions":
						if strings.Contains(logEntry.Message, "批量更新场次状态失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "updateNormalExams":
						if strings.Contains(logEntry.Message, "批量更新考试状态失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					}
				}

				if !foundExpectedError {
					t.Errorf("期望在日志中找到 %s 相关的错误信息，但未找到", tc.forceError)
				}
			}

			// 如果不是强制错误测试，验证正常结果
			if !tc.expectError {
				// 验证结果
				for _, event := range tc.events {
					// 检查考生数量来判断是正常还是异常场景
					var examineeCount int
					err := pgxConn.QueryRow(ctx, `
						SELECT COUNT(*) FROM t_examinee 
						WHERE exam_session_id = $1 AND status != '08'
					`, event.ExamSessionID).Scan(&examineeCount)
					require.NoError(t, err)

					if examineeCount == 0 {
						// 无考生场景 - 应该设为异常
						var examStatus string
						err := pgxConn.QueryRow(ctx, `
							SELECT status FROM t_exam_info WHERE id = $1
						`, event.ExamID).Scan(&examStatus)
						require.NoError(t, err)
						assert.Equal(t, "10", examStatus, "无考生的考试应该设为异常状态")
					} else {
						// 有考生场景 - 应该正常开始
						var sessionStatus, examStatus string
						err := pgxConn.QueryRow(ctx, `
							SELECT status FROM t_exam_session WHERE id = $1
						`, event.ExamSessionID).Scan(&sessionStatus)
						require.NoError(t, err)

						err = pgxConn.QueryRow(ctx, `
							SELECT status FROM t_exam_info WHERE id = $1
						`, event.ExamID).Scan(&examStatus)
						require.NoError(t, err)

						assert.Equal(t, "04", sessionStatus, "场次状态应该更新为进行中")
						assert.Equal(t, "04", examStatus, "考试状态应该更新为进行中")
					}
				}
			} else {
				t.Logf("强制错误测试 %s 完成：%s", tc.forceError, tc.description)
			}
		})
	}
}

// TestHandleExamSessionEndBatch 测试批量处理考试场次结束事件
func TestHandleExamSessionEndBatch(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 定义测试用例数组
	testCases := []struct {
		name        string
		events      []ExamEvent
		setupFunc   func()
		description string
		forceError  string
		expectError bool
		checkLogs   bool
	}{
		{
			name: "批量结束-已完成场次",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.EndedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				// 设置为进行中状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			description: "已完成考生的场次应该直接结束",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
		},
		{
			name: "批量结束-未完成场次",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.UnfinishedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				// 设置为进行中状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.UnfinishedSessionID)
			},
			description: "有未完成考生的场次结束时，应该自动处理考生状态并结束场次",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
		},
		{
			name: "强制错误-更新考生状态失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.EndedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			description: "测试更新考生状态失败的错误处理",
			forceError:  "updateExaminees",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-检查未完成考生失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.UnfinishedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.UnfinishedSessionID)
			},
			description: "测试检查未完成考生失败的错误处理",
			forceError:  "checkUnfinishedExaminees",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-扫描未结束考生失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.UnfinishedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.UnfinishedSessionID)
			},
			description: "测试扫描未结束考生失败的错误处理",
			forceError:  "scanUnfinishedExaminees",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-更新场次状态为已结束失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.EndedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			description: "测试更新场次状态为已结束失败的错误处理",
			forceError:  "updateSessionEndStatus",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-更新考试状态为已结束失败",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.EndedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			description: "测试更新考试状态为已结束失败的错误处理",
			forceError:  "updateExamEndStatus",
			expectError: true,
			checkLogs:   true,
		},
		{
			name: "强制错误-panic",
			events: []ExamEvent{
				{
					Type:          EVENT_TYPE_EXAM_SESSION_END,
					ExamSessionID: testData.EndedSessionID,
					ExamID:        testData.NormalExamID,
				},
			},
			setupFunc: func() {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			description: "强制错误-panic",
			forceError:  "panic",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行setup函数
			tc.setupFunc()

			// 日志捕获
			var observedZapCore zapcore.Core
			var observedLogs *observer.ObservedLogs
			var originalLogger *zap.Logger

			if tc.checkLogs {
				observedZapCore, observedLogs = observer.New(zap.ErrorLevel)
				observedLogger := zap.New(observedZapCore)
				originalLogger = z
				z = observedLogger
				defer func() {
					z = originalLogger
				}()
			}

			// 设置错误注入
			testCtx := ctx
			if tc.forceError != "" {
				testCtx = context.WithValue(ctx, "handleExamSessionEndBatch-force-error", tc.forceError)
			}

			// 执行批量处理
			handleExamSessionEndBatch(testCtx, tc.events)

			// 如果期望错误，则检查日志中的错误信息
			if tc.expectError && tc.checkLogs {
				allLogs := observedLogs.All()
				foundExpectedError := false

				for _, logEntry := range allLogs {
					switch tc.forceError {
					case "updateExaminees":
						if strings.Contains(logEntry.Message, "批量更新考生状态失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "checkUnfinishedExaminees":
						if strings.Contains(logEntry.Message, "批量检查未结束考生失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "scanUnfinishedExaminees":
						if strings.Contains(logEntry.Message, "获取未结束的考生数量失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "updateSessionEndStatus":
						if strings.Contains(logEntry.Message, "批量更新场次状态为已结束失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "updateExamEndStatus":
						if strings.Contains(logEntry.Message, "批量更新考试状态为已结束失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					}
				}

				if !foundExpectedError {
					t.Errorf("未在日志中找到预期的错误信息，forceError: %s", tc.forceError)
					for _, logEntry := range allLogs {
						t.Logf("实际日志: %s", logEntry.Message)
					}
				} else {
					t.Logf("%s: 错误处理测试通过 - %s", tc.name, tc.description)
				}
				return
			}

			// 验证正常情况的结果
			if !tc.expectError {
				for _, event := range tc.events {
					// 检查场次状态 - 应该都已结束
					var sessionStatus string
					err := pgxConn.QueryRow(ctx, `
						SELECT status FROM t_exam_session WHERE id = $1
					`, event.ExamSessionID).Scan(&sessionStatus)
					require.NoError(t, err)
					assert.Equal(t, "06", sessionStatus, "场次状态应该更新为已结束")

					// 验证考生状态是否被正确处理
					if event.ExamSessionID == testData.UnfinishedSessionID {
						// 对于原本未完成的考生，应该被自动更新为缺考或已交卷状态
						var unfinishedCount int
						err := pgxConn.QueryRow(ctx, `
							SELECT COUNT(*) FROM t_examinee 
							WHERE exam_session_id = $1 AND status NOT IN ('02', '06', '08', '10', '12')
						`, event.ExamSessionID).Scan(&unfinishedCount)
						require.NoError(t, err)
						assert.Equal(t, 0, unfinishedCount, "所有考生应该已被处理，不应该有未完成状态的考生")

						// 验证考生被更新为合适的结束状态
						var processedCount int
						err = pgxConn.QueryRow(ctx, `
							SELECT COUNT(*) FROM t_examinee 
							WHERE exam_session_id = $1 AND status IN ('02', '06', '08', '10', '12')
						`, event.ExamSessionID).Scan(&processedCount)
						require.NoError(t, err)
						assert.Greater(t, processedCount, 0, "应该有考生被更新为结束状态")
					}
				}
				t.Logf("%s: 正常流程测试通过 - %s", tc.name, tc.description)
			}
		})
	}
}

// TestAutoUpdateExamineeStatusOnSessionEnd 测试场次结束时自动更新考生状态
func TestAutoUpdateExamineeStatusOnSessionEnd(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	// 设置场次为进行中状态
	pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
	pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.UnfinishedSessionID)

	// 验证处理前的考生状态
	var beforeUnfinishedCount int
	err := pgxConn.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_examinee 
		WHERE exam_session_id = $1 AND status = '00'
	`, testData.UnfinishedSessionID).Scan(&beforeUnfinishedCount)
	require.NoError(t, err)
	assert.Greater(t, beforeUnfinishedCount, 0, "处理前应该有正在考试的考生")

	// 创建结束事件并处理
	events := []ExamEvent{
		{
			Type:          EVENT_TYPE_EXAM_SESSION_END,
			ExamSessionID: testData.UnfinishedSessionID,
			ExamID:        testData.NormalExamID,
		},
	}

	// 执行批量处理
	handleExamSessionEndBatch(ctx, events)

	// 验证处理后的考生状态
	var afterUnfinishedCount int
	err = pgxConn.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_examinee 
		WHERE exam_session_id = $1 AND status = '00'
	`, testData.UnfinishedSessionID).Scan(&afterUnfinishedCount)
	require.NoError(t, err)
	assert.Equal(t, 0, afterUnfinishedCount, "处理后不应该有正在考试的考生")

	// 验证考生被更新为结束状态（缺考或已交卷）
	var finishedCount int
	err = pgxConn.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_examinee 
		WHERE exam_session_id = $1 AND status IN ('02', '06', '08', '10', '12')
	`, testData.UnfinishedSessionID).Scan(&finishedCount)
	require.NoError(t, err)
	assert.Equal(t, beforeUnfinishedCount, finishedCount, "所有原本未完成的考生都应该被更新为结束状态")

	// 验证场次状态
	var sessionStatus string
	err = pgxConn.QueryRow(ctx, `
		SELECT status FROM t_exam_session WHERE id = $1
	`, testData.UnfinishedSessionID).Scan(&sessionStatus)
	require.NoError(t, err)
	assert.Equal(t, "06", sessionStatus, "场次状态应该更新为已结束")
}

// TestHandleTimerEventsBatch 测试批量处理定时器事件
func TestHandleTimerEventsBatch(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	tests := []struct {
		name         string
		description  string
		eventStrings []string
		setupFunc    func(*testing.T)
		verifyFunc   func(*testing.T)
		expectPanic  bool
		forceError   string
		expectError  bool
	}{
		{
			name:         "空事件列表",
			description:  "处理空的事件字符串列表应该正常返回",
			eventStrings: []string{},
			setupFunc:    func(t *testing.T) {},
			verifyFunc: func(t *testing.T) {
				// 没有事件处理，验证数据库状态未改变
				var examStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", testData.NormalExamID).Scan(&examStatus)
				require.NoError(t, err)
				assert.Equal(t, "02", examStatus, "考试状态应该保持不变")
			},
		},
		{
			name:        "单个场次开始事件",
			description: "处理单个考试场次开始事件",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.NormalExamID, testData.NormalSessionID),
			},
			setupFunc: func(t *testing.T) {
				// 确保考试和场次状态为待开始
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.NormalSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证考试状态更新为进行中
				var examStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", testData.NormalExamID).Scan(&examStatus)
				require.NoError(t, err)
				assert.Equal(t, "04", examStatus, "考试状态应该更新为进行中")

				// 验证场次状态更新为进行中
				var sessionStatus string
				err = pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.NormalSessionID).Scan(&sessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "04", sessionStatus, "场次状态应该更新为进行中")
			},
		},
		{
			name:        "单个场次结束事件",
			description: "处理单个考试场次结束事件",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_END, testData.NormalExamID, testData.EndedSessionID),
			},
			setupFunc: func(t *testing.T) {
				// 设置考试和场次为进行中状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证场次状态更新为已结束
				var sessionStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.EndedSessionID).Scan(&sessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "06", sessionStatus, "场次状态应该更新为已结束")
			},
		},
		{
			name:        "混合事件类型",
			description: "同时处理开始和结束事件",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.MultiSessionExamID, testData.MultiSession1ID),
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_END, testData.NormalExamID, testData.EndedSessionID),
			},
			setupFunc: func(t *testing.T) {
				// 设置多场次考试状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.MultiSessionExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.MultiSession1ID)

				// 设置要结束的场次状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04' WHERE id = $1", testData.EndedSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证开始事件的处理结果
				var multiSessionStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.MultiSession1ID).Scan(&multiSessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "04", multiSessionStatus, "多场次考试的场次状态应该更新为进行中")

				// 验证结束事件的处理结果
				var endedSessionStatus string
				err = pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.EndedSessionID).Scan(&endedSessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "06", endedSessionStatus, "已结束场次状态应该更新为已结束")
			},
		},
		{
			name:        "无效JSON格式",
			description: "处理包含无效JSON的事件字符串",
			eventStrings: []string{
				`{"type":"exam_session_start","exam_id":123}`, // 有效事件
				`{invalid json}`, // 无效JSON
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.NormalExamID, testData.NormalSessionID), // 另一个有效事件
			},
			setupFunc: func(t *testing.T) {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.NormalSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证有效事件仍然被处理
				var sessionStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.NormalSessionID).Scan(&sessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "04", sessionStatus, "有效的开始事件应该被处理")
			},
		},
		{
			name:        "未知事件类型",
			description: "处理包含未知事件类型的事件",
			eventStrings: []string{
				`{"type":"unknown_event_type","exam_id":123,"exam_session_id":456}`,
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.NormalExamID, testData.NormalSessionID),
			},
			setupFunc: func(t *testing.T) {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.NormalSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证已知事件类型仍然被处理
				var sessionStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", testData.NormalSessionID).Scan(&sessionStatus)
				require.NoError(t, err)
				assert.Equal(t, "04", sessionStatus, "已知的开始事件应该被处理")
			},
		},
		{
			name:        "无考生的场次开始事件",
			description: "处理没有考生的场次开始事件，应该将考试设为异常状态",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.AbnormalExamID, testData.AbnormalSessionID),
			},
			setupFunc: func(t *testing.T) {
				// 确保没有考生数据
				pgxConn.Exec(ctx, "DELETE FROM t_examinee WHERE exam_session_id = $1", testData.AbnormalSessionID)
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.AbnormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.AbnormalSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证考试状态更新为异常状态
				var examStatus string
				err := pgxConn.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", testData.AbnormalExamID).Scan(&examStatus)
				require.NoError(t, err)
				assert.Equal(t, "10", examStatus, "没有考生的考试应该设为异常状态")
			},
		},
		{
			name:        "强制错误-JSON解析失败",
			description: "测试JSON解析失败的错误处理",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.NormalExamID, testData.NormalSessionID),
			},
			setupFunc: func(t *testing.T) {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.NormalSessionID)
			},
			verifyFunc: func(t *testing.T) {
				// 验证函数正确处理了JSON解析错误，没有崩溃
				t.Log("JSON解析错误测试完成")
			},
			forceError:  "unmarshalEvent",
			expectError: true,
		},
		{
			name:        "强制错误-panic",
			description: "强制错误-panic",
			eventStrings: []string{
				fmt.Sprintf(`{"type":"%s","exam_id":%d,"exam_session_id":%d}`,
					EVENT_TYPE_EXAM_SESSION_START, testData.NormalExamID, testData.NormalSessionID),
			},
			setupFunc: func(t *testing.T) {
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE id = $1", testData.NormalSessionID)
			},
			forceError:  "panic",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试前置条件
			tt.setupFunc(t)

			// 设置强制错误上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "handleTimerEventsBatch-force-error", tt.forceError)
			} else if tt.name == "强制错误-JSON解析失败" {
				// 保持向后兼容
				testCtx = context.WithValue(ctx, "handleTimerEventsBatch-force-error", "unmarshalEvent")
			}

			// 如果期望panic，使用defer recover
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("%s: 期望发生panic，但实际没有panic", tt.description)
					}
				}()
			}

			// 执行测试
			handleTimerEventsBatch(testCtx, tt.eventStrings)

			// 如果期望错误，则主要验证错误处理逻辑
			if tt.expectError {
				t.Logf("错误分支测试完成: %s", tt.description)
				return
			}

			// 验证结果
			if !tt.expectPanic && tt.verifyFunc != nil {
				tt.verifyFunc(t)
			}

			t.Logf("%s: 测试完成 - %s", tt.name, tt.description)
		})
	}
}

// TestSetExamTimers
func TestSetExamTimers(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	tests := []struct {
		name        string
		examID      int64
		forceError  string
		expectError bool
		description string
	}{
		{
			name:        "正常设置考试定时器",
			examID:      testData.NormalExamID,
			forceError:  "",
			expectError: false,
			description: "正常考试应该成功设置定时器",
		},
		{
			name:        "不存在的考试ID",
			examID:      999999,
			forceError:  "",
			expectError: false,
			description: "不存在的考试ID应该正常返回但不设置定时器",
		},
		{
			name:        "强制查询考试场次信息失败",
			examID:      testData.NormalExamID,
			forceError:  "queryExamSessions",
			expectError: true,
			description: "查询考试场次信息失败时应该返回错误",
		},
		{
			name:        "强制获取考试场次信息失败",
			examID:      testData.NormalExamID,
			forceError:  "scanExamSessionInfo",
			expectError: true,
			description: "获取考试场次信息失败时应该返回错误",
		},
		{
			name:        "多场次考试正常设置",
			examID:      testData.MultiSessionExamID,
			forceError:  "",
			expectError: false,
			description: "多场次考试应该成功设置所有场次的定时器",
		},
		{
			name:        "异常考试设置定时器",
			examID:      testData.AbnormalExamID,
			forceError:  "",
			expectError: false,
			description: "异常考试也应该能正常设置定时器",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理Redis状态
			conn := cmn.GetRedisConn()
			conn.Del(ctx, EXAM_TIMER_SET_KEY)

			// 创建带强制错误的上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "SetExamTimers-force-error", tt.forceError)
			}

			// 执行SetExamTimers
			err := SetExamTimers(testCtx, tt.examID)

			// 验证错误结果
			if tt.expectError {
				assert.Error(t, err, tt.description)
				t.Logf("%s: 成功捕获预期错误: %v", tt.name, err)
			} else {
				assert.NoError(t, err, tt.description)

				// 对于成功的情况，验证Redis中的定时器数量
				if tt.forceError == "" {
					count, redisErr := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
					require.NoError(t, redisErr)

					// 根据考试ID验证预期的定时器数量
					switch tt.examID {
					case testData.NormalExamID:
						assert.Equal(t, 6, int(count), "正常考试应该有6个定时器事件（3个场次 x 2个事件类型）")
					case testData.MultiSessionExamID:
						assert.Equal(t, 4, int(count), "多场次考试应该有4个定时器事件（2个场次 x 2个事件类型）")
					case testData.AbnormalExamID:
						assert.Equal(t, 2, int(count), "异常考试应该有2个定时器事件（1个场次 x 2个事件类型）")
					case 999999:
						assert.Equal(t, 0, int(count), "不存在的考试不应该设置任何定时器")
					}

					// 验证事件内容的正确性
					if count > 0 {
						events, err := conn.ZRange(ctx, EXAM_TIMER_SET_KEY, 0, -1).Result()
						require.NoError(t, err)

						startEventCount := 0
						endEventCount := 0

						for _, eventStr := range events {
							var event ExamEvent
							err := json.Unmarshal([]byte(eventStr), &event)
							require.NoError(t, err, "事件JSON应该能正确解析")

							assert.Equal(t, tt.examID, event.ExamID, "事件应该包含正确的考试ID")
							assert.NotZero(t, event.ExamSessionID, "事件应该包含有效的场次ID")

							switch event.Type {
							case EVENT_TYPE_EXAM_SESSION_START:
								startEventCount++
							case EVENT_TYPE_EXAM_SESSION_END:
								endEventCount++
							default:
								t.Errorf("未知的事件类型: %s", event.Type)
							}
						}

						assert.Equal(t, startEventCount, endEventCount, "开始事件和结束事件数量应该相等")
					}
				}

				t.Logf("%s: 测试通过 - %s", tt.name, tt.description)
			}
		})
	}
}

// TestCancelExamTimers
func TestInitializeExamTimers(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	tests := []struct {
		name        string
		forceError  string
		expectError bool
		checkLogs   bool
		description string
		setupFunc   func(t *testing.T, ctx context.Context) // 测试前的设置函数
		verifyFunc  func(t *testing.T, ctx context.Context) // 测试后的验证函数
	}{
		{
			name:        "正常初始化考试定时器",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
			description: "应该成功初始化所有符合条件的考试定时器",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)

				// 设置考试和场次为符合条件的状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id IN ($1, $2, $3)",
					testData.NormalExamID, testData.AbnormalExamID, testData.MultiSessionExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE exam_id IN ($1, $2, $3)",
					testData.NormalExamID, testData.AbnormalExamID, testData.MultiSessionExamID)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中是否设置了正确数量的定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)

				// 验证定时器事件的内容
				events, err := conn.ZRange(ctx, EXAM_TIMER_SET_KEY, 0, -1).Result()
				require.NoError(t, err)

				startEventCount := 0
				endEventCount := 0
				examIDSet := make(map[int64]bool)

				for _, eventStr := range events {
					var event ExamEvent
					err := json.Unmarshal([]byte(eventStr), &event)
					require.NoError(t, err)

					examIDSet[event.ExamID] = true

					if event.Type == EVENT_TYPE_EXAM_SESSION_START {
						startEventCount++
					} else if event.Type == EVENT_TYPE_EXAM_SESSION_END {
						endEventCount++
					}
				}

				// 验证设置了正确的考试
				assert.Contains(t, examIDSet, testData.NormalExamID, "应该包含正常考试")
				assert.Contains(t, examIDSet, testData.AbnormalExamID, "应该包含异常考试")
				assert.Contains(t, examIDSet, testData.MultiSessionExamID, "应该包含多场次考试")

				assert.Greater(t, int(count), 0, "应该有定时器被设置")
			},
		},
		{
			name:        "强制查询考试场次信息失败",
			forceError:  "queryExamSessions",
			expectError: false, // initializeExamTimers 在出错时只记录日志，不返回错误
			checkLogs:   true,
			description: "查询考试场次信息失败时应该记录错误日志并返回",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中没有设置任何定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)
				assert.Equal(t, int64(0), count, "查询失败时不应该设置任何定时器")
			},
		},
		{
			name:        "强制获取考试场次信息错误",
			forceError:  "scanExamSessionInfo",
			expectError: false,
			checkLogs:   true,
			description: "获取考试场次信息失败时应该记录错误日志并返回",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)

				// 设置考试和场次为符合条件的状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02' WHERE exam_id = $1", testData.NormalExamID)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中没有设置任何定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)
				assert.Equal(t, int64(0), count, "扫描失败时不应该设置任何定时器")
			},
		},
		{
			name:        "强制panic错误",
			forceError:  "panic",
			expectError: false,
			checkLogs:   true,
			description: "panic错误应该被recover并记录日志",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
			},
		},
		{
			name:        "没有符合条件的考试场次",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
			description: "没有符合条件的考试场次时应该正常返回",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)

				// 设置所有考试为已结束状态，同时设置场次也为已结束状态
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '06' WHERE id IN ($1, $2, $3)",
					testData.NormalExamID, testData.AbnormalExamID, testData.MultiSessionExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '06' WHERE exam_id IN ($1, $2, $3)",
					testData.NormalExamID, testData.AbnormalExamID, testData.MultiSessionExamID)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中没有设置任何定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)
				assert.Equal(t, int64(0), count, "没有符合条件的场次时不应该设置任何定时器")
			},
		},
		{
			name:        "只有过去时间的场次",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
			description: "只有过去时间的场次不应该设置定时器",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)

				// 设置考试和场次为符合条件的状态，但时间都在过去
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '02' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '02', start_time = $1, end_time = $2 WHERE exam_id = $3",
					testData.PastTime-3600000, testData.PastTime-1800000, testData.NormalExamID)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中没有设置任何定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)
				assert.Equal(t, int64(0), count, "过去时间的场次不应该设置定时器")
			},
		},
		{
			name:        "混合时间场次-部分过期",
			forceError:  "",
			expectError: false,
			checkLogs:   false,
			description: "对于混合时间的场次，只有未来时间的事件应该被设置",
			setupFunc: func(t *testing.T, ctx context.Context) {
				// 清理Redis状态
				conn := cmn.GetRedisConn()
				conn.Del(ctx, EXAM_TIMER_SET_KEY)

				// 设置考试和场次：开始时间已过，结束时间未来
				pgxConn.Exec(ctx, "UPDATE t_exam_info SET status = '04' WHERE id = $1", testData.NormalExamID)
				pgxConn.Exec(ctx, "UPDATE t_exam_session SET status = '04', start_time = $1, end_time = $2 WHERE exam_id = $3",
					testData.PastTime-1800000, testData.FutureTime, testData.NormalExamID)
			},
			verifyFunc: func(t *testing.T, ctx context.Context) {
				// 验证Redis中只设置了结束定时器
				conn := cmn.GetRedisConn()
				count, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
				require.NoError(t, err)

				events, err := conn.ZRange(ctx, EXAM_TIMER_SET_KEY, 0, -1).Result()
				require.NoError(t, err)

				endEventCount := 0
				startEventCount := 0

				for _, eventStr := range events {
					var event ExamEvent
					err := json.Unmarshal([]byte(eventStr), &event)
					require.NoError(t, err)

					if event.Type == EVENT_TYPE_EXAM_SESSION_START {
						startEventCount++
					} else if event.Type == EVENT_TYPE_EXAM_SESSION_END {
						endEventCount++
					}
				}

				assert.Equal(t, 0, startEventCount, "过去的开始时间不应该设置定时器")
				assert.Greater(t, endEventCount, 0, "未来的结束时间应该设置定时器")
				assert.Equal(t, int64(endEventCount), count, "定时器总数应该等于结束事件数量")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行测试前的设置
			if tt.setupFunc != nil {
				tt.setupFunc(t, ctx)
			}

			// 日志捕获
			var observedZapCore zapcore.Core
			var observedLogs *observer.ObservedLogs
			var originalLogger *zap.Logger

			if tt.checkLogs {
				observedZapCore, observedLogs = observer.New(zap.ErrorLevel)
				observedLogger := zap.New(observedZapCore)
				originalLogger = z
				z = observedLogger
				defer func() {
					z = originalLogger
				}()
			}

			// 创建带强制错误的上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "initializeExamTimers-force-error", tt.forceError)
			}

			// 执行initializeExamTimers
			initializeExamTimers(testCtx)

			// 如果需要检查日志，则验证错误信息
			if tt.checkLogs {
				allLogs := observedLogs.All()
				foundExpectedError := false

				for _, logEntry := range allLogs {
					switch tt.forceError {
					case "queryExamSessions":
						if strings.Contains(logEntry.Message, "查询考试场次信息失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "scanExamSessionInfo":
						if strings.Contains(logEntry.Message, "获取考试场次信息失败") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					case "panic":
						if strings.Contains(logEntry.Message, "Panic recovered in initializeExamTimers") {
							foundExpectedError = true
							t.Logf("成功在日志中捕获到预期错误: %s", logEntry.Message)
						}
					}
				}

				if tt.forceError != "" {
					assert.True(t, foundExpectedError, "应该在日志中找到预期的错误信息: %s", tt.forceError)
				}
			}

			// 执行测试后的验证
			if tt.verifyFunc != nil {
				tt.verifyFunc(t, ctx)
			}

			t.Logf("%s: 测试通过 - %s", tt.name, tt.description)
		})
	}
}

func TestCancelExamTimers(t *testing.T) {
	cmn.ConfigureForTest()
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	ctx := context.Background()
	testData := createUnifiedTestData(t, ctx)

	tests := []struct {
		name        string
		examID      int64
		forceError  string
		expectError bool
		description string
		setupFunc   func(t *testing.T, ctx context.Context, examID int64) // 测试前的设置函数
	}{
		{
			name:        "正常取消考试定时器",
			examID:      testData.NormalExamID,
			forceError:  "",
			expectError: false,
			description: "正常考试应该成功取消定时器",
			setupFunc: func(t *testing.T, ctx context.Context, examID int64) {
				// 先设置定时器，然后取消
				err := SetExamTimers(ctx, examID)
				require.NoError(t, err, "设置定时器应该成功")
			},
		},
		{
			name:        "不存在的考试ID",
			examID:      999999,
			forceError:  "",
			expectError: false, // 不存在考试ID时函数不会报错，只是不取消任何定时器
			description: "不存在的考试ID应该正常返回但不取消任何定时器",
			setupFunc:   nil,
		},
		{
			name:        "强制查询考试场次信息失败",
			examID:      testData.NormalExamID,
			forceError:  "queryExamSessions",
			expectError: true,
			description: "查询考试场次信息失败时应该返回错误",
			setupFunc:   nil,
		},
		{
			name:        "强制无场次ID情况",
			examID:      testData.NormalExamID,
			forceError:  "noSessionIDs",
			expectError: false, // 无场次ID时返回nil，不报错
			description: "无场次ID时应该正常返回",
			setupFunc:   nil,
		},
		{
			name:        "强制执行Lua脚本错误",
			examID:      testData.NormalExamID,
			forceError:  "evalLuaScriptError",
			expectError: true,
			description: "执行Lua脚本失败时应该返回错误",
			setupFunc: func(t *testing.T, ctx context.Context, examID int64) {
				// 先设置定时器
				err := SetExamTimers(ctx, examID)
				require.NoError(t, err, "设置定时器应该成功")
			},
		},
		{
			name:        "强制获取已删除计数器数量错误",
			examID:      testData.NormalExamID,
			forceError:  "getRemovedCountError",
			expectError: true,
			description: "获取已删除计数器数量失败时应该返回错误",
			setupFunc: func(t *testing.T, ctx context.Context, examID int64) {
				// 先设置定时器
				err := SetExamTimers(ctx, examID)
				require.NoError(t, err, "设置定时器应该成功")
			},
		},
		{
			name:        "多场次考试取消定时器",
			examID:      testData.MultiSessionExamID,
			forceError:  "",
			expectError: false,
			description: "多场次考试应该成功取消所有场次的定时器",
			setupFunc: func(t *testing.T, ctx context.Context, examID int64) {
				// 先设置定时器
				err := SetExamTimers(ctx, examID)
				require.NoError(t, err, "设置定时器应该成功")
			},
		},
		{
			name:        "异常考试取消定时器",
			examID:      testData.AbnormalExamID,
			forceError:  "",
			expectError: false,
			description: "异常考试也应该能正常取消定时器",
			setupFunc: func(t *testing.T, ctx context.Context, examID int64) {
				// 先设置定时器
				err := SetExamTimers(ctx, examID)
				require.NoError(t, err, "设置定时器应该成功")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理Redis状态
			conn := cmn.GetRedisConn()
			conn.Del(ctx, EXAM_TIMER_SET_KEY)

			// 执行测试前的设置
			if tt.setupFunc != nil {
				tt.setupFunc(t, ctx, tt.examID)
			}

			// 记录取消前的定时器数量
			beforeCount, err := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
			require.NoError(t, err)

			// 创建带强制错误的上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "CancelExamTimers-force-error", tt.forceError)
			}

			// 执行CancelExamTimers
			err = CancelExamTimers(testCtx, tt.examID)

			// 验证错误结果
			if tt.expectError {
				assert.Error(t, err, tt.description)
				t.Logf("%s: 成功捕获预期错误: %v", tt.name, err)
			} else {
				assert.NoError(t, err, tt.description)

				// 对于成功的情况，验证Redis中的定时器是否被正确取消
				if tt.forceError == "" || tt.forceError == "noSessionIDs" {
					afterCount, redisErr := conn.ZCard(ctx, EXAM_TIMER_SET_KEY).Result()
					require.NoError(t, redisErr)

					switch tt.examID {
					case testData.NormalExamID:
						if tt.setupFunc != nil {
							assert.Equal(t, int64(0), afterCount, "正常考试的定时器应该被完全取消")
							assert.Greater(t, int(beforeCount), 0, "取消前应该有定时器存在")
						} else {
							assert.Equal(t, beforeCount, afterCount, "没有设置定时器的情况下，取消前后数量应该相同")
						}
					case testData.MultiSessionExamID:
						if tt.setupFunc != nil {
							assert.Equal(t, int64(0), afterCount, "多场次考试的定时器应该被完全取消")
							assert.Greater(t, int(beforeCount), 0, "取消前应该有定时器存在")
						}
					case testData.AbnormalExamID:
						if tt.setupFunc != nil {
							assert.Equal(t, int64(0), afterCount, "异常考试的定时器应该被完全取消")
							assert.Greater(t, int(beforeCount), 0, "取消前应该有定时器存在")
						}
					case 999999:
						assert.Equal(t, beforeCount, afterCount, "不存在的考试ID不应该影响现有定时器")
					}

					// 验证特定考试的事件确实被删除
					if tt.setupFunc != nil && tt.forceError == "" {
						events, err := conn.ZRange(ctx, EXAM_TIMER_SET_KEY, 0, -1).Result()
						require.NoError(t, err)

						// 验证没有该考试的事件残留
						for _, eventStr := range events {
							var event ExamEvent
							err := json.Unmarshal([]byte(eventStr), &event)
							require.NoError(t, err)
							assert.NotEqual(t, tt.examID, event.ExamID, "取消后不应该还有该考试的定时器事件")
						}
					}
				}

				t.Logf("%s: 测试通过 - %s", tt.name, tt.description)
			}
		})
	}
}
