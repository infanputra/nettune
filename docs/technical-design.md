# Nettune 技术实现方案

> 本文档基于 [PRD](./prd.md) 提供详细的技术架构设计、模块划分和实现方案。

---

## 1. 整体架构设计

### 1.0 核心设计理念

**LLM 驱动的智能优化：**

Nettune 采用"工具平台 + LLM 决策"的架构理念：
- **Nettune 提供**：测速工具、配置管理、安全回滚等基础能力
- **LLM 负责**：分析数据、推理决策、解释建议、迭代优化

这种设计使得优化策略可以：
- 实时适应网络优化领域的最新知识
- 根据用户特定场景灵活调整
- 用自然语言与用户交互，解释每个决策
- 无需代码更新即可应用新的优化策略

### 1.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         User (Chat GUI)                         │
└──────────────────────────┬──────────────────────────────────────┘
                           │ MCP stdio
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    nettune client (本机)                         │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ MCP stdio Server (9 tools)                                │ │
│  │  - test_rtt, test_throughput, test_latency_under_load    │ │
│  │  - snapshot_server, list_profiles, show_profile          │ │
│  │  - apply_profile, rollback, status                       │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ HTTP Client (with Bearer auth)                            │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP + Bearer token
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                   nettune server (远端服务器)                     │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ HTTP API Server (Gin framework)                           │ │
│  │  - Bearer auth middleware                                 │ │
│  │  - Request validation & rate limiting                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Probe Service                                             │ │
│  │  - /probe/echo, /probe/download, /probe/upload           │ │
│  │  - /probe/info                                            │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ System Service                                            │ │
│  │  - Profile Manager (load, validate)                       │ │
│  │  - Snapshot Manager (create, restore)                     │ │
│  │  - Apply Engine (plan, execute, verify, rollback)        │ │
│  │  - History & Audit Logger                                 │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ System Adapter (Linux)                                    │ │
│  │  - Sysctl Manager                                         │ │
│  │  - Qdisc Manager (netlink)                                │ │
│  │  - Systemd Service Manager                                │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ State Storage                                             │ │
│  │  - profiles/, snapshots/, history/                        │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 NPM Wrapper 架构

```
┌────────────────────────────────────────────────────────────┐
│                    npx nettune-mcp                         │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ CLI Parser (parse --api-key, --server, etc.)        │ │
│  └──────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ Platform Detector (darwin/linux, amd64/arm64)       │ │
│  └──────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ Binary Manager                                       │ │
│  │  - Download from GitHub releases                     │ │
│  │  - Cache to ~/.cache/nettune/                        │ │
│  │  - Hash verification                                 │ │
│  │  - chmod +x                                          │ │
│  └──────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ Process Spawner                                      │ │
│  │  - spawn(binary, ["client", ...args])               │ │
│  │  - stdio: inherit                                    │ │
│  └──────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────┘
```

---

## 2. Go 项目架构

### 2.1 目录结构

```
nettune/
├── cmd/
│   └── nettune/
│       └── main.go                 # 入口，解析 client/server 子命令
├── internal/
│   ├── client/                     # client 模式实现
│   │   ├── mcp/
│   │   │   ├── server.go          # MCP stdio server
│   │   │   ├── tools.go           # 9 个 tools 定义
│   │   │   ├── schema.go          # JSON schema 定义
│   │   │   └── handler.go         # tool 调用处理器
│   │   ├── http/
│   │   │   └── client.go          # HTTP client (with bearer)
│   │   └── probe/
│   │       ├── rtt.go             # RTT 测试逻辑
│   │       ├── throughput.go     # 吞吐测试逻辑
│   │       └── latency_load.go   # 负载延迟测试
│   ├── server/                     # server 模式实现
│   │   ├── api/
│   │   │   ├── server.go          # Gin HTTP server
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go        # Bearer 认证
│   │   │   │   ├── logger.go      # 请求日志
│   │   │   │   └── ratelimit.go   # 速率限制
│   │   │   ├── handlers/
│   │   │   │   ├── probe.go       # /probe/* handlers
│   │   │   │   ├── profile.go     # /profiles/* handlers
│   │   │   │   └── system.go      # /sys/* handlers
│   │   │   └── response.go        # 统一响应格式
│   │   ├── service/
│   │   │   ├── probe.go           # Probe 服务
│   │   │   ├── profile.go         # Profile 管理
│   │   │   ├── snapshot.go        # Snapshot 管理
│   │   │   ├── apply.go           # Apply 引擎
│   │   │   └── history.go         # 历史记录
│   │   └── adapter/
│   │       ├── sysctl.go          # Sysctl 操作
│   │       ├── qdisc.go           # Qdisc (netlink)
│   │       ├── systemd.go         # Systemd 操作
│   │       └── system_info.go     # 系统信息收集
│   ├── shared/
│   │   ├── types/
│   │   │   ├── profile.go         # Profile 数据结构
│   │   │   ├── snapshot.go        # Snapshot 数据结构
│   │   │   ├── probe.go           # Probe 结果结构
│   │   │   └── errors.go          # 错误定义
│   │   ├── config/
│   │   │   └── config.go          # 配置管理
│   │   └── utils/
│   │       ├── file.go            # 文件操作工具
│   │       ├── lock.go            # 并发锁
│   │       └── hash.go            # 哈希计算
│   └── testutil/                   # 测试工具
│       ├── mock/
│       │   ├── server.go          # Mock HTTP server
│       │   └── system.go          # Mock system calls
│       └── fixtures/
│           ├── profiles.go        # 测试用 profiles
│           └── snapshots.go       # 测试用 snapshots
├── pkg/                            # 可导出的公共库
│   └── version/
│       └── version.go             # 版本信息
├── configs/                        # 默认配置
│   └── profiles/
│       ├── bbr-fq.default.json
│       └── bbr-fq.tuned-32mb.json
├── scripts/
│   ├── build.sh                   # 构建脚本
│   └── test.sh                    # 测试脚本
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 2.2 技术栈选型

| 组件      | 技术选型                                | 理由                              |
| --------- | --------------------------------------- | --------------------------------- |
| HTTP 框架 | Gin                                     | 高性能、轻量级、中间件生态完善    |
| CLI 解析  | cobra                                   | 标准库，支持子命令和参数验证      |
| 配置管理  | viper                                   | 支持多种配置源，与 cobra 集成良好 |
| Netlink   | vishvananda/netlink                     | 成熟的 Go netlink 库              |
| JSON 处理 | encoding/json + go-playground/validator | 标准库 + 验证器                   |
| 日志      | zap                                     | 高性能、结构化日志                |
| 测试框架  | testify                                 | 断言和 mock 工具                  |
| 并发锁    | sync.Mutex + 文件锁 (flock)             | 进程内和跨进程并发控制            |

### 2.3 核心模块设计

#### 2.3.1 MCP Server (Client 侧)

```go
// internal/client/mcp/server.go
package mcp

type Server struct {
    httpClient *http.Client
    serverURL  string
    apiKey     string
    logger     *zap.Logger
}

func NewServer(serverURL, apiKey string) *Server
func (s *Server) Start() error  // 启动 stdio MCP server
func (s *Server) Stop() error   // 优雅停止
```

**关键点：**

- 使用标准输入/输出进行 MCP 通信
- 所有日志输出到 stderr
- 实现 9 个 tools 的 schema 和 handler（不包含推荐功能）
- 推荐策略完全由 LLM 通过组合使用其他 tools 自主决定
- 错误处理和超时控制

#### 2.3.2 HTTP API Server (Server 侧)

```go
// internal/server/api/server.go
package api

type Server struct {
    router     *gin.Engine
    config     *config.ServerConfig
    probeService    *service.ProbeService
    profileService  *service.ProfileService
    snapshotService *service.SnapshotService
    applyService    *service.ApplyService
    logger     *zap.Logger
}

func NewServer(cfg *config.ServerConfig) *Server
func (s *Server) Start() error
func (s *Server) Stop(ctx context.Context) error
```

**Gin 中间件链：**

```go
router.Use(
    middleware.Logger(logger),           // 请求日志
    middleware.Recovery(),                // Panic 恢复
    middleware.BearerAuth(apiKey),       // Bearer 认证
    middleware.RateLimit(limiter),       // 速率限制
    middleware.RequestSizeLimit(maxSize), // Body 大小限制
)
```

#### 2.3.3 Profile Manager

```go
// internal/server/service/profile.go
package service

type ProfileService struct {
    profilesDir string
    cache       map[string]*types.Profile
    mu          sync.RWMutex
    logger      *zap.Logger
}

func NewProfileService(dir string) (*ProfileService, error)
func (s *ProfileService) List() ([]*types.ProfileMeta, error)
func (s *ProfileService) Get(id string) (*types.Profile, error)
func (s *ProfileService) Validate(p *types.Profile) error
func (s *ProfileService) Reload() error  // 热重载 profiles
```

**Profile 验证规则：**

- ID 格式检查（字母数字和连字符）
- Sysctl 键值合法性检查
- Qdisc 类型合法性检查
- 危险操作检测（如修改内核核心参数）

#### 2.3.4 Snapshot Manager

```go
// internal/server/service/snapshot.go
package service

type SnapshotService struct {
    snapshotsDir string
    adapter      *adapter.SystemAdapter
    mu           sync.Mutex
    logger       *zap.Logger
}

func (s *SnapshotService) Create() (*types.Snapshot, error)
func (s *SnapshotService) List() ([]*types.SnapshotMeta, error)
func (s *SnapshotService) Get(id string) (*types.Snapshot, error)
func (s *SnapshotService) Delete(id string) error
```

**Snapshot 内容：**

- 当前所有相关 sysctl 值
- 当前所有网卡的 qdisc 配置
- 相关配置文件的内容和哈希
- Systemd unit 状态
- 创建时间和元数据

#### 2.3.5 Apply Engine

```go
// internal/server/service/apply.go
package service

type ApplyService struct {
    profileService  *ProfileService
    snapshotService *SnapshotService
    adapter         *adapter.SystemAdapter
    historyService  *HistoryService
    mu              sync.Mutex  // 全局 apply 锁
    logger          *zap.Logger
}

func (s *ApplyService) Apply(req *types.ApplyRequest) (*types.ApplyResult, error)
func (s *ApplyService) Rollback(snapshotID string) error
```

**Apply 流程：**

1. 加锁（确保同一时间只有一个 apply）
2. 验证 profile（调用 ProfileService.Validate）
3. 生成执行计划（Plan）
4. dry_run 模式：返回 plan
5. commit 模式：
   - 创建 snapshot
   - 执行配置变更（sysctl → qdisc → systemd）
   - 验证生效（读回 sysctl、检查 qdisc、检查 systemd）
   - 失败自动 rollback
   - 记录历史
6. 解锁

#### 2.3.6 System Adapter

```go
// internal/server/adapter/adapter.go
package adapter

type SystemAdapter struct {
    sysctlMgr  *SysctlManager
    qdiscMgr   *QdiscManager
    systemdMgr *SystemdManager
    infoMgr    *SystemInfoManager
    logger     *zap.Logger
}

// internal/server/adapter/sysctl.go
type SysctlManager struct{}
func (m *SysctlManager) Get(key string) (string, error)
func (m *SysctlManager) Set(key, value string) error
func (m *SysctlManager) WriteToFile(path string, kvs map[string]string) error

// internal/server/adapter/qdisc.go
type QdiscManager struct{}
func (m *QdiscManager) Get(iface string) (*types.QdiscInfo, error)
func (m *QdiscManager) Set(iface, qdiscType string, params map[string]interface{}) error
func (m *QdiscManager) GetDefaultRouteInterface() (string, error)

// internal/server/adapter/systemd.go
type SystemdManager struct{}
func (m *SystemdManager) IsActive(unit string) (bool, error)
func (m *SystemdManager) Enable(unit string) error
func (m *SystemdManager) Start(unit string) error
func (m *SystemdManager) Stop(unit string) error
func (m *SystemdManager) CreateUnit(name, content string) error
```

**实现策略：**

- Sysctl：优先使用 `/proc/sys/` 文件操作，降级到 `sysctl` 命令
- Qdisc：使用 `vishvananda/netlink` 库
- Systemd：调用 `systemctl` 命令（检查依赖可用性）

---

## 3. JS/TypeScript Wrapper 设计

### 3.1 目录结构

```
js/
├── src/
│   ├── index.ts               # 入口
│   ├── cli.ts                 # CLI 参数解析
│   ├── platform.ts            # 平台检测
│   ├── binary-manager.ts      # 二进制下载和管理
│   ├── spawner.ts             # 进程启动
│   └── types.ts               # 类型定义
├── test/
│   ├── platform.test.ts
│   ├── binary-manager.test.ts
│   └── integration.test.ts
├── package.json
├── tsconfig.json
├── bun.lockb
└── README.md
```

### 3.2 技术栈

| 组件      | 技术                      | 理由                  |
| --------- | ------------------------- | --------------------- |
| 包管理器  | Bun                       | 高性能、兼容 npm 生态 |
| 运行时    | Node.js                   | MCP stdio 标准环境    |
| 语言      | TypeScript                | 类型安全              |
| CLI 解析  | commander                 | 轻量级 CLI 框架       |
| HTTP 下载 | node-fetch / native fetch | 标准化                |
| 哈希校验  | crypto (native)           | 内置库                |
| 测试框架  | vitest                    | 快速、现代化          |

### 3.3 核心模块

#### 3.3.1 Platform Detector

```typescript
// src/platform.ts
export interface PlatformInfo {
  os: 'darwin' | 'linux' | 'win32';
  arch: 'x64' | 'arm64';
}

export function detectPlatform(): PlatformInfo;
export function getBinaryName(platform: PlatformInfo): string;
// 例如：nettune-darwin-arm64, nettune-linux-x64
```

#### 3.3.2 Binary Manager

```typescript
// src/binary-manager.ts
export interface BinaryManagerConfig {
  cacheDir: string;          // 默认 ~/.cache/nettune
  githubRepo: string;        // 例如 "jtsang4/nettune"
  version?: string;          // 默认 "latest"
}

export class BinaryManager {
  constructor(config: BinaryManagerConfig);

  async ensureBinary(platform: PlatformInfo): Promise<string>;
  // 返回本地二进制路径

  private async download(url: string, dest: string): Promise<void>;
  private async verifyChecksum(file: string, expectedHash: string): Promise<boolean>;
  private async makeExecutable(file: string): Promise<void>;
}
```

**下载策略：**

1. 检查缓存目录是否存在对应版本的二进制
2. 验证缓存二进制的哈希（如果存在）
3. 如果不存在或哈希不匹配，从 GitHub Releases 下载
4. 下载到临时文件，验证哈希
5. 移动到缓存目录并 chmod +x
6. 返回路径

#### 3.3.3 Spawner

```typescript
// src/spawner.ts
export interface SpawnOptions {
  binaryPath: string;
  args: string[];
  env?: Record<string, string>;
}

export function spawnNettuneClient(options: SpawnOptions): Promise<void> {
  // 使用 child_process.spawn
  // stdio: 'inherit' 确保 stdin/stdout 透传
  // 所有日志写到 stderr
  // 处理进程退出和信号
}
```

---

## 4. 数据模型设计

### 4.1 Profile Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["id", "name", "risk_level"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^[a-z0-9-]+$"
    },
    "name": { "type": "string" },
    "description": { "type": "string" },
    "risk_level": {
      "type": "string",
      "enum": ["low", "medium", "high"]
    },
    "requires_reboot": {
      "type": "boolean",
      "default": false
    },
    "sysctl": {
      "type": "object",
      "patternProperties": {
        "^[a-z0-9._]+$": {
          "oneOf": [
            { "type": "string" },
            { "type": "number" }
          ]
        }
      }
    },
    "qdisc": {
      "type": "object",
      "properties": {
        "type": {
          "type": "string",
          "enum": ["fq", "fq_codel", "cake", "pfifo_fast"]
        },
        "interfaces": {
          "type": "string",
          "enum": ["default-route", "all"]
        },
        "params": {
          "type": "object"
        }
      }
    },
    "systemd": {
      "type": "object",
      "properties": {
        "ensure_qdisc_service": {
          "type": "boolean"
        }
      }
    }
  }
}
```

**Go 结构体：**

```go
// internal/shared/types/profile.go
type Profile struct {
    ID             string                 `json:"id" validate:"required,alphanum-hyphen"`
    Name           string                 `json:"name" validate:"required"`
    Description    string                 `json:"description,omitempty"`
    RiskLevel      string                 `json:"risk_level" validate:"required,oneof=low medium high"`
    RequiresReboot bool                   `json:"requires_reboot"`
    Sysctl         map[string]interface{} `json:"sysctl,omitempty"`
    Qdisc          *QdiscConfig           `json:"qdisc,omitempty"`
    Systemd        *SystemdConfig         `json:"systemd,omitempty"`
}

type QdiscConfig struct {
    Type       string                 `json:"type" validate:"oneof=fq fq_codel cake pfifo_fast"`
    Interfaces string                 `json:"interfaces" validate:"oneof=default-route all"`
    Params     map[string]interface{} `json:"params,omitempty"`
}

type SystemdConfig struct {
    EnsureQdiscService bool `json:"ensure_qdisc_service"`
}

type ProfileMeta struct {
    ID             string `json:"id"`
    Name           string `json:"name"`
    Description    string `json:"description"`
    RiskLevel      string `json:"risk_level"`
    RequiresReboot bool   `json:"requires_reboot"`
}
```

### 4.2 Snapshot Schema

```go
// internal/shared/types/snapshot.go
type Snapshot struct {
    ID          string                 `json:"id"`
    CreatedAt   time.Time              `json:"created_at"`
    State       *SystemState           `json:"state"`
    Backups     map[string]string      `json:"backups"` // 文件路径 -> 备份内容
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type SystemState struct {
    Sysctl      map[string]string      `json:"sysctl"`
    Qdisc       map[string]*QdiscInfo  `json:"qdisc"` // 网卡名 -> qdisc 信息
    SystemdUnits map[string]bool       `json:"systemd_units"` // unit 名 -> 是否 active
    FileHashes  map[string]string      `json:"file_hashes"`
}

type QdiscInfo struct {
    Type   string                 `json:"type"`
    Handle string                 `json:"handle"`
    Params map[string]interface{} `json:"params,omitempty"`
}

type SnapshotMeta struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    Size      int64     `json:"size"` // 快照大小（字节）
}
```

### 4.3 Probe Result Schema

```go
// internal/shared/types/probe.go
type RTTResult struct {
    Count      int                `json:"count"`
    Successful int                `json:"successful"`
    Failed     int                `json:"failed"`
    RTT        *LatencyStats      `json:"rtt"` // 单位：毫秒
    Jitter     float64            `json:"jitter"` // 单位：毫秒
    Errors     []string           `json:"errors,omitempty"`
}

type LatencyStats struct {
    Min    float64 `json:"min"`
    Max    float64 `json:"max"`
    Mean   float64 `json:"mean"`
    P50    float64 `json:"p50"`
    P90    float64 `json:"p90"`
    P99    float64 `json:"p99"`
}

type ThroughputResult struct {
    Direction      string        `json:"direction"` // "download" | "upload"
    Bytes          int64         `json:"bytes"`
    DurationMs     int64         `json:"duration_ms"`
    ThroughputMbps float64       `json:"throughput_mbps"`
    Parallel       int           `json:"parallel"`
    Errors         []string      `json:"errors,omitempty"`
}

type LatencyUnderLoadResult struct {
    Baseline       *LatencyStats `json:"baseline"` // 无负载时的延迟
    UnderLoad      *LatencyStats `json:"under_load"` // 负载时的延迟
    InflationP50   float64       `json:"inflation_p50"` // p50 膨胀倍数
    InflationP99   float64       `json:"inflation_p99"` // p99 膨胀倍数
    LoadDurationMs int64         `json:"load_duration_ms"`
    LoadMbps       float64       `json:"load_mbps"`
}
```

### 4.4 Apply Request/Result Schema

```go
// internal/shared/types/apply.go
type ApplyRequest struct {
    ProfileID            string `json:"profile_id" validate:"required"`
    Mode                 string `json:"mode" validate:"required,oneof=dry_run commit"`
    AutoRollbackSeconds  int    `json:"auto_rollback_seconds,omitempty"`
}

type ApplyResult struct {
    Mode         string              `json:"mode"`
    ProfileID    string              `json:"profile_id"`
    SnapshotID   string              `json:"snapshot_id,omitempty"`
    Plan         *ApplyPlan          `json:"plan"`
    Success      bool                `json:"success"`
    AppliedAt    time.Time           `json:"applied_at,omitempty"`
    Verification *VerificationResult `json:"verification,omitempty"`
    Errors       []string            `json:"errors,omitempty"`
}

type ApplyPlan struct {
    SysctlChanges  map[string]*Change    `json:"sysctl_changes"`
    QdiscChanges   map[string]*Change    `json:"qdisc_changes"`
    SystemdChanges map[string]*Change    `json:"systemd_changes"`
}

type Change struct {
    From interface{} `json:"from"`
    To   interface{} `json:"to"`
}

type VerificationResult struct {
    SysctlOK  bool     `json:"sysctl_ok"`
    QdiscOK   bool     `json:"qdisc_ok"`
    SystemdOK bool     `json:"systemd_ok"`
    Errors    []string `json:"errors,omitempty"`
}
```

---

## 5. API 设计详细说明

### 5.1 认证机制

**Bearer Token 认证：**

```go
// internal/server/api/middleware/auth.go
func BearerAuth(expectedKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(401, gin.H{"error": "invalid authorization header format"})
            c.Abort()
            return
        }

        if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expectedKey)) != 1 {
            c.JSON(401, gin.H{"error": "invalid api key"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 5.2 统一响应格式

```go
// internal/server/api/response.go
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{Success: true, Data: data})
}

func Error(c *gin.Context, statusCode int, code, message string) {
    c.JSON(statusCode, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    code,
            Message: message,
        },
    })
}
```

### 5.3 路由定义

```go
// internal/server/api/server.go
func (s *Server) setupRoutes() {
    // Probe endpoints (测速)
    probe := s.router.Group("/probe")
    {
        probe.GET("/echo", s.handlers.ProbeEcho)
        probe.GET("/download", s.handlers.ProbeDownload)
        probe.POST("/upload", s.handlers.ProbeUpload)
        probe.GET("/info", s.handlers.ProbeInfo)
    }

    // Profile endpoints (配置版本)
    profiles := s.router.Group("/profiles")
    {
        profiles.GET("", s.handlers.ListProfiles)
        profiles.GET("/:id", s.handlers.GetProfile)
    }

    // System endpoints (系统配置与回滚)
    sys := s.router.Group("/sys")
    {
        sys.POST("/snapshot", s.handlers.CreateSnapshot)
        sys.GET("/snapshot/:id", s.handlers.GetSnapshot)
        sys.POST("/apply", s.handlers.ApplyProfile)
        sys.POST("/rollback", s.handlers.Rollback)
        sys.GET("/status", s.handlers.GetStatus)
    }
}
```

### 5.4 关键 Handler 实现伪码

#### /probe/echo

```go
func (h *ProbeHandler) ProbeEcho(c *gin.Context) {
    // 返回服务器时间戳
    response.Success(c, gin.H{
        "ts": time.Now().UnixMilli(),
        "ok": true,
    })
}
```

#### /probe/download

```go
func (h *ProbeHandler) ProbeDownload(c *gin.Context) {
    bytesStr := c.Query("bytes")
    bytes, err := strconv.ParseInt(bytesStr, 10, 64)
    if err != nil || bytes <= 0 || bytes > MaxDownloadBytes {
        response.Error(c, 400, "INVALID_BYTES", "invalid bytes parameter")
        return
    }

    c.Header("Content-Type", "application/octet-stream")
    c.Header("Content-Length", fmt.Sprintf("%d", bytes))
    c.Header("Content-Encoding", "identity")

    // 流式输出随机数据
    buf := make([]byte, 64*1024) // 64KB buffer
    written := int64(0)
    for written < bytes {
        toWrite := min(int64(len(buf)), bytes-written)
        rand.Read(buf[:toWrite])
        if _, err := c.Writer.Write(buf[:toWrite]); err != nil {
            return
        }
        written += toWrite
    }
}
```

#### /sys/apply

```go
func (h *SystemHandler) ApplyProfile(c *gin.Context) {
    var req types.ApplyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, 400, "INVALID_REQUEST", err.Error())
        return
    }

    result, err := h.applyService.Apply(&req)
    if err != nil {
        response.Error(c, 500, "APPLY_FAILED", err.Error())
        return
    }

    response.Success(c, result)
}
```

---

## 6. MCP 工具详细设计

### 6.1 MCP Server 实现

```go
// internal/client/mcp/server.go
type MCPServer struct {
    httpClient *http.Client
    serverURL  string
    apiKey     string
    tools      []Tool
    logger     *zap.Logger
}

type Tool struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema interface{} `json:"inputSchema"`
}

func (s *MCPServer) Start() error {
    // 实现 MCP stdio 协议
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := scanner.Bytes()
        var msg MCPMessage
        if err := json.Unmarshal(line, &msg); err != nil {
            s.logger.Error("invalid mcp message", zap.Error(err))
            continue
        }

        response := s.handleMessage(&msg)
        responseBytes, _ := json.Marshal(response)
        fmt.Fprintln(os.Stdout, string(responseBytes))
    }
    return scanner.Err()
}
```

### 6.2 Tools 定义

#### Tool 1: nettune.test_rtt

```go
// internal/client/mcp/tools.go
var TestRTTTool = Tool{
    Name: "nettune.test_rtt",
    Description: "Measure RTT (Round-Trip Time) to the server",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "count": map[string]interface{}{
                "type":        "number",
                "description": "Number of echo requests",
                "default":     30,
            },
            "concurrency": map[string]interface{}{
                "type":        "number",
                "description": "Concurrent requests",
                "default":     1,
            },
            "keepalive": map[string]interface{}{
                "type":        "boolean",
                "description": "Use HTTP keepalive",
                "default":     true,
            },
        },
    },
}

// Handler
func (s *MCPServer) handleTestRTT(params map[string]interface{}) (*types.RTTResult, error) {
    count := getInt(params, "count", 30)
    concurrency := getInt(params, "concurrency", 1)
    keepalive := getBool(params, "keepalive", true)

    return s.probeClient.TestRTT(s.serverURL, count, concurrency, keepalive)
}
```

#### Tool 2: nettune.test_throughput

```json
{
  "name": "nettune.test_throughput",
  "description": "Measure throughput (upload/download bandwidth)",
  "inputSchema": {
    "type": "object",
    "required": ["direction"],
    "properties": {
      "direction": {
        "type": "string",
        "enum": ["download", "upload"],
        "description": "Test direction"
      },
      "bytes": {
        "type": "number",
        "description": "Number of bytes to transfer",
        "default": 104857600
      },
      "parallel": {
        "type": "number",
        "description": "Number of parallel connections",
        "default": 1
      }
    }
  }
}
```

#### Tool 3: nettune.test_latency_under_load

```json
{
  "name": "nettune.test_latency_under_load",
  "description": "Measure latency while under network load (detect bufferbloat)",
  "inputSchema": {
    "type": "object",
    "properties": {
      "duration": {
        "type": "number",
        "description": "Load duration in seconds",
        "default": 10
      },
      "load_parallel": {
        "type": "number",
        "description": "Parallel connections for load generation",
        "default": 4
      },
      "echo_interval": {
        "type": "number",
        "description": "Echo probe interval in milliseconds",
        "default": 100
      }
    }
  }
}
```

#### Tool 4-9: 其他工具定义

```go
// Tool 4: snapshot_server
var SnapshotServerTool = Tool{
    Name:        "nettune.snapshot_server",
    Description: "Create a snapshot of current server configuration",
    InputSchema: map[string]interface{}{"type": "object"},
}

// Tool 5: list_profiles
var ListProfilesTool = Tool{
    Name:        "nettune.list_profiles",
    Description: "List all available configuration profiles",
    InputSchema: map[string]interface{}{"type": "object"},
}

// Tool 6: show_profile
var ShowProfileTool = Tool{
    Name:        "nettune.show_profile",
    Description: "Show details of a specific profile",
    InputSchema: map[string]interface{}{
        "type":     "object",
        "required": []string{"profile_id"},
        "properties": map[string]interface{}{
            "profile_id": map[string]interface{}{
                "type":        "string",
                "description": "Profile ID",
            },
        },
    },
}

// Tool 7: apply_profile
var ApplyProfileTool = Tool{
    Name:        "nettune.apply_profile",
    Description: "Apply a configuration profile to the server",
    InputSchema: map[string]interface{}{
        "type":     "object",
        "required": []string{"profile_id", "mode"},
        "properties": map[string]interface{}{
            "profile_id": map[string]interface{}{
                "type": "string",
            },
            "mode": map[string]interface{}{
                "type": "string",
                "enum": []string{"dry_run", "commit"},
            },
            "auto_rollback_seconds": map[string]interface{}{
                "type":    "number",
                "default": 60,
            },
        },
    },
}

// Tool 8: rollback
var RollbackTool = Tool{
    Name:        "nettune.rollback",
    Description: "Rollback to a previous snapshot",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "snapshot_id": map[string]interface{}{
                "type": "string",
            },
            "rollback_last": map[string]interface{}{
                "type":    "boolean",
                "default": false,
            },
        },
    },
}

// Tool 9: status
var StatusTool = Tool{
    Name:        "nettune.status",
    Description: "Get current server status and configuration",
    InputSchema: map[string]interface{}{"type": "object"},
}
```

### 6.3 推荐策略说明

**重要：nettune 不实现内置的推荐功能。**

配置优化策略完全由用户端的 LLM（如 Claude）通过组合使用上述 9 个工具来自主决定和实现。LLM 可以：

1. **收集数据**：调用 `test_rtt`、`test_throughput`、`test_latency_under_load` 获取网络性能指标
2. **分析现状**：调用 `status` 和 `show_profile` 了解当前配置
3. **查看选项**：调用 `list_profiles` 查看可用的配置 profiles
4. **推理决策**：根据收集的数据、用户目标和最佳实践，LLM 自主推理出最佳配置方案
5. **安全应用**：先调用 `apply_profile(mode="dry_run")` 预览变更，再 `apply_profile(mode="commit")` 应用
6. **验证效果**：重新测试性能指标，必要时调用 `rollback` 回滚

**优势：**
- **灵活性**：LLM 可以根据最新的网络优化知识和用户特定场景做出决策
- **可解释性**：LLM 可以用自然语言解释推荐理由和权衡
- **可扩展性**：无需修改代码即可适应新的优化策略和场景
- **用户参与**：用户可以在整个过程中与 LLM 交互，提供反馈和偏好

---

## 7. 测试策略

### 7.1 Go 项目测试

#### 7.1.1 单元测试结构

```
internal/
├── server/
│   ├── service/
│   │   ├── profile.go
│   │   ├── profile_test.go          # 单元测试
│   │   ├── snapshot.go
│   │   ├── snapshot_test.go
│   │   ├── apply.go
│   │   └── apply_test.go
│   └── adapter/
│       ├── sysctl.go
│       ├── sysctl_test.go
│       └── sysctl_integration_test.go  # 集成测试 (需要 Linux 环境)
```

#### 7.1.2 Mock 策略

```go
// internal/testutil/mock/system.go
type MockSystemAdapter struct {
    mock.Mock
}

func (m *MockSystemAdapter) GetSysctl(key string) (string, error) {
    args := m.Called(key)
    return args.String(0), args.Error(1)
}

func (m *MockSystemAdapter) SetSysctl(key, value string) error {
    args := m.Called(key, value)
    return args.Error(0)
}

// 使用示例
func TestApplyService_Apply_DryRun(t *testing.T) {
    mockAdapter := new(MockSystemAdapter)
    mockAdapter.On("GetSysctl", "net.core.default_qdisc").Return("pfifo_fast", nil)

    service := NewApplyService(mockAdapter, ...)
    result, err := service.Apply(&types.ApplyRequest{
        ProfileID: "bbr-fq.default",
        Mode:      "dry_run",
    })

    assert.NoError(t, err)
    assert.NotNil(t, result.Plan)
    mockAdapter.AssertExpectations(t)
}
```

#### 7.1.3 测试覆盖目标

| 模块              | 单元测试覆盖率 | 集成测试              |
| ----------------- | -------------- | --------------------- |
| MCP Server        | 80%+           | ✓ (mock HTTP server) |
| HTTP API Handlers | 85%+           | ✓ (httptest)         |
| Profile Service   | 90%+           | ✓ (文件系统)         |
| Snapshot Service  | 85%+           | ✓ (文件系统)         |
| Apply Engine      | 90%+           | ✓ (mock adapter)     |
| System Adapter    | 70%+           | ✓ (需要 Linux，可选) |

#### 7.1.4 测试命令

```bash
# 单元测试（不需要 root 权限）
go test ./... -v -short

# 集成测试（需要 Linux + root 权限）
sudo go test ./... -v -tags=integration

# 覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

#### 7.1.5 测试用例示例

**Profile 验证测试：**

```go
func TestProfileService_Validate(t *testing.T) {
    tests := []struct {
        name    string
        profile *types.Profile
        wantErr bool
    }{
        {
            name: "valid profile",
            profile: &types.Profile{
                ID:        "test-profile",
                Name:      "Test",
                RiskLevel: "low",
                Sysctl: map[string]interface{}{
                    "net.core.default_qdisc": "fq",
                },
            },
            wantErr: false,
        },
        {
            name: "invalid sysctl key",
            profile: &types.Profile{
                ID:        "test-profile",
                Name:      "Test",
                RiskLevel: "low",
                Sysctl: map[string]interface{}{
                    "invalid key": "value",
                },
            },
            wantErr: true,
        },
        // 更多测试用例...
    }

    service := NewProfileService(t.TempDir())
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.Validate(tt.profile)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**Apply Engine 测试：**

```go
func TestApplyService_Apply_Commit(t *testing.T) {
    // Setup
    mockAdapter := new(MockSystemAdapter)
    mockAdapter.On("GetSysctl", mock.Anything).Return("old_value", nil)
    mockAdapter.On("SetSysctl", mock.Anything, mock.Anything).Return(nil)
    mockAdapter.On("GetDefaultRouteInterface").Return("eth0", nil)
    mockAdapter.On("SetQdisc", "eth0", "fq", mock.Anything).Return(nil)

    profileService := NewProfileService("testdata/profiles")
    snapshotService := NewSnapshotService(t.TempDir(), mockAdapter)
    service := NewApplyService(profileService, snapshotService, mockAdapter, nil, logger)

    // Execute
    result, err := service.Apply(&types.ApplyRequest{
        ProfileID: "bbr-fq.default",
        Mode:      "commit",
    })

    // Assert
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.NotEmpty(t, result.SnapshotID)
    assert.NotNil(t, result.Verification)
    assert.True(t, result.Verification.SysctlOK)
    mockAdapter.AssertExpectations(t)
}
```

### 7.2 JS/TypeScript 测试

#### 7.2.1 测试结构

```
js/
├── src/
│   └── *.ts
└── test/
    ├── unit/
    │   ├── platform.test.ts
    │   ├── binary-manager.test.ts
    │   └── cli.test.ts
    ├── integration/
    │   └── spawn.test.ts
    └── fixtures/
        └── mock-binary.sh
```

#### 7.2.2 测试配置

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.ts'],
      exclude: ['src/**/*.test.ts', 'src/types.ts'],
      lines: 80,
      functions: 80,
      branches: 75,
      statements: 80,
    },
  },
});
```

#### 7.2.3 测试用例示例

**Platform 检测测试：**

```typescript
// test/unit/platform.test.ts
import { describe, it, expect, vi } from 'vitest';
import { detectPlatform, getBinaryName } from '../../src/platform';

describe('Platform', () => {
  describe('detectPlatform', () => {
    it('should detect darwin arm64', () => {
      vi.stubGlobal('process', { platform: 'darwin', arch: 'arm64' });
      const platform = detectPlatform();
      expect(platform).toEqual({ os: 'darwin', arch: 'arm64' });
    });

    it('should detect linux x64', () => {
      vi.stubGlobal('process', { platform: 'linux', arch: 'x64' });
      const platform = detectPlatform();
      expect(platform).toEqual({ os: 'linux', arch: 'x64' });
    });

    it('should throw on unsupported platform', () => {
      vi.stubGlobal('process', { platform: 'win32', arch: 'x64' });
      expect(() => detectPlatform()).toThrow();
    });
  });

  describe('getBinaryName', () => {
    it('should generate correct binary name', () => {
      expect(getBinaryName({ os: 'darwin', arch: 'arm64' }))
        .toBe('nettune-darwin-arm64');
      expect(getBinaryName({ os: 'linux', arch: 'x64' }))
        .toBe('nettune-linux-x64');
    });
  });
});
```

**Binary Manager 测试：**

```typescript
// test/unit/binary-manager.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BinaryManager } from '../../src/binary-manager';
import * as fs from 'fs/promises';
import * as path from 'path';

describe('BinaryManager', () => {
  let tmpDir: string;
  let manager: BinaryManager;

  beforeEach(async () => {
    tmpDir = await fs.mkdtemp('/tmp/nettune-test-');
    manager = new BinaryManager({
      cacheDir: tmpDir,
      githubRepo: 'test/nettune',
      version: 'v0.1.0',
    });
  });

  it('should download binary if not cached', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      arrayBuffer: () => Promise.resolve(Buffer.from('fake binary')),
    });
    vi.stubGlobal('fetch', mockFetch);

    const binaryPath = await manager.ensureBinary({
      os: 'darwin',
      arch: 'arm64',
    });

    expect(binaryPath).toContain(tmpDir);
    expect(mockFetch).toHaveBeenCalledOnce();
  });

  it('should use cached binary if available', async () => {
    const cachedPath = path.join(tmpDir, 'nettune-darwin-arm64');
    await fs.writeFile(cachedPath, 'fake binary');
    await fs.chmod(cachedPath, 0o755);

    const mockFetch = vi.fn();
    vi.stubGlobal('fetch', mockFetch);

    const binaryPath = await manager.ensureBinary({
      os: 'darwin',
      arch: 'arm64',
    });

    expect(binaryPath).toBe(cachedPath);
    expect(mockFetch).not.toHaveBeenCalled();
  });
});
```

#### 7.2.4 测试命令

```json
// package.json
{
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest watch",
    "test:coverage": "vitest run --coverage",
    "test:integration": "vitest run --config vitest.integration.config.ts"
  }
}
```

---

## 8. 配置与部署

### 8.1 默认 Profiles

**Profile 1: bbr-fq.default.json**

```json
{
  "id": "bbr-fq-default",
  "name": "BBR + FQ (Conservative)",
  "description": "Enable BBR congestion control with FQ qdisc, using conservative buffer sizes",
  "risk_level": "low",
  "requires_reboot": false,
  "sysctl": {
    "net.core.default_qdisc": "fq",
    "net.ipv4.tcp_congestion_control": "bbr",
    "net.ipv4.tcp_mtu_probing": 1
  },
  "qdisc": {
    "type": "fq",
    "interfaces": "default-route"
  },
  "systemd": {
    "ensure_qdisc_service": true
  }
}
```

**Profile 2: bbr-fq.tuned-32mb.json**

```json
{
  "id": "bbr-fq-tuned-32mb",
  "name": "BBR + FQ (Tuned 32MB buffers)",
  "description": "BBR with FQ and increased buffer sizes for high-bandwidth long-distance connections",
  "risk_level": "low",
  "requires_reboot": false,
  "sysctl": {
    "net.core.default_qdisc": "fq",
    "net.ipv4.tcp_congestion_control": "bbr",
    "net.core.rmem_max": 33554432,
    "net.core.wmem_max": 33554432,
    "net.ipv4.tcp_rmem": "4096 87380 33554432",
    "net.ipv4.tcp_wmem": "4096 65536 33554432",
    "net.ipv4.tcp_mtu_probing": 1,
    "net.ipv4.tcp_slow_start_after_idle": 0
  },
  "qdisc": {
    "type": "fq",
    "interfaces": "default-route"
  },
  "systemd": {
    "ensure_qdisc_service": true
  }
}
```

### 8.2 Systemd Service（可选）

**qdisc 保活服务：**

```ini
# /etc/systemd/system/nettune-qdisc.service
[Unit]
Description=Nettune Qdisc Persistence
After=network.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/nettune-qdisc-setup.sh
ExecStop=/bin/true

[Install]
WantedBy=multi-user.target
```

**设置脚本：**

```bash
#!/bin/bash
# /usr/local/bin/nettune-qdisc-setup.sh
DEFAULT_IFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
tc qdisc replace dev "$DEFAULT_IFACE" root fq
```

### 8.3 构建和发布流程

#### 8.3.1 构建脚本

```bash
#!/bin/bash
# scripts/build.sh
set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty)}
OUTPUT_DIR="dist"

mkdir -p "$OUTPUT_DIR"

# Build for multiple platforms
PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

for platform in "${PLATFORMS[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "$platform"
  output="$OUTPUT_DIR/nettune-$GOOS-$GOARCH"

  echo "Building for $GOOS/$GOARCH..."
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags "-X github.com/jtsang4/nettune/pkg/version.Version=$VERSION" \
    -o "$output" \
    ./cmd/nettune
done

# Generate checksums
cd "$OUTPUT_DIR"
sha256sum nettune-* > checksums.txt
```

#### 8.3.2 GitHub Actions 工作流

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test ./... -v

      - name: Build binaries
        run: ./scripts/build.sh
        env:
          VERSION: ${{ github.ref_name }}

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*
          generate_release_notes: true
```

---

## 9. 开发流程与规范

### 9.1 Git 工作流

- **主分支：** `main`（稳定版本）
- **开发分支：** `develop`（集成分支）
- **特性分支：** `feature/<name>`
- **修复分支：** `fix/<name>`

### 9.2 提交规范

遵循 Conventional Commits：

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型：

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档
- `test`: 测试
- `refactor`: 重构
- `chore`: 构建/工具

示例：

```
feat(server): implement profile validation

- Add schema validation for profile JSON
- Add dangerous sysctl detection
- Add unit tests

Closes #123
```

### 9.3 代码审查清单

**Go 代码：**

- [ ] 所有导出函数有文档注释
- [ ] 错误处理完整（不忽略 error）
- [ ] 日志级别合理（Error/Warn/Info/Debug）
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 无 race condition（`go test -race`）
- [ ] 使用 `go fmt` 和 `golangci-lint`

**TypeScript 代码：**

- [ ] 类型定义完整（无 `any`）
- [ ] 错误处理完整（try-catch 或 Promise.catch）
- [ ] 日志输出到 stderr
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 通过 `eslint` 和 `prettier`

---

## 10. 风险与缓解措施

### 10.1 技术风险

| 风险                 | 影响 | 概率 | 缓解措施                        |
| -------------------- | ---- | ---- | ------------------------------- |
| Netlink 库兼容性问题 | 高   | 中   | 提供 tc 命令降级方案；充分测试  |
| Root 权限滥用        | 高   | 低   | 白名单机制；输入校验；审计日志  |
| 并发 apply 冲突      | 中   | 中   | 文件锁 + 进程内互斥锁           |
| Snapshot 回滚失败    | 高   | 低   | 多重验证；失败后人工介入指南    |
| MCP stdio 通道污染   | 中   | 中   | 严格日志路由到 stderr；集成测试 |

### 10.2 运维风险

| 风险                 | 影响 | 概率 | 缓解措施                           |
| -------------------- | ---- | ---- | ---------------------------------- |
| Apply 后系统无法连接 | 高   | 低   | Auto-rollback 机制（默认 60s）     |
| API Key 泄露         | 高   | 中   | 临时 key；使用后立即更换；不持久化 |
| 磁盘空间耗尽         | 中   | 低   | Snapshot 自动清理策略；监控        |

---

## 11. 后续扩展方向

### 11.1 Phase 2 功能（未来）

1. **高级整形策略**

   - CAKE qdisc with rate limiting
   - FQ_CODEL 优化
   - 针对 ISP policer 的对策
2. **多链路对比**

   - 支持配置多个 server
   - 并行测试和对比
   - 最优路径推荐
3. **安全增强**

   - TLS 支持（HTTPS + cert pinning）
   - IP allowlist
   - 更细粒度的 rate limiting
4. **监控与告警**

   - Prometheus metrics 导出
   - 长期性能趋势分析
   - 配置漂移检测

### 11.2 生态集成

- **Terraform Provider**：IaC 方式管理配置
- **Ansible Module**：批量部署和配置
- **Grafana Dashboard**：可视化性能指标

---

## 12. 交付验收标准

### 12.1 功能验收

- [ ] Server 模式可启动并响应所有 API 端点
- [ ] Client 模式可启动 MCP stdio server
- [ ] NPM wrapper 可通过 `npx` 启动 client
- [ ] 9 个 MCP tools 全部可用（不包含推荐功能）
- [ ] LLM 可以通过组合使用 tools 完成优化建议
- [ ] Profile apply/rollback 流程完整
- [ ] Snapshot 机制可靠

### 12.2 质量验收

- [ ] Go 代码单元测试覆盖率 ≥ 80%
- [ ] TypeScript 代码单元测试覆盖率 ≥ 80%
- [ ] 通过集成测试（至少在 Ubuntu 24.04）
- [ ] 无明显内存泄漏（压力测试）
- [ ] 日志不污染 MCP stdio 通道

### 12.3 文档验收

- [ ] README 包含快速开始指南
- [ ] API 文档完整
- [ ] MCP tools 使用说明
- [ ] 故障排查指南

---

## 附录

### A. 参考资料

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Gin Framework Documentation](https://gin-gonic.com/docs/)
- [BBR Congestion Control](https://queue.acm.org/detail.cfm?id=3022184)
- [Linux tc qdisc](https://man7.org/linux/man-pages/man8/tc.8.html)
- [vishvananda/netlink](https://github.com/vishvananda/netlink)

### B. 术语表

| 术语        | 说明                                                    |
| ----------- | ------------------------------------------------------- |
| BBR         | Bottleneck Bandwidth and RTT，Google 开发的拥塞控制算法 |
| FQ          | Fair Queue，公平队列调度器                              |
| Qdisc       | Queue Discipline，Linux 流量控制的排队规则              |
| Netlink     | Linux 内核与用户空间通信的 socket 接口                  |
| Sysctl      | Linux 内核参数配置接口                                  |
| MCP         | Model Context Protocol，AI 与工具通信协议               |
| BDP         | Bandwidth-Delay Product，带宽延迟积                     |
| Bufferbloat | 缓冲区膨胀，过大缓冲导致的延迟问题                      |

---

**文档版本：** v1.0
**最后更新：** 2025-12-06
**维护者：** jtsang4
