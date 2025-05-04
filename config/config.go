package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	Port        string `mapstructure:"PORT"`
	GinMode     string `mapstructure:"GIN_MODE"`
	DBSSLMode   string `mapstructure:"DB_SSLMODE"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
}

func LoadConfig() (config Config, err error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: Error reading config file: %v. Will try to use environment variables instead.", err)
		// Continue execution as viper will still check environment variables
	}

	err = viper.Unmarshal(&config)
	return
}
