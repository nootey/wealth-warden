package models

type Organization struct {
	ID       uint
	Name     string
	Users    []User    `gorm:"many2many:user_organizations;"`
	Features []Feature `gorm:"many2many:organization_features;"`
}
