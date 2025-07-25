package client

import (
	"errors"
	"time"
)

var _ Connection = (*Conn)(nil)

var (
	timeDeadline = 20 * time.Second // 心跳超时时间
	ErrConnNil   = errors.New("conn is nil")
)

const (
	TypeTimestamp   = 1 // 时间戳消息类型
	TypeExamEndtime = 2 // 考试结束时刻消息类型
)
