package executor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type Plan struct {
	Commands []CommandPlan
}

type CommandPlan struct {
	Source string
	Argv   []string
}

type ReviewOptions struct {
	AllowedCommands map[string]CommandRule
}

type CommandRule struct {
	AllowAnyArgs bool
	Validator    func([]string) error
}

type ReviewError struct {
	Command string
	Reason  string
}

func (e *ReviewError) Error() string {
	return fmt.Sprintf("command %q rejected: %s", e.Command, e.Reason)
}

var parser = syntax.NewParser()

var defaultReviewOptions = ReviewOptions{
	AllowedCommands: map[string]CommandRule{
		"cat":  {AllowAnyArgs: true},
		"grep": {AllowAnyArgs: true},
		"ls":   {AllowAnyArgs: true},
		"stat": {AllowAnyArgs: true},
		"ps":   {AllowAnyArgs: true},
		"df":   {AllowAnyArgs: true},
		"du":   {AllowAnyArgs: true},
		"find": {Validator: validateFindArgs},
	},
}

func ExtractCommands(text string) []string {
	const fence = "```bash"
	commands := make([]string, 0)
	remaining := text

	for {
		start := strings.Index(remaining, fence)
		if start == -1 {
			break
		}
		remaining = remaining[start+len(fence):]

		end := strings.Index(remaining, "```")
		if end == -1 {
			break
		}

		body := strings.TrimSpace(remaining[:end])
		if body != "" {
			commands = append(commands, body)
		}
		remaining = remaining[end+3:]
	}

	return commands
}

func ReviewCommands(commands []string) (*Plan, error) {
	return ReviewCommandsWithOptions(commands, defaultReviewOptions)
}

func ReviewCommandsWithOptions(commands []string, opts ReviewOptions) (*Plan, error) {
	plan := &Plan{Commands: make([]CommandPlan, 0, len(commands))}
	for _, command := range commands {
		parsed, err := reviewCommand(command, opts)
		if err != nil {
			return nil, err
		}
		plan.Commands = append(plan.Commands, parsed...)
	}
	return plan, nil
}

func reviewCommand(command string, opts ReviewOptions) ([]CommandPlan, error) {
	file, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, &ReviewError{Command: command, Reason: fmt.Sprintf("shell parse failed: %v", err)}
	}

	plans := make([]CommandPlan, 0, len(file.Stmts))
	for _, stmt := range file.Stmts {
		if err := rejectStatementStructure(command, stmt); err != nil {
			return nil, err
		}

		call, ok := stmt.Cmd.(*syntax.CallExpr)
		if !ok {
			return nil, &ReviewError{Command: command, Reason: "only simple command invocations are allowed"}
		}

		argv, err := literalArgs(call)
		if err != nil {
			return nil, &ReviewError{Command: command, Reason: err.Error()}
		}
		if len(argv) == 0 {
			return nil, &ReviewError{Command: command, Reason: "empty command is not allowed"}
		}

		if err := enforceCommandPolicy(command, argv, opts); err != nil {
			return nil, err
		}

		plans = append(plans, CommandPlan{Source: command, Argv: argv})
	}
	if len(plans) == 0 {
		return nil, &ReviewError{Command: command, Reason: "no executable command found"}
	}
	return plans, nil
}

func rejectStatementStructure(command string, stmt *syntax.Stmt) error {
	if stmt.Background {
		return &ReviewError{Command: command, Reason: "background execution is not allowed"}
	}
	if stmt.Coprocess {
		return &ReviewError{Command: command, Reason: "coprocess execution is not allowed"}
	}
	if len(stmt.Redirs) > 0 {
		return &ReviewError{Command: command, Reason: "redirection is not allowed"}
	}

	var structureErr error
	syntax.Walk(stmt.Cmd, func(node syntax.Node) bool {
		if structureErr != nil || node == nil {
			return false
		}

		switch node.(type) {
		case *syntax.BinaryCmd:
			structureErr = &ReviewError{Command: command, Reason: "pipelines and boolean command chains are not allowed"}
			return false
		case *syntax.Block:
			structureErr = &ReviewError{Command: command, Reason: "command blocks are not allowed"}
			return false
		case *syntax.Subshell:
			structureErr = &ReviewError{Command: command, Reason: "subshell is not allowed"}
			return false
		case *syntax.ForClause, *syntax.WhileClause, *syntax.IfClause, *syntax.CaseClause, *syntax.FuncDecl, *syntax.DeclClause, *syntax.TestClause:
			structureErr = &ReviewError{Command: command, Reason: "shell control structures are not allowed"}
			return false
		}

		return true
	})

	return structureErr
}

func literalArgs(call *syntax.CallExpr) ([]string, error) {
	argv := make([]string, 0, len(call.Args))
	for _, word := range call.Args {
		arg, err := literalWord(word)
		if err != nil {
			return nil, err
		}
		argv = append(argv, arg)
	}
	return argv, nil
}

func literalWord(word *syntax.Word) (string, error) {
	if len(word.Parts) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	for _, part := range word.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			buf.WriteString(x.Value)
		case *syntax.SglQuoted:
			buf.WriteString(x.Value)
		case *syntax.DblQuoted:
			for _, subPart := range x.Parts {
				subLit, ok := subPart.(*syntax.Lit)
				if !ok {
					return "", fmt.Errorf("command substitution and expansions are not allowed")
				}
				buf.WriteString(subLit.Value)
			}
		default:
			return "", fmt.Errorf("command substitution and expansions are not allowed")
		}
	}

	return buf.String(), nil
}

func enforceCommandPolicy(command string, argv []string, opts ReviewOptions) error {
	rule, ok := opts.AllowedCommands[argv[0]]
	if !ok {
		return &ReviewError{Command: command, Reason: fmt.Sprintf("command %q is not in the allowlist", argv[0])}
	}
	if rule.AllowAnyArgs {
		return nil
	}
	if rule.Validator != nil {
		if err := rule.Validator(argv); err != nil {
			return &ReviewError{Command: command, Reason: err.Error()}
		}
	}
	return nil
}

func validateFindArgs(argv []string) error {
	if len(argv) < 2 {
		return fmt.Errorf("find requires at least a path argument")
	}

	allowedFlags := map[string]bool{
		"-maxdepth": true,
		"-mindepth": true,
		"-name":     true,
		"-iname":    true,
		"-type":     true,
		"-mtime":    true,
		"-size":     true,
		"-user":     true,
		"-group":    true,
		"-path":     true,
		"-ipath":    true,
		"-print":    true,
	}

	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		if arg == "-exec" || arg == "-delete" || arg == "-ok" || arg == "-fprintf" || arg == "-fprint" || arg == "-fls" || arg == "-ls" {
			return fmt.Errorf("find action %q is not allowed", arg)
		}
		if !allowedFlags[arg] {
			return fmt.Errorf("find flag %q is not allowed", arg)
		}
		if arg != "-print" {
			i++
			if i >= len(argv) {
				return fmt.Errorf("find flag %q requires a value", arg)
			}
		}
	}

	return nil
}

func ConfirmExecution(in io.Reader, out io.Writer, plan *Plan) (bool, error) {
	if plan == nil || len(plan.Commands) == 0 {
		return false, nil
	}

	_, err := fmt.Fprintln(out, "主人，本喵翻出了这段危险的 Bash 魔法，真的要挥动爪子执行吗？弄坏了服务器本喵可不负责哦！(Y/N)")
	if err != nil {
		return false, err
	}
	for _, cmd := range plan.Commands {
		if _, err := fmt.Fprintf(out, "\n%s\n", strings.Join(cmd.Argv, " ")); err != nil {
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

func RunPlan(out io.Writer, plan *Plan) error {
	if plan == nil {
		return nil
	}
	for _, command := range plan.Commands {
		cmd := exec.Command(command.Argv[0], command.Argv[1:]...)
		cmd.Stdout = out
		cmd.Stderr = out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("execute command failed: %w", err)
		}
	}
	return nil
}
