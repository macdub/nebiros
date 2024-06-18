package Config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
)

type ClientConfig struct {
	User string
	Host string    `json:"remote_host"`
	Port string    `json:"remote_port"`
	Tls  TlsConfig `json:"tls_config"`
}

type TlsConfig struct {
	UseTls bool   `json:"use_tls"`
	CaFile string `json:"ca_file,omitempty"`
}

func (c *ClientConfig) Show() string {
	var sb strings.Builder

	sb.WriteString("--- Client Config ---\n")
	sb.WriteString("Host Name: " + c.Host + "\n")
	sb.WriteString("Port: " + c.Port + "\n")
	sb.WriteString(fmt.Sprintf("Use TLS? %t\n", c.Tls.UseTls))
	sb.WriteString("CA file: " + c.Tls.CaFile + "\n")
	sb.WriteString("---------------------\n")

	return sb.String()
}

func (c *ClientConfig) UpdateUserConfig(kvString string) (bool, error) {
	kv := strings.Split(kvString, "=")
	if len(kv) != 2 {
		return false, fmt.Errorf("invalid input KV String: %s", kvString)
	}

	switch kv[0] {
	case "host":
		c.Host = kv[1]
	case "port":
		c.Port = kv[1]
	case "usetls":
		c.Tls.UseTls = kv[1] == "true"
	case "cafile":
		c.Tls.CaFile = kv[1]
	default:
		return false, fmt.Errorf("invalid input command: %s", kv[0])
	}

	return true, nil
}

func (c *ClientConfig) WriteClientConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := home + "/.nebirosrc"
	_, err = os.Stat(path)
	if err != nil {
		return err
	}

	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()

	data, err := json.Marshal(c)
	fh.Write(data)

	return nil
}

func GetClientConfig(args ...string) (*ClientConfig, error) {
	cc := &ClientConfig{}

	var configpath string
	if len(args) > 0 {
		configpath = args[0] + "/.nebirosrc"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		if len(args) > 0 {
			home = args[0]
		}

		configpath = home + "/.nebirosrc"
	}
	if fi, err := os.Stat(configpath); err != nil || fi.Size() == 0 {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No '.nebirosrc' found. Creating the needful")
		} else {
			fmt.Println("Existing '.nebirosrc' file is empty. Let's fill it")
		}

		err = CreateUserConfig(configpath, cc)
		if err != nil {
			return nil, err
		}
		return cc, nil
	}

	data, err := os.ReadFile(configpath)
	if err != nil {
		return nil, err
	}

	if !json.Valid(data) {
		err = fmt.Errorf("input date is invalid json. Filepath: %s/.nebirorc", configpath)
		return nil, err
	}

	err = json.Unmarshal(data, cc)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func CreateUserConfig(filepath string, cc *ClientConfig) error {
	reader := bufio.NewReader(os.Stdin)

	fh, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fh.Close()

	fmt.Print("Enter hostname: ")
	text, _ := reader.ReadString('\n')
	cc.Host = strings.TrimSpace(text)

	fmt.Print("Enter host port: ")
	text, _ = reader.ReadString('\n')
	cc.Port = strings.TrimSpace(text)

	fmt.Print("Do you want to use TLS? [y/n] ")
	text, _ = reader.ReadString('\n')
	cc.Tls.UseTls = strings.TrimSpace(text) == "y"

	// omit this for now
	//fmt.Print("Do you have a CA cert file you wish to use? [y/n] ")
	//text, _ = reader.ReadString('\n')
	//if strings.TrimSpace(text) == "y" {
	//	fmt.Print("CA Cert Filepath: ")
	//	text, _ = reader.ReadString('\n')
	//	cc.Tls.CaFile = strings.TrimSpace(text)
	//}

	u, err := user.Current()
	if err != nil {
		return err
	}
	cc.User = u.Username

	data, err := json.Marshal(cc)
	if err != nil {
		return err
	}

	fh.Write(data)

	return nil
}

func ReadClientConfig() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := home + "/.nebirosrc"
	_, err = os.Stat(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	return data, err
}
