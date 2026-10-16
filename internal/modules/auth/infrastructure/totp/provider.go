package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
)

type Provider struct {
	encryptionKey []byte
	configured    bool
	issuer        string
	digits        int
	period        int
	window        int
}

func New(encryptionKey, issuer string, digits, period, window int) *Provider {
	key := sha256.Sum256([]byte(encryptionKey))
	if issuer == "" {
		issuer = "Medusa"
	}
	if digits == 0 {
		digits = 6
	}
	if period == 0 {
		period = 30
	}
	if window == 0 {
		window = 1
	}
	return &Provider{encryptionKey: key[:], configured: encryptionKey != "", issuer: issuer, digits: digits, period: period, window: window}
}

func (p *Provider) Start(input models.AuthMFAStartInput) (string, string, string, string, error) {
	if !p.configured {
		return "", "", "", "", fmt.Errorf("MFA encryption key is required to use MFA methods")
	}
	if p.digits != 6 && p.digits != 7 && p.digits != 8 {
		return "", "", "", "", fmt.Errorf("TOTP digits must be 6, 7, or 8")
	}
	if p.period < 1 || p.window < 0 {
		return "", "", "", "", fmt.Errorf("TOTP period must be positive and window non-negative")
	}
	issuer := p.issuer
	if input.Issuer != "" {
		issuer = input.Issuer
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", "", "", err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	encrypted, err := p.encrypt(secret)
	if err != nil {
		return "", "", "", "", err
	}
	account := input.AuthIdentityID
	if input.Label != nil && *input.Label != "" {
		account = *input.Label
	}
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(p.digits))
	query.Set("period", strconv.Itoa(p.period))
	return encrypted, secret, issuer, "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?" + query.Encode(), nil
}

func (p *Provider) Verify(encryptedSecret, code string) (bool, error) {
	if !p.configured {
		return false, fmt.Errorf("MFA encryption key is required to use MFA methods")
	}
	secret, err := p.decrypt(encryptedSecret)
	if err != nil {
		return false, err
	}
	code = strings.TrimSpace(code)
	if len(code) != p.digits {
		return false, nil
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false, nil
		}
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimRight(secret, "=")))
	if err != nil {
		return false, fmt.Errorf("Invalid TOTP secret")
	}
	step := time.Now().Unix() / int64(p.period)
	for offset := -p.window; offset <= p.window; offset++ {
		candidate, err := hotp(key, uint64(step+int64(offset)), p.digits)
		if err != nil {
			return false, err
		}
		if hmac.Equal([]byte(candidate), []byte(code)) {
			return true, nil
		}
	}
	return false, nil
}

func hotp(key []byte, counter uint64, digits int) (string, error) {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(msg[:]); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	modulus := uint32(1)
	for i := 0; i < digits; i++ {
		modulus *= 10
	}
	return fmt.Sprintf("%0*d", digits, value%modulus), nil
}

func (p *Provider) encrypt(secret string) (string, error) {
	block, err := aes.NewCipher(p.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(iv); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, iv, []byte(secret), nil)
	tagSize := gcm.Overhead()
	return "v1:" + base64.RawURLEncoding.EncodeToString(iv) + ":" + base64.RawURLEncoding.EncodeToString(ciphertext[len(ciphertext)-tagSize:]) + ":" + base64.RawURLEncoding.EncodeToString(ciphertext[:len(ciphertext)-tagSize]), nil
}

func (p *Provider) decrypt(payload string) (string, error) {
	parts := strings.Split(payload, ":")
	if len(parts) != 4 || parts[0] != "v1" {
		return "", fmt.Errorf("Invalid encrypted MFA secret")
	}
	iv, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	tag, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", err
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(p.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, iv, append(body, tag...), nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

var _ interface {
	Start(models.AuthMFAStartInput) (string, string, string, string, error)
	Verify(string, string) (bool, error)
} = (*Provider)(nil)
