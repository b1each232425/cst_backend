# 1 后端代码结构说明

## 1.1 backend

## 1.1.1 创建配置文件

请根据自己的操作系统创建对应的配置文件
macOS: ```.config_darwin.json```
linux: ```.config_linux.json```
windows: ```.config_windows.json```

内容请根据相应的样例.config_${OS}_sample.json创建

## 1.1.2 修改pg与redis连接配置

```json
{
  "dbms": {
    "postgresql": {
      "addr": "localhost",
      "db": "kdb",
      "enable": true,
      "port": 6900,
      "pwd": "your pg db pwd",
      "user": "your pg db user"
    },
    "redis": {
      "addr": "localhost",
      "init": false,
      "port": 6910,
      "cert": "your redis requirepass"
    }
  }
}

```

## 1.2 添加API接口sample 

### 1.2.1 创建原代码
复制
    ```backend/serve/serve_template```
为
    ```backend/serve/sample```

### 1.2.2 修改样例代码

1）把template.go改为sample.go

2）把第一行
   ```package serve_template```
      改为
   ```package sample```

3）把下述内容
```golang
//annotation:template-service
//author:{"name":"tom sawyer","tel":"13580452503", "email":"KManager@GMail.com"}
```
中的
tempalte-service改为sample-service
tom sawyer改为你的名称
13580452503改为你的电话
KManager@GMail.com改为你的邮箱

当相应API出故障时可以用上述信息找到你来维护

4）修改以内部分为适合你需要的内容
```golang
		Fn: 你定义的接口函数,

		Path: "/API接口名称",
		Name: "API接口名称",

		WhiteList: 该接口是否可以匿名访问,true: 可以，false: 授权、鉴权后才可访问


```
## 1.3 在代码中使用日志、数据连接、redis、获取登录用户信息


要求
1）请在出错处立即使用Z.Error报告出错；
2）请把原错误对象保存到q.Err中；
3）请使用q.RespErr()向前端报送错误；
4）立即结束处理。


返回给前端的数据封装在q.Msg中，通过q.Resp()返回给前端，其结构说明如下
```golang
type ReplyProto struct {
	//Status, 0: success, others: fault
	Status int `json:"status"`

	//Msg, Action result describe by literal
	Msg string `json:"msg,omitempty"`

	//Data, operand
	Data types.JSONText `json:"data,omitempty"`

	// RowCount, just row count
	RowCount int64 `json:"rowCount,omitempty"`

	//API, call target
	API string `json:"API,omitempty"`

	//Method, using http method
	Method string `json:"method,omitempty"`

	//SN, call order
	SN int `json:"SN,omitempty"`
}
```

细节请参考样例代码

### 1.3.1 源代码相关功能解释
use serve/user.go as example

```golang

//目录名称即为包名称

//Package user management
package user

//user-mgmt即为API名称，请更改为功能对应的名称           
//annotation:user-mgmt-service

//该API对应的作者信息，name:表示名称, tel:表示电话， email:表示邮件
//author:{"name":"user","tel":"18928776452","email":"XUnion@GMail.com"}

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

// z 代表日志输出
var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

//Enroll 注册API到web路由
func Enroll(author string) {
	z.Info("user.Enroll called")

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

	cmn.AddService(&cmn.ServeEndPoint{
		Fn: user,// user即为你定义的服务函数，请根据功能更改为有意义的名称

		Path: "/user",// URL路径，即，前端将通过 SERVER-HOST/api/user 来访问此功能
		Name: "user",// 用来做报错/跆显示功能的名称

		Developer: developer,//
		WhiteList: true,// 如果是true则不需要登录/权限就可以访问此API

		DomainID:      int64(cmn.CDomainSys),
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

//user API功能实现函数
func user(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)

	// 此处会在日志中记录访问的API名称
	z.Info("---->" + cmn.FncName())
	/*
    q.W: 代表请求中的 http.ResponseWriter
    q.R: 代表请求中的 http.Request

    q.Err: 代表后端报的错误
    q.Msg.Msg： 代表后端对运行结果的简单描述，出错时存储错误信息
    q.Msg.Status: 如果为非零，则表示后端出错
    q.Msg.Data: 后端传输给前端的数据，JSON格式
    q.Msg.RowCount: 数据的行数，如果有意义
	*/

	s := "select 1"

	//代表标准的sql连接
	sqldb := cmn.GetDbConn()
	row := sqldb.QueryRow(s)
	var i null.Int
	q.Err = row.Scan(&i)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	z.Info(fmt.Sprintf("%d", i.Int64))

	//代表pgxconn连接, 性能更好，功能更多，但可能不具备迁移性
	pgxdb := cmn.GetPgxConn()
	pgxrow := pgxdb.QueryRow(context.Background(), s)
	q.Err = pgxrow.Scan(&i)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	z.Info(fmt.Sprintf("%d", i.Int64))

	//代表标准的redis连接
	redisconn := cmn.GetRedisConn()
	_, q.Err = redisconn.Do("PING")
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Msg = cmn.FncName()
	q.Resp()
}


```






