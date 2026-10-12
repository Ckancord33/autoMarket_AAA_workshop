package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// HashPassword genera un hash seguro SHA-256 combinado con un salt criptográfico aleatorio (crypto/rand).
// El valor resultante se almacena en el formato estándar "salt:hash" en codificación hexadecimal.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:]), nil
}

// CheckPassword valida si una contraseña en texto plano coincide con el hash "salt:hash" almacenado.
// Utiliza subtle.ConstantTimeCompare para mitigar ataques de temporización (timing attacks).
func CheckPassword(password, storedHash string) bool {
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actualHash := sha256.Sum256(append(salt, []byte(password)...))
	return subtle.ConstantTimeCompare(actualHash[:], expectedHash) == 1
}
