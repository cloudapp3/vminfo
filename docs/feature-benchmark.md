# vminfo 功能对标与路线建议

本文记录 vminfo 对标同类开源项目后的功能取舍。它不是承诺清单，而是后续规划、Issue 拆分和 PR 评审时的参考。

## 产品定位

vminfo 应优先保持以下边界：

- **单二进制、低依赖、低配置**：安装后即可用，不要求守护进程、数据库或中心端。
- **本机实时诊断优先**：TUI、CLI、JSON、Web/API 都围绕“快速看清当前机器状态”设计。
- **脚本与嵌入友好**：保持 JSON schema 和 Go public API 稳定，适合 CI、自动化和其他 Go 工具集成。
- **轻量 Web，而不是监控平台**：可以借鉴 Netdata 的一眼看懂，但不追完整云端、多节点、长期存储平台。
- **平台语义清晰**：`summary` / `watch` 跨平台；`ps` / `kill` 继续只在 Linux 支持，非 Linux 保留 unsupported stub。

## 对标项目与可借鉴功能

| 对标项目 | 代表能力 | vminfo 可借鉴方向 |
| --- | --- | --- |
| [btop](https://github.com/aristocratos/btop) | 高完成度 TUI、资源图、进程筛选、树形进程、信号操作、主题 | 强化 TUI 视觉与进程交互：进程详情、网络/磁盘历史小图、主题、follow 模式 |
| [bottom / btm](https://github.com/ClementTsang/bottom) | 跨平台终端系统监控、widget 聚焦、布局配置、时间窗口 | 增加 focus/basic 模式，提升宽窄屏布局和短期历史窗口 |
| [Glances](https://github.com/nicolargo/glances) | TUI、Web、REST API、JSON/CSV/stdout、client/server、MCP | 补强脚本输出、字段选择、CSV、API 文档，后续可选 MCP / 只读远程模式 |
| [procs](https://github.com/dalance/procs) | 现代 `ps` 替代、关键词搜索、tree、watch、端口、IO、Docker 信息 | 将 `vminfo ps` 做成更实用的排障入口：搜索、tree、watch、limit、ports、io、container |
| [htop](https://github.com/htop-dev/htop) | 成熟进程操作习惯、排序、过滤、kill/renice、完整命令查看 | TUI 进程页对齐常见按键习惯，补横向命令查看、信号选择、更多进程列 |
| [Netdata](https://github.com/netdata/netdata) | 零配置实时 Web、自动发现、资源异常提示、丰富 dashboard | 只借鉴“零配置 + 一眼看懂”：Web 问题摘要、资源压力提示、健康评分；不追平台化 |
| [gopsutil](https://github.com/shirou/gopsutil) / [psutil](https://github.com/giampaolo/psutil) | 稳定跨平台采集库、CPU/内存/磁盘/网络/进程模块化 API | 保持 vminfo 高层 Go API 稳定，补充示例、版本化 schema、可嵌入文档 |

## 推荐路线

### P0：优先落地，收益最高

1. **增强 `vminfo ps`**（已开始落地）
   - `vminfo ps <keyword>`：按进程名、用户、PID、命令行过滤。
   - `vminfo ps --filter <keyword>`：显式过滤参数，便于脚本使用。
   - `vminfo ps --tree`：树形进程视图。
   - `vminfo ps --watch`：持续刷新进程表，支持 `--count` 和 `--interval`。
   - `vminfo ps --limit 20`：排序 / 过滤后限制输出行数。
   - `vminfo ps --sort cpu|mem|pid|name`：保持现有排序，并配合过滤 / limit。
   - 文本输出新增 `AGE` / `COMMAND` 列，JSON 新增 `command` / `started_at_unix`。

2. **增加进程详情**
   - TUI 选中进程后展示 PID、PPID、user、command line、CPU、RSS、启动时间。
   - Web `/api/v1/processes` 和进程表补充详情入口（过滤、排序、limit 已开始落地）。
   - Linux 上可继续扩展 open ports、I/O、cgroup/container 信息。

3. **Web dashboard 增加“问题摘要”**
   - 首页顶部给出当前最重要的 3～5 个信号：
     - CPU / load 是否过高。
     - 内存或 swap 是否紧张。
     - 磁盘空间是否接近阈值。
     - 网卡 errors / drops 是否异常。
     - 高 CPU / 高内存进程 Top N。
   - 保持只读、无状态、无长期存储。

### P1：增强自动化和可集成能力

4. **字段选择与表格输出**
   - `vminfo watch --fields cpu,mem,load`
   - `vminfo summary --fields cpu,mem,disk`
   - `vminfo watch --csv --fields cpu,mem,load`
   - 注意字段命名要和 JSON schema 尽量一致，避免创建第二套概念。

5. **补充 HTTP API 文档**
   - 新增 `docs/api.md`，说明：
     - `GET /healthz`
     - `GET /api/v1/snapshot`
     - `GET /api/v1/cpu`
     - `GET /api/v1/memory`
     - `GET /api/v1/disk`
     - `GET /api/v1/network`
     - `GET /api/v1/processes`
     - `GET /api/v1/system`
     - `GET /ws`
   - 记录鉴权 token、cookie、CORS、WebSocket origin 规则。

6. **短期历史窗口**
   - Collector 内存中保留最近 60～300 个采样点。
   - TUI 和 Web 可展示 CPU、内存、网络、磁盘 I/O 的短历史趋势。
   - 不引入数据库，不默认落盘。

7. **Go library 示例**
   - README / `docs/` 中补更多 import 用法：
     - 采集单次快照。
     - 持续采样。
     - 输出自定义 JSON。
     - 嵌入 TUI。

### P2：有价值，但不应抢占轻量核心

8. **MCP server**
   - 可选命令形态：`vminfo mcp`。
   - 可暴露工具：
     - `get_snapshot`
     - `list_processes`
     - `get_top_processes`
     - `get_system_health`
   - 适合 AI 运维助手读取本机状态，但不应影响普通 CLI 使用。

9. **容器识别**
   - Linux 上从 cgroup / proc 信息中识别 Docker/containerd 容器 ID 或名称。
   - 在 `ps`、TUI、Web process table 中显示 container 字段。

10. **轻量健康评分**
    - 输出 `health_score` 和 `warnings`。
    - 示例：

      ```json
      {
        "health_score": 86,
        "warnings": ["disk / usage over 85%"]
      }
      ```

    - 评分逻辑必须透明、可解释，避免黑盒告警。

## 暂不建议投入的方向

- 长期指标存储、数据库、复杂 retention 策略。
- 云端账号、多节点集中管理、告警平台。
- 重插件系统或复杂配置文件。
- 默认暴露 Prometheus / InfluxDB 全量 exporter。
- 将 Web dashboard 做成需要构建链路的大型前端应用。

这些方向会削弱 vminfo 当前最重要的优势：轻量、直接、单机可用、脚本友好。

## 落地约束

- CLI 行为、JSON 输出、Web API 或导出 Go API 发生变化时，需要同步更新 README、`docs/README.zh-CN.md`、`docs/CHANGELOG.md` 和相关测试。
- `ps` / `kill` 的 Linux-only 约束不能被无意打破；非 Linux 必须继续返回 unsupported。
- Web 远程访问相关能力必须优先考虑 token、cookie、CORS 和 WebSocket origin 安全边界。
- 如果后续重新启用其他语言实现，应保持主线 Go 版本为发布基准，并避免 CI / release 依赖未纳入 Git 的本地工作区。
