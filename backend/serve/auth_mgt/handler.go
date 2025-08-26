package auth_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"w2w.io/cmn"
)

type Handler interface {
	HandleAddInstitution(ctx context.Context)
	HandleQuerySelectableAPIs(ctx context.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleQuerySelectableAPIs 处理查询可选API的请求
func (h *handler) HandleQuerySelectableAPIs(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("invalid method: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 解析父域
	parentDomain := q.R.URL.Query().Get("parentDomain")
	if parentDomain != "" && !IsValidDomain(parentDomain) {
		q.Err = fmt.Errorf("invalid parentDomain format")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	apis, err := querySelectableAPIs(ctx, parentDomain)
	if err != nil || forceErr == "querySelectableAPIs" {
		q.Err = fmt.Errorf("failed to query selectable APIs: %v", err)
		q.RespErr()
		return
	}

	apisBytes, err := json.Marshal(apis)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal APIs: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = apisBytes
	q.Resp()
	return
}

// HandleAddInstitution 处理新增机构的请求
func (h *handler) HandleAddInstitution(ctx context.Context) {
	// TODO
}
