package service

import (
	"testing"

	"hrselect/internal/models"
)

func TestScoreCandidatePerfectMatch(t *testing.T) {
	c := models.Candidate{
		City: "Екатеринбург", EmploymentType: "Полная занятость", Education: "Высшее",
		ExperienceYears: 4, SalaryExpectation: 180000,
		Skills: []models.Skill{{Name: "Go"}, {Name: "Fiber"}, {Name: "PostgreSQL"}},
	}
	v := models.Vacancy{
		City: "Екатеринбург", EmploymentType: "Полная занятость", EducationLevel: "Высшее",
		MinExperience: 3, SalaryMax: 220000,
		Skills: []models.Skill{{Name: "Go"}, {Name: "Fiber"}, {Name: "PostgreSQL"}},
	}
	got := ScoreCandidate(c, v)
	if got.Score != 100 {
		t.Fatalf("expected 100, got %d", got.Score)
	}
}
