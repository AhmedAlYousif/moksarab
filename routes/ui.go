package routes

import (
	"github.com/gofiber/fiber/v2"
)

func RegesiterUiRoutes(router fiber.Router) {

	router.Get("/", renderIndexPage)
}

func renderIndexPage(c *fiber.Ctx) error {

	return c.Render("views/index", fiber.Map{}, "views/layouts/main")
}
