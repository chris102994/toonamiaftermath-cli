package config

import (
	"github.com/chris102994/toonamiaftermath-cli/internal/cron"
	"github.com/chris102994/toonamiaftermath-cli/internal/logging"
	"github.com/chris102994/toonamiaftermath-cli/internal/run"
	"github.com/spf13/viper"
)

type Config struct {
	Logging *logging.Logging `mapstructure:",omitempty"`
	Cron    *cron.Cron       `mapstructure:",omitempty"`
	Run     *run.Run         `mapstructure:",omitempty"`
}

func (c *Config) LoadConfig() error {
	err := viper.Unmarshal(&c)
	if err != nil {
		return err
	}

	return nil
}
