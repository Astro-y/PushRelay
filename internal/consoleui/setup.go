package consoleui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	boldReverse = "\x1b[1;7m"
	resetStyle  = "\x1b[0m"
	brightCyan  = "\x1b[1;36m"
	brightGreen = "\x1b[1;32m"
	setupLogo   = `    ____             __    ____       __
   / __ \__  _______/ /_  / __ \___  / /___ ___  __
  / /_/ / / / / ___/ __ \/ /_/ / _ \/ / __ ` + "`" + `/ / / /
 / ____/ /_/ (__  ) / / / _, _/  __/ / /_/ / /_/ /
/_/    \__,_/____/_/ /_/_/ |_|\___/_/\__,_/\__, /
                                            /____/`
)

var logoGradient = [...]rgb{
	{0, 245, 255},
	{57, 255, 20},
	{255, 242, 0},
	{255, 122, 0},
	{255, 0, 170},
}

type rgb struct{ r, g, b int }

// PrintSetupToken renders the one-time administrator setup token as a
// standalone terminal notice so it cannot be missed among structured logs.
func PrintSetupToken(w io.Writer, token string) {
	printSetupToken(w, token, colorEnabled())
}

func printSetupToken(w io.Writer, token string, color bool) {
	label := " SETUP TOKEN  " + token + " "
	if color {
		label = boldReverse + label + resetStyle
	}

	logo := setupLogo
	heading := "PushRelay · 首次初始化"
	divider := "────────────────────────────────────────────────────────"
	if color {
		logo = gradientText(setupLogo)
		heading = brightCyan + heading + resetStyle
		divider = brightGreen + divider + resetStyle
	}

	_, _ = fmt.Fprintf(w, "\n%s\n\n%s\n%s\n\n  %s\n\n  请复制此 Token，并在初始化页面创建管理员。\n  创建管理员后，这段初始化提示将不再显示。\n\n", logo, heading, divider, label)
}

func gradientText(text string) string {
	lines := strings.Split(text, "\n")
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	if width < 2 {
		return text
	}

	var output bytes.Buffer
	for lineIndex, line := range lines {
		for column, char := range []byte(line) {
			if char == ' ' {
				output.WriteByte(char)
				continue
			}
			color := gradientColor(column, width)
			_, _ = fmt.Fprintf(&output, "\x1b[38;2;%d;%d;%dm%c", color.r, color.g, color.b, char)
		}
		output.WriteString(resetStyle)
		if lineIndex < len(lines)-1 {
			output.WriteByte('\n')
		}
	}
	return output.String()
}

func gradientColor(column, width int) rgb {
	position := float64(column) / float64(width-1) * float64(len(logoGradient)-1)
	segment := int(position)
	if segment >= len(logoGradient)-1 {
		return logoGradient[len(logoGradient)-1]
	}
	fraction := position - float64(segment)
	start, end := logoGradient[segment], logoGradient[segment+1]
	interpolate := func(a, b int) int { return int(float64(a) + float64(b-a)*fraction) }
	return rgb{interpolate(start.r, end.r), interpolate(start.g, end.g), interpolate(start.b, end.b)}
}

func colorEnabled() bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}
