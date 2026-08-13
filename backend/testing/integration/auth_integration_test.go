package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAuthIntegration_RegisterAndLogin(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// 1. Register User
	regPayload := map[string]string{
		"full_name": "Integration User",
		"email":     "int_user@test.com",
		"password":  "password123",
	}
	resReg := DoRequest(testApp.Router(), "POST", "/api/auth/register", regPayload, "")
	if resReg.Code != http.StatusCreated && resReg.Code != http.StatusOK {
		t.Fatalf("expected register status 200/201, got %d, body: %s", resReg.Code, resReg.Body.String())
	}

	// Verify User in DB
	var dbUserID, dbEmail string
	err := db.QueryRow("SELECT id, email FROM app_user WHERE email = $1", "int_user@test.com").Scan(&dbUserID, &dbEmail)
	if err != nil {
		t.Fatalf("user not found in DB: %v", err)
	}
	if dbEmail != "int_user@test.com" {
		t.Errorf("expected email int_user@test.com, got %s", dbEmail)
	}

	// 2. Login User
	loginPayload := map[string]string{
		"email":    "int_user@test.com",
		"password": "password123",
	}
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", loginPayload, "")
	if resLogin.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d, body: %s", resLogin.Code, resLogin.Body.String())
	}

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resLogin.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	if loginResp.Data.AccessToken == "" {
		t.Fatal("expected access_token in response body")
	}

	// Verify Refresh Token Cookie and DB
	cookies := resLogin.Result().Cookies()
	var hasRefreshCookie bool
	for _, c := range cookies {
		if c.Name == "refresh_token" && c.Value != "" {
			hasRefreshCookie = true
			break
		}
	}
	if !hasRefreshCookie {
		t.Error("expected refresh_token cookie set in login response")
	}

	var rtCount int
	err = db.QueryRow("SELECT COUNT(*) FROM refresh_token WHERE user_id = $1 AND revoked_at IS NULL", dbUserID).Scan(&rtCount)
	if err != nil || rtCount == 0 {
		t.Errorf("expected active refresh token in DB, count: %d, err: %v", rtCount, err)
	}
}

func TestAuthIntegration_LoginWrongPassword(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register user
	regPayload := map[string]string{
		"full_name": "Test User",
		"email":     "user@test.com",
		"password":  "correct_password",
	}
	DoRequest(testApp.Router(), "POST", "/api/auth/register", regPayload, "")

	// Login with wrong password
	loginPayload := map[string]string{
		"email":    "user@test.com",
		"password": "wrong_password",
	}
	res := DoRequest(testApp.Router(), "POST", "/api/auth/login", loginPayload, "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", res.Code)
	}
}

func TestAuthIntegration_ProtectedEndpoint(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register + Login
	regPayload := map[string]string{"full_name": "Auth User", "email": "auth@test.com", "password": "password123"}
	DoRequest(testApp.Router(), "POST", "/api/auth/register", regPayload, "")

	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "auth@test.com", "password": "password123"}, "")
	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)

	// Unauthenticated call -> 401
	resNoAuth := DoRequest(testApp.Router(), "GET", "/api/characters/me", nil, "")
	if resNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", resNoAuth.Code)
	}

	// Authenticated call -> 404 (because no character created yet, but auth passed)
	resAuth := DoRequest(testApp.Router(), "GET", "/api/characters/me", nil, loginResp.Data.AccessToken)
	if resAuth.Code != http.StatusNotFound {
		t.Errorf("expected 404 (auth passed, character not created) for authenticated request, got %d, body: %s", resAuth.Code, resAuth.Body.String())
	}
}

func TestAuthIntegration_Logout(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// Register + Login
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Logout User", "email": "logout@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "logout@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)

	var refreshTokenCookie *http.Cookie
	for _, c := range resLogin.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshTokenCookie = c
			break
		}
	}

	// Logout
	resLogout := DoRequest(testApp.Router(), "POST", "/api/auth/logout", nil, loginResp.Data.AccessToken, refreshTokenCookie)
	if resLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 logout status, got %d, body: %s", resLogout.Code, resLogout.Body.String())
	}

	// Calling protected endpoint with blacklisted access token should now be 401
	resAfterLogout := DoRequest(testApp.Router(), "GET", "/api/characters/me", nil, loginResp.Data.AccessToken)
	if resAfterLogout.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for blacklisted access token after logout, got %d", resAfterLogout.Code)
	}
}
