package ports

import (
	"context"
	"doodhee-backend/internal/domain"
)

type Store interface {
	Get(context.Context, string, string, bool) (domain.Record, error)
	List(context.Context, string, domain.Record) ([]domain.Record, error)
	Save(context.Context, string, domain.Record) (domain.Record, error)
	Delete(context.Context, string, string) error
	Tx(context.Context, func(Store) error) error
	Lock(context.Context, string) error
	Ping(context.Context) error
}
type Security interface {
	ID() string
	Hash(string) string
	Random() string
	Token(domain.Actor) (string, error)
	Verify(string, string) (domain.Actor, error)
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
	OTP() string
	ValidateTOTP(string, string) bool
}
type OTPSender interface {
	Send(context.Context, string, string) error
}
