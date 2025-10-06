package configs

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	Redis struct {
		Host    string        `mapstructure:"host"`
		Port    int           `mapstructure:"port"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"redis"`
}

var AppConfig *Config

func LoadConfig() error {
	viper.AddConfigPath("configs/")
	viper.SetConfigName("application")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// Load base config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Fatal Error: Base config file (application.yaml) not found.")
		} else {
			log.Fatalf("Fatal error reading base config file: %s", err)
		}
	}

	// Profile override
	profile := viper.GetString("PROFILE")
	if profile != "" {
		viper.SetConfigName("application-" + profile)
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Fatalf("Fatal Error: Profile config file (application-%s.yaml) not found.", profile)
			} else {
				log.Fatalf("Fatal error merging profile config file: %s", err)
			}
		} else {
			log.Printf("Profile config file (application-%s.yaml) loaded.", profile)
		}
	} else {
		log.Printf("No profile config file (application-%s.yaml) loaded.", profile)
	}

	// Unmarshal
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to unmarshal final config into struct: %s", err)
		return err
	}
	return nil
}
