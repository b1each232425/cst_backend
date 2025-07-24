package paper

//annotation:template-service
//author:{"name":"wuzhen","tel":"13424074477", "email":"3117398733@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var actionsWithResult = map[string]bool{
	"add_question": true,
	"add_group":    true,
}

const (
	TIMEOUT = 5 * time.Second
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
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

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: ManualPaper,

		Path: "/paper/manual",
		Name: "paper_manual",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: PaperList,

		Path: "/paper",
		Name: "paper",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	//_ = cmn.AddService(&cmn.ServeEndPoint{
	//	Fn: PaperShareUsers,
	//
	//	Path: "/paper/share/users",
	//	Name: "paper_share",
	//
	//	Developer: developer,
	//	WhiteList: true,
	//
	//	//DomainID 创建该API的账号归属的domain
	//	DomainID: int64(cmn.CDomainSys),
	//
	//	//DefaultDomain 该API将默认授权给的用户
	//	DefaultDomain: int64(cmn.CDomainSys),
	//})
	//
	//_ = cmn.AddService(&cmn.ServeEndPoint{
	//	Fn: PaperShareStatus,
	//
	//	Path: "/paper/share/status",
	//	Name: "paper_status",
	//
	//	Developer: developer,
	//	WhiteList: true,
	//
	//	//DomainID 创建该API的账号归属的domain
	//	DomainID: int64(cmn.CDomainSys),
	//
	//	//DefaultDomain 该API将默认授权给的用户
	//	DefaultDomain: int64(cmn.CDomainSys),
	//})
}

// 创建试卷\更新试卷\获取试卷详情
func ManualPaper(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		db := cmn.GetPgxConn()
		tx, err := db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		var committed bool
		defer func() {
			if p := recover(); p != nil {
				q.Err = tx.Rollback(ctx)
				if q.Err != nil {
					z.Error(q.Err.Error())
				}
				panic(p)
			} else if !committed {
				q.Err = tx.Rollback(ctx)
				if q.Err != nil {
					z.Error(q.Err.Error())
				}
			}
		}()

		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var paper cmn.TPaper
		paper, q.Err = createManualPaperTx(dmlCtx, tx, userID)
		if q.Err != nil {
			q.RespErr()
			return
		}
		var groups []cmn.TPaperGroup
		groups, q.Err = initialManualPaperGroupsTx(dmlCtx, tx, paper.ID.Int64, userID)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if q.Err = tx.Commit(ctx); q.Err != nil {
			q.RespErr()
			return
		}
		committed = true

		result := InitialManualPaperResponse{
			Paper:  paper,
			Groups: groups,
		}
		var buf []byte
		buf, q.Err = json.Marshal(result)
		q.Msg.RowCount = 1
		q.Msg.Data = buf
		q.Resp()
	case "put":
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("Call /api/paper/manual with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取请求的结构体
		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取需要保存到数据库的数据
		var u UpdateManualPaperRequest
		q.Err = json.Unmarshal(qry.Data, &u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//参数校验
		q.Err = cmn.Validate(u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		var results []ActionResult
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//检测试卷是否存在
		var exists bool
		exists, q.Err = paperExists(paperID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if !exists {
			q.Err = fmt.Errorf("试卷不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		results, q.Err = updateManualPaper(dmlCtx, paperID, userID, u)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if results != nil {
			data, _ := json.Marshal(results)
			q.Msg.Data = data
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	case "get":
		paperIDStr := q.R.URL.Query().Get("paper_id")
		var paperID int64
		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//检测试卷是否存在
		var exists bool
		exists, q.Err = paperExists(paperID)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if !exists {
			q.Err = fmt.Errorf("试卷不存在")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		var result *cmn.TVPaper
		result, q.Err = GetManualPaperDetailByPaperID(dmlCtx, paperID)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if result != nil {
			data, _ := json.Marshal(result)
			q.Msg.Data = data
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	}

}

// 试卷首页  列表获取\删除试卷
func PaperList(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		//创建请求体并绑定参数
		var req PaperListRequest
		queryParams := q.R.URL.Query()

		if name := queryParams.Get("name"); name != "" {
			req.Name = name
		}
		if tags := queryParams.Get("tags"); tags != "" {
			req.Tags = tags
		}
		if category := queryParams.Get("category"); category != "" {
			req.Category = category
		}
		req.Page = 1
		if page := queryParams.Get("page"); page != "" {
			if p, err := strconv.Atoi(page); err == nil {
				req.Page = p
			}
		}
		req.PageSize = 10
		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
			if p, err := strconv.Atoi(pageSize); err == nil {
				req.PageSize = p
			}
		}
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		var result []cmn.TVPaper
		var totalCount int64
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		result, totalCount, q.Err = getPaperList(dmlCtx, req, userID)
		if q.Err != nil {
			q.RespErr()
			return
		}
		if result != nil {
			data, _ := json.Marshal(result)
			q.Msg.Data = data
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.RowCount = totalCount
		q.Msg.Msg = "success"
		q.Resp()
	case "delete":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("Call /api/paper/manual with empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取请求的结构体
		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取需要保存到数据库的数据
		var u []int64
		q.Err = json.Unmarshal(qry.Data, &u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		userID := q.SysUser.ID.Int64
		if userID <= 0 {
			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		db := cmn.GetPgxConn()
		var tx pgx.Tx
		tx, q.Err = db.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.Serializable,
		})
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var committed bool
		defer func() {
			if p := recover(); p != nil {
				q.Err = tx.Rollback(ctx)
				if q.Err != nil {
					z.Error(q.Err.Error())
				}
				panic(p)
			} else if !committed {
				q.Err = tx.Rollback(ctx)
				if q.Err != nil {
					z.Error(q.Err.Error())
				}
			}
		}()

		q.Err = deletePapers(dmlCtx, tx, u, userID)
		if q.Err != nil {
			tx.Rollback(ctx)
			q.RespErr()
			return
		}
		if q.Err = tx.Commit(ctx); q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		committed = true
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()

	}
}

// 更新试卷流程
func updateManualPaper(ctx context.Context, paperID, userID int64, req UpdateManualPaperRequest) ([]ActionResult, error) {
	var results []ActionResult

	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				z.Error(err.Error())
				return
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				err = commitErr
			}
		}
	}()

	for _, act := range req.Actions {
		var result interface{}

		switch act.Action {
		case "update_info":
			var basicInfo UpdatePaperBasicInfoRequest
			err := json.Unmarshal(act.Payload, &basicInfo)
			if err != nil {
				z.Error("failed to unmarshal basic info payload: " + err.Error())
				return nil, err
			}
			err = handleUpdateInfo(ctx, tx, paperID, userID, basicInfo)
			if err != nil {
				return nil, err
			}
		case "add_group":
			var req AddQuestionGroupRequest
			if err := json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return nil, err
			}
			err := cmn.Validate(req)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			result, err = handleAddGroup(ctx, tx, paperID, userID, req)
			if err != nil {
				return nil, err
			}
		case "delete_group":
			//解析结构体
			var groupID int64
			err = json.Unmarshal(act.Payload, &groupID)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			err = handleDeleteGroup(ctx, tx, paperID, userID, groupID)
			if err != nil {
				return nil, err
			}
		case "add_question":
			var req []AddQuestionsRequest
			if err := json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return nil, err
			}
			result, err = handleAddQuestions(ctx, tx, userID, req)
			if err != nil {
				return nil, err
			}
		case "delete_question":
			var questionIDs []int64
			err := json.Unmarshal(act.Payload, &questionIDs)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			if len(questionIDs) == 0 {
				z.Error(ErrEmptyQuestionIDs.Error())
				return nil, ErrEmptyQuestionIDs
			}
			err = handleDeleteQuestions(ctx, tx, userID, questionIDs)
			if err != nil {
				return nil, err
			}
		case "update_question":
			var req []UpdatePaperQuestionRequest
			if err := json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return nil, err
			}
			err = handleUpdateQuestions(ctx, tx, userID, req)
			if err != nil {
				return nil, err
			}
		case "update_group":
			var req UpdateQuestionsGroupRequest
			if err := json.Unmarshal(act.Payload, &req); err != nil {
				z.Error(err.Error())
				return nil, err
			}
			err := cmn.Validate(req)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			err = handleUpdateGroup(ctx, tx, userID, req)
			if err != nil {
				return nil, err
			}
		case "move_question":
			var orders []int64
			err := json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			if err := validateIDs(orders); err != nil {
				return nil, err
			}
			err = handleMoveQuestion(ctx, tx, userID, orders)
			if err != nil {
				return nil, err
			}
		case "move_group":
			var orders []int64
			err := json.Unmarshal(act.Payload, &orders)
			if err != nil {
				z.Error(err.Error())
				return nil, err
			}
			if err := validateIDs(orders); err != nil {
				return nil, err
			}
			err = handleMoveQuestionGroup(ctx, tx, paperID, userID, orders)
			if err != nil {
				return nil, err
			}
		default:
			err = fmt.Errorf("unsupported action type: %s", act.Action)
		}

		// 只有指定的操作需要返回结果
		if actionsWithResult[act.Action] {
			results = append(results, ActionResult{
				Action: act.Action,
				Result: result,
			})
		}
	}

	return results, nil
}

//// 试卷共享
//func PaperShareUsers(ctx context.Context) {
//	q := cmn.GetCtxValue(ctx)
//	z.Info("---->" + cmn.FncName())
//
//	method := strings.ToLower(q.R.Method)
//	switch method {
//	case "get":
//		//创建请求体并绑定参数
//		var req GetSharedUserListRequest
//		queryParams := q.R.URL.Query()
//		req.Page = 1
//		if page := queryParams.Get("page"); page != "" {
//			if p, err := strconv.Atoi(page); err == nil {
//				req.Page = p
//			}
//		}
//		req.PageSize = 10
//		if pageSize := queryParams.Get("pageSize"); pageSize != "" {
//			if p, err := strconv.Atoi(pageSize); err == nil {
//				req.PageSize = p
//			}
//		}
//		//获取试卷ID
//		paperIDStr := q.R.URL.Query().Get("paper_id")
//		var paperID int64
//		paperID, q.Err = strconv.ParseInt(paperIDStr, 10, 64)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取过滤值
//		if filter := q.R.URL.Query().Get("filter"); filter != "" {
//			req.Filter = filter
//		}
//		userID := q.SysUser.ID.Int64
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//
//		//获取用户ID
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, paperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//		var shared_users []cmn.TVPaperShare
//		var rouCount int64
//		shared_users, rouCount, q.Err = getPaperShareInfo(dmlCtx, tx, paperID, req)
//		if q.Err != nil {
//			q.RespErr()
//			return
//		}
//		if len(shared_users) != 0 {
//			data, _ := json.Marshal(shared_users)
//			q.Msg.Data = data
//			q.Msg.RowCount = rouCount
//		}
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	case "post":
//		var buf []byte
//		buf, q.Err = io.ReadAll(q.R.Body)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		defer func() {
//			q.Err = q.R.Body.Close()
//			if q.Err != nil {
//				z.Error(q.Err.Error())
//				q.RespErr()
//				return
//			}
//		}()
//
//		if len(buf) == 0 {
//			q.Err = fmt.Errorf("Call /api/paper/share/users with empty body")
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取请求的结构体
//		var qry cmn.ReqProto
//		q.Err = json.Unmarshal(buf, &qry)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取需要保存到数据库的数据
//		var u ManagePaperShareRequest
//		q.Err = json.Unmarshal(qry.Data, &u)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//		userID := q.SysUser.ID.Int64
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, u.PaperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//
//		q.Err = managePaperShareUsers(dmlCtx, tx, u, userID)
//		if q.Err != nil {
//			tx.Rollback()
//			q.RespErr()
//			return
//		}
//		if q.Err = tx.Commit(); q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		committed = true
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	}
//}
//
//// 设置试卷共享状态
//func PaperShareStatus(ctx context.Context) {
//	q := cmn.GetCtxValue(ctx)
//	z.Info("---->" + cmn.FncName())
//	method := strings.ToLower(q.R.Method)
//	switch method {
//	case "put":
//		var buf []byte
//		buf, q.Err = io.ReadAll(q.R.Body)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		defer func() {
//			q.Err = q.R.Body.Close()
//			if q.Err != nil {
//				z.Error(q.Err.Error())
//				q.RespErr()
//				return
//			}
//		}()
//
//		if len(buf) == 0 {
//			q.Err = fmt.Errorf("Call /api/paper/share/status with empty body")
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取请求的结构体
//		var qry cmn.ReqProto
//		q.Err = json.Unmarshal(buf, &qry)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		//获取需要保存到数据库的数据
//		var u UpdatePaperAccessModeRequest
//		q.Err = json.Unmarshal(qry.Data, &u)
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		userID := q.SysUser.ID.Int64
//		if userID <= 0 {
//			q.Err = fmt.Errorf("Invalid UserID: %d", userID)
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
//		defer cancel()
//
//		db := cmn.GetDbConn()
//		var tx *sql.Tx
//		tx, q.Err = db.BeginTx(ctx, &sql.TxOptions{
//			Isolation: sql.LevelLinearizable,
//			ReadOnly:  false,
//		})
//		if q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		var committed bool
//		defer func() {
//			if p := recover(); p != nil {
//				tx.Rollback()
//				panic(p)
//			} else if !committed {
//				tx.Rollback()
//			}
//		}()
//		var isCreator bool
//		isCreator, q.Err = validateUserIsPaperCreator(dmlCtx, tx, u.PaperID, userID)
//		if q.Err != nil {
//			q.RespErr()
//		}
//		if !isCreator {
//			q.Err = ErrNotPaperCreator
//			z.Error(q.Err.Error())
//			q.RespErr()
//		}
//		q.Err = updatePaperShareStatus(dmlCtx, tx, u, userID)
//		if q.Err != nil {
//			tx.Rollback()
//			q.RespErr()
//			return
//		}
//		if q.Err = tx.Commit(); q.Err != nil {
//			z.Error(q.Err.Error())
//			q.RespErr()
//			return
//		}
//		committed = true
//		q.Err = nil
//		q.Msg.Status = 0
//		q.Msg.Msg = "success"
//		q.Resp()
//	}
//}
