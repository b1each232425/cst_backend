/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷数据库层单元测试
 * @Date: 2025-07-28 19:55:28
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-29 11:24:26
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
	               [
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
	               ]
	           ]`,
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
	_, err = tx.Exec(ctx, pq, 11, groupIDs[4], 12, 5, uid, "00", types.JSONText(`[5]`))
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
