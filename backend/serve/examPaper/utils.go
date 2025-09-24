/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-23 14:19:00
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-05 23:42:54
 */
package examPaper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"math/rand"
	"w2w.io/serve/auth_mgt"
)

type JSONText = types.JSONText
type Map map[string]interface{}

func Json(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}
func shuffleOptionsAndMapAnswers(r *rand.Rand, qid int64, Options, Answers []byte) ([]byte, []byte, error) {
	var options []QuestionOption
	if err := json.Unmarshal(Options, &options); err != nil {
		return nil, nil, fmt.Errorf("考题ID：%v的选项反序列化失败: %w", qid, err)
	}

	labelToValue := make(map[string]string)
	for _, opt := range options {
		labelToValue[opt.Label] = opt.Value
	}

	values := make([]string, len(options))
	for i, opt := range options {
		values[i] = opt.Value
	}
	r.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})

	for i := range options {
		options[i].Value = values[i] // 仅替换值，标签不变
	}

	var originalAnswers []string
	if err := json.Unmarshal(Answers, &originalAnswers); err != nil {
		return nil, nil, fmt.Errorf("考题ID:%v答案反序列化失败: %w", qid, err)
	}

	newAnswers := make([]string, 0, len(originalAnswers))
	for _, origLabel := range originalAnswers {
		originalValue := labelToValue[origLabel]
		for _, opt := range options {
			if opt.Value == originalValue {
				newAnswers = append(newAnswers, opt.Label)
				break
			}
		}
	}

	newOptionsJSON, _ := json.Marshal(options)
	newAnswersJSON, _ := json.Marshal(newAnswers)
	return newAnswersJSON, newOptionsJSON, nil
}
func GetAuthAPIAccessible(ctx context.Context, authority *auth_mgt.Authority, apiPath string) (bool, bool, bool, bool, error) {
	full, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "full")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	if full {
		return true, true, true, true, nil
	}
	read, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "read")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	create, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "create")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	update, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "update")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	deleteble, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "delete")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	return read, create, update, deleteble, nil
}
