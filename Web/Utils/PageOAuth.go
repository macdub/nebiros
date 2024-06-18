package Utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func (p *PageHandler) CheckToken(bearer string) (string, error) {
	idToken := strings.TrimSpace(strings.Replace(bearer, "Bearer", "", 1))

	var username string
	if u, found := p.Cache.Cache.Get(idToken); found {
		username = u.(string)
	}

	if username == "" {
		return username, fmt.Errorf("access denied: %d", http.StatusUnauthorized)
	}

	return username, nil
}

func (p *PageHandler) OauthMSLogin(w http.ResponseWriter, r *http.Request) {
	state := generateOAuthState(w)
	u := p.OAuthCfg.AuthCodeURL(state)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func (p *PageHandler) OauthMSCallback(w http.ResponseWriter, r *http.Request) {
	oauthstate, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthstate.Value {
		log.Printf("invalid oauthstate: %s expected: %s\n", r.FormValue("state"), oauthstate.Value)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	cookie, err := p.setTokenCookie(w, r.FormValue("code"))
	if err != nil {
		log.Printf("error setting token cookie: %s\n", err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	data, err := getUserDataFromMicrosoft(cookie.Value)
	if err != nil {
		log.Printf("error getting user data: %s\n", err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	userInfo := &UserInfo{}
	json.Unmarshal(data, userInfo)
	userInfo.ExpiresAt = cookie.Expires

	p.setUserCookie(w, userInfo.UserPrincipalName, cookie.Expires)

	err = p.Cache.Cache.Add(cookie.Value, userInfo.UserPrincipalName, 24*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	nr, err := http.NewRequest("GET", r.Header.Get("Referer"), nil)
	http.Redirect(w, nr, nr.Header.Get("Referer"), http.StatusTemporaryRedirect)
}

func (p *PageHandler) setTokenCookie(w http.ResponseWriter, code string) (*http.Cookie, error) {
	token, err := p.OAuthCfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.AccessToken,
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
	}

	http.SetCookie(w, cookie)

	return cookie, nil
}

func (p *PageHandler) setUserCookie(w http.ResponseWriter, username string, expiry time.Time) {
	cookie := &http.Cookie{
		Name:    "user_id",
		Value:   username,
		Expires: expiry,
		Secure:  true,
		Path:    "/",
	}

	http.SetCookie(w, cookie)
}
