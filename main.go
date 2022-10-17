package main

import (
	"log"

	"example.com/account"
	"example.com/settings"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
)

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

	account.
		NewServer(router, storage, *configuration).
		Route(router.Group(""))

	if err = router.Listen(configuration.Listen); err != nil {
		log.Fatal(err)
	}
}
