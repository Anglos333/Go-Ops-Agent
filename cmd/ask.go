package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/prompt"
	"go-ops-agent/internal/ui"
)

func newAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the AI assistant an operations question",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cfgFile)
			if err != nil {
				return err
			}

			client, err := newChatClient(cfg.Provider)
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
			if len(plan.Rejected) > 0 {
				reasons := make([]string, 0, len(plan.Rejected))
				for _, rejected := range plan.Rejected {
					reasons = append(reasons, fmt.Sprintf("- %s", rejected.Source))
				}
				ui.PrintCatReply("小猫安检", fmt.Sprintf("本喵刚刚拦下了 %d 条不安全命令，已经自动没收，不会触发冷冰冰的系统报错喵：\n%s", len(plan.Rejected), strings.Join(reasons, "\n")))
			}
			if len(plan.Commands) == 0 {
				ui.PrintCatReply("小猫提醒", "这次候选命令已经被本喵全部拦下，没有留下任何可执行命令喵。")
				return nil
			}

			approved, err := confirmExecution(cmd.InOrStdin(), cmd.OutOrStdout(), plan)
			if err != nil {
				return err
			}
			if !approved {
				ui.PrintCatReply("小猫提醒", "本喵把危险爪印收回去了，没有执行任何系统命令喵。")
				return nil
			}

			ui.PrintCatReply("小猫行动", "本喵要开始挥爪执行已经审查通过的命令了喵。")
			return executePlan(os.Stdout, plan)
		},
	}

	return cmd
}
