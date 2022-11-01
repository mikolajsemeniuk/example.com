package main

import (
	"log"

	"example.com/account"
	_ "example.com/docs"
	"example.com/settings"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// @title           API
// @version         1.0
// @description     some API
// @BasePath /
// @schemes http https
func main() {
	configuration, err := settings.NewConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	router := fiber.New()
	router.Get("/swagger/*", swagger.HandlerDefault)
	router.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).Redirect("/swagger/index.html")
	})

	account.NewServer(storage, *configuration).Chain(router.Group(""))

	if err = router.Listen(configuration.Listen); err != nil {
		log.Fatal(err)
	}
}
