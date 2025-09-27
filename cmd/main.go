package main

import (
	"fmt"
	"os"

	"github.com/Revolyssup/arp/pkg/arp"
	"github.com/spf13/cobra"
)

var configFile string

func main() {
	cmd := newARPCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newARPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arp",
		Short: "ARP - Another Reverse Proxy",
		Long: `ARP is a dynamic reverse proxy with service discovery 
and plugin support for advanced routing capabilities.`,
		RunE: runARP,
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "./static.yaml", "Path to configuration file")

	// Set default from environment variable
	if envConfig := os.Getenv("ARP_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	return cmd
}

func runARP(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	app, err := arp.NewARP(configFile)
	if err != nil {
		return fmt.Errorf("failed to initialize ARP: %w", err)
	}

	return app.Run(ctx)
}
