package service

import (
	"testing"

	"hrselect/internal/models"
)

func TestParseOptionalInt(t *testing.T) {
	if v, present, err := ParseOptionalInt(""); err != nil || present || v != 0 {
		t.Fatalf("пустая строка должна быть (0,false,nil), получено (%d,%v,%v)", v, present, err)
	}
	if v, present, err := ParseOptionalInt("25"); err != nil || !present || v != 25 {
		t.Fatalf("ожидалось (25,true,nil), получено (%d,%v,%v)", v, present, err)
	}
	if _, present, err := ParseOptionalInt("abc"); err == nil || !present {
		t.Fatalf("мусор должен давать ошибку с present=true")
	}
}

func TestParseOptionalFloatRejectsNaNInf(t *testing.T) {
	for _, raw := range []string{"NaN", "nan", "+Inf", "-Inf", "Inf", "abc"} {
		if _, _, err := ParseOptionalFloat(raw); err == nil {
			t.Fatalf("ParseOptionalFloat(%q) должен вернуть ошибку", raw)
		}
	}
	if v, present, err := ParseOptionalFloat("3.5"); err != nil || !present || v != 3.5 {
		t.Fatalf("ожидалось (3.5,true,nil)")
	}
}

func TestParseID(t *testing.T) {
	for _, bad := range []string{"", "0", "-1", "abc", "1.5", " "} {
		if _, err := ParseID(bad); err == nil {
			t.Fatalf("ParseID(%q) должен вернуть ошибку", bad)
		}
	}
	if id, err := ParseID("42"); err != nil || id != 42 {
		t.Fatalf("ParseID(42) failed: %v", err)
	}
}

func TestValidateCandidateRejectsBadInput(t *testing.T) {
	base := models.Candidate{
		FullName: "Иван Петров", Age: 25, Gender: "Мужской",
		Email: "ivan@example.com", Phone: "+7 900 111-22-33",
		City: "Екатеринбург", Position: "Go Developer",
		ExperienceYears: 3.5, Education: "Высшее",
		EmploymentType: "Полная занятость", SalaryExpectation: 160000,
	}
	if err := ValidateCandidate(base); err != nil {
		t.Fatalf("валидный кандидат отклонен: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*models.Candidate)
	}{
		{"пустое ФИО", func(c *models.Candidate) { c.FullName = "" }},
		{"возраст 0", func(c *models.Candidate) { c.Age = 0 }},
		{"возраст 999", func(c *models.Candidate) { c.Age = 999 }},
		{"возраст -5", func(c *models.Candidate) { c.Age = -5 }},
		{"пол мусор", func(c *models.Candidate) { c.Gender = "XXX" }},
		{"email мусор", func(c *models.Candidate) { c.Email = "not-an-email" }},
		{"email без зоны", func(c *models.Candidate) { c.Email = "a@b" }},
		{"телефон мусор", func(c *models.Candidate) { c.Phone = "abc" }},
		{"опыт отрицательный", func(c *models.Candidate) { c.ExperienceYears = -1 }},
		{"опыт огромный", func(c *models.Candidate) { c.ExperienceYears = 1000 }},
		{"зарплата отрицательная", func(c *models.Candidate) { c.SalaryExpectation = -10 }},
		{"зарплата огромная", func(c *models.Candidate) { c.SalaryExpectation = 999999999 }},
		{"занятость мусор", func(c *models.Candidate) { c.EmploymentType = "Хакер" }},
	}
	for _, tc := range cases {
		c := base
		tc.mutate(&c)
		if err := ValidateCandidate(c); err == nil {
			t.Fatalf("кейс %q: ожидалась ошибка валидации", tc.name)
		}
	}
}

func TestValidateVacancySalaryRange(t *testing.T) {
	v := models.Vacancy{
		Title: "Go Dev", City: "Екатеринбург", EmploymentType: "Полная занятость",
		MinExperience: 2, SalaryMin: 140000, SalaryMax: 220000,
	}
	if err := ValidateVacancy(v); err != nil {
		t.Fatalf("валидная вакансия отклонена: %v", err)
	}
	v.SalaryMax = 100000 // max < min
	if err := ValidateVacancy(v); err == nil {
		t.Fatalf("инверсия зарплатной вилки должна отклоняться")
	}
}

func TestEscapeLike(t *testing.T) {
	if got := EscapeLike("100%_ok\\"); got != `100\%\_ok\\` {
		t.Fatalf("неверное экранирование: %q", got)
	}
	if got := EscapeLike("обычный"); got != "обычный" {
		t.Fatalf("обычная строка не должна меняться: %q", got)
	}
}

func TestScoreCandidateEmptyStringsNoPhantomPoints(t *testing.T) {
	c := models.Candidate{City: "", EmploymentType: "", Education: "", ExperienceYears: 0, SalaryExpectation: 0}
	v := models.Vacancy{City: "", EmploymentType: "", EducationLevel: "", MinExperience: 0, SalaryMax: 0}
	got := ScoreCandidate(c, v)
	// Разбор: опыт 0>=0 -> +15, образование (уровень пуст) -> +5, зарплата (max=0 -> не задана) -> +10.
	// Город и занятость пустые давать баллы НЕ должны (раньше давали +5+5).
	// Итого ожидаем 30, а не 40.
	if got.Score != 30 {
		t.Fatalf("пустые строки не должны давать баллы за город/занятость, ожидалось 30, получено %d (%v)", got.Score, got.Reasons)
	}
}

func TestScoreCandidateNegativeSalaryNoPoints(t *testing.T) {
	c := models.Candidate{
		City: "Екатеринбург", EmploymentType: "Полная занятость", Education: "Высшее",
		ExperienceYears: 5, SalaryExpectation: -1000,
		Skills: []models.Skill{{Name: "Go"}},
	}
	v := models.Vacancy{
		City: "Екатеринбург", EmploymentType: "Полная занятость", EducationLevel: "Высшее",
		MinExperience: 1, SalaryMax: 200000,
		Skills: []models.Skill{{Name: "Go"}},
	}
	withNeg := ScoreCandidate(c, v)
	c.SalaryExpectation = 150000
	withOk := ScoreCandidate(c, v)
	if !(withOk.Score > withNeg.Score) {
		t.Fatalf("отрицательная зарплата не должна получать баллы: neg=%d ok=%d", withNeg.Score, withOk.Score)
	}
}

func TestScoreCandidateTrimsSpaces(t *testing.T) {
	c := models.Candidate{
		City: "  Екатеринбург ", EmploymentType: "Полная занятость",
		Education: "Высшее", ExperienceYears: 3, SalaryExpectation: 150000,
		Skills: []models.Skill{{Name: "Go"}},
	}
	v := models.Vacancy{
		City: "Екатеринбург", EmploymentType: "Полная занятость", EducationLevel: "Высшее",
		MinExperience: 2, SalaryMax: 200000,
		Skills: []models.Skill{{Name: "Go"}},
	}
	got := ScoreCandidate(c, v)
	found := false
	for _, r := range got.Reasons {
		if r == "город подходит" {
			found = true
		}
	}
	if !found {
		t.Fatalf("город с пробелами должен засчитываться, reasons=%v", got.Reasons)
	}
}

func TestValidateSkillNamesLimits(t *testing.T) {
	long := make([]string, MaxSkillsCount+1)
	for i := range long {
		long[i] = "Go"
	}
	if err := ValidateSkillNames(long); err == nil {
		t.Fatalf("превышение лимита навыков должно отклоняться")
	}
	if err := ValidateSkillNames([]string{string(make([]rune, MaxSkillLen+1))}); err == nil {
		t.Fatalf("слишком длинный навык должен отклоняться")
	}
}
