# CST Backend - 考试管理系统后端服务

> 一个高性能、企业级的后端服务，采用 Go 语言编写，为 CST（Comprehensive Smart Testing）考试管理系统提供全面的 API 支持。

![GitHub License](https://img.shields.io/badge/license-MIT-blue)
![Go Version](https://img.shields.io/badge/Go-1.23.8+-00ADD8?logo=go)
![Code Coverage](https://img.shields.io/badge/Code%20Coverage-High-green)
![Architecture](https://img.shields.io/badge/Architecture-Microservices-blue)

## 📖 项目概述

**CST Backend** 是一个功能完整的后端系统，为前端应用（Svelte/Vue）提供高效的 REST API 和 GraphQL 接口。系统采用现代 Go 语言技术栈，支持高并发、数据持久化、消息队列、缓存管理等企业级功能。

### 🎯 核心能力

- 🔐 **用户认证与授权** - 基于 JWT 的身份验证、权限管理和会话管理
- 📚 **题库数据管理** - 支持多种题型的增删改查和批量操作
- 📝 **试卷组卷引擎** - 智能组卷算法和手动组卷支持
- 📋 **考试生命周期管理** - 从创建、分配、进行到评分的全流程管理
- 👁️ **实时监考功能** - 通过 WebSocket 实现实时数据同步
- 📊 **数据统计分析** - 成绩统计、分析和可视化数据接口
- 📤 **文件处理系统** - Excel 导入导出、文件上传、图片处理等
- 🔌 **第三方集成** - 支持云存储（Google Cloud、Azure、AWS S3）、邮件服务等
- ⚡ **高性能缓存** - Redis 缓存层、内存数据库、分布式锁
- 📬 **消息队列** - 异步任务处理、事件驱动架构

## 🏗️ 系统架构

### 整体架构设计

```
┌──────────────────────────────────────────────────────────────┐
│                        前端应用层                              │
│                  (Svelte/Vue/React)                          │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                      API 网关层                               │
│               (REST/GraphQL/WebSocket)                       │
└────────────────┬──────────────────────────┬──────────────────┘
                 │                          │
                 ▼                          ▼
        ┌──────────────────┐      ┌──────────────────┐
        │   业务服务层      │      │   中间件层        │
        │  (Authentication)│      │  (Logging, Auth) │
        │  (ExamService)   │      │                  │
        │  (UserService)   │      │                  │
        └────────┬─────────┘      └──────────────────┘
                 │
        ┌────────┴────────────────────────────┐
        │                                     │
        ▼                                     ▼
┌──────────────────────┐        ┌──────────────────────┐
│    数据持久化层      │        │   缓存层              │
│  (PostgreSQL)        │        │ (Redis)              │
│  (BoltDB)            │        │                      │
└──────────────────────┘        └──────────────────────┘
        │
        ▼
┌────────────────���─────────────────────────────────────┐
│          存储与外部服务                                │
│  (Google Cloud Storage / Azure Blob / AWS S3)       │
│  (Email Service / Async Task Queue)                 │
└──────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈详解

### 核心框架与库

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **Web 框架** | Gorilla Mux | 1.8.1 | HTTP 路由和中间件 |
| **GraphQL** | gqlgen | 0.17.78 | GraphQL 服务器生成 |
| **WebSocket** | Gorilla WebSocket | 1.5.3 | 实时双向通信 |
| **日志** | Zap | 1.27.0 | 高性能结构化日志 |
| **日志** | zerolog | 1.32.0 | 轻量级 JSON 日志 |

### 数据存储

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **关系数据库** | pgx | 5.7.5 | PostgreSQL 驱动（高性能） |
| **SQL 工具** | sqlx | 1.4.0 | SQL 查询工具库 |
| **Postgres 驱动** | pq | 1.10.9 | 标准 PostgreSQL 驱动 |
| **嵌入式数据库** | BBolt | 1.4.2 | BoltDB（嵌入式键值存储） |
| **内存数据库** | storm/v3 | 3.2.1 | BoltDB 的 ORM 层 |

### 缓存与消息队列

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **缓存** | go-redis | 9.11.0 | Redis 客户端 |
| **任务队列** | asynq | 0.25.1 | 异步任务队列（基于 Redis） |
| **HTTP 连接池** | fasthttp | 1.64.0 | 高性能 HTTP 客户端 |

### 文件处理

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **Excel 操作** | excelize | - | Excel 文件读写 |
| **Excel 解析** | mscfb | 1.0.4 | OLE2 格式支持 |
| **Excel OLE 解析** | msoleps | 1.0.4 | Office 文档支持 |
| **XSD 解析** | efp | 0.0.1 | Excel 格式处理 |
| **JSON 处理** | gjson/sjson | 1.18.0 / 1.2.5 | JSON 查询和修改 |
| **Pretty JSON** | pretty | 1.2.1 | JSON 格式化 |

### 云存储与上传

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **Google Cloud** | google.golang.org/api | 0.243.0 | Google Cloud API |
| **Google Storage** | cloud.google.com/go/storage | 1.56.0 | Google Cloud Storage 操作 |
| **AWS SDK** | aws-sdk-go-v2 | 1.36.6 | AWS 服务集成 |
| **AWS S3** | aws-sdk-go-v2/service/s3 | 1.84.1 | AWS S3 操作 |
| **Azure SDK** | azure-sdk-for-go | - | Azure 服务集成 |
| **Azure Blob** | azure-sdk-for-go/sdk/storage/azblob | 1.6.2 | Azure Blob Storage |
| **文件上传** | tus-js-client | - | TUS 协议支持（断点续传） |
| **tusd 服务** | tusd | - | TUS 服务器实现 |

### 工具与实用库

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **数据验证** | validator/v10 | 10.27.0 | 结构体字段验证 |
| **UUID 生成** | xid | 1.5.0 | 全局唯一 ID 生成 |
| **电话号码** | phonenumbers | 1.6.5 | 电话号码解析和验证 |
| **邮件服务** | go-mail | 0.6.2 | SMTP 邮件发送 |
| **HTTP 录制** | httptest-recorder | 1.0.0 | HTTP 请求录制和回放 |
| **Toxiproxy** | toxiproxy | 2.7.0 | 网络混沌测试工具 |
| **深拷贝** | deepcopy | - | 对象深层复制 |
| **命令行** | cobra | 1.9.1 | CLI 框架 |
| **配置管理** | viper | 1.20.1 | 配置文件解析 |

### 并发与异步

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **协程池** | ants | 2.11.3 | 高性能协程池 |
| **错误处理** | errors | 0.9.1 | 错误链处理 |
| **并发工具** | conc | 0.3.0 | 并发原语工具库 |
| **断路器** | gobreaker | 2.2.0 | 故障恢复模式 |
| **重试工具** | pester | 1.2.0 | HTTP 重试框架 |

### gRPC 与协议

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **gRPC** | google.golang.org/grpc | 1.74.2 | RPC 框架 |
| **Protocol Buffers** | protobuf | 1.36.6 | 数据序列化 |
| **gRPC 中间件** | go-grpc-middleware | 1.4.0 | gRPC 拦截器 |
| **gRPC Parser** | gqlparser | 2.5.30 | GraphQL 解析器 |

### 监控与性能

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **Prometheus** | client_golang | 1.22.0 | 指标收集和导出 |
| **OpenTelemetry** | go.opentelemetry.io | 1.37.0 | 分布式追踪和指标 |
| **xxHash** | xxhash/v2 | 2.3.0 | 高速哈希算法 |
| **压缩** | compress | 1.18.0 | 数据压缩（Brotli） |

### 安全

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **加密** | golang.org/x/crypto | 0.40.0 | 加密算法库 |
| **JWT** | golang-jwt/jwt | 5.3.0 | JWT 令牌处理 |
| **OpenID Connect** | go-jose | 4.1.1 | OIDC 和 JWT 处理 |
| **SPIFFE** | go-spiffe | 2.5.0 | SPIFFE/SPIRE 支持 |

### 测试

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| **单元测试** | testify | 1.10.0 | 断言库 |
| **Mock 工具** | golang/mock | 1.6.0 | Mock 代码生成 |
| **差异对比** | go-diff | 1.4.0 | 代码差异展示 |
| **E2E 测试** | Playwright | 1.54.2 | 端到端测试框架 |

## 📁 项目结构

```
backend/
├── cmd/                          # 命令行程序入口
│   ├── tusd/                    # TUS 上传服务
│   └── main/                    # 主程序入口
│
├── serve/                       # REST API 服务层
│   ├── user/                    # 用户管理服务
│   │   ├── user.go             # 用户查询和管理
│   │   ├── auth.go             # 认证相关接口
│   │   └── profile.go           # 用户资料接口
│   │
│   ├── exam/                    # 考试管理服务
│   │   ├── exam.go             # 考试 CRUD 操作
│   │   ├── exam_session.go     # 考试场次管理
│   │   └── invigilation.go     # 监考信息接口
│   │
│   ├── question/               # 题库管理服务
│   │   ├── question.go         # 题目 CRUD 操作
│   │   ├── question_bank.go    # 题库管理
│   │   └─�� import.go           # 批量导入处理
│   │
│   ├── paper/                  # 试卷管理服务
│   │   ├── paper.go            # 试卷创建和编辑
│   │   └── assembly.go         # 试卷组卷逻辑
│   │
│   ├── score/                  # 评分服务
│   │   ├── score.go            # 成绩查询
│   │   └── analysis.go         # 成绩分析
│   │
│   └── serve_template/         # API 开发模板
│       └── template.go         # 模板文件
│
├── sckserve/                    # Secure Socket 服务层
│   └── ...                     # 加密通信处理
│
├── graph/                      # GraphQL 层
│   ├── schema.graphql          # GraphQL Schema 定义
│   ├── resolver.go             # GraphQL 解析器
│   └── schema/                 # Schema 生成文件
│
├── service/                    # 业务逻辑层（可选）
│   ├── exam_service/           # 考试业务逻辑
│   │   ├── exam.go
│   │   └── invigilation.go
│   │
│   └── question_service/       # 题库业务逻辑
│       ├── question.go
│       └── import.go
│
├── model/                      # 数据模型层
│   ├── user.go                # 用户模型
│   ├── exam.go                # 考试模型
│   ├── question.go            # 题目模型
│   ├── paper.go               # 试卷模型
│   └── ...
│
├── dao/                        # 数据访问层（可选）
│   ├── user_dao.go
│   ├── exam_dao.go
│   ├── question_dao.go
│   └── ...
│
├── mux/                        # 路由多路复用器
│   └── mux.go
│
├── cmn/                        # 通用组件和工具
│   ├── logger.go               # 日志工具
│   ├── db.go                   # 数据库连接管理
│   ├── redis.go                # Redis 连接管理
│   ├── config.go               # 配置管理
│   ├── error.go                # 错误处理
│   ├── auth.go                 # 认证工具
│   └── utils.go                # 通用工具函数
│
├── tools/                      # 工具脚本
│   ├── modelGen.sh            # 模型生成脚本
│   └── proto-gen.sh           # Protocol Buffer 生成
│
├── w2wproto/                   # Protocol Buffer 定义
│   └── *.proto
│
├── qproto/                     # 查询 Protocol Buffer
│   └── query.proto
│
├── tmpl/                       # 文本模板
│   └── email/                 # 邮件模板
│
├── tusd/                       # TUS 文件上传服务
│   ├── cmd/
│   │   └── tusd/
│   │       └── main.go        # TUS 服务器入口
│   └── internal/
│       └── e2e/
│           └── e2e_test.go    # E2E 测试
│
├── bpmn/                       # BPMN 工作流（可选）
│   └── ...
│
├── qrcode/                     # 二维码生成
│   └── qrcode.go
│
├── excelize/                   # Excel 操作库
│   └── ...
│
├── scalar/                     # GraphQL 标量类型
│   └── scalar.go
│
├── null/                       # NULL 类型处理
│   └── null.go
│
├── lockfile/                   # 文件锁机制
│   └── lockfile.go
│
├── main.go                     # 主入口文件
├── go.mod                      # Go 模块定义
├── go.sum                      # Go 依赖版本锁定
├── gqlgen.yml                  # GraphQL 生成配置
├── default.conf                # 默认配置
│
├── .config_darwin_sample.json  # macOS 配置示例
├── .config_linux_sample.json   # Linux 配置示例
├── .config_windows_sample.json # Windows 配置示例
│
├── deploy.sh                   # 测试环境部署脚本
├── deploy_p.sh                 # 生产环境部署脚本
├── run.sh                      # 本地运行脚本
├── Makefile                    # Make 构建
│
└── README.md                   # 项目文档
```

## 🚀 快速开始

### 环境要求

- **Go** 1.23.8 或更高版本（推荐 1.24.2+）
- **PostgreSQL** 12.0 或更高版本
- **Redis** 6.0 或更高版本
- **Docker** 和 **Docker Compose**（可选）

### 安装步骤

#### 1. 克隆项目

```bash
git clone https://github.com/b1each232425/cst_backend.git
cd cst_backend/backend
```

#### 2. 创建配置文件

根据你的操作系统创建配置文件：

```bash
# macOS
cp .config_darwin_sample.json .config_darwin.json

# Linux
cp .config_linux_sample.json .config_linux.json

# Windows
cp .config_windows_sample.json .config_windows.json
```

#### 3. 编辑配置文件

编辑相应的配置文件，配置数据库和 Redis 连接信息：

```json
{
  "dbms": {
    "postgresql": {
      "addr": "localhost",
      "port": 5432,
      "db": "cst_db",
      "user": "postgres",
      "pwd": "your_password",
      "enable": true
    },
    "redis": {
      "addr": "localhost",
      "port": 6379,
      "cert": "your_redis_password"
    }
  },
  "server": {
    "port": 8080,
    "host": "0.0.0.0"
  }
}
```

#### 4. 下载依赖

```bash
go mod download
go mod tidy
```

#### 5. 初始化数据库

```bash
# 创建数据库（如果还没有）
# 根据项目的 SQL 初始化脚本执行

# 或使用 migrate 工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### 6. 编译和运行

```bash
# 构建
go build -o server main.go

# 运行
./server

# 或直接使用 make
make
```

服务器将在 `http://localhost:8080` 启动。

### Docker 运行

```bash
# 构建 Docker 镜像
docker build -t cst-backend .

# 运行容器
docker run -p 8080:8080 \
  -e DB_HOST=localhost \
  -e DB_USER=postgres \
  -e DB_PASSWORD=password \
  -e REDIS_ADDR=localhost:6379 \
  cst-backend
```

## 📚 API 文档

### REST API 基础 URL

```
http://localhost:8080/api
```

### GraphQL 端点

```
POST http://localhost:8080/graphql
```

### WebSocket 实时连接

```
ws://localhost:8080/ws
```

### 主要 API 端点

#### 认证相关

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 用户注册 |
| POST | `/api/auth/login` | 用户登录 |
| POST | `/api/auth/logout` | 用户登出 |
| POST | `/api/auth/refresh` | 刷新令牌 |
| GET | `/api/auth/profile` | 获取当前用户信息 |

#### 用户管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/user` | 获取用户列表 |
| GET | `/api/user/:id` | 获取用户详情 |
| PUT | `/api/user/:id` | 更新用户信息 |
| DELETE | `/api/user/:id` | 删除用户 |

#### 题库管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/question` | 获取题目列表 |
| POST | `/api/question` | 创建题目 |
| GET | `/api/question/:id` | 获取题目详情 |
| PUT | `/api/question/:id` | 更新题目 |
| DELETE | `/api/question/:id` | 删除题目 |
| POST | `/api/question/import` | 批量导入题目 |

#### 试卷管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/paper` | 获取试卷列表 |
| POST | `/api/paper` | 创建试卷 |
| GET | `/api/paper/:id` | 获取试卷详情 |
| PUT | `/api/paper/:id` | 更新试卷 |
| DELETE | `/api/paper/:id` | 删除试卷 |

#### 考试管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/exam` | 获取考试列表 |
| POST | `/api/exam` | 创建考试 |
| GET | `/api/exam/:id` | 获取考试详情 |
| PUT | `/api/exam/:id` | 更新考试 |
| DELETE | `/api/exam/:id` | 删除考试 |

#### 监考管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/invigilation/list` | 获取监考列表 |
| GET | `/api/invigilation/:id` | 获取监考详情 |
| PATCH | `/api/invigilation` | 更新监考信息 |

#### 文件处理

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | `/api/file/upload` | 文件上传（TUS 协议） |
| GET | `/api/file/:id` | 获取文件 |
| POST | `/api/file/export` | 导出文件 |

## 🔑 关键特性详解

### 1. 认证与授权系统

```go
// JWT 令牌生成
token, err := cmn.GenerateToken(userID, role)

// 权限检查
if !hasPermission(userID, resource) {
    return errors.New("unauthorized")
}
```

**特点：**
- JWT 无状态认证
- 基于角色的访问控制（RBAC）
- 会话管理和令牌刷新
- 支持多个认证渠道

### 2. 高性能数据库操作

```go
// PostgreSQL 连接（高性能）
pgxConn := cmn.GetPgxConn()
rows, err := pgxConn.Query(ctx, query)

// 标准 SQL 连接
sqlConn := cmn.GetDbConn()
rows := sqlConn.Query(query)
```

**特点：**
- 支持 PostgreSQL 和 MySQL
- 连接池管理
- 事务支持
- 查询优化

### 3. Redis 缓存层

```go
// 获取 Redis 连接
redis := cmn.GetRedisConn()

// 缓存操作
redis.Set(ctx, key, value, expiration)
value, err := redis.Get(ctx, key).Result()

// 分布式锁
lock := redis.SetNX(ctx, lockKey, "", 30*time.Second)
```

**特点：**
- 自动过期管理
- 分布式锁
- Pub/Sub 发布订阅
- 高可用集群支持

### 4. 异步任务处理

```go
// 创建异步任务
task := &asynq.Task{
    Type: "send_email",
    Payload: payload,
}

// 入队
inspector := asynqmon.New(opts)
taskInfo, err := client.Enqueue(task)
```

**特点：**
- 基于 Redis 的任务队列
- 支持延迟任务
- 失败重试机制
- 任务优先级

### 5. GraphQL API

```go
# 查询示例
query {
  exam(id: "123") {
    id
    name
    status
    questions {
      id
      content
    }
  }
}

# 变更示例
mutation {
  createExam(input: {
    name: "数学考试",
    paperId: "456"
  }) {
    id
    name
  }
}
```

**特点：**
- 强类型 Schema
- 自动代码生成
- 灵活的查询
- 实时订阅支持

### 6. 文件上传（TUS 协议）

```go
// 断点续传支持
POST /files/
Content-Type: application/offset+octet-stream
Upload-Offset: 1000
Upload-Length: 5000
```

**特点：**
- 断点续传
- 并行上传
- 大文件支持
- 云存储集成（GCS、Azure、S3）

### 7. Excel 导入导出

```go
// 导入 Excel 文件
f, err := excelize.OpenFile("questions.xlsx")
rows, err := f.GetRows("Sheet1")

// 导出 Excel 文件
f := excelize.NewFile()
f.SetCellValue("Sheet1", "A1", "题目内容")
f.SaveAs("output.xlsx")
```

**特点：**
- 批量导入题目
- 成绩导出
- 格式校验
- 错误报告

### 8. 实时监考（WebSocket）

```go
// WebSocket 连接示例
ws://localhost:8080/ws/exam/{examId}

// 实时消息格式
{
  "type": "answer_submitted",
  "studentId": "123",
  "questionId": "456",
  "timestamp": "2026-03-26T10:30:00Z"
}
```

**特点：**
- 实时双向通信
- 考生答题进度实时同步
- 监考员异常处理
- 消息广播

## 🧪 测试

### 单元测试

```bash
# 运行所有单元测试
go test ./...

# 运行特定包的测试
go test ./serve/exam -v

# 生成覆盖率报告
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### E2E 测试

```bash
# 运行 E2E 测试
go test -tags=e2e ./...

# 使用 Playwright
npm install @playwright/test
playwright test
```

### 性能测试

```bash
# 基准测试
go test -bench=. -benchmem ./...

# 性能分析
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
go tool pprof cpu.prof
```

## 📊 监控与日志

### 日志输出

系统使用 Zap 和 zerolog 进行结构化日志记录：

```go
// Zap 日志
z := cmn.GetLogger()
z.Info("用户登录成功", zap.String("userID", userID))
z.Error("数据库连接失败", zap.Error(err))
```

### Prometheus 指标

```
# HTTP 请求延迟
http_request_duration_seconds{method="GET", path="/api/exam"}

# 活跃连接数
active_connections_total

# 错误率
error_rate_percentage
```

## 🔒 安全性

### 实施的安全措施

- ✅ **JWT 令牌** - 无状态身份认证
- ✅ **CORS 配置** - 跨域资源共享
- ✅ **参数验证** - 输入数据校验
- ✅ **SQL 防注入** - 参数化查询
- ✅ **加密存储** - 敏感信息加密
- ✅ **HTTPS/TLS** - 传输层加密
- ✅ **速率限制** - API 调用频率限制
- ✅ **审计日志** - 操作日志记录
- ✅ **权限检查** - 基于角色的访问控制

## 📦 部署

### 本地部署

```bash
# 克隆和编译
git clone <repo>
cd backend
go build -o server main.go

# 运行
./server
```

### Docker 部署

```bash
# 构建镜像
docker build -t cst-backend:latest .

# 运行容器
docker run -d \
  --name cst-backend \
  -p 8080:8080 \
  -e CONFIG_FILE=/etc/cst/.config.json \
  -v $(pwd)/.config.json:/etc/cst/.config.json \
  cst-backend:latest
```

### Kubernetes 部署

```bash
# 应用 Kubernetes 配置
kubectl apply -f k8s/

# 检查部署状态
kubectl get pods
kubectl logs <pod-name>
```

### 部署脚本

```bash
# 测试环境部署
./deploy.sh

# 生产环境部署
./deploy_p.sh
```

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发工作流

1. **Fork** 本仓库
2. **创建** 特性分支 (`git checkout -b feature/amazing-feature`)
3. **提交** 更改 (`git commit -m 'Add amazing feature'`)
4. **推送** 到分支 (`git push origin feature/amazing-feature`)
5. **打开** Pull Request

### 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go) 编码规范
- 使用 `gofmt` 格式化代码
- 为函数编写 Godoc 注释
- 编写单元测试和文档

### 提交信息规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 类型：**
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档
- `style`: 代码风格
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试


## 👨‍💻 作者

**b1each232425**

## 📞 联系方式

- GitHub: [@b1each232425](https://github.com/b1each232425)
- Issues: [项目问题](https://github.com/b1each232425/cst_backend/issues)


**创建时间**: 2026 年 3 月 12 日  
**最后更新**: 2026 年 3 月 26 日  
**版本**: v0.0.1  
**Go 版本**: 1.23.8+
