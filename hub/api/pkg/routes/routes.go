package routes

import (
	"github.com/gorilla/mux"
	"github.com/tektoncd/hub/api/pkg/api"
	"github.com/tektoncd/hub/api/pkg/app"
)

// Register registers all routes with router
func Register(r *mux.Router, conf app.Config) {
	api := api.New(conf)

	r.HandleFunc("/resources", api.GetAllResources).Methods("GET")                          //
	r.HandleFunc("/resource/{resourceID}/versions", api.GetResourceVersions).Methods("GET") //
	r.HandleFunc("/categories", api.GetAllCategorieswithTags).Methods("GET")                //
	r.HandleFunc("/resource/{resourceID}/rating", api.GetResourceRating).Methods("GET")     //
	r.HandleFunc("/resource/{resourceID}/rating", api.UpdateResourceRating).Methods("PUT")  //
	r.HandleFunc("/oauth/redirect", api.GithubAuth).Methods("POST")                         //
	r.HandleFunc("/resources/sync", api.SyncResources).Methods("POST")                      //

}
