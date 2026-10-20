package service

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
	"strings"

	"hrselect/internal/models"
)

type MatchResult struct {
	Candidate models.Candidate
	Score     int
	Reasons   []string
}

func NormalizeSkills(raw string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func CountSkillMatches(candidateSkills []models.Skill, requested []string) int {
	set := make(map[string]struct{}, len(candidateSkills))
	for _, s := range candidateSkills {
		set[strings.ToLower(strings.TrimSpace(s.Name))] = struct{}{}
	}
	count := 0
	for _, r := range requested {
		if _, ok := set[strings.ToLower(strings.TrimSpace(r))]; ok {
			count++
		}
	}
	return count
}

// VacancyProfile — внутренний (вспомогательный) класс алгоритма подбора.
// В предметной области ему ничего не соответствует: в нем один раз на
// вакансию подготовлено все, что простой алгоритм (matching_simple.go)
// заново вычислял для каждого кандидата: список требуемых навыков,
// приведение регистра, очистка строк, проверка числовых полей.
// Классы Candidate, Vacancy и MatchResult образуют «звезду» с центром в
// вакансии, поэтому операция оценки помещена в класс вокруг вакансии.
type VacancyProfile struct {
	Vacancy models.Vacancy

	skillSlot   map[string]int // навык в нижнем регистре -> номер ячейки
	rawSlot     map[string]int // кэш: название навыка как в БД -> ячейка (-1 — не требуется)
	skillWeight []int          // сколько раз навык указан в вакансии
	skillTotal  int            // len(v.Skills) — знаменатель доли навыков

	city      string
	remote    bool
	emp       string
	eduLower  string
	eduOK     map[string]bool // кэш: образование кандидата -> подходит ли
	minExp    float64
	salaryMax int
}

// NewVacancyProfile подготавливает вакансию к подбору за O(m),
// где m — число требуемых навыков.
func NewVacancyProfile(v models.Vacancy) *VacancyProfile {
	p := &VacancyProfile{
		Vacancy:    v,
		skillSlot:  make(map[string]int, len(v.Skills)),
		rawSlot:    make(map[string]int, 64),
		eduOK:      make(map[string]bool, 8),
		skillTotal: len(v.Skills),
		city:       strings.TrimSpace(v.City),
		emp:        strings.TrimSpace(v.EmploymentType),
		eduLower:   strings.ToLower(strings.TrimSpace(v.EducationLevel)),
		minExp:     sanitizeYears(v.MinExperience),
		salaryMax:  v.SalaryMax,
	}
	p.remote = strings.EqualFold(p.city, "Удаленно")
	for _, s := range v.Skills {
		key := strings.ToLower(strings.TrimSpace(s.Name))
		if slot, ok := p.skillSlot[key]; ok {
			p.skillWeight[slot]++
			continue
		}
		p.skillSlot[key] = len(p.skillWeight)
		p.skillWeight = append(p.skillWeight, 1)
	}
	return p
}

// sanitizeYears: NaN/Inf/отрицательные значения из поврежденных строк БД
// не должны давать правдоподобные баллы (защитная проверка из ПР 4-5).
func sanitizeYears(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 {
		return 0
	}
	return x
}

// maxSkillScore — верхняя граница баллов за навыки (для отсечения).
func (p *VacancyProfile) maxSkillScore() int {
	if p.skillTotal == 0 {
		return 0
	}
	return 60
}

// baseScore считает «дешевую» часть рейтинга без навыков: опыт, город,
// занятость, образование, зарплата — до 40 баллов. При reasons == nil
// пояснения не формируются.
func (p *VacancyProfile) baseScore(c *models.Candidate, reasons *[]string) int {
	score := 0
	exp := sanitizeYears(c.ExperienceYears)
	if exp >= p.minExp {
		score += 15
		addReason(reasons, "опыт подходит")
	} else if p.minExp > 0 {
		if partial := int((exp / p.minExp) * 15); partial > 0 {
			score += partial
		}
	}
	cityC := strings.TrimSpace(c.City)
	if cityC != "" && (p.remote || strings.EqualFold(cityC, p.city)) {
		score += 5
		addReason(reasons, "город подходит")
	}
	empC := strings.TrimSpace(c.EmploymentType)
	if empC != "" && p.emp != "" && strings.EqualFold(empC, p.emp) {
		score += 5
		addReason(reasons, "занятость подходит")
	}
	if p.eduLower == "" {
		score += 5
		addReason(reasons, "образование подходит")
	} else if p.educationFits(c.Education) {
		score += 5
		addReason(reasons, "образование подходит")
	}
	if c.SalaryExpectation >= 0 && (p.salaryMax <= 0 || c.SalaryExpectation <= p.salaryMax) {
		score += 10
		addReason(reasons, "зарплатные ожидания в бюджете")
	}
	return score
}

// educationFits: уровней образования немного, поэтому результат
// сравнения кэшируется по исходной строке (без повторного ToLower).
func (p *VacancyProfile) educationFits(edu string) bool {
	if ok, cached := p.eduOK[edu]; cached {
		return ok
	}
	eduC := strings.TrimSpace(edu)
	ok := eduC != "" && strings.Contains(strings.ToLower(eduC), p.eduLower)
	p.eduOK[edu] = ok
	return ok
}

func addReason(reasons *[]string, s string) {
	if reasons != nil {
		*reasons = append(*reasons, s)
	}
}

// slotOf возвращает ячейку требуемого навыка или -1. Названия навыков
// берутся из справочника skills и повторяются у многих кандидатов, поэтому
// результат нормализации (TrimSpace + ToLower) кэшируется по исходной
// строке: повторный поиск — одно обращение к хеш-таблице без выделения
// памяти. Из-за кэша профиль не предназначен для параллельного использования.
func (p *VacancyProfile) slotOf(name string) int {
	if slot, ok := p.rawSlot[name]; ok {
		return slot
	}
	slot, ok := p.skillSlot[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		slot = -1
	}
	p.rawSlot[name] = slot
	return slot
}

// skillMatches — один проход по навыкам кандидата с поиском в хеш-таблице
// требований. seen — переиспользуемый буфер: навык, записанный у кандидата
// дважды («Go» и «go»), засчитывается один раз.
func (p *VacancyProfile) skillMatches(c *models.Candidate, seen []bool) int {
	for i := range seen {
		seen[i] = false
	}
	matched := 0
	for _, s := range c.Skills {
		slot := p.slotOf(s.Name)
		if slot >= 0 && !seen[slot] {
			seen[slot] = true
			matched += p.skillWeight[slot]
		}
	}
	return matched
}

func (p *VacancyProfile) skillScore(matched int) int {
	return int(float64(matched) / float64(p.skillTotal) * 60.0)
}

// quickScore — итоговый рейтинг без формирования пояснений.
func (p *VacancyProfile) quickScore(c *models.Candidate, seen []bool) int {
	score := p.baseScore(c, nil)
	if p.skillTotal > 0 {
		score += p.skillScore(p.skillMatches(c, seen))
	}
	return clampScore(score)
}

// Score — полный расчет рейтинга кандидата с пояснениями.
func (p *VacancyProfile) Score(c models.Candidate) MatchResult {
	return p.score(&c, make([]bool, len(p.skillWeight)))
}

func (p *VacancyProfile) score(c *models.Candidate, seen []bool) MatchResult {
	reasons := make([]string, 0, 6)
	score := 0
	if p.skillTotal > 0 {
		matched := p.skillMatches(c, seen)
		score += p.skillScore(matched)
		reasons = append(reasons, fmt.Sprintf("навыки %d/%d", matched, p.skillTotal))
	}
	score += p.baseScore(c, &reasons)
	return MatchResult{Candidate: *c, Score: clampScore(score), Reasons: reasons}
}

func clampScore(s int) int {
	if s > 100 {
		return 100
	}
	if s < 0 {
		return 0
	}
	return s
}

// ScoreCandidate — оценка одного кандидата (оптимизированная версия).
func ScoreCandidate(c models.Candidate, v models.Vacancy) MatchResult {
	return NewVacancyProfile(v).Score(c)
}

// RankCandidates упорядочивает всех кандидатов по убыванию рейтинга.
func RankCandidates(candidates []models.Candidate, vacancy models.Vacancy) []MatchResult {
	return RankTopCandidates(candidates, vacancy, 0)
}

// rankItem — результат с позицией кандидата во входном списке:
// позиция заменяет устойчивую сортировку при равенстве рейтинга и опыта.
type rankItem struct {
	res MatchResult
	pos int
}

// outranks задает порядок выдачи: рейтинг по убыванию, затем опыт
// по убыванию, затем исходный порядок (как у sort.SliceStable).
func outranks(score int, exp float64, pos int, b *rankItem) bool {
	if score != b.res.Score {
		return score > b.res.Score
	}
	ea, eb := expKey(exp), expKey(b.res.Candidate.ExperienceYears)
	if ea != eb {
		return ea > eb
	}
	return pos < b.pos
}

// expKey: NaN в опыте (поврежденная строка БД) ставится в конец. В простом
// алгоритме сравнение с NaN нарушало транзитивность и порядок был не определен.
func expKey(x float64) float64 {
	if math.IsNaN(x) {
		return math.Inf(-1)
	}
	return x
}

func better(a, b *rankItem) bool {
	return outranks(a.res.Score, a.res.Candidate.ExperienceYears, a.pos, b)
}

// worstHeap — двоичная куча, на вершине которой худший из отобранных.
type worstHeap []*rankItem

func (h worstHeap) Len() int            { return len(h) }
func (h worstHeap) Less(i, j int) bool  { return better(h[j], h[i]) }
func (h worstHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *worstHeap) Push(x interface{}) { *h = append(*h, x.(*rankItem)) }
func (h *worstHeap) Pop() interface{} {
	old := *h
	it := old[len(old)-1]
	*h = old[:len(old)-1]
	return it
}

// RankTopCandidates возвращает limit лучших кандидатов (limit <= 0 — всех).
//
// Оптимизации относительно RankCandidatesSimple:
//  1. требования вакансии подготавливаются один раз (VacancyProfile);
//  2. совпадение навыков — один проход с поиском в хеш-таблице,
//     без построения множества навыков для каждого кандидата;
//  3. при limit > 0 отбор идет через кучу размера limit:
//     O(n log limit) вместо полной сортировки O(n log n);
//  4. отсечение: если даже при всех совпавших навыках кандидат не обгонит
//     худшего из отобранных, навыки не сравниваются; пояснения (строки)
//     формируются только для попавших в выдачу.
func RankTopCandidates(candidates []models.Candidate, vacancy models.Vacancy, limit int) []MatchResult {
	p := NewVacancyProfile(vacancy)
	seen := make([]bool, len(p.skillWeight))

	if limit <= 0 || limit >= len(candidates) {
		items := make([]rankItem, len(candidates))
		for i := range candidates {
			items[i] = rankItem{res: p.score(&candidates[i], seen), pos: i}
		}
		sort.Slice(items, func(i, j int) bool { return better(&items[i], &items[j]) })
		out := make([]MatchResult, len(items))
		for i := range items {
			out[i] = items[i].res
		}
		return out
	}

	h := make(worstHeap, 0, limit)
	for i := range candidates {
		c := &candidates[i]
		if len(h) < limit {
			heap.Push(&h, &rankItem{res: p.score(c, seen), pos: i})
			continue
		}
		worst := h[0]
		if clampScore(p.baseScore(c, nil)+p.maxSkillScore()) < worst.res.Score {
			continue // отсечение по верхней границе
		}
		if !outranks(p.quickScore(c, seen), c.ExperienceYears, i, worst) {
			continue
		}
		h[0] = &rankItem{res: p.score(c, seen), pos: i}
		heap.Fix(&h, 0)
	}

	out := make([]MatchResult, len(h))
	for i := len(h) - 1; i >= 0; i-- {
		out[i] = heap.Pop(&h).(*rankItem).res
	}
	return out
}
