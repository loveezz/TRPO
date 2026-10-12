package repository

import (
	"errors"
	"log"
	"sort"
	"strings"

	"gorm.io/gorm"
	"hrselect/internal/models"
	"hrselect/internal/service"
)

type Repository struct {
	DB *gorm.DB
}

type CandidateFilter struct {
	Name           string
	City           string
	Position       string
	Skills         string
	AgeMin         int
	AgeMax         int
	Gender         string
	ExperienceMin  float64
	Education      string
	EmploymentType string
	SalaryMax      int
	Sort           string
}

type CandidateSearchResult struct {
	Candidate       models.Candidate
	SkillMatchCount int
}

func (r *Repository) FindUserByEmail(email string) (models.User, error) {
	var u models.User
	err := r.DB.Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email)).First(&u).Error
	return u, err
}

func (r *Repository) FindUserByID(id uint) (models.User, error) {
	var u models.User
	err := r.DB.First(&u, id).Error
	return u, err
}

func (r *Repository) DashboardStats() (map[string]int64, error) {
	stats := map[string]int64{}
	var candidates int64
	if err := r.DB.Model(&models.Candidate{}).Count(&candidates).Error; err != nil {
		return nil, err
	}
	stats["candidates"] = candidates
	var active int64
	if err := r.DB.Model(&models.Vacancy{}).Where("is_active = ?", true).Count(&active).Error; err != nil {
		return nil, err
	}
	stats["vacancies"] = active
	return stats, nil
}

func (r *Repository) SearchCandidates(f CandidateFilter) ([]CandidateSearchResult, error) {
	// Защитная нормализация входов фильтра: TrimSpace + экранирование LIKE.
	// Печать отброшенных/исправленных значений — требование локальных проверок (п. 5.1).
	f.Name = strings.TrimSpace(f.Name)
	f.City = strings.TrimSpace(f.City)
	f.Position = strings.TrimSpace(f.Position)
	f.Skills = strings.TrimSpace(f.Skills)
	f.Gender = strings.TrimSpace(f.Gender)
	f.Education = strings.TrimSpace(f.Education)
	f.EmploymentType = strings.TrimSpace(f.EmploymentType)
	f.Sort = service.NormalizeSort(f.Sort)

	// Некорректный диапазон возраста раньше давал пустую правдоподобную выборку.
	// Локализация: меняем границы местами и печатаем диагностику.
	ageMin, ageMax := f.AgeMin, f.AgeMax
	if ageMin > 0 && ageMax > 0 && ageMin > ageMax {
		log.Printf("[WARN] SearchCandidates: age_min=%d > age_max=%d, границы переставлены", ageMin, ageMax)
		ageMin, ageMax = ageMax, ageMin
	}

	// Белые списки: произвольный gender/employment_type из URL раньше давал
	// пустую выборку без диагностики. Теперь игнорируем с печатью переменной.
	if f.Gender != "" && !service.IsAllowedGender(f.Gender) {
		log.Printf("[WARN] SearchCandidates: gender=%q не из белого списка, фильтр проигнорирован", f.Gender)
		f.Gender = ""
	}
	if f.EmploymentType != "" && !service.IsAllowedEmploymentType(f.EmploymentType) {
		log.Printf("[WARN] SearchCandidates: employment_type=%q не из белого списка, фильтр проигнорирован", f.EmploymentType)
		f.EmploymentType = ""
	}
	if f.ExperienceMin < 0 {
		log.Printf("[WARN] SearchCandidates: experience_min=%v < 0, фильтр проигнорирован", f.ExperienceMin)
		f.ExperienceMin = 0
	}
	if f.SalaryMax < 0 {
		log.Printf("[WARN] SearchCandidates: salary_max=%d < 0, фильтр проигнорирован", f.SalaryMax)
		f.SalaryMax = 0
	}

	q := r.DB.Model(&models.Candidate{}).Preload("Skills")
	if f.Name != "" {
		q = q.Where("full_name ILIKE ? ESCAPE '\\'", "%"+service.EscapeLike(f.Name)+"%")
	}
	if f.City != "" {
		q = q.Where("city ILIKE ? ESCAPE '\\'", "%"+service.EscapeLike(f.City)+"%")
	}
	if f.Position != "" {
		q = q.Where("position ILIKE ? ESCAPE '\\'", "%"+service.EscapeLike(f.Position)+"%")
	}
	if ageMin > 0 {
		q = q.Where("age >= ?", ageMin)
	}
	if ageMax > 0 {
		q = q.Where("age <= ?", ageMax)
	}
	if f.Gender != "" {
		q = q.Where("gender = ?", f.Gender)
	}
	if f.ExperienceMin > 0 {
		q = q.Where("experience_years >= ?", f.ExperienceMin)
	}
	if f.Education != "" {
		q = q.Where("education ILIKE ? ESCAPE '\\'", "%"+service.EscapeLike(f.Education)+"%")
	}
	if f.EmploymentType != "" {
		q = q.Where("employment_type = ?", f.EmploymentType)
	}
	if f.SalaryMax > 0 {
		q = q.Where("salary_expectation <= ?", f.SalaryMax)
	}

	var candidates []models.Candidate
	if err := q.Find(&candidates).Error; err != nil {
		return nil, err
	}

	requested := service.NormalizeSkills(f.Skills)
	results := make([]CandidateSearchResult, 0, len(candidates))
	for _, c := range candidates {
		matches := service.CountSkillMatches(c.Skills, requested)
		if len(requested) > 0 && matches == 0 {
			continue
		}
		results = append(results, CandidateSearchResult{Candidate: c, SkillMatchCount: matches})
	}

	switch f.Sort {
	case "skills":
		sort.SliceStable(results, func(i, j int) bool {
			if results[i].SkillMatchCount == results[j].SkillMatchCount {
				return results[i].Candidate.ExperienceYears > results[j].Candidate.ExperienceYears
			}
			return results[i].SkillMatchCount > results[j].SkillMatchCount
		})
	case "salary":
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Candidate.SalaryExpectation < results[j].Candidate.SalaryExpectation
		})
	case "name":
		sort.SliceStable(results, func(i, j int) bool { return results[i].Candidate.FullName < results[j].Candidate.FullName })
	default:
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Candidate.ExperienceYears > results[j].Candidate.ExperienceYears
		})
	}
	return results, nil
}

func (r *Repository) ListCandidates() ([]models.Candidate, error) {
	var items []models.Candidate
	err := r.DB.Preload("Skills").Order("experience_years DESC").Find(&items).Error
	return items, err
}

func (r *Repository) CandidateByID(id uint) (models.Candidate, error) {
	var c models.Candidate
	err := r.DB.Preload("Skills").First(&c, id).Error
	return c, err
}

func (r *Repository) SaveCandidate(c *models.Candidate, skillNames []string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(c).Error; err != nil {
			return err
		}
		skills, err := ensureSkills(tx, skillNames)
		if err != nil {
			return err
		}
		return tx.Model(c).Association("Skills").Replace(skills)
	})
}

func (r *Repository) DeleteCandidate(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var c models.Candidate
		if err := tx.First(&c, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&c).Association("Skills").Clear(); err != nil {
			return err
		}
		return tx.Delete(&c).Error
	})
}

func (r *Repository) ListVacancies() ([]models.Vacancy, error) {
	var items []models.Vacancy
	err := r.DB.Preload("Skills").Order("is_active DESC, created_at DESC").Find(&items).Error
	return items, err
}

func (r *Repository) VacancyByID(id uint) (models.Vacancy, error) {
	var v models.Vacancy
	err := r.DB.Preload("Skills").First(&v, id).Error
	return v, err
}

func (r *Repository) SaveVacancy(v *models.Vacancy, skillNames []string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(v).Error; err != nil {
			return err
		}
		skills, err := ensureSkills(tx, skillNames)
		if err != nil {
			return err
		}
		return tx.Model(v).Association("Skills").Replace(skills)
	})
}

func (r *Repository) DeleteVacancy(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var v models.Vacancy
		if err := tx.First(&v, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&v).Association("Skills").Clear(); err != nil {
			return err
		}
		return tx.Delete(&v).Error
	})
}

func (r *Repository) ToggleVacancy(id uint) error {
	var v models.Vacancy
	if err := r.DB.First(&v, id).Error; err != nil {
		return err
	}
	return r.DB.Model(&v).Update("is_active", !v.IsActive).Error
}

func ensureSkills(tx *gorm.DB, names []string) ([]models.Skill, error) {
	clean := service.NormalizeSkills(strings.Join(names, ","))
	skills := make([]models.Skill, 0, len(clean))
	for _, name := range clean {
		var s models.Skill
		err := tx.Where("LOWER(name) = LOWER(?)", name).First(&s).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s = models.Skill{Name: name}
			if err := tx.Create(&s).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, nil
}
