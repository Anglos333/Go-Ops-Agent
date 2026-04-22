package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestExtractCommands(t *testing.T) {
	text := "before\n```bash\nls -l\n```\ntext\n```bash\nps aux\n```"
	got := ExtractCommands(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 command blocks, got %d", len(got))
	}
	if got[0] != "ls -l" || got[1] != "ps aux" {
		t.Fatalf("unexpected commands: %#v", got)
	}
}

func TestReviewCommandsAllowsSimpleCommands(t *testing.T) {
	plan, err := ReviewCommands([]string{"ls -l\nps aux"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(plan.Commands))
	}
	if strings.Join(plan.Commands[0].Argv, " ") != "ls -l" {
		t.Fatalf("unexpected argv: %#v", plan.Commands[0].Argv)
	}
}

func TestReviewCommandsRejectsPipeline(t *testing.T) {
	_, err := ReviewCommands([]string{"ps aux | grep nginx"})
	if err == nil || !strings.Contains(err.Error(), "pipelines") {
		t.Fatalf("expected pipeline rejection, got %v", err)
	}
}

func TestReviewCommandsRejectsCommandSubstitution(t *testing.T) {
	_, err := ReviewCommands([]string{"ls $(pwd)"})
	if err == nil || !strings.Contains(err.Error(), "substitution") {
		t.Fatalf("expected substitution rejection, got %v", err)
	}
}

func TestReviewCommandsRejectsNonAllowlistedCommand(t *testing.T) {
	_, err := ReviewCommands([]string{"rm -rf /tmp/demo"})
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist rejection, got %v", err)
	}
}

func TestReviewCommandsRestrictsFindActions(t *testing.T) {
	_, err := ReviewCommands([]string{"find /tmp -name *.log -exec rm {} ;"})
	if err == nil {
		t.Fatal("expected find action rejection")
	}
}

func TestConfirmExecutionUsesReviewedArgv(t *testing.T) {
	plan := &Plan{Commands: []CommandPlan{{Argv: []string{"ls", "-l"}}}}
	input := strings.NewReader("Y\n")
	var out bytes.Buffer

	approved, err := ConfirmExecution(input, &out, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatal("expected approval")
	}
	if !strings.Contains(out.String(), "ls -l") {
		t.Fatalf("expected reviewed argv in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "危险的 Bash 魔法") {
		t.Fatalf("expected cat-styled confirmation message, got %q", out.String())
	}
	if !strings.Contains(out.String(), "风险等级: LOW") {
		t.Fatalf("expected low risk level, got %q", out.String())
	}
}

func TestConfirmExecutionRequiresSecondConfirmationForMediumRisk(t *testing.T) {
	plan := &Plan{Commands: []CommandPlan{{Argv: []string{"find", "/tmp", "-name", "*.log"}}, {Argv: []string{"ls", "-l"}}, {Argv: []string{"ps", "aux"}}}, Risk: RiskMedium}
	input := strings.NewReader("Y\nEXECUTE\n")
	var out bytes.Buffer

	approved, err := ConfirmExecution(input, &out, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatal("expected approval after second confirmation")
	}
	if !strings.Contains(out.String(), "二次确认") {
		t.Fatalf("expected second confirmation prompt, got %q", out.String())
	}
}

func TestConfirmExecutionRejectsWhenSecondConfirmationMissing(t *testing.T) {
	plan := &Plan{Commands: []CommandPlan{{Argv: []string{"find", "/", "-name", "*.log"}}}, Risk: RiskHigh}
	input := strings.NewReader("Y\nNOPE\n")
	var out bytes.Buffer

	approved, err := ConfirmExecution(input, &out, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejection when second confirmation mismatches")
	}
}

func TestReviewCommandsAssignsHighRisk(t *testing.T) {
	plan, err := ReviewCommands([]string{"find / -name *.log"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Risk != RiskHigh {
		t.Fatalf("expected high risk, got %s", plan.Risk)
	}
}
