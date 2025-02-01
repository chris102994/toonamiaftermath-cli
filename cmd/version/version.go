package version

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	branch         string
	buildTimestamp string
	commitHash     string
	Version        string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print detailed version information for toonamiaftermath-cli",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.WithFields(log.Fields{
			"Version":        Version,
			"Branch":         branch,
			"CommitHash":     commitHash,
			"BuildTimestamp": buildTimestamp,
		}).Println("Version")
		return nil
	},
}

func NewVersionCmd(_branch string, _buildTimeStamp string, _commitHash string, _version string) *cobra.Command {
	branch = _branch
	buildTimestamp = _buildTimeStamp
	commitHash = _commitHash
	Version = _version

	return versionCmd
}
