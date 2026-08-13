package consoleui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintSetupTokenUsesReverseVideo(t *testing.T) {
	var output bytes.Buffer

	printSetupToken(&output, "setup-token", true)

	text := output.String()
	if !strings.Contains(text, boldReverse+" SETUP TOKEN  setup-token "+resetStyle) {
		t.Fatalf("PrintSetupToken() output does not contain reverse-video token: %q", text)
	}
	if !strings.Contains(text, "\x1b[38;2;") {
		t.Fatalf("PrintSetupToken() output does not contain true-color logo: %q", text)
	}
	if !strings.Contains(text, "PushRelay · 首次初始化") {
		t.Fatalf("PrintSetupToken() output does not contain heading: %q", text)
	}
}

func TestPrintSetupTokenHonorsNoColor(t *testing.T) {
	var output bytes.Buffer

	printSetupToken(&output, "plain-token", false)

	text := output.String()
	if strings.Contains(text, "\x1b[") {
		t.Fatalf("PrintSetupToken() output contains ANSI escapes with NO_COLOR: %q", text)
	}
	if !strings.Contains(text, "SETUP TOKEN  plain-token") {
		t.Fatalf("PrintSetupToken() output does not contain token: %q", text)
	}
	if !strings.Contains(text, "/ __ \\") {
		t.Fatalf("PrintSetupToken() output does not contain ASCII logo: %q", text)
	}
}

func TestGradientColorUsesPaletteEndpoints(t *testing.T) {
	if got := gradientColor(0, 10); got != logoGradient[0] {
		t.Fatalf("gradientColor(0) = %#v, want %#v", got, logoGradient[0])
	}
	if got := gradientColor(9, 10); got != logoGradient[len(logoGradient)-1] {
		t.Fatalf("gradientColor(9) = %#v, want %#v", got, logoGradient[len(logoGradient)-1])
	}
}
