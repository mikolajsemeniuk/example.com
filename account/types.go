package account

import (
	"encoding/json"
	"errors"
	"net/mail"
)

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

type Password []byte

func (e *Password) UnmarshalJSON(data []byte) error {
	password := string(data)
	if len(password) < 5 || len(password) > 32 {
		return errors.New("password should be between 5 and 32 characters")
	}
	return nil
}
