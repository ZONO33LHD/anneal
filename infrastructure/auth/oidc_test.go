package auth

import (
	"context"
	"testing"
)

func TestOIDCValidatorRejectsMissingBearerToken(t *testing.T) {
	validator := NewOIDCValidator("https://example.test", "scheduler@example.test")

	if err := validator.AuthenticateInternalTask(context.Background(), ""); err == nil {
		t.Fatal("expected missing bearer token to fail")
	}
}
