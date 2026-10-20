package service

// Простой (неоптимизированный) алгоритм подбора кандидатов на вакансию,
// действовавший до ПР 6-7. Оставлен как эталон: он легко читается и
// модифицируется, а тест TestRankCandidatesMatchesSimple сверяет с ним
// результаты оптимизированного алгоритма из matching.go.

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"hrselect/internal/models"
)

func ScoreCandidateSimple(c models.Candidate, v models.Vacancy) MatchResult {
	score := 0
	reasons := make([]string, 0, 6)

	required := make([]string, 0, len(v.Skills))
	for _, s := range v.Skills {
		required = append(required, s.Name)
	}
	if len(required) > 0 {
		matched := CountSkillMatches(c.Skills, required)
		skillScore := int(float64(matched) / float64(len(required)) * 60.0)
		score += skillScore
		reasons = append(reasons, fmt.Sprintf("навыки %d/%d", matched, len(required)))
	}

	// Защитная нормализация числовых входов: NaN/Inf/отрицательные значения
	// из legacy-строк БД не должны давать правдоподобные баллы.
	exp := c.ExperienceYears
	if math.IsNaN(exp) || math.IsInf(exp, 0) || exp < 0 {
		exp = 0
	}
	minExp := v.MinExperience
	if math.IsNaN(minExp) || math.IsInf(minExp, 0) || minExp < 0 {
		minExp = 0
	}
	if exp >= minExp {
		score += 15
		reasons = append(reasons, "опыт подходит")
	} else if minExp > 0 {
		partial := int((exp / minExp) * 15)
		if partial > 0 {
			score += partial
		}
	}

	// Защитная нормализация строк: TrimSpace + проверка на пустоту.
	// Раньше EqualFold("", "") давал +5 за «совпадение» пустых городов/занятостей.
	cityC := strings.TrimSpace(c.City)
	cityV := strings.TrimSpace(v.City)
	empC := strings.TrimSpace(c.EmploymentType)
	empV := strings.TrimSpace(v.EmploymentType)
	eduC := strings.TrimSpace(c.Education)
	eduV := strings.TrimSpace(v.EducationLevel)

	if cityC != "" && (strings.EqualFold(cityC, cityV) || strings.EqualFold(cityV, "Удаленно")) {
		score += 5
		reasons = append(reasons, "город подходит")
	}
	if empC != "" && empV != "" && strings.EqualFold(empC, empV) {
		score += 5
		reasons = append(reasons, "занятость подходит")
	}
	if eduV == "" || (eduC != "" && strings.Contains(strings.ToLower(eduC), strings.ToLower(eduV))) {
		score += 5
		reasons = append(reasons, "образование подходит")
	}
	// Зарплата: отрицательные ожидания из поврежденных данных не должны
	// получать баллы как «в бюджете» (т.к. -1 <= max всегда истинно).
	salary := c.SalaryExpectation
	salaryMax := v.SalaryMax
	if salary < 0 {
		// поврежденные данные — баллов нет
	} else if salaryMax <= 0 || salary <= salaryMax {
		score += 10
		reasons = append(reasons, "зарплатные ожидания в бюджете")
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return MatchResult{Candidate: c, Score: score, Reasons: reasons}
}

func RankCandidatesSimple(candidates []models.Candidate, vacancy models.Vacancy) []MatchResult {
	results := make([]MatchResult, 0, len(candidates))
	for _, c := range candidates {
		results = append(results, ScoreCandidateSimple(c, vacancy))
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Candidate.ExperienceYears > results[j].Candidate.ExperienceYears
		}
		return results[i].Score > results[j].Score
	})
	return results
}
