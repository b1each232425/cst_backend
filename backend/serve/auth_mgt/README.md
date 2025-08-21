# Auth Management Service 权限管理服务

本文档描述了权限管理服务中公开函数的使用方法和调用示例。

## 函数列表

### 1. GetUserAuthority

**功能描述：** 获取用户权限信息，包括角色、域、API列表和可读域列表。

**函数签名：**
```go
func GetUserAuthority(ctx context.Context) (a *Authority, err error)
```

**参数说明：**
- `ctx`: 上下文对象，需要包含用户信息

**返回值：**

- `*Authority`: 用户权限信息结构体
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    // 假设 ctx 中已经包含了用户信息
    
    authority, err := auth_mgt.GetUserAuthority(ctx)
    if err != nil {
        fmt.Printf("获取用户权限失败: %v\n", err)
        return
    }
    
    fmt.Printf("用户角色: %s\n", authority.Role.Name)
    fmt.Printf("用户域: %s\n", authority.Domain.Name)
    fmt.Printf("API数量: %d\n", len(authority.APIs))
    fmt.Printf("可读域数量: %d\n", len(authority.ReadableDomains))
}
```

### 2. CheckUserAPIAccessible

**功能描述：** 检查用户是否可访问特定API，支持读、写、完全访问模式。

**函数签名：**
```go
func CheckUserAPIAccessible(ctx context.Context, authority *Authority, apiPath string, accessMode string) (bool, error)
```

**参数说明：**
- `ctx`: 上下文对象
- `authority`: 用户权限信息，可以为 nil（会自动获取）
- `apiPath`: API路径
- `accessMode`: 访问模式（"read", "write", "full"）

**返回值：**

- `bool`: 是否有访问权限
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    
    // 方式1：传入已获取的权限信息
    authority, _ := auth_mgt.GetUserAuthority(ctx)
    accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/users", "read")
    if err != nil {
        fmt.Printf("检查API访问权限失败: %v\n", err)
        return
    }
    fmt.Printf("用户是否可访问 /api/users (读): %t\n", accessible)
    
    // 方式2：传入 nil，自动获取权限信息
    accessible, err = auth_mgt.CheckUserAPIAccessible(ctx, nil, "/api/users", "write")
    if err != nil {
        fmt.Printf("检查API访问权限失败: %v\n", err)
        return
    }
    fmt.Printf("用户是否可访问 /api/users (写): %t\n", accessible)
}
```

### 3. CheckUserAPIWritable

**功能描述：** 检查用户对特定API是否有写权限。

**函数签名：**
```go
func CheckUserAPIWritable(ctx context.Context, authority *Authority, apiPath string) (bool, error)
```

**参数说明：**
- `ctx`: 上下文对象
- `authority`: 用户权限信息
- `apiPath`: API路径

**返回值：**
- `bool`: 是否有写权限
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    authority, _ := auth_mgt.GetUserAuthority(ctx)
    
    writable, err := auth_mgt.CheckUserAPIWritable(ctx, authority, "/api/users")
    if err != nil {
        fmt.Printf("检查API写权限失败: %v\n", err)
        return
    }
    
    if writable {
        fmt.Println("用户对 /api/users 有写权限")
    } else {
        fmt.Println("用户对 /api/users 没有写权限")
    }
}
```

### 4. CheckUserAPIReadable

**功能描述：** 检查用户对特定API是否有读权限。

**函数签名：**
```go
func CheckUserAPIReadable(ctx context.Context, authority *Authority, apiPath string) (bool, error)
```

**参数说明：**
- `ctx`: 上下文对象
- `authority`: 用户权限信息
- `apiPath`: API路径

**返回值：**
- `bool`: 是否有读权限
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    authority, _ := auth_mgt.GetUserAuthority(ctx)
    
    readable, err := auth_mgt.CheckUserAPIReadable(ctx, authority, "/api/users")
    if err != nil {
        fmt.Printf("检查API读权限失败: %v\n", err)
        return
    }
    
    if readable {
        fmt.Println("用户对 /api/users 有读权限")
    } else {
        fmt.Println("用户对 /api/users 没有读权限")
    }
}
```

### 5. CheckUserAPIEditable

**功能描述：** 检查用户对特定API是否有读权限。

**函数签名：**
```go
func CheckUserAPIEditable(ctx context.Context, authority *Authority, apiPath string) (bool, error)
```

**参数说明：**
- `ctx`: 上下文对象
- `authority`: 用户权限信息
- `apiPath`: API路径

**返回值：**
- `bool`: 是否有读权限
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    authority, _ := auth_mgt.GetUserAuthority(ctx)
    
    readable, err := auth_mgt.CheckUserAPIEditable(ctx, authority, "/api/users")
    if err != nil {
        fmt.Printf("检查API编辑权限失败: %v\n", err)
        return
    }
    
    if readable {
        fmt.Println("用户对 /api/users 有编辑权限")
    } else {
        fmt.Println("用户对 /api/users 没有编辑权限")
    }
}
```

### 6. GetDomainRelationship

**功能描述：** 获得用户所在域与目标域的关系，判断目标域是用户所在域的什么关系域。

**函数签名：**
```go
func GetDomainRelationship(ctx context.Context, authority *Authority, targetDomain string) (string, error)
```

**参数说明：**
- `ctx`: 上下文对象
- `authority`: 用户权限信息
- `targetDomain`: 目标域名

**返回值：**
- `string`: 域关系类型（"self", "parent", "child", "peer"）
- `error`: 错误信息

**调用示例：**
```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func main() {
    ctx := context.Background()
    authority, _ := auth_mgt.GetUserAuthority(ctx)
    
    relationship, err := auth_mgt.GetDomainRelationship(ctx, authority, "example.com")
    if err != nil {
        fmt.Printf("获取域关系失败: %v\n", err)
        return
    }
    
    switch relationship {
    case "self":
        fmt.Println("目标域是用户当前域")
    case "parent":
        fmt.Println("目标域是用户域的父域")
    case "child":
        fmt.Println("目标域是用户域的子域")
    case "peer":
        fmt.Println("目标域是用户域的同级域")
    default:
        fmt.Printf("未知的域关系: %s\n", relationship)
    }
}
```

## 数据结构

### Authority 结构体

```go
type Authority struct {
    Role                cmn.TDomain         // 用户角色信息
    Domain              cmn.TDomain         // 用户所在域信息
    APIs                []cmn.TVDomainAPI   // 用户的API列表
    AccessibleDomains   []int64             // 用户可访问域ID数组（可访问代表既可读也可编辑，查询数据时可以直接将该ID数组拼接到查询SQL，用户要编辑已有数据时需要先检查目标数据的域是否在这之中）
}
```

## 常量定义

### 访问模式常量
- `CDataAccessModeRead`: 读权限
- `CDataAccessModeWrite`: 写权限
- `CDataAccessModeEdit`: 编辑权限
- `CDataAccessModeFull`: 完全权限

### 域关系常量
- `CDomainRelationshipSelf`: 同一域
- `CDomainRelationshipParent`: 父域
- `CDomainRelationshipChild`: 子域
- `CDomainRelationshipPeer`: 同级域

### 角色优先级常量
- `CDomainPrioritySuperAdmin`: 超级管理员优先级
- `CDomainPriorityAdmin`: 普通管理员优先级
- `CDomainPriorityUser`: 普通用户优先级

## 注意事项

1. **上下文要求**: 所有函数都需要传入包含用户信息的上下文对象。
2. **权限缓存**: 建议在同一个请求中复用 `Authority` 对象，避免重复查询数据库。
5. **API路径**: API路径应该是完整的暴露路径，如 "/api/question-banks"。

## 完整使用示例

```go
package main

import (
    "context"
    "fmt"
    "w2w.io/serve/auth_mgt"
)

func handleUserRequest(ctx context.Context, apiPath string) {
    // 1. 获取用户权限信息
    authority, err := auth_mgt.GetUserAuthority(ctx)
    if err != nil {
        fmt.Printf("获取用户权限失败: %v\n", err)
        return
    }
    
    // 2. 检查API访问权限
    accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "read")
    if err != nil {
        fmt.Printf("检查API访问权限失败: %v\n", err)
        return
    }
    
    if !accessible {
        fmt.Println("用户没有访问权限")
        return
    }
    
    // 3. 检查域关系
    relationship, err := auth_mgt.GetDomainRelationship(ctx, authority, "target.domain.com")
    if err != nil {
        fmt.Printf("获取域关系失败: %v\n", err)
        return
    }
    
    fmt.Printf("用户权限检查通过，域关系: %s\n", relationship)
    
    // 4. 执行业务逻辑
    // ...
}
```