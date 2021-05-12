package controller

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v32/github"
)

type GitHubClientFactory struct {
	// These fields should be treated as a oneof.
	static *github.Client
	at     *ghinstallation.AppsTransport
}

func NewStatic(client *github.Client) *GitHubClientFactory {
	return &GitHubClientFactory{
		static: client,
	}
}

func NewApp(rt http.RoundTripper, appID int64, privateKeyPath string) (*GitHubClientFactory, error) {
	at, err := ghinstallation.NewAppsTransportKeyFromFile(rt, appID, os.Getenv("GITHUB_APP_KEY"))
	if err != nil {
		return nil, fmt.Errorf("error reading GitHub App private key: %v", err)
	}

	return &GitHubClientFactory{
		at: at,
	}, nil
}

// NewClient provides a GitHub API client based on the configured factory.
// If an Static factory is configured, the installation ID is ignored. An empty installation ID is valid for static factories.
// If an App factory is configured, the installation ID is used to configure an installation client.
func (f *GitHubClientFactory) NewClient(installationID string) (*github.Client, error) {
	if f.static != nil {
		return f.static, nil
	}

	n, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: ghinstallation.NewFromAppsTransport(f.at, n)}), nil

}
