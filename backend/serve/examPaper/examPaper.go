package examPaper

//annotation:exam-paper-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"w2w.io/cmn"
	"w2w.io/serve/auth_mgt"
	"w2w.io/serve/registration"
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
		Fn: paperTemplateH,

		Path: "/paperTemplate",
		Name: "paperTemplate",
		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷管理.查询试卷模板",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// just a trial
func paperTemplateH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	userID := q.SysUser.ID.Int64
	if userID <= 0 {
		q.Err = fmt.Errorf("用户ID不合法")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	authority, err := auth_mgt.GetUserAuthority(ctx)
	if err != nil {
		q.Err = fmt.Errorf("获取用户权限失败: %v", err.Error())
		q.RespErr()
		return
	}
	ctx = context.WithValue(ctx, "authority", authority)
	read, _, _, _, err := registration.GetAuthAPIAccessible(ctx, authority, q.Ep.Path)

	// 这里只需要测试每一个接口，返回是否正常数据就可以了

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/practiceS with  http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if !read {
		q.Err = fmt.Errorf("该用户没有读数据的权限")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	id := q.R.URL.Query().Get("id")
	// 代表此时进入到编辑页面了
	var pid int64
	pid, q.Err = strconv.ParseInt(id, 10, 64)
	if q.Err != nil {
		q.Err = fmt.Errorf("练习ID获取失败：%v", q.Err.Error())
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	p, pg, pq, err := LoadPaperTemplateById(ctx, pid, true)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	result := Map{}
	result["paperinfo"] = *p
	result["paperGroup"] = pg
	result["paperQuestion"] = pq

	data, err := json.Marshal(result)
	if err != nil {
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}
	q.Msg.Data = data
	z.Info("---->" + cmn.FncName())
	q.Msg.Msg = "OK"
	q.Resp()
	return

}
