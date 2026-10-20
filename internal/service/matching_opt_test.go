package service

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"hrselect/internal/models"
)

var (
	testSkills = []string{"Go", "Python", "FastAPI", "PostgreSQL", "Docker", "Git", "Linux", "Redis",
		"REST API", "SQL", "Django", "Kafka", "Kubernetes", "HTML/CSS", "JavaScript", "Fiber"}
	testCities = []string{"Екатеринбург", "Москва", "Казань", "Пермь", " Екатеринбург ", "", "Удаленно"}
	testEmp    = []string{"Полная занятость", "Удаленно", "Частичная занятость", ""}
	testEdu    = []string{"Высшее", "Среднее профессиональное", "высшее техническое", ""}
)

// genCandidates строит воспроизводимый набор кандидатов, включая
// «неудобные» данные: пробелы, разный регистр, повтор навыка, NaN и
// отрицательные значения.
func genCandidates(n int, seed int64) []models.Candidate {
	return genCandidatesOpt(n, seed, true)
}

func genCandidatesOpt(n int, seed int64, withNaN bool) []models.Candidate {
	rnd := rand.New(rand.NewSource(seed))
	out := make([]models.Candidate, n)
	for i := range out {
		k := rnd.Intn(9)
		skills := make([]models.Skill, 0, k+1)
		for j := 0; j < k; j++ {
			name := testSkills[rnd.Intn(len(testSkills))]
			if rnd.Intn(5) == 0 {
				name = " " + name + " "
			}
			skills = append(skills, models.Skill{Name: name})
		}
		if k > 0 && rnd.Intn(10) == 0 {
			skills = append(skills, models.Skill{Name: "go"})
		}
		exp := float64(rnd.Intn(25)) / 2
		switch rnd.Intn(40) {
		case 0:
			if withNaN {
				exp = math.NaN()
			}
		case 1:
			exp = -1
		}
		salary := 50000 + rnd.Intn(30)*10000
		if rnd.Intn(40) == 0 {
			salary = -1
		}
		out[i] = models.Candidate{
			ID:                uint(i + 1),
			FullName:          fmt.Sprintf("Кандидат %d", i+1),
			City:              testCities[rnd.Intn(len(testCities))],
			EmploymentType:    testEmp[rnd.Intn(len(testEmp))],
			Education:         testEdu[rnd.Intn(len(testEdu))],
			ExperienceYears:   exp,
			SalaryExpectation: salary,
			Skills:            skills,
		}
	}
	return out
}

func testVacancies() []models.Vacancy {
	skills := func(names ...string) []models.Skill {
		out := make([]models.Skill, len(names))
		for i, n := range names {
			out[i] = models.Skill{Name: n}
		}
		return out
	}
	return []models.Vacancy{
		{City: "Екатеринбург", EmploymentType: "Полная занятость", MinExperience: 2, EducationLevel: "Высшее",
			SalaryMax: 220000, Skills: skills("Python", "FastAPI", "PostgreSQL", "Docker", "Git", "REST API")},
		{City: "Удаленно", EmploymentType: "Удаленно", MinExperience: 3, SalaryMax: 260000,
			Skills: skills("Python", "SQL", "Docker", "Git")},
		{City: " москва ", EmploymentType: "", MinExperience: 0, EducationLevel: "высшее"},
		{City: "Казань", MinExperience: 5, SalaryMax: 0, Skills: skills("Go", "go", "Kafka")},
	}
}

// Эталонная проверка: оптимизированный алгоритм обязан давать тот же
// рейтинг и те же пояснения, что и простой, для каждого кандидата.
func TestScoreCandidateMatchesSimple(t *testing.T) {
	candidates := genCandidates(3000, 1)
	for vi, v := range testVacancies() {
		p := NewVacancyProfile(v)
		for _, c := range candidates {
			want := ScoreCandidateSimple(c, v)
			got := p.Score(c)
			if got.Score != want.Score || !reflect.DeepEqual(got.Reasons, want.Reasons) {
				t.Fatalf("вакансия %d, кандидат %d: ожидалось %d %v, получено %d %v",
					vi, c.ID, want.Score, want.Reasons, got.Score, got.Reasons)
			}
		}
	}
}

func resultIDs(rs []MatchResult) []uint {
	ids := make([]uint, len(rs))
	for i, r := range rs {
		ids[i] = r.Candidate.ID
	}
	return ids
}

// Полная выдача совпадает с устойчивой сортировкой простого алгоритма,
// а выдача top-N — с ее первыми N строками. NaN в опыте исключен: для него
// порядок простого алгоритма не определен (см. TestRankPutsNaNExperienceLast).
func TestRankCandidatesMatchesSimple(t *testing.T) {
	candidates := genCandidatesOpt(3000, 2, false)
	for vi, v := range testVacancies() {
		want := resultIDs(RankCandidatesSimple(candidates, v))
		if got := resultIDs(RankCandidates(candidates, v)); !reflect.DeepEqual(got, want) {
			t.Fatalf("вакансия %d: порядок полной выдачи отличается", vi)
		}
		for _, limit := range []int{1, 5, 20, 100, 2999, 3000, 5000} {
			got := resultIDs(RankTopCandidates(candidates, v, limit))
			n := limit
			if n > len(want) {
				n = len(want)
			}
			if !reflect.DeepEqual(got, want[:n]) {
				t.Fatalf("вакансия %d, top=%d: выдача не совпадает с эталоном", vi, limit)
			}
		}
	}
}

func TestRankPutsNaNExperienceLast(t *testing.T) {
	v := models.Vacancy{}
	cs := []models.Candidate{
		{ID: 1, ExperienceYears: math.NaN(), City: "Пермь"},
		{ID: 2, ExperienceYears: 3, City: "Пермь"},
		{ID: 3, ExperienceYears: 1, City: "Пермь"},
	}
	for _, limit := range []int{0, 2} {
		got := resultIDs(RankTopCandidates(cs, v, limit))
		want := []uint{2, 3, 1}[:len(got)]
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("top=%d: ожидалось %v, получено %v", limit, want, got)
		}
	}
}

func TestRankTopCandidatesEdgeCases(t *testing.T) {
	v := testVacancies()[0]
	if got := RankTopCandidates(nil, v, 10); len(got) != 0 {
		t.Fatalf("пустой список кандидатов должен давать пустую выдачу, получено %d", len(got))
	}
	one := genCandidates(1, 3)
	if got := RankTopCandidates(one, v, 10); len(got) != 1 {
		t.Fatalf("ожидался 1 результат, получено %d", len(got))
	}
	if got := RankTopCandidates(genCandidates(50, 4), models.Vacancy{}, 7); len(got) != 7 {
		t.Fatalf("вакансия без требований: ожидалось 7 результатов, получено %d", len(got))
	}
}

func benchCandidates(b *testing.B, n int) ([]models.Candidate, models.Vacancy) {
	b.Helper()
	return genCandidates(n, 42), testVacancies()[0]
}

func BenchmarkRankSimple1k(b *testing.B) {
	cs, v := benchCandidates(b, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankCandidatesSimple(cs, v)
	}
}

func BenchmarkRankOptimizedAll1k(b *testing.B) {
	cs, v := benchCandidates(b, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankCandidates(cs, v)
	}
}

func BenchmarkRankOptimizedTop20_1k(b *testing.B) {
	cs, v := benchCandidates(b, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankTopCandidates(cs, v, 20)
	}
}

func BenchmarkRankSimple10k(b *testing.B) {
	cs, v := benchCandidates(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankCandidatesSimple(cs, v)
	}
}

func BenchmarkRankOptimizedAll10k(b *testing.B) {
	cs, v := benchCandidates(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankCandidates(cs, v)
	}
}

func BenchmarkRankOptimizedTop20_10k(b *testing.B) {
	cs, v := benchCandidates(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankTopCandidates(cs, v, 20)
	}
}

func BenchmarkRankSimple100k(b *testing.B) {
	cs, v := benchCandidates(b, 100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankCandidatesSimple(cs, v)
	}
}

func BenchmarkRankOptimizedTop20_100k(b *testing.B) {
	cs, v := benchCandidates(b, 100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankTopCandidates(cs, v, 20)
	}
}
