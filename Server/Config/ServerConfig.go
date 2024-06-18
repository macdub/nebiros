package Config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type ServerConfig struct {
	Host       string       `json:"hostname"`
	Port       int          `json:"port"`
	Tls        TlsConfig    `json:"tls"`
	OraCfg     OracleConfig `json:"oracle"`
	WatchSleep int          `json:"watcher_sleep_seconds"`
}

type TlsConfig struct {
	UseTls   bool   `json:"use_tls"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

type OracleConfig struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Sid     string `json:"sid"`
	User    string `json:"username"`
	Pass    string `json:"password"`
}

func LoadServerConfig(root string) (*ServerConfig, error) {
	sc := &ServerConfig{}

	cfgpath := fmt.Sprintf("%sConfig/server_config.json", root)
	if fi, err := os.Stat(cfgpath); err != nil || fi.Size() == 0 {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No 'server_config.json' found.")
			return nil, err
		} else {
			fmt.Println("Existing 'server_config.json' is empty.")
			return nil, fmt.Errorf("existing 'server_config.json' is empty")
		}
	}

	data, err := os.ReadFile(cfgpath)
	if err != nil {
		return nil, err
	}

	if !json.Valid(data) {
		err = fmt.Errorf("input data is invalid json. Filepath: %s", cfgpath)
		return nil, err
	}

	err = json.Unmarshal(data, sc)
	if err != nil {
		return nil, err
	}

	if sc.Tls.CertFile != "" {
		sc.Tls.CertFile = fmt.Sprintf("%s/%s", root, sc.Tls.CertFile)
	}

	if sc.Tls.KeyFile != "" {
		sc.Tls.KeyFile = fmt.Sprintf("%s/%s", root, sc.Tls.KeyFile)
	}

	return sc, nil
}
