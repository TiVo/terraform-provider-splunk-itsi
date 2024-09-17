package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/thld"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

var (
	thresholdCmdDryRun                          bool
	thresholdRecommendCmdUseLatestData          bool
	thresholdRecommendCmdInsufficientDataAction string
)

var thresholdCmd = &cobra.Command{
	Use:     "threshold",
	Aliases: []string{"thld"},
	Short:   "Manage KPI thresholds",
	Long: util.Dedent(`
The "threshold" command allows you to manage KPI thresholds in Splunk ITSI.

It provides subcommands to reset thresholds or apply machine learning-assisted thresholds to your KPIs.
`),
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset thresholds for selected KPIs/services",
	Long: util.Dedent(`
	Reset the thresholding configuration for specified KPIs/services.

	  Matching KPIs will be updated to use the following thresholding configuration:
	    adaptive thresholds:     disabled
	    aggregate thresholds:    normal severity
	    entity thresholds:       normal severity
	    time variate thresholds: disabled
	    outlier detection:       disabled

	  To prevent accidental resetting of thresholds for all services, the "reset" command requires at least one service selector to be provided using the '--service' flag.`),
	Example: `
  - Reset thresholds for all KPIs in a specific service:

    itsictl threshold reset --service service1

  - Reset thresholds for all KPIs in multiple services:

    itsictl threshold reset --service service1 --service2

  - Reset thresholds for the "errors" KPI and all KPIs starting with "network" in the services that start with "sample service":

    itsictl threshold reset -s "sample service*" -k errors -k "network*"
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(services) == 0 {
			fmt.Println("No services specified. You must provide one or more service selectors using the --service argument.")
			os.Exit(1)
		}

		w := thld.NewThresholdResetWorkflow(cfg, services, kpis, thresholdCmdDryRun)
		err := w.Execute(context.Background())
		if err != nil {
			fmt.Printf("Workflow has completed with errors: %s\n", err.Error())
			os.Exit(1)
		}
	},
}

var recommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Apply ML-assisted thresholds",
	Long: util.Dedent(`
	Performs historical data analysis on matching KPIs that are configured to use ML-assisted thresholds and updates the threshold configurations based on the recommendations derived from the ML analysis.
	  * Only KPIs configured for ML-assisted thresholds are analyzed. KPIs using threshold templates or custom thresholds are skipped.
	  * By default, the analysis starts from the 'start_date' specified at the KPI level. You can override this by using the '--use-latest-data' flag to analyze the most recent data instead.
	  * If the ML analysis cannot recommend any thresholds due to insufficient historical data or constant KPI values during the analysis window, the KPI is skipped by default, retaining its current configuration.
	    You can change this behavior using the '--insufficient-data-action' flag and opt in for resetting such KPI's thresholds instead.
	`),
	Example: `
  - Analyze and update the thresholds for all KPIs configued for ML-assisted thresholds:

    itsictl threshold recommend

  - Analyze and update the thresholds for all KPIs within service1 that are configured to use ML-assisted thresholds:

    itsictl threshold recommend --service service1

  - Apply ML-assisted thresholds to relevant KPIs in multiple services:

    itsictl threshold recommend --service service1 --service service2

  - Apply ML-assisted thresholds to the "errors" KPI and all KPIs starting with "network" in services that start with "sample service":

    itsictl threshold recommend --service "sample service*" --kpi errors --kpi "network*"

  - Apply ML-assisted thresholds for relevant KPIs in service1, using the latest data for analysis, ignoring the stored starting date:

    itsictl threshold recommend --service service1 --use-latest-data

  - Apply ML-assisted thresholds and reset configurations for KPIs with insufficient data:

    itsictl threshold recommend --service service1 --insufficient-data-action reset

  - Perform a dry run to see what changes would be made without applying them:

    itsictl threshold recommend --service service1 --dry-run
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if !(thresholdRecommendCmdInsufficientDataAction == "skip" ||
			thresholdRecommendCmdInsufficientDataAction == "reset") {
			fmt.Println("Error: invalid value for '--insufficient-data-action'. Must be 'skip' or 'reset'.")
			os.Exit(1)
		}

		w := thld.NewThresholdRecommendationWorkflow(
			cfg,
			services,
			kpis,
			thresholdCmdDryRun,
			thresholdRecommendCmdUseLatestData,
			thresholdRecommendCmdInsufficientDataAction,
		)

		err := w.Execute(context.Background())
		if err != nil {
			fmt.Printf("Workflow has completed with errors: %s\n", err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(thresholdCmd)

	thresholdCommands := []*cobra.Command{resetCmd, recommendCmd}

	for _, cmd := range thresholdCommands {
		thresholdCmd.AddCommand(cmd)
		cmd.Flags().StringArrayVarP(&services, "service", "s", []string{}, "Specify Service Selector (service ID, title, or a wildcard pattern; can be used multiple times)")
		cmd.Flags().StringArrayVarP(&kpis, "kpi", "k", []string{}, "Specify KPI Selector (KPI ID, title, or a wildcard pattern; can be used multiple times)")
		cmd.Flags().BoolVar(&thresholdCmdDryRun, "dry-run", false, "Run the command without actaully changing anything")
	}

	recommendCmd.Flags().BoolVar(&thresholdRecommendCmdUseLatestData, "use-latest-data", false, "Use the latest KPI data for analysis (ignore the stored starting date)")
	recommendCmd.Flags().StringVar(&thresholdRecommendCmdInsufficientDataAction, "insufficient-data-action", "skip", "Action to take for KPIs with insufficient data: 'skip' or 'reset'")

}
