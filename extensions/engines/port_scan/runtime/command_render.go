package portscanruntime

import (
	"strings"
)

func commandLabelForProgress(command naabuCommand) string {
	label := strings.TrimSpace(command.label)
	if label != "" {
		return label
	}
	return "naabu"
}

func renderNaabuCommand(command naabuCommand) string {
	parts := []string{"naabu"}
	for _, arg := range command.args {
		parts = append(parts, quoteCommandPartForProgress(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandPartForProgress(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '=' || r == '@' || r == '%')
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
