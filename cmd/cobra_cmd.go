package cmd

import (
	"github.com/spf13/cobra"
	"github.com/startswithzed/blitz/core"
	"github.com/startswithzed/blitz/tui"
	"log"
	"time"
)

var config core.Config
var rootCmd *cobra.Command

func createRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "webruckus --req-spec /path/to/spec.json --duration 60 --num-clients 10",
		Long: "Load test your web server.",
		Run: func(cmd *cobra.Command, args []string) {
			ticker := time.NewTicker(time.Second)

			runner := core.NewRunner(config, ticker)
			runner.LoadTest()

			dc := tui.DashboardConfig{
				Duration:    config.Duration,
				Ticker:      ticker,
				Cancel:      runner.Cancel,
				ReqPS:       runner.ReqPS,
				ResPS:       runner.ResPS,
				ResTimes:    runner.ResTimesOut,
				ResStats:    runner.ResStats,
				ErrorStream: runner.ErrOut,
				ErrorCount:  runner.ErrCountChan,
			}

			dashboard := tui.NewDashboard(dc)
			dashboard.DrawDashboard()
			defer close(dashboard.RefreshReqChan)

			log.Println("Shutting down load test...")

			<-runner.Done // wait for the done channel to close before exiting the program
		},
	}

	cmd.Flags().StringVarP(&config.ReqSpecPath, "req-spec", "r", "", "Path to the request specification json file")
	cmd.Flags().DurationVarP(&config.Duration, "duration", "d", time.Minute, "Duration of the test in minutes")
	cmd.Flags().IntVarP(&config.NumClients, "num-clients", "c", 1, "Number of concurrent clients sending requests to the server")
	cmd.Flags().StringVarP(&config.MetricsEndpoint, "metrics-endpoint", "m", "", "Server metrics endpoint (optional)")

	cmd.MarkFlagRequired("req-spec")

	return cmd
}

func GetRootCmd() *cobra.Command {
	if rootCmd == nil {
		rootCmd = createRootCmd()
	}
	return rootCmd
}
