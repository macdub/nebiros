package Utils

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type RestPayload map[string]string

func NewRestPayload() RestPayload {
	return make(RestPayload)
}

type RestAPI struct {
	EntMap   *Entitlement
	NebCache *NebirosCache
}

func NewRestAPI(ent *Entitlement, cache *NebirosCache) *RestAPI {
	return &RestAPI{
		EntMap:   ent,
		NebCache: cache,
	}
}

func (r *RestAPI) IsEntitled(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	payload := NewRestPayload()

	token, err := req.Cookie("access_token")
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		payload["error"] = err.Error()

		data, _ := json.Marshal(payload)
		res.Write(data)
		return
	}

	if ok := r.checkToken(token.Value); !ok {
		payload["is_entitled"] = "false"
		data, _ := json.Marshal(payload)
		res.Write(data)
		return
	}

	userCookie, err := req.Cookie("user_id")
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		payload["error"] = err.Error()
		payload["is_entitled"] = "false"

		data, _ := json.Marshal(payload)
		res.Write(data)
		return
	}

	res.WriteHeader(http.StatusAccepted)
	payload["is_entitled"] = strconv.FormatBool(r.EntMap.IsEntitled(userCookie.Value))
	data, _ := json.Marshal(payload)
	res.Write(data)
}

func (r *RestAPI) checkToken(bearer string) bool {
	idToken := strings.TrimSpace(strings.Replace(bearer, "Bearer", "", 1))

	var username string
	if u, found := r.NebCache.Cache.Get(idToken); found {
		username = u.(string)
	}

	if username == "" {
		return false
	}

	return true
}
