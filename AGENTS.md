# LLMIO - AI Agents 协作指南

> 本文档为 AI 编程助手提供项目开发规范和协作指引

## 项目概述

LLMIO 是一个基于 Go 的 LLM 代理服务，提供统一的 API 接口来访问多个大语言模型提供商，支持智能负载均衡和现代化的 Web 管理界面。

### 核心技术栈
- **后端**: Go 1.25.0+ / Gin / GORM / SQLite
- **前端**: React 19 / TypeScript / Vite / Tailwind CSS 4
- **架构**: 分层架构 (Handler → Service → Provider)

## AI 协作原则

### 1. 代码变更规范
- **优先编辑现有文件**，避免创建新文件（除非明确需要）
- **遵循 KISS 原则**：保持代码简单直观
- **践行 DRY 原则**：识别并消除重复代码
- **应用 YAGNI 原则**：只实现当前明确需要的功能
- **符合 SOLID 原则**：确保代码的可维护性和可扩展性

### 2. 工作流程要求
- **先读后写**：修改前必须先阅读理解现有代码
- **基于事实**：充分使用工具收集信息，不要猜测
- **持续解决**：工作直到问题完全解决，不留半成品
- **充分规划**：每次操作前深思熟虑并反思

### 3. Git 操作约束
> **重要**: 未经用户明确要求，禁止执行以下 Git 操作：
- ❌ `git commit` - 不要自动提交代码
- ❌ `git push` - 不要推送到远程仓库
- ❌ `git branch` - 不要创建或切换分支
- ❌ `git merge` - 不要合并分支
- ❌ 其他可能影响版本控制的操作

用户会在需要时明确指示执行这些操作。

---

## 后端开发规范

### Go 编码标准

#### 代码格式
- 使用 `gofmt` 格式化所有代码
- 行长度限制：120 字符
- 使用制表符缩进

#### 命名约定
- **包名**: 全小写，简短语义化 (`handlers`, `models`, `providers`)
- **结构体**: 驼峰命名，大写开头 (`ChatRequest`, `ProviderConfig`)
- **接口**: 以 `-er` 结尾 (`Provider`, `Balancer`)
- **函数**: 动词+名词 (`getProviderByID`, `validateAPIKey`)
- **变量**: 驼峰命名，避免缩写 (`requestData`, `providerList`)
- **常量**: 驼峰或全大写 (`MaxRetries`, `DEFAULT_TIMEOUT`)

#### 项目结构
```
├── handler/          # HTTP 处理器层
├── service/          # 业务逻辑层
├── providers/        # LLM 提供商实现
├── models/           # 数据模型
├── middleware/       # 中间件 (认证、日志等)
├── balancer/         # 负载均衡算法
└── common/           # 公共工具
```

#### 错误处理
- 使用标准 `errors` 包
- 自定义错误类型：`ErrProviderUnavailable`, `ErrInvalidRequest`
- 统一错误响应格式 (见 API 规范)

#### 日志规范
- 使用结构化日志 (JSON 格式)
- 日志级别：DEBUG, INFO, WARN, ERROR
- 关键操作包含 `request_id` 追踪
- 格式：`时间戳 | 级别 | 组件 | 消息 | 上下文`

### 数据库规范

#### GORM 模型标准
```go
type Model struct {
    ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    Name        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
    Description string    `gorm:"type:text" json:"description"`
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`  // 软删除
}
```

#### 查询原则
- **明确排除软删除记录**：查询时添加 `WHERE deleted_at IS NULL`
- 合理使用索引提升性能
- 避免 N+1 查询问题

---

## 前端开发规范

### TypeScript/React 标准

#### 代码格式
- ESLint + Prettier
- 行长度：100 字符
- 2 空格缩进，必须使用分号

#### 命名约定
- **组件**: PascalCase (`ChatInterface`, `ModelCard`)
- **函数**: camelCase (`fetchData`, `handleSubmit`)
- **类型**: PascalCase + Type (`UserType`, `ApiResponseType`)
- **接口**: PascalCase + Props (`ChatProps`, `FormProps`)
- **常量**: SCREAMING_SNAKE_CASE

#### 组件规范
- 优先使用函数组件
- TypeScript 严格模式
- Props 类型必须定义
- 使用 React.FC 类型

#### 项目结构
```
src/
├── components/       # 可复用组件
│   ├── ui/          # 基础 UI (Button, Input)
│   ├── charts/      # 图表组件
│   └── forms/       # 表单组件
├── routes/          # 页面路由
├── lib/             # 工具函数
├── hooks/           # 自定义 Hooks
└── types/           # 类型定义
```

#### 状态管理
- React Context → 全局状态
- useState/useReducer → 局部状态
- SWR/React Query → 数据获取

#### 样式规范
- Tailwind CSS 优先
- CSS Modules 用于复杂场景
- 响应式优先设计 (移动优先)

---

## API 设计规范

### RESTful 标准

#### 统一响应格式
```json
{
  "success": true,
  "data": {},
  "message": "操作成功"
}
```

#### 错误响应格式
```json
{
  "success": false,
  "error": "VALIDATION_ERROR",
  "message": "请求参数验证失败"
}
```

#### HTTP 状态码
- `200` - 成功 (GET/PUT)
- `201` - 创建成功 (POST)
- `204` - 成功无内容 (DELETE)
- `400` - 请求错误
- `401` - 未授权
- `404` - 资源不存在
- `422` - 验证失败
- `500` - 服务器错误

#### 标准端点设计
```
GET    /api/resources          # 列表
POST   /api/resources          # 创建
GET    /api/resources/:id      # 详情
PUT    /api/resources/:id      # 更新
DELETE /api/resources/:id      # 删除
```

---

## 安全规范

### 认证授权
- Bearer Token 认证
- 环境变量存储密钥
- 生产环境必须 HTTPS

### 输入验证
- 后端验证所有输入
- GORM 防 SQL 注入
- 输出转义防 XSS
- API 限流防滥用

### 安全配置
- 避免明文存储敏感信息
- 日志不记录密钥/密码
- 生产环境隐藏详细错误

---

## Git 工作流规范

### 分支策略
- `main` - 生产分支
- `develop` - 开发分支
- `feature/*` - 功能分支
- `hotfix/*` - 热修复分支

### 提交信息格式
```
[类型]: [简要描述]

[详细描述，每行一个改进点]

[相关 issue 编号]

类型: feat, fix, docs, style, refactor, test, chore
```

### 提交信息示例
```
feat: 增强日志系统与请求追踪功能

- 新增请求日志中间件，为所有请求生成唯一 UUID 追踪标识
- 完善认证中间件日志，记录认证成功/失败详情
- 优化聊天服务日志，添加完整的负载均衡追踪
- 修复数据库查询，明确排除软删除记录

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 代码审查要求
- Pull Request 必须评审
- 所有测试必须通过
- 代码覆盖率 > 80%

---

## 开发环境配置

### 前置要求
- Go 1.25.0+
- Node.js 20+
- pnpm (推荐) / npm
- SQLite3

### 快速启动
```bash
# 使用一键启动脚本
chmod +x start.sh
./start.sh
```

### 手动启动
```bash
# 1. 安装依赖
go mod tidy
cd webui && pnpm install

# 2. 初始化数据库
mkdir db
export TOKEN=12345
go run main.go

# 3. 构建前端
cd webui && pnpm build

# 4. 开发模式
# 后端: go run main.go
# 前端: cd webui && pnpm dev
```

### 环境变量
- `TOKEN` - API 访问令牌 (默认 `12345`)
- `GIN_MODE` - 运行模式 (`debug` / `release`)
- `TZ` - 时区 (推荐 `Asia/Shanghai`)

---

## 测试规范

### 单元测试
```bash
# 后端测试
go test ./...

# 前端测试
cd webui && pnpm test
```

### 集成测试
- API 测试：Postman 集合
- E2E 测试：Playwright

### 代码质量
- 静态检查：`golangci-lint run`
- 安全扫描：`govulncheck`
- 定期依赖更新

---

## 性能优化指南

### 后端优化
- 数据库索引优化
- API 响应时间 < 500ms
- 合理使用缓存
- 及时释放资源

### 前端优化
- 代码分割与懒加载
- 虚拟滚动处理长列表
- 防抖/节流优化交互
- 图片压缩与懒加载

---

## 监控与日志

### 日志规范
- 结构化 JSON 格式
- 包含 `request_id` 追踪
- 分级记录 (DEBUG/INFO/WARN/ERROR)
- 敏感信息脱敏

### 监控指标
- 响应时间
- 错误率
- 吞吐量
- 资源使用率

---

## 故障处理

### 错误处理原则
1. 用户友好的错误提示
2. 详细的日志记录
3. 优雅降级机制
4. 快速恢复策略

### 常见场景
- 数据库连接失败 → 重试机制 + 连接池
- 外部 API 超时 → 超时配置 + 重试策略
- 资源耗尽 → 限流 + 队列管理

---

## 文档更新记录

- **版本**: 2.0.0
- **日期**: 2025-10-06
- **说明**: 从 Claude 专用格式迁移为通用 AI Agents 协作指南

---

**注意**: 此文档适用于所有 AI 编程助手，请严格遵循规范以确保代码质量和项目一致性。
