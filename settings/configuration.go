package settings

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Configuration struct {
	Listen     string `envconfig:"LISTEN" default:":3000"`
	Secret     string `envconfig:"SECRET" default:"yfasdhudashnjdas"`
	Index      string `envconfig:"INDEX" default:"organizations"`
	Cookie     string `envconfig:"COOKIE" default:"cookie"`
	Expiration int    `envconfig:"EXPIRATION" default:"2"`
}

func NewConfiguration() (*Configuration, error) {
	var configuration Configuration

	if err := envconfig.Process("", &configuration); err != nil {
		return nil, err
	}

	return &configuration, nil
}
