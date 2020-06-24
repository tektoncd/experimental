package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tektoncd/hub/api/pkg/app"
	"github.com/tektoncd/hub/api/pkg/routes"
)

func main() {
	app, err := app.FromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to initialise: %s", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	db := app.DB()
	db.LogMode(true)

	//HTTP
	router := mux.NewRouter()
	routes.Register(router, app)

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedHeaders([]string{
			"X-Requested-With", "Content-Type", "Authorization",
		}),
		handlers.AllowedMethods([]string{
			"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE",
		}),
	)

	log := app.Logger()
	log.Infof("Listening on %s", app.Addr())
	log.Fatal(http.ListenAndServe(app.Addr(), cors(router)))
}
