package main

import (
	"github.com/tektoncd/experimental/workflows/pkg/reconciler/workflows"
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	sharedmain.Main("controller",
		workflows.NewController,
	)
}
