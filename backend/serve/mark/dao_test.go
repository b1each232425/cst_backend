package mark

import (
	"context"
	"database/sql"
	"encoding/json"
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

func init() {
	cmn.ConfigureForTest()
	z = cmn.GetLogger()
}

func initTestData() {
	var sqls []string
	var args [][][]interface{}

	sqls = append(sqls, `INSERT INTO t_user(id, account, category) VALUES($1, $2, $3)`)
	args = append(args, [][]interface{}{
		{1101, "test account 1", "test user"},
		{1102, "test account 2", "test user"},
		{1103, "test account 3", "test user"},
		{1201, "test account 4", "test student"},
		{1202, "test account 5", "test student"},
		{1203, "test account 6", "test student"},
		{1204, "test account 7", "test student"},
	})

	sqls = append(sqls, `INSERT INTO t_exam_info(id, name, type, creator, create_time, status) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{11, "test exam 1", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{12, "test exam 2", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
		{13, "test exam 3", "00", 1101, time.Now().Add(-24 * 7 * time.Hour).UnixMilli(), "00"},
	})

	sqls = append(sqls, `INSERT INTO t_exam_session(id, exam_id, paper_id, mark_method, mark_mode, start_time, end_time, session_num, status, creator) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`)
	now := time.Now()
	args = append(args, [][]interface{}{
		{101, 11, 101, "02", "00", now.Add(-25 * time.Hour).UnixMilli(), now.Add(-24 * time.Hour).UnixMilli(), 1, "00", 1101}, // 纯理论
		{102, 11, 102, "00", "00", now.Add(-23 * time.Hour).UnixMilli(), now.Add(-22 * time.Hour).UnixMilli(), 2, "00", 1101}, // 理论+主观
		{103, 11, 103, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 题目答案缺失
		{104, 11, 104, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
		{105, 11, 105, "00", "", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101}, // 批改模式缺失
		{106, 11, 106, "00", "10", now.Add(-21 * time.Hour).UnixMilli(), now.Add(-20 * time.Hour).UnixMilli(), 3, "00", 1101},
	})

	sqls = append(sqls, `INSERT INTO t_practice(id, name, correct_mode, type, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	args = append(args, [][]interface{}{
		{21, "test practice 1", "00", "00", "02", 1101},
		{22, "test practice 2", "10", "00", "02", 1101},
		{23, "test practice 3", "00", "00", "02", 1101},
		{24, "test practice 4", "00", "00", "02", 1101}, // 主观 AI 批改
	})

	sqls = append(sqls, `INSERT INTO t_paper(id, exampaper_id, domain_id, creator) VALUES($1, $2, $3, $4)`)
	args = append(args, [][]interface{}{
		{101, 101, 1999, 1101},
		{102, 102, 1999, 1101},
		{103, 103, 1999, 1101},
		{104, 104, 1999, 1101},
		{124, 124, 1999, 1101},
		{125, 125, 1999, 1101},
		{126, 126, 1999, 1101},
		{127, 127, 1999, 1101},
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

	sqls = append(sqls, `INSERT INTO t_examinee(id, student_id, exam_session_id, status, creator) VALUES($1, $2, $3, $4, $5)`)
	args = append(args, [][]interface{}{
		{2201, 1201, 101, "10", 1101},
		{2202, 1202, 101, "10", 1101},
		{2203, 1203, 103, "10", 1101},
		{2204, 1204, 103, "10", 1101},
		{2205, 1201, 102, "10", 1101},
		{2206, 1201, 103, "10", 1101},
		{2207, 1202, 103, "10", 1101},
	})

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

	// 考试作答
	//sqls = append(sqls, `INSERT INTO t_student_answers(id, examinee_id, question_id, actual_answers, status, creator) VALUES($1, $2, $3, $4, $5, $6)`)
	//args = append(args, [][]interface{}{
	//	{33, 2205, 406, `[]`, "00", 1101},
	//})

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

	for i, query := range sqls {
		var ids []interface{}

		for _, arg := range args[i] {
			ids = append(ids, arg[0])

			_, err = tx.Exec(query, arg...)
			if err != nil {
				err = fmt.Errorf("insert test data SQL error: %v", err)
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

// 辅助函数，快速生成 JSON []byte
func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func TestQueryExamList(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		req              QueryMarkingListReq
		forceErr         string
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
			expectedList:     []Exam{},
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
				Id:    11,
				Name:  "test exam 1",
				Type:  "00",
				Class: "",
				ExamSessions: []ExamSession{{
					Id:                   101,
					Name:                 "1",
					PaperName:            "test exam paper 1",
					ExamSessionType:      "",
					MarkMethod:           "02",
					MarkMode:             "00",
					RespondentCount:      2,
					UnMarkedStudentCount: 0,
					StartTime:            1758896281126,
					EndTime:              1758899881126,
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
						UnMarkedStudentCount: 1,
						StartTime:            1758903481126,
						EndTime:              1758907081126,
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
						StartTime:            1758910681126,
						EndTime:              1758914281126,
						Status:               "00",
						MarkStatus:           "00",
					},
				},
			}},
			expectedRowCount: 3,
		},
		{
			name: "success: 根据 时间 查询",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:     10,
				Offset:    0,
				StartTime: time.Now().Add(-22 * time.Hour),
				EndTime:   time.Now().Add(-20 * time.Hour),
			},
			expectedList:     []Exam{},
			expectedRowCount: 3,
		},
		{
			name: "error: 缺少 user",
			req: QueryMarkingListReq{
				Limit:  10,
				Offset: 0,
			},
			expectedErrStr: "无效的 query cond",
		},
		{
			name: "error: limit == 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  0,
				Offset: 0,
			},
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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

			list, rowCount, err := QueryExamList(ctx, tt.req)
			z.Sugar().Infof("-->(%d)list: %+v", rowCount, list)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
				assert.Nilf(t, list, "期待获取 nil， 实际却获取 %v", list)
				assert.Equalf(t, rowCount, -1, "期待得到 -1， 实际却是 %v", rowCount)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
				//assert.Equalf(t, list, tt.expectedList, "期待获取 %v, 实际获取 %v", tt.expectedList, list)
				assert.Equalf(t, rowCount, tt.expectedRowCount, "期待获取 %v, 实际获取 %v", tt.expectedRowCount, rowCount)
			}
		})
	}
}

func TestQueryPracticeList(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name             string
		req              QueryMarkingListReq
		forceErr         string
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
		},
		{
			name: "success: admin",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: true,
				},
				Limit:  10,
				Offset: 0,
			},
		},
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
		},
		{
			name: "error: limit == 0",
			req: QueryMarkingListReq{
				User: &User{
					ID:      testedTeacherID,
					IsAdmin: false,
				},
				Limit:  0,
				Offset: 0,
			},
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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
			expectedErrStr: "无效的 query cond",
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

			list, rowCount, err := QueryPracticeList(ctx, tt.req)
			z.Sugar().Infof("-->(%d)list: %+v", rowCount, list)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQuerySubjectiveQuestionsMarkingResults(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
		expectedErrStr string
	}{
		{
			name: "success: 考试场次下获取所有阅卷结果",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
		},
		{
			name: "success: 考试场次下获取单个考生阅卷结果",
			cond: QueryCondition{
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
		},
		{
			name: "success: 练习下获取所有阅卷结果",
			cond: QueryCondition{
				PracticeID: 22,
			},
		},
		{
			name: "success: 练习下获取单个提交的阅卷结果",
			cond: QueryCondition{
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
		},
		{
			name:           "error: 请求参数必须包含考试场次ID或者练习ID中的一个",
			cond:           QueryCondition{},
			expectedErrStr: "无效的请求参数，必须包含 考试场次ID 或者 练习ID 中的一个",
		},
		{
			name: "error: 请求参数不能同时包含练习ID和考试场次ID",
			cond: QueryCondition{
				ExamSessionID: 101,
				PracticeID:    21,
			},
			expectedErrStr: "无效的请求参数，不能同时包含练习ID和考试场次ID",
		},
		{
			name: "error: SQL 查询执行失败",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			forceErr:       "getMarkingResultQuery",
			expectedErrStr: "获取批改结果 sql 失败",
		},
		{
			name: "error: 行扫描失败",
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 读取结果失败",
		},
		{
			name: "error: 行遍历出错",
			cond: QueryCondition{
				ExamSessionID: 103,
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

			markingResults, err := QuerySubjectiveQuestionsMarkingResults(ctx, tt.cond)
			z.Sugar().Infof("markingResults: %+v", markingResults)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQueryMarkerInfo(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
		expectedErrStr string
	}{
		{
			name: "success: 考试场次下获取批改配置信息",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
		},
		{
			name: "success: 考试场次下获取指定教师批改配置信息",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 102,
			},
		},
		{
			name: "success: 练习下获取批改配置信息",
			cond: QueryCondition{
				TeacherID:  testedTeacherID,
				PracticeID: 22,
			},
		},
		{
			name: "error: 请求参数必须包含考试场次ID或者练习ID中的一个",
			cond: QueryCondition{
				TeacherID: testedTeacherID,
			},
			expectedErrStr: "无效的请求参数，请求参数必须包含 考试场次ID 或者 练习ID 中的一个",
		},
		{
			name: "error: SQL 查询执行失败",
			cond: QueryCondition{
				TeacherID:     testedTeacherID,
				ExamSessionID: 101,
			},
			forceErr:       "QueryMarkerInfo-pgxConn.Query",
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
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQueryStudentAnswers(t *testing.T) {
	cleanTestData()

	// 模拟 MarkerInfo
	markerInfo := MarkerInfo{
		MarkMode: "00",
		MarkInfos: []cmn.TMarkInfo{
			{
				MarkExamineeIds: mustMarshal([]int64{2205, 2206}),
				QuestionIds:     mustMarshal([]int64{301, 302}),
			},
		},
	}

	tests := []struct {
		name           string
		tx             pgx.Tx
		questionType   string
		cond           QueryCondition
		markerInfo     MarkerInfo
		forceErr       string
		expectedErrStr string
	}{
		{
			name:         "success: 考试-单人批改-选择题",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
				ExamineeID:    2205,
			},
			markerInfo: markerInfo,
		},
		{
			name:         "success: 考试-题组专评-简答题",
			questionType: "06",
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: mustMarshal([]int64{301, 302}),
					},
				},
			},
		},
		{
			name:         "success: 练习-查询练习学生答案",
			questionType: "08",
			cond: QueryCondition{
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
			markerInfo: MarkerInfo{
				MarkMode: "12",
			},
		},
		{
			name:         "error: 无效的题目类型",
			questionType: "99",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			expectedErrStr: "无效的题目类型",
		},
		{
			name:         "error: 无效的 practice_submission_id",
			questionType: "08",
			cond: QueryCondition{
				PracticeID: 22,
			},
			markerInfo: MarkerInfo{
				MarkMode: "12",
			},
			expectedErrStr: "无效的 practice_submission_id",
		},
		{
			name:         "error: SQL 执行失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			forceErr:       "query",
			expectedErrStr: "获取学生答案 sql 错误",
		},
		{
			name:         "error: 行扫描失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name:         "error: 行遍历失败",
			questionType: "00",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
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
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQuerySubjectiveQuestions(t *testing.T) {
	cleanTestData()

	// 模拟 MarkerInfo
	markerInfo := MarkerInfo{
		MarkMode: "06", // 题组专评
		MarkInfos: []cmn.TMarkInfo{
			{
				QuestionIds: mustMarshal([]int64{301, 302}),
			},
		},
	}

	tests := []struct {
		name           string
		tx             pgx.Tx
		cond           QueryCondition
		markerInfo     MarkerInfo
		forceErr       string
		expectedErrStr string
	}{
		{
			name: "success: 考试-获取主观题",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo: markerInfo,
		},
		{
			name: "success: 练习-获取主观题",
			cond: QueryCondition{
				PracticeID: 22,
			},
			markerInfo: markerInfo,
		},
		{
			name: "success: 查询单道题目",
			cond: QueryCondition{
				ExamSessionID: 101,
				QuestionID:    301,
			},
			markerInfo: markerInfo,
		},
		{
			name: "success: 无效的 markerInfo QuestionIds",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo: MarkerInfo{
				MarkMode: "06",
				MarkInfos: []cmn.TMarkInfo{
					{
						QuestionIds: []byte("invalid json"),
					},
				},
			},
			expectedErrStr: "unable to unmarshal question ids",
		},
		{
			name: "error: SQL 执行失败",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			forceErr:       "getQuestionsQuery",
			expectedErrStr: "获取主观题 sql 失败",
		},
		{
			name: "error: 行扫描失败",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name: "error: 行遍历失败",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			markerInfo:     markerInfo,
			forceErr:       "rows.Err",
			expectedErrStr: "行遍历",
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			questionSets, err := QuerySubjectiveQuestions(ctx, nil, tt.cond, tt.markerInfo)
			z.Sugar().Infof("questionSets: %+v", questionSets)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQueryExamSessionStatus(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		tx             pgx.Tx
		examSessionID  int64
		forceErr       string
		expectedErrStr string
	}{
		{
			name:          "success: 成功获取考试状态",
			examSessionID: 101,
		},
		{
			name:          "success: 成功获取考试状态-带事务",
			tx:            nil, // 如果你有事务 tx 可以替换
			examSessionID: 102,
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
			examSessionID:  101,
			forceErr:       "QueryExamSessionStatus",
			expectedErrStr: ForceErr.Error(),
		},
	}

	initTestData()
	defer cleanTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			status, err := QueryExamSessionStatus(ctx, tt.tx, tt.examSessionID)
			z.Sugar().Infof("examSessionID=%d, status=%s", tt.examSessionID, status)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestQueryStudentInfos(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		markerInfo     MarkerInfo
		forceErr       string
		expectedErrStr string
	}{
		// success
		{
			name: "success: 考试场次，获取所有学生信息",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
		},
		{
			name: "success: 考试场次，单个考生",
			cond: QueryCondition{
				ExamSessionID: 102,
				ExamineeID:    2205,
			},
		},
		{
			name: "success: 练习，获取所有待批改学生",
			cond: QueryCondition{
				PracticeID: 22,
			},
		},
		{
			name: "success: 练习，单个提交",
			cond: QueryCondition{
				PracticeID:           22,
				PracticeSubmissionID: 2404,
			},
		},
		{
			name: "success: 试卷分配，考试带 markerInfo",
			cond: QueryCondition{
				ExamSessionID: 103,
			},
			markerInfo: MarkerInfo{
				MarkMode: "04",
				MarkInfos: []cmn.TMarkInfo{
					{
						MarkExamineeIds: mustMarshal(`[2205,2206]`),
					},
				},
			},
		},

		// error
		{
			name:           "error: 无效请求参数，没有考试场次ID或练习ID",
			cond:           QueryCondition{},
			expectedErrStr: "无效的请求参数，必须包含 考试场次ID 或者 练习ID 中的一个",
		},
		{
			name: "error: 请求参数不能同时包含 练习ID 和 考试场次ID",
			cond: QueryCondition{
				ExamSessionID: 101,
				PracticeID:    22,
			},
			expectedErrStr: "无效的请求参数，不能同时包含 练习ID 和 考试场次ID",
		},
		{
			name: "error: 强制 SQL 错误",
			cond: QueryCondition{
				ExamSessionID: 101,
			},
			forceErr:       "query",
			expectedErrStr: "查询学生信息 sql 失败",
		},
		{
			name: "error: 行扫描失败",
			cond: QueryCondition{
				ExamSessionID: 102,
			},
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name: "error: 行遍历失败",
			cond: QueryCondition{
				ExamSessionID: 103,
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

			studentInfos, err := QueryStudentInfos(ctx, tt.cond, tt.markerInfo)
			z.Sugar().Infof("studentInfos: %+v", studentInfos)

			if tt.expectedErrStr != "" {
				assert.Error(t, err, "期待获取到错误，但却没有错误")
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
			}
		})
	}
}

func TestUpdateMarkerInfoStatus(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		teacherID      int64
		ids            []int64
		mode           string
		status         string
		forceErr       string
		expectedErrStr string
	}{
		{
			name:      "success:考试",
			teacherID: testedTeacherID,
			ids:       []int64{101},
			mode:      "00",
			status:    "00",
		},
		{
			name:      "success:练习",
			teacherID: testedTeacherID,
			ids:       []int64{201},
			mode:      "02",
			status:    "02",
		},
		{
			name:           "error:teacherID无效",
			teacherID:      0,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			expectedErrStr: "无效的 teacher ID",
		},
		{
			name:           "error:ids为空",
			teacherID:      testedTeacherID,
			ids:            nil,
			mode:           "00",
			status:         "00",
			expectedErrStr: "没有 exam_session_ids",
		},
		{
			name:           "error:status无效",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "99",
			expectedErrStr: "status 错误",
		},
		{
			name:           "error:mode无效",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "99",
			status:         "00",
			expectedErrStr: "无效的模式",
		},
		{
			name:           "error:强制错误 updateQuery",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			forceErr:       "updateQuery",
			expectedErrStr: "更新批改配置信息 sql 失败",
		},
		{
			name:           "error:强制错误 rows.Scan",
			teacherID:      testedTeacherID,
			ids:            []int64{101},
			mode:           "00",
			status:         "00",
			forceErr:       "rows.Scan",
			expectedErrStr: "从 row 中读取结果失败",
		},
		{
			name:           "error:强制错误 rows.Err",
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			tx, err := cmn.GetPgxConn().Begin(ctx)
			assert.NoError(t, err)
			defer tx.Rollback(ctx)

			targetIDs, err := UpdateMarkerInfoStatus(ctx, tx, tt.teacherID, tt.ids, tt.mode, tt.status)
			z.Sugar().Infof("targetIDs: %+v", targetIDs)

			if tt.expectedErrStr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.ids), len(targetIDs))
			}
		})
	}
}

func TestUpsertMarkingResults(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		markingResults []cmn.TMark
		forceErr       string
		expectedErrStr string
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
			tx, err := cmn.GetPgxConn().Begin(ctx)
			assert.NoError(t, err)
			defer tx.Rollback(ctx)

			ids, err := UpsertMarkingResults(ctx, tx, tt.markingResults)

			if tt.expectedErrStr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.markingResults), len(ids), "返回的 ID 数量应与插入记录数量一致")
				for _, id := range ids {
					assert.Greater(t, id, int64(0), "返回的 ID 应大于 0")
				}
			}
		})
	}
}

func TestQueryMarkedPersonAndQuestionCount(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		cond           QueryCondition
		forceErr       string
		expectedPerson int64
		expectedCount  int64
		expectedErrStr string
	}{
		{
			name:           "success:考试",
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 101},
			expectedPerson: 5,
			expectedCount:  50,
		},
		{
			name:           "success:练习",
			cond:           QueryCondition{PracticeID: 201},
			expectedPerson: 3,
			expectedCount:  30,
		},
		{
			name:           "forceErr: QueryMarkedPersonAndQuestionCount",
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 101},
			forceErr:       "QueryMarkedPersonAndQuestionCount",
			expectedErrStr: ForceErr.Error(),
		},
		{
			name:           "forceErr: queryMarkedPerson",
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 101},
			forceErr:       "queryMarkedPerson",
			expectedErrStr: "获取已批改人数 sql 失败: ",
		},
		{
			name:           "forceErr: queryQuestionCount",
			cond:           QueryCondition{TeacherID: 1, ExamSessionID: 101},
			forceErr:       "queryQuestionCount",
			expectedErrStr: "获取已批改问题数 sql 失败: ",
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
				if err == nil {
					t.Fatalf("预期错误，但返回 nil")
				}
				if err.Error() != tt.expectedErrStr {
					t.Fatalf("错误信息不匹配，got: %v, want: %v", err.Error(), tt.expectedErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if person != tt.expectedPerson {
				t.Errorf("已批改人数不匹配，got: %v, want: %v", person, tt.expectedPerson)
			}
			if count != tt.expectedCount {
				t.Errorf("已批改问题数不匹配，got: %v, want: %v", count, tt.expectedCount)
			}
		})
	}
}

func TestUpdateStudentAnswerScore(t *testing.T) {
	cleanTestData()

	tests := []struct {
		name           string
		markingResults []cmn.TMark
		cond           QueryCondition
		forceErr       string
		expectedErrStr string
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, ForceErrKey, tt.forceErr)
			}

			ids, err := UpdateStudentAnswerScore(ctx, nil, tt.markingResults, tt.cond)
			if tt.expectedErrStr != "" {
				if err == nil {
					t.Fatalf("预期错误，但返回 nil")
				}
				if err.Error() != tt.expectedErrStr {
					t.Fatalf("错误信息不匹配，got: %v, want: %v", err.Error(), tt.expectedErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(ids) != len(tt.markingResults) {
				t.Errorf("返回的更新ID数量不匹配, got: %d, want: %d", len(ids), len(tt.markingResults))
			}
		})
	}
}

func TestUpdateExamSessionOrPracticeSubmissionStatus(t *testing.T) {
	tests := []struct {
		name                  string
		teacherID             int64
		examSessionIDs        []int64
		practiceSubmissionIDs []int64
		status                string
		forceErr              string
		expectedErrStr        string
	}{
		{
			name:           "成功: 更新考试状态",
			teacherID:      1,
			examSessionIDs: []int64{101},
			status:         "04",
		},
		{
			name:                  "成功: 更新练习状态",
			teacherID:             1,
			practiceSubmissionIDs: []int64{201},
			status:                "04",
		},
		{
			name:           "错误: teacherID 无效",
			teacherID:      0,
			examSessionIDs: []int64{101},
			status:         "04",
			expectedErrStr: "无效的 teacher_id",
		},
		{
			name:           "错误: examSessionIDs 和 practiceSubmissionIDs 都为空",
			teacherID:      1,
			status:         "04",
			expectedErrStr: "无效的 exam_session_ids 和 practice_submission_ids",
		},
		{
			name:                  "错误: examSessionIDs 和 practiceSubmissionIDs 同时存在",
			teacherID:             1,
			examSessionIDs:        []int64{101},
			practiceSubmissionIDs: []int64{201},
			status:                "04",
			expectedErrStr:        "exam_session_ids 和 practice_submission_ids 不能同时存在",
		},
		{
			name:           "错误: status 为空",
			teacherID:      1,
			examSessionIDs: []int64{101},
			status:         "",
			expectedErrStr: "无效的 考试/练习 状态",
		},
		{
			name:           "forceErr: updateQuery",
			teacherID:      1,
			examSessionIDs: []int64{101},
			status:         "04",
			forceErr:       "updateQuery",
			expectedErrStr: ForceErr.Error(),
		},
		{
			name:           "forceErr: rows.Scan",
			teacherID:      1,
			examSessionIDs: []int64{101},
			status:         "04",
			forceErr:       "rows.Scan",
			expectedErrStr: ForceErr.Error(),
		},
		{
			name:           "forceErr: rows.Err",
			teacherID:      1,
			examSessionIDs: []int64{101},
			status:         "04",
			forceErr:       "rows.Err",
			expectedErrStr: ForceErr.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)

			ids, err := UpdateExamSessionOrPracticeSubmissionStatus(ctx, nil, tt.teacherID, tt.examSessionIDs, tt.practiceSubmissionIDs, tt.status)
			if tt.expectedErrStr != "" {
				if err == nil {
					t.Fatalf("预期错误，但返回 nil")
				}
				if err.Error() != tt.expectedErrStr {
					t.Fatalf("错误信息不匹配，got: %v, want: %v", err.Error(), tt.expectedErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(ids) != len(tt.examSessionIDs)+len(tt.practiceSubmissionIDs) {
				t.Errorf("返回的更新ID数量不匹配, got: %d, want: %d", len(ids), len(tt.examSessionIDs)+len(tt.practiceSubmissionIDs))
			}
		})
	}
}

func TestUpdatePracticeWrongSubmissionStatus(t *testing.T) {
	testedTeacherID := int64(1)
	pgConn := cmn.GetPgxConn()

	tests := []struct {
		name                       string
		teacherID                  int64
		practiceWrongSubmissionIDs []int64
		status                     string
		forceErr                   string
		expectedErrStr             string
	}{
		{
			name:                       "success",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "04",
		},
		{
			name:                       "error: teacherID",
			teacherID:                  0,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "04",
			expectedErrStr:             "无效的 teacher_id",
		},
		{
			name:                       "error: empty ids",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: nil,
			status:                     "04",
			expectedErrStr:             "无效的 practice_wrong_submission_ids",
		},
		{
			name:                       "error: empty status",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "",
			expectedErrStr:             "无效的练习状态",
		},
		{
			name:                       "force: tx.Query",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "04",
			forceErr:                   "UpdatePracticeWrongSubmissionStatus-tx.Query",
			expectedErrStr:             ForceErr.Error(),
		},
		{
			name:                       "force: rows.Scan",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "04",
			forceErr:                   "rows.Scan",
			expectedErrStr:             ForceErr.Error(),
		},
		{
			name:                       "force: rows.Err",
			teacherID:                  testedTeacherID,
			practiceWrongSubmissionIDs: []int64{301},
			status:                     "04",
			forceErr:                   "rows.Err",
			expectedErrStr:             ForceErr.Error(),
		},
	}

	for _, tt := range tests {
		for _, useTx := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s: tx=%v", tt.name, useTx), func(t *testing.T) {
				var tx pgx.Tx
				if useTx {
					var err error
					tx, err = pgConn.Begin(context.Background())
					if err != nil {
						t.Fatalf("开启事务失败: %v", err)
					}
					defer tx.Rollback(context.Background())
				}

				ctx := context.WithValue(context.Background(), ForceErrKey, tt.forceErr)
				list, err := UpdatePracticeWrongSubmissionStatus(ctx, tx, tt.teacherID, tt.practiceWrongSubmissionIDs, tt.status)

				z.Sugar().Infof("--> 更新练习错题返回 IDs: %+v", list)

				if tt.expectedErrStr != "" {
					assert.Error(t, err, "期待获取到错误，但却没有错误")
					assert.Contains(t, err.Error(), tt.expectedErrStr)
				} else {
					assert.NoErrorf(t, err, "期待没有错误，但却获取到错误: %v", err)
				}
			})
		}
	}
}
