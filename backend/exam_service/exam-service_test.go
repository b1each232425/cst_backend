package exam_service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"w2w.io/cmn"
	"w2w.io/null"
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

	testAcademicAffair                   = int64(99901)
	testStudent1                         = int64(99902)
	testGrader                           = int64(99903) // 用于考试批阅员
	testExamSession1StartTime            = time.Now().Add(-20 * time.Minute).UnixMilli()
	testExamSession1EndTime              = time.Now().Add(-10 * time.Minute).UnixMilli()
	testExamSession2StartTime            = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSession2EndTime              = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionStartTime       = time.Now().Add(30 * time.Minute).UnixMilli()
	testDeleteExamSessionEndTime         = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID1StartTime = time.Now().Add(10 * time.Minute).UnixMilli()
	testExamSessionToPublishID1EndTime   = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2StartTime = time.Now().Add(20 * time.Minute).UnixMilli()
	testExamSessionToPublishID2EndTime   = time.Now().Add(30 * time.Minute).UnixMilli()

	// 测试考试场次开始和结束事件的相关变量
	testSessionStartID                       = int64(99950) // 用于测试场次开始事件
	testSessionEndID                         = int64(99951) // 用于测试场次结束事件
	testSessionNoExamineeID                  = int64(99952) // 用于测试没有考生的场次
	testSessionEndWithUnfinishedID           = int64(99953) // 用于测试有未完成考生的场次结束
	testExamForSessionStart                  = int64(99950) // 用于场次开始测试的考试
	testExamForSessionEnd                    = int64(99951) // 用于场次结束测试的考试
	testExamineeForSessionStart1             = int64(99950) // 用于测试的考生1
	testExamineeForSessionStart2             = int64(99951) // 用于测试的考生2
	testExamineeForSessionEnd1               = int64(99952) // 用于测试结束的考生1
	testExamineeForSessionEnd2               = int64(99953) // 用于测试结束的考生2
	testExamSessionToPublishID3StartTime     = time.Now().Add(30 * time.Minute).UnixMilli()
	testExamSessionToPublishID3EndTime       = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4StartTime     = time.Now().Add(40 * time.Minute).UnixMilli()
	testExamSessionToPublishID4EndTime       = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5StartTime     = time.Now().Add(50 * time.Minute).UnixMilli()
	testExamSessionToPublishID5EndTime       = time.Now().Add(60 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDStartTime = time.Now().Add(-10 * time.Minute).UnixMilli()
	testErrorExamSessionToPublishIDEndTime   = time.Now().UnixMilli()
	BankQuestionIDs                          = []int64{10000001, 10000002, 10000003, 10000004, 10000005}

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

// 定义自定义类型用于 context key，避免键冲突
type contextKey string

// 生成一张测试试卷
func CreateTestPaperWithGroupsAndQuestions(ctx context.Context, tx pgx.Tx, bankQuestionIDs []int64, testUserID int64) (groupIDs []int64, questionIDs []int64, err error) {
	now := time.Now().UnixMilli()

	paperID := testPaperToPublishID

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

	// 插入测试教务员数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testAcademicAffair, "sys^admin", "测试用户", "test_user", 2002)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 插入测试批阅员数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testGrader, "sys^admin", "测试批阅员", "test_grader", 2005)
	if err != nil {
		t.Fatalf("创建测试批阅员失败: %v", err)
	}

	// 插入测试学生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testStudent1, "sys^student", "测试学生", "test_student", 2008)
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
		_, err = tx.Exec(ctx, `
			INSERT INTO assessuser.t_question (id, type, difficulty, creator,status)
			VALUES ($1, $2, $3, $4, $5)
		`, q.id, q.qtype, q.difficulty, q.creator, q.status)
		if err != nil {
			t.Fatalf("插入测试题目数据失败: %v", err)
		}
	}

	// 创建用于测试发布的试卷
	_, _, err = CreateTestPaperWithGroupsAndQuestions(ctx, tx, BankQuestionIDs, testAcademicAffair)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("创建试卷失败: %v", err)
	}

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
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time, domain_id, files)
		VALUES ($1, '测试正常考试', '00', '00', '02', $2, $3, $2, $3, $4, '[99903]'), 
		($5, '测试已删除的考试', '00', '00', '12', $2, $3, $2, $3, $4, '[]'),
		($6, '测试正常考试2', '00', '00', '02', $2, $3, $2, $3, $4, '[]'),
		($7, '测试发布考试', '00', '00', '00', $2, $3, $2, $3, $4, '[]'),
		($8, '测试发布错误考试', '00', '00', '00', $2, $3, $2, $3, $4, '[]'),
		($9, '测试已结束的考试', '00', '00', '06', $2, $3, $2, $3, $4, '[]'),
		($10, '测试已发布的考试', '00', '00', '02', $2, $3, $2, $3, $4, '[]')
	`, testNormalExamID, testAcademicAffair, time.Now().UnixMilli(), 2002, testDeleteExamID,
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID)
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
		VALUES ($1, $2, $3, $19, '00', '00', 1, '02', $4, $5, $4, $5, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $19, '00', '00', 2, '02', $4, $5, $4, $5, $9, $10, '00', 10, '00'), 
		($11, $12, $3, $19, '00', '00', 3, '12', $3, $4, $3, $4, $13, $14, '00', 10, '00'),
		($15, $16, $3, $20, '00', '00', 4, '02', $3, $4, $3, $4, $17, $18, '00', 10, '00')
	`, testExamSessionID1, testNormalExamID, testPaperID, testAcademicAffair, time.Now().UnixMilli(),
		testExamSession1StartTime, testExamSession1EndTime, testExamSessionID2, testExamSession2StartTime, testExamSession2EndTime,
		testDeleteExamSessionID, testDeleteExamID, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime,
		testExamSessionID3, testNormalExamID2, testDeleteExamSessionStartTime, testDeleteExamSessionEndTime, reviewerIDs, nilReviewerIDs)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试场次数据失败: %v", err)
	}

	// 插入要发布的考试场次数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, reviewer_ids, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time, start_time, end_time, period_mode, duration, question_shuffled_mode)
		VALUES ($1, $2, $3, $23, '00', '00', 1, '00', $4, $5, $4, $5, $6, $7, '00', 10, '00'), 
		($8, $2, $3, $23, '00', '00', 2, '00', $4, $5, $4, $5, $9, $10, '00', 10, '02'),
		($11, $2, $3, $23, '00', '00', 3, '00', $4, $5, $4, $5, $12, $13, '00', 10, '04'),
		($14, $2, $3, $24, '00', '00', 4, '00', $4, $5, $4, $5, $15, $16, '00', 10, '06'),
		($17, $2, $3, $23, '00', '00', 5, '00', $4, $5, $4, $5, $18, $19, '00', 10, '08'),
		($20, $25, $3, $23, '00', '00', 6, '00', $4, $5, $4, $5, $21, $22, '00', 10, '10'),
		($26, $27, $3, $23, '00', '00', 7, '02', $4, $5, $4, $5, $6, $7, '00', 10, '12')
	`, testExamSessionToPublishID1, testExamToPublishID, testPaperToPublishID, testAcademicAffair, time.Now().UnixMilli(), testExamSessionToPublishID1StartTime, testExamSessionToPublishID1EndTime,
		testExamSessionToPublishID2, testExamSessionToPublishID2StartTime, testExamSessionToPublishID2EndTime,
		testExamSessionToPublishID3, testExamSessionToPublishID3StartTime, testExamSessionToPublishID3EndTime,
		testExamSessionToPublishID4, testExamSessionToPublishID4StartTime, testExamSessionToPublishID4EndTime,
		testExamSessionToPublishID5, testExamSessionToPublishID5StartTime, testExamSessionToPublishID5EndTime,
		testErrorExamSessionToPublishID, testErrorExamSessionToPublishIDStartTime, testErrorExamSessionToPublishIDEndTime,
		reviewerIDs, nilReviewerIDs, testErrorExamToPublishID1, testPublishedExamSessionID, testPublishedExamID)

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

	// 插入考卷数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_paper (id, exam_session_id, name, creator, status)
		VALUES ($1, $2, $3, $4, $5)
	`, testExamSessionID1, testExamSessionID1, "testPaper", testAcademicAffair, "00")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考卷数据失败: %v", err)
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

	// 删除考卷数据
	_, err = tx.Exec(ctx, `
		DELETE FROM t_exam_paper WHERE creator = $1
	`, testAcademicAffair)

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
		testNormalExamID2, testExamToPublishID, testErrorExamToPublishID1, testEndExamID, testPublishedExamID)
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

	t.Logf("删除测试文件1: %s", testFile1Path)
	t.Logf("删除测试文件2: %s", testFile2Path)
	t.Logf("删除测试文件3: %s", testFile3Path)
	t.Logf("删除测试文件4: %s", testSameFilePath)
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

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSetExamTimers(t *testing.T) {

	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer CleanTestExamData(t)

	tests := []struct {
		name          string
		examID        int64
		expectError   bool
		errorContains string
		checkTimers   bool
		forceError    string
	}{
		{
			name:        "正常设置考试定时器",
			examID:      testNormalExamID,
			expectError: false,
			checkTimers: true,
		},
		{
			name:        "设置不存在的考试定时器",
			examID:      999999,
			expectError: false,
			checkTimers: false,
		},
		{
			name:          "查询考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制查询考试场次信息错误",
			checkTimers:   false,
			forceError:    "queryExamSessions",
		},
		{
			name:          "扫描考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制获取考试场次信息错误",
			checkTimers:   false,
			forceError:    "scanExamSessionInfo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			if tt.forceError != "" {
				// 强制模拟错误
				ctx = context.WithValue(ctx, "SetExamTimers-force-error", tt.forceError)
			}

			// 初始化全局定时器管理器
			examTimerMgr = NewExamTimerManager(ctx, cancel)
			defer examTimerMgr.StopAll()

			// 执行测试
			err := SetExamTimers(ctx, tt.examID)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("期望错误包含 '%s'，但得到 '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到错误: %v", err)
				}
			}

			// 验证定时器设置
			if tt.checkTimers && !tt.expectError {
				examTimerMgr.mutex.Lock()
				startTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_START, testExamSessionID2)
				endTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_END, testExamSessionID2)

				if _, exists := examTimerMgr.timers[startTimerKey]; !exists {
					t.Error("考试场次开始定时器未设置")
				}

				if _, exists := examTimerMgr.timers[endTimerKey]; !exists {
					t.Error("考试场次结束定时器未设置")
				}
				examTimerMgr.mutex.Unlock()

				t.Logf("成功设置定时器，当前定时器数量: %d", len(examTimerMgr.timers))
			}
		})
	}
}

func TestCancelExamTimers(t *testing.T) {
	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer CleanTestExamData(t)

	tests := []struct {
		name          string
		examID        int64
		expectError   bool
		errorContains string
		setupTimers   bool
		forceError    string
	}{
		{
			name:        "正常取消考试定时器",
			examID:      testNormalExamID2,
			expectError: false,
			setupTimers: true,
		},
		{
			name:        "取消不存在的考试定时器",
			examID:      999999,
			expectError: false,
			setupTimers: false,
		},
		{
			name:          "查询考试场次信息错误",
			examID:        testNormalExamID2,
			expectError:   true,
			errorContains: "强制查询考试场次信息错误",
			setupTimers:   false,
			forceError:    "CancelExamTimers-force-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			if tt.forceError != "" {
				// 强制模拟错误
				ctx = context.WithValue(ctx, "CancelExamTimers-force-error", tt.forceError)
			}

			// 初始化全局定时器管理器
			examTimerMgr = NewExamTimerManager(ctx, cancel)
			defer examTimerMgr.StopAll()

			// 如果需要，先设置定时器
			if tt.setupTimers {
				err := SetExamTimers(context.Background(), tt.examID)
				if err != nil {
					t.Fatalf("设置考试定时器失败: %v", err)
				}

				// 验证定时器已设置
				examTimerMgr.mutex.Lock()
				initialTimerCount := len(examTimerMgr.timers)
				examTimerMgr.mutex.Unlock()

				if initialTimerCount == 0 {
					t.Fatal("定时器设置失败，无法进行取消测试")
				}
				t.Logf("设置了 %d 个定时器", initialTimerCount)
			}

			// 执行测试 - 取消定时器
			err := CancelExamTimers(ctx, tt.examID)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("期望错误包含 '%s'，但得到 '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到错误: %v", err)
				}

				// 验证定时器已取消
				if tt.setupTimers {
					examTimerMgr.mutex.Lock()
					startTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_START, testExamSessionID1)
					endTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_END, testExamSessionID1)

					if _, exists := examTimerMgr.timers[startTimerKey]; exists {
						t.Error("考试场次开始定时器未正确取消")
					}

					if _, exists := examTimerMgr.timers[endTimerKey]; exists {
						t.Error("考试场次结束定时器未正确取消")
					}

					finalTimerCount := len(examTimerMgr.timers)
					examTimerMgr.mutex.Unlock()

					t.Logf("取消定时器后，剩余定时器数量: %d", finalTimerCount)
				}
			}
		})
	}
}

// 测试考试场次开始事件处理
func TestHandleExamSessionStart(t *testing.T) {
	cmn.ConfigureForTest()

	// 准备测试数据
	CleanTestExamData(t)
	CreateTestExamData(t)
	defer CleanTestExamData(t)

	tests := []struct {
		name               string
		event              ExamEvent
		expectError        bool
		expectedStatus     string
		expectedCount      int
		forceError         string
		checkExamStatus    bool
		expectedExamStatus string
	}{
		{
			name: "正常场次开始-有考生",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError:        false,
			expectedStatus:     "04", // 进行中
			expectedCount:      1,    // 1个考生
			checkExamStatus:    true,
			expectedExamStatus: "04", // 考试进行中
		},
		{
			name: "场次开始-无考生-强制更新考试为异常状态错误",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testPublishedExamID,
				ExamSessionID: testPublishedExamSessionID,
			},
			expectError:        true,
			expectedStatus:     "02",
			expectedCount:      0, // 0个考生
			checkExamStatus:    true,
			expectedExamStatus: "10",
			forceError:         "updateAbnormalExam",
		},
		{
			name: "场次开始-无考生-取消考试定时器失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testPublishedExamID,
				ExamSessionID: testPublishedExamSessionID,
			},
			expectError:        true,
			expectedStatus:     "02",
			expectedCount:      0, // 0个考生
			checkExamStatus:    true,
			expectedExamStatus: "10",
			forceError:         "cancelAbnormalExamTimers",
		},
		{
			name: "正常场次开始-无考生",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testPublishedExamID,
				ExamSessionID: testPublishedExamSessionID,
			},
			expectError:        false,
			expectedStatus:     "02",
			expectedCount:      0, // 0个考生
			checkExamStatus:    true,
			expectedExamStatus: "10",
		},
		{
			name: "开启事务失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "beginTx",
		},
		{
			name: "查询场次信息失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "queryExamSessionInfo",
		},
		{
			name: "更新考试状态失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "updateExam",
		},
		{
			name: "更新场次状态失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "updateExamSession",
		},
		{
			name: "处理过程中发生panic",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: false,
			forceError:  "panic",
		},
		{
			name: "事务回滚失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: false,
			forceError:  "rollback",
		},
		{
			name: "强制提交错误",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_START,
				ExamID:        testNormalExamID2,
				ExamSessionID: testExamSessionID1,
			},
			expectError: false,
			forceError:  "commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "handleExamSessionStart-force-error", tt.forceError)
			}

			examTimerMgr = NewExamTimerManager(ctx, cancel)

			// 执行测试
			err := handleExamSessionStart(ctx, tt.event)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				}
				return
			}

			if tt.forceError != "" {
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但得到错误: %v", err)
				return
			}

			// 验证场次状态
			var sessionStatus string
			conn := cmn.GetPgxConn()
			err = conn.QueryRow(ctx, `
				SELECT status FROM t_exam_session WHERE id = $1
			`, tt.event.ExamSessionID).Scan(&sessionStatus)

			if err != nil {
				t.Errorf("查询场次状态失败: %v", err)
				return
			}

			if sessionStatus != tt.expectedStatus {
				t.Errorf("期望场次状态为 %s，但得到: %s", tt.expectedStatus, sessionStatus)
			}

			// 验证考试状态
			if tt.checkExamStatus {
				var examStatus string
				err = conn.QueryRow(ctx, `
					SELECT status FROM t_exam_info WHERE id = $1
				`, tt.event.ExamID).Scan(&examStatus)

				if err != nil {
					t.Errorf("查询考试状态失败: %v", err)
				} else if examStatus != tt.expectedExamStatus {
					t.Errorf("期望考试状态为 %s，但得到: %s", tt.expectedExamStatus, examStatus)
				}
			}

			// 验证考生数量
			var actualCount int
			err = conn.QueryRow(ctx, `
				SELECT COUNT(*) FROM t_examinee 
				WHERE exam_session_id = $1 AND status != '08'
			`, tt.event.ExamSessionID).Scan(&actualCount)

			if err != nil {
				t.Errorf("查询考生数量失败: %v", err)
			} else if actualCount != tt.expectedCount {
				t.Errorf("期望考生数量为 %d，但得到: %d", tt.expectedCount, actualCount)
			}

			t.Logf("场次 %d 状态: %s, 考生数量: %d", tt.event.ExamSessionID, sessionStatus, actualCount)
		})
	}
}

// 测试考试场次结束事件处理
func TestHandleExamSessionEnd(t *testing.T) {
	cmn.ConfigureForTest()

	// 准备测试数据
	defer CleanTestExamData(t)
	CleanTestExamData(t)
	CreateTestExamData(t)

	tests := []struct {
		name             string
		event            ExamEvent
		expectError      bool
		expectedStatus   string
		expectDelayTimer bool
		forceError       string
		checkExamStatus  bool
	}{
		{
			name: "强制更新考试状态为已结束错误",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError:      true,
			expectedStatus:   "06", // 已结束
			expectDelayTimer: false,
			checkExamStatus:  false,
			forceError:       "updateExamEndStatus",
		},
		{
			name: "正常场次结束-有考生",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError:      false,
			expectedStatus:   "06", // 已结束
			expectDelayTimer: false,
			checkExamStatus:  false,
		},
		{
			name: "事务回滚失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError:      false,
			expectDelayTimer: false,
			checkExamStatus:  false,
			forceError:       "rollback",
		},
		{
			name: "事务提交失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError:      false,
			expectDelayTimer: false,
			checkExamStatus:  false,
			forceError:       "commit",
		},
		{
			name: "开启事务失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "beginTx",
		},
		{
			name: "更新考生状态失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "updateExaminees",
		},
		{
			name: "检查未完成考生失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "scanUnfinishedExaminees",
		},
		{
			name: "更新场次状态失败",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError: true,
			forceError:  "updateSessionEndStatus",
		},
		{
			name: "处理过程中发生panic",
			event: ExamEvent{
				Type:          EVENT_TYPE_EXAM_SESSION_END,
				ExamID:        testNormalExamID,
				ExamSessionID: testExamSessionID1,
			},
			expectError: false,
			forceError:  "panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "handleExamSessionEnd-force-error", tt.forceError)
			}

			// 初始化定时器管理器
			examTimerMgr = NewExamTimerManager(ctx, cancel)
			defer examTimerMgr.StopAll()

			// 执行测试
			err := handleExamSessionEnd(ctx, tt.event)

			// 验证错误
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但得到错误: %v", err)
				return
			}

			// 验证场次状态
			var sessionStatus string
			conn := cmn.GetPgxConn()
			err = conn.QueryRow(ctx, `
				SELECT status FROM t_exam_session WHERE id = $1
			`, tt.event.ExamSessionID).Scan(&sessionStatus)

			if err != nil {
				t.Errorf("查询场次状态失败: %v", err)
				return
			}

			if sessionStatus != tt.expectedStatus && tt.expectedStatus != "" {
				t.Errorf("期望场次状态为 %s，但得到: %s", tt.expectedStatus, sessionStatus)
			}

			// 验证考试状态
			if tt.checkExamStatus {
				var examStatus string
				err = conn.QueryRow(ctx, `
					SELECT status FROM t_exam_info WHERE id = $1
				`, tt.event.ExamID).Scan(&examStatus)

				if err != nil {
					t.Errorf("查询考试状态失败: %v", err)
				} else if examStatus != "06" {
					t.Errorf("期望考试状态为已结束(06)，但得到: %s", examStatus)
				}
			}

			// 验证延迟定时器
			if tt.expectDelayTimer {
				examTimerMgr.mutex.Lock()
				delayTimerKey := fmt.Sprintf("%s_%d", EVENT_TYPE_EXAM_SESSION_END, tt.event.ExamSessionID)
				_, hasDelayTimer := examTimerMgr.timers[delayTimerKey]
				examTimerMgr.mutex.Unlock()

				if !hasDelayTimer {
					t.Error("期望设置延迟定时器，但未找到")
				}
			}

			t.Logf("场次 %d 状态: %s", tt.event.ExamSessionID, sessionStatus)
		})
	}
}

// TestCleanupTempExams 测试cleanupTempExams函数
func TestCleanupTempExams(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 临时考试ID列表
	tempExamIDs := []int64{89990, 89991, 89992, 89993, 89994}
	testUserID := int64(89990)

	// 准备测试数据
	t.Cleanup(func() {
		cleanupTempExamTestData(t, tempExamIDs, testUserID)
	})

	tests := []struct {
		name          string
		forceError    string
		expectedError bool
		description   string
		checkDeletion bool
		expectedCount int64 // 期望删除的记录数
	}{
		{
			name:          "成功清理过期临时考试",
			forceError:    "",
			expectedError: false,
			description:   "正常清理24小时前创建的临时考试",
			checkDeletion: true,
			expectedCount: 3, // 3个过期的临时考试
		},
		{
			name:          "数据库删除操作失败",
			forceError:    "deleteTempExams",
			expectedError: true,
			description:   "模拟数据库删除操作失败",
			checkDeletion: false,
			expectedCount: 0,
		},
		{
			name:          "强制处理删除考试文件错误",
			forceError:    "handleDeleteExamFile",
			expectedError: true,
			description:   "强制处理删除考试文件错误",
			checkDeletion: false,
			expectedCount: 0,
		},
		{
			name:          "扫描文件数组失败",
			forceError:    "scanFiles",
			expectedError: true,
			description:   "模拟扫描文件数组失败",
			checkDeletion: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cleanupTempExamTestData(t, tempExamIDs, testUserID)

			// 创建测试数据
			setupTempExamTestData(t, tempExamIDs, testUserID)

			// 创建包含强制错误的上下文
			ctx := context.Background()
			if tt.forceError != "" {
				if tt.forceError == "handleDeleteExamFile.tx.QueryRow" {
					ctx = context.WithValue(ctx, "force-error", tt.forceError)
				} else {
					ctx = context.WithValue(ctx, "cleanupTempExams-force-error", tt.forceError)
				}
			}

			// 记录清理前的临时考试数量
			var beforeCount int64
			err := pgxConn.QueryRow(ctx, `
				SELECT COUNT(*) FROM t_exam_info 
				WHERE status = '14' AND create_time < $1 AND id = ANY($2)
			`, time.Now().Add(-24*time.Hour).UnixMilli(), tempExamIDs).Scan(&beforeCount)
			if err != nil {
				t.Fatalf("查询清理前临时考试数量失败: %v", err)
			}

			// 执行清理函数
			var cleanupErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if !tt.expectedError {
							t.Errorf("cleanupTempExams() 意外panic: %v", r)
						}
					}
				}()

				cleanupErr = cleanupTempExams(ctx)
			}()

			// 验证错误
			if tt.expectedError {
				assert.Error(t, cleanupErr, "期望出现错误，但函数成功执行")
				if cleanupErr != nil {
					t.Logf("预期错误: %v", cleanupErr)
				}
			} else {
				assert.NoError(t, cleanupErr, "不期望出现错误，但函数执行失败")
			}

			if tt.checkDeletion {
				// 验证清理后的临时考试数量
				var afterCount int64
				err := pgxConn.QueryRow(context.Background(), `
					SELECT COUNT(*) FROM t_exam_info 
					WHERE status = '14' AND create_time < $1 AND id = ANY($2)
				`, time.Now().Add(-24*time.Hour).UnixMilli(), tempExamIDs).Scan(&afterCount)
				if err != nil {
					t.Fatalf("查询清理后临时考试数量失败: %v", err)
				}

				deletedCount := beforeCount - afterCount
				if deletedCount != tt.expectedCount {
					t.Errorf("期望删除 %d 个临时考试，实际删除 %d 个", tt.expectedCount, deletedCount)
				}

				// 验证未过期的临时考试仍然存在
				var recentCount int64
				err = pgxConn.QueryRow(context.Background(), `
					SELECT COUNT(*) FROM t_exam_info 
					WHERE status = '14' AND create_time >= $1 AND id = ANY($2)
				`, time.Now().Add(-24*time.Hour).UnixMilli(), []int64{tempExamIDs[3], tempExamIDs[4]}).Scan(&recentCount)
				if err != nil {
					t.Fatalf("查询未过期临时考试数量失败: %v", err)
				}

				if recentCount != 2 { // 应该还有2个未过期的临时考试
					t.Errorf("期望保留 2 个未过期的临时考试，实际保留 %d 个", recentCount)
				}
			}

			t.Logf("测试完成: %s", tt.description)
		})
	}
}

// setupTempExamTestData 创建临时考试测试数据
func setupTempExamTestData(t *testing.T, tempExamIDs []int64, testUserID int64) {
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Fatalf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Fatalf("事务回滚: %v", err)
			} else {
				err = tx.Commit(ctx)
				if err != nil {
					t.Fatalf("事务提交失败: %v", err)
				}
			}
		}
	}()

	// 创建测试用户
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, category, official_name, account, role) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING`, testUserID, "sys^admin", "临时考试测试用户", "temp_exam_test_user", 2002)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	hour25Ago := time.Now().Add(-25 * time.Hour).UnixMilli() // 25小时前，应该被清理
	hour26Ago := time.Now().Add(-26 * time.Hour).UnixMilli() // 26小时前，应该被清理
	hour27Ago := time.Now().Add(-27 * time.Hour).UnixMilli() // 27小时前，应该被清理
	hour23Ago := time.Now().Add(-23 * time.Hour).UnixMilli() // 23小时前，不应该被清理
	hour22Ago := time.Now().Add(-22 * time.Hour).UnixMilli() // 22小时前，不应该被清理

	// 创建临时考试数据
	examData := []struct {
		id          int64
		name        string
		createTime  int64
		status      string
		files       string
		shouldClean bool
	}{
		{tempExamIDs[0], "过期临时考试1", hour25Ago, "14", "[99901]", true}, // 应该被清理
		{tempExamIDs[1], "过期临时考试2", hour26Ago, "14", "[]", true},      // 应该被清理
		{tempExamIDs[2], "过期临时考试3", hour27Ago, "14", "[]", true},      // 应该被清理
		{tempExamIDs[3], "未过期临时考试1", hour23Ago, "14", "[]", false},    // 不应该被清理
		{tempExamIDs[4], "未过期临时考试2", hour22Ago, "14", "[]", false},    // 不应该被清理
	}

	for _, exam := range examData {
		_, err = tx.Exec(ctx, `
			INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time, domain_id, files)
			VALUES ($1, $2, '00', '00', $3, $4, $5, $4, $5, 2000, $6)
		`, exam.id, exam.name, exam.status, testUserID, exam.createTime, exam.files)
		if err != nil {
			t.Fatalf("创建临时考试数据失败 (ID: %d): %v", exam.id, err)
		}
	}

	// 插入文件记录到数据库
	uploadDir := "./uploads"
	testFile1Path := filepath.Join(uploadDir, testFile1CheckSum)
	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, testFile1ID, testFile1CheckSum, testFile1Name, testFile1Path, "/test/files", len(testFile1Content), testAcademicAffair, time.Now().UnixMilli()).Scan(&testFile1ID)
	if err != nil {
		t.Fatalf("插入测试文件1记录失败: %v", err)
	}

	t.Logf("成功创建 %d 个临时考试测试数据", len(examData))
}

// cleanupTempExamTestData 清理临时考试测试数据
func cleanupTempExamTestData(t *testing.T, tempExamIDs []int64, testUserID int64) {
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("清理事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("清理事务回滚: %v", err)
			} else {
				tx.Commit(ctx)
			}
		}
	}()

	// 删除临时考试数据
	for _, examID := range tempExamIDs {
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
		if err != nil {
			t.Logf("删除临时考试数据失败 (ID: %d): %v", examID, err)
		}
	}

	// 删除测试用户
	_, err = tx.Exec(ctx, `DELETE FROM t_user WHERE id = $1`, testUserID)
	if err != nil {
		t.Logf("删除测试用户失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
        DELETE FROM t_file WHERE id = ANY($1)
    `, []int64{testFile1ID, testFile2ID, testFileID3, testSameFileID, notExistsFileID})
	if err != nil {
		t.Logf("删除测试文件记录失败: %v", err)
	}

	t.Logf("清理临时考试测试数据完成")
}

// TestInitializeExamTimers 测试InitializeExamTimers函数
func TestInitializeExamTimers(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name           string
		forceError     string
		expectedError  bool
		description    string
		checkTimers    bool
		expectedTimers int // 期望设置的定时器数量
	}{
		{
			name:           "成功初始化考试定时器",
			forceError:     "",
			expectedError:  false,
			description:    "正常初始化现有考试的定时器",
			checkTimers:    false,
			expectedTimers: 4, // 每个考试场次2个定时器（开始+结束）
		},
		{
			name:          "查询考试场次信息失败",
			forceError:    "queryExamSessions",
			expectedError: true,
			description:   "模拟查询考试场次信息失败",
			checkTimers:   false,
		},
		{
			name:          "扫描考试场次信息失败",
			forceError:    "scanExamSessionInfo",
			expectedError: true,
			description:   "模拟扫描考试场次信息失败",
			checkTimers:   false,
		},
		{
			name:          "强制触发panic",
			forceError:    "panic",
			expectedError: false, // panic被recover，不返回error
			description:   "测试panic处理机制",
			checkTimers:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每个子测试都重新创建和清理数据
			CleanTestExamData(t)
			CreateTestExamData(t)
			defer CleanTestExamData(t)

			// 创建带超时的上下文，防止无限阻塞
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "initializeExamTimers-force-error", tt.forceError)
			}

			// 初始化定时器管理器
			examTimerMgr = NewExamTimerManager(ctx, cancel)
			defer func() {
				// 确保定时器管理器被正确停止
				if examTimerMgr != nil {
					examTimerMgr.StopAll()
				}
			}()

			// 执行测试
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if !tt.expectedError && tt.forceError != "panic" {
							t.Errorf("InitializeExamTimers() 意外panic: %v", r)
						}
					}
				}()

				err = InitializeExamTimers(ctx)
			}()

			// 验证错误
			if tt.expectedError {
				assert.Error(t, err, "期望出现错误，但函数成功执行")
				if err != nil {
					t.Logf("预期错误: %v", err)
				}
				return
			}

			assert.NoError(t, err, "不期望出现错误，但函数执行失败")

			// 验证定时器设置
			if tt.checkTimers {
				examTimerMgr.mutex.Lock()
				timerCount := len(examTimerMgr.timers)
				examTimerMgr.mutex.Unlock()

				if timerCount != tt.expectedTimers {
					t.Errorf("期望设置 %d 个定时器，实际设置 %d 个", tt.expectedTimers, timerCount)
				}

				// 验证定时器类型
				examTimerMgr.mutex.Lock()
				startTimers := 0
				endTimers := 0
				for key := range examTimerMgr.timers {
					if strings.Contains(key, EVENT_TYPE_EXAM_SESSION_START) {
						startTimers++
					}
					if strings.Contains(key, EVENT_TYPE_EXAM_SESSION_END) {
						endTimers++
					}
				}
				examTimerMgr.mutex.Unlock()

				expectedSessionTimers := tt.expectedTimers / 2
				if startTimers != expectedSessionTimers {
					t.Errorf("期望设置 %d 个开始定时器，实际设置 %d 个", expectedSessionTimers, startTimers)
				}
				if endTimers != expectedSessionTimers {
					t.Errorf("期望设置 %d 个结束定时器，实际设置 %d 个", expectedSessionTimers, endTimers)
				}

				t.Logf("成功设置定时器: 开始定时器=%d, 结束定时器=%d", startTimers, endTimers)
			}

			t.Logf("测试完成: %s", tt.description)
		})
	}
}

func TestProcessEventDefaultCase(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 测试默认事件处理
	event := ExamEvent{
		Type:          "unknown_event",
		ExamSessionID: 123,
		ExamID:        456,
	}
	err := examTimerMgr.processEvent(event, 1)
	assert.Error(t, err, "期望处理未知事件时返回错误")
}
