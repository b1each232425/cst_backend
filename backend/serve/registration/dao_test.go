package registration

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"reflect"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/practice_mgt"
)

//annotation:register-service
//author:{"name":"LilEYi","tel":"13535215794", "email":"3102128343@qq.com"}

var (
	ctx = context.Background()
	now = time.Now().UnixMilli()
	uid = null.IntFrom(10086)
)

func TestAddRegister(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := ""
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
	_, err = db.Exec(ctx, s, registerName)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_exam_plan_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	type args struct {
		registration *cmn.TRegisterPlan
		practiceIds  []int64
		userID       int64
		expectErr    error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正确传入.完整报名计划信息 + 练习数组",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID:         null.IntFrom(1),
					Name:       null.StringFrom(registerName),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				practiceIds: []int64{
					1,
					2,
					3,
				},
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "正常测试2 完整的报名计划信息 但无练习数据",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID:         null.IntFrom(1),
					Name:       null.StringFrom(registerName),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "异常1 ,触发",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name:       null.StringFrom(registerName),
					Creator:    null.IntFrom(uid),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().Add(time.Hour * 24).UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				userID:    uid,
				expectErr: errors.New("beginTx called failed"),
			},
		},
		{
			name: "异常2 触发新增报名计划错误",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name:       null.StringFrom(registerName),
					Creator:    null.IntFrom(uid),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().Add(time.Hour * 24).UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				practiceIds: []int64{
					1,
				},
				userID:    uid,
				expectErr: errors.New("query failed"),
			},
		},
		{
			name: "异常3 触发rollback",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name:       null.StringFrom(registerName),
					Creator:    null.IntFrom(uid),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().Add(time.Hour * 24).UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				practiceIds: []int64{
					1,
				},
				userID:    uid,
				expectErr: errors.New("触发回滚"),
			}},
		{
			name: "异常4 触发commit错误",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name:       null.StringFrom(registerName),
					Creator:    null.IntFrom(uid),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().Add(time.Hour * 24).UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				practiceIds: []int64{
					1,
				},
				userID:    uid,
				expectErr: errors.New("commit failed"),
			}},
		{
			name: "异常5 触发更新updatePractice错误",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name:       null.StringFrom(registerName),
					Creator:    null.IntFrom(uid),
					Course:     null.StringFrom("00"),
					Status:     null.StringFrom("02"),
					StartTime:  null.IntFrom(time.Now().UnixMilli()),
					EndTime:    null.IntFrom(time.Now().Add(time.Hour * 24).UnixMilli()),
					CreateTime: null.IntFrom(time.Now().UnixMilli()),
					UpdateTime: null.IntFrom(time.Now().UnixMilli()),
				},
				practiceIds: []int64{
					1,
				},
				userID:    uid,
				expectErr: errors.New("添加名单失败"),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "rollback")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "commit")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else {
				ctx = context.Background()
			}
			err := AddRegister(ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID)
			if tt.args.expectErr != nil {
				//传入绑定的练习id
				if err != nil {
					if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
						t.Errorf("报错与预期：%v  %v", err, tt.args.expectErr)
					}
				} else {
					t.Errorf("预期有错误但是没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("AddRegister() 期望没有报错但是报错, wantErr %v", err)
				}
				//执行一下查询
				s = `SELECT COUNT(*) FROM assessuser.t_register_plan WHERE name = $1`
				var count int
				_ = db.QueryRow(ctx, s, tt.args.registration.Name).Scan(&count)
				if count != 1 {
					t.Errorf("AddRegister() count = %v, want %v", count, 1)
				}
				if strings.Contains(tt.name, "期望正常1") {
					//如果包含这个的话就去查询练习数量
					s = `SELECT COUNT(*) FROM  assessuser.t_register_practice `
					_ = db.QueryRow(ctx, s, tt.args.registration.Name).Scan(&count)
					if count != 1 {
						t.Errorf("AddRegister() count = %v, want %v", count, 1)
					}
				}

			}
			t.Cleanup(func() {
				//去除之前创建的所有数据
				s := `DELETE FROM assessuser.t_register_plan`
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
				}
			})
		})
	}

}

func TestDeleteRegisterPracticeStudent(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	registerName := ""
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
	_, err = db.Exec(ctx, s, registerName)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_exam_plan_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	//插入报名计划数据
	//插入练习数据
	// 先创建这个数据，最后测试完毕再删掉
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 1, 1, uid, uid, "00", 2, 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (name, status,creator ,create_time) VALUES  ($1 ,$2 ,$3,$4)`
	_, err = db.Exec(ctx, s, "报名计划", "00", uid, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Delete)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 1, 1, uid, "00", 2, 1, uid, "02")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerIDs []int64
		expectErr   error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				registerIDs: []int64{1},
				expectErr:   nil,
			},
		},
		{
			name: "异常1",
			args: args{
				registerIDs: []int64{},
				expectErr:   errors.New("查询已删除的练习ID失败:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerIDs: []int64{1},
				expectErr:   errors.New("扫描已删除的练习ID失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerIDs: []int64{1},
				expectErr:   errors.New("beginTx called failed:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Errorf("OperateRegisterStudentStatus() error = %v", err)
				}
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if strings.Contains(tt.name, "异常3") {
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Errorf("OperateRegisterStudentStatus() error = %v", err)
				}
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else {
				ctx = context.Background()
			}
			err := DeleteRegisterPracticeStudent(ctx, tx, uid, tt.args.registerIDs)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Errorf("%v报错与预期：%v", err, tt.args.expectErr)
				}
			} else {
				if err != nil {
					t.Errorf("没期望报错但是报错")
				}
				//搜索一下数据库还有没有学生
				s = `SELECT COUNT(*) FROM t_practice_student WHERE practice_id = $1 AND status =$2 `
				var count int
				err := db.QueryRow(ctx, s, uid, practice_mgt.PracticeStudentStatus.Deleted).Scan(&count)
				if err != nil {
					t.Errorf("查询数据库错误")
				}
				if count != 2 {
					t.Errorf("删除practice_student数据失败 %d", count)
				}
			}
		})
		t.Cleanup(func() {
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice`
			_, err := db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice_student `
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_paper`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划学生
			s = `DELETE FROM assessuser.t_exam_plan_student`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetRegisterStudentById(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := ""
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
	_, err = db.Exec(ctx, s, registerName)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_exam_plan_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	//插入报名计划数据
	//插入练习数据
	// 先创建这个数据，最后测试完毕再删掉
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	// 这里也随便插入几个学生
	s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 1, 1, uid, uid, "00", 2, 2, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time) VALUES  ($1 ,$2 ,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Delete)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00", 20023, 1, uid, "02", "00")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerID   int64
		message      string
		registerType string
		status       string
		orderBy      []string
		page         int
		pageSize     int
		searchType   string
		expectErr    error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试报名计划",
			args: args{
				registerID:   1,
				message:      "1",
				registerType: "00",
				status:       "02",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "00",
				expectErr:  nil,
			},
		},
		{
			name: "正常测试考试",
			args: args{
				registerID:   1,
				message:      "1",
				registerType: "00",
				status:       "02",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  nil,
			},
		},
		{
			name: "异常1",
			args: args{
				registerID:   1,
				message:      "1",
				registerType: "00",
				status:       "02",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  errors.New("query register failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerID:   1,
				message:      "1",
				registerType: "00",
				status:       "02",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  errors.New("查询总数失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerID:   1,
				message:      "1",
				registerType: "00",
				status:       "02",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  errors.New("查询总数失败:"),
			},
		},
		{
			name: "异常4",
			args: args{
				registerID:   1,
				message:      "",
				registerType: "",
				status:       "",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  errors.New("row close failed:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registerID:   1,
				message:      "",
				registerType: "",
				status:       "",
				orderBy: []string{
					"eps.create_time",
				},
				page:       1,
				pageSize:   10,
				searchType: "00",
				expectErr:  errors.New("解析数据失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "scan2")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "close")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else {
				ctx = context.Background()
			}

			_, total, err := GetRegisterStudentById(ctx, tt.args.registerID, tt.args.message, tt.args.registerType, tt.args.status, tt.args.orderBy, tt.args.page, tt.args.pageSize, uid, tt.args.searchType)
			if tt.args.expectErr != nil {
				if err == nil {
					t.Error("期望有错误但没有返回错误")
					return
				}
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 报错与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Fatal("预期没有错误但是返回错误")
				}
				if total != 2 {
					z.Fatal(fmt.Sprintf("返回的total为 %d", total))
				}

			}
		})
		t.Cleanup(func() {
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice`
			_, err := db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice_student `
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_paper`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划学生
			s = `DELETE FROM assessuser.t_exam_plan_student`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestListRegisterS(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := ""
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
	_, err = db.Exec(ctx, s, registerName)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_exam_plan_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	//插入报名计划数据
	//插入练习数据

	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,course) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), "00")
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		name      string
		course    string
		status    string
		orderBy   []string
		page      int
		pageSize  int
		userID    int64
		expectErr error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "正常测试2",
			args: args{
				name:      "",
				course:    "",
				status:    "",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "异常1",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: errors.New("search register failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: errors.New("查询数据失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				name:      "",
				course:    "",
				status:    "",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: errors.New("解析数据失败:"),
			},
		},
		{
			name: "异常4",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: errors.New("解析数据失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				userID:    uid,
				expectErr: errors.New("row failed to close:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "queryx")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "lScan")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "row close")
			} else {
				ctx = context.Background()
			}
			_, total, err := ListRegisterS(ctx, tt.args.name, tt.args.course, tt.args.status, []string{"r.create_time"}, tt.args.page, tt.args.pageSize, tt.args.userID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if total != 1 {
					z.Error(fmt.Sprintf("返回的total为 %d", total))
				}

			}

		})
		t.Cleanup(func() {

			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestListRegisterT(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := ""
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
	_, err = db.Exec(ctx, s, registerName)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_exam_plan_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice_student`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	//插入报名计划数据
	//插入练习数据

	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,course) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "06", uid, time.Now().UnixMilli(), "00")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		name       string
		course     string
		status     string
		orderBy    []string
		page       int
		pageSize   int
		searchType string
		expectErr  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试1",
			args: args{
				name:       "报名计划",
				course:     "00",
				status:     "06",
				orderBy:    []string{"r.create_time"},
				page:       1,
				pageSize:   10,
				searchType: "00",
				expectErr:  nil,
			},
		},
		{
			name: "正常测试2",
			args: args{
				name:       "报名计划",
				course:     "00",
				status:     "",
				orderBy:    []string{"r.create_time"},
				page:       1,
				pageSize:   10,
				searchType: "02",
				expectErr:  nil,
			},
		},
		{
			name: "异常1",
			args: args{
				name:     "报名计划",
				course:   "00",
				status:   "06",
				orderBy:  []string{"r.create_time"},
				page:     1,
				pageSize: 10,

				expectErr: errors.New("search register failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				name:     "报名计划",
				course:   "00",
				status:   "06",
				orderBy:  []string{"r.create_time"},
				page:     1,
				pageSize: 10,

				expectErr: errors.New("查询数据失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				name:     "",
				course:   "",
				status:   "",
				orderBy:  []string{"r.create_time"},
				page:     1,
				pageSize: 10,

				expectErr: errors.New("解析数据失败:"),
			},
		},
		{
			name: "异常4",
			args: args{
				name:     "报名计划",
				course:   "00",
				status:   "06",
				orderBy:  []string{"r.create_time"},
				page:     1,
				pageSize: 10,

				expectErr: errors.New("解析数据失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				name:      "报名计划",
				course:    "00",
				status:    "00",
				orderBy:   []string{"r.create_time"},
				page:      1,
				pageSize:  10,
				expectErr: errors.New("row failed to close:"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "queryx")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "scan")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "lScan")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "row close")
			} else {
				ctx = context.Background()
			}
			_, total, err := ListRegisterT(ctx, tt.args.name, tt.args.course, tt.args.status, tt.args.orderBy, tt.args.page, tt.args.pageSize, uid, tt.args.searchType)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if total != 1 {
					z.Error(fmt.Sprintf("返回的total为 %d", total))
				}
			}

		})
		t.Cleanup(func() {
			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestListReviewers(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := "报名计划"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
	}

	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,course,reviewer_ids) VALUES  ($1 ,$2 ,$3,$4,$5,$6,$7)`
	_, err = db.Exec(ctx, s, 1, registerName, "06", uid, time.Now().UnixMilli(), "00", []int64{10100})
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerID int64
		name       string
		page       int
		pageSize   int
		orderBy    []string
		userId     int64
		expectErr  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  nil,
			},
		},
		{
			name: "正常测试2",
			args: args{
				registerID: 1,
				name:       "",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  nil,
			},
		},
		{
			name: "异常0",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     0,
				expectErr:  errors.New("查询报名计划下的审查人失败:"),
			},
		},
		{
			name: "异常registerID",
			args: args{
				registerID: 0,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("查询报名计划下的审查人失败:"),
			},
		},
		{
			name: "异常1",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("查询报名计划下的审查人失败:"),
			},
		},

		{
			name: "异常2",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("search register failed:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				expectErr:  errors.New("row scan failed:"),
				userId:     uid,
				orderBy:    []string{"u.create_time"},
			},
		},
		{
			name: "异常4",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("查询数据失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("解析数据失败:"),
			},
		},
		{
			name: "异常6",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("row failed to close:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "queryr")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "row scan")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "queryx")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "lScan")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "row close")
			} else {
				ctx = context.Background()
			}

			_, total, err := ListReviewers(ctx, tt.args.userId, tt.args.registerID, tt.args.name, tt.args.page, tt.args.pageSize, tt.args.orderBy)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if total != 1 {
					z.Error(fmt.Sprintf("返回的total为 %d", total))
				}
			}
		})
		t.Cleanup(func() {
			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})

	}
}

func TestLoadRegisterById(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	registerName := "报名计划"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
	}

	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,course,reviewer_ids) VALUES  ($1 ,$2 ,$3,$4,$5,$6,$7)`
	_, err = db.Exec(ctx, s, 1, registerName, "06", uid, time.Now().UnixMilli(), "00", []int64{10100})
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerID int64
		expectErr  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				registerID: 1,
				expectErr:  nil,
			},
		},
		{
			name: "异常1",

			args: args{
				registerID: 1,
				expectErr:  errors.New("查询报名计划失败:"),
			},
		},
		{
			name: "异常2",

			args: args{
				registerID: 0,
				expectErr:  errors.New("查询报名计划下的练习失败:"),
			},
		},
		{
			name: "异常3",

			args: args{
				registerID: 1,
				expectErr:  errors.New("扫描报名计划下的练习失败:"),
			},
		},
		{
			name: "异常4",

			args: args{
				registerID: 1,
				expectErr:  errors.New("查询报名计划下的审查人失败:"),
			},
		},
		{
			name: "异常5",

			args: args{
				registerID: 1,
				expectErr:  errors.New("扫描报名计划下的审查人失败:"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "scann")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "query3")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "scanl")
			} else {
				ctx = context.Background()
			}
			_, _, _, _, err := LoadRegisterById(ctx, tt.args.registerID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}

			}

		})
		t.Cleanup(func() {
			//删除报名计划
			s = `DELETE FROM assessuser.t_register_plan`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			//删除报名计划练习
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})

	}
}

func TestLoadRegisterByIds(t *testing.T) {
	type args struct {
		ctx         context.Context
		registerIDs []int64
	}
	tests := []struct {
		name          string
		args          args
		wantRegisters []*cmn.TRegisterPlan
		wantErr       bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegisters, err := LoadRegisterByIds(tt.args.ctx, tt.args.registerIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRegisterByIds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRegisters, tt.wantRegisters) {
				t.Errorf("LoadRegisterByIds() gotRegisters = %v, want %v", gotRegisters, tt.wantRegisters)
			}
		})
	}
}

func TestLoadRegisterStudentStatusByIds(t *testing.T) {
	type args struct {
		ctx        context.Context
		ids        []int64
		registerID int64
	}
	tests := []struct {
		name         string
		args         args
		wantStudents []cmn.TExamPlanStudent
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStudents, err := LoadRegisterStudentStatusByIds(tt.args.ctx, tt.args.ids, tt.args.registerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRegisterStudentStatusByIds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotStudents, tt.wantStudents) {
				t.Errorf("LoadRegisterStudentStatusByIds() gotStudents = %v, want %v", gotStudents, tt.wantStudents)
			}
		})
	}
}

func TestMoveStudent(t *testing.T) {
	type args struct {
		ctx            context.Context
		fromRegisterID int64
		toRegisterID   int64
		students       []registerStudentType
		status         string
		userID         int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MoveStudent(tt.args.ctx, tt.args.fromRegisterID, tt.args.toRegisterID, tt.args.students, tt.args.status, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("MoveStudent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOperateRegisterStatus(t *testing.T) {
	type args struct {
		ctx         context.Context
		registerIDs []int64
		status      string
		userID      int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := OperateRegisterStatus(tt.args.ctx, tt.args.registerIDs, tt.args.status, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("OperateRegisterStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOperateRegisterStudentStatus(t *testing.T) {
	type args struct {
		ctx        context.Context
		tx         pgx.Tx
		ids        []int64
		status     string
		userID     int64
		RegisterID int64
		failReason string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := OperateRegisterStudentStatus(tt.args.ctx, tt.args.tx, tt.args.ids, tt.args.status, tt.args.userID, tt.args.RegisterID, tt.args.failReason); (err != nil) != tt.wantErr {
				t.Errorf("OperateRegisterStudentStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStudentRegister(t *testing.T) {
	type args struct {
		ctx          context.Context
		registerID   int64
		status       string
		RegisterType string
		students     []registerStudentType
		userID       int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StudentRegister(tt.args.ctx, tt.args.registerID, tt.args.status, tt.args.RegisterType, tt.args.students, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("StudentRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateRegister(t *testing.T) {
	type args struct {
		ctx          context.Context
		registration *cmn.TRegisterPlan
		practiceIds  []int64
		userID       int64
		action       string
		reviewers    []int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateRegister(tt.args.ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID, tt.args.action, tt.args.reviewers); (err != nil) != tt.wantErr {
				t.Errorf("UpdateRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertRegister(t *testing.T) {
	type args struct {
		ctx          context.Context
		registration *cmn.TRegisterPlan
		practiceIds  []int64
		userID       int64
		action       string
		reviewers    []int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpsertRegister(tt.args.ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID, tt.args.action, tt.args.reviewers); (err != nil) != tt.wantErr {
				t.Errorf("UpsertRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertRegisterPractice(t *testing.T) {
	type args struct {
		ctx         context.Context
		tx          pgx.Tx
		registerID  int64
		practiceIds []int64
		userID      int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpsertRegisterPractice(tt.args.ctx, tt.args.tx, tt.args.registerID, tt.args.practiceIds, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("UpsertRegisterPractice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertRegisterStudent(t *testing.T) {
	type args struct {
		ctx        context.Context
		registerID int64
		studentIDs []registerStudentType
		userID     int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpsertRegisterStudent(tt.args.ctx, tt.args.registerID, tt.args.studentIDs, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("UpsertRegisterStudent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertReviewers(t *testing.T) {
	type args struct {
		ctx         context.Context
		tx          pgx.Tx
		registerID  int64
		userID      int64
		reviewerIds []int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpsertReviewers(tt.args.ctx, tt.args.tx, tt.args.registerID, tt.args.userID, tt.args.reviewerIds); (err != nil) != tt.wantErr {
				t.Errorf("UpsertReviewers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
