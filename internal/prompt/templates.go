package prompt

import "fmt"

const SystemPrompt = `你是一个资深的 Linux SRE 专家。

规则：
1. 只输出精简的诊断思路和安全的 Shell 命令。
2. 先给出简短结论，再给出必要命令。
3. 如果提供命令，必须使用 markdown 的 bash 代码块包裹。
4. 禁止输出危险或高破坏性的命令，例如无确认的 rm -rf、mkfs、shutdown、reboot。
5. 若信息不足，明确指出还需补充哪些 Linux 指标或日志。
6. 优先基于用户提供的真实主机指标与日志做判断，不要臆测。`

func BuildAskPrompt(question string) string {
	return question
}

func BuildDiagPrompt(snapshotSummary string, logSummary string, question string) string {
	base := fmt.Sprintf("请基于以下 Linux 主机实时数据进行诊断。\n\n[系统指标]\n%s\n\n[最近日志]\n%s", snapshotSummary, logSummary)
	if question == "" {
		return base + "\n\n请输出精简诊断结论、风险判断、建议排查步骤，以及必要的安全 Shell 命令。"
	}

	return base + fmt.Sprintf("\n\n[用户问题]\n%s\n\n请结合问题输出精简诊断结论、风险判断、建议排查步骤，以及必要的安全 Shell 命令。", question)
}

func BuildLogPrompt(question string, logs string) string {
	return fmt.Sprintf("用户问题：%s\n\n以下是最近截取的 Linux 日志，请提炼重点并判断是否存在 OOM、异常重启、服务崩溃或资源耗尽迹象。\n\n[日志片段]\n%s", question, logs)
}
