package exam_site

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"crypto/rand"


	"w2w.io/serve/auth_mgt"
	"w2w.io/cmn"
)

const (
	SubServer = "SubServer"
)

type copyInfo struct {
	Sql   string
	Table string
}

// OrderByJoin 将返回拼接好的 Order By 语句, 如果传入的 map 为空, 则返回默认值 defaultOrderBy
func OrderByJoin(defaultOrderBy string, orderByMap []map[string]string) (r string, err error) {

	orderByList := []string{}

	for _, o := range orderByMap {
		for k, v := range o {

			if k == "" || v == "" {
				continue
			}

			v = strings.ToUpper(v)

			if v != "ASC" && v != "DESC" {
				err = fmt.Errorf("不支持的排序方式: %s key: %s", v, k)
				z.Error(err.Error())
				return
			}

			orderByList = append(orderByList, fmt.Sprintf("%s %s", k, v))
		}
	}

	r = strings.Join(orderByList, ", ")
	if r == "" {
		r = defaultOrderBy
	}

	return
}

// getApiPermissions 获取当前用户在使用指定接口时是否可读/可创建/可编辑/可删除
func getApiPermissions(ctx context.Context, apiPath string) (readable, creatable, editable, deletable bool) {

	q := cmn.GetCtxValue(ctx)

	readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionRead)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiReadableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiReadableErr"].(error)
		}

		return
	}

	creatable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionCreate)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiCreatableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiCreatableErr"].(error)
		}

		return
	}

	editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionUpdate)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiEditableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiEditableErr"].(error)
		}

		return
	}

	deletable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, apiPath, auth_mgt.CAPIAccessActionDelete)
	if q.Err != nil || (cmn.InDebugMode && q.Tag["checkUserApiDeletableErr"] != nil) {

		if q.Err == nil {
			q.Err = q.Tag["checkUserApiDeletableErr"].(error)
		}

		return
	}

	return
}

// createSysUser 创建考点服务器系统账号
func createSysUser(ctx context.Context, tx *sql.Tx, siteID int64) (sysUserID int64, accessToken string, err error) {

	q := cmn.GetCtxValue(ctx)

	officialName := fmt.Sprintf("考点%d", siteID)

	var account, userToken string

	b1 := make([]byte, 4)

	b2 := make([]byte, 32)

	// 该Read从不返回错误，并且始终填充 b, 一旦发生错误就直接Panic， 所以这里就不需要接收err
	rand.Read(b1)

	rand.Read(b2)

	account = fmt.Sprintf("exam-site-%x", b1)

	userToken = fmt.Sprintf("%x", b2)

	// 注册考点服务器系统账号，用于给考点服务器与中心服务器进行http通信验证
	sqlStr := fmt.Sprintf(`INSERT INTO t_user (category, type, official_name, account, user_token, creator, role)
	VALUES ('sys^admin', '08', $1, $2, crypt($3,gen_salt('bf')), 1000, %d)
	RETURNING 
		id`, cmn.CDomainAssessExamSite)

	var stmt1 *sql.Stmt
	stmt1, err = tx.Prepare(sqlStr)
	if err != nil || (cmn.InDebugMode && q.Tag["prepareCreateSysUserErr"] != nil) {

		if err == nil {
			err = q.Tag["prepareCreateSysUserErr"].(error)
		}

		z.Error(err.Error())
		return
	}

	defer stmt1.Close()

	err = stmt1.QueryRowContext(ctx, officialName, account, userToken).Scan(&sysUserID)
	if err != nil || (cmn.InDebugMode && q.Tag["sqlExecCreateSysUserErr"] != nil) {

		if err == nil {
			err = q.Tag["sqlExecCreateSysUserErr"].(error)
		}

		z.Error(err.Error())
		return
	}

	accessToken = generateAccessToken(sysUserID, userToken)

	return
}

// generateAccessToken 生成访问令牌
func generateAccessToken(userID int64, userToken string) (accessToken string) {
	return fmt.Sprintf("%d-%s", userID, userToken)
}

// parseAccessToken 解析访问令牌
func parseAccessToken(accessToken string) (userID int64, userToken string, err error) {

	parts := strings.SplitN(accessToken, "-", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("invalid access token")
		z.Error(err.Error())
		return
	}

	userID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		z.Error(err.Error())
		return
	}

	userToken = parts[1]
	return
}

// generateExportScript 生成导出脚本文件
// sysUser 为考点服务器系统账号ID
// destDir 为数据保存目录
// fileName 为脚本文件名
// isSubServerSide 为服务器端类型
func generateExportScript(sysUser int64, destDir string, fileName string, isSubServerSide bool) (recentExamID int64, tableFileList []string, err error) {

	if sysUser <= 0 {
		err = fmt.Errorf("invalid sysUser: %d", sysUser)
		z.Error(err.Error())
		return
	}

	dbConn := cmn.GetDbConn()

	// 考点服务器数据导出
	exportInfo := []copyInfo{

		//======考生数据======
		{
			Sql: fmt.Sprintf(`SELECT t_examinee.*
FROM t_examinee
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
WHERE t_exam_site.sys_user = %d`, sysUser),
			Table: "t_examinee",
		},

		//=====考生作答数据=====
		{
			Sql: fmt.Sprintf(`SELECT t_student_answers.*
FROM t_student_answers
	JOIN t_examinee ON t_examinee.id = t_student_answers.examinee_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
WHERE t_exam_site.sys_user = %d`, sysUser),
			Table: "t_student_answers",
		},

		// 作答上传的附件
		{
			Sql: fmt.Sprintf(`SELECT t_file.*
FROM t_student_answers
	JOIN t_examinee ON t_examinee.id = t_student_answers.examinee_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	CROSS JOIN LATERAL jsonb_array_elements(t_student_answers.files) AS file
	JOIN t_file ON t_file.id = file.value::int
WHERE jsonb_typeof(files) = 'array' AND t_exam_site.sys_user = %d
GROUP BY
	t_file.id`, sysUser),
			Table: "t_file",
		},

		//=======监考数据=======
		// 考场记录
		{
			Sql: fmt.Sprintf(`SELECT t_exam_record.*
FROM t_exam_record
	JOIN t_exam_room ON t_exam_room.id = t_exam_record.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
WHERE t_exam_site.sys_user = %d`, sysUser),
			Table: "t_exam_record",
		},

		// 考场附件
		{
			Sql: fmt.Sprintf(`SELECT t_file.* 
FROM t_exam_record
	JOIN t_exam_room ON t_exam_room.id = t_exam_record.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
  	CROSS JOIN LATERAL jsonb_array_elements(t_exam_record.files) AS file
	JOIN t_file ON t_file.id = file.value::int
WHERE jsonb_typeof(files) = 'array' AND t_exam_site.sys_user = %d
GROUP BY
	t_file.id`, sysUser),
			Table: "t_file",
		},

	}

	// 中心服务器数据导出
	if !isSubServerSide {

		// 获取最近一个未开始的考试ID

		s := `SELECT 
		t_exam_info.id
	FROM t_exam_info
		JOIN t_exam_session ON ((EXTRACT(epoch FROM CURRENT_TIMESTAMP) * (1000)::numeric))::bigint < t_exam_session.start_time AND t_exam_session.exam_id = t_exam_info.id 
	WHERE t_exam_info.mode = '02' AND t_exam_info.status = '02'
	GROUP BY
		t_exam_info.id
	ORDER BY MIN(t_exam_session.start_time) ASC
	LIMIT 1`

		err = dbConn.QueryRow(s).Scan(&recentExamID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			z.Error(err.Error())
			return
		}

		exportInfo = []copyInfo{

			//====系统基本数据====
			{
				Sql:   `SELECT t_domain.* FROM t_domain`,
				Table: "t_domain",
			},
			{
				Sql:   `SELECT t_api.* FROM t_api`,
				Table: "t_api",
			},
			{
				Sql:   `SELECT t_domain_api.* FROM t_domain_api`,
				Table: "t_domain_api",
			},
			//===================

			// 考点系统账号数据
			{
				Sql:   fmt.Sprintf(`SELECT t_user.* FROM t_user WHERE t_user.id = %d`, sysUser),
				Table: "t_user",
			},
			{
				Sql: fmt.Sprintf(`SELECT t_user_domain.* FROM t_user_domain WHERE t_user_domain.sys_user = %d
				`, sysUser),
				Table: "t_user_domain",
			},

			// 考点数据
			{
				Sql:   fmt.Sprintf(`SELECT t_exam_site.* FROM t_exam_site WHERE sys_user=%d`, sysUser),
				Table: "t_exam_site",
			},

			// 考场数据
			{
				Sql: fmt.Sprintf(`SELECT t_exam_room.* FROM t_exam_room
JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
WHERE t_exam_site.sys_user = %d`, sysUser),
				Table: "t_exam_room",
			},

			// 考点负责人账号数据
			{
				Sql: fmt.Sprintf(`SELECT t_user.* 
FROM t_user 
	JOIN t_exam_site ON t_exam_site.admin = t_user.id 
WHERE t_exam_site.sys_user = %d`,
					sysUser),
				Table: "t_user",
			},
			{
				Sql: fmt.Sprintf(`SELECT t_user_domain.*
FROM t_user_domain
	JOIN t_exam_site ON t_exam_site.admin = t_user_domain.sys_user
WHERE t_exam_site.sys_user = %d
				`, sysUser),
				Table: "t_user_domain",
			},

			// 最近一个未开始考试的监考员账号数据
			{
				Sql: fmt.Sprintf(`SELECT t_user.*
FROM t_invigilation
	JOIN t_user ON t_user.id = t_invigilation.invigilator
	JOIN t_exam_room ON t_exam_room.id = t_invigilation.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_invigilation.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user.id`, recentExamID, sysUser),
				Table: "t_user",
			},
			{
				Sql: fmt.Sprintf(`SELECT t_user_domain.*
FROM t_invigilation
	JOIN t_user_domain ON t_user_domain.sys_user = t_invigilation.invigilator
	JOIN t_exam_room ON t_exam_room.id = t_invigilation.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_invigilation.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user_domain.id`, recentExamID, sysUser),
				Table: "t_user_domain",
			},

			// 最近一个未开始的考试的考生账号数据
			{
				Sql: fmt.Sprintf(`SELECT t_user.*
FROM t_examinee
	JOIN t_user ON t_user.id = t_examinee.student_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user.id`, recentExamID, sysUser),
				Table: "t_user",
			},
			{
				Sql: fmt.Sprintf(`SELECT t_user_domain.*
FROM t_examinee
	JOIN t_user_domain ON t_user_domain.sys_user = t_examinee.student_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user_domain.id`, recentExamID, sysUser),
				Table: "t_user_domain",
			},

			//======考试数据======
			// 考试信息
			{
				Sql:   fmt.Sprintf(`SELECT t_exam_info.* FROM t_exam_info WHERE id = %d`, recentExamID),
				Table: "t_exam_info",
			},

			// 考试场次
			{
				Sql:   fmt.Sprintf(`SELECT t_exam_session.* FROM t_exam_session WHERE exam_id = %d`, recentExamID),
				Table: "t_exam_session",
			},

			// 考试附件数据
			{
				Sql: fmt.Sprintf(`SELECT t_file.*
FROM t_exam_info
	CROSS JOIN LATERAL jsonb_array_elements(t_exam_info.files) AS file
	JOIN t_file ON t_file.id = file.value::int
WHERE jsonb_typeof(files) = 'array' AND  t_exam_info.id = %d
GROUP BY
	t_file.id`, recentExamID),
				Table: "t_file",
			},

			// 试卷数据
			{
				Sql: fmt.Sprintf(`SELECT t_paper.* 
FROM t_exam_session
	JOIN t_paper ON t_paper.id = t_exam_session.paper_id
WHERE t_exam_session.exam_id = %d`, recentExamID),
				Table: "t_paper",
			},

			// 考卷数据
			{
				Sql: fmt.Sprintf(`SELECT t_exam_paper.* 
FROM t_exam_session
	JOIN t_paper ON t_paper.id = t_exam_session.paper_id
	JOIN t_exam_paper ON t_exam_paper.id = t_paper.exampaper_id
WHERE t_exam_session.exam_id = %d
				`, recentExamID),
				Table: "t_exam_paper",
			},

			// 考卷题组数据
			{
				Sql: fmt.Sprintf(`SELECT t_exam_paper_group.* 
FROM t_exam_session
	JOIN t_paper ON t_paper.id = t_exam_session.paper_id
	JOIN t_exam_paper ON t_exam_paper.id = t_paper.exampaper_id
	JOIN t_exam_paper_group ON t_exam_paper_group.exam_paper_id = t_exam_paper.id
WHERE t_exam_session.exam_id = %d
			`, recentExamID),
				Table: "t_exam_paper_group",
			},

			// 考卷题目数据
			{
				Sql: fmt.Sprintf(`SELECT t_exam_paper_question.* 
FROM t_exam_session
	JOIN t_paper ON t_paper.id = t_exam_session.paper_id
	JOIN t_exam_paper ON t_exam_paper.id = t_paper.exampaper_id
	JOIN t_exam_paper_group ON t_exam_paper_group.exam_paper_id = t_exam_paper.id
	JOIN t_exam_paper_question ON t_exam_paper_question.group_id = t_exam_paper_group.id
WHERE t_exam_session.exam_id = %d`, recentExamID),
				Table: "t_exam_paper_question",
			},

			// 考卷题目附件数据
			{
				Sql: fmt.Sprintf(`SELECT t_file.*
FROM t_exam_paper_question
	JOIN t_exam_paper_group ON t_exam_paper_group.id = t_exam_paper_question.group_id
	JOIN t_exam_paper ON t_exam_paper.id = t_exam_paper_group.exam_paper_id
	JOIN t_paper ON t_paper.exampaper_id = t_exam_paper.id
	JOIN t_exam_session ON t_exam_session.paper_id = t_paper.id
	CROSS JOIN LATERAL jsonb_array_elements(t_exam_paper_question.files) AS file
	JOIN t_file ON t_file.id = file.value::int
WHERE jsonb_typeof(files) = 'array' AND  t_exam_session.exam_id = %d
GROUP BY
	t_file.id`, recentExamID),
				Table: "t_file",
			},

			//======考生数据======
			{
				Sql: fmt.Sprintf(`SELECT t_examinee.*
FROM t_examinee
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d`, recentExamID, sysUser),
				Table: "t_examinee",
			},

			// 考生作答数据
			{
				Sql: fmt.Sprintf(`SELECT t_student_answers.*
FROM t_student_answers
	JOIN t_examinee ON t_examinee.id = t_student_answers.examinee_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.exam_id = %d AND t_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d
				`, recentExamID, sysUser),
				Table: "t_student_answers",
			},

			//===================

			//监考数据
			{
				Sql: fmt.Sprintf(`SELECT t_invigilation.* 
FROM t_invigilation
	JOIN t_exam_room ON t_exam_room.id = t_invigilation.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.id = t_invigilation.exam_session_id
	JOIN t_exam_info ON t_exam_info.id = %d AND t_exam_info.id = t_exam_session.exam_id
WHERE t_exam_site.sys_user = %d`, recentExamID, sysUser),
				Table: "t_invigilation",
			},

			// 考场记录数据
			{
				Sql: fmt.Sprintf(`SELECT t_exam_record.* 
FROM t_exam_record
	JOIN t_exam_room ON t_exam_room.id = t_exam_record.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN t_exam_session ON t_exam_session.id = t_exam_record.exam_session
	JOIN t_exam_info ON t_exam_info.id = %d AND t_exam_info.id = t_exam_session.exam_id
WHERE t_exam_site.sys_user = %d`, recentExamID, sysUser),
				Table: "t_exam_record",
			},

			
		}
	}

	var tableCount = make(map[string]int)

	exportScriptFile, err := os.Create(filepath.Join(destDir, fileName))
	if err != nil || (cmn.InDebugMode && fileName == "force-create-export-script-file-err-^a1^2*zc$32h@g4") {

		if err == nil {
			err = fmt.Errorf("%s", fileName)
		}

		z.Error(err.Error())
		return
	}

	defer exportScriptFile.Close()

	for _, info := range exportInfo {

		if _, ok := tableCount[info.Table]; !ok {
			tableCount[info.Table] = -1
		}

		tableCount[info.Table]++

		fName := fmt.Sprintf("%s_%d.csv", info.Table, tableCount[info.Table])

		sql := strings.ReplaceAll(info.Sql, "\n", " ")

		line := fmt.Sprintf("\\copy (%s) TO '%s' CSV HEADER\n", sql, filepath.Join(destDir, fName))

		_, err = exportScriptFile.WriteString(line)
		if err != nil || (cmn.InDebugMode && fileName == "force-write-export-script-file-err-^a1^2*zc$32h@g4") {

			if err == nil {
				err = fmt.Errorf("%s", fileName)
			}

			z.Error(err.Error())
			return
		}

		tableFileList = append(tableFileList, fName)
	}

	return
}

// generateImportScript 生成导入脚本文件
func generateImportScript(tableFileList []string, destDir string, fileName string, isSubServerSide bool) (err error) {

	f, err := os.Create(filepath.Join(destDir, fileName))
	if err != nil {
		z.Error(err.Error())
		return
	}

	defer f.Close()

	content := "BEGIN;\n"
	end := "\nCOMMIT;"
	// if !isSubServerSide {
	// 	content = "BEGIN;\n"
	// 	end  = "\nCOMMIT;"
	// }

	for _, fName := range tableFileList {

		// fName 格式为 t_{name_part1}_{name_part2}_{编号}
		// 将fName转为 t_{name} 表格名称格式
		re := regexp.MustCompile(`^(t_[a-zA-Z0-9_]+)_\d+\.csv$`)
		matches := re.FindStringSubmatch(fName)
		tableName := fName
		if len(matches) == 2 {
			tableName = matches[1]
		}

		// if isSubServerSide {
		// 	content += fmt.Sprintf("\\copy %s FROM '%s' CSV HEADER\n", tableName, filepath.Join(destDir, fName))
		// 	continue
		// }

		// 读取文件第一行获取表头
		var tableFile *os.File
		tableFile, err = os.Open(filepath.Join(destDir, fName))
		if err != nil {
			z.Error(err.Error())
			return
		}
		defer tableFile.Close()

		reader := csv.NewReader(tableFile)

		var cols []string
		cols, err = reader.Read()
		if err != nil {
			z.Error(err.Error())
			return
		}

		for i, col := range cols {
			cols[i] = fmt.Sprintf(`"%s"`, col)
		}

		updateSets := []string{}

		for _, col := range cols {
			updateSets = append(updateSets, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}

		content += fmt.Sprintf(`
CREATE TEMP TABLE temp_%s (LIKE %s INCLUDING ALL) ON COMMIT DROP;	
\copy temp_%s FROM '%s' CSV HEADER
INSERT INTO %s (%s) 
	SELECT %s FROM temp_%s
	ON CONFLICT(id) DO UPDATE SET %s;
DROP TABLE IF EXISTS temp_%s;
		`,
			tableName,
			tableName,
			tableName,
			filepath.Join(destDir, fName),
			tableName,
			strings.Join(cols, ", "),
			strings.Join(cols, ", "),
			tableName,
			strings.Join(updateSets, ", "),
			tableName)

	}

	content += end

	_, err = f.WriteString(content)
	if err != nil || (cmn.InDebugMode && fileName == "force-write-import-script-file-err-^a1^2*zc$32h@g4") {

		if err == nil {
			err = fmt.Errorf("%s", fileName)
		}

		z.Error(err.Error())
		return
	}

	return
}

// ReadColumnFromCSV 从CSV文件中读取指定列的数据
func ReadColumnFromCSV(filePath string, columnName string) ([]string, error) {
    f, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    reader := csv.NewReader(f)
    header, err := reader.Read()
    if err != nil {
        return nil, err
    }

    // 找到目标列索引
    colIdx := -1
    for i, name := range header {
        if name == columnName {
            colIdx = i
            break
        }
    }
    if colIdx == -1 {
        return nil, fmt.Errorf("列名不存在: %s", columnName)
    }

    var result []string
    for {
        record, err := reader.Read()
        if err != nil {
            break // EOF
        }
        result = append(result, record[colIdx])
    }
    return result, nil
}