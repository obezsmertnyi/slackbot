/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
// It serves as the foundation for the Slackbot CLI application, setting up the
// primary command structure and metadata.
var rootCmd = &cobra.Command{
	Use:   "slackbot",
	Short: "Slackbot is a bot for managing deployments and notifications in Slack",
	Long:  `A Slack bot that integrates with Kubernetes, GitHub, and Slack API to manage deployments, check status, and send notifications.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Your bot's startup logic goes here
		fmt.Println("Welcome to Slackbot! Use 'slackbot [command]' to interact with your Slack workspace.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
