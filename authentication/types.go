package authentication

import (
	"encoding/json"
	"errors"
	"net/mail"
	"time"
)

type Request struct {
	Email    Email    `json:"email" example:"mike@mock.com"`
	Password Password `json:"password" example:"P@ssw0rd"`
}

type Account struct {
	Key      string    `json:"key"`
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
	password := string(data)
	if len(password) < 5 || len(password) > 32 {
		return errors.New("password should be between 5 and 32 characters")
	}
	*p = Password(password)
	return nil
}
