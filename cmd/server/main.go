package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"hrselect/internal/config"
	"hrselect/internal/database"
	"hrselect/internal/handlers"
	"hrselect/internal/middleware"
	"hrselect/internal/models"
	"hrselect/internal/repository"
	"hrselect/internal/seed"
	"hrselect/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
)

func main() {
	cfg := config.Load()
	db, err := database.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Candidate{}, &models.Vacancy{}, &models.Skill{}); err != nil {
		log.Fatal(err)
	}
	if err := seed.Run(db); err != nil {
		log.Fatal(err)
	}

	engine := html.New(templateDir(), ".html")
	engine.AddFunc("joinReasons", func(items []string) string {
		out := ""
		for i, s := range items {
			if i > 0 {
				out += " · "
			}
			out += s
		}
		return out
	})
	engine.AddFunc("formatSalary", formatSalary)
	engine.AddFunc("inc", func(i int) int { return i + 1 })

	app := fiber.New(fiber.Config{Views: engine, AppName: "Candidate Matcher"})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Static("/static", staticDir())

	repo := &repository.Repository{DB: db}
	sessions := service.NewSessionManager(cfg.SessionSecret)
	authMW := middleware.Auth{Repo: repo, Sessions: sessions}
	h := &handlers.App{Repo: repo, Sessions: sessions}

	app.Use(authMW.LoadUser)
	app.Get("/", h.Home)
	app.Get("/login", h.LoginPage)
	app.Post("/login", h.Login)
	app.Post("/logout", h.Logout)

	protected := app.Group("", authMW.RequireLogin)
	protected.Get("/dashboard", h.Dashboard)
	protected.Get("/candidates", h.Candidates)
	protected.Get("/candidates/new", h.NewCandidatePage)
	protected.Post("/candidates/new", h.CreateCandidate)
	protected.Get("/candidates/:id/edit", h.EditCandidatePage)
	protected.Post("/candidates/:id/edit", h.UpdateCandidate)
	protected.Post("/candidates/:id/delete", h.DeleteCandidate)
	protected.Get("/vacancies", h.Vacancies)
	protected.Get("/vacancies/new", h.NewVacancyPage)
	protected.Post("/vacancies/new", h.CreateVacancy)
	protected.Get("/vacancies/:id/edit", h.EditVacancyPage)
	protected.Post("/vacancies/:id/edit", h.UpdateVacancy)
	protected.Post("/vacancies/:id/delete", h.DeleteVacancy)
	protected.Post("/vacancies/:id/toggle", h.ToggleVacancy)
	protected.Get("/vacancies/:id/match", h.MatchVacancy)

	log.Printf("Server started: http://localhost:%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func templateDir() string {
	if _, err := os.Stat("web/templates"); err == nil {
		return "web/templates"
	}
	return filepath.Join("/app", "web", "templates")
}

func staticDir() string {
	if _, err := os.Stat("web/static"); err == nil {
		return "web/static"
	}
	return filepath.Join("/app", "web", "static")
}

func formatSalary(v int) string {
	// Защитная проверка выходного данного: отрицательная зарплата
	// из поврежденных строк не должна ломать группировку разрядов.
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	s := fmt.Sprintf("%d", v)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + " " + s[i:]
	}
	return sign + s
}
