package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"cpp/config"

	"golang.org/x/oauth2"
)

type Key struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	Alg string   `json:"alg"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type KeycloakPublicKeyResponse struct {
	Keys []Key `json:"keys"`
}

func SetupOAuth2Config(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  fmt.Sprintf("%s/callback", cfg.CppHost),
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.KeycloakURL + "/auth/realms/" + cfg.KeycloakRealm + "/protocol/openid-connect/auth",
			TokenURL: cfg.KeycloakURL + "/auth/realms/" + cfg.KeycloakRealm + "/protocol/openid-connect/token",
		},
	}
}

func GetPublicKey(keycloakURL, realm string) (*rsa.PublicKey, error) {
	publicKeyString, err := fetchKeycloakPublicKey(keycloakURL, realm)
	if err != nil {
		return nil, err
	}
	return parseRSAPublicKeyFromPEM(publicKeyString)
}

func fetchKeycloakPublicKey(keycloakURL, realm string) (string, error) {
	url := fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/certs", keycloakURL, realm)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var keyResponse KeycloakPublicKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResponse); err != nil {
		return "", err
	}

	if len(keyResponse.Keys) == 0 {
		return "", fmt.Errorf("no public keys found in the response")
	}

	var key *Key
	for _, k := range keyResponse.Keys {
		if k.Alg == "RS256" {
			key = &k
			break
		}
	}

	if key == nil {
		return "", fmt.Errorf("no RS256 public key found in the response")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return "", err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return "", err
	}
	eInt := big.NewInt(0).SetBytes(eBytes)

	publicKey := &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(nBytes),
		E: int(eInt.Int64()),
	}

	pemBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	pemString := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pemBytes,
	})

	responseString := "-----BEGIN PUBLIC KEY-----\n" + string(pemString) + "\n-----END PUBLIC KEY-----"
	return responseString, nil
}

func parseRSAPublicKeyFromPEM(keyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("unexpected type of public key")
	}
	return rsaPub, nil
}
