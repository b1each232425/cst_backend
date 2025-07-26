/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-23 14:19:00
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-23 15:11:22
 */
package examPaper

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"math/rand"
)

type JSONText = types.JSONText

func shuffleOptionsAndMapAnswers(r *rand.Rand, Options, Answers []byte) ([]byte, []byte, error) {
	var options []QuestionOption
	if err := json.Unmarshal(Options, &options); err != nil {
		return nil, nil, fmt.Errorf("选项反序列化失败: %w", err)
	}

	//初始化双向映射
	oldToNew := make(map[int]int)
	for i := range options {
		oldToNew[i] = i
	}

	r.Shuffle(len(options), func(i, j int) {
		options[i].Value, options[j].Value = options[j].Value, options[i].Value
		// 更新映射表
		oldToNew[i], oldToNew[j] = oldToNew[j], oldToNew[i]
	})

	var originalAnswers []string
	if err := json.Unmarshal(Answers, &originalAnswers); err != nil {
		return nil, nil, fmt.Errorf("答案反序列化失败: %w", err)
	}

	newAnswers := make([]string, 0)
	for _, originalLabel := range originalAnswers {
		for oldPo, opt := range options {
			if opt.Label == originalLabel {
				newPos := oldToNew[oldPo]
				newAnswers = append(newAnswers, options[newPos].Label)
				break
			}
		}
	}

	// 序列化存储
	newanswers, _ := json.Marshal(newAnswers)
	newoptions, _ := json.Marshal(options)
	return newanswers, newoptions, nil
}
