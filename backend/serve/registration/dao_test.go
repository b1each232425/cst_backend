package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
			authority := &auth_mgt.Authority{
				Domain: cmn.TDomain{
					ID: null.IntFrom(1),
				},
			}

			ctx = context.WithValue(ctx, "authority", authority)
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
				tx, err = db.Begin(ctx)
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				tx, err = db.Begin(ctx)
				if err != nil {
					t.Errorf("OperateRegisterStudentStatus() error = %v", err)
				}
				ctx = context.WithValue(ctx, "force-error", "scand")
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
			if err != nil {
				err := tx.Rollback(ctx)
				if err != nil {
					return
				}
			} else {
				err := tx.Commit(ctx)
				if err != nil {
					return
				}
			}
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
				message:      "",
				registerType: "",
				status:       "",
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
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 报错与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Fatal("预期没有错误但是返回错误")
				}
				if total != 2 {
					z.Error(fmt.Sprintf("返回的total为 %d", total))
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// 添加超时控制，避免无限等待
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
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
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if total != 1 {
					t.Error(fmt.Sprintf("返回的total为 %d", total))
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

func TestListRegisterT(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
			name: "异常1",
			args: args{
				registerID: 1,
				name:       "李玟筱",
				page:       1,
				pageSize:   10,
				orderBy:    []string{"u.create_time"},
				userId:     uid,
				expectErr:  errors.New("查询报名计划失败:"),
			},
		},
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
				expectErr:  errors.New("用户ID错误:"),
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
				expectErr:  errors.New("报名计划ID错误"),
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
		}, {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
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
			s = `DELETE FROM assessuser.t_practice`
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
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
	registerName := "报名计划"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
	}
	s = `DELETE FROM assessuser.t_register_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,course,reviewer_ids) VALUES  ($1 ,$2 ,$3,$4,$5,$6,$7)`
	_, err = db.Exec(ctx, s, 1, registerName, "06", uid, time.Now().UnixMilli(), "00", []int64{10100})
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
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
			name: "异常1",

			args: args{
				registerID: 1,
				expectErr:  errors.New("查询报名计划失败:"),
			},
		},
		{
			name: "正常测试",
			args: args{
				registerID: 1,
				expectErr:  nil,
			},
		},
		{
			name: "异常2",

			args: args{
				registerID: 1,
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
				ctx = context.WithValue(ctx, "force-error", "query3l")
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
			s = `DELETE FROM assessuser.t_register_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			s = `DELETE FROM assessuser.t_practice`
			_, err = db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

		})

	}
}

func TestLoadRegisterByIds(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
	registerName := "报名计划"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
	if err != nil {
		t.Fatal(err.Error())
	}
	// 这里再删除这个练习，随后再重新创建
	s = `DELETE FROM assessuser.t_practice`
	_, err = db.Exec(ctx, s)
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
	s = `INSERT INTO  assessuser.t_register_plan  (id, name , course , review_end_time , max_number , start_time , end_time , reviewer_ids , exam_plan_location ,status,creator,create_time)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err = db.Exec(ctx, s, 1, registerName, "00", time.Now().UnixMilli(), 5, time.Now().UnixMilli()+100000, time.Now().UnixMilli()+100000, []int64{10100}, "广东省", "00", uid, time.Now().UnixMilli())
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
				registerIDs: []int64{1},
				expectErr:   errors.New("查询报名计划失败:"),
			},
		},

		{
			name: "异常2",
			args: args{
				registerIDs: []int64{1},
				expectErr:   errors.New("获取报名计划信息失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始执行测试: %s", tt.name)
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "queryls")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "lsScan")
			} else {
				ctx = context.Background()
			}
			t.Log("调用 LoadRegisterByIds 函数")
			gotRegisters, err := LoadRegisterByIds(ctx, tt.args.registerIDs)
			t.Log("LoadRegisterByIds 函数返回")
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if len(gotRegisters) != 1 {
					z.Error("没有返回数据")
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

func TestLoadRegisterStudentStatusByIds(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}

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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 1, uid, uid, "00", 2, uid, uid, "00")
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
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00", 20023, 1, uid, "04", "00")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ids        []int64
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
				ids:        []int64{20022, 20023},
				registerID: 1,
				expectErr:  nil,
			},
		},
		{
			name: "异常1",
			args: args{
				ids:        []int64{20022, 20023},
				registerID: 1,
				expectErr:  errors.New("查询学生状态失败:"),
			},
		},
		{
			name: "异常2",
			args: args{
				ids:        []int64{20022, 20023},
				registerID: 1,
				expectErr:  errors.New("获取学生状态失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "querylrs")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "lrsScan")
			} else {
				ctx = context.Background()
			}
			gotStudents, err := LoadRegisterStudentStatusByIds(ctx, tt.args.ids, tt.args.registerID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				if len(gotStudents) != 2 {
					z.Error("返回数据长度错误")
				}
				if gotStudents[0].Status.String != "02" || gotStudents[1].Status.String != "04" {
					z.Error("返回数据错误")
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

func TestMoveStudent(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}

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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}

	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00", 20023, 1, uid, "04", "00")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		fromRegisterID int64
		toRegisterID   int64
		students       []registerStudentType
		status         string
		userID         int64
		expectErr      error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status: "08",
				userID: uid,
			},
		},
		{
			name: "异常测试缺少报名计划id",
			args: args{
				fromRegisterID: 0,
				toRegisterID:   0,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("报名计划ID错误"),
			},
		},

		{
			name: "异常测试userid不合法",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    0,
				expectErr: errors.New("用户ID错误"),
			},
		},
		{
			name: "异常测试目标报名计划状态不为已发布",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   3,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("目标报名计划状态为：04，不能移动学生"),
			},
		},
		{
			name: "异常测试移动人数超过目标报名计划可容纳人数",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   4,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("目标报名计划可容纳人数不足，无法进行移动 ,剩余人数为: 1"),
			},
		},
		{
			name: "异常1",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("查询报名计划下的审查人失败:"),
			},
		}, {
			name: "异常2",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("开启事务失败:"),
			},
		},
		//{
		//	name: "异常3",
		//	args: args{
		//		fromRegisterID: 1,
		//		toRegisterID:   2,
		//		students: []registerStudentType{
		//			{
		//				StudentID: 20022,
		//				ExamType:  "00",
		//			},
		//			{
		//				StudentID: 20023,
		//				ExamType:  "00",
		//			},
		//		},
		//		status:    "08",
		//		userID:    uid,
		//		expectErr: errors.New("回滚失败"),
		//	},
		//},
		//{
		//	name: "异常4",
		//	args: args{
		//		fromRegisterID: 1,
		//		toRegisterID:   2,
		//		students: []registerStudentType{
		//			{
		//				StudentID: 20022,
		//				ExamType:  "00",
		//			},
		//			{
		//				StudentID: 20023,
		//				ExamType:  "00",
		//			},
		//		},
		//		status:    "08",
		//		userID:    uid,
		//		expectErr: errors.New("触发commit"),
		//	},
		//},
		{
			name: "异常5",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("批量移动学生参数处理失败:"),
			},
		},
		{
			name: "异常6",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("批量移动学生失败:"),
			},
		},
		{
			name: "异常7",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("delete PracticeStudent call failed:"),
			},
		},
		{
			name: "异常8",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("设置迁移学生状态失败:"),
			},
		},

		{
			name: "异常一零",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("扫描已删除的练习ID失败:"),
			},
		},
		{
			name: "异常一一",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("获取学生状态失败:"),
			},
		},
		{
			name: "异常一二",
			args: args{
				fromRegisterID: 1,
				toRegisterID:   2,
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				status:    "08",
				userID:    uid,
				expectErr: errors.New("扫描学生状态失败 错误: "),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query3l")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "begin")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "rollbackm")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "commitm")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "sqlxIn")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "pQuery")
			} else if strings.Contains(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "query3")
			} else if strings.Contains(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "opQuery")
			} else if strings.Contains(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常一零") {
				ctx = context.WithValue(ctx, "force-error", "scand")
			} else if strings.Contains(tt.name, "异常一一") {
				ctx = context.WithValue(ctx, "force-error", "queryMove")
			} else if strings.Contains(tt.name, "异常一二") {
				ctx = context.WithValue(ctx, "force-error", "scanMove")
			} else {
				ctx = context.Background()
			}
			err = MoveStudent(ctx, tt.args.fromRegisterID, tt.args.toRegisterID, tt.args.students, tt.args.status, tt.args.userID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					z.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				s = `SELECT COUNT(*) FROM assessuser.t_exam_plan_student eps WHERE eps.register_id = $1`
				var count int
				err = db.QueryRow(ctx, s, tt.args.fromRegisterID).Scan(&count)
				if err != nil {
					t.Error(err)
				}
				if count != 2 {
					z.Error("返回数据长度错误")
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

func TestOperateRegisterStatus(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}

	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}

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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number,end_time,start_time) VALUES  ($1 ,$2 ,$3,$4,$5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0, time.Now().UnixMilli()+1000000000, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "02", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00", 20023, 2, uid, "04", "00")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerIDs []int64
		status      string
		userID      int64
		expectErr   error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试发布",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      uid,
				expectErr:   nil,
			},
		},
		{
			name: "正常测试作废",
			args: args{
				registerIDs: []int64{3},
				status:      "08",
				userID:      uid,
				expectErr:   nil,
			},
		},
		{
			name: "正常测试删除",
			args: args{
				registerIDs: []int64{5},
				status:      "10",
				userID:      uid,
				expectErr:   nil,
			},
		},
		{
			name: "测试报名id为空",
			args: args{
				registerIDs: []int64{},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("registerIDs不能为空"),
			},
		},
		{
			name: "userid小于0",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      -1,
				expectErr:   errors.New("userID不能小于等于0"),
			},
		},
		{
			name: "测试报名计划状态不一样无法进行批量操作",
			args: args{
				registerIDs: []int64{1, 2},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("此时要批量操作的报名计划状态不一，无法进行批量操作"),
			},
		},
		{
			name: "测试报名计划已作废，无法进行操作",
			args: args{
				registerIDs: []int64{4},
				status:      "02",
				userID:      uid,
				expectErr:   errors.New("此时要批量操作的报名计划状态不一，无法进行批量操作"),
			},
		},
		{
			name: "测试要删除的报名计划已经有学生报名无法删除",
			args: args{
				registerIDs: []int64{2},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("此时要批量操作的报名计划状态不一，无法进行批量操作"),
			},
		},
		{
			name: "异常1",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("查询报名计划失败:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("beginTx called failed:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("更新报名计划状态失败:"),
			},
		},
		{
			name: "异常4",
			args: args{
				registerIDs: []int64{1},
				status:      "08",
				userID:      uid,
				expectErr:   errors.New("更新报名计划状态失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registerIDs: []int64{1},
				status:      "08",
				userID:      uid,
				expectErr:   errors.New("更新报名计划状态失败:"),
			},
		},
		{
			name: "异常6",
			args: args{
				registerIDs: []int64{1},
				status:      "08",
				userID:      uid,
				expectErr:   errors.New("更新报名计划状态失败:"),
			},
		},
		{
			name: "异常7",
			args: args{
				registerIDs: []int64{1},
				status:      "08",
				userID:      uid,
				expectErr:   errors.New("扫描已删除的练习ID失败:"),
			},
		},
		{
			name: "异常8",
			args: args{
				registerIDs: []int64{1},
				status:      "08",
				userID:      uid,
				expectErr:   errors.New("取消报名计划定时器失败:"),
			},
		},
		{
			name: "异常9",
			args: args{
				registerIDs: []int64{2},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("查询报名计划是否被使用失败:"),
			},
		},
		{
			name: "异常一零",
			args: args{
				registerIDs: []int64{6},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("删除报名计划失败:"),
			},
		},
		{
			name: "异常一一",
			args: args{
				registerIDs: []int64{6},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("更新关联的练习失败:"),
			},
		},
		{
			name: "异常一二",
			args: args{
				registerIDs: []int64{6},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("查询已删除的练习ID失败:"),
			},
		},
		{
			name: "异常一三",
			args: args{
				registerIDs: []int64{6},
				status:      "10",
				userID:      uid,
				expectErr:   errors.New("取消报名计划定时器失败:"),
			},
		},
		{
			name: "异常一四",
			args: args{
				registerIDs: []int64{1},
				status:      "00",
				userID:      uid,
				expectErr:   errors.New("设置报名计划定时器失败:"),
			},
		}, {
			name: "测试操作其他的状态",
			args: args{
				registerIDs: []int64{7},
				status:      "12",
				userID:      uid,
				expectErr:   errors.New("异常状态， 无法操作"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "queryls")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "operate")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "operate1")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "operate2")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "operate3")
			} else if strings.Contains(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "scand")
			} else if strings.Contains(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "cancelRegisterTimers1")
			} else if strings.Contains(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "queryIsUsed")
			} else if strings.Contains(tt.name, "异常一零") {
				ctx = context.WithValue(ctx, "force-error", "operateRegister")
			} else if strings.Contains(tt.name, "异常一一") {
				ctx = context.WithValue(ctx, "force-error", "tx.UpdateRegisterPractice")
			} else if strings.Contains(tt.name, "异常一二") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常一三") {
				ctx = context.WithValue(ctx, "force-error", "cancelTimers")
			} else if strings.Contains(tt.name, "异常一四") {
				ctx = context.WithValue(ctx, "force-error", "setTimers")
			} else {
				ctx = context.Background()
			}
			if strings.Contains(tt.name, "测试报名id为空") || strings.Contains(tt.name, "userid小于0") {

			} else {
				_, cancel := context.WithCancel(context.Background())
				registerTimerManager = NewRegistrationTimerManager(ctx, cancel)
				err = SetRegisterTimers(ctx, tt.args.registerIDs[0])
				if err != nil {
					t.Error(fmt.Sprintf("启动报名计划定时器失败 %v", err))
				}
			}

			// 添加数据库连接池检查
			stat := db.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
			}
			// 原有的测试逻辑
			err = OperateRegisterStatus(ctx, tt.args.registerIDs, tt.args.status, tt.args.userID)
			// 添加数据库连接池检查
			stat = db.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
			}
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error(fmt.Sprintf("预期没有错误但是实际上有错误 %v", err))
				}
				s = `SELECT status FROM  assessuser.t_register_plan WHERE id = $1`
				var status string
				err := db.QueryRow(ctx, s, tt.args.registerIDs[0]).Scan(&status)
				if err != nil {
					t.Fatal(err.Error())
				}
				if status != tt.args.status {
					t.Logf(fmt.Sprintf("状态错误，预期是%s 实际是%s", tt.args.status, status))
				}

			}
		})
		t.Cleanup(func() {

			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice`
			_, err = db.Exec(ctx, s)
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
func TestOperateRegisterStudentStatus(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "02", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00", 20023, 1, uid, "04", "00")
	type args struct {
		tx         pgx.Tx
		ids        []int64
		status     string
		userID     int64
		RegisterID int64
		failReason string
		expectErr  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试没有事务情况",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "06",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  nil,
			},
		},
		{
			name: "正常测试有事务情况",
			args: args{
				tx:         tx,
				ids:        []int64{20022},
				status:     "06",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  nil,
			},
		},
		{
			name: "测试status为空",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("请选择操作"),
			},
		},
		{
			name: "测试UserId不合法",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     0,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("请选择操作"),
			},
		},
		{
			name: "测试报名计划id不合法",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 0,
				failReason: "",
				expectErr:  fmt.Errorf("请选择操作"),
			},
		},
		{
			name: "测试报名计划已作废",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 4,
				failReason: "",
				expectErr:  fmt.Errorf("当前报名计划状态为：08，不能进行操作"),
			},
		},
		{
			name: "测试要操作的学生的状态不一样",
			args: args{
				tx:         nil,
				ids:        []int64{20022, 20023},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("此时要批量操作的学生状态不一，无法进行批量操作"),
			},
		},
		{
			name: "测试没有选择学生",
			args: args{
				tx:         nil,
				ids:        []int64{},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("请选择学生"),
			},
		},
		{
			name: "测试移动学生",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "08",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  nil,
			},
		},
		{
			name: "异常1",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("查询报名计划失败"),
			},
		},
		{
			name: "异常2",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("查询学生状态失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				tx:         tx,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("设置迁移学生状态失败:")},
		},
		{
			name: "异常4",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("开启事务失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				tx:         nil,
				ids:        []int64{20022},
				status:     "04",
				userID:     uid,
				RegisterID: 1,
				failReason: "",
				expectErr:  fmt.Errorf("更新学生状态失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "querylrs")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "opQuery")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "begin")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "upQuery")
			}
			if tt.args.tx != nil {
				tt.args.tx, _ = db.Begin(ctx)
			}
			// 添加数据库连接池检查
			stat := db.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				t.Logf(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
			}
			err := OperateRegisterStudentStatus(ctx, tt.args.tx, tt.args.ids, tt.args.status, tt.args.userID, tt.args.RegisterID, tt.args.failReason)
			// 添加数据库连接池检查
			stat = db.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				t.Logf(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
			}
			if tt.args.tx != nil {
				if err != nil {
					z.Error(err.Error())
					err := tt.args.tx.Rollback(ctx)
					if err != nil {
						return
					}
				} else {
					err := tt.args.tx.Commit(ctx)
					if err != nil {
						return
					}
				}
			}
			// 添加数据库连接池检查
			stat = db.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				t.Logf(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
			}
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					z.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Fatal("预期没错误但返回错误")
				}
				s = `SELECT status FROM assessuser.t_exam_plan_student WHERE register_id = $1 `
				var status string
				err = db.QueryRow(ctx, s, tt.args.RegisterID).Scan(&status)
				if err != nil {
					t.Fatal(err)
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

func TestStudentRegister(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "00", "00")
	type args struct {
		registerID   int64
		status       string
		RegisterType string
		students     []registerStudentType
		userID       int64
		expectErr    error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "正常测试2",
			args: args{
				registerID:   2,
				status:       "00",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "测试报名状态不处于已发布无法报名",
			args: args{
				registerID:   3,
				status:       "00",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("报名计划状态不处于已发布，无法报名"),
			},
		},
		{
			name: "测试报名计划已满人",
			args: args{
				registerID:   2,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
					{
						StudentID: 20023,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("报名计划人数已满"),
			},
		},
		{
			name: "异常1",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("beginTx called failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("query register failed:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("query student failed:"),
			},
		},
		{
			name: "异常4",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("查询报名计划失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registerID:   1,
				status:       "02",
				RegisterType: "00",
				students: []registerStudentType{
					{
						StudentID: 20022,
						ExamType:  "00",
					},
				},
				userID:    uid,
				expectErr: fmt.Errorf("更新当前报名信息失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "querysr")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "queryss")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "exec")
			} else {
				ctx = context.Background()
			}
			err = StudentRegister(ctx, tt.args.registerID, tt.args.status, tt.args.RegisterType, tt.args.students, tt.args.userID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没有错误但实际报错")
				}
				s = `SELECT COUNT(*) FROM  assessuser.t_exam_plan_student WHERE status IN ($1,$2) AND register_id =$3 `
				var count int
				err = db.QueryRow(ctx, s, RegisterStudentStatus.Apply, RegisterStudentStatus.Pending, tt.args.registerID).Scan(&count)
				if err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Error("预期报名计划学生数量为1，实际为：", count)
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

func TestUpdateRegister(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "00", "00")
	type args struct {
		registration *cmn.TRegisterPlan
		practiceIds  []int64
		userID       int64
		action       string
		reviewers    []int64
		expectErr    error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID: uid,
			},
		},
		{
			name: "测试用户ID小于等于0",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    0,
				expectErr: errors.New("用户ID不能小于等于0"),
			},
		},
		{
			name: "测试报名计划id不合法",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 0,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				expectErr: errors.New("报名计划ID不合法"),
			},
		},
		{
			name: "测试清空审核人",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				action:      "clearr",
				reviewers:   []int64{},
				practiceIds: []int64{1},
			},
		},
		{
			name: "测试清空练习",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				action:    "clearp",
				reviewers: []int64{1},
			},
		},
		{
			name: "测试清空全部",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID: uid,
				action: "clear",
			},
		},
		{
			name: "测试只修改审核者",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				reviewers: []int64{1},
			},
		},
		{
			name: "测试只修改练习",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{1},
			},
		},
		{
			name: "测试练习和审核者的都修改",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{1},
				reviewers:   []int64{1},
			},
		},
		{
			name: "异常1",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{1},
				expectErr:   errors.New("beginTx called failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{1},
				expectErr:   errors.New("updateRegister call failed"),
			},
		},
		{
			name: "异常3",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				action:      "clearr",
				reviewers:   []int64{},
				practiceIds: []int64{1},
				expectErr:   errors.New("更新审核人失败:"),
			},
		},
		{
			name: "异常4",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				action:      "clearr",
				reviewers:   []int64{},
				practiceIds: []int64{1},
				expectErr:   errors.New("添加名单失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				action:    "clearp",
				reviewers: []int64{1},
				expectErr: errors.New("更新报名计划下的所有审核人失败"),
			},
		},
		{
			name: "异常6",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				reviewers: []int64{1},
				action:    "clearp",
				expectErr: errors.New("删除报名计划下的所有练习失败:"),
			},
		},
		{
			name: "异常7",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				action:    "clear",
				userID:    uid,
				expectErr: errors.New("更新报名计划下的所有审核人失败:"),
			},
		},
		{
			name: "异常8",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				action:    "clear",
				userID:    uid,
				expectErr: errors.New("删除报名计划下的所有练习失败:"),
			},
		},
		{
			name: "异常9",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:    uid,
				reviewers: []int64{1},
				expectErr: errors.New("更新报名计划下的所有审核人失败:"),
			},
		},
		{
			name: "异常一零",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("添加名单失败:"),
			},
		},
		{name: "异常一一",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				reviewers:   []int64{1},
				practiceIds: []int64{1},
				expectErr:   errors.New("更新报名计划下的所有审核人失败:"),
			},
		},
		{
			name: "异常一二",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{1},
				reviewers:   []int64{1},
				expectErr:   errors.New("添加名单失败:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "queryRegisterf")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "delp")
			} else if strings.Contains(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "delp")
			} else if strings.Contains(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常一零") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常一一") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else if strings.Contains(tt.name, "异常一二") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else {
				ctx = context.Background()
			}
			err = UpdateRegister(ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID, tt.args.action, tt.args.reviewers)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没有错误但是实际报错")
				}
				s = `SELECT name FROM assessuser.t_register_plan WHERE id =$1`
				var name string
				err = db.QueryRow(ctx, s, tt.args.registration.ID.Int64).Scan(&name)
				if err != nil {
					t.Fatal(err)
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

func TestUpsertRegister(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "00", "00")
	type args struct {
		registration *cmn.TRegisterPlan
		practiceIds  []int64
		userID       int64
		action       string
		reviewers    []int64
		expectErr    error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试有id",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
			},
		},
		{
			name: "正常测试无id",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
			},
		},
		{
			name: "测试报名计划已作废无法修改",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 4,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				userID:      uid,
				practiceIds: []int64{uid},
				expectErr:   errors.New("报名计划状态为"),
			},
		},
		{
			name: "测试缺少用户id",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				expectErr:   errors.New("用户ID不能小于等于0"),
			},
		},
		{
			name: "异常1",
			args: args{
				registration: &cmn.TRegisterPlan{
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
				expectErr:   errors.New("beginTx called failed:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 1,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
				expectErr:   errors.New("查询报名计划失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 2,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
				expectErr:   errors.New("updateRegister call failed:"),
			},
		},
		{
			name: "异常4",
			args: args{
				registration: &cmn.TRegisterPlan{
					ID: null.Int{
						sql.NullInt64{
							Int64: 2,
							Valid: true,
						},
					},
					Name: null.String{
						sql.NullString{
							String: "报名计划1",
							Valid:  true,
						},
					},
				},
				practiceIds: []int64{uid},
				userID:      uid,
				expectErr:   errors.New("查询报名计划信息错误"),
			},
		},
	}
	for _, tt := range tests {
		if strings.Contains(tt.name, "异常1") {
			ctx = context.WithValue(ctx, "force-error", "beginTx")
		} else if strings.Contains(tt.name, "异常2") {
			ctx = context.WithValue(ctx, "force-error", "query")
		} else if strings.Contains(tt.name, "异常3") {
			ctx = context.WithValue(ctx, "force-error", "queryRegisterf")
		} else if strings.Contains(tt.name, "异常4") {
			ctx = context.WithValue(ctx, "force-error", "queryRegisterPlan")
		} else {
			ctx = context.Background()
		}
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "有id") {
				_, cancel := context.WithCancel(context.Background())
				registerTimerManager = NewRegistrationTimerManager(ctx, cancel)
				err = SetRegisterTimers(ctx, tt.args.registration.ID.Int64)
				if err != nil {
					t.Error(fmt.Sprintf("启动报名计划定时器失败 %v", err))
				}
			}
			err = UpsertRegister(ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID, tt.args.action, tt.args.reviewers)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Fatal("预期没错误但是报错")
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

func TestUpsertRegisterPractice(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_exam_plan_student (student_id , register_id,creator,status,type)VALUES($1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 20022, 1, uid, "02", "00")
	type args struct {
		tx          pgx.Tx
		registerID  int64
		practiceIds []int64
		userID      int64
		expectErr   error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
			},
		},
		{
			name: "正常测试清空所有练习",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{},
				userID:      uid,
			},
		},
		{
			name: "测试用户id为0",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      0,
			},
		},
		{
			name: "测试register_id为0",
			args: args{
				tx:          nil,
				registerID:  0,
				practiceIds: []int64{1},
				userID:      uid,
			},
		},
		{
			name: "异常1",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{},
				userID:      uid,
				expectErr:   errors.New("删除报名计划下的所有练习失败:"),
			},
		},
		{
			name: "异常2",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{},
				userID:      uid,
				expectErr:   errors.New("扫描已删除的练习ID失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("添加名单参数错误:"),
			},
		},
		{
			name: "异常4",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("添加名单失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("查询与报名计划相关联的学生失败:"),
			},
		},
		{
			name: "异常6",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("扫描与报名计划相关联的学生失败:"),
			},
		},
		{
			name: "异常7",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("delete PracticeStudent call failed:"),
			},
		},
		{
			name: "异常8",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("删除名单失败"),
			},
		},
		{
			name: "异常9",
			args: args{
				tx:          nil,
				registerID:  1,
				practiceIds: []int64{1},
				userID:      uid,
				expectErr:   errors.New("扫描已删除的练习ID失败"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "delp")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "scand")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "sqlxInup")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "query2")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "query1")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "scanrstudent")
			} else if strings.Contains(tt.name, "异常7") {
				ctx = context.WithValue(ctx, "force-error", "query3")
			} else if strings.Contains(tt.name, "异常8") {
				ctx = context.WithValue(ctx, "force-error", "query3p")
			} else if strings.Contains(tt.name, "异常9") {
				ctx = context.WithValue(ctx, "force-error", "scand")
			} else {
				ctx = context.Background()
			}
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			tt.args.tx = tx
			err = UpsertRegisterPractice(ctx, tt.args.tx, tt.args.registerID, tt.args.practiceIds, tt.args.userID)
			if err != nil {
				tx.Rollback(ctx)
			}
			err = tx.Commit(ctx)
			if err != nil {
				return
			}
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但是返回错误")
				}
				s = `SELECT COUNT(*) FROM assessuser.t_register_practice WHERE register_id =$1 AND status = $2`
				var count int
				err = db.QueryRow(ctx, s, tt.args.registerID, RegisterPracticeStatus.Normal).Scan(&count)
				if err != nil {
					t.Fatal(err)
				}
				if count != len(tt.args.practiceIds) {
					t.Error(fmt.Sprintf("返回的报名计划练习数量与预期不一致 %v , %v", count, len(tt.args.practiceIds)))
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

func TestUpsertRegisterStudent(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		registerID int64
		studentIDs []registerStudentType
		userID     int64
		expectErr  error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常测试",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: nil,
			},
		},
		{
			name: "测试用户id为0",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    0,
				expectErr: fmt.Errorf("用户ID错误:"),
			},
		},
		{
			name: "测试registerID为0",
			args: args{
				registerID: 0,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("报名计划ID错误:"),
			},
		},
		{
			name: "测试超过报名计划可容纳人数",
			args: args{
				registerID: 2,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}, {
					StudentID: 20024,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("导入学生人数超出报名计划可容纳人数, 剩余人数为:"),
			},
		},
		{
			name: "测试报名计划不处于已发布状态",
			args: args{
				registerID: 3,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("当前报名计划状态为："),
			},
		},
		{
			name: "异常1",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("查询报名计划失败:"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("开启事务失败:"),
			},
		},
		{
			name: "异常3",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("批量导入学生参数处理失败"),
			},
		},
		{
			name: "异常4",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("批量导入学生失败:"),
			},
		},
		{
			name: "异常5",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("查询报名计划下的所有练习失败:"),
			},
		},
		{
			name: "异常6",
			args: args{
				registerID: 1,
				studentIDs: []registerStudentType{{
					StudentID: 20022,
					ExamType:  "00",
				}, {
					StudentID: 20023,
					ExamType:  "00",
				}},
				userID:    uid,
				expectErr: fmt.Errorf("beginTx called failed:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "query")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "begin")
			} else if strings.Contains(tt.name, "异常3") {
				ctx = context.WithValue(ctx, "force-error", "sqlxIn")
			} else if strings.Contains(tt.name, "异常4") {
				ctx = context.WithValue(ctx, "force-error", "pQuery")
			} else if strings.Contains(tt.name, "异常5") {
				ctx = context.WithValue(ctx, "force-error", "loadRegister")
			} else if strings.Contains(tt.name, "异常6") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else {
				ctx = context.Background()
			}
			err = UpsertRegisterStudent(ctx, tt.args.registerID, tt.args.studentIDs, tt.args.userID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但是返回错误")
				}
				s = `SELECT COUNT(*) FROM assessuser.t_exam_plan_student WHERE register_id = $1 `
				var count int
				err = db.QueryRow(ctx, s, tt.args.registerID).Scan(&count)
				if err != nil {
					t.Fatal(err)
				}
				if count != len(tt.args.studentIDs) {
					t.Error("返回数量与预期不符")
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

func TestUpsertReviewers(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	db := cmn.GetPgxConn()
	// 添加数据库连接池检查
	stat := db.Stat()
	// 检查是否有空闲连接
	if stat.IdleConns() > 0 {
		t.Logf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
			stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns())
	}
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
	s = `INSERT INTO t_practice_student (student_id , practice_id,creator,status)VALUES($1,$2,$3,$4),($5,$6,$7,$8)`
	_, err = db.Exec(ctx, s, 20022, uid, uid, "00", 20023, uid, uid, "00")
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
	s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
	_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
	_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		tx          pgx.Tx
		registerID  int64
		userID      int64
		reviewerIds []int64
		expectErr   error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				tx:          nil,
				registerID:  1,
				userID:      uid,
				reviewerIds: []int64{20022, 20023},
			},
		},
		{
			name: "测试报名计划id为零",
			args: args{
				tx:          nil,
				registerID:  0,
				userID:      uid,
				reviewerIds: []int64{20022, 20023},
				expectErr:   fmt.Errorf("registerID不能小于等于0"),
			},
		},
		{
			name: "测试用户id为零",
			args: args{
				tx:          nil,
				registerID:  1,
				userID:      0,
				reviewerIds: []int64{20022, 20023},
				expectErr:   fmt.Errorf("userID不能小于等于0"),
			},
		},
		{
			name: "异常1",
			args: args{
				tx:          nil,
				registerID:  1,
				userID:      uid,
				reviewerIds: []int64{20022, 20023},
				expectErr:   fmt.Errorf("更新报名计划下的所有审核人失败:"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "del")
			} else {
				ctx = context.Background()
			}
			tx, err := db.Begin(ctx)
			if err != nil {
				return
			}
			tt.args.tx = tx
			err = UpsertReviewers(ctx, tt.args.tx, tt.args.registerID, tt.args.userID, tt.args.reviewerIds)
			if err != nil {
				tx.Rollback(ctx)
			} else {
				tx.Commit(ctx)
			}
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但是报错")
				}
				s = `SELECT reviewer_ids FROM assessuser.t_register_plan WHERE id =$1 `
				var reviewerIds []int64
				rows, err := db.Query(ctx, s, tt.args.registerID)
				if err != nil {
					t.Fatal(err)
				}
				for rows.Next() {
					err = rows.Scan(&reviewerIds)
					if err != nil {
						t.Fatal(err)
					}
				}
				if len(reviewerIds) != len(tt.args.reviewerIds) {
					t.Error(fmt.Sprintf("审核人id数量不一致 %v ,%v", len(reviewerIds), len(tt.args.reviewerIds)))
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
