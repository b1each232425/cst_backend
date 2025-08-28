package invigilation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/exam_mgt"
)

var (
	testPaperID                     = int64(99901)
	testPaperToPublishID            = int64(99902) // 用于测试发布考试的试卷
	testNormalExamID                = int64(99901)
	testDeleteExamID                = int64(99902)
	testNormalExamID2               = int64(99903)
	testExamToPublishID             = int64(99904) // 用于测试考试发布
	testErrorExamToPublishID1       = int64(99905) // 用于测试考试发布错误 - 时间不符合要求
	testEndExamID                   = int64(99906) // 已结束的考试
	testPublishedExamID             = int64(99907) // 已发布的考试
	testExamSessionID1              = int64(99901)
	testExamSessionID2              = int64(99902)
	testDeleteExamSessionID         = int64(99903)
	testExamSessionID3              = int64(99904)
	testExamSessionToPublishID1     = int64(99905) // 用于测试考试发布
	testExamSessionToPublishID2     = int64(99906) // 用于测试考试发布
	testExamSessionToPublishID3     = int64(99907) // 用于测试考试发布
	testExamSessionToPublishID4     = int64(99908) // 用于测试考试发布
	testExamSessionToPublishID5     = int64(99909) // 用于测试考试发布
	testErrorExamSessionToPublishID = int64(99910) // 用于测试考试发布错误 - 时间不符合要求
	testPublishedExamSessionID      = int64(99911) // 已发布的考试场次
	testExamPaperID                 = int64(99912) // 测试用的考试试卷
	testOfflineExamID               = int64(99913) // 用于测试线下考试
	testOfflineExamSessionID        = int64(99912)

	testAcademicAffair                       = int64(99901)
	testStudent1                             = int64(99902)
	testGrader                               = int64(99903) // 用于考试批阅员
	testExamSession1StartTime                = time.Now().Add(10 * time.Minute).UnixMilli()
	testExamSession1EndTime                  = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSession2StartTime                = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSession2EndTime                  = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionStartTime           = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionEndTime             = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID1StartTime     = time.Now().Add(10 * time.Minute).UnixMilli()
	testExamSessionToPublishID1EndTime       = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2StartTime     = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2EndTime       = time.Now().Add(30 * time.Minute).UnixMilli()
	testExamSessionToPublishID3StartTime     = time.Now().Add(30 * time.Minute).UnixMilli()
	testExamSessionToPublishID3EndTime       = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4StartTime     = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4EndTime       = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5StartTime     = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5EndTime       = time.Now().Add(60 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDStartTime = time.Now().Add(-10 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDEndTime   = time.Now().UnixMilli()
	BankQuestionIDs                          = []int64{10000001, 10000002, 10000003, 10000004, 10000005}
	testUpdateTime                           = time.Now().UnixMilli()

	testFile1ID       = int64(99901)
	testFile2ID       = int64(99902)
	testFile1CheckSum = "bc8e94630e020929"
	testFile2CheckSum = "94195fd3746f1460"
	testFile1Name     = "testFile1.txt"
	testFile2Name     = "testFile2.txt"
	testFile1Content  = "This is the content of testFile1."
	testFile2Content  = "This is the content of testFile2."

	testFileID3          = int64(99903)
	testSameFileID       = int64(99904)
	testFile3CheckSum    = "2bbd5436cae65e1e"
	testSameFileCheckSum = "bc8e94630e020929"
	testFile3Name        = "testFile3.txt"
	testSameFileName     = "testFile4.txt"
	testFile3Content     = "This is the content of testFile3."
	testSameFileContent  = "This is the content of testFile1."

	notExistsFileID      = int64(99905)
	notExistFileName     = "notExistsFile.txt"
	notExistFileCheckSum = "notExists"

	testExamSiteID        = int64(99901)
	testExamRoomID        = int64(99901)
	testExamRoomCapacity  = int64(30)
	testExamRoomID2       = int64(99902)
	testExamRoomCapacity2 = int64(1)
)

// 生成文件的 XXHash64 校验和
func FastDigest(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建 XXHash64 hasher
	hasher := xxhash.New()

	// 使用缓冲读取，提高性能
	reader := bufio.NewReader(file)
	buffer := make([]byte, 4*1024*1024) // 4MB 块大小，与前端保持一致

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			hasher.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	// 获取最终哈希值并转换为十六进制字符串
	hashSum := hasher.Sum64()
	return fmt.Sprintf("%016x", hashSum), nil
}

// 计算内容的 XXHash64 校验和
func calculateContentCheckSum(content []byte) string {
	hasher := xxhash.New()
	hasher.Write(content)
	hashSum := hasher.Sum64()
	return fmt.Sprintf("%016x", hashSum)
}

// createMockContextWithRole 创建带用户角色的模拟上下文
func createMockContextWithRole(method, path string, queryParams url.Values, forceError string, userID, userRole int64) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2002),
		Domain: "cst.school.academicAffair^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2003),
		Domain: "cst.school^teacher",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2008),
		Domain: "cst.school^student",
	})

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
			ID:   null.NewInt(userID, true),
			Role: null.NewInt(userRole, true),
		},
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}

	return ctx
}

func createMockContextWithBody(method, path string, data string, forceError string, userID int64, userRole int64) context.Context {
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

	// Domains
	domains := make([]cmn.TDomain, 0)

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2001),
		Domain: "cst.school^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2002),
		Domain: "cst.school.academicAffair^admin",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2003),
		Domain: "cst.school^teacher",
	})

	domains = append(domains, cmn.TDomain{
		ID:     null.IntFrom(2008),
		Domain: "cst.school^student",
	})

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
			ID:   null.NewInt(userID, true),   // 请求用户ID
			Role: null.NewInt(userRole, true), // 用户角色ID
		},
		Domains:     domains,
		RedisClient: cmn.GetRedisConn(),
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 辅助函数：JSON序列化
func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}
	return data
}

// 辅助函数：生成指定长度的字符串
func generateLongString(length int) string {
	if length <= 0 {
		return ""
	}

	// 使用中文字符测试 rune 计算
	baseString := "这是一个测试字符串用于验证长度限制功能"
	baseLength := len([]rune(baseString))

	if length <= baseLength {
		return string([]rune(baseString)[:length])
	}

	// 如果需要更长的字符串，重复基础字符串
	var result strings.Builder
	result.Grow(length * 3) // 预分配空间，考虑中文字符的字节数

	for result.Len() < length*3 { // 估算字节数
		result.WriteString(baseString)
	}

	// 截取到确切的字符数
	resultRunes := []rune(result.String())
	if len(resultRunes) > length {
		resultRunes = resultRunes[:length]
	}

	return string(resultRunes)
}

// 生成一张测试试卷
func CreateTestPaperWithGroupsAndQuestions(ctx context.Context, bankQuestionIDs []int64, testUserID int64) (groupIDs []int64, questionIDs []int64, err error) {
	now := time.Now().UnixMilli()

	paperID := testPaperToPublishID
	conn := cmn.GetPgxConn()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("开始事务失败: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
		} else {
			if err != nil {
				tx.Rollback(ctx)
			} else {
				err = tx.Commit(ctx)
			}
		}
	}()

	// 创建试卷
	paper := &cmn.TPaper{
		Name:              null.StringFrom("Test Paper"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("00"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("Test Description"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           null.IntFrom(testUserID),
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         null.IntFrom(testUserID),
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
	}

	//初始化一张空试卷
	err = tx.QueryRow(ctx, `
		INSERT INTO t_paper 
			(id, name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, domain_id) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
		RETURNING id`,
		testPaperToPublishID,
		paper.Name.String,
		paper.AssemblyType.String,
		paper.Category.String,
		paper.Level.String,
		paper.SuggestedDuration.Int64,
		paper.Tags,
		paper.Creator.Int64,
		paper.CreateTime.Int64,
		paper.UpdatedBy.Int64,
		paper.UpdateTime.Int64,
		paper.Status.String,
		2000,
	).Scan(&paperID)

	if err != nil {
		return nil, nil, fmt.Errorf("创建试卷失败: %v", err)
	}

	// 定义题组
	groupNames := []string{"Group A", "Group B"}
	groupIDMap := make(map[string]int64)

	// 创建题组
	for i, name := range groupNames {
		var groupID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO t_paper_group 
				(paper_id, name, "order", creator, create_time, updated_by, update_time, status) 
			VALUES 
				($1, $2, $3, $4, $5, $4, $5, $6) 
			RETURNING id`,
			paperID,
			name,
			i+1,
			testUserID,
			now,
			"00",
		).Scan(&groupID)

		if err != nil {
			return nil, nil, fmt.Errorf("创建题组失败: %v", err)
		}

		groupIDs = append(groupIDs, groupID)
		groupIDMap[fmt.Sprintf("g%d", i)] = groupID
	}

	// 为每个题组添加题目
	if len(bankQuestionIDs) > 0 {
		for _, groupID := range groupIDs {
			for j, bankQuestionID := range bankQuestionIDs {
				if j >= 2 { // 每个题组最多添加2道题
					break
				}
				var questionID int64
				err = tx.QueryRow(ctx, `
					INSERT INTO t_paper_question 
						(bank_question_id, group_id, "order", score, creator, create_time, updated_by, update_time, status, sub_score) 
					VALUES 
						($1, $2, $3, $4, $5, $6, $5, $6, $7, $8) 
					RETURNING id`,
					bankQuestionID,
					groupID,
					j+1,
					6.0, // 默认分数
					testUserID,
					now,
					"00",
					types.JSONText(`[1,2,3]`),
				).Scan(&questionID)

				if err != nil {
					return groupIDs, nil, fmt.Errorf("创建试题失败: %v", err)
				}
				questionIDs = append(questionIDs, questionID)
			}
		}
	}

	return groupIDs, questionIDs, nil
}

func CreateTestExamData(t *testing.T) {

	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 插入测试教务员数据
	_, err := conn.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role, status) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, testAcademicAffair, "sys^admin", "测试用户", "test_user", 2002, "00")
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 插入测试批阅员数据
	_, err = conn.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role, status) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, testGrader, "sys^admin", "测试批阅员", "test_grader", 2005, "00")
	if err != nil {
		t.Fatalf("创建测试批阅员失败: %v", err)
	}

	// 插入测试学生数据
	_, err = conn.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role,status) 
		VALUES ($1, $2, $3, $4, $5,$6)
		ON CONFLICT (id) DO NOTHING`, testStudent1, "sys^student", "测试学生", "test_student", 2008, "00")
	if err != nil {
		t.Fatalf("创建测试学生失败: %v", err)
	}

	// 创建题库数据

	questions := []struct {
		id         int64
		qtype      string
		difficulty string
		creator    int64
		status     string
	}{
		{BankQuestionIDs[0], "00", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[1], "02", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[2], "04", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[3], "06", "1", testAcademicAffair, "00"},
		{BankQuestionIDs[4], "08", "1", testAcademicAffair, "00"},
	}
	for _, q := range questions {
		_, err = conn.Exec(ctx, `
			INSERT INTO assessuser.t_question (id, type, difficulty, creator,create_time,updated_by,update_time,status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, q.id, q.qtype, q.difficulty, q.creator, time.Now().UnixMilli(), q.creator, time.Now().UnixMilli(), q.status)
		if err != nil {
			t.Fatalf("插入测试题目数据失败: %v", err)
		}
	}

	// 创建用于测试发布的试卷
	_, _, err = CreateTestPaperWithGroupsAndQuestions(ctx, BankQuestionIDs, testAcademicAffair)
	if err != nil {
		t.Fatalf("创建试卷失败: %v", err)
	}

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				err = tx.Commit(ctx)
			}
		}
	}()

	// 生成考卷
	var examPaperID *int64
	examPaperID, err = examPaper.GenerateExamPaper(ctx, tx, testPaperToPublishID, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("创建考卷失败: %v", err)
	}

	testExamPaperID = *examPaperID

	_, err = tx.Exec(ctx, `
	UPDATE t_paper SET exampaper_id = $1 WHERE id = $2
	`, examPaperID, testPaperToPublishID)

	// 创建测试试卷
	_, err = tx.Exec(ctx, `
		INSERT INTO t_paper (id, name, category, creator, status, domain_id) 
		VALUES ($1, '测试试卷', '00', $2, '00', 2000) `, testPaperID, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("创建测试试卷失败: %v", err)
	}

	// 插入考试信息
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time, domain_id, files, exam_room_invigilator_count)
		VALUES ($1, '测试正常考试', '00', '00', '00', $2, $3, $2, $11, $4, '[99903]', '[]'), 
		($5, '测试已删除的考试', '00', '00', '12', $2, $3, $2, $11, $4, '[]', '[]'),
		($6, '测试正常考试2', '00', '00', '02', $2, $3, $2, $11, $4, '[]', '[]'),
		($7, '测试发布考试', '00', '00', '00', $2, $3, $2, $11, $4, '[]', '[]'),
		($8, '测试发布错误考试', '00', '00', '00', $2, $3, $11, $3, $4, '[]', '[]'),
		($9, '测试已结束的考试', '00', '00', '06', $2, $3, $11, $3, $4, '[]', '[]'),
		($10, '测试已发布的考试', '00', '00', '02', $2, $3, $11, $3, $4, '[]','[]')
	`, testNormalExamID, testAcademicAffair, time.Now().UnixMilli(), 2002, testDeleteExamID,
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID, testUpdateTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入线下考试信息
	testExamRoomInvigilatorCount := []exam_mgt.ExamRoomConfig{
		exam_mgt.ExamRoomConfig{
			RoomID:           1,
			Capacity:         30,
			InvigilatorCount: 1,
		},
	}

	testExamRoomInvigilatorCountBytes, _ := json.Marshal(testExamRoomInvigilatorCount)

	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time, domain_id, files, exam_room_invigilator_count)
		VALUES ($1, '测试正常考试', '00', '02', '00', $2, $3, $2, $3, $4, '[99903]', $5)
	`, testOfflineExamID, testAcademicAffair, time.Now().UnixMilli(), 2002, testExamRoomInvigilatorCountBytes)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	var reviewerIDs []int64
	if testGrader > 0 {
		reviewerIDs = []int64{testGrader}
	}

	var nilReviewerIDs []int64
	nilReviewerIDs = make([]int64, 0)

	// 插入考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $19, '00', '00', 1, '02', $4, $5, $4, $21, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $19, '00', '00', 2, '02', $4, $5, $4, $21, $9, $10, '00', 10, '00'), 
		($11, $12, $3, $19, '00', '00', 3, '12', $3, $4, $3, $21, $13, $14, '00', 10, '00'),
		($15, $16, $3, $20, '00', '00', 4, '02', $3, $4, $3, $21, $17, $18, '00', 10, '00')
	`, testExamSessionID1, testNormalExamID, testPaperID, testAcademicAffair, time.Now().UnixMilli(),
		testExamSession1StartTime, testExamSession1EndTime, testExamSessionID2, testExamSession2StartTime, testExamSession2EndTime,
		testDeleteExamSessionID, testDeleteExamID, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime,
		testExamSessionID3, testNormalExamID2, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime, reviewerIDs, nilReviewerIDs, testUpdateTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试场次数据失败: %v", err)
	}

	// 插入要发布的考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $23, '00', '00', 1, '00', $4, $5, $4, $28, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $23, '00', '00', 2, '00', $4, $5, $4, $28, $9, $10, '00', 10, '02'),
		($11, $2, $3, $23, '00', '00', 3, '00', $4, $5, $4, $28, $12, $13, '00', 10, '04'),
		($14, $2, $3, $24, '00', '00', 4, '00', $4, $5, $4, $28, $15, $16, '00', 10, '06'),
		($17, $2, $3, $23, '00', '00', 5, '00', $4, $5, $4, $28, $18, $19, '00', 10, '08'),
		($20, $25, $3, $23, '00', '00', 6, '00', $4, $5, $4, $28, $21, $22, '00', 10, '10'),
		($26, $27, $3, $23, '00', '00', 7, '02', $4, $5, $4, $28, $6, $7, '00', 10, '12')
	`, testExamSessionToPublishID1, testExamToPublishID, testPaperToPublishID, testAcademicAffair, time.Now().UnixMilli(), testExamSessionToPublishID1StartTime, testExamSessionToPublishID1EndTime,
		testExamSessionToPublishID2, testExamSessionToPublishID2StartTime, testExamSessionToPublishID2EndTime,
		testExamSessionToPublishID3, testExamSessionToPublishID3StartTime, testExamSessionToPublishID3EndTime,
		testExamSessionToPublishID4, testExamSessionToPublishID4StartTime, testExamSessionToPublishID4EndTime,
		testExamSessionToPublishID5, testExamSessionToPublishID5StartTime, testExamSessionToPublishID5EndTime,
		testErrorExamSessionToPublishID, testErrorExamSessionToPublishIDStartTime, testErrorExamSessionToPublishIDEndTime,
		reviewerIDs, nilReviewerIDs, testErrorExamToPublishID1, testPublishedExamSessionID, testPublishedExamID, testUpdateTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试发布考试场次数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $4, '00', '00', 1, '02', $5, $6, $5, $6, $7, $8, '00', 10, '00')
	`, testOfflineExamSessionID, testOfflineExamID, testPaperToPublishID, nilReviewerIDs, testAcademicAffair, time.Now().UnixMilli(), testExamSessionToPublishID1StartTime, testExamSessionToPublishID1EndTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试离线考试场次数据失败: %v", err)
	}

	// 插入考生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_examinee (exam_session_id, student_id, serial_number, status, creator, create_time)
		VALUES ($1, $2, 1, '00', $3, $4), 
		($5, $2, 2, '00', $3, $4),
		($6, $2, 3, '00', $3, $4),
		($7, $2, 1, '00', $3, $4),
		($8, $2, 1, '00', $3, $4),
		($9, $2, 1, '00', $3, $4),
		($10, $2, 1, '00', $3, $4)
	`, testExamSessionID1, testStudent1, testAcademicAffair, time.Now().UnixMilli(),
		testExamSessionID2, testExamSessionID3, testExamSessionToPublishID1, testExamSessionToPublishID2,
		testExamSessionToPublishID3, testExamSessionToPublishID4)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考生数据失败: %v", err)
	}

	// 插入考点和考场数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_site (
			id, creator
		)VALUES ($1, $2)
	`, testExamSiteID, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考点数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_room (
			id, creator, exam_site, capacity
		)VALUES ($1, $2, $3, $4)
	`, testExamRoomID, testAcademicAffair, testExamSiteID, testExamRoomCapacity)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考场数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_room (
			id, creator, exam_site, capacity
		)VALUES ($1, $2, $3, $4)
	`, testExamRoomID2, testAcademicAffair, testExamSiteID, testExamRoomCapacity2)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考场数据失败: %v", err)
	}

	// 插入测试监考数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_invigilation (
		exam_session_id, exam_room, invigilator, creator, create_time,
		updated_by, update_time, addi
		)VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, testOfflineExamSessionID, testExamRoomID, testAcademicAffair, testAcademicAffair, time.Now().UnixMilli(), testAcademicAffair, time.Now().UnixMilli(), "{}")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试监考数据失败: %v", err)
	}

	// 创建文件
	uploadDir := "./uploads"

	// 确保上传目录存在
	err = os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatalf("创建上传目录失败: %v", err)
	}

	// 计算文件内容的校验和
	testFile1CheckSum = calculateContentCheckSum([]byte(testFile1Content))
	testFile2CheckSum = calculateContentCheckSum([]byte(testFile2Content))
	testFile3CheckSum = calculateContentCheckSum([]byte(testFile3Content))
	testSameFileCheckSum = calculateContentCheckSum([]byte(testSameFileContent))

	// 以校验和作为文件名创建文件
	testFile1Path := filepath.Join(uploadDir, testFile1CheckSum)
	err = os.WriteFile(testFile1Path, []byte(testFile1Content), 0644)
	if err != nil {
		t.Fatalf("创建测试文件1失败: %v", err)
	}

	testFile2Path := filepath.Join(uploadDir, testFile2CheckSum)
	err = os.WriteFile(testFile2Path, []byte(testFile2Content), 0644)
	if err != nil {
		t.Fatalf("创建测试文件2失败: %v", err)
	}

	testFile3Path := filepath.Join(uploadDir, testFile3CheckSum)
	err = os.WriteFile(testFile3Path, []byte(testFile3Content), 0644)
	if err != nil {
		t.Fatalf("创建测试文件3失败: %v", err)
	}

	testSameFilePath := filepath.Join(uploadDir, testSameFileCheckSum)
	err = os.WriteFile(testSameFilePath, []byte(testSameFileContent), 0644)
	if err != nil {
		t.Fatalf("创建测试文件4失败: %v", err)
	}

	// 创建对应的.info文件
	testFile1InfoPath := testFile1Path + ".info"
	err = os.WriteFile(testFile1InfoPath, []byte(fmt.Sprintf(`{"name":"%s","size":%d}`, testFile1Name, len(testFile1Content))), 0644)
	if err != nil {
		t.Fatalf("创建测试文件1信息文件失败: %v", err)
	}

	testFile2InfoPath := testFile2Path + ".info"
	err = os.WriteFile(testFile2InfoPath, []byte(fmt.Sprintf(`{"name":"%s","size":%d}`, testFile2Name, len(testFile2Content))), 0644)
	if err != nil {
		t.Fatalf("创建测试文件2信息文件失败: %v", err)
	}

	testFile3InfoPath := testFile3Path + ".info"
	err = os.WriteFile(testFile3InfoPath, []byte(fmt.Sprintf(`{"name":"%s","size":%d}`, testFile3Name, len(testFile3Content))), 0644)
	if err != nil {
		t.Fatalf("创建测试文件3信息文件失败: %v", err)
	}

	testSameFileInfoPath := testSameFilePath + ".info"
	err = os.WriteFile(testSameFileInfoPath, []byte(fmt.Sprintf(`{"name":"%s","size":%d}`, testSameFileName, len(testSameFileContent))), 0644)
	if err != nil {
		t.Fatalf("创建测试文件4信息文件失败: %v", err)
	}

	// 插入文件记录到数据库
	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, testFile1ID, testFile1CheckSum, testFile1Name, testFile1Path, "/test/files", len(testFile1Content), testAcademicAffair, time.Now().UnixMilli()).Scan(&testFile1ID)
	if err != nil {
		t.Fatalf("插入测试文件1记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 2, $7, $8, 2000, '0')
        RETURNING id
    `, testFile2ID, testFile2CheckSum, testFile2Name, testFile2Path, "/test/files", len(testFile2Content), testAcademicAffair, time.Now().UnixMilli()).Scan(&testFile2ID)
	if err != nil {
		t.Fatalf("插入测试文件2记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, testFileID3, testFile3CheckSum, testFile3Name, testFile3Path, "/test/files", len(testFile3Content), testAcademicAffair, time.Now().UnixMilli()).Scan(&testFileID3)
	if err != nil {
		t.Fatalf("插入测试文件3记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, testSameFileID, testSameFileCheckSum, testSameFileName, testSameFilePath, "/test/files", len(testSameFileContent), testAcademicAffair, time.Now().UnixMilli()).Scan(&testSameFileID)
	if err != nil {
		t.Fatalf("插入测试文件4记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, notExistsFileID, notExistFileCheckSum, notExistFileName, "/test/files", "/test/files", len(testSameFileContent), testAcademicAffair, time.Now().UnixMilli()).Scan(&notExistsFileID)
	if err != nil {
		t.Fatalf("插入测试文件4记录失败: %v", err)
	}

	return
}

func CleanTestExamData(t *testing.T) {
	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				tx.Commit(ctx)
			}
		}
	}()

	// 删除考点和考场数据
	_, err = tx.Exec(ctx, `DELETE FROM t_exam_record WHERE creator = $1`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试记录数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM t_invigilation WHERE creator = $1`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试监考数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_room WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考场数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_site WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考点数据失败: %v", err)
	}

	// 删除批改相关数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_mark_info WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试批改数据失败: %v", err)
	}

	// 删除答卷相关数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_student_answers WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试答卷数据失败: %v", err)
	}

	// 删除生成的考卷数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper_question WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷题目数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper_group WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷题组数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考卷数据失败: %v", err)
	}

	// 删除试卷1数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper WHERE id = $1
	`, testPaperID)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷数据失败: %v", err)
	}

	// 删除试卷数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper_question WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷题目数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper_group WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷题组数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM t_paper WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试试卷数据失败: %v", err)
	}

	// 删除题库数据
	_, err = tx.Exec(ctx, `
		DELETE FROM assessuser.t_question WHERE creator = $1
	`, testAcademicAffair)

	// 删除测试考生数据
	var testSessionIDs []int64
	testSessionIDs = append(testSessionIDs, testExamSessionID1, testExamSessionID2,
		testDeleteExamSessionID, testExamSessionID3, testExamSessionToPublishID1,
		testExamSessionToPublishID2, testExamSessionToPublishID3, testExamSessionToPublishID4, testExamSessionToPublishID5,
		testErrorExamSessionToPublishID, testPublishedExamSessionID)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_examinee WHERE exam_session_id = ANY($1)
	`, testSessionIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考生数据失败: %v", err)
	}

	// 删除测试考生数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_examinee WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考生数据失败: %v", err)
	}

	// 删除测试考试场次数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_session WHERE id = ANY($1)
	`, testSessionIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试场次数据失败: %v", err)
	}

	// 删除测试考试场次数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_session WHERE creator = $1
	`, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试场次数据失败: %v", err)
	}

	// 删除测试考试信息
	var testExamIDs []int64
	testExamIDs = append(testExamIDs, testNormalExamID, testDeleteExamID,
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID, testOfflineExamID)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_info WHERE id = ANY($1)
	`, testExamIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试考试信息失败: %v", err)
	}

	// 删除测试用户数据
	var testUserIDs []int64
	testUserIDs = append(testUserIDs, testAcademicAffair, testStudent1, testGrader)
	_, err = tx.Exec(ctx, `
		DELETE FROM t_user WHERE id = ANY($1)
	`, testUserIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("删除测试用户数据失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
        DELETE FROM t_file WHERE id = ANY($1)
    `, []int64{testFile1ID, testFile2ID, testFileID3, testSameFileID, notExistsFileID})
	if err != nil {
		t.Logf("删除测试文件记录失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
        DELETE FROM t_file WHERE creator = $1
    `, testAcademicAffair)
	if err != nil {
		t.Logf("删除测试文件记录失败: %v", err)
	}

	// 删除实际的测试文件
	uploadDir := "./uploads"

	// 计算校验和
	file1CheckSum := calculateContentCheckSum([]byte(testFile1Content))
	file2CheckSum := calculateContentCheckSum([]byte(testFile2Content))
	file3CheckSum := calculateContentCheckSum([]byte(testFile3Content))
	sameFileCheckSum := calculateContentCheckSum([]byte(testSameFileContent))

	// 以校验和命名的文件路径
	testFile1Path := filepath.Join(uploadDir, file1CheckSum)
	testFile2Path := filepath.Join(uploadDir, file2CheckSum)
	testFile3Path := filepath.Join(uploadDir, file3CheckSum)
	testSameFilePath := filepath.Join(uploadDir, sameFileCheckSum)

	if err := os.Remove(testFile1Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件1失败: %v", err)
	}

	if err := os.Remove(testFile2Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件2失败: %v", err)
	}

	if err := os.Remove(testFile3Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件3失败: %v", err)
	}

	if err := os.Remove(testSameFilePath); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件4失败: %v", err)
	}

	// 删除对应的.info文件（如果存在）
	infoFile1Path := testFile1Path + ".info"
	infoFile2Path := testFile2Path + ".info"
	infoFile3Path := testFile3Path + ".info"
	infoSameFilePath := testSameFilePath + ".info"

	if err := os.Remove(infoFile1Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件1信息文件失败: %v", err)
	}

	if err := os.Remove(infoFile2Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件2信息文件失败: %v", err)
	}

	if err := os.Remove(infoFile3Path); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件3信息文件失败: %v", err)
	}

	if err := os.Remove(infoSameFilePath); err != nil && !os.IsNotExist(err) {
		t.Logf("删除测试文件4信息文件失败: %v", err)
	}

}

func Test_invigilationList(t *testing.T) {
	cmn.ConfigureForTest()
	CleanTestExamData(t)
	CreateTestExamData(t)
	t.Cleanup(func() {
		CleanTestExamData(t)
	})

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name          string
		forceError    string
		userID        int64
		userRole      int64
		queryParams   url.Values
		wantErr       bool
		checkResult   func(t *testing.T, svc *cmn.ServiceCtx)
		errorContains string
		method        string
	}{
		{
			name:       "正常查询监考列表",
			forceError: "",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				v := url.Values{}
				v.Set("q", `{
					"Action": "select",
					"OrderBy": [{"ID": "DESC"}],
					"Filter": {
						"ExamSessionName": "",
						"ExamRoomName": "",
						"StartTime": -1,
						"EndTime": -1,
						"ExamSessionStatus": ""
					},
					"Page": -1,
					"PageSize": 10
				}`)
				return v
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, svc *cmn.ServiceCtx) {
				if svc == nil || svc.Msg == nil {
					t.Fatalf("serviceCtx or Msg is nil")
				}
				if svc.Msg.Data == nil {
					t.Errorf("预期返回数据, 结果为空")
				}
			},
		},
		{
			name:       "强制JSON解析错误",
			forceError: "json.Unmarshal1",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				return url.Values{}
			}(),
			wantErr:       true,
			errorContains: "强制JSON解析错误",
		},
		{
			name:       "强制JSON序列化错误",
			forceError: "json.Marshal",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				return url.Values{}
			}(),
			wantErr:       true,
			errorContains: "强制JSON序列化错误",
		},
		{
			name:       "强制数据库查询错误",
			forceError: "QueryRow",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				return url.Values{}
			}(),
			wantErr:       true,
			errorContains: "强制执行错误",
		},
		{
			name:       "强制数据库查询错误2",
			forceError: "conn.Query",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				v := url.Values{}
				v.Set("q", `{
					"Action": "select",
					"OrderBy": [{"ID": "DESC"}],
					"Filter": {
						"ExamSessionName": "123",
						"ExamRoomName": "123",
						"StartTime": -1,
						"EndTime": -1,
						"ExamSessionStatus": "12"
					},
					"Page": 0,
					"PageSize": 10
				}`)
				return v
			}(),
			wantErr:       true,
			errorContains: "强制执行错误",
		},
		{
			name:       "强制数据库查询错误3",
			forceError: "rows.Scan",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				v := url.Values{}
				v.Set("q", `{
					"Action": "select",
					"OrderBy": [{"ID": "DESC"}],
					"Filter": {
						"ExamSessionName": "",
						"ExamRoomName": "",
						"StartTime": -1,
						"EndTime": -1,
						"ExamSessionStatus": ""
					},
					"Page": 0,
					"PageSize": 10
				}`)
				return v
			}(),
			wantErr:       true,
			errorContains: "强制执行错误",
		},
		{
			name:       "空结果集（不存在的场次名）",
			forceError: "",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				v := url.Values{}
				v.Set("q", `{"Filter":{"ExamSessionName":"不存在的场次名"}}`)
				return v
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, svc *cmn.ServiceCtx) {

			},
		},
		{
			name:       "无效用户ID",
			forceError: "",
			userID:     0,
			userRole:   2002,
			queryParams: func() url.Values {
				return url.Values{}
			}(),
			wantErr: true,
		},
		{
			name:       "无效的方法",
			forceError: "",
			userID:     testAcademicAffair,
			userRole:   2002,
			queryParams: func() url.Values {
				return url.Values{}
			}(),
			wantErr:       true,
			errorContains: "unsupported method",
			method:        "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.method == "" {
				ctx = createMockContextWithRole("GET", "/invigilation/list", tt.queryParams, tt.forceError, tt.userID, tt.userRole)
			} else {
				ctx = createMockContextWithRole(tt.method, "/invigilation/list", tt.queryParams, tt.forceError, tt.userID, tt.userRole)
			}

			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "invigilationList-force-error", tt.forceError)
			}

			invigilationList(ctx)

			// 获取 ServiceCtx 以检查结果
			svc := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)
			if svc == nil {
				t.Fatalf("未能获取 ServiceCtx")
			}

			if tt.wantErr {
				if svc.Msg == nil || svc.Err == nil {
					t.Errorf("期望错误但未收到")
					return
				}
				if tt.errorContains != "" && !strings.Contains(svc.Err.Error(), tt.errorContains) {
					t.Errorf("期望错误包含: %q, 但实际错误: %v", tt.errorContains, svc.Err)
				}
				return
			}

			// 期望无错误
			if svc.Msg != nil && svc.Err != nil {
				t.Errorf("期望无错误但收到: %v", svc.Err)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, svc)
			}
		})
	}
}
