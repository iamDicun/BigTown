package entity

// MapInfo là metadata của 1 map (bảng `maps`). Character dùng để resolve map_id lúc tạo/đồng bộ;
// realtime bootstrap dùng để trả tilemap/tileset/spawn point thật cho frontend (không hardcode).
type MapInfo struct {
	ID                string
	Code              string
	Name              string
	TilemapAssetKey   string
	TilesetAssetKey   string
	CollisionAssetKey *string
	SpawnX            int
	SpawnY            int
	Width             int
	Height            int
}
