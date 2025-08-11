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
