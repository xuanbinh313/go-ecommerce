package security

import (
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/user/domain"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type InviteTokenManager struct{ secret []byte }
type inviteClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func NewInviteTokenManager(secret string) InviteTokenManager {
	return InviteTokenManager{secret: []byte(secret)}
}
func (m InviteTokenManager) Issue(payload domain.InviteClaims, duration time.Duration) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(duration)
	claims := inviteClaims{ID: payload.ID, Email: payload.Email, RegisteredClaims: jwt.RegisteredClaims{ID: fmt.Sprintf("%d", now.UnixNano()), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expires, err
}
func (m InviteTokenManager) Parse(token string) (domain.InviteClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &inviteClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return domain.InviteClaims{}, err
	}
	claims, ok := parsed.Claims.(*inviteClaims)
	if !ok || !parsed.Valid {
		return domain.InviteClaims{}, fmt.Errorf("invalid invite token")
	}
	return domain.InviteClaims{ID: claims.ID, Email: claims.Email}, nil
}

var _ domain.InviteTokenManager = InviteTokenManager{}
