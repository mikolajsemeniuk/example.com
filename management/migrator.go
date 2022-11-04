package management

import (
	"github.com/elastic/go-elasticsearch/v8"
)

func Migrate(s *elasticsearch.Client, c Config) error {
	response, err := s.Indices.Exists([]string{c.Index})
	if err != nil {
		return err
	}

	if response.StatusCode == 200 {
		return nil
	}

	if _, err = s.Indices.Create(c.Index); err != nil {
		return err
	}

	return nil
}
