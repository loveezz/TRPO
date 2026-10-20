package handlers

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"hrselect/internal/models"
	"hrselect/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

// Проверка шаблона страницы подбора: выдача top-N и счетчик кандидатов.
func TestMatchingTemplateRendersTopN(t *testing.T) {
	engine := html.New("../../web/templates", ".html")
	engine.AddFunc("joinReasons", func(items []string) string { return strings.Join(items, " · ") })
	engine.AddFunc("formatSalary", func(v int) string { return "" })
	engine.AddFunc("inc", func(i int) int { return i + 1 })
	app := fiber.New(fiber.Config{Views: engine})

	v := models.Vacancy{Title: "Go Developer", City: "Екатеринбург", Skills: []models.Skill{{Name: "Go"}}}
	candidates := []models.Candidate{
		{ID: 1, FullName: "Анна", City: "Екатеринбург", ExperienceYears: 2, Skills: []models.Skill{{Name: "Go"}}},
		{ID: 2, FullName: "Борис", City: "Москва", ExperienceYears: 5},
		{ID: 3, FullName: "Вера", City: "Пермь", ExperienceYears: 1},
	}
	app.Get("/m", func(c *fiber.Ctx) error {
		return c.Render("matching", fiber.Map{
			"Title": "Подбор", "Vacancy": v, "Total": len(candidates), "Limit": 2,
			"Results": service.RankTopCandidates(candidates, v, 2),
		})
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/m", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	page := string(body)
	if resp.StatusCode != 200 {
		t.Fatalf("код ответа %d: %s", resp.StatusCode, page)
	}
	for _, want := range []string{"Показано 2 из 3 кандидатов", "Анна", "Борис", `value="20"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("на странице нет %q", want)
		}
	}
	if strings.Contains(page, "Вера") {
		t.Fatalf("кандидат вне top-2 не должен выводиться")
	}
	if strings.Index(page, "Анна") > strings.Index(page, "Борис") {
		t.Fatalf("Анна (больше баллов) должна быть выше Бориса")
	}
}
