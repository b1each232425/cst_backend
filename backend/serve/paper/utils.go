/*
 * @Author: WangKaidun 1597225095@qq.com
 * @Date: 2025-10-01 10:58:31
 * @LastEditors: WangKaidun 1597225095@qq.com
 * @LastEditTime: 2025-10-03 22:36:16
 * @FilePath: \assess\backend\serve\paper\utils.go
 * @Description: 试卷管理关于工具函数的实现
 * Copyright (c) 2025 by WangKaidun 1597225095@qq.com, All Rights Reserved.
 */
package paper

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
	"w2w.io/cmn"
	"w2w.io/null"
)

func validateIDs(ids []int64) error {
	// 检测数组是否为空
	if len(ids) == 0 {
		err := errors.New("ID List cannot be empty")
		z.Error(err.Error())
		return err
	}
	// 检查ID是否大于0并且没有重复
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			err := errors.Errorf("ID must be greater than 0: %d", id)
			z.Error(err.Error())
			return err
		}
		if seen[id] {
			err := errors.Errorf("Duplicate ID found: %d", id)
			z.Error(err.Error())
			return err
		}
		seen[id] = true
	}
	return nil
}

func ConvertToTPaperGenerationPlan(req PostPaperPlanRequest) *cmn.TPaperGenerationPlan {
	// 转换 Tags
	tagsJSON, _ := json.Marshal(req.Tags)

	// 转换 QuestionBankIDs
	questionBankIDsJSON, _ := json.Marshal(req.QuestionBankIDs)

	// 转换 QuestionConfig
	questionConfigJSON, _ := json.Marshal(req.QuestionConfig)

	return &cmn.TPaperGenerationPlan{
		ID:                null.IntFrom(req.ID),
		Name:              null.StringFrom(req.Name),
		Category:          null.StringFrom(req.Category),
		Level:             null.StringFrom(req.Level),
		SuggestedDuration: null.IntFrom(req.SuggestedDuration),
		Description:       null.StringFrom(req.Description),
		Tags:              types.JSONText(tagsJSON),
		KnowledgeBankID:   null.IntFrom(req.KnowledgeBankID),
		QuestionBankIds:   types.JSONText(questionBankIDsJSON),
		PaperCount:        null.IntFrom(req.PaperCount),
		QuestionConfig:    types.JSONText(questionConfigJSON),
	}
}
