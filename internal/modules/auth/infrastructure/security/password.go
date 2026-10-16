package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

type PasswordHasher struct{}

func NewPasswordHasher() PasswordHasher { return PasswordHasher{} }

func (PasswordHasher) Hash(password string) (string, error) {
	const logN, r, p = 15, 8, 1
	key := make([]byte, 96)
	copy(key[:6], "scrypt")
	key[7] = logN
	binary.BigEndian.PutUint32(key[8:12], r)
	binary.BigEndian.PutUint32(key[12:16], p)
	if _, err := rand.Read(key[16:48]); err != nil {
		return "", err
	}
	checksum := sha256.Sum256(key[:48])
	copy(key[48:64], checksum[:16])
	derived, err := scrypt.Key([]byte(password), key[16:48], 1<<logN, r, p, 64)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, derived[32:])
	_, _ = mac.Write(key[:64])
	copy(key[64:], mac.Sum(nil))
	return base64.StdEncoding.EncodeToString(key), nil
}

func (PasswordHasher) Verify(encoded, password string) (bool, bool, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil && len(key) == 96 && string(key[:6]) == "scrypt" {
		logN, r, p := int(key[7]), int(binary.BigEndian.Uint32(key[8:12])), int(binary.BigEndian.Uint32(key[12:16]))
		if logN < 1 || logN > 20 || r < 1 || p < 1 {
			return false, false, errors.New("invalid scrypt parameters")
		}
		checksum := sha256.Sum256(key[:48])
		if subtle.ConstantTimeCompare(checksum[:16], key[48:64]) != 1 {
			return false, false, errors.New("invalid scrypt checksum")
		}
		derived, err := scrypt.Key([]byte(password), key[16:48], 1<<logN, r, p, 64)
		if err != nil {
			return false, false, err
		}
		mac := hmac.New(sha256.New, derived[32:])
		_, _ = mac.Write(key[:64])
		return subtle.ConstantTimeCompare(mac.Sum(nil), key[64:]) == 1, false, nil
	}
	if strings.HasPrefix(encoded, "$2a$") || strings.HasPrefix(encoded, "$2b$") || strings.HasPrefix(encoded, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password)) == nil, true, nil
	}
	return false, false, nil
}
