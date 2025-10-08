package mark

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

const testedTeacherID = 1101

var now = time.Now()

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func initTestData() {
	var sqls []string
	var args [][][]interface{}

	sqls = append(sqls, `INSERT INTO t_user(id, account, official_name, category) VALUES($1, $2, $3, $4)`)
	args = append(args, [][]interface{}{
		{1101, "test account 1", "user1", "test user"},
		{1102, "test account 2", "user2", "test user"},
		{1103, "test account 3", "user3", "test user"},
		{1201, "test account 4", "student1", "test student"},
		{1202, "test account 5", "student2", "test student"},
		{1203, "test account 6", "student3", "test student"},
		{1204, "test account 7", "student4", "test student"},
	})

	sqls = append(sqls, `INSERT INTO t_exam_info(id, name, type, creator, create_time, status) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{11, "test exam 1", "00", 1101, now.Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{12, "test exam 2", "00", 1101, now.Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{13, "test exam 3", "00", 1101, now.Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
	})

	sqls = append(sqls, `INSERT INTO t_exam_session(id, exam_id, paper_id, mark_method, mark_mode, start_time, end_time, session_num, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`)
	args = append(args, [][]interface{}{
		{101, 11, 101, "02", "00", now.Add(-25 * time.Hour).UnixMilli(), now.Add(-24 * time.Hour).UnixMilli(), 1, "00", 1101}, // 纯理论
		{102, 11, 102, "00", "00", now.Add(-23 * time.Hour).UnixMilli(), now.Add(-22 * time.Hour).UnixMilli(), 2, "00", 1101}, // 理论+主观
		{103, 11, 103, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 题目答案缺失
		{104, 11, 104, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
		{105, 11, 105, "00", "", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 批改模式缺失
		{106, 11, 106, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_practice(id, name, paper_id, correct_mode, type, status, creator, create_time) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`)
	args = append(args, [][]interface{}{
		{21, "test practice 1", 124, "00", "00", "02", 1101, now.Add(-25 * time.Hour).UnixMilli()},
		{22, "test practice 2", 125, "10", "00", "02", 1101, now.Add(-24 * time.Hour).UnixMilli()},
		{23, "test practice 3", 126, "00", "00", "02", 1101, now.Add(-23 * time.Hour).UnixMilli()},
		{24, "test practice 4", 127, "00", "00", "02", 1101, now.Add(-22 * time.Hour).UnixMilli()}, // 主观 AI 批改
	})

	sqls = append(sqls, `INSERT INTO t_paper(id, exampaper_id, name, domain_id, creator) VALUES($1, $2, $3, $4, $5)`)
	args = append(args, [][]interface{}{
		{101, 101, "paper 1", 1999, 1101},
		{102, 102, "paper 2", 1999, 1101},
		{103, 103, "paper 3", 1999, 1101},
		{104, 104, "paper 4", 1999, 1101},
		{124, 124, "paper 24", 1999, 1101},
		{125, 125, "paper 25", 1999, 1101},
		{126, 126, "paper 26", 1999, 1101},
		{127, 127, "paper 27", 1999, 1101},
	})

	sqls = append(sqls, `INSERT INTO t_exam_paper(id, exam_session_id, practice_id, name, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{101, 101, null.NewInt(0, false), "test exam paper 1", "00", 1101},
		{102, 102, null.NewInt(0, false), "test exam paper 2", "00", 1101},
		{103, 103, null.NewInt(0, false), "test exam paper 3", "00", 1101},
		{104, 104, null.NewInt(0, false), "test exam paper 4", "00", 1101},
		{124, null.NewInt(0, false), 21, "test practice paper 1", "00", 1101},
		{125, null.NewInt(0, false), 22, "test practice paper 2", "00", 1101},
		{126, null.NewInt(0, false), 23, "test practice paper 3", "00", 1101},
		{127, null.NewInt(0, false), 24, "test practice paper 4", "00", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_exam_paper_group(id, exam_paper_id, name, status, creator) VALUES($1, $2, $3, $4, $5)`)
	args = append(args, [][]interface{}{
		{301, 101, "test group 1", "00", 1101},
		{302, 102, "test group 2", "00", 1101},
		{303, 103, "test group 3", "00", 1101},
		{304, 104, "test group 4", "00", 1101},
		{324, 124, "test group 24", "00", 1101},
		{325, 125, "test group 25", "00", 1101},
		{326, 126, "test group 26", "00", 1101},
		{327, 127, "test group 27", "00", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_exam_paper_question(id, score, type, content, answers, group_id, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`)
	args = append(args, [][]interface{}{
		{401, 3, "00", "单选1", `["C"]`, 301, "00", 1101},
		{402, 3, "02", "多选1", `["B", "D"]`, 301, "00", 1101},
		{403, 3, "04", "判断1", `["A"]`, 301, "00", 1101},
		{404, 3, "00", "单选1", `["C"]`, 302, "00", 1101},
		{405, 3, "06", "填空1", `[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`, 302, "00", 1101},
		{406, 10, "08", "简答题1", `[{"index": 1, "score": 10, "answer": "简答题答案1", "grading_rule": "1", "alternative_answer": null}]`, 302, "00", 1101},
		{411, 3, "00", "单选2", `[]`, 303, "00", 1101}, // 异常数据
		{412, 3, "02", "多选2", `["B", "D"]`, 304, "00", 1101},
		{421, 3, "00", "单选2", `[]`, 326, "00", 1101}, // 异常数据
		{422, 3, "06", "填空2", `[{"index": 1, "score": 1, "answer": "填空1第一空答案", "grading_rule": "1"},{"index": 2, "score": 2, "answer": "填空2第二空答案", "grading_rule": "2"}]`, 302, "00", 1101},
		{423, 3, "06", "异常答案填空1", `{}`, 303, "00", 1101},
		{424, 3, "00", "单选1", `["C"]`, 324, "00", 1101}, // 练习
		{425, 3, "02", "多选1", `["B", "D"]`, 324, "00", 1101},
		{426, 3, "04", "判断1", `["A"]`, 324, "00", 1101},
		{427, 3, "06", "练习-填空1", `[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`, 325, "00", 1101},
		{428, 3, "08", "练习-简答题1", `[{"index": 1, "score": 3, "answer": "简答题答案1", "grading_rule": "1", "alternative_answer": null}]`, 327, "00", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_examinee(id, student_id, exam_session_id, serial_number, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{2201, 1201, 101, 1, "10", 1101},
		{2202, 1202, 101, 2, "10", 1101},
		{2203, 1203, 103, 3, "10", 1101},
		{2204, 1204, 103, 4, "10", 1101},
		{2205, 1201, 102, 1, "10", 1101},
		{2206, 1201, 103, 1, "10", 1101},
		{2207, 1202, 103, 2, "10", 1101},
	})

	// 21练习有4人，22一人，24一人
	sqls = append(sqls, `INSERT INTO t_practice_submissions(id, student_id, practice_id, exam_paper_id, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{2401, 1201, 21, 124, "06", 1101},
		{2402, 1202, 21, 124, "06", 1101},
		{2403, 1203, 21, 124, "06", 1101},
		{2404, 1201, 22, 125, "06", 1101},
		{2405, 1202, 24, 127, "06", 1101},
		{2406, 1204, 21, 124, "06", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_practice_wrong_submissions(id, practice_submission_id, status, creator) VALUES($1, $2, $3, $4)`)
	args = append(args, [][]interface{}{
		{2601, 2401, "02", testedTeacherID},
	})

	// 考试作答
	sqls = append(sqls, `INSERT INTO t_student_answers(id, examinee_id, question_id, answer, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7)`)
	args = append(args, [][]interface{}{
		{21, 2201, 401, `{"answer": ["C"]}`, `["C"]`, "00", 1101},
		{22, 2201, 402, `{"answer": ["B", "D"]}`, `["B", "D"]`, "00", 1101},
		{23, 2201, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{24, 2202, 401, `{"answer": ["B"]}`, `["C"]`, "00", 1101},
		{25, 2202, 402, `{"answer": ["B"]}`, `["B", "D"]`, "00", 1101},
		{26, 2202, 403, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{27, 2203, 411, `{"answer": ["B"]}`, `[]`, "00", 1101},
		{28, 2205, 404, `{"answer": ["B"]}`, `["B"]`, "00", 1101},
		{29, 2205, 405, `{"answer": ["填空学生作答1"]}`, `[]`, "00", 1101},
		{30, 2204, 411, `{"answer": {}}`, `["B"]`, "00", 1101},
		{31, 2206, 411, `{"answer": ["B"]}`, `{}`, "00", 1101},
		{32, 2207, 411, `{"answer": []}`, `["B"]`, "00", 1101},
		{33, 2205, 406, `{"answer": ["简答1学生作答1"]}`, `[]`, "00", 1101},
	})

	// 练习作答
	sqls = append(sqls, `INSERT INTO t_student_answers(id, practice_submission_id, question_id, answer, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7)`)
	args = append(args, [][]interface{}{
		{41, 2401, 424, `{"answer": ["C"]}`, `["C"]`, "00", 1101},
		{42, 2401, 425, `{"answer": ["B", "D"]}`, `["B", "D"]`, "00", 1101},
		{43, 2401, 426, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{44, 2402, 424, `{"answer": ["B"]}`, `["C"]`, "00", 1101},
		{45, 2402, 425, `{"answer": ["B"]}`, `["B", "D"]`, "00", 1101},
		{46, 2402, 426, `{"answer": ["A"]}`, `["A"]`, "00", 1101},
		{47, 2404, 427, `{"answer": ["填空学生作答1"]}`, `[]`, "00", 1101},
		{48, 2405, 428, `{"answer": ["简答作答1"]}`, `[]`, "00", 1101},
		{49, 2406, 424, `{"answer": ["C"]}`, `["A"]`, "00", 1101}, // 错题集作答
	})

	sqls = append(sqls, `INSERT INTO t_mark_info(id, exam_session_id, practice_id, mark_teacher_id, creator, status) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{201, 101, null.NewInt(0, false), 1101, 1101, "00"},
		{202, 102, null.NewInt(0, false), 1101, 1101, "00"},
		{203, 103, null.NewInt(0, false), 1101, 1101, "00"},
		{204, 108, null.NewInt(0, false), 1101, 1101, "00"}, // 测试软删除接口
		{205, null.NewInt(0, false), 21, 1101, 1101, "00"},
		{206, 106, null.NewInt(0, false), null.NewInt(0, false), 1101, "00"},
		{207, 105, null.NewInt(0, false), 1101, 1101, "00"},
		{208, null.NewInt(0, false), 22, 1101, 1101, "00"},
	})

	// 考试批改结果
	sqls = append(sqls, `INSERT INTO t_mark(id, teacher_id, exam_session_id, examinee_id, question_id, mark_details, score, creator, status) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	args = append(args, [][]interface{}{
		{9901, testedTeacherID, 102, 2205, 405, `{}`, 2, testedTeacherID, "00"},
	})

	// 练习批改结果
	sqls = append(sqls, `INSERT INTO t_mark(id, teacher_id, practice_id, practice_submission_id, question_id, mark_details, score, creator, status) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	args = append(args, [][]interface{}{
		{9921, testedTeacherID, 22, 2404, 427, `{}`, 2, testedTeacherID, "00"},
	})

	if len(sqls) != len(args) {
		panic("sqls and args length not equal")
		return
	}

	var (
		tx  *sql.Tx
		err error
	)

	sqlxDB := cmn.GetDbConn()
	tx, err = sqlxDB.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		panic(err)
		return
	}

	defer func() {
		if err != nil {
			err = tx.Rollback()
			panic(err)
		}
	}()

	for i, sql := range sqls {
		var ids []interface{}

		for _, arg := range args[i] {
			ids = append(ids, arg[0])

			_, err = tx.Exec(sql, arg...)
			if err != nil {
				err = fmt.Errorf("sql: %v, insert test data SQL error: %v", sql, err)
				panic(err)
				return
			}
		}

		testedIDGroups = append(testedIDGroups, ids)
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		panic(err)
	}
}

func cleanTestData() {
	var sqls []string

	sqls = append(sqls, `DELETE FROM t_mark;`)

	sqls = append(sqls, `DELETE FROM t_student_answers;`)

	sqls = append(sqls, `DELETE FROM t_examinee;`)

	sqls = append(sqls, `DELETE FROM t_exam_paper_question;`)

	sqls = append(sqls, `DELETE FROM t_exam_paper_group;`)

	sqls = append(sqls, `DELETE FROM t_user WHERE id < 1600;`)

	sqls = append(sqls, `DELETE FROM t_mark_info;`)

	sqls = append(sqls, `DELETE FROM t_paper_group;`)

	sqls = append(sqls, `DELETE FROM t_paper;`)

	sqls = append(sqls, `DELETE FROM t_exam_paper;`)

	sqls = append(sqls, `DELETE FROM t_exam_session;`)

	sqls = append(sqls, `DELETE FROM t_practice_submissions;`)

	sqls = append(sqls, `DELETE FROM t_practice_wrong_submissions;`)

	sqls = append(sqls, `DELETE FROM t_practice;`)

	sqls = append(sqls, `DELETE FROM t_exam_info;`)

	pgxConn := cmn.GetPgxConn()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		panic(err)
		return
	}

	defer func() {
		if err != nil {
			e := tx.Rollback(ctx)
			if e != nil {
				panic(e)
			}
		} else {
			e := tx.Commit(ctx)
			if e != nil {
				panic(e)
			}
		}
	}()

	for _, sql := range sqls {
		_, err = tx.Exec(ctx, sql)
		if err != nil {
			panic(err)
			return
		}
	}
}

// TODO
func TestQueryExams(t *testing.T) {
	cleanTestData()

	expectedExamSessions := []ExamSession{
		{
			Id:                   101,
			Name:                 "1",
			PaperName:            "test exam paper 1",
			ExamSessionType:      "",
			MarkMethod:           "02",
			MarkMode:             "00",
			RespondentCount:      2,
			UnMarkedStudentCount: 0,
			StartTime:            now.Add(-25 * time.Hour).UnixMilli(),
			EndTime:              now.Add(-24 * time.Hour).UnixMilli(),
			Status:               "00",
			MarkStatus:           "00",
		},
		{
			Id:                   102,
			Name:                 "2",
			PaperName:            "test exam paper 2",
			ExamSessionType:      "",
			MarkMethod:           "00",
			MarkMode:             "00",
			RespondentCount:      1,
			UnMarkedStudentCount: 0,
			StartTime:            now.Add(-23 * time.Hour).UnixMilli(),
			EndTime:              now.Add(-22 * time.Hour).UnixMilli(),
			Status:               "00",
			MarkStatus:           "00",
		},
		{
			Id:                   103,
			Name:                 "3",
			PaperName:            "test exam paper 3",
			ExamSessionType:      "",
			MarkMethod:           "00",
			MarkMode:             "10",
			RespondentCount:      4,
			UnMarkedStudentCount: 0,
			StartTime:            now.Add(-21 * time.Hour).UnixMilli(),
			EndTime:              now.Add(-20 * time.Hour).UnixMilli(),
			Status:               "00",
			MarkStatus:           "00",
		},
	}

	tests := []struct {
		name string

		req QueryMarkingListReq

		forceErr string

		expectedList     []Exam
		expectedRowCount int
		expectedErrStr   string
	}{
		{
			name: "success",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: expectedExamSessions,
			}},
			expectedRowCount: 3,
		},
		{
			name: "success: 查询第二页",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  2,
				Offset: 2,
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: []ExamSession{expectedExamSessions[2]},
			}},
			expectedRowCount: 3,
		},
		{
			name: "success: 根据 exam name 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:    10,
				Offset:   0,
				ExamName: "exam 1",
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: expectedExamSessions,
			}},
			expectedRowCount: 3,
		},
		{
			name: "success: 根据 开始时间、结束时间 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:     10,
				Offset:    0,
				StartTime: now.Add(-21 * time.Hour),
				EndTime:   now.Add(-20 * time.Hour),
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: []ExamSession{expectedExamSessions[2]}},
			},
			expectedRowCount: 1,
		},
		{
			name: "success: 根据 开始时间 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:     10,
				Offset:    0,
				StartTime: now.Add(-23 * time.Hour),
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: []ExamSession{expectedExamSessions[1], expectedExamSessions[2]},
			}},
			expectedRowCount: 2,
		},
		{
			name: "success: 根据 结束时间 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:   10,
				Offset:  0,
				EndTime: now.Add(-22 * time.Hour),
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: []ExamSession{expectedExamSessions[0], expectedExamSessions[1]},
			}},
			expectedRowCount: 2,
		},
		{
			name: "success: 根据 exam_session_id 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				ExamSessionID: 101,
			},
			expectedList: []Exam{{
				Id:           11,
				Name:         "test exam 1",
				Type:         "00",
				Class:        "",
				ExamSessions: []ExamSession{expectedExamSessions[0]},
			}},
			expectedRowCount: 1,
		},
		{
			name: "error: 缺少 user",
			req: QueryMarkingListReq{
				Limit:  10,
				Offset: 0,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: limit < 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  -3,
				Offset: 0,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: limit > 1000",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  1001,
				Offset: 0,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: offset < 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: -3,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: getExamCountQuery 执行失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "getExamCountQuery",
			expectedErrStr: "获取考试列表总数失败",
		},
		{
			name: "error: examListQuery 执行失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "examListQuery",
			expectedErrStr: "获取考试列表 sql 失败",
		},
		{
			name: "error: 行读取失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中获取值失败",
		},
		{
			name: "error: 行遍历失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			list, rowCount, err := QueryExams(ctx, tt.req)
			z.Sugar().Infof("-->(%d)list: %+v", rowCount, list)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Nil(t, list)
				assert.Equal(t, -1, rowCount)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
				assert.Equal(t, tt.expectedList, list)
				assert.Equal(t, tt.expectedRowCount, rowCount)
			}
		})
	}
}

func TestQueryPractices(t *testing.T) {
	cleanTestData()

	practices := []Practice{
		{
			ID:                   22,
			Name:                 "test practice 2",
			MarkMode:             "10",
			Type:                 "00",
			RespondentCount:      1,
			UnMarkedStudentCount: 1,
		},
		{
			ID:                   21,
			Name:                 "test practice 1",
			MarkMode:             "00",
			Type:                 "00",
			RespondentCount:      4,
			UnMarkedStudentCount: 4,
		},
	}

	tests := []struct {
		name string

		req QueryMarkingListReq

		forceErr string

		expectedList     []Practice
		expectedRowCount int
		expectedErrStr   string
	}{
		{
			name: "success",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			expectedList:     practices,
			expectedRowCount: 2,
		},
		//{
		//	name: "success: admin",
		//	req: QueryMarkingListReq{
		//		User: &User{
		//			ID:      testedTeacherID,
		//			IsAdmin: true,
		//		},
		//		Limit:  10,
		//		Offset: 0,
		//	},
		//	expectedList: []Practice{
		//		{
		//			ID:                   21,
		//			Name:                 "test practice 1",
		//			MarkMode:             "00",
		//			Type:                 "00",
		//			UnMarkedStudentCount: 1,
		//		},
		//		{
		//			ID:                   22,
		//			Name:                 "test practice 2",
		//			MarkMode:             "00",
		//			Type:                 "00",
		//			UnMarkedStudentCount: 1,
		//		},
		//	},
		//	expectedRowCount: 2,
		//},
		{
			name: "success: 根据 practice name 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:        10,
				Offset:       0,
				PracticeName: "practice 1",
			},
			expectedList:     []Practice{practices[1]},
			expectedRowCount: 1,
		},
		{
			name: "error: limit < 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  -3,
				Offset: 0,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: limit > 1000",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  1001,
				Offset: 0,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: offset < 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: -3,
			},
			expectedErrStr: "无效的 req",
		},
		{
			name: "error: listQuery 执行失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "listQuery",
			expectedErrStr: "获取所有练习数目 sql 失败",
		},
		{
			name: "error: rowCountQuery 执行失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "rowCountQuery",
			expectedErrStr: "查询练习列表失败",
		},
		{
			name: "error: 行读取失败",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "扫描解析练习数据失败",
		},
		{
			name: "error: 行遍历错误",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  10,
				Offset: 0,
			},
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历错误",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			list, rowCount, err := QueryPractices(ctx, tt.req)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Nil(t, list)
				assert.Equal(t, -1, rowCount)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
				assert.Equal(t, tt.expectedList, list)
				assert.Equal(t, tt.expectedRowCount, rowCount)
			}
		})
	}
}

//func TestQueryMarkingResults(t *testing.T) {
//	cleanTestData()
//
//	expectedMarkingResults := []cmn.TMark{
//		{
//			TeacherID:     null.IntFrom(testedTeacherID),
//			ExamSessionID: null.IntFrom(102),
//			ExamineeID:    null.IntFrom(2205),
//			QuestionID:    null.IntFrom(405),
//			MarkDetails:   types.JSONText("{}"),
//			Score:         null.FloatFrom(2),
//		},
//		{
//			TeacherID:            null.IntFrom(testedTeacherID),
//			PracticeID:           null.IntFrom(22),
//			PracticeSubmissionID: null.IntFrom(2404),
//			QuestionID:           null.IntFrom(427),
//			MarkDetails:          types.JSONText("{}"),
//			Score:                null.FloatFrom(2),
//		},
//	}
//
//	tests := []struct {
//		name string
//
//		cond      QueryCondition
//		teacherID int64
//
//		forceErr string
//
//		expectedMarkingResults []cmn.TMark
//		expectedErrStr         string
//	}{
//		{
//			name: "success: 考试场次下获取所有阅卷结果",
//			cond: QueryCondition{
//				ExamSessionID: 102,
//			},
//			expectedMarkingResults: []cmn.TMark{expectedMarkingResults[0]},
//		},
//		{
//			name: "success: 考试场次下获取单个考生阅卷结果",
//			cond: QueryCondition{
//				ExamSessionID: 102,
//				ExamineeID:    2205,
//			},
//			expectedMarkingResults: []cmn.TMark{expectedMarkingResults[0]},
//		},
//		{
//			name: "success: 练习下获取所有阅卷结果",
//			cond: QueryCondition{
//				PracticeID: 22,
//			},
//			expectedMarkingResults: []cmn.TMark{expectedMarkingResults[1]},
//		},
//		{
//			name: "success: 练习下获取单个提交的阅卷结果",
//			cond: QueryCondition{
//				PracticeID:           22,
//				PracticeSubmissionID: 2404,
//			},
//			expectedMarkingResults: []cmn.TMark{expectedMarkingResults[1]},
//		},
//		{
//			name: "error: SQL 查询执行失败",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			forceErr:       "getMarkingResultQuery",
//			expectedErrStr: "获取批改结果 sql 失败",
//		},
//		{
//			name: "error: 行扫描失败",
//			cond: QueryCondition{
//				ExamSessionID: 102,
//			},
//			forceErr:       "rows.Scan",
//			expectedErrStr: "从 row 读取结果失败",
//		},
//		{
//			name: "error: 行遍历出错",
//			cond: QueryCondition{
//				ExamSessionID: 103,
//			},
//			forceErr:       "rows.Err",
//			expectedErrStr: "行遍历错误",
//		},
//	}
//
//	initTestData()
//	defer cleanTestData()
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//
//			markingResults, err := QueryMarkingResults(ctx, tt.cond, tt.teacherID)
//			z.Sugar().Infof("expectedMarkingResults: %+v", markingResults)
//
//			if tt.expectedErrStr != "" {
//				assert.Error(t, err, "期待获取到错误，但却没有错误")
//				assert.Contains(t, err.Error(), tt.expectedErrStr)
//				assert.Nil(t, markingResults)
//			} else {
//				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
//				assert.Equal(t, tt.expectedMarkingResults, markingResults)
//			}
//		})
//	}
//}

func TestQueryMarkerInfo(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		cond QueryCondition

		forceErr string

		expectedMarkInfo MarkerInfo
		expectedErrStr   string
	}{
		{
			name: "success: 考试场次下获取所有批改配置信息",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			expectedMarkInfo: MarkerInfo{
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkTeacherID:      null.IntFrom(testedTeacherID),
						MarkQuestionGroups: types.JSONText{},
						MarkExamineeIds:    types.JSONText{},
						QuestionIds:        types.JSONText{},
					},
				},
				MarkMode:   "00",
				MarkMethod: "02",
			},
		},
		{
			name: "success: 考试场次下获取指定教师批改配置信息",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			expectedMarkInfo: MarkerInfo{
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkTeacherID:      null.IntFrom(testedTeacherID),
						MarkQuestionGroups: types.JSONText{},
						MarkExamineeIds:    types.JSONText{},
						QuestionIds:        types.JSONText{},
					},
				},
				MarkMode:   "00",
				MarkMethod: "02",
			},
		},
		{
			name: "success: 练习下获取批改配置信息",
			cond: QueryCondition{
				PracticeID: 22,
			},
			expectedMarkInfo: MarkerInfo{
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkTeacherID:      null.IntFrom(testedTeacherID),
						MarkQuestionGroups: types.JSONText{},
						MarkExamineeIds:    types.JSONText{},
						QuestionIds:        types.JSONText{},
					},
				},
				MarkMode: "10",
			},
		},
		{
			name: "success: 练习下获取指定教师批改配置信息",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 22,
			},
			expectedMarkInfo: MarkerInfo{
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkTeacherID:      null.IntFrom(testedTeacherID),
						MarkQuestionGroups: types.JSONText{},
						MarkExamineeIds:    types.JSONText{},
						QuestionIds:        types.JSONText{},
					},
				},
				MarkMode: "10",
			},
		},
		{
			name: "error: SQL 查询执行失败",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "query",
			expectedErrStr: "获取批改配置信息 sql 失败",
		},
		{
			name: "error: 行扫描失败",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 读取结果失败",
		},
		{
			name: "error: 行遍历失败",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 103,
			},
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			markerInfo, err := QueryMarkerInfo(ctx, tt.cond)
			z.Sugar().Infof("markerInfo: %+v", markerInfo)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Equal(t, MarkerInfo{}, markerInfo)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %+v", err)
				assert.Equal(t, tt.expectedMarkInfo, markerInfo)
			}
		})
	}
}

func TestQueryStudentAnswers(t *testing.T) {
	cleanTestData()

	// 公用 markerInfo
	baseMarkerInfo := MarkerInfo{
		MarkMode: "00",
	}

	// 0到1：考试客观题 3到4：考试主观题 5到7：练习客观题
	studentAnswers := []cmn.TVStudentAnswerQuestion{
		{
			ExamSessionID: null.IntFrom(101),
			ExamineeID:    null.IntFrom(2201),
			QuestionID:    null.IntFrom(401),
			Type:          null.StringFrom("00"),
			QuestionScore: null.FloatFrom(3),
			Answer:        types.JSONText(`{"answer": ["C"]}`),
			ActualAnswers: types.JSONText(`["C"]`),
			ActualOptions: types.JSONText{},
		},
		{
			ExamSessionID: null.IntFrom(101),
			ExamineeID:    null.IntFrom(2201),
			QuestionID:    null.IntFrom(402),
			Type:          null.StringFrom("02"),
			QuestionScore: null.FloatFrom(3),
			Answer:        types.JSONText(`{"answer": ["B", "D"]}`),
			ActualAnswers: types.JSONText(`["B", "D"]`),
			ActualOptions: types.JSONText{},
		},
		{
			ExamSessionID: null.IntFrom(101),
			ExamineeID:    null.IntFrom(2201),
			QuestionID:    null.IntFrom(403),
			Type:          null.StringFrom("04"),
			QuestionScore: null.FloatFrom(3),
			Answer:        types.JSONText(`{"answer": ["A"]}`),
			ActualAnswers: types.JSONText(`["A"]`),
			ActualOptions: types.JSONText{},
		},
		{
			ExamSessionID: null.IntFrom(102),
			ExamineeID:    null.IntFrom(2205),
			QuestionID:    null.IntFrom(405),
			Type:          null.StringFrom("06"),
			QuestionScore: null.FloatFrom(3),
			Answer:        types.JSONText(`{"answer": ["填空学生作答1"]}`),
			ActualAnswers: types.JSONText(`[]`),
			ActualOptions: types.JSONText{},
		},
		{
			ExamSessionID: null.IntFrom(102),
			ExamineeID:    null.IntFrom(2205),
			QuestionID:    null.IntFrom(406),
			Type:          null.StringFrom("08"),
			QuestionScore: null.FloatFrom(10),
			Answer:        types.JSONText(`{"answer": ["简答1学生作答1"]}`),
			ActualAnswers: types.JSONText(`[]`),
			ActualOptions: types.JSONText{},
		},
		{
			PracticeID:           null.IntFrom(21),
			PracticeSubmissionID: null.IntFrom(2401),
			QuestionID:           null.IntFrom(424),
			Type:                 null.StringFrom("00"),
			QuestionScore:        null.FloatFrom(3),
			Answer:               types.JSONText(`{"answer": ["C"]}`),
			ActualAnswers:        types.JSONText(`["C"]`),
			ActualOptions:        types.JSONText{},
		},
		{
			PracticeID:           null.IntFrom(21),
			PracticeSubmissionID: null.IntFrom(2401),
			QuestionID:           null.IntFrom(425),
			Type:                 null.StringFrom("02"),
			QuestionScore:        null.FloatFrom(3),
			Answer:               types.JSONText(`{"answer": ["B", "D"]}`),
			ActualAnswers:        types.JSONText(`["B", "D"]`),
			ActualOptions:        types.JSONText{},
		},
		{
			PracticeID:           null.IntFrom(21),
			PracticeSubmissionID: null.IntFrom(2401),
			QuestionID:           null.IntFrom(426),
			Type:                 null.StringFrom("04"),
			QuestionScore:        null.FloatFrom(3),
			Answer:               types.JSONText(`{"answer": ["A"]}`),
			ActualAnswers:        types.JSONText(`["A"]`),
			ActualOptions:        types.JSONText{},
		},
		{
			PracticeID:           null.IntFrom(22),
			PracticeSubmissionID: null.IntFrom(2404),
			QuestionID:           null.IntFrom(427),
			Type:                 null.StringFrom("06"),
			QuestionScore:        null.FloatFrom(3),
			Answer:               types.JSONText(`{"answer": ["填空学生作答1"]}`),
			ActualAnswers:        types.JSONText(`[]`),
			ActualOptions:        types.JSONText{},
		},
	}

	tests := []struct {
		name string

		tx           pgx.Tx
		questionType string
		cond         QueryCondition
		markerInfo   MarkerInfo

		forceErr string

		expectedStudentAnswers []cmn.TVStudentAnswerQuestion
		expectedErrStr         string
	}{
		{
			name:         "success: 考试-客观题",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2201,
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{
				studentAnswers[0], studentAnswers[1], studentAnswers[2],
			},
		},
		{
			name:         "success: 考试-主观题",
			questionType: "02",
			cond: QueryCondition{
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{
				studentAnswers[3], studentAnswers[4],
			},
		},
		{
			name:         "success: 考试-主观题-题组专评",
			questionType: "02",
			cond: QueryCondition{
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText("[405]"),
					},
				},
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{studentAnswers[3]},
		},
		{
			name:         "success: 考试-主观题-题目专评",
			questionType: "02",
			cond: QueryCondition{
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
			markerInfo: MarkerInfo{
				MarkMode: "08",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText("[405]"),
					},
				},
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{studentAnswers[3]},
		},
		{
			name:         "success: 练习-客观题",
			questionType: "00",
			cond: QueryCondition{
				PracticeID:           21,
				PracticeSubmissionID: 2401,
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{
				studentAnswers[5], studentAnswers[6], studentAnswers[7],
			},
		},
		{
			name:         "success: 练习-主观题",
			questionType: "02",
			cond: QueryCondition{
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
			expectedStudentAnswers: []cmn.TVStudentAnswerQuestion{studentAnswers[8]},
		},
		{
			name:         "error: 无效的题目类型",
			questionType: "99",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     baseMarkerInfo,
			expectedErrStr: "无效的题目类型",
		},
		{
			name:         "error: 无效的 考生ID",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     baseMarkerInfo,
			expectedErrStr: "无效的 考生ID",
		},
		{
			name:         "error: 无效的 练习提交ID",
			questionType: "00",
			cond: QueryCondition{
				PracticeID: 22,
			},
			markerInfo:     baseMarkerInfo,
			expectedErrStr: "无效的 练习提交ID",
		},
		{
			name:         "error: 题组专评-批改配置信息错误",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     MarkerInfo{MarkMode: "06"},
			expectedErrStr: "批改配置信息错误",
		},
		{
			name:         "error: 题目专评-批改配置信息错误",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     MarkerInfo{MarkMode: "08"},
			expectedErrStr: "批改配置信息错误",
		},
		{
			name:         "error: 题目专评-json 解析失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo: MarkerInfo{
				MarkMode: "08",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText("invalid json"),
					},
				},
			},
			expectedErrStr: "解析 markerInfo.MarkInfos[0].QuestionIds 失败",
		},
		// TODO 练习错题
		{
			name:         "error: SQL 执行失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     baseMarkerInfo,
			forceErr:       "query",
			expectedErrStr: "获取学生答案 sql 错误",
		},
		{
			name:         "error: 行扫描失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     baseMarkerInfo,
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name:         "error: 行遍历失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo:     baseMarkerInfo,
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			answers, err := QueryStudentAnswers(ctx, nil, tt.questionType, tt.cond, tt.markerInfo)
			z.Sugar().Infof("answers: %+v", answers)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Nil(t, answers)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %+v", err)
				assert.ElementsMatch(t, tt.expectedStudentAnswers, answers)
			}
		})
	}
}

func TestQuerySubjectiveQuestions(t *testing.T) {
	cleanTestData()

	expectedQuestionSets := []QuestionSet{
		{
			TExamPaperGroup: cmn.TExamPaperGroup{
				ID:   null.IntFrom(302),
				Name: null.StringFrom("test group 2"),
			},
			Score: float64(3 + 10 + 3), // 405 + 406 + 422
			Questions: []*cmn.TExamPaperQuestion{
				{ID: null.IntFrom(405), Score: null.FloatFrom(3), Type: null.StringFrom("06"), GroupID: null.IntFrom(302), Content: null.StringFrom("填空1"), Options: types.JSONText{}, Answers: types.JSONText(`[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`)},
				{ID: null.IntFrom(406), Score: null.FloatFrom(10), Type: null.StringFrom("08"), GroupID: null.IntFrom(302), Content: null.StringFrom("简答题1"), Options: types.JSONText{}, Answers: types.JSONText(`[{"index": 1, "score": 10, "answer": "简答题答案1", "grading_rule": "1", "alternative_answer": null}]`)},
				{ID: null.IntFrom(422), Score: null.FloatFrom(3), Type: null.StringFrom("06"), GroupID: null.IntFrom(302), Content: null.StringFrom("填空2"), Options: types.JSONText{}, Answers: types.JSONText(`[{"index": 1, "score": 1, "answer": "填空1第一空答案", "grading_rule": "1"}, {"index": 2, "score": 2, "answer": "填空2第二空答案", "grading_rule": "2"}]`)},
			},
		},
		{
			TExamPaperGroup: cmn.TExamPaperGroup{
				ID:   null.IntFrom(302),
				Name: null.StringFrom("test group 2"),
			},
			Score: float64(3),
			Questions: []*cmn.TExamPaperQuestion{
				{ID: null.IntFrom(405), Score: null.FloatFrom(3), Type: null.StringFrom("06"), GroupID: null.IntFrom(302), Content: null.StringFrom("填空1"), Options: types.JSONText{}, Answers: types.JSONText(`[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`)},
			},
		},
		{
			TExamPaperGroup: cmn.TExamPaperGroup{
				ID:   null.IntFrom(325),
				Name: null.StringFrom("test group 25"),
			},
			Score: float64(3),
			Questions: []*cmn.TExamPaperQuestion{
				{ID: null.IntFrom(427), Score: null.FloatFrom(3), Type: null.StringFrom("06"), GroupID: null.IntFrom(325), Content: null.StringFrom("练习-填空1"), Options: types.JSONText{}, Answers: types.JSONText(`[{"index": 1, "score": 3, "answer": "填空答案1", "grading_rule": "1", "alternative_answer": null}]`)},
			},
		},
	}

	tests := []struct {
		name string

		tx         pgx.Tx
		types      []string
		cond       QueryCondition
		markerInfo MarkerInfo

		forceErr string

		expectedQuestionSets []QuestionSet
		expectedErrStr       string
	}{
		{
			name:  "success: 考试",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[0]},
		},
		{
			name:  "success: 考试-题组专评",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06", // 题组专评
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405]`),
					},
				},
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[1]},
		},
		{
			name:  "success: 考试-题目专评",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkMode: "08", // 题目专评
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405]`),
					},
				},
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[1]},
		},
		{
			name:  "success: 练习",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				PracticeID: 22,
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[2]},
		},
		{
			name:  "success: 练习-存在题目专评-但是仅根据PracticeID查询",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				PracticeID: 22,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06", // 题组专评
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: types.JSONText(`[405]`),
					},
				},
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[2]},
		},
		{
			name:  "success: 查询单道题目",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				QuestionID: 405,
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[1]},
		},
		{
			name:  "success: 查询单道题目-同时存在考试场次ID-仅根据QuestionID",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				ExamSessionID: 102,
				QuestionID:    427,
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[2]},
		},
		{
			name:  "success: 查询单道题目-同时存在练习ID-仅根据QuestionID",
			types: []string{QuestionTypeFillBlank, QuestionTypeShortAnswer, QuestionTypeApplication},
			cond: QueryCondition{
				PracticeID: 22,
				QuestionID: 405,
			},
			expectedQuestionSets: []QuestionSet{expectedQuestionSets[1]},
		},
		{
			name:  "error: 缺少题目类型",
			types: []string{},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			expectedErrStr: "缺少题目类型",
		},
		{
			name:  "error: 批改配置信息错误",
			types: []string{QuestionTypeFillBlank},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkMode:  "08", // 题目专评
				MarkInfos: []cmn.TMarkInfo{},
			},
			expectedErrStr: "批改配置信息错误",
		},
		{
			name:  "error: 无效的 markerInfo QuestionIds",
			types: []string{QuestionTypeFillBlank},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: []byte("invalid json"),
					},
				},
			},
			expectedErrStr: "解析 markerInfo.MarkInfos[0].QuestionIds 失败",
		},
		{
			name:  "error: SQL 执行失败",
			types: []string{QuestionTypeFillBlank},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			forceErr:       "getQuestionsQuery",
			expectedErrStr: "获取主观题 sql 失败",
		},
		{
			name:  "error: 行扫描失败",
			types: []string{QuestionTypeFillBlank},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name:  "error: 行遍历失败",
			types: []string{QuestionTypeFillBlank},
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			questionSets, err := QuerySubjectiveQuestions(ctx, nil, tt.types, tt.cond, tt.markerInfo)
			z.Sugar().Infof("questionSets: %+v", questionSets)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Nil(t, questionSets)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
				assert.Equal(t, tt.expectedQuestionSets, questionSets)
			}
		})
	}
}

func TestQueryExamSessionStatus(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		examSessionID int64

		forceErr string

		expectedExamSessionStatus string
		expectedErrStr            string
	}{
		{
			name:                      "success: 成功获取考试状态",
			examSessionID:             101,
			expectedExamSessionStatus: "00",
		},
		{
			name:           "error: 无效的 考试场次ID",
			examSessionID:  -1,
			expectedErrStr: "无效的 考试场次ID",
		},
		{
			name:           "error: 不存在的考试场次",
			examSessionID:  9999,
			expectedErrStr: "查询不到考试场次",
		},
		{
			name:           "error: SQL 执行失败",
			examSessionID:  101,
			forceErr:       "queryExamSessionSql",
			expectedErrStr: "执行 queryExamSessionSql 失败",
		},
		{
			name:           "error: 强制错误 ForceErr",
			forceErr:       "QueryExamSession",
			expectedErrStr: ForceErr.Error(),
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					pgConn := cmn.GetPgxConn()

					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}
					defer func() {
						_ = tx.Rollback(context.Background())
					}()
				}
				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

				examSessionStatus, err := QueryExamSessionStatus(ctx, tx, tt.examSessionID)
				z.Sugar().Infof("examSessionID=%d, examSessionStatus=%s", tt.examSessionID, examSessionStatus)

				if tt.expectedErrStr != "" {
					assert.Error(t, err, "期待获取到错误，但却没有错误")
					assert.Contains(t, err.Error(), tt.expectedErrStr)
					assert.Equal(t, ExamSession{}, examSessionStatus)
				} else {
					assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
					assert.Equal(t, tt.expectedExamSessionStatus, examSessionStatus)
				}
			})
		}
	}
}

//func TestQuerySubmittedStudents(t *testing.T) {
//	cleanTestData()
//
//	// 0到1是考试，2到5是练习
//	expectedStudentInfos := []StudentInfo{
//		{
//			OfficialName: null.StringFrom("student1"),
//			ExamineeID:   null.IntFrom(2201),
//			SerialNumber: null.IntFrom(1),
//		},
//		{
//			OfficialName: null.StringFrom("student2"),
//			ExamineeID:   null.IntFrom(2202),
//			SerialNumber: null.IntFrom(2),
//		},
//		{
//			OfficialName:         null.StringFrom("student1"),
//			PracticeSubmissionID: null.IntFrom(2401),
//		},
//		{
//			OfficialName:         null.StringFrom("student2"),
//			PracticeSubmissionID: null.IntFrom(2402),
//		},
//		{
//			OfficialName:         null.StringFrom("student3"),
//			PracticeSubmissionID: null.IntFrom(2403),
//		},
//		{
//			OfficialName:         null.StringFrom("student4"),
//			PracticeSubmissionID: null.IntFrom(2406),
//		},
//	}
//
//	tests := []struct {
//		name string
//
//		cond       QueryCondition
//		markerInfo MarkerInfo
//
//		forceErr string
//
//		expectedStudentInfos []StudentInfo
//		expectedErrStr       string
//	}{
//		{
//			name: "success: 考试-获取所有学生信息",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			expectedStudentInfos: []StudentInfo{expectedStudentInfos[0], expectedStudentInfos[1]},
//		},
//		{
//			name: "success: 考试-单个考生",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//			},
//			expectedStudentInfos: []StudentInfo{expectedStudentInfos[0]},
//		},
//		{
//			name: "success: 考试-试卷分配",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			markerInfo: MarkerInfo{
//				MarkMode: "04",
//				MarkInfos: []cmn.TMarkInfo{
//					{
//						MarkExamineeIds: types.JSONText(`[2202]`),
//					},
//				},
//			},
//			expectedStudentInfos: []StudentInfo{expectedStudentInfos[1]},
//		},
//		{
//			name: "success: 练习-获取所有待批改学生",
//			cond: QueryCondition{
//				PracticeID: 21,
//			},
//			expectedStudentInfos: []StudentInfo{expectedStudentInfos[2], expectedStudentInfos[3], expectedStudentInfos[4], expectedStudentInfos[5]},
//		},
//		{
//			name: "success: 练习-单个提交",
//			cond: QueryCondition{
//				PracticeID:           21,
//				PracticeSubmissionID: 2401,
//			},
//			expectedStudentInfos: []StudentInfo{expectedStudentInfos[2]},
//		},
//		{
//			name: "error: 考生ID 和 试卷分配模式 不能同时存在",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//				ExamineeID:    2201,
//			},
//			markerInfo:     MarkerInfo{MarkMode: "04"},
//			expectedErrStr: "考生ID 和 试卷分配模式 不能同时存在",
//		},
//		{
//			name: "error: 试卷分配，批改配置信息错误",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			markerInfo:     MarkerInfo{MarkMode: "04"},
//			expectedErrStr: "批改配置信息错误",
//		},
//		{
//			name: "error: 强制 SQL 错误",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			forceErr:       "query",
//			expectedErrStr: "查询学生信息 sql 失败",
//		},
//		{
//			name: "error: 行扫描失败",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			forceErr:       "rows.Scan",
//			expectedErrStr: "从 row 中读取结果失败",
//		},
//		{
//			name: "error: 行遍历失败",
//			cond: QueryCondition{
//				ExamSessionID: 101,
//			},
//			forceErr:       "rows.Err",
//			expectedErrStr: "行遍历错误",
//		},
//	}
//
//	initTestData()
//	defer cleanTestData()
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
//
//			studentInfos, err := QueryStudents(ctx, tt.cond, tt.markerInfo)
//			z.Sugar().Infof("expectedStudentInfos: %+v", studentInfos)
//
//			if tt.expectedErrStr != "" {
//				assert.Error(t, err, "期待获取到错误，但却没有错误")
//				assert.Contains(t, err.Error(), tt.expectedErrStr)
//				assert.Equal(t, 0, len(studentInfos))
//			} else {
//				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
//				assert.Equal(t, tt.expectedStudentInfos, studentInfos)
//			}
//		})
//	}
//}

func TestUpdateMarkerInfoStatus(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		teacherID int64
		ids       []int64
		mode      string
		status    string

		forceErr string

		expectedTargetIDs []int64
		expectedErrStr    string
	}{
		{
			name:      "success: 考试",
			teacherID: testedTeacherID,
			ids:       []int64{101},
			mode:      "00",
			status:    "00",
		},
		{
			name:      "success: 练习",
			teacherID: testedTeacherID,
			ids:       []int64{201},
			mode:      "02",
			status:    "02",
		},
		{
			name:           "error: teacherID无效",
			teacherID:      0,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			expectedErrStr: "无效的 teacher ID",
		},
		{
			name:           "error: ids为空",
			teacherID:      testedTeacherID,
			ids:            nil,
			mode:           "00",
			status:         "00",
			expectedErrStr: "没有 exam_session_ids",
		},
		{
			name:           "error: status无效",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "99",
			expectedErrStr: "status 错误",
		},
		{
			name:           "error: mode无效",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "99",
			status:         "00",
			expectedErrStr: "无效的模式",
		},
		{
			name:           "error: 强制错误 updateQuery",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			forceErr:       "updateQuery",
			expectedErrStr: "更新批改配置信息 sql 失败",
		},
		{
			name:           "error: 强制错误 rows.Scan",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name:           "error: 强制错误 rows.Err",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历错误",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					pgConn := cmn.GetPgxConn()
					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}

					// 测试，回滚即可
					defer func() {
						_ = tx.Rollback(context.Background())
					}()
				}

				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

				ids, err := UpdateMarkerInfoStatus(ctx, tx, tt.teacherID, tt.ids, tt.mode, tt.status)

				if tt.expectedErrStr != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErrStr)
					assert.Equal(t, 0, len(tt.expectedTargetIDs))
				} else {
					assert.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedTargetIDs, ids) // 比较更新的行
					for _, id := range ids {
						assert.Greater(t, id, int64(0), "返回的 ID 应大于 0")
					}
				}
			})
		}
	}
}

func TestUpsertMarkingResults(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		markingResults []cmn.TMark

		forceErr string

		expectedTargetIDs []int64
		expectedErrStr    string
	}{
		{
			name: "success: 考试",
			markingResults: []cmn.TMark{
				{
					ExamineeID:    null.IntFrom(1001),
					QuestionID:    null.IntFrom(501),
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(101),
					MarkDetails:   types.JSONText(`[{"index":1,"score":2}]`),
					Score:         null.FloatFrom(2.0),
					Creator:       null.IntFrom(testedTeacherID),
					Status:        null.StringFrom("00"),
					UpdatedBy:     null.IntFrom(testedTeacherID),
					UpdateTime:    null.IntFrom(time.Now().UnixMilli()),
				},
			},
		},
		{
			name: "success: 练习",
			markingResults: []cmn.TMark{
				{
					PracticeID:           null.IntFrom(22),
					PracticeSubmissionID: null.IntFrom(2401),
					QuestionID:           null.IntFrom(502),
					TeacherID:            null.IntFrom(testedTeacherID),
					MarkDetails:          types.JSONText(`[{"index":1,"score":3}]`),
					Score:                null.FloatFrom(3.0),
					Creator:              null.IntFrom(testedTeacherID),
					Status:               null.StringFrom("00"),
					UpdatedBy:            null.IntFrom(testedTeacherID),
					UpdateTime:           null.IntFrom(time.Now().UnixMilli()),
				},
			},
		},
		{
			name: "error: questionID 缺失",
			markingResults: []cmn.TMark{
				{
					ExamineeID:    null.IntFrom(1001),
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(101),
					MarkDetails:   types.JSONText(`[{"index":1,"score":2}]`),
					Score:         null.FloatFrom(2.0),
				},
			},
			expectedErrStr: "questionID 用于记录批改过程是必需的",
		},
		{
			name: "error: 参数非法",
			markingResults: []cmn.TMark{
				{
					QuestionID: null.IntFrom(501),
					TeacherID:  null.IntFrom(testedTeacherID),
				},
			},
			expectedErrStr: "无效的插入参数",
		},
		{
			name: "forceErr: UpsertMarkingResults",
			markingResults: []cmn.TMark{
				{
					ExamineeID:    null.IntFrom(1001),
					QuestionID:    null.IntFrom(501),
					TeacherID:     null.IntFrom(testedTeacherID),
					ExamSessionID: null.IntFrom(101),
					MarkDetails:   types.JSONText(`[{"index":1,"score":2}]`),
					Score:         null.FloatFrom(2.0),
				},
			},
			forceErr:       "UpsertMarkingResults",
			expectedErrStr: ForceErr.Error(),
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					pgConn := cmn.GetPgxConn()
					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}
					defer func() {
						_ = tx.Rollback(context.Background())
					}()
				}

				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

				ids, err := UpsertMarkingResults(ctx, tx, tt.markingResults)

				if tt.expectedErrStr != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErrStr)
					assert.Equal(t, 0, len(tt.expectedTargetIDs))
				} else {
					assert.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedTargetIDs, ids)
					for _, id := range ids {
						assert.Greater(t, id, int64(0), "返回的 ID 应大于 0")
					}
				}
			})
		}
	}
}

func TestQueryMarkedPersonAndQuestionCount(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		cond QueryCondition

		forceErr string

		expectedPerson int64
		expectedCount  int64
		expectedErrStr string
	}{
		{
			name:           "success:考试",
			cond:           QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 102},
			expectedPerson: 0,
			expectedCount:  1,
		},
		{
			name:           "success:练习",
			cond:           QueryCondition{PracticeID: 22}, // 该练习只有一人一题，且已批改
			expectedPerson: 0,                              // 但是练习需要提交，人数才会-1
			expectedCount:  1,
		},
		{
			name:           "error: 获取已批改人数 sql 失败",
			cond:           QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 102},
			forceErr:       "queryMarkedPerson",
			expectedErrStr: "获取已批改人数 sql 失败",
		},
		{
			name:           "error: 获取已批改问题数 sql 失败",
			cond:           QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 102},
			forceErr:       "queryQuestionCount",
			expectedErrStr: "获取已批改问题数 sql 失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			person, count, err := QueryMarkedPersonAndQuestionCount(ctx, nil, tt.cond)

			if tt.expectedErrStr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Equal(t, int64(-1), person)
				assert.Equal(t, int64(-1), count)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPerson, person, "期待已批改人数错误")
				assert.Equal(t, tt.expectedCount, count, "期待已批改问题数错误")
			}
		})
	}
}

func TestUpdateStudentAnswerScore(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		markingResults []cmn.TMark
		cond           QueryCondition

		forceErr string

		expectedTargetIDs []int64
		expectedErrStr    string
	}{
		{
			name: "成功: 考试更新分数",
			markingResults: []cmn.TMark{
				{ExamineeID: null.IntFrom(101), QuestionID: null.IntFrom(201), Score: null.FloatFrom(5)},
			},
			cond: QueryCondition{TeacherID: 1, ExamSessionID: 1001},
		},
		{
			name: "成功: 练习更新分数",
			markingResults: []cmn.TMark{
				{PracticeSubmissionID: null.IntFrom(301), QuestionID: null.IntFrom(201), Score: null.FloatFrom(5)},
			},
			cond: QueryCondition{TeacherID: 1, PracticeID: 2001},
		},
		{
			name:           "错误: 没有批改结果",
			markingResults: nil,
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 1001},
			expectedErrStr: "没有批改结果可用于更新",
		},
		{
			name: "错误: 缺少教师ID",
			markingResults: []cmn.TMark{
				{ExamineeID: null.IntFrom(101), QuestionID: null.IntFrom(201), Score: null.FloatFrom(5)},
			},
			cond:           QueryCondition{TeacherID: 0, ExamSessionID: 1001},
			expectedErrStr: "缺少 教师ID",
		},
		{
			name: "错误: 无效的 exam_session_id 和 practice_id",
			markingResults: []cmn.TMark{
				{ExamineeID: null.IntFrom(101), QuestionID: null.IntFrom(201), Score: null.FloatFrom(5)},
			},
			cond:           QueryCondition{TeacherID: 1},
			expectedErrStr: "无效的 exam_session_id 和 practice_id",
		},
		{
			name: "forceErr: updateQuery",
			markingResults: []cmn.TMark{
				{ExamineeID: null.IntFrom(101), QuestionID: null.IntFrom(201), Score: null.FloatFrom(5)},
			},
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 1001},
			forceErr:       "updateQuery",
			expectedErrStr: ForceErr.Error(),
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					pgConn := cmn.GetPgxConn()
					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}
					defer func() {
						_ = tx.Rollback(context.Background())
					}()
				}

				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

				ids, err := UpdateStudentAnswerScore(ctx, nil, tt.markingResults, tt.cond)

				if tt.expectedErrStr != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErrStr)
					assert.Equal(t, 0, len(tt.expectedTargetIDs))
				} else {
					assert.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedTargetIDs, ids)
					for _, id := range ids {
						assert.Greater(t, id, int64(0), "返回的 ID 应大于 0")
					}
				}
			})
		}
	}
}

func TestUpdateExamSessionOrPracticeSubmissionStatus(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name string

		cond   QueryCondition
		status string

		forceErr string

		expectedErrStr string
	}{
		{
			name:   "success: 更新考试状态",
			cond:   QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 101},
			status: "04",
		},
		{
			name:   "success: 更新练习状态",
			cond:   QueryCondition{TeacherID: testedTeacherID, PracticeSubmissionID: 2401},
			status: "04",
		},
		{
			name:   "success: 更新练习错题状态",
			cond:   QueryCondition{TeacherID: testedTeacherID, PracticeWrongSubmissionID: 2601},
			status: "04",
		},
		{
			name:           "error: teacherID 无效",
			cond:           QueryCondition{TeacherID: 0, ExamSessionID: 101},
			status:         "04",
			expectedErrStr: "无效的 teacher_id",
		},
		{
			name:           "error: 缺少 必要ID",
			cond:           QueryCondition{TeacherID: testedTeacherID},
			status:         "04",
			expectedErrStr: "缺少 考试场次ID/练习提交ID/练习错题提交ID",
		},
		{
			name:           "error: status 为空",
			cond:           QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 101},
			status:         "",
			expectedErrStr: "无效的 考试/练习 状态",
		},
		{
			name:           "error: SQL 执行失败",
			cond:           QueryCondition{TeacherID: testedTeacherID, ExamSessionID: 101},
			status:         "04",
			forceErr:       "updateQuery",
			expectedErrStr: "更新 考试/练习 状态 sql 失败",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					pgConn := cmn.GetPgxConn()
					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}
					defer func() {
						_ = tx.Rollback(context.Background())
					}()
				}

				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

				id, err := UpdateExamSessionOrPracticeSubmissionStatus(ctx, tx, tt.cond, tt.status)

				if tt.expectedErrStr != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErrStr)
					assert.Equal(t, int64(-1), id)
				} else {
					assert.NoError(t, err)
					assert.Greater(t, id, int64(0), "返回的 ID 应大于 0")
				}
			})
		}
	}
}
