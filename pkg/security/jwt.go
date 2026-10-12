package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken     = errors.New("token inválido")
	ErrInvalidSignature = errors.New("firma no válida")
	ErrTokenExpired     = errors.New("el token ha expirado")
)

const SecretKey = "mi_clave_super_secreta_del_servidor"

type JWTPayload struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

// GenerateJWT emite un token JWT firmado mediante HMAC-SHA256 con validez de 24 horas.
func GenerateJWT(userID, role string) (string, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	expirationTime := time.Now().Add(24 * time.Hour).Unix()

	payloadData := JWTPayload{UserID: userID, Role: role, Exp: expirationTime}
	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(header))
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	unsignedToken := encodedHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, []byte(SecretKey))
	mac.Write([]byte(unsignedToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsignedToken + "." + signature, nil
}

// VerifyJWT valida la firma criptográfica y expiración del token JWT, extrayendo los datos de identidad.
func VerifyJWT(tokenString string) (*JWTPayload, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsignedToken := parts[0] + "." + parts[1]
	signature := parts[2]

	mac := hmac.New(sha256.New, []byte(SecretKey))
	mac.Write([]byte(unsignedToken))
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, ErrInvalidSignature
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	if time.Now().Unix() > payload.Exp {
		return nil, ErrTokenExpired
	}

	return &payload, nil
}
