package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type User struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	ID            int    `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	AccountNumber string `json:"account_number"`
	AddressString string `json:"address_string"`
	IsActive      bool   `json:"is_active"`
	IsOrgAdmin    bool   `json:"is_org_admin"`
	IsInternal    bool   `json:"is_internal"`
	Locale        string `json:"locale"`
	OrgID         int    `json:"org_id"`
	DisplayName   string `json:"display_name"`
	Type          string `json:"type"`
}

var KEYCLOAK_SERVER string

func init() {
	KEYCLOAK_SERVER = os.Getenv("KEYCLOAK_SERVER")
}

type usersByInput struct {
	PrimaryEmail        string `json:"primaryEmail"`
	EmailStartsWith     string `json:"emailStartsWith"`
	PrincipalStartsWith string `json:"principalStartsWith"`
}

var USERS = []User{
	{
		Username:      "jdoe",
		Password:      "redhat",
		ID:            123456,
		Email:         "jdoe@redhat.com",
		FirstName:     "John",
		LastName:      "Doe",
		AccountNumber: "000006",
		AddressString: "Not Known",
		IsActive:      true,
		IsOrgAdmin:    true,
		IsInternal:    false,
		Locale:        "en_US",
		OrgID:         1234567,
		DisplayName:   "JDOE",
		Type:          "User",
	},
	{
		Username:      "mdoe",
		Password:      "redhat",
		ID:            123457,
		Email:         "mdoe@redhat.com",
		FirstName:     "Marge",
		LastName:      "Doe",
		AccountNumber: "000006",
		AddressString: "Not Known",
		IsActive:      true,
		IsOrgAdmin:    false,
		IsInternal:    false,
		Locale:        "en_US",
		OrgID:         1234567,
		DisplayName:   "JDOE",
		Type:          "User",
	},
}

type Resp struct {
	User      User   `json:"user"`
	Mechanism string `json:"mechanism"`
}

type AccV2Resp struct {
	Users     []User `json:"users"`
	UserCount int    `json:"userCount"`
}

type Realm struct {
	Realm     string `json:"realm"`
	PublicKey string `json:"public_key"`
}

type V1UserInput struct {
	Users []string `json:"users"`
}

func findUserById(username string) (*User, error) {
	for _, user := range USERS {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("User is not known")
}

func findUsersBy(accountNo string, adminOnly string, input *usersByInput, users *V1UserInput) []User {
	out := []User{}
	for _, user := range USERS {
		if adminOnly == "true" && !user.IsOrgAdmin {
			continue
		}
		if accountNo != "" && user.AccountNumber != accountNo {
			continue
		}
		if input != nil {
			if input.PrimaryEmail != "" && user.Email != input.PrimaryEmail {
				continue
			}
			if input.EmailStartsWith != "" && !strings.HasPrefix(user.Email, input.EmailStartsWith) {
				continue
			}
			if input.PrincipalStartsWith != "" && !strings.HasPrefix(user.Username, input.PrincipalStartsWith) {
				continue
			}
		}
		if users != nil {
			found := false
			for _, userCheck := range users.Users {
				if userCheck == user.Username {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, user)
	}
	return out
}

func jwtHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(fmt.Sprintf("%s/auth/realms/redhat-external/", KEYCLOAK_SERVER))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	realm := &Realm{}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &realm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, realm.PublicKey)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "no auth header found", http.StatusForbidden)
		return
	}
	if !strings.Contains(auth, "Basic") {
		http.Error(w, "auth header is not basic", http.StatusForbidden)
		return
	}

	data, err := base64.StdEncoding.DecodeString(auth[6:])

	if err != nil {
		http.Error(w, "could not split header", http.StatusForbidden)
		return
	}
	parts := strings.Split(string(data), ":")

	user := parts[0]

	userObj, err := findUserById(user)
	if err != nil {
		http.Error(w, "user not found", http.StatusForbidden)
		return
	}
	if userObj.Password != string(parts[1]) {
		http.Error(w, "bad creds", http.StatusUnauthorized)
		return
	} else {
		respObj := Resp{
			User:      *userObj,
			Mechanism: "Basic",
		}
		str, err := json.Marshal(respObj)
		if err != nil {
			http.Error(w, "could not create response", http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(str))
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
}

func usersV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filt := &V1UserInput{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "malformed input", http.StatusInternalServerError)
		return
	}
	if string(data) != "" {
		err = json.Unmarshal(data, filt)
		if err != nil {
			http.Error(w, "malformed input", http.StatusInternalServerError)
			return
		}
	}
	users := findUsersBy("", "false", nil, filt)

	str, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(str))
}

func usersV1Handler(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	accountId := urlParts[2]
	switch {
	case urlParts[3] == "users" && r.Method == "GET":
		adminOnly := r.URL.Query().Get("admin_only")
		users := findUsersBy(accountId, adminOnly, nil, nil)
		str, err := json.Marshal(users)
		if err != nil {
			http.Error(w, "could not create response", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(str))
	case urlParts[3] == "usersBy" && r.Method == "POST":
		filt := &usersByInput{}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "malformed input", http.StatusInternalServerError)
			return
		}
		if string(data) != "" {
			err = json.Unmarshal(data, filt)
			if err != nil {
				http.Error(w, "malformed input", http.StatusInternalServerError)
				return
			}
		}
		users := findUsersBy(accountId, "false", filt, nil)
		str, err := json.Marshal(users)
		if err != nil {
			http.Error(w, "could not create response", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(str))
	}
}

func usersV2Handler(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	accountId := urlParts[2]
	users := findUsersBy(accountId, "false", nil, nil)
	respObj := AccV2Resp{
		Users:     users,
		UserCount: len(users),
	}
	str, err := json.Marshal(respObj)
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(str))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
	switch {
	case r.URL.Path == "/":
		statusHandler(w, r)
	case r.URL.Path == "/v1/users":
		usersV1(w, r)
	case r.URL.Path == "/v1/jwt":
		jwtHandler(w, r)
	case r.URL.Path == "/v1/auth":
		authHandler(w, r)
	case r.URL.Path[:12] == "/v1/accounts":
		usersV1Handler(w, r)
	case r.URL.Path[:12] == "/v2/accounts":
		usersV2Handler(w, r)
	}
}

func main() {
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
