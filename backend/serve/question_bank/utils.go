package question_bank

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
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

// 验证单选题
func validateSingleChoiceQuestion(question *cmn.TQuestion) (bool, error) {
	var options []QuestionOption
	err := json.Unmarshal(question.Options, &options)
	if err != nil {
		err = fmt.Errorf("single choice question options format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(options) < 2 {
		err = fmt.Errorf("single choice question must have at least 2 options")
		z.Error(err.Error())
		return false, err
	}

	// 验证选项格式
	optionLabels := make(map[string]bool)
	for _, option := range options {
		if strings.TrimSpace(option.Label) == "" {
			err = fmt.Errorf("single choice question option label cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		if strings.TrimSpace(option.Value) == "" {
			err = fmt.Errorf("single choice question option value cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		// 检查标签重复
		if optionLabels[option.Label] {
			err = fmt.Errorf("single choice question option labels must be unique: %s", option.Label)
			z.Error(err.Error())
			return false, err
		}
		optionLabels[option.Label] = true
	}

	// 验证答案
	var answers []string
	err = json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("single choice question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) != 1 {
		err = fmt.Errorf("single choice question must have exactly one answer, got %d", len(answers))
		z.Error(err.Error())
		return false, err
	}

	// 验证答案是否存在于选项中
	answer := answers[0]
	if !optionLabels[answer] {
		err = fmt.Errorf("single choice question answer '%s' not found in options", answer)
		z.Error(err.Error())
		return false, err
	}

	return true, nil
}

// 验证多选题
func validateMultipleChoiceQuestion(question *cmn.TQuestion) (bool, error) {
	var options []QuestionOption
	err := json.Unmarshal(question.Options, &options)
	if err != nil {
		err = fmt.Errorf("multiple choice question options format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(options) < 3 {
		err = fmt.Errorf("multiple choice question must have at least 3 options")
		z.Error(err.Error())
		return false, err
	}

	// 验证选项格式
	optionLabels := make(map[string]bool)
	for _, option := range options {
		if strings.TrimSpace(option.Label) == "" {
			err = fmt.Errorf("multiple choice question option label cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		if strings.TrimSpace(option.Value) == "" {
			err = fmt.Errorf("multiple choice question option value cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		// 检查标签重复
		if optionLabels[option.Label] {
			err = fmt.Errorf("multiple choice question option labels must be unique: %s", option.Label)
			z.Error(err.Error())
			return false, err
		}
		optionLabels[option.Label] = true
	}

	// 验证答案
	var answers []string
	err = json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("multiple choice question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) >= len(options) {
		err = fmt.Errorf("multiple choice question cannot have all options as correct answers")
		z.Error(err.Error())
		return false, err
	}

	// 验证答案是否存在于选项中，且无重复
	answerSet := make(map[string]bool)
	for _, answer := range answers {
		if !optionLabels[answer] {
			err = fmt.Errorf("multiple choice question answer '%s' not found in options", answer)
			z.Error(err.Error())
			return false, err
		}
		if answerSet[answer] {
			err = fmt.Errorf("multiple choice question has duplicate answer: %s", answer)
			z.Error(err.Error())
			return false, err
		}
		answerSet[answer] = true
	}

	return true, nil
}

// 验证判断题
func validateTrueFalseQuestion(question *cmn.TQuestion) (bool, error) {
	var options []QuestionOption
	err := json.Unmarshal(question.Options, &options)
	if err != nil {
		err = fmt.Errorf("true/false question options format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(options) != 2 {
		err = fmt.Errorf("true/false question must have exactly 2 options, got %d", len(options))
		z.Error(err.Error())
		return false, err
	}

	// 验证选项必须是A和B
	expectedLabels := map[string]bool{"A": false, "B": false}
	for _, option := range options {
		if strings.TrimSpace(option.Label) == "" {
			err = fmt.Errorf("true/false question option label cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		if strings.TrimSpace(option.Value) == "" {
			err = fmt.Errorf("true/false question option value cannot be empty")
			z.Error(err.Error())
			return false, err
		}
		if _, exists := expectedLabels[option.Label]; !exists {
			err = fmt.Errorf("true/false question option labels must be A and B, got: %s", option.Label)
			z.Error(err.Error())
			return false, err
		}
		expectedLabels[option.Label] = true
	}

	// 验证答案
	var answers []string
	err = json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("true/false question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) != 1 {
		err = fmt.Errorf("true/false question must have exactly one answer, got %d", len(answers))
		z.Error(err.Error())
		return false, err
	}

	answer := answers[0]
	if answer != "A" && answer != "B" {
		err = fmt.Errorf("true/false question answer must be A or B, got: %s", answer)
		z.Error(err.Error())
		return false, err
	}

	return true, nil
}

// 验证填空题
func validateFillInBlankQuestion(question *cmn.TQuestion) (bool, error) {
	var answers []SubjectiveAnswer
	err := json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("fill-in-blank question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) == 0 {
		err = fmt.Errorf("fill-in-blank question must have at least one answer")
		z.Error(err.Error())
		return false, err
	}

	// 验证题目内容包含填空标记 - 通过span标签的class属性计算
	content := question.Content.String
	blankCount := countFillInBlanks(content)
	if blankCount == 0 {
		err = fmt.Errorf("fill-in-blank question content must contain blank markers with span tags")
		z.Error(err.Error())
		return false, err
	}

	if blankCount != len(answers) {
		err = fmt.Errorf("fill-in-blank question blank count (%d) does not match answer count (%d)", blankCount, len(answers))
		z.Error(err.Error())
		return false, err
	}

	// 验证每个答案
	indexSet := make(map[int]bool)
	for _, answer := range answers {
		if answer.Index < 1 {
			err = fmt.Errorf("fill-in-blank question answer index must be greater than 0, got: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		if answer.Index > len(answers) {
			err = fmt.Errorf("fill-in-blank question answer index (%d) exceeds answer count (%d)", answer.Index, len(answers))
			z.Error(err.Error())
			return false, err
		}
		if indexSet[answer.Index] {
			err = fmt.Errorf("fill-in-blank question has duplicate answer index: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		indexSet[answer.Index] = true

		if answer.Score <= 0 {
			err = fmt.Errorf("fill-in-blank question answer score must be greater than 0, got: %f", answer.Score)
			z.Error(err.Error())
			return false, err
		}
		if strings.TrimSpace(answer.Answer) == "" {
			err = fmt.Errorf("fill-in-blank question answer content cannot be empty for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		if strings.TrimSpace(answer.GradingRule) == "" {
			err = fmt.Errorf("fill-in-blank question grading rule cannot be empty for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
	}

	return true, nil
}

// 验证简答题
func validateEssayQuestion(question *cmn.TQuestion) (bool, error) {
	var answers []SubjectiveAnswer
	err := json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("essay question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) == 0 {
		err = fmt.Errorf("essay question must have at least one answer")
		z.Error(err.Error())
		return false, err
	}

	// 验证每个小问的答案
	indexSet := make(map[int]bool)
	totalScore := 0.0
	for _, answer := range answers {
		if answer.Index < 1 {
			err = fmt.Errorf("essay question answer index must be greater than 0, got: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		if answer.Index > len(answers) {
			err = fmt.Errorf("essay question answer index (%d) exceeds answer count (%d)", answer.Index, len(answers))
			z.Error(err.Error())
			return false, err
		}
		if indexSet[answer.Index] {
			err = fmt.Errorf("essay question has duplicate answer index: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		indexSet[answer.Index] = true

		if answer.Score <= 0 {
			err = fmt.Errorf("essay question answer score must be greater than 0, got: %f", answer.Score)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.Answer) == "" {
			err = fmt.Errorf("essay question must have answer template for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.GradingRule) == "" {
			err = fmt.Errorf("essay question must have grading rule for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		totalScore += answer.Score
	}

	// 验证总分数是否匹配
	if totalScore != question.Score.Float64 {
		err = fmt.Errorf("essay question total answer score (%f) must match question score (%f)", totalScore, question.Score.Float64)
		z.Error(err.Error())
		return false, err
	}

	return true, nil
}

// 验证综合应用题
func validateComprehensiveQuestion(question *cmn.TQuestion) (bool, error) {
	var answers []SubjectiveAnswer
	err := json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("comprehensive question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) == 0 {
		err = fmt.Errorf("comprehensive question must have at least one answer")
		z.Error(err.Error())
		return false, err
	}

	// 验证每个小问的答案
	indexSet := make(map[int]bool)
	totalScore := 0.0
	for _, answer := range answers {
		if answer.Index < 1 {
			err = fmt.Errorf("comprehensive question answer index must be greater than 0, got: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		if answer.Index > len(answers) {
			err = fmt.Errorf("comprehensive question answer index (%d) exceeds answer count (%d)", answer.Index, len(answers))
			z.Error(err.Error())
			return false, err
		}
		if indexSet[answer.Index] {
			err = fmt.Errorf("comprehensive question has duplicate answer index: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		indexSet[answer.Index] = true

		if answer.Score <= 0 {
			err = fmt.Errorf("comprehensive question answer score must be greater than 0, got: %f", answer.Score)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.Answer) == "" {
			err = fmt.Errorf("comprehensive question must have answer template for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.GradingRule) == "" {
			err = fmt.Errorf("comprehensive question must have grading rule for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		totalScore += answer.Score
	}

	// 验证总分数是否匹配
	if totalScore != question.Score.Float64 {
		err = fmt.Errorf("comprehensive question total answer score (%f) must match question score (%f)", totalScore, question.Score.Float64)
		z.Error(err.Error())
		return false, err
	}

	return true, nil
}

// 验证综合演练题
func validateExerciseQuestion(question *cmn.TQuestion) (bool, error) {
	var answers []SubjectiveAnswer
	err := json.Unmarshal(question.Answers, &answers)
	if err != nil {
		err = fmt.Errorf("exercise question answers format invalid: %v", err)
		z.Error(err.Error())
		return false, err
	}

	if len(answers) == 0 {
		err = fmt.Errorf("exercise question must have at least one answer")
		z.Error(err.Error())
		return false, err
	}

	// 验证每个小问的答案
	indexSet := make(map[int]bool)
	totalScore := 0.0
	for _, answer := range answers {
		if answer.Index < 1 {
			err = fmt.Errorf("exercise question answer index must be greater than 0, got: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		if answer.Index > len(answers) {
			err = fmt.Errorf("exercise question answer index (%d) exceeds answer count (%d)", answer.Index, len(answers))
			z.Error(err.Error())
			return false, err
		}
		if indexSet[answer.Index] {
			err = fmt.Errorf("exercise question has duplicate answer index: %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}
		indexSet[answer.Index] = true

		if answer.Score <= 0 {
			err = fmt.Errorf("exercise question answer score must be greater than 0, got: %f", answer.Score)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.Answer) == "" {
			err = fmt.Errorf("exercise question must have answer template for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		if strings.TrimSpace(answer.GradingRule) == "" {
			err = fmt.Errorf("exercise question must have grading rule for index %d", answer.Index)
			z.Error(err.Error())
			return false, err
		}

		totalScore += answer.Score
	}

	// 验证总分数是否匹配
	if totalScore != question.Score.Float64 {
		err = fmt.Errorf("exercise question total answer score (%f) must match question score (%f)", totalScore, question.Score.Float64)
		z.Error(err.Error())
		return false, err
	}

	return true, nil
}

// countFillInBlanks 计算HTML内容中填空项的数量
// 通过匹配带有特定class的span标签来识别填空项
func countFillInBlanks(content string) int {
	// 只匹配精确的class="blank-item"的span标签
	blankItemPattern := `<span[^>]*class="blank-item"[^>]*>.*?</span>`

	content = strings.ToLower(content) // 转为小写进行匹配

	// 优先匹配blank-item类
	re := regexp.MustCompile(blankItemPattern)
	matches := re.FindAllString(content, -1)
	totalCount := len(matches)

	return totalCount
}

// getKnowledgeBankKnowledges 根据题库ID获取关联的知识点库的knowledges
func getKnowledgeBankKnowledges(ctx context.Context, bankID int64) ([]byte, error) {
	conn := cmn.GetPgxConn()

	// 查询题库关联的知识点库ID，使用null.Int来处理可能的NULL值
	var knowledgeBankID null.Int
	query := `
		SELECT knowledge_bank_id
		FROM t_question_bank
		WHERE id = $1 AND status = '00'
	`

	err := conn.QueryRow(ctx, query, bankID).Scan(&knowledgeBankID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// 题库不存在或已删除，返回空的知识点
			return json.Marshal([]interface{}{})
		}
		return nil, fmt.Errorf("查询题库失败: %v", err)
	}

	// 如果没有关联的知识点库或knowledge_bank_id为NULL，返回空数组
	if !knowledgeBankID.Valid || knowledgeBankID.Int64 == 0 {
		return json.Marshal([]interface{}{})
	}

	// 查询知识点库的knowledges
	var knowledges []byte
	knowledgeQuery := `
		SELECT knowledges
		FROM t_knowledge_bank
		WHERE id = $1 AND status = '00'
	`

	err = conn.QueryRow(ctx, knowledgeQuery, knowledgeBankID.Int64).Scan(&knowledges)
	if err != nil {
		if err == pgx.ErrNoRows {
			// 知识点库不存在或已删除，返回空的知识点
			return json.Marshal([]interface{}{})
		}
		return nil, fmt.Errorf("查询知识点库失败: %v", err)
	}

	// 如果knowledges为空，返回空数组
	if len(knowledges) == 0 || string(knowledges) == "null" {
		return json.Marshal([]interface{}{})
	}

	return knowledges, nil
}

// enrichQuestionsWithAllKnowledges 为题目列表添加allKnowledges字段
func enrichQuestionsWithAllKnowledges(ctx context.Context, questions []cmn.TQuestion, bankID int64) ([]QuestionWithAllKnowledges, error) {
	// 获取知识点库的knowledges
	allKnowledges, err := getKnowledgeBankKnowledges(ctx, bankID)
	if err != nil {
		return nil, fmt.Errorf("获取知识点库失败: %v", err)
	}

	// 构建结果列表
	var result []QuestionWithAllKnowledges
	for _, question := range questions {
		questionWithKnowledges := QuestionWithAllKnowledges{
			TQuestion:     question,
			AllKnowledges: allKnowledges,
		}
		result = append(result, questionWithKnowledges)
	}

	return result, nil
}
