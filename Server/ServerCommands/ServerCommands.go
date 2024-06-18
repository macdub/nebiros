package ServerCommands

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"os"
	"strings"
	"time"

	// external packages
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"google.golang.org/protobuf/types/known/durationpb"

	// internal packages
	"nebiros"
	"nebiros/Commands"
	"nebiros/Utils"
)

// Status Command
func NewAksStatusCommand(entries *Utils.AksEntryList) *AksStatusCommand {
	sc := &AksStatusCommand{
		fs:      flag.NewFlagSet("aks-status", flag.ContinueOnError),
		entries: entries,
	}

	sc.fs.StringVar(&sc.clusterName, "cluster", "ALL",
		"Cluster name to query. Omit to get the status of all in the configuration")

	return sc
}

type AksStatusCommand struct {
	fs          *flag.FlagSet
	clusterName string
	entries     *Utils.AksEntryList
}

func (cmd *AksStatusCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *AksStatusCommand) Init(args []string) error {
	cmd.clusterName = "ALL"
	return cmd.fs.Parse(args)
}

func (cmd *AksStatusCommand) Run() (*Commands.Response, error) {
	cr := &nebiros.CommandResponse{}
	response := &Commands.Response{Command: cmd}
	start := time.Now().UTC()
	cr.StartTime = timestamppb.New(start)

	azauth, err := azidentity.NewEnvironmentCredential(nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	ctx := context.Background()

	client, err := armcontainerservice.NewManagedClustersClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), azauth, nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	var statuses []string
	for _, entry := range cmd.entries.Entries {
		if len(entry.ResourceGroup) == 0 || len(entry.ClusterName) == 0 {
			continue
		}

		if cmd.clusterName != "ALL" && cmd.clusterName != entry.ClusterName {
			continue
		}

		res, err := client.Get(ctx, entry.ResourceGroup, entry.ClusterName, nil)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		entry.ClusterResponse = res
		entry.PowerState = string(*res.Properties.PowerState.Code)

		// @TODO: using provisioning state would be better than power state
		// entry.ProvisioningState = res.Properties.ProvisioningState

		statuses = append(statuses,
			fmt.Sprintf(`{"name": "%s", "resourceGroup": "%s", "clusterName": "%s", "powerState": "%s"}`,
				entry.Name, entry.ResourceGroup, entry.ClusterName, entry.PowerState))
	}

	cr.CmdResult = fmt.Sprintf("[%s]", strings.Join(statuses, ","))
	end := time.Now().UTC()

	cr.ExecTime = durationpb.New(end.Sub(start))
	cr.EndTime = timestamppb.New(end)

	response.CommandResult = *cr
	return response, nil
}

func (cmd *AksStatusCommand) Usage() string {
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s\n", cmd.Name()))
	cmd.fs.SetOutput(buf)
	cmd.fs.PrintDefaults()

	return buf.String()
}

// List Config Command
func NewListAksConfigCommand(entries *Utils.AksEntryList) *ListAksConfigCommand {
	lcc := &ListAksConfigCommand{
		entries: entries,
	}
	return lcc
}

type ListAksConfigCommand struct {
	entries *Utils.AksEntryList
}

func (cmd *ListAksConfigCommand) Name() string { return "aks-config" }

func (cmd *ListAksConfigCommand) Init(args []string) error {
	return nil
}

func (cmd *ListAksConfigCommand) Run() (*Commands.Response, error) {
	start := time.Now().UTC()
	cr := &nebiros.CommandResponse{
		CmdResult: cmd.entries.PrintConfigTable(),
	}
	end := time.Now().UTC()

	cr.ExecTime = durationpb.New(end.Sub(start))
	cr.StartTime = timestamppb.New(start)
	cr.EndTime = timestamppb.New(end)
	return &Commands.Response{Command: cmd, CommandResult: *cr}, nil
}

func (cmd *ListAksConfigCommand) Usage() string {
	return fmt.Sprintf("\n%s\n", cmd.Name())
}

// Start/Stop may be good to have these as stream executes
// Start Command
func NewAksStartCommand(entries *Utils.AksEntryList) *AksStartCommand {
	sc := &AksStartCommand{
		fs:      flag.NewFlagSet("aks-start", flag.ContinueOnError),
		entries: entries,
	}

	sc.fs.StringVar(&sc.clusterName, "cluster", "", "Cluster to start")

	return sc
}

type AksStartCommand struct {
	fs          *flag.FlagSet
	clusterName string
	entries     *Utils.AksEntryList
}

func (cmd *AksStartCommand) Name() string { return cmd.fs.Name() }

func (cmd *AksStartCommand) Init(args []string) error {
	return cmd.fs.Parse(args)
}

func (cmd *AksStartCommand) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()
	cr.StartTime = timestamppb.New(start)
	azauth, err := azidentity.NewEnvironmentCredential(nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	ctx := context.Background()

	client, err := armcontainerservice.NewManagedClustersClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), azauth, nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	// find the cluster configuration
	for _, entry := range cmd.entries.Entries {
		if entry.ResourceGroup == "" || entry.ClusterName == "" {
			continue
		}

		if cmd.clusterName != entry.ClusterName {
			continue
		}

		if entry.PowerState != "Stopped" {
			stateError := fmt.Errorf("cluster %s is already running", entry.ClusterName)
			Commands.HandleCommandError(cr, stateError, start)
			response.CommandResult = *cr
			return response, stateError
		}

		fmt.Printf("Starting %s ...", entry.ClusterName)
		poller, err := client.BeginStart(ctx, entry.ResourceGroup, entry.ClusterName, nil)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		fmt.Printf(" complete\n")
		cr.CmdResult = fmt.Sprintf("Started %s successfully\n", entry.ClusterName)
	}

	end := time.Now().UTC()
	cr.ExecTime = durationpb.New(end.Sub(start))
	cr.EndTime = timestamppb.New(end)

	response.CommandResult = *cr
	return response, err
}

func (cmd *AksStartCommand) Usage() string {
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s\n", cmd.Name()))
	cmd.fs.SetOutput(buf)
	cmd.fs.PrintDefaults()

	return buf.String()
}

func (cmd *AksStartCommand) GetClusterName() string { return cmd.clusterName }

// Stop Command
func NewAksStopCommand(entries *Utils.AksEntryList) *AksStopCommand {
	sc := &AksStopCommand{
		fs:      flag.NewFlagSet("aks-stop", flag.ContinueOnError),
		entries: entries,
	}

	sc.fs.StringVar(&sc.clusterName, "cluster", "", "Cluster to stop")

	return sc
}

type AksStopCommand struct {
	fs          *flag.FlagSet
	clusterName string
	entries     *Utils.AksEntryList
}

func (cmd *AksStopCommand) Name() string { return cmd.fs.Name() }

func (cmd *AksStopCommand) Init(args []string) error {
	return cmd.fs.Parse(args)
}

func (cmd *AksStopCommand) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()
	cr.StartTime = timestamppb.New(start)
	azauth, err := azidentity.NewEnvironmentCredential(nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	ctx := context.Background()

	client, err := armcontainerservice.NewManagedClustersClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), azauth, nil)
	if err != nil {
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	for _, entry := range cmd.entries.Entries {
		if len(entry.ResourceGroup) == 0 || len(entry.ClusterName) == 0 {
			continue
		}

		// find the cluster configuration
		if cmd.clusterName != entry.ClusterName {
			continue
		}

		if entry.PowerState != "Running" {
			stateError := fmt.Errorf("cluster %s is already stopped", entry.ClusterName)
			Commands.HandleCommandError(cr, stateError, start)
			response.CommandResult = *cr
			return response, stateError
		}

		fmt.Printf("Stopping %s ...", entry.ClusterName)
		poller, err := client.BeginStop(ctx, entry.ResourceGroup, entry.ClusterName, nil)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		fmt.Printf(" complete\n")
		cr.CmdResult = fmt.Sprintf("Stopped %s successfully\n", entry.ClusterName)
	}

	end := time.Now().UTC()
	cr.ExecTime = durationpb.New(end.Sub(start))
	cr.EndTime = timestamppb.New(end)

	response.CommandResult = *cr
	return response, err
}

func (cmd *AksStopCommand) Usage() string {
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s\n", cmd.Name()))
	cmd.fs.SetOutput(buf)
	cmd.fs.PrintDefaults()

	return buf.String()
}

func (cmd *AksStopCommand) GetClusterName() string { return cmd.clusterName }

func NewHelpCommand(entries *Utils.AksEntryList) *HelpCommand {
	hc := &HelpCommand{
		fs: flag.NewFlagSet("help", flag.ContinueOnError),
	}

	return hc
}

type HelpCommand struct {
	fs *flag.FlagSet
}

func (cmd *HelpCommand) Name() string { return cmd.fs.Name() }

func (cmd *HelpCommand) Init(args []string) error { return nil }

func (cmd *HelpCommand) Run() (*Commands.Response, error) {
	response := &Commands.Response{Command: cmd}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()

	cr.StartTime = timestamppb.New(start)
	var sb strings.Builder
	entries := &Utils.AksEntryList{}
	aksStatus := NewAksStatusCommand(entries)
	aksConfig := NewListAksConfigCommand(entries)
	aksStart := NewAksStartCommand(entries)
	aksStop := NewAksStopCommand(entries)

	sb.WriteString(aksStatus.Usage())
	sb.WriteString(aksConfig.Usage())
	sb.WriteString(aksStart.Usage())
	sb.WriteString(aksStop.Usage())

	cr.CmdResult = sb.String()

	end := time.Now().UTC()
	cr.ExecTime = durationpb.New(end.Sub(start))
	cr.EndTime = timestamppb.New(end)

	response.CommandResult = *cr
	return response, nil
}

func (cmd *HelpCommand) Usage() string { return "" }
