package entity

type UserIdentity struct {
	ID              string
	UserID          string
	Provider        string
	ExternalSubject string
	TenantID        string
	Email           string
}
