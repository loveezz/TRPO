package seed

import (
	"hrselect/internal/models"
	"hrselect/internal/repository"
	"hrselect/internal/service"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	if err := ensureAdmin(db); err != nil {
		return err
	}

	var count int64
	if err := db.Model(&models.Candidate{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	repo := &repository.Repository{DB: db}
	candidates := []struct {
		Candidate models.Candidate
		Skills    []string
	}{
		{models.Candidate{FullName: "Иван Петров", Age: 25, Gender: "Мужской", Email: "ivan@example.com", Phone: "+7 900 111-22-33", City: "Екатеринбург", Position: "Python Backend Developer", ExperienceYears: 3.5, Education: "Высшее", EmploymentType: "Полная занятость", SalaryExpectation: 160000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "FastAPI", "PostgreSQL", "Docker", "Git", "REST API", "Linux"}},
		{models.Candidate{FullName: "Мария Соколова", Age: 28, Gender: "Женский", Email: "m.sokolova@example.com", Phone: "+7 900 222-33-44", City: "Москва", Position: "Backend Developer", ExperienceYears: 5.0, Education: "Высшее", EmploymentType: "Удаленно", SalaryExpectation: 220000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "Django", "PostgreSQL", "Redis", "Docker", "Git"}},
		{models.Candidate{FullName: "Алексей Морозов", Age: 23, Gender: "Мужской", Email: "a.morozov@example.com", Phone: "+7 900 333-44-55", City: "Екатеринбург", Position: "Junior Python Developer", ExperienceYears: 1.5, Education: "Среднее профессиональное", EmploymentType: "Полная занятость", SalaryExpectation: 100000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "SQL", "Git", "Linux"}},
		{models.Candidate{FullName: "Елена Волкова", Age: 31, Gender: "Женский", Email: "e.volkova@example.com", Phone: "+7 900 444-55-66", City: "Казань", Position: "Senior Backend Developer", ExperienceYears: 7.0, Education: "Высшее", EmploymentType: "Удаленно", SalaryExpectation: 280000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "FastAPI", "PostgreSQL", "Redis", "Docker", "Git", "REST API", "Linux"}},
		{models.Candidate{FullName: "Дмитрий Орлов", Age: 27, Gender: "Мужской", Email: "d.orlov@example.com", Phone: "+7 900 555-66-77", City: "Екатеринбург", Position: "DevOps / Backend", ExperienceYears: 4.0, Education: "Высшее", EmploymentType: "Полная занятость", SalaryExpectation: 190000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "PostgreSQL", "Docker", "Git", "Linux"}},
		{models.Candidate{FullName: "Ольга Белова", Age: 26, Gender: "Женский", Email: "o.belova@example.com", Phone: "+7 900 666-77-88", City: "Пермь", Position: "Python Developer", ExperienceYears: 3.0, Education: "Высшее", EmploymentType: "Удаленно", SalaryExpectation: 170000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "FastAPI", "SQL", "Docker", "Git"}},
		{models.Candidate{FullName: "Сергей Кузнецов", Age: 35, Gender: "Мужской", Email: "s.kuznetsov@example.com", Phone: "+7 900 777-88-99", City: "Москва", Position: "Team Lead", ExperienceYears: 10.0, Education: "Высшее", EmploymentType: "Удаленно", SalaryExpectation: 350000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "FastAPI", "PostgreSQL", "Redis", "Docker", "Git", "REST API", "Linux"}},
		{models.Candidate{FullName: "Наталья Федорова", Age: 24, Gender: "Женский", Email: "n.fedorova@example.com", Phone: "+7 900 888-99-00", City: "Екатеринбург", Position: "Web Developer", ExperienceYears: 2.0, Education: "Высшее", EmploymentType: "Частичная занятость", SalaryExpectation: 120000, About: "Кандидат из демонстрационного набора данных."}, []string{"Python", "HTML/CSS", "SQL", "Git"}},
	}
	for i := range candidates {
		if err := repo.SaveCandidate(&candidates[i].Candidate, candidates[i].Skills); err != nil {
			return err
		}
	}

	vacancies := []struct {
		Vacancy models.Vacancy
		Skills  []string
	}{
		{models.Vacancy{Title: "Python Backend Developer", Department: "Разработка", City: "Екатеринбург", EmploymentType: "Полная занятость", MinExperience: 2.0, EducationLevel: "Высшее", SalaryMin: 140000, SalaryMax: 220000, Description: "Разработка API для внутренней HR-платформы.", Requirements: "Опыт Python, FastAPI, PostgreSQL, Docker, Git.", IsActive: true}, []string{"Python", "FastAPI", "PostgreSQL", "Docker", "Git", "REST API"}},
		{models.Vacancy{Title: "Python Developer (Remote)", Department: "R&D", City: "Удаленно", EmploymentType: "Удаленно", MinExperience: 3.0, EducationLevel: "", SalaryMin: 160000, SalaryMax: 260000, Description: "Backend-разработка удаленного продукта.", Requirements: "Python, SQL, Docker, Git.", IsActive: true}, []string{"Python", "SQL", "Docker", "Git"}},
	}
	for i := range vacancies {
		if err := repo.SaveVacancy(&vacancies[i].Vacancy, vacancies[i].Skills); err != nil {
			return err
		}
	}
	return nil
}

func ensureAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Where("LOWER(email) = LOWER(?)", "admin@example.com").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := service.HashPassword("admin123")
	if err != nil {
		return err
	}
	return db.Create(&models.User{
		FullName:     "Администратор",
		Email:        "admin@example.com",
		PasswordHash: hash,
	}).Error
}
