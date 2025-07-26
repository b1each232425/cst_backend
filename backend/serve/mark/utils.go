package mark

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"math/rand"
	"time"
)

func convertAnswerData(rawAnswer RawAnswer) []*AnswerDetails {
	var answerDetails []*AnswerDetails
	for i, answer := range rawAnswer.SubAnswers {
		if answer == "" {
			answer = "无"
		}

		answerDetails = append(answerDetails, &AnswerDetails{
			Content: answer,
			Index:   i,
			Type:    "",
		})
	}
	return answerDetails
}

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

	// 比较两个 map
	if len(countA) != len(countB) {
		return false
	}

	for k, v := range countA {
		if countB[k] != v {
			return false
		}
	}

	return true
}

func ConvertRawStandardAnswerData(raw types.JSONText, type_ string) ([]*StandardAnswer, []string, error) {
	if type_ == "06" || type_ == "08" {
		var resp []*StandardAnswer
		err := json.Unmarshal(raw, &resp)
		if err != nil {
			return nil, nil, err
		}
		return resp, nil, nil
	} else if type_ == "00" || type_ == "02" || type_ == "04" {
		var resp []string
		err := json.Unmarshal(raw, &resp)
		if err != nil {
			return nil, nil, err
		}
		return nil, resp, nil
	} else {
		return nil, nil, fmt.Errorf("unknown question type")
	}

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
