package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// allowedNamespaces defines the namespaces that are allowed for certain operations.
var (
	allowedNamespaces = map[string]bool{"qa": true, "stage": true, "prod": true}
)

// handleSlashCommand processes slash commands input by users in Slack.
func handleSlashCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	switch command.Command {
	case "/hello":
		return nil, handleHelloCommand(command, client)
	case "/help":
		return handleHelpCommand(command, client)
	case "/list":
		return handleListPods(command, client)
	case "/diff":
		return handleDiffCommand(command, client)
	case "/promote":
		return handlePromoteCommand(command, client)
	case "/rollback":
		return handleRollbackCommand(command, client)
	default:
		message := fmt.Sprintf("Current Date and Time: %s\nUnknown command: %s. Please use a supported command.", time.Now().Format("2006-01-02 15:04:05"), command.Command)
		client.PostMessage(command.ChannelID, slack.MsgOptionText(message, false))
		return nil, nil
	}
}

// handleHelloCommand handles the "/hello" slash command.
func handleHelloCommand(command slack.SlashCommand, client *slack.Client) error {
	attachment := slack.Attachment{}
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Date",
			Value: time.Now().String(),
		}, {
			Title: "Initializer",
			Value: command.UserName,
		},
	}

	attachment.Text = fmt.Sprintf("Hello %s", command.Text)
	attachment.Color = "#4af030"

	_, _, err := client.PostMessage(command.ChannelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

// handleHelpCommand provides users with information about available commands.
func handleHelpCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	commands := []string{
		"/hello - Greet the bot",
		"/help - Get this help message",
		"/list <namespace> - List Kubernetes pods",
		"/diff <label> - Show differences in deployments",
		"/promote <namespace> <label> - Promote a deployment to the next environment",
		"/rollback <namespace> <label> - Rollback a deployment to the previous version",
	}

	message := fmt.Sprintf("Here are the commands you can use:\n```\n%s\n```", strings.Join(commands, "\n"))

	attachment := slack.Attachment{
		Text:  message,
		Color: "#4af030",
	}

	_, _, err := client.PostMessage(command.ChannelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return nil, fmt.Errorf("failed to post message: %w", err)
	}

	return nil, nil
}

// handleListPods lists Kubernetes pods in a specified namespace.
func handleListPods(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	// Increment total requests metric
	totalRequests.WithLabelValues("/list").Inc()

	// Define allowed namespace
	allowedNamespaces := map[string]bool{"dev": true, "qa": true, "stage": true, "prod": true}

	parts := strings.Fields(command.Text)
	if len(parts) != 1 {
		// Increment total errors metric
		totalErrors.WithLabelValues("/list").Inc()
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, "Invalid command format. Expected format: /list <namespace>")
	}

	namespace := parts[0]

	if !allowedNamespaces[namespace] {
		totalErrors.WithLabelValues("/list").Inc()
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Namespace `%s` is not allowed for listing pods. Please choose from: dev, qa, stage, prod.", namespace))
	}

	// Get the list of pods in the specified namespace
	podNames, versions, labelSelectors, err := getPodsInfoWithRetries(namespace, 3, 30*time.Second)
	if err != nil {
		totalErrors.WithLabelValues("/list").Inc()
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get pod information: %s", err))
	}

	// Create a formatted message with the list of pods, versions, statuses, and labels
	var messages []string
	for i, podName := range podNames {
		// Get the status of the pod
		status, errStatus := GetPodStatus(podName, namespace)

		// Handle potential errors
		if errStatus != nil {
			errMessage := "Error:"
			if errStatus != nil {
				errMessage += fmt.Sprintf(" Failed to get status for pod %s; ", podName)
			}
			messages = append(messages, errMessage)
			continue
		}

		// Create a message for each pod with version, status, and labels
		message := fmt.Sprintf("Pod: `%s`, Version: `%s`, Status: `%s`, Label: `%s`", podName, versions[i], status, labelSelectors[i])
		messages = append(messages, message)
	}

	// Use sendSuccessMessage to send the pod information
	return sendSuccessMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Namespace: `%s`\n%s", namespace, strings.Join(messages, "\n")))
}

// handleDiffCommand shows differences in deployments between environments.
func handleDiffCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	// Check if the command format is correct: /diff <label>
	parts := strings.Fields(command.Text)
	if len(parts) != 1 {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, "Invalid command format. Expected format: /diff <label>")
	}

	label := parts[0] // Retrieve the label for version comparison

	// Retrieve a list of namespaces to be checked
	orderedNamespaces := []string{"dev", "qa", "stage", "prod"}

	// Create maps for versions and statuses in each namespace
	versionMap := make(map[string]string)
	statusMap := make(map[string]string)

	// Check if at least one pod with the specified label is found
	foundPodWithLabel := false

	for _, ns := range orderedNamespaces {
		podNames, versions, labelSelectors, err := getPodsInfoWithRetries(ns, 3, 30*time.Second)
		if err != nil {
			return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get pod information in namespace %s: %s", ns, err))
		}

		for i, podName := range podNames {
			// Check if the label matches the specified one
			if labelSelectors[i] == label {
				foundPodWithLabel = true

				status, err := GetPodStatus(podName, ns)
				if err != nil {
					return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get pod status for %s in namespace %s: %s", podName, ns, err))
				}

				// Add only the running pods
				if status == "Running" {
					versionMap[ns] = versions[i]
					statusMap[ns] = status
					break // Break the loop after finding the first running pod with the matching label
				}
			}
		}
	}

	// Check again if at least one pod with the specified label is found
	if !foundPodWithLabel {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("No pods with label '%s' found in any namespace.", label))
	}

	// Create a message with the differences in versions and statuse
	var messages []string
	var allSameVersion = true
	var firstVersion string

	for _, ns := range orderedNamespaces {
		version := versionMap[ns]
		status := statusMap[ns]

		if firstVersion == "" && version != "" {
			firstVersion = version
		} else if version != "" && firstVersion != version {
			allSameVersion = false
		}

		if version != "" {
			messages = append(messages, fmt.Sprintf("Namespace: `%s`, Version: `%s`, Status: `%s`", ns, version, status))
		}
	}

	var finalMessage string
	if allSameVersion {
		finalMessage = "All applications are running the same version across namespaces. No promotion needed."
	} else {
		finalMessage = fmt.Sprintf("Differences found in application versions across namespaces:\n%s", strings.Join(messages, "\n"))
	}

	return sendSuccessMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, finalMessage)
}

// handlePromoteCommand handles promotion of deployments to the next environment.
func handlePromoteCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	parts := strings.Fields(command.Text)
	if len(parts) != 2 {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, "Invalid command format. Expected format: /promote <namespace> <label>")
	}

	namespace, label := parts[0], parts[1]

	// Check if namespace is allowed for promotion
	if !allowedNamespaces[namespace] {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Namespace `%s` is not allowed for promotion. Please choose from: qa, stage, prod.", namespace))
	}

	// Determine the source environment for the version
	var sourceNamespace string
	switch namespace {
	case "qa":
		sourceNamespace = "dev"
	case "stage":
		sourceNamespace = "qa"
	case "prod":
		sourceNamespace = "stage"
	}

	// Retrieve the current version with the "Running" status and the specified label
	var currentVersion string
	_, versions, labelSelectors, err := getPodsInfoWithRetries(namespace, 3, 30*time.Second)
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get pods in namespace `%s`: %s", namespace, err))
	}

	for i, labelSelector := range labelSelectors {
		if labelSelector == label {
			currentVersion = versions[i]
			break
		}
	}

	if currentVersion == "" {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("No pods with label `%s` found in namespace `%s` with status 'Running'.", label, namespace))
	}

	// Retrieve the version to be promoted from the source environment
	var versionToPromote string
	_, sourceVersions, sourceLabelSelectors, err := getPodsInfoWithRetries(sourceNamespace, 3, 30*time.Second)
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get pods in source namespace `%s`: %s", sourceNamespace, err))
	}

	for i, sourceLabelSelector := range sourceLabelSelectors {
		if sourceLabelSelector == label {
			versionToPromote = sourceVersions[i]
			break
		}
	}

	if versionToPromote == "" {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("No pods with label `%s` found in source namespace `%s` with status 'Running'.", label, sourceNamespace))
	}

	// Check if the current version is already the version to be promoted
	if currentVersion == versionToPromote {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Version `%s` is already deployed in namespace `%s`. No promotion needed.", currentVersion, namespace))
	}

	if err := checkReleaseHistoryTable(); err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to check release history table: %s", err.Error()))
	}

	// Update the version in the GitHub file and deploy
	err = updateVersionInGitHubFile(namespace, versionToPromote, fmt.Sprintf("Promote %s", label))
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to promote version `%s` to namespace `%s`: %s", versionToPromote, namespace, err))
	}

	// Asynchronously check the status of pods after promotion
	go checkPodStatusAfterPromotion(namespace, versionToPromote, command.Command+" "+command.Text, client, command.ChannelID, command.UserID)

	if err := AddReleaseHistory(namespace, versionToPromote, label); err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Не вдалося додати історію релізу: %s", err.Error()))
	}

	return sendSuccessMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Promotion of version `%s` to namespace `%s` has been initiated. Please wait for the deployment to complete.", versionToPromote, namespace))
}

// handleRollbackCommand handles rollback of deployments to a previous version.
func handleRollbackCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	parts := strings.Fields(command.Text)
	if len(parts) != 2 { // Очікуємо два аргументи: namespace та label
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, "Invalid command format. Expected format: /rollback <namespace> <label>")
	}

	namespace, label := parts[0], parts[1]

	// Checks if the namespace is permitted for rollback operations
	if !allowedNamespaces[namespace] {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Namespace `%s` is not allowed. Please choose from: qa, stage, prod.", namespace))
	}

	// Retrieves the current deployed version in the namespace with the specified label
	_, versions, labelSelectors, err := getPodsInfoWithRetries(namespace, 3, 30*time.Second)
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to get current version from pods in namespace `%s`: %s", namespace, err))
	}

	// Finds the current version associated with the label
	var currentVersion string
	for i, labelSelector := range labelSelectors {
		if labelSelector == label {
			currentVersion = versions[i]
			break
		}
	}

	if currentVersion == "" {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("No pods with label `%s` found in namespace `%s`.", label, namespace))
	}

	// Retrieves the version to roll back to from the release history
	rollbackVersion, err := getPreviousVersion(namespace, currentVersion, label) // Виправлено параметри функції
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to determine rollback version for namespace `%s`: %s", namespace, err))
	}

	if rollbackVersion == "" {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, "No previous version found for rollback.")
	}

	// Initiates the rollback process to the previous version
	err = updateVersionInGitHubFile(namespace, rollbackVersion, fmt.Sprintf("Rollback %s", label))
	if err != nil {
		return sendErrorMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Failed to rollback to version `%s` in namespace `%s`: %s", rollbackVersion, namespace, err))
	}

	// Asynchronously checks the status of pods after the rollback operation
	go checkPodStatusAfterPromotion(namespace, rollbackVersion, command.Command+" "+command.Text, client, command.ChannelID, command.UserID)

	return sendSuccessMessage(client, command.ChannelID, command.UserID, command.Command+" "+command.Text, fmt.Sprintf("Rollback to version `%s` in namespace `%s` has been initiated. Please wait for the deployment to complete.", rollbackVersion, namespace))
}

// sendErrorMessage sends an error message to the user in Slack.
func sendErrorMessage(client *slack.Client, channelID, userID, command, text string) (interface{}, error) {
	user, err := client.GetUserInfo(userID)
	if err != nil {
		log.Printf("Error fetching user info: %v", err)
		user = &slack.User{Name: "Unknown user"} // Fallback if user info is unavailable
	}

	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Command:* `%s`", command),
		Text:    fmt.Sprintf("Current Date and Time: %s\n%s", time.Now().Format("2006-01-02 15:04:05"), text),
		Color:   "#ff0000", // Red color indicates error
		Fields: []slack.AttachmentField{
			{
				Title: "Initializer",
				Value: user.Name,
				Short: true,
			},
		},
	}

	_, _, err = client.PostMessage(channelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return nil, fmt.Errorf("failed to post error message: %w", err)
	}
	return nil, nil
}

// sendSuccessMessage sends a success message to the user in Slack.
func sendSuccessMessage(client *slack.Client, channelID, userID, command, text string) (interface{}, error) {
	user, err := client.GetUserInfo(userID)
	if err != nil {
		log.Printf("Error fetching user info: %v", err)
		user = &slack.User{Name: "Unknown user"} // Fallback if user info is unavailable
	}

	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Command:* `%s`", command),
		Text:    fmt.Sprintf("Current Date and Time: %s\n%s", time.Now().Format("2006-01-02 15:04:05"), text),
		Color:   "#36a64f", // Green color indicates success
		Fields: []slack.AttachmentField{
			{
				Title: "Initializer",
				Value: user.Name,
				Short: true,
			},
		},
	}

	_, _, err = client.PostMessage(channelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return nil, fmt.Errorf("failed to post success message: %w", err)
	}
	return nil, nil
}

// HandleAppMentionEvent обробляє події згадок бота в Slack.
func handleAppMentionEvent(event *slackevents.AppMentionEvent, client *slack.Client) error {
	user, err := client.GetUserInfo(event.User)
	if err != nil {
		log.Println(err)
		return err
	}

	text := strings.ToLower(event.Text)

	attachment := slack.Attachment{}
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Date",
			Value: time.Now().String(),
		}, {
			Title: "Initializer",
			Value: user.Name,
		},
	}

	if strings.Contains(text, "hello") {
		attachment.Text = fmt.Sprintf("Hello %s", user.Name)
		attachment.Pretext = "Greetings"
		attachment.Color = "#4af030"
	} else {
		attachment.Text = fmt.Sprintf("How can I help you %s?", user.Name)
		attachment.Pretext = "How can I be of service"
		attachment.Color = "#3d3d3d"
	}

	_, _, err = client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// handleAppMentionEvent handles events where the bot is mentioned in Slack.
func handleInteractionEvent(interaction slack.InteractionCallback, client *slack.Client) error {
	// This is where we would handle the interaction
	// Switch depending on the Type
	log.Printf("The action called is: %s\n", interaction.ActionID)
	log.Printf("The response was of type: %s\n", interaction.Type)
	switch interaction.Type {
	case slack.InteractionTypeBlockActions:

		for _, action := range interaction.ActionCallback.BlockActions {
			log.Printf("%+v", action)
			log.Println("Selected option: ", action.SelectedOptions)

		}

	default:

	}

	return nil
}

// handleInteractionEvent handles interactive events in Slack (like button clicks).
func handleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client) error {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			err := handleAppMentionEvent(ev, client)
			if err != nil {
				log.Println(err)
			}
		}
	default:
		log.Println("unsupported event type")
	}
	return nil
}
