package exam_mgt

import (
	"fmt"
	"strings"
	"time"

	"w2w.io/cmn"
)

// 检查用户是否具有指定域权限的辅助函数
func hasAnyDomainPermissionByID(userDomainIDs []int64, requiredDomainIDs []int64) bool {
	domainSet := make(map[int64]bool)
	for _, domainID := range userDomainIDs {
		domainSet[domainID] = true
	}

	for _, requiredID := range requiredDomainIDs {
		if domainSet[requiredID] {
			return true
		}
	}
	return false
}

// 检查考试数据的有效性
func validateExamData(examData ExamData, isUpdate bool) error {
	z.Info("---->" + cmn.FncName())
	if isUpdate && examData.ExamInfo.ID.Int64 <= 0 {
		err := fmt.Errorf("更新考试时传入的考试ID无效: %d", examData.ExamInfo.ID.Int64)
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
	}

	return nil
}

func validateUserForExamCreate(domain string) (bool, error) {
	z.Info("---->" + cmn.FncName())
	if domain == "" {
		err := fmt.Errorf("无效的用户域: %s", domain)
		z.Error(err.Error())
		return false, err
	}

	// 检查域名是否包含 cst 前缀和 ^admin 权限标识
	if !strings.HasPrefix(domain, "cst") {
		return false, nil
	}

	if !strings.Contains(domain, "^admin") {
		return false, nil
	}

	return true, nil
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
