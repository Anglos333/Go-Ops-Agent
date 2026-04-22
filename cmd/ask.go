package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/llm"
	"go-ops-agent/internal/prompt"
	"go-ops-agent/internal/ui"
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

			spinner, _ := ui.StartCatSpinner("本喵正在分析主人的问题")
			resp, err := client.Chat(context.Background(), prompt.BuildAskPrompt(strings.Join(args, " ")))
			if err != nil {
				spinner.Fail("AI 请求失败")
				return err
			}
			spinner.Success("小猫助手已经叼回结果")

			ui.PrintCatReply("小猫回复", resp)

			commands := executor.ExtractCommands(resp)
			plan, err := executor.ReviewCommands(commands)
			if err != nil {
				return fmt.Errorf("AI 返回了未通过安全审查的命令: %w", err)
			}

			approved, err := executor.ConfirmExecution(cmd.InOrStdin(), cmd.OutOrStdout(), plan)
			if err != nil {
				return err
			}
			if !approved {
				ui.PrintCatReply("小猫提醒", "本喵把危险爪印收回去了，没有执行任何系统命令喵。")
				return nil
			}

			ui.PrintCatReply("小猫行动", "本喵要开始挥爪执行已经审查通过的命令了喵。")
			return executor.RunPlan(os.Stdout, plan)
		},
	}

	return cmd
}
