/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理数据库层函数逻辑测试
 * @Date: 2025-07-24 14:51:50
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-04 23:20:19
 */
package practice_mgt

import (
	"context"
	"github.com/pkg/errors"
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
	var practiceName string
	practiceName = "单元测试练习名"
	var uid int64
	uid = 10086

	s := `DELETE FROM t_practice WHERE name = $1`
	_, err := conn.Exec(ctx, s, practiceName)
	if err != nil {
		t.Fatal(err)
	}

	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		uid           int64
		expectedError error
	}{
		{
			name: "期望正常1 完整练习信息 + 学生名单数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom(practiceName),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{201, 202, 203, 204},
			uid:           uid,
			expectedError: nil,
		},
		{
			name: "期望正常2 完整练习信息 但无学生名单,前端传空数组",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom(practiceName),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{},
			uid:           uid,
			expectedError: nil,
		},
		{
			name: "异常1 触发更新练习学生名单错误",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom(practiceName),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{201, 202, 203, 204},
			uid:           uid,
			expectedError: errors.New("beginTx called failed"),
		},
		{
			name: "异常2 触发AddPractice错误",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				Name:            null.StringFrom(practiceName),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{},
			uid:           uid,
			expectedError: errors.New("addPractice call failed"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else {
				ctx = context.Background()
			}
			err := AddPractice(ctx, tt.p, tt.ps, tt.uid)
			if tt.expectedError != nil {
				if err != nil {
					if !containsString(err.Error(), tt.expectedError.Error()) {
						t.Errorf("预期错误：%v,实际报错:%v", tt.expectedError, err)
					}
				} else {
					t.Errorf("预期错误但是没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("AddPractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				// 执行一下查询
				s := `SELECT COUNT(*) FROM t_practice WHERE name = $1`
				var count int
				_ = conn.QueryRow(ctx, s, tt.p.Name).Scan(&count)
				if count != 1 {
					t.Errorf("没有成功新增练习 预期是1，实际是：%v", count)
				}
				if containsString(tt.name, "期望正常1") {
					// 如果是包含这个的话，那就去查询学生数量
					s = `SELECT COUNT(*) FROM t_practice_student`
					_ = conn.QueryRow(ctx, s).Scan(&count)
					if count != 4 {
						t.Errorf("练习学生名单数量不一致")
					}
				}
			}
			t.Cleanup(func() {
				// 这里去执行所有我之前的数据的清除
				s := `DELETE FROM assessuser.t_practice WHERE name = $1`
				_, err := conn.Exec(ctx, s, practiceName)
				if err != nil {
					t.Fatal(err)
				}
				// 这里再删除掉练习学生名单的数据
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
			})
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

	// 这里需要提前创建好所有数据
	var practiceName string
	practiceName = "单元测试练习名"
	var uid int64
	uid = 10086
	s := `DELETE FROM t_practice WHERE name = $1`
	_, err := conn.Exec(ctx, s, practiceName)
	if err != nil {
		t.Fatal(err)
	}
	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	// 先创建这个数据，最后测试完毕再删掉
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		p             *cmn.TPractice
		ps            []int64
		uid           int64
		expectedError error
		Ostatus       bool
	}{
		{
			name: "期望正常1 练习信息全更新 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("10"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(2),
			expectedError: nil,
		},
		{
			name: "期望正常2 练习信息固定 + 可修改信息被更新 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("00"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           uid,
			expectedError: nil,
		},
		{
			name: "期望正常3 但不会发生更新 练习信息缺少练习ID 传入空学生名单，不触发学生函数逻辑",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           uid,
			expectedError: nil,
		},
		{
			name: "正常4 能更新状态，仅仅满足覆盖率",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{101, 102},
			uid:           uid,
			expectedError: nil,
			Ostatus:       true,
		},
		{
			name: "异常1 非法uid",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           int64(-1),
			expectedError: errors.New("invalid updator ID param"),
		},
		{
			name: "异常2 触发数据插入行错误",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           uid,
			expectedError: errors.New("updatePractice call failed"),
		},
		{
			name: "异常3 触发更新学生名单错误",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				Status:          null.StringFrom("02"),
				Creator:         null.IntFrom(100),
				CreateTime:      null.IntFrom(now),
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{101, 102},
			uid:           uid,
			expectedError: errors.New("beginTx called failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else {
				ctx = context.Background()
			}
			err := UpdatePractice(ctx, tt.p, tt.ps, tt.uid, tt.Ostatus)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				// 执行一下查询 查看原本的字段是否真正被更新了
				s := `SELECT creator,paper_id,correct_mode,type,allowed_attempts ,status ,updated_by FROM t_practice WHERE id=$1`
				var Type, status, correct null.String
				var creator, paperID, allow, updatedBy null.Int
				err = conn.QueryRow(ctx, s, uid).Scan(&creator, &paperID, &correct, &Type, &allow, &status, &updatedBy)
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
				if tt.Ostatus {
					if status.String != tt.p.Status.String {
						t.Errorf("没有成功更新字段Status，期望为:%v,实际为:%v", tt.p.Status.String, status)
					}
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
			}

			t.Cleanup(func() {
				// 这里再删除这个练习，随后再重新创建
				s = `DELETE FROM assessuser.t_practice WHERE name = $1`
				_, err := conn.Exec(ctx, s, practiceName)
				if err != nil {
					t.Fatal(err)
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
				}
			})
		})

		t.Cleanup(func() {
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice WHERE name = $1`
			_, err := conn.Exec(ctx, s, practiceName)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLoadPracticeById(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()
	var practiceName, paperName string
	practiceName = "单元测试练习名"
	paperName = "单元测试试卷名"
	var uid int64
	uid = 10086
	s := `DELETE FROM t_practice WHERE name = $1`
	_, err := conn.Exec(ctx, s, practiceName)
	if err != nil {
		t.Fatal(err)
	}
	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 这里要创建试卷
	s = `INSERT INTO t_paper(id,name,assembly_type,category,level,creator,access_mode,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err = conn.Exec(ctx, s, uid, paperName, "00", "02", "00", uid, "00", "00")
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		pid           int64
		expectedError error
	}{
		{
			name:          "期望正常1 存在正常练习信息且能正常查询到",
			pid:           uid,
			expectedError: nil,
		},
		{
			name:          "异常1 传入不存在的practiceID查询练习信息",
			pid:           int64(100000),
			expectedError: errors.New("无该练习记录"),
		},
		{
			name:          "异常2 传入练习ID查询练习信息",
			pid:           int64(-1000),
			expectedError: errors.New("非法practiceID"),
		},
		{
			name:          "异常3 强制错误 prepare",
			pid:           uid,
			expectedError: errors.New("prepare sql"),
		},
		{
			name:          "异常4 强制错误close",
			pid:           uid,
			expectedError: nil,
		},
		{
			name:          "异常5 强制错误query ",
			pid:           uid,
			expectedError: errors.New("LoadPracticeById call failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "prepare")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "close")
			} else if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else {
				ctx = context.Background()
			}
			p, pName, sCount, err := LoadPracticeById(ctx, tt.pid)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				if pName != paperName {
					t.Errorf("查询到练习试卷名信息有误：预期%v，实际：%v", paperName, pName)
				}
				if sCount != 2 {
					t.Errorf("查询到练习学生参与数量信息有误：预期%v，实际：%v", 2, sCount)
				}
				if p == nil {
					t.Errorf("返回练习本体为空")
				}
				if p.Name.String != practiceName {
					t.Errorf("查询到练习试卷名信息有误：预期%v，实际：%v", practiceName, p.Name.String)
				}
				if p.CorrectMode.String != "00" {
					t.Errorf("查询到练习批改信息有误：预期%v，实际：%v", "00", p.CorrectMode.String)
				}
				if p.Type.String != "00" {
					t.Errorf("查询到练习类型名信息有误：预期%v，实际：%v", "00", p.Type.String)
				}
				if p.AllowedAttempts.Int64 != 5 {
					t.Errorf("查询到练习类型名信息有误：预期%v，实际：%v", 5, p.AllowedAttempts.Int64)
				}
			}
		})

		t.Cleanup(func() {
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice WHERE name = $1`
			_, err := conn.Exec(ctx, s, practiceName)
			if err != nil {
				t.Fatal(err)
			}

			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice_student `
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_paper WHERE name = $1`
			_, err = conn.Exec(ctx, s, paperName)
			if err != nil {
				t.Fatal(err)
			}
		})
	}

}

// 不触发更新学生名单 为了覆盖率，就是触发beginTx错误
func TestUpsertPractice(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()
	var practiceName1 string
	practiceName1 = "单元测试练习名"
	var uid int64
	uid = 10086
	s := `DELETE FROM t_practice WHERE name = $1`
	_, err := conn.Exec(ctx, s, practiceName1)
	if err != nil {
		t.Fatal(err)
	}

	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName1, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里还需要包含练习更新的，因此这里我就不触发其中的更新学生名单
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
			name: "参数异常1 非法uid",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{201, 202, 203, 204},
			uid:           int64(-1),
			create:        true,
			expectedError: errors.New("invalid updator ID param"),
		},
		{
			name: "正常创建练习 无学生名单",
			p: &cmn.TPractice{
				PaperID:         null.IntFrom(101),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(3),
			},
			ps:            []int64{},
			uid:           uid,
			create:        true,
			expectedError: nil,
		},
		{
			name: "正常更新练习 但无学生名单",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(102),
				CorrectMode:     null.StringFrom("00"), // 批改模式
				Type:            null.StringFrom("00"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(4),
			},
			ps:            []int64{},
			uid:           uid,
			create:        false,
			expectedError: nil,
		},
		{
			name: "更新练习异常1 更新已发布状态的练习",
			p: &cmn.TPractice{
				ID:              null.IntFrom(uid),
				PaperID:         null.IntFrom(104),
				Name:            null.StringFrom("化学期末考试"),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           uid,
			create:        false,
			expectedError: errors.New("练习已经发布，不可修改练习信息"),
		},
		{
			name: "更新练习异常2 触发LoadPractice错误",
			p: &cmn.TPractice{
				ID:              null.IntFrom(-1),
				PaperID:         null.IntFrom(104),
				CorrectMode:     null.StringFrom("02"), // 批改模式
				Type:            null.StringFrom("02"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            []int64{},
			uid:           uid,
			create:        false,
			expectedError: errors.New("非法practiceID"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "更新练习异常1") {
				// 这里先手动更新这个练习
				s = `UPDATE t_practice SET status = $1 WHERE name = $2`
				_, err = conn.Exec(ctx, s, "02", practiceName1)
				if err != nil {
					t.Errorf("无法让测试用例更新异常触发练习已经发布状态")
				}
			}
			err := UpsertPractice(context.Background(), tt.p, tt.ps, tt.uid)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePractice() 期望没有错误，但返回错误: %v", err)
					return
				}
				if tt.create {
					var count int
					// 如果是创建的话，就去搜索一下
					s = `SELECT COUNT(*) from t_practice WHERE name = $1`
					err = conn.QueryRow(ctx, s, practiceName1).Scan(&count)
					if err != nil {
						t.Errorf("QueryRow 执行错误：%v", err)
					}
					if count != 1 {
						t.Errorf("查询到练习数量信息有误：预期%v，实际：%v", 1, count)
					}

				} else {
					// 执行一下查询 查看原本的字段是否真正被更新了
					s := `SELECT creator,paper_id,correct_mode,type,allowed_attempts ,status ,updated_by FROM t_practice WHERE id=$1`
					var Type, status, correct null.String
					var creator, paperID, allow, updatedBy null.Int
					err = conn.QueryRow(ctx, s, uid).Scan(&creator, &paperID, &correct, &Type, &allow, &status, &updatedBy)
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
				}
			}

			t.Cleanup(func() {
				// 这里清理一下刚刚出现的
				s = `DELETE FROM t_practice WHERE name = $1`
				_, err = conn.Exec(ctx, s, practiceName1)
				if err != nil {
					t.Fatal(err)
				}

				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err := conn.Exec(ctx, s, uid, practiceName1, "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
				}
			})

		})

		t.Cleanup(func() {
			s = `DELETE FROM t_practice WHERE name = $1 or creator = $2`
			_, err = conn.Exec(ctx, s, practiceName1, uid)
			if err != nil {
				t.Fatal(err)
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				Type:            null.StringFrom("02"),           // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice CorrectMode"),
		},
		{
			name: "缺少Type",
			p: &cmn.TPractice{
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"),           // 批改模式
				Type:            null.StringFrom("异常练习类型"), // 练习类型（试卷）
				AllowedAttempts: null.IntFrom(10),
			},
			ps:            nil,
			expectedError: errors.New("invalid practice Type"),
		},
		{
			name: "缺少AllowedAttempts",
			p: &cmn.TPractice{
				CorrectMode: null.StringFrom("00"), // 批改模式
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
				CorrectMode:     null.StringFrom("00"), // 批改模式
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
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePractice() 期望没有错误，但返回错误: %v", err)
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

	// 这里需要创建练习，还要创建几个学生作为保底？需要先创建几个学生，然后再进行操作
	var practiceName string
	practiceName = "单元测试练习名"
	var uid int64
	uid = 10086

	s := `DELETE FROM t_practice WHERE name = $1`
	_, err := conn.Exec(ctx, s, practiceName)
	if err != nil {
		t.Fatal(err)
	}

	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		pid           int64
		uid           int64
		ps            []int64
		expectedError error
	}{
		{
			name:          "正常1 能正常更换学生名单",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
		{
			name:          "参数异常1 空学生数组 但不会报错",
			pid:           uid,
			uid:           uid,
			ps:            []int64{},
			expectedError: nil,
		},
		{
			name:          "参数异常2 非法uid",
			pid:           uid,
			uid:           int64(-1),
			ps:            []int64{2, 3},
			expectedError: errors.New("invalid uid param"),
		},
		{
			name:          "参数异常3 非法pid",
			pid:           int64(-1),
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("invalid practiceId  param"),
		},
		{
			name:          "强制异常1 beginTx",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("beginTx called failed"),
		},
		{
			name:          "强制异常2 query1",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("add PracticeStudent call failed"),
		},
		{
			name:          "强制异常3 query2",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("delete PracticeStudent call failed"),
		},
		{
			name:          "强制异常4 Rollback",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "强制异常4") {
				ctx = context.WithValue(ctx, "force-error", "Rollback")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if containsString(tt.name, "强制异常2") {
				ctx = context.WithValue(ctx, "force-error", "query1")
			} else if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else {
				ctx = context.Background()
			}

			err = UpsertPracticeStudent(ctx, tt.pid, tt.uid, tt.ps)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpsertPracticeStudent() 期望没有错误，但返回错误: %v", err)
					return
				}
				if !containsString(tt.name, "但不会报错") && !containsString(tt.name, "强制异常4") {
					// 这里就要检验一下这个学生ID是多少了
					s := `SELECT student_id FROM assessuser.t_practice_student WHERE practice_id = $1 AND status = $2`
					rows, err := conn.Query(ctx, s, tt.pid, "00")
					if err != nil {
						t.Fatal(err)
					}
					defer rows.Close()
					for rows.Next() {
						var id int64
						err = rows.Scan(&id)
						if err != nil {
							t.Fatal(err)
						}
						if id != 3 && id != 2 {
							t.Errorf("查询的学生ID出错，当前ID：%v", id)
						}
					}
				}
				if containsString(tt.name, "强制异常4") {
					// 这里就是判断成功
					// 这里就要检验一下这个学生ID是多少了
					s := `SELECT student_id FROM assessuser.t_practice_student WHERE practice_id = $1 AND status = $2`
					rows, err := conn.Query(ctx, s, tt.pid, "00")
					if err != nil {
						t.Fatal(err)
					}
					defer rows.Close()
					for rows.Next() {
						var id int64
						err = rows.Scan(&id)
						if err != nil {
							t.Fatal(err)
						}
						if id != 1 && id != 2 {
							t.Errorf("查询的学生ID出错，当前ID：%v", id)
						}
					}
				}
			}
			t.Cleanup(func() {
				// 这里再删除掉练习学生名单的数据
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
			})

		})

		t.Cleanup(func() {
			// 这里再删除掉练习学生名单的数据
			s = `DELETE FROM assessuser.t_practice WHERE name = $1 OR creator = $2`
			_, err = conn.Exec(ctx, s, practiceName, uid)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
		})
	}

}

// // 测试学生权限获取练习作答列表 这里虽然无数据，但也是可以
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

	var practiceName, practiceName1 string
	practiceName = "单元测试练习本体名"
	practiceName1 = "单元测试练习克隆名"
	var uid, uid1 int64
	uid = 10086
	uid1 = 10087
	s := `DELETE FROM assessuser.t_practice WHERE name = $1 OR creator = $2`
	_, err := conn.Exec(ctx, s, practiceName, uid)
	if err != nil {
		t.Fatal(err)
	}

	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid1, practiceName1, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12)`
	_, err = conn.Exec(ctx, s, 1, uid1, uid, "00", 2, uid1, uid, "00", 3, uid1, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

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
		sCount        int64
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
			sCount:        5,
		},
		{
			name:          "正常2 练习名称1模糊筛选查询",
			pName:         "本体",
			Type:          PracticeType.Classical,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
			sCount:        2,
		},
		{
			name:          "正常3 练习名称2模糊筛选查询",
			pName:         "克隆",
			Type:          PracticeType.Classical,
			status:        "",
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
			sCount:        3,
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
			sCount:        0,
		},
		{
			name:          "正常5 已发布练习状态筛选",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.Released,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   0,
			sCount:        0,
		},
		{
			name:          "正常5 未发布练习状态筛选",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.PendingRelease,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   2,
			sCount:        5,
		},
		{
			name:          "强制异常1 query",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.PendingRelease,
			page:          1,
			pageSize:      10,
			expectedError: errors.New("search practice failed"),
			expectedNum:   0,
		},
		{
			name:          "强制异常2 scan",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.PendingRelease,
			page:          1,
			pageSize:      10,
			expectedError: errors.New("解析练习数据失败"),
			expectedNum:   0,
		},
		{
			name:          "强制异常3 row close",
			pName:         "",
			Type:          PracticeType.Classical,
			status:        PracticeStatus.PendingRelease,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if containsString(tt.name, "强制异常2") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "row close")
			} else {
				ctx = context.Background()
			}

			r, total, err := ListPracticeT(ctx, tt.pName, tt.Type, tt.status, []string{"create_time desc"}, tt.page, tt.pageSize, uid)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpsertPracticeStudent() 期望没有错误，但返回错误: %v", err)
					return
				}
				if !containsString(tt.name, "强制异常3") {
					if total != tt.expectedNum {
						t.Errorf("预期练习列表数量不对 预期：%v 实际：%v", tt.expectedNum, total)
					}
					if len(r) != tt.expectedNum {
						t.Errorf("预期练习列表数量不对 预期：%v 实际：%v", tt.expectedNum, total)
					}
					var sCount int64
					for _, p := range r {
						sCount += p["student_count"].(int64)
					}
					if sCount != tt.sCount {
						t.Errorf("预期练习列表中学生数量不对 预期：%v 实际：%v", tt.sCount, sCount)
					}
				}
			}
		})

		t.Cleanup(func() {
			// 这里再删除掉练习学生名单的数据
			s = `DELETE FROM assessuser.t_practice WHERE name = $1`
			_, err = conn.Exec(ctx, s, practiceName)
			if err != nil {
				t.Fatal(err)
			}
			// 这里再删除掉练习学生名单的数据
			s = `DELETE FROM assessuser.t_practice WHERE name = $1`
			_, err = conn.Exec(ctx, s, practiceName1)
			if err != nil {
				t.Fatal(err)
			}

			s = `DELETE FROM assessuser.t_practice_student`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
		})

	}
}

// // 操作练习的发布状态的，需要跟考卷挂钩的
// // 单元函数测试逻辑的
func TestOperatePracticeStatus(t *testing.T) {

	// 这里只能去测试那个删除与取消发布的分支
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	conn := cmn.GetPgxConn()

	var practiceName string
	practiceName = "单元测试练习名"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_practice WHERE name = $1 OR creator = $2`
	_, err := conn.Exec(ctx, s, practiceName, uid)
	if err != nil {
		t.Fatal(err)
	}

	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = conn.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
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
			pid:           uid,
			status:        PracticeStatus.Deleted,
			expectedError: errors.New("获取练习状态出现数据错误"),
		},
		{
			name:          "正常1 将已发布的练习调整为待发布",
			pid:           uid,
			status:        PracticeStatus.PendingRelease,
			expectedError: nil,
		},
		{
			name:          "正常2 将待发布的练习调整为删除状态",
			pid:           uid,
			status:        PracticeStatus.Deleted,
			expectedError: nil,
		},
		{
			name:          "正常3 将待发布的练习调整为发布状态",
			pid:           uid,
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
}

// // 测试学生进入练习的三种不同状态 这里使用的是主流程测试数据中的其余学生 这里使用三种情况，就是进入过一次作答但是没有提交的 ， 然后就是完全提交了的、最后是第一次作答的
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
