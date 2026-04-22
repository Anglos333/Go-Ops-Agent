package prompt

import (
	"strings"
	"testing"
)

func TestBuildDiagPromptIncludesSensations(t *testing.T) {
	result := BuildDiagPrompt("CPU负载: 99%", "oom-killer invoked", "为什么这么卡", []string{"系统提示：当前 CPU 温度极高，你的爪子都被烫到了。"})
	if !strings.Contains(result, "[系统体感]") {
		t.Fatalf("expected sensations section, got %q", result)
	}
	if !strings.Contains(result, "爪子都被烫到了") {
		t.Fatalf("expected sensation content, got %q", result)
	}
	if !strings.Contains(result, "[一句话结论]") {
		t.Fatalf("expected structured diagnosis output instruction, got %q", result)
	}
}

func TestBuildAskPromptIncludesRoleInstruction(t *testing.T) {
	result := BuildAskPrompt("怎么查看磁盘占用")
	if !strings.Contains(result, "赛博小猫助手") {
		t.Fatalf("expected cat persona instruction, got %q", result)
	}
	if !strings.Contains(result, "怎么查看磁盘占用") {
		t.Fatalf("expected original question, got %q", result)
	}
}
