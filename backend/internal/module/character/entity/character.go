package entity

type Character struct {
	ID           string
	UserID       string
	Name         string
	MapID        *string
	BaseAssetKey string
	Coins        int
	Score        int
	LastX        *int
	LastY        *int
}
