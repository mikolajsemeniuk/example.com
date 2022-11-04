package main

import (
	"log"

	_ "example.com/docs"
	"example.com/management"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

// @title           API
// @version         1.0
// @description     some API
// @BasePath /
// @schemes http https
func main() {
	configuration, err := management.MakeConfig()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	router := fiber.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Content-Length, Accept-Language, Accept-Encoding, Connection, Access-Control-Allow-Origin",
		AllowCredentials: true,
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
	}))
	router.Get("/swagger/*", swagger.HandlerDefault)
	management.NewServer(storage, configuration).Chain(router.Group(""))
	router.Use(func(c *fiber.Ctx) error { return c.Status(fiber.StatusNotFound).Redirect("/swagger/index.html") })

	if err = router.Listen(configuration.Listen); err != nil {
		log.Fatal(err)
	}
}
