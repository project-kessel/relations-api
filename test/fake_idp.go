package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/mendsley/gojwk"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
)

// Initialize RSA keys and generate kid
func initKeys() error {
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicKey = &privateKey.PublicKey

	// Generate kid as SHA-256 hash of the public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(pubKeyBytes)
	kid = base64.RawURLEncoding.EncodeToString(hash[:])
	return nil
}

// Encode the key to base64
func safeEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// GenerateJWK generates a JSON Web Key
func GenerateJWK() (map[string]interface{}, error) {
	jwk := &gojwk.Key{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   safeEncode(publicKey.N.Bytes()),
		E:   safeEncode(big.NewInt(int64(publicKey.E)).Bytes()),
	}

	// Marshal the key to JWK format
	jwkJSON, err := json.Marshal(jwk)
	if err != nil {
		return nil, err
	}

	var jwkMap map[string]interface{}
	err = json.Unmarshal(jwkJSON, &jwkMap)
	if err != nil {
		return nil, err
	}

	return jwkMap, nil
}

// GenerateToken generates a JWT token
func GenerateToken() (string, error) {
	claims := jwt.MapClaims{
		"sub":  "1234567890",
		"name": "John Doe",
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(privateKey)
}

func jwksHandler(w http.ResponseWriter, r *http.Request) {
	jwk, err := GenerateJWK()
	if err != nil {
		http.Error(w, "Failed to generate JWK", http.StatusInternalServerError)
		return
	}

	jwks := map[string]interface{}{
		"keys": []interface{}{jwk},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(jwks)
	if err != nil {
		fmt.Print(err)
	}
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := GenerateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"token": "%s"}`, token)
}

func FakeIdpHost() {
	mux := http.NewServeMux()
	if err := initKeys(); err != nil {
		log.Fatalf("Failed to initialize keys: %v", err)
	}

	mux.HandleFunc("/jwks", jwksHandler)
	mux.HandleFunc("/token", tokenHandler)

	err := http.ListenAndServe(":8180", mux)
	if err != nil {
		panic(err)
	}
	fmt.Println("Starting fake IDP server on :8180")
}
