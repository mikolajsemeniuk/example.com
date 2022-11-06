package management

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	storage       *elasticsearch.Client
	configuration Config
}

func NewServer(s *elasticsearch.Client, c Config) *server {
	return &server{
		storage:       s,
		configuration: c,
	}
}

func (s *server) Chain(r fiber.Router) {
	// authentication
	r.Post("/account/register", s.Register)
	r.Post("/account/login", s.Login)
	r.Get("/account/logout", s.Logout)
	r.Get("/account/authorize", s.Authorize)

	// campaigns
	r.Get("/campaigns", s.ListCampaigns)
	r.Post("/campaign/create", s.CreateCampaign)
}

// @Summary Register
// @Schemes
// @Description Register new user
// @Tags account
// @Accept application/json
// @Param payload body RegisterRequest true "body"
// @Success 200 {object} string
// @Success 400
// @Failure 500
// @Failure 503
// @Router /account/register [post]
func (s *server) Register(c *fiber.Ctx) error {
	var request RegisterRequest
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	query := strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "accounts.email": "%s" } } }`, request.Email))
	response, err := s.storage.Search(s.storage.Search.WithIndex(s.configuration.Index), s.storage.Search.WithBody(query))
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Hits []struct {
				Source Organization `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	if len(payload.Hits.Hits) > 0 {
		return fiber.NewError(http.StatusBadRequest, "user with this email already exists")
	}

	query = strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "name": "%s" } } }`, request.Company))
	response, err = s.storage.Search(s.storage.Search.WithIndex(s.configuration.Index), s.storage.Search.WithBody(query))
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
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

	organization := Organization{
		Key:       uuid.New(),
		Name:      string(request.Company),
		Accounts:  []Account{account},
		Campaigns: []Campaign{},
		Created:   time.Now(),
	}

	if len(payload.Hits.Hits) > 0 {
		organization = payload.Hits.Hits[0].Source
		organization.Accounts = append(organization.Accounts, account)
	}

	data, err := json.Marshal(organization)
	if err != nil {
		return err
	}

	create := esapi.IndexRequest{Index: s.configuration.Index, DocumentID: organization.Key.String(), Body: bytes.NewReader(data)}
	response, err = create.Do(context.Background(), s.storage)
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	claims := jwt.MapClaims{
		"Issuer":    organization.Key,
		"Company":   organization.Name,
		"ExpiresAt": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(s.configuration.Secret))
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name: s.configuration.Cookie, Value: value,
		Secure:   true,
		HTTPOnly: false,
		SameSite: "none",
		Expires:  time.Now().Add(time.Duration(s.configuration.Expiration) * time.Hour),
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

	query := strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "accounts.email": "%s" } } }`, request.Email))
	response, err := s.storage.Search(s.storage.Search.WithIndex(s.configuration.Index), s.storage.Search.WithBody(query))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Hits []struct {
				Source Organization `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	if len(payload.Hits.Hits) == 0 {
		return fiber.NewError(http.StatusBadRequest, "no user with this email address")
	}

	if err := bcrypt.CompareHashAndPassword(payload.Hits.Hits[0].Source.Accounts[0].Password, []byte(request.Password)); err != nil {
		return fiber.NewError(http.StatusBadRequest, "incorrect password")
	}

	claims := jwt.MapClaims{
		"Issuer":    payload.Hits.Hits[0].Source.Key,
		"Company":   payload.Hits.Hits[0].Source.Name,
		"ExpiresAt": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(s.configuration.Secret))
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name: s.configuration.Cookie, Value: value,
		Secure:   true,
		HTTPOnly: false,
		SameSite: "none",
		Expires:  time.Now().Add(time.Duration(s.configuration.Expiration) * time.Hour),
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
		Name:     s.configuration.Cookie,
		Value:    "",
		Secure:   true,
		HTTPOnly: false,
		SameSite: "none",
		Expires:  time.Now().Add(-time.Second),
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
	parser := func(token *jwt.Token) (interface{}, error) { return []byte(s.configuration.Secret), nil }

	token, err := jwt.Parse(cookie, parser)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, "unauthorized")
	}

	payload := token.Claims.(jwt.MapClaims)

	id := payload["Issuer"].(string)

	return c.SendString("user with id: " + id)
}

// Campaigns
// TODO: move it later

// @Summary ListCampaigns
// @Schemes
// @Description ListCampaigns
// @Tags campaigns
// @Accept application/json
// @Success 200 {object} string
// @Router /campaigns [get]
func (s *server) ListCampaigns(c *fiber.Ctx) error {
	// TODO: refactor later
	cookie := c.Cookies(s.configuration.Cookie)
	parser := func(token *jwt.Token) (interface{}, error) { return []byte(s.configuration.Secret), nil }

	token, err := jwt.Parse(cookie, parser)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, "unauthorized")
	}

	claims := token.Claims.(jwt.MapClaims)
	company := claims["Company"].(string)

	query := strings.NewReader(fmt.Sprintf(`{ "_source": { "include": [ "campaigns" ] }, "query": { "match_phrase": { "name": "%s" } } }`, company))
	response, err := s.storage.Search(s.storage.Search.WithIndex(s.configuration.Index), s.storage.Search.WithBody(query))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Campaigns []Campaign `json:"campaigns"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	output, err := json.Marshal(payload.Hits.Hits[0].Source.Campaigns)
	if err != nil {
		return err
	}

	return c.Send(output)
}

// @Summary CreateCampaign
// @Schemes
// @Description CreateCampaign
// @Tags campaigns
// @Accept application/json
// @Param payload body CreateCampaignRequest true "body"
// @Success 200 {object} string
// @Router /campaign/create [post]
func (s *server) CreateCampaign(c *fiber.Ctx) error {
	var request CreateCampaignRequest
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return err
	}

	// TODO: refactor later
	cookie := c.Cookies(s.configuration.Cookie)
	parser := func(token *jwt.Token) (interface{}, error) { return []byte(s.configuration.Secret), nil }

	token, err := jwt.Parse(cookie, parser)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, "unauthorized")
	}

	claims := token.Claims.(jwt.MapClaims)
	company := claims["Company"].(string)

	query := strings.NewReader(fmt.Sprintf(`{ "query": { "match_phrase": { "name": "%s" } } }`, company))
	response, err := s.storage.Search(s.storage.Search.WithIndex(s.configuration.Index), s.storage.Search.WithBody(query))
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	var payload struct {
		Hits struct {
			Hits []struct {
				Source Organization `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}

	organization := payload.Hits.Hits[0].Source
	organization.Campaigns = append(organization.Campaigns, Campaign{
		Key:     uuid.New(),
		Name:    request.Name,
		Created: time.Now(),
	})

	data, err := json.Marshal(organization)
	if err != nil {
		return err
	}

	create := esapi.IndexRequest{
		Index:      s.configuration.Index,
		DocumentID: payload.Hits.Hits[0].Source.Key.String(),
		Body:       bytes.NewReader(data),
	}

	response, err = create.Do(context.Background(), s.storage)
	if err != nil {
		return fiber.NewError(http.StatusServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	return c.SendString("campaign created")
}
