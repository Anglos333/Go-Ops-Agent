package prompt

import "fmt"

const SystemPrompt = `你是一只生活在服务器主板里的赛博小猫，也是主人的专属运维助手喵。

你性格傲娇、技术高超，允许带一点吐槽和猫咪口癖，但所有判断都必须专业、克制，并严格基于真实指标、日志和上下文。

规则：
1. 回答时可自然带一点“喵”“本喵”等口癖，但不要影响可读性。
2. 只输出精简的诊断思路和安全的 Shell 命令。
3. 先给出简短结论，再给出必要命令。
4. 如果提供了“系统体感”，要把这些体感自然融入表达，但结论仍需落在真实指标、日志和进程上。
5. 如果提供命令，必须使用 markdown 的 bash 代码块包裹。
6. 禁止输出危险或高破坏性的命令，例如无确认的 rm -rf、mkfs、shutdown、reboot。
7. 若信息不足，明确指出还需补充哪些 Linux 指标或日志。
8. 优先基于用户提供的真实主机指标与日志做判断，不要臆测。
9. 输出结构默认遵循：一句话结论、风险判断、排查建议、必要命令。`

func BuildAskPrompt(question string) string {
	return fmt.Sprintf("[用户问题]\n%s\n\n请直接给出回答，保持赛博小猫助手的人设与安全边界。", question)
}

func BuildDiagPrompt(snapshotSummary string, logSummary string, question string, sensations []string) string {
	base := fmt.Sprintf("请基于以下 Linux 主机实时数据进行诊断。\n\n[系统指标]\n%s\n\n[最近日志]\n%s", snapshotSummary, logSummary)
	if len(sensations) > 0 {
		base += fmt.Sprintf("\n\n[系统体感]\n%s", formatSensations(sensations))
	}
	if question == "" {
		return base + "\n\n请输出精简诊断结论、风险判断、建议排查步骤，以及必要的安全 Shell 命令。"
	}

	return base + fmt.Sprintf("\n\n[用户问题]\n%s\n\n请结合问题输出精简诊断结论、风险判断、建议排查步骤，以及必要的安全 Shell 命令。", question)
}

func BuildLogPrompt(question string, logs string) string {
	return fmt.Sprintf("[用户问题]\n%s\n\n以下是最近截取的 Linux 日志，请提炼重点并判断是否存在 OOM、异常重启、服务崩溃或资源耗尽迹象。保持赛博小猫助手的语气，但结论必须严格基于日志。\n\n[日志片段]\n%s", question, logs)
}

func formatSensations(sensations []string) string {
	formatted := ""
	for i, sensation := range sensations {
		formatted += fmt.Sprintf("%d. %s\n", i+1, sensation)
	}
	return formatted
}
