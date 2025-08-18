package exam_mgt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"w2w.io/cmn"
)

// 检查考试数据的有效性
func validateExamData(examData ExamData, isUpdate bool) error {
	z.Info("---->" + cmn.FncName())
	if isUpdate && examData.ExamInfo.ID.Int64 <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examData.ExamInfo.ID.Int64)
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Name.String == "" {
		err := fmt.Errorf("考试名称不能为空")
		z.Error(err.Error())
		return err
	}

	if len([]rune(examData.ExamInfo.Name.String)) > 50 {
		err := fmt.Errorf("考试名称过长: %d", len([]rune(examData.ExamInfo.Name.String)))
		z.Error(err.Error())
		return err
	}

	if len([]rune(examData.ExamInfo.Rules.String)) > 5000 {
		err := fmt.Errorf("考试规则过长: %d", len([]rune(examData.ExamInfo.Rules.String)))
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Type.String == "" || (examData.ExamInfo.Type.String != "00" && examData.ExamInfo.Type.String != "02" && examData.ExamInfo.Type.String != "04") {
		err := fmt.Errorf("无效的考试类型: %s", examData.ExamInfo.Type.String)
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Mode.String == "" || (examData.ExamInfo.Mode.String != "00" && examData.ExamInfo.Mode.String != "02") {
		err := fmt.Errorf("无效的考试方式: %s", examData.ExamInfo.Mode.String)
		z.Error(err.Error())
		return err
	}

	if len(examData.ExamSessions) == 0 {
		err := fmt.Errorf("考试场次不能为空")
		z.Error(err.Error())
		return err
	}

	for _, examSession := range examData.ExamSessions {
		if examSession.PaperID.Int64 <= 0 {
			err := fmt.Errorf("考试场次的试卷ID无效: %d", examSession.PaperID.Int64)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMethod == "" || (examSession.MarkMethod != "00" && examSession.MarkMethod != "02") {
			err := fmt.Errorf("考试场次的批卷方式无效: %s", examSession.MarkMethod)
			z.Error(err.Error())
			return err
		}

		if examSession.PeriodMode.String == "" || (examSession.PeriodMode.String != "00" && examSession.PeriodMode.String != "02") {
			err := fmt.Errorf("考试场次的考试时段模式无效: %s", examSession.PeriodMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.Duration.Int64 <= 0 {
			err := fmt.Errorf("考试场次的时长无效: %d", examSession.Duration.Int64)
			z.Error(err.Error())
			return err
		}

		if examSession.QuestionShuffledMode.String == "" || (examSession.QuestionShuffledMode.String != "00" && examSession.QuestionShuffledMode.String != "02" && examSession.QuestionShuffledMode.String != "04" && examSession.QuestionShuffledMode.String != "06") {
			err := fmt.Errorf("考试场次的乱序方式无效: %s", examSession.QuestionShuffledMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMode.String == "" || (examSession.MarkMode.String != "00" && examSession.MarkMode.String != "02" && examSession.MarkMode.String != "04" && examSession.MarkMode.String != "06" && examSession.MarkMode.String != "08" && examSession.MarkMode.String != "10") {
			err := fmt.Errorf("考试场次的批改配置无效: %s", examSession.MarkMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.StartTime.Int64 >= examSession.EndTime.Int64 {
			err := fmt.Errorf("考试场次的开始时间晚于或等于结束时间")
			z.Error(err.Error())
			return err
		}

		if examSession.StartTime.Int64 < time.Now().UnixMilli() {
			err := fmt.Errorf("考试场次的开始时间晚于当前时间")
			z.Error(err.Error())
			return err
		}

		//检查设定的考试时长是否大于考试总时长
		startTime := time.UnixMilli(examSession.StartTime.Int64)
		endTime := time.UnixMilli(examSession.EndTime.Int64)
		totalDuration := endTime.Sub(startTime).Minutes()
		if float64(examSession.Duration.Int64) > totalDuration {
			err := fmt.Errorf("设定的考试时长: %d 大于总时长: %f", examSession.Duration.Int64, totalDuration)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMethod != "00" {
			examSession.MarkMode.String = "00"
		}

		if examSession.LateEntryTime.Int64 > int64(totalDuration) {
			err := fmt.Errorf("设定的最迟进入考试时长: %d 不能大于等于总时长: %f", examSession.LateEntryTime.Int64, totalDuration)
			z.Error(err.Error())
			return err
		}

		if examSession.EarlySubmissionTime.Int64 > int64(totalDuration) {
			err := fmt.Errorf("设定的最早交卷时间: %d 不能大于等于总时长: %f", examSession.EarlySubmissionTime.Int64, totalDuration)
			z.Error(err.Error())
			return err
		}

		if examSession.LateEntryTime.Int64 < 0 {
			err := fmt.Errorf("设定的最迟进入考试时长: %d 不能小于0", examSession.LateEntryTime.Int64)
			z.Error(err.Error())
			return err
		}

		if examSession.EarlySubmissionTime.Int64 < 0 {
			err := fmt.Errorf("设定的最早交卷时间: %d 不能小于0", examSession.EarlySubmissionTime.Int64)
			z.Error(err.Error())
			return err
		}
	}

	return nil
}

func validateUserForExamCreateOrUpdate(domain string) bool {
	z.Info("---->" + cmn.FncName())

	// 检查域名是否包含 academicAffair 前缀和 ^admin 权限标识
	// if !strings.HasPrefix(domain, "academicAffair") {
	// 	return false
	// }
	if strings.Contains(domain, "^student") {
		return false
	}

	if !strings.Contains(domain, "^admin") && !strings.Contains(domain, "^superAdmin") && !strings.Contains(domain, "^teacher") {
		return false
	}

	return true
}

// getDomainByUserRole 根据用户角色ID从用户域列表中查找对应的域字符串
func getDomainByUserRole(userRole int64, userDomains []cmn.TDomain) (string, error) {
	z.Info("---->" + cmn.FncName())

	for _, d := range userDomains {
		if d.ID.Valid && d.ID.Int64 == userRole {
			return d.Domain, nil
		}
	}

	err := fmt.Errorf("未找到角色ID %d 对应的域", userRole)
	z.Error(err.Error())
	return "", err
}

// getDomainPrefix 获取域字符串中^前面的部分
func getDomainPrefix(domain string) string {
	parts := strings.Split(domain, "^")
	return parts[0]
}

func getQuestionShuffledMode(mode string) (isQuestionRandom, isOptionRandom bool) {

	isOptionRandom = false
	isQuestionRandom = false

	switch mode {
	case "00": // 既有试题乱序也有选项乱序
		isQuestionRandom = true
		isOptionRandom = true
	case "02": // 选项乱序
		isQuestionRandom = false
		isOptionRandom = true
	case "04": // 试题乱序
		isQuestionRandom = true
		isOptionRandom = false
	case "06": // 都不选择
		isQuestionRandom = false
		isOptionRandom = false
	}

	return
}

// convertToInt64Array 将interface{}转换为[]int64数组
func convertToInt64Array(ctx context.Context, data interface{}) ([]int64, error) {
	if data == nil {
		return []int64{}, nil
	}

	// 如果已经是[]int64类型，直接返回
	if int64Array, ok := data.([]int64); ok {
		return int64Array, nil
	}

	// 如果是[]interface{}类型，需要转换每个元素
	if interfaceArray, ok := data.([]interface{}); ok {
		result := make([]int64, len(interfaceArray))
		for i, item := range interfaceArray {
			switch v := item.(type) {
			case int64:
				result[i] = v
			case float64:
				result[i] = int64(v)
			case int:
				result[i] = int64(v)
			case int32:
				result[i] = int64(v)
			default:
				return nil, fmt.Errorf("unsupported type in array: %T", v)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("unsupported data type: %T", data)
}
