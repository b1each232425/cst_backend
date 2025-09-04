package registration

import (
	"context"
	"github.com/jackc/pgx/v5"
	"reflect"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
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
	reigsterName := "单元测试报名计划名"
	var uid int64
	uid = 10086
	s := `DELETE FROM assessuser.t_register_plan `
	_, err := db.Exec(ctx, s)
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
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正确传入",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddRegister(tt.args.ctx, tt.args.registration, tt.args.practiceIds, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("AddRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
}

func TestDeleteRegisterPracticeStudent(t *testing.T) {
	type args struct {
		ctx         context.Context
		tx          pgx.Tx
		userID      int64
		registerIDs []int64
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
			if err := DeleteRegisterPracticeStudent(tt.args.ctx, tt.args.tx, tt.args.userID, tt.args.registerIDs); (err != nil) != tt.wantErr {
				t.Errorf("DeleteRegisterPracticeStudent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRegisterStudentById(t *testing.T) {
	type args struct {
		ctx          context.Context
		registerID   int64
		message      string
		registerType string
		status       string
		orderBy      []string
		page         int
		pageSize     int
		userID       int64
	}
	tests := []struct {
		name    string
		args    args
		want    []Map
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetRegisterStudentById(tt.args.ctx, tt.args.registerID, tt.args.message, tt.args.registerType, tt.args.status, tt.args.orderBy, tt.args.page, tt.args.pageSize, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRegisterStudentById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRegisterStudentById() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetRegisterStudentById() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestListRegisterS(t *testing.T) {
	type args struct {
		ctx      context.Context
		name     string
		course   string
		status   string
		orderBy  []string
		page     int
		pageSize int
		userID   int64
	}
	tests := []struct {
		name    string
		args    args
		want    []Map
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ListRegisterS(tt.args.ctx, tt.args.name, tt.args.course, tt.args.status, tt.args.orderBy, tt.args.page, tt.args.pageSize, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRegisterS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListRegisterS() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ListRegisterS() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestListRegisterT(t *testing.T) {
	type args struct {
		ctx      context.Context
		name     string
		course   string
		status   string
		orderBy  []string
		page     int
		pageSize int
		userID   int64
	}
	tests := []struct {
		name    string
		args    args
		want    []Map
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ListRegisterT(tt.args.ctx, tt.args.name, tt.args.course, tt.args.status, tt.args.orderBy, tt.args.page, tt.args.pageSize, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRegisterT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListRegisterT() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ListRegisterT() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestListReviewers(t *testing.T) {
	type args struct {
		ctx        context.Context
		userID     int64
		registerID int64
		name       string
		page       int
		pageSize   int
		orderBy    []string
	}
	tests := []struct {
		name    string
		args    args
		want    []Map
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ListReviewers(tt.args.ctx, tt.args.userID, tt.args.registerID, tt.args.name, tt.args.page, tt.args.pageSize, tt.args.orderBy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListReviewers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListReviewers() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ListReviewers() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestLoadRegisterById(t *testing.T) {
	type args struct {
		ctx        context.Context
		registerID int64
	}
	tests := []struct {
		name    string
		args    args
		want    *cmn.TRegisterPlan
		want1   []cmn.TPractice
		want2   []Reviewer
		want3   int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := LoadRegisterById(tt.args.ctx, tt.args.registerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRegisterById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadRegisterById() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("LoadRegisterById() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("LoadRegisterById() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("LoadRegisterById() got3 = %v, want %v", got3, tt.want3)
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
