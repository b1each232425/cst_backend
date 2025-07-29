package exam_service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

//annotation:exam_service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

// 基于Redis有序集合的考试状态维护服务

const (
	EXAM_TIMER_SET_KEY = "exam:timers"

	// 事件类型
	EVENT_TYPE_EXAM_SESSION_START = "exam_session_start"
	EVENT_TYPE_EXAM_SESSION_END   = "exam_session_end"
)

// 事件数据结构
type ExamEvent struct {
	Type          string `json:"type"` // 事件类型：exam_session_start, exam_session_end
	ExamID        int64  `json:"exam_id,omitempty"`
	ExamSessionID int64  `json:"exam_session_id,omitempty"`
}

var (
	z       *zap.Logger
	pgxConn *pgxpool.Pool
)

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		pgxConn = cmn.GetPgxConn()

		z.Info("exam_service zLogger settled")
	})
}

// 初始化现有考试的Redis定时器
func initializeExamTimers() {
	z.Info("---->" + cmn.FncName())
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in initializeExamTimers",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	ctx := context.Background()
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
            AND es.status != '14' -- 未删除的场次
            AND (es.start_time > $1 OR es.end_time > $1)
        ORDER BY es.exam_id, es.id
    `

	rows, err := pgxConn.Query(ctx, query, now)
	if err != nil {
		z.Error("查询考试场次信息失败", zap.Error(err))
		return
	}
	defer rows.Close()

	sessionCount := 0

	// 设置场次定时器
	for rows.Next() {
		var examSessionID, examID, startTime, endTime int64
		err := rows.Scan(&examSessionID, &examID, &startTime, &endTime)
		if err != nil {
			z.Error("获取考试场次信息失败", zap.Error(err))
			continue
		}

		// 场次开始定时器
		if startTime > now {
			setRedisTimer(EVENT_TYPE_EXAM_SESSION_START, examSessionID, startTime, ExamEvent{
				ExamSessionID: examSessionID,
				ExamID:        examID,
			})
		}

		// 场次结束定时器
		if endTime > now {
			setRedisTimer(EVENT_TYPE_EXAM_SESSION_END, examSessionID, endTime, ExamEvent{
				ExamSessionID: examSessionID,
				ExamID:        examID,
			})
		}
		sessionCount++
	}

	return
}

// 轮询Redis有序集合检查到期事件
func pollRedisTimers() {
	z.Info("---->" + cmn.FncName())

	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in pollRedisTimers",
				zap.Any("panic", r),
				zap.Stack("stack"))

			// 重新启动轮询
			time.Sleep(5 * time.Second)
			go pollRedisTimers()
		}
	}()

	// 计算到下一个整分钟的等待时间
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	waitDuration := nextMinute.Sub(now)
	time.Sleep(waitDuration)

	luaScript := `
		local scheduled_key = KEYS[1]
		local now = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])

		-- 获取到期的事件
		local events = redis.call("ZRANGEBYSCORE", scheduled_key, 0, now, "LIMIT", 0, limit)
		
		if #events > 0 then
			-- 从有序集合中删除这些事件
			for i = 1, #events do
				redis.call("ZREM", scheduled_key, events[i])
			end
		end
		
		return events
	`

	for {
		now := time.Now()

		func() {
			conn := cmn.GetRedisConn()
			defer conn.Close()

			// 获取并删除到期事件
			result, err := conn.Do("EVAL", luaScript, 1, EXAM_TIMER_SET_KEY, now.UnixMilli(), 100)
			if err != nil {
				z.Error("Failed to execute Lua script for timer events", zap.Error(err))
				return
			}

			events, err := redis.Strings(result, nil)
			if err != nil {
				z.Error("Failed to convert Lua script result", zap.Error(err))
				return
			}

			// 处理到期事件
			if len(events) > 0 {
				handleTimerEventsBatch(events)
			}
		}()

		// 等待到下一个整分钟
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)
		waitDuration := nextMinute.Sub(now)
		time.Sleep(waitDuration)
	}
}

// 批量处理定时器事件
func handleTimerEventsBatch(eventStrs []string) {
	z.Info("---->" + cmn.FncName())
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in handleTimerEventsBatch",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	// 分类事件
	var sessionStartEvents []ExamEvent
	var sessionEndEvents []ExamEvent

	for _, eventStr := range eventStrs {
		var event ExamEvent
		if err := json.Unmarshal([]byte(eventStr), &event); err != nil {
			z.Error("Failed to unmarshal event", zap.Error(err))
			continue
		}

		switch event.Type {
		case EVENT_TYPE_EXAM_SESSION_START:
			sessionStartEvents = append(sessionStartEvents, event)
		case EVENT_TYPE_EXAM_SESSION_END:
			sessionEndEvents = append(sessionEndEvents, event)
		default:
			z.Warn("未知的事件类型", zap.String("type", event.Type))
		}
	}

	// 批量处理开始事件
	if len(sessionStartEvents) > 0 {
		handleExamSessionStartBatch(sessionStartEvents)
	}

	// 批量处理结束事件
	if len(sessionEndEvents) > 0 {
		handleExamSessionEndBatch(sessionEndEvents)
	}

	return
}

// 批量处理考试场次开始事件
func handleExamSessionStartBatch(events []ExamEvent) {
	z.Info("---->" + cmn.FncName())
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in handleExamSessionStartBatch",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	ctx := context.Background()
	now := time.Now().UnixMilli()

	examSessionIDs := make([]int64, 0, len(events))
	for _, event := range events {
		examSessionIDs = append(examSessionIDs, event.ExamSessionID)
	}

	// 批量查询场次信息和考生数量
	query := `
		SELECT 
			es.id as session_id,
			es.exam_id,
			COALESCE(e_count.count, 0) as examinee_count
		FROM t_exam_session es
		LEFT JOIN (
			SELECT exam_session_id, COUNT(*) as count
			FROM t_examinee 
			WHERE status != '08' AND exam_session_id = ANY($1)
			GROUP BY exam_session_id
		) e_count ON es.id = e_count.exam_session_id
		WHERE es.id = ANY($1) AND es.status != '14'
	`

	rows, err := pgxConn.Query(ctx, query, examSessionIDs)
	if err != nil {
		z.Error("批量查询场次信息失败", zap.Error(err))
		return
	}
	defer rows.Close()

	// 收集需要处理的数据
	type examSessionInfo struct {
		examSessionID int64
		ExamID        int64
		ExamineeCount int
	}

	sessionMap := make(map[int64]examSessionInfo)
	var abnormalExamIDs []int64
	var normalExamSessions []examSessionInfo

	for rows.Next() {
		var info examSessionInfo
		err := rows.Scan(&info.examSessionID, &info.ExamID, &info.ExamineeCount)
		if err != nil {
			z.Error("扫描场次信息失败", zap.Error(err))
			return
		}

		sessionMap[info.examSessionID] = info

		if info.ExamineeCount == 0 {
			// 没有考生的考试需要设为异常状态
			abnormalExamIDs = append(abnormalExamIDs, info.ExamID)
		} else {
			// 有考生的场次正常开始
			normalExamSessions = append(normalExamSessions, info)
		}
	}

	// 批量处理异常考试
	if len(abnormalExamIDs) > 0 {
		abnormalQuery := `
			UPDATE t_exam_info
			SET status = '10', -- 考试异常
				update_time = $1
			WHERE id = ANY($2)
		`
		_, err := pgxConn.Exec(ctx, abnormalQuery, now, abnormalExamIDs)
		if err != nil {
			z.Error("批量更新考试为异常状态失败", zap.Error(err))
		}

		// 取消异常考试的定时器
		for _, examID := range abnormalExamIDs {
			err := CancelExamTimers(ctx, examID)
			if err != nil {
				z.Error("取消考试定时器失败", zap.Int64("exam_id", examID), zap.Error(err))
			}
		}
	}

	// 批量处理正常开始的场次
	if len(normalExamSessions) > 0 {
		normalExamSessionIDs := make([]int64, 0, len(normalExamSessions))
		examIDSet := make(map[int64]bool)

		for _, info := range normalExamSessions {
			normalExamSessionIDs = append(normalExamSessionIDs, info.examSessionID)
			examIDSet[info.ExamID] = true
		}

		// 批量更新场次状态
		sessionUpdateQuery := `
			UPDATE t_exam_session 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = ANY($2) AND status = '02'
		`
		_, err := pgxConn.Exec(ctx, sessionUpdateQuery, now, normalExamSessionIDs)
		if err != nil {
			z.Error("批量更新场次状态失败", zap.Error(err))
		}

		// 批量更新考试状态
		examIDs := make([]int64, 0, len(examIDSet))
		for examID := range examIDSet {
			examIDs = append(examIDs, examID)
		}

		examUpdateQuery := `
			UPDATE t_exam_info 
			SET status = '04', -- 进行中
				update_time = $1
			WHERE id = ANY($2) AND status = '02'
		`
		_, err = pgxConn.Exec(ctx, examUpdateQuery, now, examIDs)
		if err != nil {
			z.Error("批量更新考试状态失败", zap.Error(err))
		}
	}

	return
}

// 批量处理考试场次结束事件
func handleExamSessionEndBatch(events []ExamEvent) {
	z.Info("---->" + cmn.FncName())
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in handleExamSessionEndBatch",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	ctx := context.Background()
	now := time.Now().UnixMilli()

	// 提取所有场次ID
	examSessionIDs := make([]int64, 0, len(events))
	for _, event := range events {
		examSessionIDs = append(examSessionIDs, event.ExamSessionID)
	}

	// 批量更新考生状态
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
			WHERE e.exam_session_id = ANY($1)
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

	_, err := pgxConn.Exec(ctx, updateExamineesQuery, examSessionIDs, now)
	if err != nil {
		z.Error("批量更新考生状态失败", zap.Error(err))
	}
	// 检查每个场次是否还有未结束的考生
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
				AND exam_session_id = ANY($1)
			GROUP BY exam_session_id
		) unfinished ON es.id = unfinished.exam_session_id
		WHERE es.id = ANY($1)
	`

	rows, err := pgxConn.Query(ctx, checkQuery, examSessionIDs)
	if err != nil {
		z.Error("批量检查未结束考生失败", zap.Error(err))
		return
	}
	defer rows.Close()

	var readyToEndExamSessions []int64
	var delayedExamSessions []struct {
		ExamSessionID int64
		ExamID        int64
	}

	for rows.Next() {
		var examSessionID, examID int64
		var unfinishedCount int
		err := rows.Scan(&examSessionID, &examID, &unfinishedCount)
		if err != nil {
			z.Error("获取未结束的考生数量失败", zap.Error(err))
			continue
		}

		if unfinishedCount > 0 {
			// 还有考生未结束，需要延迟处理
			delayedExamSessions = append(delayedExamSessions, struct {
				ExamSessionID int64
				ExamID        int64
			}{examSessionID, examID})
		} else {
			// 可以结束的场次
			readyToEndExamSessions = append(readyToEndExamSessions, examSessionID)
		}
	}

	// 处理延迟的场次
	if len(delayedExamSessions) > 0 {
		delayedEndTime := now + 60*1000
		for _, delayed := range delayedExamSessions {
			setRedisTimer(EVENT_TYPE_EXAM_SESSION_END, delayed.ExamSessionID, delayedEndTime, ExamEvent{
				ExamSessionID: delayed.ExamSessionID,
				ExamID:        delayed.ExamID,
			})
		}
	}

	// 处理可以结束的场次
	if len(readyToEndExamSessions) > 0 {
		// 更新场次状态为已结束
		sessionEndQuery := `
			UPDATE t_exam_session 
			SET status = '06', -- 已结束
				update_time = $1
			WHERE id = ANY($2) AND status = '04'
		`
		_, err := pgxConn.Exec(ctx, sessionEndQuery, now, readyToEndExamSessions)
		if err != nil {
			z.Error("批量更新场次状态为已结束失败", zap.Error(err))
		}

		// 检查并更新考试状态
		examUpdateQuery := `
			UPDATE t_exam_info ei
			SET status = '06', -- 已结束
				update_time = $1
			WHERE ei.status = '04'
				AND NOT EXISTS (
					SELECT 1 FROM t_exam_session es
					WHERE es.exam_id = ei.id AND es.status = '04'
				)
				AND EXISTS (
					SELECT 1 FROM t_exam_session es2
					WHERE es2.exam_id = ei.id AND es2.id = ANY($2)
				)
		`
		_, err = pgxConn.Exec(ctx, examUpdateQuery, now, readyToEndExamSessions)
		if err != nil {
			z.Error("批量更新考试状态为已结束失败", zap.Error(err))
		}
	}
}

// 设置考试定时器
func SetExamTimers(ctx context.Context, examID int64) error {
	z.Info("---->" + cmn.FncName())

	// 查询考试场次信息
	query := `
		SELECT 
			es.id,
			es.start_time,
			es.end_time
		FROM t_exam_session es
		WHERE es.exam_id = $1 AND es.status != '14'
	`

	rows, err := pgxConn.Query(ctx, query, examID)
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
		if err != nil {
			z.Error("获取考试场次信息失败", zap.Error(err))
			continue
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
		setRedisTimer(EVENT_TYPE_EXAM_SESSION_START, examSession.ExamSessionID, examSession.StartTime, ExamEvent{
			ExamSessionID: examSession.ExamSessionID,
			ExamID:        examID,
		})

		// 设置场次结束定时器
		setRedisTimer(EVENT_TYPE_EXAM_SESSION_END, examSession.ExamSessionID, examSession.EndTime, ExamEvent{
			ExamSessionID: examSession.ExamSessionID,
			ExamID:        examID,
		})
	}

	return nil
}

// 取消考试相关的定时器
func CancelExamTimers(ctx context.Context, examID int64) error {
	z.Info("---->" + cmn.FncName())

	// 查询该考试的所有场次
	examSessionRows, err := pgxConn.Query(ctx, `
		SELECT id FROM t_exam_session WHERE exam_id = $1 AND status != '14'
	`, examID)
	if err != nil {
		z.Error("Failed to query sessions for timer cancellation", zap.Error(err))
		return err
	}
	defer examSessionRows.Close()

	// 收集所有场次ID
	var sessionIDs []int64
	for examSessionRows.Next() {
		var sessionID int64
		if examSessionRows.Scan(&sessionID) == nil {
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	if len(sessionIDs) == 0 {
		z.Info("没有需要取消的定时器", zap.Int64("exam_id", examID))
		return nil
	}

	// 删除指定场次的所有事件
	luaScript := `
		local timer_key = KEYS[1]
		local session_ids = {}
		
		-- 将参数转换为场次ID集合
		for i = 1, #ARGV do
			session_ids[tonumber(ARGV[i])] = true
		end
		
		-- 获取所有定时器事件
		local all_events = redis.call("ZRANGE", timer_key, 0, -1)
		local removed_count = 0
		
		-- 遍历所有事件，删除匹配的场次事件
		for i = 1, #all_events do
			local event_str = all_events[i]
			local event = cjson.decode(event_str)
			
			-- 检查是否是目标场次的事件
			if event.exam_session_id and session_ids[tonumber(event.exam_session_id)] then
				redis.call("ZREM", timer_key, event_str)
				removed_count = removed_count + 1
			end
		end
		
		return removed_count
	`

	luaArgs := make([]interface{}, 0, len(sessionIDs)+2)
	luaArgs = append(luaArgs, luaScript, 1, EXAM_TIMER_SET_KEY)
	for _, sessionID := range sessionIDs {
		luaArgs = append(luaArgs, sessionID)
	}

	conn := cmn.GetRedisConn()
	defer conn.Close()

	removedCount, err := redis.Int(conn.Do("EVAL", luaArgs...))
	if err != nil {
		z.Error("Failed to execute Lua script for timer cancellation", zap.Error(err))
		return err
	}

	z.Info("成功取消考试定时器",
		zap.Int64("exam_id", examID),
		zap.Int("session_count", len(sessionIDs)),
		zap.Int("removed_timers", removedCount))

	return nil
}

func ExamMaintainService() {

	// 重新初始化定时器
	initializeExamTimers()

	// 启动Redis定时器轮询
	pollRedisTimers()
}

// // 处理定时器事件
// func handleTimerEvent(event ExamEvent) {
// 	z.Info("---->" + cmn.FncName())
// 	defer func() {
// 		if r := recover(); r != nil {
// 			z.Error("Panic recovered in handleTimerEvent",
// 				zap.Any("panic", r),
// 				zap.String("event_type", event.Type),
// 				zap.Int64("exam_session_id", event.ExamSessionID),
// 				zap.Stack("stack"))
// 		}
// 	}()

// 	switch event.Type {
// 	case EVENT_TYPE_SESSION_START:
// 		handleExamSessionStart(event.ExamSessionID)
// 	case EVENT_TYPE_SESSION_END:
// 		handleExamSessionEnd(event.ExamSessionID)
// 	default:
// 		z.Warn("未知的事件类型", zap.String("type", event.Type))
// 	}
// }

// // 处理考试场次开始事件
// func handleExamSessionStart(examSessionID int64) {
// 	z.Info("---->" + cmn.FncName())

// 	ctx := context.Background()
// 	now := time.Now().UnixMilli()

// 	// 先检查该场次是否有考生
// 	var exameeCount int
// 	var examID int64

// 	// 先获取考试ID
// 	err := pgxConn.QueryRow(ctx, `
// 		SELECT exam_id FROM t_exam_session WHERE id = $1 AND status != '14'
// 	`, examSessionID).Scan(&examID)
// 	if err != nil {
// 		z.Error("获取考试ID失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 		return
// 	}

// 	// 查询考生数量
// 	err = pgxConn.QueryRow(ctx, `
// 		SELECT COUNT(*)
// 		FROM t_examinee e
// 		WHERE e.exam_session_id = $1 AND e.status != '08'
// 	`, examSessionID).Scan(&exameeCount)
// 	if err != nil {
// 		z.Error("检查考试场次考生数量失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 		return
// 	}

// 	if exameeCount == 0 {

// 		// 没有考生时，设置考试状态为异常(10)
// 		examAbnormalQuery := `
// 			UPDATE t_exam_info
// 			SET status = '10', -- 考试异常
// 				update_time = $1
// 			WHERE id = $2
// 		`
// 		_, err := pgxConn.Exec(ctx, examAbnormalQuery, now, examID)
// 		if err != nil {
// 			z.Error("更新考试为异常状态失败",
// 				zap.Int64("exam_id", examID),
// 				zap.Error(err))
// 		} else {
// 			z.Info("考试未设置考生，更新为异常状态",
// 				zap.Int64("exam_id", examID),
// 				zap.Int64("exam_session_id", examSessionID))
// 		}

// 		// 取消该考试的所有定时器
// 		err = CancelExamTimers(ctx, examID)
// 		if err != nil {
// 			z.Error("取消考试定时器失败",
// 				zap.Int64("exam_id", examID),
// 				zap.Error(err))
// 		}
// 		return
// 	}

// 	// 有考生则同时更新场次和考试状态为进行中
// 	updateQuery := `
// 		WITH session_update AS (
// 			UPDATE t_exam_session es
// 			SET status = '04', -- 设置为进行中
// 				update_time = $1
// 			WHERE es.id = $2
// 				AND es.status = '02' -- 只有待开始状态才能设置为进行中
// 			RETURNING es.exam_id
// 		)
// 		UPDATE t_exam_info ei
// 		SET status = '04', -- 设置为进行中
// 			update_time = $1
// 		FROM session_update su
// 		WHERE ei.id = su.exam_id
// 			AND ei.status = '02' -- 只有待开始状态才能设置为进行中
// 	`

// 	_, err = pgxConn.Exec(ctx, updateQuery, now, examSessionID)
// 	if err != nil {
// 		z.Error("更新考试场次和考试状态为进行中失败",
// 			zap.Int64("session_id", examSessionID),
// 			zap.Int64("exam_id", examID),
// 			zap.Error(err))
// 	} else {
// 		z.Info("成功更新考试场次和考试状态为进行中",
// 			zap.Int64("session_id", examSessionID),
// 			zap.Int64("exam_id", examID))
// 	}

// 	return
// }

// // 处理考试场次结束事件
// func handleExamSessionEnd(examSessionID int64) {
// 	z.Info("---->" + cmn.FncName())

// 	ctx := context.Background()
// 	now := time.Now().UnixMilli()

// 	// 先获取考试ID
// 	var examID int64
// 	err := pgxConn.QueryRow(ctx, `
// 		SELECT exam_id FROM t_exam_session WHERE id = $1
// 	`, examSessionID).Scan(&examID)
// 	if err != nil {
// 		z.Error("获取考试ID失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 		return
// 	}

// 	// 先更新该考试场次的考生状态
// 	updateExamineesQuery := `
// 		WITH examinee_to_update AS (
// 			SELECT
// 				e.id,
// 				CASE
// 					WHEN e.start_time IS NOT NULL THEN '10' -- 如果该场考试考生有开始时间，设为已交卷
// 					ELSE '02' -- 如果该场考试考生没有开始时间，设为已缺考
// 				END AS new_status,
// 				CASE
// 					WHEN e.start_time IS NOT NULL THEN v.actual_end_time
// 					ELSE NULL
// 				END AS new_end_time
// 			FROM t_examinee e
// 			JOIN v_examinee_info v ON e.id = v.id
// 			WHERE e.exam_session_id = $1
// 				AND v.actual_end_time <= $2
// 				AND e.status IN ('00', '16') -- 只更新处于正常考和监考员允许进入考试状态的考生
// 		)
// 		UPDATE t_examinee te
// 		SET status = u.new_status,
// 			end_time = COALESCE(u.new_end_time, te.end_time),
// 			update_time = NOW()
// 		FROM examinee_to_update u
// 		WHERE te.id = u.id
// 	`

// 	_, err = pgxConn.Exec(ctx, updateExamineesQuery, examSessionID, now)
// 	if err != nil {
// 		z.Error("更新考生状态失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 	}

// 	// 检查该场次是否还有未结束的考生
// 	var unfinishedCount int
// 	err = pgxConn.QueryRow(ctx, `
// 		SELECT COUNT(*)
// 		FROM t_examinee e
// 		WHERE e.exam_session_id = $1
// 			AND e.status NOT IN ('02', '06', '08', '10', '12')
// 	`, examSessionID).Scan(&unfinishedCount)
// 	if err != nil {
// 		z.Error("检查未结束的考生数量失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 		return
// 	}

// 	if unfinishedCount > 0 {
// 		// 如果还有考生未结束，延迟1分钟后重新检查
// 		delayedEndTime := now + 60*1000
// 		setRedisTimer(EVENT_TYPE_SESSION_END, examSessionID, delayedEndTime, ExamEvent{
// 			ExamSessionID: examSessionID,
// 			ExamID:        examID,
// 		})
// 		return
// 	}

// 	// 当满足场次结束的条件时，设置场次状态为已结束
// 	examSessionEndQuery := `
// 		UPDATE t_exam_session es
// 		SET status = '06', -- 设置为已结束
// 			update_time = $1
// 		WHERE es.id = $2
// 			AND es.status = '04' -- 只有进行中状态才能设置为已结束
// 	`

// 	cmdTag1, err := pgxConn.Exec(ctx, examSessionEndQuery, now, examSessionID)
// 	if err != nil {
// 		z.Error("更新场次状态为已结束失败",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Error(err))
// 		return
// 	}

// 	if cmdTag1.RowsAffected() > 0 {
// 		z.Info("考试场次状态更新为已结束",
// 			zap.Int64("exam_session_id", examSessionID),
// 			zap.Int64("exam_id", examID))
// 	}

// 	// 检查该考试的所有场次是否都已结束，如果都结束则同时更新考试状态
// 	examUpdateQuery := `
// 		UPDATE t_exam_info ei
// 		SET status = '06', -- 设置为已结束
// 			update_time = $1
// 		WHERE ei.id = $2
// 			AND ei.status = '04' -- 只有进行中状态才能设置为已结束
// 			AND NOT EXISTS (
// 				SELECT 1 FROM t_exam_session es
// 				WHERE es.exam_id = ei.id
// 					AND es.status = '04' -- 进行中
// 			)
// 	`

// 	cmdTag2, err := pgxConn.Exec(ctx, examUpdateQuery, now, examID)
// 	if err != nil {
// 		z.Error("更新考试状态为已结束时失败",
// 			zap.Int64("exam_id", examID),
// 			zap.Error(err))
// 	}

// 	if cmdTag2.RowsAffected() > 0 {
// 		z.Info("考试状态更新为已结束",
// 			zap.Int64("exam_id", examID))
// 	}

// 	return
// }

// // 考生状态维护，将所有考生状态更新为正确的状态
// func examineesStatusMaintain() {
// 	z.Info("---->" + cmn.FncName())
// 	defer func() {
// 		if r := recover(); r != nil {
// 			z.Error("Panic recovered in examineesStatusMaintain",
// 				zap.Any("panic", r),
// 				zap.Stack("stack"))
// 		}
// 	}()

// 	now := time.Now().UnixMilli()
// 	query := `
// 		WITH examinee_to_update AS (
// 			SELECT
// 				e.id,
// 				CASE
// 					WHEN e.start_time IS NOT NULL THEN '10' -- 如果该场考试考生有开始时间，设为已交卷
// 					ELSE '02' -- 如果该场考试考生没有开始时间，设为已缺考
// 				END AS new_status,
// 				CASE
// 					WHEN e.start_time IS NOT NULL THEN v.actual_end_time
// 					ELSE NULL
// 				END AS new_end_time
// 			FROM t_examinee e
// 			JOIN t_exam_session es ON e.exam_session_id = es.id
// 			JOIN t_exam_info ei ON es.exam_id = ei.id
// 			JOIN v_examinee_info v ON e.id = v.id
// 			WHERE v.actual_end_time <= $1
// 				AND e.status IN ('00', '16') -- 只更新处于正常考和监考员允许进入考试状态的考生
// 				AND es.status != '14' -- 只更新未被删除的考试场次
// 				AND ei.status IN ('04', '06')
// 		)
// 		UPDATE t_examinee t
// 		SET status = u.new_status,
// 			end_time = COALESCE(u.new_end_time, t.end_time),
// 			update_time = $1
// 		FROM examinee_to_update u
// 		WHERE t.id = u.id
// 	`

// 	_, err := pgxConn.Exec(context.Background(), query, now)
// 	if err != nil {
// 		z.Error("更新考生状态失败", zap.Error(err))
// 	}
// }

// // 考试场次状态维护，将考试场次状态更新为正确的状态
// func examSessionStatusMaintain() {
// 	z.Info("---->" + cmn.FncName())
// 	defer func() {
// 		if r := recover(); r != nil {
// 			z.Error("Panic recovered in examSessionStatusMaintain",
// 				zap.Any("panic", r),
// 				zap.Stack("stack"))
// 		}
// 	}()

// 	now := time.Now().UnixMilli()
// 	start_query := `
// 		Update t_exam_session es
// 		SET status = '04', -- 进行中
// 			update_time = $1
// 		FROM t_exam_info ei
// 		WHERE es.exam_id = ei.id
// 			AND es.status = '02'
// 			AND es.start_time <= $1
// 			AND ei.status NOT IN ('00', '10', '12'); -- 排除未发布、考试异常和已删除的考试
// 	`
// 	_, err := pgxConn.Exec(context.Background(), start_query, now)
// 	if err != nil {
// 		z.Error("更新考试场次状态为'04'时失败", zap.Error(err))
// 	}

// 	end_query := `
// 		Update t_exam_session es
// 		SET status = '06', -- 已结束
// 			update_time = $1
// 		FROM t_exam_info ei
// 		WHERE es.exam_id = ei.id
// 			AND es.status = '04'
// 			AND es.end_time <= $1
// 			AND ei.status != '00' -- 排除未发布的考试
// 			AND NOT EXISTS (
// 				SELECT 1 FROM t_examinee e
// 				WHERE e.exam_session_id = es.id
// 					AND e.status NOT IN ('02', '06', '08', '10', '12')
// 			);
// 	`
// 	_, err = pgxConn.Exec(context.Background(), end_query, now)
// 	if err != nil {
// 		z.Error("更新考试场次状态为'06'时失败", zap.Error(err))
// 	}
// }

// // 考试状态维护，将目前所有考试更新到正确的状态
// func examStatusMaintain() {
// 	z.Info("---->" + cmn.FncName())
// 	defer func() {
// 		if r := recover(); r != nil {
// 			z.Error("Panic recovered in examStatusMaintain",
// 				zap.Any("panic", r),
// 				zap.Stack("stack"))
// 		}
// 	}()

// 	now := time.Now().UnixMilli()
// 	start_query := `
// 		UPDATE t_exam_info ei
// 		SET status = CASE
// 			WHEN EXISTS (
// 				SELECT 1
// 				FROM t_exam_session es
// 				WHERE es.exam_id = ei.id
// 					AND es.status != '14'
// 					AND es.start_time <= $1
// 			)
// 			AND EXISTS (
// 				SELECT 1
// 				FROM t_exam_session es2
// 				WHERE es2.exam_id = ei.id
// 					AND es2.status != '14'
// 					AND NOT EXISTS (
// 						SELECT 1
// 						FROM t_examinee e
// 						WHERE e.exam_session_id = es2.id
// 					)
// 			)
// 			THEN '10' -- 如果尚未配置考生，并且考试开始时间到了，则设置为考试异常状态
// 			WHEN EXISTS (
// 				SELECT 1
// 				FROM t_exam_session es
// 				WHERE es.exam_id = ei.id
// 					AND es.status != '14'
// 					AND es.start_time <= $1
// 			)
// 			THEN '04' -- 如果考试开始时间到了，则设置为进行中状态
// 			ELSE ei.status -- 否则保持原状态
// 		END,
// 		update_time = CASE
// 			WHEN (
// 				EXISTS (
// 					SELECT 1
// 					FROM t_exam_session es
// 					WHERE es.exam_id = ei.id
// 						AND es.status != '14'
// 						AND es.start_time <= $1
// 				)
// 			)
// 			THEN $1
// 			ELSE ei.update_time
// 		END
// 		WHERE ei.status = '02'
// 	`
// 	_, err := pgxConn.Exec(context.Background(), start_query, now)
// 	if err != nil {
// 		z.Error("更新考试状态为'04'时失败", zap.Error(err))
// 	}

// 	end_query := `
// 		UPDATE t_exam_info ei
// 		SET status = '06', -- 已结束
// 			update_time = $1
// 		WHERE ei.status = '04'
// 			AND(
// 				SELECT MAX(es.end_time)
// 				FROM t_exam_session es
// 				WHERE es.exam_id = ei.id
// 					AND es.status != '14'
// 			) <= $1
// 			AND NOT EXISTS (
// 			 	SELECT 1 FROM t_examinee e
// 				JOIN t_exam_session tes ON e.exam_session_id = tes.id
// 				WHERE tes.exam_id = ei.id
// 					AND e.status NOT IN ('02', '06', '08', '10', '12')
// 			)
// 	`

// 	_, err = pgxConn.Exec(context.Background(), end_query, now)
// 	if err != nil {
// 		z.Error("更新考试状态为'06'时失败", zap.Error(err))
// 	}
// }

// func handleMissedEvents() {
// 	z.Info("---->" + cmn.FncName())
// 	defer func() {
// 		if r := recover(); r != nil {
// 			z.Error("Panic recovered in handleMissedEvents",
// 				zap.Any("panic", r),
// 				zap.Stack("stack"))
// 		}
// 	}()

// 	examineesStatusMaintain()
// 	examStatusMaintain()
// 	examSessionStatusMaintain()
// }
