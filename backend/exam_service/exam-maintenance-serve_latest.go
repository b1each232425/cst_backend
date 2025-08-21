package exam_service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

//annotation:exam_service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

// 基于Go原生定时器的考试状态维护服务

const (
	// 事件类型
	EVENT_TYPE_EXAM_SESSION_START = "exam_session_start"
	EVENT_TYPE_EXAM_SESSION_END   = "exam_session_end"

	// 默认最大并发数
	DEFAULT_MAX_WORKERS = 10
)

var (
	z            *zap.Logger
	pgxConn      *pgxpool.Pool
	examTimerMgr *ExamTimerManager
)

// 事件数据结构
type ExamEvent struct {
	Type          string `json:"type"` // 事件类型：exam_session_start, exam_session_end
	ExamID        int64  `json:"exam_id,omitempty"`
	ExamSessionID int64  `json:"exam_session_id,omitempty"`
}

// 定时器管理器
type ExamTimerManager struct {
	timers     map[string]*time.Timer // key: "type_sessionID", value: timer
	mutex      sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	eventQueue chan ExamEvent
	maxWorkers int // 最大并发worker数量
}

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		pgxConn = cmn.GetPgxConn()
		z.Info("exam_service zLogger settled")
	})
}

// 创建定时器管理器的方法
func NewExamTimerManager(ctx context.Context, cancel context.CancelFunc) *ExamTimerManager {
	z.Info("---->" + cmn.FncName())
	maxWorkers := DEFAULT_MAX_WORKERS

	tm := &ExamTimerManager{
		timers:     make(map[string]*time.Timer),
		ctx:        ctx,
		cancel:     cancel,
		eventQueue: make(chan ExamEvent, maxWorkers*15),
		maxWorkers: maxWorkers,
	}

	// 启动固定数量的worker协程
	tm.startEventWorkers()

	return tm
}

// 启动固定数量的事件处理协程
func (tm *ExamTimerManager) startEventWorkers() {
	z.Info("---->" + cmn.FncName())
	for i := 0; i < tm.maxWorkers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-tm.ctx.Done():
					return
				case event := <-tm.eventQueue:
					tm.processEvent(event, workerID)
				}
			}
		}(i)
	}
}

// 处理单个事件
func (tm *ExamTimerManager) processEvent(event ExamEvent, workerID int) error {
	z.Info("---->" + cmn.FncName())
	switch event.Type {
	case "exam_session_start":
		handleExamSessionStart(tm.ctx, event)
	case "exam_session_end":
		handleExamSessionEnd(tm.ctx, event)
	default:
		err := fmt.Errorf("未知事件类型: %s", event.Type)
		z.Error("未知事件类型",
			zap.String("event_type", event.Type),
			zap.Int64("session_id", event.ExamSessionID))
		return err
	}

	return nil
}

// 设置定时器
func (tm *ExamTimerManager) SetTimer(examSessionID int64, triggerTime int64, event ExamEvent) {
	z.Info("---->" + cmn.FncName())
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	timerKey := fmt.Sprintf("%s_%d", event.Type, examSessionID)

	// 如果已存在同类型的定时器，先停止它
	if existingTimer, exists := tm.timers[timerKey]; exists {
		existingTimer.Stop()
		delete(tm.timers, timerKey)
	}

	// 计算延迟时间
	delay := time.Duration(triggerTime-time.Now().UnixMilli()) * time.Millisecond

	// 如果时间已过，立即将事件添加到处理队列
	if delay <= 0 {
		select {
		case tm.eventQueue <- event:
		case <-tm.ctx.Done():
			return
		}
		return
	}

	// 创建新的定时器
	timer := time.AfterFunc(delay, func() {
		// 将事件添加到队列
		select {
		case tm.eventQueue <- event:
		case <-tm.ctx.Done():
			return
		}

		// 从map中移除定时器
		tm.mutex.Lock()
		delete(tm.timers, timerKey)
		tm.mutex.Unlock()
	})

	tm.timers[timerKey] = timer

	z.Info("设置定时器",
		zap.String("event_type", event.Type),
		zap.Int64("exam_session_id", examSessionID),
		zap.Duration("delay", delay))
}

// 取消定时器
func (tm *ExamTimerManager) CancelTimer(eventType string, examSessionID int64) {
	z.Info("---->" + cmn.FncName())
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	timerKey := fmt.Sprintf("%s_%d", eventType, examSessionID)
	if timer, exists := tm.timers[timerKey]; exists {
		timer.Stop()
		delete(tm.timers, timerKey)
		z.Info("取消定时器",
			zap.String("event_type", eventType),
			zap.Int64("exam_session_id", examSessionID))
	}
}

// 设置考试定时器
func SetExamTimers(ctx context.Context, examID int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("SetExamTimers-force-error"); val != nil {
		forceErr = val.(string)
	}

	// 查询考试场次信息
	query := `
		SELECT 
			es.id,
			es.start_time,
			es.end_time
		FROM t_exam_session es
		WHERE es.exam_id = $1 AND es.status NOT IN ('14', '16')
	`

	rows, err := pgxConn.Query(ctx, query, examID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if forceErr == "queryExamSessions" {
		err = fmt.Errorf("强制查询考试场次信息错误")
	}
	if err != nil {
		z.Error("查询考试场次信息失败",
			zap.Int64("exam_id", examID),
			zap.Error(err))
		return err
	}

	var examSessionInfo []struct {
		ExamSessionID int64 `json:"exam_session_id"`
		StartTime     int64 `json:"start_time"`
		EndTime       int64 `json:"end_time"`
	}

	for rows.Next() {
		var sessionID, startTime, endTime int64
		err := rows.Scan(&sessionID, &startTime, &endTime)
		if forceErr == "scanExamSessionInfo" {
			err = fmt.Errorf("强制获取考试场次信息错误")
		}
		if err != nil {
			z.Error("获取考试场次信息失败", zap.Error(err))
			return err
		}

		examSessionInfo = append(examSessionInfo, struct {
			ExamSessionID int64 `json:"exam_session_id"`
			StartTime     int64 `json:"start_time"`
			EndTime       int64 `json:"end_time"`
		}{
			ExamSessionID: sessionID,
			StartTime:     startTime,
			EndTime:       endTime,
		})

		z.Info("查询到考试场次信息",
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", sessionID),
			zap.Int64("start_time", startTime),
			zap.Int64("end_time", endTime))
	}

	for _, examSession := range examSessionInfo {
		// 设置场次开始定时器
		examTimerMgr.SetTimer(examSession.ExamSessionID, examSession.StartTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_START,
			ExamSessionID: examSession.ExamSessionID,
			ExamID:        examID,
		})

		// 设置场次结束定时器
		examTimerMgr.SetTimer(examSession.ExamSessionID, examSession.EndTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_END,
			ExamSessionID: examSession.ExamSessionID,
			ExamID:        examID,
		})
	}

	return nil
}

// 取消考试的所有定时器
func CancelExamTimers(ctx context.Context, examID int64) error {

	forceErr := ""
	if val := ctx.Value("CancelExamTimers-force-error"); val != nil {
		forceErr = val.(string)
	}

	// 查询该考试的所有场次
	examSessionRows, err := pgxConn.Query(ctx, `
		SELECT id FROM t_exam_session WHERE exam_id = $1 AND status NOT IN ('14', '16')
	`, examID)
	if forceErr == "CancelExamTimers-force-error" {
		err = fmt.Errorf("强制查询考试场次信息错误")
	}

	defer examSessionRows.Close()
	if err != nil {
		z.Error("Failed to query sessions for timer cancellation", zap.Error(err))
		return err
	}

	// 收集所有场次ID并取消对应的定时器
	cancelCount := 0
	for examSessionRows.Next() {
		var sessionID int64
		if examSessionRows.Scan(&sessionID) == nil {
			examTimerMgr.CancelTimer(EVENT_TYPE_EXAM_SESSION_START, sessionID)
			examTimerMgr.CancelTimer(EVENT_TYPE_EXAM_SESSION_END, sessionID)
			cancelCount++
		}
	}

	z.Info("成功取消考试定时器",
		zap.Int64("exam_id", examID),
		zap.Int("cancelled_sessions", cancelCount))

	return nil
}

// 停止所有定时器
func (tm *ExamTimerManager) StopAll() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 停止所有定时器
	for key, timer := range tm.timers {
		timer.Stop()
		delete(tm.timers, key)
	}

	tm.cancel()
}

// 初始化现有考试的定时器
func InitializeExamTimers(ctx context.Context) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("initializeExamTimers-force-error"); val != nil {
		forceErr = val.(string)
	}

	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("panic") // 强制触发的panic错误
		}
		if r != nil {
			z.Error("Panic recovered in initializeExamTimers",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	// 查询场次信息
	query := `
        SELECT
            es.id as session_id,
            es.exam_id,
            es.start_time,
            es.end_time
        FROM t_exam_info ei
        JOIN t_exam_session es ON ei.id = es.exam_id
        WHERE ei.status IN ('02', '04') -- 待开始或进行中的考试
            AND es.status NOT IN ('14', '16') -- 未删除的场次
        ORDER BY es.exam_id, es.id
    `
	rows, err := pgxConn.Query(ctx, query)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if forceErr == "queryExamSessions" {
		err = fmt.Errorf("强制查询考试场次信息错误")
	}
	if err != nil {
		z.Error("查询考试场次信息失败", zap.Error(err))
		return err
	}

	// 设置场次定时器
	for rows.Next() {
		var examSessionID, examID, startTime, endTime int64
		err := rows.Scan(&examSessionID, &examID, &startTime, &endTime)
		if forceErr == "scanExamSessionInfo" {
			err = fmt.Errorf("强制获取考试场次信息错误")
		}
		if err != nil {
			z.Error("获取考试场次信息失败", zap.Error(err))
			return err
		}

		// 场次开始定时器
		examTimerMgr.SetTimer(examSessionID, startTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_START,
			ExamSessionID: examSessionID,
			ExamID:        examID,
		})

		// 场次结束定时器
		examTimerMgr.SetTimer(examSessionID, endTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_END,
			ExamSessionID: examSessionID,
			ExamID:        examID,
		})
	}

	return nil
}

// 处理考试场次开始事件
func handleExamSessionStart(ctx context.Context, event ExamEvent) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleExamSessionStart-force-error"); val != nil {
		forceErr = val.(string)
	}

	tx, err := pgxConn.Begin(ctx)
	if forceErr == "beginTx" {
		err = fmt.Errorf("强制开启事务错误")
	}
	if err != nil {
		z.Error("开启事务失败", zap.Error(err))
		return err
	}

	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("强制panic触发")
		}
		if r != nil {
			z.Error("Panic recovered in handleExamSessionStart",
				zap.Any("panic", r),
				zap.Stack("stack"))

			if rbErr := tx.Rollback(ctx); rbErr != nil || forceErr == "panic" {
				z.Error("panic后事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		if err != nil || forceErr == "rollback" {
			rbErr := tx.Rollback(ctx)
			if forceErr == "rollback" {
				rbErr = fmt.Errorf("强制回滚错误")
			}
			if rbErr != nil {
				z.Error("事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		cmErr := tx.Commit(ctx)
		if forceErr == "commit" {
			cmErr = fmt.Errorf("强制提交错误")
		}
		if cmErr != nil {
			z.Error("事务提交失败", zap.Error(cmErr))
		}
	}()

	now := time.Now().UnixMilli()

	// 查询场次信息和考生数量
	query := `
		SELECT 
			es.id as session_id,
			es.exam_id,
			COALESCE(e_count.count, 0) as examinee_count
		FROM t_exam_session es
		LEFT JOIN (
			SELECT exam_session_id, COUNT(*) as count
			FROM t_examinee 
			WHERE status != '08' AND exam_session_id = $1
			GROUP BY exam_session_id
		) e_count ON es.id = e_count.exam_session_id
		WHERE es.id = $1 AND es.status NOT IN ('14', '16')
	`

	var examSessionID, examID int64
	var examineeCount int
	err = tx.QueryRow(ctx, query, event.ExamSessionID).Scan(&examSessionID, &examID, &examineeCount)
	if forceErr == "queryExamSessionInfo" {
		err = fmt.Errorf("强制获取考试场次信息错误")
	}
	if err != nil {
		z.Error("查询场次信息失败", zap.Error(err),
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", examSessionID))
		return err
	}

	if examineeCount == 0 {
		// 没有考生的考试需要设为异常状态
		_, err := tx.Exec(ctx, `
			UPDATE t_exam_info
			SET status = '10', -- 考试异常
				update_time = $1
			WHERE id = $2
		`, now, examID)
		if forceErr == "updateAbnormalExam" {
			err = fmt.Errorf("强制更新考试为异常状态错误")
		}
		if err != nil {
			z.Error("更新考试为异常状态失败", zap.Error(err),
				zap.Int64("exam_id", examID),
				zap.Int64("exam_session_id", examSessionID))

			return err
		}

		// 取消该考试的所有定时器
		err = CancelExamTimers(ctx, examID)
		if forceErr == "cancelAbnormalExamTimers" {
			err = fmt.Errorf("强制取消考试定时器错误")
		}
		if err != nil {
			z.Error("取消考试定时器失败", zap.Error(err),
				zap.Int64("exam_id", examID),
				zap.Int64("exam_session_id", examSessionID))

			return err
		}

		return nil
	}

	// 有考生的场次正常开始
	_, err = tx.Exec(ctx, `
			UPDATE t_exam_session 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = $2 AND status = '02'
		`, now, examSessionID)
	if forceErr == "updateExamSession" {
		err = fmt.Errorf("强制更新场次状态错误")
	}
	if err != nil {
		z.Error("更新场次状态失败", zap.Error(err),
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", examSessionID))
		return err
	}

	// 更新考试状态
	_, err = tx.Exec(ctx, `
			UPDATE t_exam_info 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = $2 AND status = '02'
		`, now, examID)
	if forceErr == "updateExam" {
		err = fmt.Errorf("强制更新考试状态错误")
	}
	if err != nil {
		z.Error("更新考试状态失败", zap.Error(err),
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", examSessionID))
		return err
	}

	z.Info("考试场次开始事件处理",
		zap.Int64("exam_id", examID),
		zap.Int64("exam_session_id", examSessionID))

	return nil
}

// 处理考试场次结束事件
func handleExamSessionEnd(ctx context.Context, event ExamEvent) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleExamSessionEnd-force-error"); val != nil {
		forceErr = val.(string)
	}

	tx, err := pgxConn.Begin(ctx)
	if forceErr == "beginTx" {
		err = fmt.Errorf("强制开启事务错误")
	}
	if err != nil {
		z.Error("开启事务失败", zap.Error(err))
		return err
	}

	defer func() {
		r := recover()
		if forceErr == "panic" {
			r = fmt.Errorf("强制触发的panic错误")
		}
		if r != nil {
			z.Error("Panic recovered in handleExamSessionEnd",
				zap.Any("panic", r),
				zap.Stack("stack"))

			if rbErr := tx.Rollback(ctx); rbErr != nil || forceErr == "panic" {
				z.Error("panic后事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		if err != nil || forceErr == "rollback" {
			rbErr := tx.Rollback(ctx)
			if forceErr == "rollback" {
				rbErr = fmt.Errorf("强制回滚错误")
			}
			if rbErr != nil {
				z.Error("事务回滚失败", zap.Error(rbErr))
			}
			return
		}

		cmErr := tx.Commit(ctx)
		if forceErr == "commit" {
			cmErr = fmt.Errorf("强制提交错误")
		}
		if cmErr != nil {
			z.Error("事务提交失败", zap.Error(cmErr))
		}
	}()

	now := time.Now().UnixMilli()

	// 更新考生状态
	updateExamineesQuery := `
		WITH examinee_to_update AS (
			SELECT 
				e.id,
				CASE
					WHEN e.start_time IS NOT NULL THEN '10' -- 如果该场考试考生有开始时间，设为已交卷
					ELSE '02' -- 如果该场考试考生没有开始时间，设为已缺考
				END AS new_status,
				CASE
					WHEN e.start_time IS NOT NULL THEN v.actual_end_time
					ELSE NULL
				END AS new_end_time
			FROM t_examinee e
			JOIN v_examinee_info v ON e.id = v.id
			JOIN t_exam_info ei ON v.exam_id = ei.id
			WHERE e.exam_session_id = $1
				AND v.actual_end_time <= $2
				AND e.status IN ('00', '16') -- 只更新处于正常考和监考员允许进入考试状态的考生
				AND ei.status IN ('02', '04', '06')
		)
		UPDATE t_examinee te
		SET status = u.new_status,
			end_time = COALESCE(u.new_end_time, te.end_time),
			update_time = $2
		FROM examinee_to_update u
		WHERE te.id = u.id
	`

	_, err = tx.Exec(ctx, updateExamineesQuery, event.ExamSessionID, now)
	if forceErr == "updateExaminees" {
		err = fmt.Errorf("强制更新考生状态错误")
	}
	if err != nil {
		z.Error("更新考生状态失败", zap.Error(err),
			zap.Int64("exam_id", event.ExamID),
			zap.Int64("exam_session_id", event.ExamSessionID))
		return err
	}

	// 检查场次是否还有未结束的考生
	checkQuery := `
		SELECT 
			es.id as session_id,
			es.exam_id,
			COALESCE(unfinished.count, 0) as unfinished_count
		FROM t_exam_session es
		LEFT JOIN (
			SELECT exam_session_id, COUNT(*) as count
			FROM t_examinee 
			WHERE status NOT IN ('02', '06', '08', '10', '12') 
				AND exam_session_id = $1
			GROUP BY exam_session_id
		) unfinished ON es.id = unfinished.exam_session_id
		WHERE es.id = $1
	`

	var examSessionID, examID int64
	var unfinishedCount int
	err = tx.QueryRow(ctx, checkQuery, event.ExamSessionID).Scan(&examSessionID, &examID, &unfinishedCount)
	if forceErr == "scanUnfinishedExaminees" {
		err = fmt.Errorf("强制获取未结束考生数量错误")
	}
	if err != nil {
		z.Error("获取未结束的考生数量失败", zap.Error(err))
		return err
	}

	if unfinishedCount > 0 {
		// 还有考生未结束，延迟1分钟后重新处理
		delayedEndTime := now + 60*1000
		examTimerMgr.SetTimer(examSessionID, delayedEndTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_END,
			ExamSessionID: examSessionID,
			ExamID:        examID,
		})
		return nil
	}

	// 更新场次状态为已结束
	_, err = tx.Exec(ctx, `
		UPDATE t_exam_session 
		SET status = '06', -- 已结束
			update_time = $1
		WHERE id = $2 AND status IN ('02','04')
	`, now, examSessionID)
	if forceErr == "updateSessionEndStatus" {
		err = fmt.Errorf("强制更新场次状态为已结束错误")
	}
	if err != nil {
		z.Error("更新场次状态为已结束失败", zap.Error(err),
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", examSessionID))
		return err
	}

	// 检查并更新考试状态
	_, err = tx.Exec(ctx, `
		UPDATE t_exam_info ei
		SET status = '06', -- 已结束
			update_time = $1
		WHERE ei.status IN ('02','04')
			AND NOT EXISTS (
				SELECT 1 FROM t_exam_session es
				WHERE es.exam_id = ei.id AND es.status IN ('02','04')
			)
			AND ei.id = $2
	`, now, examID)
	if forceErr == "updateExamEndStatus" {
		err = fmt.Errorf("强制更新考试状态为已结束错误")
	}
	if err != nil {
		z.Error("更新考试状态为已结束失败", zap.Error(err),
			zap.Int64("exam_id", examID),
			zap.Int64("exam_session_id", examSessionID))
		return err
	}

	z.Info("考试场次结束事件处理",
		zap.Int64("exam_id", examID),
		zap.Int64("exam_session_id", examSessionID))

	return nil
}

func cleanupTempExams(ctx context.Context) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("cleanupTempExams-force-error"); val != nil {
		forceErr = val.(string)
	}

	// 计算24小时前的时间戳
	cutoffTime := time.Now().Add(-24 * time.Hour).UnixMilli()

	conn := cmn.GetPgxConn()

	// 删除临时考试记录并返回 files 字段
	rows, err := conn.Query(ctx, `
        DELETE FROM t_exam_info 
        WHERE status = '14' AND create_time < $1
        RETURNING files
    `, cutoffTime)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if forceErr == "deleteTempExams" {
		err = fmt.Errorf("强制删除临时考试错误")
	}
	if err != nil {
		z.Error("删除临时考试记录失败", zap.Error(err))
		return err
	}

	// 统计文件ID出现次数
	var allFileIDs []int64
	var examsDeleted int64 = 0

	for rows.Next() {
		var files []int64

		err := rows.Scan(&files)
		if forceErr == "scanFiles" {
			err = fmt.Errorf("强制扫描文件数组错误")
		}
		if err != nil {
			z.Error("扫描文件数组失败", zap.Error(err))
			return err
		}

		examsDeleted++

		// 统计每个文件ID的出现次数
		allFileIDs = append(allFileIDs, files...)
	}

	// 处理文件删除
	for _, fileID := range allFileIDs {
		err := cmn.DeleteFileRecord(ctx, nil, fileID)
		if forceErr == "handleDeleteExamFile" {
			err = fmt.Errorf("强制处理删除考试文件错误")
		}
		if err != nil {
			z.Error("处理删除考试文件失败", zap.Error(err))
			return err
		}
	}

	z.Info("临时考试清理完成",
		zap.Int64("deleted_exams", examsDeleted),
		zap.Int64("cutoff_time", cutoffTime))

	return nil
}

func startTempExamCleanup(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// 创建间隔为24小时的定时器
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 启动时执行一次清理
	cleanupTempExams(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupTempExams(ctx)
		}
	}
}

func ExamMaintainService() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	examTimerMgr = NewExamTimerManager(ctx, cancel)

	// 初始化定时器
	InitializeExamTimers(ctx)
	startTempExamCleanup(ctx)
}
