package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"doodhee-backend/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"log/slog"
	"math/big"
	"time"
)

type Security struct{ Key []byte }

func (s *Security) ID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func (s *Security) Random() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Security) Hash(v string) string {
	h := hmac.New(sha256.New, s.Key)
	h.Write([]byte(v))
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Security) OTP() string {
	v, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", v.Int64())
}
func (s *Security) Token(a domain.Actor) (string, error) {
	ttl := 15 * time.Minute
	if a.Role == "admin" {
		ttl = 10 * time.Minute
	}
	amr := []string{"otp"}
	if a.Role == "admin" {
		amr = append(amr, "mfa")
	}
	claims := jwt.MapClaims{"iss": "doodhee", "aud": a.Audience, "sub": a.ID, "sid": a.Session, "role": a.Role, "amr": amr, "jti": s.ID(), "iat": time.Now().Unix(), "exp": time.Now().Add(ttl).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Key)
}
func (s *Security) Verify(raw, aud string) (domain.Actor, error) {
	t, e := jwt.Parse(raw, func(t *jwt.Token) (any, error) { return s.Key, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("doodhee"), jwt.WithAudience(aud), jwt.WithExpirationRequired())
	if e != nil {
		return domain.Actor{}, domain.Fail(401, "AUTH_INVALID_TOKEN", "Please sign in again.")
	}
	c := t.Claims.(jwt.MapClaims)
	sub, _ := c["sub"].(string)
	sid, _ := c["sid"].(string)
	role, _ := c["role"].(string)
	return domain.Actor{ID: sub, Session: sid, Role: role, Audience: aud}, nil
}
func (s *Security) Encrypt(v string) (string, error) {
	k := sha256.Sum256(s.Key)
	block, _ := aes.NewCipher(k[:])
	g, _ := cipher.NewGCM(block)
	n := make([]byte, g.NonceSize())
	rand.Read(n)
	return base64.StdEncoding.EncodeToString(g.Seal(n, n, []byte(v), nil)), nil
}
func (s *Security) Decrypt(v string) (string, error) {
	b, e := base64.StdEncoding.DecodeString(v)
	if e != nil {
		return "", e
	}
	k := sha256.Sum256(s.Key)
	block, _ := aes.NewCipher(k[:])
	g, _ := cipher.NewGCM(block)
	if len(b) < g.NonceSize() {
		return "", fmt.Errorf("invalid encrypted secret")
	}
	p, e := g.Open(nil, b[:g.NonceSize()], b[g.NonceSize():], nil)
	return string(p), e
}
func (s *Security) ValidateTOTP(code, secret string) bool { return totp.Validate(code, secret) }

type DevOTP struct{}

func (DevOTP) Send(ctx context.Context, phone, code string) error {
	slog.Warn("LOCAL DEVELOPMENT OTP ONLY", "challenge_code", code)
	return nil
}
