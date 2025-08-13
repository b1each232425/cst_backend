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
	timers map[string]*time.Timer // key: "type_sessionID", value: timer
	mutex  sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		pgxConn = cmn.GetPgxConn()
		z.Info("exam_service zLogger settled")
	})
}

// 创建定时器管理器的方法
func NewExamTimerManager(ctx context.Context) *ExamTimerManager {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	return &ExamTimerManager{
		timers: make(map[string]*time.Timer),
		ctx:    ctx,
		cancel: cancel,
	}
}

// 设置定时器
func (tm *ExamTimerManager) SetTimer(examSessionID int64, triggerTime int64, event ExamEvent) {
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

	// 如果时间已过，立即执行
	if delay <= 0 {
		go tm.handleEvent(event)
		return
	}

	// 创建新的定时器
	timer := time.AfterFunc(delay, func() {
		tm.handleEvent(event)

		// 定时器执行后从map中移除
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
	if forceErr == "queryExamSessions" {
		err = fmt.Errorf("强制查询考试场次信息错误")
	}
	if err != nil {
		z.Error("查询考试场次信息失败",
			zap.Int64("exam_id", examID),
			zap.Error(err))
		return err
	}
	defer rows.Close()

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

	for key, timer := range tm.timers {
		timer.Stop()
		delete(tm.timers, key)
	}

	tm.cancel()
	z.Info("停止所有定时器")
}

// 处理定时器事件
func (tm *ExamTimerManager) handleEvent(event ExamEvent) {
	switch event.Type {
	case EVENT_TYPE_EXAM_SESSION_START:
		handleExamSessionStart(tm.ctx, event)
	case EVENT_TYPE_EXAM_SESSION_END:
		handleExamSessionEnd(tm.ctx, event)
	default:
		z.Warn("未知的事件类型", zap.String("type", event.Type))
	}
}

// 初始化现有考试的定时器
func InitializeExamTimers(ctx context.Context) {
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

	now := time.Now().UnixMilli()

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
            AND (es.start_time > $1 OR es.end_time > $1)
        ORDER BY es.exam_id, es.id
    `
	rows, err := pgxConn.Query(ctx, query, now)
	if forceErr == "queryExamSessions" {
		err = fmt.Errorf("强制查询考试场次信息错误")
	}
	if err != nil {
		z.Error("查询考试场次信息失败", zap.Error(err))
		return
	}
	defer rows.Close()

	// 设置场次定时器
	for rows.Next() {
		var examSessionID, examID, startTime, endTime int64
		err := rows.Scan(&examSessionID, &examID, &startTime, &endTime)
		if forceErr == "scanExamSessionInfo" {
			err = fmt.Errorf("强制获取考试场次信息错误")
		}
		if err != nil {
			z.Error("获取考试场次信息失败", zap.Error(err))
			return
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
}

// 处理考试场次开始事件
func handleExamSessionStart(ctx context.Context, event ExamEvent) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleExamSessionStart-force-error"); val != nil {
		forceErr = val.(string)
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
	err := pgxConn.QueryRow(ctx, query, event.ExamSessionID).Scan(&examSessionID, &examID, &examineeCount)
	if err != nil || forceErr == "queryExamSessionInfo" {
		z.Error("查询场次信息失败", zap.Error(err))
		return
	}

	if examineeCount == 0 {
		// 没有考生的考试需要设为异常状态
		_, err := pgxConn.Exec(ctx, `
			UPDATE t_exam_info
			SET status = '10', -- 考试异常
				update_time = $1
			WHERE id = $2
		`, now, examID)
		if err != nil || forceErr == "updateAbnormalExam" {
			z.Error("更新考试为异常状态失败", zap.Error(err))
			return
		}

		// 取消该考试的所有定时器
		err = CancelExamTimers(ctx, examID)
		if err != nil || forceErr == "cancelAbnormalExamTimers" {
			z.Error("取消考试定时器失败", zap.Int64("exam_id", examID), zap.Error(err))
		}
	} else {
		// 有考生的场次正常开始
		// 更新场次状态
		_, err := pgxConn.Exec(ctx, `
			UPDATE t_exam_session 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = $2 AND status = '02'
		`, now, examSessionID)
		if err != nil || forceErr == "updateExamSession" {
			z.Error("更新场次状态失败", zap.Error(err))
			return
		}

		// 更新考试状态
		_, err = pgxConn.Exec(ctx, `
			UPDATE t_exam_info 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = $2 AND status = '02'
		`, now, examID)
		if err != nil || forceErr == "updateExam" {
			z.Error("更新考试状态失败", zap.Error(err))
			return
		}
	}

	z.Info("考试场次开始处理完成",
		zap.Int64("exam_session_id", examSessionID),
		zap.Int64("exam_id", examID),
		zap.Int("examinee_count", examineeCount))
}

// 处理考试场次结束事件
func handleExamSessionEnd(ctx context.Context, event ExamEvent) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("handleExamSessionEnd-force-error"); val != nil {
		forceErr = val.(string)
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
			JOIN t_exam_session es ON e.exam_session_id = es.id
			JOIN t_exam_info ei ON es.exam_id = ei.id
			WHERE e.exam_session_id = $1
				AND v.actual_end_time <= $2
				AND e.status IN ('00', '16') -- 只更新处于正常考和监考员允许进入考试状态的考生
				AND ei.status IN ('04', '06')
		)
		UPDATE t_examinee te
		SET status = u.new_status,
			end_time = COALESCE(u.new_end_time, te.end_time),
			update_time = $2
		FROM examinee_to_update u
		WHERE te.id = u.id
	`

	_, err := pgxConn.Exec(ctx, updateExamineesQuery, event.ExamSessionID, now)
	if err != nil || forceErr == "updateExaminees" {
		z.Error("更新考生状态失败", zap.Error(err))
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
	err = pgxConn.QueryRow(ctx, checkQuery, event.ExamSessionID).Scan(&examSessionID, &examID, &unfinishedCount)
	if forceErr == "scanUnfinishedExaminees" {
		err = fmt.Errorf("强制获取未结束考生数量错误")
	}
	if err != nil {
		z.Error("获取未结束的考生数量失败", zap.Error(err))
		return
	}

	if unfinishedCount > 0 {
		// 还有考生未结束，延迟1分钟后重新处理
		delayedEndTime := now + 60*1000
		examTimerMgr.SetTimer(examSessionID, delayedEndTime, ExamEvent{
			Type:          EVENT_TYPE_EXAM_SESSION_END,
			ExamSessionID: examSessionID,
			ExamID:        examID,
		})

		z.Info("场次还有未结束考生，延迟处理",
			zap.Int64("exam_session_id", examSessionID),
			zap.Int("unfinished_count", unfinishedCount))
		return
	}

	// 更新场次状态为已结束
	_, err = pgxConn.Exec(ctx, `
		UPDATE t_exam_session 
		SET status = '06', -- 已结束
			update_time = $1
		WHERE id = $2 AND status IN ('02','04')
	`, now, examSessionID)
	if forceErr == "updateSessionEndStatus" {
		err = fmt.Errorf("强制更新场次状态为已结束错误")
	}
	if err != nil {
		z.Error("更新场次状态为已结束失败", zap.Error(err))
		return
	}

	// 检查并更新考试状态
	_, err = pgxConn.Exec(ctx, `
		UPDATE t_exam_info ei
		SET status = '06', -- 已结束
			update_time = $1
		WHERE ei.status = '04'
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
		z.Error("更新考试状态为已结束失败", zap.Error(err))
	}

	z.Info("考试场次结束处理完成",
		zap.Int64("exam_session_id", examSessionID),
		zap.Int64("exam_id", examID))
}

func ExamMaintainService() {
	ctx := context.Background()

	examTimerMgr = NewExamTimerManager(ctx)

	// 初始化定时器
	InitializeExamTimers(ctx)
}
