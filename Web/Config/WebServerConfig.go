package Config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	ServerConfig "nebiros/Server/Config"
	"os"
)

// web configuration
type WebConfig struct {
	Port         int                       `json:"port"`
	UseTls       bool                      `json:"use_tls"`
	Tls          TlsConfig                 `json:"tls_config,omitempty"`
	ClientConfig string                    `json:"client_config_path"`
	OraCfg       ServerConfig.OracleConfig `json:"oracle"`
	OauthCfg     OauthConfig               `json:"oauth"`
}

type TlsConfig struct {
	CertFilepath string `json:"cert_file_path,omitempty"`
	KeyFilepath  string `json:"key_file_path,omitempty"`
	PemFilePath  string `json:"pem_file_path,omitempty"`
}

type OauthConfig struct {
	ClientID string   `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Tenant   string   `json:"tenant"`
	Scopes   []string `json:"scopes"`
	Redirect string   `json:"redirectUrl"`
}

func LoadConfiguration(path string, rootDirectory string) *WebConfig {
	webconfig := &WebConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to load web server configuration: %s", err.Error())
	}

	if !json.Valid(data) {
		log.Fatalf("invalid json: %s", err.Error())
	}

	err = json.Unmarshal(data, webconfig)
	if err != nil {
		log.Fatalf("failed to unmarshal configuration: %s", err.Error())
	}

	webconfig.ClientConfig = fmt.Sprintf("%s/%s", rootDirectory, webconfig.ClientConfig)

	if webconfig.Tls.PemFilePath != "" {
		webconfig.Tls.PemFilePath = fmt.Sprintf("%s/%s", rootDirectory, webconfig.Tls.PemFilePath)
	}

	if webconfig.Tls.KeyFilepath != "" {
		webconfig.Tls.KeyFilepath = fmt.Sprintf("%s/%s", rootDirectory, webconfig.Tls.KeyFilepath)
	}

	if webconfig.Tls.CertFilepath != "" {
		webconfig.Tls.CertFilepath = fmt.Sprintf("%s/%s", rootDirectory, webconfig.Tls.CertFilepath)
	}

	if ok, err := webconfig.CheckTlsConfig(); !ok {
		webconfig.UseTls = false
		log.Printf("[WARN] TLS config is missing information. Setting TLS to false. - %s\n", err.Error())
	}

	return webconfig
}

func (w *WebConfig) CheckTlsConfig() (bool, error) {
	if !w.UseTls {
		return true, nil
	}

	if len(w.Tls.CertFilepath) < 1 {
		return false, fmt.Errorf("tls cert file path not set")
	}

	if fi, err := os.Stat(w.Tls.CertFilepath); err != nil || fi.Size() == 0 {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("tls cert file not found: %s", err.Error())
		} else {
			return false, fmt.Errorf("tls cert file is empty: %s", err.Error())
		}
	}

	if len(w.Tls.KeyFilepath) < 1 {
		return false, fmt.Errorf("tls key file path not set")
	}

	if fi, err := os.Stat(w.Tls.KeyFilepath); err != nil || fi.Size() == 0 {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("tls key file not found: %s", err.Error())
		} else {
			return false, fmt.Errorf("tls key file is empty: %s", err.Error())
		}
	}

	return true, nil
}
