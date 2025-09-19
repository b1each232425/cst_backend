package registration

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"w2w.io/cmn"
)

func TestCancelRegisterTimers(t *testing.T) {
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
				registerID: 2,
				expectErr:  fmt.Errorf("查询报名计划信息错误"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "queryRegisterPlan")
			} else {
				ctx = context.Background()
			}
			_, cancel := context.WithCancel(context.Background())
			registerTimerManager = NewRegistrationTimerManager(ctx, cancel)
			SetRegisterTimers(ctx, tt.args.registerID)
			err = CancelRegisterTimers(ctx, tt.args.registerID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但报错")
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

func TestEnroll(t *testing.T) {
	type args struct {
		author string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Enroll(tt.args.author)
		})
	}
}

func TestInitializeRegisterTimers(t *testing.T) {
	type args struct {
		ctx context.Context
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
			if err := InitializeRegisterTimers(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("InitializeRegisterTimers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewRegistrationTimerManager(t *testing.T) {
	type args struct {
		ctx    context.Context
		cancel context.CancelFunc
	}
	tests := []struct {
		name string
		args args
		want *RegistrationTimerManager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRegistrationTimerManager(tt.args.ctx, tt.args.cancel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRegistrationTimerManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegisterMaintainService(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterMaintainService()
		})
	}
}

func TestRegistrationTimerManager_CancelTimer(t *testing.T) {
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	type args struct {
		eventType  string
		registerID int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			tm.CancelTimer(tt.args.eventType, tt.args.registerID)
		})
	}
}

func TestRegistrationTimerManager_SetTimer(t *testing.T) {
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	type args struct {
		registerID  int64
		triggerTime int64
		event       RegisterEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			tm.SetTimer(tt.args.registerID, tt.args.triggerTime, tt.args.event)
		})
	}
}

func TestRegistrationTimerManager_StopAll(t *testing.T) {
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			tm.StopAll()
		})
	}
}

func TestRegistrationTimerManager_processEvent(t *testing.T) {
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	type args struct {
		event    RegisterEvent
		workerID int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			if err := tm.processEvent(tt.args.event, tt.args.workerID); (err != nil) != tt.wantErr {
				t.Errorf("processEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistrationTimerManager_startEventWorkers(t *testing.T) {
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			tm.startEventWorkers()
		})
	}
}

func TestSetRegisterTimers(t *testing.T) {
	type args struct {
		ctx        context.Context
		registerID int64
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
			if err := SetRegisterTimers(tt.args.ctx, tt.args.registerID); (err != nil) != tt.wantErr {
				t.Errorf("SetRegisterTimers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleRegisterEndEvent(t *testing.T) {
	type args struct {
		ctx   context.Context
		event RegisterEvent
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
			if err := handleRegisterEndEvent(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("handleRegisterEndEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleRegisterReviewEndEvent(t *testing.T) {
	type args struct {
		ctx   context.Context
		event RegisterEvent
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
			if err := handleRegisterReviewEndEvent(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("handleRegisterReviewEndEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_register(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			register(tt.args.ctx)
		})
	}
}

func Test_registerReviewer(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerReviewer(tt.args.ctx)
		})
	}
}

func Test_registerStudentH(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerStudentH(tt.args.ctx)
		})
	}
}
