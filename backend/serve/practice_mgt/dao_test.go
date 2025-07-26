/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理数据库层函数逻辑测试
 * @Date: 2025-07-24 14:51:50
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-26 15:10:39
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
			err := UpdatePractice(ctx, tt.p, tt.ps, tt.uid)
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
			pid:           int64(2086),
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
						ID:              null.IntFrom(2086),
						PaperID:         null.IntFrom(102),
						Name:            null.StringFrom("数学期末考试"),
						CorrectMode:     null.StringFrom("02"), // 批改模式
						Type:            null.StringFrom("02"), // 练习类型（试卷）
						Status:          null.StringFrom("02"),
						Creator:         null.IntFrom(100),
						CreateTime:      null.IntFrom(now),
						AllowedAttempts: null.IntFrom(10),
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
					if sCount != 0 {
						t.Errorf("查询练习参会学生名单不符合；预期:%v,实际:%v", 0, sCount)
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

// 测试学生权限获取练习作答列表
func TestListPracticeS(t *testing.T) {}

// 测试教师及以上权限获取练习作答列表
func TestListPracticeT(t *testing.T) {}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
