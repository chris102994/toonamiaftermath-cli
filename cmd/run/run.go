package run

import (
	"encoding/xml"
	m3u "github.com/chris102994/go-m3u/pkg/m3u/models"
	"github.com/chris102994/toonamiaftermath-cli/internal/config"
	"github.com/chris102994/toonamiaftermath-cli/pkg/toonamiaftermath"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var inputConfig *config.Config

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the toonamiaftermath-cli",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.WithFields(log.Fields{
			"runConfig": inputConfig.Run,
		}).Trace("Running toonamiaftermath-cli run")

		if inputConfig.Cron.Expression != "" {
			log.WithFields(log.Fields{
				"runConfig": inputConfig.Run,
			}).Info("Running in cron mode")

			logger := log.StandardLogger()

			c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(logger)))
			_, err := c.AddFunc(inputConfig.Cron.Expression, runToonamiAftermathScraping)
			cobra.CheckErr(err)

			go runToonamiAftermathScraping()

			c.Run()

		} else {
			log.WithFields(log.Fields{}).Trace("No Cron Schedule detected")
			err := handleToonamiAftermathScraping()
			cobra.CheckErr(err)
		}

		return nil
	},
}

func init() {
	viper.SetDefault("Run.xmltv_output", "index.xml")
	viper.SetDefault("Run.m3u_output", "index.m3u")

	runCmd.Flags().StringP("cron-expression", "c", "", "The cron schedule to run the command")
	runCmd.Flags().StringP("xmltv-output", "x", "index.xml", "Path to the XMLTV output file")
	runCmd.Flags().StringP("m3u-output", "m", "index.m3u", "Path to the M3U output file")

	viper.BindPFlag("Cron.Expression", runCmd.Flag("cron-expression"))
	viper.BindPFlag("Run.xmltv_output", runCmd.Flag("xmltv-output"))
	viper.BindPFlag("Run.m3u_output", runCmd.Flag("m3u-output"))
}

func NewRunCmd(c *config.Config) *cobra.Command {
	inputConfig = c
	return runCmd
}

func runToonamiAftermathScraping() {
	log.WithFields(log.Fields{}).Info("Running Cron Schedule")
	err := handleToonamiAftermathScraping()
	cobra.CheckErr(err)
	log.WithFields(log.Fields{}).Info("Cron Schedule ran")
}

func handleToonamiAftermathScraping() error {
	toonamiaftermathConfig := toonamiaftermath.New()
	err := toonamiaftermathConfig.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to run ToonamiAftermath scraping")
		return err
	}

	// Write the XMLTV output to a file
	xmlTvOutput, err := xml.MarshalIndent(toonamiaftermathConfig.XMLTVOutput, "", "  ")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to marshal XMLTV output")
		return err
	}

	err = os.WriteFile(inputConfig.Run.XMLTVOutput, xmlTvOutput, 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to write XMLTV output to file")
		return err
	}

	// Write the M3U output to a file
	m3uOutput, err := m3u.Marshal(&toonamiaftermathConfig.M3UOutput)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to marshal M3U output")
		return err
	}

	err = os.WriteFile(inputConfig.Run.M3UOutput, m3uOutput, 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to write M3U output to file")
		return err
	}

	return nil
}
