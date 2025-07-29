package exam_mgt

//annotation:exam_mgt-service
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/exam_service"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/mark"
)

var z *zap.Logger

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
}

// 考试场次时间
type ExamSession struct {
	ID             int64   `json:"id"`         //场次ID
	StartTime      int64   `json:"start_time"` //场次开始时间
	EndTime        int64   `json:"end_time"`   //场次结束时间
	SessionNum     int64   `json:"session_num"`
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
	IdCardNo       string      `json:"id_card_no"`   // 学生身份证号
	MobilePhone    string      `json:"mobile_phone"` // 学生电话
}

func init() {
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("exam_mgt zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("exam_mgt.Enroll called")

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
}

// 检查考试是否存在
func examExists(ctx context.Context, examID int64) (bool, error) {
	z.Info("---->" + cmn.FncName())

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
	if err != nil {
		z.Error(err.Error())
		return false, err
	}

	return exists, nil
}

// // 获取考场容量
// func getExamRoomCapacity(roomIDs []int64) ([]cmn.TExamRoom, error) {
// 	z.Info("---->" + cmn.FncName())

// 	if len(roomIDs) == 0 {
// 		return nil, nil
// 	}

// 	conn := cmn.GetPgxConn()
// 	query := `
// 		SELECT
// 			id, capacity
// 		FROM t_exam_room
// 		WHERE id = ANY($1) AND status != '06'
// 		ORDER BY id
// 	`
// 	rows, err := conn.Query(context.Background(), query, roomIDs)
// 	if err != nil {
// 		z.Error(err.Error())
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var examRooms []cmn.TExamRoom
// 	for rows.Next() {
// 		var room cmn.TExamRoom
// 		if err := rows.Scan(
// 			&room.ID,
// 			&room.Capacity,
// 		); err != nil {
// 			z.Error(err.Error())
// 			return nil, err
// 		}
// 		examRooms = append(examRooms, room)
// 	}

// 	return examRooms, nil
// }

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
// 参数: ctx: 上下文, examID: 考试ID, role: 用户角色
// 返回: cmn.TExamInfo, error
func GetExamInfo(ctx context.Context, examID int64, role int64) (cmn.TExamInfo, error) {
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

	if role == 3 || role == 2 {
		query := `
			SELECT 
				id, name, rules, type, mode, files, submitted, creator,
				create_time, updated_by, update_time, status, addi, exam_room_ids, exam_delivery_status, exam_room_invigilator_count
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
			&examInfo.ExamDeliveryStatus,
			&examInfo.ExamRoomInvigilatorCount,
		)
		if forceErr == "conn.Scan" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return cmn.TExamInfo{}, err
		}
	} else if role == 1 {
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
	}

	return examInfo, nil
}

// 获取考试场次信息
// 参数: ctx: 上下文, examID: 考试ID, role: 用户角色(角色信息暂定)
// 返回: cmn.TExamInfo, error
func GetExamSessions(ctx context.Context, examID int64, role int64) ([]cmn.TExamSession, error) {
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

	if examID <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examID)
		z.Error(err.Error())
		return []cmn.TExamSession{}, err
	}

	var es_array []cmn.TExamSession

	conn := cmn.GetPgxConn()
	if role == 1 {
		es_query := `
			SELECT
				id, exam_id, paper_id, mark_method, period_mode,
				start_time, end_time, duration, session_num, late_entry_time,
				early_submission_time
			FROM t_exam_session
			WHERE exam_id = $1 AND status != '14'
			ORDER BY session_num
		`
		var es_rows pgx.Rows
		es_rows, err := conn.Query(context.Background(), es_query, examID)
		if forceErr == "conn.Query" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		defer es_rows.Close()
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
	} else if role == 3 || role == 2 {
		es_query := `
			SELECT
				es.id, es.exam_id, es.paper_id, es.mark_method, es.period_mode,
				es.start_time, es.end_time, es.duration, es.question_shuffled_mode,
				es.mark_mode, es.name_visibility_in, es.session_num, es.late_entry_time,
				es.early_submission_time, es.reviewer_ids,
				p.name as paper_name, p.category as paper_category
			FROM t_exam_session es
			LEFT JOIN t_paper p ON es.paper_id = p.id
			WHERE es.exam_id = $1 AND es.status != '14'
			ORDER BY es.session_num
		`
		var es_rows pgx.Rows
		es_rows, err := conn.Query(context.Background(), es_query, examID)
		if forceErr == "conn.Query" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		defer es_rows.Close()
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

// 验证用户对考试的权限
func validateUserExamPermission(ctx context.Context, userID, examID int64, role int64) (bool, error) {
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
	if role == 3 {
		exists = true
	} else if role == 2 {
		err := conn.QueryRow(context.Background(), `
			SELECT EXISTS(
			SELECT 1
			FROM t_exam_info
			WHERE id = $1
			AND creator = $2)    -- 检查创建者是否为当前用户
		`, examID, userID).Scan(&exists)
		if forceErr == "conn.QueryRow" {
			err = fmt.Errorf("force error: %s", forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return false, err
		}
	} else if role == 1 {
		err := conn.QueryRow(context.Background(), `
			SELECT EXISTS(
			SELECT 1
			FROM t_examinee e
			INNER JOIN t_exam_session es ON e.exam_session_id = es.id
			WHERE es.exam_id = $1
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

func updateExamStatus(ctx context.Context, tx pgx.Tx, examID int64, newStatus string, userID int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if examID <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examID)
		z.Error(err.Error())
		return err
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
		WHERE id = $4 AND status != '12'
	`, newStatus, time.Now().UnixMilli(), userID, examID)
	if forceErr == "tx.Exec" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}

func updateExamSessionStatus(ctx context.Context, tx pgx.Tx, examID int64, newStatus string, userID int64) error {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if examID <= 0 {
		err := fmt.Errorf("无效的考试ID: %d", examID)
		z.Error(err.Error())
		return err
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
		WHERE exam_id = $4 AND status != '14'
	`, newStatus, time.Now().UnixMilli(), userID, examID)
	if forceErr == "tx.Exec" {
		err = fmt.Errorf("force error: %s", forceErr)
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}

func validateExamData(examData ExamData, isUpdate bool) error {
	z.Info("---->" + cmn.FncName())
	if isUpdate && examData.ExamInfo.ID.Int64 <= 0 {
		err := fmt.Errorf("更新考试时传入的考试ID无效: %d", examData.ExamInfo.ID.Int64)
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Name.String == "" {
		err := fmt.Errorf("考试名称不能为空")
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Type.String == "" || (examData.ExamInfo.Type.String != "00" && examData.ExamInfo.Type.String != "02" && examData.ExamInfo.Type.String != "04") {
		err := fmt.Errorf("无效的考试类型: %s", examData.ExamInfo.Type.String)
		z.Error(err.Error())
		return err
	}

	if examData.ExamInfo.Mode.String == "" || (examData.ExamInfo.Mode.String != "00" && examData.ExamInfo.Mode.String != "02") {
		err := fmt.Errorf("无效的考试方式: %s", examData.ExamInfo.Mode.String)
		z.Error(err.Error())
		return err
	}

	if len(examData.ExamSessions) == 0 {
		err := fmt.Errorf("考试场次不能为空")
		z.Error(err.Error())
		return err
	}

	for _, examSession := range examData.ExamSessions {
		if examSession.PaperID.Int64 <= 0 {
			err := fmt.Errorf("考试场次的试卷ID无效: %d", examSession.PaperID.Int64)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMethod == "" || (examSession.MarkMethod != "00" && examSession.MarkMethod != "02") {
			err := fmt.Errorf("考试场次的批卷方式无效: %s", examSession.MarkMethod)
			z.Error(err.Error())
			return err
		}

		if examSession.PeriodMode.String == "" || (examSession.PeriodMode.String != "00" && examSession.PeriodMode.String != "02") {
			err := fmt.Errorf("考试场次的考试时段模式无效: %s", examSession.PeriodMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.Duration.Int64 <= 0 {
			err := fmt.Errorf("考试场次的时长无效: %d", examSession.Duration.Int64)
			z.Error(err.Error())
			return err
		}

		if examSession.QuestionShuffledMode.String == "" || (examSession.QuestionShuffledMode.String != "00" && examSession.QuestionShuffledMode.String != "02" && examSession.QuestionShuffledMode.String != "04" && examSession.QuestionShuffledMode.String != "06") {
			err := fmt.Errorf("考试场次的乱序方式无效: %s", examSession.QuestionShuffledMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMode.String == "" || (examSession.MarkMode.String != "00" && examSession.MarkMode.String != "02" && examSession.MarkMode.String != "04" && examSession.MarkMode.String != "06" && examSession.MarkMode.String != "08" && examSession.MarkMode.String != "10") {
			err := fmt.Errorf("考试场次的批改配置无效: %s", examSession.MarkMode.String)
			z.Error(err.Error())
			return err
		}

		if examSession.StartTime.Int64 >= examSession.EndTime.Int64 {
			err := fmt.Errorf("考试场次的开始时间晚于或等于结束时间")
			z.Error(err.Error())
			return err
		}

		//检查设定的考试时长是否大于考试总时长
		startTime := time.UnixMilli(examSession.StartTime.Int64)
		endTime := time.UnixMilli(examSession.EndTime.Int64)
		totalDuration := endTime.Sub(startTime).Minutes()
		if float64(examSession.Duration.Int64) > totalDuration {
			err := fmt.Errorf("设定的考试时长: %d 大于总时长: %f", examSession.Duration.Int64, totalDuration)
			z.Error(err.Error())
			return err
		}

		if examSession.MarkMethod != "00" {
			examSession.MarkMode.String = "00"
		}
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

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		// 创建考试

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
				err = fmt.Errorf("强制关闭请求体错误")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("call /api/course by post with empty body")
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

		q.Err = validateExamData(ExamData, false)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 为结构体添加创建者和更新者信息
		ExamData.ExamInfo.Creator.Int64 = userID
		ExamData.ExamInfo.UpdatedBy.Int64 = userID
		for i := 0; i < len(ExamData.ExamSessions); i++ {
			ExamData.ExamSessions[i].Creator.Int64 = userID
			ExamData.ExamSessions[i].UpdatedBy.Int64 = userID
		}

		if forceErr == "tx.Begin" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(context.Background())
			cancel()
		}

		var newExamID int64

		// 开启事务
		var tx pgx.Tx
		tx, q.Err = conn.Begin(ctx)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if q.Err != nil || forceErr == "tx.Rollback" {
				if forceErr == "tx.Rollback" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(context.Background())
					cancel()
				}
				err := tx.Rollback(ctx)
				if err != nil {
					z.Error(err.Error())
				}
			} else {
				err := tx.Commit(ctx)
				if forceErr == "tx.Commit" {
					err = fmt.Errorf("强制提交事务错误")
				}
				if err != nil {
					z.Error(err.Error())
				}
			}
		}()

		// 创建考试信息和考试场次

		// // 将考场监考员配置转换为JSON
		// var examRoomInvigilatorCount []byte
		// examRoomInvigilatorCount, q.Err = json.Marshal(ExamData.ExamRooms)
		// if q.Err != nil {
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		// // 获取考场ID数组
		// var examRoomIDs []int64
		// for _, room := range ExamData.ExamRooms {
		// 	examRoomIDs = append(examRoomIDs, room.RoomID)
		// }

		// 插入考试基本信息
		currentTime := time.Now().UnixMilli()
		q.Err = tx.QueryRow(ctx, `
			INSERT INTO t_exam_info (
				name, rules, type, mode, files, submitted, creator, create_time, 
				updated_by, update_time, status, addi, exam_room_ids, exam_delivery_status, 
				exam_room_invigilator_count
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
			) RETURNING id
		`,
			ExamData.ExamInfo.Name.String,
			ExamData.ExamInfo.Rules.String,
			ExamData.ExamInfo.Type.String,
			ExamData.ExamInfo.Mode.String,
			ExamData.ExamInfo.Files,
			false,
			ExamData.ExamInfo.Creator.Int64,
			currentTime,
			ExamData.ExamInfo.UpdatedBy.Int64,
			currentTime,
			"00",
			ExamData.ExamInfo.Addi,
			nil,
			"00",
			nil,
		).Scan(&newExamID)
		if forceErr == "tx.QueryRow1" {
			q.Err = fmt.Errorf("强制查询错误")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 插入考试场次信息
		var examSessionIDs []int64
		for _, examSession := range ExamData.ExamSessions {

			var sessionID int64
			sessionCurrentTime := time.Now().UnixMilli()
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
				newExamID,
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
				"00",
				examSession.Creator.Int64,
				sessionCurrentTime,
				examSession.UpdatedBy.Int64,
				sessionCurrentTime,
				examSession.LateEntryTime.Int64,
				examSession.EarlySubmissionTime.Int64,
				examSession.ReviewerIds,
			).Scan(&sessionID)
			if forceErr == "tx.QueryRow2" {
				q.Err = fmt.Errorf("强制查询错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examSessionIDs = append(examSessionIDs, sessionID)
		}

		// 监考员处理

		// // 如果配置了监考员，检查监考员数量是否足够
		// if len(ExamData.InvigilatorIDs) > 0 && ExamData.ExamInfo.Mode.String == "02" {
		// 	// 计算所有考场需要的监考员总数
		// 	totalRequiredInvigilators := int64(0)
		// 	for _, room := range ExamData.ExamRooms {
		// 		if room.InvigilatorCount > 0 {
		// 			totalRequiredInvigilators += room.InvigilatorCount
		// 		} else {
		// 			// 如果没有配置监考员数量，默认每个考场需要1个监考员
		// 			totalRequiredInvigilators++
		// 		}
		// 	}

		// 	if int64(len(ExamData.InvigilatorIDs)) < totalRequiredInvigilators {
		// 		q.Err = fmt.Errorf("监考员数量不足，至少需要 %d 名监考员，但只提供了 %d 名", totalRequiredInvigilators, len(ExamData.InvigilatorIDs))
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}
		// }

		// 创建考生信息
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

		// 线下考试需要分配考生到考场中

		// // 获取考场容量
		// var examRooms []cmn.TExamRoom
		// examRooms, q.Err = getExamRoomCapacity(examRoomIDs)
		// if q.Err != nil {
		// 	q.RespErr()
		// 	return
		// }

		// var roomCapacity int64
		// roomCapacity = 0
		// for _, room := range examRooms {
		// 	roomCapacity += room.Capacity.Int64
		// }

		// if roomCapacity < int64(len(ExamData.ExamineeIDs)) && ExamData.ExamInfo.Mode.String == "02" {
		// 	q.Err = fmt.Errorf("考场容量不足")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		// if len(ExamData.ExamineeIDs) > 0 && len(examRooms) > len(ExamData.ExamineeIDs) && ExamData.ExamInfo.Mode.String == "02" {
		// 	q.Err = fmt.Errorf("考场过多")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		// // 线下考试时，将考生分配到考场
		// if ExamData.ExamInfo.Mode.String == "02" && len(ExamData.ExamineeIDs) > 0 {
		// 	examinees, q.Err = allocateExamineesToRooms(examinees, examRooms)
		// 	if q.Err != nil {
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}
		// }

		// // 如果配置了监考员，分配监考员到考场
		// var invigilations []cmn.TInvigilation
		// if len(ExamData.InvigilatorIDs) > 0 && len(ExamData.ExamRooms) > 0 && ExamData.ExamInfo.Mode.String == "02" {
		// 	invigilations, q.Err = allocateInvigilatorsToRooms(examSessionIDs, ExamData.ExamRooms, ExamData.InvigilatorIDs)
		// 	if q.Err != nil {
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}

		// 	// 批量插入监考员记录
		// 	if len(invigilations) > 0 {
		// 		invValueStrings := make([]string, 0, len(invigilations))
		// 		invValueArgs := make([]interface{}, 0, len(invigilations)*8)
		// 		invParamCount := 1

		// 		for _, invigilation := range invigilations {
		// 			invValueStrings = append(invValueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
		// 				invParamCount, invParamCount+1, invParamCount+2, invParamCount+3,
		// 				invParamCount+4, invParamCount+5, invParamCount+6, invParamCount+7))

		// 			invValueArgs = append(invValueArgs,
		// 				invigilation.ExamSessionID, // exam_session_id
		// 				invigilation.ExamRoom,      // exam_room
		// 				invigilation.Invigilator,   // invigilator
		// 				userID,                     // creator
		// 				currentTime,                // create_time
		// 				userID,                     // updated_by
		// 				currentTime,                // update_time
		// 				types.JSONText("{}"),       // addi (空JSON对象)
		// 			)
		// 			invParamCount += 8
		// 		}

		// 		// 执行批量插入监考员记录
		// 		invInsertQuery := fmt.Sprintf(`
		// 			INSERT INTO t_invigilation (
		// 				exam_session_id, exam_room, invigilator, creator, create_time,
		// 				updated_by, update_time, addi
		// 			) VALUES %s`, strings.Join(invValueStrings, ","))

		// 		_, q.Err = tx.Exec(ctx, invInsertQuery, invValueArgs...)
		// 		if q.Err != nil {
		// 			z.Error(q.Err.Error())
		// 			q.RespErr()
		// 			return
		// 		}
		// 	}
		// }

		// 创建考生
		valueStrings := make([]string, 0, len(examinees)*len(examSessionIDs))
		valueArgs := make([]interface{}, 0, len(examinees)*len(examSessionIDs)*14)
		paramCount := 1

		for _, examinee := range examinees {
			for _, examSessionID := range examSessionIDs {
				// 为每个学生在每个场次生成一条记录
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
					paramCount, paramCount+1, paramCount+2, paramCount+3, paramCount+4,
					paramCount+5, paramCount+6, paramCount+7, paramCount+8, paramCount+9, paramCount+10, paramCount+11, paramCount+12, paramCount+13))

				valueArgs = append(valueArgs,
					examinee.StudentID,      // student_id
					examinee.ExamRoom,       // exam_room
					examSessionID,           // exam_session_id
					nil,                     // exam_paper_id 生成考卷时才插入，现在为空
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
		if len(examinees) > 0 {
			insertQuery := fmt.Sprintf(`
				INSERT INTO t_examinee (
					student_id, exam_room, exam_session_id, 
					exam_paper_id, start_time, end_time, creator, create_time, updated_by, update_time,
					status, addi, serial_number, examinee_number
				) VALUES %s`, strings.Join(valueStrings, ","))

			_, q.Err = tx.Exec(ctx, insertQuery, valueArgs...)
			if forceErr == "tx.Exec" {
				q.Err = fmt.Errorf("强制执行批量插入考生错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// // 创建考场记录表
		// if ExamData.ExamInfo.Mode.String == "02" && len(ExamData.ExamRooms) > 0 {

		// 	// 构建批量插入的SQL
		// 	query := `
		// 		INSERT INTO t_exam_record (
		// 			exam_room, exam_session, content, basic_eval,
		// 			creator, create_time, updated_by, update_time,
		// 			addi, status
		// 		) VALUES %s
		// 	`

		// 	status := "00"
		// 	content := ""
		// 	basicEval := "00"
		// 	addi := types.JSONText("{}")

		// 	// 准备批量插入的数据
		// 	valueStrings := make([]string, 0, len(examRoomIDs)*len(ExamData.ExamSessions))
		// 	valueArgs := make([]interface{}, 0, len(examRoomIDs)*len(ExamData.ExamSessions)*10)
		// 	argID := 1

		// 	for _, roomID := range examRoomIDs {
		// 		for _, examSessionID := range examSessionIDs {
		// 			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
		// 				argID, argID+1, argID+2, argID+3, argID+4, argID+5, argID+6, argID+7, argID+8, argID+9))
		// 			valueArgs = append(valueArgs,
		// 				roomID, examSessionID, content, basicEval,
		// 				userID, currentTime, userID, currentTime,
		// 				addi, status)
		// 			argID += 10
		// 		}
		// 	}

		// 	// 批量插入考场记录
		// 	sql := fmt.Sprintf(query, strings.Join(valueStrings, ","))
		// 	_, q.Err = tx.Exec(ctx, sql, valueArgs...)
		// 	if q.Err != nil {
		// 		z.Error(q.Err.Error())
		// 		q.RespErr()
		// 		return
		// 	}
		// }

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

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userRole := int64(1)
		if userRole <= 0 {
			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
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
		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
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
		ei, q.Err = GetExamInfo(ctx, examID, userRole)
		if q.Err != nil {
			q.RespErr()
			return
		}

		// 考试场次信息
		var es_array []cmn.TExamSession
		es_array, q.Err = GetExamSessions(ctx, examID, userRole)
		if q.Err != nil {
			q.RespErr()
			return
		}

		if userRole == 1 {
			examData := ExamData{
				ExamInfo:       ei,
				ExamSessions:   es_array,
				ExamineeIDs:    nil,
				InvigilatorIDs: nil,
				ExamRooms:      nil,
				TimeStamp:      time.Now().UnixMilli(),
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
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
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

		userRole, err := strconv.ParseInt(q.R.URL.Query().Get("role"), 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("无效的用户角色: %s", q.R.URL.Query().Get("role"))
			z.Error(q.Err.Error())
			q.RespErr()
			return
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

		// userRole = q.SysUser.Role.Int64
		// if userRole <= 0 {
		// 	q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		// userID = 1574
		// userRole = 2

		if examFilter.StartTime > examFilter.EndTime && examFilter.EndTime != 0 {
			q.Err = fmt.Errorf("查询考试时筛选的开始时间不能晚于结束时间")
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
		if userRole == 2 {
			conditionBuilder.WriteString(fmt.Sprintf(" AND ei.creator = $%d", argIdx))
			args = append(args, userID)
			argIdx++
		}

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

		if userRole == 1 {
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
		}

		countSQL := "SELECT COUNT(ei.id) FROM t_exam_info ei WHERE ei.status!='12' " + conditionBuilder.String()

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

		if userRole == 1 {
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
					sc.total_score
				FROM (
					SELECT ei.id, ei.name, ei.update_time
					FROM t_exam_info ei
					WHERE ei.status NOT IN ('00','10','12') ` + conditionBuilder.String() + `
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
				ORDER BY es.session_num
			`
			var rows pgx.Rows
			rows, q.Err = conn.Query(ctx, searchSQL, append(args, req.PageSize, offset, userID)...)
			if forceErr == "conn.Query" {
				q.Err = fmt.Errorf("强制查询考试列表错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer rows.Close()

			examMap = make(map[int64]*ExamList)
			for rows.Next() {
				var (
					id                int64
					name              string
					sessionID         int64
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
					&id, &name, &sessionID, &start_time, &end_time, &sessionNum, &paperName, &examSessionStatus,
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
			WHERE 
				ei.status != '12' ` + conditionBuilder.String() + ` 
			ORDER BY 
				ei.update_time DESC, 
				ei.id DESC, 
				es.session_num 
			LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)

			var rows pgx.Rows
			rows, q.Err = conn.Query(ctx, searchSQL, append(args, req.PageSize, offset)...)
			if forceErr == "conn.Query" {
				q.Err = fmt.Errorf("强制查询考试列表错误")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer rows.Close()

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
					updateTime     int64
					numOfExaminees int64
				)
				q.Err = rows.Scan(
					&id, &name, &typ, &mode, &status,
					&start_time, &end_time, &duration, &sessionNum, &updateTime, &numOfExaminees,
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
				})
				item.Duration += duration.Int64
			}
		}

		// 将考试信息转换为切片
		var examList []ExamList
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

// 考试批阅员
// func examReviewer(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())
// 	q.Msg.Msg = cmn.FncName()
// 	q.Resp()
// }

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
		userRole := q.SysUser.Role.Int64
		if userRole == 0 {
			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// userRole := int64(2)

		// 验证用户对考试的访问权限
		var hasPermission bool
		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
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
		if forceErr == "conn.Query" {
			q.Err = fmt.Errorf("force error: %s", forceErr)
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer rows.Close()

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
// func examLock(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())

// 	// 处理考试锁逻辑
// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// 获取考试锁
// 		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
// 		if err != nil || examID <= 0 {
// 			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		userID := q.SysUser.ID.Int64
// 		if userID <= 0 {
// 			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		userRole := q.SysUser.Role.Int64
// 		if userRole == 0 {
// 			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 检查用户是否有权限获取考试锁
// 		var hasPermission bool
// 		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
// 		if q.Err != nil {
// 			q.RespErr()
// 			return
// 		}
// 		if !hasPermission {
// 			q.Err = fmt.Errorf("用户没有权限获取考试锁")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		var tryLockSuccess bool
// 		tryLockSuccess, q.Err = cmn.TryLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Minute)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		if !tryLockSuccess {
// 			q.Err = fmt.Errorf("考试正在被其他用户编辑")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 成功获取考试锁
// 		q.Msg.Msg = "成功获取考试锁"
// 		q.Resp()
// 		return

// 	case "put":
// 		// 刷新考试锁
// 		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
// 		if err != nil || examID <= 0 {
// 			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		userID := q.SysUser.ID.Int64
// 		if userID <= 0 {
// 			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		userRole := q.SysUser.Role.Int64
// 		if userRole == 0 {
// 			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 检查用户是否有权限刷新考试锁
// 		var hasPermission bool
// 		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
// 		if q.Err != nil {
// 			q.RespErr()
// 			return
// 		}
// 		if !hasPermission {
// 			q.Err = fmt.Errorf("用户没有权限刷新考试锁")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		q.Err = cmn.RefreshLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Minute)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 成功刷新考试锁
// 		q.Msg.Msg = "成功刷新考试锁"
// 		q.Resp()
// 		return

// 	case "delete":
// 		// 清除考试锁
// 		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
// 		if err != nil || examID <= 0 {
// 			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		userID := q.SysUser.ID.Int64
// 		if userID <= 0 {
// 			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		userRole := q.SysUser.Role.Int64
// 		if userRole == 0 {
// 			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 检查用户是否有权限清除考试锁
// 		var hasPermission bool
// 		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
// 		if q.Err != nil {
// 			q.RespErr()
// 			return
// 		}
// 		if !hasPermission {
// 			q.Err = fmt.Errorf("用户没有权限清除考试锁")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		q.Err = cmn.ReleaseLock(ctx, examID, userID, REDIS_LOCK_PREFIX)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 成功清除考试锁
// 		q.Msg.Msg = "成功清除考试锁"
// 		q.Resp()
// 		return

// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// 考试状态更新
func examStatus(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

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

		userRole := q.SysUser.Role.Int64
		if userRole == 0 {
			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		userRole = int64(2)

		examID := gjson.Get(qry, "data.ID").Int()
		if examID <= 0 {
			q.Err = fmt.Errorf("数据中没有包含考试编号")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 检查用户是否有权限获取考试锁
		var hasPermission bool
		hasPermission, q.Err = validateUserExamPermission(ctx, userID, examID, userRole)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if !hasPermission {
			q.Err = fmt.Errorf("用户没有权限获取考试锁")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var tryLockSuccess bool
		tryLockSuccess, q.Err = cmn.TryLock(ctx, examID, userID, REDIS_LOCK_PREFIX, 5*time.Minute)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !tryLockSuccess {
			q.Err = fmt.Errorf("考试正在被其他用户编辑")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 操作结束后释放考试锁
		defer func() {
			if tryLockSuccess {
				err := cmn.ReleaseLock(ctx, examID, userID, REDIS_LOCK_PREFIX)
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

		// 检查考试是否存在
		var exists bool
		exists, q.Err = examExists(ctx, examID)
		if q.Err != nil {
			q.RespErr()
			return
		}

		if !exists {
			q.Err = fmt.Errorf("考试不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取当前考试状态
		var nowStatus string
		query := `SELECT status FROM t_exam_info WHERE id = $1`
		q.Err = conn.QueryRow(context.Background(), query, examID).Scan(&nowStatus)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		now := time.Now().UnixMilli()

		var tx pgx.Tx
		tx, q.Err = conn.Begin(context.Background())
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if p := recover(); p != nil {
				z.Error(fmt.Sprintf("recovered from panic: %v", p))
				if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
					z.Error(fmt.Sprintf("failed to rollback during panic: %s", rollbackErr.Error()))
				}
			}
			if q.Err != nil {
				if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
					z.Error(fmt.Sprintf("failed to rollback transaction: %s", rollbackErr.Error()))
				}
			} else {
				if commitErr := tx.Commit(context.Background()); commitErr != nil {
					z.Error(fmt.Sprintf("failed to commit transaction: %s", commitErr.Error()))
				}
			}
		}()

		switch status {
		case "00":
			// 取消考试
			if nowStatus != "02" && nowStatus != "10" {
				q.Err = fmt.Errorf("当前考试状态不支持取消操作: %s", nowStatus)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 清除批改和考卷配置

			q.Err = updateExamStatus(ctx, tx, examID, "00", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Err = updateExamSessionStatus(ctx, tx, examID, "00", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Err = exam_service.CancelExamTimers(ctx, examID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "02":
			// 发布考试
			if nowStatus != "00" {
				q.Err = fmt.Errorf("当前考试状态不支持发布操作: %s", nowStatus)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 获取当前考试场次信息
			var examSessions []cmn.TExamSession
			examSessions, q.Err = GetExamSessions(ctx, examID, userRole)
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
				if err != nil {
					q.Err = fmt.Errorf("查询考生失败: %s", err.Error())
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				defer rows.Close()

				var examineeIDs []int64
				for rows.Next() {
					var examineeID int64
					if err := rows.Scan(&examineeID); err != nil {
						q.Err = fmt.Errorf("获取考生ID失败: %s", err.Error())
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					examineeIDs = append(examineeIDs, examineeID)
				}

				if len(examineeIDs) <= 0 {
					break // 如果没有考生，跳过生成考卷和批改配置
				}

				var examPaperID *int64
				var subjectiveQuestionGroups []examPaper.SubjectiveQuestionGroup
				examPaperID, subjectiveQuestionGroups, q.Err = examPaper.GenerateExamPaper(ctx, tx, "00", examSession.PaperID.Int64, 0, examSession.ID.Int64, userID, true)
				if q.Err != nil {
					q.RespErr()
					return
				}
				_ = subjectiveQuestionGroups

				var isQuestionRandom, isOptionRandom bool

				switch examSession.QuestionShuffledMode.String {
				case "00": // 既有试题乱序也有选项乱序
					isQuestionRandom = true
					isOptionRandom = true
				case "02": // 选项乱序
					isQuestionRandom = false
					isOptionRandom = true
				case "04": // 试题乱序
					isQuestionRandom = true
					isOptionRandom = false
				case "06": // 都不选择
					isQuestionRandom = false
					isOptionRandom = false
				default:
					isQuestionRandom = false
					isOptionRandom = false
				}

				var generateAnswerQuestionsRequest examPaper.GenerateAnswerQuestionsRequest
				generateAnswerQuestionsRequest.ExamPaperID = *examPaperID
				generateAnswerQuestionsRequest.ExamineeIDs = examineeIDs
				generateAnswerQuestionsRequest.IsQuestionRandom = isQuestionRandom
				generateAnswerQuestionsRequest.IsOptionRandom = isOptionRandom
				generateAnswerQuestionsRequest.Category = "00"

				q.Err = examPaper.GenerateAnswerQuestion(ctx, tx, generateAnswerQuestionsRequest, userID)
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
				_, q.Err = tx.Exec(ctx, assign_query, *examPaperID, userID, now, examSession.ID.Int64)
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				// 配置批改
				var reviewerIDs []int64
				switch v := examSession.ReviewerIds.(type) {
				case []interface{}:
					reviewerIDs = make([]int64, len(v))
					for i, item := range v {
						switch id := item.(type) {
						case int64:
							reviewerIDs[i] = id
						case int:
							reviewerIDs[i] = int64(id)
						default:
							q.Err = fmt.Errorf("无效的批改员ID类型: %T", id)
							z.Error(q.Err.Error())
							q.RespErr()
							return
						}
					}
				default:
					z.Info("Unknown type", zap.String("type", fmt.Sprintf("%T", v)), zap.Any("value", v))
					q.Err = fmt.Errorf("无效的批改员ID列表类型: %T", v)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}

				var examCreator int64
				q.Err = tx.QueryRow(ctx, "SELECT creator FROM t_exam_info WHERE id = $1", examID).Scan(&examCreator)
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
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			q.Err = updateExamStatus(ctx, tx, examID, "02", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Err = updateExamSessionStatus(ctx, tx, examID, "02", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Err = exam_service.SetExamTimers(ctx, examID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Resp()
			return
		case "12":
			// 删除考试
			if nowStatus != "00" {
				q.Err = fmt.Errorf("当前考试状态不支持删除操作: %s", nowStatus)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 清除考卷配置
			q.Err = updateExamStatus(ctx, tx, examID, "12", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			q.Resp()

			q.Err = updateExamSessionStatus(ctx, tx, examID, "14", userID)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持更新的考试状态: %s", status)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// func examSessions(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())

// 	conn := cmn.GetPgxConn()

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		examID, err := strconv.ParseInt(q.R.URL.Query().Get("exam_id"), 10, 64)
// 		if err != nil {
// 			q.Err = fmt.Errorf("无效的考试ID: %s", q.R.URL.Query().Get("exam_id"))
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		if examID <= 0 {
// 			q.Err = fmt.Errorf("无效的考试ID: %d", examID)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		userID := q.SysUser.ID.Int64
// 		if userID <= 0 {
// 			q.Err = fmt.Errorf("无效的用户ID: %d", userID)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		userRole := q.SysUser.Role.Int64
// 		if userRole == 0 {
// 			q.Err = fmt.Errorf("无效的用户角色: %d", userRole)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 查询考试是否存在
// 		var exists bool
// 		exists, q.Err = examExists(examID)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		if !exists {
// 			q.Err = fmt.Errorf("考试不存在")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 验证用户对考试的访问权限
// 		var hasPermission bool
// 		hasPermission, q.Err = validateUserExamPermission(userID, examID, userRole)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		if !hasPermission {
// 			q.Err = fmt.Errorf("无权限访问该考试")
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		query := `
// 			WITH examinee_papers AS (
// 				SELECT DISTINCT exam_session_id, exam_paper_id
// 				FROM t_examinee
// 				WHERE exam_session_id IN (
// 					SELECT id
// 					FROM t_exam_session
// 					WHERE exam_id = $1 AND status != '14'
// 				)
// 				AND exam_paper_id IS NOT NULL
// 			)
// 			SELECT
// 				es.id as session_id,
// 				es.session_num,
// 				es.start_time,
// 				es.end_time,
// 				es.status,
// 				ep.name as paper_name
// 			FROM t_exam_session es
// 			LEFT JOIN examinee_papers exp ON es.id = exp.exam_session_id
// 			LEFT JOIN t_exam_paper ep ON exp.exam_paper_id = ep.id
// 			WHERE es.exam_id = $1
// 			AND es.status != '14'
// 			ORDER BY es.session_num ASC`

// 		var rows pgx.Rows
// 		rows, q.Err = conn.Query(ctx, query, examID)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		defer rows.Close()

// 		var sessions []cmn.TExamSession
// 		for rows.Next() {
// 			var session cmn.TExamSession

// 			q.Err = rows.Scan(
// 				&session.ID,
// 				&session.SessionNum,
// 				&session.StartTime,
// 				&session.EndTime,
// 				&session.Status,
// 				&session.PaperName,
// 			)
// 			if q.Err != nil {
// 				z.Error(q.Err.Error())
// 				q.RespErr()
// 				return
// 			}
// 			sessions = append(sessions, session)
// 		}

// 		if rows.Err() != nil {
// 			q.Err = rows.Err()
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		// 返回考试场次信息
// 		var jsonData []byte
// 		jsonData, q.Err = json.Marshal(sessions)
// 		if q.Err != nil {
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		q.Msg.Data = jsonData
// 		q.Resp()
// 		return
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func examUser(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())

// 	// conn := cmn.GetPgxConn()

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":

// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }
