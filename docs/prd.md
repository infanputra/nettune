# Nettune 需求与设计文档（交付研发版）

> 本文档定义一个“单可执行文件 + client/server 双模式”的网络诊断与 TCP 优化工具链。
>
> 目标是让用户在 **一个 Chat GUI** 内完成：发起测速 → 自动诊断 → 生成优化建议 → 通过工具下发配置 → 可回滚验证。

---

## 1. 背景与问题

用户希望对 Linux 服务器的 TCP 网络体验进行优化（以 BBR + FQ 为基础），但“最优配置”取决于真实的  **client↔server 路径特性** （RTT、抖动、丢包、上游策略/policer、拥塞时段等）。传统“一键脚本”容易误判与不可回滚。

本项目采用“端到端测量 + 声明式配置版本化 + 可回滚 apply”的方法，并通过本机 MCP（stdio）让 Chat GUI/LLM 能稳定调用工具完成闭环。

---

## 2. 目标（Goals）

### 2.1 核心目标

1. **单二进制分发** ：一个 Go 可执行文件，根据参数启动 client 或 server。
2. **端到端测量** ：测速主要从用户本机（client 侧）发起，测得 client→server 的真实体验。
3. **Chat GUI 单点交互** ：用户仅在 Chat GUI 内操作，由 GUI 调用本机 client 模式的 MCP（stdio）工具完成全部流程。
4. **server 只做 HTTP agent（非 MCP）** ：server 仅提供 HTTP API（含 bearer 认证），承载测速端点与系统配置 apply/rollback 能力。
5. **配置版本化 + 可回滚** ：

* server 端维护 profiles（配置版本）与 snapshots（回滚快照）
* apply 前自动 snapshot；支持 rollback 到任意 snapshot

1. **默认值优先** ：除 `API Key` 外，其余参数尽量有合理默认并可覆盖。

### 2.2 非目标（Non-goals）

* 不追求成为通用性能测试套件（不等价于 iperf/ookla）。
* 不强制支持所有 OS；主要面向 Linux 服务器（优先 Ubuntu 24.04 / systemd）。
* server 公网暴露仅用于临时调试：不做完整企业级安全体系（但必须有基本 bearer 认证与输入校验）。

---

## 3. 术语

* **client 模式** ：用户本机运行的进程，提供 MCP stdio server 给 Chat GUI 调用；同时作为 agent 客户端连接远端 server。
* **server 模式** ：服务器上运行的进程，暴露 HTTP API（公网端口），执行测速端点与系统配置变更。
* **Profile（配置版本）** ：声明式描述一组系统网络配置（如 BBR+FQ、buffer 调优、systemd qdisc 保活等）。
* **Snapshot（回滚快照）** ：apply 前自动捕获的系统状态与文件备份，用于回滚。
* **Plan** ：一次 apply 的具体执行计划（通常由 profile + 少量参数渲染而来，必须受控且可校验）。

---

## 4. 高层架构

### 4.1 组件

1. **Go 二进制：`nettune`**
   * `nettune client`：
     * 启动 MCP stdio server（本机）
     * HTTP 调用远端 `nettune server` API
     * 执行 client 侧测试（对 server 的 echo/down/upload 等）
   * `nettune server`：
     * 启动 HTTP 服务（公网监听）
     * 提供 `/probe/*` 测速端点
     * 提供 `/profiles/*` `/sys/*` 配置读取与 apply/rollback 端点
     * 维护 profiles/snapshots/history
2. **NPM wrapper（TypeScript/Node）**
   * 提供 `npx` 启动入口（STDIO MCP 常见启动方式）
   * wrapper 负责下载/定位对应平台的 `nettune` 二进制并 `spawn nettune client ...`

### 4.2 数据流（典型闭环）

Chat GUI →（stdio MCP）→ `nettune client` →（HTTP + bearer）→ `nettune server`

* 测试：client 拉 `/probe/echo|download|upload` 并统计
* 获取 server 状态：client 调 `/sys/snapshot`、`/profiles`
* 推荐：client 本地推导建议（可选也可由 server 推导，但建议 client 推导）
* 应用：client 调 `/sys/apply`
* 验证：重复测试 + 需要时 `/sys/rollback`

---

## 5. 运行模式与 CLI 设计

### 5.1 通用约定

* 所有日志默认输出到  **stderr** （尤其 client 的 MCP stdio 不能污染 stdout）。
* API Key 不应写入日志或持久化（除非明确配置）。

### 5.2 server 模式

命令：

```bash
nettune server --api-key <KEY> [--listen 0.0.0.0:9876] [--state-dir <DIR>] [...]
```

必填：

* `--api-key`：Bearer token（唯一必须项）

推荐默认：

* `--listen`：`0.0.0.0:9876`
* `--state-dir`：
  * 优先：`$XDG_CONFIG_HOME/nettune`
  * 否则：`~/.config/nettune`
  * 注意：若以 root 运行，`~` 为 `/root`；允许用户覆盖为 `/var/lib/nettune` 等更合理目录

可选（建议实现但默认不必强制）：

* `--read-timeout` / `--write-timeout`
* `--max-body-bytes`（防止大请求打爆内存）
* `--allow-unsafe-http`（默认 true；若未来加 TLS 反代，可设 false 控制）

### 5.3 client 模式

命令：

```bash
nettune client --api-key <KEY> [--server http://127.0.0.1:9876] [...]
```

必填：

* `--api-key`

默认：

* `--server`：`http://127.0.0.1:9876`（方便本机自测；实际使用时由用户指定）
* `--mcp-name`：`nettune`（可选，仅用于 Chat GUI 配置标识）

---

## 6. server：HTTP API 设计（非 MCP）

### 6.1 认证

* Header：`Authorization: Bearer <KEY>`
* 所有端点必须校验 key；拒绝未授权请求（401）。

### 6.2 Probe（测速端点）

> 设计原则：尽量不用 ICMP/raw socket（跨平台/权限复杂），使用 HTTP 贴近真实业务路径。

1. `GET /probe/echo`

* 返回小 JSON（例如 `{ "ts": <server_time_ms>, "ok": true }`）
* client 用于统计 RTT/jitter（多次请求、记录分位数）

2. `GET /probe/download?bytes=<N>`

* 流式输出 N 字节随机/伪随机数据（避免压缩影响，`Content-Encoding: identity`）
* client 用于测下载吞吐（单连接/多连接）

3. `POST /probe/upload`

* 读取请求 body（N 字节），返回 `{ received_bytes, duration_ms }`
* client 用于测上传吞吐

4. `GET /probe/info`

* 返回 server 侧环境摘要（用于诊断解释）：
  * 内核版本、发行版信息
  * 当前拥塞控制、默认 qdisc
  * 默认路由出口网卡、MTU、速率（若可得）
  * 近期接口丢包计数（若可得）

### 6.3 Profiles / System（配置与回滚端点）

1. `GET /profiles`

* 列出可用 profile：
  * `id`, `name`, `description`, `risk_level`, `requires_reboot`(通常 false)

2. `GET /profiles/{id}`

* 返回 profile 声明式内容（见 7.2 schema）

3. `POST /sys/snapshot`

* 创建快照并返回：
  * `snapshot_id`
  * `current_state`（拥塞控制/qdisc/关键 sysctl/网卡信息/相关文件哈希）

4. `POST /sys/apply`

* 入参：
  * `profile_id`（优先）
  * `mode`: `"dry_run" | "commit"`
  * 可选 `auto_rollback_seconds`（建议默认 60，可配置为 0 关闭）
* 返回：
  * `plan`（dry_run 展示即将变更项）
  * `snapshot_id`（commit 前自动创建）
  * `apply_result`（commit 后验证结果）

> commit 模式必须：apply 前 snapshot；apply 后校验；失败自动 rollback 并返回错误。

5. `POST /sys/rollback`

* 入参：`snapshot_id` 或 `rollback_last: true`
* 返回：回滚结果 + 当前状态

6. `GET /sys/status`

* 返回当前生效状态：
  * 最近一次 apply 的 profile_id、时间、结果
  * 当前拥塞控制/qdisc/关键 sysctl

---

## 7. server：配置版本化与回滚机制

### 7.1 目录布局（state-dir）

推荐结构（位于 `state-dir` 下）：

```
nettune/
  profiles/
    bbr-fq.default.json
    bbr-fq.tuned-32mb.json
  snapshots/
    2025-12-06T10-20-33Z_<rand>/
      state.json
      backups/
        etc_sysctl_d_99-nettune.conf
        systemd_units/...
      qdisc.json
  history/
    journal.jsonl
```

* `profiles/`：声明式配置版本（可内置默认 profiles，并允许用户放自定义 profile）
* `snapshots/`：每次 commit apply 前自动生成快照
* `history/`：审计日志（不含 api key）

### 7.2 Profile（声明式）建议 schema

Profile 不存“任意命令”，只存受控字段。例如：

```json
{
  "id": "bbr-fq.tuned-32mb",
  "name": "BBR + FQ (tuned buffers 32MB)",
  "risk_level": "low",
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

### 7.3 Apply 的实现策略（必须点）

* **原子写入** ：生成配置文件写到临时文件再 rename 覆盖（避免半写）
* **并发锁** ：同一时间只允许一个 apply/rollback（文件锁或进程内互斥 + state 记录）
* **验证** ：commit 后必须验证：
* 读回 sysctl 是否生效
* 读取 qdisc 状态是否为预期（默认路由出口网卡）
* systemd unit 是否启用（若 profile 需要）
* **失败回滚** ：验证失败立即 rollback 到 snapshot

### 7.4 “系统改动落点”

建议统一落到可控文件（便于维护与回滚）：

* sysctl：写入 `/etc/sysctl.d/99-nettune.conf`（由 nettune 管理）
* qdisc：使用 netlink 或 tc 设置默认路由出口网卡 root qdisc
* systemd：如需“重启后确保 qdisc”，写入 `bbr-fq-qdisc.service` + 脚本（或使用内置方式）

> 研发可选择：

* **优先使用 Go netlink** （减少对 `tc` 外部依赖）
* systemd 操作可调用 `systemctl`（Ubuntu 24.04 默认可用）

---

## 8. client：MCP（stdio）工具设计

### 8.1 MCP 形态

* client 模式启动后，使用 MCP stdio 与 Chat GUI 通信
* stdout 严禁打印非 MCP 内容；日志写 stderr
* MCP 内仅提供 tools（不强制 resources/prompts，避免客户端不适配问题）

### 8.2 Tools 清单（建议最小闭环）

工具命名以 `nettune.*` 为前缀：

1. `nettune.test_rtt`

* 参数：`server?`, `count?`, `concurrency?`, `keepalive?`
* 行为：多次调用 `/probe/echo`，输出 p50/p90/p99、抖动、错误率

2. `nettune.test_throughput`

* 参数：`server?`, `direction`(`download|upload`), `bytes?`, `parallel?`
* 行为：download 调 `/probe/download`；upload 调 `/probe/upload`
* 输出：单连接/多连接吞吐（Mbps）、耗时、失败率

3. `nettune.test_latency_under_load`

* 参数：`server?`, `duration?`, `load_parallel?`, `echo_interval?`
* 行为：背景跑 download/upload 负载 + 前台 echo 采样
* 输出：负载前后 RTT 分位数变化、最大尖峰

4. `nettune.snapshot_server`

* 调用 `/sys/snapshot`，返回 `snapshot_id` 与 `current_state`

5. `nettune.list_profiles`

* 调用 `/profiles`

6. `nettune.show_profile`

* 参数：`profile_id`
* 调用 `/profiles/{id}`

7. `nettune.recommend`

* 参数：`goal`（如 `throughput|latency|balanced`）、以及（可选）指定使用最近测试结果
* 输出：建议结果（结构化）：
  * `diagnosis`（分类）
  * `recommended_profile_id`（或多个候选）
  * `reasons`（可解释文本）
  * `risks`（风险提示）
  * `next_steps`（建议验证步骤）

8. `nettune.apply_profile`

* 参数：`profile_id`, `mode`(`dry_run|commit`), `auto_rollback_seconds?`
* 调用 `/sys/apply`

9. `nettune.rollback`

* 参数：`snapshot_id` 或 `rollback_last`
* 调用 `/sys/rollback`

10. `nettune.status`

* 调用 `/sys/status` + `/probe/info` 聚合输出

---

## 9. 推荐（Recommend）逻辑（确定性优先）

> 目标：不做“玄学 AI 调参”，而做“规则/统计驱动的可解释建议”。LLM 主要负责编排工具调用与输出呈现，核心判定尽量确定性。

### 9.1 输入

注意输入信息 AI 会在 Chat 过程中自动获取，这里不需要单独实现“"推荐策略相关的功能”"，只需要能够接受 AI 给出的推荐配置结果进行应用，下面是说明 AI 的推荐逻辑。后面给用户使用的时候，会有一个推荐的 system prompt（里面包含了根据不同的输入信息进行配置调优的策略）和推荐的 tool 使用方法。

* 最近一次或多次测试结果：
  * RTT 分位数、抖动、错误率
  * 吞吐（单/多连接）
  * latency-under-load 的 RTT 膨胀程度
* server 当前状态（snapshot/status）
* 用户目标（吞吐优先 / 延迟优先 / 平衡）

### 9.2 输出分类（示例）

* **BDP/窗口不足型** ：吞吐明显跑不满，延迟不爆；建议 buffer 调优 profile
* **负载延迟膨胀型（疑似 bufferbloat/policer）** ：吞吐上来时 RTT p90/p99 大幅飙升；提示可能需要整形（可作为未来 profile 扩展）
* **路径/拥塞主导型** ：不同目标/时段差异极大且和配置关系弱；提示可能换线路/入口更有效


---

## 10. NPM wrapper（TypeScript）设计

### 10.1 目标

* 支持用户通过 `npx` 启动本机 MCP stdio server（即 `nettune client`）
* wrapper 仅负责“拉起二进制并透传 stdio”，不承载业务逻辑

### 10.2 包形态

* npm 包（例如 `nettune-mcp`）：
  * `bin`: `nettune-mcp` → `dist/index.js`
* 行为：
  1. 解析参数（至少 `--api-key`、`--server` 透传）
  2. 确定平台 arch（darwin/linux, amd64/arm64）
  3. 定位对应二进制：
     * 推荐：发行版 release 附带多平台二进制；wrapper 首次运行下载并缓存到 `~/.cache/nettune/`
  4. `spawn(binary, ["client", ...args], { stdio: "inherit" })`

### 10.3 关键注意

* wrapper 的非必要输出必须写到 stderr（避免污染 MCP 通道）
* 下载需要校验 hash（最小保障）
* 二进制需 `chmod +x`

---

## 11. 依赖与兼容性

### 11.1 server 侧（Linux）

* 目标：Ubuntu 24.04（systemd）
* 必要能力：
  * 写 `/etc/sysctl.d/`
  * 配置 qdisc（建议 Go netlink；或依赖 iproute2/tc）
  * 可选：systemd unit enable/start（调用 systemctl）

> 若依赖外部 `systemctl`/`tc`，server 应在启动时自检并在 API /probe/info 中报告依赖缺失与解决建议。

### 11.2 client 侧（本机）

* 主要执行 HTTP 测试（echo/download/upload），无需额外系统依赖
* 支持 macOS/Linux 优先（Windows 可作为后续扩展）

---

## 12. 运行与运维约束

* server 模式默认不 daemonize：临时启动，调试结束即停
* 但 apply/rollback 必须具备基本健壮性：
  * 并发锁
  * 验证与失败回滚
  * history 记录（不含敏感信息）

---

## 13. 交互流程（给研发用于验收）

### 13.1 最小闭环（推荐验收用例）

1. 用户在服务器启动：

```bash
sudo nettune server --api-key XXX
```

2. 用户在本机 Chat GUI 配置 MCP：通过 `npx nettune-mcp --api-key XXX --server http://<server>:9876`
3. 在 Chat GUI 发起：
   * `nettune.test_rtt`
   * `nettune.test_throughput(download/upload)`
   * `nettune.test_latency_under_load`
   * `nettune.snapshot_server`
   * `nettune.recommend(goal="balanced")`
   * `nettune.apply_profile(mode="dry_run")`
   * `nettune.apply_profile(mode="commit")`
   * 重测验证
   * 必要时 `nettune.rollback`

### 13.2 验收标准

* MCP tools 可在 GUI 内被调用且输出结构化结果
* profile 列表可读，apply 可落地且可回滚
* apply 过程不会留下无法解释的“散落改动”（所有改动可追溯到 nettune 管理的文件/快照）
* 日志不污染 stdio MCP 通道

---

## 14. 未来扩展（不在本期必须交付）

* 引入整形类 profiles（cake/fq_codel + rate），用于 policer/延迟膨胀型问题
* 支持多 server、多链路对比
* 更丰富的路径诊断（mtr/tracepath 作为可选插件）
* 更完善的安全策略（TLS、IP allowlist、速率限制策略等）

---

## 15. 交付物清单（研发输出）

1. Go 项目：

   * `nettune` 单二进制：`client`/`server` 两模式
   * server HTTP API（含 bearer、profiles、snapshots、probe）
   * client MCP stdio tools 实现（调用 server + 本机测速聚合）
2. 默认 profiles（至少两份）：

   * `bbr-fq.default`
   * `bbr-fq.tuned-32mb`
3. TS NPM wrapper(使用 bunjs 作为包管理器，代码放在 /js 目录))：

   * `npx nettune-mcp ...` 可启动 `nettune client` 并透传 stdio
   * 二进制下载/缓存/校验机制
4. 使用说明（README）：

   * server 启动方式
   * npx/本地二进制启动方式
   * Chat GUI 如何配置 MCP（stdio）
   * 常见故障排查（端口、认证、依赖缺失、回滚）
