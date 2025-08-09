package grade

import "errors"

var (
	ErrExamIsNotOver   = errors.New("考试未结束")
	ErrInvalidID       = errors.New("无效ID")
	ErrInvalidPage     = errors.New("无效页码")
	ErrInvalidPageSize = errors.New("无效每页数量")
	ErrEmptyQuery      = errors.New("空查询")

	ErrNilDBConn = errors.New("获取数据库连接为空")
)
