package domain

import "strings"

func NormalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func NormalizeOrganizationName(name string) string {
	return strings.TrimSpace(name)
}

func NormalizeOrganizationDescription(description string) string {
	return strings.TrimSpace(description)
}
