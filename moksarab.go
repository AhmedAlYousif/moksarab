package main

import (
	"moksarab/config"
	"moksarab/database"
	"moksarab/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

var (
	Version = "dev"
)

func InitilizeMocSarabServer() *fiber.App {
	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		ServerHeader:  "MokSarab v" + Version,
		AppName:       "MokSarab Mock Server v" + Version,
	})

	var sarab fiber.Router
	if config.WorkspaceEnabled {
		sarab = app.Use("/sarab/:workspaceId/*", routes.HandleSarabRequests)
	} else {
		sarab = app.Use("/sarab/*", routes.HandleSarabRequests)
	}

	api := app.Group("/api")
	// ui := app.Group("/")

	routes.RegisterAPIRoutes(api)
	sarab.Use(routes.HandleSarabRequests)
	return app
}

func main() {
	database.InitilizeDatabase()
	defer database.Db.Close()

	app := InitilizeMocSarabServer()

	log.Fatal(app.Listen(":" + config.Port))
}
