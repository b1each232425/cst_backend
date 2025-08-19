package exam_mgt

//annotation:exam_mgt-service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/exam_service"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/mark"
)

var (
	z         *zap.Logger
	uploadDir string
)

const (
	TEST              = "test"
	REDIS_LOCK_PREFIX = "exam_lock:"
)

type ExamRoomConfig struct {
	RoomID           int64 `json:"roomID" validate:"required"`
	Capacity         int64 `json:"capacity" validate:"required"`
	InvigilatorCount int64 `json:"invigilator_count" validate:"required"`
}

type ExamData struct {
	ExamInfo       cmn.TExamInfo      `json:"examInfo" validate:"required"`
	ExamSessions   []cmn.TExamSession `json:"examSessions" validate:"required"`
	ExamineeIDs    []int64            `json:"examinee"`
	ExamRooms      []ExamRoomConfig   `json:"examRooms"`
	InvigilatorIDs []int64            `json:"invigilators"`
	TimeStamp      int64              `json:"timeStamp"` //时间戳
	Files          []ExamFile         `json:"files"`
}

// 考试场次时间
type ExamSession struct {
	ID             int64   `json:"id"`         //场次ID
	StartTime      int64   `json:"start_time"` //场次开始时间
	EndTime        int64   `json:"end_time"`   //场次结束时间
	SessionNum     int64   `json:"session_num"`
	PaperID        int64   `json:"paper_id"`
	PaperName      string  `json:"paper_name"`
	Status         string  `json:"status"`
	ExamineeStatus string  `json:"examinee_status"`
	StudentScore   float64 `json:"student_score"`
	TotalScore     float64 `json:"total_score"`
}

type ExamList struct {
	ID             int64         `json:"id"`               //考试id
	ExamName       string        `json:"name"`             //考试名
	ExamType       string        `json:"type"`             //考试类型 00：平时考试 02：期末成绩考试  04：资格证考试
	ExamMethod     string        `json:"method"`           //考试方式 00：线上考试  02：线下考试
	ExamSession    []ExamSession `json:"exam_sessions"`    //考试场次信息
	Duration       int64         `json:"duration"`         //考试时长
	Status         string        `json:"status"`           //考试状态  00：未发布 02：待开始  04：进行中 08：已结束 10：已归档 12：考试异常 14：已删除
	NumOfExaminees int64         `json:"num_of_examinees"` //考生数量
}

type Examinee struct {
	StudentID      int64       `json:"id"`
	SerialNumber   int64       `json:"serial_number"`
	ExamineeNumber null.String `json:"examinee_number"` // 准考证号
	OfficialName   string      `json:"official_name"`
	Account        string      `json:"account"`      // 学生账号
	Passwd         string      `json:"passwd"`       // 学生密码
	IdCardNo       null.String `json:"id_card_no"`   // 学生身份证号
	MobilePhone    string      `json:"mobile_phone"` // 学生电话
}

type ExamUserInfo struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	IDCardNo    null.String `json:"id_card_no"`   // 身份证号
	MobilePhone string      `json:"mobile_phone"` // 电话
	Gender      string      `json:"gender"`
}

type ExamFile struct {
	ExamID   int64  `json:"exam_id"`
	CheckSum string `json:"checksum"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
}

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("exam_mgt zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("exam_mgt.Enroll called")

	key := "tusd.fileStorePath"
	uploadDir = "./uploads"
	if viper.IsSet(key) {
		uploadDir = viper.GetString(key)
	}
	var err error
	uploadDir, err = filepath.Abs(uploadDir)
	if err != nil {
		z.Fatal(fmt.Sprintf("Unable to make absolute path: %s", err))
	}

	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: exam,

		Path: "/exam",
		Name: "exam",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examList,

		Path: "/exam/list",
		Name: "examList",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examList,

		Path: "/exam/sessions",
		Name: "examSessions",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examStatus,

		Path: "/exam/status",
		Name: "examStatus",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examinee,

		Path: "/exam/examinee",
		Name: "examinee",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examUser,

		Path: "/exam/user",
		Name: "examUser",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examFile,

		Path: "/exam/file",
		Name: "examFile",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// 检查考试是否存在
func examExists(ctx context.Context, examID int64) (bool, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("examExists-force-error"); val != nil {
		forceErr = val.(string)
	}

	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			return true, nil
		case "examExists-error":
			return false, errors.New("examExists error")
		}
	}

	if examID <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examID)
		z.Error(err.Error())
		return false, err
	}

	conn := cmn.GetPgxConn()
	var exists bool
	err := conn.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM t_exam_info WHERE id=$1 AND status!= '12')", examID).Scan(&exists)
	if forceErr == "conn.QueryRow" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return false, err
	}

	return exists, nil
}

// 生成准考证号
func generateExamineeNumber(serialNumber int64, examInfo cmn.TExamInfo, examSessions []cmn.TExamSession) string {

	// 非线下考试不生成准考证号
	if examInfo.Mode.String == "00" {
		return ""
	}

	// 获取考试年份的后两位
	var examYear string
	examYear = strconv.Itoa(time.Now().Year() % 100)

	if len(examSessions) > 0 {
		var examTime time.Time

		// 获取考试开始时间
		if examSessions[0].StartTime.Valid && examSessions[0].StartTime.Int64 > 0 {
			examTime = time.UnixMilli(examSessions[0].StartTime.Int64)
		} else {
			examTime = time.Now()
		}

		year := examTime.Year()
		examYear = strconv.Itoa(year % 100)
	}

	// 生成考试ID字符串
	examIDStr := strconv.FormatInt(examInfo.ID.Int64, 10)

	// 生成序号字符串，固定6位，不足补0，超出正常显示
	serialNumStr := fmt.Sprintf("%06d", serialNumber)

	// 拼接准考证号
	return examYear + examIDStr + serialNumStr
}

// 获取考试信息
// 参数: ctx: 上下文, examID: 考试ID, domain: 用户域
// 返回: cmn.TExamInfo, error
func GetExamInfo(ctx context.Context, examID int64, domain string) (cmn.TExamInfo, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	//mock数据返回
	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			return cmn.TExamInfo{
				ID:          null.NewInt(1, true),
				Name:        null.NewString("测试考试", true),
				Rules:       null.NewString("考试规则", true),
				Type:        null.NewString("00", true),
				Mode:        null.NewString("02", true),
				Files:       nil,
				Submitted:   null.NewBool(true, true),
				ExamRoomIds: nil,
				Creator:     null.NewInt(10001, true),
				CreateTime:  null.NewInt(time.Now().UnixMilli(), true),
				UpdatedBy:   null.NewInt(10001, true),
				UpdateTime:  null.NewInt(time.Now().UnixMilli(), true),
				Status:      null.NewString("00", true),
				Addi:        nil,
			}, nil
		case "bad-resp":
			return cmn.TExamInfo{}, nil
		case "GetExamInfo-error":
			return cmn.TExamInfo{}, errors.New("GetExamInfo error")
		}
	}

	if examID <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examID)
		z.Error(err.Error())
		return cmn.TExamInfo{}, err
	}

	var examInfo cmn.TExamInfo
	examInfo = cmn.TExamInfo{}

	conn := cmn.GetPgxConn()

	if strings.Contains(domain, "^student") {
		query := `
			SELECT 
				id, name, rules, type, mode, files, status
			FROM t_exam_info
			WHERE id = $1 AND status != '12'
		`
		row := conn.QueryRow(context.Background(), query, examID)
		err := row.Scan(
			&examInfo.ID,
			&examInfo.Name,
			&examInfo.Rules,
			&examInfo.Type,
			&examInfo.Mode,
			&examInfo.Files,
			&examInfo.Status,
		)
		if forceErr == "conn.Scan" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return cmn.TExamInfo{}, err
		}
	} else {
		query := `
			SELECT 
				id, name, rules, type, mode, files, submitted, creator,
				create_time, updated_by, update_time, status, addi, exam_room_ids, exam_room_invigilator_count
			FROM t_exam_info
			WHERE id = $1 AND status != '12'
		`

		row := conn.QueryRow(context.Background(), query, examID)
		err := row.Scan(
			&examInfo.ID,
			&examInfo.Name,
			&examInfo.Rules,
			&examInfo.Type,
			&examInfo.Mode,
			&examInfo.Files,
			&examInfo.Submitted,
			&examInfo.Creator,
			&examInfo.CreateTime,
			&examInfo.UpdatedBy,
			&examInfo.UpdateTime,
			&examInfo.Status,
			&examInfo.Addi,
			&examInfo.ExamRoomIds,
			&examInfo.ExamRoomInvigilatorCount,
		)
		if forceErr == "conn.Scan" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return cmn.TExamInfo{}, err
		}
	}

	return examInfo, nil
}

// 获取考试场次信息
// 参数: ctx: 上下文, examID: 考试ID, domain: 用户域
// 返回: cmn.TExamInfo, error
func GetExamSessions(ctx context.Context, domain string, examIDs ...int64) ([]cmn.TExamSession, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	//mock数据返回
	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			return []cmn.TExamSession{
				{
					ID:                   null.NewInt(10001, true),
					ExamID:               null.NewInt(10001, true),
					PaperID:              null.NewInt(10001, true),
					MarkMethod:           "00",
					PeriodMode:           null.NewString("02", true),
					StartTime:            null.NewInt(1700000000000, true),
					EndTime:              null.NewInt(1700003600000, true),
					Duration:             null.NewInt(60, true),
					SessionNum:           null.NewInt(1, true),
					LateEntryTime:        null.NewInt(5, true),
					EarlySubmissionTime:  null.NewInt(5, true),
					QuestionShuffledMode: null.NewString("00", true),
					MarkMode:             null.NewString("01", true),
					NameVisibilityIn:     null.NewBool(false, true),
					ReviewerIds:          null.NewString("1,2", true),
				},
			}, nil
		case "bad-resp":
			return []cmn.TExamSession{}, nil
		case "GetExamSessions-error":
			return nil, errors.New("GetExamSessions error")
		}
	}

	for _, examID := range examIDs {
		if examID <= 0 {
			err := fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(err.Error())
			return []cmn.TExamSession{}, err
		}
	}

	var es_array []cmn.TExamSession

	conn := cmn.GetPgxConn()
	if strings.Contains(domain, "^student") {
		es_query := `
			SELECT
				id, exam_id, paper_id, mark_method, period_mode,
				start_time, end_time, duration, session_num, late_entry_time,
				early_submission_time
			FROM t_exam_session
			WHERE exam_id = ANY($1) AND status != '14'
			ORDER BY session_num
		`
		var es_rows pgx.Rows
		es_rows, err := conn.Query(context.Background(), es_query, examIDs)
		defer es_rows.Close()
		if forceErr == "conn.Query" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		for es_rows.Next() {
			var es cmn.TExamSession
			err := es_rows.Scan(
				&es.ID,
				&es.ExamID,
				&es.PaperID,
				&es.MarkMethod,
				&es.PeriodMode,
				&es.StartTime,
				&es.EndTime,
				&es.Duration,
				&es.SessionNum,
				&es.LateEntryTime,
				&es.EarlySubmissionTime,
			)
			if forceErr == "conn.Scan" {
				err = fmt.Errorf("force error: %s", forceErr)
			}
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			es_array = append(es_array, es)
		}
	} else {
		es_query := `
			SELECT
				es.id, es.exam_id, es.paper_id, es.mark_method, es.period_mode,
				es.start_time, es.end_time, es.duration, es.question_shuffled_mode,
				es.mark_mode, es.name_visibility_in, es.session_num, es.late_entry_time,
				es.early_submission_time, es.reviewer_ids,
				p.name as paper_name, p.category as paper_category
			FROM t_exam_session es
			LEFT JOIN t_paper p ON es.paper_id = p.id
			WHERE es.exam_id = ANY($1) AND es.status != '14'
			ORDER BY es.session_num
		`
		var es_rows pgx.Rows
		es_rows, err := conn.Query(context.Background(), es_query, examIDs)
		defer es_rows.Close()
		if forceErr == "conn.Query" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		for es_rows.Next() {
			var es cmn.TExamSession
			err := es_rows.Scan(
				&es.ID,
				&es.ExamID,
				&es.PaperID,
				&es.MarkMethod,
				&es.PeriodMode,
				&es.StartTime,
				&es.EndTime,
				&es.Duration,
				&es.QuestionShuffledMode,
				&es.MarkMode,
				&es.NameVisibilityIn,
				&es.SessionNum,
				&es.LateEntryTime,
				&es.EarlySubmissionTime,
				&es.ReviewerIds,
				&es.PaperName,
				&es.PaperCategory,
			)
			if forceErr == "conn.Scan" {
				err = fmt.Errorf("force error: %s", forceErr)
			}
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			es_array = append(es_array, es)
		}
	}
	return es_array, nil

}

func getExamSessionIDs(ctx context.Context, examIDs ...int64) ([]int64, error) {
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()

	// 获取旧的考试场次信息
	getExamSessionIDsQuery := `SELECT id FROM t_exam_session WHERE exam_id = ANY($1) AND status != '14'`
	var oldExamSessionIDs []int64
	var oldExamSessionRows pgx.Rows
	oldExamSessionRows, err := conn.Query(ctx, getExamSessionIDsQuery, examIDs)
	defer oldExamSessionRows.Close()
	if forceErr == "conn.QueryOldExamSessionRows" {
		err = fmt.Errorf("强制查询旧考试场次错误")
	}
	if err != nil {
		z.Error(err.Error())
		return nil, err
	}
	for oldExamSessionRows.Next() {
		var sessionID int64
		err := oldExamSessionRows.Scan(&sessionID)
		if forceErr == "oldExamSessionRows.Scan" {
			err = fmt.Errorf("强制扫描旧考试场次ID错误")
		}
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		oldExamSessionIDs = append(oldExamSessionIDs, sessionID)
	}

	return oldExamSessionIDs, nil
}

// 验证用户对考试的访问权限
func validateUserExamPermission(ctx context.Context, userID, examID int64, domain string) (bool, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-resp":
			return true, nil
		case "validateUserExamPermission-false":
			return false, nil
		case "validateUserExamPermission-error":
			return false, errors.New("validateUserExamPermission error")
		}
	}

	if userID <= 0 || examID <= 0 {
		err := fmt.Errorf("无效的用户ID或考试ID: userID=%d, examID=%d", userID, examID)
		z.Error(err.Error())
		return false, err
	}

	conn := cmn.GetPgxConn()
	var exists bool
	exists = false

	// 根据角色查询是否能访问该考试
	if strings.Contains(domain, "^student") {
		err := conn.QueryRow(context.Background(), `
			SELECT EXISTS(
			SELECT 1
			FROM t_examinee e
			INNER JOIN t_exam_session es ON e.exam_session_id = es.id
			WHERE es.exam_id = $1
			AND es.status NOT IN ('00', '14')
			AND e.student_id = $2
			AND e.status != '08'
			)
		`, examID, userID).Scan(&exists)
		if forceErr == "conn.QueryRow" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
	} else if strings.Contains(domain, "^admin") || strings.Contains(domain, "^superAdmin") || strings.Contains(domain, "^teacher") {
		domainPrefix := getDomainPrefix(domain)

		err := conn.QueryRow(context.Background(), `
		SELECT EXISTS(
		SELECT 1
		FROM t_exam_info ei
		JOIN t_domain d ON ei.domain_id = d.id
		WHERE ei.id = $1
			AND (
				d.domain LIKE $2 || '^%'       
				OR 
				d.domain LIKE $2 || '.%^%'     
			)
		)
		`, examID, domainPrefix).Scan(&exists)
		if forceErr == "conn.QueryRow" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
	}

	return exists, nil
}

// // 将考生分配到考场
// func allocateExamineesToRooms(examinees []cmn.TExaminee, examRooms []cmn.TExamRoom) ([]cmn.TExaminee, error) {
// 	z.Info("---->" + cmn.FncName())

// 	if len(examinees) == 0 || len(examRooms) == 0 {
// 		err := fmt.Errorf("分配考生时考生或考场数据为空")
// 		z.Error(err.Error())
// 		return nil, err
// 	}

// 	// 按考场容量排序（从大到小）
// 	sort.Slice(examRooms, func(i, j int) bool {
// 		return examRooms[i].Capacity.Int64 > examRooms[j].Capacity.Int64
// 	})

// 	// 创建考场分配计划
// 	type roomAllocation struct {
// 		roomID    int64
// 		capacity  int64
// 		allocated int64
// 	}

// 	allocations := make([]roomAllocation, len(examRooms))
// 	for i, room := range examRooms {
// 		allocations[i] = roomAllocation{
// 			roomID:    room.ID.Int64,
// 			capacity:  room.Capacity.Int64,
// 			allocated: 0,
// 		}
// 	}

// 	// 使用轮询算法平均分配考生
// 	currentRoomIndex := 0
// 	for i := range examinees {

// 		// 找到下一个有容量的考场
// 		attempts := 0
// 		for attempts < len(allocations) {
// 			if allocations[currentRoomIndex].allocated < allocations[currentRoomIndex].capacity {

// 				// 将考场ID分配给考生的ExamRoom字段
// 				examinees[i].ExamRoom.Int64 = allocations[currentRoomIndex].roomID
// 				examinees[i].ExamRoom.Valid = true
// 				allocations[currentRoomIndex].allocated++

// 				// 设置考生序号
// 				examinees[i].SerialNumber.Int64 = allocations[currentRoomIndex].allocated
// 				examinees[i].SerialNumber.Valid = true
// 				break
// 			}
// 			// 切换到下一个考场
// 			currentRoomIndex = (currentRoomIndex + 1) % len(allocations)
// 			attempts++
// 		}

// 		if attempts >= len(allocations) {
// 			err := fmt.Errorf("无法为考生分配考场，考场容量不足")
// 			z.Error(err.Error())
// 			return nil, err
// 		}

// 		// 切换到下一个考场
// 		currentRoomIndex = (currentRoomIndex + 1) % len(allocations)
// 	}

// 	return examinees, nil
// }

// // 将监考员分配到考场和场次
// func allocateInvigilatorsToRooms(examSessionIDs []int64, examRooms []ExamRoomConfig, invigilatorIDs []int64) ([]cmn.TInvigilation, error) {
// 	z.Info("---->" + cmn.FncName())

// 	// 随机打乱监考员顺序
// 	rand.Shuffle(len(invigilatorIDs), func(i, j int) {
// 		invigilatorIDs[i], invigilatorIDs[j] = invigilatorIDs[j], invigilatorIDs[i]
// 	})

// 	var invigilations []cmn.TInvigilation
// 	currentInvigilatorIndex := 0

// 	// 为每个场次的每个考场分配监考员
// 	for _, examSessionID := range examSessionIDs {
// 		for _, room := range examRooms {

// 			// 为当前考场分配指定数量的监考员
// 			for i := int64(0); i < room.InvigilatorCount; i++ {
// 				if currentInvigilatorIndex >= len(invigilatorIDs) {
// 					return nil, fmt.Errorf("监考员数量不足，需要 %d 名监考员", currentInvigilatorIndex+1)
// 				}
// 				var esID null.Int
// 				esID.Int64 = examSessionID
// 				esID.Valid = true

// 				var er null.Int
// 				er.Int64 = room.RoomID
// 				er.Valid = true

// 				var i null.Int
// 				i.Int64 = invigilatorIDs[currentInvigilatorIndex]
// 				i.Valid = true

// 				invigilation := cmn.TInvigilation{
// 					ExamSessionID: esID,
// 					ExamRoom:      er,
// 					Invigilator:   i,
// 				}

// 				invigilations = append(invigilations, invigilation)
// 				currentInvigilatorIndex++
// 			}
// 		}
// 	}

// 	return invigilations, nil
// }

func updateExamStatus(ctx context.Context, tx pgx.Tx, newStatus string, userID int64, examIDs ...int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if len(examIDs) == 0 {
		err := fmt.Errorf("考试ID数组不能为空")
		z.Error(err.Error())
		return err
	}

	for _, examID := range examIDs {
		if examID <= 0 {
			err := fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(err.Error())
			return err
		}
	}

	if userID <= 0 {
		err := fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(err.Error())
		return err
	}

	if newStatus == "" {
		err := fmt.Errorf("更新状态不能为空")
		z.Error(err.Error())
		return err
	}

	_, err := tx.Exec(ctx, `
		UPDATE t_exam_info 
		SET status = $1, update_time = $2, updated_by = $3
		WHERE id = ANY($4) AND status != '12'
	`, newStatus, time.Now().UnixMilli(), userID, examIDs)
	if forceErr == "tx.Exec" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}

func updateExamSessionStatus(ctx context.Context, tx pgx.Tx, newStatus string, userID int64, examIDs ...int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if len(examIDs) == 0 {
		err := fmt.Errorf("考试ID数组不能为空")
		z.Error(err.Error())
		return err
	}

	for _, examID := range examIDs {
		if examID <= 0 {
			err := fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(err.Error())
			return err
		}
	}

	if userID <= 0 {
		err := fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(err.Error())
		return err
	}

	if newStatus == "" {
		err := fmt.Errorf("更新状态不能为空")
		z.Error(err.Error())
		return err
	}

	_, err := tx.Exec(ctx, `
		UPDATE t_exam_session 
		SET status = $1, update_time = $2, updated_by = $3
		WHERE exam_id = ANY($4) AND status != '14'
	`, newStatus, time.Now().UnixMilli(), userID, examIDs)
	if forceErr == "tx.Exec" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}

func updateExamineeStatus(ctx context.Context, tx pgx.Tx, newStatus string, userID int64, examIDs ...int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if len(examIDs) == 0 {
		err := fmt.Errorf("考试ID数组不能为空")
		z.Error(err.Error())
		return err
	}

	for _, examID := range examIDs {
		if examID <= 0 {
			err := fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(err.Error())
			return err
		}
	}

	if userID <= 0 {
		err := fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(err.Error())
		return err
	}

	if newStatus == "" {
		err := fmt.Errorf("更新状态不能为空")
		z.Error(err.Error())
		return err
	}

	_, err := tx.Exec(ctx, `
		UPDATE t_examinee 
        SET status = $1, update_time = $2, updated_by = $3
        FROM t_exam_session es
        WHERE t_examinee.exam_session_id = es.id
        AND es.exam_id = ANY($4) 
        AND t_examinee.status != '08'
	`, newStatus, time.Now().UnixMilli(), userID, examIDs)
	if forceErr == "updateExamineeStatus.Exec" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}

// 创建/更新/获取考试信息
func exam(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	conn := cmn.GetPgxConn()

	// 用于测试，强制执行某些错误分支
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 用户拥有的域
	userDomains := q.Domains

	// 当前用户登录选择的域
	userRole := q.SysUser.Role.Int64

	var userDomain string
	userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	currentTime := time.Now().UnixMilli()

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		// 创建考试

		// 检查是否具备创建考试的权限
		var result bool
		result = validateUserForExamCreateOrUpdate(userDomain)
		if !result {
			q.Err = fmt.Errorf("用户没有创建考试的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var newExamID int64

		// 创建临时考试信息
		q.Err = conn.QueryRow(ctx, `
			INSERT INTO t_exam_info (
				creator, create_time, updated_by, update_time, status, domain_id
			) VALUES (
				$1, $2, $3, $4, $5, $6
			) RETURNING id
		`,
			userID, currentTime, userID, currentTime, "14", userRole,
		).Scan(&newExamID)
		if forceErr == "tx.QueryRow1" {
			q.Err = fmt.Errorf("强制查询错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examIDStruct := struct {
			ID int64 `json:"id"`
		}{
			ID: newExamID,
		}

		q.Msg.Data, q.Err = json.Marshal(examIDStruct)
		if forceErr == "json.Marshal" {
			q.Msg.Data = nil
			q.Err = fmt.Errorf("强制json.Marshal错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Resp()
		return

	case "get":
		// 获取考试信息
		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if examID <= 0 {
			q.Err = fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 查询考试是否存在
		var exists bool
		exists, q.Err = examExists(ctx, examID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !exists {
			q.Err = fmt.Errorf("考试不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 验证用户对考试的访问权限
		var hasPermission bool
		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userDomain)
		if q.Err != nil {
			q.RespErr()
			return
		}

		if !hasPermission {
			q.Err = fmt.Errorf("无权限访问该考试")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 考试基本信息
		var ei cmn.TExamInfo
		ei, q.Err = GetExamInfo(ctx, examID, userDomain)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 考试场次信息
		var es_array []cmn.TExamSession
		es_array, q.Err = GetExamSessions(ctx, userDomain, examID)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 获取考试附件信息
		var examFiles []ExamFile
		queryExamFilesSQL := `
			SELECT digest, file_name
			FROM v_exam_file
			WHERE exam_id = $1
		`
		var examFilesRows pgx.Rows
		examFilesRows, q.Err = conn.Query(context.Background(), queryExamFilesSQL, examID)
		defer examFilesRows.Close()
		if forceErr == "conn.QueryExamFilesRows" {
			q.Err = fmt.Errorf("强制查询错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer examFilesRows.Close()

		for examFilesRows.Next() {
			var ef ExamFile
			q.Err = examFilesRows.Scan(&ef.CheckSum, &ef.Name)
			if forceErr == "examFilesRows.Scan" {
				q.Err = fmt.Errorf("强制获取考试文件错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examFiles = append(examFiles, ef)
		}

		if strings.Contains(userDomain, "^student") {
			examData := ExamData{
				ExamInfo:       ei,
				ExamSessions:   es_array,
				ExamineeIDs:    nil,
				InvigilatorIDs: nil,
				ExamRooms:      nil,
				TimeStamp:      currentTime,
				Files:          examFiles,
			}

			var jsonData []byte
			jsonData, q.Err = json.Marshal(examData)
			if forceErr == "json.Marshal" {
				q.Err = fmt.Errorf("强制JSON序列化错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Msg.Data = jsonData
			q.Resp()

			return
		}

		// 获取考生对应的用户ID数组
		var examineeIDs []int64
		examinee_query := `
			SELECT DISTINCT e.student_id 
			FROM t_examinee e
			INNER JOIN t_exam_session es ON e.exam_session_id = es.id
			WHERE es.exam_id = $1 AND e.status != '08' AND es.status != '14'
		`
		var examinee_rows pgx.Rows
		examinee_rows, q.Err = conn.Query(context.Background(), examinee_query, examID)
		defer func() {
			if examinee_rows != nil {
				examinee_rows.Close()
			}
		}()
		if forceErr == "conn.Query" {
			q.Err = fmt.Errorf("强制查询错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer examinee_rows.Close()
		for examinee_rows.Next() {
			var examineeID int64
			q.Err = examinee_rows.Scan(&examineeID)
			if forceErr == "examinee_rows.Scan" {
				q.Err = fmt.Errorf("强制扫描考生ID错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examineeIDs = append(examineeIDs, examineeID)
		}

		// // 获取监考员信息
		// var invigilatorIDs []int64
		// var examSessionIDs []int64
		// for _, es := range es_array {
		// 	examSessionIDs = append(examSessionIDs, es.ID.Int64)
		// }

		// invigilator_query := `
		// 	SELECT DISTINCT i.invigilator
		// 	FROM t_invigilation i
		// 	INNER JOIN t_exam_session es ON i.exam_session_id IN ($1)
		// 	WHERE i.invigilator IS NOT NULL
		// 	ORDER BY i.invigilator
		// `
		// var invigilator_rows pgx.Rows
		// invigilator_rows, q.Err = conn.Query(context.Background(), invigilator_query, examSessionIDs)
		// if q.Err != nil {
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// defer invigilator_rows.Close()

		// for invigilator_rows.Next() {
		// 	var invigilatorID int64
		// 	q.Err = invigilator_rows.Scan(&invigilatorID)
		// 	if q.Err != nil {
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}
		// 	invigilatorIDs = append(invigilatorIDs, invigilatorID)
		// }

		// // 获取考场信息
		// var examRoomsConfigs []ExamRoomConfig
		// if ei.ExamRoomInvigilatorCount != nil {
		// 	if err := json.Unmarshal([]byte(ei.ExamRoomInvigilatorCount), &examRoomsConfigs); err != nil {
		// 		q.Err = fmt.Errorf("解析考场监考员配置失败: %s", err.Error())
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}
		// }

		// var examRoomIDs []int64
		// for _, config := range examRoomsConfigs {
		// 	examRoomIDs = append(examRoomIDs, config.RoomID)
		// }

		// var examRooms []cmn.TExamRoom
		// if len(examRoomIDs) > 0 {
		// 	examRooms, q.Err = getExamRoomCapacity(examRoomIDs)
		// 	if q.Err != nil {
		// 		q.RespErr()
		// 		return
		// 	}
		// }

		// // 将考场容量转换为映射
		// roomCapacityMap := make(map[int64]int64)
		// for _, room := range examRooms {
		// 	roomCapacityMap[room.ID.Int64] = room.Capacity.Int64
		// }

		// // 根据考场ID匹配对应的容量
		// for i, config := range examRoomsConfigs {
		// 	if capacity, exists := roomCapacityMap[config.RoomID]; exists {
		// 		config.Capacity = capacity
		// 		examRoomsConfigs[i] = config
		// 	}
		// }

		// 构建返回数据
		examData := ExamData{
			ExamInfo:       ei,
			ExamSessions:   es_array,
			ExamineeIDs:    examineeIDs,
			InvigilatorIDs: nil,
			ExamRooms:      nil,
			Files:          examFiles,
		}

		var jsonData []byte
		jsonData, q.Err = json.Marshal(examData)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = jsonData
		q.Resp()

		return

	case "put":
		// 更新考试信息
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查是否具备更新考试的权限
		var result bool
		result = validateUserForExamCreateOrUpdate(userDomain)

		if !result {
			q.Err = fmt.Errorf("用户没有考试相关的权限")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var ExamData ExamData
		q.Err = json.Unmarshal(qry.Data, &ExamData)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Err = validateExamData(ExamData, true)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 检查考试是否存在
		var exists bool
		exists, q.Err = examExists(ctx, ExamData.ExamInfo.ID.Int64)
		if forceErr == "examExists" {
			q.Err = fmt.Errorf("强制检查考试存在错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		if !exists {
			q.Err = fmt.Errorf("考试不存在，请重新创建")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取当前考试状态
		var nowStatus string
		query := `SELECT status FROM t_exam_info WHERE id = $1`
		q.Err = conn.QueryRow(context.Background(), query, ExamData.ExamInfo.ID.Int64).Scan(&nowStatus)
		if forceErr == "conn.QueryRow" {
			q.Err = fmt.Errorf("强制查询当前考试状态错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查考试状态是否允许更新(只有处于未发布、待开始和临时状态的才允许更新)
		if nowStatus != "00" && nowStatus != "02" && nowStatus != "14" {
			q.Err = fmt.Errorf("当前考试状态不允许更新: %s", nowStatus)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取旧考试场次ID
		var oldExamSessionIDs []int64
		oldExamSessionIDs, q.Err = getExamSessionIDs(ctx, ExamData.ExamInfo.ID.Int64)
		if forceErr == "getExamSessionIDs" {
			q.Err = fmt.Errorf("强制获取旧考试场次ID错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 开启事务
		var tx pgx.Tx
		tx, q.Err = conn.Begin(ctx)
		if forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("强制开启事务错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if q.Err != nil || forceErr == "tx.Rollback" {
				err := tx.Rollback(ctx)
				if forceErr == "tx.Rollback" {
					err = fmt.Errorf("强制回滚事务错误")
				}
				if err != nil {
					z.Error(err.Error())
				}

			}
			err := tx.Commit(ctx)
			if forceErr == "tx.Commit" {
				err = fmt.Errorf("强制提交事务错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		// 如果当前考试已经发布（处于待开始状态），则需要先删除已经生成的考卷和答卷以及批改配置
		if nowStatus == "02" {
			// 删除考卷、答卷
			q.Err = examPaper.DeleteExamPaperById(ctx, tx, oldExamSessionIDs, nil)
			if forceErr == "examPaper.DeleteExamPaperById" {
				q.Err = fmt.Errorf("强制删除考卷和答卷错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			// 删除批改配置
			var handleMarkerInfoReq mark.HandleMarkerInfoReq
			handleMarkerInfoReq.ExamSessionIDs = oldExamSessionIDs
			handleMarkerInfoReq.Status = "02"

			q.Err = mark.HandleMarkerInfo(ctx, &tx, userID, handleMarkerInfoReq)
			if forceErr == "mark.HandleMarkerInfo" {
				q.Err = fmt.Errorf("强制处理批改信息错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}
		}

		var updateStatus string
		if nowStatus == "14" {
			updateStatus = "00"
		} else {
			updateStatus = nowStatus
		}

		// 更新考试信息
		updateExamSQL := `
		UPDATE t_exam_info SET
			name = COALESCE(NULLIF($1, ''), name),
			rules = COALESCE(NULLIF($2, ''), rules),
			type = COALESCE(NULLIF($3, ''), type),
			mode = COALESCE(NULLIF($4, ''), mode),
			submitted = COALESCE($5, submitted),
			creator = CASE WHEN status = '14' THEN $6 ELSE creator END,
			create_time = CASE WHEN status = '14' THEN $7 ELSE create_time END,
			updated_by = $6,
			update_time = $7,
			status = COALESCE(NULLIF($8, ''), status),
			addi = COALESCE(NULLIF($9, '{}'::jsonb), addi)
		WHERE id = $10 AND status != '12'`
		_, q.Err = tx.Exec(ctx, updateExamSQL,
			ExamData.ExamInfo.Name.String,
			ExamData.ExamInfo.Rules.String,
			ExamData.ExamInfo.Type.String,
			ExamData.ExamInfo.Mode.String,
			ExamData.ExamInfo.Submitted.Bool,
			userID,
			currentTime,
			updateStatus,
			ExamData.ExamInfo.Addi,
			ExamData.ExamInfo.ID.Int64)
		if forceErr == "tx.UpdateExamInfo" {
			q.Err = fmt.Errorf("强制更新考试信息错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 软删除之前创建的考试场次和考生
		deleteExamSessionsSQL := `
		UPDATE t_exam_session SET status = '14', updated_by = $1, update_time = $2
		WHERE exam_id = $3 AND status != '14'`
		_, q.Err = tx.Exec(ctx, deleteExamSessionsSQL, userID, currentTime, ExamData.ExamInfo.ID.Int64)
		if forceErr == "tx.SoftDeleteExamSessions" {
			q.Err = fmt.Errorf("强制删除考试场次错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		deleteExamineeSQL := `
		UPDATE t_examinee SET status = '08', updated_by = $1, update_time = $2
		FROM t_exam_session es 
		WHERE t_examinee.exam_session_id = es.id 
		AND es.exam_id = $3 
		AND t_examinee.status != '08'`
		_, q.Err = tx.Exec(ctx, deleteExamineeSQL, userID, currentTime, ExamData.ExamInfo.ID.Int64)
		if forceErr == "tx.SoftDeleteExaminee" {
			q.Err = fmt.Errorf("强制删除考生错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 创建新的考试场次
		// 插入考试场次信息
		for index, examSession := range ExamData.ExamSessions {
			var sessionID int64
			q.Err = tx.QueryRow(ctx, `
				INSERT INTO t_exam_session (
					exam_id, session_num, paper_id, start_time, end_time, duration,
					question_shuffled_mode, name_visibility_in, mark_method, mark_mode,
					period_mode, status, creator, create_time, updated_by, update_time,
					late_entry_time, early_submission_time, reviewer_ids
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
				) RETURNING id
			`,
				ExamData.ExamInfo.ID.Int64,
				examSession.SessionNum.Int64,
				examSession.PaperID.Int64,
				examSession.StartTime.Int64,
				examSession.EndTime.Int64,
				examSession.Duration.Int64,
				examSession.QuestionShuffledMode.String,
				examSession.NameVisibilityIn.Bool,
				examSession.MarkMethod,
				examSession.MarkMode.String,
				examSession.PeriodMode.String,
				updateStatus,
				userID,
				currentTime,
				userID,
				currentTime,
				examSession.LateEntryTime.Int64,
				examSession.EarlySubmissionTime.Int64,
				examSession.ReviewerIds,
			).Scan(&sessionID)
			if forceErr == "tx.QueryExamSession" {
				q.Err = fmt.Errorf("强制查询错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			ExamData.ExamSessions[index].ID.Int64 = sessionID
		}

		var examinees []cmn.TExaminee
		for index, examineeID := range ExamData.ExamineeIDs {

			var studentID null.Int
			studentID.Int64 = examineeID
			studentID.Valid = true

			var serialNumber null.Int
			serialNumber.Int64 = int64(index + 1)
			serialNumber.Valid = true

			// 生成准考证号
			var examineeNumber null.String
			examineeNumber.Valid = true
			examineeNumber.String = generateExamineeNumber(serialNumber.Int64, ExamData.ExamInfo, ExamData.ExamSessions)

			examinees = append(examinees, cmn.TExaminee{
				StudentID:      studentID,
				SerialNumber:   serialNumber,
				ExamineeNumber: examineeNumber,
			})
		}

		valueStrings := make([]string, 0, len(examinees)*len(ExamData.ExamSessions))
		valueArgs := make([]interface{}, 0, len(examinees)*len(ExamData.ExamSessions)*14)
		paramCount := 1

		for _, examinee := range examinees {
			for _, examSession := range ExamData.ExamSessions {
				// 为每个学生在每个场次生成一条记录
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
					paramCount, paramCount+1, paramCount+2, paramCount+3, paramCount+4,
					paramCount+5, paramCount+6, paramCount+7, paramCount+8, paramCount+9, paramCount+10, paramCount+11, paramCount+12, paramCount+13))

				valueArgs = append(valueArgs,
					examinee.StudentID,      // student_id
					examinee.ExamRoom,       // exam_room
					examSession.ID.Int64,    // exam_session_id
					nil,                     // exam_paper_id
					nil,                     //start_time 考生作答开始时间
					nil,                     //end_time 作答结束时间
					userID,                  // creator
					currentTime,             // create_time
					userID,                  // updated_by
					currentTime,             // updated_time
					"00",                    // status (正常考试)
					"{}",                    // addi (空JSON对象)
					examinee.SerialNumber,   // serial_number
					examinee.ExamineeNumber, // examinee_number 准考证号
				)

				paramCount += 14
			}
		}

		// 批量插入考生
		if len(valueStrings) > 0 {
			insertQuery := fmt.Sprintf(`
				INSERT INTO t_examinee (
					student_id, exam_room, exam_session_id, 
					exam_paper_id, start_time, end_time, creator, create_time, updated_by, update_time,
					status, addi, serial_number, examinee_number
				) VALUES %s`, strings.Join(valueStrings, ","))

			_, q.Err = tx.Exec(ctx, insertQuery, valueArgs...)
			if forceErr == "tx.InsertExaminees" {
				q.Err = fmt.Errorf("强制执行批量插入考生错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 如果考试未发布，则不需要执行后续的考卷创建等操作
		if nowStatus == "00" || nowStatus == "14" {
			q.Resp()
			return
		}

		// 处于发布状态的考试要重新创建考卷、答卷、定时器
		for _, examSession := range ExamData.ExamSessions {

			examinee_query := `
					SELECT id FROM t_examinee WHERE exam_session_id = $1 AND status = '00'
				`
			rows, err := tx.Query(ctx, examinee_query, examSession.ID.Int64)
			defer func() {
				if rows != nil {
					rows.Close()
				}
			}()
			if forceErr == "tx.SearchExaminee" {
				err = fmt.Errorf("强制查询考生错误")
			}
			if err != nil {
				q.Err = fmt.Errorf("查询考生失败: %s", err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var examineeIDs []int64
			for rows.Next() {
				var examineeID int64
				err := rows.Scan(&examineeID)
				if forceErr == "rows.ScanExamineeID" {
					err = fmt.Errorf("强制获取考生ID错误")
				}
				if err != nil {
					q.Err = fmt.Errorf("获取考生ID失败: %s", err.Error())
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				examineeIDs = append(examineeIDs, examineeID)
			}
			rows.Close() // 立即关闭rows

			if len(examineeIDs) <= 0 {
				break // 如果没有考生，跳过生成考卷和批改配置
			}

			var examPaperID *int64
			var subjectiveQuestionGroups []examPaper.SubjectiveQuestionGroup
			examPaperID, subjectiveQuestionGroups, q.Err = examPaper.GenerateExamPaper(ctx, tx, "00", examSession.PaperID.Int64, 0, examSession.ID.Int64, userID, true)
			if forceErr == "examPaper.GenerateExamPaper" {
				q.Err = fmt.Errorf("强制生成考卷错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}
			_ = subjectiveQuestionGroups

			var isQuestionRandom, isOptionRandom bool
			isOptionRandom = false
			isQuestionRandom = false

			isQuestionRandom, isOptionRandom = getQuestionShuffledMode(examSession.QuestionShuffledMode.String)

			var generateAnswerQuestionsRequest examPaper.GenerateAnswerQuestionsRequest
			generateAnswerQuestionsRequest.ExamPaperID = *examPaperID
			generateAnswerQuestionsRequest.ExamineeIDs = examineeIDs
			generateAnswerQuestionsRequest.IsQuestionRandom = isQuestionRandom
			generateAnswerQuestionsRequest.IsOptionRandom = isOptionRandom
			generateAnswerQuestionsRequest.Category = "00"

			q.Err = examPaper.GenerateAnswerQuestion(ctx, tx, generateAnswerQuestionsRequest, userID)
			if forceErr == "examPaper.GenerateAnswerQuestion" {
				q.Err = fmt.Errorf("强制生成答卷错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 将考卷ID记录到考生表中
			assign_query := `
					UPDATE t_examinee
					SET exam_paper_id = $1,
						updated_by = $2,
						update_time = $3
					WHERE exam_session_id = $4
					AND status != '08'
					`
			_, q.Err = tx.Exec(ctx, assign_query, *examPaperID, userID, currentTime, examSession.ID.Int64)
			if forceErr == "tx.UpdateExamineeExamPaperID" {
				q.Err = fmt.Errorf("强制更新考生考卷ID错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 配置批改员
			var reviewerIDs []int64

			reviewerIDs, q.Err = convertToInt64Array(ctx, examSession.ReviewerIds)
			if forceErr == "convertToInt64Array" {
				q.Err = fmt.Errorf("强制转换批改员ID失败")
			}
			if q.Err != nil {
				q.Err = fmt.Errorf("转换批改员ID失败: %v", q.Err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var examCreator int64
			q.Err = tx.QueryRow(ctx, "SELECT creator FROM t_exam_info WHERE id = $1", ExamData.ExamInfo.ID.Int64).Scan(&examCreator)
			if forceErr == "tx.SearchExamCreator" {
				q.Err = fmt.Errorf("强制查询考试创建者错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if len(reviewerIDs) <= 0 {
				// 如果没有指定批改员，则将考试创建者作为批改员
				reviewerIDs = []int64{examCreator}
				examSession.MarkMode.String = "10"
				examSession.MarkMethod = "00"
			}

			var handleMarkerInfoReq mark.HandleMarkerInfoReq
			handleMarkerInfoReq.ExamSessionID = examSession.ID.Int64
			handleMarkerInfoReq.Markers = reviewerIDs
			handleMarkerInfoReq.ExamineeIDs = examineeIDs
			handleMarkerInfoReq.MarkMode = examSession.MarkMode.String
			handleMarkerInfoReq.QuestionGroups = subjectiveQuestionGroups
			handleMarkerInfoReq.Status = "00"

			q.Err = mark.HandleMarkerInfo(ctx, &tx, userID, handleMarkerInfoReq)
			if forceErr == "mark.HandleMarkerInfo2" {
				q.Err = fmt.Errorf("强制处理批改员信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		q.Err = exam_service.SetExamTimers(ctx, ExamData.ExamInfo.ID.Int64)
		if forceErr == "exam_service.SetExamTimers" {
			q.Err = fmt.Errorf("强制设置考试计时器错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	case "delete":
		// 读取请求体
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "exam-delete-io.ReadAll-err" {
			q.Err = errors.New(forceErr)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			err := q.R.Body.Close()
			if forceErr == "exam-delete-io.Close-err" {
				err = errors.New(forceErr)
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()

		if forceErr == "exam-delete-io.Close-err" {
			return
		}

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if strings.Contains(userDomain, "^student") {
			q.Err = fmt.Errorf("学生用户不能删除考试")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "exam-delete-json.Unmarshal1-err" {
			q.Err = errors.New(forceErr)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examIDs []int64
		q.Err = json.Unmarshal(qry.Data, &examIDs)
		if forceErr == "exam-delete-json.Unmarshal2-err" {
			q.Err = errors.New(forceErr)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(examIDs) == 0 {
			q.Err = fmt.Errorf("没有提供要删除的考试ID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var tryLockSuccess bool

		// 检查用户是否能够删除考试
		for _, examID := range examIDs {

			if examID <= 0 {
				q.Err = fmt.Errorf("无效的考试ID: %d", examID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 获取考试锁
			tryLockSuccess, q.Err = cmn.TryLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Second)
			if forceErr == "cmn.TryLock" {
				q.Err = fmt.Errorf("强制获取考试锁错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if forceErr == "cmn.TryLockFailed" {
				tryLockSuccess = false
			}
			if !tryLockSuccess {
				q.Err = fmt.Errorf("考试正在被其他用户编辑")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		defer func() {
			// 释放考试锁
			for _, examID := range examIDs {
				err := cmn.ReleaseLock(ctx, examID, userID, REDIS_LOCK_PREFIX)
				if forceErr == "cmn.ReleaseLock" {
					err = fmt.Errorf("强制释放考试锁错误")
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
		}()

		// 只有未发布的考试才允许删除
		checkSQL := `
			SELECT COUNT(*) = $2 AS all_exist
			FROM t_exam_info 
			WHERE id = ANY($1) AND status IN ('00', '12')
		`
		var canBeDeleted bool
		q.Err = conn.QueryRow(ctx, checkSQL, examIDs, len(examIDs)).Scan(&canBeDeleted)
		if forceErr == "checkExam" {
			q.Err = fmt.Errorf("强制检查考试存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !canBeDeleted {
			if len(examIDs) == 1 {
				q.Err = fmt.Errorf("考试无法删除")
			} else {
				q.Err = fmt.Errorf("部分考试无法删除")
			}
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if forceErr == "cmn.ReleaseLock" {
			return
		}

		// 获取对应考试的所有考试场次ID
		var examSessionIDs []int64
		examSessionIDs, q.Err = getExamSessionIDs(ctx, examIDs...)
		if forceErr == "getExamSessionIDs" {
			q.Err = fmt.Errorf("强制获取考试场次ID错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开启事务
		var tx pgx.Tx
		tx, q.Err = conn.Begin(ctx)
		if forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("强制开启事务错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if q.Err != nil || forceErr == "tx.Rollback" {
				err := tx.Rollback(ctx)
				if forceErr == "tx.Rollback" {
					err = fmt.Errorf("强制回滚事务错误")
				}
				if err != nil {
					z.Error(err.Error())
				}

			}
			err := tx.Commit(ctx)
			if forceErr == "tx.Commit" {
				err = fmt.Errorf("强制提交事务错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if forceErr == "tx.Commit" || forceErr == "tx.Rollback" {
			return
		}

		// 删除对应的批改信息
		var handleMarkerInfoReq mark.HandleMarkerInfoReq
		handleMarkerInfoReq.ExamSessionIDs = examSessionIDs
		handleMarkerInfoReq.Status = "02"
		q.Err = mark.HandleMarkerInfo(ctx, &tx, userID, handleMarkerInfoReq)
		if forceErr == "mark.HandleMarkerInfo" {
			q.Err = fmt.Errorf("强制删除批改信息错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 删除考卷、答卷等相关信息
		q.Err = examPaper.DeleteExamPaperById(ctx, tx, examSessionIDs, nil)
		if forceErr == "examPaper.DeleteExamPaperById" {
			q.Err = fmt.Errorf("强制删除考卷和答卷错误")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		deleteExamineeSQL := `
		UPDATE t_examinee SET status = '08', updated_by = $1, update_time = $2
		FROM t_exam_session es 
		WHERE t_examinee.exam_session_id = es.id 
		AND es.exam_id = ANY($3) 
		AND t_examinee.status != '08'`
		_, q.Err = tx.Exec(ctx, deleteExamineeSQL, userID, currentTime, examIDs)
		if forceErr == "tx.SoftDeleteExaminee" {
			q.Err = fmt.Errorf("强制删除考生错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 软删除之前创建的考试场次和考生
		deleteExamSessionsSQL := `
		UPDATE t_exam_session SET status = '14', updated_by = $1, update_time = $2
		WHERE exam_id = ANY($3) AND status != '14'`
		_, q.Err = tx.Exec(ctx, deleteExamSessionsSQL, userID, currentTime, examIDs)
		if forceErr == "tx.SoftDeleteExamSessions" {
			q.Err = fmt.Errorf("强制删除考试场次错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		deleteExamInfoSQL := `
		UPDATE t_exam_info SET status = '12', updated_by = $1, update_time = $2
		WHERE id = ANY($3) AND status != '12'`
		_, q.Err = tx.Exec(ctx, deleteExamInfoSQL, userID, currentTime, examIDs)
		if forceErr == "tx.SoftDeleteExamInfo" {
			q.Err = fmt.Errorf("强制删除考试信息错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for _, examID := range examIDs {
			q.Err = exam_service.CancelExamTimers(ctx, examID)
			if forceErr == "exam_service.CancelExamTimers" {
				q.Err = fmt.Errorf("强制删除考试定时器错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// 获取考试列表
func examList(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	// 过滤条件
	type ExamListFilter struct {
		Name      string `json:"Name"`
		Status    string `json:"Status"`
		StartTime int64  `json:"StartTime"`
		EndTime   int64  `json:"EndTime"`
	}

	conn := cmn.GetPgxConn()

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			qry = `{
				"Action": "select",
				"OrderBy": [{"ID": "DESC"}],
				"Filter": {"Name": "", "Status": "", "StartTime": 0, "EndTime": 0},
				"Page": 1,
				"PageSize": 10
			}`
		}

		var req cmn.ReqProto
		q.Err = json.Unmarshal([]byte(qry), &req)
		if forceErr == "json.Unmarshal1" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examFilter ExamListFilter
		if req.Filter != nil {
			var filterBytes []byte
			filterBytes, q.Err = json.Marshal(req.Filter)
			if forceErr == "json.Marshal1" {
				q.Err = fmt.Errorf("强制JSON序列化错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Err = json.Unmarshal(filterBytes, &examFilter)
			if forceErr == "json.Unmarshal2" {
				q.Err = fmt.Errorf("强制JSON解析错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		var userID int64

		userID = q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if examFilter.StartTime > examFilter.EndTime && examFilter.EndTime != 0 {
			q.Err = fmt.Errorf("查询考试时筛选的开始时间不能晚于结束时间")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户拥有的域
		userDomains := q.Domains

		// 当前用户登录选择的域
		userRole := q.SysUser.Role.Int64

		var userDomain string
		userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if req.Page < 0 {
			req.Page = 0
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		var (
			args             []interface{}
			argIdx           = 1
			conditionBuilder strings.Builder
		)

		// 条件语句
		if examFilter.Name != "" {
			conditionBuilder.WriteString(fmt.Sprintf(" AND ei.name ILIKE $%d", argIdx))
			args = append(args, "%"+examFilter.Name+"%")
			argIdx++
		}

		if examFilter.Status != "" {
			conditionBuilder.WriteString(fmt.Sprintf(" AND ei.status = $%d", argIdx))
			args = append(args, examFilter.Status)
			argIdx++
		}

		if examFilter.StartTime > 0 && examFilter.EndTime > 0 {
			conditionBuilder.WriteString(fmt.Sprintf(`
				AND ei.id IN (
					SELECT exam_id FROM t_exam_session
					GROUP BY exam_id
					HAVING MIN(start_time) >= $%d AND MIN(start_time) <= $%d
				)`, argIdx, argIdx+1))
			args = append(args, examFilter.StartTime)
			args = append(args, examFilter.EndTime)
			argIdx += 2
		}

		if strings.Contains(userDomain, "^student") {
			// 学生只能查看自己参与的考试
			conditionBuilder.WriteString(fmt.Sprintf(`
				AND EXISTS (
					SELECT 1 FROM t_examinee e
					JOIN t_exam_session es ON e.exam_session_id = es.id
					WHERE es.exam_id = ei.id 
					  AND e.student_id = $%d 
					  AND e.status != '08' 
					  AND es.status != '14'
				)`, argIdx))
			args = append(args, userID)
			argIdx++
		} else {
			// 其他角色只能查看本人所在域的考试
			domainPrefix := getDomainPrefix(userDomain)

			conditionBuilder.WriteString(fmt.Sprintf(`
			AND (
				d.domain LIKE $%d || '^%%'       
				OR 
				d.domain LIKE $%d || '.%%^%%'     
			)`, argIdx, argIdx+1))
			args = append(args, domainPrefix, domainPrefix)
			argIdx += 2
		}

		var countSQL string

		if strings.Contains(userDomain, "^student") {
			countSQL = "SELECT COUNT(ei.id) FROM t_exam_info ei LEFT JOIN t_domain d ON ei.domain_id = d.id WHERE ei.status NOT IN ('12', '14', '16')" + conditionBuilder.String()
		} else {
			countSQL = "SELECT COUNT(ei.id) FROM t_exam_info ei LEFT JOIN t_domain d ON ei.domain_id = d.id WHERE ei.status NOT IN ('12', '14')" + conditionBuilder.String()
		}

		var rowCount int64
		q.Err = conn.QueryRow(ctx, countSQL, args...).Scan(&rowCount)
		if forceErr == "conn.QueryRow" {
			q.Err = fmt.Errorf("强制查询考试数量错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		offset := (req.Page - 1) * req.PageSize

		var searchSQL string
		var examMap map[int64]*ExamList

		if strings.Contains(userDomain, "^student") {
			searchSQL = `
				SELECT 
					ei.id, 
					ei.name,
					es.id,
					es.start_time, 
					es.end_time,
					es.session_num,
					vep.name,
					es.status,
					e.status,
					vep.total_score,
					CASE 
						WHEN ei.submitted = false THEN -1
						ELSE sc.total_score
					END as total_score
				FROM (
					SELECT ei.id, ei.name, ei.update_time, ei.submitted
					FROM t_exam_info ei
					WHERE ei.status NOT IN ('00','10','12','14','16') ` + conditionBuilder.String() + `
					ORDER BY ei.update_time DESC, ei.id DESC
					LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1) + `
				) ei
				JOIN t_exam_session es
					ON ei.id = es.exam_id
					AND es.status != '14'
				JOIN t_examinee e
					ON e.student_id = $` + strconv.Itoa(argIdx+2) + `
					AND e.exam_session_id = es.id
					AND e.status != '08'
				LEFT JOIN v_exam_paper vep
					ON vep.id = e.exam_paper_id
					AND vep.status != '04'
				LEFT JOIN v_student_exam_total_score sc
					ON sc.exam_session_id = es.id
					AND sc.student_id = $` + strconv.Itoa(argIdx+2) + `
				ORDER BY es.session_num,
					ei.update_time DESC, 
					ei.id DESC
			`
			var rows pgx.Rows
			rows, q.Err = conn.Query(ctx, searchSQL, append(args, req.PageSize, offset, userID)...)
			defer func() {
				if rows != nil {
					rows.Close()
				}
			}()
			if forceErr == "conn.Query" {
				q.Err = fmt.Errorf("强制查询考试列表错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			examMap = make(map[int64]*ExamList)
			for rows.Next() {
				var (
					id                int64
					name              string
					examSessionID     int64
					start_time        int64
					end_time          int64
					sessionNum        null.Int
					paperName         null.String
					examSessionStatus string
					examineeStatus    string
					totalScore        null.Float
					studentScore      null.Float
				)
				q.Err = rows.Scan(
					&id, &name, &examSessionID, &start_time, &end_time, &sessionNum, &paperName, &examSessionStatus,
					&examineeStatus, &totalScore, &studentScore,
				)
				if forceErr == "rows.Scan" {
					q.Err = fmt.Errorf("强制扫描考试列表错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				item, ok := examMap[id]
				if !ok {
					item = &ExamList{
						ID:          id,
						ExamName:    name,
						ExamSession: []ExamSession{},
					}
					examMap[id] = item
				}
				item.ExamSession = append(item.ExamSession, ExamSession{
					ID:             examSessionID,
					StartTime:      start_time,
					EndTime:        end_time,
					SessionNum:     sessionNum.Int64,
					PaperName:      paperName.String,
					Status:         examSessionStatus,
					ExamineeStatus: examineeStatus,
					TotalScore:     totalScore.Float64,
					StudentScore:   studentScore.Float64,
				})
			}
		} else {
			searchSQL = `
			SELECT 
				ei.id, 
				ei.name, 
				ei.type, 
				ei.mode, 
				ei.status, 
				es.start_time, 
				es.end_time, 
				es.duration, 
				es.session_num, 
				es.paper_id,
				ei.update_time,
				COALESCE(
					(SELECT COUNT(*) 
					FROM t_examinee e 
					JOIN t_exam_session es2 ON e.exam_session_id = es2.id 
					WHERE es2.exam_id = ei.id 
					AND e.status != '08' 
					AND es2.status != '14'
					AND es2.session_num = (
						SELECT MIN(session_num) 
						FROM t_exam_session 
						WHERE exam_id = ei.id AND status != '14'
					)
					), 0
				) as num_of_examinees
			FROM t_exam_info ei 
			LEFT JOIN t_exam_session es ON 
				ei.id = es.exam_id AND 
				es.status != '14' 
			LEFT JOIN t_domain d ON
				ei.domain_id = d.id
			WHERE 
				ei.status NOT IN ('12','14') ` + conditionBuilder.String() + ` 
			ORDER BY 
				ei.update_time DESC, 
				ei.id DESC, 
				es.session_num 
			LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)

			var rows pgx.Rows
			rows, q.Err = conn.Query(ctx, searchSQL, append(args, req.PageSize, offset)...)
			defer func() {
				if rows != nil {
					rows.Close()
				}
			}()
			if forceErr == "conn.Query" {
				q.Err = fmt.Errorf("强制查询考试列表错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			examMap = make(map[int64]*ExamList)
			for rows.Next() {
				var (
					id             int64
					name           string
					typ            string
					mode           string
					status         string
					start_time     null.Int
					end_time       null.Int
					duration       null.Int
					sessionNum     null.Int
					paperID        null.Int
					updateTime     int64
					numOfExaminees int64
				)
				q.Err = rows.Scan(
					&id, &name, &typ, &mode, &status,
					&start_time, &end_time, &duration, &sessionNum, &paperID, &updateTime, &numOfExaminees,
				)
				if forceErr == "rows.Scan" {
					q.Err = fmt.Errorf("强制扫描考试列表错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				item, ok := examMap[id]
				if !ok {
					item = &ExamList{
						ID:             id,
						ExamName:       name,
						ExamType:       typ,
						ExamMethod:     mode,
						Status:         status,
						ExamSession:    []ExamSession{},
						Duration:       0,
						NumOfExaminees: numOfExaminees,
					}
					examMap[id] = item
				}
				item.ExamSession = append(item.ExamSession, ExamSession{
					StartTime:  start_time.Int64,
					EndTime:    end_time.Int64,
					SessionNum: sessionNum.Int64,
					PaperID:    paperID.Int64,
				})
				item.Duration += duration.Int64
			}
		}

		// 将考试信息转换为切片
		examList := make([]ExamList, 0)
		for _, item := range examMap {
			examList = append(examList, *item)
		}

		q.Msg.RowCount = rowCount
		q.Msg.Data, q.Err = json.Marshal(examList)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制JSON序列化错误2")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// 考试考生
func examinee(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	// 处理考生相关逻辑
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		// 获取考生信息
		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if examID <= 0 {
			q.Err = fmt.Errorf("无效的考试ID: %d", examID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户拥有的域
		userDomains := q.Domains

		// 当前用户登录选择的域
		userRole := q.SysUser.Role.Int64

		// 从用户域列表中查找当前角色对应的域
		var userDomain string
		userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 验证用户对考试的访问权限
		var hasPermission bool
		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userDomain)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if !hasPermission {
			q.Err = fmt.Errorf("无权限访问该考试")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		query := `
			SELECT DISTINCT e.student_id, e.serial_number, e.examinee_number, u.official_name, u.account, u.id_card_no
			FROM t_examinee e
			JOIN t_exam_session es ON e.exam_session_id = es.id
			INNER JOIN t_user u ON e.student_id = u.id
			WHERE es.exam_id = $1
			AND e.status != '08'
			AND es.status != '14'
			ORDER BY e.serial_number ASC
		`

		var rows pgx.Rows
		rows, q.Err = cmn.GetPgxConn().Query(ctx, query, examID)
		defer func() {
			if rows != nil {
				rows.Close()
			}
		}()
		if forceErr == "conn.Query" {
			q.Err = fmt.Errorf("force error: %s", forceErr)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examinees []Examinee
		for rows.Next() {
			var examinee Examinee
			q.Err = rows.Scan(&examinee.StudentID, &examinee.SerialNumber, &examinee.ExamineeNumber,
				&examinee.OfficialName, &examinee.Account, &examinee.IdCardNo)
			if forceErr == "conn.Scan" {
				q.Err = fmt.Errorf("force error: %s", forceErr)
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examinees = append(examinees, examinee)
		}

		if rows.Err() != nil || forceErr == "rows.Err" {
			q.Err = rows.Err()
			if forceErr == "rows.Err" {
				q.Err = fmt.Errorf("强制行扫描错误")
			}
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 将考生信息转换为JSON
		var jsonData []byte
		jsonData, q.Err = json.Marshal(examinees)
		if forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("强制JSON序列化错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Data = jsonData
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// 考试锁
func examLock(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var userID int64
	userID = q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 用户拥有的域
	userDomains := q.Domains

	// 当前用户登录选择的域
	userRole := q.SysUser.Role.Int64

	var userDomain string
	userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
	if err != nil || examID <= 0 {
		q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var hasPermission bool
	hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userDomain)
	if forceErr == "validateUserExamPermission" {
		q.Err = fmt.Errorf("强制验证用户考试权限错误")
	}
	if q.Err != nil {
		q.RespErr()
		return
	}

	if forceErr == "validateUserExamPermissionFailed" {
		hasPermission = false
	}
	if !hasPermission {
		q.Err = fmt.Errorf("无权访问该考试")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 处理考试锁逻辑
	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":

		// 获取考试锁
		var tryLockSuccess bool
		tryLockSuccess, q.Err = cmn.TryLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Minute)
		if forceErr == "cmn.TryLock" {
			q.Err = fmt.Errorf("强制尝试获取考试锁错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if forceErr == "cmn.TryLockFailed" {
			tryLockSuccess = false
		}
		if !tryLockSuccess {
			q.Err = fmt.Errorf("考试正在被其他用户编辑")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 成功获取考试锁
		q.Msg.Msg = "成功获取考试锁"
		q.Resp()
		return

	case "put":

		// 刷新考试锁
		q.Err = cmn.RefreshLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Minute)
		if forceErr == "cmn.RefreshLock" {
			q.Err = fmt.Errorf("强制刷新考试锁错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 成功刷新考试锁
		q.Msg.Msg = "成功刷新考试锁"
		q.Resp()
		return

	case "delete":

		// 清除考试锁
		q.Err = cmn.ReleaseLock(ctx, examID, userID, REDIS_LOCK_PREFIX)
		if forceErr == "cmn.ReleaseLock" {
			q.Err = fmt.Errorf("强制释放考试锁错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 成功清除考试锁
		q.Msg.Msg = "成功清除考试锁"
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// 考试状态更新
func examStatus(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("examStatus-force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()

	method := strings.ToLower(q.R.Method)
	switch method {
	case "put":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("请指定更新参数q")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户拥有的域
		userDomains := q.Domains

		// 当前用户登录选择的域
		userRole := q.SysUser.Role.Int64

		// 从用户域列表中查找当前角色对应的域
		var userDomain string
		userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if strings.Contains(userDomain, "^student") {
			q.Err = fmt.Errorf("无权限访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examIDsResult := gjson.Get(qry, "data.IDs")
		if !examIDsResult.IsArray() {
			q.Err = fmt.Errorf("data.IDs必须是数组格式")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examIDs []int64
		for _, item := range examIDsResult.Array() {
			examIDs = append(examIDs, item.Int())
		}

		var tryLockSuccess bool

		// 检查用户是否能够操作考试
		for _, examID := range examIDs {
			if examID <= 0 {
				q.Err = fmt.Errorf("无效的考试ID: %d", examID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 获取考试锁
			tryLockSuccess, q.Err = cmn.TryLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Second)
			if forceErr == "cmn.TryLock" {
				q.Err = fmt.Errorf("强制尝试获取考试锁错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if forceErr == "cmn.TryLockFailed" {
				tryLockSuccess = false
			}
			if !tryLockSuccess {
				q.Err = fmt.Errorf("考试正在被其他用户编辑")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		checkExistsSQL := `
			SELECT COUNT(*) = $2 AS all_exist
			FROM t_exam_info 
			WHERE id = ANY($1) AND status != '12'
		`
		var allExist bool
		q.Err = conn.QueryRow(ctx, checkExistsSQL, examIDs, len(examIDs)).Scan(&allExist)
		if forceErr == "checkExamExists" {
			q.Err = fmt.Errorf("强制检查考试存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !allExist {
			if len(examIDs) == 1 {
				q.Err = fmt.Errorf("考试不存在或已被删除")
			} else {
				q.Err = fmt.Errorf("部分考试不存在或已被删除")
			}
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 操作结束后释放考试锁
		defer func() {
			for _, examID := range examIDs {
				err := cmn.ReleaseLock(ctx, examID, userID, REDIS_LOCK_PREFIX)
				if forceErr == "cmn.ReleaseLock" {
					err = fmt.Errorf("强制释放考试锁错误")
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
		}()

		// 获取要修改的考试状态
		status := gjson.Get(qry, "data.Status").String()
		if status == "" {
			q.Err = fmt.Errorf("请指定更新参数data.Status")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		now := time.Now().UnixMilli()

		var tx pgx.Tx
		tx, q.Err = conn.Begin(context.Background())
		if forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("强制开始事务错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if q.Err != nil {
				rollbackErr := tx.Rollback(context.Background())
				if forceErr == "tx.Rollback" {
					rollbackErr = fmt.Errorf("强制回滚事务错误")
				}
				if rollbackErr != nil {
					z.Error(fmt.Sprintf("failed to rollback transaction: %s", rollbackErr.Error()))
				}
				return
			}

			commitErr := tx.Commit(context.Background())
			if forceErr == "tx.Commit" {
				commitErr = fmt.Errorf("强制提交事务错误")
			}
			if commitErr != nil {
				z.Error(fmt.Sprintf("failed to commit transaction: %s", commitErr.Error()))
			}
		}()

		if forceErr == "tx.Rollback" {
			q.Err = fmt.Errorf("强制回滚事务错误")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		switch status {
		case "16":
			// 检查是否有考试的状态不满足作废考试
			checkSQL := `
				SELECT EXISTS(
					SELECT 1 FROM t_exam_info 
					WHERE id = ANY($1) AND status NOT IN ('02','10')
				)
			`
			var result bool
			q.Err = conn.QueryRow(ctx, checkSQL, examIDs).Scan(&result)
			if forceErr == "QueryRow.CheckStatus" {
				q.Err = fmt.Errorf("强制查询错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if result {
				q.Err = fmt.Errorf("尝试作废不属于待开始状态的考试，无法执行作废操作")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 获取当前考试场次ID
			var examSessionIDs []int64
			examSessionIDs, q.Err = getExamSessionIDs(ctx, examIDs...)
			if forceErr == "GetExamSessionIDs" {
				q.Err = fmt.Errorf("强制获取考试场次ID错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 清除批改和考卷配置
			var handleMarkerInfoReq mark.HandleMarkerInfoReq
			handleMarkerInfoReq.ExamSessionIDs = examSessionIDs
			handleMarkerInfoReq.Status = "02"
			q.Err = mark.HandleMarkerInfo(ctx, &tx, userID, handleMarkerInfoReq)
			if forceErr == "mark.HandleMarkerInfo" {
				q.Err = fmt.Errorf("强制处理批改信息错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			// 删除考卷
			q.Err = examPaper.DeleteExamPaperById(ctx, tx, examSessionIDs, nil)
			if forceErr == "examPaper.DeleteExamPaperById" {
				q.Err = fmt.Errorf("强制删除考卷和答卷错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			// 作废考试
			q.Err = updateExamStatus(ctx, tx, "16", userID, examIDs...)
			if forceErr == "updateExamStatus" {
				q.Err = fmt.Errorf("强制更新考试状态错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			q.Err = updateExamSessionStatus(ctx, tx, "16", userID, examIDs...)
			if forceErr == "updateExamSessionStatus" {
				q.Err = fmt.Errorf("强制更新考试场次状态错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			q.Err = updateExamineeStatus(ctx, tx, "16", userID, examIDs...)
			if forceErr == "updateExamineeStatus" {
				q.Err = fmt.Errorf("强制更新考生状态错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			for _, examID := range examIDs {
				q.Err = exam_service.CancelExamTimers(ctx, examID)
				if forceErr == "exam_service.CancelExamTimers" {
					q.Err = fmt.Errorf("强制作废考试定时器错误")
				}
				if q.Err != nil {
					q.RespErr()
					return
				}
			}

		case "02":
			checkSQL := `
				SELECT EXISTS(
					SELECT 1 FROM t_exam_info 
					WHERE id = ANY($1) AND status != '00'
				)
			`
			var result bool
			q.Err = conn.QueryRow(ctx, checkSQL, examIDs).Scan(&result)
			if forceErr == "QueryRow.CheckStatus" {
				q.Err = fmt.Errorf("强制查询错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if result {
				q.Err = fmt.Errorf("尝试发布不属于未发布状态的考试，无法执行发布操作")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 获取当前考试场次信息
			var examSessions []cmn.TExamSession
			examSessions, q.Err = GetExamSessions(ctx, userDomain, examIDs...)
			if forceErr == "GetExamSessions" {
				q.Err = fmt.Errorf("强制获取考试场次错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			for _, examSession := range examSessions {
				// 检查考试场次的开始时间是否在当前时间之前
				if examSession.StartTime.Int64 < now {
					q.Err = fmt.Errorf("考试的开始时间已过，请重新设置考试时间")
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			// 生成考卷并配置批改（如果有考生）
			for _, examSession := range examSessions {

				examinee_query := `
					SELECT id FROM t_examinee WHERE exam_session_id = $1 AND status = '00'
				`
				rows, err := tx.Query(ctx, examinee_query, examSession.ID.Int64)
				if forceErr == "tx.Query" {
					err = fmt.Errorf("强制查询考生错误")
				}
				if err != nil {
					q.Err = fmt.Errorf("查询考生失败: %s", err.Error())
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				var examineeIDs []int64
				for rows.Next() {
					var examineeID int64
					err := rows.Scan(&examineeID)
					if forceErr == "rows.Scan" {
						err = fmt.Errorf("强制获取考生ID错误")
					}
					if err != nil {
						rows.Close()
						q.Err = fmt.Errorf("获取考生ID失败: %s", err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					examineeIDs = append(examineeIDs, examineeID)
				}

				// 立即关闭rows
				rows.Close()

				if len(examineeIDs) <= 0 {
					break // 如果没有考生，跳过生成考卷和批改配置
				}

				var examPaperID *int64
				var subjectiveQuestionGroups []examPaper.SubjectiveQuestionGroup
				examPaperID, subjectiveQuestionGroups, q.Err = examPaper.GenerateExamPaper(ctx, tx, "00", examSession.PaperID.Int64, 0, examSession.ID.Int64, userID, true)
				if forceErr == "examPaper.GenerateExamPaper" {
					q.Err = fmt.Errorf("强制生成考卷错误")
				}
				if q.Err != nil {
					q.RespErr()
					return
				}
				_ = subjectiveQuestionGroups

				var isQuestionRandom, isOptionRandom bool
				isOptionRandom = false
				isQuestionRandom = false

				isQuestionRandom, isOptionRandom = getQuestionShuffledMode(examSession.QuestionShuffledMode.String)

				var generateAnswerQuestionsRequest examPaper.GenerateAnswerQuestionsRequest
				generateAnswerQuestionsRequest.ExamPaperID = *examPaperID
				generateAnswerQuestionsRequest.ExamineeIDs = examineeIDs
				generateAnswerQuestionsRequest.IsQuestionRandom = isQuestionRandom
				generateAnswerQuestionsRequest.IsOptionRandom = isOptionRandom
				generateAnswerQuestionsRequest.Category = "00"

				q.Err = examPaper.GenerateAnswerQuestion(ctx, tx, generateAnswerQuestionsRequest, userID)
				if forceErr == "examPaper.GenerateAnswerQuestion" {
					q.Err = fmt.Errorf("强制生成答卷错误")
				}
				if q.Err != nil {
					q.RespErr()
					return
				}

				// 将考卷ID记录到考生表中
				assign_query := `
					UPDATE t_examinee
					SET exam_paper_id = $1,
						updated_by = $2,
						update_time = $3
					WHERE exam_session_id = $4
					AND status != '08'
					`
				_, q.Err = tx.Exec(ctx, assign_query, *examPaperID, userID, now, examSession.ID.Int64)
				if forceErr == "tx.Exec" {
					q.Err = fmt.Errorf("强制更新考生考卷ID错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				// 配置批改员
				var reviewerIDs []int64
				reviewerIDs, q.Err = convertToInt64Array(ctx, examSession.ReviewerIds)
				if forceErr == "convertToInt64Array" {
					q.Err = fmt.Errorf("强制转换批改员ID失败")
				}
				if q.Err != nil {
					q.Err = fmt.Errorf("转换批改员ID失败: %v", q.Err)
					q.RespErr()
					return
				}

				var examCreator int64
				q.Err = tx.QueryRow(ctx, "SELECT creator FROM t_exam_info WHERE id = $1", examSession.ExamID.Int64).Scan(&examCreator)
				if forceErr == "tx.QueryRow" {
					q.Err = fmt.Errorf("强制查询考试创建者错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				if len(reviewerIDs) <= 0 {
					// 如果没有指定批改员，则将考试创建者作为批改员
					reviewerIDs = []int64{examCreator}
					examSession.MarkMode.String = "10"
					examSession.MarkMethod = "00"
				}

				var handleMarkerInfoReq mark.HandleMarkerInfoReq
				handleMarkerInfoReq.ExamSessionID = examSession.ID.Int64
				handleMarkerInfoReq.Markers = reviewerIDs
				handleMarkerInfoReq.ExamineeIDs = examineeIDs
				handleMarkerInfoReq.MarkMode = examSession.MarkMode.String
				handleMarkerInfoReq.QuestionGroups = subjectiveQuestionGroups
				handleMarkerInfoReq.Status = "00"

				q.Err = mark.HandleMarkerInfo(ctx, &tx, userID, handleMarkerInfoReq)
				if forceErr == "mark.HandleMarkerInfo" {
					q.Err = fmt.Errorf("强制处理批改员信息错误")
				}
				if q.Err != nil {
					q.RespErr()
					return
				}
			}

			q.Err = updateExamStatus(ctx, tx, "02", userID, examIDs...)
			if forceErr == "updateExamStatus" {
				q.Err = fmt.Errorf("强制更新考试状态错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			q.Err = updateExamSessionStatus(ctx, tx, "02", userID, examIDs...)
			if forceErr == "updateExamSessionStatus" {
				q.Err = fmt.Errorf("强制更新考试场次状态错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}

			for _, examID := range examIDs {
				q.Err = exam_service.SetExamTimers(ctx, examID)
				if forceErr == "exam_service.SetExamTimers" {
					q.Err = fmt.Errorf("强制设置考试计时器错误")
				}
				if q.Err != nil {
					q.RespErr()
					return
				}
			}

		default:
			q.Err = fmt.Errorf("不支持更新的考试状态: %s", status)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

func examUser(ctx context.Context) {
	// 获取考试相关的用户信息
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		qry := q.R.URL.Query().Get("q")
		if qry == "" {
			q.Err = fmt.Errorf("请指定参数q")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户拥有的域
		userDomains := q.Domains

		// 当前用户登录选择的域
		userRole := q.SysUser.Role.Int64

		// 从用户域列表中查找当前角色对应的域
		var userDomain string
		userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if strings.Contains(userDomain, "^student") {
			q.Err = fmt.Errorf("无权限访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userIDsResult := gjson.Get(qry, "data.IDs")
		if !userIDsResult.IsArray() || forceErr == "notArray" {
			q.Err = fmt.Errorf("data.IDs必须是数组格式")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var userIDs []int64
		for _, item := range userIDsResult.Array() {
			userIDs = append(userIDs, item.Int())
		}

		if len(userIDs) <= 0 {
			q.Msg.Data, q.Err = json.Marshal([]ExamUserInfo{})
			if forceErr == "json.Marshal1" {
				q.Err = fmt.Errorf("强制JSON序列化错误1")
				q.Msg.Data = nil
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Resp()
			return
		}

		query := `
		SELECT 
			id,
			COALESCE(official_name, nickname, account) as name,
			COALESCE(mobile_phone, '') as phone,
			COALESCE(id_card_no, '') as id_card,
			COALESCE(gender, '') as gender
		FROM t_user 
		WHERE id = ANY($1) 
		AND status = '00'
		ORDER BY id`

		var rows pgx.Rows
		rows, q.Err = conn.Query(context.Background(), query, userIDs)
		defer func() {
			if rows != nil {
				rows.Close()
			}
		}()
		if forceErr == "conn.Query" {
			q.Err = fmt.Errorf("强制查询用户信息错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var userInfos []ExamUserInfo
		for rows.Next() {
			var userInfo ExamUserInfo
			q.Err = rows.Scan(
				&userInfo.ID,
				&userInfo.Name,
				&userInfo.MobilePhone,
				&userInfo.IDCardNo,
				&userInfo.Gender,
			)
			if forceErr == "rows.Scan" {
				q.Err = fmt.Errorf("强制获取用户信息错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			userInfos = append(userInfos, userInfo)
		}

		q.Msg.Data, q.Err = json.Marshal(userInfos)
		if forceErr == "json.Marshal2" {
			q.Err = fmt.Errorf("强制JSON序列化错误2")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// 处理考试相关的附件上传逻辑
func examFile(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	conn := cmn.GetPgxConn()

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("无效的用户ID: %d", userID)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 用户拥有的域
	userDomains := q.Domains

	// 当前用户登录选择的域
	userRole := q.SysUser.Role.Int64

	var userDomain string
	userDomain, q.Err = getDomainByUserRole(userRole, userDomains)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 检查是否具备更新考试的权限
	var result bool
	result = validateUserForExamCreateOrUpdate(userDomain)

	if forceErr == "examFile.NoPermission" {
		result = false
	}
	if !result {
		q.Err = fmt.Errorf("用户没有考试相关的权限")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	currentTime := time.Now().UnixMilli()

	// 开启事务
	var tx pgx.Tx
	tx, q.Err = conn.Begin(ctx)
	if forceErr == "tx.Begin" {
		q.Err = fmt.Errorf("强制开始事务错误")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil || forceErr == "tx.Rollback" {
			rollbackErr := tx.Rollback(context.Background())
			if forceErr == "tx.Rollback" {
				rollbackErr = fmt.Errorf("强制回滚事务错误")
			}
			if rollbackErr != nil {
				z.Error(fmt.Sprintf("failed to rollback transaction: %s", rollbackErr.Error()))
			}
			return
		}

		commitErr := tx.Commit(context.Background())
		if forceErr == "tx.Commit" {
			commitErr = fmt.Errorf("强制提交事务错误")
		}
		if commitErr != nil {
			z.Error(fmt.Sprintf("failed to commit transaction: %s", commitErr.Error()))
		}
	}()

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		// 更新考试附件
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examFile ExamFile
		q.Err = json.Unmarshal(qry.Data, &examFile)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		z.Sugar().Infof("examID: %d", examFile.ExamID)

		// 检查考试是否存在
		checkExistsSQL := `
			SELECT EXISTS (
				SELECT 1
				FROM t_exam_info
				WHERE id = $1 AND status != '12'
			)
		`
		var examExist bool
		q.Err = conn.QueryRow(ctx, checkExistsSQL, examFile.ExamID).Scan(&examExist)
		if forceErr == "checkExamExists" {
			q.Err = fmt.Errorf("强制检查考试存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !examExist {
			q.Err = fmt.Errorf("考试不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var filePath string
		filePath = filepath.Join(uploadDir, fmt.Sprintf("%s", examFile.CheckSum))

		var fileID int64
		var digest string

		// 标记是否为新文件
		var isNew bool
		isNew = false

		// 查询是否已存在
		err := tx.QueryRow(ctx, `
			SELECT id, digest FROM t_file
			WHERE digest = $1 AND file_name = $2 AND creator = $3
			LIMIT 1
		`, examFile.CheckSum, examFile.Name, userID).Scan(&fileID, &digest)

		// 如果查询失败，则标记为新文件
		if err == pgx.ErrNoRows {
			isNew = true
		}
		if forceErr == "examFile.tx.QueryRow1" {
			q.Err = fmt.Errorf("强制查询文件错误")
		}
		if err != pgx.ErrNoRows && err != nil {
			q.Err = fmt.Errorf("查询文件失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 不存在则插入
		if isNew {
			q.Err = tx.QueryRow(ctx, `
				INSERT INTO t_file (digest, file_name, path, belongto_path, size, count, creator, create_time, status)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				RETURNING id, digest
			`, examFile.CheckSum, examFile.Name, filePath, filePath, examFile.Size, 1, userID, currentTime, "0").Scan(&fileID, &digest)
			if forceErr == "examFile.tx.QueryRow2" {
				q.Err = fmt.Errorf("强制插入文件信息错误")
			}
			if q.Err != nil {
				q.Err = fmt.Errorf("插入文件信息失败: %v", q.Err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 获取该考试的所有附件，并检查是否有digest相同的文件
		var examFiles []cmn.TVExamFile
		getExamFilesSQL := `
			SELECT file_id, digest, file_name
			FROM v_exam_file
			WHERE exam_id = $1
		`
		var examFileRows pgx.Rows
		examFileRows, q.Err = tx.Query(ctx, getExamFilesSQL, examFile.ExamID)
		defer examFileRows.Close()
		if forceErr == "examFiles.tx.Query" {
			q.Err = fmt.Errorf("强制查询考试文件错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for examFileRows.Next() {
			var examFile cmn.TVExamFile
			q.Err = examFileRows.Scan(&examFile.FileID, &examFile.Digest, &examFile.FileName)
			if forceErr == "examFiles.rows.Scan" {
				q.Err = fmt.Errorf("强制扫描考试文件行错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examFiles = append(examFiles, examFile)
		}

		var handleCase string
		handleCase = "new_file"

		var replacedFileID int64

		// 如果存在相同digest，则查看ID和file_name是否一致
		for i := 0; i < len(examFiles); i++ {

			// 如果file_name不一致，则算作新增
			if examFiles[i].Digest.String == examFile.CheckSum && examFiles[i].FileName.String != examFile.Name {
				handleCase = "new_file"
			}

			// 如果ID和file_name都一致，则不用做任何处理
			if examFiles[i].Digest.String == examFile.CheckSum && examFiles[i].FileName.String == examFile.Name && examFiles[i].FileID.Int64 == fileID {
				handleCase = "no_change"
				break
			}

			// 如果只有ID不一致，则替换该ID，减少被替换的文件的引用计数，并更新files字段
			if examFiles[i].Digest.String == examFile.CheckSum && examFiles[i].FileName.String == examFile.Name && examFiles[i].FileID.Int64 != fileID {
				replacedFileID = examFiles[i].FileID.Int64
				examFiles[i].FileID = null.IntFrom(fileID)
				handleCase = "replace_id"
				break
			}
		}

		// 拿到新的考试附件ID数组
		var examFileIDs []int64
		for i := 0; i < len(examFiles); i++ {
			examFileIDs = append(examFileIDs, examFiles[i].FileID.Int64)
		}

		// 组装返回给前端的文件信息
		var examFileReply []ExamFile
		for i := 0; i < len(examFiles); i++ {
			var fileInfo ExamFile
			fileInfo.CheckSum = examFiles[i].Digest.String
			fileInfo.Name = examFiles[i].FileName.String
			examFileReply = append(examFileReply, fileInfo)
		}

		if handleCase == "new_file" {

			// 在考试附件数组中增加新文件ID
			examFileIDs = append(examFileIDs, fileID)
			examFileReply = append(examFileReply, examFile)
		}

		if handleCase == "replace_id" {

			// 处理被替换的文件
			q.Err = handleDeleteExamFile(ctx, tx, replacedFileID, 1)
			if forceErr == "handleDeleteExamFile.tx.Exec" {
				q.Err = fmt.Errorf("强制删除考试文件错误")
			}
			if q.Err != nil {
				q.RespErr()
				return
			}
		}

		// 更新考试附件字段
		if handleCase != "no_change" {

			// 如果该文件不是新上传的，则增加引用计数
			if !isNew {
				_, q.Err = tx.Exec(ctx, `
					UPDATE t_file
					SET count = count + 1
					WHERE id = $1
				`, fileID)
				if forceErr == "examFile.tx.UpdateFileCount" {
					q.Err = fmt.Errorf("强制更新文件引用计数错误")
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			var filesJSON []byte
			filesJSON, q.Err = json.Marshal(examFileIDs)
			if forceErr == "examFiles.json.Marshal" {
				q.Err = fmt.Errorf("强制序列化考试附件ID数组错误")
				q.Msg.Data = nil
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			_, q.Err = tx.Exec(ctx, `
				UPDATE t_exam_info
				SET files = $1
				WHERE id = $2
			`, filesJSON, examFile.ExamID)
			if forceErr == "examInfo.tx.UpdateFiles" {
				q.Err = fmt.Errorf("强制更新考试附件字段错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		q.Msg.Data, q.Err = json.Marshal(examFileReply)
		if forceErr == "examFiles.json.Marshal2" {
			q.Err = fmt.Errorf("强制序列化考试文件回复错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	case "delete":

		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("强制读取请求体错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "io.Close" {
				err = fmt.Errorf("强制关闭IO错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("强制JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var examFile ExamFile
		q.Err = json.Unmarshal(qry.Data, &examFile)
		if forceErr == "json.Unmarshal2" {
			q.Err = fmt.Errorf("强制第二次JSON解析错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查考试是否存在
		checkExistsSQL := `
			SELECT EXISTS (
				SELECT 1
				FROM t_exam_info
				WHERE id = $1 AND status != '12'
			)
		`
		var examExist bool
		q.Err = tx.QueryRow(ctx, checkExistsSQL, examFile.ExamID).Scan(&examExist)
		if forceErr == "checkExamExists" {
			q.Err = fmt.Errorf("强制检查考试存在错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !examExist {
			q.Err = fmt.Errorf("考试不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取考试附件信息
		getExamFilesSQL := `
			SELECT file_id, digest, file_name
			FROM v_exam_file
			WHERE exam_id = $1
		`
		var examFiles []cmn.TVExamFile
		var examFileRows pgx.Rows
		examFileRows, q.Err = tx.Query(ctx, getExamFilesSQL, examFile.ExamID)
		defer examFileRows.Close()
		if forceErr == "getExamFiles" {
			q.Err = fmt.Errorf("强制获取考试附件信息错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		for examFileRows.Next() {
			var examFile cmn.TVExamFile
			if err := examFileRows.Scan(&examFile.FileID, &examFile.Digest, &examFile.FileName); err != nil {
				q.Err = fmt.Errorf("强制扫描考试附件信息错误: %w", err)
			}
			examFiles = append(examFiles, examFile)
		}

		// 检查该考试附件是否存在于考试中
		var deleteFileID int64
		deleteFileID = -1
		deleteIndex := -1
		for i, examFileInfo := range examFiles {
			if examFileInfo.Digest.String == examFile.CheckSum && examFileInfo.FileName.String == examFile.Name {

				// 找到匹配的考试附件
				deleteFileID = examFileInfo.FileID.Int64
				deleteIndex = i
				break
			}
		}

		if deleteIndex != -1 {
			// 删除考试附件
			examFiles = append(examFiles[:deleteIndex], examFiles[deleteIndex+1:]...)
		}

		// 组装返回给前端的文件信息
		var examFileReply []ExamFile
		for i := 0; i < len(examFiles); i++ {
			var fileInfo ExamFile
			fileInfo.CheckSum = examFiles[i].Digest.String
			fileInfo.Name = examFiles[i].FileName.String
			examFileReply = append(examFileReply, fileInfo)
		}

		if deleteFileID != -1 {
			q.Err = handleDeleteExamFile(ctx, tx, deleteFileID, 1)
			if forceErr == "handleDeleteExamFile" {
				q.Err = fmt.Errorf("强制删除考试附件错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		q.Msg.Data, q.Err = json.Marshal(examFileReply)
		if forceErr == "examFiles.json.Marshal" {
			q.Err = fmt.Errorf("强制序列化考试文件错误")
			q.Msg.Data = nil
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}
