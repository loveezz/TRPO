package models

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey"`
	FullName     string `gorm:"size:200;not null"`
	Email        string `gorm:"size:200;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	CreatedAt    time.Time
}

type Candidate struct {
	ID                uint    `gorm:"primaryKey"`
	FullName          string  `gorm:"size:200;not null;index"`
	Age               int     `gorm:"not null;index"`
	Gender            string  `gorm:"size:32;not null;index"`
	Email             string  `gorm:"size:200;not null;index"`
	Phone             string  `gorm:"size:64;not null"`
	City              string  `gorm:"size:120;not null;index"`
	Position          string  `gorm:"size:160;not null;index"`
	ExperienceYears   float64 `gorm:"not null;index"`
	Education         string  `gorm:"size:160;not null;index"`
	EmploymentType    string  `gorm:"size:64;not null;index"`
	SalaryExpectation int     `gorm:"not null;index"`
	About             string  `gorm:"type:text"`
	Skills            []Skill `gorm:"many2many:candidate_skills;constraint:OnDelete:CASCADE;"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Vacancy struct {
	ID             uint    `gorm:"primaryKey"`
	Title          string  `gorm:"size:200;not null;index"`
	Department     string  `gorm:"size:160"`
	City           string  `gorm:"size:120;not null;index"`
	EmploymentType string  `gorm:"size:64;not null;index"`
	MinExperience  float64 `gorm:"not null;default:0"`
	EducationLevel string  `gorm:"size:160"`
	SalaryMin      int     `gorm:"not null;default:0"`
	SalaryMax      int     `gorm:"not null;default:0"`
	Description    string  `gorm:"type:text"`
	Requirements   string  `gorm:"type:text"`
	IsActive       bool    `gorm:"not null;default:true;index"`
	Skills         []Skill `gorm:"many2many:vacancy_skills;constraint:OnDelete:CASCADE;"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Skill struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100;uniqueIndex;not null"`
}
