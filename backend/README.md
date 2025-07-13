# 1 how to integration

## 1.1 backend

```shell
git clone https://git.xkb66.com/dawnfire/w2w-module.git
cd w2w-module
git switch backend

## update to latest
git pull

## "open" directory w2w-module/backend in your goland/vscode 
## take "w2w-module/backend" as __PROJECT_ROOT__
cd backend

## add new backend api to __PROJECT_ROOT__/serve/your_new_api
## !!! DON'T add files directly to __PROJECT_ROOT__/serve/ !!!
## your api should keep in a single directory your_new_api/

# after your_new_api added/tested
git add .
git commit -m "change describe message"
git push

# waiting ci to building complete

# access https://a.qnear.cn/api/your_new_api to verify your code function

# attention
# DON'T alter anything other then __PROJECT_ROOT__/serve/, else ci will ignore it. 

```

## 1.2 frontend

### 1.2.1 从命令行执行命令，获得源代码
```shell
git clone https://git.xkb66.com/dawnfire/w2w-module.git
cd w2w-module
git switch frontend
## update to latest
git pull

## "open" directory w2w-module/frontend/vite-vue-antdv in your vscode 
## take "w2w-module/frontend/vite-vue-antdv" as __PROJECT_ROOT__
cd frontend/vite-vue-antdv

## add new frontend component to __PROJECT_ROOT__/src/modules/your_new_component
## !!! DON'T add files directly to __PROJECT_ROOT__/src/modules/ !!!
## your components should keep in a single directory __PROJECT_ROOT__/src/modules/your_new_component/

# after your_new_component added/tested
git add .
git commit -m "change describe message"
git push

# waiting ci to building complete

# access https://a.qnear.cn/your_new_component to verify your code function

# attention
# DON'T alter anything other then __PROJECT_ROOT__/src/modules/, else ci will ignore it. 

```

## 1.3 backend api structure

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

## 1.4 frontend structure

前端代码结构说明

use /src/modules/user as example

### 1.4.1 directory structure
```
user
├───router
│       user.route.ts
│
└───views
        user-mgr.vue
```

### 1.4.2 user.route.ts
```javascript
import { createRouter, createWebHistory } from 'vue-router';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/user',// 路由路径， 通过 https://a.qnear.cn/user 来访问
      name: 'user', 
      component: () => import('../views/user-mgr.vue'), // 对应的 component
    },
  ],
});

export default router;
```
### 1.4.3 user-mgr.vue

样例```component```没什么特殊的地方。

```javascript
<template>
  hello user jjj pqr
  <a-button type="primary">aaa</a-button>
  mgr
</template>

```

# 2 build protobuf from source

```bash
export ALL_PROXY=socks5://127.0.0.1:9980

git clone https://github.com/protocolbuffers/protobuf.git

cd /Users/kzz/kUser/cUser/protobuf

git submodule update --init --recursive

mkdir build && cd build

cmake .. \
	-DCMAKE_CXX_STANDARD=14 \
	-DCMAKE_INSTALL_PREFIX=$GOPATH \
	-DCMAKE_OSX_ARCHITECTURES="arm64" 


cmake --build . --target install

```

# 3 build grpc from source

```bash
export ALL_PROXY=socks5://127.0.0.1:9980

git clone https://github.com/grpc/grpc.git

cd /Users/kzz/kUser/cUser/grpc

git submodule update --init --recursive

mkdir b && cd b

cmake .. -DCMAKE_CXX_STANDARD=14 -DCMAKE_INSTALL_PREFIX=$GOPATH 

cmake --build . -j8 --target install

```








