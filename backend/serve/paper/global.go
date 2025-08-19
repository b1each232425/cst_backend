/*
 * @Author: wusaber33
 * @Date: 2025-08-16 16:30:00
 * @LastEditors: wusaber33
 * @LastEditTime: 2025-08-18 23:46:22
 * @FilePath: \assess\backend\serve\paper\global.go
 * @Description: Paper module global constants and variables
 * Copyright (c) 2025 by wusaber33, All Rights Reserved.
 */
package paper

import (
	"time"
)

// actionsWithResult 定义需要返回结果的操作类型集合
var actionsWithResult = map[string]bool{
	"add_question": true, // 添加试题操作需返回新试题ID
	"add_group":    true, // 添加分组操作需返回新分组ID
}

// Constants 定义HTTP请求超时时间
const (
	TIMEOUT = 5 * time.Second // HTTP请求处理超时时间
)

// Constants 定义试卷相关的业务常量
const (
	// 默认分组名称
	DefaultGroup1Name = "一、单选题"
	DefaultGroup2Name = "二、多选题"
	DefaultGroup3Name = "三、判断题"
	DefaultGroup4Name = "四、填空题"
	DefaultGroup5Name = "五、简答题"

	// 记录状态定义
	StatusUnPublished = "00" // 未发布状态
	StatusPublished   = "06" // 已发布状态
	StatusNormal      = "00" // 正常状态
	StatusDeleted     = "02" // 删除状态
	StatusUnNormal    = "04" // 异常状态

	// 试卷分类
	PaperCategoryExam     = "00" // 考试试卷
	PaperCategoryPractice = "02" // 练习试卷

	// 题目类型定义
	QuestionTypeMultiChoice  = "00" // 多选题
	QuestionTypeSingleChoice = "02" // 单选题
	QuestionTypeJudgement    = "04" // 判断题
	QuestionTypeFillBlank    = "06" // 填空题
	QuestionTypeShortAnswer  = "08" // 简答题

	// 试卷难度等级
	Simple = "00" // 简单
	Medium = "02" // 中等
	Hard   = "04" // 困难

	// 默认配置项
	DefaultSuggestedDuration                                                = 120    // 默认答题时长(分钟)
	DefaultPaperName                                                        = "新建试卷" // 默认试卷名称
	PaperShareStatusPrivate, PaperShareStatusShared, PaperShareStatusPublic = "00", "02", "04"
	ManualAssemblyType                                                      = "00"

	//试卷长度限制
	MaxDescription = 500
	MaxPaperName   = 50

	//试卷编辑锁前缀
	REDIS_LOCK_PREFIX     = "paper_lock:"
	REDIS_LOCK_EXPRIATION = 5 * time.Minute
)
