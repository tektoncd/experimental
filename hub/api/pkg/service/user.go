package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/tektoncd/hub/api/pkg/app"
	"github.com/tektoncd/hub/api/pkg/db/model"
	"go.uber.org/zap"
)

// User Service
type User struct {
	db  *gorm.DB
	log *zap.SugaredLogger
	gh  *app.GitHub
}

// GHUserDetails model represents user details
type GHUserDetails struct {
	UserName string `json:"login"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

// OAuthAuthorizeToken represents Authorise token
type OAuthAuthorizeToken struct {
	Token string `json:"token"`
}

// OAuthAccessToken represents Access token
type OAuthAccessToken struct {
	AccessToken string `json:"access_token"`
}

// OAuthResponse Api reponse
type OAuthResponse struct {
	Token string `json:"token"`
}

// Claims Object to decode JWT
type Claims struct {
	Authorized bool `json:"authorized"`
	ID         int  `json:"id"`
}

// VerifyToken checks if user token is associated with a user and returns its id
func (u *User) VerifyToken(token string) int {

	var id int
	u.db.Table("users").Where("token = ?", token).Select("id").Row().Scan(&id)

	return id
}

// Add insert user in database
func (u *User) Add(ud GHUserDetails) (*model.User, error) {

	user := &model.User{}
	if err := u.db.Where("user_name = ?", ud.UserName).
		Assign(&model.User{Token: ud.Token}).
		FirstOrCreate(&model.User{
			UserName: ud.UserName,
			Name:     ud.Name,
			Email:    ud.Email,
			Token:    ud.Token,
		}).Scan(&user).Error; err != nil {
		return &model.User{}, errors.New("Failed to add user to db")
	}

	return user, nil
}

// GetOAuthURL return url for getting access token
func (u *User) GetOAuthURL(token string) string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s",
		u.gh.OAuthClientID, u.gh.OAuthSecret, token)
}

// GetGitHubAccessToken fetch User GithubAccessToken
func (u *User) GetGitHubAccessToken(authToken OAuthAuthorizeToken) (string, error) {

	reqURL := u.GetOAuthURL(authToken.Token)
	u.log.Info("User's Request for GH Token - ", reqURL)

	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stdout, "could not create HTTP request: %v", err)
	}
	req.Header.Set("accept", "application/json")

	httpClient := http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stdout, "could not send HTTP request: %v", err)
	}
	defer res.Body.Close()

	var oat OAuthAccessToken
	if err := json.NewDecoder(res.Body).Decode(&oat); err != nil {
		fmt.Fprintf(os.Stdout, "could not parse JSON response: %v", err)
	}
	if oat.AccessToken == "" {
		u.log.Info("Failed to get user's access token.")
		return "", errors.New("Failed to get Access Token")
	}
	u.log.Info("User's Access Token - ", oat.AccessToken)

	return oat.AccessToken, nil
}

// GetUserDetails fetch user details using GitHub Api
func (u *User) GetUserDetails(oat OAuthAccessToken) (GHUserDetails, error) {

	httpClient := http.Client{}
	reqURL := fmt.Sprintf("https://api.github.com/user")

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Authorization", "token "+oat.AccessToken)
	if err != nil {
		u.log.Error(err)
	}
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stdout, "could not send HTTP request: %v", err)
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	userDetails := GHUserDetails{}
	if err := json.Unmarshal(body, &userDetails); err != nil {
		u.log.Error(err)
	}
	if userDetails.UserName == "" {
		return GHUserDetails{}, errors.New("Failed to get User Details from GitHub")
	}
	userDetails.Token = oat.AccessToken
	u.log.Info("User's GitHub Username - ", userDetails.UserName)

	return userDetails, nil
}

// GenerateJWT a new JWT token
func (u *User) GenerateJWT(user *model.User) (string, error) {

	jwtSigningKey := []byte(u.gh.JWTSigningKey)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["authorized"] = true
	claims["id"] = user.ID
	claims["name"] = user.Name
	claims["userName"] = user.UserName

	tokenString, err := token.SignedString(jwtSigningKey)
	if err != nil {
		return "", errors.New("Failed to create JWT")
	}
	return tokenString, nil
}

// VerifyJWT verifies a JWT token
func (u *User) VerifyJWT(token string) (int, error) {

	jwtSecretKey := []byte(u.gh.JWTSigningKey)
	splitToken := strings.Split(token, "Bearer ")
	reqToken := splitToken[1]
	var c Claims
	parsedToken, _ := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Failed to Decode JWT")
		}
		return []byte(jwtSecretKey), nil
	})
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		mapstructure.Decode(claims, &c)
	} else {
		return 0, errors.New("Invalid JWT")
	}
	return c.ID, nil
}
