package main

import (
	"context"
	"doodhee-backend/internal/bootstrap"
	"fmt"
	"os"
)

func main() {
	s, p, e := bootstrap.Open(context.Background())
	if e != nil {
		panic(e)
	}
	defer p.Close()
	phone, secret := os.Getenv("ADMIN_PHONE"), os.Getenv("ADMIN_TOTP_SECRET")
	if phone == "" || len(secret) < 16 {
		panic("ADMIN_PHONE and a base32 ADMIN_TOTP_SECRET are required")
	}
	if e = s.SeedAdmin(context.Background(), phone, "Doodhee Admin", secret); e != nil {
		panic(e)
	}
	fmt.Println("Admin created. Enroll your supplied TOTP secret in your authenticator.")
}
