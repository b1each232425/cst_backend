package mark

import (
	"math/rand"
	"time"
)

func validateMarkMode(markMode string) bool {
	switch markMode {
	case "00", "02", "04", "06", "08", "10":
		return true
	default:
		return false
	}
}

func CompareSlices(a, b []string) bool {
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(slice) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// 随机均分切片到 n 个新切片
func randomSplit[T any](slice []T, n int) [][]T {
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

	return result
}

// splitSlice 泛型函数，非递归地分割切片，直到每个子切片都满足条件（shouldSplit 返回 false）
// T: 任意类型
// slice: 待分割的切片
// shouldSplit: 判断函数，传入一个子切片，返回 true 表示需要继续分割（不满足条件），false 表示满足条件
func splitSlice[T any](slice []T, shouldSplit func([]T) bool) [][]T {
	if len(slice) == 0 {
		return [][]T{}
	}

	// 使用 slice 模拟队列，存储待处理的切片
	var queue [][]T = [][]T{slice}
	var result [][]T

	for len(queue) > 0 {
		// 取出队首切片
		current := queue[0]
		queue = queue[1:]

		// 判断当前切片是否需要继续分割
		if !shouldSplit(current) {
			// 满足条件，加入结果集
			result = append(result, current)
			continue
		}

		// 如果只有一个元素，但依然需要分割，也只好保留（无法再分割）
		if len(current) == 1 {
			result = append(result, current)
			continue
		}

		// 对半分割
		n := len(current)
		mid := n / 2
		left := current[:mid]
		right := current[mid:]

		// 将左右子切片加入队列继续处理
		queue = append(queue, left, right)
	}

	return result
}
