/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理数据库层函数逻辑测试
 * @Date: 2025-07-24 14:51:50
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-25 10:31:28
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
	examPaperService "w2w.io/serve/examPaper"
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

	s = `DELETE FROM t_practice WHERE name = $1`
	_, err = conn.Exec(ctx, s, practiceName)
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

	// 这里再删除掉练习学生名单的数据
	s := `DELETE FROM assessuser.t_practice_student`
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
	s = `DELETE FROM t_practice`
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
		isClear       bool
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
			err := UpdatePractice(ctx, tt.p, tt.ps, tt.uid, tt.Ostatus, tt.isClear)
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

	s := `DELETE FROM assessuser.t_exam_paper_question`
	_, err := conn.Exec(ctx, s)
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
	s = `DELETE FROM t_practice`
	_, err = conn.Exec(ctx, s)
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
	s = `INSERT INTO t_paper(id,name,assembly_type,category,level,creator,status,domain_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err = conn.Exec(ctx, s, uid, paperName, "00", "02", "00", uid, "00", 10086)
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
			s = `DELETE FROM assessuser.t_practice`
			_, err := conn.Exec(ctx, s)
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
			s = `DELETE FROM assessuser.t_paper`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
		})
	}

}

// 仅仅是获取这个练习信息
func TestLoadPracticeByIDs(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	conn := cmn.GetPgxConn()
	var practiceName1, practiceName2, practiceName3 string
	practiceName1 = "单元测试练习名1"
	practiceName2 = "单元测试练习名2"
	practiceName3 = "单元测试练习名3"
	var uid int64
	uid = 10086

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
		t.Fatal(err)
	}
	// 这里再删除掉练习学生名单的数据
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	// 先创建这个数据，最后测试完毕再删掉
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,exam_paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7,$8),($9,$10,$11,$12,$13,$14,$15,$16),($17,$18,$19,$20,$21,$22,$23,$24)`
	_, err = conn.Exec(ctx, s, 10086, practiceName1, "00", uid, 10086, "00", 10086, 10086, 10087, practiceName2, "00", uid, 10087, "00", 10087, 10087, 10088, practiceName3, "00", uid, 10088, "00", 10088, 10088)
	if err != nil {
		t.Fatal(err)
	}

	s = `UPDATE t_practice SET status = $1 , addi = $2`
	_, err = conn.Exec(ctx, s, PracticeStatus.PendingRelease, types.JSONText(`[]`))
	if err != nil {
		t.Fatal(err)
	}

	// 这里去更新一下的
	s = `UPDATE t_practice SET status = $1 WHERE name = $2`
	_, err = conn.Exec(ctx, s, PracticeStatus.Deleted, practiceName3)
	if err != nil {
		t.Fatal(err)
	}

	pids := []int64{10086, 10087, 10088}
	// 这里准备几个猜测是用例

	tests := []struct {
		name        string
		ids         []int64
		expectedErr error
	}{
		{
			name:        "正常查询1 能返回两个数据",
			ids:         pids,
			expectedErr: nil,
		},
		{
			name:        "异常1 非法练习ID数组",
			ids:         []int64{},
			expectedErr: errors.New("非法practiceIDs"),
		},
		{
			name:        "异常2 强制触发查询失败",
			ids:         pids,
			expectedErr: errors.New("批量查询练习数据失败"),
		},
		{
			name:        "异常3 强制触发查询成功后扫描失败",
			ids:         pids,
			expectedErr: errors.New("批量解析练习数据失败"),
		},
		{
			name:        "异常4 触发查询练习记录为空",
			ids:         []int64{100000, 100001},
			expectedErr: errors.New("批量查询练习记录失败，记录为空"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx = context.Background()
			// 这里去准备这个语句
			if containsString(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "lQuery")
			} else if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "lScan")
			} else {
				ctx = context.Background()
			}

			ps, err := LoadPracticeByIDs(ctx, tt.ids)
			if tt.expectedErr != nil {
				if !containsString(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedErr)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePractice() 期望没有错误，但返回错误: %v", err)
					return
				}

				if len(ps) != 2 {
					// 数量
					t.Errorf("查询出来的数量与预期不符 实际%v", len(ps))
				}
				// 这里能成功拿到之后，就需要遍历这个数量
				for id, p := range ps {
					if id != p.ID.Int64 {
						t.Errorf("映射的练习ID错误")
					}
					// 判断这些数据是否与预期一致
					if p.AllowedAttempts.Int64 != p.ID.Int64 {
						t.Errorf("查询的允许作答次数与预期不符 实际：%v", p.AllowedAttempts.Int64)
					}
					if p.PaperID.Int64 != p.ID.Int64 {
						t.Errorf("查询的试卷ID与预期不符 实际：%v", p.PaperID.Int64)
					}
					if p.ExamPaperID.Int64 != p.ID.Int64 {
						t.Errorf("查询的考卷ID与预期不符 实际：%v", p.ExamPaperID.Int64)
					}
				}

			}
		})

		t.Cleanup(func() {
			// 清理数据
			s := `DELETE FROM t_practice`
			_, err := conn.Exec(ctx, s)
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
		isClear       bool
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
			err := UpsertPractice(context.Background(), tt.p, tt.ps, tt.uid, tt.isClear)
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
			name:          "强制异常4 rollback",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
		{
			name:          "强制异常5 commit",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsString(tt.name, "强制异常4") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if containsString(tt.name, "强制异常2") {
				ctx = context.WithValue(ctx, "force-error", "query1")
			} else if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "强制异常5") {
				ctx = context.WithValue(ctx, "force-error", "commit")
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
				if !containsString(tt.name, "但不会报错") && !containsString(tt.name, "强制异常4") && !containsString(tt.name, "强制异常5") {
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

func TestUpsertPracticeStudentV2(t *testing.T) {
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

	// 现在是已经插入了两个学生 就可以去查看一下这个学生名单，看看是否符合

	tests := []struct {
		name          string
		pid           int64
		uid           int64
		ps            []int64
		expectedError error
	}{
		{
			name:          "正常1 能正常处理空数组 全删除学生名单",
			pid:           uid,
			uid:           uid,
			ps:            []int64{},
			expectedError: nil,
		},
		{
			name:          "正常2 能正常处理有数据数组 更换学生名单",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
		{
			name:          "参数异常1 非法uid",
			pid:           uid,
			uid:           int64(-1),
			ps:            []int64{2, 3},
			expectedError: errors.New("invalid uid param"),
		},
		{
			name:          "参数异常2 非法pid",
			pid:           int64(-1),
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("invalid practiceId param"),
		},
		{
			name:          "强制异常1 beginTx",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("beginTx called failed"),
		},
		{
			name:          "强制异常2 query2",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("add PracticeStudent call failed"),
		},
		{
			name:          "强制异常3 query1",
			pid:           uid,
			uid:           uid,
			ps:            []int64{},
			expectedError: errors.New("clear PracticeStudent call failed"),
		},
		{
			name:          "强制异常4 rollback",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
		{
			name:          "强制异常5 commit",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: nil,
		},
		{
			name:          "强制异常6 query3",
			pid:           uid,
			uid:           uid,
			ps:            []int64{2, 3},
			expectedError: errors.New("delete PracticeStudent call failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()
			if containsString(tt.name, "强制异常4") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if containsString(tt.name, "强制异常3") {
				ctx = context.WithValue(ctx, "force-error", "query1")
			} else if containsString(tt.name, "强制异常6") {
				ctx = context.WithValue(ctx, "force-error", "query3")
			} else if containsString(tt.name, "强制异常2") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "强制异常5") {
				ctx = context.WithValue(ctx, "force-error", "commit")
			} else {
				ctx = context.Background()
			}

			err = UpsertPracticeStudentV2(ctx, tt.pid, tt.uid, tt.ps)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpsertPracticeStudent() 期望没有错误，但返回错误: %v", err)
					return
				}

				if !containsString(tt.name, "强制异常4") && !containsString(tt.name, "强制异常5") && !containsString(tt.name, "正常1") {
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
				if containsString(tt.name, "正常1") {
					// 正常应该是全删除了
					s := `SELECT COUNT(*) FROM t_practice_student WHERE practice_id = $1 AND status = $2`
					var count int
					err := conn.QueryRow(ctx, s, uid, "00").Scan(&count)
					if err != nil {
						t.Fatal(err)
					}
					if count != 0 {
						t.Errorf("此时没有全删除")
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
		})
	}
}

// // 测试学生权限获取练习作答列表 这里虽然无数据，但也是可以
func TestListPracticeS(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 在这里需要有提交记录 还有
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

	tx, _ := conn.Begin(ctx)

	var practiceName1, practiceName2 string

	practiceName1 = "单元测试发布练习1"
	practiceName2 = "单元测试发布练习2"

	// 先创建数据，不管到底是不是这个测试用例需要的 三种练习状态 然后还有状态不一的练习
	// 删除 exam_paper_id 字段后，每组数据保留 8 个字段，调整占位符编号
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
VALUES 
($1, $2, $3, $4, $5, $6, $7, $8),
($9, $10, $11, $12, $13, $14, $15, $16)`

	_, err = tx.Exec(ctx, s,
		// 第 1 组数据
		10086, practiceName1, "00", uid, 10086, "00", 10086, PracticeStatus.Released,
		// 第 2 组数据
		10087, practiceName2, "00", uid, 10087, "00", 10086, PracticeStatus.Released,
	)

	// 这里也随便插入已发布练习的几个学生 每个练习都创建两个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
	_, err = tx.Exec(ctx, s,
		1, 10086, uid, "00",
		2, 10086, uid, "00",
		1, 10087, uid, "00",
		2, 10087, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	//paperID := []int64{10086, 10087}
	//pid2ExamPaperID := make(map[int64]int64)
	//for _, id := range paperID {
	//	epid, _, err := examPaperService.GenerateExamPaper(ctx, tx, "02", uid.Int64, id, 0, uid.Int64, false)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	pid2ExamPaperID[id] = *epid
	//
	//	// 这里要更新对应的考卷ID
	//	s = `UPDATE t_practice SET exam_paper_id = $1 WHERE id = $2`
	//	_, err = tx.Exec(ctx, s, *epid, id)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//}
	paperID := []int64{10086, 10087}
	pid2ExamPaperID := make(map[int64]int64)
	for _, id := range paperID {
		// 1. 加载试卷模板
		p, pg, pq, err := examPaperService.LoadPaperTemplateById(ctx, uid.Int64, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(pg) == 0 {
			t.Fatal("invalid paper question group")
		}
		now := time.Now().UnixMilli()

		// 2. 插入考卷
		var epID int64
		insertPaperSQL := `
        INSERT INTO assessuser.t_exam_paper 
            (exam_session_id, practice_id, name, creator, create_time, update_time) 
        VALUES ($1, $2, $3, $4, $5, $6) 
        RETURNING id`
		err = tx.QueryRow(ctx, insertPaperSQL, nil, id, p.Name, uid.Int64, now, now).Scan(&epID)
		if err != nil {
			t.Fatal(fmt.Errorf("插入考卷失败: %w", err))
		}
		pid2ExamPaperID[id] = epID

		// 3. 插入题组
		groupArgs := make([]interface{}, 0, len(pg)*6)
		groupPlaceholders := make([]string, 0, len(pg))
		for i, g := range pg {
			ph := fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6)
			groupPlaceholders = append(groupPlaceholders, ph)
			groupArgs = append(groupArgs, epID, g.Name, g.Order, uid.Int64, now, now)
		}
		groupSQL := fmt.Sprintf(
			`INSERT INTO assessuser.t_exam_paper_group 
            (exam_paper_id, name, "order", creator, create_time, update_time) 
        VALUES %s RETURNING id, "order"`, strings.Join(groupPlaceholders, ","))
		rows, err := tx.Query(ctx, groupSQL, groupArgs...)
		if err != nil {
			t.Fatal(fmt.Errorf("插入题组失败: %w", err))
		}
		defer rows.Close()
		orderToNewIDMap := make(map[int64]int64)
		for rows.Next() {
			var newGroupID, order int64
			if err := rows.Scan(&newGroupID, &order); err != nil {
				t.Fatal(fmt.Errorf("扫描题组信息失败:%v", err))
			}
			orderToNewIDMap[order] = newGroupID
		}
		if err := rows.Err(); err != nil {
			t.Fatal(fmt.Errorf("遍历扫描题组信息中出错：%v", err))
		}

		// 4. 插入题目
		var tqs []*cmn.TExamPaperQuestion
		for _, origGroup := range pg {
			newGroupID, exists := orderToNewIDMap[origGroup.Order.Int64]
			if !exists {
				t.Fatal(fmt.Errorf("无法找到顺序为:%v的题组", origGroup.Order.Int64))
			}
			qs, exists := pq[origGroup.ID.Int64]
			if !exists || len(qs) == 0 {
				continue
			}
			for _, q := range qs {
				tq := &cmn.TExamPaperQuestion{
					Score:                   q.Score,
					Type:                    null.StringFrom(q.Type),
					Content:                 q.Content,
					Options:                 q.Options,
					Answers:                 q.Answers,
					Analysis:                q.Analysis,
					Title:                   q.Title,
					AnswerFilePath:          q.AnswerFilePath,
					TestFilePath:            q.TestFilePath,
					Input:                   q.Input,
					Output:                  q.Output,
					Example:                 q.Example,
					Repo:                    q.Repo,
					Order:                   q.Order,
					Creator:                 null.IntFrom(uid.Int64),
					CreateTime:              null.IntFrom(now),
					UpdateTime:              null.IntFrom(now),
					Addi:                    q.Addi,
					GroupID:                 null.IntFrom(newGroupID),
					QuestionAttachmentsPath: q.QuestionAttachmentsPath,
				}
				tqs = append(tqs, tq)
			}
		}
		// 批量插入题目
		_limit := 1000
		questionSQL := `INSERT INTO assessuser.t_exam_paper_question (
        score, type, content, options, answers, analysis, title, 
        answer_file_path, test_file_path, input, output, example, repo, 
        creator, create_time, update_time, addi, "order", group_id, question_attachments_path
    ) VALUES %s RETURNING id`
		for start := 0; start < len(tqs); start += _limit {
			end := start + _limit
			if end > len(tqs) {
				end = len(tqs)
			}
			batch := tqs[start:end]
			var (
				placeholders []string
				questionArgs []interface{}
				paramIndex   = 1
			)
			for _, s := range batch {
				phFields := make([]string, 20)
				for i := 0; i < 20; i++ {
					phFields[i] = fmt.Sprintf("$%d", paramIndex)
					paramIndex++
				}
				placeholders = append(placeholders, "("+strings.Join(phFields, ",")+")")
				questionArgs = append(questionArgs,
					s.Score,
					s.Type,
					s.Content,
					s.Options,
					s.Answers,
					s.Analysis,
					s.Title,
					s.AnswerFilePath,
					s.TestFilePath,
					s.Input,
					s.Output,
					s.Example,
					s.Repo,
					s.Creator,
					s.CreateTime,
					s.UpdateTime,
					s.Addi,
					s.Order,
					s.GroupID,
					s.QuestionAttachmentsPath,
				)
			}
			fullQuestionSQL := fmt.Sprintf(questionSQL, strings.Join(placeholders, ","))
			batchRows, err := tx.Query(ctx, fullQuestionSQL, questionArgs...)
			if err != nil {
				t.Fatal(fmt.Errorf("插入题目批次 [%d-%d] 失败: %v", start, end-1, err))
			}
			batchIndex := start
			for batchRows.Next() {
				var id int64
				if err := batchRows.Scan(&id); err != nil {
					batchRows.Close()
					t.Fatal(fmt.Errorf("扫描题目ID失败 [%d]: %v", batchIndex, err))
				}
				if batchIndex < len(tqs) {
					tqs[batchIndex].ID = null.IntFrom(id)
				}
				batchIndex++
			}
			batchRows.Close()
			if err := batchRows.Err(); err != nil {
				t.Fatal(fmt.Errorf("读取题目ID失败 [%d-%d]: %v", start, end-1, err))
			}
			if batchIndex != end {
				t.Fatal(fmt.Errorf("题目ID数量不匹配: 预期%d, 实际%d", end-start, batchIndex-start))
			}
		}

		// 5. 更新 t_practice 表的 exam_paper_id 字段
		s := `UPDATE t_practice SET exam_paper_id = $1 WHERE id = $2`
		_, err = tx.Exec(ctx, s, epID, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 给每一个练习都创建练习提交记录
	for practiceID, examPaperID := range pid2ExamPaperID {
		var id int64
		PracticeSubmissionsIDs := []int64{}

		// 这里为什么只生成了一个练习的提交记录呢
		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 1, examPaperID, uid, 1, "08").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 1, examPaperID, uid, 2, "00").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

		// 如果是学生2的话，就剩下一个记录给他
		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 2, examPaperID, uid, 1, "08").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

		// 那这里估计要构建学生答卷 然后是那种有答卷的才对
		// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
		now := time.Now().UnixMilli()
		r := rand.New(rand.NewSource(now))
		// 获取考题
		_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx, examPaperID, true, true, true)
		if err != nil {
			t.Fatalf("查询考卷题目错误：%v", err)
		}
		var Answers []*cmn.TStudentAnswers
		for _, pid := range PracticeSubmissionsIDs {
			order := 1
			for _, g := range pg {
				pqList := pq[g.ID.Int64]
				r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
				for _, q := range pqList {
					answer := &cmn.TStudentAnswers{
						QuestionID:           q.ID,
						GroupID:              g.ID,
						Order:                null.IntFrom(int64(order)),
						PracticeSubmissionID: null.IntFrom(pid),
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
					"02", a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
					uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
				)
			}
			insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
			t.Logf("打印一下，能成功执行到插入学生作答前")
			_, err = tx.Exec(ctx, insertQuery, args...)
			if err != nil {
				t.Fatalf("插入学生答卷失败：%v", err)
			}

			// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis 这里是全部都更新，感觉不对
			s = `UPDATE assessuser.t_student_answers sa
				SET answer = $1, answer_score = $2
				FROM assessuser.t_practice_submissions pb  -- 关联表放在 FROM 子句
				WHERE sa.practice_submission_id = pb.id   -- 连接条件在 WHERE 中
				AND pb.status = $3;                       -- 业务过滤条件 `
			answer := JSONText(`"hello"`)
			answerScore := 1
			_, err = tx.Exec(ctx, s, answer, answerScore, PracticeSubmissionStatus.Marked)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// 这里就已经准备成功了
	err = tx.Commit(ctx)
	if err != nil {
		t.Errorf("事务提交失败：%v", err)
	}

	tests := []struct {
		name          string
		pType         string
		PName         string
		difficulty    string
		order         []string
		page          int
		pageSize      int
		expectedError error
		expectedNum   int
	}{
		{
			name:          "正常1 不进行其余筛选查询",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   2,
		},
		{
			name:          "正常2 练习名称1模糊搜索",
			pType:         PracticeType.Classical,
			PName:         "1",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
		{
			name:          "正常2 练习名称2模糊搜索",
			pType:         PracticeType.Classical,
			PName:         "2",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   1,
		},
		{
			name:          "正常3 练习简单难度搜索",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "00",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   0,
		},
		{
			name:          "正常4 练习困难难度搜索",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "04",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   0,
		},
		{
			name:          "正常5 练习中等难度搜索",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "02",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   2,
		},
		{
			name:          "正常6 练习ID倒序排序",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "02",
			order:         []string{"id desc"},
			page:          1,
			pageSize:      10,
			expectedError: nil,
			expectedNum:   2,
		},
		{
			name:          "异常1 练习类型缺少",
			pType:         "",
			PName:         "",
			difficulty:    "02",
			order:         []string{"id desc"},
			page:          1,
			pageSize:      10,
			expectedError: errors.New("invalid practice type param"),
		},
		{
			name:          "异常2 强制错误 sQuery1",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: errors.New("查询学生权限练习列表失败"),
		},
		{
			name:          "异常3 强制触发错误关闭数据库连接数据 row close",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: nil,
		},
		{
			name:          "异常4 强制触发解析练习数据错误 sQuery2",
			pType:         PracticeType.Classical,
			PName:         "",
			difficulty:    "",
			order:         nil,
			page:          1,
			pageSize:      10,
			expectedError: errors.New("解析练习数据失败"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()
			if containsString(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "sQuery1")
			} else if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "row close")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "sQuery2")
			} else {
				ctx = context.Background()
			}

			// 这个是学生1的记录
			psLists, total, err := ListPracticeS(ctx, tt.pType, tt.PName, tt.difficulty, tt.order, tt.page, tt.pageSize, 1)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("UpsertPracticeStudent() 期望没有错误，但返回错误: %v", err)
					return
				}

				if !containsString(tt.name, "异常3") {

					if len(psLists) != tt.expectedNum {
						t.Errorf("返回的练习列表数量%v,与预期%v不一致", len(psLists), tt.expectedNum)
					}
					if total != tt.expectedNum {
						t.Errorf("返回的练习列表数量%v,与预期%v不一致", total, tt.expectedNum)
					}

					for idx, ps := range psLists {
						// 开始遍历返回的数据
						if containsString(tt.name, "正常6") {
							if idx == 0 {
								if ps.ID.Int64 != 10087 {
									t.Errorf("不是以id的降序进行排序")
								}
							}
						}
						if !containsString(ps.Name.String, "单元测试发布练习") {
							t.Errorf("查询的练习名称有问题")
						}
						//每个练习都有这个记录的
						if ps.WrongCount.Int64 != 13 {
							// 这里应该是错题数是0，因此只会看最近的提交记录
							t.Errorf("此时错题数应该是13 实际上：%v", ps.WrongCount.Int64)
						}
						if ps.TotalScore.Float64 != 13 {
							// 这里是曾经的最高分
							t.Errorf("此时得分应该是13 ，实际上：%v", ps.TotalScore.Float64)
						}
						if ps.HighestScore.Float64 != 13 {
							t.Errorf("此时最得分应该是13 ，实际上：%v", ps.HighestScore.Float64)
						}
						if ps.PaperTotalScore.Float64 != 48 {
							// 这里是曾经的最高分
							t.Errorf("此时试卷总分应该是48 ，实际上：%v", ps.PaperTotalScore.Float64)
						}
						if ps.Difficulty.String != "02" {
							//此时的错题数
							t.Errorf("此时练习的难度不是中等02 实际：%v", ps.Difficulty.String)
						}
						if ps.AttemptCount.Int64 != 2 {
							t.Errorf("此时的作答次数不是2 实际：%v", ps.AttemptCount.Int64)
						}
						if ps.AllowedAttempts.Int64 != ps.ID.Int64 {
							t.Errorf("此时的可作答次数不为：%v,实际为：%v", ps.ID.Int64, ps.AllowedAttempts.Int64)
						}
						// 这里要加上这个如果有未提交的话，就应该是要有值的
						if !ps.LatestUnsubmittedID.Valid || ps.LatestUnsubmittedID.Int64 <= 0 {
							t.Errorf("此时应该出现有未提交的ID")
						}
					}
					// 这个是学生2的记录
					psLists, total, err = ListPracticeS(ctx, tt.pType, tt.PName, tt.difficulty, tt.order, tt.page, tt.pageSize, 2)
					if tt.expectedError != nil {
						if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
							t.Errorf("%v报错与预期：%v", err, tt.expectedError)
						}
					} else {
						if err != nil {
							t.Errorf("UpsertPracticeStudent() 期望没有错误，但返回错误: %v", err)
							return
						}
						if len(psLists) != tt.expectedNum {
							t.Errorf("返回的练习列表数量%v,与预期%v不一致", len(psLists), tt.expectedNum)
						}
						if total != tt.expectedNum {
							t.Errorf("返回的练习列表数量%v,与预期%v不一致", total, tt.expectedNum)
						}

						for _, ps := range psLists {
							// 开始遍历返回的数据
							if !containsString(ps.Name.String, "单元测试发布练习") {
								t.Errorf("查询的练习名称有问题")
							}
							//每个练习都有这个记录的
							if ps.WrongCount.Int64 != 13 {
								// 这里应该是错题数是0，因此只会看最近的提交记录
								t.Errorf("此时错题数应该是13 实际上：%v", ps.WrongCount.Int64)
							}
							if ps.TotalScore.Float64 != 13 {
								// 这里是曾经的最高分
								t.Errorf("此时最高分应该是13 ，实际上：%v", ps.TotalScore.Float64)
							}
							if ps.HighestScore.Float64 != 13 {
								t.Errorf("此时最高得分应该是13 ，实际上：%v", ps.HighestScore.Float64)
							}
							if ps.PaperTotalScore.Float64 != 48 {
								// 这里是曾经的最高分
								t.Errorf("此时试卷总分应该是48 ，实际上：%v", ps.PaperTotalScore.Float64)
							}
							if ps.Difficulty.String != "02" {
								//此时的错题数
								t.Errorf("此时练习的难度不是中等02 实际：%v", ps.Difficulty.String)
							}
							if ps.AttemptCount.Int64 != 1 {
								t.Errorf("此时的作答次数不是1 实际：%v", ps.AttemptCount.Int64)
							}
							if ps.AllowedAttempts.Int64 != ps.ID.Int64 {
								t.Errorf("此时的可作答次数不为：%v,实际为：%v", ps.ID.Int64, ps.AllowedAttempts.Int64)
							}
							if ps.LatestUnsubmittedID.Valid && ps.LatestUnsubmittedID.Int64 > 0 {
								t.Errorf("此时不应该出现有未提交的ID")
							}

						}
					}
				}
			}
		})

		t.Cleanup(func() {
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
		})
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
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
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
		name                     string
		pid                      int64
		status                   string
		expectedSubmissionStatus string
		expectedError            error
	}{
		// 当前name加上，当前我应该准备一个当前新增练习应该处理的状态
		{
			name:                     "正常1 将待发布练习调整为发布状态 PendingRelease",
			pid:                      uid.Int64,
			status:                   PracticeStatus.Released,
			expectedSubmissionStatus: PracticeSubmissionStatus.Allow,
			expectedError:            nil,
		},
		{
			name:                     "正常2 将已经发布练习调整为作废 Released",
			pid:                      uid.Int64,
			status:                   PracticeStatus.Disabled,
			expectedSubmissionStatus: PracticeSubmissionStatus.Disabled,
			expectedError:            nil,
		},
		{
			name:                     "正常3 将发布练习调整为删除状态 Released",
			pid:                      uid.Int64,
			status:                   PracticeStatus.Deleted,
			expectedSubmissionStatus: PracticeSubmissionStatus.Deleted,
			expectedError:            nil,
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
		//{
		//	name:          "异常9 触发GenerateExamPaper错误  pg PendingRelease",
		//	pid:           uid.Int64,
		//	status:        PracticeStatus.Released,
		//	expectedError: errors.New("invalid paper question group"),
		//},
		{
			name:          "异常10 触发生成考卷错误  PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: errors.New("查看练习绑定的试卷中已发布的考卷ID失败"),
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
			name:          "异常21 强制获取到学生作答 pQuery2 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Deleted,
			expectedError: errors.New("遍历查询是否有学生作答记录失败"),
		},
		{
			name:          "异常24 强制获取到有学生曾经作答过练习 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Deleted,
			expectedError: errors.New("练习已有学生参与作答，不能删除"),
		},
		{
			name:          "异常22 强制更新练习失败 pQuery3 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Deleted,
			expectedError: errors.New("更新练习状态 发布-> 未发布 或 未发布-> 删除 失败"),
		},
		//{
		//	name:          "异常23 强制更新练习提交失败 pQuery4 Released",
		//	pid:           uid.Int64,
		//	status:        PracticeStatus.Deleted,
		//	expectedError: errors.New("重置学生练习提交记录信息失败"),
		//},
		//{
		//	name:          "异常29 强制触发删除考卷失败 delete Released",
		//	pid:           uid.Int64,
		//	status:        PracticeStatus.Deleted,
		//	expectedError: errors.New("级联删除对应考试场次与练习考卷、学生答卷数据失败"),
		//},
		{
			name:          "异常16 触发清除批改配置错误 mark1 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Deleted,
			expectedError: nil,
		},
		{
			name:          "异常17 要更换的练习状态非法 PendingRelease",
			pid:           uid.Int64,
			status:        "08",
			expectedError: errors.New("非法,请传入合法的练习状态"),
		},
		{
			name:          "异常19 强制触发回滚 rollback PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常20 强制触发回滚 commit PendingRelease",
			pid:           uid.Int64,
			status:        PracticeStatus.Released,
			expectedError: nil,
		},
		{
			name:          "异常25 强制触发更新练习信息失败 pQuery5 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Disabled,
			expectedError: errors.New("更新练习状态 发布-> 作废失败"),
		},
		{
			name:          "异常26 强制触发更新练习提交信息失败 pQuery6 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Disabled,
			expectedError: errors.New("重置学生练习提交记录信息失败"),
		},
		//{
		//	name:          "异常27 强制触发更新练习提交信息失败 delete Released",
		//	pid:           uid.Int64,
		//	status:        PracticeStatus.Disabled,
		//	expectedError: errors.New("级联删除对应考试场次与练习考卷、学生答卷数据失败"),
		//},
		{
			name:          "异常28 触发清除批改配置错误 mark2 Released",
			pid:           uid.Int64,
			status:        PracticeStatus.Disabled,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()
			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "pg")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "empty")
			} else if containsString(tt.name, "异常11") {
				ctx = context.WithValue(ctx, "force-error", "pQuery1")
			} else if containsString(tt.name, "异常12") {
				ctx = context.WithValue(ctx, "force-error", "mark")
			} else if containsString(tt.name, "异常21") {
				ctx = context.WithValue(ctx, "force-error", "pQuery2")
			} else if containsString(tt.name, "异常22") {
				ctx = context.WithValue(ctx, "force-error", "pQuery3")
			} else if containsString(tt.name, "异常29") {
				ctx = context.WithValue(ctx, "force-error", "delete")
			} else if containsString(tt.name, "异常16") {
				ctx = context.WithValue(ctx, "force-error", "mark1")
			} else if containsString(tt.name, "异常25") {
				ctx = context.WithValue(ctx, "force-error", "pQuery5")
			} else if containsString(tt.name, "异常26") {
				ctx = context.WithValue(ctx, "force-error", "pQuery6")
			} else if containsString(tt.name, "异常27") {
				ctx = context.WithValue(ctx, "force-error", "delete")
			} else if containsString(tt.name, "异常28") {
				ctx = context.WithValue(ctx, "force-error", "mark2")
			} else if containsString(tt.name, "异常19") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if containsString(tt.name, "异常20") {
				ctx = context.WithValue(ctx, "force-error", "commit")
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

			initQuestion(t, conn)

			initPaper(t, conn)
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

			_, pg, _, err := examPaperService.LoadPaperTemplateById(ctx, uid.Int64, true)
			if len(pg) == 0 {
				err = fmt.Errorf("invalid paper question group")
				t.Errorf("查询此时的试卷模版视图失败:%v", err)
			}
			var examPaperID1 *int64
			examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid.Int64, uid.Int64, false)
			if err != nil {
				t.Errorf("生成考卷逻辑出错：%v", err)
			}
			if examPaperID1 == nil {
				t.Error("生成考卷逻辑出错,返回考卷ID为空")
			}

			if !containsString(tt.name, "异常10") {
				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}
			}

			// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除
			// 这里也要生成对应的practice_submission记录的
			if !containsString(tt.name, "正常3") && !containsString(tt.name, "异常22") && !containsString(tt.name, "异常23") && !containsString(tt.name, "异常29") && !containsString(tt.name, "异常16") {
				s = `INSERT INTO assessuser.t_practice_submissions (id,	practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6,$7
			),($8,$9,$10,$11,$12,$13,$14)`
				_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "00", 10087, uid, 2, *examPaperID1, uid, 1, "00")
				if err != nil {
					t.Error(err)
				}
				// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
			}

			err = tx1.Commit(context.Background())
			if err != nil {
				t.Error(err)
			}

			err = OperatePracticeStatus(ctx, tt.pid, tt.status, uid.Int64)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("OperatePracticeStatus() 期望没有错误，但返回错误: %v", err)
					return
				}
				if !containsString(tt.name, "异常19") && !containsString(tt.name, "异常20") && !containsString(tt.name, "异常28") {

					// 这里没错的话，就需要去查询一下当前的数据了
					if tt.status == PracticeStatus.Released {
						// 这里应该成功生成了考卷 然后就查询一下是否真的存在 但是此时可能会算上原本的数量的
						s = `SELECT COUNT(*) FROM t_exam_paper`
						pCount := 0
						err = conn.QueryRow(ctx, s).Scan(&pCount)
						if err != nil {
							t.Error(err)
						}
						if containsString(tt.name, "异常4") {
							if pCount != 0 {
								t.Errorf("这里rollback之后生成的考卷数量不为0：实际为：%v", pCount)
							}
						} else {
							if pCount != 1 {
								t.Errorf("生成的考卷数量不为1：实际为：%v", pCount)
							}
						}

					} else {
						// 这里需要判断一下这个
						if tt.status == PracticeStatus.Disabled {
							nowA := 0
							nowS := PracticeSubmissionStatus.Allow
							s = `SELECT attempt,status FROM t_practice_submissions WHERE practice_id = $1 LIMIT 1`
							err = conn.QueryRow(ctx, s, uid).Scan(&nowA, &nowS)
							if err != nil {
								t.Error(err)
							}
							if nowA != 1 {
								t.Errorf("当前的练习提交记录的尝试次数数量不为1：实际为：%v", nowA)
							}
							if nowS != tt.expectedSubmissionStatus {
								t.Errorf("当前的练习提交记录的状态不为预期：%v：实际为：%v", tt.expectedSubmissionStatus, nowS)
							}
						} else {

						}

					}
				}
			}

			t.Cleanup(func() {
				// 每一个测试用例之间都需要清除掉这个考卷题目信息
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

// 这里测试批量去操作练习状态
func TestOperatePracticeStatusV2(t *testing.T) {
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

	tx, _ := conn.Begin(ctx)

	// 测试数据准备
	// 6个练习 每种状态的练习各占2个
	// 给已发布的2个练习各自创建考卷、创建练习提交记录、创建学生答卷
	// 剩下的两种状态 不创建考卷与练习提交 记录

	var practiceName1, practiceName2, practiceName3, practiceName4, practiceName5, practiceName6, practiceName7 string

	practiceName1 = "单元测试待发布练习"
	practiceName2 = "单元测试发布练习"
	practiceName3 = "单元测试删除练习"
	practiceName4 = "单元测试待发布练习1"
	practiceName5 = "单元测试发布练习1"
	practiceName6 = "单元测试删除练习1"
	practiceName7 = "单元测试作废练习1"

	// 先创建数据，不管到底是不是这个测试用例需要的 三种练习状态 然后还有状态不一的练习
	// 删除 exam_paper_id 字段后，每组数据保留 8 个字段，调整占位符编号
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
VALUES 
($1, $2, $3, $4, $5, $6, $7, $8),
($9, $10, $11, $12, $13, $14, $15, $16),
($17, $18, $19, $20, $21, $22, $23, $24),
($25, $26, $27, $28, $29, $30, $31, $32),
($33, $34, $35, $36, $37, $38, $39, $40),
($41, $42, $43, $44, $45, $46, $47, $48),
($49 ,$50, $51, $52, $53, $54, $55, $56)`

	_, err = tx.Exec(ctx, s,
		// 第 1 组数据
		10086, practiceName1, "00", uid, 10086, "00", 10086, PracticeStatus.PendingRelease,
		// 第 2 组数据
		10087, practiceName2, "00", uid, 10087, "00", 10086, PracticeStatus.Released,
		// 第 3 组数据
		10088, practiceName3, "00", uid, 10088, "00", 10086, PracticeStatus.Deleted,
		// 第 4 组数据
		10089, practiceName4, "00", uid, 10086, "00", 10086, PracticeStatus.PendingRelease,
		// 第 5 组数据
		10090, practiceName5, "00", uid, 10087, "00", 10086, PracticeStatus.Released,
		// 第 6 组数据
		10091, practiceName6, "00", uid, 10088, "00", 10086, PracticeStatus.Deleted,
		// 第 7 组数据
		10092, practiceName7, "00", uid, 10089, "00", 10089, PracticeStatus.Disabled,
	)
	if err != nil {
		t.Errorf("插入练习失败：%v", err)
	}

	s = `SELECT COUNT(*) FROM t_practice`
	tCount := 0
	err = tx.QueryRow(ctx, s).Scan(&tCount)
	if err != nil {
		t.Errorf("无法查询此时练习数量:%v", err)
	}
	if tCount != 7 {
		t.Errorf("此时练习数量不为7，实际为：%v", tCount)
	}

	// 这里也随便插入已发布练习的几个学生
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
	_, err = tx.Exec(ctx, s,
		1, 10086, uid, "00",
		2, 10086, uid, "00",
		1, 10090, uid, "00",
		2, 10090, uid, "00")
	if err != nil {
		t.Fatal(err)
	}

	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
	_, err = tx.Exec(ctx, s,
		1, 10088, uid, "00",
		2, 10088, uid, "00",
		1, 10091, uid, "00",
		2, 10091, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = tx.Exec(ctx, s,
		1, 10087, uid, "00",
		2, 10087, uid, "00",
	)
	if err != nil {
		t.Fatal(err)
	}

	// 这里先查询一下
	count := 0
	s = `SELECT COUNT(*) FROM t_practice WHERE status = $1`
	err = tx.QueryRow(ctx, s, PracticeStatus.Released).Scan(&count)
	if err != nil {
		t.Errorf("无法查询到当前已经插入的数据：%v", err)
	}
	if count != 2 {
		t.Errorf("查询已发布练习失败:实际上：%v", count)
	}
	pid2ExamPaperID := make(map[int64]int64)
	pid2ExamPaperID[10087] = 10087
	pid2ExamPaperID[10090] = 10090
	// 这里构建一个map，考卷与练习之间映射
	// 这里还需要创建考卷 但是此时发布的两次练习，使用的都是同一个试卷，但是应该生成两个考卷
	var examPaperID1 *int64
	examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Errorf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Error("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
	}

	// 给每一个练习都创建练习提交记录 然后对应的插入对应的练习提交记录
	for practiceID, _ := range pid2ExamPaperID {
		var id int64
		PracticeSubmissionsIDs := []int64{}

		// 这里给
		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 1, *examPaperID1, uid, 1, "08").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}

		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 1, *examPaperID1, uid, 2, "00").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 2, *examPaperID1, uid, 1, "08").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

		// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
		s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
		err = tx.QueryRow(ctx, s,
			practiceID, 2, *examPaperID1, uid, 2, "00").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}

		PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)
		// 那这里估计要构建学生答卷 然后是那种有答卷的才对
		// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
		now := time.Now().UnixMilli()
		r := rand.New(rand.NewSource(now))
		// 获取考题
		_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx, *examPaperID1, true, true, true)
		if err != nil {
			t.Fatalf("查询考卷题目错误：%v", err)
		}
		var Answers []*cmn.TStudentAnswers
		for _, pid := range PracticeSubmissionsIDs {
			order := 1
			for _, g := range pg {
				pqList := pq[g.ID.Int64]
				r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
				for _, q := range pqList {
					answer := &cmn.TStudentAnswers{
						QuestionID:           q.ID,
						GroupID:              g.ID,
						Order:                null.IntFrom(int64(order)),
						PracticeSubmissionID: null.IntFrom(pid),
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
					"02", a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
					uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
				)
			}
			insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
			t.Logf("打印一下，能成功执行到插入学生作答前")
			_, err = tx.Exec(ctx, insertQuery, args...)
			if err != nil {
				t.Fatalf("插入学生答卷失败：%v", err)
			}

			// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis
			s = `UPDATE assessuser.t_student_answers SET answer = $1, answer_score = $2`
			answer := JSONText(`"hello"`)
			answerScore := 2.5
			_, err = tx.Exec(ctx, s, answer, answerScore)
			if err != nil {
				t.Fatal(err)
			}
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Errorf("事务提交失败：%v", err)
	}

	// 并且此时要考虑练习状态如果不一样怎么办？ 也需要处理
	// 这里我要处理的搜索条件是：考虑上之前创建好的数据
	// 要分为批量与单个的用例
	// 这里创建对应的练习记录 并且需要查看是否正常的更改完成 要实现多个实现 要根据每一个用例的不同情况，去动态创建对应的练习状态
	tests := []struct {
		name           string
		status         string
		subMissionsNum int // 表示当前测试用例 满足成功设置非法的练习提交记录数量
		pNum           int // 表示当前测试用例所有为当前用例预期练习数量
		epNum          int // 表示当前测试用例正确的考卷数量
		pids           []int64
		expectedError  error
	}{
		{
			name:          "正常1 单一 将待发布练习调整为发布状态 PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086},
			pNum:          3,
			expectedError: nil,
		},
		{
			name:           "正常2 单一 将已经发布练习调整为作废状态 Released",
			status:         PracticeStatus.Disabled,
			pids:           []int64{10087},
			subMissionsNum: 4,
			pNum:           2,
			expectedError:  nil,
		},
		{
			// 不涉及到任何的考卷信息增删的 因为我这里根本都没有给这两种练习创建记录
			name:           "正常3 单一 将发布练习调整为删除状态 Released",
			status:         PracticeStatus.Deleted,
			pids:           []int64{10087},
			subMissionsNum: 0, // 这里的0 是因为如果是删除的话，不可能有学生作答了
			pNum:           3,
			expectedError:  nil,
		},
		{
			name:          "正常4 批量 2个 将待发布练习调整为发布状态 PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086, 10089},
			pNum:          4,
			expectedError: nil,
		},
		{
			name:           "正常5 批量 2个 将已经发布练习调整为作废状态 Released",
			status:         PracticeStatus.Disabled,
			pids:           []int64{10087, 10090},
			subMissionsNum: 8,
			pNum:           3,
			expectedError:  nil,
		},
		{
			name:           "正常6 批量 2个 将发布练习调整为删除状态 Released",
			status:         PracticeStatus.Deleted,
			pids:           []int64{10087, 10090},
			subMissionsNum: 0,
			pNum:           4,
			expectedError:  nil,
		},
		{
			name:          "异常1 触发LoadPracticeByIDs参数检测错误",
			status:        PracticeStatus.Deleted,
			pids:          []int64{},
			expectedError: errors.New("非法practiceIDs"),
		},
		{
			name:          "异常9 触发事务开启错误 beginTx  PendingRelease",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10086, 10089},
			expectedError: errors.New("beginTx called failed"),
		},
		{
			name:          "异常3 触发要批量操作的练习状态不一致 Released",
			status:        PracticeStatus.PendingRelease,
			pids:          []int64{10086, 10087},
			expectedError: errors.New("此时要批量操作的练习状态不一，无法进行批量操作"),
		},
		{
			name:          "异常17 触发操作已作废练习 Disabled",
			status:        PracticeStatus.Released,
			pids:          []int64{10092},
			pNum:          3,
			expectedError: errors.New("不能操作已作废的练习"),
		},
		//{
		//	name:          "异常4 发布练习 触发生成考卷失败 pg PendingRelease",
		//	status:        PracticeStatus.Released,
		//	pids:          []int64{10086},
		//	expectedError: errors.New("invalid paper question group"),
		//},
		//{
		//	name:          "异常18 触发操作生成的考卷ID为空 empty PendingRelease",
		//	status:        PracticeStatus.Released,
		//	pids:          []int64{10086},
		//	pNum:          3,
		//	expectedError: errors.New("生成练习考卷返回的考卷ID为空"),
		//},
		{
			name:          "异常5 发布练习 触发生成更新练习信息错误 pQuery1 PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086, 10089},
			expectedError: errors.New("更新练习状态 未发布->发布 失败"),
		},
		{
			name:          "异常6 发布练习 触发配置批改信息 mark PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086, 10089},
			expectedError: errors.New("新增练习批改配置失败"),
		},
		{
			name:          "异常7 删除练习 触发查询练习是否有学生作答失败 pQuery2 Released",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10087, 10090},
			expectedError: errors.New("遍历查询是否有学生作答记录失败"),
		},
		{
			name:          "异常19 删除练习 触发查询练习已有学生作答 Released",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10087, 10090},
			pNum:          3,
			expectedError: errors.New("的练习已有学生参与作答，不能删除"),
		},
		{
			name:          "异常8 删除练习 触发更新练习错误 pQuery3 Released",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10087, 10090},
			expectedError: errors.New("更新练习状态 发布-> 删除 失败"),
		},
		{
			name:          "异常20 删除练习 触发查询练习已有学生作答 pQuery4 Released",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10087, 10090},
			pNum:          3,
			expectedError: errors.New("批量重置学生练习提交记录信息失"),
		},
		//{
		//	name:          "异常23 删除练习 触发删除考卷失败 delete Released",
		//	status:        PracticeStatus.Deleted,
		//	pids:          []int64{10087, 10090},
		//	pNum:          3,
		//	epNum:         3,
		//	expectedError: errors.New("级联删除对应考试场次与练习考卷、学生答卷数据失败"),
		//},
		{
			name:          "异常24 删除练习 触发清除批改配置 mark1 Released",
			status:        PracticeStatus.Deleted,
			pids:          []int64{10087, 10090},
			pNum:          3,
			expectedError: errors.New("清除批改配置失败"),
		},
		{
			name:          "异常21 作废练习  pQuery5 Released",
			status:        PracticeStatus.Disabled,
			pids:          []int64{10087, 10090},
			pNum:          3,
			expectedError: errors.New("更新练习状态 发布->作废 失败"),
		},
		{
			name:          "异常22 作废练习  pQuery6 Released",
			status:        PracticeStatus.Disabled,
			pids:          []int64{10087, 10090},
			pNum:          3,
			expectedError: errors.New("批量重置学生练习提交记录信息失败"),
		},
		//{
		//	name:          "异常109 作废 触发批量删除考卷、学生答卷错误 delete Released",
		//	status:        PracticeStatus.Disabled,
		//	pids:          []int64{10087, 10090},
		//	expectedError: errors.New("级联删除对应考试场次与练习考卷、学生答卷数据失败"),
		//},
		{
			name:          "异常10 取消发布/删除 触发批量清除批改配置失败 mark2 Released",
			status:        PracticeStatus.Disabled,
			pids:          []int64{10087, 10090},
			expectedError: errors.New("清除批改配置失败"),
		},
		{
			name:          "异常11 触发更新练习非法状态",
			status:        "100",
			pids:          []int64{10087, 10090},
			expectedError: errors.New("非法,请传入合法的练习状态"),
		},
		{
			name:          "异常12 更新删除状态练习为其余状态 Deleted",
			status:        PracticeStatus.PendingRelease,
			pids:          []int64{10088, 10091},
			expectedError: errors.New("批量查询练习记录失败，记录为空"),
		},
		{
			name:          "异常15 强制触发回滚 rollback PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086},
			pNum:          3,
			expectedError: nil,
		},
		{
			name:          "异常16 强制触发回滚 commit PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086},
			pNum:          3,
			expectedError: nil,
		},
		{
			name:          "异常29 强制触发发布练习时，查询试卷中考卷为空  PendingRelease",
			status:        PracticeStatus.Released,
			pids:          []int64{10086},
			pNum:          3,
			expectedError: errors.New("查看练习绑定的试卷中已发布的考卷ID失败"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// 构建对应上下文
			ctx = context.Background()
			if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "pg")
			} else if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "pQuery1")
			} else if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "mark")
			} else if containsString(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "pQuery2")
			} else if containsString(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "pQuery3")
			} else if containsString(tt.name, "异常109") {
				ctx = context.WithValue(ctx, "force-error", "delete")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "mark2")
			} else if containsString(tt.name, "异常13") {
				ctx = context.WithValue(ctx, "force-error", "empty")
			} else if containsString(tt.name, "异常15") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if containsString(tt.name, "异常16") {
				ctx = context.WithValue(ctx, "force-error", "commit")
			} else if containsString(tt.name, "异常18") {
				ctx = context.WithValue(ctx, "force-error", "empty")
			} else if containsString(tt.name, "异常20") {
				ctx = context.WithValue(ctx, "force-error", "pQuery4")
			} else if containsString(tt.name, "异常21") {
				ctx = context.WithValue(ctx, "force-error", "pQuery5")
			} else if containsString(tt.name, "异常22") {
				ctx = context.WithValue(ctx, "force-error", "pQuery6")
			} else if containsString(tt.name, "异常23") {
				ctx = context.WithValue(ctx, "force-error", "delete")
			} else if containsString(tt.name, "异常24") {
				ctx = context.WithValue(ctx, "force-error", "mark1")
			} else {
				ctx = context.Background()
			}

			// 这里如果是单个的话，那就需要将另外的进行删除
			if containsString(tt.name, "单一") {
				// 如果包含单一的话，那就拿出这个id，然后删除他那加了3的
				id := tt.pids[0]
				needDid := id + 3

				s = `DELETE FROM t_practice_student WHERE practice_id = $1`
				_, err = conn.Exec(ctx, s, needDid)
				if err != nil {
					t.Errorf("删除对应练习学生失败：%v", err)
				}

				s = `DELETE FROM t_practice WHERE id = $1`
				_, err = conn.Exec(ctx, s, needDid)
				if err != nil {
					t.Errorf("删除对应练习失败：%v", err)
				}
			}

			if containsString(tt.name, "正常6") || containsString(tt.name, "正常3") || containsString(tt.name, "异常8") || containsString(tt.name, "异常20") || containsString(tt.name, "异常23") || containsString(tt.name, "异常24") {
				s = `DELETE FROM t_practice_submissions`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					t.Errorf("删除对应练习学生提交记录失败：%v", err)
				}
			}

			if containsString(tt.name, "异常29") {
				// 这里应该删掉这个试卷的考卷ID
				s = `UPDATE t_paper SET status = $1 WHERE id = $2`
				_, err = conn.Exec(ctx, s, examPaperService.PaperStatus.Disabled, uid.Int64)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}
			}

			err = OperatePracticeStatusV2(ctx, tt.pids, tt.status, uid.Int64)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("%v报错与预期%v不符合", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("OperatePracticeStatus() 期望没有错误，但返回错误: %v", err)
					return
				}
				if !containsString(tt.name, "异常15") && !containsString(tt.name, "异常16") {

					// 如果没错的话，那就需要判断目前更换的状态是什么了，然后查看此时的是否成功
					if tt.status == PracticeStatus.Released {
						var count int
						// 然后就是查询一下这个练习的状态，是否成功改变
						s = `SELECT COUNT(*) FROM t_practice WHERE status = $1`
						err = conn.QueryRow(ctx, s, PracticeStatus.Released).Scan(&count)
						if err != nil {
							t.Errorf("执行查询失败：%v", err)
						}
						if count != tt.pNum {
							t.Errorf("返回的成功发布练习返回的数量不为%v：实际为：%v", tt.pNum, count)
						}
						// 发布状态，那就要看此时是否有生成对应的数据了
						if containsString(tt.name, "单一") {
							// 然后查一下这个考卷拥有的题目，看看是否符合要求
							ep := `SELECT id,name,exam_session_id,practice_id,creator ,total_score,question_count,group_count,groups_data FROM v_exam_paper`
							examPaper := cmn.TVExamPaper{}
							err := conn.QueryRow(ctx, ep).Scan(&examPaper.ID, &examPaper.Name, &examPaper.ExamSessionID, &examPaper.PracticeID, &examPaper.Creator, &examPaper.TotalScore,
								&examPaper.QuestionCount, &examPaper.GroupCount, &examPaper.GroupsData)
							if err != nil {
								t.Fatal(err)
							}
							if examPaper.Name.String != "考卷单元测试试卷名" {
								t.Errorf("生成的考卷名称错误 预期：考卷单元测试试卷名，实际：%v", examPaper.Name.String)
							}
							if examPaper.Creator.Int64 != uid.Int64 {
								t.Errorf("生成的考卷创建者错误 预期：%v，实际：%v", uid, examPaper.Creator)
							}
							var groupData []examPaperService.ExamGroup
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
							examQuestions := make(map[int64][]*examPaperService.ExamQuestion)
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
									examQuestions[v.ID.Int64] = make([]*examPaperService.ExamQuestion, 0)
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

						} else {
							// 这里就是批量了，要是生成的考卷给判别
							for _, _ = range tt.pids {
								// 然后查一下这个考卷拥有的题目，看看是否符合要求
								ep := `SELECT id,name,exam_session_id,practice_id,creator ,total_score,question_count,group_count,groups_data FROM v_exam_paper`
								examPaper := cmn.TVExamPaper{}
								err := conn.QueryRow(ctx, ep).Scan(&examPaper.ID, &examPaper.Name, &examPaper.ExamSessionID, &examPaper.PracticeID, &examPaper.Creator, &examPaper.TotalScore,
									&examPaper.QuestionCount, &examPaper.GroupCount, &examPaper.GroupsData)
								if err != nil {
									t.Fatal(err)
								}
								if examPaper.Name.String != "考卷单元测试试卷名" {
									t.Errorf("生成的考卷名称错误 预期：考卷单元测试试卷名，实际：%v", examPaper.Name.String)
								}
								if examPaper.Creator.Int64 != uid.Int64 {
									t.Errorf("生成的考卷创建者错误 预期：%v，实际：%v", uid, examPaper.Creator)
								}
								var groupData []examPaperService.ExamGroup
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
								examQuestions := make(map[int64][]*examPaperService.ExamQuestion)
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
										examQuestions[v.ID.Int64] = make([]*examPaperService.ExamQuestion, 0)
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
							}
						}
					} else if tt.status == PracticeStatus.Disabled {
						var count int
						s = `SELECT COUNT(*) FROM t_practice_submissions WHERE status = $1`
						err = conn.QueryRow(ctx, s, PracticeSubmissionStatus.Disabled).Scan(&count)
						if err != nil {
							t.Errorf("查询练习提交记录失败：%v", err)
						}
						if count != tt.subMissionsNum {
							t.Errorf("返回的符合要求的练习记录数量不为%v 实际返回：%v", tt.subMissionsNum, count)
						}
						s = `SELECT COUNT(*) FROM t_practice WHERE status = $1`
						err = conn.QueryRow(ctx, s, tt.status).Scan(&count)
						if err != nil {
							t.Errorf("查询练习记录失败：%v", err)
						}
						if count != tt.pNum {
							t.Errorf("返回的符合要求的练习记录数量不为%v 实际返回：%v", tt.pNum, count)
						}
					} else {
						// 这里是取消发布与设置为待发布的 需要删除考卷的，那就是需要统计现在的考卷与学生答卷的
						// 这里是删除或者是待发布状态
						// 这里要对生成的考卷跟练习提交记录进行状态

						var count int
						s = `SELECT COUNT(*) FROM t_practice_submissions WHERE status = $1`
						err = conn.QueryRow(ctx, s, PracticeSubmissionStatus.Deleted).Scan(&count)
						if err != nil {
							t.Errorf("查询练习提交记录失败：%v", err)
						}
						if count != tt.subMissionsNum {
							t.Errorf("返回的符合要求的练习记录数量不为%v 实际返回：%v", tt.subMissionsNum, count)
						}
						s = `SELECT COUNT(*) FROM t_practice WHERE status = $1`
						err = conn.QueryRow(ctx, s, tt.status).Scan(&count)
						if err != nil {
							t.Errorf("查询练习提交记录失败：%v", err)
						}
						if count != tt.pNum {
							t.Errorf("返回的符合要求的练习记录数量不为%v 实际返回：%v", tt.pNum, count)
						}

					}
				}
			}
			t.Cleanup(func() {

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

				tx, _ := conn.Begin(ctx)

				initQuestion(t, conn)

				initPaper(t, conn)
				var practiceName1, practiceName2, practiceName3, practiceName4, practiceName5, practiceName6, practiceName7 string

				practiceName1 = "单元测试待发布练习"
				practiceName2 = "单元测试发布练习"
				practiceName3 = "单元测试删除练习"
				practiceName4 = "单元测试待发布练习1"
				practiceName5 = "单元测试发布练习1"
				practiceName6 = "单元测试删除练习1"
				practiceName7 = "单元测试作废练习1"

				// 先创建数据，不管到底是不是这个测试用例需要的 三种练习状态 然后还有状态不一的练习
				// 删除 exam_paper_id 字段后，每组数据保留 8 个字段，调整占位符编号
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
VALUES 
($1, $2, $3, $4, $5, $6, $7, $8),
($9, $10, $11, $12, $13, $14, $15, $16),
($17, $18, $19, $20, $21, $22, $23, $24),
($25, $26, $27, $28, $29, $30, $31, $32),
($33, $34, $35, $36, $37, $38, $39, $40),
($41, $42, $43, $44, $45, $46, $47, $48),
($49 ,$50, $51, $52, $53, $54, $55, $56)`

				_, err = tx.Exec(ctx, s,
					// 第 1 组数据
					10086, practiceName1, "00", uid, 10086, "00", 10086, PracticeStatus.PendingRelease,
					// 第 2 组数据
					10087, practiceName2, "00", uid, 10087, "00", 10086, PracticeStatus.Released,
					// 第 3 组数据
					10088, practiceName3, "00", uid, 10088, "00", 10086, PracticeStatus.Deleted,
					// 第 4 组数据
					10089, practiceName4, "00", uid, 10086, "00", 10086, PracticeStatus.PendingRelease,
					// 第 5 组数据
					10090, practiceName5, "00", uid, 10087, "00", 10086, PracticeStatus.Released,
					// 第 6 组数据
					10091, practiceName6, "00", uid, 10088, "00", 10086, PracticeStatus.Deleted,
					// 第 7 组数据
					10092, practiceName7, "00", uid, 10089, "00", 10089, PracticeStatus.Disabled,
				)
				if err != nil {
					t.Errorf("插入练习失败：%v", err)
				}

				s = `SELECT COUNT(*) FROM t_practice`
				tCount := 0
				err = tx.QueryRow(ctx, s).Scan(&tCount)
				if err != nil {
					t.Errorf("无法查询此时练习数量:%v", err)
				}
				if tCount != 7 {
					t.Errorf("此时练习数量不为7，实际为：%v", tCount)
				}

				// 这里也随便插入已发布练习的几个学生
				s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = tx.Exec(ctx, s,
					1, 10086, uid, "00",
					2, 10086, uid, "00",
					1, 10090, uid, "00",
					2, 10090, uid, "00")
				if err != nil {
					t.Fatal(err)
				}

				s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = tx.Exec(ctx, s,
					1, 10088, uid, "00",
					2, 10088, uid, "00",
					1, 10091, uid, "00",
					2, 10091, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
				s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
				_, err = tx.Exec(ctx, s,
					1, 10087, uid, "00",
					2, 10087, uid, "00",
				)
				if err != nil {
					t.Fatal(err)
				}

				// 这里先查询一下
				count := 0
				s = `SELECT COUNT(*) FROM t_practice WHERE status = $1`
				err = tx.QueryRow(ctx, s, PracticeStatus.Released).Scan(&count)
				if err != nil {
					t.Errorf("无法查询到当前已经插入的数据：%v", err)
				}
				if count != 2 {
					t.Errorf("查询已发布练习失败:实际上：%v", count)
				}
				pid2ExamPaperID := make(map[int64]int64)
				pid2ExamPaperID[10087] = 10087
				pid2ExamPaperID[10090] = 10090
				// 这里构建一个map，考卷与练习之间映射
				// 这里还需要创建考卷 但是此时发布的两次练习，使用的都是同一个试卷，但是应该生成两个考卷
				var examPaperID1 *int64
				examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx, uid.Int64, uid.Int64, false)
				if err != nil {
					t.Errorf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Error("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}

				// 给每一个练习都创建练习提交记录 然后对应的插入对应的练习提交记录
				for practiceID, _ := range pid2ExamPaperID {
					var id int64
					PracticeSubmissionsIDs := []int64{}

					// 这里给
					// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
					s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
					err = tx.QueryRow(ctx, s,
						practiceID, 1, *examPaperID1, uid, 1, "08").Scan(&id)
					if err != nil {
						t.Fatal(err)
					}

					// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
					s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
					err = tx.QueryRow(ctx, s,
						practiceID, 1, *examPaperID1, uid, 2, "00").Scan(&id)
					if err != nil {
						t.Fatal(err)
					}
					PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

					// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
					s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
					err = tx.QueryRow(ctx, s,
						practiceID, 2, *examPaperID1, uid, 1, "08").Scan(&id)
					if err != nil {
						t.Fatal(err)
					}
					PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)

					// 这里要对应创建学生提交记录 给已经发布的练习创建练习提交记录 这里需要分为两种情况：已经提交的，还有已经作答了没有提交的 每一种
					s = `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6
			) returning id`
					err = tx.QueryRow(ctx, s,
						practiceID, 2, *examPaperID1, uid, 2, "00").Scan(&id)
					if err != nil {
						t.Fatal(err)
					}

					PracticeSubmissionsIDs = append(PracticeSubmissionsIDs, id)
					// 那这里估计要构建学生答卷 然后是那种有答卷的才对
					// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
					now := time.Now().UnixMilli()
					r := rand.New(rand.NewSource(now))
					// 获取考题
					_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx, *examPaperID1, true, true, true)
					if err != nil {
						t.Fatalf("查询考卷题目错误：%v", err)
					}
					var Answers []*cmn.TStudentAnswers
					for _, pid := range PracticeSubmissionsIDs {
						order := 1
						for _, g := range pg {
							pqList := pq[g.ID.Int64]
							r.Shuffle(len(pqList), func(i, j int) { pqList[i], pqList[j] = pqList[j], pqList[i] })
							for _, q := range pqList {
								answer := &cmn.TStudentAnswers{
									QuestionID:           q.ID,
									GroupID:              g.ID,
									Order:                null.IntFrom(int64(order)),
									PracticeSubmissionID: null.IntFrom(pid),
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
								"02", a.ExamineeID, a.PracticeSubmissionID, a.QuestionID,
								uid, now, now, a.GroupID, a.Order, a.ActualOptions, a.ActualAnswers,
							)
						}
						insertQuery := fmt.Sprintf(query, strings.Join(values, ","))
						t.Logf("打印一下，能成功执行到插入学生作答前")
						_, err = tx.Exec(ctx, insertQuery, args...)
						if err != nil {
							t.Fatalf("插入学生答卷失败：%v", err)
						}

						// 生成答卷之后，还需要对学生对应的作答、赋予分数 我这里主要是看是否真的能查到吧，具体数据是多少，数据的问题是不管我事的 然后还需要插入analysis
						s = `UPDATE assessuser.t_student_answers SET answer = $1, answer_score = $2`
						answer := JSONText(`"hello"`)
						answerScore := 2.5
						_, err = tx.Exec(ctx, s, answer, answerScore)
						if err != nil {
							t.Fatal(err)
						}
					}

				}

				err = tx.Commit(ctx)
				if err != nil {
					t.Errorf("事务提交失败：%v", err)
				}

			})
		})

		t.Cleanup(func() {
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
	examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}
	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
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
	_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
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
				// 最后再生成一下

				initQuestion(t, conn)

				initPaper(t, conn)
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
				examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid.Int64, uid.Int64, false)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}
				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
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
				_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
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

// 主要测试这个查询出来的错题是否满足
func TestEnterPracticeWrongCollection(t *testing.T) {
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
	s = `DELETE FROM t_practice_wrong_submissions`
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
	examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}
	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
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
		)`
	_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08")
	if err != nil {
		t.Fatal(err)
	}

	// 创建一个练习提交记录，然后再创建多种的错题即可,但是此时想实现学生进入过几次的

	pscount := 0
	s = `SELECT COUNT(*) FROM t_practice_submissions`
	err = tx1.QueryRow(ctx, s).Scan(&pscount)
	if err != nil {
		t.Fatal(err)
	}
	if pscount != 1 {
		t.Errorf("此时生成的练习提交记录失败，数量不为3")
	}
	// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
	// TODO 下面是去插入一个本身就已经创建好的已作答但是未提交的学生记录的
	// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
	submissionIDs := []int64{10086}
	now := time.Now().UnixMilli()
	// 获取考题
	_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	// 这里不实现乱系就好
	Answers := []*cmn.TStudentAnswers{}
	for _, id := range submissionIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID:           q.ID,
					GroupID:              g.ID,
					Order:                null.IntFrom(int64(order)),
					PracticeSubmissionID: null.IntFrom(id),
				}
				if q.Type.String == "00" || q.Type.String == "02" || q.Type.String == "04" {
					answer.ActualAnswers = q.Answers
					answer.ActualOptions = q.Options
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

	tests := []struct {
		name          string
		pid           int64
		uid           int64
		StudentStatus string
		expectedError error
	}{
		{
			name:          "正常1 从来没有进入过错题集的 能查询到错题集试卷与题组",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: nil,
		},
		{
			name:          "正常2 进入过错题集但是没有提交 能查询到错题集试卷与题组",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: nil,
		},
		{
			name:          "正常3 进入过错题集已经提交 然后此时也还是有错题的 能查询到错题集试卷与题组",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.Submitted,
			expectedError: nil,
		},
		{
			name:          "强制异常1 强制触发查询练习提交状态失败 cQuery1",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("查询学生练习记录信息失败"),
		},
		{
			name:          "异常2 当前练习提交记录为空，导致查询出来的LastestSubmittedID==0 ",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("请求学生错题集失败 , 此时学生没有提交过练习，无法进入错题练习"),
		},
		{
			name:          "异常90 当前练习最新提交记录还没有成功提交，导致查询出来的LatestUnsubmittedID > 0 ",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("请求学生错题集失败 , 此时学生已经进入一次练习，但是未提交，无法进入错题练习"),
		},
		{
			name:          "异常3 当前练习最新提交记录还没有被老师批改，导致查询出来的PendingMarkID > 0 ",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("请求学生错题集失败 , 此时学生已经进入一次练习，已提交但是待教师批改出成绩，无法进入错题练习"),
		},
		{
			name:          "异常4 当前练习最新提交记录中全对 WrongCount == 0 因此没有错题可练习",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("请求学生错题集失败 , 学生最新练习提交记录中无错题，无需进入错题练习"),
		},
		{
			name:          "异常5 强制触发查询错题练习提交状态失败 cQuery2",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.Submitted,
			expectedError: errors.New("查询学生错题练习记录信息失败"),
		},
		{
			name:          "异常6 强制触发查询错题练习提交状态失败 cQuery3",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.Submitted,
			expectedError: errors.New("创建错题练习提交记录失败"),
		},
		{
			name:          "异常7 强制触发查询错题练习提交状态失败 cQuery4",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("创建错题练习提交记录失败"),
		},
		{
			name:          "异常8 强制触发查询错题视图失败 ecQuery1",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.NeverAnswer,
			expectedError: errors.New("查询学生错题集信息失败"),
		},
		{
			name:          "异常9 强制触发查询学生答卷失败 cQuery5",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("query statement failed"),
		},
		{
			name:          "异常10 强制触发查询错题视图后的扫描学生作答失败 scan",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("scan student answer failed"),
		},
		{
			name:          "异常11 强制触发查询错题视图后解析答案与解析失败 emptyAnswer",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("query empty student answer, please checkout generate exam paper or student answer"),
		},
		{
			name:          "异常12 强制触发查询错题视图后构建的题目映射结构失效 exist",
			pid:           uid.Int64,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New(" not found in exam paper"),
		},
		{
			name:          "异常13 强制触发参数检测 pid缺失",
			pid:           -1,
			uid:           1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("invalid practiceID | uid param"),
		},
		{
			name:          "异常14 强制触发参数检测 uid缺失",
			pid:           uid.Int64,
			uid:           -1,
			StudentStatus: StudentSubmissionStatus.UnSubmitted,
			expectedError: errors.New("invalid practiceID | uid param"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里去跑测试，然后需要根据不一样的状态，去准备不一样的数据
			ctx = context.Background()
			if containsString(tt.name, "强制异常1") {
				ctx = context.WithValue(ctx, "force-error", "cQuery1")
			} else if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "cQuery2")
			} else if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "cQuery3")
			} else if containsString(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "cQuery4")
			} else if containsString(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "ecQuery1")
			} else if containsString(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "cQuery5")
			} else if containsString(tt.name, "异常10") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if containsString(tt.name, "异常11") {
				ctx = context.WithValue(ctx, "force-error", "emptyAnswer")
			} else if containsString(tt.name, "异常12") {
				ctx = context.WithValue(ctx, "force-error", "exist")
			} else {
				ctx = context.Background()
			}

			switch tt.StudentStatus {
			// 如果是有一个待提交的话
			case StudentSubmissionStatus.UnSubmitted:
				{
					// 要准备一个还能进行的作答
					s = `INSERT  INTO t_practice_wrong_submissions (practice_submission_id,attempt,creator,create_time,update_time,status)VALUES(
						$1,$2,$3,$4,$5,$6
					)`
					_, err := conn.Exec(ctx, s, uid.Int64, 1, uid, now, now, "00")
					if err != nil {
						t.Fatal("插入一条错题练习记录失败")
					}
				}
			case StudentSubmissionStatus.NeverAnswer:
				{
					t.Logf("此时没有作答过错题，也就不需要插入这个行")
					if containsString(tt.name, "异常2") {
						// 删除掉这个作答记录
						s = `DELETE FROM t_practice_submissions WHERE id = $1`
						_, err := conn.Exec(ctx, s, uid.Int64)
						if err != nil {
							t.Fatal("删除掉一条已成功出了分的练习记录失败")
						}
					} else if containsString(tt.name, "异常90") {

						s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6,$7
			)`
						_, err = conn.Exec(ctx, s, 10087, uid, 1, *examPaperID1, uid, 2, "00")
						if err != nil {
							t.Fatal(err)
						}
					} else if containsString(tt.name, "异常3") {
						// 更改这个最新的
						s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6,$7
			)`
						_, err = conn.Exec(ctx, s, 10087, uid, 1, *examPaperID1, uid, 2, "06")
						if err != nil {
							t.Fatal(err)
						}
					} else if containsString(tt.name, "异常4") {
						// 更改这个最新的
						// 按照t_question表中每个题目的score字段值，定义对应顺序的满分
						fullScoreData := []struct {
							order      int
							questionID int // 明确关联t_question的id，双重验证
							answer     JSONText
							fullScore  int // 与t_question.score类型一致（整数）
						}{
							{order: 1, questionID: 1, answer: JSONText(`"A,D"`), fullScore: 2},
							{order: 2, questionID: 2, answer: JSONText(`"A,D"`), fullScore: 2},
							{order: 3, questionID: 6, answer: JSONText(`"A"`), fullScore: 2},
							{order: 4, questionID: 3, answer: JSONText(`"A,B,C,D"`), fullScore: 5},
							{order: 5, questionID: 4, answer: JSONText(`"A,B"`), fullScore: 5},
							{order: 6, questionID: 5, answer: JSONText(`"A,B,C"`), fullScore: 5},
							{order: 7, questionID: 7, answer: JSONText(`"A"`), fullScore: 2},
							{order: 8, questionID: 8, answer: JSONText(`"A"`), fullScore: 2},
							{order: 9, questionID: 9, answer: JSONText(`"螺旋模型"`), fullScore: 5},
							{order: 10, questionID: 10, answer: JSONText(`"需求规格说明书"`), fullScore: 5},
							{order: 11, questionID: 11, answer: JSONText(`"封装,继承,多态"`), fullScore: 3},
							{order: 12, questionID: 12, answer: JSONText(`"通过网络下载、移动存储设备、电子邮件等方式传播"`), fullScore: 5},
							{order: 13, questionID: 13, answer: JSONText(`"进程管理、内存管理、文件管理、设备管理、作业管理"`), fullScore: 5},
						}

						// 执行更新操作，确保每个题目的answer_score等于其score字段值
						for _, data := range fullScoreData {
							updateSQL := `UPDATE assessuser.t_student_answers 
                  SET answer = $1, answer_score = $2 
                  WHERE "order" = $3` // 按题目顺序定位

							_, err := conn.Exec(ctx, updateSQL, data.answer, data.fullScore, data.order)
							if err != nil {
								t.Fatalf("更新顺序为%d的题目失败: %v", data.order, err)
							}
						}
					} else {
					}

				}
			case StudentSubmissionStatus.Submitted:
				{
					s = `INSERT  INTO t_practice_wrong_submissions (practice_submission_id,attempt,creator,create_time,update_time,status)VALUES(
						$1,$2,$3,$4,$5,$6
					)`
					_, err := conn.Exec(ctx, s, uid.Int64, 1, uid, now, now, "04")
					if err != nil {
						t.Fatal("插入一条错题练习记录失败")
					}
				}
			}

			tx1, _ := conn.Begin(ctx)
			defer tx1.Rollback(ctx)

			i, pg, pq, err := EnterPracticeWrongCollection(ctx, tx1, tt.pid, tt.uid)
			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("返回的错误不符合预期：%v，实际为：%v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if i == nil {
					t.Errorf("返回的试卷信息指针为空：异常")
				}

				if len(pg) != 3 || i.GroupCount != 3 {
					t.Errorf("此时错题集的题组实际个数为%v,期望为%v", i.GroupCount, 3)
				}
				if i.QuestionCount != 8 {
					t.Errorf("此时错题集的题目题目实际个数为%v,期望为%v", i.QuestionCount, 8)
				}
				if !containsString(i.PaperName, "错题") {
					t.Errorf("此时返回的错题试卷名不包含错题")
				}
				qCount := 0
				for groupID, _ := range pg {
					qList := pq[groupID]
					for _, q := range qList {
						qCount++
						if !containsString(tt.name, "正常4") {
							if tt.StudentStatus == StudentSubmissionStatus.UnSubmitted {
								// 如果是待提交的话，那此时的学生作答不应该为空
								expected := []byte(`"hello"`)
								if !bytes.Equal(q.StudentAnswer, expected) {
									t.Errorf("此时没有正确返回学生作答内容")
								}
							}
						}
					}
				}
				if qCount != 8 {
					t.Errorf("此时错题集的题目题目实际个数为%v,期望为%v", qCount, 8)
				}

			}

			t.Cleanup(func() {

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
				s = `DELETE FROM t_practice_wrong_submissions`
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
				tx2, _ := conn.Begin(ctx)

				var practiceName string
				practiceName = "单元测试练习名"
				// 先创建这个数据，最后测试完毕再删掉 用于更新用的
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
				VALUES ($1, $2, $3, $4, $5, $6, $7,$8)`
				_, err = tx2.Exec(ctx, s, uid, practiceName, "00", uid, 1, "00", uid, PracticeStatus.Released)
				if err != nil {
					t.Fatal(err)
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = tx2.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
				var examPaperID1 *int64
				examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx2, uid.Int64, uid.Int64, false)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx2.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}

				// 这里需要再update一下考卷ID
				s = `UPDATE t_practice SET exam_paper_id = $1 WHERE id = $2`
				_, err = tx2.Exec(ctx, s, *examPaperID1, uid)
				if err != nil {
					t.Fatal(err)
				}
				// 然后 要准备多个情况 去测试状态是否满足 这里我优先准备 两三个数据 练习提交记录，用于删除或者取消发布操作查看是否真的全部清除 一个是08 一个00 分别代表三种状态中的其中两种
				// 这里也要生成对应的practice_submission记录的
				s = `INSERT INTO assessuser.t_practice_submissions (id,practice_id,student_id,exam_paper_id,creator,attempt,status) VALUES (
				$1,$2,$3,$4,$5,$6,$7
				)`
				_, err = tx2.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08")
				if err != nil {
					t.Fatal(err)
				}

				// 创建一个练习提交记录，然后再创建多种的错题即可,但是此时想实现学生进入过几次的

				pscount := 0
				s = `SELECT COUNT(*) FROM t_practice_submissions`
				err = tx2.QueryRow(ctx, s).Scan(&pscount)
				if err != nil {
					t.Fatal(err)
				}
				if pscount != 1 {
					t.Errorf("此时生成的练习提交记录失败，数量不为3")
				}
				// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
				// TODO 下面是去插入一个本身就已经创建好的已作答但是未提交的学生记录的
				// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
				submissionIDs := []int64{10086}
				now := time.Now().UnixMilli()
				// 获取考题
				_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx2, *examPaperID1, true, true, true)
				if err != nil {
					t.Fatalf("查询考卷题目错误：%v", err)
				}
				Answers := []*cmn.TStudentAnswers{}
				for _, id := range submissionIDs {
					order := 1
					for _, g := range pg {
						pqList := pq[g.ID.Int64]
						for _, q := range pqList {
							answer := &cmn.TStudentAnswers{
								QuestionID:           q.ID,
								GroupID:              g.ID,
								Order:                null.IntFrom(int64(order)),
								PracticeSubmissionID: null.IntFrom(id),
							}
							if q.Type.String == "00" || q.Type.String == "02" || q.Type.String == "04" {
								answer.ActualAnswers = q.Answers
								answer.ActualOptions = q.Options
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
					_, err := tx2.Exec(ctx, insertQuery, args...)
					if err != nil {
						t.Fatalf("插入学生答卷失败:%v", err)
					}
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
			// 再删除练习
			s = `DELETE FROM t_practice_wrong_submissions`
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

func TestLoadErrorCollectionDetailsById(t *testing.T) {
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
	s = `DELETE FROM t_practice_wrong_submissions`
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
	examPaperID1, _, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid.Int64, uid.Int64, false)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid.Int64)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
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
		)`
	_, err = tx1.Exec(ctx, s, 10086, uid, 1, *examPaperID1, uid, 1, "08")
	if err != nil {
		t.Fatal(err)
	}

	// 创建一个练习提交记录，然后再创建多种的错题即可,但是此时想实现学生进入过几次的

	pscount := 0
	s = `SELECT COUNT(*) FROM t_practice_submissions`
	err = tx1.QueryRow(ctx, s).Scan(&pscount)
	if err != nil {
		t.Fatal(err)
	}
	if pscount != 1 {
		t.Errorf("此时生成的练习提交记录失败，数量不为3")
	}
	// 这里定义了两个这种状态，就代表已经提交跟已作答但是未提交的状态了
	// TODO 下面是去插入一个本身就已经创建好的已作答但是未提交的学生记录的
	// 第一个是练习，第二个是考试 在这个逻辑就不测试多的情况了
	submissionIDs := []int64{10086}
	now := time.Now().UnixMilli()
	// 获取考题
	_, pg, pq, err := examPaperService.LoadExamPaperDetailsById(ctx, tx1, *examPaperID1, true, true, true)
	if err != nil {
		t.Fatalf("查询考卷题目错误：%v", err)
	}
	// 这里不实现乱系就好
	Answers := []*cmn.TStudentAnswers{}
	for _, id := range submissionIDs {
		order := 1
		for _, g := range pg {
			pqList := pq[g.ID.Int64]
			for _, q := range pqList {
				answer := &cmn.TStudentAnswers{
					QuestionID:           q.ID,
					GroupID:              g.ID,
					Order:                null.IntFrom(int64(order)),
					PracticeSubmissionID: null.IntFrom(id),
				}
				if q.Type.String == "00" || q.Type.String == "02" || q.Type.String == "04" {
					answer.ActualAnswers = q.Answers
					answer.ActualOptions = q.Options
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

	tests := []struct {
		name          string
		pid           int64
		uid           int64
		withAnswers   bool
		withAnalysis  bool
		expectedError error
	}{
		{
			name:          "正常1 能正常查询到学生错题试卷 携带题目答案 携带题目解析",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   true,
			withAnalysis:  true,
			expectedError: nil,
		},
		{
			name:          "正常2 能正常查询到学生错题试卷 不携带题目答案 携带题目解析",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  true,
			expectedError: nil,
		},
		{
			name:          "正常3 能正常查询到学生错题试卷 携带题目答案 不携带题目解析",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   true,
			withAnalysis:  false,
			expectedError: nil,
		},
		{
			name:          "正常4 能正常查询到学生错题试卷 不携带题目答案 不携带题目解析",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: nil,
		},

		{
			name:          "异常3 强制查询失败 ecQuery1",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: errors.New("查询学生错题集信息失败"),
		},
		{
			name:          "异常4 强制查询题组解析失败 json",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: errors.New("unmarshal group data failed"),
		},
		{
			name:          "异常5 强制查询题组解析失败 jsonA",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: errors.New("数据错误：此时返回的错题数组为空！"),
		},
		{
			name:          "异常6 强制查询题组解析答案失败 jsonB",
			pid:           uid.Int64,
			uid:           1,
			withAnswers:   true,
			withAnalysis:  false,
			expectedError: errors.New("failed to unmarshal Answers questionId"),
		},
		{
			name:          "异常1 参数检测pid非法",
			pid:           -1,
			uid:           1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: errors.New("invalid pid or uid param"),
		},
		{
			name:          "异常2 参数检测uid非法",
			pid:           uid.Int64,
			uid:           -1,
			withAnswers:   false,
			withAnalysis:  false,
			expectedError: errors.New("invalid pid or uid param"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = context.Background()

			tx1, _ := conn.Begin(ctx)

			defer tx1.Commit(ctx)

			if containsString(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "ecQuery1")
			} else if containsString(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "jsonA")
			} else if containsString(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "jsonB")
			} else if containsString(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "json")
			} else {
				ctx = context.Background()
			}

			i, pg, pq, err := LoadErrorCollectionDetailsById(ctx, tx1, tt.pid, tt.uid, tt.withAnswers, tt.withAnalysis)

			if tt.expectedError != nil {
				if err == nil || !containsString(err.Error(), tt.expectedError.Error()) {
					t.Errorf("返回的错误不符合预期：%v，实际为：%v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("预期无错误，但返回错误：%v", err)
				}
				if i == nil {
					t.Errorf("返回的错题练习信息指针为空")
				}

				if len(pg) != 3 || i.GroupCount != 3 {
					t.Errorf("此时错题集的题组实际个数为%v,期望为%v", i.GroupCount, 3)
				}
				if i.QuestionCount != 8 {
					t.Errorf("此时错题集的题目题目实际个数为%v,期望为%v", i.QuestionCount, 8)
				}
				qCount := 0
				for _, g := range pg {
					qList := pq[g.ID.Int64]
					for _, q := range qList {
						qCount++
						// 如果是待提交的话，那此时的学生作答不应该为空
						if tt.withAnalysis {
							if q.Analysis.String != "hello" {
								t.Errorf("此时没有正确包含解析内容")
							}
						}
						if tt.withAnswers {
							if q.Answers.String() == "" {
								t.Errorf("此时查询出来的答案为空，错误")
							}
						}
					}
				}
				if qCount != 8 {
					t.Errorf("此时错题集的题目题目实际个数为%v,期望为%v", qCount, 8)
				}

			}
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
			// 再删除练习
			s = `DELETE FROM t_practice_wrong_submissions`
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

	// 这里进行准备，然后还要查看具体的这个搜索的是否为确定的错题，仅此而已
}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 这里初始化题目，到达其他的函数，我这里一样可以进行删除
func initQuestion(t *testing.T, tx *pgxpool.Pool) {
	qb := `INSERT INTO t_question_bank  (id,type,name,creator,create_time,status,domain_id) VALUES($1, $2, $3, $4, $5, $6,$7)`
	_, err := tx.Exec(ctx, qb, uid, "00", "考卷单元测试题库", uid, now, "00", 10086)
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
		DomainID:          uid,
	}
	p := `INSERT INTO t_paper
			(id,name, assembly_type, category, level, suggested_duration, tags, creator, create_time, updated_by, update_time, status,domain_id)
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
		paper.DomainID.Int64,
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
	var options []examPaperService.QuestionOption
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
