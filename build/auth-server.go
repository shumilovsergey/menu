package main

// auth-server.go — server-to-server calls to auth-center.
// Handles actions that originate from this app's backend, not from the browser.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// delegateCode obtains a one-time cross-app login code from auth-center,
// letting the current user open another app without re-authenticating.
// Usage: redirect the user to https://other-app/?code=<returned code>
func delegateCode(uid int64) (string, error) {
	user, err := getUserByID(uid)
	if err != nil || user == nil {
		return "", fmt.Errorf("user not found")
	}
	body, _ := json.Marshal(map[string]string{
		"user_id":   user.AuthID,
		"app_token": appToken,
	})
	resp, err := httpClient.Post(authInternal+"/delegate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("delegate: %w", err)
	}
	defer resp.Body.Close()
	var data struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&data) //nolint:errcheck
	if data.Code == "" {
		return "", fmt.Errorf("delegate failed: %s", data.Error)
	}
	return data.Code, nil
}

func handleOpenFoodScanner(w http.ResponseWriter, r *http.Request) {
	uid := sessionUserID(r)
	if uid == 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	code, err := delegateCode(uid)
	if err != nil {
		log.Printf("open-food-scanner uid=%d error=%v", uid, err)
		http.Error(w, "could not open app", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "https://food-scaner.sh-development.ru/?code="+code, http.StatusFound)
}
