package keycloakuserservice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
)

type UserServiceClient struct {
	client *http.Client
}

func (userService *UserServiceClient) InitKeycloakUserServiceConnection() error {
	userService.client = &http.Client{
		Timeout: time.Duration(config.Get().KeyCloakUserServiceTimeout * int64(time.Second)),
	}

	return nil
}

func (userService *UserServiceClient) GetUsers(token string, u models.UserBody, q models.UserV1Query) (models.Users, error) {
	users := models.Users{Users: []models.User{}}
	url, err := createV1RequestURL(u, q)
	if err != nil {
		return users, err
	}

	body, err := userService.sendKeycloakGetRequest(url, token)
	if err != nil {
		l.Log.Error(err, "/v3/users error sending request")
		return users, err
	}

	unmarshaledResponse := models.KeycloakResponses{}
	err = json.Unmarshal(body, &unmarshaledResponse)
	if err != nil {
		return users, err
	}

	return keycloakResponseToUsers(unmarshaledResponse.Users), err
}

func (userService *UserServiceClient) GetAccountV3Users(orgID string, token string, q models.UserV3Query) (models.Users, error) {
	users := models.Users{Users: []models.User{}}
	url, err := createV3UsersRequestURL(orgID, q)
	if err != nil {
		return users, err
	}

	body, err := userService.sendKeycloakGetRequest(url, token)
	if err != nil {
		l.Log.Error(err, "/v3/users error sending request")
		return users, err
	}

	unmarshaledResponse := models.KeycloakResponses{}
	err = json.Unmarshal(body, &unmarshaledResponse)
	if err != nil {
		return users, err
	}

	return keycloakResponseToUsers(unmarshaledResponse.Users), nil
}

func (userService *UserServiceClient) GetAccountV3UsersBy(orgID string, token string, q models.UserV3Query, usersByBody models.UsersByBody) (models.Users, error) {
	users := models.Users{Users: []models.User{}}
	url, err := createV3UsersByRequestURL(orgID, q, usersByBody)
	if err != nil {
		return users, err
	}

	body, err := userService.sendKeycloakGetRequest(url, token)
	if err != nil {
		l.Log.Error(err, "/v3/usersBy error sending request")
		return users, err
	}

	unmarshaledResponse := models.KeycloakResponses{}
	err = json.Unmarshal(body, &unmarshaledResponse)
	if err != nil {
		return users, err
	}

	return keycloakResponseToUsers(unmarshaledResponse.Users), nil
}

func (userService *UserServiceClient) sendKeycloakGetRequest(url *url.URL, token string) ([]byte, error) {
	var responseBody []byte

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return responseBody, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := userService.client.Do(req)
	if err != nil {
		l.Log.Error(err, "error fetching keycloak response")
		return responseBody, err
	}

	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		l.Log.Error(err, "error reading keycloak response body")
		return responseBody, err
	}

	// Close response body
	defer resp.Body.Close()

	return responseBody, nil
}

// MAKE response to users function to massage data back to regular format
func createV1RequestURL(usernames models.UserBody, q models.UserV1Query) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s%s/users?limit=100", config.Get().KeyCloakUserServiceScheme, config.Get().KeyCloakUserServiceHost, config.Get().KeyCloakUserServicePort))
	if err != nil {
		return nil, fmt.Errorf("error creating (keycloak) /users url: %s", err)
	}

	queryParams := url.Query()

	if q.QueryBy != "" {
		queryParams.Add("order", q.QueryBy)
	}

	if q.SortOrder != "" {
		queryParams.Add("direction", q.SortOrder)
	}

	queryParams.Add("usernames", createUsernamesQuery(usernames.Users))

	url.RawQuery = queryParams.Encode()
	return url, err
}

func createV3UsersRequestURL(orgID string, q models.UserV3Query) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s%s/users", config.Get().KeyCloakUserServiceScheme, config.Get().KeyCloakUserServiceHost, config.Get().KeyCloakUserServicePort))
	if err != nil {
		return nil, fmt.Errorf("error creating (keycloak) /v3/users url: %s", err)
	}
	queryParams := url.Query()

	// default ordering
	queryParams.Add("order", "username")
	queryParams.Add("direction", "asc")

	if q.SortOrder != "" {
		queryParams.Set("direction", q.SortOrder)
	}

	queryParams.Add("org_id", orgID)
	queryParams.Add("limit", strconv.Itoa(q.Limit))
	queryParams.Add("offset", strconv.Itoa(q.Offset))

	url.RawQuery = queryParams.Encode()

	return url, err
}

func createV3UsersByRequestURL(orgID string, q models.UserV3Query, usersByBody models.UsersByBody) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s%s/users", config.Get().KeyCloakUserServiceScheme, config.Get().KeyCloakUserServiceHost, config.Get().KeyCloakUserServicePort))
	if err != nil {
		return nil, fmt.Errorf("error creating (keycloak) /v3/usersBy url: %s", err)
	}
	queryParams := url.Query()

	if usersByBody.EmailStartsWith != "" {
		queryParams.Add("emails", usersByBody.EmailStartsWith)
	}

	if usersByBody.PrimaryEmail != "" {
		queryParams.Add("emails", usersByBody.PrimaryEmail)
	}

	if usersByBody.PrincipalStartsWith != "" {
		queryParams.Add("usernames", usersByBody.PrincipalStartsWith)
	}

	// default ordering
	queryParams.Add("order", "username")
	queryParams.Add("direction", "asc")

	if q.SortOrder != "" {
		queryParams.Set("direction", q.SortOrder)
	}

	queryParams.Add("org_id", orgID)
	queryParams.Add("limit", strconv.Itoa(q.Limit))
	queryParams.Add("offset", strconv.Itoa(q.Offset))

	url.RawQuery = queryParams.Encode()

	return url, err
}

func createUsernamesQuery(usernames []string) string {
	usernameQuery := ""

	for _, username := range usernames {
		if usernameQuery == "" {
			usernameQuery += username
		} else {
			usernameQuery += fmt.Sprintf(",%s", username)
		}
	}

	return usernameQuery
}

func keycloakResponseToUsers(r []models.KeycloakResponse) models.Users {
	users := models.Users{Users: []models.User{}}

	for _, response := range r {
		users.AddUser(models.User{
			Username:      response.Username,
			ID:            response.ID,
			Email:         response.Email,
			FirstName:     response.FirstName,
			LastName:      response.LastName,
			AddressString: "",
			IsActive:      response.IsActive,
			IsInternal:    response.IsInternal,
			Locale:        "en_US",
			OrgID:         response.OrgID,
			DisplayName:   response.UserID,
			Type:          response.Type,
			IsOrgAdmin:    response.IsOrgAdmin,
		})
	}

	return users
}
