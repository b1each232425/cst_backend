package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// 定义提交信息的正则表达式规则
// 例如：要求以 "feat:", "fix:", "docs:" 等开头，后跟描述
// 更多规则可以根据你的团队规范来调整
var validTypes = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"}

func main() {
	// 获取提交信息文件的路径
	// Git Hook 会将提交信息文件路径作为第一个参数传递
	if len(os.Args) < 2 {
		fmt.Println("Error: No commit message file path provided.")
		os.Exit(1)
	}
	commitMsgFile := os.Args[1]

	// 读取提交信息
	file, err := os.Open(commitMsgFile)
	if err != nil {
		fmt.Printf("Error opening commit message file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	commitMsg := scanner.Text()

	// 检查提交信息是否为空
	if strings.TrimSpace(commitMsg) == "" {
		fmt.Println("Error: Commit message cannot be empty.")
		os.Exit(1)
	}

	// 检查是否为合并分支格式
    mergeRe := regexp.MustCompile(`^Merge branch .* into .*$`)
    if mergeRe.MatchString(commitMsg) {
        fmt.Println("Commit message is valid (merge branch detected)!")
        os.Exit(0)
    }

	// 验证提交信息格式
	// 格式：<type>(<scope>): <description>
	re := regexp.MustCompile(`^(\w+)(\(.*\))?: (.*)$`)
	matches := re.FindStringSubmatch(commitMsg)
	if len(matches) < 4 {
		fmt.Printf("Error: Commit message format is incorrect. It should be '<type>(<scope>): <description>'.\n")
		os.Exit(1)
	}

	commitType := matches[1]
	isValidType := false
	for _, t := range validTypes {
		if t == commitType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		fmt.Printf("Error: Invalid commit type '%s'. Valid types are: %s\n", commitType, strings.Join(validTypes, ", "))
		os.Exit(1)
	}

	fmt.Println("Commit message is valid!")
	os.Exit(0)
}