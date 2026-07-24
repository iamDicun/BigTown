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
	TileSize          int
	LayerNames        []string
	AboveLayerName    string
	CollisionLayerName string
}

func (m *MapInfo) MaxPixelX() int { return m.Width*m.TileSize - 1 }
func (m *MapInfo) MaxPixelY() int { return m.Height*m.TileSize - 1 }
