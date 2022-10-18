package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"example.com/settings"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const index = "accounts"

type server struct {
	router        *fiber.App
	storage       *elasticsearch.Client
	configuration settings.Configuration
}

func NewServer(r *fiber.App, s *elasticsearch.Client, c settings.Configuration) *server {
	return &server{
		router:        r,
		storage:       s,
		configuration: c,
	}
}

func (s *server) Route(r fiber.Router) {
	r.Post("/account/register", s.Register)
	r.Post("/account/login", func(c *fiber.Ctx) error { return nil })
	r.Post("/account/logout", func(c *fiber.Ctx) error { return nil })
}

func (s *server) Register(c *fiber.Ctx) error {
	var input struct {
		Key      uuid.UUID `json:"key"`
		Email    Email     `json:"email"`
		Password Password  `json:"password"`
		Created  time.Time `json:"created"`
		Updated  time.Time `json:"updated"`
	}

	if err := json.Unmarshal(c.Body(), &input); err != nil {
		return err
	}

	query := strings.NewReader(fmt.Sprintf(`{
    	"query": {
        	"match_phrase": {
				"email": "%s"
			}
    	}
	}`, input.Email))

	response, err := s.storage.Search(s.storage.Search.WithIndex(index), s.storage.Search.WithBody(query))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var body struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Key      uuid.UUID `json:"key"`
					Email    Email     `json:"email"`
					Password []byte    `json:"password"`
					Created  time.Time `json:"created"`
					Updated  time.Time `json:"updated"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		return err
	}

	if len(body.Hits.Hits) > 0 {
		return fiber.NewError(http.StatusBadRequest, "user with this email already exists")
	}

	password, err := bcrypt.GenerateFromPassword(input.Password, 14)
	if err != nil {
		return err
	}

	input.Key = uuid.New()
	input.Created = time.Now()
	input.Password = password

	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	create := esapi.IndexRequest{Index: index, DocumentID: input.Key.String(), Body: bytes.NewReader(data)}
	response, err = create.Do(context.Background(), s.storage)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	claims := jwt.MapClaims{"Issuer": input.Key.String(), "ExpiresAt": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(s.configuration.Secret))
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    value,
		Expires:  time.Now().Add(time.Duration(s.configuration.Expiration) * time.Hour),
		HTTPOnly: true,
		SameSite: "strict",
	})

	return c.SendString("user created")
}

func (s *server) LoginUser(c *fiber.Ctx) error {
	return c.SendString("login user")
}

func (s *server) LogoutUser(c *fiber.Ctx) error {
	return c.SendString("logout user")
}
