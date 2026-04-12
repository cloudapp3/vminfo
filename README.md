# vminfo

`vminfo` 是一个独立的主机运行时信息工具，目标形态是：

- 可嵌入的 Go 库
- 正式可用的 CLI
- 默认进入 TUI
- 支持单次摘要、持续观察、进程列表与 Linux `SIGTERM`

## 命令

```bash
vminfo                 # 默认进入 TUI
vminfo info            # TUI 别名
vminfo version         # 版本信息
vminfo version --json  # 应用元数据 JSON
vminfo summary         # 单次采样
vminfo summary --json
vminfo watch           # 持续输出快照
vminfo watch --json    # JSON Lines
vminfo watch --count 1 # 仅输出一次，便于脚本验证
vminfo ps              # 进程列表
vminfo ps --json
vminfo kill <pid>      # Linux: 发送 SIGTERM
vminfo --version       # 版本信息快捷方式
```

## 当前状态

当前仓库已完成：

- 初始化独立 Go module
- 静态信息与动态指标采集
- 默认 `vminfo` TUI
- `summary / watch / ps / kill` 非 TUI CLI
- `version / --version` 与应用元数据输出
- Linux-only `ps / kill`

## App metadata

`vminfo version --json` 会输出应用自身元数据，例如：

```json
{
  "name": "vminfo",
  "version": "dev",
  "channel": "dev",
  "repository": "https://github.com/VPSMarket/vminfo",
  "homepage": "https://github.com/VPSMarket/vminfo",
  "description": "Host runtime information toolkit",
  "schema_version": "v1"
}
```

对嵌入方也提供：

```go
meta := vminfo.Metadata()
```

## Version 注入

建议通过 `-ldflags` 在构建时注入版本字段：

```bash
go build -ldflags "\
  -X github.com/VPSMarket/vminfo/pkg/vminfo.Version=v0.1.0 \
  -X github.com/VPSMarket/vminfo/pkg/vminfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/VPSMarket/vminfo/pkg/vminfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/VPSMarket/vminfo/pkg/vminfo.Channel=stable" \
  ./cmd/vminfo
```

## GitLab CI 与 Debian 运行

仓库内置 `.gitlab-ci.yml`，会为 `main` 分支构建 Linux amd64 二进制产物：

- job 名：`build-linux-amd64`
- artifact 路径：`dist/vminfo-linux-amd64`

在 Debian x86_64 VPS 上，可用仓库脚本下载并运行最新成功构建：

```bash
bash scripts/run_latest_vminfo.sh
```

默认执行：

```bash
vminfo summary
```

也可以传入参数：

```bash
bash scripts/run_latest_vminfo.sh version --json
bash scripts/run_latest_vminfo.sh watch --count 1
bash scripts/run_latest_vminfo.sh ps
```

如果仓库是私有的，先设置：

```bash
export GITLAB_TOKEN=your_gitlab_pat
```

## 本地运行

```bash
go run ./cmd/vminfo
```

查看帮助：

```bash
go run ./cmd/vminfo --help
```

单次 watch 验证：

```bash
go run ./cmd/vminfo watch --count 1
go run ./cmd/vminfo watch --json --count 1
```
