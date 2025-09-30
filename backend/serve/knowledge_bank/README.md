# 知识点库管理模块

## 概述

知识点库管理模块提供了完整的知识点库增删改查功能，支持知识点树的存储和管理。

## 功能特性

- **知识点库管理**: 支持创建、查询、更新、删除知识点库
- **知识点树存储**: 支持复杂的知识点树结构存储
- **权限控制**: 基于用户角色的访问控制
- **域隔离**: 支持多域环境下的数据隔离
- **标签系统**: 支持知识点库标签管理

## API 接口

### 1. 查询知识点库 (GET)

**接口**: `/api/knowledge-banks`

**参数**:

- `keyword`: 关键词搜索（可选）
- `page`: 页码（默认 1）
- `pageSize`: 每页大小（默认 99）
- `bankID`: 指定知识点库 ID（可选）

**示例**:

```
GET /api/knowledge-banks?keyword=数学&page=1&pageSize=10
```

### 2. 创建知识点库 (POST)

**接口**: `/api/knowledge-banks`

**请求体**:

```json
{
  "name": "数学知识点库",
  "tags": ["数学", "基础"],
  "knowledges": [
    {
      "node_type": "考核范围",
      "code": "001",
      "name": "基础概念",
      "weight": 1.0,
      "importance": "X",
      "children": []
    }
  ]
}
```

### 3. 更新知识点库 (PUT)

**接口**: `/api/knowledge-banks`

**请求体**:

```json
{
  "id": 1,
  "name": "更新后的知识点库",
  "tags": ["更新", "知识点"]
}
```

### 4. 删除知识点库 (DELETE)

**接口**: `/api/knowledge-banks`

**请求体**:

```json
[1, 2, 3]
```

## 数据模型

### TKnowledgeBank (知识点库表)

| 字段        | 类型    | 说明         |
| ----------- | ------- | ------------ |
| id          | SERIAL  | 知识点库 ID  |
| domain_id   | INT8    | 所属域 ID    |
| name        | VARCHAR | 知识点库名称 |
| tags        | JSONB   | 知识点库标签 |
| creator     | INT8    | 创建者       |
| updated_by  | INT8    | 更新者       |
| create_time | INT8    | 创建时间     |
| update_time | INT8    | 更新时间     |
| knowledges  | JSONB   | 知识点树     |
| addi        | JSONB   | 附加信息     |
| status      | VARCHAR | 状态         |

### 知识点树结构

```json
{
  "knowledges": [
    {
      "node_type": "考核范围|知识点",
      "code": "节点代码",
      "name": "节点名称",
      "weight": 1.0,
      "importance": "X|Y|Z",
      "children": [
        {
          "node_type": "知识点",
          "code": "001",
          "name": "子知识点",
          "weight": 0.8,
          "importance": "Y",
          "children": []
        }
      ]
    }
  ]
}
```

## 权限控制

- **查询权限**: 用户只能查看自己域内的知识点库，普通用户只能查看自己创建的知识点库
- **创建权限**: 需要知识点库创建权限
- **更新权限**: 需要知识点库更新权限
- **删除权限**: 只有知识点库创建者可以删除

## 状态码

- `00`: 已创建
- `02`: 已删除

## 错误处理

系统提供完整的错误处理机制，包括：

- 参数验证错误
- 权限检查错误
- 数据库操作错误
- 业务逻辑错误

## 使用示例

### 创建知识点库

```bash
curl -X POST http://localhost:8080/api/knowledge-banks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "计算机基础知识点库",
    "tags": ["计算机", "基础"],
    "knowledges": [
      {
        "node_type": "考核范围",
        "code": "001",
        "name": "计算机基础",
        "weight": 1.0,
        "importance": "X",
        "children": [
          {
            "node_type": "知识点",
            "code": "001001",
            "name": "计算机组成",
            "weight": 0.8,
            "importance": "Y",
            "children": []
          }
        ]
      }
    ]
  }'
```

### 查询知识点库

```bash
curl -X GET "http://localhost:8080/api/knowledge-banks?page=1&pageSize=10&keyword=计算机"
```

### 更新知识点库

```bash
curl -X PUT http://localhost:8080/api/knowledge-banks \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "name": "更新后的计算机基础知识点库",
    "tags": ["计算机", "基础", "更新"]
  }'
```

### 删除知识点库

```bash
curl -X DELETE http://localhost:8080/api/knowledge-banks \
  -H "Content-Type: application/json" \
  -d '[1, 2, 3]'
```

## 注意事项

1. 知识点库的创建者拥有删除权限
2. 删除操作会直接删除知识点库，请谨慎操作
3. 知识点树结构支持无限层级嵌套
4. 标签支持 JSON 数组格式存储
5. 所有时间字段使用毫秒时间戳
