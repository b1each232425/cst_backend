package exam_site

import (
	"context"
	"database/sql"
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
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"

	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
)

var (
	store          = sessions.NewCookieStore([]byte("secret-key"))
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

	accessToken = viper.GetString("examSiteServerSync.accessToken")

	maxRetry = viper.GetInt("examSiteServerSync.maxRetry")

	centralServerUrl = viper.GetString("examSiteServerSync.centralServerUrl")

	sshUser = viper.GetString("examSiteServerSync.centralServerSSH.user")

	sshHost = viper.GetString("examSiteServerSync.centralServerSSH.host")

	sshPort = viper.GetInt("examSiteServerSync.centralServerSSH.port")

	o, err := exec.Command("bash", "-c", "service ssh restart").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to restart SSH service: %v\n", err)
		return
	}
	fmt.Printf("SSH service restarted successfully: %s\n", o)

	m.Run()
}

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
	ON CONFLICT (api, domain) DO NOTHING`

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

func removeTestDomainApis(domain int64) (err error) {

	dbConn := cmn.GetDbConn()

	for _, v := range testDomainApis {

		s := `DELETE FROM t_domain_api
		USING t_api
		WHERE t_api.id = t_domain_api.api
		  AND t_domain_api.domain = $1
		  AND t_api.expose_path = $2
		  AND t_api.access_action = $3`

		_, err := dbConn.Exec(s, domain, v.ApiPath, v.AccessAction)
		if err != nil {
			return err
		}
	}

	return
}

func mockExamSiteSyncData(sysUser int64, nowTime int64) (cleanup func() error, err error) {

	dbConn := cmn.GetDbConn()

	ctx := context.Background()

	tx, err := dbConn.BeginTx(ctx, nil)

	defer func() {

		if err != nil {
			tx.Rollback()
			return
		}

		tx.Commit()

	}()

	testID := nowTime / 100

	sqls := []string{
		fmt.Sprintf(`WITH ins_site AS (
  	INSERT INTO t_exam_site (id, name, address, server_host, creator, admin, sys_user, domain_id)
  	VALUES 
		(%d, 'test-site-%d', 'test,address', 'localhost', 1000, %d, %d, 1999),
		(%d, 'test-site-%d', 'test,address', 'localhost', 1000, %d, %d, 1999),
		(%d, 'test-site-%d', 'test,address', 'localhost', 1000, %d, %d, 1999)
  	ON CONFLICT DO NOTHING
  	RETURNING id
),
ins_rooms AS (
  	INSERT INTO t_exam_room (id, exam_site, name, capacity, creator)
  	VALUES
		(%d, %d, 'test-room-1', 30, 1000),
		(%d, %d, 'test-room-2', 30, 1000),
		(%d, %d, 'test-room-3', 30, 1000),
		(%d, %d, 'test-room-4', 30, 1000),
		(%d, %d, 'test-room-5', 30, 1000)
  	ON CONFLICT DO NOTHING
  	RETURNING id
),
ins_exam_info AS (
  	INSERT INTO t_exam_info (id, name, type, mode, creator, status, files)
  	VALUES (%d, 'test-exam', '04', '02', 1000, '02', '[%d,%d]')
  	ON CONFLICT(id) DO NOTHING
  	RETURNING id
),
ins_exam_session AS (
  	INSERT INTO t_exam_session (id, exam_id, paper_id, mark_method, start_time, end_time, creator)
  	VALUES (%d, %d, %d, '00', %d, %d, 1000)
  	ON CONFLICT DO NOTHING
  	RETURNING id
),
ins_paper AS (
	INSERT INTO t_paper (id, name, exampaper_id, creator, domain_id)
	VALUES (%d, 'test-paper', %d, 1000, 10098)
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_exam_paper AS (
  	INSERT INTO t_exam_paper (id, exam_session_id, creator)
  	VALUES (%d, %d, 1000)
  	ON CONFLICT DO NOTHING
  	RETURNING id
),
ins_exam_paper_group AS (
	INSERT INTO t_exam_paper_group (id, exam_paper_id, name, "order", creator) 
	VALUES
		(%d, %d, 'group-1', 1, 1000),
		(%d, %d, 'group-2', 2, 1000),
		(%d, %d, 'group-3', 3, 1000)
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_exam_paper_question AS (
	INSERT INTO t_exam_paper_question (id, group_id, creator, content, options, answers)
	VALUES
		(%d, %d, 1000, '<p><span style="font-size: 12pt">操作系统A</span></p>', '[{"label": "A","value": "对"},{ "label": "B","value": "错"}]', '["A"]'),
		(%d, %d, 1000, '<p><span style="font-size: 12pt">A</span></p>', '[{"label": "A","value": "对"},{ "label": "B","value": "错"}]', '["B","D"]'),
		(%d, %d, 1000, '<p><span style="font-size: 12pt">操作系统A</span></p>', '[{"label": "A","value": "对"},{ "label": "B","value": "错"}]', '["C"]')
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_users AS (
  	INSERT INTO t_user (id, category, account, domain_id) 
	VALUES
		(%d, 'sys^user', 'test-examinee-%d', %d),
		(%d, 'sys^user', 'test-examinee-%d', %d),
		(%d, 'sys^user', 'test-examinee-%d', %d),
		(%d1, 'sys^user', 'test-invigilator-%d', %d)
  	ON CONFLICT DO NOTHING
  	RETURNING id
),
ins_domain AS (
	INSERT INTO t_domain (id, name, domain, priority, creator) 
	VALUES
		(1999, '考试系统', 'assess', 0, 1000),
		(10128, '考试系统.学生', 'assess^student', 7, 1000),
		(10116, '考试系统.监考员', 'assess^examSupervisor', 7, 1000)
	ON CONFLICT DO NOTHING
),
ins_user_domains AS (
  	INSERT INTO t_user_domain (sys_user, domain, data_access_mode, domain_id) 
	VALUES
		(%d, 10128, 'full', %d),
		(%d, 10128, 'full', %d),
		(%d, 10128, 'full', %d),
		(%d1, 10116, 'full', %d)
  	ON CONFLICT(sys_user, domain) DO NOTHING
  	RETURNING sys_user
),
ins_examinees AS (
  	INSERT INTO t_examinee (id, student_id, exam_room, exam_session_id, creator) 
	VALUES
		(%d, %d, %d, %d, 1000),
		(%d, %d, %d, %d, 1000),
		(%d, %d, %d, %d, 1000)
	ON CONFLICT DO NOTHING
  	RETURNING student_id
),
ins_student_answers AS (
	INSERT INTO t_student_answers (id, examinee_id, question_id, creator, answer)
	VALUES
		(1, %d, %d, 1000, '[]'),
		(2, %d, %d, 1000, '[]'),
		(3, %d, %d, 1000, '[]')
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_exam_record AS (
	INSERT INTO t_exam_record (exam_room, exam_session, creator)
	VALUES
		(%d, %d, 1000),
		(%d, %d, 1000),
		(%d, %d, 1000)
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_invigilation AS (
	INSERT INTO t_invigilation (exam_session_id, exam_room, invigilator, creator)
	VALUES
		(%d, %d, %d1, 1000)
	ON CONFLICT DO NOTHING
	RETURNING id
),
ins_file AS (
	INSERT INTO t_file (id, file_name, path, belongto_path, digest, creator, domain_id)
	VALUES
		(%d, 'test-file-1', '/uploads/test-file-1', '/uploads/test-file-1', '%d', 1000, %d),
		(%d, 'test-file-2', '/uploads/test-file-2', '/uploads/test-file-2', '%d', 1000, %d)
	ON CONFLICT(id) DO NOTHING
	RETURNING id
)
SELECT 1;
`,
			// ins_site: (id, name, admin, sys_user)
			testID, testID, testID, sysUser,
			testID+1, testID+1, testID, sysUser+1,
			testID+2, testID+2, testID, sysUser+2,

			// ins_rooms: (id, exam_site) * 5
			testID+1, testID,
			testID+2, testID,
			testID+3, testID,
			testID+4, testID,
			testID+5, testID,

			// ins_exam_info: (id, files)
			testID, testID+1, testID+2,

			// ins_exam_session: (id, exam_id, paper_id, start_time, end_time)
			testID, testID, testID, (nowTime+3*60)*1000, (nowTime+13*60)*1000,

			// ins_paper: (id, exampaper_id)
			testID, testID,

			// ins_exam_paper (id, exam_session_id)
			testID, testID,

			// ins_exam_paper_group (id, exam_paper_id)
			testID+1, testID,
			testID+2, testID,
			testID+3, testID,

			// ins_exam_paper_question (id, group_id)
			testID+1, testID+1,
			testID+2, testID+2,
			testID+3, testID+3,

			// ins_users: (id, account, domain_id)
			testID+1, testID+1, testID,
			testID+2, testID+2, testID,
			testID+3, testID+3, testID,
			testID, testID, testID,

			// ins_user_domains: (sys_user, domain_id) *3
			testID+1, testID,
			testID+2, testID,
			testID+3, testID,
			testID, testID,

			// ins_examinees: (id, student_id, exam_room, exam_session_id) *3
			testID+1, testID+1, testID+1, testID,
			testID+2, testID+2, testID+2, testID,
			testID+3, testID+3, testID+3, testID,

			// ins_student_answers (examinee_id, question_id)
			testID+1, testID+1,
			testID+2, testID+2,
			testID+3, testID+3,

			// ins_exam_record (exam_room, exam_session)
			testID+1, testID,
			testID+2, testID,
			testID+3, testID,

			// ins_invigilation (exam_session_id, exam_room, invigilator)
			testID, testID+1, testID,

			// ins_file (id, digest, domain_id)
			testID+1, testID+1, testID,
			testID+2, testID+2, testID,
		),
	}

	os.WriteFile("./data/mock-data.sql", []byte(strings.Join(sqls, "\n")), 0755)

	for _, sql := range sqls {
		_, err = tx.ExecContext(ctx, sql)
		if err != nil {
			break
		}
	}

	totalFileNum := 2

	err = os.MkdirAll("./uploads", 0755)

	for i := 1; i <= totalFileNum; i++ {
		err = os.WriteFile(fmt.Sprintf("./uploads/%d", testID+int64(i)), []byte("test file content"), 0644)
		if err != nil {
			break
		}

		os.WriteFile(fmt.Sprintf("./uploads/%d.info", testID+int64(i)), []byte("test file info"), 0644)
		if err != nil {
			break
		}
	}

	cleanup = func() (err error) {
		dbConn := cmn.GetDbConn()

		tx, err := dbConn.Begin()

		defer func() {

			if err != nil {
				tx.Rollback()
				return
			}

			tx.Commit()

		}()

		sqls := []string{

			// 清除文件上传记录
			fmt.Sprintf(`DELETE FROM t_file WHERE domain_id = %d`, testID),

			// 清除考场记录
			fmt.Sprintf(`DELETE FROM t_exam_record WHERE exam_session = %d`, testID),

			// 清除监考记录
			fmt.Sprintf(`DELETE FROM t_invigilation WHERE exam_session_id = %d`, testID),

			// 清除考生
			fmt.Sprintf(`DELETE FROM t_examinee WHERE exam_session_id = %d`, testID),

			fmt.Sprintf(`DELETE FROM t_user_domain WHERE domain_id = %d`, testID),

			fmt.Sprintf(`DELETE FROM t_user WHERE domain_id = %d`, testID),

			// 清除考卷
			fmt.Sprintf(`DELETE FROM t_exam_paper WHERE exam_session_id = %d`, testID),

			// 清除考试场次
			fmt.Sprintf(`DELETE FROM t_exam_session WHERE exam_id = %d`, testID),

			// 清除考试
			fmt.Sprintf(`DELETE FROM t_exam_info WHERE id = %d`, testID),

			// 清除考场
			fmt.Sprintf(`DELETE FROM t_exam_room WHERE exam_site = %d`, testID),

			// 清除考点
			fmt.Sprintf(`DELETE FROM t_exam_site WHERE admin = %d`, testID),
		}

		for _, sql := range sqls {
			_, err = tx.Exec(sql)
			if err != nil {
				break
			}
		}

		os.RemoveAll("./uploads/*")

		return
	}

	return
}

func addTestUser(userID int64, domain int64) (err error) {

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

		tx.Commit()

	}()

	sql := `INSERT INTO t_user (id, category, account, official_name) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	_, err = tx.Exec(sql, userID, "sys^user", fmt.Sprintf("testuser%d", userID), fmt.Sprintf("testuser%d", userID))
	if err != nil {
		return
	}

	sql = fmt.Sprintf(`WITH temp_domain AS (
		INSERT INTO t_domain (id, name, domain, priority, creator) VALUES
		($2, '考试系统.考点测试-%d', 'assess^testUser-%d', 7, 1000)
		ON CONFLICT DO NOTHING
	)
	INSERT INTO t_user_domain (sys_user, domain) VALUES ($1, $2) ON CONFLICT(sys_user, domain) DO NOTHING`,
		userID, userID)
	_, err = tx.Exec(sql, userID, domain)
	if err != nil {
		return
	}

	return
}

func removeTestUser(userID int64) (err error) {

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

		tx.Commit()

	}()

	sql := `DELETE FROM t_user_domain WHERE sys_user = $1`
	_, err = tx.Exec(sql, userID)
	if err != nil {
		return
	}

	sql = `DELETE FROM t_user WHERE id = $1`
	_, err = tx.Exec(sql, userID)
	if err != nil {
		return
	}

	return
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

	_, err := cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
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

	dbConn := cmn.GetDbConn()

	var cleanupTestData func() error

	testID := nowTime / 100

	defaultSetup := func() (err error) {

		for _, p := range permissions {
			err = addTestDomainApi("/api/exam-site", p, int64(cmn.CDomainAssessExamSiteAdmin))
			if err != nil {
				t.Errorf("failed to add domain api: %v", err)
				return
			}
		}

		err = addTestUser(testID, int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Errorf("failed to add test user: %v", err)
			return
		}

		cleanupTestData, err = mockExamSiteSyncData(testID, nowTime)
		if err != nil {
			t.Errorf("failed to mock exam site sync data: %v", err)
			return
		}

		return
	}

	defaultAddCheck := func(q *cmn.ServiceCtx, passExpected bool) (err error) {
		var data examSiteInfo
		err = json.Unmarshal(q.Msg.Data, &data)
		if err != nil {
			t.Errorf("failed to unmarshal response data: %v", err)
			return
		}

		if data.AccessToken.String == "" && passExpected {
			err = fmt.Errorf("got empty, but not empty")
			t.Error(err.Error())
			return
		}

		if data.AccessToken.String == "" && passExpected {
			err = fmt.Errorf("got empty, but not empty")
			t.Error(err.Error())
			return
		}

		return
	}

	defaultGetCheck := func(q *cmn.ServiceCtx, passExpected bool) (err error) {

		var info examSiteInfo

		err = json.Unmarshal(q.Msg.Data, &info)
		if err != nil {
			t.Error(err.Error())
			return
		}

		var actualInfo struct {
			ID        int64 `json:"id"`
			RoomCount int64 `json:"roomCount"`
		}

		err = dbConn.QueryRow(`SELECT 
	t_exam_site.id,
	COUNT(t_exam_room.id) AS room_count
FROM t_exam_site
	JOIN t_user ON t_user.id = t_exam_site.admin
	JOIN t_exam_room ON t_exam_room.exam_site = t_exam_site.id
WHERE 
	t_exam_site.id = $1 AND 
	t_exam_site.name = $2 AND
	t_exam_site.address = $3 AND
	t_user.id = $4 AND
	t_user.official_name = $5
GROUP BY
	t_exam_site.id,
	t_user.id`, info.ID, info.Name, info.Address, info.Admin, info.AdminName,
		).Scan(
			&actualInfo.ID,
			&actualInfo.RoomCount,
		)
		if err != nil {
			t.Error(err.Error())
			return
		}

		if info.RoomCount.Int64 != actualInfo.RoomCount && passExpected {
			err = fmt.Errorf("roomCount got %d, but %d", info.RoomCount.Int64, actualInfo.RoomCount)
			t.Error(err.Error())
			return
		}

		return

	}

	defaultEditCheck := func(q *cmn.ServiceCtx, passExpected bool) (err error) {

		var editInfo struct {
			examSiteInfo
			ResetAccessToken bool `json:"resetAccessToken"`
		}

		err = json.Unmarshal(q.Msg.Data, &editInfo)
		if err != nil {
			t.Error(err.Error())
			return
		}

		var data examSiteInfo

		err = dbConn.QueryRow(`SELECT name,address,server_host,admin FROM t_exam_site WHERE id = $1`,
			editInfo.ID.Int64).Scan(&data.Name, &data.Address, &data.ServerHost, &data.Admin)
		if err != nil {
			t.Error(err.Error())
			return
		}

		if editInfo.Name != "" && editInfo.Name != data.Name && passExpected {
			err = fmt.Errorf("got %s, but %s", data.Name, editInfo.Name)
			t.Error(err.Error())
			return
		}

		if editInfo.Address != "" && editInfo.Address != data.Address && passExpected {
			err = fmt.Errorf("got %s, but %s", data.Address, editInfo.Address)
			t.Error(err.Error())
			return
		}

		if editInfo.ServerHost.Valid && editInfo.ServerHost != data.ServerHost && passExpected {
			err = fmt.Errorf("got %s, but %s", data.ServerHost.String, editInfo.ServerHost.String)
			t.Error(err.Error())
			return
		}

		if editInfo.Admin > 0 && editInfo.Admin != data.Admin && passExpected {
			err = fmt.Errorf("got %d, but %d", data.Admin, editInfo.Admin)
			t.Error(err.Error())
			return
		}

		if editInfo.ResetAccessToken && editInfo.AccessToken.String == "" && passExpected {
			err = fmt.Errorf("got empty, but not empty")
			t.Error(err.Error())
			return
		}

		return
	}

	defaultDeleteCheck := func(q *cmn.ServiceCtx, passExpected bool) (err error) {

		var delInfo struct {
			IDs []int64 `json:"ids"`
		}

		err = json.Unmarshal(q.Msg.Data, &delInfo)
		if err != nil {
			t.Error(err.Error())
			return
		}

		var rc int
		err = dbConn.QueryRow(`SELECT COUNT(id) FROM t_exam_site WHERE status = '04' AND id = ANY($1)`, delInfo.IDs).Scan(&rc)
		if err != nil {
			t.Error(err.Error())
			return
		}

		if rc != len(delInfo.IDs) && passExpected {
			err = fmt.Errorf("delete got %d, but %d", rc, len(delInfo.IDs))
		}

		return
	}

	defaultCleanup := func() {

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

		err = removeTestUser(testID)
		if err != nil {
			t.Fatalf("failed to remove test user: %v", err)
		}

		err = removeTestDomainApis(int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Fatalf("failed to remove test domain api: %v", err)
		}

		if cleanupTestData != nil {
			err = cleanupTestData()
			if err != nil {
				t.Fatalf("failed to cleanup test data: %v", err)
			}
		}

	}

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func() error
		cleanup      func()
		check        func(q *cmn.ServiceCtx, passExpected bool) error
	}{

		{
			name: "不支持的Http方法",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("Unknown255", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},

		// ==============创建考点测试============= coverage: 100%
		//       .o.       oooooooooo.   oooooooooo.
		//      .888.      `888'   `Y8b  `888'   `Y8b
		//     .8"888.      888      888  888      888
		//    .8' `888.     888      888  888      888
		//   .88ooo8888.    888      888  888      888
		//  .8'     `888.   888     d88'  888     d88'
		// o88o     o8888o o888bood8P'   o888bood8P'
		//
		//
		//
		{
			name: "创建考点成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点成功-缺少sever_host",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-缺少name",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-name类型为非字符串",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": 123,
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-admin为非数字类型",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(`{
					"data": {
						"name": "test-site",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": "testID"
					}
				}`)),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制开启事务失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制事务提交失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制事务回滚失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": %d,
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制读取Body失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制添加系统账号SQL Prepare 失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareCreateSysUserErr": fmt.Errorf("force add sys user sql prepare err"),
				},
			},
			passExpected: false,
			errWanted:    "force add sys user sql prepare err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制执行添加系统账号sql失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecCreateSysUserErr": fmt.Errorf("force execute add sys user sql err"),
				},
			},
			passExpected: false,
			errWanted:    "force execute add sys user sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制添加考点SQL Prepare 失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareErr": fmt.Errorf("force add exam site prepare err"),
				},
			},
			passExpected: false,
			errWanted:    "force add exam site prepare err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制执行添加考点sql失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr": fmt.Errorf("force execute add exam site sql err"),
				},
			},
			passExpected: false,
			errWanted:    "force execute add exam site sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-强制返回json Marshal失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},
		{
			name: "创建考点失败-没有创建权限",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "当前用户没有权限创建该数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultAddCheck,
		},

		// ======================================

		//   .oooooo.    oooooooooooo ooooooooooooo
		//  d8P'  `Y8b   `888'     `8 8'   888   `8
		// 888            888              888
		// 888            888oooo8         888
		// 888     ooooo  888    "         888
		// `88.    .88'   888       o      888
		//  `Y8bood8P'   o888ooooood8     o888o
		//
		//
		//
		{
			name: "获取考点成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						}
					}`, testID))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点失败-没有权限获取",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						}
					}`, testID))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
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
			errWanted:    "当前用户没有权限获取该数据",
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点失败-解析请求体数据失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						
					}`, testID))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点失败-无效的考点ID",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						}
					}`, -123))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			errWanted:    "无效的考点ID: -123, 请传入一个大于0的值",
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点失败-强制准备查询SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						}
					}`, testID))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareGetExamSiteSqlErr": fmt.Errorf("forced prepare query sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare query sql err",
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点失败-不存在的考点",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"data": {
							"examSiteID": %d
						}
					}`, testID+123))), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    sql.ErrNoRows.Error(),
			setup:        defaultSetup,
			check:        defaultGetCheck,
			cleanup:      defaultCleanup,
		},

		// ======================================

		// oooooooooooo oooooooooo.   ooooo ooooooooooooo
		// `888'     `8 `888'   `Y8b  `888' 8'   888   `8
		//  888          888      888  888       888
		//  888oooo8     888      888  888       888
		//  888    "     888      888  888       888
		//  888       o  888     d88'  888       888
		// o888ooooood8 o888bood8P'   o888o     o888o
		//
		//
		//
		{
			name: "编辑考点成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "重置考点访问密钥成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-没有权限编辑",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
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
			errWanted:    "当前用户没有权限修改该数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-解析请求体中的编辑信息失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-无效的考点ID",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, -123, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			errWanted:    "考点ID必须大于0, 当前传入: -123",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-强制准备更新考点信息SQL语句失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareErr": fmt.Errorf("forced prepare update exam site info err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare update exam site info err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-强制执行更新考点信息SQL语句失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr": fmt.Errorf("forced exec update exam site info err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec update exam site info err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-强制获取更新后的变化行数失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"rowsAffectedErr": fmt.Errorf("forced get affected rows after updated exam site info err"),
				},
			},
			passExpected: false,
			errWanted:    "forced get affected rows after updated exam site info err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-不存在的考点",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"name": "test-site-%d",
						"address": "test-site-addr",
						"serverHost": "t.test.top:6443",
						"admin": %d
					}
				}`, testID+123, nowTime, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("无权编辑该考点或该考点不存在(id: %d)", testID+123),
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-准备更新系统账号的user_token SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareErr2": fmt.Errorf("forced prepare update sys user token err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare update sys user token err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-执行更新系统账号的user_token SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr2": fmt.Errorf("forced exec update sys user token err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec update sys user token err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-获取更新系统账号的user_token的变化行数失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"rowsAffectedErr2": fmt.Errorf("forced get update sys user affected row err"),
				},
			},
			passExpected: false,
			errWanted:    "forced get update sys user affected row err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},
		{
			name: "编辑考点失败-系统账号不存在",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {

				err = defaultSetup()
				if err != nil {
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`WITH d_user_domain AS (
					DELETE FROM t_user_domain WHERE sys_user = %d
				)
				DELETE FROM t_user WHERE id = %d`, testID, testID))
				if err != nil {
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
			check:   defaultEditCheck,
		},
		{
			name: "编辑考点失败-系统账号不存在并强制创建新账号失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecCreateSysUserErr": fmt.Errorf("forced create new sys user error"),
				},
			},
			passExpected: false,
			errWanted:    "forced create new sys user error",
			setup: func() (err error) {

				err = defaultSetup()
				if err != nil {
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`WITH d_user_domain AS (
					DELETE FROM t_user_domain WHERE sys_user = %d
				)
				DELETE FROM t_user WHERE id = %d`, testID, testID))
				if err != nil {
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
			check:   defaultEditCheck,
		},
		{
			name: "编辑考点失败-系统账号不存在并强制准备更新考点系统账号SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareErr3": fmt.Errorf("forced prepare update exam site new sys user sql error"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare update exam site new sys user sql error",
			setup: func() (err error) {

				err = defaultSetup()
				if err != nil {
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`WITH d_user_domain AS (
					DELETE FROM t_user_domain WHERE sys_user = %d
				)
				DELETE FROM t_user WHERE id = %d`, testID, testID))
				if err != nil {
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
			check:   defaultEditCheck,
		},
		{
			name: "编辑考点失败-系统账号不存在并强制执行更新考点系统账号SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr3": fmt.Errorf("forced exec update exam site new sys user sql error"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec update exam site new sys user sql error",
			setup: func() (err error) {

				err = defaultSetup()
				if err != nil {
					return
				}

				_, err = dbConn.Exec(fmt.Sprintf(`WITH d_user_domain AS (
					DELETE FROM t_user_domain WHERE sys_user = %d
				)
				DELETE FROM t_user WHERE id = %d`, testID, testID))
				if err != nil {
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
			check:   defaultEditCheck,
		},
		{
			name: "编辑考点失败-强制Marshal返回数据失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("PATCH", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"id": %d,
						"resetAccessToken":true
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "PATCH",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"jsonMarshalErr": fmt.Errorf("forced json marshal err"),
				},
			},
			passExpected: false,
			errWanted:    "forced json marshal err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultEditCheck,
		},

		// ======================================

		// oooooooooo.   oooooooooooo ooooo        oooooooooooo ooooooooooooo oooooooooooo
		// `888'   `Y8b  `888'     `8 `888'        `888'     `8 8'   888   `8 `888'     `8
		//  888      888  888          888          888              888       888
		//  888      888  888oooo8     888          888oooo8         888       888oooo8
		//  888      888  888    "     888          888    "         888       888    "
		//  888     d88'  888       o  888       o  888       o      888       888       o
		// o888bood8P'   o888ooooood8 o888ooooood8 o888ooooood8     o888o     o888ooooood8
		//
		//
		{
			name: "删除考点成功-单个考点",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点成功-多个考点",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d, %d]
					}
				}`, testID, testID+1))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-没有删除权限",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d, %d]
					}
				}`, testID, testID+1))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
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
			errWanted:    "当前用户没有权限删除该数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-解析请求体失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d, %d]
					
				}`, testID, testID+1))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-缺少考点id",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(`{
					"data": {
						"ids": []
					}
				}`)),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
			errWanted:    "validation failed:Key: 'IDs' Error:Field validation for 'IDs' failed on the 'gt' tag",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-强制准备删除SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"prepareErr1": fmt.Errorf("forced prepare delete exam site sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare delete exam site sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-强制执行删除SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr1": fmt.Errorf("forced exec delete exam site sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec delete exam site sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-强制执行删除SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"sqlExecErr1": fmt.Errorf("forced exec delete exam site sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec delete exam site sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-强制获取删除后变化行数失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
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
					"rowsAffectedErr1": fmt.Errorf("forced get delete exam site affected rows err"),
				},
			},
			passExpected: false,
			errWanted:    "forced get delete exam site affected rows err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
		},
		{
			name: "删除考点失败-考点不存在",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site",
				},
				R: httptest.NewRequest("DELETE", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"ids": [%d]
					}
				}`, testID+123))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "DELETE",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "无权删除所选的部分/全部考点或不存在所选的部分/全部考点",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
			check:        defaultDeleteCheck,
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
				err := tt.setup()
				if err != nil {
					t.Errorf("failed to setup test: %v", err)
					return
				}
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
				err := tt.check(tt.q, tt.passExpected)
				if err != nil {
					t.Errorf("check failed")
					return
				}
			}

		})

	}

}

func TestExamSiteList(t *testing.T) {

	nowTime := time.Now().Unix()

	testUserID := nowTime

	defaultSetup := func() (err error) {

		for _, p := range permissions {
			err = addTestDomainApi("/api/exam-site/list", p, int64(cmn.CDomainAssessExamSiteAdmin))
			if err != nil {
				t.Errorf("failed to add domain api: %v", err)
				return
			}
		}

		err = addTestUser(testUserID, int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Errorf("failed to add test user: %v", err)
			return
		}

		dbConn := cmn.GetDbConn()

		siteSets := []string{}

		for i := 0; i < 10; i++ {
			siteSets = append(siteSets, fmt.Sprintf(`(%d, 'test-site-%d', %d, 'localhost','address', %d)`, nowTime+int64(i), nowTime, testUserID, testUserID))
		}

		roomSets := []string{}

		for i := 0; i < 10; i++ {
			for j := 0; j < i; j++ {
				roomSets = append(roomSets, fmt.Sprintf(`('test-room-%d', %d, %d)`, nowTime, nowTime+int64(i), testUserID))
			}
		}

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin) VALUES %s`, strings.Join(siteSets, ",")))
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
			return
		}

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator) VALUES %s`, strings.Join(roomSets, ",")))
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
			return
		}

		return
	}

	// defaultCheck := func() {

	// }

	defaultCleanup := func() {

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

		err = removeTestUser(testUserID)
		if err != nil {
			t.Fatalf("failed to remove test user: %v", err)
		}

		err = removeTestDomainApis(int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Fatalf("failed to remove test domain api: %v", err)
		}

	}

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func() error
		cleanup      func()
	}{

		{
			name: "获取考点列表失败-无效的HTTP方法",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("Unknown255", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},

		// ==============获取考点列表测试=============
		//   .oooooo.    oooooooooooo ooooooooooooo
		//  d8P'  `Y8b   `888'     `8 8'   888   `8
		// 888            888              888
		// 888            888oooo8         888
		// 888     ooooo  888    "         888
		// `88.    .88'   888       o      888
		//  `Y8bood8P'   o888ooooood8     o888o
		//
		//
		//
		{
			name: "获取考点列表成功-带筛选降序",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表成功-无筛选升序",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表成功-筛选结果为空",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"": "DESC"
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-没有可读权限",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessStudent)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessStudent)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
			},
			passExpected: false,
			errWanted:    "当前用户没有权限获取数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-无效的排序方式",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-无效的URL参数",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
						"pageSize": 10`,
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-页码小于1",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
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
					ID:   null.NewInt(testUserID, true),
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
			errWanted:    "页码不能小于1",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-每页条数小于1",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-页码类型为非数字",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制查询数据总行数SQL Prepare失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制查询数据总行数SQL 执行失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制获取查询数据总行数SQL Affected Rows失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制查询数据SQL Prepare失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制查询数据SQL 执行失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制Scan数据失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考点列表失败-强制Marshal返回的数据失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-site/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-site/list?q=%s`, url.QueryEscape(`{
						"page": 1,
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
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
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
				err := tt.setup()
				if err != nil {
					t.Errorf("failed to setup test: %v", err)
					return
				}
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

				if !item.ID.Valid {
					t.Errorf("expected exam site ID to be greater than 0, got %d", item.ID.Int64)
					return
				}

				if item.Name == "" {
					t.Errorf("expected exam site name to be not empty, got empty")
					return
				}

				if item.Address == "" {
					t.Errorf("expected exam site address to be not empty, got empty")
					return
				}

				if !item.ServerHost.Valid {
					t.Errorf("expected exam site server host to be not empty, got empty")
					return
				}

				if item.Admin <= 0 {
					t.Errorf("expected admin to be not empty, got empty")
					return
				}

				if !item.AdminName.Valid {
					t.Errorf("expected admin name to be not empty, got empty")
					return
				}

			}

		})

	}

}

func TestExamRoom(t *testing.T) {

	nowTime := time.Now().Unix()

	dbConn := cmn.GetDbConn()

	testUserID := nowTime

	defaultSetup := func() (err error) {

		for _, p := range permissions {
			err = addTestDomainApi("/api/exam-room", p, int64(cmn.CDomainAssessExamSiteAdmin))
			if err != nil {
				t.Errorf("failed to add domain api: %v", err)
				return
			}
		}

		err = addTestUser(testUserID, int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Errorf("failed to add test user: %v", err)
			return
		}

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, creator) VALUES (%d, %d)`, nowTime, testUserID))
		if err != nil {
			t.Errorf("failed create exam sit: %v", err)
			return
		}

		return
	}

	// defaultCheck := func() {

	// }

	defaultCleanup := func() {

		r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name='test-room-%d'`, nowTime))
		if err != nil {
			t.Fatalf("failed delete exam sit: %v", err)
			return
		}

		c, err := r.RowsAffected()
		if err != nil {
			t.Fatalf("failed get affected rows: %v", err)
			return
		}

		t.Logf("Have already cleaned up %d row from t_exam_room", c)

		r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE id=%d`, nowTime))
		if err != nil {
			t.Fatalf("failed delete exam sit: %v", err)
			return
		}

		c, err = r.RowsAffected()
		if err != nil {
			t.Fatalf("failed get affected rows: %v", err)
			return
		}

		t.Logf("Have already cleaned up %d row from t_exam_site", c)

		err = removeTestUser(testUserID)
		if err != nil {
			t.Fatalf("failed to remove test user: %v", err)
		}

		err = removeTestDomainApis(int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Fatalf("failed to remove test domain api: %v", err)
		}

	}

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func() error
		cleanup      func()
		check        func(q *cmn.ServiceCtx, passExpected bool) (err error)
	}{

		{
			name: "不支持的Http方法",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("Unknown255", "/api/exam-room", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 10,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-room",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "强制开启事务失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("Unknown255", "/api/exam-room", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 10,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-room",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"txBeginErr": fmt.Errorf("force begin tx err"),
				},
			},
			passExpected: false,
			errWanted:    "force begin tx err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "强制提交事务失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"commitErr": fmt.Errorf("force commit tx err"),
				},
			},
			passExpected: false,
			errWanted:    "force commit tx err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "强制解析请求体失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("Unknown255", "/api/exam-room", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 10,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-room",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"readBodyErr": fmt.Errorf("force unmarshal req body err"),
				},
			},
			passExpected: false,
			errWanted:    "force unmarshal req body err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "强制解析请求体失败并触发回滚事务失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("Unknown255", "/api/exam-room", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 10,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-room",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"readBodyErr": fmt.Errorf("force unmarshal req body err"),
					"rollbackErr": fmt.Errorf("force rollback tx err"),
				},
			},
			passExpected: false,
			errWanted:    "force rollback tx err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},

		// 添加考场测试
		//       .o.       oooooooooo.   oooooooooo.
		//      .888.      `888'   `Y8b  `888'   `Y8b
		//     .8"888.      888      888  888      888
		//    .8' `888.     888      888  888      888
		//   .88ooo8888.    888      888  888      888
		//  .8'     `888.   888     d88'  888     d88'
		// o88o     o8888o o888bood8P'   o888bood8P'
		//
		//
		//
		{
			name: "添加考场成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-没有权限",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
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
			errWanted:    "当前用户没有权限创建该数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-无权访问考点数据",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(99999999, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("无权在所选考点创建考场或所选考点不存在, id: %d", nowTime),
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-解析请求体Data失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "unexpected end of JSON input",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-缺少必要参数",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "validation failed:Key: 'examRoomInfo.ExamSiteID' Error:Field validation for 'ExamSiteID' failed on the 'required' tag",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-考场容量小于0",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": -30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "考场容量必须大于0",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-数据类型错误",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": "undefined"
					}
				}`, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "json: cannot unmarshal string into Go struct field examRoomInfo.examSiteID of type int",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-强制准备SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"prepareStmtErr": fmt.Errorf("force prepare sql err"),
				},
			},
			passExpected: false,
			errWanted:    "force prepare sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "添加考场失败-强制执行SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room",
				},
				R: httptest.NewRequest("POST", "/api/exam-site", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"name": "test-room-%d",
						"capacity": 30,
						"examSiteID": %d
					}
				}`, nowTime, nowTime))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site",
					Method: "POST",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"execSQLErr": fmt.Errorf("force exec sql err"),
				},
			},
			passExpected: false,
			errWanted:    "force exec sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
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
				err := tt.setup()
				if err != nil {
					t.Errorf("failed to setup test: %v", err)
					return
				}
			}

			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			ctx := context.WithValue(context.Background(), cmn.QNearKey, tt.q)

			examRoom(ctx)

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

func TestExamRoomList(t *testing.T) {

	nowTime := time.Now().Unix()

	dbConn := cmn.GetDbConn()

	testUserID := nowTime

	var cleanupTestData func() error

	defaultSetup := func() (err error) {

		for _, p := range permissions {
			err = addTestDomainApi("/api/exam-room/list", p, int64(cmn.CDomainAssessExamSiteAdmin))
			if err != nil {
				t.Errorf("failed to add domain api: %v", err)
				return
			}
		}

		err = addTestUser(testUserID, int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Errorf("failed to add test user: %v", err)
			return
		}

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name, creator) VALUES (%d, 'test-site-%d', %d)`, nowTime, nowTime, testUserID))
		if err != nil {
			t.Errorf("failed create exam sit: %v", err)
			return
		}

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_room (name, exam_site, creator, capacity) VALUES ('test-room-%d', %d, %d, %d)`, nowTime, nowTime, testUserID, 30))
		if err != nil {
			t.Errorf("failed create exam room: %v", err)
			return
		}

		cleanupTestData, err = mockExamSiteSyncData(testUserID, nowTime)
		if err != nil {
			t.Errorf("failed to mock exam site sync data: %v", err)
			return
		}

		return
	}

	// defaultCheck := func() {

	// }

	defaultCleanup := func() {

		r, err := dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_room WHERE name='test-room-%d'`, nowTime))
		if err != nil {
			t.Fatalf("failed delete exam sit: %v", err)
			return
		}

		c, err := r.RowsAffected()
		if err != nil {
			t.Fatalf("failed get affected rows: %v", err)
			return
		}

		t.Logf("Have already cleaned up %d row from t_exam_room", c)

		r, err = dbConn.Exec(fmt.Sprintf(`DELETE FROM t_exam_site WHERE id=%d`, nowTime))
		if err != nil {
			t.Fatalf("failed delete exam sit: %v", err)
			return
		}

		c, err = r.RowsAffected()
		if err != nil {
			t.Fatalf("failed get affected rows: %v", err)
			return
		}

		t.Logf("Have already cleaned up %d row from t_exam_site", c)

		err = removeTestUser(testUserID)
		if err != nil {
			t.Fatalf("failed to remove test user: %v", err)
		}

		err = removeTestDomainApis(int64(cmn.CDomainAssessExamSiteAdmin))
		if err != nil {
			t.Fatalf("failed to remove test domain api: %v", err)
		}

		if cleanupTestData != nil {
			err = cleanupTestData()
			if err != nil {
				t.Fatalf("failed to clean up test data: %v", err)
			}
		}

	}

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func() error
		check        func(q *cmn.ServiceCtx) (err error)
		cleanup      func()
	}{
		{
			name: "获取考场列表失败-无效的HTTP方法",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("Unknown255", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "Unknown255",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},

		// 获取考场列表测试
		//   .oooooo.    oooooooooooo ooooooooooooo
		//  d8P'  `Y8b   `888'     `8 8'   888   `8
		// 888            888              888
		// 888            888oooo8         888
		// 888     ooooo  888    "         888
		// `88.    .88'   888       o      888
		//  `Y8bood8P'   o888ooooood8     o888o
		//
		//
		//
		{
			name: "获取考场列表成功",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表成功-获取指定时间范围内可用的考场",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": 0
						},
						"filter": {
							"name": "",
							"available": true,
							"startTime": %d,
							"endTime":%d
						}
					}`, (nowTime+2*60)*1000, (nowTime+14*60)*1000),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1000, true),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				var d []examRoomInfo
				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if len(d) < 2 {
					err = fmt.Errorf("got %d, but greater than or equal to 2", len(d))
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "获取考场列表成功-获取所有可访问的考场",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"filter": {
							"startTime": %d,
							"endTime":%d
						}
					}`, 0, 0),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1000, true),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				var d []examRoomInfo
				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if len(d) < 2 {
					err = fmt.Errorf("got %d, but greater than or equal to 2", len(d))
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "获取考场列表失败-传入的开始时间大于结束时间",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": 0
						},
						"filter": {
							"name": "",
							"available": true,
							"startTime": %d,
							"endTime":%d
						}
					}`, (nowTime+14*60)*1000, (nowTime+2*60)*1000),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1000, true),
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
			errWanted:    fmt.Sprintf(`开始时间不能大于结束时间, 当前开始时间: %d, 结束时间: %d`, (nowTime+14*60)*1000, (nowTime+2*60)*1000),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				var d []examRoomInfo
				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if len(d) < 2 {
					err = fmt.Errorf("expected get 2 rooms data, got %d", len(d))
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "获取考场列表失败-未指定开始时间和结束时间",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": 0
						},
						"filter": {
							"name": "",
							"available": true,
							"startTime": %d,
							"endTime":%d
						}
					}`, 0, 0),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(1000, true),
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
			errWanted:    fmt.Sprintf(`当按可用状态获取时, 必须指定开始时间与结束时间, 当前开始时间: %d, 结束时间: %d`, 0, 0),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				var d []examRoomInfo
				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal response data: %v", err)
					return
				}

				if len(d) < 2 {
					err = fmt.Errorf("expected get 2 rooms data, got %d", len(d))
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "获取考场列表失败-无权获取",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessStudent), true),
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
			errWanted:    "当前无权获取考场列表数据",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制开启事务失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"txBeginErr": fmt.Errorf("forced begin tx err"),
				},
			},
			passExpected: false,
			errWanted:    "forced begin tx err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制读取失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"readBodyErr": fmt.Errorf("forced read body err"),
				},
			},
			passExpected: false,
			errWanted:    "forced read body err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制解析请求体失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制回滚失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"rollbackErr": fmt.Errorf("forced rollback err"),
				},
			},
			passExpected: false,
			errWanted:    "forced rollback err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制提交失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"commitErr": fmt.Errorf("forced commit err"),
				},
			},
			passExpected: false,
			errWanted:    "forced commit err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-当前无权获取该考点数据",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(99999999, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: true,
			errWanted:    "",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				var roomList []examRoomInfo

				err = json.Unmarshal(q.Msg.Data, &roomList)
				if err != nil {
					t.Error(err.Error())
					return
				}

				if len(roomList) != 0 {
					err = fmt.Errorf("room list got not empty, but empty")
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "获取考场列表失败-页码小于1",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": -5,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "页码不能小于1",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-页大小小于1",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": -10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "每页条数不能小于1",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-不支持的排序",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "UNKNOWN255"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
					Role: null.NewInt(int64(cmn.CDomainAssessExamSiteAdmin), true),
				},
				Domains: []cmn.TDomain{
					{
						ID:     null.IntFrom(int64(cmn.CDomainAssessExamSiteAdmin)),
						Domain: cmn.RoleName(cmn.CDomain(cmn.CDomainAssessExamSiteAdmin)),
					},
				},
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("不支持的排序方式: %s key: %s", "UNKNOWN255", "capacity"),
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制准备获取考场数据SQL失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"prepareSqlErr1": fmt.Errorf("forced prepare sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制执行获取考场总行数 SQL 失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"sqlExecErr1": fmt.Errorf("forced exec sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制获取考场数据SQL Affected Rows 失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制准备获取考场列表数据失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"prepareErr2": fmt.Errorf("forced prepare sql err"),
				},
			},
			passExpected: false,
			errWanted:    "forced prepare sql err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制执行查询获取考场列表数据 SQL 失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"queryErr": fmt.Errorf("forced exec sql query err"),
				},
			},
			passExpected: false,
			errWanted:    "forced exec sql query err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制执行查询获取考场列表数据 SQL 取值失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
					"scanErr": fmt.Errorf("forced scan data err"),
				},
			},
			passExpected: false,
			errWanted:    "forced scan data err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "获取考场列表失败-强制 Marshal 查询结果返回失败",
			q: &cmn.ServiceCtx{
				Ep: &cmn.ServeEndPoint{
					Path: "/api/exam-room/list",
				},
				R: httptest.NewRequest("GET", fmt.Sprintf(`/api/exam-room/list?q=%s`, url.QueryEscape(fmt.Sprintf(`{
						"page": 1,
						"pageSize": 10,
						"orderBy": [
							{
								"capacity": "DESC"
							}
						],
						"data": {
							"examSiteID": %d
						},
						"filter": {
							"name": "test"
						}
					}`, nowTime),
				)), nil),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/list",
					Method: "GET",
				},
				BeginTime: time.Now(),
				SysUser: &cmn.TUser{
					ID:   null.NewInt(testUserID, true),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				err := tt.setup()
				if err != nil {
					return
				}
			}

			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			ctx := context.WithValue(context.Background(), cmn.QNearKey, tt.q)

			examRoomList(ctx)

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
					t.Error("check did not passed")
				}
			}

		})

	}
}

func TestExamSiteSyncInit(t *testing.T) {

	nowTime := time.Now().Unix()

	dbConn := cmn.GetDbConn()

	var serverCtx *cmn.ServiceCtx

	var cleanupTestData func() error

	testUserID := nowTime / 1000

	defaultSetup := func() (err error) {

		viper.Set("examSiteServerSync.maxRetry", 0)

		viper.Set("examSiteServerSync.syncInterval", 3*60)

		viper.Set("examSiteServerSync.syncDelay", 5*60)

		_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
		if err != nil {
			t.Errorf("failed to set sync status: %v", err)
			return
		}

		err = addTestUser(testUserID, int64(cmn.CDomainAssessExamSite))
		if err != nil {
			t.Errorf("failed to add test user: %v", err)
			return
		}

		cleanupTestData, err = mockExamSiteSyncData(testUserID, nowTime)
		if err != nil {
			t.Errorf("failed to mock data: %v", err)
			return
		}

		return

	}

	defaultCheck := func(q *cmn.ServiceCtx) (err error) {
		return
	}

	defaultCleanup := func() {
		if pullChan != nil {
			close(pullChan)
		}

		if pushChan != nil {
			close(pushChan)
		}

		pullChan = nil
		pushChan = nil

		_, err := cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
		if err != nil {
			t.Fatalf("failed to set sync status: %v", err)
			return
		}

		if cleanupTestData != nil {
			err = cleanupTestData()
			if err != nil {
				t.Fatalf("failed to clean up test data: %v", err)
				return
			}
		}

		err = removeTestUser(testUserID)
		if err != nil {
			t.Fatalf("failed to remove test user: %v", err)
		}

	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var respBody cmn.ReplyProto

		// body, err := io.ReadAll(r.Body)
		// if err != nil {
		// 	t.Errorf("failed to read request body: %v", err)
		// 	return
		// }

		q := &cmn.ServiceCtx{
			SysUser: &cmn.TUser{
				ID: null.IntFrom(testUserID),
			},
			R:           r,
			W:           w,
			Msg:         &cmn.ReplyProto{},
			RedisClient: cmn.GetRedisConn(),
		}

		if serverCtx != nil {
			q = serverCtx
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

			respBody.Status = 0

			respBody.Data, err = json.Marshal(cmn.TUser{
				ID: null.IntFrom(testUserID),
			})
			if err != nil {
				respBody.Status = -1
				respBody.Msg = "failed to marshal user data: " + err.Error()
			}

		case "/api/exam-site/sync":

			centralServerUrl = ""

			examSiteSync(ctx)

			centralServerUrl = viper.GetString("examSiteServerSync.centralServerUrl")

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
		setup        func() error
		check        func(q *cmn.ServiceCtx) (err error)
		cleanup      func()
	}{

		//     .    o8o
		//   .o8    `"'
		// .o888oo oooo  ooo. .oo.  .oo.    .ooooo.  oooo d8b
		//   888   `888  `888P"Y88bP"Y88b  d88' `88b `888""8P
		//   888    888   888   888   888  888ooo888  888
		//   888 .  888   888   888   888  888    .o  888
		//   "888" o888o o888o o888o o888o `Y8bod8P' d888b
		//
		//
		//
		{
			name: "定时同步成功-执行推送成功",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
					"pushDone":     make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = dbConn.Exec(`UPDATE t_exam_info SET status = '06'`)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["pushDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "定时同步成功-执行拉取成功",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
					"pullDone":     make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					return
				}

				_, err = dbConn.Exec(`UPDATE t_exam_info SET status = '06'`)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "定时同步-当前仍有考试未结束",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":    make(chan int),
					"haveOngoingExam": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = dbConn.Exec(`UPDATE t_exam_session SET status = '04'`)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["haveOngoingExam"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "定时同步-当前有临近开始的考试",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":    make(chan int),
					"haveOngoingExam": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = dbConn.Exec(`UPDATE t_exam_session SET start_time = $1, status = '02'`, time.Now().Add(5*time.Minute).Unix()*1000)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["haveOngoingExam"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "定时同步失败-强制获取尚未结束的考试失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":        make(chan int),
					"queryOngoingExamErr": fmt.Errorf("forced query ongoing exam err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced query ongoing exam err",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = dbConn.Exec(`UPDATE t_exam_info SET status = '04'`)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "定时同步失败-强制获取当前同步状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":            make(chan int),
					"getSyncStatusErrInTimer": fmt.Errorf("forced get sync status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced get sync status err",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.syncInterval", 1)

				viper.Set("examSiteServerSync.syncDelay", 1)

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = dbConn.Exec(`UPDATE t_exam_info SET status = '06'`)
				if err != nil {
					t.Error(err.Error())
					return
				}

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},

		// ooooo              o8o      .
		// `888'              `"'    .o8
		//  888  ooo. .oo.   oooo  .o888oo
		//  888  `888P"Y88b  `888    888
		//  888   888   888   888    888
		//  888   888   888   888    888 .
		// o888o o888o o888o o888o   "888"
		//
		//
		//
		{
			name: "同步初始化成功-中心服务器",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {
				err = defaultSetup()
				if err != nil {
					return
				}

				viper.Set("examSiteServerSync.centralServerUrl", "")

				return
			},
			check: defaultCheck,
			cleanup: func() {
				defaultCleanup()
				viper.Set("examSiteServerSync.centralServerUrl", server.URL)
			},
		},
		{
			name: "同步初始化成功-考点服务器",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化成功后发送拉取通知成功",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化成功并完成一次完整的同步",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int, 1),
					"pullDone":     make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Errorf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				testID := nowTime / 100

				// 更新考场记录
				_, err = dbConn.Exec(`UPDATE t_exam_record SET basic_eval = '02' WHERE exam_session = $1`, testID)
				if err != nil {
					t.Errorf("failed to update exam record: %v", err)
					return
				}

				// 更新考生信息
				_, err = dbConn.Exec(`UPDATE t_examinee SET status = '10' WHERE exam_session_id = $1`, testID)
				if err != nil {
					t.Errorf("failed to update examinee: %v", err)
					return
				}

				// 更新考生作答数据
				_, err = dbConn.Exec(`UPDATE t_student_answers SET answer = '["A"]' WHERE examinee_id = $1`, testID+1)
				if err != nil {
					t.Errorf("failed to update student answers: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				// 检查是否有相应的更新
				c := 0
				err = dbConn.QueryRow(`SELECT COUNT(id) FROM t_exam_record WHERE basic_eval = '02' AND exam_session = $1`, testID).Scan(&c)
				if err != nil {
					t.Errorf("failed to query t_exam_record: %v", err)
					return
				}

				if c != 3 {
					err = fmt.Errorf("unexpected query t_exam_record, expect 3")
					t.Error(err.Error())
					return
				}

				err = dbConn.QueryRow(`SELECT COUNT(id) FROM t_examinee WHERE status = '10' AND exam_session_id = $1`, testID).Scan(&c)
				if err != nil {
					t.Errorf("failed to query t_examinee: %v", err)
					return
				}

				if c != 3 {
					err = fmt.Errorf("unexpected query t_examinee, expect 3")
					t.Error(err.Error())
					return
				}

				err = dbConn.QueryRow(`SELECT COUNT(id) FROM t_student_answers WHERE answer = '["A"]' AND examinee_id = $1`, testID+1).Scan(&c)
				if err != nil {
					t.Errorf("failed to query t_student_answer: %v", err)
					return
				}

				if c < 1 {
					err = fmt.Errorf("unexpected query t_student_answers, expect bigger than 1")
					t.Error(err.Error())
					return
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化失败-强制创建 .pgpass 文件失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":    make(chan int, 1),
					"pullDone":        make(chan int),
					"createPgpassErr": fmt.Errorf("forced create .pgpass file err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced create .pgpass file err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-强制写入 .pgpass 文件失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":   make(chan int, 1),
					"pullDone":       make(chan int),
					"writePgpassErr": fmt.Errorf("forced write .pgpass file err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced write .pgpass file err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-强制更改 .pgpass 文件权限失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":   make(chan int, 1),
					"pullDone":       make(chan int),
					"chmodPgpassErr": fmt.Errorf("forced chmod .pgpass file err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced chmod .pgpass file err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-强制关闭 .pgpass 文件失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":   make(chan int, 1),
					"pullDone":       make(chan int),
					"closePgpassErr": fmt.Errorf("forced close .pgpass file err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced close .pgpass file err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-强制获取同步状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":     make(chan int, 1),
					"pullDone":         make(chan int),
					"getSyncStatusErr": fmt.Errorf("forced get sync status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced get sync status err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-强制设置同步状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":     make(chan int, 1),
					"pullDone":         make(chan int),
					"setSyncStatusErr": fmt.Errorf("forced set sync status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set sync status err",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化失败-在Pull中获取同步状态Key值失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":           make(chan int, 1),
					"pullDone":               make(chan int),
					"getSyncStatusErrInPull": fmt.Errorf("forced get sync status err in pull"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced get sync status err in pull",
			setup:        defaultSetup,
			check:        defaultCheck,
			cleanup:      defaultCleanup,
		},
		{
			name: "同步初始化成功-初始化前处于 PULLING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
				Msg:         &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {

				viper.Set("examSiteServerSync.maxRetry", 0)

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				return

			},
			check: func(q *cmn.ServiceCtx) (err error) {

				r, err := cmn.GetRedisConn().Get(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				if r != PULLED {
					err = fmt.Errorf("expect syncStatus is %s, got %s", PULLED, r)
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化失败-初始化前处于 PULLED 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
				Msg:         &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前数据尚未推送, 请先进行推送",
			setup: func() (err error) {

				viper.Set("examSiteServerSync.maxRetry", 0)

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				return

			},
			check: func(q *cmn.ServiceCtx) (err error) {

				r, err := cmn.GetRedisConn().Get(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				if r != PULLED {
					err = fmt.Errorf("expect syncStatus is %s, got %s", PULLED, r)
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化成功-初始化前处于 PUSHING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag:         map[string]interface{}{},
				Msg:         &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup: func() (err error) {

				viper.Set("examSiteServerSync.maxRetry", 0)

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				return

			},
			check: func(q *cmn.ServiceCtx) (err error) {

				r, err := cmn.GetRedisConn().Get(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				if r != PUSHED {
					err = fmt.Errorf("expected get syncStatus is %s, got %s", PUSHED, r)
				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化失败-初始化前处于 PULLING 状态但设置为 PUSHED 状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"setPushedStatusErr": fmt.Errorf("forced set pushed status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pushed status err",
			setup: func() (err error) {

				viper.Set("examSiteServerSync.maxRetry", 0)

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				return
			},
			check:   defaultCheck,
			cleanup: defaultCleanup,
		},
		{
			name: "同步初始化失败-初始化前处于 PUSHING 状态但设置为 PULLED 状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"setPulledStatusErr": fmt.Errorf("forced set pulled status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pulled status err",
			setup: func() (err error) {

				viper.Set("examSiteServerSync.maxRetry", 0)

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				return

			},
			check:   defaultCheck,
			cleanup: defaultCleanup,
		},
		{
			name: "强制关闭Pull管道和Push管道",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"closeChan": make(chan int, 1),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: true,
			errWanted:    "",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				close(pullChan)

				close(pushChan)

				<-q.Tag["closeChan"].(chan int)

				t.Log("got end msg listen")

				return
			},
			cleanup: defaultCleanup,
		},

		// ooooooooo.               oooo  oooo
		// `888   `Y88.             `888  `888
		//  888   .d88' oooo  oooo   888   888
		//  888ooo88P'  `888  `888   888   888
		//  888          888   888   888   888
		//  888          888   888   888   888
		// o888o         `V88V"V8P' o888o o888o
		//
		//
		//
		{
			name: "Pull拉取数据同步失败-当前处于 PULLING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前正在拉取数据中, 不允许重复拉取",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-当前处于 PULLED 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前数据尚未推送, 请先进行推送",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-当前处于 PUSHING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone": make(chan int),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前正在推送数据中, 不允许进行拉取",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制 PULLING 状态设置失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                  make(chan int),
					"setPullingStatusErrInPull": fmt.Errorf("forced set pulling status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pulling status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制 PULLED 状态设置失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                 make(chan int),
					"setPulledStatusErrInPull": fmt.Errorf("forced set pulled status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pulled status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制 rsync 失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":       make(chan int),
					"rsyncErrInPull": fmt.Errorf("forced rsync err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced rsync err", ""),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制 PUSHED 状态设置失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                 make(chan int),
					"rsyncErrInPull":           fmt.Errorf("forced rsync err in pull"),
					"setPushedStatusErrInPull": fmt.Errorf("forced set pushed status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pushed status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制登录请求发送错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":    make(chan int),
					"loginReqErr": fmt.Errorf("forced login req err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced login req err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制登录请求体解析错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":              make(chan int),
					"loginBodyUnmarshalErr": fmt.Errorf("forced login body unmarshal err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced login body unmarshal err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制登录请求状态错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":           make(chan int),
					"loginRespStatusErr": fmt.Errorf("forced login status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced login status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制登录请求体Data解析错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                  make(chan int),
					"loginRespDataUnmarshalErr": fmt.Errorf("forced login resp data unmarshal err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced login resp data unmarshal err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制同步请求体解析错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                 make(chan int),
					"pullRespBodyUnmarshalErr": fmt.Errorf("forced sync resp body unmarshal err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced sync resp body unmarshal err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制同步请求状态错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":          make(chan int),
					"pullRespStatusErr": fmt.Errorf("forced sync resp status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced sync resp status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制同步请求体 data 解析错误",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                 make(chan int),
					"pullRespDataUnmarshalErr": fmt.Errorf("forced sync resp data unmarshal err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced sync resp data unmarshal err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制设置同步信息快照失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                   make(chan int),
					"pullSetSyncInfoSnapShotErr": fmt.Errorf("forced set sync info snapshot err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set sync info snapshot err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制创建同步数据报错目录失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":               make(chan int),
					"pullMkdirAllDestDirErr": fmt.Errorf("forced create sync data error dir err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced create sync data error dir err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制清除已同步的数据目录失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                make(chan int),
					"pullRemoveAllDestDirErr": fmt.Errorf("forced remove all destination dir err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced remove all destination dir err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制创建模式失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":            make(chan int),
					"pullCreateSchemaErr": fmt.Errorf("forced create schema err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced create schema err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},

		{
			name: "Pull拉取数据同步失败-强制执行 pg_restore 失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":           make(chan int),
					"pgRestoreErrInPull": fmt.Errorf("forced pg_restore err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced pg_restore err", ""),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Pull拉取数据同步失败-强制执行 psql 导入脚本失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"pullDone":                  make(chan int),
					"psqlImportScriptInPullErr": fmt.Errorf("forced psql import script err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced psql import script err", ""),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				maxRetry = 1

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPullMsg()

				<-q.Tag["pullDone"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},

		// ooooooooo.                        oooo
		// `888   `Y88.                      `888
		//  888   .d88' oooo  oooo   .oooo.o  888 .oo.
		//  888ooo88P'  `888  `888  d88(  "8  888P"Y88b
		//  888          888   888  `"Y88b.   888   888
		//  888          888   888  o.  )88b  888   888
		// o888o         `V88V"V8P' 8""888P' o888o o888o
		//
		//
		//
		{
			name: "Push推送数据同步失败-强制获取同步状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":           make(chan int, 1),
					"getSyncStatusErrInPush": fmt.Errorf("forced get sync status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced get sync status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-当前处于 PULLING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int, 1),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前正在拉取数据中, 不允许进行推送",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PULLING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-当前处于 PUSHING 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int, 1),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前正在推送数据中，不允许重复推送",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHING, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-当前处于 PUSHED 状态",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen": make(chan int, 1),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "当前不允许推送数据,请先进行拉取同步数据",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Set(context.Background(), SyncStatusKey, PUSHED, 0).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制设置 PUSHING 失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":              make(chan int, 1),
					"SetPushingStatusErrInPush": fmt.Errorf("forced set pushing status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pushing status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制登录失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":              make(chan int, 1),
					"loginRespDataUnmarshalErr": fmt.Errorf("forced login resp data unmarshal err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced login resp data unmarshal err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制设置 PULLED 状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":              make(chan int, 1),
					"loginRespDataUnmarshalErr": fmt.Errorf("forced login resp data unmarshal err"),
					"setPulledStatusErrInPush":  fmt.Errorf("forced set pulled status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pulled status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制设置 PUSHED 状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":             make(chan int, 1),
					"setPushedStatusErrInPush": fmt.Errorf("forced set pushed status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced set pushed status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制获取同步信息快照失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":         make(chan int, 1),
					"getSyncInfoErrInPush": fmt.Errorf("forced get sync info err: %w", redis.Nil),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("没有找到同步信息, 请先进行拉取操作, err: %s", fmt.Errorf("forced get sync info err: %w", redis.Nil).Error()),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制解析同步信息快照失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":               make(chan int, 1),
					"unmarshalSyncInfoErrInPush": fmt.Errorf("forced unmarshal sync info err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced unmarshal sync info err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制创建源目录失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":         make(chan int, 1),
					"mkdirAllSourceInPush": fmt.Errorf("forced mkdirAll source dir err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced mkdirAll source dir err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制移除源目录失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":          make(chan int, 1),
					"removeAllSourceInPush": fmt.Errorf("forced removeAll source dir err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced removeAll source dir err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制执行 psql 导出脚本失败 ",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":              make(chan int, 1),
					"psqlExportScriptErrInPush": fmt.Errorf("forced psql export script err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    fmt.Sprintf("COMMAND: %s\t ERR: %s\t DETAIL: %s", "", "forced psql export script err", ""),
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制发送同步请求失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":     make(chan int, 1),
					"syncReqErrInPush": fmt.Errorf("forced send sync req err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced send sync req err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制解析同步响应体失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":                  make(chan int, 1),
					"syncReqBodyUnmarshalErrInPush": fmt.Errorf("forced unmarshal sync resp err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced unmarshal sync resp err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制同步响应状态失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":           make(chan int, 1),
					"syncReqStatusErrInPush": fmt.Errorf("forced sync resp status err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced sync resp status err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "Push推送数据同步失败-强制删除同步信息快照失败",
			q: &cmn.ServiceCtx{
				RedisClient: cmn.GetRedisConn(),
				Tag: map[string]interface{}{
					"endMsgListen":                 make(chan int, 1),
					"delSyncInfoSnapShotErrInPush": fmt.Errorf("forced delete sync info snapshot err"),
				},
				Msg: &cmn.ReplyProto{},
			},
			passExpected: false,
			errWanted:    "forced delete sync info snapshot err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {

				_, err = cmn.GetRedisConn().Del(context.Background(), SyncStatusKey).Result()
				if err != nil {
					t.Fatalf("failed to set sync status: %v", err)
					return
				}

				SendPushMsg()

				<-q.Tag["endMsgListen"].(chan int)

				return
			},
			cleanup: defaultCleanup,
		},
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
				err := tt.setup()
				if err != nil {
					return
				}
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

	dbConn := cmn.GetDbConn()

	defaultSetup := func() (err error) {

		_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
			(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
			return
		}

		return
	}

	// defaultCheck := func(q *cmn.ServiceCtx) (err error) {

	// 	return
	// }

	defaultCleanup := func() {

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
	}

	tests := []struct {
		name         string
		q            *cmn.ServiceCtx
		passExpected bool
		errWanted    string
		setup        func() error
		check        func(q *cmn.ServiceCtx) (err error)
		cleanup      func()
	}{

		{
			name: "考点同步失败-无效的HTTP方法",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("Unknown255", "/api/exam-site/sync", nil),
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
			setup: func() (err error) {
				return
			},
			cleanup: func() {
			},
		},
		{
			name: "准备数据失败-强制读取请求体失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
					"readBodyErr": fmt.Errorf("forced read body err"),
				},
			},
			passExpected: false,
			errWanted:    "forced read body err",
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},

		// ===========准备考点数据测试============
		//   .oooooo.    oooooooooooo ooooooooooooo
		//  d8P'  `Y8b   `888'     `8 8'   888   `8
		// 888            888              888
		// 888            888oooo8         888
		// 888     ooooo  888    "         888
		// `88.    .88'   888       o      888
		//  `Y8bood8P'   o888ooooood8     o888o
		{
			name: "准备数据成功",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			cleanup:      defaultCleanup,
		},
		{
			name: "准备数据失败-无效的账号ID",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制获取同步信息快照失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
					"getSyncInfoSnapshotErr": fmt.Errorf("forced get sync info snapshot err"),
				},
			},
			passExpected: false,
			errWanted:    "forced get sync info snapshot err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制创建临时目录失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制执行 pg_dump 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制执行 psql 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制json marshal 失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制清除临时目录失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制创建导出脚本失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制写入导出脚本失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-强制保存同步信息快照失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
					"saveSyncInfoSnapshotErr": fmt.Errorf("forced save sync info snapshot err"),
				},
			},
			passExpected: false,
			errWanted:    "forced save sync info snapshot err",
			setup:        defaultSetup,
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: defaultCleanup,
		},
		{
			name: "准备数据失败-无效的同步信息快照",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("GET", "/api/exam-site/sync", nil),
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
				Tag:         map[string]interface{}{},
			},
			passExpected: false,
			errWanted:    "unexpected end of JSON input",
			setup: func() (err error) {

				redisConn := cmn.GetRedisConn()

				_, err = redisConn.Set(context.Background(), fmt.Sprintf("%s:%d", ExamSiteSyncPrefix, 2025), "{", 0).Result()
				if err != nil {
					t.Fatalf("failed to set redis key: %v", err)
					return
				}

				dbConn := cmn.GetDbConn()

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				return
			},
			check: func(q *cmn.ServiceCtx) (err error) {
				d := syncInfo{
					Path:          "",
					TableFileList: []string{},
				}

				err = json.Unmarshal(q.Msg.Data, &d)
				if err != nil {
					t.Errorf("failed to unmarshal data: %v", err)
					return
				}

				// 检查目录下是否有相应的文件
				for _, f := range d.TableFileList {

					_, err = os.Stat(filepath.Join(d.Path, f))
					if err != nil {
						t.Errorf("failed to stat file while it shouldn't: %v", err)
						return
					}

				}

				return
			},
			cleanup: func() {

				redisConn := cmn.GetRedisConn()

				_, err := redisConn.Del(context.Background(), fmt.Sprintf("%s:%d", ExamSiteSyncPrefix, 2025)).Result()
				if err != nil {
					t.Fatalf("failed to delete redis key: %v", err)
					return
				}

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

		// =============反向同步测试==============
		// ooooooooo.     .oooooo.    .oooooo..o ooooooooooooo
		// `888   `Y88.  d8P'  `Y8b  d8P'    `Y8 8'   888   `8
		//  888   .d88' 888      888 Y88bo.           888
		//  888ooo88P'  888      888  `"Y8888o.       888
		//  888         888      888      `"Y88b      888
		//  888         `88b    d88' oo     .d8P      888
		// o888o         `Y8bood8P'  8""88888P'      o888o
		//
		{
			name: "反向同步成功",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": []
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-无效的请求data",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": 123
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
			errWanted:    "json: cannot unmarshal number into Go struct field syncInfo.tableFileList of type []string",
			setup: func() (err error) {

				dbConn := cmn.GetDbConn()

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-请求data缺少必要数据",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s"
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
			errWanted:    "validation failed:Key: 'syncInfo.TableFileList' Error:Field validation for 'TableFileList' failed on the 'required' tag",
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-请求data中的path为不存在",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": []
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp/123123/456456")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
			errWanted:    fmt.Sprintf("open %s: no such file or directory", filepath.Join(os.Getenv("PWD"), "data/tmp/123123/456456/import_script.sql")),
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-强制执行psql失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": []
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-强制删除同步信息快照key失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": []
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
					"deleteKeyErr": fmt.Errorf("forced delete key err"),
				},
			},
			passExpected: false,
			errWanted:    "forced delete key err",
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
					return
				}

			},
		},
		{
			name: "反向同步失败-强制删除已同步的数据所在目录失败",
			q: &cmn.ServiceCtx{
				R: httptest.NewRequest("POST", "/api/exam-site/sync", strings.NewReader(fmt.Sprintf(`{
					"data": {
						"path": "%s",
						"tableFileList": []
					}
				}`, filepath.Join(os.Getenv("PWD"), "data/tmp")))),
				W: httptest.NewRecorder(),
				Msg: &cmn.ReplyProto{
					API:    "/api/exam-site/sync",
					Method: "POST",
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
					"removeAllErr": fmt.Errorf("forced remove all err"),
				},
			},
			passExpected: false,
			errWanted:    "forced remove all err",
			setup: func() (err error) {

				_, err = dbConn.Exec(fmt.Sprintf(`INSERT INTO t_exam_site (id, name,creator,server_host,address,admin,sys_user) VALUES 
					(%d,'test-site-%d', 1622, 'localhost','address','1622','2025')`, nowTime, nowTime))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
					return
				}

				folderPath := filepath.Join(os.Getenv("PWD"), "data/tmp")

				err = os.MkdirAll(folderPath, 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(folderPath, "import_script.sql"), []byte("SELECT * FROM t_exam_site"), 0755)
				if err != nil {
					t.Fatalf("failed to create test sql file: %v", err)
					return
				}

				return
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

				filePath := filepath.Join(os.Getenv("PWD"), "import_script.sql")

				err = os.RemoveAll(filePath)
				if err != nil {
					t.Fatalf("failed to remove test sql file: %v", err)
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

			oldCentralServerUrl := centralServerUrl

			centralServerUrl = ""

			if tt.setup != nil {
				err := tt.setup()
				if err != nil {
					t.Error(err)
					return
				}
			}

			defer func() {
				centralServerUrl = oldCentralServerUrl
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

		})

	}
}
