package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestEditorIntegration_GetEditorData(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Setup User & Character
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Editor User", "email": "editor@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "editor@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "Builder", "base_asset_key": "player"}, token)

	// Get Editor Data
	res := DoRequest(testApp.Router(), "GET", "/api/editor?map_code=village_adventure", nil, token)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 status for editor data, got %d, body: %s", res.Code, res.Body.String())
	}

	var dataResp struct {
		Data struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Coins int `json:"coins"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &dataResp); err != nil {
		t.Fatalf("failed to parse editor data response: %v", err)
	}
	if len(dataResp.Data.Items) == 0 {
		t.Error("expected items loaded from seed")
	}
}

func TestEditorIntegration_PlaceAndDelete(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Setup User & Character
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Placer", "email": "place@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "place@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "Placer Char", "base_asset_key": "player"}, token)

	// 1. Get first decoration item ID from DB
	var itemID string
	err := db.QueryRow("SELECT id FROM items LIMIT 1").Scan(&itemID)
	if err != nil {
		t.Fatalf("no items found in DB seed: %v", err)
	}

	// 2. Place Item (x: 32, y: 32 - multiple of tile_size 16)
	placePayload := map[string]any{
		"item_id":  itemID,
		"map_code": "village_adventure",
		"x":        32,
		"y":        32,
		"rotation": 0,
	}
	resPlace := DoRequest(testApp.Router(), "POST", "/api/editor/place", placePayload, token)
	if resPlace.Code != http.StatusOK && resPlace.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 for place item, got %d, body: %s", resPlace.Code, resPlace.Body.String())
	}

	var placeResp struct {
		Data struct {
			Placement struct {
				ID string `json:"id"`
				X  int    `json:"x"`
				Y  int    `json:"y"`
			} `json:"placement"`
			NewCoins int `json:"new_coins"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resPlace.Body.Bytes(), &placeResp); err != nil {
		t.Fatalf("failed to unmarshal place item response: %v", err)
	}

	placementID := placeResp.Data.Placement.ID
	if placementID == "" {
		t.Fatal("expected non-empty placement ID in response")
	}
	if placeResp.Data.Placement.X != 32 || placeResp.Data.Placement.Y != 32 {
		t.Errorf("expected placement coordinates (32, 32), got (%d, %d)", placeResp.Data.Placement.X, placeResp.Data.Placement.Y)
	}

	// 3. Delete Item Placement
	resDelete := DoRequest(testApp.Router(), "DELETE", "/api/editor/place/"+placementID+"?map_code=village_adventure", nil, token)
	if resDelete.Code != http.StatusOK {
		t.Fatalf("expected 200 status for delete placement, got %d, body: %s", resDelete.Code, resDelete.Body.String())
	}
}
