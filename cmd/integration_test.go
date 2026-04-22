package cmd

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/executor"
	"go-ops-agent/internal/sysinfo"
)

type fakeChatClient struct {
	resp       string
	lastPrompt string
}

func (f *fakeChatClient) Chat(_ context.Context, prompt string) (string, error) {
	f.lastPrompt = prompt
	return f.resp, nil
}

func TestAskCommandIntegration(t *testing.T) {
	resetCommandDeps()
	t.Cleanup(resetCommandDeps)

	fakeClient := &fakeChatClient{resp: "[一句话结论]\n喵\n\n[必要命令]\n```bash\nls -l\n```"}
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Provider: config.ProviderConfig{APIKey: "test", Model: "fake"}}, nil
	}
	newChatClient = func(config.ProviderConfig) (chatClient, error) { return fakeClient, nil }
	confirmExecution = func(_ io.Reader, _ io.Writer, plan *executor.Plan) (bool, error) {
		if len(plan.Commands) != 1 || strings.Join(plan.Commands[0].Argv, " ") != "ls -l" {
			t.Fatalf("unexpected plan: %#v", plan)
		}
		return false, nil
	}
	runPlan = func(io.Writer, *executor.Plan) error {
		t.Fatal("runPlan should not be called when approval is false")
		return nil
	}

	cmd := newAskCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"检查", "磁盘"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fakeClient.lastPrompt, "检查 磁盘") {
		t.Fatalf("expected prompt to include question, got %q", fakeClient.lastPrompt)
	}
}

func TestDiagCommandIntegration(t *testing.T) {
	resetCommandDeps()
	t.Cleanup(resetCommandDeps)

	fakeClient := &fakeChatClient{resp: "[一句话结论]\n喵呜\n\n[必要命令]\n暂无"}
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Provider: config.ProviderConfig{APIKey: "test", Model: "fake"}}, nil
	}
	newChatClient = func(config.ProviderConfig) (chatClient, error) { return fakeClient, nil }
	collectSnapshot = func() (sysinfo.Snapshot, error) {
		return sysinfo.Snapshot{CPUUsagePercent: 95, MemoryUsedMB: 9000, MemoryTotalMB: 10000, MemoryFreeMB: 500}, nil
	}
	readRecentLogs = func(int) (string, error) {
		return "kernel: Out of memory: Killed process 1234 (java)", nil
	}
	confirmExecution = func(_ io.Reader, _ io.Writer, plan *executor.Plan) (bool, error) {
		if len(plan.Commands) != 0 {
			t.Fatalf("expected no commands, got %#v", plan)
		}
		return false, nil
	}

	cmd := newDiagCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"为什么机器卡住了"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := []string{"[系统体感]", "爪子都被烫到了", "非常饥饿", "为什么机器卡住了"}
	for _, check := range checks {
		if !strings.Contains(fakeClient.lastPrompt, check) {
			t.Fatalf("expected prompt to contain %q, got %q", check, fakeClient.lastPrompt)
		}
	}
}

func TestAskCommandIntegrationGracefullyHandlesRejectedCommands(t *testing.T) {
	resetCommandDeps()
	t.Cleanup(resetCommandDeps)

	fakeClient := &fakeChatClient{resp: "[一句话结论]\n喵\n\n[必要命令]\n```bash\nfind . -type f -size +100M -exec du -h {} \\;\nls -lh\n```"}
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Provider: config.ProviderConfig{APIKey: "test", Model: "fake"}}, nil
	}
	newChatClient = func(config.ProviderConfig) (chatClient, error) { return fakeClient, nil }
	confirmExecution = func(_ io.Reader, _ io.Writer, plan *executor.Plan) (bool, error) {
		if len(plan.Commands) != 1 || strings.Join(plan.Commands[0].Argv, " ") != "ls -lh" {
			t.Fatalf("unexpected approved plan: %#v", plan)
		}
		if len(plan.Rejected) != 1 {
			t.Fatalf("expected rejected command, got %#v", plan.Rejected)
		}
		return false, nil
	}

	cmd := newAskCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"找出大文件"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiagCommandIntegrationReturnsNilWhenAllCommandsRejected(t *testing.T) {
	resetCommandDeps()
	t.Cleanup(resetCommandDeps)

	fakeClient := &fakeChatClient{resp: "[一句话结论]\n喵呜\n\n[必要命令]\n```bash\nfind . -type f -size +100M -exec du -h {} \\;\n```"}
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Provider: config.ProviderConfig{APIKey: "test", Model: "fake"}}, nil
	}
	newChatClient = func(config.ProviderConfig) (chatClient, error) { return fakeClient, nil }
	collectSnapshot = func() (sysinfo.Snapshot, error) {
		return sysinfo.Snapshot{}, nil
	}
	readRecentLogs = func(int) (string, error) {
		return "", nil
	}
	confirmExecution = func(_ io.Reader, _ io.Writer, plan *executor.Plan) (bool, error) {
		t.Fatalf("confirmExecution should not be called when all commands are rejected: %#v", plan)
		return false, nil
	}

	cmd := newDiagCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
