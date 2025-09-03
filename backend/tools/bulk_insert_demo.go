/*
 * @Author: wusaber33
 * @Date: 2025-01-18 10:00:00
 * @LastEditors: wusaber33
 * @LastEditTime: 2025-09-03 16:47:58
 * @FilePath: \assess\backend\tools\bulk_insert_demo.go
 * @Description: Demo file showing how to use bulk paper insertion
 * Copyright (c) 2025 by wusaber33, All Rights Reserved.
 */
package tools

import (
	"fmt"
	"log"
	"time"
)

// RunBulkInsertDemo 运行批量插入演示
func RunBulkInsertDemo() {
	fmt.Println("=== 批量插入试卷数据演示 ===")

	// 初始化必要的组件
	fmt.Println("1. 初始化系统组件...")

	// 这里需要根据实际情况初始化数据库连接等
	// cmn.InitializeSystem() // 假设有这样的初始化函数

	fmt.Println("2. 演示数据生成...")

	// 演示小规模数据生成
	demoConfig := BulkPaperConfig{
		PaperCount:        10, // 演示只生成10份试卷
		QuestionsPerPaper: 15, // 每份15道题
		CreatorID:         "demo_user",
		BatchSize:         5, // 每批5份
	}

	fmt.Printf("配置信息：\n")
	fmt.Printf("- 试卷数量: %d\n", demoConfig.PaperCount)
	fmt.Printf("- 每份试卷题目数: %d\n", demoConfig.QuestionsPerPaper)
	fmt.Printf("- 创建者ID: %s\n", demoConfig.CreatorID)
	fmt.Printf("- 批次大小: %d\n", demoConfig.BatchSize)

	fmt.Println("\n3. 生成试卷数据...")
	start := time.Now()
	papers := createPaperData(demoConfig)
	generateDuration := time.Since(start)

	fmt.Printf("✅ 成功生成 %d 份试卷，耗时: %v\n", len(papers), generateDuration)

	// 展示第一份试卷的详细信息
	if len(papers) > 0 {
		paper := papers[0]
		fmt.Printf("\n📋 示例试卷信息：\n")
		fmt.Printf("- ID: %s\n", paper.ID)
		fmt.Printf("- 名称: %s\n", paper.Name)
		fmt.Printf("- 描述: %s\n", paper.Description)
		fmt.Printf("- 类别: %s\n", paper.Category)
		fmt.Printf("- 难度: %s\n", paper.Difficulty)
		fmt.Printf("- 建议时长: %d分钟\n", paper.SuggestedDuration)
		fmt.Printf("- 分组数量: %d\n", len(paper.Groups))

		// 展示分组信息
		totalQuestions := 0
		for i, group := range paper.Groups {
			fmt.Printf("  分组%d: %s (%d道题)\n", i+1, group.Name, len(group.Questions))
			totalQuestions += len(group.Questions)
		}
		fmt.Printf("- 总题目数: %d\n", totalQuestions)
	}

	fmt.Println("\n4. 运行单元测试...")
	runQuickTests()

	fmt.Println("\n5. 性能统计...")
	showPerformanceStats(demoConfig, generateDuration)

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("\n💡 如需执行实际的数据库插入，请使用：")
	fmt.Println("   go run main.go bulk-insert [试卷数量] [每份题目数] [创建者ID]")
	fmt.Println("\n   示例：")
	fmt.Println("   go run main.go bulk-insert 1000 20 test_user")
	fmt.Println("   go run main.go bulk-insert  # 使用默认配置（10000份试卷）")
}

// runQuickTests 运行快速测试
func runQuickTests() {
	tests := []struct {
		name string
		fn   func() bool
	}{
		{"随机名称生成", testRandomNameGeneration},
		{"随机ID生成", testRandomIDGeneration},
		{"数据一致性", testDataConsistency},
	}

	for _, test := range tests {
		if test.fn() {
			fmt.Printf("  ✅ %s: 通过\n", test.name)
		} else {
			fmt.Printf("  ❌ %s: 失败\n", test.name)
		}
	}
}

// testRandomNameGeneration 测试随机名称生成
func testRandomNameGeneration() bool {
	names := make(map[string]bool)
	for i := 0; i < 50; i++ {
		name := generateRandomPaperName()
		if name == "" {
			return false
		}
		names[name] = true
	}
	return len(names) >= 20 // 至少生成20种不同的名称
}

// testRandomIDGeneration 测试随机ID生成
func testRandomIDGeneration() bool {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateRandomID("test")
		if id == "" || ids[id] {
			return false
		}
		ids[id] = true
	}
	return true
}

// testDataConsistency 测试数据一致性
func testDataConsistency() bool {
	config := BulkPaperConfig{
		PaperCount:        5,
		QuestionsPerPaper: 10,
		CreatorID:         "test_user",
		BatchSize:         2,
	}

	papers := createPaperData(config)
	if len(papers) != config.PaperCount {
		return false
	}

	for _, paper := range papers {
		if len(paper.Groups) != 5 {
			return false
		}

		totalQuestions := 0
		for _, group := range paper.Groups {
			if group.PaperID != paper.ID {
				return false
			}

			for _, question := range group.Questions {
				if question.PaperID != paper.ID || question.GroupID != group.ID {
					return false
				}
				totalQuestions++
			}
		}

		if totalQuestions != config.QuestionsPerPaper {
			return false
		}
	}

	return true
}

// showPerformanceStats 显示性能统计
func showPerformanceStats(config BulkPaperConfig, duration time.Duration) {
	fmt.Printf("📊 性能统计：\n")
	fmt.Printf("- 数据生成速度: %.2f 试卷/秒\n", float64(config.PaperCount)/duration.Seconds())
	fmt.Printf("- 平均每份试卷生成时间: %v\n", duration/time.Duration(config.PaperCount))
	fmt.Printf("- 内存使用估算: ~%.2f MB\n", estimateMemoryUsage(config))

	// 预估完整批量插入的时间
	fullScale := 10000
	estimatedTime := duration * time.Duration(fullScale/config.PaperCount)
	fmt.Printf("- 预估生成10000份试卷需要: %v\n", estimatedTime)
}

// estimateMemoryUsage 估算内存使用量
func estimateMemoryUsage(config BulkPaperConfig) float64 {
	// 粗略估算每份试卷的内存占用
	paperSize := 1.0    // 试卷基本信息 ~1KB
	groupSize := 0.2    // 每个分组 ~0.2KB
	questionSize := 0.1 // 每道题 ~0.1KB

	totalSize := float64(config.PaperCount) * (paperSize + 5*groupSize + float64(config.QuestionsPerPaper)*questionSize)
	return totalSize / 1024 // 转换为MB
}

// ExecuteBulkInsertWithLogging 执行带日志的批量插入
func ExecuteBulkInsertWithLogging(paperCount, questionsPerPaper int, creatorID string) error {
	start := time.Now()

	fmt.Printf("🚀 开始批量插入 %d 份试卷...\n", paperCount)
	fmt.Printf("📊 配置详情：每份 %d 道题，创建者：%s\n", questionsPerPaper, creatorID)

	err := BulkInsertPapersCustom(paperCount, questionsPerPaper, creatorID)

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 批量插入失败：%v\n", err)
		return err
	}

	fmt.Printf("✅ 批量插入成功完成！\n")
	fmt.Printf("⏱️  总耗时：%v\n", duration)
	fmt.Printf("📈 插入速度：%.2f 试卷/秒\n", float64(paperCount)/duration.Seconds())

	return nil
}

// main 函数用于独立运行演示
func main() {
	// 如果直接运行此文件，执行演示
	if err := recover(); err != nil {
		log.Printf("演示运行出错: %v", err)
	}

	RunBulkInsertDemo()
}
