package management

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Listen     string `envconfig:"LISTEN" default:":5000"`
	Secret     string `envconfig:"SECRET" default:"yfasdhudashnjdas"`
	Index      string `envconfig:"INDEX" default:"organizations"`
	Cookie     string `envconfig:"COOKIE" default:"cookie"`
	Expiration int    `envconfig:"EXPIRATION" default:"2"`
}

func MakeConfiguration() (Config, error) {
	var configuration Config

	if err := envconfig.Process("", &configuration); err != nil {
		return configuration, err
	}

	return configuration, nil
}
