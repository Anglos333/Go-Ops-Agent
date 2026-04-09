package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/llm"
	"go-ops-agent/internal/prompt"
)

func newAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the AI assistant an operations question",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			client, err := llm.NewClient(cfg.Provider)
			if err != nil {
				return err
			}

			spinner, _ := pterm.DefaultSpinner.Start("AI 正在分析问题")
			resp, err := client.Chat(context.Background(), prompt.BuildAskPrompt(strings.Join(args, " ")))
			if err != nil {
				spinner.Fail("AI 请求失败")
				return err
			}
			spinner.Success("AI 已返回结果")

			pterm.DefaultBox.WithTitle("AI 回复").Println(resp)

			commands := executor.ExtractCommands(resp)
			approved, err := executor.ConfirmExecution(cmd.InOrStdin(), cmd.OutOrStdout(), commands)
			if err != nil {
				return err
			}
			if !approved {
				pterm.Info.Println("未执行任何系统命令")
				return nil
			}

			pterm.Success.Println("开始执行 AI 建议的命令")
			return executor.RunCommands("bash", os.Stdout, commands)
		},
	}

	return cmd
}
