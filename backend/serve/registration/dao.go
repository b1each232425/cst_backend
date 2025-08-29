package registration

import (
	"context"
	"w2w.io/cmn"
)

// 学生查看报名计划列表
func ListRegisterS(ctx context.Context, name string, course string, status string, orderBy []string, page int, pageSize int, userID int64) ([]*cmn.TRegisterPlan, int, error) {
	result := make([]Map, 0)
	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}

}
