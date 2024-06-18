package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"nebiros/Server/Data"
	"net"
	"os"
	"sync"
	"time"

	// external packages
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	// internal packages
	"nebiros"
	"nebiros/Commands"
	"nebiros/Server/Config"
	"nebiros/Server/ServerCommands"
	"nebiros/Utils"
)

var version = "undefined"

type nebirosServer struct {
	nebiros.UnimplementedNebirosServer
	mu        sync.Mutex
	entries   *Utils.AksEntryList
	cmds      []Commands.Command
	validCmds []Commands.ValidCommand
	keeper    *Data.Registrar
	auditChan chan *Data.RecMessage
}

func newServer(entries *Utils.AksEntryList, serverCfg *Config.ServerConfig, ch chan *Data.RecMessage, noexec bool) *nebirosServer {
	registrar, err := Data.NewRegistrar(&serverCfg.OraCfg, serverCfg.WatchSleep, noexec)
	if err != nil {
		log.Fatalf("error setting up database connection: %s", err.Error())
		return nil
	}

	server := &nebirosServer{
		entries: entries,
		cmds: []Commands.Command{
			ServerCommands.NewAksStatusCommand(entries),
			ServerCommands.NewListAksConfigCommand(entries),
			ServerCommands.NewAksStartCommand(entries),
			ServerCommands.NewAksStopCommand(entries),
			ServerCommands.NewHelpCommand(entries),
		},
		validCmds: []Commands.ValidCommand{
			{Name: "aks-status", Func: ServerCommands.NewAksStatusCommand},
			{Name: "aks-config", Func: ServerCommands.NewListAksConfigCommand},
			{Name: "aks-start", Func: ServerCommands.NewAksStartCommand},
			{Name: "aks-stop", Func: ServerCommands.NewAksStopCommand},
			{Name: "help", Func: ServerCommands.NewHelpCommand},
		},
		keeper:    registrar,
		auditChan: ch,
	}

	return server
}

func (ns *nebirosServer) ExecCommand(ctx context.Context, command *nebiros.Command) (*nebiros.CommandResponse, error) {
	response := &nebiros.CommandResponse{}
	if command.UserID == "" {
		return response, fmt.Errorf("no valid user set. aborting command execution. COMMAND: %+v", command)
	}
	result, err := ns.DoCommand(command)

	if err != nil {
		response.CmdError = result.CommandResult.CmdError
	} else {
		response.CmdResult = fmt.Sprintf("%s\n", result.CommandResult.CmdResult)
	}

	response.StartTime = result.CommandResult.StartTime
	response.EndTime = result.CommandResult.EndTime
	response.ExecTime = result.CommandResult.ExecTime

	auditRecord := Data.NewAuditRecord(command, response)

	log.Printf("[ExecCommand] sending audit record to registrar: %v\n", auditRecord)
	ns.auditChan <- &Data.RecMessage{Rec: auditRecord}

	if result.Command.Name() == "aks-start" || result.Command.Name() == "aks-stop" {
		var record *Data.KubeStatusRecord
		switch result.Command.Name() {
		case "aks-start":
			record = Data.NewKubeStatusRecord(
				result.Command.(*ServerCommands.AksStartCommand).GetClusterName(),
				command.UserID,
				"Running",
			)
		case "aks-stop":
			record = Data.NewKubeStatusRecord(
				result.Command.(*ServerCommands.AksStopCommand).GetClusterName(),
				command.UserID,
				"Stopped",
			)
		}
		log.Printf("[ExecCommand] sending kube status record to registrar: %v\n", record)
		ns.auditChan <- &Data.RecMessage{Rec: record}
	}

	return response, err
}

func (ns *nebirosServer) CommandListener(stream nebiros.Nebiros_CommandListenerServer) error {
	for {
		incmd, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		response := &nebiros.CommandResponse{}
		if incmd.UserID == "" {
			return fmt.Errorf("no valid user set. aborting command execution. COMMAND: %+v", incmd)
		}
		result, err := ns.DoCommand(incmd)

		log.Printf("[CommandListener] Result: %v\n", result)
		if err != nil {
			response.CmdError = result.CommandResult.CmdError
		} else {
			response.CmdResult = fmt.Sprintf("%s\n", result.CommandResult.CmdResult)
		}

		response.StartTime = result.CommandResult.StartTime
		response.EndTime = result.CommandResult.EndTime
		response.ExecTime = result.CommandResult.ExecTime

		err = stream.SendMsg(response)
		if err != nil {
			fmt.Println("error sending response:", err)
		}

		auditRecord := Data.NewAuditRecord(incmd, response)

		log.Printf("[CommandListener] sending audit record to registrar: %v\n", auditRecord)
		ns.auditChan <- &Data.RecMessage{Rec: auditRecord}

		if result.Command.Name() == "aks-start" || result.Command.Name() == "aks-stop" {
			record := &Data.KubeStatusRecord{
				Username: Data.GetNullString(incmd.UserID),
				Tstp:     time.Now().UTC(),
			}
			switch result.Command.Name() {
			case "aks-start":
				record.ClusterName = Data.GetNullString(result.Command.(*ServerCommands.AksStartCommand).GetClusterName())
				record.Status = Data.GetNullString("Running")
			case "aks-stop":
				record.ClusterName = Data.GetNullString(result.Command.(*ServerCommands.AksStopCommand).GetClusterName())
				record.Status = Data.GetNullString("Stopped")
			}
			ns.auditChan <- &Data.RecMessage{Rec: record}
		}
	}
}

func (ns *nebirosServer) DoCommand(command *nebiros.Command) (*Commands.Response, error) {
	response := &Commands.Response{}
	cr := &nebiros.CommandResponse{}
	start := time.Now().UTC()
	cr.StartTime = timestamppb.New(start)

	if command.CmdName == "" {
		err := errors.New("no sub-command provided")
		Commands.HandleCommandError(cr, err, start)
		response.CommandResult = *cr
		return response, err
	}

	if ok, i := Commands.InCommandList(ns.validCmds, command.CmdName); ok {
		cmd, err := ns.validCmds[i].Call(ns.entries)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		err = cmd.Init(command.CmdOpts)
		if err != nil {
			Commands.HandleCommandError(cr, err, start)
			response.CommandResult = *cr
			return response, err
		}

		response, err = cmd.Run()

		return response, err
	}

	err := fmt.Errorf("unknown sub-command: %s", command.CmdName)
	cr.CmdError = err.Error()
	response.CommandResult = *cr
	return response, err
}

func (ns *nebirosServer) GetRemoteCommands(ctx context.Context, RemoteCmdOpts *nebiros.GetRemoteCommandOpts) (*nebiros.RemoteCommandUsageResults, error) {
	rcur := &nebiros.RemoteCommandUsageResults{}

	for _, cmd := range ns.cmds {
		rcur.RemoteCommandUsage = append(rcur.GetRemoteCommandUsage(),
			fmt.Sprintf("%s\n", cmd.Name()),
			cmd.Usage(),
		)
	}

	return rcur, nil
}

func (ns *nebirosServer) AksStatusWatcher(interval int) {
	statusCommand := &nebiros.Command{CmdName: "aks-status", UserID: "NebirosServer"}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	for {
		if ServerStop {
			break
		}

		select {
		case <-ticker.C:
			_, err := ns.DoCommand(statusCommand)
			if err != nil {
				log.Printf("[AksStatusWatcher] error getting status: %s\n", err)
			}
		}
	}
}

var (
	showVersion   bool
	NoExec        bool
	ServerStop    bool
	rootDirectory string
)

func main() {
	flag.BoolVar(&NoExec, "noexec", false, "do not execute command")
	flag.BoolVar(&showVersion, "V", false, "display version information")
	flag.StringVar(&rootDirectory, "d", "/opt/Nebiros/server/", "working directory")
	flag.Parse()

	if rootDirectory[len(rootDirectory)-1] != '/' {
		rootDirectory += "/"
	}
	log.Printf("Using root directory: %s\n", rootDirectory)

	if showVersion {
		fmt.Printf("Nebiros Server version: %s\n", version)
		os.Exit(0)
	}

	serverConfig, err := Config.LoadServerConfig(rootDirectory)
	if err != nil {
		log.Fatalf("failed to load server configuration: %s", err.Error())
	}

	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = godotenv.Load(fmt.Sprintf("%sConfig/.env", rootDirectory))
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}

	updateEnvVars(rootDirectory)

	AksEntries, err := Utils.NewAksEntryList(fmt.Sprintf("%sConfig/aks_config.json", rootDirectory))
	if err != nil {
		log.Fatalf("failed to load AKS entries configuration: %v", err)
	}

	// set up the channel for the registrar
	//auditChannel := make(chan *Data.AuditRecord, 20)
	auditChannel := make(chan *Data.RecMessage, 20)

	log.Printf("Listening on: %s:%d\n", serverConfig.Host, serverConfig.Port)

	var opts []grpc.ServerOption

	if serverConfig.Tls.UseTls {
		log.Printf("TLS not currently implemented\n")
	}

	grpcServer := grpc.NewServer(opts...)
	server := newServer(AksEntries, serverConfig, auditChannel, NoExec)
	log.Printf("Starting Registrar watcher\n")
	go server.keeper.Watch(auditChannel) // start a go routine to just watch for audit messages

	log.Printf("Starting AKS status watcher\n")
	go server.AksStatusWatcher(serverConfig.WatchSleep) // start go routine to watch aks status

	defer func() {
		err = server.keeper.Shutdown()
		if err != nil {
			log.Printf("error shutting down keeper: %s\n", err.Error())
		}
	}()

	nebiros.RegisterNebirosServer(grpcServer, server)
	err = grpcServer.Serve(listen)

	if err != nil {
		log.Fatalln(err)
	}
}

func updateEnvVars(rootDirectory string) {
	clientCert := os.Getenv("AZURE_CLIENT_CERTIFICATE_PATH")
	clientPem := os.Getenv("AZURE_CLIENT_PEM_PATH")

	os.Setenv("AZURE_CLIENT_CERTIFICATE_PATH", fmt.Sprintf("%s/%s", rootDirectory, clientCert))
	os.Setenv("AZURE_CLIENT_PEM_PATH", fmt.Sprintf("%s/%s", rootDirectory, clientPem))
}
