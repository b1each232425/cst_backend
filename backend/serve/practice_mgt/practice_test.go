/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-25 23:20:51
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-31 12:45:37
 */
package practice_mgt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	examPaperService "w2w.io/serve/examPaper"
)

// 按照接口定义去测试这个函数罢了 分为两类权限 就是老师跟学生就好 是
func TestPracticeH(t *testing.T) {

	if z == nil {
		cmn.ConfigureForTest()
	}
	// 要给这个上下文区分到底是学生还是教师权限

	conn := cmn.GetPgxConn()
	var uid int64
	uid = 10086
	now := time.Now().UnixMilli()

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

	// 这里直接先准备好这个试卷就好了
	initQuestion(t, conn)

	initPaper(t, conn)

	// 初始化题目跟试卷，这个肯定是要有的
	tx1, _ := conn.Begin(ctx)

	var examPaperID1 *int64
	examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
	}

	err = tx1.Commit(ctx)
	if err != nil {
		t.Errorf("提交事务失败:%v", err)
	}
	testCases := []struct {
		name     string
		method   string
		url      string
		reqBody  *cmn.ReqProto
		userId   int64
		forceErr string
		ctxKey   string
		ctxValue string
		// 预期结果
		expectSuccess       bool            // 是否期望成功
		expectedMessage     string          // 预期错误消息
		expectFailedMessage string          // 预期成功消息
		expectedData        json.RawMessage // 预期数据（可选）
		setup               func() error
		Domain              []cmn.TDomain
		create              bool //是否是创建的练习
	}{
		{
			//POST 创建/更新练习数据
			name:   "POST 教师创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "00",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Teacher)},
			},
			create: true,
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 管理员创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "00",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Admin)},
			},
			create: true,
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 超级管理员创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			create: true,
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 学生创建练习",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "please call /api/practice with  http GET method",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
		},

		{
			//POST 创建/更新练习数据
			name:   "POST 异常1 上下文触发 io.ReadAll",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			forceErr:        "io.ReadAll",
			expectSuccess:   false,
			expectedMessage: "尝试打开前端数据流失败",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 异常2 上下文触发 Close",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			forceErr:        "Close",
			expectedMessage: "",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 异常3 上下文触发 readReqProtoJson",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			forceErr:        "readReqProtoJson",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "读取前端数据结构体失败",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 异常4 上下文触发 readPJson",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			forceErr:        "readPJson",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "读取前端数据practice字段结构体失败",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 异常5 触发validatePractrice函数错误",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "invalid practice Name",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 异常6 触发UpsertPractice函数错误 query",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				},
    				"student": [
        				10086,
						10087,
        				10088
    				]
				}`),
			},
			forceErr:        "query",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "addPractice call failed",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
		},
		{
			//POST 更新练习数据
			name:   "POST 学生更新练习 不携带action字段，不清空学生名单",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice": {
						"ID":10086,
						"PaperID":10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 10
    				},
    				"student": [
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
		},
		{
			//POST 创建/更新练习数据
			name:   "POST 学生更新练习清除学生名单 携带action字段，操作清空学生名单",
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Action: "clear",
				Data: json.RawMessage(`{
					"practice":{
						"ID":10086,
						"PaperID":10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 10
    				},
					"student": [
    				]
				}`),
			},
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			expectedData:    nil,
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)
				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
		},
		{
			name: "POST 请求 - empty-buf error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practice",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{
					"practice":{
						"ID":10086,
        				"PaperID": 10086,
        				"Name": "练习模拟数据期末考试",
        				"CorrectMode": "00",
        				"Type": "02",
        				"AllowedAttempts": 5
    				}
				}`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "Call /api/practice with  empty body",
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=1&page_size=10",
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 强制触发错误缺少分页号",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page_size=10",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页号",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 强制触发错误缺少分页大小",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=10",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页大小",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 强制触发错误 pageParseInt",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=10",
			userId:          uid,
			forceErr:        "pageParseInt",
			expectSuccess:   false,
			expectedMessage: "将字符串转化为整形失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 强制触发错误 pageSizeParseInt",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=10&page_size=10",
			userId:          uid,
			forceErr:        "pageSizeParseInt",
			expectSuccess:   false,
			expectedMessage: "将字符串转化为整形失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 强制触发错误 pageSize too big",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=10&page_size=1000000",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "页数量过大，不允许访问",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 反序列失败 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=1&page_size=10",
			userId:          uid,
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "将要返回数据结构反序列化失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习列表 query",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?page=1&page_size=10",
			userId:          uid,
			forceErr:        "query",
			expectSuccess:   false,
			expectedMessage: "search practice failed",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET请求教师及其以上权限获取练习详情 触发接收错误 idParseInt",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?id=10086",
			userId:          uid,
			forceErr:        "idParseInt",
			expectSuccess:   false,
			expectedMessage: "将字符串转化为整形失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习详情 触发查询单一练习失败",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?id=100000",
			userId:          uid,
			forceErr:        "",
			expectSuccess:   false,
			expectedMessage: "非法practiceID , 无该练习记录",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习详情 触发反序列错误 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?id=10086",
			userId:          uid,
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "将返回数据结构反序列化失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 教师及其以上权限获取练习详情 触发反序列错误 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?id=10086",
			userId:          uid,
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "将返回数据结构反序列化失败",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 正常教师及其以上权限获取练习详情",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practice?id=10086",
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid)

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")

				initQuestion(t, conn)

				initPaper(t, conn)
				tx1, _ := conn.Begin(ctx)
				var examPaperID1 *int64
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}

				err = tx1.Commit(ctx)
				if err != nil {
					t.Errorf("提交事务失败：%v", err)
				}
				// 这里得更新这个东西

				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 无法操作练习为待发布状态",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=10086&status=00",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "请传入合法的练习状态",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 缺少操作练习数组",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?status=00",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "请传入要操作的练习ID",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 非法练习数组",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=abc,2bc&status=00",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "传入的练习ID中存在非法值",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 非法练习数组",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=%20%20%20&status=00",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "请传入有效的需要操作的练习ID",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 触发OperatePracticeStatusV2失败 beginTx",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=10086&status=00",
			forceErr:        "beginTx",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "beginTx called failed",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作练习状态 为发布状态",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=10086&status=02",
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "00")

				initQuestion(t, conn)

				initPaper(t, conn)
				tx1, _ := conn.Begin(ctx)
				var examPaperID1 *int64
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}

				err = tx1.Commit(ctx)
				if err != nil {
					t.Errorf("提交事务失败：%v", err)
				}
				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作发布练习状态 为删除状态",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=10086&status=04",
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "02")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "PATCH 请求 - 正常教师及其以上权限操作发布练习状态 为作废状态",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "PATCH",
			url:             "/api/practice?id=10086&status=06",
			userId:          uid,
			expectSuccess:   true,
			expectedMessage: "OK",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "02")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "DELETE 请求 - 触发default分支",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "DELETE",
			url:             "/api/practice?id=10086&status=06",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "http POST/GET/PATCH method",
			setup: func() error {
				_, err := conn.Exec(context.Background(), `
	INSERT INTO assessuser.t_practice (id,name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id,status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11)`, 10086, "练习模拟数据期末考试", "00", uid, now, now, nil, 5, "00", uid, "02")

				// 然后这里还要去创建对应的学生4个
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8),($9,$10,$11,$12),($13,$14,$15,$16)`
				_, err = conn.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00", 3, uid, uid, "00", 4, uid, uid, "00")
				return err
			},
			expectedData: json.RawMessage(`{
			"total":1,
			"practices":[
				{
					"practice":{
					"ID":10086,
					"Name":"练习模拟数据期末考试",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":5,
					"Status":"00"
					},
					"student_count":4
				}
			]
			}`,
			),
		},
		{
			name: "GET 请求 - 学生成功权限获取列表",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1&page_size=10",
			userId:          1,
			expectSuccess:   true,
			expectedMessage: "OK",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限不使用正确方法",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "DELETE",
			url:             "/api/practice",
			userId:          1,
			expectSuccess:   false,
			expectedMessage: "please call /api/practice with  http GET method",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 强制参数错误",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1",
			userId:          1,
			forceErr:        "pageParseInt",
			expectSuccess:   false,
			expectedMessage: "将页面大小转化为整形失败",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型获取列表",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=",
			userId:          1,
			expectSuccess:   false,
			expectedMessage: "缺失练习类型",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 强制参数错误",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1&page_size=10",
			userId:          1,
			forceErr:        "pageSizeParseInt",
			expectSuccess:   false,
			expectedMessage: "将分页大小转化为整形失败",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 强制参数错误 sQuery1",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1&page_size=10",
			userId:          1,
			forceErr:        "sQuery1",
			expectSuccess:   false,
			expectedMessage: "查询学生权限练习列表失败",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 强制参数错误 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1&page_size=10",
			userId:          1,
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "构建前端返回数据反序列化失败",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 缺少页号",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page_size=10",
			userId:          1,
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页号",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 缺少页大小",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1",
			userId:          1,
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页大小",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
		{
			name: "GET 请求 - 学生权限缺少练习类型 非法用户ID",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practice?type=00&page=1",
			userId:          -1,
			expectSuccess:   false,
			expectedMessage: "invalid UserID",
			setup: func() error {
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
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
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

				return err
			},
			expectedData: json.RawMessage(`{
				"practice": [
					{
						"ID": 10086,
						"Name": "单元测试练习名",
						"Type": "02",
						"AttemptCount": 1,
						"Difficulty": "02",
						"AllowedAttempts": 1,
						"QuestionCount": 13,
						"WrongCount": 8,
						"TotalScore": 32.5,
						"HighestScore": 32.5,
						"PaperTotalScore": 48,
						"PaperID": 10086,
						"LatestUnsubmittedID": 0,
						"LatestSubmittedID": 10086,
						"PendingMarkID": 0
					}
				],
				"total": 1
			}`,
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.setup != nil {
				err := tc.setup()
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

			}

			var req *http.Request
			var err error

			if tc.reqBody != nil {

				// 对于 POST 请求，准备请求体
				bodyBytes, err := json.Marshal(tc.reqBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(bodyBytes))
				if tc.name == "buf-zero" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(nil))
				} else if tc.name == "POST 请求 - empty-buf error" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer([]byte("")))
				}
			} else {
				// 对于 GET 请求，没有请求体
				req, err = http.NewRequest(tc.method, tc.url, nil)
			}
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			// 创建模拟的上下文，应用自定义选项
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, "force-error", tc.forceErr)
			}

			practiceH(ctx)
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			if tc.expectSuccess {
				switch tc.method {
				case "POST":
					{
						assert.Equal(t, resp.Msg, "OK")
						assert.Equal(t, resp.Status, 0)

						// 这里要查询一下这个练习，如果是成功创建的话
						if tc.create {
							count := 0
							//如果是创建的话 就要去查询练习数量了
							s := `SELECT COUNT(*) FROM t_practice`
							err = conn.QueryRow(ctx, s).Scan(&count)
							if err != nil {
								t.Errorf("查询对应练习失败：%v", err)
							}
							if count != 1 {
								t.Errorf("查询到的练习数量为%v,预期为%v", count, 1)
							}
						} else {
							// 如果是更新的话 那就是别的检测路径 练习学生的数量
							count := 0
							s := `SELECT COUNT(*) FROM t_practice_student WHERE status = $1`
							err = conn.QueryRow(ctx, s, PracticeStudentStatus.Deleted).Scan(&count)
							if err != nil {
								t.Errorf("查询练习参与学生失败：%v", err)
							}
							if containsString(tc.name, "学生更新练习清除学生名单") {
								assert.Equal(t, count, 4)
							} else {
								assert.Equal(t, count, 0)
							}
						}
					}
				case "GET":
					{

						// 这里要区分学生权限
						if tc.userId == 1 {
							// 这里就代表是学生了
							type Practice struct {
								ID                  int64   `json:"ID"`
								Name                string  `json:"Name"`
								Type                string  `json:"Type"`
								AttemptCount        int     `json:"AttemptCount"`
								Difficulty          string  `json:"Difficulty"`
								AllowedAttempts     int64   `json:"AllowedAttempts"`
								QuestionCount       int64   `json:"QuestionCount"`
								WrongCount          int64   `json:"WrongCount"`
								TotalScore          float64 `json:"TotalScore"`
								HighestScore        float64 `json:"HighestScore"`
								PaperTotalScore     float64 `json:"PaperTotalScore"`
								PaperID             int64   `json:"PaperID"`
								LatestUnsubmittedID int64   `json:"LatestUnsubmittedID"`
								LatestSubmittedID   int64   `json:"LatestSubmittedID"`
								PendingMarkID       int64   `json:"PendingMarkID"`
							}

							// 最外层响应
							type Response struct {
								Practice []Practice `json:"practice"`
								Total    int        `json:"total"`
							}
							var testResp Response
							if err := json.Unmarshal(resp.Data, &resp); err != nil {
								t.Errorf("JSON 解析失败:%v", err)
								return
							}

							var mockReturn Response
							if err := json.Unmarshal([]byte(tc.expectedData), &mockReturn); err != nil {
								t.Errorf("JSON 解析失败:%v", err)
								return
							}

							// 安全遍历
							for _, p := range testResp.Practice {
								// 那个练习的东西，就必须检测
								for _, mp := range mockReturn.Practice {
									if p.AllowedAttempts != mp.AllowedAttempts {
										t.Errorf("返回的预期可练习次数：%v,实际为:%v", mp.AllowedAttempts, p.AllowedAttempts)
									}
									if p.Name != mp.Name {
										t.Errorf("返回的预期练习名称：%v,实际为:%v", mp.Name, p.Name)
									}
									if p.QuestionCount != mp.QuestionCount {
										t.Errorf("返回的预期练习试卷的题目数量：%v,实际为:%v", mp.QuestionCount, p.QuestionCount)
									}
									if p.TotalScore != mp.TotalScore {
										t.Errorf("返回的预期练习试卷学生的作答分数：%v,实际为:%v", mp.TotalScore, p.TotalScore)
									}
									if p.HighestScore != mp.HighestScore {
										t.Errorf("返回的预期练习试卷学生的历史最大作答分数：%v,实际为:%v", mp.HighestScore, p.HighestScore)
									}
									if p.WrongCount != mp.WrongCount {
										t.Errorf("返回的预期练习试卷学生的本次练习错题：%v,实际为:%v", mp.WrongCount, p.WrongCount)
									}
								}
							}
						} else {
							if containsString(tc.name, "详情") {
								//这里要进行练习的接收
								type Resp struct {
									PaperName  string        `json:"paper_name"`
									StudentCnt int64         `json:"student_count"`
									Practice   cmn.TPractice `json:"practice"`
								}
								var testresp Resp
								if err := json.Unmarshal(resp.Data, &testresp); err != nil {
									// 这里可以返回错误，而不是 panic
									fmt.Println("JSON 解析失败:", err)
									return
								}
								if testresp.Practice.Name.String != "练习模拟数据期末考试" {
									t.Errorf("此时练习返回的练习名称数据不对")
								}
								if testresp.Practice.CorrectMode.String != "00" {
									t.Errorf("此时练习返回的批改配置不对")
								}
								if testresp.Practice.AllowedAttempts.Int64 != 5 {
									t.Errorf("此时练习返回的可练习次数不对")
								}
								if testresp.StudentCnt != 4 {
									t.Errorf("学生数量不为4，实际为：%v", testresp.StudentCnt)
								}
								if testresp.PaperName != "考卷单元测试试卷名" {
									t.Errorf("试卷名称不为%v，实际为：%v", "考卷单元测试试卷名", testresp.PaperName)
								}

							} else {
								// 为这次解析额外定义两个“壳”结构体
								type practiceWrapper struct {
									Practice     cmn.TPractice `json:"practice"`
									StudentCount int           `json:"student_count"`
								}

								type response struct {
									Total     int               `json:"total"`
									Practices []practiceWrapper `json:"practices"`
								}

								var testresp response
								if err := json.Unmarshal(resp.Data, &testresp); err != nil {
									// 这里可以返回错误，而不是 panic
									fmt.Println("JSON 解析失败:", err)
									return
								}
								for _, p := range testresp.Practices {
									practice := p.Practice   // 已经就是 TPractice 类型
									stuCnt := p.StudentCount // int
									if practice.Name.String != "练习模拟数据期末考试" {
										t.Errorf("此时练习返回的练习名称数据不对")
									}
									if practice.CorrectMode.String != "00" {
										t.Errorf("此时练习返回的批改配置不对")
									}
									if practice.AllowedAttempts.Int64 != 5 {
										t.Errorf("此时练习返回的可练习次数不对")
									}
									if stuCnt != 4 {
										t.Errorf("学生数量不为4，实际为：%v", stuCnt)
									}
								}
								assert.Equal(t, resp.Status, 0)
							}
						}
					}

				case "PATCH":
					{
						// 如果是发布的话 那就要查看一下这个状态是否都变成对应的
					}
				}
			} else {
				// 这里是代表着的是出错的，此时要进行判断

				if !(tc.expectedMessage == "") {
					if !containsString(resp.Msg, tc.expectedMessage) {
						t.Errorf("返回的错误与预期不符")
					}
				}
				if containsString(tc.name, "异常2") {
					assert.Equal(t, resp.Status, 0)
				} else {
					assert.Equal(t, resp.Status, -1)
					assert.NotEmpty(t, resp.Msg)
					assert.Empty(t, resp.Data)
				}

			}

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
				_, err = conn.Exec(context.Background(), `DELETE FROM t_practice`)
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
		})
	}
}

// 对操作更新学生函数进行测试
func TestPracticeStudentListH(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	// 要给这个上下文区分到底是学生还是教师权限

	conn := cmn.GetPgxConn()
	var uid int64
	uid = 10086

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

	s = `DELETE FROM t_user_domain`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 这里还要删除user
	s = `DELETE FROM t_user`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}

	// 这里直接先准备好这个试卷就好了
	initQuestion(t, conn)

	initPaper(t, conn)

	// 初始化题目跟试卷，这个肯定是要有的
	tx1, _ := conn.Begin(ctx)

	var examPaperID1 *int64
	examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
	if err != nil {
		t.Fatalf("生成考卷逻辑出错：%v", err)
	}
	if examPaperID1 == nil {
		t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
	}

	// 这里要插入这个需要更新到那个paper的字段
	s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
	_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
	if err != nil {
		t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
	}

	// 这里插入几个学生，就代表是用户了
	s = `INSERT INTO t_user (id,category,account,official_name,id_card_no,mobile_phone)VALUES($1,$2,$3,$4,$5,$6),($7,$8,$9,$10,$11,$12)ON CONFLICT (id) DO NOTHING`
	_, err = tx1.Exec(ctx, s, 1, "sys^user", "38ieqiwutnfg", "邹德伦", "44018427781715606X", "15920440457", 2, "sys^user", "ic053cwd0dw0", "邹德伦克隆体", "440184277817156060", "15920440907")
	if err != nil {
		t.Errorf("插入用户失败:%v", err)
	}

	s = `INSERT INTO t_user_domain (sys_user, domain)
				VALUES ($1, (SELECT id FROM t_domain WHERE domain = $2)),($3,(SELECT id FROM t_domain WHERE domain = $4))`
	_, err = tx1.Exec(ctx, s, 1, "sys^user", 2, "sys^user")
	if err != nil {
		t.Errorf("插入用户失败:%v", err)
	}

	// 这里创建练习跟学生
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
	s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	err = tx1.Commit(ctx)
	if err != nil {
		t.Errorf("提交事务失败:%v", err)
	}
	testCases := []struct {
		name     string
		method   string
		url      string
		reqBody  *cmn.ReqProto
		userId   int64
		forceErr string
		ctxKey   string
		ctxValue string
		// 预期结果
		expectSuccess       bool            // 是否期望成功
		expectedMessage     string          // 预期错误消息
		expectFailedMessage string          // 预期成功消息
		expectedData        json.RawMessage // 预期数据（可选）
		setup               func() error
		Domain              []cmn.TDomain
		create              bool //是否是创建的练习
	}{
		{
			name: "GET 请求 - 学生权限缺少练习类型 非法用户ID",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?type=00&page=1",
			userId:          -1,
			expectSuccess:   false,
			expectedMessage: "invalid UserID",
		},
		{
			name: "GET 请求 - 学生权限无法操作学生列表 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.Student)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?type=00&page=1",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "当前权限无法操作学生名单",
		},
		{
			name: "GET 请求 - 请求缺少id参数 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "缺失练习ID",
		},
		{
			name: "GET 请求 - 触发ParseInt失败 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=abc",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "练习ID解析为整形失败",
		},
		{
			name: "GET 请求 - 触发查询练习学生失败  query",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "query",
			expectSuccess:   false,
			expectedMessage: "query studentList in practice failed",
		},
		{
			name: "GET 请求 - 全过程成功，但在关闭数据库连接失败 Close",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "Close",
			expectSuccess:   true,
			expectedMessage: "OK",
		},
		{
			name: "GET 请求 - 全过程成功，但在关闭数据库连接失败 Scan",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "Scan",
			expectSuccess:   false,
			expectedMessage: "解析数据库数据失败",
		},
		{
			name: "GET 请求 - 全过程成功，但是无法此时练习名单为空",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "GET",
			url:    "/api/practiceStudentList?id=10086",
			userId: uid,
			setup: func() error {
				//这里要更新学生名单
				s = `DELETE FROM t_practice_student`
				_, err := conn.Exec(ctx, s)
				if err != nil {
					t.Errorf("无法成功删除练习学生名单")
				}
				return err
			},
			expectSuccess:   true,
			expectedMessage: "OK but empty result",
		},
		{
			name: "GET 请求 - 预备查询用户信息SQL失败 sqlx-error",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "sqlx-error",
			expectSuccess:   false,
			expectedMessage: "prepare sqlx.In sql query failed",
		},
		{
			name: "GET 请求 - 预备查询用户信息SQL失败 query1",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "query1",
			expectSuccess:   false,
			expectedMessage: "查询练习中学生账号信息失败",
		},
		{
			name: "GET 请求 - 全过程成功，但在关闭数据库连接失败 Close1",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "Close1",
			expectSuccess:   true,
			expectedMessage: "OK",
		},
		{
			name: "GET 请求 - 全过程成功，但在解析数据库连接失败 Scan1",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "Scan1",
			expectSuccess:   false,
			expectedMessage: "解析数据库数据失败",
		},
		{
			name: "GET 请求 - 全过程成功，但反序列数据结构失败 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "GET",
			url:             "/api/practiceStudentList?id=10086",
			userId:          uid,
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "返回数据反序列失败",
		},
		{
			name: "POST 请求 - 全过程成功，但反序列数据结构失败 io.ReadAll",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			userId: uid,
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			forceErr:        "io.ReadAll",
			expectSuccess:   false,
			expectedMessage: "读取连接字节流失败",
		},
		{
			name: "POST 请求 - 全过程成功，但反序列数据结构失败 Close",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			userId:          uid,
			forceErr:        "Close",
			expectSuccess:   true,
			expectedMessage: "OK",
		},
		{
			name: "POST 请求 - 传入空参数",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method:          "POST",
			url:             "/api/practiceStudentList",
			reqBody:         &cmn.ReqProto{},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "Call /api/practiceStudentList with  empty body",
		},
		{
			name: "POST 请求 - 全过程成功，但反序列数据结构失败 json1",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			forceErr:        "json1",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "反序列前端data字段传入的json数据失败",
		},
		{
			name: "POST 请求 - 全过程成功，但反序列数据结构失败 json",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			forceErr:        "json",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "反序列前端传入的json数据失败",
		},
		{
			name: "POST 请求 - 触发参数检测失败 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":-10086, "student":[] }`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "validation failed",
		},
		{
			name: "POST 请求 - 触发LoadPraticeById失败 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			forceErr:        "prepare",
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "prepare sql err",
		},
		{
			name: "POST 请求 - 触发LoadPraticeById失败 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "此时练习状态不为已发布状态",
			setup: func() error {
				// 更新一下这个练习状态
				s := `UPDATE t_practice SET status = $1`
				_, err := conn.Exec(ctx, s, PracticeStatus.PendingRelease)
				if err != nil {
					t.Errorf("更新练习状态失败")
				}
				return err
			},
		},
		{
			name: "POST 请求 - 触发UpsertPraticeV2失败 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "POST",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			userId:          uid,
			forceErr:        "beginTx",
			expectSuccess:   false,
			expectedMessage: "beginTx called failed",
		},
		{
			name: "DELETE 请求 - 触发没有请求方法 ",
			Domain: []cmn.TDomain{
				{ID: null.IntFrom(PracticeDomainID.SuperAdmin)},
			},
			method: "DELETE",
			url:    "/api/practiceStudentList",
			reqBody: &cmn.ReqProto{
				Data: json.RawMessage(`{ "practice_id":10086, "student":[] }`),
			},
			userId:          uid,
			expectSuccess:   false,
			expectedMessage: "please call /api/practiceStudentList with  http POST/GET method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 这里需要去准备这个东西
			if tc.setup != nil {
				err := tc.setup()
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

			}

			var req *http.Request
			var err error

			if tc.reqBody != nil {

				// 对于 POST 请求，准备请求体
				bodyBytes, err := json.Marshal(tc.reqBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(bodyBytes))
				if tc.name == "buf-zero" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(nil))
				} else if containsString(tc.name, "传入空参数") {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer([]byte("")))
				}
			} else {
				// 对于 GET 请求，没有请求体
				req, err = http.NewRequest(tc.method, tc.url, nil)
			}
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			// 创建模拟的上下文，应用自定义选项
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, "force-error", tc.forceErr)
			}
			practiceStudentListH(ctx)

			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)

			if tc.expectSuccess {
				if tc.expectedMessage != resp.Msg {
					t.Errorf("预期正确返回的信息：%v,实际返回：%v", resp.Msg, tc.expectedMessage)
				}
				assert.Equal(t, resp.Status, 0)
				if tc.method == "GET" {
					// 这里如果是正常的话 那就需要根据哪个学生的具体数量去判断了
					if !containsString(tc.name, "但是无法此时练习名单为空") {
						// 这里就去查询学生的名称了
						// 1. 结构体
						type Student struct {
							ID           int64  `json:"id"`
							Account      string `json:"account"`
							OfficialName string `json:"official_name"`
							Password     string `json:"password"`
							IDCardNo     string `json:"id_card_no"`
							Phone        string `json:"phone"`
						}

						// 2. 解析
						var students []Student
						if err := json.Unmarshal(resp.Data, &students); err != nil {
							// 处理错误
							t.Errorf("反序列结构失败：%v", err)
						}

						// 3. 使用
						for _, stu := range students {
							if stu.Account != "38ieqiwutnfg" && stu.Account != "ic053cwd0dw0" {
								t.Errorf("返回的学生账号不正确")
							}
							if stu.OfficialName != "邹德伦" && stu.OfficialName != "邹德伦克隆体" {
								t.Errorf("返回的学生名称不正确")
							}
							if stu.IDCardNo != "440184277817156060" && stu.IDCardNo != "44018427781715606X" {
								t.Errorf("返回的学生名称不正确")
							}
							if stu.Phone != "15920440907" && stu.Phone != "15920440457" {
								t.Errorf("返回的学生电话号码不正确")
							}
						}
					}
				}
			} else {
				if !containsString(resp.Msg, tc.expectedMessage) {
					t.Errorf("预期正确返回的信息：%v,实际返回：%v", resp.Msg, tc.expectedMessage)
				}
				assert.Equal(t, resp.Status, -1)
				assert.NotEmpty(t, resp.Msg)
				assert.Empty(t, resp.Data)
			}

			t.Cleanup(func() {
				// 这里要重新生成一下
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
				s = `DELETE FROM t_user_domain`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 这里还要删除user
				s = `DELETE FROM t_user`
				_, err = conn.Exec(ctx, s)
				if err != nil {
					z.Fatal(err.Error())
				}
				// 这里直接先准备好这个试卷就好了
				initQuestion(t, conn)

				initPaper(t, conn)

				// 初始化题目跟试卷，这个肯定是要有的
				tx1, _ := conn.Begin(ctx)

				var examPaperID1 *int64
				examPaperID1, err = examPaperService.GenerateExamPaper(context.Background(), tx1, uid, uid)
				if err != nil {
					t.Fatalf("生成考卷逻辑出错：%v", err)
				}
				if examPaperID1 == nil {
					t.Fatal("生成考卷逻辑出错,返回考卷ID为空")
				}

				// 这里要插入这个需要更新到那个paper的字段
				s = `UPDATE t_paper SET exam_paper_id = $1 , status = $2 WHERE id = $3`
				_, err = tx1.Exec(ctx, s, *examPaperID1, examPaperService.PaperStatus.Published, uid)
				if err != nil {
					t.Errorf("更新试卷的考卷ID与状态失败:%v", err)
				}

				// 这里插入几个学生，就代表是用户了
				s = `INSERT INTO t_user (id,category,account,official_name,id_card_no,mobile_phone)VALUES($1,$2,$3,$4,$5,$6),($7,$8,$9,$10,$11,$12)ON CONFLICT (id) DO NOTHING`
				_, err = tx1.Exec(ctx, s, 1, "sys^user", "38ieqiwutnfg", "邹德伦", "44018427781715606X", "15920440457", 2, "sys^user", "ic053cwd0dw0", "邹德伦克隆体", "440184277817156060", "15920440907")
				if err != nil {
					t.Errorf("插入用户失败:%v", err)
				}

				s = `INSERT INTO t_user_domain (sys_user, domain)
				VALUES ($1, (SELECT id FROM t_domain WHERE domain = $2)),($3,(SELECT id FROM t_domain WHERE domain = $4))`
				_, err = tx1.Exec(ctx, s, 1, "sys^user", 2, "sys^user")
				if err != nil {
					t.Errorf("插入用户失败:%v", err)
				}

				// 这里创建练习跟学生
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
				s = `INSERT INTO t_practice_student (student_id,practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
				_, err = tx1.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
				}
				err = tx1.Commit(ctx)
				if err != nil {
					t.Errorf("提交事务失败:%v", err)
				}
			})
		})

		t.Cleanup(func() {
			// 这里要重新生成一下
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
			s = `DELETE FROM t_user_domain`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 这里还要删除user
			s = `DELETE FROM t_user`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}
}

func TestRegisterPracticeH(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	// 要给这个上下文区分到底是学生还是教师权限

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

	s = `DELETE FROM t_user_domain`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 这里还要删除user
	s = `DELETE FROM t_user`
	_, err = conn.Exec(ctx, s)
	if err != nil {
		z.Fatal(err.Error())
	}
	// 这里直接先准备好这个试卷就好了
	tx, _ := conn.Begin(ctx)

	// 增加几个报名计划 练习
	var practiceName1, practiceName2, practiceName3, practiceName4 string

	practiceName1 = "单元测试发布练习1"
	practiceName2 = "单元测试发布练习2"
	practiceName3 = "单元测试发布练习3"
	practiceName4 = "单元测试发布练习4"

	// 先创建数据，不管到底是不是这个测试用例需要的 三种练习状态 然后还有状态不一的练习
	// 删除 exam_paper_id 字段后，每组数据保留 8 个字段，调整占位符编号
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id,status)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8),
($9, $10, $11, $12, $13, $14, $15, $16),
($17, $18, $19, $20, $21, $22, $23, $24),
($25, $26, $27, $28, $29, $30, $31, $32)`

	_, err = tx.Exec(ctx, s,
		// 第 1 组数据
		10086, practiceName1, "00", 10086, 10086, "00", 10086, PracticeStatus.Released,
		// 第 2 组数据
		10087, practiceName2, "00", 10087, 10086, "00", 10086, PracticeStatus.Released,
		// 第 3 组数据
		10088, practiceName3, "00", 10086, 10086, "00", 10086, PracticeStatus.Released,
		// 第 4 组数据
		10089, practiceName4, "00", 10087, 10086, "00", 10086, PracticeStatus.Released,
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
	if tCount != 4 {
		t.Errorf("此时练习数量不为4，实际为：%v", tCount)
	}

	// 这里还是要插入姓名
	// 这里插入几个学生，就代表是用户了
	s = `INSERT INTO t_user (id,category,account,official_name,id_card_no,mobile_phone)VALUES($1,$2,$3,$4,$5,$6),($7,$8,$9,$10,$11,$12)ON CONFLICT (id) DO NOTHING`
	_, err = tx.Exec(ctx, s, 10086, "sys^user", "38ieqiwutnfg", "邹德伦", "44018427781715606X", "15920440457", 10087, "sys^user", "ic053cwd0dw0", "邹德伦克隆体", "440184277817156060", "15920440907")
	if err != nil {
		t.Errorf("插入用户失败:%v", err)
	}

	s = `INSERT INTO t_user_domain (sys_user, domain)
				VALUES ($1, (SELECT id FROM t_domain WHERE domain = $2)),($3,(SELECT id FROM t_domain WHERE domain = $4))`
	_, err = tx.Exec(ctx, s, 10086, "sys^user", 10087, "sys^user")
	if err != nil {
		t.Errorf("插入用户失败:%v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Errorf("提交失败哦：%v", err)
	}

	// 这里成功插入多个练习了
	testCases := []struct {
		name     string
		method   string
		url      string
		reqBody  *cmn.ReqProto
		userId   int64
		forceErr string
		ctxKey   string
		ctxValue string
		// 预期结果
		expectSuccess       bool            // 是否期望成功
		expectedMessage     string          // 预期错误消息
		expectFailedMessage string          // 预期成功消息
		expectedData        json.RawMessage // 预期数据（可选）
		setup               func() error
		Domain              []cmn.TDomain
	}{
		{
			name:            "正常获取列表",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.Admin)}},
			expectSuccess:   true,
			expectedMessage: "",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "权限不足",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.Student)}},
			expectSuccess:   false,
			expectedMessage: "当前权限无法操作获取可选择练习",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "携带非法用户id",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          -10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			expectedMessage: "invalid UserID",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "使用的不是GET方法",
			method:          "POST",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			expectedMessage: "请使用GET方法请求该路径",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "缺少分页号码",
			method:          "GET",
			url:             "/api/registerPractice?page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页号",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "强制解析前端发送的分页号大小失败",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			forceErr:        "pageParseInt",
			expectedMessage: "分页查询的页号解析失败",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "强制解析前端发送的分页大小失败",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			forceErr:        "pageSizeParseInt",
			expectedMessage: "分页页大小解析失败",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "缺少分页大小",
			method:          "GET",
			url:             "/api/registerPractice?page=1",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			expectedMessage: "缺失分页查询页大小",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "分页大小过大",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=1000000",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			expectSuccess:   false,
			expectedMessage: "页数量过大，不允许访问",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "强制调用函数失败",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			forceErr:        "query",
			expectSuccess:   false,
			expectedMessage: "查询可用于报名计划绑定的练习失败",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
		{
			name:            "强制将查询结果序列化失败 ",
			method:          "GET",
			url:             "/api/registerPractice?page=1&page_size=10",
			userId:          10086,
			Domain:          []cmn.TDomain{{ID: null.IntFrom(PracticeDomainID.SuperAdmin)}},
			forceErr:        "json",
			expectSuccess:   false,
			expectedMessage: "返回数据反序列失败",
			expectedData: json.RawMessage(`
			[
				{
					"ID":10086,
					"Name":"单元测试发布练习1",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10087,
					"Name":"单元测试发布练习2",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				},
				{
					"ID":10088,
					"Name":"单元测试发布练习3",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦"
				},
				{
					"ID":10089,
					"Name":"单元测试发布练习4",
					"CorrectMode":"00",
					"Type":"00",
					"AllowedAttempts":10086,
					"Status":"00",
					"TeacherName":"邹德伦克隆体"
				}
			]`,
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				err := tc.setup()
				if err != nil {
					t.Fatalf("Failed to setup database: %v", err)
				}

			}

			var req *http.Request
			var err error

			if tc.reqBody != nil {

				// 对于 POST 请求，准备请求体
				bodyBytes, err := json.Marshal(tc.reqBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(bodyBytes))
				if tc.name == "buf-zero" {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(nil))
				} else if containsString(tc.name, "传入空参数") {
					req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer([]byte("")))
				}
			} else {
				// 对于 GET 请求，没有请求体
				req, err = http.NewRequest(tc.method, tc.url, nil)
			}
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			// 创建模拟的上下文，应用自定义选项
			ctx := createMockContext(req, tc.userId, tc.Domain)

			//传入强制err
			if tc.forceErr != "" {
				ctx = context.WithValue(ctx, "force-error", tc.forceErr)
			}

			registerPracticeH(ctx)

			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)
			if tc.expectSuccess {
				if resp.Msg != "OK" {
					t.Errorf("成功返回的信息不为OK")
				}
				// 如果有的话，那就需要获取后端返回数据
				var testResp1, testResp2 []RegisterPractice
				if err := json.Unmarshal(resp.Data, &testResp1); err != nil {
					t.Errorf("JSON 解析失败:%v", err)
					return
				}
				if err := json.Unmarshal(tc.expectedData, &testResp2); err != nil {
					t.Errorf("JSON 解析失败:%v", err)
					return
				}
				for idx, v := range testResp1 {
					// 遍历检测是否相等
					expectD := testResp2[idx]
					if v.Name.String != expectD.Name.String {
						t.Errorf("预期返回的练习名称：%v，实际返回的练习名称：%v", expectD.Name.String, v.Name.String)
					}
					if v.TeacherName != expectD.TeacherName {
						t.Errorf("预期返回的练习教师名称：%v，实际返回的练习教师名称：%v", expectD.TeacherName, v.TeacherName)
					}
					if v.AllowedAttempts.Int64 != expectD.AllowedAttempts.Int64 {
						t.Errorf("预期返回的练习最大尝试次数：%v，实际返回的练习最大尝试次数：%v", expectD.AllowedAttempts.Int64, v.AllowedAttempts.Int64)
					}
				}
			} else {
				if !containsString(resp.Msg, tc.expectedMessage) {
					t.Errorf("此时返回的错误信息：%v,实际为：%v", resp.Msg, tc.expectedMessage)
				}
			}
		})
		t.Cleanup(func() {
			// 这里要重新生成一下
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
			s = `DELETE FROM t_user_domain`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
			// 这里还要删除user
			s = `DELETE FROM t_user`
			_, err = conn.Exec(ctx, s)
			if err != nil {
				z.Fatal(err.Error())
			}
		})
	}
}

// 创建模拟的上下文，更加通用的版本，支持自定义用户ID和请求头
func createMockContext(req *http.Request, userId int64, domain []cmn.TDomain, role ...int) context.Context {
	// 创建基本的上下文
	ctx := context.Background()

	// 创建响应记录器
	rec := httptest.NewRecorder()
	var r int64 = 0
	if len(role) > 0 {
		r = int64(role[0])
	}
	// 创建默认的服务上下文
	q := &cmn.ServiceCtx{
		R: req,
		W: rec,
		SysUser: &cmn.TUser{
			ID: null.IntFrom(userId), // 默认用户ID
		},
		Msg:     &cmn.ReplyProto{},
		Domains: domain,
		Role:    r,
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}
