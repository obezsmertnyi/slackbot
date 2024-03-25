package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/spf13/cobra"
)

// startCmd represents the start command for the Slackbot. It initializes and starts the Slackbot,
// setting up the necessary clients for Slack API and Socket Mode, as well as initializing the database,
// Kubernetes, and GitHub clients.
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the Slackbot",
	Long:  `This command initializes and starts the Slackbot.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Slackbot started")

		// Retrieve Slack API and App-Level tokens from environment variables
		token := os.Getenv("SLACK_AUTH_TOKEN")
		appToken := os.Getenv("SLACK_APP_TOKEN")

		// Initialize Slack client with debugging enabled
		client := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))
		socketClient := socketmode.New(
			client,
			socketmode.OptionDebug(true),
			socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
		)

		// Create a context to manage the lifecycle of the Socket Mode listener
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize the database, Kubernetes client, and GitHub client
		initDatabase()
		initKubernetesClient()
		initGitHubClient()
		go startMetricsServer()


		// Start a goroutine to listen for and handle incoming events from Slack
		go func(ctx context.Context, client *slack.Client, socketClient *socketmode.Client) {
			for {
				select {
				case <-ctx.Done():
					log.Println("Shutting down socketmode listener")
					return
				case event := <-socketClient.Events:
					switch event.Type {
					case socketmode.EventTypeEventsAPI:
						eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
						if !ok {
							log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
							continue
						}
						socketClient.Ack(*event.Request)
						err := handleEventMessage(eventsAPIEvent, client)
						if err != nil {
							log.Fatal(err)
						}
					case socketmode.EventTypeSlashCommand:
						command, ok := event.Data.(slack.SlashCommand)
						if !ok {
							log.Printf("Could not type cast the message to a SlashCommand: %v\n", command)
							continue
						}
						payload, err := handleSlashCommand(command, client)
						if err != nil {
							log.Fatal(err)
						}
						socketClient.Ack(*event.Request, payload)
					case socketmode.EventTypeInteractive:
						interaction, ok := event.Data.(slack.InteractionCallback)
						if !ok {
							log.Printf("Could not type cast the message to an Interaction callback: %v\n", interaction)
							continue
						}
						err := handleInteractionEvent(interaction, client)
						if err != nil {
							log.Fatal(err)
						}
						socketClient.Ack(*event.Request)
					}
				}
			}
		}(ctx, client, socketClient)

		// Start the Socket Mode client to listen for incoming events
		socketClient.Run()
	},
}

// Register the start command to the root command of the CLI application
func init() {
	rootCmd.AddCommand(startCmd)
}
