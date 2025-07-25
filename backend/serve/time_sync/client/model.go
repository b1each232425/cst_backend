package client

type ReceiveMsg struct {
	MsgType int64 `json:"msg_type"` // 消息类型
}

type SendMsg struct {
	MsgType   int64 `json:"msg_type"`  // 消息类型
	EndTime   int64 `json:"end_time"`  // 考试结束时间
	Timestamp int64 `json:"timestamp"` // 当前时间戳
}
