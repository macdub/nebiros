package Utils

import "net/url"

type PageType int

const (
	INDEX PageType = iota
	LOGIN
	LOGOUT
	ENTITLEMENT
	UNAUTHORIZED
)

type AksStatusRecords struct {
	ResourceGroup string `json:"resourceGroup"`
	ClusterName   string `json:"clusterName"`
	Status        string `json:"powerState"`
	Timestamp     string `json:"Timestamp,omitempty"`
	User          string `json:"User,omitempty"`
}

type Page interface {
	Init()
	GetPageType() PageType
	GetTemplates() []string
	GetFields() map[string]string
	SetField(string, string)
	GetField(string) string
}

// Index
type IndexPage struct {
	templates      []string
	Fields         map[string]string
	AksStatusTable []AksStatusRecords
	EntMap         *Entitlement
}

func NewIndexPage(templates []string, entmap *Entitlement) *IndexPage {
	page := &IndexPage{templates: templates, EntMap: entmap}
	page.Init()
	return page
}

func (p *IndexPage) Init() {
	p.Fields = make(map[string]string)
}

func (p *IndexPage) GetPageType() PageType { return INDEX }

func (p *IndexPage) GetTemplates() []string {
	return p.templates
}

func (p *IndexPage) GetFields() map[string]string {
	return p.Fields
}

func (p *IndexPage) SetField(key string, value string) {
	p.Fields[key] = value
}

func (p *IndexPage) GetField(key string) string {
	if val, ok := p.Fields[key]; ok {
		return val
	}

	return ""
}

// Login / Logout
type LoginPage struct {
	templates []string
}

func NewLoginPage(templates []string) *LoginPage {
	page := &LoginPage{templates: templates}
	return page
}

func (p *LoginPage) Init() {}

func (p *LoginPage) GetPageType() PageType { return LOGIN }

func (p *LoginPage) GetTemplates() []string { return p.templates }

func (p *LoginPage) GetFields() map[string]string { return nil }

func (p *LoginPage) SetField(key string, value string) {}

func (p *LoginPage) GetField(key string) string { return "" }

// Authentication
type AuthPage struct{}

func NewAuthPage(templates []string) *AuthPage {
	return &AuthPage{}
}

func (p *AuthPage) Init() {}

func (p *AuthPage) GetPageType() PageType { return LOGIN }

func (p *AuthPage) GetTemplates() []string { return nil }

func (p *AuthPage) GetFields() map[string]string { return nil }

func (p *AuthPage) SetField(key string, value string) {}

func (p *AuthPage) GetField(key string) string { return "" }

// Login
type LogoutPage struct {
	templates []string
}

func NewLogoutPage(templates []string) *LogoutPage {
	page := &LogoutPage{templates: templates}
	return page
}

func (p *LogoutPage) Init() {}

func (p *LogoutPage) GetPageType() PageType { return LOGOUT }

func (p *LogoutPage) GetTemplates() []string { return p.templates }

func (p *LogoutPage) GetFields() map[string]string { return nil }

func (p *LogoutPage) SetField(key string, value string) {}

func (p *LogoutPage) GetField(key string) string { return "" }

// Entitlement
type EntitlementPage struct {
	templates []string
	EntMap    *Entitlement
}

func NewEntitlementPage(templates []string, ent *Entitlement) *EntitlementPage {
	return &EntitlementPage{
		templates: templates,
		EntMap:    ent,
	}
}

func (p *EntitlementPage) Init()                             {}
func (p *EntitlementPage) GetPageType() PageType             { return ENTITLEMENT }
func (p *EntitlementPage) GetTemplates() []string            { return p.templates }
func (p *EntitlementPage) GetFields() map[string]string      { return nil }
func (p *EntitlementPage) GetField(key string) string        { return "" }
func (p *EntitlementPage) SetField(key string, value string) {}
func (p *EntitlementPage) ParseForm(formData url.Values) {
	userAction := formData.Get("userAction")
	userEmail := formData.Get("userEmail")
	var userEnabled bool
	if formData.Get("userEnabled") == "true" {
		userEnabled = true
	} else {
		userEnabled = false
	}

	switch userAction {
	case "add":
		p.EntMap.AddUser(userEmail, userEnabled)
	case "disable":
		p.EntMap.DelUser(userEmail)
	}

	p.EntMap.Save()
}

// Unauthorized
type UnauthorizedPage struct {
	templates []string
}

func NewUnauthorizedPage(templates []string) *UnauthorizedPage {
	return &UnauthorizedPage{
		templates: templates,
	}
}

func (p *UnauthorizedPage) Init()                             {}
func (p *UnauthorizedPage) GetPageType() PageType             { return UNAUTHORIZED }
func (p *UnauthorizedPage) GetTemplates() []string            { return p.templates }
func (p *UnauthorizedPage) GetFields() map[string]string      { return nil }
func (p *UnauthorizedPage) GetField(key string) string        { return "" }
func (p *UnauthorizedPage) SetField(key string, value string) {}
