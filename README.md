# go-ops-agent

一个 Linux 优先的终端运维助手。当前版本支持：

- 使用 [`github.com/sashabaranov/go-openai`](go.mod) 连接兼容 OpenAI Chat Completions 的模型服务。
- 使用 [`github.com/shirou/gopsutil/v3`](go.mod) 采集 CPU、内存和 Top 5 进程信息。
- 使用 [`github.com/pterm/pterm`](go.mod) 提供 Spinner、彩色提示和终端渲染。
- [`ops-agent ask`](cmd/ask.go) 调用兼容 OpenAI Chat Completions 的模型服务进行运维问答，并在发现 ```bash``` 代码块时要求用户确认后执行。
- [`ops-agent diag`](cmd/diag.go) 自动采集主机指标和最近 100 行日志，请 AI 基于真实数据做诊断。
- 统一使用带有“赛博小猫”人格的系统提示词，并在终端中展示小猫风格的 spinner、ASCII 立绘与回复气泡。

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

- 模型若返回 `bash` 代码块，将由 [`ExtractCommands()`](internal/executor/executor.go:56) 提取。
- 所有候选命令会先经过 [`ReviewCommands()`](internal/executor/executor.go:83) 的白名单与结构审查。
- 执行前会显示带人格化语气的确认提示。
- 只有用户明确输入 `Y` 或 `YES` 才会逐条执行已审查通过的命令并回显输出。

`go-ops-agent` 是一个面向终端场景的运维辅助 CLI，目标是把本机诊断信息采集、LLM 分析与后续执行建议串起来，形成一个轻量的 AI Ops Assistant。

当前仓库已经完成了基础命令行骨架、配置加载能力、真实的模型调用链路、系统信息采集、命令审查执行，以及带人格化终端表现的问答/诊断流程。

## 项目目标

- 提供统一的命令行入口 [`ops-agent`](cmd/root.go:14)
- 支持运维问答入口 [`ask`](cmd/ask.go:18)
- 支持主机诊断入口 [`diag`](cmd/diag.go:19)
- 加载 LLM Provider 配置，便于后续接入模型服务 [`Load()`](internal/config/config.go:21)
- 提供系统信息采集、提示词生成、命令提取与审查、终端展示等模块化能力 [`Snapshot`](internal/sysinfo/sysinfo.go:18)、[`Client`](internal/llm/client.go:14)、[`Plan`](internal/executor/executor.go:14)

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

### 3. 问答与诊断命令

#### [`ask`](cmd/ask.go:18)

- 命令形式：`ops-agent ask [question]`
- 已完成参数个数校验，使用 [`cobra.MinimumNArgs(1)`](cmd/ask.go:22)
- 会加载配置并创建 LLM 客户端，逻辑见 [`RunE`](cmd/ask.go:23)
- 会通过 [`BuildAskPrompt()`](internal/prompt/templates.go:18) 组装问答提示词，并调用 [`(*Client).Chat()`](internal/llm/client.go:35)
- 使用 [`internal/ui/cat.go`](internal/ui/cat.go) 提供小猫 spinner 与回复展示
- 若模型返回 `bash` 代码块，会进入命令提取、审查、确认与执行链路

#### [`diag`](cmd/diag.go:19)

- 命令形式：`ops-agent diag [question]`
- 会采集 CPU、内存、Top 5 进程以及最近日志，逻辑见 [`RunE`](cmd/diag.go:23)
- 会在高 CPU、高内存压力或存在 OOM 日志时注入“系统体感”，逻辑见 [`buildSensations()`](cmd/diag.go:86)
- 会通过 [`BuildDiagPrompt()`](internal/prompt/templates.go:22) 组装诊断提示词并请求模型分析
- 诊断结果同样会进入命令提取、审查、确认与执行链路

### 4. LLM、Prompt 与诊断能力

下面这些模块已经完成并参与实际流程：

- LLM 客户端结构 [`Client`](internal/llm/client.go:14)
- LLM 客户端构造函数 [`NewClient()`](internal/llm/client.go:19)
- 真实聊天请求逻辑 [`(*Client).Chat()`](internal/llm/client.go:35)
- 赛博小猫系统提示词 [`SystemPrompt`](internal/prompt/templates.go:5)
- 问答、诊断、日志提示词构造函数 [`BuildAskPrompt()`](internal/prompt/templates.go:18)、[`BuildDiagPrompt()`](internal/prompt/templates.go:22)、[`BuildLogPrompt()`](internal/prompt/templates.go:33)
- 系统快照结构 [`Snapshot`](internal/sysinfo/sysinfo.go:18)
- 进程信息结构 [`ProcessInfo`](internal/sysinfo/sysinfo.go:26)
- 执行计划结构 [`Plan`](internal/executor/executor.go:14)

### 5. 系统信息采集与命令执行

以下能力已经具备：

- CPU、内存、Top 5 进程采集 [`CollectSnapshot()`](internal/sysinfo/sysinfo.go:33)
- 日志读取与 OOM 过滤 [`ReadRecentLogs()`](internal/sysinfo/sysinfo.go:94)、[`FilterOOMLogs()`](internal/sysinfo/sysinfo.go:113)
- `bash` 代码块命令提取 [`ExtractCommands()`](internal/executor/executor.go:56)
- 命令白名单与结构安全审查 [`ReviewCommands()`](internal/executor/executor.go:83)
- 执行前二次确认 [`ConfirmExecution()`](internal/executor/executor.go:272)

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

执行问答命令：

```bash
go run . ask "磁盘使用率高怎么办"
```

当前预期输出为小猫风格的 spinner 与 AI 回复，例如：

```text
 /[?25l
( o.o )
 > ^ <   赛博运维猫在线值守喵

主人，报告在这里喵：

先看磁盘占用最大的目录和文件，再确认是否是日志、缓存或异常增长的业务数据。
```

执行诊断命令：

```bash
go run . diag
```

当前预期输出会先显示系统快照，再显示 AI 诊断结果，例如：

```text
系统快照
CPU负载: 92.10%
内存: 已用 7560 MB / 总计 8192 MB / 可用 420 MB
Top 5 进程:
- PID=1234 Name=java CPU=88.12% RSS=2048MB

主人，报告在这里喵：

喵呜，当前主机已经明显过热，Java 进程正在持续吞噬 CPU，需要优先排查这个进程的线程与 GC 状态。
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

如果从工程成熟度来看，当前仓库已经从“骨架阶段”进入“**基础能力可用、可继续增强体验与安全策略**”的状态：

- **已完成**：命令组织、配置体系、真实 LLM 请求、系统指标采集、诊断链路、命令提取与安全审查、终端人格化展示
- **部分完成**：整体功能已可用，但文档、测试覆盖率与更多诊断规则仍有继续完善空间
- **未完成**：更细粒度的结构化诊断输出、更丰富的 Linux 指标采集、更完善的集成测试

这意味着当前仓库已经具备继续向“更强诊断能力”和“更稳定执行体验”演进的基础，下一步可以优先考虑：

1. 扩展 [`internal/sysinfo/sysinfo.go`](internal/sysinfo/sysinfo.go) 采集磁盘、负载、网络等更多系统指标
2. 为 [`cmd/diag.go`](cmd/diag.go) 增加更细粒度的异常识别与结构化输出
3. 为 [`internal/executor/executor.go`](internal/executor/executor.go) 增加更多命令白名单规则和测试覆盖
4. 完善 [`README.md`](README.md) 的截图、示例输出与配置说明
5. 增加集成测试与模拟 LLM Provider 的测试场景

## 适合作为下一步迭代的方向

- 接入实际 LLM Provider HTTP 调用
- 增加 Linux 主机指标采集
- 为诊断结果增加结构化输出
- 提供安全的命令执行确认机制
- 增加单元测试与集成测试

---

当前 README 已根据现有实现重新整理，重点反映真实可用能力，并把后续工作聚焦到增强项而非历史占位项。
