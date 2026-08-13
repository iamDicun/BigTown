package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCharacterIntegration_ListOptions(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register & Login
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Opt User", "email": "opt@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "opt@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	res := DoRequest(testApp.Router(), "GET", "/api/characters/options", nil, token)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 status for options, got %d, body: %s", res.Code, res.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse options response: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Error("expected non-empty character options list")
	}
}

func TestCharacterIntegration_CreateAndGetMe(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register & Login
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Char User", "email": "char@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "char@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	// 1. Get Me before creation -> 404 Not Found
	resMeBefore := DoRequest(testApp.Router(), "GET", "/api/characters/me", nil, token)
	if resMeBefore.Code != http.StatusNotFound {
		t.Errorf("expected 404 before creation, got %d", resMeBefore.Code)
	}

	// 2. Create Character
	createPayload := map[string]string{
		"name":           "Hero Character",
		"base_asset_key": "player",
	}
	resCreate := DoRequest(testApp.Router(), "POST", "/api/characters", createPayload, token)
	if resCreate.Code != http.StatusOK && resCreate.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 status for create character, got %d, body: %s", resCreate.Code, resCreate.Body.String())
	}

	// Verify Character in DB
	var dbCharName, dbAssetKey string
	err := db.QueryRow("SELECT name, base_asset_key FROM characters WHERE name = $1", "Hero Character").Scan(&dbCharName, &dbAssetKey)
	if err != nil {
		t.Fatalf("character record not found in DB: %v", err)
	}
	if dbCharName != "Hero Character" || dbAssetKey != "player" {
		t.Errorf("expected Hero Character / player, got %s / %s", dbCharName, dbAssetKey)
	}

	// 3. Get Me after creation -> 200 OK
	resMeAfter := DoRequest(testApp.Router(), "GET", "/api/characters/me", nil, token)
	if resMeAfter.Code != http.StatusOK {
		t.Fatalf("expected 200 status for get me after creation, got %d, body: %s", resMeAfter.Code, resMeAfter.Body.String())
	}

	var meResp struct {
		Data struct {
			Name         string `json:"name"`
			BaseAssetKey string `json:"base_asset_key"`
		} `json:"data"`
	}
	json.Unmarshal(resMeAfter.Body.Bytes(), &meResp)
	if meResp.Data.Name != "Hero Character" {
		t.Errorf("expected Hero Character, got %s", meResp.Data.Name)
	}
}

func TestCharacterIntegration_DuplicateCharacter(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register & Login
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Dup User", "email": "dupchar@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "dupchar@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	// Create First Character
	DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "First Char", "base_asset_key": "player"}, token)

	// Try creating Second Character -> Should fail
	resDup := DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "Second Char", "base_asset_key": "player"}, token)
	if resDup.Code == http.StatusOK || resDup.Code == http.StatusCreated {
		t.Errorf("expected error when creating duplicate character for same user, got status %d", resDup.Code)
	}
}
