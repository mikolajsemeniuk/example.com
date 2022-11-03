package management

import (
	"encoding/json"
	"errors"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    Email    `json:"email" example:"mike@mock.com"`
	Password Password `json:"password" example:"P@ssw0rd"`
	Company  Company  `json:"company" example:"ey"`
}

type Request struct {
	Email    Email    `json:"email" example:"mike@mock.com"`
	Password Password `json:"password" example:"P@ssw0rd"`
}

type Organization struct {
	Key      uuid.UUID `json:"key"`
	Name     string    `json:"name"`
	Accounts []Account `json:"accounts"`
	Created  time.Time `json:"created"`
}

type Account struct {
	Key      uuid.UUID `json:"key"`
	Email    string    `json:"email"`
	Password []byte    `json:"password"`
	Created  time.Time `json:"created"`
}

type Email string

func (e *Email) UnmarshalJSON(data []byte) error {
	var email string
	if err := json.Unmarshal(data, &email); err != nil {
		return err
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return err
	}

	*e = Email(email)
	return nil
}

func (e *Email) String() string {
	return string(*e)
}

type Password string

func (p *Password) UnmarshalJSON(data []byte) error {
	var password string
	if err := json.Unmarshal(data, &password); err != nil {
		return err
	}

	if len(password) < 5 || len(password) > 32 {
		return errors.New("password should be between 5 and 32 characters")
	}

	*p = Password(password)
	return nil
}

type Company string

func (c *Company) UnmarshalJSON(data []byte) error {
	var company string
	if err := json.Unmarshal(data, &company); err != nil {
		return err
	}

	if len(company) < 2 || len(company) > 32 {
		return errors.New("company should be between 2 and 32 characters")
	}

	*c = Company(company)
	return nil
}
