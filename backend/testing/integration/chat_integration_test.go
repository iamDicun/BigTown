package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestChatIntegration_SendMessageAndList(t *testing.T) {
	db := GetTestDB(t)
	TruncateTables(t, db)
	testApp := NewTestApp(db)

	// 1. Setup User & Character
	DoRequest(testApp.Router(), "POST", "/api/auth/register", map[string]string{"full_name": "Chatter", "email": "chat@test.com", "password": "password123"}, "")
	resLogin := DoRequest(testApp.Router(), "POST", "/api/auth/login", map[string]string{"email": "chat@test.com", "password": "password123"}, "")

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.AccessToken

	DoRequest(testApp.Router(), "POST", "/api/characters", map[string]string{"name": "Chat Hero", "base_asset_key": "player"}, token)

	// 2. Send Chat Message HTTP POST
	msgPayload := map[string]string{
		"message": "Hello from Integration Test!",
	}
	resSend := DoRequest(testApp.Router(), "POST", "/api/rooms/village_adventure/chat/messages", msgPayload, token)
	if resSend.Code != http.StatusOK && resSend.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 status for send message, got %d, body: %s", resSend.Code, resSend.Body.String())
	}

	// Wait briefly for background async worker to insert message into DB
	time.Sleep(200 * time.Millisecond)

	// 3. Verify Message Row in PostgreSQL Table `chat_messages`
	var dbMessage, dbRoomID string
	err := db.QueryRow("SELECT message, room_id FROM chat_messages WHERE message = $1", "Hello from Integration Test!").Scan(&dbMessage, &dbRoomID)
	if err != nil {
		t.Fatalf("chat message record not found in DB: %v", err)
	}
	if dbRoomID != "village_adventure" {
		t.Errorf("expected room_id village_adventure, got %s", dbRoomID)
	}

	// 4. Query Chat History via HTTP GET
	resList := DoRequest(testApp.Router(), "GET", "/api/rooms/village_adventure/chat/messages", nil, token)
	if resList.Code != http.StatusOK {
		t.Fatalf("expected 200 status for get messages, got %d, body: %s", resList.Code, resList.Body.String())
	}

	var listResp struct {
		Data []struct {
			Message       string `json:"message"`
			CharacterName string `json:"character_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to unmarshal chat history response: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least 1 message in history")
	}
	if listResp.Data[0].Message != "Hello from Integration Test!" {
		t.Errorf("expected 'Hello from Integration Test!', got %s", listResp.Data[0].Message)
	}
}
