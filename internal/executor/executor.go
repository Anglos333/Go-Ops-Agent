package executor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

type Plan struct {
	Commands []string
}

var bashBlockPattern = regexp.MustCompile("(?s)```bash\\s*(.*?)```")

func ExtractCommands(text string) []string {
	matches := bashBlockPattern.FindAllStringSubmatch(text, -1)
	commands := make([]string, 0, len(matches))
	for _, match := range matches {
		body := strings.TrimSpace(match[1])
		if body != "" {
			commands = append(commands, body)
		}
	}
	return commands
}

func ConfirmExecution(in io.Reader, out io.Writer, commands []string) (bool, error) {
	if len(commands) == 0 {
		return false, nil
	}

	_, err := fmt.Fprintln(out, "警告：即将执行以下系统级命令，是否继续？(Y/N)")
	if err != nil {
		return false, err
	}
	for _, cmd := range commands {
		if _, err := fmt.Fprintf(out, "\n%s\n", cmd); err != nil {
			return false, err
		}
	}

	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToUpper(answer))
	return answer == "Y" || answer == "YES", nil
}

func RunCommands(shell string, out io.Writer, commands []string) error {
	for _, command := range commands {
		cmd := exec.Command(shell, "-lc", command)
		cmd.Stdout = out
		cmd.Stderr = out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("execute command failed: %w", err)
		}
	}
	return nil
}
