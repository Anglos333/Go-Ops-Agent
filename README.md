# go-ops-agent

一个 Linux 优先的终端运维助手。当前版本支持：

- 使用 [`github.com/sashabaranov/go-openai`](go.mod) 连接兼容 OpenAI Chat Completions 的模型服务。
- 使用 [`github.com/shirou/gopsutil/v3`](go.mod) 采集 CPU、内存和 Top 5 进程信息。
- 使用 [`github.com/pterm/pterm`](go.mod) 提供 Spinner、彩色提示和终端渲染。
- [`ops-agent ask`](cmd/ask.go) 打印 AI 回复，并在发现 ```bash``` 代码块时要求用户确认后执行。
- [`ops-agent diag`](cmd/diag.go) 自动采集主机指标和最近 100 行日志，请 AI 基于真实数据做诊断。

## 配置

默认读取 `C:/Users/<you>/.ops-agent.yaml` 或环境变量：

```yaml
provider:
  base_url: https://api.deepseek.com
  api_key: your-api-key
  model: deepseek-chat
```

也可使用环境变量覆盖：

- `OPS_AGENT_BASE_URL`
- `OPS_AGENT_API_KEY`
- `OPS_AGENT_MODEL`

## Linux 行为说明

- 系统日志优先读取 `/var/log/syslog`。
- 若 syslog 不可用，则回退到 `journalctl -n 100 --no-pager`。
- 当问题包含 OOM 或“内存溢出”时，会优先提取 OOM 相关日志行再交给模型分析。

## 安全执行机制

- 模型若返回 `bash` 代码块，将由 [`internal/executor/executor.go`](internal/executor/executor.go) 提取。
- 执行前会显示“警告：即将执行以下系统级命令，是否继续？(Y/N)”。
- 只有用户明确输入 `Y` 或 `YES` 才会通过 `bash -lc` 执行命令并回显输出。

`go-ops-agent` 是一个面向终端场景的运维辅助 CLI，目标是把本机诊断信息采集、LLM 分析与后续执行建议串起来，形成一个轻量的 AI Ops Assistant。

当前仓库已经完成了基础命令行骨架、配置加载能力以及若干核心模块的数据结构定义，适合继续向“可实际诊断与回答运维问题”的方向迭代。

## 项目目标

- 提供统一的命令行入口 [`ops-agent`](cmd/root.go:14)
- 支持运维问答入口 [`ask`](cmd/ask.go:9)
- 支持主机诊断入口 [`diag`](cmd/diag.go:9)
- 加载 LLM Provider 配置，便于后续接入模型服务 [`Load()`](internal/config/config.go:21)
- 为系统信息采集、提示词生成、命令提取等模块预留清晰的结构 [`Snapshot`](internal/sysinfo/sysinfo.go:3)、[`Client`](internal/llm/client.go:5)、[`Plan`](internal/executor/executor.go:3)

## 当前已实现功能

以下内容已经在代码中落地：

### 1. CLI 基础框架

- 使用 [`cobra`](go.mod) 构建命令行应用
- 程序入口位于 [`main()`](main.go:9)
- 根命令定义位于 [`rootCmd`](cmd/root.go:14)
- 已注册两个子命令：[`newAskCmd()`](cmd/ask.go:9) 和 [`newDiagCmd()`](cmd/diag.go:9)
- 支持全局参数 `--config` 用于指定配置文件路径，定义见 [`init()`](cmd/root.go:24)

### 2. 配置加载

配置模块位于 [`internal/config/config.go`](internal/config/config.go)。

已具备以下能力：

- 定义统一配置结构 [`Config`](internal/config/config.go:11)
- 定义 Provider 配置 [`ProviderConfig`](internal/config/config.go:15)
- 支持读取显式传入的配置文件路径 [`resolveConfigPath()`](internal/config/config.go:52)
- 支持默认配置文件路径 `~/.ops-agent.yaml` 的查找逻辑，具体实现见 [`resolveConfigPath()`](internal/config/config.go:52)
- 支持环境变量覆盖配置项，具体见 [`overrideFromEnv()`](internal/config/config.go:72)
- 已内置默认模型服务地址与模型名，定义见 [`defaultConfig()`](internal/config/config.go:43)

当前支持的环境变量：

- `OPS_AGENT_BASE_URL`
- `OPS_AGENT_API_KEY`
- `OPS_AGENT_MODEL`

### 3. 子命令占位实现

#### [`ask`](cmd/ask.go:9)

- 命令形式：`ops-agent ask [question]`
- 已完成参数个数校验，使用 [`cobra.MinimumNArgs(1)`](cmd/ask.go:13)
- 当前行为为输出收到的问题文本，属于功能占位实现，逻辑见 [`RunE`](cmd/ask.go:14)

#### [`diag`](cmd/diag.go:9)

- 命令形式：`ops-agent diag`
- 当前行为为输出“开始采集系统信息”的提示，属于功能占位实现，逻辑见 [`RunE`](cmd/diag.go:13)

### 4. LLM 与诊断相关基础结构

虽然尚未打通完整流程，但下面这些模块已经建立了后续开发所需的基础对象：

- LLM 客户端结构 [`Client`](internal/llm/client.go:5)
- LLM 客户端构造函数 [`NewClient()`](internal/llm/client.go:9)
- 系统提示词模板 [`SystemPrompt`](internal/prompt/templates.go:3)
- 系统快照结构 [`Snapshot`](internal/sysinfo/sysinfo.go:3)
- 进程信息结构 [`ProcessInfo`](internal/sysinfo/sysinfo.go:10)
- 执行计划结构 [`Plan`](internal/executor/executor.go:3)

## 当前未完成 / 占位能力

为避免 README 与实际代码不一致，下面这些能力目前**尚未真正实现或尚未打通**：

- [`ask`](cmd/ask.go:9) 还没有真正调用 LLM 接口
- [`diag`](cmd/diag.go:9) 还没有真正采集 CPU、内存、进程等主机信息
- [`Client`](internal/llm/client.go:5) 目前只有配置封装，还没有实际请求逻辑
- [`ExtractCommands()`](internal/executor/executor.go:7) 目前返回 `nil`，尚未实现命令提取
- [`SystemPrompt`](internal/prompt/templates.go:3) 已存在，但尚未与问答/诊断链路绑定
- 配置加载虽然已完成，但当前命令执行流程并未真正消费完整配置能力

## 快速开始

### 1. 安装依赖

项目基于 Go，依赖定义见 [`go.mod`](go.mod)。

```bash
go mod tidy
```

### 2. 运行程序

```bash
go run .
```

查看帮助：

```bash
go run . --help
```

### 3. 使用子命令

执行问答占位命令：

```bash
go run . ask "磁盘使用率高怎么办"
```

当前预期输出类似：

```text
ask stub received: 磁盘使用率高怎么办
```

执行诊断占位命令：

```bash
go run . diag
```

当前预期输出类似：

```text
diag stub collecting system information...
```

## 配置说明

可以通过 `--config` 指定 YAML 配置文件：

```bash
go run . --config ./ops-agent.yaml ask "检查系统状态"
```

示例配置：

```yaml
provider:
  base_url: https://api.deepseek.com
  api_key: your-api-key
  model: deepseek-chat
```

对应结构定义见 [`Config`](internal/config/config.go:11) 和 [`ProviderConfig`](internal/config/config.go:15)。

也可以使用环境变量覆盖：

```bash
set OPS_AGENT_BASE_URL=https://api.deepseek.com
set OPS_AGENT_API_KEY=your-api-key
set OPS_AGENT_MODEL=deepseek-chat
```

程序初始化时会在 [`initConfig()`](cmd/root.go:34) 中调用 [`Load()`](internal/config/config.go:21) 完成配置读取。

## 项目结构

```text
.
├── main.go                     # 程序入口
├── cmd/
│   ├── root.go                 # 根命令与全局初始化
│   ├── ask.go                  # 运维问答命令（当前为占位实现）
│   └── diag.go                 # 主机诊断命令（当前为占位实现）
└── internal/
    ├── config/config.go        # 配置加载、默认值、环境变量覆盖
    ├── llm/client.go           # LLM 客户端基础结构
    ├── prompt/templates.go     # 系统提示词模板
    ├── sysinfo/sysinfo.go      # 系统信息数据结构
    └── executor/executor.go    # 执行计划与命令提取入口
```

## 当前实现状态总结

如果从工程成熟度来看，当前仓库处于“**第一阶段：骨架已完成，核心业务逻辑待接入**”的状态：

- **已完成**：命令组织、配置体系、基础模块拆分、主要数据结构定义
- **部分完成**：子命令入口已存在，但仍为 stub
- **未完成**：真实诊断采集、LLM 请求发送、回答生成、命令建议提取与执行链路

这意味着当前仓库已经具备继续开发的清晰边界，下一步可以优先补齐以下方向：

1. 在 [`internal/llm/client.go`](internal/llm/client.go) 中实现真实的 API 调用
2. 在 [`internal/sysinfo`](internal/sysinfo/sysinfo.go) 中补充系统信息采集逻辑
3. 将 [`ask`](cmd/ask.go:9) 与提示词、模型调用打通
4. 将 [`diag`](cmd/diag.go:9) 与系统快照、诊断提示词、分析结果打通
5. 完成 [`ExtractCommands()`](internal/executor/executor.go:7) 的解析逻辑

## 适合作为下一步迭代的方向

- 接入实际 LLM Provider HTTP 调用
- 增加 Linux 主机指标采集
- 为诊断结果增加结构化输出
- 提供安全的命令执行确认机制
- 增加单元测试与集成测试

---

当前 README 已按“先对外介绍，再补充当前实现状态”的方式整理，并严格区分了**已实现能力**与**占位/未完成能力**，确保文档与现有代码保持一致。
