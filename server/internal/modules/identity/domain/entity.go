package domain

import "time"

// User represents identity user in domain layer.
type User struct {
	ID          int
	Password    string
	LastLogin   *time.Time
	IsSuperuser bool
	Username    string
	FirstName   string
	LastName    string
	Email       string
	IsStaff     bool
	IsActive    bool
	DateJoined  time.Time
}

// Organization represents tenant-like business unit.
type Organization struct {
	ID          int
	Name        string
	Description string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

// OrganizationWithTargetCount is a query projection for organization listing/details.
type OrganizationWithTargetCount struct {
	Organization
	TargetCount int64
}

// OrganizationTargetRef is a projection for org-target relation reads.
type OrganizationTargetRef struct {
	ID            int
	Name          string
	Type          string
	CreatedAt     time.Time
	LastScannedAt *time.Time
	DeletedAt     *time.Time
}

func NewUser(username, email, hashedPassword string) *User {
	return &User{
		Username: NormalizeUsername(username),
		Email:    NormalizeEmail(email),
		Password: hashedPassword,
		IsActive: true,
	}
}

func (user *User) UpdatePassword(hashedPassword string) {
	user.Password = hashedPassword
}

func NewOrganization(name, description string) *Organization {
	return &Organization{
		Name:        NormalizeOrganizationName(name),
		Description: NormalizeOrganizationDescription(description),
	}
}

func (organization *Organization) UpdateProfile(name, description string) {
	organization.Name = NormalizeOrganizationName(name)
	organization.Description = NormalizeOrganizationDescription(description)
}
