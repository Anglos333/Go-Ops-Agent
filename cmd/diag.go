package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/prompt"
	"go-ops-agent/internal/sysinfo"
	"go-ops-agent/internal/ui"
)

func newDiagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diag [question]",
		Short: "Collect host diagnostics and request AI analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cfgFile)
			if err != nil {
				return err
			}
			client, err := newChatClient(cfg.Provider)
			if err != nil {
				return err
			}

			spinner, _ := ui.StartCatSpinner("正在竖起耳朵采集系统指标与日志")
			snapshot, err := collectSnapshot()
			if err != nil {
				spinner.Fail("系统指标采集失败")
				return err
			}
			logs, err := readRecentLogs(100)
			if err != nil {
				spinner.Fail("日志采集失败")
				return err
			}

			question := strings.Join(args, " ")
			logPayload := logs
			if strings.Contains(strings.ToLower(question), "oom") || strings.Contains(question, "内存溢出") {
				if oom := sysinfo.FilterOOMLogs(logs); strings.TrimSpace(oom) != "" {
					logPayload = oom
				}
			}

			sensations := buildSensations(snapshot, logPayload)
			resp, err := client.Chat(context.Background(), prompt.BuildDiagPrompt(snapshot.Summary(), logPayload, question, sensations))
			if err != nil {
				spinner.Fail("AI 诊断失败")
				return err
			}
			spinner.Success("小猫已经完成诊断巡视")

			pterm.DefaultSection.Println("系统快照")
			fmt.Fprintln(cmd.OutOrStdout(), snapshot.Summary())
			ui.PrintCatReply("小猫诊断", resp)

			commands := executor.ExtractCommands(resp)
			plan, err := executor.ReviewCommands(commands)
			if err != nil {
				return fmt.Errorf("AI 返回了未通过安全审查的命令: %w", err)
			}
			if len(plan.Rejected) > 0 {
				reasons := make([]string, 0, len(plan.Rejected))
				for _, rejected := range plan.Rejected {
					reasons = append(reasons, fmt.Sprintf("- %s", rejected.Source))
				}
				ui.PrintCatReply("小猫安检", fmt.Sprintf("本喵刚刚从诊断建议里删掉了 %d 条违规命令，只保留安全部分继续给主人确认喵：\n%s", len(plan.Rejected), strings.Join(reasons, "\n")))
			}
			if len(plan.Commands) == 0 {
				ui.PrintCatReply("小猫提醒", "诊断建议里的候选命令都被本喵拦住了，这次不会执行任何系统命令喵。")
				return nil
			}

			approved, err := confirmExecution(cmd.InOrStdin(), cmd.OutOrStdout(), plan)
			if err != nil {
				return err
			}
			if !approved {
				ui.PrintCatReply("小猫提醒", "本喵没有继续挥爪，系统命令一条都没执行喵。")
				return nil
			}
			return executePlan(cmd.OutOrStdout(), plan)
		},
	}

	return cmd
}

func buildSensations(snapshot sysinfo.Snapshot, logs string) []string {
	sensations := make([]string, 0, 3)
	if snapshot.CPUUsagePercent >= 90 {
		sensations = append(sensations, "系统提示：当前 CPU 温度极高，你的爪子都被烫到了。")
	}
	if snapshot.MemoryTotalMB > 0 {
		usedRatio := float64(snapshot.MemoryUsedMB) / float64(snapshot.MemoryTotalMB)
		if usedRatio >= 0.9 {
			sensations = append(sensations, "系统提示：内存快被挤满了，你的猫窝越来越窄，已经有点炸毛。")
		}
	}
	if strings.TrimSpace(sysinfo.FilterOOMLogs(logs)) != "" {
		sensations = append(sensations, "系统提示：内存见底了，你感觉非常饥饿。")
	}
	return sensations
}
