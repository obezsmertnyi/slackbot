package main

import (
	"log"
	"os"

	"github.com/obezsmertnyi/slackbot/cmd"
    "k8s.io/client-go/rest"
)

func checkEnv(varName string) {
	value := os.Getenv(varName)
	if value == "" {
		log.Fatalf("Environment variable %s is not set. Please check your configuration.", varName)
	} else {
		log.Printf("Environment variable %s is set.", varName)
	}
}

func init() {
	// Check necessary environment variables
	checkEnv("SLACK_AUTH_TOKEN")
	checkEnv("SLACK_CHANNEL_ID")
	checkEnv("SLACK_APP_TOKEN")
	checkEnv("YOUR_GITHUB_TOKEN")
	checkEnv("GITHUB_OWNER")
	checkEnv("GITHUB_REPO")
	
	// Check KUBECONFIG or its alternatives
	checkKubeConfig()
}

func checkKubeConfig() {
	// Check if KUBECONFIG environment variable is set
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		log.Println("KUBECONFIG is set.")
	} else {
		// If KUBECONFIG is not set, check for KUBE_SERVER, KUBE_CA, and KUBE_TOKEN
		if kubeServer := os.Getenv("KUBE_SERVER"); kubeServer != "" {
			log.Println("KUBE_SERVER is set.")
			checkEnv("KUBE_CA")
			checkEnv("KUBE_TOKEN")
		} else {
			// If KUBE_SERVER, KUBE_CA, and KUBE_TOKEN are not set, check for in-cluster configuration
			if _, err := rest.InClusterConfig(); err != nil {
				log.Println("In-cluster configuration is not available.")
			} else {
				log.Println("Using in-cluster configuration.")
			}
		}
	}
}

func main() {
	// Initialize and start your application
	cmd.Execute()
}
