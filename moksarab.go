package main

import (
	"embed"
	"moksarab/config"
	"moksarab/database"
	"moksarab/routes"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/template/html/v2"
)

var (
	Version = "dev"

	//go:embed views/*.html
	//go:embed views/**/*.html
	viewsFS embed.FS

	///go:embed public/*
	//go:embed public/**/*
	publicFS embed.FS
)

func InitilizeMocSarabServer() *fiber.App {

	templateEngine := html.NewFileSystem(http.FS(viewsFS), ".html")

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		ServerHeader:  "MokSarab v" + Version,
		AppName:       "MokSarab Mock Server v" + Version,
		Views:         templateEngine,
	})

	routes.RegesiterUiRoutes(app)
	app.Use("/public", adaptor.HTTPHandler(http.FileServer(http.FS(publicFS))))

	var sarab fiber.Router
	if config.WorkspaceEnabled {
		sarab = app.Use("/sarab/:workspaceId/*", routes.HandleSarabRequests)
	} else {
		sarab = app.Use("/sarab/*", routes.HandleSarabRequests)
	}

	api := app.Group("/api")

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
