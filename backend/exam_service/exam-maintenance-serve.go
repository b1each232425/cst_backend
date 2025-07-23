package exam_service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

//annotation:exam_mgt
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

// 更新/维护考试状态的轮询服务

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

// 考生状态维护
func examineesStatusMaintain() {
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in examineesStatusMaintain",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	now := time.Now().UnixMilli()
	query := `
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
			JOIN t_exam_session es ON e.exam_session_id = es.id
			JOIN t_exam_info ei ON es.exam_id = ei.id
			JOIN v_examinee_info v ON e.id = v.id
			WHERE v.actual_end_time <= $1
				AND e.status IN ('00', '16') -- 只更新处于正常考和监考员允许进入考试状态的考生
				AND es.status != '14' -- 只更新未被删除的考试场次
				AND ei.status IN ('04', '06') 
		)
		UPDATE t_examinee t
		SET status = u.new_status,
			end_time = COALESCE(u.new_end_time, t.end_time),
			update_time = NOW()
		FROM examinee_to_update u
		WHERE t.id = u.id
	`

	cmdTag, err := pgxConn.Exec(context.Background(), query, now)
	if err != nil {
		z.Error("Failed to update examinee status", zap.Error(err))
	}

	z.Debug("Examinee status updated",
		zap.Int64("rows_affected", cmdTag.RowsAffected()),
	)

}

// 考试场次状态维护
func examSessionStatusMaintain() {
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in examSessionStatusMaintain",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	now := time.Now().UnixMilli()
	start_query := `
		Update t_exam_session es
		SET status = '04', -- 进行中
			update_time = $1
		FROM t_exam_info ei
		WHERE es.exam_id = ei.id
			AND es.status = '02'
			AND es.start_time <= $1
			AND ei.status NOT IN ('00', '10', '12'); -- 排除未发布、考试异常和已删除的考试
	`
	cmdTag1, err := pgxConn.Exec(context.Background(), start_query, now)
	if err != nil {
		z.Error("Failed to update exam session status to '04'", zap.Error(err))
	} else {
		z.Debug("Exam session status updated to '04'",
			zap.Int64("rows_affected", cmdTag1.RowsAffected()),
		)
	}

	end_query := `
		Update t_exam_session es 
		SET status = '06', -- 已结束
			update_time = $1
		FROM t_exam_info ei
		WHERE es.exam_id = ei.id
			AND es.status = '04'
			AND es.end_time <= $1
			AND ei.status != '00' -- 排除未发布的考试
			AND NOT EXISTS (
				SELECT 1 FROM t_examinee e
				WHERE e.exam_session_id = es.id
					AND e.status NOT IN ('02', '06', '08', '10', '12')
			);
	`
	cmdTag2, err := pgxConn.Exec(context.Background(), end_query, now)
	if err != nil {
		z.Error("Failed to update exam session status to '06'", zap.Error(err))
	} else {
		z.Debug("Exam session status updated to '06'",
			zap.Int64("rows_affected", cmdTag2.RowsAffected()),
		)
	}
}

// 考试状态维护
func examStatusMaintain() {
	defer func() {
		if r := recover(); r != nil {
			z.Error("Panic recovered in examStatusMaintain",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	now := time.Now().UnixMilli()
	start_query := `
		UPDATE t_exam_info ei
		SET status = CASE
			WHEN EXISTS (
				SELECT 1
				FROM t_exam_session es
				WHERE es.exam_id = ei.id
					AND es.status != '14'
					AND es.start_time <= $1
			)
			AND EXISTS (
				SELECT 1
				FROM t_exam_session es2
				WHERE es2.exam_id = ei.id
					AND es2.status != '14'
					AND NOT EXISTS (
						SELECT 1
						FROM t_examinee e
						WHERE e.exam_session_id = es2.id
					)
			)
			THEN '10' -- 如果尚未配置考生，并且考试开始时间到了，则设置为考试异常状态
			WHEN EXISTS (
				SELECT 1
				FROM t_exam_session es
				WHERE es.exam_id = ei.id
					AND es.status != '14'
					AND es.start_time <= $1
			)
			THEN '04' -- 如果考试开始时间到了，则设置为进行中状态
			ELSE ei.status -- 否则保持原状态
		END,
		update_time = CASE
			WHEN (
				EXISTS (
					SELECT 1
					FROM t_exam_session es
					WHERE es.exam_id = ei.id
						AND es.status != '14'
						AND es.start_time <= $1
				)
			)
			THEN $1
			ELSE ei.update_time
		END
		WHERE ei.status = '02'
	`
	cmdTag1, err := pgxConn.Exec(context.Background(), start_query, now)
	if err != nil {
		z.Error("Failed to update exam status to '04'", zap.Error(err))
	} else {
		z.Debug("Exam status updated to '04'",
			zap.Int64("rows_affected", cmdTag1.RowsAffected()),
		)
	}

	end_query := `
		UPDATE t_exam_info ei
		SET status = '06', -- 已结束
			update_time = $1
		WHERE ei.status = '04'
			AND(
				SELECT MAX(es.end_time)
				FROM t_exam_session es
				WHERE es.exam_id = ei.id
					AND es.status != '14'
			) <= $1
			AND NOT EXISTS (
			 	SELECT 1 FROM t_examinee e
				JOIN t_exam_session tes ON e.exam_session_id = tes.id
				WHERE tes.exam_id = ei.id
					AND e.status NOT IN ('02', '06', '08', '10', '12')
			)
	`

	cmdTag2, err := pgxConn.Exec(context.Background(), end_query, now)
	if err != nil {
		z.Error("Failed to update exam status to '06'", zap.Error(err))
	} else {
		z.Debug("Exam status updated to '06'",
			zap.Int64("rows_affected", cmdTag2.RowsAffected()),
		)
	}
}

func ExamMaintainService() {
	z.Info("Exam maintenance service started")

	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))

		examineesStatusMaintain()
		examStatusMaintain()
		examSessionStatusMaintain()
	}

}
