/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷数据库层单元测试
 * @Date: 2025-07-28 19:55:28
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-19 11:46:02
 */
package examPaper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"
	"math/rand"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

var (
	ctx = context.Background()
	now = time.Now().UnixMilli()
	uid = null.IntFrom(10086)
)

func TestLoadPaperTemplateById(t *testing.T) {
	// 这里需要先准备合适的数据，先构成一张完整的试卷
	// 创建对应的题库

	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()
	// 这里的话先插入准备好的题库、题目跟试卷
	s := `DELETE FROM t_paper_question`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	initQuestion(t, conn)

	initPaper(t, conn)
	paper := &cmn.TPaper{
		ID:                uid,
		Name:              null.StringFrom("考卷单元测试试卷名"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("02"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("考卷单元测试服务"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           uid,
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         uid,
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
	}
	groupNames := []string{"一、单选题", "二、多选题", "三、判断题", "四、填空题", "五、简答题"}

	tests := []struct {
		name          string
		pid           int64
		withQuestions bool
		expectedErr   error
	}{
		{
			name:          "正常1 正确paperID 携带问题查询",
			pid:           uid.Int64,
			withQuestions: true,
			expectedErr:   nil,
		},
		{
			name:          "正常1 正确paperID 不携带问题查询",
			pid:           uid.Int64,
			withQuestions: false,
			expectedErr:   nil,
		},
		{
			name:          "异常1 不存在paperID 携带问题查询",
			pid:           100000,
			withQuestions: false,
			expectedErr:   errors.New("select paper failed"),
		},
		{
			name:          "异常2 不存在paperID 不携带问题查询",
			pid:           100000,
			withQuestions: false,
			expectedErr:   errors.New("select paper failed"),
		},
		{
			name:          "异常3 非法paperID",
			pid:           -1,
			withQuestions: false,
			expectedErr:   errors.New("invalid practice ID param"),
		},
		{
			name:          "异常3 json.unmarshal触发失败",
			pid:           uid.Int64,
			withQuestions: true,
			expectedErr:   errors.New("unmarshal group data failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "json")
			} else {
				ctx = context.Background()
			}
			tp, pg, pq, err := LoadPaperTemplateById(ctx, tt.pid, tt.withQuestions)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				// 这里如果没有错误返回的话，那正常就需要获取到这些试卷题目
				// 下面就是检测这个试卷本身
				if tp == nil {
					t.Errorf("期望正常返回，但是没有返回正常数据出来")
				}
				if tp.Name != paper.Name {
					t.Errorf("查询的试卷名出错 期望：%v , 实际：%v", paper.Name, tp.Name)
				}
				if tp.AssemblyType != paper.AssemblyType {
					t.Errorf("查询的试卷组卷方式出错 期望：%v , 实际：%v", paper.AssemblyType, tp.AssemblyType)
				}
				if tp.Category != paper.Category {
					t.Errorf("查询的试卷用途方式出错 期望%v,实际：%v", paper.Category, tp.Category)
				}
				if tp.Level != paper.Level {
					t.Errorf("查询的试卷难度出错 期望%v,实际：%v", paper.Level, tp.Level)
				}
				if tp.SuggestedDuration != paper.SuggestedDuration {
					t.Errorf("查询的试卷建议时长出错 期望%v,实际：%v", paper.SuggestedDuration, tp.SuggestedDuration)
				}
				if tp.Creator != paper.Creator {
					t.Errorf("查询的试卷创建者出错 期望%v,实际：%v", paper.Creator, tp.Creator)
				}
				if tp.TotalScore.Float64 != 48 {
					t.Errorf("查询的试卷总分出错 期望%v,实际：%v", 48, tp.TotalScore.Float64)
				}
				if tp.QuestionCount.Int64 != 13 {
					t.Errorf("查询的试卷题目总数出错 期望%v,实际：%v", 13, tp.QuestionCount.Int64)
				}
				if tp.GroupCount.Int64 != 5 {
					t.Errorf("查询的试卷题组总数出错 期望%v,实际：%v", 5, tp.GroupCount.Int64)
				}
				// 这里还要去解析当前这个题目
				if tt.withQuestions {
					// 下面是检测题组
					if pg[0].Name.String != groupNames[0] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[0], pg[0].Name.String)
					}
					if pg[0].Order.Int64 != 1 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 1, pg[0].Order.Int64)
					}
					if pg[1].Name.String != groupNames[1] {
						t.Errorf("查询的试卷题组2名称出错 期望%v,实际：%v", groupNames[1], pg[1].Name.String)
					}
					if pg[1].Order.Int64 != 2 {
						t.Errorf("查询的试卷题组2顺序出错 期望%v,实际：%v", 2, pg[1].Order.Int64)
					}
					if pg[2].Name.String != groupNames[2] {
						t.Errorf("查询的试卷题组3名称出错 期望%v,实际：%v", groupNames[2], pg[2].Name.String)
					}
					if pg[2].Order.Int64 != 3 {
						t.Errorf("查询的试卷题组3顺序出错 期望%v,实际：%v", 3, pg[2].Order.Int64)
					}
					if pg[3].Name.String != groupNames[3] {
						t.Errorf("查询的试卷题组4名称出错 期望%v,实际：%v", groupNames[3], pg[3].Name.String)
					}
					if pg[3].Order.Int64 != 4 {
						t.Errorf("查询的试卷题组4顺序出错 期望%v,实际：%v", 4, pg[3].Order.Int64)
					}
					if pg[4].Name.String != groupNames[4] {
						t.Errorf("查询的试卷题组5名称出错 期望%v,实际：%v", groupNames[4], pg[4].Name.String)
					}
					if pg[4].Order.Int64 != 5 {
						t.Errorf("查询的试卷题组5顺序出错 期望%v,实际：%v", 5, pg[4].Order.Int64)
					}

					var questionCount int64

					// 再下面是统一题目本身 如果满足每一个题组下的题目个数都符合预算的话，那就算通过
					for _, questions := range pq {
						for _, _ = range questions {
							questionCount++
						}
					}
					if questionCount != tp.QuestionCount.Int64 {
						t.Errorf("统计的题目数量错误，期望：%v,实际%v", tp.QuestionCount.Int64, questionCount)
					}
				} else {
					if len(pg) != 0 {
						t.Errorf("不携带试卷题组时，但是还是返回了")
					}
				}
			}
		})

		t.Cleanup(func() {
			// 这里的话先插入准备好的题库、题目跟试卷
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}
}

// 生成考卷测试
func TestGenerateExamPaper(t *testing.T) {
	// 由于这里已经有创建好试卷了，就直接使用就好

	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()
	// 这里的话先删除之前创建的所有数据
	// 这里需要清除所有考卷信息

	s := `DELETE FROM assessuser.t_student_answers`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM assessuser.t_exam_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM t_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	// 然后考试
	s = `DELETE FROM t_examinee`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_exam_session`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_info`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	initQuestion(t, conn)

	initPaper(t, conn)

	var practiceName string
	practiceName = "单元测试练习名"
	// 这里也要创建练习
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}

	// 创建对应的考试，生成对应的场次ID
	_, err = conn.Exec(ctx, `
		INSERT INTO t_exam_info (id,
			name, type, mode, creator, status
		) VALUES (
			$1,'考卷生成单元测试考试', '00', '00', $2, '02'
		)`,
		uid, uid)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}

	_, err = conn.Exec(ctx, `
		INSERT INTO t_exam_session (id,
			exam_id, session_num, paper_id,
			status, creator,mark_method
		) VALUES (
			$1,$2, 1, $3,'00',$4,$5
		)`,
		uid, uid, uid, uid, "02")
	if err != nil {
		t.Fatalf("创建测试考试场次失败: %v", err)
	}

	tests := []struct {
		name        string
		category    string
		pid         int64
		practiceID  int64
		sessionID   int64
		uid         int64
		genMarkInfo bool
		expectedErr error
		practice    bool
	}{
		{
			name:        "正常生成练习考卷逻辑 并返回批改信息",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: nil,
			practice:    true,
		},
		{
			name:        "正常生成考试考卷逻辑 并返回批改信息",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  0,
			sessionID:   uid.Int64,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: nil,
			practice:    false,
		},
		{
			name:        "正常生成练习考卷逻辑 不返回批改信息",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: false,
			expectedErr: nil,
			practice:    true,
		},
		{
			name:        "正常生成考试考卷逻辑 不返回批改信息",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  0,
			sessionID:   uid.Int64,
			uid:         uid.Int64,
			genMarkInfo: false,
			expectedErr: nil,
			practice:    false,
		},
		{
			name:        "异常1 参数检测非法pid",
			category:    PaperCategory.Practice,
			pid:         -1,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid paperId ID param"),
		},
		{
			name:        "异常2 参数检测非法练习试卷类型",
			category:    "06",
			pid:         uid.Int64,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid category param"),
		},
		{
			name:        "异常3 参数检测非法练习ID与考试场次ID",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  0,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid practice or examSession ID param"),
		},
		{
			name:        "异常4 参数检测非法操作者ID",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  1,
			sessionID:   0,
			uid:         -1,
			genMarkInfo: true,
			expectedErr: errors.New("invalid uid ID param"),
		},
		{
			name:        "异常5 触发LoadPaperTemplateById函数错误",
			category:    PaperCategory.Practice,
			pid:         1000000,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("select paper failed"),
		},
		{
			name:        "异常6 pg空",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid paper question group"),
		},
		{
			name:        "异常7 触发插入考卷失败 query1",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("插入考卷失败"),
		},
		{
			name:        "异常8 触发插入题组失败 query2",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("插入题组失败"),
		},
		{
			name:        "异常9 触发扫描题组ID失败 scan",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("扫描题组信息失败"),
		},
		{
			name:        "异常10 触发扫描题组ID失败 row",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("遍历扫描题组信息中出错"),
		},
		{
			name:        "异常11 触发构建插入考卷题目映射不存在 notExists",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("无法找到顺序为"),
		},
		{
			name:        "异常12 触发题目JSON格式不对 jsonA",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("解析答案失败"),
		},
		{
			name:        "异常13 触发题目答案与小题分长度不一 jsonB",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("不匹配"),
		},
		{
			name:        "异常14 触发批量插入考卷题目错误 query3",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("插入题目批次"),
		},
		{
			name:        "异常15 触发批量插入考卷题目扫描考题ID错误 scan1",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("扫描题目ID失败"),
		},
		{
			name:        "异常16 触发批量插入考卷题目遍历扫描考题ID错误 row1",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("读取题目ID失败"),
		},
		{
			name:        "异常17 触发批量插入时数量索引操作错误 notMatch",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("题目ID数量不匹配"),
		},
		{
			name:        "异常18 触发新考题与旧题组映射 qs",
			category:    PaperCategory.Practice,
			pid:         uid.Int64,
			practiceID:  uid.Int64,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 现在已经是已经有练习跟考试了的

			// 每一个用例再创建独立的事务
			tx, _ := conn.Begin(ctx)
			defer func() {
				_ = tx.Rollback(ctx)
			}()

			if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "pg")
			} else if containsString(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "query1")
			} else if containsString(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "row")
			} else if containsString(tt.name, "异常11") {
				ctx = context.WithValue(ctx, "force-error", "notExists")
			} else if containsString(tt.name, "异常12") {
				ctx = context.WithValue(ctx, "force-error", "jsonA")
			} else if containsString(tt.name, "异常13") {
				ctx = context.WithValue(ctx, "force-error", "jsonB")
			} else if containsString(tt.name, "异常14") {
				ctx = context.WithValue(ctx, "force-error", "query3")
			} else if containsString(tt.name, "异常15") {
				ctx = context.WithValue(ctx, "force-error", "scan1")
			} else if containsString(tt.name, "异常16") {
				ctx = context.WithValue(ctx, "force-error", "row1")
			} else if containsString(tt.name, "异常17") {
				ctx = context.WithValue(ctx, "force-error", "notMatch")
			} else if containsString(tt.name, "异常18") {
				ctx = context.WithValue(ctx, "force-error", "qs")
			} else {
				ctx = context.Background()
			}

			epid, mark, err := GenerateExamPaper(ctx, tx, tt.category, tt.pid, tt.practiceID, tt.sessionID, tt.uid, tt.genMarkInfo)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				// 就需要检测此时生成的考卷信息了
				// 这里获取到考卷的ID，就能去检验是否生成成功
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if epid == nil {
					t.Fatalf("返回的考卷ID为空")
				}

				if containsString(tt.name, "异常18") {
					// 此时的题组应该是0
					if len(mark) != 0 {
						t.Errorf("此时返回的批改题组应该是空：实际：%v", len(mark))
					}
				} else {

					// 这里需要查询一下是否为练习或者是考试，看这个考卷是否成功完成
					ep := `SELECT id,name,exam_session_id,practice_id,creator ,total_score,question_count,group_count,groups_data FROM v_exam_paper WHERE name=$1`
					examPaper := cmn.TVExamPaper{}
					err := tx.QueryRow(ctx, ep, "考卷单元测试试卷名").Scan(&examPaper.ID, &examPaper.Name, &examPaper.ExamSessionID, &examPaper.PracticeID, &examPaper.Creator, &examPaper.TotalScore,
						&examPaper.QuestionCount, &examPaper.GroupCount, &examPaper.GroupsData)
					if err != nil {
						t.Fatal(err)
					}
					if tt.practice {
						if examPaper.PracticeID.Int64 != tt.practiceID {
							t.Errorf("生成的考卷练习ID错误 预期：%v，实际：%v", tt.practiceID, examPaper.PracticeID.Int64)
						}
					} else {
						if examPaper.ExamSessionID.Int64 != tt.sessionID {
							t.Errorf("生成的考卷考试ID错误 预期：%v，实际：%v", tt.sessionID, examPaper.ExamSessionID.Int64)
						}
					}

					if examPaper.Name.String != "考卷单元测试试卷名" {
						t.Errorf("生成的考卷名称错误 预期：考卷单元测试试卷名，实际：%v", examPaper.Name.String)
					}
					if examPaper.Creator.Int64 != uid.Int64 {
						t.Errorf("生成的考卷创建者错误 预期：%v，实际：%v", uid, examPaper.Creator)
					}
					var groupData []ExamGroup
					err = json.Unmarshal(examPaper.GroupsData, &groupData)
					if err != nil {
						t.Errorf("无法解析题组、题目数据：%v", err)
					}
					if len(groupData) == 0 {
						t.Errorf("解析题组、题目数据为空：%v", err)
					}
					// 一张考卷卷拥有的题组map
					var examGroups []*cmn.TExamPaperGroup
					// 一个题组下拥有的题目数组
					examQuestions := make(map[int64][]*ExamQuestion)
					for _, v := range groupData {
						examGroups = append(examGroups, &cmn.TExamPaperGroup{
							ID:          v.ID,
							ExamPaperID: v.ExamPaperID,
							Name:        v.Name,
							Order:       v.Order,
							Creator:     v.Creator,
							CreateTime:  v.CreateTime,
							UpdatedBy:   v.UpdatedBy,
							UpdateTime:  v.UpdateTime,
							Addi:        v.Addi,
							Status:      v.Status,
						})
						if _, exists := examQuestions[v.ID.Int64]; !exists {
							examQuestions[v.ID.Int64] = make([]*ExamQuestion, 0)
						}
						for idx := range v.Questions {
							q := v.Questions[idx]
							examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
						}
					}
					var gplen int64
					gplen = int64(len(examGroups))
					if gplen != examPaper.GroupCount.Int64 {
						t.Errorf("题组数量与查询出来的预期不一致 预期：%v,实际：%v", examPaper.GroupCount.Int64, gplen)
					}
					// 这里去解析他，然后导出这个数量来
					// 如果此时是选择了生成这个的话
					if tt.genMarkInfo {
						if len(mark) == 0 {
							t.Errorf("返回的批改信息为空")
						}
						groups := []int64{}
						questions := []int64{}
						for _, m := range mark {
							t.Logf("打印输出一下返回的题组信息%v", m.GroupID)
							groups = append(groups, m.GroupID)
							for _, q := range m.QuestionIDs {
								t.Logf("打印输出一下返回的题目信息%v", q)
								questions = append(questions, q)
							}
						}
						var pgCount, qCount int

						// 处理题组计数
						pg := `SELECT COUNT(*) FROM t_exam_paper_group WHERE id = ANY($1)`
						err = tx.QueryRow(ctx, pg, groups).Scan(&pgCount)
						if err != nil {
							t.Fatal(err)
						}

						// 处理问题计数
						q := `SELECT COUNT(*) FROM t_exam_paper_question WHERE id = ANY($1)`
						err = tx.QueryRow(ctx, q, questions).Scan(&qCount)
						if err != nil {
							t.Fatal(err)
						}

						// 这里只会对主观题进行批改的配置
						if pgCount != 2 || qCount != 5 {
							t.Errorf("生成的考卷题组与考卷题目数据有误：实际上：题组%v,题目%v", pgCount, qCount)
						}

					} else {
						var pgCount, qCount int
						// 如果没返回这个的话，还是需要去查这个视图，去看这个数量是否一致 如果选择不返回AI批改的配置，则会返回所有题组的信息，因此与批改的数量会不一样
						v := `SELECT group_count,question_count FROM v_exam_paper WHERE id = $1`
						err = tx.QueryRow(ctx, v, *epid).Scan(&pgCount, &qCount)
						if err != nil {
							t.Fatal(err)
						}
						if pgCount != 5 || qCount != 13 {
							t.Errorf("生成的考卷题组与考卷题目数据有误：实际上：题组%v,题目%v", pgCount, qCount)
						}
					}
				}
			}

			t.Cleanup(func() {
				// 这里需要清除所有考卷信息
				s = `DELETE FROM assessuser.t_exam_paper_question`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				s = `DELETE FROM assessuser.t_exam_paper_group`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				s = `DELETE FROM assessuser.t_exam_paper`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
				// 这里还要再重新删除练习跟考试，然后再重新创建？
			})
		})
		t.Cleanup(func() {
			// 这里的话先插入准备好的题库、题目跟试卷
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			// 再删除练习
			s = `DELETE FROM t_practice`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_examinee`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_session`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_info`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}

}

func TestLoadExamPaperDetailsById(t *testing.T) {
	// 现在已经成功生成了考卷了，那就需要去测试这个数据是否存在
	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()

	//这里查看考卷详情，需要生成一张考卷就好了，由于练习会比较简单，我这里就直接使用练习去生成了
	s := `DELETE FROM assessuser.t_student_answers`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM assessuser.t_exam_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM t_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_practice_submissions`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	initQuestion(t, conn)

	initPaper(t, conn)
	var practiceName string
	practiceName = "单元测试练习名"
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}

	var ep *int64

	tx1, _ := conn.Begin(ctx)
	// 这里直接使用函数插入
	ep, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Practice, uid.Int64, uid.Int64, 0, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if ep == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}
	err = tx1.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		epid          int64
		withQuestions bool
		withAnswers   bool
		withAnalysis  bool
		expectedErr   error
	}{
		{
			name:          "正常1 能查询出一张完整的考卷试卷 不携带题目答案 不携带解析",
			epid:          *ep,
			withQuestions: false,
			withAnswers:   false,
			withAnalysis:  false,
			expectedErr:   nil,
		},
		{
			name:          "正常2 能查询出一张完整的考卷试卷 携带题目答案 不携带解析",
			epid:          *ep,
			withQuestions: true,
			withAnswers:   true,
			withAnalysis:  false,
			expectedErr:   nil,
		},
		{
			name:          "正常3 能查询出一张完整的考卷试卷 不携带题目答案 携带解析",
			epid:          *ep,
			withQuestions: true,
			withAnswers:   false,
			withAnalysis:  true,
			expectedErr:   nil,
		},

		{
			name:          "异常1 参数检测 非法考卷ID",
			epid:          0,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("invalid examPaper ID param"),
		},
		{
			name:          "异常2 参数检测 不存在的考卷ID",
			epid:          100000,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("select examPaper failed"),
		},
		{
			name:          "强制异常3 空groupsData jsonA",
			epid:          *ep,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("empty examPaper groups data"),
		},
		{
			name:          "强制异常4 非法groupsData jsonUnmarshal json",
			epid:          *ep,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("unmarshal group data failed"),
		},
		{
			name:          "强制异常5 非法Answers jsonUnmarshal jsonB",
			epid:          *ep,
			withQuestions: true,
			withAnswers:   false,
			expectedErr:   errors.New("failed to unmarshal Answers questionId"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "强制异常5") {
				ctx = context.WithValue(ctx, "force-error", "jsonB")
			} else if containsString(tt.name, "强制异常4") {
				ctx = context.WithValue(ctx, "force-error", "json")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "jsonA")
			} else {
				ctx = context.Background()
			}

			// 每一个用例再创建独立的事务
			tx, _ := conn.Begin(ctx)
			defer func() {
				_ = tx.Rollback(ctx)
			}()
			p, pg, pq, err := LoadExamPaperDetailsById(ctx, tx, tt.epid, tt.withQuestions, tt.withAnswers, tt.withAnalysis)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if p == nil {
					t.Fatal("返回的考卷信息体为空")
				}
				if p.Name.String != "考卷单元测试试卷名" {
					t.Errorf("查询的考卷信息名称错误，实际：%v", p.Name.String)
				}
				if p.TotalScore.Float64 != 48 {
					t.Errorf("查询的考卷信息总分错误，实际：%v", p.TotalScore.Float64)
				}
				if p.QuestionCount.Int64 != 13 {
					t.Errorf("查询的考卷信息题目数量错误，实际：%v", p.QuestionCount.Int64)
				}
				if p.GroupCount.Int64 != 5 {
					t.Errorf("查询的考卷信息题组数量错误，实际：%v", p.GroupCount.Int64)
				}
				if tt.withQuestions {
					if len(pg) != 5 {
						t.Errorf("查询的题组数量错误，实际：%v", len(pg))
					}
					var qCount int
					for _, g := range pg {
						quesitons := pq[g.ID.Int64]
						for _, q := range quesitons {
							qCount++
							if tt.withAnswers {
								// 这里的答案是每一道题不一样的
								if len(q.Answers) == 0 {
									t.Errorf("查询出来的答案为空 预期应该是不为空的")
								}
							} else {
								if len(q.Answers) != 0 {
									t.Errorf("查询出来答案:%v 预期应该是为空的", q.Answers.String())
								}
							}

							if tt.withAnalysis {
								if !q.Analysis.Valid || q.Analysis.String != "hello" {
									t.Errorf("此时没有正确返回题目解析 实际返回：%v", q.Analysis.String)
								}
							} else {
								if q.Analysis.Valid && q.Analysis.String == "hello" {
									t.Errorf("此时应该不返回题目解析 实际返回")
								}
							}
						}
					}
					if qCount != 13 {
						t.Errorf("考卷题目数量错误")
					}
				}
			}
		})

		t.Cleanup(func() {
			// 这里的话先插入准备好的题库、题目跟试卷
			// 这里需要清除所有考卷信息
			s = `DELETE FROM assessuser.t_exam_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里还要再重新删除练习跟考试，然后再重新创建？
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_practice_submissions`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}

}

// // 测试根据考卷题目与信息，批量插入学生考卷
func TestGenerateAnswerQuestion(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()
	// 这里的话先删除之前创建的所有数据
	// 这里需要清除所有考卷信息
	s := `DELETE FROM assessuser.t_student_answers`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM assessuser.t_exam_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM t_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice_submissions`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	// 然后考试
	// 然后考试
	s = `DELETE FROM t_examinee`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_exam_session`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_info`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	initQuestion(t, conn)

	initPaper(t, conn)

	tx1, _ := conn.Begin(ctx)

	// 创建对应的考试，生成对应的场次ID
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_info (id,
			name, type, mode, creator, status
		) VALUES (
			$1,'考卷生成单元测试考试', '00', '00', $2, '02'
		)`,
		uid, uid)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}

	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_session (id,
			exam_id, session_num, paper_id,
			status, creator,mark_method
		) VALUES (
			$1,$2, 1, $3,'00',$4,$5
		)`,
		uid, uid, uid, uid, "02")

	baseID := uid.Int64
	ExamineeIDs := make([]int64, 0)
	for i := 0; i < 200; i++ {
		currentID := baseID + int64(i)
		// 创建考生记录
		_, err = tx1.Exec(ctx, `
		INSERT INTO t_examinee (id,
			student_id, exam_session_id, creator
		) VALUES (
			$1, $2, $3, $4
		)`,
			currentID, currentID, uid, uid)
		ExamineeIDs = append(ExamineeIDs, currentID)
		if err != nil {
			t.Fatal(err)
		}
	}

	//创建一个新的练习吧
	var practiceName string
	practiceName = "考卷生成逻辑测试练习"
	p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx1.Exec(ctx, p, uid.Int64, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	// 然后还要现场生成考卷
	var examPaperID1, examPaperID2 *int64

	// 这里直接使用函数插入
	examPaperID1, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Practice, uid.Int64, uid.Int64, 0, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	examPaperID2, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Exam, uid.Int64, 0, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID2 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}
	// 这里也要生成对应的practice_submission记录的
	s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt) VALUES (
			$1,$2,$3,$4,$5,$6	
		),($7,$8,$9,$10,$11,$12)`
	_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, 10087, uid, 2, *examPaperID1, uid, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = tx1.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	validError := errors.New("validation failed")
	submissionIDs := []int64{10086, 10087}

	// 现在全部都生成了 直接构造函数记录即可
	tests := []struct {
		name        string
		req         GenerateAnswerQuestionsRequest
		practice    bool
		expectedErr error
	}{
		{
			name: "正常1 能生成练习学生考卷 携带两个乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     true,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			practice:    true,
			expectedErr: nil,
		},
		{
			name: "正常2 能生成考试学生考卷 携带两个乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:      *examPaperID2,
				Category:         PaperCategory.Exam,
				ExamineeIDs:      ExamineeIDs,
				IsQuestionRandom: true,
				IsOptionRandom:   true,
			},
			expectedErr: nil,
		},
		{
			name: "正常3 能生成学生练习考卷 携带题组内乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     true,
				IsOptionRandom:       false,
				Attempt:              1,
			},
			practice:    true,
			expectedErr: nil,
		},
		{
			name: "正常4 能生成学生考试考卷 携带题组内乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:      *examPaperID2,
				Category:         PaperCategory.Exam,
				ExamineeIDs:      ExamineeIDs,
				IsQuestionRandom: true,
				IsOptionRandom:   false,
			},
			expectedErr: nil,
		},
		{
			name: "正常5 能生成学生练习考卷 携带题目选项乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			practice:    true,
			expectedErr: nil,
		},
		{
			name: "正常6 能生成学生考试考卷 携带题目选项乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:      *examPaperID2,
				Category:         PaperCategory.Exam,
				ExamineeIDs:      ExamineeIDs,
				IsQuestionRandom: false,
				IsOptionRandom:   true,
			},
			expectedErr: nil,
		},
		{
			name: "异常2 参数检测 非法考卷ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          -1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常3 参数检测 非法考卷用途",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             "04",
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常4 参数检测 非法练习提交记录ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, -2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常5 参数检测 非法考生记录ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, -2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常6 触发获取考卷错误",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          1000000,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: errors.New("select examPaper failed"),
		},
		{
			name: "异常7 强制触发批量插入错误 query",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: errors.New("failed batch insert student answer"),
		},
		{
			name: "正常7 强制触发批量插入错误 Rand",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          *examPaperID1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每一个用例再创建独立的事务
			tx, _ := conn.Begin(ctx)
			defer func() {
				_ = tx.Rollback(ctx)
			}()
			if containsString(tt.name, "异常7 ") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if containsString(tt.name, "正常7 ") {
				ctx = context.WithValue(ctx, "force-error", "Rand")
			} else {
				ctx = context.Background()
			}
			err = GenerateAnswerQuestion(ctx, tx, tt.req, uid.Int64)

			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if containsString(tt.name, "正常7") {
					// 这里就是没有东西的
				} else {
					// 这里就再去实现查询这个作答记录
					// 如果是练习的话，那就直接统计数量即可
					s = `SELECT COUNT(*) FROM t_student_answers `
					var count, expectedCount int
					err = tx.QueryRow(ctx, s).Scan(&count)
					if err != nil {
						t.Errorf("统计学生作答时出错：%v", err)
					}
					if tt.practice {
						expectedCount = 26
					} else {
						expectedCount = 2600
					}
					if count != expectedCount {
						t.Errorf("实际生成的学生作答数量与预期不符 预期%v 实际：%v", expectedCount, count)
					}

					// 这里查出所有的学生作答，对应查出对应的考卷
					// 如果是题组内部乱序，这里需要查询出题目来看看是否满足
					s = `SELECT "order", question_id , group_id , actual_options,actual_answers FROM t_student_answers`
					rows, err := tx.Query(ctx, s)
					if err != nil {
						t.Fatalf("无法执行查询学生作答数据：%v", err.Error())
					}
					var Answers []*cmn.TStudentAnswers
					for rows.Next() {
						var a cmn.TStudentAnswers
						err = rows.Scan(&a.Order, &a.QuestionID, &a.GroupID, &a.ActualOptions, &a.ActualAnswers)
						if err != nil {
							t.Fatalf("无法执行扫描学生作答数据：%v", err.Error())
						}
						Answers = append(Answers, &a)
					}

					s = `SELECT "order",id,options,answers,type FROM t_exam_paper_question`
					rows, err = tx.Query(ctx, s)
					if err != nil {
						t.Fatalf("无法执行查询原始考题数据：%v", err.Error())
					}
					var questions []*cmn.TExamPaperQuestion
					for rows.Next() {
						var q cmn.TExamPaperQuestion
						err = rows.Scan(&q.Order, &q.ID, &q.Options, &q.Answers, &q.Type)
						if err != nil {
							t.Fatalf("无法执行扫描原始考题数据：%v", err.Error())
						}
						questions = append(questions, &q)
					}
					// 这个表示这个order，然后值是出现的次数
					orderMap := make(map[int64]int)
					isQuestionRandom := false
					// 还要看这个乱序情况 是否真的乱序到了
					if tt.req.IsQuestionRandom {
						//这里就需要遍历循环查询 这个题目的order是否发生了变化
						for _, a := range Answers {
							// 这里取出对应的question_id
							for i := 0; i < len(questions); i++ {
								// 此时顺序相同，但是如果此时题目内容不一样的话，那就代表此时有乱序了
								if a.ID == questions[i].ID {
									// 这里就开始比较 如果发现不相同，就表示此时是成功的，但是要全部的order都出现过 并且是只出现一次
									orderMap[a.Order.Int64]++
								}
								if a.Order == questions[i].Order {
									// 这里就开始比较 如果发现不相同，就表示此时是成功的，但是要全部的order都出现过 并且是只出现一次
									if a.QuestionID != questions[i].ID {
										isQuestionRandom = true
									}
								}
							}
						}
						var needNum int
						if tt.req.Category == PaperCategory.Practice {
							needNum = 2
						} else {
							needNum = 200
						}
						for k, v := range orderMap {
							if v != needNum {
								t.Errorf("生成出来的题目order数据错误：order数量与题目数量不符合:实际order为%v 的出现次数为%v", k, v)
							}
						}
						if !isQuestionRandom {
							t.Errorf("没有成功题组内乱序成功")
						}
					}

					if tt.req.IsOptionRandom {
						var isOptionRandom bool
						// 如果是选项乱序的话 ， 这里需要对比这个题目的类型了
						for _, a := range Answers {
							q := &cmn.TExamPaperQuestion{}
							for i := 0; i < len(questions); i++ {
								// 这里需要找到这个题目
								if a.QuestionID == questions[i].ID {
									q = questions[i]
								}
							}
							// 这里去判断他是否为选择题
							if q.Type.String == QuestionCategory.FillInBlank || q.Type.String == QuestionCategory.ShortAnswer {

							} else {
								if a.ActualAnswers == nil || string(a.ActualAnswers) == "{}" || string(a.ActualAnswers) == "[]" {
									t.Errorf("乱序结果不符合预期 实际上：%v 选择题或判断题没有进行选项乱序", a.ActualAnswers.String())
								}
								// 这里还要去比较他是否已经成功交换顺序
								var aOptions []QuestionOption
								if err := json.Unmarshal(a.ActualOptions, &aOptions); err != nil {
									t.Errorf("json 反序列化失败：%v", err)
								}
								// 这里还要去比较他是否已经成功交换顺序
								var qOptions []QuestionOption
								if err := json.Unmarshal(q.Options, &qOptions); err != nil {
									t.Errorf("json 反序列化失败：%v", err)
								}

								for idx, v := range aOptions {
									// 总有可能是不可能的，但是又有可能是一样的 因此这里我只用一个变量，只要发现一个不一样就代表成功
									if v.Label == qOptions[idx].Label && v.Value != qOptions[idx].Value {
										isOptionRandom = true
										break
									}
								}
							}
						}
						if !isOptionRandom {
							t.Errorf("没有成功乱序成功")
						}
					}
				}
			}

			t.Cleanup(func() {
				// 这里直接全部删除所有学生作答
				s = `DELETE FROM t_student_answers`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
			})

		})

		t.Cleanup(func() {

			// 这里直接全部删除所有学生作答
			s = `DELETE FROM t_student_answers`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里的话先插入准备好的题库、题目跟试卷
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice_submissions`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_examinee`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_session`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_info`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}
}

// // 这里的话需要拿已经拥有的练习数据去
func TestLoadExamPaperDetailByUserId(t *testing.T) {

	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()
	//这里查看考卷详情，需要生成一张考卷就好了，由于练习会比较简单，我这里就直接使用练习去生成了
	s := `DELETE FROM assessuser.t_student_answers`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM assessuser.t_exam_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `DELETE FROM t_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_practice_submissions`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_examinee`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_session`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_info`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	initQuestion(t, conn)

	initPaper(t, conn)
	tx1, _ := conn.Begin(ctx)

	// 创建对应的考试，生成对应的场次ID
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_info (id,
			name, type, mode, creator, status
		) VALUES (
			$1,'考卷生成单元测试考试', '00', '00', $2, '02'
		)`,
		uid, uid)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_session (id,
			exam_id, session_num, paper_id,
			status, creator,mark_method
		) VALUES (
			$1,$2, 1, $3,'00',$4,$5
		)`,
		uid, uid, uid, uid, "02")

	// 创建考生记录 这里我创建100个考生
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_examinee (id,
			student_id, exam_session_id, creator
		) VALUES (
			$1, $2, $3, $4
		)`,
		uid, uid, uid, uid)

	// 然后还要现场生成考卷
	var examPaperID1, examPaperID2 *int64
	//创建一个新的练习吧
	var practiceName string
	practiceName = "考卷生成逻辑测试练习"
	p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx1.Exec(ctx, p, uid.Int64, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4)`
	_, err = tx1.Exec(ctx, s, 1, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	// 这里直接使用函数插入
	examPaperID1, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Practice, uid.Int64, uid.Int64, 0, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	examPaperID2, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Exam, uid.Int64, 0, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID2 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里调用一下
	_, pg, pq, err := LoadExamPaperDetailsById(ctx, tx1, *examPaperID2, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷逻辑出错：%v", err)
	}
	if len(pg) == 0 {
		t.Fatalf("此时生成的考卷题组为空")
	}
	if len(pq) == 0 {
		t.Fatalf("此时生成的考卷题目为空")
	}

	// 这里也要生成对应的practice_submission记录的
	s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt) VALUES (
			$1,$2,$3,$4,$5,$6	
		)`
	_, err = tx1.Exec(ctx, s, uid, uid, 1, *examPaperID1, uid, 1)
	if err != nil {
		t.Fatal(err)
	}
	submissionIDs := []int64{10086}
	ExamineeIDs := []int64{10086}
	// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
	now := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(now))
	// 获取考题
	_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx1, *examPaperID2, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	var Answers []*cmn.TStudentAnswers
	for _, id := range ExamineeIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID: q.ID,
					GroupID:    g.ID,
					Order:      null.IntFrom(int64(order)),
					ExamineeID: null.IntFrom(id),
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
					if err != nil {
						t.Fatal("乱序失败")
					}
				}
				Answers = append(Answers, answer)
				order++
			}
		}
	}

	t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
	// 限制学生答卷一次性插入的数量
	_limit := 500
	for i := 0; i < len(Answers); i += _limit {
		end := i + _limit
		if end > len(Answers) {
			end = len(Answers)
		}
		batchStudentAnswers := Answers[i:end]
		// 执行单次操作语句
		query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
		values := make([]string, 0, len(batchStudentAnswers))
		args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
		idx := 1
		for _, a := range batchStudentAnswers {
			placeholders := make([]string, 11)
			for i := 0; i < 11; i++ {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				idx++
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				PaperCategory.Exam, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		t.Logf("打印一下，能成功执行到插入学生作答前")
		_, err := tx1.Exec(ctx, insertQuery, args...)
		if err != nil {
			t.Fatalf("插入学生答卷失败：%v", err)
		}
	}

	// 获取考题
	_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	Answers = []*cmn.TStudentAnswers{}
	for _, id := range submissionIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID:           q.ID,
					GroupID:              g.ID,
					Order:                null.IntFrom(int64(order)),
					PracticeSubmissionID: null.IntFrom(id),
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
					if err != nil {
						t.Fatal("乱序失败")
					}
				}
				Answers = append(Answers, answer)
				order++
			}
		}
	}
	t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
	// 限制学生答卷一次性插入的数量
	for i := 0; i < len(Answers); i += _limit {
		end := i + _limit
		if end > len(Answers) {
			end = len(Answers)
		}
		batchStudentAnswers := Answers[i:end]
		// 执行单次操作语句
		query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
		values := make([]string, 0, len(batchStudentAnswers))
		args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
		idx := 1
		for _, a := range batchStudentAnswers {
			placeholders := make([]string, 11)
			for i := 0; i < 11; i++ {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				idx++
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				PaperCategory.Practice, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		_, err := tx1.Exec(ctx, insertQuery, args...)
		if err != nil {
			t.Fatalf("插入学生答卷失败:%v", err)
		}
	}
	qCount := 0
	s = `SELECT COUNT(*) FROM assessuser.t_exam_paper_question`
	err = tx1.QueryRow(ctx, s).Scan(&qCount)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("打印一下考卷题目数量：%v", qCount)
	if qCount == 0 {
		t.Fatal("考卷题目数量不符合")
	}

	// 这里查询一下这个学生答卷数量
	saCount := 0
	s = `SELECT COUNT(*) FROM assessuser.t_student_answers WHERE examinee_id=$1`
	err = tx1.QueryRow(ctx, s, ExamineeIDs[0]).Scan(&saCount)
	if err != nil {
		t.Fatal(err)
	}
	if saCount == 0 {
		t.Fatal("数量不符合")
	}
	// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis
	s = `UPDATE assessuser.t_student_answers SET answer = $1, answer_score = $2`
	answer := JSONText(`"hello"`)
	answerScore := 2.5
	_, err = tx1.Exec(ctx, s, answer, answerScore)
	if err != nil {
		t.Fatal(err)
	}

	err = tx1.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	//这里需要手动去给某个学生的答卷进行植入分数与答案
	//这里选择不去进行任何的乱序，此时就可以根据order去植入学生的答案与得分了 就是执行多次sql达到这个植入的目的， 这里就
	//保持这个结构，然后给出对应的答案去插入进去

	tests := []struct {
		name              string
		examPaper         int64
		psid              int64
		eid               int64
		withStudentAnswer bool
		withAnswer        bool
		withAnswerScore   bool
		expectedErr       error
	}{
		// 下面可以通过两种方式去触发这个函数的 第一种就是练习
		{
			// 满足学生查询自己作答详情时 能获取试卷、作答内容、批改分数、解析、AI解析
			name:              "正常1 携带练习学生答案 携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: true,
			withAnswer:        true,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生继续作答获取上一次的作答
			name:              "正常2 不携带练习学生答案 携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: false,
			withAnswer:        true,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常3 不携带练习学生答案 不携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常4 不携带练习学生答案 不携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常5 不携带练习学生答案 携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: false,
			withAnswer:        true,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常6 携带练习学生答案 不携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID1,
			psid:              submissionIDs[0],
			eid:               0,
			withStudentAnswer: true,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生查询自己作答详情时 能获取试卷、作答内容、批改分数、解析、AI解析
			name:              "正常7 携带考试学生答案 携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: true,
			withAnswer:        true,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生继续作答获取上一次的作答
			name:              "正常2 不携带考试学生答案 携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        true,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常3 不携带考试学生答案 不携带题目答案 携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   true,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常4 不携带考试学生答案 不携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常5 不携带考试学生答案 携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        true,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常6 携带考试学生答案 不携带题目答案 不携带学生答案批改后分数",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: true,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "正常7 测试覆盖率 mock数据返回",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: true,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       nil,
		},
		{
			// 满足学生查询自己作答详情时 能获取试卷、作答内容、批改分数、解析、AI解析
			name:              "异常1 参数检测 非法考卷ID",
			examPaper:         0,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: true,
			withAnswer:        true,
			withAnswerScore:   true,
			expectedErr:       errors.New("invalid examPaperId ID param"),
		},
		{
			// 满足学生继续作答获取上一次的作答
			name:              "异常2 参数检测 非法练习与考试ID",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               0,
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("invalid pSubmissionId、examineeId  param"),
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "强制异常1 数据库错误 query",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("query statement failed"),
		},
		{
			// 此时随便也测试了两个同时大于0的话，还是会选取考试
			name:              "强制异常2 数据库错误 scan",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: true,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("scan student answer failed"),
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "强制异常3 数据库错误 扫描出来学生答卷为空 emptyAnswer",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("query empty student answer, please checkout generate exam paper or student answer"),
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "强制异常4 LoadExamPaperDetailsById 触发错误 json",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("unmarshal group data failed"),
		},
		{
			// 满足学生进入练习或考试时获取作答试卷（若已乱序，则每个学生试卷内容都会被乱序过）
			name:              "强制异常5 强制让答卷与考题映射不准确 exist",
			examPaper:         *examPaperID2,
			psid:              0,
			eid:               submissionIDs[0],
			withStudentAnswer: false,
			withAnswer:        false,
			withAnswerScore:   false,
			expectedErr:       errors.New("not found in exam paper"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()

			// 开一个独立的事务
			tx, _ := conn.Begin(ctx)

			defer tx.Rollback(ctx)
			// 每一个用例再创建独立的事务
			if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if containsString(tt.name, "强制异常2") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "emptyAnswer")
			} else if containsString(tt.name, "强制异常4") {
				ctx = context.WithValue(ctx, "force-error", "json")
			} else if containsString(tt.name, "强制异常5") {
				ctx = context.WithValue(ctx, "force-error", "exist")
			} else if containsString(tt.name, "正常7") {
				ctx = context.WithValue(ctx, "test", "normal-resp")
			} else {
				ctx = context.Background()
			}
			p, pg, pq, err := LoadExamPaperDetailByUserId(ctx, tx, tt.examPaper, tt.psid, tt.eid, tt.withStudentAnswer, tt.withAnswer, tt.withAnswerScore)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if p == nil {
					t.Errorf("返回的试卷信息指针为空：异常")
				}
				if containsString(tt.name, "正常7") {

				} else {
					if p.Name.String != "考卷单元测试试卷名" {
						t.Errorf("获取的考卷名称不一致 实际为：%v", p.Name.String)
					}

					// 这里先查询出所有的考题了，不过已经是有了
					for _, qList := range pq {
						for _, q := range qList {
							// 这里开始检测
							if tt.withStudentAnswer {
								// 这里就要解码出来，然后判断是否与他相等
								// 3. 严格顺序验证
								expected := []byte(`"hello"`)
								if !bytes.Equal(q.StudentAnswer, expected) {
									t.Errorf("此时没有正确返回学生作答内容")
								}
							} else {
								expected := []byte(`"hello"`)
								if bytes.Equal(q.StudentAnswer, expected) {
									t.Errorf("此时不应该返回学生作答内容")
								}
							}
							if tt.withAnswerScore {
								if !q.StudentScore.Valid || q.StudentScore.Float64 != 2.5 {
									t.Errorf("此时没有正确返回学生作答分数")
								}
								if !q.Analysis.Valid || q.Analysis.String != "hello" {
									t.Errorf("此时没有正确返回题目解析 实际返回：%v", q.Analysis.String)
								}
							} else {
								if q.StudentScore.Valid && q.StudentScore.Float64 == 2.5 {
									t.Errorf("此时不应该返回学生作答分数 却返回了 与预期不符")
								}
								if q.Analysis.Valid && q.Analysis.String == "hello" {
									t.Errorf("此时不应该返回题目解析 却返回了 与预期不符")
								}
							}
							if tt.withAnswer {
								// 这里的答案是每一道题不一样的
								if len(q.Answers) == 0 {
									t.Errorf("查询出来的答案为空 预期应该是不为空的")
								}
							} else {
								if len(q.Answers) != 0 {
									t.Errorf("查询出来答案:%v 预期应该是为空的", q.Answers.String())
								}
							}
						}
					}
					for GroupID, g := range pg {
						if GroupID != g.ID.Int64 {
							t.Errorf("返回的题组map不符合预期")
						}
					}
				}
			}
		})
		t.Cleanup(func() {
			t.Logf("这里只会打印一次")
			s = `DELETE FROM assessuser.t_student_answers`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里需要清除所有考卷信息
			s = `DELETE FROM assessuser.t_exam_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里的话先插入准备好的题库、题目跟试卷
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_practice_submissions`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			// 再删除练习
			s = `DELETE FROM t_practice`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_examinee`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_session`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_info`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}

}

// 测试级联是否成功的，前置条件，需要设置好数据库考卷、考卷题组、考卷题目、学生作答的外键约束及其级联条件
func TestDeleteExamPaperById(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()

	s := `DELETE FROM assessuser.t_student_answers`
	_, err := conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 这里需要清除所有考卷信息
	s = `DELETE FROM assessuser.t_exam_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_exam_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 这里的话先插入准备好的题库、题目跟试卷
	// 先删除
	s = `DELETE FROM t_paper_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper_group`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	s = `DELETE FROM t_paper`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_question`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除题库
	s = `DELETE FROM t_question_bank`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	s = `DELETE FROM t_practice_submissions`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 再删除练习
	s = `DELETE FROM t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	// 再删除练习
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_examinee`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_session`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 然后考试
	s = `DELETE FROM t_exam_info`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	// 全部删除之后，就开始创建考试跟练习

	initQuestion(t, conn)

	initPaper(t, conn)

	tx1, _ := conn.Begin(ctx)

	// 创建对应的考试，生成对应的场次ID
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_info (id,
			name, type, mode, creator, status
		) VALUES (
			$1,'考卷生成单元测试考试', '00', '00', $2, '02'
		)`,
		uid, uid)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_exam_session (id,
			exam_id, session_num, paper_id,
			status, creator,mark_method
		) VALUES (
			$1,$2, 1, $3,'00',$4,$5
		)`,
		uid, uid, uid, uid, "02")

	// 创建考生记录 这里我创建100个考生
	_, err = tx1.Exec(ctx, `
		INSERT INTO t_examinee (id,
			student_id, exam_session_id, creator
		) VALUES (
			$1, $2, $3, $4
		)`,
		uid, uid, uid, uid)

	// 然后还要现场生成考卷
	var examPaperID1, examPaperID2 *int64
	//创建一个新的练习吧
	var practiceName string
	practiceName = "考卷生成逻辑测试练习"
	p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx1.Exec(ctx, p, uid.Int64, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4)`
	_, err = tx1.Exec(ctx, s, 1, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	// 这里直接使用函数插入
	examPaperID1, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Practice, uid.Int64, uid.Int64, 0, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	examPaperID2, _, err = GenerateExamPaper(context.Background(), tx1, PaperCategory.Exam, uid.Int64, 0, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID2 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里调用一下
	_, pg, pq, err := LoadExamPaperDetailsById(ctx, tx1, *examPaperID2, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷逻辑出错：%v", err)
	}
	if len(pg) == 0 {
		t.Fatalf("此时生成的考卷题组为空")
	}
	if len(pq) == 0 {
		t.Fatalf("此时生成的考卷题目为空")
	}

	// 这里也要生成对应的practice_submission记录的
	s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt) VALUES (
			$1,$2,$3,$4,$5,$6	
		)`
	_, err = tx1.Exec(ctx, s, uid, uid, 1, *examPaperID1, uid, 1)
	if err != nil {
		t.Fatal(err)
	}
	submissionIDs := []int64{10086}
	ExamineeIDs := []int64{10086}
	// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
	now := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(now))
	// 获取考题
	_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx1, *examPaperID2, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	var Answers []*cmn.TStudentAnswers
	for _, id := range ExamineeIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID: q.ID,
					GroupID:    g.ID,
					Order:      null.IntFrom(int64(order)),
					ExamineeID: null.IntFrom(id),
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
					if err != nil {
						t.Fatal("乱序失败")
					}
				}
				Answers = append(Answers, answer)
				order++
			}
		}
	}

	t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
	// 限制学生答卷一次性插入的数量
	_limit := 500
	for i := 0; i < len(Answers); i += _limit {
		end := i + _limit
		if end > len(Answers) {
			end = len(Answers)
		}
		batchStudentAnswers := Answers[i:end]
		// 执行单次操作语句
		query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
		values := make([]string, 0, len(batchStudentAnswers))
		args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
		idx := 1
		for _, a := range batchStudentAnswers {
			placeholders := make([]string, 11)
			for i := 0; i < 11; i++ {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				idx++
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				PaperCategory.Exam, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		t.Logf("打印一下，能成功执行到插入学生作答前")
		_, err := tx1.Exec(ctx, insertQuery, args...)
		if err != nil {
			t.Fatalf("插入学生答卷失败：%v", err)
		}
	}

	// 获取考题
	_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	Answers = []*cmn.TStudentAnswers{}
	for _, id := range submissionIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID:           q.ID,
					GroupID:              g.ID,
					Order:                null.IntFrom(int64(order)),
					PracticeSubmissionID: null.IntFrom(id),
				}
				actualAnswers := q.Answers
				actualOptions := q.Options
				if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
					answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
					if err != nil {
						t.Fatal("乱序失败")
					}
				}
				Answers = append(Answers, answer)
				order++
			}
		}
	}
	t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
	// 限制学生答卷一次性插入的数量
	for i := 0; i < len(Answers); i += _limit {
		end := i + _limit
		if end > len(Answers) {
			end = len(Answers)
		}
		batchStudentAnswers := Answers[i:end]
		// 执行单次操作语句
		query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
		values := make([]string, 0, len(batchStudentAnswers))
		args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
		idx := 1
		for _, a := range batchStudentAnswers {
			placeholders := make([]string, 11)
			for i := 0; i < 11; i++ {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				idx++
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				PaperCategory.Practice, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		_, err := tx1.Exec(ctx, insertQuery, args...)
		if err != nil {
			t.Fatalf("插入学生答卷失败:%v", err)
		}
	}
	qCount := 0
	s = `SELECT COUNT(*) FROM assessuser.t_exam_paper_question`
	err = tx1.QueryRow(ctx, s).Scan(&qCount)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("打印一下考卷题目数量：%v", qCount)
	if qCount == 0 {
		t.Fatal("考卷题目数量不符合")
	}

	// 这里查询一下这个学生答卷数量
	saCount := 0
	s = `SELECT COUNT(*) FROM assessuser.t_student_answers WHERE examinee_id=$1`
	err = tx1.QueryRow(ctx, s, ExamineeIDs[0]).Scan(&saCount)
	if err != nil {
		t.Fatal(err)
	}
	if saCount == 0 {
		t.Fatal("数量不符合")
	}
	// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis
	s = `UPDATE assessuser.t_student_answers SET answer = $1, answer_score = $2`
	answer := JSONText(`"hello"`)
	answerScore := 2.5
	_, err = tx1.Exec(ctx, s, answer, answerScore)
	if err != nil {
		t.Fatal(err)
	}

	err = tx1.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// 这里同样去准备一样的考试跟练习数据，包括学生作答了

	tests := []struct {
		name           string
		examSessionIDs []int64
		practiceIDs    []int64
		expectedEPNum  int
		expectedPPNum  int
		expectedError  error
	}{
		{
			name:           "正常1 将考试与练习一起删除",
			examSessionIDs: []int64{uid.Int64},
			practiceIDs:    []int64{uid.Int64},
			expectedEPNum:  0,
			expectedPPNum:  0,
			expectedError:  nil,
		},
		{
			name:           "正常2 将考试旗下的考卷、答卷删除",
			examSessionIDs: []int64{uid.Int64},
			practiceIDs:    []int64{},
			expectedEPNum:  0,
			expectedPPNum:  1,
			expectedError:  nil,
		},
		{
			name:           "正常3 将练习旗下的考卷、答卷删除",
			examSessionIDs: []int64{},
			practiceIDs:    []int64{uid.Int64},
			expectedEPNum:  1,
			expectedPPNum:  0,
			expectedError:  nil,
		},
		{
			name:           "异常1 参数检测 无删除对象",
			examSessionIDs: []int64{},
			practiceIDs:    []int64{},
			expectedError:  errors.New("invalid examSessionIDs or practiceIDs param"),
		},
		{
			name:           "异常2 参数检测 空删除对象",
			examSessionIDs: nil,
			practiceIDs:    nil,
			expectedError:  errors.New("invalid examSessionIDs or practiceIDs param"),
		},
		{
			name:           "异常3 参数检测 传入列有非法对象",
			examSessionIDs: []int64{uid.Int64, -1, 0, 10087},
			practiceIDs:    []int64{uid.Int64, -4, 1000, 10087},
			expectedError:  errors.New("要删除的考试场次或者练习考卷信息的ID非法 错误如下"),
		},
		{
			name:           "异常4 强制触发删除失败逻辑 delete",
			examSessionIDs: []int64{uid.Int64},
			practiceIDs:    []int64{uid.Int64},
			expectedError:  errors.New("级联删除对应考试场次与练习考卷、学生答卷数据失败"),
		},
	}

	for _, tt := range tests {
		// 遍历
		t.Run(tt.name, func(t *testing.T) {

			tx, _ := conn.Begin(ctx)

			defer func() {
				err = tx.Rollback(ctx)
			}()
			// 构建对应上下文
			ctx = context.Background()
			if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "delete")
			} else {
				ctx = context.Background()
			}

			err = DeleteExamPaperById(ctx, tx, tt.examSessionIDs, tt.practiceIDs)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("报错：%v与预期错误：%v不符合", err, tt.expectedError)
				}
			} else {
				var count int
				if len(tt.examSessionIDs) > 0 {
					// 同样要出现这么多的考卷数量
					s = `SELECT COUNT(*) FROM t_exam_paper WHERE exam_session_id = $1`
					err = tx.QueryRow(ctx, s, tt.examSessionIDs[0]).Scan(&count)
					if err != nil {
						t.Errorf("查询此时为考试用的考卷数量失败：%v", err)
					}
					if count != tt.expectedEPNum {
						t.Errorf("查询此时为考试用的考卷数量不与预期：%v,实际：%v：", tt.expectedEPNum, count)
					}

					s = `SELECT COUNT(*) FROM t_student_answers WHERE type = $1`
					err = tx.QueryRow(ctx, s, PaperCategory.Exam).Scan(&count)
					if err != nil {
						t.Errorf("查询此时为练习用的考卷数量失败：%v", err)
					}
					if count != tt.expectedEPNum*13 {
						t.Errorf("查询此时为考试用的考卷数量不与预期：%v,实际：%v：", tt.expectedEPNum*13, count)
					}
				}
				if len(tt.practiceIDs) > 0 {
					s = `SELECT COUNT(*) FROM t_exam_paper WHERE practice_id = $1`
					err = tx.QueryRow(ctx, s, tt.practiceIDs[0]).Scan(&count)
					if err != nil {
						t.Errorf("查询此时为练习用的考卷数量失败：%v", err)
					}
					if count != tt.expectedPPNum {
						t.Errorf("查询此时为考试用的考卷数量不与预期：%v,实际：%v：", tt.expectedPPNum, count)
					}

					s = `SELECT COUNT(*) FROM t_student_answers WHERE type = $1`
					err = tx.QueryRow(ctx, s, PaperCategory.Practice).Scan(&count)
					if err != nil {
						t.Errorf("查询此时为练习用的考卷数量失败：%v", err)
					}
					if count != tt.expectedPPNum*13 {
						t.Errorf("查询此时练习答卷数量不与预期：%v,实际：%v：", tt.expectedPPNum*13, count)
					}
				}
			}

			t.Cleanup(func() {
				// 所有重新创建
				s := `DELETE FROM assessuser.t_student_answers`
				_, err := conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
				// 这里需要清除所有考卷信息
				s = `DELETE FROM assessuser.t_exam_paper_question`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				s = `DELETE FROM assessuser.t_exam_paper_group`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				s = `DELETE FROM assessuser.t_exam_paper`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
				s = `DELETE FROM t_practice_submissions`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 再删除练习
				s = `DELETE FROM t_practice_student`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}

				// 再删除练习
				s = `DELETE FROM t_practice`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 然后考试
				s = `DELETE FROM t_examinee`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 然后考试
				s = `DELETE FROM t_exam_session`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 然后考试
				s = `DELETE FROM t_exam_info`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}

				tx2, _ := conn.Begin(ctx)

				// 创建对应的考试，生成对应的场次ID
				_, err = tx2.Exec(ctx, `
		INSERT INTO t_exam_info (id,
			name, type, mode, creator, status
		) VALUES (
			$1,'考卷生成单元测试考试', '00', '00', $2, '02'
		)`,
					uid, uid)
				if err != nil {
					t.Fatalf("创建测试考试失败: %v", err)
				}
				_, err = tx2.Exec(ctx, `
		INSERT INTO t_exam_session (id,
			exam_id, session_num, paper_id,
			status, creator,mark_method
		) VALUES (
			$1,$2, 1, $3,'00',$4,$5
		)`,
					uid, uid, uid, uid, "02")

				// 创建考生记录 这里我创建100个考生
				_, err = tx2.Exec(ctx, `
		INSERT INTO t_examinee (id,
			student_id, exam_session_id, creator
		) VALUES (
			$1, $2, $3, $4
		)`,
					uid, uid, uid, uid)

				// 然后还要现场生成考卷
				var examPaperID1, examPaperID2 *int64
				//创建一个新的练习吧
				var practiceName string
				practiceName = "考卷生成逻辑测试练习"
				p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = tx2.Exec(ctx, p, uid.Int64, practiceName, "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4)`
				_, err = tx2.Exec(ctx, s, 1, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
				// 这里直接使用函数插入
				examPaperID1, _, err = GenerateExamPaper(context.Background(), tx2, PaperCategory.Practice, uid.Int64, uid.Int64, 0, uid.Int64, false)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				examPaperID2, _, err = GenerateExamPaper(context.Background(), tx2, PaperCategory.Exam, uid.Int64, 0, uid.Int64, uid.Int64, false)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID2 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里调用一下
				_, pg, pq, err := LoadExamPaperDetailsById(ctx, tx2, *examPaperID2, true, true, true)
				if err != nil {
					t.Fatalf("查询考卷逻辑出错：%v", err)
				}
				if len(pg) == 0 {
					t.Fatalf("此时生成的考卷题组为空")
				}
				if len(pq) == 0 {
					t.Fatalf("此时生成的考卷题目为空")
				}

				// 这里也要生成对应的practice_submission记录的
				s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt) VALUES (
			$1,$2,$3,$4,$5,$6	
		)`
				_, err = tx2.Exec(ctx, s, uid, uid, 1, *examPaperID1, uid, 1)
				if err != nil {
					t.Fatal(err)
				}
				submissionIDs := []int64{10086}
				ExamineeIDs := []int64{10086}
				// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
				now := time.Now().UnixMilli()
				r := rand.New(rand.NewSource(now))
				// 获取考题
				_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx2, *examPaperID2, true, true, true)
				if err != nil {
					t.Fatalf("查询考卷题目错误：%v", err)
				}
				var Answers []*cmn.TStudentAnswers
				for _, id := range ExamineeIDs {
					order := 1
					for _, g := range pg {
						pqList := pq[g.ID.Int64]
						r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
						for _, q := range pqList {
							answer := &cmn.TStudentAnswers{
								QuestionID: q.ID,
								GroupID:    g.ID,
								Order:      null.IntFrom(int64(order)),
								ExamineeID: null.IntFrom(id),
							}
							actualAnswers := q.Answers
							actualOptions := q.Options
							if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
								answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
								if err != nil {
									t.Fatal("乱序失败")
								}
							}
							Answers = append(Answers, answer)
							order++
						}
					}
				}

				t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
				// 限制学生答卷一次性插入的数量
				_limit := 500
				for i := 0; i < len(Answers); i += _limit {
					end := i + _limit
					if end > len(Answers) {
						end = len(Answers)
					}
					batchStudentAnswers := Answers[i:end]
					// 执行单次操作语句
					query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
					values := make([]string, 0, len(batchStudentAnswers))
					args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
					idx := 1
					for _, a := range batchStudentAnswers {
						placeholders := make([]string, 11)
						for i := 0; i < 11; i++ {
							placeholders[i] = fmt.Sprintf("$%d", idx)
							idx++
						}
						values = append(values, "("+strings.Join(placeholders, ",")+")")
						args = append(args,
							PaperCategory.Exam, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
							uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
						)
					}
					insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
					t.Logf("打印一下，能成功执行到插入学生作答前")
					_, err := tx2.Exec(ctx, insertQuery, args...)
					if err != nil {
						t.Fatalf("插入学生答卷失败：%v", err)
					}
				}

				// 获取考题
				_, pg, pq, err = LoadExamPaperDetailsById(ctx, tx2, *examPaperID1, true, true, true)
				if err != nil {
					t.Fatalf("查询考卷题目错误：%v", err)
				}
				Answers = []*cmn.TStudentAnswers{}
				for _, id := range submissionIDs {
					order := 1
					for _, g := range pg {
						pqList := pq[g.ID.Int64]
						r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
						for _, q := range pqList {
							answer := &cmn.TStudentAnswers{
								QuestionID:           q.ID,
								GroupID:              g.ID,
								Order:                null.IntFrom(int64(order)),
								PracticeSubmissionID: null.IntFrom(id),
							}
							actualAnswers := q.Answers
							actualOptions := q.Options
							if q.Type.String == QuestionCategory.SingleChoice || q.Type.String == QuestionCategory.MultiChoice || q.Type.String == QuestionCategory.TrueFalse {
								answer.ActualAnswers, answer.ActualOptions, err = shuffleOptionsAndMapAnswers(r, q.ID.Int64, actualOptions, actualAnswers)
								if err != nil {
									t.Fatal("乱序失败")
								}
							}
							Answers = append(Answers, answer)
							order++
						}
					}
				}
				t.Logf("打印一下要插入学生作答的数组长度：%v", len(Answers))
				// 限制学生答卷一次性插入的数量
				for i := 0; i < len(Answers); i += _limit {
					end := i + _limit
					if end > len(Answers) {
						end = len(Answers)
					}
					batchStudentAnswers := Answers[i:end]
					// 执行单次操作语句
					query := `
		INSERT INTO assessuser.t_student_answers (
			type, examinee_id, practice_submission_id, question_id,
			creator, create_time, update_time, group_id, "order",
			actual_options, actual_answers
		) VALUES %s
	`
					values := make([]string, 0, len(batchStudentAnswers))
					args := make([]interface{}, 0, len(batchStudentAnswers)*11) // 11个字段
					idx := 1
					for _, a := range batchStudentAnswers {
						placeholders := make([]string, 11)
						for i := 0; i < 11; i++ {
							placeholders[i] = fmt.Sprintf("$%d", idx)
							idx++
						}
						values = append(values, "("+strings.Join(placeholders, ",")+")")
						args = append(args,
							PaperCategory.Practice, a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
							uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
						)
					}
					insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
					_, err := tx2.Exec(ctx, insertQuery, args...)
					if err != nil {
						t.Fatalf("插入学生答卷失败:%v", err)
					}
				}
				qCount := 0
				s = `SELECT COUNT(*) FROM assessuser.t_exam_paper_question`
				err = tx2.QueryRow(ctx, s).Scan(&qCount)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("打印一下考卷题目数量：%v", qCount)
				if qCount == 0 {
					t.Fatal("考卷题目数量不符合")
				}

				// 这里查询一下这个学生答卷数量
				saCount := 0
				s = `SELECT COUNT(*) FROM assessuser.t_student_answers WHERE examinee_id=$1`
				err = tx2.QueryRow(ctx, s, ExamineeIDs[0]).Scan(&saCount)
				if err != nil {
					t.Fatal(err)
				}
				if saCount == 0 {
					t.Fatal("数量不符合")
				}
				// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis
				s = `UPDATE assessuser.t_student_answers SET answer = $1, answer_score = $2`
				answer := JSONText(`"hello"`)
				answerScore := 2.5
				_, err = tx2.Exec(ctx, s, answer, answerScore)
				if err != nil {
					t.Fatal(err)
				}

				err = tx2.Commit(context.Background())
				if err != nil {
					t.Fatal(err)
				}
			})
		})

		t.Cleanup(func() {

			s := `DELETE FROM assessuser.t_student_answers`
			_, err := conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里需要清除所有考卷信息
			s = `DELETE FROM assessuser.t_exam_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_exam_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里的话先插入准备好的题库、题目跟试卷
			// 先删除
			s = `DELETE FROM t_paper_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper_group`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			s = `DELETE FROM t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_question`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除题库
			s = `DELETE FROM t_question_bank`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			s = `DELETE FROM t_practice_submissions`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 再删除练习
			s = `DELETE FROM t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}

			// 再删除练习
			s = `DELETE FROM t_practice`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_examinee`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_session`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 然后考试
			s = `DELETE FROM t_exam_info`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}

}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 这里初始化题目，到达其他的函数，我这里一样可以进行删除
func initQuestion(t *testing.T, tx *pgxpool.Pool) {
	qb := `INSERT INTO t_question_bank  (id,type,name,creator,create_time,status) VALUES($1, $2, $3, $4, $5, $6)`
	_, err := tx.Exec(ctx, qb, uid, "00", "考卷单元测试题库", uid, now, "00")
	if err != nil {
		t.Fatal(err)
	}
	q := `INSERT INTO t_question (id,type,content,options,answers,score,analysis,title,creator,status,belong_to,difficulty) VALUES ($1, $2, $3, $4, $5, $6,$7,$8,$9,$10,$11,$12)`
	qArgs := [][]interface{}{
		{1, "00", "<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			`[
				{
            		"label": "A",
            		"value": "<p><span style=\"font-family: 等线; font-size: 12pt\">瀑布模型</span></p>"
        		},
        		{
            		"label": "B",
            		"value": "<p><span style=\"font-family: 等线; font-size: 12pt\">喷泉模型</span></p>"
        		},
        		{
            		"label": "C",
            		"value": "<p><span style=\"font-family: 等线; font-size: 12pt\">螺旋模型</span></p>"
        		},
        		{
            		"label": "D",
            		"value": "<p><span style=\"font-family: 等线; font-size: 12pt\">增量模型</span></p>"
        		}
			]`,
			`["A", "D"]`, 2, "hello", nil, uid, "00", uid, 1,
		},
		{2, "00", "<p><span style=\"font-size: 12pt\">H3C公司的总部位于哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">2</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">3</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">4</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">5</span></p>"}]`,
			`["A", "D"]`, 2, "hello", nil, uid, "00", uid, 1,
		},
		{3, "02", "<p><span style=\"font-size: 12pt\">H3C公司的分部遍布哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">广州</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">上海</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">北京</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">深圳</span></p>"}]`,
			`["A", "B","C","D"]`, 5, "hello", nil, uid, "00", uid, 1,
		},
		{4, "02", "<p><span style=\"font-size: 12pt\"> 以下哪些是常见的网络设备品牌？</p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">华为</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">思科</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\"> Juniper</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">中兴</span></p>"}]`,
			`["A", "B"]`, 5, "hello", nil, uid, "00", uid, 1,
		},
		{5, "02", "<p><span style=\"font-size: 12pt\">以下属于H3C主要产品线的有哪些？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">路由器</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">交换机</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">防火墙</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">服务器</span></p>"}]`,
			`["A","B","C"]`, 5, "hello", nil, uid, "00", uid, 1,
		},
		{6, "00", "<p><span style=\"font-size: 12pt\">以下哪些属于H3C的核心技术领域？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">云计算</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">大数据</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">人工智能</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">物联网</span></p>"}]`,
			`["A"]`, 2, "hello", nil, uid, "00", uid, 1,
		},
		{7, "04", "<p><span style=\"font-size: 12pt\">在H3C设备上，'undo shutdown'命令可以启用一个物理接口。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, "hello", nil, uid, "00", uid, 1,
		},
		{8, "04", "<p><span style=\"font-size: 12pt\">H3C交换机的默认管理VLAN编号是1。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, "hello", nil, uid, "00", uid, 1,
		},
		{9, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "螺旋模型","grading_rule": "答案必须准确匹配“螺旋模型”","alternative_answers": []}]`, 5, "hello", nil, uid, "00", uid, 1,
		},
		{10, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">软件开发中，用于描述系统功能的文档是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "需求规格说明书","grading_rule": "答案必须准确匹配“需求规格说明书”","alternative_answers": []}]`, 5, "hello", nil, uid, "00", uid, 1,
		},
		{11, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">面向对象编程的三大基本特性分别是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 1,"answer": "封装","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 2,"score": 1,"answer": "继承","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 3,"score": 1,"answer": "多态","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []}]`,
			3, "hello", nil, uid, "00", uid, 1,
		},
		{12, "08", "<p>简述计算机病毒的传播途径。</p>", nil, `[{"index": 1,"score": 5,"answer": "通过网络下载、移动存储设备、电子邮件等方式传播","grading_rule": "答出3种主要传播方式即可得满分","alternative_answers": []}]`,
			5, "hello", nil, uid, "00", uid, 1,
		},
		{13, "08", "<p>简述操作系统的主要功能。</p>", nil, `[{"index": 1,"score": 5,"answer": "进程管理、内存管理、文件管理、设备管理、作业管理","grading_rule": "答出3种主要功能即可得满分","alternative_answers": []}]`,
			5, "hello", nil, uid, "00", uid, 1,
		},
	}
	// 1. 创建 Batch 对象
	batch := &pgx.Batch{}
	// 2. 遍历 qArgs，将每条插入语句加入 Batch
	for _, args := range qArgs {
		batch.Queue(q, args...) // q 是你的 INSERT SQL 语句
	}
	// 3. 在事务中发送 Batch
	br := tx.SendBatch(context.Background(), batch)
	defer br.Close()

	// 4. 检查每条执行结果（可选，建议保留）
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			t.Errorf("插入题库错误:%v", err)
		}
	}
}

func initPaper(t *testing.T, tx *pgxpool.Pool) {
	// 成功插入之后，就可以开始创建试卷跟题组了
	// 创建试卷
	paper := &cmn.TPaper{
		Name:              null.StringFrom("考卷单元测试试卷名"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("02"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("考卷单元测试服务"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           uid,
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         uid,
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
	}
	p := `INSERT INTO t_paper
			(id,name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := tx.Exec(ctx, p, uid, paper.Name.String,
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
	)
	if err != nil {
		t.Errorf("创建试卷失败：%v", err)
		return
	}

	groupNames := []string{"一、单选题", "二、多选题", "三、判断题", "四、填空题", "五、简答题"}
	groupIDMap := make(map[string]int64)
	groupIDs := []int64{}

	// 创建题组
	for i, name := range groupNames {
		var groupID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO t_paper_group
				(paper_id, name, "order", creator, create_time, updated_by, update_time, status)
			VALUES
				($1, $2, $3, $4, $5, $4, $5, $6)
			RETURNING id`,
			uid,
			name,
			i+1,
			uid,
			now,
			"00",
		).Scan(&groupID)

		if err != nil {
			t.Errorf("创建试卷题组失败:%v", err)
		}

		groupIDs = append(groupIDs, groupID)
		groupIDMap[fmt.Sprintf("g%d", i)] = groupID
	}

	pq := `INSERT INTO t_paper_question (bank_question_id, group_id, "order", score, creator, status, sub_score) VALUES
	($1, $2, $3, $4, $5, $6,$7)`
	_, err = tx.Exec(ctx, pq, 1, groupIDs[0], 1, 2, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 2, groupIDs[0], 2, 2, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 6, groupIDs[0], 3, 2, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 3, groupIDs[1], 4, 5, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 4, groupIDs[1], 5, 5, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 5, groupIDs[1], 6, 5, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 7, groupIDs[2], 7, 2, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 8, groupIDs[2], 8, 2, uid, "00", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 9, groupIDs[3], 9, 5, uid, "00", types.JSONText(`[5]`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 10, groupIDs[3], 10, 5, uid, "00", types.JSONText(`[5]`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 11, groupIDs[3], 11, 3, uid, "00", types.JSONText(`[1,1,1]`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 12, groupIDs[4], 12, 5, uid, "00", types.JSONText(`[5]`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 13, groupIDs[4], 13, 5, uid, "00", types.JSONText(`[5]`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitData(t *testing.T) {
	t.Run("c直接生成数据", func(t *testing.T) {
		if z == nil {
			cmn.ConfigureForTest()
		}
		conn := cmn.GetPgxConn()

		initQuestion(t, conn)

		initPaper(t, conn)
	})
}
