package ui

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
)

var spinnerFrames = []string{"ฅ^•ﻌ•^ฅ", "ฅ^•ﻌ•^ฅ ~", "ฅ^•ﻌ•^ฅ ~~", "ฅ-_-ฅ zZ"}

func StartCatSpinner(message string) (*pterm.SpinnerPrinter, error) {
	spinner := pterm.DefaultSpinner
	spinner.Sequence = spinnerFrames
	return spinner.Start("小猫助手出动中｜" + message)
}

func PrintCatReply(title string, resp string) {
	pterm.DefaultBasicText.Println(catASCII)
	pterm.DefaultBox.WithTitle(title).WithLeftPadding(2).WithRightPadding(2).Println(toBubble(resp))
}

func toBubble(resp string) string {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return "喵……这次模型没有吐出内容，本喵先趴一下。"
	}
	return fmt.Sprintf("主人，报告在这里喵：\n\n%s", resp)
}

const catASCII = ` /\_/\\
( o.o )
 > ^ <   赛博运维猫在线值守喵`
