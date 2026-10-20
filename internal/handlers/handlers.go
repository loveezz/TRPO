package handlers

import (
	"errors"
	"log"
	"strings"

	"hrselect/internal/models"
	"hrselect/internal/repository"
	"hrselect/internal/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type App struct {
	Repo     *repository.Repository
	Sessions *service.SessionManager
}

func (a *App) data(c *fiber.Ctx, title string) fiber.Map {
	data := fiber.Map{"Title": title}
	if u := c.Locals("user"); u != nil {
		data["User"] = u.(models.User)
	}
	return data
}

// parseFilterInt — терпимый разбор чисел фильтра: пусто -> 0 без ошибки,
// мусор -> 0 + печать переменной (локальная проверка по п. 5.1) и игнорирование фильтра.
// Раньше toInt молча превращал "abc" в 0 без диагностики.
func parseFilterInt(raw, field string) int {
	v, present, err := service.ParseOptionalInt(raw)
	if err != nil {
		log.Printf("[WARN] фильтр %s=%q некорректен (%v), значение проигнорировано", field, raw, err)
		return 0
	}
	if present {
		return v
	}
	return 0
}

func parseFilterFloat(raw, field string) float64 {
	v, present, err := service.ParseOptionalFloat(raw)
	if err != nil {
		log.Printf("[WARN] фильтр %s=%q некорректен (%v), значение проигнорировано", field, raw, err)
		return 0
	}
	if present {
		return v
	}
	return 0
}

// parseID — строгий разбор :id. Ошибка ввода -> 400, а не 404.
func parseID(c *fiber.Ctx) (uint, error) {
	id, err := service.ParseID(c.Params("id"))
	if err != nil {
		log.Printf("[WARN] некорректный id в %s: raw=%q err=%v", c.Path(), c.Params("id"), err)
		return 0, err
	}
	return id, nil
}

func internalError(c *fiber.Ctx, err error, where string) error {
	// Защитное программирование: внутренние ошибки БД не отдаем наружу,
	// но печатаем их с контекстом для локализации.
	log.Printf("[ERROR] %s: %v", where, err)
	return fiber.ErrInternalServerError
}

// lookupError отделяет «записи нет» (404) от сбоя БД (500).
// Раньше любая ошибка поиска превращалась в 404, а удаление несуществующей
// записи — в 500, что маскировало настоящую причину.
func lookupError(c *fiber.Ctx, err error, where string, id uint) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[WARN] %s: запись id=%d не найдена", where, id)
		return fiber.ErrNotFound
	}
	return internalError(c, err, where)
}

func (a *App) Home(c *fiber.Ctx) error {
	if c.Locals("user") == nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}
	return c.Redirect("/dashboard", fiber.StatusSeeOther)
}

func (a *App) LoginPage(c *fiber.Ctx) error {
	if c.Locals("user") != nil {
		return c.Redirect("/dashboard", fiber.StatusSeeOther)
	}
	return c.Render("login", fiber.Map{"Title": "Вход"})
}

func (a *App) Login(c *fiber.Ctx) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	if email == "" || password == "" {
		log.Printf("[WARN] Login: пустой email или пароль")
		return c.Status(fiber.StatusUnauthorized).Render("login", fiber.Map{
			"Title": "Вход",
			"Error": "Введите email и пароль.",
			"Email": email,
		})
	}
	if err := service.ValidateEmail(email); err != nil {
		log.Printf("[WARN] Login: email=%q не прошел проверку: %v", email, err)
		return c.Status(fiber.StatusUnauthorized).Render("login", fiber.Map{
			"Title": "Вход",
			"Error": "Неверный email или пароль.",
			"Email": email,
		})
	}
	user, err := a.Repo.FindUserByEmail(email)
	if err != nil || !service.CheckPassword(user.PasswordHash, password) {
		log.Printf("[WARN] Login: неудачная попытка для email=%q", email)
		return c.Status(fiber.StatusUnauthorized).Render("login", fiber.Map{
			"Title": "Вход",
			"Error": "Неверный email или пароль.",
			"Email": email,
		})
	}
	c.Cookie(&fiber.Cookie{
		Name:     "hr_session",
		Value:    a.Sessions.Sign(user.ID),
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		MaxAge:   24 * 60 * 60,
	})
	return c.Redirect("/dashboard", fiber.StatusSeeOther)
}

func (a *App) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{Name: "hr_session", Value: "", Path: "/", HTTPOnly: true, MaxAge: -1})
	return c.Redirect("/login", fiber.StatusSeeOther)
}

func (a *App) Dashboard(c *fiber.Ctx) error {
	stats, err := a.Repo.DashboardStats()
	if err != nil {
		return internalError(c, err, "DashboardStats")
	}
	data := a.data(c, "Обзор")
	data["Stats"] = stats
	return c.Render("dashboard", data)
}

func (a *App) Candidates(c *fiber.Ctx) error {
	f := repository.CandidateFilter{
		Name: strings.TrimSpace(c.Query("name")), City: strings.TrimSpace(c.Query("city")),
		Position: strings.TrimSpace(c.Query("position")), Skills: strings.TrimSpace(c.Query("skills")),
		AgeMin: parseFilterInt(c.Query("age_min"), "age_min"), AgeMax: parseFilterInt(c.Query("age_max"), "age_max"),
		Gender:        strings.TrimSpace(c.Query("gender")),
		ExperienceMin: parseFilterFloat(c.Query("experience_min"), "experience_min"),
		Education:     strings.TrimSpace(c.Query("education")), EmploymentType: strings.TrimSpace(c.Query("employment_type")),
		SalaryMax: parseFilterInt(c.Query("salary_max"), "salary_max"), Sort: strings.TrimSpace(c.Query("sort", "experience")),
	}
	if raw := c.Query("sort", "experience"); !service.IsValidSort(raw) && strings.TrimSpace(raw) != "" {
		log.Printf("[WARN] Candidates: sort=%q неизвестна, применена сортировка experience", raw)
	}
	items, err := a.Repo.SearchCandidates(f)
	if err != nil {
		return internalError(c, err, "SearchCandidates")
	}
	data := a.data(c, "Кандидаты")
	data["Candidates"] = items
	data["Filters"] = f
	return c.Render("candidates", data)
}

func (a *App) NewCandidatePage(c *fiber.Ctx) error {
	data := a.data(c, "Новый кандидат")
	data["IsEdit"] = false
	return c.Render("candidate_form", data)
}

func (a *App) EditCandidatePage(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор кандидата.")
	}
	candidate, err := a.Repo.CandidateByID(id)
	if err != nil {
		return lookupError(c, err, "EditCandidatePage", id)
	}
	data := a.data(c, "Редактирование кандидата")
	data["Candidate"] = candidate
	data["SkillNames"] = joinSkillNames(candidate.Skills)
	data["IsEdit"] = true
	return c.Render("candidate_form", data)
}

func (a *App) CreateCandidate(c *fiber.Ctx) error {
	candidate, skillNames, err := candidateFromForm(c)
	if err != nil {
		log.Printf("[WARN] CreateCandidate: %v (full_name=%q age=%q email=%q)",
			err, c.FormValue("full_name"), c.FormValue("age"), c.FormValue("email"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := validateCandidate(candidate); err != nil {
		log.Printf("[WARN] CreateCandidate validate: %v (candidate=%+v)", err, candidate)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := service.ValidateSkillNames(skillNames); err != nil {
		log.Printf("[WARN] CreateCandidate skills: %v raw=%q", err, c.FormValue("skills"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := a.Repo.SaveCandidate(&candidate, skillNames); err != nil {
		return internalError(c, err, "SaveCandidate")
	}
	return c.Redirect("/candidates", fiber.StatusSeeOther)
}

func (a *App) UpdateCandidate(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор кандидата.")
	}
	candidate, err := a.Repo.CandidateByID(id)
	if err != nil {
		return lookupError(c, err, "UpdateCandidate", id)
	}
	if err := applyCandidateForm(&candidate, c); err != nil {
		log.Printf("[WARN] UpdateCandidate id=%d: %v", id, err)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := validateCandidate(candidate); err != nil {
		log.Printf("[WARN] UpdateCandidate id=%d validate: %v (candidate=%+v)", id, err, candidate)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	skillNames := service.NormalizeSkills(c.FormValue("skills"))
	if err := service.ValidateSkillNames(skillNames); err != nil {
		log.Printf("[WARN] UpdateCandidate id=%d skills: %v raw=%q", id, err, c.FormValue("skills"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := a.Repo.SaveCandidate(&candidate, skillNames); err != nil {
		return internalError(c, err, "SaveCandidate")
	}
	return c.Redirect("/candidates", fiber.StatusSeeOther)
}

func (a *App) DeleteCandidate(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор кандидата.")
	}
	if err := a.Repo.DeleteCandidate(id); err != nil {
		return lookupError(c, err, "DeleteCandidate", id)
	}
	return c.Redirect("/candidates", fiber.StatusSeeOther)
}

// candidateFromForm строго разбирает форму: числовой мусор -> 400, а не 0.
func candidateFromForm(c *fiber.Ctx) (models.Candidate, []string, error) {
	var candidate models.Candidate
	if err := applyCandidateForm(&candidate, c); err != nil {
		return models.Candidate{}, nil, err
	}
	skillNames := service.NormalizeSkills(c.FormValue("skills"))
	return candidate, skillNames, nil
}

func applyCandidateForm(candidate *models.Candidate, c *fiber.Ctx) error {
	age, err := service.MustInt(c.FormValue("age"), "возраст")
	if err != nil {
		return err
	}
	exp, err := service.MustFloat(c.FormValue("experience_years"), "опыт")
	if err != nil {
		return err
	}
	salary, err := service.MustInt(c.FormValue("salary_expectation"), "зарплата")
	if err != nil {
		return err
	}
	candidate.FullName = strings.TrimSpace(c.FormValue("full_name"))
	candidate.Age = age
	candidate.Gender = strings.TrimSpace(c.FormValue("gender"))
	candidate.Email = strings.TrimSpace(c.FormValue("email"))
	candidate.Phone = strings.TrimSpace(c.FormValue("phone"))
	candidate.City = strings.TrimSpace(c.FormValue("city"))
	candidate.Position = strings.TrimSpace(c.FormValue("position"))
	candidate.ExperienceYears = exp
	candidate.Education = strings.TrimSpace(c.FormValue("education"))
	candidate.EmploymentType = strings.TrimSpace(c.FormValue("employment_type"))
	candidate.SalaryExpectation = salary
	candidate.About = strings.TrimSpace(c.FormValue("about"))
	return nil
}

func validateCandidate(c models.Candidate) error {
	return service.ValidateCandidate(c)
}

func (a *App) Vacancies(c *fiber.Ctx) error {
	items, err := a.Repo.ListVacancies()
	if err != nil {
		return internalError(c, err, "ListVacancies")
	}
	data := a.data(c, "Вакансии")
	data["Vacancies"] = items
	return c.Render("vacancies", data)
}

func (a *App) NewVacancyPage(c *fiber.Ctx) error {
	data := a.data(c, "Новая вакансия")
	data["IsEdit"] = false
	data["DefaultSkills"] = "Python, FastAPI, PostgreSQL, Docker, Git"
	return c.Render("vacancy_form", data)
}

func (a *App) EditVacancyPage(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор вакансии.")
	}
	v, err := a.Repo.VacancyByID(id)
	if err != nil {
		return lookupError(c, err, "EditVacancyPage", id)
	}
	data := a.data(c, "Редактирование вакансии")
	data["Vacancy"] = v
	data["SkillNames"] = joinSkillNames(v.Skills)
	data["IsEdit"] = true
	return c.Render("vacancy_form", data)
}

func (a *App) CreateVacancy(c *fiber.Ctx) error {
	v, skillNames, err := vacancyFromForm(c)
	if err != nil {
		log.Printf("[WARN] CreateVacancy: %v (title=%q)", err, c.FormValue("title"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := validateVacancy(v); err != nil {
		log.Printf("[WARN] CreateVacancy validate: %v (vacancy=%+v)", err, v)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := service.ValidateSkillNames(skillNames); err != nil {
		log.Printf("[WARN] CreateVacancy skills: %v raw=%q", err, c.FormValue("skills"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := a.Repo.SaveVacancy(&v, skillNames); err != nil {
		return internalError(c, err, "SaveVacancy")
	}
	return c.Redirect("/vacancies", fiber.StatusSeeOther)
}

func (a *App) UpdateVacancy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор вакансии.")
	}
	v, err := a.Repo.VacancyByID(id)
	if err != nil {
		return lookupError(c, err, "UpdateVacancy", id)
	}
	if err := applyVacancyForm(&v, c); err != nil {
		log.Printf("[WARN] UpdateVacancy id=%d: %v", id, err)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := validateVacancy(v); err != nil {
		log.Printf("[WARN] UpdateVacancy id=%d validate: %v (vacancy=%+v)", id, err, v)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	skillNames := service.NormalizeSkills(c.FormValue("skills"))
	if err := service.ValidateSkillNames(skillNames); err != nil {
		log.Printf("[WARN] UpdateVacancy id=%d skills: %v raw=%q", id, err, c.FormValue("skills"))
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := a.Repo.SaveVacancy(&v, skillNames); err != nil {
		return internalError(c, err, "SaveVacancy")
	}
	return c.Redirect("/vacancies", fiber.StatusSeeOther)
}

func (a *App) DeleteVacancy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор вакансии.")
	}
	if err := a.Repo.DeleteVacancy(id); err != nil {
		return lookupError(c, err, "DeleteVacancy", id)
	}
	return c.Redirect("/vacancies", fiber.StatusSeeOther)
}

func (a *App) ToggleVacancy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор вакансии.")
	}
	if err := a.Repo.ToggleVacancy(id); err != nil {
		return lookupError(c, err, "ToggleVacancy", id)
	}
	return c.Redirect("/vacancies", fiber.StatusSeeOther)
}

func vacancyFromForm(c *fiber.Ctx) (models.Vacancy, []string, error) {
	v := models.Vacancy{IsActive: true}
	if err := applyVacancyForm(&v, c); err != nil {
		return models.Vacancy{}, nil, err
	}
	return v, service.NormalizeSkills(c.FormValue("skills")), nil
}

func applyVacancyForm(v *models.Vacancy, c *fiber.Ctx) error {
	minExpRaw := strings.TrimSpace(c.FormValue("min_experience"))
	var minExp float64
	if minExpRaw == "" {
		minExp = 0
	} else {
		var err error
		minExp, err = service.MustFloat(minExpRaw, "минимальный опыт")
		if err != nil {
			return err
		}
	}
	salaryMinRaw := strings.TrimSpace(c.FormValue("salary_min"))
	salaryMaxRaw := strings.TrimSpace(c.FormValue("salary_max"))
	var salaryMin, salaryMax int
	if salaryMinRaw == "" {
		salaryMin = 0
	} else {
		var err error
		salaryMin, err = service.MustInt(salaryMinRaw, "зарплата от")
		if err != nil {
			return err
		}
	}
	if salaryMaxRaw == "" {
		salaryMax = 0
	} else {
		var err error
		salaryMax, err = service.MustInt(salaryMaxRaw, "зарплата до")
		if err != nil {
			return err
		}
	}
	v.Title = strings.TrimSpace(c.FormValue("title"))
	v.Department = strings.TrimSpace(c.FormValue("department"))
	v.City = strings.TrimSpace(c.FormValue("city"))
	v.EmploymentType = strings.TrimSpace(c.FormValue("employment_type"))
	v.MinExperience = minExp
	v.EducationLevel = strings.TrimSpace(c.FormValue("education_level"))
	v.SalaryMin = salaryMin
	v.SalaryMax = salaryMax
	v.Description = strings.TrimSpace(c.FormValue("description"))
	v.Requirements = strings.TrimSpace(c.FormValue("requirements"))
	return nil
}

func validateVacancy(v models.Vacancy) error {
	return service.ValidateVacancy(v)
}

// defaultMatchLimit — сколько лучших кандидатов показывать при подборе,
// если параметр ?top= не задан (top=0 — показать всех).
const defaultMatchLimit = 20

func (a *App) MatchVacancy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный идентификатор вакансии.")
	}
	v, err := a.Repo.VacancyByID(id)
	if err != nil {
		return lookupError(c, err, "MatchVacancy", id)
	}
	limit := parseFilterInt(c.Query("top"), "top")
	if limit < 0 {
		log.Printf("[WARN] MatchVacancy: top=%d < 0, выводятся все кандидаты", limit)
		limit = 0
	}
	if c.Query("top") == "" {
		limit = defaultMatchLimit
	}
	candidates, err := a.Repo.CandidatesForMatching()
	if err != nil {
		return internalError(c, err, "CandidatesForMatching")
	}
	results := service.RankTopCandidates(candidates, v, limit)
	data := a.data(c, "Подбор")
	data["Vacancy"] = v
	data["Results"] = results
	data["Total"] = len(candidates)
	data["Limit"] = limit
	return c.Render("matching", data)
}

func joinSkillNames(skills []models.Skill) string {
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}
