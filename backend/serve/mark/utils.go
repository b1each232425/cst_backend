package mark

import (
	"errors"
	"math/rand"
	"strings"
	"time"
	"w2w.io/cmn"
)

// 比较切片内容（不限数据顺序）
func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	countA := make(map[string]int)
	countB := make(map[string]int)

	// 统计 a 中的元素
	for _, v := range a {
		countA[v]++
	}

	// 统计 b 中的元素
	for _, v := range b {
		countB[v]++
	}

	for k, v := range countA {
		if countB[k] != v {
			return false
		}
	}

	return true
}

// 随机打乱切片（Fisher-Yates 算法）
func shuffleSlice[T any](slice []T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) // 使用自定义随机源，保证并发安全

	r.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

// 随机均分切片到 n 个新切片
func randomSplit[T any](slice []T, n int) ([][]T, error) {
	if n <= 0 {
		return nil, errors.New("分片数量 n 必须大于 0")
	}

	// 先打乱原切片
	shuffled := make([]T, len(slice))
	copy(shuffled, slice)
	shuffleSlice(shuffled)

	// 计算每个新切片的大小
	size := len(shuffled) / n
	remainder := len(shuffled) % n

	result := make([][]T, n)
	start := 0

	for i := 0; i < n; i++ {
		end := start + size
		if i < remainder {
			end++ // 前 remainder 个切片多分一个元素
		}
		result[i] = shuffled[start:end]
		start = end
	}

	return result, nil
}

// 校验 QueryCondition
func validateExamSessionOrPractice(cond QueryCondition) error {
	if cond.PracticeID <= 0 && cond.ExamSessionID <= 0 {
		return errors.New("无效的 cond 参数，必须包含 考试场次ID 或者 练习ID 中的一个")
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		return errors.New("无效的 cond 参数，不能同时包含 考试场次ID 和 练习ID")
	}

	return nil
}

// 用于 sql 拼接
func joinWhereClause(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " AND " + strings.Join(clauses, " AND ")
}

// 校验 markingResult 是否具有必要的 ID
func validateMarkingResult(markingResult cmn.TMark) error {
	if markingResult.QuestionID.Int64 <= 0 {
		return errors.New("缺少 问题ID")
	}

	if markingResult.TeacherID.Int64 <= 0 {
		return errors.New("缺少 老师ID")
	}

	if markingResult.Creator.Int64 <= 0 {
		return errors.New("缺少 创建者ID")
	}

	switch {
	case markingResult.ExamSessionID.Int64 > 0:
		if markingResult.ExamineeID.Int64 <= 0 {
			return errors.New("缺少 考生ID")
		}

	case markingResult.PracticeID.Int64 > 0:
		if markingResult.PracticeSubmissionID.Int64 <= 0 {
			return errors.New("缺少 练习提交ID")
		}

	default:
		return errors.New("必须包含 考试场次ID 或者 练习ID 其中一个")
	}

	return nil
}

// splitSlice 泛型函数，非递归地分割切片，直到每个子切片都满足条件（shouldSplit 返回 false）
// T: 任意类型
// slice: 待分割的切片
// shouldSplit: 判断函数，传入一个子切片，返回 true 表示需要继续分割（不满足条件），false 表示满足条件
//func splitSlice[T any](slice []T, shouldSplit func([]T) bool) [][]T {
//	if len(slice) == 0 {
//		return [][]T{}
//	}
//
//	// 使用 slice 模拟队列，存储待处理的切片
//	var queue [][]T = [][]T{slice}
//	var result [][]T
//
//	for len(queue) > 0 {
//		// 取出队首切片
//		current := queue[0]
//		queue = queue[1:]
//
//		// 判断当前切片是否需要继续分割
//		if !shouldSplit(current) {
//			// 满足条件，加入结果集
//			result = append(result, current)
//			continue
//		}
//
//		// 如果只有一个元素，但依然需要分割，也只好保留（无法再分割）
//		if len(current) == 1 {
//			result = append(result, current)
//			continue
//		}
//
//		// 对半分割
//		n := len(current)
//		mid := n / 2
//		left := current[:mid]
//		right := current[mid:]
//
//		// 将左右子切片加入队列继续处理
//		queue = append(queue, left, right)
//	}
//
//	return result
//}
