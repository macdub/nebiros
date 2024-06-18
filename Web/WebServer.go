package main

import (
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
	"log"
	"nebiros/Client"
	ClientConfig "nebiros/Client/Config"
	WebConfig "nebiros/Web/Config"
	WebUtils "nebiros/Web/Utils"
	"net/http"
	"os"
	"time"
)

var (
	version       = "undefined"
	showVersion   bool
	rootDirectory string
)

func main() {
	flag.BoolVar(&showVersion, "V", false, "show version")
	flag.StringVar(&rootDirectory, "d", "/opt/Nebiros/web", "working directory")
	flag.Parse()

	if rootDirectory[len(rootDirectory)-1] != '/' {
		rootDirectory += "/"
	}

	if showVersion {
		fmt.Printf("Nebiros Web Server version: %s\n", version)
		os.Exit(0)
	}

	log.Println("Loading configuration")
	cfg := WebConfig.LoadConfiguration(fmt.Sprintf("%s/Config/webconfig.json", rootDirectory), rootDirectory)
	clientCfg, err := ClientConfig.GetClientConfig(cfg.ClientConfig)
	if err != nil {
		log.Fatalf("error loading client config: %s", err)
		return
	}

	log.Println("Loading client")
	client, err := Client.NewNebirosClient(clientCfg)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
		return
	}

	log.Println("Setting up cache")
	cache := WebUtils.NewNebirosCache(24*time.Hour, time.Hour, nil)

	log.Println("Creating OAuth Configuration")
	MicrosoftOAuthConfig := &oauth2.Config{
		RedirectURL:  cfg.OauthCfg.Redirect,
		ClientID:     cfg.OauthCfg.ClientID,
		ClientSecret: cfg.OauthCfg.ClientSecret, // @TODO: pull from vault
		Scopes:       cfg.OauthCfg.Scopes,
		Endpoint:     microsoft.AzureADEndpoint(cfg.OauthCfg.Tenant),
	}

	_, err = client.Connect()
	if err != nil {
		log.Fatalf("failed to connect to Nebiros: %+v\n", err)
	}
	defer client.Connection.Close()

	// setup oracle datasource
	OraDataSource, err := WebUtils.NewOracleSource(cfg.OraCfg)
	if err != nil {
		log.Fatalf("failed to initialize OraDataSource: %s", err)
	}
	defer OraDataSource.Close()

	err = OraDataSource.SetDateTimeFormat("YYYYMMDD HH24:MI:SS")
	if err != nil {
		log.Fatalf("failed to set datetime format: %s", err)
	}

	entCfg := WebUtils.NewEntitlement(fmt.Sprintf("%sConfig/user-data", rootDirectory))
	entCfg.Load()

	// setup routes
	mux := SetupRoutes(rootDirectory, client, cache, OraDataSource, MicrosoftOAuthConfig, entCfg)

	// serve
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Println("Listening on ", addr)
	if cfg.UseTls {
		http.ListenAndServeTLS(addr, cfg.Tls.CertFilepath, cfg.Tls.KeyFilepath, mux)
	} else {
		http.ListenAndServe(addr, mux)
	}
}

func SetupRoutes(rootDiretory string, client *Client.NebirosClient,
	cache *WebUtils.NebirosCache, OraDataSource *WebUtils.OracleSource,
	MicrosoftOAuthConfig *oauth2.Config, entCfg *WebUtils.Entitlement) *http.ServeMux {
	log.Printf("Setting up static file server\n")
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(fmt.Sprintf("%sstatic", rootDiretory)))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// setup routes
	log.Printf("Creating routes\n")
	phi := &WebUtils.PageHandler{
		PageData: WebUtils.NewIndexPage([]string{
			fmt.Sprintf("%stemplates/layout.html", rootDirectory),
			fmt.Sprintf("%stemplates/index.html", rootDirectory),
		}, entCfg),
		Client:    client,
		OraClient: OraDataSource,
		Cache:     cache,
		OAuthCfg:  MicrosoftOAuthConfig,
	}

	phlio := &WebUtils.PageHandler{
		PageData: WebUtils.NewLoginPage([]string{
			fmt.Sprintf("%stemplates/layout.html", rootDirectory),
			fmt.Sprintf("%stemplates/login.html", rootDirectory),
		}),
		Cache:    cache,
		OAuthCfg: MicrosoftOAuthConfig,
	}

	phlo := &WebUtils.PageHandler{
		PageData: WebUtils.NewLogoutPage([]string{
			fmt.Sprintf("%stemplates/layout.html", rootDirectory),
		}),
		Cache:    cache,
		OAuthCfg: MicrosoftOAuthConfig,
	}

	phMsAuth := &WebUtils.PageHandler{
		PageData: WebUtils.NewAuthPage([]string{}),
		Cache:    cache,
		OAuthCfg: MicrosoftOAuthConfig,
	}

	phUsers := &WebUtils.PageHandler{
		PageData: WebUtils.NewEntitlementPage(
			[]string{
				fmt.Sprintf("%stemplates/layout.html", rootDirectory),
				fmt.Sprintf("%stemplates/entitlement.html", rootDirectory),
			},
			entCfg),
		Cache:    cache,
		OAuthCfg: MicrosoftOAuthConfig,
	}

	phAuthorized := &WebUtils.PageHandler{
		PageData: WebUtils.NewUnauthorizedPage(
			[]string{
				fmt.Sprintf("%stemplates/layout.html", rootDirectory),
				fmt.Sprintf("%stemplates/unauthorized.html", rootDirectory),
			}),
	}

	// page handlers
	mux.HandleFunc("/", phi.Handler)
	mux.HandleFunc("/users", phUsers.Handler)
	mux.HandleFunc("/login", phlio.Handler)
	mux.HandleFunc("/logout", phlo.Handler)
	mux.HandleFunc("/unauthorized", phAuthorized.Handler)
	mux.HandleFunc("/auth/ms/login", phMsAuth.OauthMSLogin)
	mux.HandleFunc("/auth/ms/callback", phMsAuth.OauthMSCallback)

	rEntitled := WebUtils.NewRestAPI(entCfg, cache)

	// rest api handlers
	mux.HandleFunc("/api/v1/isEntitled", rEntitled.IsEntitled)

	return mux
}

/*

/api/v1/isEntitled
*/
