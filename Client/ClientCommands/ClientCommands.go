package ClientCommands

import (
	"bytes"
	"flag"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"time"

	// external packages
	"google.golang.org/protobuf/types/known/durationpb"

	// internal packages
	"nebiros"
	"nebiros/Client/Config"
	"nebiros/Commands"
)

/** ShowClientConfig **/
type ShowClientConfig struct {
	fs           *flag.FlagSet
	rawConfig    bool
	parsedConfig *Config.ClientConfig
}

func NewShowClientConfig(cfg *Config.ClientConfig) *ShowClientConfig {
	cmd := &ShowClientConfig{
		fs:           flag.NewFlagSet("showconfig", flag.ContinueOnError),
		parsedConfig: cfg,
	}

	cmd.fs.BoolVar(&cmd.rawConfig, "raw", false, "Display the raw configuration text")

	return cmd
}

func (cmd *ShowClientConfig) Init(args []string) error {
	return cmd.fs.Parse(args)
}

func (cmd *ShowClientConfig) Name() string {
	return cmd.fs.Name()
}

func (cmd *ShowClientConfig) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	start := time.Now().UTC()
	cr := &nebiros.CommandResponse{}

	if cmd.rawConfig {
		data, err := Config.ReadClientConfig()

		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		if len(data) > 0 {
			cr.CmdResult = string(data)
		}
	} else {
		parsedCfg := cmd.parsedConfig.Show()
		cr.CmdResult = parsedCfg
	}

	end := time.Now().UTC()
	cr.EndTime = timestamppb.New(end)
	cr.ExecTime = durationpb.New(end.Sub(start))

	response.CommandResult = *cr
	return response, nil
}

func (cmd *ShowClientConfig) Usage() string {
	buf := new(bytes.Buffer)
	cmd.fs.SetOutput(buf)
	cmd.fs.PrintDefaults()

	return buf.String()
}

/** UpdateClientConfig **/
type UpdateClientConfig struct {
	fs     *flag.FlagSet
	setKV  string
	config *Config.ClientConfig
}

func NewUpdateClientConfig(cfg *Config.ClientConfig) *UpdateClientConfig {
	cmd := &UpdateClientConfig{
		fs:     flag.NewFlagSet("updateconfig", flag.ContinueOnError),
		config: cfg,
	}

	cmd.fs.StringVar(&cmd.setKV, "set", "",
		"Set KV value\nform: 'key=value'\nValid Keys: host,port,usetls,cafile")

	return cmd
}

func (cmd *UpdateClientConfig) Init(args []string) error {
	return cmd.fs.Parse(args)
}

func (cmd *UpdateClientConfig) Name() string {
	return cmd.fs.Name()
}

func (cmd *UpdateClientConfig) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()

	if ok, err := cmd.config.UpdateUserConfig(cmd.setKV); !ok {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	err := cmd.config.WriteClientConfig()
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	cr.CmdResult = fmt.Sprintf("Client Configuration successfully updated\n")

	end := time.Now().UTC()
	cr.EndTime = timestamppb.New(end)
	cr.ExecTime = durationpb.New(end.Sub(start))

	response.CommandResult = *cr
	return response, nil
}

func (cmd *UpdateClientConfig) Usage() string {
	buf := new(bytes.Buffer)
	cmd.fs.SetOutput(buf)
	cmd.fs.PrintDefaults()

	return buf.String()
}

/** ClientHelp **/
type ClientHelp struct {
	fs          *flag.FlagSet
	cmds        []Commands.Command
	remoteUsage []string
	getRemote   bool
}

func NewClientHelp(remoteUsage []string) *ClientHelp {
	ch := &ClientHelp{
		fs:          flag.NewFlagSet("help", flag.ContinueOnError),
		remoteUsage: remoteUsage,
	}

	ch.fs.BoolVar(&ch.getRemote, "remote", false, "Get usage for server side commands")

	return ch
}

func (cmd *ClientHelp) Init(args []string) error {
	return cmd.fs.Parse(args)
}

func (cmd *ClientHelp) Name() string {
	return cmd.fs.Name()
}

func (cmd *ClientHelp) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()
	var sb strings.Builder

	sb.WriteString("Usage for NebirosClient\n")
	sb.WriteString("COMMANDS:\n")
	for _, c := range cmd.cmds {
		sb.WriteString("\n" + c.Name() + "\n")
		sb.WriteString(c.Usage())
	}

	for _, usage := range cmd.remoteUsage {
		sb.WriteString(usage)
	}

	sb.WriteString("help\n")

	cr.CmdResult = sb.String()

	end := time.Now().UTC()
	cr.EndTime = timestamppb.New(end)
	cr.ExecTime = durationpb.New(end.Sub(start))

	response.CommandResult = *cr
	return response, nil
}

func (cmd *ClientHelp) Usage() string { return "" }

func (cmd *ClientHelp) SetAvailableCommands(cmds []Commands.Command) {
	cmd.cmds = append(cmd.cmds, cmds...)
}
