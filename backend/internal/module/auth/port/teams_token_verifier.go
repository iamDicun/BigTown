package port

import "context"

type TeamsUserClaims struct {
	ExternalSubject string
	TenantID        string
	Email           string
	FullName        string
}

type TeamsTokenVerifier interface {
	Verify(ctx context.Context, ssoToken string) (*TeamsUserClaims, error)
}
