module github.com/tektoncd/hub/api

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/martian v2.1.0+incompatible
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/jinzhu/gorm v1.9.12
	github.com/joho/godotenv v1.3.0
	github.com/labstack/gommon v0.3.0
	github.com/lib/pq v1.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pkg/errors v0.8.1
	github.com/tektoncd/pipeline v0.11.0
	go.uber.org/zap v1.13.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/gormigrate.v1 v1.6.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.1
)

go 1.13
