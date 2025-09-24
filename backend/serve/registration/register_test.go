package registration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
)

var (
	testDomainApis = []struct {
		ApiPath      string
		AccessAction string
	}{}
	permissions = []string{
		auth_mgt.CAPIAccessActionCreate,
		auth_mgt.CAPIAccessActionRead,
		auth_mgt.CAPIAccessActionUpdate,
		auth_mgt.CAPIAccessActionDelete,
	}
)

func addTestDomainApi(apiPath string, accessAction string, domain int64) (err error) {

	dbConn := cmn.GetDbConn()

	tx, err := dbConn.Begin()
	if err != nil {
		return
	}

	defer func() {

		if err != nil {
			tx.Rollback()
			return
		}

		err = tx.Commit()

	}()

	s := `INSERT INTO t_domain_api (api, domain, data_access_mode)
	SELECT id, $2, $3 FROM t_api WHERE expose_path = $1 AND access_action = $4
	ON CONFLICT DO NOTHING`

	r, err := tx.Exec(s, apiPath, domain, "full", accessAction)
	if err != nil {
		return
	}

	c, err := r.RowsAffected()
	if err != nil {
		return
	}

	if c == 0 {
		return
	}

	// record for cleanup
	testDomainApis = append(testDomainApis, struct {
		ApiPath      string
		AccessAction string
	}{ApiPath: apiPath, AccessAction: accessAction})

	return
}

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
	s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
	_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
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
func TestInitializeRegisterTimers(t *testing.T) {
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
	_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
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
		expectErr error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				expectErr: nil,
			},
		},
		{
			name: "异常1",
			args: args{
				expectErr: errors.New("查询报名计划信息错误"),
			},
		},
		{
			name: "异常2",
			args: args{
				expectErr: errors.New("扫描报名计划信息错误"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "queryRegisterPlan")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "scanRegisterPlan")
			}
			err := InitializeRegisterTimers(ctx)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Errorf("InitializeRegisterTimers() error = %v, wantErr %v", err, tt.args.expectErr)
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

func TestNewRegistrationTimerManager(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	_, cancel := context.WithCancel(context.Background())
	ctx = context.Background()
	type args struct {
		ctx    context.Context
		cancel context.CancelFunc
	}
	tests := []struct {
		name string
		args args
		want *RegistrationTimerManager
	}{
		{
			name: "正常测试",
			args: args{
				ctx:    ctx,
				cancel: cancel,
			},
			want: &RegistrationTimerManager{
				timers:     make(map[string]*time.Timer),
				ctx:        ctx,
				cancel:     cancel,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRegistrationTimerManager(tt.args.ctx, tt.args.cancel)
			if got.eventQueue == nil {
				t.Error("eventQueue is nil")
			}
		})
	}
}

func TestRegisterMaintainService(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
	tests := []struct {
		name string
	}{
		{
			name: "正常测试",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterMaintainService()
		})
	}
}

func TestRegistrationTimerManager_CancelTimer(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
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
		{
			name: "正常测试",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        context.Background(),
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				eventType:  EVENT_TYPE_REGISTER_END,
				registerID: 1,
			},
		},
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
			tm.SetTimer(tt.args.registerID, time.Now().UnixMilli()+10000, RegisterEvent{
				RegisterID: 1,
				Type:       EVENT_TYPE_REGISTER_END,
			})

			tm.CancelTimer(tt.args.eventType, tt.args.registerID)
		})
	}
}

func TestRegistrationTimerManager_SetTimer(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
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
		{
			name: "正常测试",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        context.Background(),
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				registerID:  1,
				triggerTime: time.Now().UnixMilli() + 10000,
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
			},
		},
		{
			name: "正常测试2",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        context.Background(),
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				registerID:  1,
				triggerTime: time.Now().UnixMilli() + 1,
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
			},
		},
		{
			name: "正常测试3",
			fields: fields{
				timers: make(map[string]*time.Timer),
				mutex:  sync.Mutex{},
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					// 立即取消context
					cancel()
					return ctx
				}(),
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				registerID:  2,
				triggerTime: time.Now().UnixMilli() + 10000,
				event: RegisterEvent{
					RegisterID: 2,
					Type:       EVENT_TYPE_REGISTER_END,
				},
			},
		},
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
			tm.SetTimer(tt.args.registerID, tt.args.triggerTime, tt.args.event)
			// 给一些时间让定时器执行
			time.Sleep(100 * time.Millisecond)
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
	_, cancel := context.WithCancel(context.Background())
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "正常测试",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        context.Background(),
				cancel:     cancel,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
		},
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
	if z == nil {
		cmn.ConfigureForTest()
	}
	type fields struct {
		timers     map[string]*time.Timer
		mutex      sync.Mutex
		ctx        context.Context
		cancel     context.CancelFunc
		eventQueue chan RegisterEvent
		maxWorkers int
	}
	type args struct {
		event     RegisterEvent
		workerID  int
		expectErr error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "正常测试1",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
				expectErr: nil,
			},
		},
		{
			name: "正常测试2",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_START,
				},
				expectErr: nil,
			},
		},
		{
			name: "正常测试3",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_REVIEW_END,
				},
				expectErr: nil,
			},
		},
		{
			name: "正常测试4",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       "",
				},
				expectErr: nil,
			},
		},
		{
			name: "异常1",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
				expectErr: errors.New("exec failed:"),
			},
		},
		{
			name: "异常2",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_REVIEW_END,
				},
				expectErr: errors.New("exec failed:"),
			},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "exec")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "exec")
			}
			tm := &RegistrationTimerManager{
				timers:     tt.fields.timers,
				mutex:      tt.fields.mutex,
				ctx:        ctx,
				cancel:     tt.fields.cancel,
				eventQueue: tt.fields.eventQueue,
				maxWorkers: tt.fields.maxWorkers,
			}
			err := tm.processEvent(tt.args.event, tt.args.workerID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但实际报错")
				}
			}
		})
	}
}

func TestRegistrationTimerManager_startEventWorkers(t *testing.T) {
	if z == nil {
		cmn.ConfigureForTest()
	}
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
		{
			name: "正常测试",
			fields: fields{
				timers:     make(map[string]*time.Timer),
				mutex:      sync.Mutex{},
				ctx:        ctx,
				cancel:     nil,
				eventQueue: make(chan RegisterEvent, 5*15),
				maxWorkers: 5,
			},
		},
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
	_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
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
				registerID: 1,
				expectErr:  errors.New("查询报名计划信息错误"),
			},
		},
		{
			name: "异常2",
			args: args{
				registerID: 1,
				expectErr:  errors.New("获取报名计划信息错误"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "queryRegisterPlan")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "scanRegisterPlanInfo")
			} else {
				ctx = context.Background()
			}
			_, cancel := context.WithCancel(ctx)
			registerTimerManager = NewRegistrationTimerManager(ctx, cancel)
			err := SetRegisterTimers(ctx, tt.args.registerID)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Error(fmt.Sprintf("%s 错误与预期：  %s", err.Error(), tt.args.expectErr.Error()))
				}
			} else {
				if err != nil {
					t.Error("预期没错误但实际报错")
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

func Test_handleRegisterEndEvent(t *testing.T) {
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
	_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
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
		event     RegisterEvent
		expectErr error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
				expectErr: nil,
			},
		},
		{
			name: "异常1",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
				expectErr: fmt.Errorf("强制开启事务错误"),
			},
		},
		{
			name: "异常2",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_END,
				},
				expectErr: fmt.Errorf("exec failed:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "exec")
			} else {
				ctx = context.Background()
			}
			err := handleRegisterEndEvent(ctx, tt.args.event)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Errorf("handleRegisterEndEvent() error = %v, wantErr %v", err, tt.args.expectErr)
				}
			} else {
				if err != nil {
					t.Errorf("预期没错误但实际报错")
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

func Test_handleRegisterReviewEndEvent(t *testing.T) {
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
	_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
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
		event     RegisterEvent
		expectErr error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常测试",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_REVIEW_END,
				},
				expectErr: nil,
			},
		},
		{
			name: "异常1",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_REVIEW_END,
				},
				expectErr: fmt.Errorf("强制开启事务错误"),
			},
		},
		{
			name: "异常2",
			args: args{
				event: RegisterEvent{
					RegisterID: 1,
					Type:       EVENT_TYPE_REGISTER_REVIEW_END,
				},
				expectErr: fmt.Errorf("exec failed:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "异常1") {
				ctx = context.WithValue(ctx, "force-error", "beginTx")
			} else if strings.Contains(tt.name, "异常2") {
				ctx = context.WithValue(ctx, "force-error", "exec")
			} else {
				ctx = context.Background()
			}
			err := handleRegisterReviewEndEvent(ctx, tt.args.event)
			if tt.args.expectErr != nil {
				if !strings.Contains(err.Error(), tt.args.expectErr.Error()) {
					t.Errorf("handleRegisterEndEvent() error = %v, wantErr %v", err, tt.args.expectErr)
				}
			} else {
				if err != nil {
					t.Errorf("预期没错误但实际报错")
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
func createMockContext(req *http.Request, userId int64, role ...int) context.Context {
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
			Role: null.Int{
				sql.NullInt64{
					Valid: true,
					Int64: 10100,
				},
			},
		},
		Msg:  &cmn.ReplyProto{},
		Role: r,
	}

	// 将服务上下文存储到上下文中
	ctx = context.WithValue(ctx, cmn.QNearKey, q)

	return ctx
}
func addDomain() {

}

func Test_register(t *testing.T) {
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

	tests := []struct {
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
		authority           auth_mgt.Authority
		accessAction        string
		create              bool //是否是创建的练习
	}{
		{
			name:            "GET 教师获取报名计划",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          20086,
			forceErr:        "full",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "教师获取报名计划缺少用户id",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          0,
			forceErr:        "full",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "教师获取报名计划测试获取权限失败",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          20086,
			forceErr:        "QueryRole",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试获取可执行权限失败",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "1",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试没有读数据权限",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试搜索类型不合法",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=06",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "searchtype",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试获取page失败",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "pageParseInt",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试获取页面大小失败",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "pageSizeParseInt",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试dao层出现错误",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "query",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:            "测试返回数据时反序列化失败",
			method:          "GET",
			url:             "/api/registration?page=1&pageSize=10&name=&status=&course=&search_type=00",
			reqBody:         &cmn.ReqProto{},
			userId:          1,
			forceErr:        "json",
			expectedMessage: "OK",
			expectedData: json.RawMessage(`{"code":0,"message":"OK","data":[
{
  "practiceName": "",
                "register": {
                    "ID": 44,
                    "Name": "软件工程考试报名33",
                    "Course": "00",
                    "ReviewEndTime": 1745109693215,
                    "MaxNumber": 9,
                    "StartTime": 1745109693215,
                    "EndTime": 1745109693215,
                    "ExamPlanLocation": "广东省湛江市",
                    "Creator": null,
                    "UpdatedBy": null,
                    "CreateTime": null,
                    "UpdateTime": null,
                    "Status": "08"
                },
                "studentCount": 0
}
]}`),

			create:    false,
			authority: auth_mgt.Authority{},
			setup: func() error {
				s := `DELETE FROM assessuser.t_register_plan `
				_, err := db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_plan WHERE name =$1 `
				_, err = db.Exec(ctx, s, registerName)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_register_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_exam_plan_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice_student`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				s = `DELETE FROM assessuser.t_practice`
				_, err = db.Exec(ctx, s)
				if err != nil {
					t.Fatal(err.Error())
					return err
				}
				// 先创建这个数据，最后测试完毕再删掉
				s = `INSERT INTO t_practice (id,name,correct_mode,creator,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
				_, err = db.Exec(ctx, s, uid, "练习", "00", uid, 5, "00", uid)
				if err != nil {
					t.Fatal(err)
					return err
				}
				// 这里也随便插入几个学生
				s = `INSERT INTO t_practice_student (id,student_id , practice_id,creator,status)VALUES($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`
				_, err = db.Exec(ctx, s, 1, 20022, uid, uid, "00", 2, 20023, uid, uid, "00")
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 1, "报名计划", "00", uid, time.Now().UnixMilli(), 0)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 2, "报名计划2", "00", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 3, "报名计划3", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 4, "报名计划4", "08", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 5, "报名计划5", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 6, "报名计划6", "04", uid, time.Now().UnixMilli(), 1)
				s = `INSERT INTO  t_register_plan (id,name, status,creator ,create_time,max_number) VALUES  ($1 ,$2 ,$3,$4,$5,$6)`
				_, err = db.Exec(ctx, s, 7, "报名计划7", "04", uid, time.Now().UnixMilli(), 1)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 1, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 2, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				s = `INSERT INTO t_register_practice (register_id, practice_id,creator,create_time,status) VALUES ( $1,$2,$3,$4,$5)`
				_, err = db.Exec(ctx, s, 6, uid, uid, time.Now().UnixMilli(), RegisterPracticeStatus.Normal)
				if err != nil {
					t.Fatal(err)
					return err
				}
				return nil
			},
			accessAction: "full",
		},
		{
			name:   "POST 添加报名计划",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "post",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试添加报名计划",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "ioReadAll",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试读取前端请求体失败",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "readReqProtoJson",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试读取register失败",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "readRJson",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试缺少开始时间",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "post",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试审核人的id数组格式不正确",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds":      "1234",
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "post",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试dao层报错",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "query",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试缺少创建数据权限",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
		{
			name:   "测试没有更新数据权限",
			method: "POST",
			url:    "/api/registration",
			reqBody: &cmn.ReqProto{

				Action: "",
				Data: func() json.RawMessage {
					data := map[string]interface{}{
						"registration": map[string]interface{}{
							"ID":               1,
							"Name":             "软件工程考试报名4",
							"StartTime":        "1745109693215",
							"EndTime":          "1745109693215",
							"ReviewEndtime":    "1745109693215",
							"MaxNumber":        9,
							"Course":           "00",
							"ExamPlanLocation": "广东省湛江市",
							"ReviewerIds": []int64{
								41,
								57,
								71,
							},
						},
						"practice_ids": []int{2168},
					}
					jsonBytes, _ := json.Marshal(data)
					return json.RawMessage(jsonBytes)
				}(),
			},
			userId:          1,
			forceErr:        "",
			expectedMessage: "OK",
			create:          false,
			authority:       auth_mgt.Authority{},
			accessAction:    "full",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := addTestDomainApi(tt.url, tt.accessAction, 20086)
			if err != nil {
				return
			}
			if tt.setup != nil {
				err := tt.setup()
				if err != nil {
					t.Fatal(err)
				}
			}
			var req *http.Request
			if tt.reqBody != nil {
				buf, err := json.Marshal(tt.reqBody)
				if err != nil {
					t.Fatal(err.Error())
				}
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewReader(buf))
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			// 创建响应记录器
			ctx := createMockContext(req, tt.userId)

			//传入强制err
			if tt.forceErr != "" {
				ctx = context.WithValue(ctx, "force-error", tt.forceErr)
			}
			register(ctx)
			q := cmn.GetCtxValue(ctx)
			resp := q.Msg
			t.Logf("resp:%v\n", resp)
		})
		t.Cleanup(func() {
			// 这里再删除这个练习，随后再重新创建
			s = `DELETE FROM assessuser.t_practice`
			_, err := db.Exec(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			s = `DELETE FROM assessuser.t_domain_api`
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
