/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理数据库层函数逻辑测试
 * @Date: 2025-07-24 14:51:50
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-31 15:40:46
 */
package practice_mgt

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

var (
	ctx = context.Background()
	now = time.Now().UnixMilli()
)

// 测试新增一个练习 测试边缘,参数残缺或者别的情况
// 练习的数据 数据层本身的操作，不涉及handler前面的数据校验
func TestAddPractice(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()

	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		uid           int64
		expectedError bool
	}{
		{
			name: "期望正常1 完整练习信息 + 学生名单数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom("英语期末考试"),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{201, 202, 203, 204},
			uid:           int64(1),
			expectedError: false,
		},
		{
			name: "期望正常2 完整练习信息 但无学生名单,前端传空数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom("英语期末考试"),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{},
			uid:           int64(1),
			expectedError: false,
		},
	}
	var expectCount int
	expectCount++
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPractice(ctx, tt.p, tt.ps, tt.uid)
			if tt.expectedError {
				if err != nil {
					if !containsString(err.Error(), "addPractice call") {
						// 这里如果没有的话，那就是错了
						t.Errorf("预期查询不到会为空，但是此时没有经过这个分支，查看具体报错:%v", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("AddPractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				// 执行一下查询
				s := `SELECT COUNT(*) FROM t_practice`
				var count int
				err = conn.QueryRow(ctx, s).Scan(&count)
				if err != nil {
					t.Errorf("测试环境有问题：%v", err)
				}
				if count != expectCount {
					t.Errorf("没有成功新增练习")
				}
			}
			expectCount++
		})
	}
}

// 参数校验已经在函数调用前进行
func TestUpdatePractice(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	// 清理函数
	//defer func() {
	//	_, err := conn.Exec(ctx, `DELETE FROM t_practice`)
	//	if err != nil {
	//		if err != nil {
	//			t.Logf("清理练习记录失败: %v", err)
	//		}
	//	}
	//}()

	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		uid           int64
		expectedError bool
		updated       bool
	}{
		{
			name: "期望正常1 练习信息全更新 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				ID:              null.IntFrom(2085),
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("数学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("00"),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(2),
			expectedError: false,
			updated:       true,
		},
		{
			name: "期望正常2 练习信息固定 + 可修改信息被更新 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				ID:              null.IntFrom(2086),
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("数学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("00"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(1),
			expectedError: false,
			updated:       true,
		},
		{
			name: "期望正常3 但不会发生更新 练习信息缺少练习ID 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(10),
			expectedError: false,
			updated:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdatePractice(ctx, tt.p, tt.ps, tt.uid, false)
			if tt.expectedError {
				t.Errorf("本次执行不会有报错的才对: %v", err)
			} else {
				if err != nil {
					t.Errorf("UpdatePractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				if !tt.updated {
					var count int
					s := `SELECT COUNT(*) FROM t_practice WHERE name=$1`
					_ = conn.QueryRow(ctx, s, tt.p.Name).Scan(&count)
					if count > 0 {
						t.Errorf("此时不应该更新")
					}
				}
				// 执行一下查询 查看原本的字段是否真正被更新了
				s := `SELECT name , creator , create_time , paper_id,correct_mode,type,allowed_attempts ,status ,updated_by FROM t_practice WHERE id=$1`
				var name, Type, status, correct null.String
				var creator, createTime, paperID, allow, updatedBy null.Int
				err = conn.QueryRow(ctx, s, tt.p.ID).Scan(&name, &creator, &createTime, &paperID, &correct, &Type, &allow, &status, &updatedBy)
				if err != nil {
					t.Errorf("测试环境有问题：%v", err)
				}
				if !updatedBy.Valid {
					t.Errorf("没有成功插入更新者字段")
				} else {
					if updatedBy.Int64 != tt.uid {
						t.Errorf("插入的更新者数据错误，期望：%v,实际:%v", tt.uid, updatedBy.Int64)
					}
				}
				if name.String != tt.p.Name.String {
					t.Errorf("没有成功更新字段Name，期望为:%v,实际为:%v", tt.p.Name.String, name)
				}
				if status.String != tt.p.Status.String {
					t.Errorf("没有成功更新字段Status，期望为:%v,实际为:%v", tt.p.Status.String, status)
				}
				if correct.String != tt.p.CorrectMode.String {
					t.Errorf("没有成功更新字段correct，期望为:%v,实际为:%v", tt.p.CorrectMode.String, correct)
				}
				if Type.String != tt.p.Type.String {
					t.Errorf("没有成功更新字段type，期望为:%v,实际为:%v", tt.p.Type.String, Type)
				}
				if allow != tt.p.AllowedAttempts {
					t.Errorf("没有成功更新字段allow，期望为:%v,实际为:%v", tt.p.AllowedAttempts.Int64, allow.Int64)
				}
				if paperID != tt.p.PaperID {
					t.Errorf("没有成功更新字段allow，期望为:%v,实际为:%v", tt.p.PaperID.Int64, paperID.Int64)
				}
				if creator == tt.p.Creator {
					t.Errorf("不应该更新字段Creator，但更新,请检查UpdatePractice函数")
				}
				if createTime == tt.p.CreateTime {
					t.Errorf("不应该更新字段CreateTime，但更新,请检查UpdatePractice函数")
				}
			}
		})
	}
}

func TestLoadPracticeById(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name          string
		pid           int64
		expectedError bool
		havePractice  bool
	}{
		{
			name:          "期望正常1 存在正常练习信息且能正常查询到",
			pid:           int64(2036),
			expectedError: false,
			havePractice:  true,
		},
		{
			name:          "传入不存在的practiceID查询练习信息",
			pid:           int64(1000),
			expectedError: true,
			havePractice:  false,
		},
		{
			name:          "传入非法的练习ID查询练习信息",
			pid:           int64(-1000),
			expectedError: true,
			havePractice:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, pName, sCount, err := LoadPracticeById(ctx, tt.pid)
			if tt.expectedError {
				if err == nil {
					// 此时不应该没有错误的
					t.Errorf("此时无错误不符合预期")
					// 然后打印一下查询出来的信息
					t.Logf("打印一下输出的信息:%v,%v,%v", *p, pName, sCount)
					return
				}
				// 此时是传入一个不合格的practiceID的，所以会回出现查询不到的情况，因此也会进入到这个ErrNoRows的行列的
				if !containsString(err.Error(), "非法practiceID") {
					// 这里如果没有的话，那就是错了
					t.Errorf("预期查询不到会为空，但是此时没有经过这个分支，查看具体报错:%v", err)
				}
			} else {
				if tt.havePractice {
					p1 := &cmn.TPractice{
						ID:              null.IntFrom(2036),
						PaperID:         null.IntFrom(102),
						Name:            null.StringFrom("第一版练习数据v2"),
						CorrectMode:     null.StringFrom("00"), // 批改模式
						Type:            null.StringFrom("00"), // 练习类型（试卷）
						Status:          null.StringFrom("02"),
						Creator:         null.IntFrom(1574),
						AllowedAttempts: null.IntFrom(5),
					}
					if p.Name.String != p1.Name.String {
						t.Errorf("查询练习名称不符合；预期:%v,实际:%v", p1.Name.String, p.Name.String)
					}
					if p.CorrectMode.String != p1.CorrectMode.String {
						t.Errorf("查询练习批改模式不符合；预期:%v,实际:%v", p1.CorrectMode.String, p.CorrectMode.String)
					}
					if p.CorrectMode.String != p1.CorrectMode.String {
						t.Errorf("查询练习批改模式不符合；预期:%v,实际:%v", p1.CorrectMode.String, p.CorrectMode.String)
					}
					if p.Type.String != p1.Type.String {
						t.Errorf("查询练习类型不符合；预期:%v,实际:%v", p1.Type.String, p.Type.String)
					}
					if sCount != 7 {
						t.Errorf("查询练习参会学生名单不符合；预期:%v,实际:%v", 7, sCount)
					}
					if pName != "" {
						t.Errorf("查询练习试卷名称不符合；预期:%v,实际:%v", "", pName)
					}
				}
			}
		})
	}

}

func TestUpsertPractice(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}
	// 准备测试数据
	conn := cmn.GetPgxConn()
	ts := `SELECT COUNT(*) from t_practice`
	var nowCount int
	_ = conn.QueryRow(ctx, ts).Scan(&nowCount)
	// 这里需要自己提供比较合法的数据结构体了
	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		uid           int64
		expectedError error
		create        bool
	}{
		{
			name: "正常创建练习 有学生名单",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom("英语期末考试"),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{201, 202, 203, 204},
			uid:           int64(10),
			create:        true,
			expectedError: nil,
		},
		{
			name: "正常创建练习 但无学生名单",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("数学期末考试"),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(4),
			},
			ps:            []int64{},
			uid:           int64(10),
			create:        true,
			expectedError: nil,
		},
		{
			name: "正常更新练习 但无学生名单",
			p: &cmn.TPractice{
				ID:              null.IntFrom(2086),
				PaperID:         null.IntFrom(104),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"), // 尝试通过upsert接口进行更改，但是不会成功更改
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(10),
			create:        false,
			expectedError: nil,
		},
		{
			name: "异常 更新已发布状态的练习",
			p: &cmn.TPractice{
				ID:              null.IntFrom(2095),
				PaperID:         null.IntFrom(104),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(10),
			create:        false,
			expectedError: errors.New("练习已经发布，不可修改练习信息"),
		},
	}

	var testCount int
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpsertPractice(context.Background(), tt.p, tt.ps, tt.uid)
			if tt.expectedError != nil {
				if err.Error() != tt.expectedError.Error() {
					// 不符合预期的错误
					t.Errorf("预期错误:%v，实际:%v", tt.expectedError, err)
				}
			} else {
				if tt.create {
					// 如果是创建的话，就去搜索一下
					s := `SELECT COUNT(*) from t_practice`
					_ = conn.QueryRow(ctx, s).Scan(&testCount)
					if testCount-nowCount != 1 {
						t.Errorf("计算此时练习表中的数据错误，预期：%v,实际%v", nowCount+1, testCount)
					}
					nowCount++
				} else {
					s := `SELECT name , creator , create_time , paper_id,correct_mode,type,allowed_attempts ,status ,updated_by FROM t_practice WHERE id=$1`
					var name, Type, status, correct null.String
					var creator, createTime, paperID, allow, updatedBy null.Int
					err = conn.QueryRow(ctx, s, tt.p.ID).Scan(&name, &creator, &createTime, &paperID, &correct, &Type, &allow, &status, &updatedBy)
					if err != nil {
						t.Errorf("测试环境有问题：%v", err)
					}
					if !updatedBy.Valid {
						t.Errorf("没有成功插入更新者字段")
					} else {
						if updatedBy.Int64 != tt.uid {
							t.Errorf("插入的更新者数据错误，期望：%v,实际:%v", tt.uid, updatedBy.Int64)
						}
					}
					if name.String != tt.p.Name.String {
						t.Errorf("没有成功更新字段Name，期望为:%v,实际为:%v", tt.p.Name.String, name)
					}
					if correct.String != tt.p.CorrectMode.String {
						t.Errorf("没有成功更新字段correct，期望为:%v,实际为:%v", tt.p.CorrectMode.String, correct)
					}
					if Type.String != tt.p.Type.String {
						t.Errorf("没有成功更新字段type，期望为:%v,实际为:%v", tt.p.Type.String, Type)
					}
					if allow != tt.p.AllowedAttempts {
						t.Errorf("没有成功更新字段allow，期望为:%v,实际为:%v", tt.p.AllowedAttempts.Int64, allow.Int64)
					}
					if paperID != tt.p.PaperID {
						t.Errorf("没有成功更新字段allow，期望为:%v,实际为:%v", tt.p.PaperID.Int64, paperID.Int64)
					}
					if creator == tt.p.Creator {
						t.Errorf("不应该更新字段Creator，但更新,请检查UpdatePractice函数")
					}
					if createTime == tt.p.CreateTime {
						t.Errorf("不应该更新字段CreateTime，但更新,请检查UpdatePractice函数")
					}
				}
			}
		})
	}
}

func TestValidatePractice(t *testing.T) {
	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		expectedError error
	}{
		{
			name: "正常数据 携带学生ID数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{100, 200, 300},
			expectedError: nil,
		},
		{
			name: "正常数据 不携带学生ID数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: nil,
		},
		{
			name: "正常数据 零值学生ID数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{0, 0, 0},
			expectedError: errors.New("invalid practice studentID"),
		},
		{
			name: "正常数据 异常学生ID数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{-100, 0, -10000},
			expectedError: errors.New("invalid practice studentID"),
		},
		{
			name: "异常PaperID",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(-1),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice PaperID"),
		},
		{
			name: "缺少PaperID",
			p: &cmn.TPractice{
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice PaperID"),
		},
		{
			name: "缺少Name",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice Name"),
		},
		{
			name: "异常Name",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom(""),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice Name"),
		},
		{
			name: "缺少CorrectMode",
			p: &cmn.TPractice{
				Name:            null.StringFrom("化学期末考试"),
				PaperID:         null.IntFrom(102),
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice CorrectMode"),
		},
		{
			name: "异常CorrectMode",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("异常批改数据"), // 批改模式
				Type:            null.StringFrom("02"),     // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice CorrectMode"),
		},
		{
			name: "缺少Type",
			p: &cmn.TPractice{
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Name:            null.StringFrom("化学期末考试"),
				PaperID:         null.IntFrom(102),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice Type"),
		},
		{
			name: "异常Type",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"),     // 批改模式
				Type:            null.StringFrom("异常练习类型"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice Type"),
		},
		{
			name: "缺少AllowedAttempts",
			p: &cmn.TPractice{
				CorrectMode: null.StringFrom("02"), // 批改模式
				Name:        null.StringFrom("化学期末考试"),
				PaperID:     null.IntFrom(102),
				Type:        null.StringFrom("02"), // 练习类型（试卷）
			},
			ps:            nil,
			expectedError: errors.New("invalid practice AllowedAttempts"),
		},
		{
			name: "异常AllowedAttempts",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(-1),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice AllowedAttempts"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePractice(tt.p, tt.ps)
			if tt.expectedError == nil {
				if err != nil {
					t.Errorf("%v expected error but got nil", tt.name)
				}
				t.Logf("打印正常数据：%v", *tt.p)
			} else {
				// 此时是期望会返回错误值的
				if err.Error() != tt.expectedError.Error() {
					t.Errorf("%v expected error:%v but got %v", tt.name, tt.expectedError.Error(), err.Error())
				}
			}
		})
	}
}

// 更新一次练习参与的学生名单 这里需要要有事务的 补充覆盖率跟边界值的输入
func TestUpsertPracticeStudent(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}
	// 准备测试数据
	conn := cmn.GetPgxConn()
	var pid int64
	var uid int64
	testStudents := []int64{101, 102, 103}
	uid = 10
	a := `
	INSERT INTO assessuser.t_practice (name,creator) VALUES ($1,$2)RETURNING id`
	err := conn.QueryRow(ctx, a, "测试练习名称", 10).Scan(&pid)
	if err != nil {
		t.Error(err)
		return
	}
	// 4. 核心测试用例
	t.Run("首次添加学生", func(t *testing.T) {
		err := UpsertPracticeStudent(ctx, pid, uid, testStudents)
		require.NoError(t, err)

		// 验证结果
		var count int
		c := `
			SELECT COUNT(*) 
			FROM assessuser.t_practice_student 
			WHERE practice_id = $1 AND status = $2`
		err = conn.QueryRow(ctx, c, pid, PracticeStudentStatus.Normal).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, len(testStudents), count)
	})

	t.Run("更新学生名单", func(t *testing.T) {
		newStudents := []int64{101, 104} // 移除了102,103 新增104
		err := UpsertPracticeStudent(ctx, pid, uid, newStudents)
		require.NoError(t, err)

		// 验证新增学生
		var newCount int
		c := `
			SELECT COUNT(*) 
			FROM assessuser.t_practice_student 
			WHERE practice_id = $1 AND status = $2`
		err = conn.QueryRow(ctx, c, pid, PracticeStudentStatus.Normal).Scan(&newCount)
		require.NoError(t, err)
		assert.Equal(t, 2, newCount)

		// 验证软删除学生
		var deletedCount int
		d := `
			SELECT COUNT(*) 
			FROM assessuser.t_practice_student 
			WHERE practice_id = $1 
			AND student_id NOT IN (101,104)
			AND status != $2`
		err = conn.QueryRow(ctx, d, pid, PracticeStudentStatus.Normal).Scan(&deletedCount)
		require.NoError(t, err)
		assert.Equal(t, 2, deletedCount) // 102和103应被软删除
	})

	t.Run("参数检测：空数组或nil值、非法练习ID、非法用户ID", func(t *testing.T) {
		newStudents := []int64{} // 移除了102,103 新增104
		err := UpsertPracticeStudent(ctx, pid, uid, newStudents)
		require.NoError(t, err)

		newStudents = []int64{11}
		err = UpsertPracticeStudent(ctx, int64(0), uid, newStudents)
		require.Equal(t, err.Error(), "invalid practiceId  param")

		err = UpsertPracticeStudent(ctx, pid, int64(0), newStudents)
		require.Equal(t, err.Error(), "invalid uid param")

	})

	t.Run("数据库事务触发错误，触发回滚", func(t *testing.T) {

		// 事务触发错误
		ctx = context.Background()
		newStudents := []int64{1000} // 移除了102,103 新增104
		beginTxCtx := context.WithValue(ctx, "force-error", "beginTx")
		err = UpsertPracticeStudent(beginTxCtx, pid, uid, newStudents)
		require.Error(t, err)
		if err == nil || !containsString(err.Error(), "beginTx called failed") {
			t.Errorf("%v expected error but got nil", err.Error())
		}

		// 这里要额外的去插入一些特用的数据的
		var p2ID int64
		a := `
		INSERT INTO assessuser.t_practice (name,creator) VALUES ($1,$2)RETURNING id`
		err := conn.QueryRow(ctx, a, "测试回滚panic练习名称", 10).Scan(&p2ID)
		if err != nil {
			t.Error(err)
			return
		}
		newStudents = []int64{200}
		panicCtx := context.WithValue(ctx, "force-error", "Rollback")
		err = UpsertPracticeStudent(panicCtx, p2ID, uid, newStudents)

		//然后再去搜索一下，查询数据是否还在
		rollbackCount := 0
		q := `SELECT COUNT(*) from t_practice_student WHERE practice_id = $1`
		_ = conn.QueryRow(ctx, q, p2ID).Scan(&rollbackCount)
		if rollbackCount != 0 {
			t.Errorf("%v expected rollback count to be 0, but got %v", rollbackCount, rollbackCount)
		}

		newStudents = []int64{201}
		RollbackCtx := context.WithValue(ctx, "force-error", "Rollback")
		err = UpsertPracticeStudent(RollbackCtx, p2ID, uid, newStudents)
		//然后再去搜索一下，查询数据是否还在
		q1 := `SELECT COUNT(*) from t_practice_student WHERE practice_id = $1`
		_ = conn.QueryRow(ctx, q1, p2ID).Scan(&rollbackCount)
		if rollbackCount != 0 {
			t.Errorf("%v expected rollback count to be 0, but got %v", rollbackCount, rollbackCount)
		}

		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice_student WHERE practice_id = $1", p2ID)
		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", p2ID)

	})

	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice_student WHERE practice_id = $1", pid)
		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", pid)
	})
}

// 测试学生权限获取练习作答列表 这里虽然无数据，但也是可以
func TestListPracticeS(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

}

// 测试教师及以上权限获取练习作答列表
// 练习暂无其余练习
func TestListPracticeT(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 这里就可以插入数据的方式去进行测试了
	conn := cmn.GetPgxConn()

	var p1, p2 int64

	err := conn.QueryRow(ctx, `INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, "测试练习名称1", MarkMode.AI, 100, now, now, nil, 5, PracticeType.Classical, 62).Scan(&p1)
	if err != nil {
		t.Fatalf("创建测试练习p1失败: %v", err)
	}
	err = conn.QueryRow(ctx, `INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time,status,addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10) RETURNING id`, "测试练习名称2", MarkMode.AI, 100, now, now, PracticeStatus.Released, nil, 10, PracticeType.Classical, 62).Scan(&p2)
	if err != nil {
		t.Fatalf("创建测试练习p2失败: %v", err)
	}

	testStudents := []int64{101, 102, 103}
	// 这里还需要插入学生
	err = UpsertPracticeStudent(ctx, p1, 100, testStudents)
	require.NoError(t, err)
	err = UpsertPracticeStudent(ctx, p2, 100, testStudents)
	require.NoError(t, err)

	// 此处的分页也是要测试	 分页的逻辑检测已经放在handler层了
	tests := []struct {
		name          string
		pName         string
		Type          string
		status        string
		page          int
		pageSize      int
		expectedError error
		expectedNum   int
	}{
		{
			name:          "正常1 不进行其余筛选查询",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   2,
		},
		{
			name:          "正常2 练习名称1模糊筛选查询",
			pName:         "1",
			Type:          PracticeType.Classical,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
		{
			name:          "正常3 练习名称2模糊筛选查询",
			pName:         "2",
			Type:          PracticeType.Classical,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
		{
			name:          "正常4 其他练习类型筛选查询 无结果",
			pName:         "",
			Type:          PracticeType.PracticeNew,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   0,
		},
		{
			name:          "正常5 已发布练习状态筛选",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.Released,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
		{
			name:          "正常5 未发布练习状态筛选",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.PendingRelease,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
	}

	// 这里给一个状态的数量
	var pending, release int
	pending = 0
	release = 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa, total, err := ListPracticeT(ctx, tt.pName, tt.Type, tt.status, []string{"create_time desc"}, tt.page, tt.pageSize, 100)
			if tt.expectedError == nil {
				// 这里返回的是一个数组啊
				for _, tp := range pa {
					// 这里取出来的每一个p都是一个单独的练习实体（里面就会有练习本身+练习学生名单数量）
					p, ok := tp["practice"].(cmn.TPractice)
					if !ok {
						t.Errorf("practice 字段类型断言失败")
					}
					t.Logf("打印输出练习本身:%v", p)
					t.Logf("打印输出练习学生数量:%v", tp["student_count"])
					// 现在是分为两种了 你不可能去满足所有的测试用例去写这个
					if tt.status == "" {
						if p.Status.String == PracticeStatus.PendingRelease {
							pending++
						}
						if p.Status.String == PracticeStatus.Released {
							release++
						}
					}
					if tt.status != "" && p.Status.String != tt.status {
						t.Errorf("有发布状态筛选查询 错误练习发布状态搜索：期望%v,实际:%v", tt.status, p.Status.String)
					}
					if tt.name == "" && !containsString(p.Name.String, "测试练习") {
						t.Errorf("无筛选查询 错误练习名称")
					}
					if tt.name != "" && !containsString(p.Name.String, tt.pName) {
						t.Errorf("有练习名称筛选查询 错误练习名称模糊搜索：期望包含%v,实际没有", tt.pName)
					}
					if tt.Type == "" && p.Type.String != PracticeType.Classical {
						t.Errorf("无筛选查询 错误练习类型")
					}
					if tt.Type != "" && p.Type.String != tt.Type {
						t.Errorf("有练习类型筛选查询 错误练习类型搜索：期望%v,实际:%v", tt.Type, p.Type.String)
					}
					if tp["student_count"].(int64) != 3 {
						t.Errorf("统计练习学生数量错误")
					}
				}
				if tt.status == "" && tt.expectedNum == 2 && (pending != 1 || release != 1) {
					t.Errorf("practice 筛选的练习发布状态有误")
				}
				if total != tt.expectedNum {
					t.Errorf("统计练习列表个数错误")
				}

			} else {
				if err != nil {
					t.Errorf("%v expected error nil but got %v", tt.name, err)
				}
			}
		})

		t.Cleanup(func() {
			_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice_student WHERE practice_id = $1", p1)
			_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice_student WHERE practice_id = $1", p2)
			_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", p1)
			_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", p2)
		})
	}
}

// 操作练习的发布状态的，需要跟考卷挂钩的
// 单元函数测试逻辑的
func TestOperatePracticeStatus(t *testing.T) {

	// 这里只能去测试那个删除与取消发布的分支
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 先插入一场练习，不会依赖于其他的测试用例
	conn := cmn.GetPgxConn()

	var p1, p2 int64

	// 直接插入一个已经发布了的练习
	err := conn.QueryRow(ctx, `INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, "测试练习名称1", MarkMode.AI, 100, now, now, nil, 5, PracticeType.Classical, 62).Scan(&p1)
	if err != nil {
		t.Fatalf("创建测试练习p1失败: %v", err)
	}
	err = conn.QueryRow(ctx, `INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time,status,addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10) RETURNING id`, "测试练习名称2", MarkMode.AI, 100, now, now, PracticeStatus.Released, nil, 10, PracticeType.Classical, 62).Scan(&p2)
	if err != nil {
		t.Fatalf("创建测试练习p2失败: %v", err)
	}

	// 然后就是操作他变成待发布与删除状态
	tests := []struct {
		name          string
		pid           int64
		status        string
		expectedError error
	}{
		{
			name:          "异常1 将已发布的练习调整为删除状态",
			pid:           p1,
			status:        PracticeStatus.Deleted,
			expectedError: errors.New("获取练习状态出现数据错误"),
		},
		{
			name:          "正常1 将已发布的练习调整为待发布",
			pid:           p1,
			status:        PracticeStatus.PendingRelease,
			expectedError: nil,
		},
		{
			name:          "正常2 将待发布的练习调整为删除状态",
			pid:           p2,
			status:        PracticeStatus.Deleted,
			expectedError: nil,
		},
		{
			name:          "正常3 将待发布的练习调整为发布状态",
			pid:           6,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = OperatePracticeStatus(ctx, tt.pid, tt.status, 100)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("%v expected error but got nil", tt.name)
				}
				if err != nil && err.Error() != tt.expectedError.Error() {
					t.Errorf("%v expected error:%v but got %v", tt.name, tt.expectedError, err)
				}
			} else {
				var status string
				var epID null.Int
				s := `SELECT status ,exam_paper_id FROM t_practice WHERE id = $1`
				_ = conn.QueryRow(ctx, s, tt.pid).Scan(&status, &epID)
				if status != tt.status {
					t.Errorf("没有成功更新练习发布状态,实际：%v，预期：%v", tt.status, status)
				}
				// 如果是调整发布状态的话，那就需要
				if containsString(tt.name, "正常3") {
					if !epID.Valid || epID.Int64 < 0 {
						t.Errorf("没有成功更新练习发布状态，没有成功发布练习中的生成考卷实际：%v", epID)
					}
				}
			}
		})
	}

	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", p1)
		_, _ = conn.Exec(ctx, "DELETE FROM assessuser.t_practice WHERE id = $1", p2)
	})
}

// 测试学生进入练习的三种不同状态 这里使用的是主流程测试数据中的其余学生 这里使用三种情况，就是进入过一次作答但是没有提交的 ， 然后就是完全提交了的、最后是第一次作答的
func TestEnterPracticeGetPaperDetails(t *testing.T) {

	// 确保logger已初始化
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

	// 我这里先生成一个试试
	var unSubmit, never, submit, pid int64
	never = 1581
	submit = 1580
	unSubmit = 1579
	pid = 2036
	t.Logf("%v,%v,%v,%v", unSubmit, never, submit, pid)

	tests := []struct {
		name          string
		pid           int64
		uid           int64
		expectedError error
	}{
		{
			name:          "正常1 第一次进入练习作答 生成练习提交记录与考卷",
			pid:           pid,
			uid:           never,
			expectedError: nil,
		},
		{
			name:          "正常2 进入练习但是此时没有提交就退出了 不生成考卷与提交记录",
			pid:           pid,
			uid:           unSubmit,
			expectedError: nil,
		},
		{
			name:          "正常3 曾经进入练习已经提交了  重新生成考卷与提交记录",
			pid:           pid,
			uid:           submit,
			expectedError: nil,
		},
		{
			name:          "异常1 参数检测 非法pid与uid",
			pid:           -1,
			uid:           0,
			expectedError: errors.New("invalid practiceID | uid param"),
		},
		{
			name:          "异常2 不存在的pid与uid",
			pid:           10086,
			uid:           10086,
			expectedError: errors.New("select student practice submission failed"),
		},
		{
			name:          "异常3 此时重新作答的话，最大次数超过了",
			pid:           pid,
			uid:           submit,
			expectedError: errors.New("已达练习最大次数"),
		},
		// 如果是删除了考卷的话，因为视图那边是根据试卷进行join的，如果没有考卷，是无法创建一行记录的
		{
			name:          "异常4 删除后考卷ID再进行查询",
			pid:           pid,
			uid:           submit,
			expectedError: errors.New("select student practice submission failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常3 此时重新作答的话，最大次数超过了") {
				// 需要直接去修改练习的最大次数
				_, err := conn.Exec(ctx, "UPDATE assessuser.t_practice  SET allowed_attempts = 1 WHERE id = $1", tt.pid)
				if err != nil {
					t.Fatal(err)
				}
			}
			if containsString(tt.name, "异常4 删除后考卷ID再进行查询") {
				_, err := conn.Exec(ctx, "UPDATE assessuser.t_practice SET exam_paper_id = NULL WHERE id = $1", tt.pid)
				if err != nil {
					t.Fatal(err)
				}
			}

			info, pg, pq, err := EnterPracticeGetPaperDetails(ctx, tx, tt.pid, tt.uid)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("%v expected error but got nil", tt.name)
				}
				if err != nil && !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v expected error:%v but got %v", tt.name, tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("期望无错误，但是返回错误：%v", err)
				}
				// 如果此时没有错误的话，那就是直接查看返回的信息
				if info.PaperName != "H3C选择题练习试卷" {
					t.Errorf("返回的练习数据名字错误，实际：%v", info.PaperName)
				}
				if info.TotalScore != 20 {
					t.Errorf("返回的练习数据总分错误，实际：%v", info.TotalScore)
				}
				if info.GroupCount != 1 {
					t.Errorf("返回的练习数据题组错误，实际：%v", info.GroupCount)
				}
				if info.QuestionCount != 10 {
					t.Errorf("返回的练习数据题目数量错误，实际：%v", info.QuestionCount)
				}
				var qCount int
				for id, _ := range pg {
					quesitons := pq[id]
					for _, _ = range quesitons {
						qCount++
					}
				}
				if qCount != 10 {
					t.Errorf("考卷题目数量错误")
				}
			}
			if containsString(tt.name, "异常3 此时重新作答的话，最大次数超过了") {
				// 需要直接去修改练习的最大次数
				_, err := conn.Exec(ctx, "UPDATE assessuser.t_practice  SET allowed_attempts = 5 WHERE id = $1", tt.pid)
				if err != nil {
					t.Fatal(err)
				}
			}
			if containsString(tt.name, "异常4 删除后考卷ID再进行查询") {
				_, err := tx.Exec(ctx, "UPDATE assessuser.t_practice SET exam_paper_id = 324 WHERE id = $1", tt.pid)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}

}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
