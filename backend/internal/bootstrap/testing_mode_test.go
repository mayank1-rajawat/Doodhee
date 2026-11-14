package bootstrap

import (
	"context"
	"strings"
	"testing"
)

func TestTestingModeConfig(t *testing.T) {
	for _, tt := range []struct {
		env, raw   string
		want, fail bool
	}{
		{"development", "", false, false},
		{"development", "false", false, false},
		{"development", "true", true, false},
		{"test", "true", true, false},
		{"production", "false", false, false},
		{"production", "true", false, true},
		{"staging", "true", false, true},
		{"development", "typo", false, true},
	} {
		t.Run(tt.env+"_"+tt.raw, func(t *testing.T) {
			got, err := testingMode(tt.env, tt.raw)
			if got != tt.want || (err != nil) != tt.fail {
				t.Fatalf("got %t, error=%v; want %t, error=%t", got, err, tt.want, tt.fail)
			}
		})
	}
}

func TestOpenRejectsProductionTestingBeforeDatabaseAccess(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("TESTING_MODE", "true")
	t.Setenv("DATABASE_URL", "invalid-database-address")
	_, pool, err := Open(context.Background())
	if pool != nil || err == nil || !strings.Contains(err.Error(), "TESTING_MODE") {
		t.Fatal("Production startup must refuse testing OTP exposure before opening the database")
	}
}
