package cmd

import (
	"github.com/chris102994/toonamiaftermath-cli/cmd/run"
	"github.com/chris102994/toonamiaftermath-cli/cmd/version"
	"github.com/chris102994/toonamiaftermath-cli/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var (
	configFile  string
	InputConfig config.Config
)

var RootCmd = &cobra.Command{
	Use:     "toonamiaftermath",
	Short:   "toonamiaftermath-cli helps automate ToonamiAftermath XMLTV and M3U file generation.",
	Long:    `toonamiaftermath-cli is a tool that scrapes and Generates ToonamiAftermath XMLTV and M3U files.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := InputConfig.LoadConfig()
		if err != nil {
			return err
		}

		if err := InputConfig.Logging.SetupLogging(); err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"config":      configFile,
			"InputConfig": InputConfig,
		}).Trace("Configuration loaded")

		return nil
	},
}

func init() {
	cobra.OnInitialize(InitConfig)

	// Add Sub Commands Here
	RootCmd.AddCommand(run.NewRunCmd(&InputConfig))

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to the configuration file")
	RootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
	RootCmd.PersistentFlags().StringP("log-format", "f", "text", "Log format (text, json)")

	viper.BindPFlag("Logging.Level", RootCmd.Flag("log-level"))
	viper.BindPFlag("Logging.Format", RootCmd.Flag("log-format"))

}

func InitConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if configFile != "" {
		viper.SetConfigFile(configFile)
		err := viper.ReadInConfig()
		cobra.CheckErr(err)
	}
	viper.AutomaticEnv()
}

func NewRootCmd(_branch string, _buildTimeStamp string, _commitHash string, _version string) *cobra.Command {
	RootCmd.AddCommand(version.NewVersionCmd(_branch, _buildTimeStamp, _commitHash, _version))

	return RootCmd
}
