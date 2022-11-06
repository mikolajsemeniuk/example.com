package management

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	Key       uuid.UUID  `json:"key"`
	Name      string     `json:"name"`
	Accounts  []Account  `json:"accounts"`
	Campaigns []Campaign `json:"campaigns"`
	Created   time.Time  `json:"created"`
}

type Account struct {
	Key      uuid.UUID `json:"key"`
	Email    string    `json:"email"`
	Password []byte    `json:"password"`
	Created  time.Time `json:"created"`
}

// TODO: move it later
type Campaign struct {
	Key          uuid.UUID `json:"key"`
	Name         string    `json:"name"`
	Start        time.Time `json:"start"`
	Finish       time.Time `json:"finish"`
	Active       bool      `json:"active"`
	Wanted       int       `json:"wanted"`
	Accept       float32   `json:"accept"`
	Reject       float32   `json:"reject"`
	Education    []string  `json:"education"`
	Experience   []string  `json:"experience"`
	Certificates []string  `json:"certificates"`
	Courses      []string  `json:"courses"`
	Skills       []string  `json:"skills"`
	Languages    []string  `json:"languages"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
}
