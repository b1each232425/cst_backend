/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷数据库层单元测试
 * @Date: 2025-07-28 19:55:28
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-30 16:19:21
 */
package examPaper

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

var (
	ctx = context.Background()
	now = time.Now().UnixMilli()
	uid = null.IntFrom(100)
)

func TestLoadPaperTemplateById(t *testing.T) {
	// 这里需要先准备合适的数据，先构成一张完整的试卷
	// 创建对应的题库

	//qb := `INSERT INTO t_question_bank  (type,name,creator,create_time,status,access_mode) VALUES($1, $2, $3, $4, $5, $6) RETURNING id`
	if z == nil {
		cmn.ConfigureForTest()
	}
	paper := &cmn.TPaper{
		Name:              null.StringFrom("考卷单元测试试卷名"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("02"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("考卷专门服务"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           uid,
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         uid,
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
		AccessMode:        null.StringFrom("00"), // 默认访问模式
	}
	groupNames := []string{"一、单选题", "二、多选题", "三、判断题", "四、填空题", "五、简答题"}

	// 这里已经成功构建出题目跟试卷数据了，现在可以进行一次查询了
	// TODO 考卷后面的所有逻辑

	tests := []struct {
		name          string
		pid           int64
		withQuestions bool
		expectedErr   error
	}{
		{
			name:          "正常1 正确paperID 携带问题查询",
			pid:           68,
			withQuestions: true,
			expectedErr:   nil,
		},
		{
			name:          "正常1 正确paperID 不携带问题查询",
			pid:           68,
			withQuestions: false,
			expectedErr:   nil,
		},
		{
			name:          "异常1 不正确paperID 携带问题查询",
			pid:           1000,
			withQuestions: false,
			expectedErr:   errors.New("select paper failed"),
		},
		{
			name:          "异常2 不正确paperID 不携带问题查询",
			pid:           1000,
			withQuestions: false,
			expectedErr:   errors.New("select paper failed"),
		},
		{
			name:          "异常2 非法paperID",
			pid:           -1,
			withQuestions: false,
			expectedErr:   errors.New("invalid practice ID param"),
		},
		{
			name:          "异常3 json.unmarshal触发失败",
			pid:           68,
			withQuestions: true,
			expectedErr:   errors.New("unmarshal group data failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "test", "json")
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
				if tt.withQuestions {
					// 下面是检测题组
					if pg[0].Name.String != groupNames[0] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[0], pg[0].Name.String)
					}
					if pg[0].Order.Int64 != 1 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 1, pg[0].Order.Int64)
					}
					if pg[1].Name.String != groupNames[1] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[1], pg[1].Name.String)
					}
					if pg[1].Order.Int64 != 2 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 2, pg[1].Order.Int64)
					}
					if pg[2].Name.String != groupNames[2] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[2], pg[2].Name.String)
					}
					if pg[2].Order.Int64 != 3 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 3, pg[2].Order.Int64)
					}
					if pg[3].Name.String != groupNames[3] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[3], pg[3].Name.String)
					}
					if pg[3].Order.Int64 != 4 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 4, pg[3].Order.Int64)
					}
					if pg[4].Name.String != groupNames[4] {
						t.Errorf("查询的试卷题组1名称出错 期望%v,实际：%v", groupNames[4], pg[4].Name.String)
					}
					if pg[4].Order.Int64 != 5 {
						t.Errorf("查询的试卷题组1顺序出错 期望%v,实际：%v", 5, pg[4].Order.Int64)
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
	}
}

// 生成考卷测试
func TestGenerateExamPaper(t *testing.T) {
	// 由于这里已经有创建好试卷了，就直接使用就好

	if z == nil {
		cmn.ConfigureForTest()
	}
	var err error
	var practiceID, paperID int64
	practiceID = 6
	paperID = 68

	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 这里先测试练习那边生成的逻辑
	// 开启一个事务
	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				z.Error(err.Error())
			}
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				z.Error(err.Error())
			}
		}
	}()
	p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(ctx, p, practiceID, "考卷生成逻辑测试练习", "00", uid, 5, "00", 68)
	if err != nil {
		t.Fatal(err)
	}
	// 插入练习之后还要插入学生 3个，学生ID为1,2,3
	s := `
	INSERT INTO assessuser.t_practice_student 
    	(student_id, practice_id, creator, status) 
	VALUES ($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12)
	`
	_, err = tx.Exec(ctx, s, 1, practiceID, uid, "00", 2, practiceID, uid, "00", 3, practiceID, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	// 这里创建一个新的练习

	tests := []struct {
		name        string
		category    string
		pid         int64
		practiceID  int64
		sessionID   int64
		uid         int64
		genMarkInfo bool
		expectedErr error
	}{
		{
			name:        "正常生成练习考卷逻辑 并返回批改信息",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: nil,
		},
		{
			name:        "正常生成练习考卷逻辑 不返回批改信息",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: false,
			expectedErr: nil,
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
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid category param"),
		},
		{
			name:        "异常3 参数检测非法练习ID与考试场次ID",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  0,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid practice or examSession ID param"),
		},
		{
			name:        "异常4 参数检测非法操作者ID",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         -1,
			genMarkInfo: true,
			expectedErr: errors.New("invalid uid ID param"),
		},
		{
			name:        "异常5 触发LoadPaperTemplateById函数错误",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("unmarshal group data failed"),
		},
		{
			name:        "异常6 触发LoadPaperTemplateById函数加载的试卷数据题目为空",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("invalid paper question group"),
		},
		{
			name:        "异常7 触发解析题目答案 jsonUnmarshal错误",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("解析答案失败"),
		},
		{
			name:        "异常8 触发解析题目答案与试卷保存的更改小题分后长度不一错误",
			category:    PaperCategory.Practice,
			pid:         paperID,
			practiceID:  1,
			sessionID:   0,
			uid:         uid.Int64,
			genMarkInfo: true,
			expectedErr: errors.New("不匹配"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paperID int64
			paperID = 0
			if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "test", "json")
			} else {
				ctx = context.Background()
			}
			if containsString(tt.name, "异常6") {
				createEmpty := `
INSERT INTO assessuser.t_paper 
    (name, assembly_type, category, level, suggested_duration, tags, creator, status, access_mode) 
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
RETURNING id`
				err = conn.QueryRow(ctx, createEmpty, "生成考卷触发空数据试卷", "00", "00", "00", 120, types.JSONText("[]"), uid, "00", "00").Scan(&paperID)
				if err != nil {
					t.Errorf("无法生成一张空试卷：%v", err)
				}
			}
			if paperID != 0 {
				tt.pid = paperID
			}
			epid, mark, err := GenerateExamPaper(ctx, tx, tt.category, tt.pid, tt.practiceID, tt.sessionID, tt.uid, tt.genMarkInfo)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				// 否则的话，这里就是没有错误
				// 就需要检测此时生成的考卷信息了
				// 这里获取到考卷的ID，就能去检验是否生成成功
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if epid == nil {
					t.Fatalf("返回的考卷ID为空")
				}
				ep := `SELECT name,practice_id,creator FROM t_exam_paper WHERE id=$1 AND status = $2`
				examPaper := cmn.TExamPaper{}
				err := tx.QueryRow(ctx, ep, *epid, PaperStatus.Normal).Scan(&examPaper.Name, &examPaper.PracticeID, &examPaper.Creator)
				if err != nil {
					t.Fatal(err)
				}
				if examPaper.PracticeID.Int64 != 1 {
					t.Errorf("生成的考卷练习ID错误 预期：1，实际：%v", examPaper.PracticeID.Int64)
				}
				if examPaper.Name.String != "考卷单元测试试卷名" {
					t.Errorf("生成的考卷名称错误 预期：考卷单元测试试卷名，实际：%v", examPaper.Name.String)
				}
				if examPaper.Creator.Int64 != uid.Int64 {
					t.Errorf("生成的考卷创建者错误 预期：%v，实际：%v", uid, examPaper.Creator)
				}
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
			paperID = 0
		})
	}

}

func TestLoadExamPaperDetailsById(t *testing.T) {
	// 现在已经成功生成了考卷了，那就需要去测试这个数据是否存在
	if z == nil {
		cmn.ConfigureForTest()
	}
	var paperID int64
	paperID = 341

	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		epid          int64
		withQuestions bool
		withAnswers   bool
		expectedErr   error
	}{
		{
			name:          "正常1 能查询出一张完整的考卷试卷 不携带题目答案",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   false,
			expectedErr:   nil,
		},
		{
			name:          "正常2 能查询出一张完整的考卷试卷 携带题目答案",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   nil,
		},
		{
			name:          "正常2 能查询出一张完整的考卷试卷 携带题目答案",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   true,
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
			epid:          10086,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("select examPaper failed"),
		},
		{
			name:          "异常3 空groupsData 并非真的为空，而是json标签或者db标签的无法被识别导致失败",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("empty examPaper groups data"),
		},
		{
			name:          "异常4 非法groupsData jsonUnmarshal 失败",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   true,
			expectedErr:   errors.New("unmarshal group data failed"),
		},
		{
			name:          "异常5 非法Answers jsonUnmarshal 失败",
			epid:          paperID,
			withQuestions: true,
			withAnswers:   false,
			expectedErr:   errors.New("failed to unmarshal Answers questionId"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "test", "jsonB")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "test", "json")
			} else if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "test", "jsonA")
			} else {
				ctx = context.Background()
			}
			p, pg, pq, err := LoadExamPaperDetailsById(ctx, tx, tt.epid, tt.withQuestions, tt.withAnswers)
			if tt.expectedErr != nil {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if p == nil {
					t.Errorf("返回的考卷信息体为空")
				} else {
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
					if len(pg) != 5 {
						t.Errorf("查询的题组数量错误，实际：%v", len(pg))
					}

					var qCount int
					for _, g := range pg {
						quesitons := pq[g.ID.Int64]
						for _, _ = range quesitons {
							qCount++
						}
					}
					if qCount != 13 {
						t.Errorf("考卷题目数量错误")
					}
				}
			}
		})
	}

}

// 测试根据考卷题目与信息，批量插入学生考卷
func TestGenerateAnswerQuestion(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var examPaperID, practiceID int64
	examPaperID = 341
	practiceID = 7

	//创建一个新的练习把
	p := `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(ctx, p, practiceID, "考卷生成逻辑测试练习", "00", uid, 5, "00", 68)
	if err != nil {
		t.Fatal(err)
	}
	// 初始化参数切片（500名学生 * 4个字段 = 2000个参数）
	params := make([]interface{}, 0, 2000)
	valuePlaceholders := make([]string, 0, 500)

	// 动态生成占位符和参数
	for i := 1; i <= 500; i++ {
		offset := (i-1)*4
		valuePlaceholders = append(
			valuePlaceholders,
			fmt.Sprintf("($%d,$%d,$%d,$%d)", offset+1, offset+2, offset+3, offset+4)
		)
		params = append(params,
			i,           // student_id (从1到500)
			practiceID,  // 固定practiceID
			uid,         // 固定创建者ID
			"00",        // 固定状态
		)
	}

	// 组合完整SQL
	s := fmt.Sprintf(`
    INSERT INTO assessuser.t_practice_student 
        (student_id, practice_id, creator, status) 
    VALUES %s`,
		strings.Join(valuePlaceholders, ","),
	)
	// 执行批量插入
	_, err = tx.Exec(ctx, s, params...)
	if err != nil {
		z.Sugar().Errorf("插入学生失败: SQL=%s, 错误=%v", s, err)
		t.Fatal(err)
	}

	// 生成 SQL 的 VALUES 部分和参数切片
	valuePlaceholders = make([]string, 0, 500)
	params = make([]interface{}, 0, 500*6) // 每个学生6个参数
	n := 500
	submissionIDs := make([]int64, n)
	for i := int64(0); i < int64(n); i++ {
		submissionIDs[i] = i + 1 // 从1开始递增
	}

	for i := 1; i <= 500; i++ {
		// 每组占位符: ($1,$2,$3,$4,$5,$6) -> 扩展为100组
		placeholder := fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)",
			i*6-5, i*6-4, i*6-3, i*6-2, i*6-1, i*6)
		valuePlaceholders = append(valuePlaceholders, placeholder)

		// 添加实际参数（示例数据，按需调整）
		params = append(params,
			i,           // 动态生成ID（实际中建议用序列或UUID）
			practiceID,  // 练习ID
			i,           // 学生ID（假设学生ID从1递增）
			examPaperID, // 试卷ID
			uid,         // 创建者ID
			1,           // 尝试次数
		)
	}

	// 组合完整SQL
	m := fmt.Sprintf(
		`INSERT INTO assessuser.t_practice_submissions 
        (id, practice_id, student_id, exam_paper_id, creator, attempt) 
     VALUES %s`,
		strings.Join(valuePlaceholders, ","),
	)
	_, err = tx.Exec(ctx, m, params...)
	if err != nil {
		t.Fatal(err)
	}
	validError := errors.New("validation fail")

	// 现在全部都生成了 直接构造函数记录即可
	tests := []struct {
		name        string
		req         GenerateAnswerQuestionsRequest
		expectedErr error
	}{
		{
			name: "正常1 能生成学生考卷 携带两个乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     true,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: nil,
		},
		{
			name: "正常2 能生成学生考卷 携带题组内乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     true,
				IsOptionRandom:       false,
				Attempt:              1,
			},
			expectedErr: nil,
		},
		{
			name: "正常3 能生成学生考卷 携带题目选项乱序",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: nil,
		},
		{
			name: "正常4 能生成学生考卷 携带两个乱序，操作生成500个学生答卷",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: submissionIDs,
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: nil,
		},
		{
			name: "异常1因题目问题导致jsonUnmarshal无法解析 手动制造一个题目强行插入进入",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: nil,
		},
		{
			name: "异常2 参数检测 非法考卷ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          -1,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常3 参数检测 非法考卷用途",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             "04",
				PracticeSubmissionID: []int64{1, 2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常4 参数检测 非法练习提交记录ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, -2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
		{
			name: "异常5 参数检测 非法练习提交记录ID",
			req: GenerateAnswerQuestionsRequest{
				ExamPaperID:          examPaperID,
				Category:             PaperCategory.Practice,
				PracticeSubmissionID: []int64{1, -2, 3},
				IsQuestionRandom:     false,
				IsOptionRandom:       true,
				Attempt:              1,
			},
			expectedErr: validError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var  insertqCount int64
			err = conn.QueryRow(ctx,`SELECT COUNT(*) FROM t_student_answers WHERE status = '00' ` ).Scan(&insertqCount)
			if err != nil {
				t.Fatal(err)
			}
			err = GenerateAnswerQuestion(ctx, tx, tt.req, uid.Int64)
			if tt.expectedErr == nil {
				if err != nil {
					t.Errorf("期望无错误，但是返回错误：%v", err)
				}
				var questionCount int64
				if containsString(tt.name,"正常4"){
					err = conn.QueryRow(ctx,`SELECT COUNT(*) FROM t_student_answers WHERE status = '00' ` ).Scan(&questionCount)
					if err != nil {
						t.Fatal(err)
					}
					if questionCount != (500 * 13 + insertqCount){
						t.Errorf("生成的题目数量不对")
					}
				}else{
					// 这里需要去检测一下这个是否真的生成了
					c := `SELECT COUNT(*) FROM t_student_answers WHERE practice_submission_id = $1 OR practice_submission_id = $2 OR practice_submission_id = $3`
					err = tx.QueryRow(ctx, c, 1, 2, 3).Scan(&questionCount)
					if err != nil {
						t.Error(err)
					}

					if questionCount != 13*3 {
						t.Errorf("生成的学生答卷题目数量不一，实际：%v", questionCount)
					}
				}
				
			} else {
				if err == nil || !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func initQuestion(t *testing.T, tx pgx.Tx) {
	qb := `INSERT INTO t_question_bank  (type,name,creator,create_time,status,access_mode) VALUES($1, $2, $3, $4, $5, $6) RETURNING id`
	var qbID int64
	err := tx.QueryRow(ctx, qb, "00", "考卷测试题库", uid, now, "00", "00").Scan(&qbID)
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
			],`,
			`["A", "D"]`, 2, nil, nil, uid, "00", qbID, 1,
		},
		{2, "00", "<p><span style=\"font-size: 12pt\">H3C公司的总部位于哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">2</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">3</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">4</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">5</span></p>"}]`,
			`["A", "D"]`, 2, nil, nil, uid, "00", qbID, 1,
		},
		{3, "02", "<p><span style=\"font-size: 12pt\">H3C公司的分部遍布哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">广州</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">上海</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">北京</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">深圳</span></p>"}]`,
			`["A", "B","C","D"]`, 5, nil, nil, uid, "00", qbID, 1,
		},
		{4, "02", "<p><span style=\"font-size: 12pt\"> 以下哪些是常见的网络设备品牌？</p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">华为</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">思科</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\"> Juniper</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">中兴</span></p>"}]`,
			`["A", "B"]`, 5, nil, nil, uid, "00", qbID, 1,
		},
		{5, "02", "<p><span style=\"font-size: 12pt\">以下属于H3C主要产品线的有哪些？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">路由器</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">交换机</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">防火墙</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">服务器</span></p>"}]`,
			`["A","B","C"]`, 5, nil, nil, uid, "00", qbID, 1,
		},
		{6, "00", "<p><span style=\"font-size: 12pt\">以下哪些属于H3C的核心技术领域？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">云计算</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">大数据</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">人工智能</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">物联网</span></p>"}]`,
			`["A"]`, 2, nil, nil, uid, "00", qbID, 1,
		},
		{7, "04", "<p><span style=\"font-size: 12pt\">在H3C设备上，'undo shutdown'命令可以启用一个物理接口。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, nil, nil, uid, "00", qbID, 1,
		},
		{8, "04", "<p><span style=\"font-size: 12pt\">H3C交换机的默认管理VLAN编号是1。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, nil, nil, uid, "00", qbID, 1,
		},
		{9, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "螺旋模型","grading_rule": "答案必须准确匹配“螺旋模型”","alternative_answers": []}]`, 5, nil, nil, uid, "00", qbID, 1,
		},
		{10, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">软件开发中，用于描述系统功能的文档是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "需求规格说明书","grading_rule": "答案必须准确匹配“需求规格说明书”","alternative_answers": []}]`, 5, nil, nil, uid, "00", qbID, 1,
		},
		{11, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">面向对象编程的三大基本特性分别是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 1,"answer": "封装","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 2,"score": 1,"answer": "继承","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 3,"score": 1,"answer": "多态","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []}]`,
			3, nil, nil, uid, "00", qbID, 1,
		},
		{12, "08", "<p>简述计算机病毒的传播途径。</p>", nil, `[{"index": 1,"score": 5,"answer": "通过网络下载、移动存储设备、电子邮件等方式传播","grading_rule": "答出3种主要传播方式即可得满分","alternative_answers": []}]`,
			5, nil, nil, uid, "00", qbID, 1,
		},
		{13, "08", "<p>简述操作系统的主要功能。</p>", nil, `[{"index": 1,"score": 5,"answer": "进程管理、内存管理、文件管理、设备管理、作业管理","grading_rule": "答出3种主要功能即可得满分","alternative_answers": []}]`,
			5, nil, nil, uid, "00", qbID, 1,
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

func initPaper(t *testing.T, tx pgx.Tx) {
	// 成功插入之后，就可以开始创建试卷跟题组了
	// 创建试卷
	paper := &cmn.TPaper{
		Name:              null.StringFrom("考卷单元测试试卷名"),
		AssemblyType:      null.StringFrom("00"),
		Category:          null.StringFrom("02"),
		Level:             null.StringFrom("02"),
		Description:       null.StringFrom("考卷专门服务"),
		SuggestedDuration: null.IntFrom(60),
		Creator:           uid,
		CreateTime:        null.IntFrom(now),
		UpdatedBy:         uid,
		UpdateTime:        null.IntFrom(now),
		Status:            null.StringFrom("00"),
		Tags:              types.JSONText(`["test", "unit"]`),
		AccessMode:        null.StringFrom("00"), // 默认访问模式
	}
	var pID int64
	p := `INSERT INTO t_paper
			(name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`
	err := tx.QueryRow(ctx, p, paper.Name.String,
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
		paper.AccessMode.String,
	).Scan(&pID)
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
			pID,
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
	_, err = tx.Exec(ctx, pq, 11, groupIDs[4], 12, 5, uid, "00", types.JSONText(`[2,2,1]`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, pq, 12, groupIDs[4], 13, 5, uid, "00", types.JSONText(`[5]`))
	if err != nil {
		t.Fatal(err)
	}

	// 这里查询一下

	var tp cmn.TVPaper
	scanArgs := []interface{}{
		&tp.ID, &tp.Name, &tp.AssemblyType, &tp.Category, &tp.Level,
		&tp.SuggestedDuration, &tp.Description, &tp.Tags, &tp.Config,
		&tp.Creator, &tp.CreateTime, &tp.UpdateTime, &tp.Status,
		&tp.TotalScore, &tp.QuestionCount, &tp.GroupCount, &tp.GroupsData,
	}
	query := `SELECT id,name,assembly_type,category,level,suggested_duration,description,tags,config,creator,create_time,update_time,
	status,total_score,question_count,group_count,groups_data
	FROM v_paper WHERE id=$1 AND status != $2`
	err = tx.QueryRow(ctx, query, pID, PaperStatus.Invalid).Scan(scanArgs...)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", Json(tp))
}
