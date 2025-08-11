/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理数据库层函数逻辑测试
 * @Date: 2025-07-24 14:51:50
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-08 14:08:25
 */
package practice_mgt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
	"math/rand"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
)

var (
	ctx = context.Background()
	now = time.Now().UnixMilli()
	uid = null.IntFrom(10086)
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
				ctx = context.WithValue(ctx, "force-error", "lQuery")
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
// // 单元函数测试逻辑的 现在返回来测试练习操作 有发布，取消发布，删除的三种操作 那就只能每一次都创建新的练习，新的参与学生名单
func TestOperatePracticeStatus(t *testing.T) {

	// 这里只能去测试那个删除与取消发布的分支
	// 确保logger已初始化
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

	initQuestion(t, conn)

	initPaper(t, conn)

	// 初始化题目跟试卷，这个肯定是要有的
	//tx1, _ := conn.Begin(ctx)
	//
	//var practiceName string
	//practiceName = "单元测试练习名"
	//// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	//s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	//VALUES ($1, $2, $3, $4, $5, $6, $7)`
	//_, err = tx1.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//// 这里也随便插入几个学生
	//s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	//_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//var examPaperID1 *int64
	//examPaperID1, _, err = examPaper.GenerateExamPaper(context.Background(), tx1, "02", uid.Int64, uid.Int64, 0, uid.Int64, false)
	//if err != nil {
	//	t.Fatalf("生成考卷逻辑出错：%v", err)
	//}
	//if examPaperID1 == nil {
	//	t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	//}
	//
	//// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除
	//// 这里也要生成对应的practice_submission记录的
	//s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
	//		$1,$2,$3,$4,$5,$6
	//	),($7,$8,$9,$10,$11,$12)`
	//_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08", 10087, uid, 2, *examPaperID1, uid, 1, "00")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
	//
	//err = tx1.Commit(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//}

	// 然后就是操作他变成待发布与删除状态
	tests := []struct {
		name          string
		pid           int64
		status        string
		expectedError error
	}{
		// 当前name加上，当前我应该准备一个当前新增练习应该处理的状态
		{
			name:          "正常1 将待发布练习调整为发布状态 PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "正常2 将已经发布练习调整为待发布状态 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.PendingRelease,
			expectedError: nil,
		},
		{
			name:          "正常3 将待发布练习调整为删除状态 PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常1 触发LoadPracticeById错误 不存在的练习ID PendingRelease",
			pid:           1000000,
			status:        PracticeStatus.Released,
			expectedError: errors.New("非法practiceID , 无该练习记录"),
		},
		{
			name:          "异常2 非法PID PendingRelease",
			pid:           -1,
			status:        PracticeStatus.Released,
			expectedError: errors.New("invalid practice ID param"),
		},
		{
			name:          "异常3 开启事务错误 beginTx PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: errors.New("beginTx called failed"),
		},
		{
			name:          "异常4 触发rollback PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常5 触发rollbackFail PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常6 触发 commit  PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常9 触发GenerateExamPaper错误  pg PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: errors.New("invalid paper question group"),
		},
		{
			name:          "异常10 触发生成考卷 empty PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: errors.New("生成练习考卷返回的考卷ID为空"),
		},
		{
			name:          "异常11 更新练习信息错误 pQuery1 PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: errors.New("更新练习状态 发布->未发布 失败"),
		},
		{
			name:          "异常12 触发批改配置失败 mark PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常14 更新发布练习信息错误 query2 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.PendingRelease,
			expectedError: errors.New("更新练习状态 发布-> 未发布 或 未发布-> 删除 失败"),
		},
		{
			name:          "异常15 更新发布练习信息错误 query3 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.PendingRelease,
			expectedError: errors.New("重置学生练习提交记录信息失败"),
		},
		{
			name:          "异常16 触发清除批改配置错误 mark1 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.PendingRelease,
			expectedError: nil,
		},
		{
			name:          "异常17 要更换的练习状态非法 PendingRelease",
			pid:           uid.Int64,
			status:        "08",
			expectedError: errors.New("非法,请传入合法的练习状态"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()
			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "rollbackFail")
			} else if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "commit")
			} else if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "pg")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "empty")
			} else if containsString(tt.name, "异常11") {
				ctx = context.WithValue(ctx, "force-error", "pQuery1")
			} else if containsString(tt.name, "异常12") {
				ctx = context.WithValue(ctx, "force-error", "mark")
			} else if containsString(tt.name, "异常14") {
				ctx = context.WithValue(ctx, "force-error", "pQuery2")
			} else if containsString(tt.name, "异常15") {
				ctx = context.WithValue(ctx, "force-error", "pQuery3")
			} else if containsString(tt.name, "异常16") {
				ctx = context.WithValue(ctx, "force-error", "mark1")
			} else {
				ctx = context.Background()
			}

			var nowStatus string
			if containsString(tt.name, "PendingRelease") {
				nowStatus = PracticeStatus.PendingRelease
			} else if containsString(tt.name, "Released") {
				nowStatus = PracticeStatus.Released
			} else {
				nowStatus = PracticeStatus.Deleted
			}

			tx1, _ := conn.Begin(context.Background())
			// 这里就需要先创建好所需要的练习了
			var practiceName string
			practiceName = "单元测试练习名"
			// 先创建这个数据，最后测试完毕再删掉 用于更新用的
			s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7,$8)`
			_, err = tx1.Exec(ctx, s, uid, practiceName, "00", uid, 5, "00", uid, nowStatus)
			if err != nil {
				t.Error(err)
			}
			// 这里也随便插入几个学生
			s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
			_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
			if err != nil {
				t.Error(err)
			}
			var examPaperID1 *int64
			examPaperID1, _, err = examPaper.GenerateExamPaper(context.Background(), tx1, "02", uid.Int64, uid.Int64, 0, uid.Int64, false)
			if err != nil {
				t.Errorf("生成考卷逻辑出错：%v", err)
			}
			if examPaperID1 == nil {
				t.Error("生成考卷逻辑出错,返回考卷ID为空")
			}

			// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除
			// 这里也要生成对应的practice_submission记录的
			s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6,$7
			),($8,$9,$10,$11,$12,$13,$14)`
			_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "00", 10087, uid, 2, *examPaperID1, uid, 1, "00")
			if err != nil {
				t.Error(err)
			}
			// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了

			err = tx1.Commit(context.Background())
			if err != nil {
				t.Error(err)
			}

			err = OperatePracticeStatus(ctx, tt.pid, tt.status, uid.Int64)
			if tt.expectedError != nil {
				if !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("OperatePracticeStatus() 期望没有错误，但返回错误: %v", err)
					return
				}
				// 这里没错的话，就需要去查询一下当前的数据了
				if tt.status == PracticeStatus.Released {
					// 这里应该成功生成了考卷 然后就查询一下是否真的存在 但是此时可能会算上原本的数量的
					s = `SELECT COUNT(*) FROM t_exam_paper WHERE practice_id=$1`
					pCount := 0
					err = conn.QueryRow(ctx, s, uid).Scan(&pCount)
					if err != nil {
						t.Error(err)
					}
					if containsString(tt.name, "异常4") {
						if pCount != 1 {
							t.Errorf("这里rollback之后生成的考卷数量不为1：实际为：%v", pCount)
						}
					} else {
						if pCount != 2 {
							t.Errorf("生成的考卷数量不为1：实际为：%v", pCount)
						}
					}

				} else {
					// 这里需要判断一下这个
					nowA := 0
					nowS := PracticeSubmissionStatus.Allow
					s = `SELECT attempt,status FROM t_practice_submissions WHERE practice_id = $1 LIMIT 1`
					err = conn.QueryRow(ctx, s, uid).Scan(&nowA, &nowS)
					if err != nil {
						t.Error(err)
					}
					if nowA != -1 {
						t.Errorf("当前的练习提交记录的尝试次数数量不为-1：实际为：%v", nowA)
					}
					if nowS != PracticeSubmissionStatus.Deleted {
						t.Errorf("当前的练习提交记录的状态不为删除：实际为：%v", nowS)
					}
				}
			}

			t.Cleanup(func() {
				// 这里需要清楚所有的考卷、练习等资源
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
			})
		})

		t.Cleanup(func() {
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
		})
	}
}

// // 测试学生进入练习的三种不同状态 这里使用的是主流程测试数据中的其余学生 这里使用三种情况，就是进入过一次作答但是没有提交的 ， 然后就是完全提交了的、最后是第一次作答的
func TestEnterPracticeGetPaperDetails(t *testing.T) {

	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	//这里还是要创建 这里就是真正需要创建practice_submissions的地方了
	//触发LoadExamPaperDetailByUserId sacn
	//触发GenerateAnswerQuestion query
	//触发LoadPracticeById  不存在的practiceID
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

	initQuestion(t, conn)

	initPaper(t, conn)

	// 初始化题目跟试卷，这个肯定是要有的
	tx1, _ := conn.Begin(ctx)

	var practiceName string
	practiceName = "单元测试练习名"
	// 先创建这个数据，最后测试完毕再删掉 用于更新用的
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7,$8)`
	_, err = tx1.Exec(ctx, s, uid, practiceName, "00", uid, 1, "00", uid, PracticeStatus.Released)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
	_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	var examPaperID1 *int64
	examPaperID1, _, err = examPaper.GenerateExamPaper(context.Background(), tx1, "02", uid.Int64, uid.Int64, 0, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里需要再update一下考卷ID
	s = `UPDATE t_practice SET exam_paper_id = $1 WHERE id = $2`
	_, err = tx1.Exec(ctx, s, *examPaperID1, uid)
	if err != nil {
		t.Fatal(err)
	}
	// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除 一个是08 一个00 分别代表三种状态中的其中两种
	// 这里也要生成对应的practice_submission记录的
	s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
			$1,$2,$3,$4,$5,$6,$7
		),($8,$9,$10,$11,$12,$13,$14),($15,$16,$17,$18,$19,$20,$21)`
	_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08", 10087, uid, 2, *examPaperID1, uid, 1, "00", 10088, uid, 3, *examPaperID1, uid, 1, "06")
	if err != nil {
		t.Fatal(err)
	}

	pscount := 0
	s = `SELECT COUNT(*) FROM t_practice_submissions`
	err = tx1.QueryRow(ctx, s).Scan(&pscount)
	if err != nil {
		t.Fatal(err)
	}
	if pscount != 3 {
		t.Errorf("此时生成的练习提交记录失败，数量不为3")
	}
	// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
	// TODO 下面是去插入一个本身就已经创建好的已作答但是未提交的学生记录的
	// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
	submissionIDs := []int64{10086, 10087}
	now := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(now))
	// 获取考题
	_, pg, pq, err := examPaper.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	Answers := []*cmn.TStudentAnswers{}
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
				if q.Type.String == "00" || q.Type.String == "02" || q.Type.String == "04" {
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
	_limit := 500
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
				"02", a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
				uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
			)
		}
		insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
		_, err := tx1.Exec(ctx, insertQuery, args...)
		if err != nil {
			t.Fatalf("插入学生答卷失败:%v", err)
		}
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

	studentIDs := []int64{1, 2, 3, 4}

	// 1号是已经作答也已提交的
	// 2号是已作答但是没提交
	// 3号是已经提交但是没有批改的
	// 4号是什么都没干的
	tests := []struct {
		name          string
		pid           int64
		uid           int64
		expectedError error
	}{
		{
			name:          "正常1 第一次进入练习作答 生成练习提交记录与考卷",
			pid:           uid.Int64,
			uid:           studentIDs[3],
			expectedError: nil,
		},
		{
			name:          "正常2 进入练习但是此时没有提交就退出了 不生成考卷与提交记录 此时直接返回此时的学生的作答与试卷即可",
			pid:           uid.Int64,
			uid:           studentIDs[1],
			expectedError: nil,
		},
		{
			name:          "正常3 曾经进入练习已经提交了  重新生成考卷与提交记录",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: nil,
		},
		{
			name:          "正常4 覆盖率1 normal-withStudentAnswer-resp",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: nil,
		},
		{
			name:          "正常5 覆盖率2 normal-resp",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: nil,
		},
		{
			name:          "异常1 参数检测 非法pid",
			pid:           -1,
			uid:           studentIDs[0],
			expectedError: errors.New("invalid practiceID | uid param"),
		},
		{
			name:          "异常2 参数检测 非法uid",
			pid:           uid.Int64,
			uid:           -1,
			expectedError: errors.New("invalid practiceID | uid param"),
		},
		{
			name:          "异常2 不存在的pid与uid",
			pid:           100000,
			uid:           100000,
			expectedError: errors.New("查询学生练习视图 v_practice_summary失败"),
		},
		{
			name:          "异常3 此时练习处于待批改状态",
			pid:           uid.Int64,
			uid:           studentIDs[2],
			expectedError: errors.New("目前处于待批改的状态，无法重新进入作答"),
		},
		{
			name:          "异常4 此时练习练习次数已经到达最大了",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: errors.New("已达练习最大次数"),
		},
		{
			name:          "异常5 强制触发数据库错误 pQuery1",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: errors.New("新增一个学生二次练习作答记录失败"),
		},
		{
			name:          "异常6 强制触发重新作答学生分支GenerateAnswerQuestion错误 query",
			pid:           uid.Int64,
			uid:           studentIDs[0],
			expectedError: errors.New("failed batch insert student answer"),
		},
		{
			name:          "异常7 强制触发LoadPracticeById LoadPracticeById",
			pid:           uid.Int64,
			uid:           studentIDs[3],
			expectedError: errors.New("LoadPracticeById call faild"),
		},
		{
			name:          "异常8 强制触发数据库错误 pQuery2",
			pid:           uid.Int64,
			uid:           studentIDs[3],
			expectedError: errors.New("初始化一个学生练习作答记录失败"),
		},
		{
			name:          "异常9 强制触发第一次作答学生分支GenerateAnswerQuestion错误 query",
			pid:           uid.Int64,
			uid:           studentIDs[3],
			expectedError: errors.New("failed batch insert student answer"),
		},
		{
			name:          "异常10 强制触发LoadExamPaperDetailByUserId错误 scan",
			pid:           uid.Int64,
			uid:           studentIDs[1],
			expectedError: errors.New("scan student answer failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx = context.Background()

			// 开一个独立的事务
			tx, _ := conn.Begin(ctx)

			defer tx.Rollback(ctx)

			// 每一个用例再创建独立的事务
			if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "pQuery1")
			} else if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if containsString(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "LoadPracticeById")
			} else if containsString(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "pQuery2")
			} else if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if containsString(tt.name, "正常4") {
				ctx = context.WithValue(ctx, "test", "normal-withStudentAnswer-resp")
			} else if containsString(tt.name, "正常5") {
				ctx = context.WithValue(ctx, "test", "normal-resp")
			} else {
				ctx = context.Background()
			}

			// 这里查询一下这个视图，因为数据已经存在了的
			if containsString(tt.name, "异常4") {
				var ps cmn.TVPracticeSummary
				s := `SELECT allowed_attempts,attempt_count,latest_unsubmitted_id, latest_submitted_id,pending_mark_id, exam_paper_id,paper_name,suggested_duration
	 FROM assessuser.v_practice_summary 
	 WHERE id = $1 AND student_id = $2 AND practice_status = $3 
	 AND practice_student_status != $4`
				tx.QueryRow(context.Background(), s, tt.pid, tt.uid, PracticeStatus.Released, PracticeStudentStatus.Deleted).Scan(&ps.AllowedAttempts, &ps.AttemptCount, &ps.LatestUnsubmittedID,
					&ps.LatestSubmittedID, &ps.PendingMarkID, &ps.ExamPaperID,
					&ps.PaperName, &ps.SuggestedDuration)
				if err != nil {
					t.Errorf("无法查询到这项记录")
				}
				t.Logf("打印输出一下这个最大尝试次数：%v ; 目前次数%v", ps.AllowedAttempts.Int64, ps.AttemptCount.Int64)
			}

			if containsString(tt.name, "异常5") || containsString(tt.name, "异常6") || containsString(tt.name, "正常3") {
				s = `UPDATE t_practice SET allowed_attempts = $1`
				_, err = conn.Exec(context.Background(), s, 2)
				if err != nil {
					t.Errorf("无法更新需要的练习为需要的次数")
				}
			}

			info, pg, pq, err := EnterPracticeGetPaperDetails(ctx, tx, tt.pid, tt.uid)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("返回的错误不符合预期：%v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if info == nil {
					t.Errorf("返回的试卷信息指针为空：异常")
				}

				// 这里的如果此时是已经作答过的学生进来，必须
				if containsString(tt.name, "此时直接返回此时的学生的作答与试卷即可") {
					// 这里的话，就需要检查此时是否携带答案了
					for _, qList := range pq {
						for _, q := range qList {
							// 这里开始检测
							// 这里就要解码出来，然后判断是否与他相等
							// 3. 严格顺序验证
							expected := []byte(`"hello"`)
							if !bytes.Equal(q.StudentAnswer, expected) {
								t.Errorf("此时没有正确返回学生作答内容")
							}
							if q.StudentScore.Valid || q.StudentScore.Float64 != 0 {
								t.Errorf("此时不应该返回学生作答分数的")
							}
							if q.Analysis.Valid || q.Analysis.String != "" {
								t.Errorf("此时不应该返回题目解析 却返回了 与预期不符")
							}
							if len(q.Answers) != 0 {
								t.Errorf("查询出来答案:%v 预期应该是为空的", q.Answers.String())
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

			// 这里就要还原上面的一切
			t.Cleanup(func() {
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
				// 最后再生成一下

				tx1, _ := conn.Begin(ctx)
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7,$8)`
				_, err = tx1.Exec(ctx, s, uid, practiceName, "00", uid, 1, "00", uid, PracticeStatus.Released)
				if err != nil {
					t.Fatal(err)
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
				var examPaperID1 *int64
				examPaperID1, _, err = examPaper.GenerateExamPaper(context.Background(), tx1, "02", uid.Int64, uid.Int64, 0, uid.Int64, false)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里需要再update一下考卷ID
				s = `UPDATE t_practice SET exam_paper_id = $1 WHERE id = $2`
				_, err = tx1.Exec(ctx, s, *examPaperID1, uid)
				if err != nil {
					t.Fatal(err)
				}

				// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除 一个是08 一个00 分别代表三种状态中的其中两种
				// 这里也要生成对应的practice_submission记录的
				s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
			$1,$2,$3,$4,$5,$6,$7
		),($8,$9,$10,$11,$12,$13,$14),($15,$16,$17,$18,$19,$20,$21)`
				_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08", 10087, uid, 2, *examPaperID1, uid, 1, "00", 10088, uid, 3, *examPaperID1, uid, 1, "06")
				if err != nil {
					t.Fatal(err)
				}
				// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
				// TODO 下面是去插入一个本身就已经创建好的已作答但是未提交的学生记录的
				// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
				submissionIDs := []int64{10086, 10087}
				now := time.Now().UnixMilli()
				r := rand.New(rand.NewSource(now))
				// 获取考题
				_, pg, pq, err := examPaper.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
				if err != nil {
					t.Fatalf("查询考卷题目错误：%v", err)
				}

				Answers := []*cmn.TStudentAnswers{}
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
							if q.Type.String == "00" || q.Type.String == "02" || q.Type.String == "04" {
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
				_limit := 500
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
							"02", a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
							uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
						)
					}
					insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
					_, err := tx1.Exec(ctx, insertQuery, args...)
					if err != nil {
						t.Fatalf("插入学生答卷失败:%v", err)
					}
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
			})
		})

		t.Cleanup(func() {
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

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 这里初始化题目，到达其他的函数，我这里一样可以进行删除
func initQuestion(t *testing.T, tx *pgxpool.Pool) {
	qb := `INSERT INTO t_question_bank  (id,type,name,creator,create_time,status,access_mode) VALUES($1, $2, $3, $4, $5, $6,$7)`
	_, err := tx.Exec(ctx, qb, uid, "00", "考卷单元测试题库", uid, now, "00", "00")
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
			`["A", "D"]`, 2, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{2, "00", "<p><span style=\"font-size: 12pt\">H3C公司的总部位于哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">2</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">3</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">4</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">5</span></p>"}]`,
			`["A", "D"]`, 2, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{3, "02", "<p><span style=\"font-size: 12pt\">H3C公司的分部遍布哪个城市？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">广州</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">上海</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">北京</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">深圳</span></p>"}]`,
			`["A", "B","C","D"]`, 5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{4, "02", "<p><span style=\"font-size: 12pt\"> 以下哪些是常见的网络设备品牌？</p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">华为</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">思科</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\"> Juniper</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">中兴</span></p>"}]`,
			`["A", "B"]`, 5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{5, "02", "<p><span style=\"font-size: 12pt\">以下属于H3C主要产品线的有哪些？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">路由器</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">交换机</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">防火墙</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">服务器</span></p>"}]`,
			`["A","B","C"]`, 5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{6, "00", "<p><span style=\"font-size: 12pt\">以下哪些属于H3C的核心技术领域？</span></p>",
			`[{"label": "A", "value": "<p><span style=\"font-size: 12pt\">云计算</span></p>"}, {"label": "B", "value": "<p><span style=\"font-size: 12pt\">大数据</span></p>"}, {"label": "C", "value": "<p><span style=\"font-size: 12pt\">人工智能</span></p>"}, {"label": "D", "value": "<p><span style=\"font-size: 12pt\">物联网</span></p>"}]`,
			`["A"]`, 2, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{7, "04", "<p><span style=\"font-size: 12pt\">在H3C设备上，'undo shutdown'命令可以启用一个物理接口。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{8, "04", "<p><span style=\"font-size: 12pt\">H3C交换机的默认管理VLAN编号是1。</span></p>",
			`[{"Label": "A", "Value": "<p><span style=\"font-size: 12pt\">正确</span></p>"}, {"Label": "B", "Value": "<p><span style=\"font-size: 12pt\">错误</span></p>"}]`,
			`["A"]`, 2, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{9, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "螺旋模型","grading_rule": "答案必须准确匹配“螺旋模型”","alternative_answers": []}]`, 5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{10, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">软件开发中，用于描述系统功能的文档是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 5,"answer": "需求规格说明书","grading_rule": "答案必须准确匹配“需求规格说明书”","alternative_answers": []}]`, 5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{11, "06", "<p><span style=\"font-family: 等线; font-size: 12pt\">面向对象编程的三大基本特性分别是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>",
			nil, `[{"index": 1,"score": 1,"answer": "封装","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 2,"score": 1,"answer": "继承","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []},{"index": 3,"score": 1,"answer": "多态","grading_rule": "顺序不限，必须包含三个特性","alternative_answers": []}]`,
			3, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{12, "08", "<p>简述计算机病毒的传播途径。</p>", nil, `[{"index": 1,"score": 5,"answer": "通过网络下载、移动存储设备、电子邮件等方式传播","grading_rule": "答出3种主要传播方式即可得满分","alternative_answers": []}]`,
			5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
		},
		{13, "08", "<p>简述操作系统的主要功能。</p>", nil, `[{"index": 1,"score": 5,"answer": "进程管理、内存管理、文件管理、设备管理、作业管理","grading_rule": "答出3种主要功能即可得满分","alternative_answers": []}]`,
			5, "<p>面向对象的三大特性是封装、继承、多态。</p>", nil, uid, "00", uid, 1,
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
		AccessMode:        null.StringFrom("00"), // 默认访问模式
	}
	p := `INSERT INTO t_paper
			(id,name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status, access_mode)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,$13)`
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
		paper.AccessMode.String,
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
func shuffleOptionsAndMapAnswers(r *rand.Rand, qid int64, Options, Answers []byte) ([]byte, []byte, error) {
	var options []examPaper.QuestionOption
	if err := json.Unmarshal(Options, &options); err != nil {
		return nil, nil, fmt.Errorf("考题ID：%v的选项反序列化失败: %w", qid, err)
	}

	labelToValue := make(map[string]string)
	for _, opt := range options {
		labelToValue[opt.Label] = opt.Value
	}

	values := make([]string, len(options))
	for i, opt := range options {
		values[i] = opt.Value
	}
	r.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})

	for i := range options {
		options[i].Value = values[i] // 仅替换值，标签不变
	}

	var originalAnswers []string
	if err := json.Unmarshal(Answers, &originalAnswers); err != nil {
		return nil, nil, fmt.Errorf("考题ID:%v答案反序列化失败: %w", qid, err)
	}

	newAnswers := make([]string, 0, len(originalAnswers))
	for _, origLabel := range originalAnswers {
		originalValue := labelToValue[origLabel]
		for _, opt := range options {
			if opt.Value == originalValue {
				newAnswers = append(newAnswers, opt.Label)
				break
			}
		}
	}

	newOptionsJSON, _ := json.Marshal(options)
	newAnswersJSON, _ := json.Marshal(newAnswers)
	return newAnswersJSON, newOptionsJSON, nil
}
