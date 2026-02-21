package tui

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle          lipgloss.Style
	subtitleStyle       lipgloss.Style
	stepActiveStyle     lipgloss.Style
	stepInactiveStyle   lipgloss.Style
	cardStyle           lipgloss.Style
	menuActiveStyle     lipgloss.Style
	menuInactiveStyle   lipgloss.Style
	inputFocusedStyle   lipgloss.Style
	inputUnfocusedStyle lipgloss.Style
	labelStyle          lipgloss.Style
	hintStyle           lipgloss.Style
	errorMsgStyle       lipgloss.Style
	successMsgStyle     lipgloss.Style
	networkHintStyle    lipgloss.Style
	helpStyle           lipgloss.Style
	configLabelStyle    lipgloss.Style
	configValueStyle       lipgloss.Style
	progressDoneStyle      lipgloss.Style
	progressActiveStyle    lipgloss.Style
	progressPendingStyle   lipgloss.Style
	progressConnectorStyle lipgloss.Style
	statusOkStyle          lipgloss.Style
	statusErrStyle         lipgloss.Style
	sectionTitleStyle      lipgloss.Style
	separatorStyle         lipgloss.Style
	stylesInitialized      bool
)

func initStyles() {
	if stylesInitialized {
		return
	}
	stylesInitialized = true
	configureStyles(shouldUseColor())
}

func shouldUseColor() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	if strings.TrimSpace(os.Getenv("CLICOLOR")) == "0" {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}

func configureStyles(enableColor bool) {
	titleStyle = lipgloss.NewStyle().Bold(true)
	subtitleStyle = lipgloss.NewStyle()
	stepActiveStyle = lipgloss.NewStyle().Bold(true)
	stepInactiveStyle = lipgloss.NewStyle()
	cardStyle = lipgloss.NewStyle()
	menuActiveStyle = lipgloss.NewStyle().Bold(true)
	menuInactiveStyle = lipgloss.NewStyle()
	inputFocusedStyle = lipgloss.NewStyle().Bold(true)
	inputUnfocusedStyle = lipgloss.NewStyle()
	labelStyle = lipgloss.NewStyle()
	hintStyle = lipgloss.NewStyle()
	errorMsgStyle = lipgloss.NewStyle().Bold(true)
	successMsgStyle = lipgloss.NewStyle().Bold(true)
	networkHintStyle = lipgloss.NewStyle()
	helpStyle = lipgloss.NewStyle()
	configLabelStyle = lipgloss.NewStyle()
	configValueStyle = lipgloss.NewStyle().Bold(true)
	progressDoneStyle = lipgloss.NewStyle()
	progressActiveStyle = lipgloss.NewStyle().Bold(true)
	progressPendingStyle = lipgloss.NewStyle()
	progressConnectorStyle = lipgloss.NewStyle()
	statusOkStyle = lipgloss.NewStyle()
	statusErrStyle = lipgloss.NewStyle().Bold(true)
	sectionTitleStyle = lipgloss.NewStyle().Bold(true)

	if !enableColor {
		stepActiveStyle = stepActiveStyle.Underline(true)
		return
	}

	primaryColor := lipgloss.Color("#7D56F4")
	secondaryColor := lipgloss.Color("#00D9FF")
	successColor := lipgloss.Color("#00E676")
	warningColor := lipgloss.Color("#FFB300")
	errorColor := lipgloss.Color("#FF5252")
	textPrimary := lipgloss.Color("#FAFAFA")
	textSecondary := lipgloss.Color("#B0B0B0")
	textMuted := lipgloss.Color("#808080")
	borderColor := lipgloss.Color("#2D3561")

	titleStyle = titleStyle.Foreground(primaryColor)
	subtitleStyle = subtitleStyle.Foreground(textSecondary)
	stepActiveStyle = stepActiveStyle.Foreground(primaryColor)
	stepInactiveStyle = stepInactiveStyle.Foreground(textMuted)
	cardStyle = cardStyle.
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(borderColor)
	menuActiveStyle = menuActiveStyle.Foreground(primaryColor)
	menuInactiveStyle = menuInactiveStyle.Foreground(textSecondary)
	inputFocusedStyle = inputFocusedStyle.
		Foreground(primaryColor).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(primaryColor)
	inputUnfocusedStyle = inputUnfocusedStyle.
		Foreground(textSecondary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(borderColor)
	labelStyle = labelStyle.Foreground(textSecondary)
	hintStyle = hintStyle.Foreground(secondaryColor)
	errorMsgStyle = errorMsgStyle.Foreground(errorColor)
	successMsgStyle = successMsgStyle.Foreground(successColor)
	networkHintStyle = networkHintStyle.Foreground(warningColor)
	helpStyle = helpStyle.Foreground(textMuted)
	configLabelStyle = configLabelStyle.Foreground(textSecondary)
	configValueStyle = configValueStyle.Foreground(textPrimary)
	progressDoneStyle = progressDoneStyle.Foreground(successColor)
	progressActiveStyle = progressActiveStyle.Foreground(primaryColor)
	progressPendingStyle = progressPendingStyle.Foreground(textMuted)
	progressConnectorStyle = progressConnectorStyle.Foreground(textMuted)
	statusOkStyle = statusOkStyle.Foreground(successColor)
	statusErrStyle = statusErrStyle.Foreground(errorColor)
	sectionTitleStyle = sectionTitleStyle.Foreground(secondaryColor)
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	
	configLabelStyle = configLabelStyle.Foreground(textSecondary)
}

func renderStepIndicator(current, total int) string {
	return stepActiveStyle.Render("Step " + strconv.Itoa(current) + "/" + strconv.Itoa(total))
}

func renderCard(content string, termWidth int) string {
	_ = termWidth
	return content
}

func renderMenuItem(active bool, label string) string {
	if active {
		return "    " + menuActiveStyle.Render("▶ "+label)
	}
	return "      " + menuInactiveStyle.Render(label)
}

func renderInputLine(label, value string, focused bool) string {
	var valueRendered string
	if focused {
		valueRendered = inputFocusedStyle.Render(value)
	} else {
		valueRendered = inputUnfocusedStyle.Render(value)
	}
	return labelStyle.Render(label+": ") + valueRendered
}

func renderConfigItem(label, value string) string {
	return "    " + configLabelStyle.Render(label) + "  " + configValueStyle.Render(value)
}

func renderInputPrefix(label string) string {
	return "    " + configLabelStyle.Render(label) + "  "
}

func renderCompletedItem(label, value string) string {
	return "    " + progressDoneStyle.Render("✓") + " " + progressPendingStyle.Render(label+":  "+value)
}

func renderSeparator(termWidth int) string {
	width := 60
	if termWidth > 0 {
		width = termWidth
	}
	if width < 20 {
		width = 20
	}
	if width > 100 {
		width = 100
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}

func renderDoubleSeparator(termWidth int) string {
	width := 60
	if termWidth > 0 {
		width = termWidth
	}
	if width < 20 {
		width = 20
	}
	if width > 100 {
		width = 100
	}
	return separatorStyle.Render(strings.Repeat("═", width))
}

func renderProgressBar(labels []string, current int) string {
	var parts []string
	for i, label := range labels {
		stepNum := i + 1
		if stepNum < current {
			parts = append(parts, progressDoneStyle.Render("● "+label))
		} else if stepNum == current {
			parts = append(parts, progressActiveStyle.Render("● "+label))
		} else {
			parts = append(parts, progressPendingStyle.Render("○ "+label))
		}
	}
	connector := progressConnectorStyle.Render("  ──  ")
	return "  " + strings.Join(parts, connector)
}

func renderStatusOk(text string) string {
	return "    " + statusOkStyle.Render("✓") + " " + text
}

func renderStatusErr(text string) string {
	return "    " + statusErrStyle.Render("✗ "+text)
}

func renderHint(text string) string {
	return "    " + hintStyle.Render("· "+text)
}
