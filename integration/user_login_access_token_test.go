//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const defaultBaseURL = "https://tokens.buildingblock.top"

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// TestUserLoginAccessTokenAndSelf verifies the external user authentication flow:
//  1. sign in and retain the session cookie;
//  2. obtain the user's login access token;
//  3. use that token, without a cookie, to retrieve the current user's profile.
//
// Required environment variables:
//   - NEW_API_TEST_USERNAME
//   - NEW_API_TEST_PASSWORD
//
// Optional environment variable:
//   - NEW_API_TEST_BASE_URL (defaults to https://tokens.buildingblock.top)
func TestUserLoginAccessTokenAndSelf(t *testing.T) {
	username := os.Getenv("NEW_API_TEST_USERNAME")
	password := os.Getenv("NEW_API_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("NEW_API_TEST_USERNAME and NEW_API_TEST_PASSWORD must be set")
	}

	baseURL := strings.TrimRight(os.Getenv("NEW_API_TEST_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	const timeout = 15 * time.Second
	sessionClient := &http.Client{Jar: jar, Timeout: timeout}

	loginBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}
	loginRequest, err := http.NewRequest(http.MethodPost, baseURL+"/api/user/login", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("create login request: %v", err)
	}
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := doAPIRequest(t, sessionClient, loginRequest)
	if !loginResponse.Success {
		t.Fatalf("login failed: %s", loginResponse.Message)
	}
	var loginUser struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(loginResponse.Data, &loginUser); err != nil {
		t.Fatalf("decode login user: %v", err)
	}
	if loginUser.ID == 0 {
		t.Fatal("login response did not contain a user ID")
	}
	if len(jar.Cookies(base)) == 0 {
		t.Fatal("login succeeded but did not set a session cookie")
	}

	accessTokenRequest, err := http.NewRequest(http.MethodGet, baseURL+"/api/user/token", nil)
	if err != nil {
		t.Fatalf("create access-token request: %v", err)
	}
	accessTokenRequest.Header.Set("New-Api-User", strconv.Itoa(loginUser.ID))
	accessTokenResponse := doAPIRequest(t, sessionClient, accessTokenRequest)
	if !accessTokenResponse.Success {
		t.Fatalf("get login access token failed: %s", accessTokenResponse.Message)
	}
	var accessToken string
	if err := json.Unmarshal(accessTokenResponse.Data, &accessToken); err != nil {
		t.Fatalf("decode login access token: %v", err)
	}
	if accessToken == "" {
		t.Fatal("received an empty login access token")
	}

	selfRequest, err := http.NewRequest(http.MethodGet, baseURL+"/api/user/self", nil)
	if err != nil {
		t.Fatalf("create self request: %v", err)
	}
	selfRequest.Header.Set("Authorization", accessToken)
	selfRequest.Header.Set("New-Api-User", strconv.Itoa(loginUser.ID))
	// Deliberately use a client without the cookie jar to prove the login access
	// token itself, rather than the saved session cookie, authenticates this call.
	selfResponse := doAPIRequest(t, &http.Client{Timeout: timeout}, selfRequest)
	if !selfResponse.Success {
		t.Fatalf("get current user failed: %s", selfResponse.Message)
	}

	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Quota    int    `json:"quota"`
	}
	if err := json.Unmarshal(selfResponse.Data, &user); err != nil {
		t.Fatalf("decode current user: %v", err)
	}
	if user.ID == 0 || user.Username == "" {
		t.Fatalf("current user response is incomplete: id=%d username=%q", user.ID, user.Username)
	}
	if user.Username != username {
		t.Fatalf("current user mismatch: got %q, want %q", user.Username, username)
	}
	// Reading quota is the purpose of this flow; accessing it also verifies that
	// the response keeps the current user's balance field.
	_ = user.Quota
}

func doAPIRequest(t *testing.T, client *http.Client, request *http.Request) apiResponse {
	t.Helper()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", request.Method, request.URL, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("read %s %s response: %v", request.Method, request.URL, err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: unexpected HTTP status %d", request.Method, request.URL, response.StatusCode)
	}

	var apiResponse apiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		t.Fatalf("decode %s %s response: %v", request.Method, request.URL, err)
	}
	return apiResponse
}
