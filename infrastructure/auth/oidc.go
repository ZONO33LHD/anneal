package auth

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/api/idtoken"
)

// OIDCValidator は Cloud Scheduler 等から届く Google OIDC bearer token を検証する。
type OIDCValidator struct {
	Audience string
	Email    string
}

// NewOIDCValidator は OIDC validator を返す。
func NewOIDCValidator(audience, email string) OIDCValidator {
	return OIDCValidator{Audience: audience, Email: email}
}

// AuthenticateInternalTask は Authorization: Bearer <token> を検証する。
func (v OIDCValidator) AuthenticateInternalTask(ctx context.Context, authorization string) error {
	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		return errors.New("missing bearer token")
	}
	payload, err := idtoken.Validate(ctx, token, v.Audience)
	if err != nil {
		return err
	}
	if v.Email == "" {
		return nil
	}
	email, _ := payload.Claims["email"].(string)
	if email != v.Email {
		return errors.New("unexpected oidc email")
	}
	return nil
}
