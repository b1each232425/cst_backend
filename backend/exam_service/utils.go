package exam_service

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

// 设置Redis有序集合
func setRedisTimer(eventType string, id int64, timeStamp int64, event ExamEvent) {
	z.Info("---->" + cmn.FncName())

	ctx := context.Background()

	event.Type = eventType

	eventData, err := json.Marshal(event)
	if err != nil {
		z.Error("Failed to marshal event data", zap.Error(err))
		return
	}

	// 直接使用毫秒时间戳作为score
	score := float64(timeStamp)
	member := string(eventData)

	// 添加到有序集合
	err = redisClient.ZAdd(ctx, EXAM_TIMER_SET_KEY, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		z.Error("Failed to set Redis timer",
			zap.String("eventType", eventType),
			zap.Int64("id", id),
			zap.Float64("score", score),
			zap.Error(err))
	} else {
		z.Debug("Redis timer set",
			zap.String("eventType", eventType),
			zap.Int64("id", id),
			zap.Float64("score", score))
	}
}
