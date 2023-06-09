package config

import (
	"os"
	"strconv"
)

type MbopConfig struct {
	FromEmail              string
	SESRegion              string
	SESAccessKey           string
	SESSecretKey           string
	MailerModule           string
	JwtModule              string
	JwkURL                 string
	UsersModule            string
	CognitoAppClientID     string
	CognitoAppClientSecret string
	CognitoScope           string
	OauthTokenURL          string
	AmsURL                 string
	TokenTTL               string
	TokenKID               string
	PrivateKey             string
	PublicKey              string
	DisableCatchall        bool
	IsInternalLabel        string
	Debug                  bool

	KeyCloakScheme         string
	KeyCloakHost           string
	KeyCloakPort           string
	KeyCloakTimeout        int64
	KeyCloakTokenUsername  string
	KeyCloakTokenPassword  string
	KeyCloakTokenGrantType string
	KeyCloakTokenClientID  string

	StoreBackend     string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string

	Port    string
	TLSPort string
	UseTLS  bool
	CertDir string
}

var conf *MbopConfig

func Get() *MbopConfig {
	if conf != nil {
		return conf
	}

	disableCatchAll, _ := strconv.ParseBool(fetchWithDefault("DISABLE_CATCHALL", "false"))
	debug, _ := strconv.ParseBool(fetchWithDefault("DEBUG", "false"))
	certDir := fetchWithDefault("CERT_DIR", "/certs")
	keyCloakTimeout, _ := strconv.ParseInt(fetchWithDefault("KEYCLOAK_TIMEOUT", "60"), 0, 64)

	var tls bool
	_, err := os.Stat(certDir + "/tls.crt")
	if err == nil {
		tls = true
	}

	c := &MbopConfig{
		UsersModule:     fetchWithDefault("USERS_MODULE", ""),
		JwtModule:       fetchWithDefault("JWT_MODULE", ""),
		JwkURL:          fetchWithDefault("JWK_URL", ""),
		MailerModule:    fetchWithDefault("MAILER_MODULE", "print"),
		FromEmail:       fetchWithDefault("FROM_EMAIL", "no-reply@redhat.com"),
		SESRegion:       fetchWithDefault("SES_REGION", "us-east-1"),
		SESAccessKey:    fetchWithDefault("SES_ACCESS_KEY", ""),
		SESSecretKey:    fetchWithDefault("SES_SECRET_KEY", ""),
		DisableCatchall: disableCatchAll,

		DatabaseHost:     fetchWithDefault("DATABASE_HOST", "localhost"),
		DatabasePort:     fetchWithDefault("DATABASE_PORT", "5432"),
		DatabaseUser:     fetchWithDefault("DATABASE_USER", "postgres"),
		DatabasePassword: fetchWithDefault("DATABASE_PASSWORD", ""),
		DatabaseName:     fetchWithDefault("DATABASE_NAME", "mbop"),
		StoreBackend:     fetchWithDefault("STORE_BACKEND", "memory"),

		CognitoAppClientID:     fetchWithDefault("COGNITO_APP_CLIENT_ID", ""),
		CognitoAppClientSecret: fetchWithDefault("COGNITO_APP_CLIENT_SECRET", ""),
		CognitoScope:           fetchWithDefault("COGNITO_SCOPE", ""),
		OauthTokenURL:          fetchWithDefault("OAUTH_TOKEN_URL", ""),
		AmsURL:                 fetchWithDefault("AMS_URL", ""),
		TokenTTL:               fetchWithDefault("TOKEN_TTL_DURATION", "5m"),
		TokenKID:               fetchWithDefault("TOKEN_KID", ""),
		PrivateKey:             fetchWithDefault("TOKEN_PRIVATE_KEY", ""),
		PublicKey:              fetchWithDefault("TOKEN_PUBLIC_KEY", ""),
		IsInternalLabel:        fetchWithDefault("IS_INTERNAL_LABEL", ""),
		Debug:                  debug,

		KeyCloakHost:           fetchWithDefault("KEYCLOAK_HOST", "localhost"),
		KeyCloakPort:           fetchWithDefault("KEYCLOAK_PORT", ":8000"),
		KeyCloakScheme:         fetchWithDefault("KEYCLOAK_SCHEME", "http"),
		KeyCloakTimeout:        keyCloakTimeout,
		KeyCloakTokenUsername:  fetchWithDefault("KEYCLOAK_TOKEN_USERNAME", "admin"),
		KeyCloakTokenPassword:  fetchWithDefault("KEYCLOAK_TOKEN_PASSWORD", "admin"),
		KeyCloakTokenGrantType: fetchWithDefault("KEYCLOAK_TOKEN_GRANT_TYPE", "password"),
		KeyCloakTokenClientID:  fetchWithDefault("KEYCLOAK_TOKEN_CLIENT_ID", "admin-cli"),

		Port:    fetchWithDefault("PORT", "8090"),
		TLSPort: fetchWithDefault("TLS_PORT", "8890"),
		UseTLS:  tls,
		CertDir: certDir,
	}

	conf = c
	return conf
}

func fetchWithDefault(name, defaultValue string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return defaultValue
}

// TO BE USED FROM TESTING ONLY.
func Reset() {
	conf = nil
}
