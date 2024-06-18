package Utils

import "net/http"

type WebRequestCommand struct {
	Cluster    string `json:"cluster"`
	Command    string `json:"command"`
	UserCookie http.Cookie
}
