package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestRealtimeIntegration_Bootstrap(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Setup User & Character
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Realtime User", "email": "realtime@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "realtime@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "Explorer", "base_asset_key": "player"}, token)

	// GET Realtime Bootstrap Metadata
	res := DoRequest(testApp.Router(), "GET", "/api/realtime/bootstrap", nil, token)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 status for bootstrap, got %d, body: %s", res.Code, res.Body.String())
	}

	var bootstrapResp struct {
		Data struct {
			MapCode        string `json:"map_code"`
			DefaultRoomID  string `json:"default_room_id"`
			DefaultChannel string `json:"default_channel"`
			WebSocketPath  string `json:"websocket_path"`
			SpawnX         int    `json:"spawn_x"`
			SpawnY         int    `json:"spawn_y"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &bootstrapResp); err != nil {
		t.Fatalf("failed to parse bootstrap response: %v", err)
	}
	if bootstrapResp.Data.MapCode == "" {
		t.Error("expected non-empty map_code")
	}
	if bootstrapResp.Data.DefaultChannel == "" {
		t.Error("expected non-empty default_channel")
	}
}
