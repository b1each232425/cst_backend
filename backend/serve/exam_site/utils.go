package exam_site

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"w2w.io/cmn"
)

type copyInfo struct {
	Sql   string
	Table string
}


// generateExportScriptForCentralServer 生成导出脚本文件(中心服务器方调用)
// sysUser 为考点服务器系统账号ID
// destDir 为数据保存目录
// fileName 为脚本文件名
func generateExportScriptForCentralServer(sysUser int64, destDir string, fileName string) (tableFileList []string, err error) {

	if sysUser <= 0 {
		err = fmt.Errorf("invalid sysUser: %d", sysUser)
		z.Error(err.Error())
		return
	}

	exportInfo := []copyInfo{

		//====系统基本数据====
		{
			Sql: `SELECT t_domain.* FROM t_domain`,
			Table: "t_domain",
		},
		{
			Sql: `SELECT t_api.* FROM t_api`,
			Table: "t_api",
		},
		{
			Sql: `SELECT t_domain_api.* FROM t_domain_api`,
			Table: "t_domain_api",
		},


		//===================

		//====账号数据查询====

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
			Sql: fmt.Sprintf(`WITH recent_exam AS (
	SELECT 
		t_exam_info.id,
		MIN(t_exam_session.start_time) AS neartest_start_time
	FROM t_exam_info
		JOIN t_exam_session ON t_exam_session.exam_id = t_exam_info.id
	WHERE ((EXTRACT(epoch FROM CURRENT_TIMESTAMP) * (1000)::numeric))::bigint < start_time
	GROUP BY
		t_exam_info.id
	ORDER BY neartest_start_time ASC
)
SELECT t_user.*
FROM t_invigilation
	JOIN t_user ON t_user.id = t_invigilation.invigilator
	JOIN t_exam_room ON t_exam_room.id = t_invigilation.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN (
		SELECT t_exam_session.id
		FROM t_exam_session
			JOIN recent_exam ON recent_exam.id = t_exam_session.exam_id
	) recent_exam_session ON recent_exam_session.id = t_invigilation.exam_session_id
WHERE t_exam_site.sys_user = %d`,
				sysUser),
			Table: "t_user",
		},
		{
			Sql: fmt.Sprintf(`WITH recent_exam AS (
	SELECT 
		t_exam_info.id,
		MIN(t_exam_session.start_time) AS neartest_start_time
	FROM t_exam_info
		JOIN t_exam_session ON t_exam_session.exam_id = t_exam_info.id
	WHERE ((EXTRACT(epoch FROM CURRENT_TIMESTAMP) * (1000)::numeric))::bigint < start_time
	GROUP BY
		t_exam_info.id
	ORDER BY neartest_start_time ASC
)
SELECT t_user_domain.*
FROM t_invigilation
	JOIN t_user_domain ON t_user_domain.sys_user = t_invigilation.invigilator
	JOIN t_exam_room ON t_exam_room.id = t_invigilation.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN (
		SELECT t_exam_session.id
		FROM t_exam_session
			JOIN recent_exam ON recent_exam.id = t_exam_session.exam_id
	) recent_exam_session ON recent_exam_session.id = t_invigilation.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user_domain.id`, sysUser),
			Table: "t_user_domain",
		},

		// 最近一个未开始的考试的考生账号数据
		{
			Sql: fmt.Sprintf(`WITH recent_exam AS (
	SELECT 
		t_exam_info.id,
		MIN(t_exam_session.start_time) AS neartest_start_time
	FROM t_exam_info
		JOIN t_exam_session ON t_exam_session.exam_id = t_exam_info.id
	WHERE ((EXTRACT(epoch FROM CURRENT_TIMESTAMP) * (1000)::numeric))::bigint < start_time
	GROUP BY
		t_exam_info.id
	ORDER BY neartest_start_time ASC
)
SELECT t_user.*
FROM t_examinee
	JOIN t_user ON t_user.id = t_examinee.student_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN (
		SELECT t_exam_session.id
		FROM t_exam_session
			JOIN recent_exam ON recent_exam.id = t_exam_session.exam_id
	) recent_exam_session ON recent_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d`,
				sysUser),
			Table: "t_user",
		},
		{
			Sql: fmt.Sprintf(`WITH recent_exam AS (
	SELECT 
		t_exam_info.id,
		MIN(t_exam_session.start_time) AS neartest_start_time
	FROM t_exam_info
		JOIN t_exam_session ON t_exam_session.exam_id = t_exam_info.id
	WHERE ((EXTRACT(epoch FROM CURRENT_TIMESTAMP) * (1000)::numeric))::bigint < start_time
	GROUP BY
		t_exam_info.id
	ORDER BY neartest_start_time ASC
)
SELECT t_user_domain.*
FROM t_examinee
	JOIN t_user_domain ON t_user_domain.sys_user = t_examinee.student_id
	JOIN t_exam_room ON t_exam_room.id = t_examinee.exam_room
	JOIN t_exam_site ON t_exam_site.id = t_exam_room.exam_site
	JOIN (
		SELECT t_exam_session.id
		FROM t_exam_session
			JOIN recent_exam ON recent_exam.id = t_exam_session.exam_id
	) recent_exam_session ON recent_exam_session.id = t_examinee.exam_session_id
WHERE t_exam_site.sys_user = %d
GROUP BY
	t_user_domain.id`, sysUser),
			Table: "t_user_domain",
		},

		//===================

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

// generateImportScriptForSubServer 生成导入脚本文件(考点服务器方调用)
func generateImportScriptForSubServer(tableFileList []string, destDir string, fileName string) (err error) {

	f, err := os.Create(filepath.Join(destDir, fileName))
	if err != nil {
		z.Error(err.Error())
		return
	}

	defer f.Close()

	for _, fName := range tableFileList {

		// fName 格式为 t_{name_part1}_{name_part2}_{编号}
		// 将fName转为 t_{name} 表格名称格式
		re := regexp.MustCompile(`^(t_[a-zA-Z0-9_]+)_\d+\.csv$`)
		matches := re.FindStringSubmatch(fName)
		tableName := fName
		if len(matches) == 2 {
			tableName = matches[1]
		}

		_, err = f.WriteString(fmt.Sprintf("\\copy %s FROM '%s' CSV HEADER\n", tableName, filepath.Join(destDir, fName)))
		if err != nil || (cmn.InDebugMode && fileName == "force-write-import-script-file-err-^a1^2*zc$32h@g4") {
			
			if err == nil {
				err = fmt.Errorf("%s", fileName)
			}
			
			z.Error(err.Error())
			return
		}

	}


	return
}