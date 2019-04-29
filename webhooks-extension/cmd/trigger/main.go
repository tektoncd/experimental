package main

import (
	"context"
	"log"
	"os"

	cloudevents "github.com/cloudevents/sdk-go"
	gh "github.com/google/go-github/github"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/github"
	tknClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"

	"golang.org/x/oauth2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := gh.NewClient(tc)

	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	var cfg *rest.Config

	cfgEnv := os.Getenv("KUBECONFIG")
	if len(cfgEnv) == 0 {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Error building config from %v:\n", err)
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", cfgEnv)
		if err != nil {
			log.Fatalf("Error building config from %v: %v\n", cfgEnv, err)
		}
	}

	tektonClient, err := tknClient.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error creating tekton client: %v", err)
	}

	trigger := github.NewTrigger(ghClient, tektonClient)

	log.Printf("starting trigger service")
	err = c.StartReceiver(ctx, trigger.Handler)
	if err != nil {
		log.Fatalf("failed to start receiver: %s", err)
	}
}
