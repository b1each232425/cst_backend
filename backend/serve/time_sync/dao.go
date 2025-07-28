package time_sync

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"w2w.io/cmn"
)

type Repo interface {
	QueryActualExamEndTime(examineeId int64) (int64, error) // 查询考生的实际考试结束时间
}

type repoImpl struct {
	pgxConn *pgxpool.Pool
}

func NewRepo() Repo {
	return &repoImpl{
		pgxConn: cmn.GetPgxConn(),
	}
}

// QueryActualExamEndTime 查询考生的实际考试结束时间
func (r *repoImpl) QueryActualExamEndTime(examineeId int64) (int64, error) {
	if examineeId <= 0 {
		e := fmt.Errorf("query actual examinee id invalid: %d", examineeId)
		z.Error(e.Error())
		return 0, e
	}

	const query = `
		SELECT actual_end_time 
		FROM v_examinee_info 
		WHERE id = $1
	`

	var actualEndTime time.Time
	err := r.pgxConn.QueryRow(context.Background(), query, examineeId).Scan(&actualEndTime)
	if err != nil {
		e := fmt.Errorf("failed to query actual_end_time for examineeId %d: %w", examineeId, err)
		z.Error(e.Error())
		return 0, e
	}

	return actualEndTime.UnixMilli(), nil
}
