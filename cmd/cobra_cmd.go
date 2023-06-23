package cmd

import (
	"github.com/spf13/cobra"
	"github.com/startswithzed/web-ruckus/core"
)

var config core.Config
var rootCmd *cobra.Command

func createRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "webruckus --req-spec /path/to/spec.json --duration 60 --num-clients 10",
		Long: "Load test your web server.",
		Run: func(cmd *cobra.Command, args []string) {
			runner := core.NewRunner(config)
			runner.LoadTest()
		},
	}

	cmd.Flags().StringVarP(&config.ReqSpecPath, "req-spec", "r", "", "Path to the request specification json file")
	cmd.Flags().IntVarP(&config.Duration, "duration", "d", 1, "Duration of the test in minutes")
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