package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const secretKey = "mi_clave_super_secreta_del_servidor"

type Payload struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func GenerateJWT(userID, role string) (string, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	expirationTime := time.Now().Add(24 * time.Hour).Unix()

	payloadData := Payload{UserID: userID, Role: role, Exp: expirationTime}
	payloadBytes, _ := json.Marshal(payloadData)

	encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(header))
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	unsignedToken := encodedHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(unsignedToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsignedToken + "." + signature, nil
}

func VerifyJWT(tokenString string) (*Payload, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("token inválido")
	}

	unsignedToken := parts[0] + "." + parts[1]
	signature := parts[2]

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(unsignedToken))
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, errors.New("firma no válida")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload Payload
	json.Unmarshal(payloadBytes, &payload)

	if time.Now().Unix() > payload.Exp {
		return nil, errors.New("el token ha expirado")
	}

	return &payload, nil
}
