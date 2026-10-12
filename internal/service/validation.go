package service

import (
	"fmt"
	"math"
	"net/mail"
	"strconv"
	"strings"

	"hrselect/internal/models"
)

var allowedGenders = map[string]struct{}{
	"Мужской": {},
	"Женский": {},
}

var allowedEmploymentTypes = map[string]struct{}{
	"Полная занятость":    {},
	"Удаленно":            {},
	"Частичная занятость": {},
}

var allowedSorts = map[string]struct{}{
	"experience": {},
	"skills":     {},
	"salary":     {},
	"name":       {},
}

const (
	MaxNameLen     = 200
	MaxCityLen     = 120
	MaxPositionLen = 160
	MaxEduLen      = 160
	MaxSkillLen    = 100
	MaxSkillsCount = 30
	MaxAboutLen    = 5000
	MinAge         = 14
	MaxAge         = 100
	MaxExperience  = 60.0
	MaxSalary      = 10000000
)

// IsAllowedGender проверяет пол по белому списку.
func IsAllowedGender(s string) bool {
	_, ok := allowedGenders[strings.TrimSpace(s)]
	return ok
}

// IsAllowedEmploymentType проверяет тип занятости по белому списку.
func IsAllowedEmploymentType(s string) bool {
	_, ok := allowedEmploymentTypes[strings.TrimSpace(s)]
	return ok
}

// NormalizeSort приводит сортировку к белому списку, неизвестное -> experience.
func NormalizeSort(s string) string {
	s = strings.TrimSpace(s)
	if _, ok := allowedSorts[s]; ok {
		return s
	}
	return "experience"
}

// IsValidSort сообщает, входит ли значение в белый список сортировок.
func IsValidSort(s string) bool {
	_, ok := allowedSorts[strings.TrimSpace(s)]
	return ok
}

// ParseOptionalInt отличает «поле не заполнено» от «поле заполнено мусором».
// Пустая строка -> (0, false, nil). Нечисловой ввод -> (0, true, err).
// Раньше strconv.Atoi с игнорированием ошибки превращал "abc" в 0 —
// правдоподобный, но неверный результат (классическая ошибка ввода).
func ParseOptionalInt(raw string) (int, bool, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, false, nil
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return 0, true, fmt.Errorf("ожидалось целое число, получено %q", raw)
	}
	return n, true, nil
}

// ParseOptionalFloat — то же для дробных чисел. Отклоняет NaN и Inf,
// т.к. ParseFloat("NaN") успешно возвращает NaN, который ломает сравнения и сортировку.
func ParseOptionalFloat(raw string) (float64, bool, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, false, nil
	}
	n, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, true, fmt.Errorf("ожидалось число, получено %q", raw)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, true, fmt.Errorf("число %q вне допустимого диапазона", raw)
	}
	return n, true, nil
}

// MustInt — строгий разбор обязательного целого поля формы.
func MustInt(raw, field string) (int, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, fmt.Errorf("поле %q не заполнено", field)
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return 0, fmt.Errorf("поле %q: ожидалось целое число, получено %q", field, raw)
	}
	return n, nil
}

// MustFloat — строгий разбор обязательного дробного поля формы.
func MustFloat(raw, field string) (float64, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, fmt.Errorf("поле %q не заполнено", field)
	}
	n, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, fmt.Errorf("поле %q: ожидалось число, получено %q", field, raw)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("поле %q вне допустимого диапазона", field)
	}
	return n, nil
}

// ParseID строго разбирает :id из URL. ID должен быть целым > 0.
// Раньше ошибка ParseUint молча превращалась в 0 и смешивалась с «не найдено».
func ParseID(raw string) (uint, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, fmt.Errorf("пустой идентификатор")
	}
	n, err := strconv.ParseUint(t, 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("некорректный идентификатор %q", raw)
	}
	return uint(n), nil
}

// ValidateEmail — минимальная проверка формата email.
func ValidateEmail(email string) error {
	e := strings.TrimSpace(email)
	if e == "" {
		return fmt.Errorf("email не заполнен")
	}
	if len(e) > 254 {
		return fmt.Errorf("email слишком длинный")
	}
	if strings.Contains(e, " ") {
		return fmt.Errorf("email не должен содержать пробелы")
	}
	if _, err := mail.ParseAddress(e); err != nil {
		return fmt.Errorf("некорректный email %q", email)
	}
	if !strings.Contains(strings.Split(e, "@")[len(strings.Split(e, "@"))-1], ".") {
		return fmt.Errorf("в email %q отсутствует доменная зона", email)
	}
	return nil
}

// ValidatePhone — минимальная проверка телефона: 6..32 символа, минимум 6 цифр.
func ValidatePhone(phone string) error {
	p := strings.TrimSpace(phone)
	if p == "" {
		return fmt.Errorf("телефон не заполнен")
	}
	if len(p) < 6 || len(p) > 32 {
		return fmt.Errorf("телефон должен быть длиной 6..32 символа")
	}
	digits := 0
	for _, r := range p {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' || r == '.':
		default:
			return fmt.Errorf("телефон содержит недопустимый символ %q", string(r))
		}
	}
	if digits < 6 {
		return fmt.Errorf("в телефоне должно быть минимум 6 цифр")
	}
	return nil
}

// ValidateSkillNames проверяет имена навыков: длина, количество.
func ValidateSkillNames(names []string) error {
	if len(names) > MaxSkillsCount {
		return fmt.Errorf("слишком много навыков: %d, максимум %d", len(names), MaxSkillsCount)
	}
	for _, n := range names {
		if len([]rune(n)) > MaxSkillLen {
			return fmt.Errorf("навык %q длиннее %d символов", n, MaxSkillLen)
		}
	}
	return nil
}

// ValidateCandidate — защитная проверка кандидата с детализацией по полям.
func ValidateCandidate(c models.Candidate) error {
	if len([]rune(strings.TrimSpace(c.FullName))) < 2 || len([]rune(c.FullName)) > MaxNameLen {
		return fmt.Errorf("ФИО должно быть длиной 2..%d символов", MaxNameLen)
	}
	if c.Age < MinAge || c.Age > MaxAge {
		return fmt.Errorf("возраст должен быть %d..%d, получено %d", MinAge, MaxAge, c.Age)
	}
	if !IsAllowedGender(c.Gender) {
		return fmt.Errorf("пол должен быть «Мужской» или «Женский», получено %q", c.Gender)
	}
	if err := ValidateEmail(c.Email); err != nil {
		return err
	}
	if err := ValidatePhone(c.Phone); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(c.City))) < 2 || len([]rune(c.City)) > MaxCityLen {
		return fmt.Errorf("город должен быть длиной 2..%d символов", MaxCityLen)
	}
	if len([]rune(strings.TrimSpace(c.Position))) < 2 || len([]rune(c.Position)) > MaxPositionLen {
		return fmt.Errorf("должность должна быть длиной 2..%d символов", MaxPositionLen)
	}
	if math.IsNaN(c.ExperienceYears) || math.IsInf(c.ExperienceYears, 0) ||
		c.ExperienceYears < 0 || c.ExperienceYears > MaxExperience {
		return fmt.Errorf("опыт должен быть 0..%.0f лет", MaxExperience)
	}
	if len([]rune(strings.TrimSpace(c.Education))) < 2 || len([]rune(c.Education)) > MaxEduLen {
		return fmt.Errorf("образование должно быть длиной 2..%d символов", MaxEduLen)
	}
	if !IsAllowedEmploymentType(c.EmploymentType) {
		return fmt.Errorf("тип занятости недопустим: %q", c.EmploymentType)
	}
	if c.SalaryExpectation < 0 || c.SalaryExpectation > MaxSalary {
		return fmt.Errorf("зарплата должна быть 0..%d", MaxSalary)
	}
	if len([]rune(c.About)) > MaxAboutLen {
		return fmt.Errorf("поле «О кандидате» длиннее %d символов", MaxAboutLen)
	}
	return nil
}

// ValidateVacancy — защитная проверка вакансии.
func ValidateVacancy(v models.Vacancy) error {
	if len([]rune(strings.TrimSpace(v.Title))) < 2 || len([]rune(v.Title)) > MaxNameLen {
		return fmt.Errorf("название вакансии должно быть длиной 2..%d символов", MaxNameLen)
	}
	if len([]rune(strings.TrimSpace(v.City))) < 2 || len([]rune(v.City)) > MaxCityLen {
		return fmt.Errorf("город вакансии должен быть длиной 2..%d символов", MaxCityLen)
	}
	if !IsAllowedEmploymentType(v.EmploymentType) {
		return fmt.Errorf("тип занятости вакансии недопустим: %q", v.EmploymentType)
	}
	if math.IsNaN(v.MinExperience) || math.IsInf(v.MinExperience, 0) ||
		v.MinExperience < 0 || v.MinExperience > MaxExperience {
		return fmt.Errorf("минимальный опыт должен быть 0..%.0f лет", MaxExperience)
	}
	if v.SalaryMin < 0 || v.SalaryMin > MaxSalary {
		return fmt.Errorf("зарплата «от» должна быть 0..%d", MaxSalary)
	}
	if v.SalaryMax < 0 || v.SalaryMax > MaxSalary {
		return fmt.Errorf("зарплата «до» должна быть 0..%d", MaxSalary)
	}
	if v.SalaryMax < v.SalaryMin {
		return fmt.Errorf("зарплата «до» (%d) меньше зарплаты «от» (%d)", v.SalaryMax, v.SalaryMin)
	}
	if len([]rune(v.Description)) > MaxAboutLen {
		return fmt.Errorf("описание вакансии длиннее %d символов", MaxAboutLen)
	}
	if len([]rune(v.Requirements)) > MaxAboutLen {
		return fmt.Errorf("требования вакансии длиннее %d символов", MaxAboutLen)
	}
	return nil
}

// EscapeLike экранирует спецсимволы LIKE/ILIKE: \, %, _.
// Без этого ввод "%" или "_" работает как маска и возвращает правдоподобные,
// но неверные выборки (ошибка обработки данных).
func EscapeLike(s string) string {
	r := strings.ReplaceAll(s, `\`, `\\`)
	r = strings.ReplaceAll(r, "%", `\%`)
	r = strings.ReplaceAll(r, "_", `\_`)
	return r
}
