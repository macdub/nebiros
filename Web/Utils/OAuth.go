package Utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"
)

type UserInfo struct {
	DisplayName       string    `json:"displayName"`
	UserPrincipalName string    `json:"userPrincipalName"`
	Id                string    `json:"id"`
	ExpiresAt         time.Time `json:"expiry"`
}

func generateOAuthState(w http.ResponseWriter) string {
	var expiration = time.Now().Add(24 * time.Hour)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration, Secure: true, Path: "/", HttpOnly: true}
	http.SetCookie(w, &cookie)

	return state
}

func getUserDataFromMicrosoft(token string) ([]byte, error) {
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting user data: %s", err.Error())
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %s", err.Error())
	}

	return contents, nil
}
