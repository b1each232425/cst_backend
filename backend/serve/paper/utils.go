/*
 * @Author: WangKaidun 1597225095@qq.com
 * @Date: 2025-10-01 10:58:31
 * @LastEditors: WangKaidun 1597225095@qq.com
 * @LastEditTime: 2025-10-04 16:38:50
 * @FilePath: \assess\backend\serve\paper\utils.go
 * @Description: 试卷管理关于工具函数的实现
 * Copyright (c) 2025 by WangKaidun 1597225095@qq.com, All Rights Reserved.
 */
package paper

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
	"w2w.io/cmn"
	"w2w.io/null"
)

func init() {
	// 注册自定义验证器
	err := cmn.RegisterValidation("validate_difficulty_distribution_keys", ValidateDifficultyDistributionKeys)
	if err != nil {
		// 如果注册失败，记录错误但继续运行
		z.Error("Failed to register validate_difficulty_distribution_keys validator: " + err.Error())
	}
}

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

// 注册自定义验证器
func ValidateDifficultyDistributionKeys(fl validator.FieldLevel) bool {
	m := fl.Field().Interface().(map[string]int64)
	validKeys := []string{Easy, FairlyEasy, Medium, FairlyHard, Hard}

	for key, value := range m {
		// 验证键是否有效
		isValidKey := false
		for _, validKey := range validKeys {
			if key == validKey {
				isValidKey = true
				break
			}
		}
		if !isValidKey {
			return false
		}

		// 验证值必须在 0-100 之间
		if value < 0 || value > 100 {
			return false
		}
	}
	return true
}
