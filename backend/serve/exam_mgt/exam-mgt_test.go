package exam_mgt

//annotation:exam_mgt
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"
)

func createMockContext(method, path string, queryParams url.Values, forceError string, userID int64) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.NewInt(userID, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

func createMockContextWithBody(method, path string, data string, forceError string, userID int64) context.Context {
	var req *http.Request

	if data != "" {
		// 创建ReqProto结构体，Data字段使用json.RawMessage类型
		body := &cmn.ReqProto{
			Data: json.RawMessage(data),
		}

		// 将请求体转换为JSON字符串
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			e := fmt.Sprintf("Failed to marshal request data: %v", err)
			z.Fatal(e)
		}

		// 创建mock HTTP请求
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.Header.Set("Content-Type", "application/json")

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.NewInt(userID, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestGenerateExamineeNumber(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber int64
		examInfo     cmn.TExamInfo
		examSessions []cmn.TExamSession
		expected     string
		description  string
	}{
		{
			name:         "线上考试模式-返回空字符串",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("00"), // 线上考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "",
			description: "线上考试不生成准考证号",
		},
		{
			name:         "线下考试模式-正常生成准考证号",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("02"), // 线下考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "24123000001",
			description: "24(年份) + 123(考试ID) + 000001(序号)",
		},
		{
			name:         "大序号测试-超过6位数",
			serialNumber: 1234567,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(789),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "257891234567",
			description: "序号超过6位时正常显示",
		},
		{
			name:         "小序号测试-需要补零",
			serialNumber: 5,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(1),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2023, 7, 20, 14, 30, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "231000005",
			description: "序号5 -> 000005",
		},
		{
			name:         "序号为0的情况",
			serialNumber: 0,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "25999000000",
			description: "序号0 -> 000000",
		},
		{
			name:         "大考试ID测试",
			serialNumber: 100,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2026, 3, 15, 16, 45, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "26999999000100",
			description: "大考试ID正常处理",
		},
		{
			name:         "空考试场次-使用当前时间",
			serialNumber: 50,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(777),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{},
			expected:     "",
			description:  "空场次时使用当前年份",
		},
		{
			name:         "无效的开始时间-使用当前时间",
			serialNumber: 25,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(888),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(0), // 无效时间戳
				},
			},
			expected:    "",
			description: "无效时间戳时使用当前年份",
		},
		{
			name:         "StartTime不Valid的情况",
			serialNumber: 10,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(555),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.Int{}, // StartTime不Valid
				},
			},
			expected:    "",
			description: "StartTime不Valid时使用当前年份",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExamineeNumber(tt.serialNumber, tt.examInfo, tt.examSessions)

			if tt.expected == "" {
				// 对于使用当前时间的情况，我们需要动态验证
				if tt.examInfo.Mode.String == "00" {
					// 线上考试应该返回空字符串
					if result != "" {
						t.Errorf("generateExamineeNumber() = %v, 期望空字符串", result)
					}
				} else if len(tt.examSessions) == 0 || !tt.examSessions[0].StartTime.Valid || tt.examSessions[0].StartTime.Int64 <= 0 {
					// 使用当前时间的情况，验证格式是否正确
					currentYear := time.Now().Year() % 100
					expectedPrefix := fmt.Sprintf("%02d%d", currentYear, tt.examInfo.ID.Int64)
					expectedSuffix := fmt.Sprintf("%06d", tt.serialNumber)
					expectedResult := expectedPrefix + expectedSuffix

					if result != expectedResult {
						t.Errorf("generateExamineeNumber() = %v, 期望 %v (使用当前年份 %d)", result, expectedResult, currentYear)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("generateExamineeNumber() = %v, 期望 %v (%s)", result, tt.expected, tt.description)
				}
			}
		})
	}
}

func TestValidateExamData(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name      string
		examData  ExamData
		isUpdate  bool
		wantError bool
		errorMsg  string
	}{
		{
			name: "有效的新建考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("期末考试"),
					Type: null.StringFrom("02"), // 期末成绩考试
					Mode: null.StringFrom("00"), // 线上考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",                  // 人工批卷
						PeriodMode:           null.StringFrom("00"), // 固定时段
						Duration:             null.IntFrom(120),     // 120分钟
						QuestionShuffledMode: null.StringFrom("00"), // 既有试题乱序也有选项乱序
						MarkMode:             null.StringFrom("00"), // 不需要手动批改
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "有效的更新考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(123),
					Name: null.StringFrom("期中考试"),
					Type: null.StringFrom("00"), // 平时考试
					Mode: null.StringFrom("02"), // 线下考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(2),
						MarkMethod:           "02",                  // 自动批卷
						PeriodMode:           null.StringFrom("02"), // 灵活时段
						Duration:             null.IntFrom(90),      // 90分钟
						QuestionShuffledMode: null.StringFrom("02"), // 选项乱序
						MarkMode:             null.StringFrom("02"), // 全卷多评
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(90 * time.Minute).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: false,
		},
		{
			name: "更新时考试ID无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(0), // 无效ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "更新考试时传入的考试ID无效",
		},
		{
			name: "考试名称为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom(""), // 空名称
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试名称不能为空",
		},
		{
			name: "无效的考试类型",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("99"), // 无效类型
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "无效的考试方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("99"), // 无效方式
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "考试场次为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{}, // 空场次
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次不能为空",
		},
		{
			name: "无效的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(0), // 无效试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "无效的批卷方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "99", // 无效批卷方式
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "考试时长无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(0), // 无效时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
		{
			name: "开始时间晚于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "开始时间等于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()), // 开始时间等于结束时间
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "考试类型为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom(""), // 空字符串
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "考试方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom(""), // 空字符串
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "无效的时段模式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("99"), // 无效时段模式
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "时段模式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom(""), // 空字符串
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "无效的乱序方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("99"), // 无效乱序方式
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "乱序方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom(""), // 空字符串
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "无效的批改配置",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("99"), // 无效批改配置
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批改配置为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom(""), // 空字符串
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批卷方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "", // 空字符串
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "设定的考试时长大于总时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(180), // 180分钟，但总时长只有60分钟
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()), // 只有1小时
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "设定的考试时长",
		},
		{
			name: "负数的考试ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(-1), // 负数ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "更新考试时传入的考试ID无效",
		},
		{
			name: "负数的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(-1), // 负数试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "负数的考试时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(-10), // 负数时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExamData(tt.examData, tt.isUpdate)

			if tt.wantError {
				if err == nil {
					t.Errorf("validateExamData() 期望返回错误，但实际返回 nil")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateExamData() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateExamData() 期望返回 nil，但实际返回错误 = %v", err)
				}
			}
		})
	}
}

// cleanupTestData 清理测试过程中插入的数据
func cleanupTestData(t *testing.T, creators []int64) {
	if len(creators) == 0 {
		return
	}

	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// 删除考生记录
	for _, creator := range creators {
		_, err = tx.Exec(ctx, `
			DELETE FROM t_examinee 
			WHERE exam_session_id IN (
				SELECT id FROM t_exam_session WHERE creator = $1
			)`, creator)
		if err != nil {
			t.Logf("删除考生记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试场次记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试场次记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试信息记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试信息记录失败 (creator=%d): %v", creator, err)
		}
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Logf("提交清理事务失败: %v", err)
	} else {
		t.Logf("成功清理 %d 个测试考试的数据", len(creators))
	}
}

func TestExamPostMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	var creators []int64
	creators = append(creators, 99999)
	t.Cleanup(func() {
		cleanupTestData(t, creators)
	})

	tests := []struct {
		name          string
		requestBody   string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
	}{
		{
			name: "有效的考试创建请求",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "",
			expectedError: false,
			description:   "正常的考试创建请求",
			userID:        99999,
		},
		{
			name:          "空请求体",
			requestBody:   "",
			forceError:    "",
			expectedError: true,
			errorContains: "empty body",
			description:   "请求体为空时应该返回错误",
			userID:        99999,
		},
		{
			name: "考试名称为空",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "",
			expectedError: true,
			errorContains: "考试名称不能为空",
			description:   "考试名称为空时应该返回验证错误",
			userID:        99999,
		},
		{
			name: "强制JSON解析错误",
			requestBody: `{
				"examInfo": {
					"Name": "测试考试",
					"Type": "00",
					"Mode": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "json.Unmarshal",
			expectedError: true,
			description:   "模拟JSON解析失败",
			userID:        99999,
		},
		{
			name: "强制JSON解析错误",
			requestBody: `{
				"examInfo": {
					"Name": "测试考试",
					"Type": "00",
					"Mode": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "json.Unmarshal2",
			expectedError: true,
			description:   "模拟JSON解析失败2",
			userID:        99999,
		},
		{
			name: "强制事务开始失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Begin",
			expectedError: true,
			description:   "模拟事务开始失败",
			userID:        99999,
		},
		{
			name: "强制事务回滚失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Rollback",
			expectedError: false,
			description:   "模拟事务回滚失败（通过验证失败触发回滚）",
			userID:        99999,
		},
		{
			name: "强制事务提交失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Commit",
			expectedError: false,
			description:   "模拟事务提交失败",
			userID:        99999,
		},
		{
			name: "无效的UserID",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "",
			expectedError: true,
			description:   "无效的UserID",
			userID:        0,
		},
		{
			name: "读取请求体错误",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "io.ReadAll",
			expectedError: true,
			description:   "读取请求体错误",
			userID:        99999,
		},
		{
			name: "关闭IO错误",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "io.Close",
			expectedError: false,
			description:   "关闭IO错误",
			userID:        99999,
		},
		{
			name: "QueryRow错误1",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.QueryRow1",
			expectedError: true,
			description:   "tx.QueryRow错误1",
			userID:        99999,
		},
		{
			name: "QueryRow错误2",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.QueryRow2",
			expectedError: true,
			description:   "tx.QueryRow错误2",
			userID:        99999,
		},
		{
			name: "Exec错误",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSession": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Exec",
			expectedError: true,
			description:   "tx.Exec错误",
			userID:        99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文
			ctx := createMockContextWithBody("POST", "/api/exam", tt.requestBody, tt.forceError, tt.userID)

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("exam() 意外panic: %v", r)
						}
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}
		})
	}
}

func TestExamGetMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name          string
		query         string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
	}{
		{
			name:          "有效的考试查询请求",
			query:         "?examID=1",
			forceError:    "",
			expectedError: false,
			description:   "正常的考试查询请求",
			userID:        99999,
		},
		{
			name:          "无效的考试ID",
			query:         "?examID=abc",
			forceError:    "",
			expectedError: true,
			errorContains: "invalid exam ID",
			description:   "无效的考试ID",
			userID:        99999,
		},
		{
			name:          "缺少考试ID",
			query:         "",
			forceError:    "",
			expectedError: true,
			errorContains: "missing exam ID",
			description:   "缺少考试ID",
			userID:        99999,
		},
		{
			name:          "强制JSON解析错误",
			query:         "?examID=1",
			forceError:    "json.Unmarshal",
			expectedError: true,
			description:   "模拟JSON解析失败",
			userID:        99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

		})
	}
}

func TestValidateUserExamPermission(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据 - 需要先创建测试用的考试和考生数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的考试ID列表
	var testExamIDs []int64
	var testUserIDs []int64
	var testSessionIDs []int64

	// 清理函数
	defer func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Logf("开始清理事务失败: %v", err)
			return
		}
		defer tx.Rollback(ctx)

		// 清理考生记录
		for _, sessionID := range testSessionIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_examinee WHERE exam_session_id = $1`, sessionID)
			if err != nil {
				t.Logf("清理考生记录失败: %v", err)
			}
		}

		// 清理考试场次
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE exam_id = $1`, examID)
			if err != nil {
				t.Logf("清理考试场次失败: %v", err)
			}
		}

		// 清理考试记录
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败: %v", err)
			}
		}

		// 清理用户记录
		for _, userID := range testUserIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_user WHERE id = $1`, userID)
			if err != nil {
				t.Logf("清理用户记录失败: %v", err)
			}
		}

		err = tx.Commit(ctx)
		if err != nil {
			t.Logf("提交清理事务失败: %v", err)
		}
	}()

	// 创建测试用户
	testTeacherID := int64(99001)
	testStudentID := int64(99002)
	testAdminID := int64(99003)
	testUserIDs = append(testUserIDs, testTeacherID, testStudentID, testAdminID)

	currentTime := time.Now().UnixMilli()

	// 插入测试用户
	_, err := conn.Exec(ctx, `
		INSERT INTO t_user (id, role, account, category, official_name, create_time, update_time, status) 
		VALUES 
			($1, 2, 'test_teacher', '00', '测试教师', $4, $4, '00'),
			($2, 1, 'test_student', '00', '测试学生', $4, $4, '00'),
			($3, 3, 'test_admin', '00', '测试管理员', $4, $4, '00')
		ON CONFLICT (id) DO NOTHING`,
		testTeacherID, testStudentID, testAdminID, currentTime)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 创建测试考试（由testTeacherID创建）
	var testExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'测试权限考试', '00', '00', $1, $2, $1, $2, '00'
		) RETURNING id`,
		testTeacherID, currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建测试考试场次
	var testSessionID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 1, 123, $2, $3, 120, '00', '00', '00', '00', '00', $4, $2, $4, $2
		) RETURNING id`,
		testExamID, currentTime, currentTime+7200000, testTeacherID).Scan(&testSessionID)
	if err != nil {
		t.Fatalf("创建测试考试场次失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID)

	// 创建考生记录（将testStudentID添加为考生）
	_, err = conn.Exec(ctx, `
		INSERT INTO t_examinee (
			student_id, exam_session_id, creator, create_time, updated_by, update_time, 
			status, addi, serial_number, examinee_number
		) VALUES (
			$1, $2, $3, $4, $3, $4, '00', '{}', 1, '24001000001'
		)`,
		testStudentID, testSessionID, testTeacherID, currentTime)
	if err != nil {
		t.Fatalf("创建考生记录失败: %v", err)
	}

	// 创建另一个考试（不属于testTeacherID）
	var otherExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'其他人的考试', '00', '00', $1, $2, $1, $2, '00'
		) RETURNING id`,
		88888, currentTime).Scan(&otherExamID)
	if err != nil {
		t.Fatalf("创建其他考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, otherExamID)

	tests := []struct {
		name        string
		userID      int64
		examID      int64
		role        int64
		wantResult  bool
		wantError   bool
		errorMsg    string
		description string
		forceError  string
	}{
		{
			name:        "管理员权限-应该有权限",
			userID:      testAdminID,
			examID:      testExamID,
			role:        3, // 管理员角色
			wantResult:  true,
			wantError:   false,
			description: "管理员对所有考试都有权限",
			forceError:  "",
		},
		{
			name:        "教师权限-自己创建的考试,QueryRow 错误",
			userID:      testTeacherID,
			examID:      testExamID,
			role:        2, // 教师角色
			wantResult:  false,
			wantError:   true,
			description: "教师对自己创建的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "教师权限-自己创建的考试",
			userID:      testTeacherID,
			examID:      testExamID,
			role:        2, // 教师角色
			wantResult:  true,
			wantError:   false,
			description: "教师对自己创建的考试有权限",
			forceError:  "",
		},
		{
			name:        "教师权限-不是自己创建的考试",
			userID:      testTeacherID,
			examID:      otherExamID,
			role:        2, // 教师角色
			wantResult:  false,
			wantError:   false,
			description: "教师对不是自己创建的考试没有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1, // 学生角色
			wantResult:  true,
			wantError:   false,
			description: "学生对自己参加的考试有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试 QueryRow 错误",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1, // 学生角色
			wantResult:  false,
			wantError:   true,
			description: "学生对自己参加的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "学生权限-未参加的考试",
			userID:      testStudentID,
			examID:      otherExamID,
			role:        1, // 学生角色
			wantResult:  false,
			wantError:   false,
			description: "学生对未参加的考试没有权限",
			forceError:  "",
		},
		{
			name:        "无效的用户ID",
			userID:      0,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "用户ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数用户ID",
			userID:      -1,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数用户ID应该返回错误",
			forceError:  "",
		},
		{
			name:        "无效的考试ID",
			userID:      testStudentID,
			examID:      0,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "考试ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数考试ID",
			userID:      testStudentID,
			examID:      -1,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数考试ID应该返回错误",
			forceError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// 模拟用户上下文
			ctx = context.WithValue(ctx, "force-error", tt.forceError)
			result, err := validateUserExamPermission(ctx, tt.userID, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("validateUserExamPermission() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateUserExamPermission() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateUserExamPermission() 期望没有错误，但返回错误: %v", err)
					return
				}
			}

			// 检查结果
			if result != tt.wantResult {
				t.Errorf("validateUserExamPermission() = %v, 期望 %v (%s)", result, tt.wantResult, tt.description)
			}
		})
	}
}

func TestGetExamInfo(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的数据ID列表
	var testExamIDs []int64

	// 清理函数
	defer func() {
		for _, examID := range testExamIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败 (exam_id=%d): %v", examID, err)
			}
		}
	}()

	// 创建测试考试
	currentTime := time.Now().UnixMilli()
	var testExamID int64
	err := conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, 
			status, exam_delivery_status
		) VALUES (
			'测试考试信息', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '00', '00'
		) RETURNING id`,
		currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建已删除状态的考试
	var deletedExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, 
			status, exam_delivery_status
		) VALUES (
			'已删除考试', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '12', '00'
		) RETURNING id`,
		currentTime).Scan(&deletedExamID)
	if err != nil {
		t.Fatalf("创建已删除考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, deletedExamID)

	tests := []struct {
		name        string
		examID      int64
		role        int64
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, cmn.TExamInfo, error)
	}{
		{
			name:        "管理员角色-正常获取考试信息",
			examID:      testExamID,
			role:        3, // 管理员
			wantError:   false,
			description: "管理员获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				if !examInfo.Name.Valid || examInfo.Name.String != "测试考试信息" {
					t.Errorf("考试名称不匹配: got %v, want '测试考试信息'", examInfo.Name)
				}
				// 管理员应该能看到完整信息包括creator等字段
				if !examInfo.Creator.Valid {
					t.Errorf("管理员应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教师角色-正常获取考试信息",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   false,
			description: "教师获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教师角色-正常获取考试信息 Scan错误",
			examID:      testExamID,
			role:        2,
			wantError:   true,
			description: "教师获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-只获取部分考试信息",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   false,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
		},
		{
			name:        "学生角色-只获取部分考试信息 Scan错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			role:        3,
			wantError:   true,
			description: "不存在的考试ID应该返回sql.ErrNoRows",
		},
		{
			name:        "已删除的考试",
			examID:      deletedExamID,
			role:        3,
			wantError:   true,
			description: "已删除的考试(status='12')应该查询不到",
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != 1 {
					t.Errorf("Mock考试ID不匹配: got %v, want 1", examInfo.ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "GetExamInfo-error",
			wantError:   true,
			errorMsg:    "GetExamInfo error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testExamID,
			role:        3,
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamInfo(testCtx, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamInfo() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamInfo() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamInfo() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestGetExamSessions(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的数据ID列表
	var testExamIDs []int64
	var testSessionIDs []int64
	var testPaperIDs []int64

	// 清理函数
	defer func() {
		for _, sessionID := range testSessionIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_session WHERE id = $1`, sessionID)
			if err != nil {
				t.Logf("清理考试场次失败: %v", err)
			}
		}
		for _, examID := range testExamIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败: %v", err)
			}
		}
		for _, paperID := range testPaperIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_paper WHERE id = $1`, paperID)
			if err != nil {
				t.Logf("清理试卷记录失败: %v", err)
			}
		}
	}()

	// 创建测试试卷
	currentTime := time.Now().UnixMilli()
	var testPaperID int64
	err := conn.QueryRow(ctx, `
		INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
		VALUES ('测试试卷', '00', 99001, $1, 99001, $1, '00') 
		RETURNING id`, currentTime).Scan(&testPaperID)
	if err != nil {
		t.Fatalf("创建测试试卷失败: %v", err)
	}
	testPaperIDs = append(testPaperIDs, testPaperID)

	// 创建测试考试
	var testExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'测试考试场次', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '00'
		) RETURNING id`,
		currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建测试场次1
	var testSessionID1 int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			name_visibility_in, late_entry_time, early_submission_time,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 1, $2, $3, $4, 120, '00', '00', '00', '02', 
			true, 5, 10, '00', 99001, $3, 99001, $3
		) RETURNING id`,
		testExamID, testPaperID, currentTime, currentTime+7200000).Scan(&testSessionID1)
	if err != nil {
		t.Fatalf("创建测试场次1失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID1)

	// 创建测试场次2
	var testSessionID2 int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			name_visibility_in, late_entry_time, early_submission_time,
			status, creator, create_time, updated_by, update_time, reviewer_ids
		) VALUES (
			$1, 2, $2, $3, $4, 90, '02', '02', '01', '00', 
			false, 0, 0, '00', 99001, $3, 99001, $3, NULL
		) RETURNING id`,
		testExamID, testPaperID, currentTime+86400000, currentTime+86400000+5400000).Scan(&testSessionID2)
	if err != nil {
		t.Fatalf("创建测试场次2失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID2)

	// 创建已删除的场次
	var deletedSessionID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 3, $2, $3, $4, 60, '00', '00', '00', '00', 
			'14', 99001, $3, 99001, $3
		) RETURNING id`,
		testExamID, testPaperID, currentTime, currentTime+3600000).Scan(&deletedSessionID)
	if err != nil {
		t.Fatalf("创建已删除场次失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, deletedSessionID)

	tests := []struct {
		name        string
		examID      int64
		role        int64
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, []cmn.TExamSession, error)
	}{
		{
			name:        "教师角色-获取完整场次信息",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   false,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 教师也应该能看到完整信息
				session1 := sessions[0]
				if !session1.PaperName.Valid {
					t.Errorf("教师应该能看到试卷名称")
				}
			},
			forceError: "",
		},
		{
			name:        "教师角色-获取完整场次信息 Query错误",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   true,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "教师角色-获取完整场次信息 Scan错误",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   true,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-获取基本场次信息",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   false,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 学生不应该看到试卷名称等敏感信息
				session1 := sessions[0]
				if session1.PaperName.Valid {
					t.Errorf("学生不应该看到试卷名称，但获取到了: %v", session1.PaperName)
				}
				// 但应该能看到基本信息
				if !session1.StartTime.Valid {
					t.Errorf("学生应该能看到开始时间")
				}
			},
			forceError: "",
		},
		{
			name:        "学生角色-获取基本场次信息 Query错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "学生角色-获取基本场次信息 Scan错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			role:        3,
			wantError:   false,
			description: "不存在的考试ID应该返回空数组",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 0 {
					t.Errorf("不存在的考试应该返回空数组: got %d sessions", len(sessions))
				}
			},
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 1 {
					t.Errorf("Mock应该返回1个场次: got %d", len(sessions))
					return
				}
				if !sessions[0].ID.Valid || sessions[0].ID.Int64 != 10001 {
					t.Errorf("Mock场次ID不匹配: got %v, want 10001", sessions[0].ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "GetExamSessions-error",
			wantError:   true,
			errorMsg:    "GetExamSessions error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testExamID,
			role:        3,
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamSessions(testCtx, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamSessions() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamSessions() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamSessions() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestUpdateExamStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	// 创建基础上下文
	ctx := context.Background()

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer tx.Rollback(ctx)

	// 准备测试数据 - 插入一个测试考试
	testExamID := int64(999001)
	testUserID := int64(1)
	testExamName := "测试考试_updateStatus"

	// 插入测试考试数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, '00', $3, $4, $3, $4)
	`, testExamID, testExamName, testUserID, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	tests := []struct {
		name         string
		examID       int64
		newStatus    string
		userID       int64
		forceError   string
		wantError    bool
		errorMsg     string
		shouldVerify bool
	}{
		{
			name:         "正常更新考试状态-草稿到发布",
			examID:       testExamID,
			newStatus:    "01",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "正常更新考试状态-发布到进行中",
			examID:       testExamID,
			newStatus:    "02",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:      "无效的考试ID-零值",
			examID:    0,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的考试ID-负值",
			examID:    -1,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的用户ID-零值",
			examID:    testExamID,
			newStatus: "01",
			userID:    0,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "无效的用户ID-负值",
			examID:    testExamID,
			newStatus: "01",
			userID:    -1,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "空的状态值",
			examID:    testExamID,
			newStatus: "",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "更新状态不能为空",
		},
		{
			name:       "数据库执行错误",
			examID:     testExamID,
			newStatus:  "01",
			userID:     testUserID,
			forceError: "tx.Exec",
			wantError:  true,
			errorMsg:   "force error",
		},
		{
			name:      "不存在的考试ID",
			examID:    999999,
			newStatus: "01",
			userID:    testUserID,
			wantError: false, // SQL执行成功但影响行数为0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前的状态
			var originalStatus string
			if tt.shouldVerify && !tt.wantError {
				err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", tt.examID).Scan(&originalStatus)
				if err != nil {
					t.Fatalf("获取更新前状态失败: %v", err)
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamStatus(testCtx, tx, tt.examID, tt.newStatus, tt.userID)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify {
					var currentStatus string
					err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", tt.examID).Scan(&currentStatus)
					if err != nil {
						t.Errorf("验证更新结果失败: %v", err)
						return
					}

					if currentStatus != tt.newStatus {
						t.Errorf("updateExamStatus() 状态更新失败，期望状态 = %v, 实际状态 = %v", tt.newStatus, currentStatus)
					}

					// 验证 update_time 和 updated_by 字段
					var updatedBy int64
					var updateTime int64
					err = tx.QueryRow(ctx, "SELECT updated_by, update_time FROM t_exam_info WHERE id = $1", tt.examID).Scan(&updatedBy, &updateTime)
					if err != nil {
						t.Errorf("验证更新字段失败: %v", err)
					} else {
						if updatedBy != tt.userID {
							t.Errorf("updateExamStatus() updated_by 字段错误，期望 = %v, 实际 = %v", tt.userID, updatedBy)
						}
						// 验证 update_time 是最近更新的（容忍1分钟误差）
						if time.Since(time.UnixMilli(updateTime)) > time.Minute {
							t.Errorf("updateExamStatus() update_time 字段未正确更新，时间 = %v", updateTime)
						}
					}
				}
			}
		})
	}
}

// TestUpdateExamSessionStatus 测试 updateExamSessionStatus 函数
func TestUpdateExamSessionStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	ctx := context.Background()

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer tx.Rollback(ctx)

	// 准备测试数据 - 先插入考试信息，再插入考试场次
	testExamID := int64(999002)
	testSessionID := int64(999201)
	testUserID := int64(1)
	testExamName := "测试考试_updateSessionStatus"

	// 插入测试考试数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, '01', $3, $4, $3, $4)
	`, testExamID, testExamName, testUserID, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入测试考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, mark_method, session_num, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, 1, 1, '00', '00', $3, $4, $3, $4)
	`, testSessionID, testExamID, testUserID, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("插入测试考试场次数据失败: %v", err)
	}

	tests := []struct {
		name          string
		examSessionID int64
		newStatus     string
		userID        int64
		forceError    string
		wantError     bool
		errorMsg      string
		shouldVerify  bool
	}{
		{
			name:          "正常更新考试场次状态-待开始到进行中",
			examSessionID: testSessionID,
			newStatus:     "01",
			userID:        testUserID,
			forceError:    "",
			wantError:     false,
			shouldVerify:  true,
		},
		{
			name:          "正常更新考试场次状态-进行中到已结束",
			examSessionID: testSessionID,
			newStatus:     "02",
			userID:        testUserID,
			forceError:    "",
			wantError:     false,
			shouldVerify:  true,
		},
		{
			name:          "无效的考试场次ID-零值",
			examSessionID: 0,
			newStatus:     "01",
			userID:        testUserID,
			wantError:     true,
			errorMsg:      "无效的考试场次ID",
		},
		{
			name:          "无效的考试场次ID-负值",
			examSessionID: -1,
			newStatus:     "01",
			userID:        testUserID,
			wantError:     true,
			errorMsg:      "无效的考试场次ID",
		},
		{
			name:          "无效的用户ID-零值",
			examSessionID: testSessionID,
			newStatus:     "01",
			userID:        0,
			wantError:     true,
			errorMsg:      "无效的用户ID",
		},
		{
			name:          "无效的用户ID-负值",
			examSessionID: testSessionID,
			newStatus:     "01",
			userID:        -1,
			wantError:     true,
			errorMsg:      "无效的用户ID",
		},
		{
			name:          "空的状态值",
			examSessionID: testSessionID,
			newStatus:     "",
			userID:        testUserID,
			wantError:     true,
			errorMsg:      "更新状态不能为空",
		},
		{
			name:          "数据库执行错误",
			examSessionID: testSessionID,
			newStatus:     "01",
			userID:        testUserID,
			forceError:    "tx.Exec",
			wantError:     true,
			errorMsg:      "force error",
		},
		{
			name:          "不存在的考试场次ID",
			examSessionID: 999999,
			newStatus:     "01",
			userID:        testUserID,
			wantError:     false, // SQL执行成功但影响行数为0
		},
		{
			name:          "更新为暂停状态",
			examSessionID: testSessionID,
			newStatus:     "03",
			userID:        testUserID,
			wantError:     false,
			shouldVerify:  true,
		},
		{
			name:          "更新为取消状态",
			examSessionID: testSessionID,
			newStatus:     "04",
			userID:        testUserID,
			wantError:     false,
			shouldVerify:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前的状态
			var originalStatus string
			if tt.shouldVerify && !tt.wantError {
				err := tx.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", tt.examSessionID).Scan(&originalStatus)
				if err != nil {
					t.Fatalf("获取更新前状态失败: %v", err)
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamSessionStatus(testCtx, tx, tt.examSessionID, tt.newStatus, tt.userID)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamSessionStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamSessionStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamSessionStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify {
					var currentStatus string
					err := tx.QueryRow(ctx, "SELECT status FROM t_exam_session WHERE id = $1", tt.examSessionID).Scan(&currentStatus)
					if err != nil {
						t.Errorf("验证更新结果失败: %v", err)
						return
					}

					if currentStatus != tt.newStatus {
						t.Errorf("updateExamSessionStatus() 状态更新失败，期望状态 = %v, 实际状态 = %v", tt.newStatus, currentStatus)
					}

					// 验证 update_time 和 updated_by 字段
					var updatedBy int64
					var updateTime int64
					err = tx.QueryRow(ctx, "SELECT updated_by, update_time FROM t_exam_session WHERE id = $1", tt.examSessionID).Scan(&updatedBy, &updateTime)
					if err != nil {
						t.Errorf("验证更新字段失败: %v", err)
					} else {
						if updatedBy != tt.userID {
							t.Errorf("updateExamSessionStatus() updated_by 字段错误，期望 = %v, 实际 = %v", tt.userID, updatedBy)
						}
						// 验证 update_time 是最近更新的（容忍1分钟误差）
						if time.Since(time.UnixMilli(updateTime)) > time.Minute {
							t.Errorf("updateExamSessionStatus() update_time 字段未正确更新，时间 = %v", updateTime)
						}
					}
				}
			}
		})
	}
}
