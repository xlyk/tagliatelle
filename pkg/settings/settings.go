package settings

import (
	"github.com/compose-spec/godotenv"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	GitUser  string
	GitToken string
)

func Load() error {
	// load .env file
	if err := godotenv.Load(); err != nil {
		log.Warn("did not find a .env file, defaulting to system environment")
	}

	m := map[string]*string{
		"GIT_USER":  &GitUser,
		"GIT_TOKEN": &GitToken,
	}

	// load strings from environment variables
	for k, v := range m {
		if err := loadString(k, v); err != nil {
			return err
		}
	}

	return nil
}

func loadString(key string, ptr *string) error {
	str, ok := os.LookupEnv(key)
	if !ok {
		return eris.Errorf("failed to load string %s from environment variable", key)
	}
	*ptr = str
	return nil
}
