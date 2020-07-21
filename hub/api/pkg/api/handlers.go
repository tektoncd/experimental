package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tektoncd/hub/api/pkg/app"
	"github.com/tektoncd/hub/api/pkg/db/model"
	"github.com/tektoncd/hub/api/pkg/service"
	"github.com/tektoncd/hub/api/pkg/sync"
	"go.uber.org/zap"
)

const (
	errParseForm   string = "form-parse-error"
	errMissingKey  string = "key-not-found"
	errInvalidType string = "invalid-type"
	errAuthFailure string = "auth-failed"
)

type Api struct {
	app     app.Config
	Log     *zap.SugaredLogger
	service service.Service
}

func New(app app.Config) *Api {
	return &Api{
		app:     app,
		Log:     app.Logger().With("name", "api"),
		service: service.New(app),
	}
}

type ResponseError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *ResponseError) Error() string {
	return e.Detail
}

func intQueryVar(r *http.Request, key string, def int) (int, *ResponseError) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return def, nil
	}

	res, err := strconv.Atoi(value)
	if err != nil {
		return def, &ResponseError{
			Code:   errInvalidType,
			Detail: "query param " + key + " must be an int"}
	}

	return res, nil
}

func intPathVar(r *http.Request, key string) (int, *ResponseError) {
	value := mux.Vars(r)[key]

	res, err := strconv.Atoi(value)
	if err != nil {
		return 0, &ResponseError{
			Code:   errInvalidType,
			Detail: "Path param " + key + " must be an int"}
	}

	return res, nil
}

func invalidRequest(w http.ResponseWriter, status int, err *ResponseError) {
	type emptyList []interface{}

	res := struct {
		Data   emptyList       `json:"data"`
		Errors []ResponseError `json:"errors"`
	}{
		Data:   emptyList{},
		Errors: []ResponseError{*err},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(res)
}

func errorResponse(w http.ResponseWriter, err *ResponseError) {
	type emptyList []interface{}

	res := struct {
		Data   emptyList       `json:"data"`
		Errors []ResponseError `json:"errors"`
	}{
		Data:   emptyList{},
		Errors: []ResponseError{*err},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// Ok writes json encoded resources to ResponseWriter
func (api *Api) Ok(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetAllResources writes json encoded resources to ResponseWriter
func (api *Api) GetAllResources(w http.ResponseWriter, r *http.Request) {
	limit, err := intQueryVar(r, "limit", 100)
	if err != nil {
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	resources, resourceErr := api.service.Resource().All(service.Filter{Limit: limit})
	if resourceErr != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "db-error", Detail: resourceErr.Error()})
		return
	}
	res := struct {
		Data   []service.ResourceDetail `json:"data"`
		Errors []ResponseError          `json:"errors"`
	}{
		Data:   resources,
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetResourceVersions writes json encoded resources to ResponseWriter
func (api *Api) GetResourceVersions(w http.ResponseWriter, r *http.Request) {

	resourceID, err := intPathVar(r, "resourceID")
	if err != nil {
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	resourceVersions, retErr := api.service.Resource().AllVersions(uint(resourceID))
	if retErr != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "db-error", Detail: retErr.Error()})
		return
	}

	res := struct {
		Data   []service.ResourceVersionDetail `json:"data"`
		Errors []ResponseError                 `json:"errors"`
	}{
		Data:   resourceVersions,
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetAllCategorieswithTags writes json encoded list of categories to Responsewriter
func (api *Api) GetAllCategorieswithTags(w http.ResponseWriter, r *http.Request) {

	categories, retErr := api.service.Category().All()
	if retErr != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "db-error", Detail: retErr.Error()})
		return
	}

	res := struct {
		Data   []service.CategoryDetail `json:"data"`
		Errors []ResponseError          `json:"errors"`
	}{
		Data:   categories,
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetResourceRating returns user's rating of a resource
func (api *Api) GetResourceRating(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("Authorization")
	if token == "" {
		invalidRequest(w, http.StatusBadRequest, &ResponseError{Code: "invalid-header", Detail: "Token is missing in header"})
		return
	}
	resourceID, err := intPathVar(r, "resourceID")
	if err != nil {
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	userID, userErr := api.service.User().VerifyJWT(token)
	if userErr != nil {
		invalidRequest(w, http.StatusUnauthorized, &ResponseError{Code: "invalid-token", Detail: userErr.Error()})
		return
	}

	ids := service.UserResource{
		UserID:     userID,
		ResourceID: resourceID,
	}

	rating, _ := api.service.Rating().GetResourceRating(ids)

	res := struct {
		Data   service.RatingDetails `json:"data"`
		Errors []ResponseError       `json:"errors"`
	}{
		Data:   rating,
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// UpdateResourceRating will add/update a user's re rating
func (api *Api) UpdateResourceRating(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("Authorization")
	if token == "" {
		err := &ResponseError{Code: "invalid-header", Detail: "JWT is missing"}
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}
	resourceID, err := intPathVar(r, "resourceID")
	if err != nil {
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	userID, userErr := api.service.User().VerifyJWT(token)
	if userErr != nil {
		invalidRequest(w, http.StatusUnauthorized, &ResponseError{Code: "invalid-token", Detail: userErr.Error()})
		return
	}

	ratingRequestBody := service.UpdateRatingDetails{UserID: uint(userID), ResourceID: uint(resourceID)}
	jsonErr := json.NewDecoder(r.Body).Decode(&ratingRequestBody)
	if jsonErr != nil {
		err := &ResponseError{Code: "invalid-body", Detail: jsonErr.Error()}
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	if ratingRequestBody.ResourceRating > 5 {
		err := &ResponseError{Code: "invalid-body", Detail: "Rating should be in range 1 to 5"}
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}

	avgRating, retErr := api.service.Rating().UpdateResourceRating(ratingRequestBody)
	if retErr != nil {
		invalidRequest(w, http.StatusUnauthorized, &ResponseError{Code: "db-error", Detail: retErr.Error()})
		return
	}

	type emptyList []interface{}
	res := struct {
		Data   service.ResourceAverageRating `json:"data"`
		Errors []ResponseError               `json:"errors"`
	}{
		Data:   avgRating,
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GithubAuth handles OAuth by Github
func (api *Api) GithubAuth(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("Authorization")
	if token == "" {
		err := &ResponseError{Code: "invalid-header", Detail: "Authorization Token is missing"}
		invalidRequest(w, http.StatusBadRequest, err)
		return
	}
	api.Log.Info("User's OAuthAuthorizeToken - ", token)

	accessToken, err := api.service.User().GetGitHubAccessToken(service.OAuthAuthorizeToken{Token: token})
	if err != nil {
		err := &ResponseError{Code: "invalid-token", Detail: err.Error()}
		invalidRequest(w, http.StatusUnauthorized, err)
		return
	}

	userDetails, err := api.service.User().GetUserDetails(service.OAuthAccessToken{AccessToken: accessToken})
	if err != nil {
		err := &ResponseError{Code: "github-error", Detail: err.Error()}
		invalidRequest(w, http.StatusInternalServerError, err)
		return
	}

	user, retErr := api.service.User().Add(userDetails)
	if retErr != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "db-error", Detail: retErr.Error()})
		return
	}

	resToken, retErr := api.service.User().GenerateJWT(user)
	if retErr != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "jwt-error", Detail: retErr.Error()})
		return
	}

	res := struct {
		Data   service.OAuthResponse `json:"data"`
		Errors []ResponseError       `json:"errors"`
	}{
		Data:   service.OAuthResponse{Token: resToken},
		Errors: []ResponseError{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// SyncResources will sync the database with catalog
func (api *Api) SyncResources(w http.ResponseWriter, r *http.Request) {
	s := sync.New(api.app, "/tmp/catalog")
	s.Init()

	catalog := model.Catalog{}
	if err := api.app.DB().Model(&model.Catalog{}).First(&catalog).Error; err != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "internal-error", Detail: "Failed to get catalog"})
		return
	}

	job := model.SyncJob{Catalog: catalog, Status: "queued"}
	if err := api.app.DB().Create(&job).Error; err != nil {
		invalidRequest(w, http.StatusInternalServerError, &ResponseError{Code: "internal-error", Detail: "Failed to create a job"})
		return
	}

	s.Sync(context.Background())
}
