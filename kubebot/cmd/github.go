package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

var githubClient *github.Client

// Initializes the GitHub client using the provided token.
func initGitHubClient() {
	ctx := context.Background()
	token := os.Getenv("YOUR_GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	githubClient = github.NewClient(tc)
}

// Updates the version in a specific GitHub file within a repository.
func updateVersionInGitHubFile(namespace, newVersion, commandType string) error {
    ctx := context.Background()
    owner := os.Getenv("GITHUB_OWNER")
    repo := os.Getenv("GITHUB_REPO")
    path := fmt.Sprintf("clusters/kbot/%s/image-policy.yaml", namespace)

    // Determine the branch based on the namespace
    branch := "main" // Default branch
    if namespace == "prod" {
        branch = "prod" // Switch to the production branch for the 'prod' namespace
    }

    // Setting options to get the file content from the specified branch
    opts := &github.RepositoryContentGetOptions{Ref: branch}

    // Retrieving the current content of the file
    fileContent, _, _, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, opts)
    if err != nil {
        return fmt.Errorf("failed to retrieve file content: %w", err)
    }

    decodedContent, err := fileContent.GetContent() // Decoding the content of the file
    if err != nil {
        return fmt.Errorf("failed to decode file content: %w", err)
    }

    // Check if the 'range' field needs to be updated
    if !strings.Contains(decodedContent, fmt.Sprintf("range: '%s'", newVersion)) {
        // Updating the 'range' field and commit the changes
        updatedContent := regexp.MustCompile(`range: '.*'`).ReplaceAllString(decodedContent, fmt.Sprintf("range: '%s'", newVersion))

        message := fmt.Sprintf("%s version %s to %s", commandType, newVersion, namespace) // Creating commit message
        updateOpts := &github.RepositoryContentFileOptions{
            Message: github.String(message),
            Content: []byte(updatedContent),
            SHA:     fileContent.SHA,
            Branch:  github.String(branch),
        }

        // Updating the file with the new content
        _, _, err = githubClient.Repositories.UpdateFile(ctx, owner, repo, path, updateOpts)
        if err != nil {
            return fmt.Errorf("failed to update file: %w", err)
        }
    }

    return nil
}
