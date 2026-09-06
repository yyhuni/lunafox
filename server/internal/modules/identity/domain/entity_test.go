package domain

import "testing"

func TestNormalizeRules(t *testing.T) {
	tests := []struct {
		name      string
		normalize func(string) string
		input     string
		want      string
	}{
		{
			name:      "username trims surrounding spaces",
			normalize: NormalizeUsername,
			input:     "  alice  ",
			want:      "alice",
		},
		{
			name:      "email trims and lowercases",
			normalize: NormalizeEmail,
			input:     "  ALICE@Example.COM  ",
			want:      "alice@example.com",
		},
		{
			name:      "organization name trims surrounding spaces",
			normalize: NormalizeOrganizationName,
			input:     "  Acme Corp  ",
			want:      "Acme Corp",
		},
		{
			name:      "organization description whitespace collapses to empty string after trim",
			normalize: NormalizeOrganizationDescription,
			input:     "   ",
			want:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.normalize(test.input); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestNewUser(t *testing.T) {
	user := NewUser("  alice  ", "  ALICE@Example.COM  ", "hashed-password")

	if user.Username != "alice" {
		t.Fatalf("expected normalized username, got %q", user.Username)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.Password != "hashed-password" {
		t.Fatalf("expected password to be preserved, got %q", user.Password)
	}
	if !user.IsActive {
		t.Fatal("expected new user to be active")
	}
}

func TestUserUpdatePassword(t *testing.T) {
	user := &User{Password: "old-password"}

	user.UpdatePassword("new-password")

	if user.Password != "new-password" {
		t.Fatalf("expected updated password, got %q", user.Password)
	}
}

func TestNewOrganization(t *testing.T) {
	organization := NewOrganization("  Acme Corp  ", "  Core team  ")

	if organization.Name != "Acme Corp" {
		t.Fatalf("expected normalized organization name, got %q", organization.Name)
	}
	if organization.Description != "Core team" {
		t.Fatalf("expected normalized organization description, got %q", organization.Description)
	}
}

func TestOrganizationUpdateProfile(t *testing.T) {
	organization := &Organization{
		Name:        "Legacy Name",
		Description: "Legacy Description",
	}

	organization.UpdateProfile("  New Name  ", "   ")

	if organization.Name != "New Name" {
		t.Fatalf("expected normalized updated name, got %q", organization.Name)
	}
	if organization.Description != "" {
		t.Fatalf("expected trimmed empty description, got %q", organization.Description)
	}
}
