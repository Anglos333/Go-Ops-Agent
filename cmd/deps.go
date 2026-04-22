package cmd

import (
	"context"
	"io"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/llm"
	"go-ops-agent/internal/sysinfo"
)

type chatClient interface {
	Chat(context.Context, string) (string, error)
}

var (
	loadConfig       = config.Load
	newChatClient    = func(cfg config.ProviderConfig) (chatClient, error) { return llm.NewClient(cfg) }
	collectSnapshot  = sysinfo.CollectSnapshot
	readRecentLogs   = sysinfo.ReadRecentLogs
	confirmExecution = executor.ConfirmExecution
	runPlan          = executor.RunPlan
)

func resetCommandDeps() {
	loadConfig = config.Load
	newChatClient = func(cfg config.ProviderConfig) (chatClient, error) { return llm.NewClient(cfg) }
	collectSnapshot = sysinfo.CollectSnapshot
	readRecentLogs = sysinfo.ReadRecentLogs
	confirmExecution = executor.ConfirmExecution
	runPlan = executor.RunPlan
}

func executePlan(out io.Writer, plan *executor.Plan) error {
	return runPlan(out, plan)
}
