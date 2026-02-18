package model

import "time"

// User represents a Django auth_user compatible model.
type User struct {
	ID          int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Password    string     `gorm:"column:password;size:128" json:"-"`
	LastLogin   *time.Time `gorm:"column:last_login" json:"lastLogin"`
	IsSuperuser bool       `gorm:"column:is_superuser;default:false" json:"isSuperuser"`
	Username    string     `gorm:"column:username;size:150;uniqueIndex" json:"username"`
	FirstName   string     `gorm:"column:first_name;size:150" json:"firstName"`
	LastName    string     `gorm:"column:last_name;size:150" json:"lastName"`
	Email       string     `gorm:"column:email;size:254" json:"email"`
	IsStaff     bool       `gorm:"column:is_staff;default:false" json:"isStaff"`
	IsActive    bool       `gorm:"column:is_active;default:true" json:"isActive"`
	DateJoined  time.Time  `gorm:"column:date_joined;autoCreateTime" json:"dateJoined"`
}

func (User) TableName() string {
	return "auth_user"
}
