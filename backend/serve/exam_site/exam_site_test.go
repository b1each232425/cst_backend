package exam_site

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"

	"w2w.io/cmn"
	"w2w.io/null"
)

var (
	store = sessions.NewCookieStore([]byte("secret-key"))
)

func TestMain(m *testing.M) {

	cmn.ConfigureForTest()

	z = cmn.GetLogger()

	dbConn := cmn.GetPgxConn()

	config := dbConn.Config().ConnConfig

	dbAddr = config.Host

	dbPort = int(config.Port)

	dbName = config.Database

	dbUser = config.User

	dbPwd = config.Password

	sysUser = viper.GetString("examSiteServerSync.sysUser")

	accessToken = viper.GetString("examSiteServerSync.accessToken")

	maxRetry = viper.GetInt("examSiteServerSync.maxRetry")

	centralServerUrl = viper.GetString("examSiteServerSync.centralServerUrl")

	sshUser = viper.GetString("examSiteServerSync.centralServerSSH.user")

	sshHost = viper.GetString("examSiteServerSync.centralServerSSH.host")
	
	sshPort = viper.GetInt("examSiteServerSync.centralServerSSH.port")

	m.Run()
}

func TestEnroll(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var respBody cmn.ReplyProto

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}

		q := &cmn.ServiceCtx{
			SysUser: &cmn.TUser{
				ID: null.IntFrom(1622),
			},
			R:   r,
			W:   w,
			Msg: &cmn.ReplyProto{},
		}

		ctx := context.WithValue(context.Background(), cmn.QNearKey, q)

		switch r.URL.Path {

		case "/api/login":

			session, err := store.Get(r, "qNearSessions")
			if err != nil {
				t.Errorf("%s", err.Error())
				return
			}

			session.Values["loginType"] = "upLogin"
			session.Values["ID"] = 1000
			session.Values["Account"] = "test_account"
			session.Values["Role"] = "test_role"
			session.Values["Authenticated"] = true
			err = session.Save(r, w)
			if err != nil {
				t.Errorf("failed to save session: %v", err)
				return
			}

			respBody = cmn.ReplyProto{
				Status: 0,
				Data:   body,
			}

		case "/api/exam-site/sync":

			examSiteSync(ctx)
			return

		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			respBody = cmn.ReplyProto{
				Status: -1,
				Msg:    "unexpected request path: " + r.URL.Path,
			}
		}

		b, err := json.Marshal(respBody)
		if err != nil {
			t.Errorf("failed to marshal response body: %v", err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "debug",
			Value:    "1",
			Expires:  time.Now().Add(60 * 2 * time.Second),
			SameSite: http.SameSiteLaxMode,
		})

		w.WriteHeader(http.StatusOK)
		w.Write(b)

	}))

	viper.Set("examSiteServerSync.centralServerUrl", server.URL)

	defer func() {
		server.Close()

		err := os.RemoveAll(filepath.Join(os.Getenv("PWD"), "data/tmp"))
		if err != nil {
			t.Errorf("failed to remove test directory: %v", err)
		}

	}()

	viper.Set("examSiteServerSync.maxRetry", 1)

	_, err := cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
	if err != nil {
		t.Fatalf("failed to set sync status: %v", err)
		return
	}

	tests := []struct {
		name   string
		author string
	}{

		{
			name:   "有效author",
			author: `{"name":"Zhong Peiqi","tel":"13068178042","email":"zpekii3156@qq.com"}`,
		},
		{
			name:   "无效author",
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
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func()
		cleanup      func()
		check        func(q *cmn.ServiceCtx, passExpected bool)
	}{

		{
			name:  "不支持的Http方法",
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
			errWanted:    "不支持的HTTP方法: Unknown255",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
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
			name:  "创建考点成功",
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
			errWanted:    "",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点成功-缺少sever_host",
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
			errWanted:    "",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-缺少name",
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
			errWanted:    "validation failed:Key: 'examSiteInfo.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-name类型为非字符串",
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
			errWanted:    "json: cannot unmarshal number into Go struct field examSiteInfo.name of type string",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-admin为非数字类型",
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
			errWanted:    "json: cannot unmarshal string into Go struct field examSiteInfo.admin of type int64",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制开启事务失败",
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
			errWanted:    "force tx begin err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制事务提交失败",
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
			errWanted:    "force tx commit err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制事务回滚失败",
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
			errWanted:    "force tx rollback err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制读取Body失败",
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
			errWanted:    "force read body err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制添加系统账号SQL Prepare 失败",
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
			errWanted:    "force add sys user sql prepare err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制执行添加系统账号sql失败",
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
			errWanted:    "force execute add sys user sql err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制添加考点SQL Prepare 失败",
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
			errWanted:    "force add exam site prepare err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制执行添加考点sql失败",
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
			errWanted:    "force execute add exam site sql err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},
		{
			name:  "创建考点失败-强制返回json Marshal失败",
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
			errWanted:    "force marshal return data err",
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to clean up test data: %v", err)
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
			check: func(q *cmn.ServiceCtx, passExpected bool) {
				var data examSiteInfo
				err := json.Unmarshal(q.Msg.Data, &data)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if data.AccessToken.String == "" && passExpected {
					t.Errorf("expected non-empty access token, got empty")
					return
				}

				if len(data.AccessToken.String) != 64 && passExpected {
					t.Errorf("expected access token length of 64, got %d", len(data.AccessToken.String))
					return
				}
			},
		},

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

			if tt.q.Err != nil || (tt.passExpected && tt.q.Msg.Status != 0) || !tt.passExpected {

				if tt.q.Err == nil {
					tt.q.Err = fmt.Errorf(tt.q.Msg.Msg)
				}

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

			if tt.check != nil {
				tt.check(tt.q, tt.passExpected)
			}

		})

	}

}

func TestExamSiteList(t *testing.T) {

	nowTime := time.Now().Unix()

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func()
		cleanup      func()
	}{

		{
			name: "获取考点列表失败-无效的HTTP方法",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("Unknown255", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "Unknown255",
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
			errWanted:    "不支持的HTTP方法: Unknown255",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},

		// ==============获取考点列表测试=============
		//   .oooooo.    oooooooooooo ooooooooooooo          .oooooo..o ooooo ooooooooooooo oooooooooooo         ooooo        ooooo  .oooooo..o ooooooooooooo
		//  d8P'  `Y8b   `888'     `8 8'   888   `8         d8P'    `Y8 `888' 8'   888   `8 `888'     `8         `888'        `888' d8P'    `Y8 8'   888   `8
		// 888            888              888              Y88bo.       888       888       888                  888          888  Y88bo.           888
		// 888            888oooo8         888               `"Y8888o.   888       888       888oooo8             888          888   `"Y8888o.       888
		// 888     ooooo  888    "         888      8888888      `"Y88b  888       888       888    "    8888888  888          888       `"Y88b      888
		// `88.    .88'   888       o      888              oo     .d8P  888       888       888       o          888       o  888  oo     .d8P      888
		//  `Y8bood8P'   o888ooooood8     o888o             8""88888P'  o888o     o888o     o888ooooood8         o888ooooood8 o888o 8""88888P'      o888o
		{
			name: "以管理员角色获取考点列表成功-无筛选无排序",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": ""
							}
						],
						"filter": {
							"name": ""
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1622, true),
					Role: null.NewInt(int64(cmn.CDomainAssessAdmin), true),
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
			errWanted:    "",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表成功-带筛选降序",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表成功-无筛选升序",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "ASC"
							}
						]
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表成功-筛选结果为空",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test123"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-无效的排序方式",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DEC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "不支持的排序方式: DEC key: roomCount",
			setup: func() {
				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (name,creator) VALUES ('test-site-%d', 1622)`, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				if c < 1 {
					t.Fatalf("expected 1 affected row, got %d", c)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("")
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-无效的URL参数",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "unexpected end of JSON input",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-页码小于0",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": -10,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "页码不能小于0",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-每页条数小于1",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": -1,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "每页条数不能小于1",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-页码类型为非数字",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": "undefined",
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
			errWanted:    "json: cannot unmarshal string into Go struct field ReqProto.page of type int64",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制查询数据总行数SQL Prepare失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"prepareErr1": fmt.Errorf("forced prepare get row count sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare get row count sql err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制查询数据总行数SQL 执行失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"sqlExecErr1": fmt.Errorf("forced execute get row count sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced execute get row count sql err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制获取查询数据总行数SQL Affected Rows失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"rowsAffectedErr": fmt.Errorf("forced get affected rows err"),
				},
			},
			passExpected: false,
			errWanted:    "forced get affected rows err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制查询数据SQL Prepare失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"prepareErr2": fmt.Errorf("forced prepare get data sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare get data sql err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制查询数据SQL 执行失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"queryErr": fmt.Errorf("forced sql query err"),
				},
			},
			passExpected: false,
			errWanted:    "forced sql query err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制Scan数据失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"scanErr": fmt.Errorf("forced scan get data sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced scan get data sql err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
		{
			name: "获取考点列表失败-强制Marshal返回的数据失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site?q=%s`, url.QueryEscape(`{
						"page": 0,
						"pageSize": 10,
						"orderBy": [
							{
								"roomCount": "DESC"
							}
						],
						"filter": {
							"name": "test"
						}
					}`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
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
					"jsonMarshal": fmt.Errorf("forced json marshal err"),
				},
			},
			passExpected: false,
			errWanted:    "forced json marshal err",
			setup: func() {
				dbConn := cmn.GetDbConn()

				siteSets := []string{}

				for i := 0; i < 10; i++ {
					siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', 1622, 'localhost','address',1622)`, nowTime+int64(i), nowTime))
				}

				roomSets := []string{}

				for i := 0; i < 10; i++ {
					for j := 0; j < i; j++ {
						roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, 1622)`, nowTime, nowTime+int64(i)))
					}
				}

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name = 'test-room-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_room", c)

				r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err = r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

			},
		},
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

			examSiteList(ctx)

			if tt.q.Err != nil || (tt.passExpected && tt.q.Msg.Status != 0) || !tt.passExpected {

				if tt.q.Err == nil {
					tt.q.Err = fmt.Errorf(tt.q.Msg.Msg)
				}

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

			p := tt.q.R.URL.Query().Get("q")

			pageSize := int(gjson.Get(p, "pageSize").Int())

			nameFilter := gjson.Get(p, "filter.name").String()

			orderByList := gjson.Get(p, "orderBy").Array()

			orderByMap := orderByList[0].Map()

			var r []examSiteInfo

			err := json.Unmarshal(tt.q.Msg.Data, &r)
			if err != nil {
				t.Errorf("failed to unmarshal response data: %v", err)
				return
			}

			if len(r) > pageSize {
				t.Errorf("expected no more than %d items, got %d", pageSize, len(r))
				return
			}

			rc := 0

			if len(r) > 0 && tt.q.Msg.RowCount <= 0 {
				t.Errorf("expected row count to be greater than 0, got %d", tt.q.Msg.RowCount)
				return
			}

			for i, item := range r {

				if i == 0 {
					rc = int(item.RoomCount.Int64)
				}

				if orderByMap["roomCount"].Str == "ASC" && int(item.RoomCount.Int64) < rc {
					t.Errorf("expected order by roomCount ASC item roomCount to be greater than or equal to %d, got %d", rc, item.RoomCount.Int64)
					return
				}

				if orderByMap["roomCount"].Str == "DESC" && int(item.RoomCount.Int64) > rc {
					t.Errorf("expected order by roomCount DESC item roomCount to be less than or equal to %d, got %d", rc, item.RoomCount.Int64)
					return
				}

				if !strings.Contains(item.Name, nameFilter) && !strings.Contains(item.Address, nameFilter) && !strings.Contains(item.ServerHost.String, nameFilter) {
					t.Errorf("expected item name, address or server host to contain filter '%s', got '%s', '%s', '%s'", nameFilter, item.Name, item.Address, item.ServerHost.String)
					return
				}

				rc = int(item.RoomCount.Int64)

			}

		})

	}

}

func TestExamSiteSyncInit(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var respBody cmn.ReplyProto

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}

		q := &cmn.ServiceCtx{
			SysUser: &cmn.TUser{
				ID: null.IntFrom(1622),
			},
			R:   r,
			W:   w,
			Msg: &cmn.ReplyProto{},
			RedisClient: cmn.GetRedisConn(),
		}

		ctx := context.WithValue(context.Background(), cmn.QNearKey, q)

		switch r.URL.Path {

		case "/api/login":

			session, err := store.Get(r, "qNearSessions")
			if err != nil {
				t.Errorf("%s", err.Error())
				return
			}

			session.Values["loginType"] = "upLogin"
			session.Values["ID"] = 1000
			session.Values["Account"] = "test_account"
			session.Values["Role"] = "test_role"
			session.Values["Authenticated"] = true
			err = session.Save(r, w)
			if err != nil {
				t.Errorf("failed to save session: %v", err)
				return
			}

			respBody = cmn.ReplyProto{
				Status: 0,
				Data:   body,
			}

		case "/api/exam-site/sync":

			examSiteSync(ctx)
			return

		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			respBody = cmn.ReplyProto{
				Status: -1,
				Msg:    "unexpected request path: " + r.URL.Path,
			}
		}

		b, err := json.Marshal(respBody)
		if err != nil {
			t.Errorf("failed to marshal response body: %v", err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "debug",
			Value:    "1",
			Expires:  time.Now().Add(60 * 2 * time.Second),
			SameSite: http.SameSiteLaxMode,
		})

		w.WriteHeader(http.StatusOK)
		w.Write(b)

	}))

	viper.Set("examSiteServerSync.centralServerUrl", server.URL)

	defer func() {
		server.Close()

		err := os.RemoveAll(filepath.Join(os.Getenv("PWD"), "data/tmp"))
		if err != nil {
			t.Errorf("failed to remove test directory: %v", err)
		}

	}()

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func()
		check        func(q *cmn.ServiceCtx) (err error)
		cleanup      func()
	}{
		{
			name: "同步初始化成功",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
				},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() {

				viper.Set("examSiteServerSync.maxRetry", 1)

				_, err := cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

			},
			check: func(q *cmn.ServiceCtx) (err error) {
				return
			},
			cleanup: func() {
				close(pullChan)
				close(pushChan)
			},
		},
		{
			name: "同步初始化成功后发送拉取通知成功",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
				},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() {

				viper.Set("examSiteServerSync.maxRetry", 1)

				_, err := cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: func() {
				close(pullChan)
				close(pushChan)
			},
		},
	}

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

			examSiteSyncInit(ctx)

			if tt.check != nil {
				err := tt.check(tt.q)
				if err != nil {
					t.Errorf("execute after fun failed: %s", err.Error())
					return
				}
			}

			if tt.q.Err != nil || (tt.passExpected && tt.q.Msg.Status != 0) || !tt.passExpected {

				if tt.q.Err == nil {
					tt.q.Err = fmt.Errorf(tt.q.Msg.Msg)
				}

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

		})

	}

}

func TestExamSiteSyncApi(t *testing.T) {

	nowTime := time.Now().Unix()

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func()
		check        func(q *cmn.ServiceCtx) (err error)
		cleanup      func()
	}{

		{
			name: "考点同步失败-无效的HTTP方法",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("Unknown255", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "Unknown255",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:    "不支持的HTTP方法: Unknown255",
			setup: func() {
			},
			cleanup: func() {
			},
		},
		{
			name: "考点同步失败-无效的同步操作",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`a`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:    "不支持的同步操作: a",
			setup: func() {
			},
			cleanup: func() {
			},
		},

		// ===========准备考点数据测试============
		// ooooooooo.
		// `888   `Y88.
		//  888   .d88' oooo d8b  .ooooo.  oo.ooooo.   .oooo.   oooo d8b  .ooooo.
		//  888ooo88P'  `888""8P d88' `88b  888' `88b `P  )88b  `888""8P d88' `88b
		//  888          888     888ooo888  888   888  .oP"888   888     888ooo888
		//  888          888     888    .o  888   888 d8(  888   888     888    .o
		// o888o        d888b    `Y8bod8P'  888bod8P' `Y888""8o d888b    `Y8bod8P'
		//                                  888
		//                                 o888o
		{
			name: "准备数据成功",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: true,
			errWanted:    "",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-无效的账号ID",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(-2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:    "invalid sysUser: -2025",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制生成随机字节失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"generateRandByteErr": fmt.Errorf("forced generate rand byte err"),
				},
			},
			passExpected: false,
			errWanted:    "forced generate rand byte err",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制创建临时目录失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"mkdirAllErr": fmt.Errorf("forced mkdir all err"),
				},
			},
			passExpected: false,
			errWanted:    "forced mkdir all err",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制执行 pg_dump 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pgDumpErr": fmt.Errorf("forced pg_dump err"),
				},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced pg_dump err", ""),
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制执行 psql 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"psqlErr": fmt.Errorf("forced psql err"),
				},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced psql err", ""),
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制json marshal 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"jsonMarshalErr": fmt.Errorf("forced json marshal err"),
				},
			},
			passExpected: false,
			errWanted:    "forced json marshal err",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制清除临时目录失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"jsonMarshalErr":  fmt.Errorf("forced json marshal err"),
					"removeTmpDirErr": fmt.Errorf("forced remove tmp dir err"),
				},
			},
			passExpected: false,
			errWanted:    "forced remove tmp dir err",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制创建导出脚本失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"createExportScriptFileErr": fmt.Errorf("force-create-export-script-file-err-^a1^2*zc$32h@g4"),
				},
			},
			passExpected: false,
			errWanted:    "force-create-export-script-file-err-^a1^2*zc$32h@g4",
			setup: func() {

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},
		{
			name: "准备数据失败-强制写入导出脚本失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/sync?action=%s`, url.QueryEscape(`0`)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(2025, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSite), true),
				},
				Domains: []cmn.TDomain{
					{
						ID: null.IntFrom(int64(cmn.CDomainAssessExamSite)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"writeExportScriptFileErr": fmt.Errorf("force-write-export-script-file-err-^a1^2*zc$32h@g4"),
				},
			},
			passExpected: false,
			errWanted:    "force-write-export-script-file-err-^a1^2*zc$32h@g4",
			setup: func() {

				

				dbConn := cmn.GetDbConn()

				_, err := dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

			},
			cleanup: func() {

				dbConn := cmn.GetDbConn()

				r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE name = 'test-site-%d'`, nowTime))
				if err != nil {
					t.Fatalf("failed to delete test data: %v", err)
					return
				}

				c, err := r.RowsAffected()
				if err != nil {
					t.Fatalf("failed to get affected rows: %v", err)
					return
				}

				t.Logf("Have already cleaned up %d rows from t_exam_site", c)

				folderFullPath := path.Join(os.Getenv("PWD"), "/data/tmp/")

				o, err := exec.Command("rm", "-rvf", folderFullPath).CombinedOutput()
				if err != nil {
					t.Fatalf("failed to remove folder: %v, output: %s", err, string(o))
					return
				}

				t.Logf("Successfully cleaned up folder: %s output: %x", folderFullPath, o)

			},
		},

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

			examSiteSync(ctx)

			if tt.q.Err != nil || (tt.passExpected && tt.q.Msg.Status != 0) || !tt.passExpected {

				if tt.q.Err == nil {
					tt.q.Err = fmt.Errorf(tt.q.Msg.Msg)
				}

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

			if tt.check != nil {
				err := tt.check(tt.q)
				if err != nil {
					t.Errorf("check err: %s", err.Error())
					return
				}
			}

			d := syncInfo{
				Path:          "",
				TableFileList: []string{},
			}

			err := json.Unmarshal(tt.q.Msg.Data, &d)
			if err != nil {
				t.Errorf("failed to unmarshal data: %v", err)
				return
			}

			// 检查目录下是否有相应的文件
			for _, f := range d.TableFileList {

				_, err := os.Stat(filepath.Join(d.Path, f))
				if err != nil {
					t.Errorf("failed to stat file while it shouldn't: %v", err)
					return
				}

			}

		})

	}
}
