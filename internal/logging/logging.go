package logging

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type Logging struct {
	Level  string `mapstructure:",omitempty"`
	Format string `mapstructure:",omitempty"`
}

func init() {
	viper.SetDefault("Logging.Level", "info")
	viper.SetDefault("Logging.Format", "text")
}

func (l *Logging) SetupLogging() error {
	log.SetOutput(os.Stdout)

	if l.Level == "" {
		l.Level = "info"
	}

	level, err := log.ParseLevel(l.Level)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	var logFormatter log.Formatter

	switch strings.ToLower(l.Format) {
	case "json":
		logFormatter = &log.JSONFormatter{PrettyPrint: true}
	default:
		logFormatter = &log.TextFormatter{FullTimestamp: true}
	}
	log.SetFormatter(logFormatter)

	return nil
}
