package config

import (
	"fmt"
	"log"
	"os"
)

type Config struct {
	CppHost       string
	KeycloakURL   string
	KeycloakRealm string
	ClientID      string
	ClientSecret  string
	DatabaseDSN   string
	GrafanaURL    string
}

func LoadConfig() *Config {
	envVars := map[string]string{
		"CPP_HOST":          os.Getenv("CPP_HOST"),
		"KEYCLOAK_URL":      os.Getenv("KEYCLOAK_URL"),
		"KEYCLOAK_REALM":    os.Getenv("KEYCLOAK_REALM"),
		"CLIENT_ID":         os.Getenv("CLIENT_ID"),
		"CLIENT_SECRET":     os.Getenv("CLIENT_SECRET"),
		"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
		"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
		"GRAFANA_URL":       os.Getenv("GRAFANA_URL"),
	}

	if missingVar := checkEnvVars(envVars); missingVar != "" {
		log.Fatalf("required configuration is missing: %s", missingVar)
	}

	cfg := &Config{
		CppHost:       envVars["CPP_HOST"],
		KeycloakURL:   envVars["KEYCLOAK_URL"],
		KeycloakRealm: envVars["KEYCLOAK_REALM"],
		ClientID:      envVars["CLIENT_ID"],
		ClientSecret:  envVars["CLIENT_SECRET"],
		DatabaseDSN: fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			envVars["POSTGRES_HOST"], envVars["POSTGRES_USER"], envVars["POSTGRES_PASSWORD"], envVars["POSTGRES_DB"]),
		GrafanaURL: envVars["GRAFANA_URL"],
	}

	return cfg
}

func checkEnvVars(envVars map[string]string) string {
	for key, value := range envVars {
		if value == "" {
			return key
		}
	}
	return ""
}
