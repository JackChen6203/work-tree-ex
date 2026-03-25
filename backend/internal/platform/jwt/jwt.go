package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Header is the JWT header.
type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims is the JWT claims payload.
type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
}

// Generate creates a signed JWT token for the given user.
func Generate(userID, email string, ttl time.Duration) (string, error) {
	secret := GetSecret()
	now := time.Now()

	header := Header{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	claims := Claims{
		Sub:   userID,
		Email: email,
		Iat:   now.Unix(),
		Exp:   now.Add(ttl).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signatureInput := headerB64 + "." + claimsB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signatureInput + "." + sig, nil
}

// Validate verifies a JWT token and returns its claims.
func Validate(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	secret := GetSecret()

	// Verify signature.
	signatureInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}

	// Decode header.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrInvalidToken
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return nil, ErrInvalidToken
	}

	// Decode claims.
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.Sub == "" {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > claims.Exp {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

// GetSecret returns the JWT signing secret from environment.
func GetSecret() string {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "dev-jwt-secret-do-not-use-in-production"
	}
	return secret
}

// ErrInvalidToken is returned when token validation fails.
var ErrInvalidToken = invalidTokenError{}

type invalidTokenError struct{}

func (e invalidTokenError) Error() string { return "invalid token" }
