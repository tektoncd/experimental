package main

import (
	"fmt"
	"os"

	"github.com/tektoncd/hub/api/pkg/app"
	"github.com/tektoncd/hub/api/pkg/db/model"

	// Blank
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func main() {
	app, err := app.BaseConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to initialise: %s", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	log := app.Logger()
	if err = model.Migrate(app.DB(), log); err != nil {
		log.Fatal("DB initialisation failed", err)
	}
	log.Info("DB initialisation successful")
}
