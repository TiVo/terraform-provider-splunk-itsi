package core

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/config"
)

type Workflow struct {
	Cfg config.Config
	Log *log.Logger
}

func MakeWorkflow(config config.Config) Workflow {
	return Workflow{
		config,
		newLogger(config),
	}
}

func newLogger(config config.Config) (l *log.Logger) {
	logLevel := log.InfoLevel
	if config.Verbose {
		logLevel = log.DebugLevel
	}

	l = log.NewWithOptions(os.Stdout, log.Options{
		Level:           logLevel,
		ReportTimestamp: true,
	})

	styles := log.DefaultStyles()
	for lvl := range styles.Levels {
		styles.Levels[lvl] = styles.Levels[lvl].MaxWidth(5)
	}
	l.SetStyles(styles)

	return
}
