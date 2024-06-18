package Utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"html/template"
	"log"
	"nebiros"
	"nebiros/Client"
	"net/http"
)

type PageHandler struct {
	PageData  Page
	Client    *Client.NebirosClient
	OraClient *OracleSource
	Cache     *NebirosCache
	OAuthCfg  *oauth2.Config
}

// Parses a list of templates into a single template pointer
func templateHelper(files []string) (ts *template.Template) {
	return template.Must(template.ParseFiles(files...))
}

func (p *PageHandler) Handler(w http.ResponseWriter, r *http.Request) {
	files := p.PageData.GetTemplates()
	ts := templateHelper(files)
	buffer := &bytes.Buffer{}

	// don't care about authentication for the login page or the authentication pages
	if p.PageData.GetPageType() != LOGIN && p.PageData.GetPageType() != UNAUTHORIZED {
		c, err := r.Cookie("access_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		_, err = p.CheckToken(c.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

	}

	switch p.PageData.GetPageType() {
	case INDEX:
		userCookie, _ := r.Cookie("user_id")

		if r.Method == "POST" {
			var wrc = &WebRequestCommand{}
			json.NewDecoder(r.Body).Decode(wrc)
			wrc.UserCookie = *userCookie

			var clusterState string
			if wrc.Command == "aks-stop" {
				clusterState = "Stopping"
			} else if wrc.Command == "aks-start" {
				clusterState = "Starting"
			}

			go p.executeCommand(wrc, clusterState)

			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusTemporaryRedirect)
		}

		response, err := p.Client.DoCommand(&nebiros.Command{CmdName: "aks-status", UserID: "webserver"})
		handleError(&w, err)

		var statusRecords []AksStatusRecords
		err = json.Unmarshal([]byte(response.CmdResult), &statusRecords)
		handleError(&w, err)

		// combine results from the k8sstatus table with what is present in memory
		// - probably a better way to do this rather than sending a query to oracle
		for i, record := range statusRecords {
			query := fmt.Sprintf(`
SELECT username, cluster_name, cluster_status, MAX(tstp at time zone 'UTC') AS as_of_date 
FROM k8sstatus 
WHERE cluster_name = '%s' 
GROUP BY cluster_name, username, cluster_status 
ORDER BY as_of_date DESC 
FETCH FIRST 1 ROWS ONLY`, record.ClusterName)

			data, e := p.OraClient.ExecQuery(query)
			if e != nil {
				handleError(&w, e)
			}

			if data == nil {
				fmt.Printf("[WARN] no status record found for %s -- status %s\n", record.ClusterName, record.Status)
			}

			if clusterState, ok := p.Cache.Cache.Get(record.ClusterName); ok {
				statusRecords[i].Status = clusterState.(string)
				statusRecords[i].User = userCookie.Value
				statusRecords[i].Timestamp = ""
			} else if data != nil {
				statusRecords[i].User = data[0]["USERNAME"]
				statusRecords[i].Timestamp = data[0]["AS_OF_DATE"]
			}
		}

		p.PageData.(*IndexPage).AksStatusTable = statusRecords

		p.PageData.SetField("user_id", userCookie.Value)
		executeTemplate(&w, ts, buffer, p.PageData)

	case LOGOUT:
		tokCookie, _ := r.Cookie("access_token")

		p.Cache.Cache.Delete(tokCookie.Value)
		tokCookie.Value = ""
		tokCookie.MaxAge = -1
		http.SetCookie(w, tokCookie)

		userCookie, _ := r.Cookie("user_id")
		userCookie.Value = ""
		userCookie.MaxAge = -1
		http.SetCookie(w, userCookie)

		stateCookie, _ := r.Cookie("oauthstate")
		stateCookie.Value = ""
		stateCookie.MaxAge = -1
		http.SetCookie(w, stateCookie)

		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	case ENTITLEMENT:
		if r.Method == "POST" {
			// @TODO: check if user is permitted and redirect to "UNAUTHORIZED" page if not

			r.ParseForm()
			p.PageData.(*EntitlementPage).ParseForm(r.Form)
			http.Redirect(w, r, r.Header.Get("Referer"), 302)
		}

		executeTemplate(&w, ts, buffer, p.PageData)

	default:
		executeTemplate(&w, ts, buffer, p.PageData)
	}

	if r.Method == "GET" {
		// display the page
		buffer.WriteTo(w)
	}
}

func (p *PageHandler) executeCommand(wrc *WebRequestCommand, clusterState string) {
	if state, ok := p.Cache.Cache.Get(wrc.Cluster); ok {
		log.Printf("Cluster '%s' already actioned. State: '%s'\n", wrc.Cluster, state)
		return
	}

	p.Cache.Cache.Add(wrc.Cluster, clusterState, cache.NoExpiration)

	_, _ = p.Client.ExecCommand(&nebiros.Command{
		UserID:  wrc.UserCookie.Value,
		CmdName: wrc.Command,
		CmdOpts: []string{"-cluster", wrc.Cluster},
	})

	p.Cache.Cache.Delete(wrc.Cluster)

}

func executeTemplate(w *http.ResponseWriter, tmpl *template.Template, buffer *bytes.Buffer, data interface{}) {
	err := tmpl.ExecuteTemplate(buffer, "base", data)
	handleError(w, err)
}

func handleError(w *http.ResponseWriter, err error) {
	if err != nil {
		fmt.Println(err.Error())
		http.Error(*w, "Internal Server Error", 500)
	}
}
