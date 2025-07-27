package grade

//annotation:template-service
//author:{"name":"txl","tel":"19832706790", "email":"188307257@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

const (
	TIMEOUT = 10 * time.Second
)

func init() {
	//Setup package scope variables, just like z, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zz settled")
	})
}

func Enroll(author string) {
	z.Info("message.Enroll called")
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

	// --------------------------------------------------
	//                     成绩列表接口                  |
	// --------------------------------------------------
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeListH,

		Path: "/grade/list",
		Name: "grade/list",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	// --------------------------------------------------
	//                     成绩提交接口                  |
	// --------------------------------------------------
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeSubmissionH,

		Path: "/grade/submission",
		Name: "grade/submission",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	// // --------------------------------------------------
	// //                     成绩分布接口                  |
	// // --------------------------------------------------
	// // gradeDistributionExam
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeDistributionExam,

	// 	Path: "/grade/distribution/exam",
	// 	Name: "grade/distribution/exam",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

	// // gradeDistributionPractice
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeDistributionPractice,

	// 	Path: "/grade/distribution/practice",
	// 	Name: "grade/distribution/practice",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

	// // --------------------------------------------------
	// //                  考生成绩列表接口                 |
	// // --------------------------------------------------
	// // gradeListExamineeExam
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeListExamineeExam,

	// 	Path: "/grade/list/examinee/exam",
	// 	Name: "grade/list/examinee/exam",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

	// // gradeListExamineePractice
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeListExamineePractice,

	// 	Path: "/grade/list/examinee/practice",
	// 	Name: "grade/list/examinee/practice",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

	// // --------------------------------------------------
	// //                     成绩分析接口                  |
	// // --------------------------------------------------
	// // gradeAnalysisExam
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeAnalysisExam,

	// 	Path: "/grade/analysis/exam",
	// 	Name: "grade/analysis/exam",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

	// // gradeAnalysisPractice
	// _ = cmn.AddService(&cmn.ServeEndPoint{
	// 	Fn: gradeAnalysisPractice,

	// 	Path: "/grade/analysis/practice",
	// 	Name: "grade/analysis/practice",

	// 	Developer: developer,
	// 	WhiteList: true,

	// 	//DomainID 创建该API的账号归属的domain
	// 	DomainID: int64(cmn.CDomainSys),

	// 	//DefaultDomain 该API将默认授权给的用户
	// 	DefaultDomain: int64(cmn.CDomainSys),
	// })

}

/*
*

	*
	* @api {GET} /api/grade/list gradeListH 获取成绩列表
	* @apiDescription 获取成绩列表
	* @apiName gradeListH
	* @apiGroup Grade
	*
	* @apiParam (200) {String} [category] 	成绩类型
	* @apiParam (200) {Number} [page] 	页码
	* @apiParam (200) {Number} [pageSize] 每页数量
	* @apiParam (200) {Number} [teacherID] 教师ID
	* @apiParam (200) {Number} [examID] 考试ID
	* @apiParam (200) {String} [name] 考试名称
	* @apiParam (200) {String} [type] 考试类型
	* @apiParam (200) {Number} [submitted] 是否已提交
	*
	* @apiParamExample  {type} Request-Example:
	* {
	*   "category": "exam",
	*   "page": 1,
	*   "pageSize": 10,
	*   "teacherID": 0,
	*   "examID": 0,
	*   "name": "",
	*   "type": "",
	*   "submitted": 0
	* }
	*
	* @apiParamExample  {type} Request-Example:
	* {
	*   "category": "practice",
	*   "page": 1,
	*   "pageSize": 10,
	*   "teacherID": 0,
	*   "practice": 0,
	*   "name": ""
	* }
	*
	*
	* @apiSuccessExample {type} Success-Response:
	* {
	*
	* }
	*
	*
*/
func gradeListH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var req GradeListArgs
		var err error
		var data []byte
		var rowCount int64

		queryParams := q.R.URL.Query()

		// 成绩类型
		if category := queryParams.Get("category"); category != "" {
			req.Category = category
		}

		// 页码
		req.Page = 1
		if page := queryParams.Get("page"); page != "" {
			p, err := strconv.Atoi(page)
			if err != nil {
				q.Err = fmt.Errorf("无效页码: %s", page)
				z.Warn(q.Err.Error())
				q.RespErr()
				return
			}
			req.Page = p
		}

		// 每页数量
		req.PageSize = 10
		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
			p, err := strconv.Atoi(pageSize)
			if err != nil {
				q.Err = fmt.Errorf("无效每页数量: %s", pageSize)
				z.Warn(q.Err.Error())
				q.RespErr()
				return
			}
			req.PageSize = p
		}

		// 教师ID
		if teacherID := queryParams.Get("teacherID"); teacherID != "" {
			p, err := strconv.Atoi(teacherID)
			if err != nil {
				q.Err = fmt.Errorf("无效教师ID: %s", teacherID)
				z.Warn(q.Err.Error())
				q.RespErr()
				return
			}
			req.TeacherID = p
		}

		// 考试名称
		if name := queryParams.Get("name"); name != "" {
			req.Filter.Name = name
		}

		// 及格率
		req.PassScoreRate = 0.6

		switch req.Category {
		case "exam":
			// 考试ID
			if examID := queryParams.Get("examID"); examID != "" {
				p, err := strconv.Atoi(examID)
				if err != nil {
					q.Err = fmt.Errorf("无效考试ID: %s", examID)
					z.Warn(q.Err.Error())
					q.RespErr()
					return
				}
				req.ExamID = p
			}

			// 考试类型
			if examType := queryParams.Get("type"); examType != "" {
				req.Filter.Type = examType
			}

			// 提交状态
			if submitted := queryParams.Get("submitted"); submitted != "" {
				if submitted == "0" {
					req.Filter.Submitted = 0
				} else {
					req.Filter.Submitted = 1
				}
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradeExam
			result, rowCount, err = GradeListExam(dmlCtx, &req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			if result != nil {
				data, err = json.Marshal(result)
				if err != nil {
					q.Err = err
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

			// z.Debug("gradeListExam", zap.Any("result", result))

		case "practice":
			// 练习类型
			if practiceID := queryParams.Get("practiceID"); practiceID != "" {
				p, err := strconv.Atoi(practiceID)
				if err != nil {
					q.Err = fmt.Errorf("无效练习ID: %s", practiceID)
					z.Warn(q.Err.Error())
					q.RespErr()
					return
				}
				req.PracticeID = p
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradePractice
			result, rowCount, err = GradeListPractice(dmlCtx, &req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			if result != nil {
				data, err = json.Marshal(result)
				if err != nil {
					q.Err = err
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}

		default:
			q.Err = fmt.Errorf("unsupported category: %s", req.Category)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = data
		q.Msg.RowCount = rowCount

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

/*
*

	*
	* @api {GET} /api/grade/submission gradeSubmissionH 获取成绩提交列表
	* @apiDescription 提交成绩
	* @apiName gradeSubmissionH
	* @apiGroup Grade
	*
	* @apiParam (200) {Number} [teacher_id] 	教师ID
	* @apiParam (200) {Array}  [exam_ids] 	    考试ID
	*
	* @apiParamExample  {type} Request-Example:
	* {
	*   "teacher_id": 1,
	*   "exam_ids": [1],
	* }
	*
	* @apiParamExample  {type} Request-Example:
	* {
	*   "category": "practice",
	*   "page": 1,
	* }
	*
	*
	* @apiSuccessExample {type} Success-Response:
	* {
	*
	* }
	*
	*
*/
func gradeSubmissionH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	method := strings.ToLower(q.R.Method)
	switch method {
	case "patch":
		var err error
		var args GradeSubmitArgs

		z.Info("---->" + cmn.FncName())

		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
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

		// //
		// teacherID := int64(gjson.Get(string(buf), "data.teacher_id").Num)
		// if teacherID <= 0 {
		// 	q.Err = fmt.Errorf("教师ID无效")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		// args.TeacherID = teacherID
		args.TeacherID = 1575

		examIDQuerys := gjson.GetBytes(buf, "data.exam_ids").Array()
		if len(examIDQuerys) <= 0 {
			q.Err = fmt.Errorf("examIDs为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var examIDs []int
		for _, examIDQuery := range examIDQuerys {
			examIDs = append(examIDs, int(examIDQuery.Num))
		}
		z.Debug("examIDs", zap.Any("examIDs", examIDs))
		args.ExamIDs = examIDs

		err = SetExamGradeSubmitted(ctx, &args)
		if err != nil {
			err = fmt.Errorf("提交考试(教师ID:%v) 错误信息:%s", args.TeacherID, err.Error())
			if errors.Is(err, ErrExamIsNotOver) {
				q.Msg.Msg = "当前选择提交的考试存在未结束的考试"
			} else {
				q.Msg.Msg = fmt.Sprintf("提交考试(教师ID:%v) 错误信息:%s", args.TeacherID, err.Error())
			}
			q.Err = err
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

// func gradeDistributionExam(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())
// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// /teacher/exam-grade/distribution?examID=%d&columnNum=%d
// 		/*
// 			apiParam:
// 				examID
// 				columnNum
// 		*/
// 		var req GradeDistributionExamArgs
// 		queryParams := q.R.URL.Query()

// 		if examID := queryParams.Get("examID"); examID != "" {
// 			if c, err := strconv.Atoi(examID); err == nil {
// 				req.ExamID = c
// 			}
// 		}

// 		if columnNum := queryParams.Get("columnNum"); columnNum != "" {
// 			if c, err := strconv.Atoi(columnNum); err == nil {
// 				req.ColumnNum = c
// 			}
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		var err error
// 		var result ExamGradeDistribution

// 		result, err = GradeDistributionExam(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeDistributionExam", zap.Any("result", result))

// 		data, err := json.Marshal(result)
// 		if err != nil {
// 			q.Err = fmt.Errorf("failed to marshal result: %v", err)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = 1 // since this is a single record query
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func gradeDistributionPractice(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)
// 	z.Info("---->" + cmn.FncName())
// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// /teacher/practice-grade/distribution?practiceID=%d&columnNum=%d
// 		/*
// 			apiParam:
// 				practiceID
// 				columnNum
// 		*/
// 		var req GradeDistributionPracticeArgs
// 		queryParams := q.R.URL.Query()

// 		if practiceID := queryParams.Get("practiceID"); practiceID != "" {
// 			if c, err := strconv.Atoi(practiceID); err == nil {
// 				req.PracticeID = c
// 			}
// 		}

// 		if columnNum := queryParams.Get("columnNum"); columnNum != "" {
// 			if c, err := strconv.Atoi(columnNum); err == nil {
// 				req.ColumnNum = c
// 			}
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		result, err := GradeDistributionPractice(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeDistributionPractice", zap.Any("result", result))

// 		data, err := json.Marshal(result)
// 		if err != nil {
// 			q.Err = fmt.Errorf("failed to marshal result: %v", err)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = 1 // since this is a single record query
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func gradeListExamineeExam(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)

// 	z.Info("---->" + cmn.FncName())

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// /teacher/exam-grade/examinee-grade-list?page=%v&pageSize=%v&examID=%s&keyword=%s
// 		/*
// 			apiParam:
// 				ExamID
// 				TeacherID
// 				ClassID
// 				Page
// 				PageSize
// 				Keyword
// 		*/
// 		var req GradeListExamineeExamArgs
// 		queryParams := q.R.URL.Query()

// 		req.Page = 1
// 		if page := queryParams.Get("page"); page != "" {
// 			if c, err := strconv.Atoi(page); err == nil {
// 				req.Page = c
// 			}
// 		}

// 		req.PageSize = 10
// 		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
// 			if c, err := strconv.Atoi(pageSize); err == nil {
// 				req.PageSize = c
// 			}
// 		}

// 		if examIDs := queryParams.Get("examID"); examIDs != "" {
// 			var intSlice []int
// 			for _, examID := range bytes.Split([]byte(examIDs), []byte(",")) {
// 				intValue, err := strconv.Atoi(string(examID))
// 				if err != nil {
// 					q.Err = err
// 					z.Error(q.Err.Error())
// 					q.RespErr()
// 					return
// 				}
// 				intSlice = append(intSlice, intValue)
// 			}
// 			req.ExamID = intSlice
// 		}

// 		if keyword := queryParams.Get("keyword"); keyword != "" {
// 			req.Filter.Keyword = keyword
// 		}

// 		if teacherID := queryParams.Get("teacherID"); teacherID != "" {
// 			if c, err := strconv.Atoi(teacherID); err == nil {
// 				req.TeacherID = c
// 			}
// 		}

// 		if classID := queryParams.Get("classID"); classID != "" {
// 			if c, err := strconv.Atoi(classID); err == nil {
// 				req.ClassID = c
// 			}
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		result, rowCount, err := GradeListExamineeExam(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeListExamineeExam", zap.Any("result", result))

// 		var data []byte
// 		if result != nil {
// 			data, _ = json.Marshal(result)
// 		}

// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = rowCount
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func gradeListExamineePractice(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)

// 	z.Info("---->" + cmn.FncName())

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// teacher/exam-grade/examinee-grade-list?page=%v&pageSize=%v&examID=%s&keyword=%s?classID=%s&teacherID=%s
// 		/*
// 			apiParam:
// 				PracticeID
// 				TeacherID
// 				ClassID
// 				Page
// 				PageSize
// 				Keyword
// 		*/
// 		var req GradeListExamineePracticeArgs
// 		queryParams := q.R.URL.Query()

// 		req.Page = 1
// 		if page := queryParams.Get("page"); page != "" {
// 			if c, err := strconv.Atoi(page); err == nil {
// 				req.Page = c
// 			}
// 		}

// 		req.PageSize = 10
// 		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
// 			if c, err := strconv.Atoi(pageSize); err == nil {
// 				req.PageSize = c
// 			}
// 		}

// 		if practiceIDs := queryParams.Get("practiceID"); practiceIDs != "" {
// 			var intSlice []int
// 			for _, practiceID := range bytes.Split([]byte(practiceIDs), []byte(",")) {
// 				intValue, err := strconv.Atoi(string(practiceID))
// 				if err != nil {
// 					q.Err = err
// 					z.Error(q.Err.Error())
// 					q.RespErr()
// 					return
// 				}
// 				intSlice = append(intSlice, intValue)
// 			}
// 			req.PracticeID = intSlice
// 		}

// 		if classID := queryParams.Get("classID"); classID != "" {
// 			if c, err := strconv.Atoi(classID); err == nil {
// 				req.ClassID = c
// 			}
// 		}

// 		if teacherID := queryParams.Get("teacherID"); teacherID != "" {
// 			if c, err := strconv.Atoi(teacherID); err == nil {
// 				req.TeacherID = c
// 			}
// 		}

// 		if keyword := queryParams.Get("keyword"); keyword != "" {
// 			req.Filter.Keyword = keyword
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		result, rowCount, err := GradeListExamineePractice(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeListExamineePractice", zap.Any("result", result))

// 		var data []byte
// 		if result != nil {
// 			data, _ = json.Marshal(result)
// 		}

// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = rowCount
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func gradeAnalysisExam(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)

// 	z.Info("---->" + cmn.FncName())

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// teacher/exam-analysis?examSessionID=%d
// 		/*
// 			apiParam:
// 				examSessionID
// 		*/
// 		var req GradeAnalysisExamArgs
// 		queryParams := q.R.URL.Query()

// 		if examSessionID := queryParams.Get("examSessionID"); examSessionID != "" {
// 			if c, err := strconv.Atoi(examSessionID); err == nil {
// 				req.ExamSessionID = c
// 			}
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		result, err := GradeAnalysisExam(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeAnalysisExam", zap.Any("result", result))

// 		var data []byte
// 		data, err = json.Marshal(result)
// 		if err != nil {
// 			q.Err = fmt.Errorf("failed to marshal result: %v", err)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = 1 // since this is a single record query
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }

// func gradeAnalysisPractice(ctx context.Context) {
// 	q := cmn.GetCtxValue(ctx)

// 	z.Info("---->" + cmn.FncName())

// 	method := strings.ToLower(q.R.Method)
// 	switch method {
// 	case "get":
// 		// teacher/practice-analysis?practiceID=%d
// 		/*
// 			apiParam:
// 				practiceID
// 		*/
// 		var req GradeAnalysisPracticeArgs
// 		queryParams := q.R.URL.Query()

// 		if practiceID := queryParams.Get("practiceID"); practiceID != "" {
// 			if c, err := strconv.Atoi(practiceID); err == nil {
// 				req.PracticeID = c
// 			}
// 		}

// 		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
// 		defer cancel()

// 		result, err := GetAnalysisPractice(dmlCtx, req)
// 		if err != nil {
// 			q.Err = err
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}

// 		z.Debug("gradeAnalysisPractice", zap.Any("result", result))

// 		var data []byte
// 		data, err = json.Marshal(result)
// 		if err != nil {
// 			q.Err = fmt.Errorf("failed to marshal result: %v", err)
// 			z.Error(q.Err.Error())
// 			q.RespErr()
// 			return
// 		}
// 		q.Msg.Data = data

// 		q.Err = nil
// 		q.Msg.Status = 0
// 		q.Msg.RowCount = 1 // since this is a single record query
// 		q.Msg.Msg = "success"
// 		q.Resp()
// 	default:
// 		q.Err = fmt.Errorf("unsupported method: %s", method)
// 		z.Warn(q.Err.Error())
// 		q.RespErr()
// 		return
// 	}
// }
