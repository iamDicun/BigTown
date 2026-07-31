package entity

import "time"

type Placement struct {
	ID          string    `json:"id"`
	MapID       string    `json:"map_id"`
	CharacterID string    `json:"character_id"`
	ItemID      string    `json:"item_id"`
	X           int       `json:"x"`
	Y           int       `json:"y"`
	CreatedAt   time.Time `json:"created_at"`
}

type DecorationItem struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	AssetKey     string `json:"asset_key"`
	Price        int    `json:"price"`
	MetadataJSON string `json:"metadata_json"`
}
