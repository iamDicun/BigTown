package teams

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"backend/internal/module/auth/port"

	"github.com/golang-jwt/jwt/v5"
)

const microsoftJWKSCacheTTL = time.Hour

type MicrosoftTokenVerifier struct {
	clientID string
	tenantID string
	client   *http.Client

	mu          sync.RWMutex
	cachedKeys  map[string]*rsa.PublicKey
	cachedUntil time.Time
}

type microsoftClaims struct {
	OID               string `json:"oid"`
	TID               string `json:"tid"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	UniqueName        string `json:"unique_name"`
	jwt.RegisteredClaims
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewMicrosoftTokenVerifier(clientID string, tenantID string) *MicrosoftTokenVerifier {
	if strings.TrimSpace(tenantID) == "" {
		tenantID = "common"
	}

	return &MicrosoftTokenVerifier{
		clientID: strings.TrimSpace(clientID),
		tenantID: strings.TrimSpace(tenantID),
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (v *MicrosoftTokenVerifier) Verify(ctx context.Context, ssoToken string) (*port.TeamsUserClaims, error) {
	if v.clientID == "" {
		return nil, errors.New("TEAMS_CLIENT_ID is not configured")
	}

	claims := &microsoftClaims{}
	token, err := jwt.ParseWithClaims(ssoToken, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected Teams token signing method")
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("Teams token missing kid")
		}

		return v.publicKey(ctx, kid)
	}, jwt.WithAudience(v.clientID))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid Teams token")
	}

	if claims.OID == "" || claims.TID == "" {
		return nil, errors.New("Teams token missing oid or tid")
	}
	if v.tenantID != "common" && v.tenantID != "organizations" && !strings.EqualFold(claims.TID, v.tenantID) {
		return nil, errors.New("Teams token tenant is not allowed")
	}

	email := firstNonEmpty(claims.Email, claims.PreferredUsername, claims.UniqueName)
	return &port.TeamsUserClaims{
		ExternalSubject: claims.OID,
		TenantID:        claims.TID,
		Email:           email,
		FullName:        claims.Name,
	}, nil
}

func (v *MicrosoftTokenVerifier) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if time.Now().Before(v.cachedUntil) && v.cachedKeys != nil {
		key, ok := v.cachedKeys[kid]
		v.mu.RUnlock()
		if ok {
			return key, nil
		}
	} else {
		v.mu.RUnlock()
	}

	keys, err := v.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	v.cachedKeys = keys
	v.cachedUntil = time.Now().Add(microsoftJWKSCacheTTL)
	key := keys[kid]
	v.mu.Unlock()

	if key == nil {
		return nil, errors.New("Teams token signing key not found")
	}
	return key, nil
}

func (v *MicrosoftTokenVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	url := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", v.tenantID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to fetch Microsoft JWKS: status %d", res.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(res.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Kid == "" || key.N == "" || key.E == "" {
			continue
		}

		publicKey, err := rsaPublicKeyFromJWK(key)
		if err != nil {
			continue
		}
		keys[key.Kid] = publicKey
	}

	return keys, nil
}

func rsaPublicKeyFromJWK(key jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("invalid RSA exponent")
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
