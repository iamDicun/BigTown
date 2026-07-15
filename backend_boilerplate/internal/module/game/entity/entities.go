package entity

type Character struct {
	ID           string
	UserID       string
	Name         string
	MapID        string
	BaseAssetKey string
	Coins        int
	Score        int
	LastX        *int
	LastY        *int
}

type Item struct {
	ID          string
	Code        string
	Name        string
	Type        string
	Slot        string
	AssetKey    string
	Price       int
	AttackBonus int
	HPBonus     int
}

type NPCType struct {
	ID          string
	Code        string
	Name        string
	AssetKey    string
	MaxHP       int
	Attack      int
	RewardScore int
	RewardCoin  int
	RespawnMS   int
}

type MapNPCSpawn struct {
	ID        string
	MapID     string
	NPCTypeID string
	SpawnX    int
	SpawnY    int
	RespawnMS *int
}
