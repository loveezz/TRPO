package middleware

import (
	"hrselect/internal/repository"
	"hrselect/internal/service"

	"github.com/gofiber/fiber/v2"
)

type Auth struct {
	Repo     *repository.Repository
	Sessions *service.SessionManager
}

func (a Auth) LoadUser(c *fiber.Ctx) error {
	token := c.Cookies("hr_session")
	if token != "" {
		if id, ok := a.Sessions.Verify(token); ok {
			if user, err := a.Repo.FindUserByID(id); err == nil {
				c.Locals("user", user)
			}
		}
	}
	return c.Next()
}

func (a Auth) RequireLogin(c *fiber.Ctx) error {
	if c.Locals("user") == nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}
	return c.Next()
}
