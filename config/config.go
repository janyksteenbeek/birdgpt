package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Config struct {
	Moneybird struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
		RedirectURI  string `mapstructure:"redirect_uri,omitempty"`
		Token        string `mapstructure:"token"`
		AdminID      string `mapstructure:"admin_id"`
		Country      string `mapstructure:"country"`
	} `mapstructure:"moneybird"`

	Gmail struct {
		CredentialsFile string `mapstructure:"credentials_file"`
		Token           string `mapstructure:"token"`
		 SearchLabel     string `mapstructure:"search_label"`
	} `mapstructure:"gmail"`

	OpenAI struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"openai"`

	App struct {
		LastUpdate  string        `mapstructure:"last_update"`
		SleepTime   time.Duration `mapstructure:"sleep_time"`
		TriggerWord string        `mapstructure:"trigger_word"`
	} `mapstructure:"app"`
}

func (c *Config) Validate() error {
	var checks = []struct {
		condition bool
		message   string
	}{
		{c.Moneybird.ClientID == "", "moneybird client_id is required"},
		{c.Moneybird.ClientSecret == "", "moneybird client_secret is required"},
		{c.Moneybird.AdminID == "", "moneybird admin_id is required"},
		{c.Gmail.CredentialsFile == "", "gmail credentials_file is required"},
		{c.OpenAI.APIKey == "", "openai api_key is required"},
		{c.App.SleepTime < time.Second, "app sleep_time must be at least 1 second"},
		{c.Gmail.SearchLabel == "", "gmail search_label is required"},
	}

	for _, check := range checks {
		if check.condition {
			return fmt.Errorf(check.message)
		}
	}

	if _, err := os.Stat(c.Gmail.CredentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("gmail credentials file does not exist: %s", c.Gmail.CredentialsFile)
	}

	return nil
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func SaveConfig(config *Config) error {
	viper.Set("moneybird.token", config.Moneybird.Token)
	viper.Set("gmail.token", config.Gmail.Token)
	viper.Set("app.last_update", config.App.LastUpdate)
	return viper.WriteConfig()
}
