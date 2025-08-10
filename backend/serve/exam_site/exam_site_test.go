package exam_site

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"
)

func TestMain(m *testing.M) {

	cmn.ConfigureForTest()

	z = cmn.GetLogger()

	m.Run()
}

func TestEnroll(t *testing.T) {

	tests := []struct{
		name string
		author string
	}{

		{
			name: "有效author",
			author: `{"name":"Zhong Peiqi","tel":"13068178042","email":"zpekii3156@qq.com"}`,
		},
		{
			name: "无效author",
			author: "Test",
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Enroll(tt.author)
		})
	}

}


func TestExamSite(t *testing.T) {

	nowTime := time.Now().Unix()

	tests := []struct {
		name        string
		q           *cmn.ServiceCtx
		passExpected bool
		errWanted   string
		setup       func()
		cleanup     func()
	}{

		{
			name: "不支持的Http方法",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("Unknown255", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:   "不支持的HTTP方法: Unknown255",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},

		// ==============创建考点测试============= coverage: 100%
		//       .o.       oooooooooo.   oooooooooo.            .oooooo..o ooooo ooooooooooooo oooooooooooo 
		//      .888.      `888'   `Y8b  `888'   `Y8b          d8P'    `Y8 `888' 8'   888   `8 `888'     `8 
		//     .8"888.      888      888  888      888         Y88bo.       888       888       888         
		//    .8' `888.     888      888  888      888          `"Y8888o.   888       888       888oooo8    
		//   .88ooo8888.    888      888  888      888 8888888      `"Y88b  888       888       888    "    
		//  .8'     `888.   888     d88'  888     d88'         oo     .d8P  888       888       888       o 
		// o88o     o8888o o888bood8P'   o888bood8P'           8""88888P'  o888o     o888o     o888ooooood8 	

		{
			name: "创建考点成功",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"server_host": "t.test.top:6443",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: true,
			errWanted:   "",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点成功-缺少sever_host",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: true,
			errWanted:   "",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-缺少name",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(`{
					"data": {
						"name": "",
						"address": "test-site-addr",
						"server_host": "t.test.top:6443",
						"admin": 1622
					}
				}`)),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:   "validation failed:Key: 'examSiteInfo.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-name类型为非字符串",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(`{
					"data": {
						"name": 123,
						"address": "test-site-addr",
						"server_host": "t.test.top:6443",
						"admin": 1622
					}
				}`)),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:   "json: cannot unmarshal number into Go struct field examSiteInfo.name of type string",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-admin为非数字类型",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(`{
					"data": {
						"name": "test-site",
						"address": "test-site-addr",
						"server_host": "t.test.top:6443",
						"admin": "1622"
					}
				}`)),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:   "json: cannot unmarshal string into Go struct field examSiteInfo.admin of type int64",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制开启事务失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"txBeginErr": fmt.Errorf("force tx begin err"),
				},
			},
			passExpected: false,
			errWanted:   "force tx begin err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制事务提交失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"commitErr": fmt.Errorf("force tx commit err"),
				},
			},
			passExpected: false,
			errWanted:   "force tx commit err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制事务回滚失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": %d,
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"rollbackErr": fmt.Errorf("force tx rollback err"),
				},
			},
			passExpected: false,
			errWanted:   "force tx rollback err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制读取Body失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"readBodyErr": fmt.Errorf("force read body err"),
				},
			},
			passExpected: false,
			errWanted:   "force read body err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制添加系统账号SQL Prepare 失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"prepareErr1": fmt.Errorf("force add sys user sql prepare err"),
				},
			},
			passExpected: false,
			errWanted:   "force add sys user sql prepare err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制执行添加系统账号sql失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"sqlExecErr1": fmt.Errorf("force execute add sys user sql err"),
				},
			},
			passExpected: false,
			errWanted:   "force execute add sys user sql err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制添加考点SQL Prepare 失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"prepareErr2": fmt.Errorf("force add exam site prepare err"),
				},
			},
			passExpected: false,
			errWanted:   "force add exam site prepare err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制执行添加考点sql失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"sqlExecErr2": fmt.Errorf("force execute add exam site sql err"),
				},
			},
			passExpected: false,
			errWanted:   "force execute add exam site sql err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "创建考点失败-强制返回json Marshal失败",
			setup: func() {},
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": 1622
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"jsonMarshalErr": fmt.Errorf("force marshal return data err"),
				},
			},
			passExpected: false,
			errWanted:   "force marshal return data err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Errorf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Errorf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		

		// ======================================

		// ===========获取考点列表测试============
																														
		// ======================================																																	

	}

	// ooooooooo.   ooooo     ooo ooooo      ooo 
	// `888   `Y88. `888'     `8' `888b.     `8' 
	//  888   .d88'  888       8   8 `88b.    8  
	//  888ooo88P'   888       8   8   `88b.  8  
	//  888`88b.     888       8   8     `88b.8  
	//  888  `88b.   `88.    .8'   8       `888  
	// o888o  o888o    `YbodP'    o8o        `8  								
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			ctx := context.WithValue(context.Background(), cmn.QNearKey, tt.q)

			examSite(ctx)

			if tt.q.Err != nil {

				if !tt.passExpected && tt.q.Err.Error() != tt.errWanted {
					t.Errorf("expected error: %s, got: %s", tt.errWanted, tt.q.Err.Error())
					return
				}

				if tt.passExpected {
					t.Errorf("expected no error, got: %s", tt.q.Err.Error())
					return
				}

				return
			}

			var data examSiteInfo
			err := json.Unmarshal(tt.q.Msg.Data, &data)
			if err != nil {
				t.Errorf("failed to unmarshal response data: %v", err)
				return
			}

			if data.AccessToken == "" && tt.passExpected {
				t.Errorf("expected non-empty access token, got empty")
				return
			}

			if len(data.AccessToken) != 64 && tt.passExpected {
				t.Errorf("expected access token length of 64, got %d", len(data.AccessToken))
				return
			}

		})

	}

}
