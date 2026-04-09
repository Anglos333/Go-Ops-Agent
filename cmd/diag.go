package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/llm"
	"go-ops-agent/internal/prompt"
	"go-ops-agent/internal/sysinfo"
)

func newDiagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diag [question]",
		Short: "Collect host diagnostics and request AI analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			client, err := llm.NewClient(cfg.Provider)
			if err != nil {
				return err
			}

			spinner, _ := pterm.DefaultSpinner.Start("正在采集系统指标与日志")
			snapshot, err := sysinfo.CollectSnapshot()
			if err != nil {
				spinner.Fail("系统指标采集失败")
				return err
			}
			logs, err := sysinfo.ReadRecentLogs(100)
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

			resp, err := client.Chat(context.Background(), prompt.BuildDiagPrompt(snapshot.Summary(), logPayload, question))
			if err != nil {
				spinner.Fail("AI 诊断失败")
				return err
			}
			spinner.Success("诊断完成")

			pterm.DefaultSection.Println("系统快照")
			fmt.Fprintln(cmd.OutOrStdout(), snapshot.Summary())
			pterm.DefaultBox.WithTitle("AI 诊断").Println(resp)

			commands := executor.ExtractCommands(resp)
			approved, err := executor.ConfirmExecution(cmd.InOrStdin(), cmd.OutOrStdout(), commands)
			if err != nil {
				return err
			}
			if !approved {
				pterm.Info.Println("未执行任何系统命令")
				return nil
			}
			return executor.RunCommands("bash", cmd.OutOrStdout(), commands)
		},
	}

	return cmd
}
