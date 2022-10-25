package authentication

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

func (s *server) Chain(r fiber.Router) {
	r.Post("/account/register", s.Register)
	r.Post("/account/login", s.Login)
	r.Get("/account/logout", s.Logout)
	r.Get("/account/authorize", s.Authorize)
}

// @Summary Register
// @Schemes
// @Description Register new user
// @Tags account
// @Accept application/json
// @Param payload body Request true "body"
// @Success 200 {object} string
// @Success 400
// @Failure 500
// @Failure 503
// @Router /account/register [post]
func (s *server) Register(c *fiber.Ctx) error {
	var request Request
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	query := strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "email": "%s" } } }`, request.Email))
	response, err := s.storage.Search(s.storage.Search.WithIndex(index), s.storage.Search.WithBody(query))
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	if payload.Hits.Total.Value > 0 {
		return fiber.NewError(http.StatusBadRequest, "user with this email already exists")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(request.Password), 14)
	if err != nil {
		return err
	}

	account := Account{
		Key:      uuid.New(),
		Email:    request.Email.String(),
		Password: password,
		Created:  time.Now(),
	}

	data, err := json.Marshal(account)
	if err != nil {
		return err
	}

	create := esapi.IndexRequest{Index: index, DocumentID: account.Key.String(), Body: bytes.NewReader(data)}
	response, err = create.Do(context.Background(), s.storage)
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	claims := jwt.MapClaims{"Issuer": account.Key.String(), "ExpiresAt": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(s.configuration.Secret))
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name: s.configuration.Cookie, Value: value, HTTPOnly: true, SameSite: "strict",
		Expires: time.Now().Add(time.Duration(s.configuration.Expiration) * time.Hour),
	})

	return c.SendString("user registered")
}

// @Summary Login
// @Schemes
// @Description Login existing user
// @Tags account
// @Accept application/json
// @Param payload body Request true "body"
// @Success 200 {object} string
// @Success 400
// @Failure 500
// @Failure 503
// @Router /account/login [post]
func (s *server) Login(c *fiber.Ctx) error {
	var request Request
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	query := strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "email": "%s" } } }`, request.Email))
	response, err := s.storage.Search(s.storage.Search.WithIndex(index), s.storage.Search.WithBody(query))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Hits []struct {
				Source Account `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	if len(payload.Hits.Hits) == 0 {
		return fiber.NewError(http.StatusBadRequest, "no user with this email address")
	}

	if err := bcrypt.CompareHashAndPassword(payload.Hits.Hits[0].Source.Password, []byte(request.Password)); err != nil {
		return fiber.NewError(http.StatusBadRequest, "incorrect password")
	}

	claims := jwt.MapClaims{"Issuer": payload.Hits.Hits[0].Source.Key, "ExpiresAt": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(s.configuration.Secret))
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name: s.configuration.Cookie, Value: value, HTTPOnly: true, SameSite: "strict",
		Expires: time.Now().Add(time.Duration(s.configuration.Expiration) * time.Hour),
	})

	return c.SendString("user logged")
}

// @Summary Logout
// @Schemes
// @Description Logout existing user
// @Tags account
// @Accept application/json
// @Success 200 {object} string
// @Router /account/logout [get]
func (s *server) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name: s.configuration.Cookie, Value: "", HTTPOnly: true, SameSite: "strict",
		Expires: time.Now().Add(-time.Second),
	})
	return c.SendString("user logout")
}

// @Summary Authorize
// @Schemes
// @Description Authorize existing user
// @Tags account
// @Accept application/json
// @Success 200 {object} string
// @Success 401 {object} string
// @Router /account/authorize [get]
func (s *server) Authorize(c *fiber.Ctx) error {
	cookie := c.Cookies(s.configuration.Cookie)

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) { return []byte(s.configuration.Secret), nil })
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, "unauthorized")
	}

	payload := token.Claims.(jwt.MapClaims)

	id := payload["Issuer"].(string)

	return c.SendString("user with id: " + id)
}
