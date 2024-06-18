package Client

import (
	"context"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"nebiros/Utils"
	"time"

	// external packages
	"google.golang.org/grpc"
	// internal packages
	"nebiros"
	"nebiros/Client/ClientCommands"
	"nebiros/Client/Config"
	"nebiros/Commands"
)

type NebirosClient struct {
	RpcClient     nebiros.NebirosClient
	ClientConfig  *Config.ClientConfig
	Connection    *grpc.ClientConn
	Commands      []Commands.Command
	ValidCommands []Commands.ValidCommand
	RemoteUsage   []string
}

func NewNebirosClient(cc *Config.ClientConfig) (*NebirosClient, error) {
	nc := &NebirosClient{
		ClientConfig: cc,
	}

	nc.Commands = []Commands.Command{
		ClientCommands.NewShowClientConfig(nc.ClientConfig),
		ClientCommands.NewUpdateClientConfig(nc.ClientConfig),
	}

	nc.ValidCommands = []Commands.ValidCommand{
		{Name: "showconfig", Func: ClientCommands.NewShowClientConfig},
		{Name: "updateconfig", Func: ClientCommands.NewUpdateClientConfig},
		{Name: "help", Func: ClientCommands.NewClientHelp},
	}

	helpcmd := ClientCommands.NewClientHelp(nc.RemoteUsage)
	helpcmd.SetAvailableCommands(nc.Commands)
	nc.Commands = append(nc.Commands, helpcmd)

	return nc, nil
}

func (nc *NebirosClient) Connect() (bool, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	serverAddr := nc.ClientConfig.Host + ":" + nc.ClientConfig.Port
	conn, err := grpc.Dial(serverAddr, opts...)

	if err != nil {
		return false, fmt.Errorf("failed to dial: %+v\n", err)
	}

	nc.Connection = conn
	nc.RpcClient = nebiros.NewNebirosClient(conn)
	nc.RemoteUsage = nc.GetRemoteCommands(nc.RpcClient)

	return true, nil
}

func (nc *NebirosClient) IsLocalCommand(inCommand *nebiros.Command) bool {
	ok, _ := Commands.InCommandList(nc.ValidCommands, inCommand.CmdName)
	return ok
}

func (nc *NebirosClient) GetRemoteCommands(client nebiros.NebirosClient) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	rmtUsages, err := nc.RpcClient.GetRemoteCommands(ctx, &nebiros.GetRemoteCommandOpts{})
	if err != nil {
		log.Fatalf("client.GetRemoteCommands failed: %s", err.Error())
	}

	return rmtUsages.RemoteCommandUsage
}

func (nc *NebirosClient) runCommand(command *nebiros.Command) (*nebiros.CommandResponse, error) {
	/*
		@TODO: the start/stop commands don't really need the response
		  	   just message that the action has started and have the
		 	   client manually check the status
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	stream, err := nc.RpcClient.CommandListener(ctx)
	if err != nil {
		log.Printf("client.CommandListener failed: %v", err)
	}

	response := &nebiros.CommandResponse{}
	waitc := make(chan *nebiros.CommandResponse)
	go func() (*nebiros.CommandResponse, error) {
		for {
			err = stream.RecvMsg(response)

			if err == io.EOF {
				close(waitc)
				return response, nil
			}

			if err != nil {
				log.Printf("client.CommandListener failed: %v", err)
				return response, err
			}

			waitc <- response
		}
	}()

	if err := stream.Send(command); err != nil {
		log.Fatalf("client.CommandListener failed: %v %v", command, err)
	}
	stream.CloseSend()
	response = <-waitc

	return response, nil
}

func (nc *NebirosClient) ExecCommand(command *nebiros.Command) (*nebiros.CommandResponse, error) {
	response := &nebiros.CommandResponse{}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	response, err := nc.RpcClient.ExecCommand(ctx, command)
	if err != nil {
		log.Printf("client.ExecCommand failed: %s", err.Error())
	}

	return response, err
}

func (nc *NebirosClient) DoCommand(command *nebiros.Command) (*nebiros.CommandResponse, error) {
	cr := &nebiros.CommandResponse{}
	if command.CmdName == "" {
		err := fmt.Errorf("no sub-command provided")
		cr.CmdError = err.Error()
		return cr, err
	}

	if ok, i := Commands.InCommandList(nc.ValidCommands, command.CmdName); ok {
		// check to see if the command is a client command
		err := nc.Commands[i].Init(command.CmdOpts)
		if err != nil {
			cr.CmdError = err.Error()
			return cr, err
		}

		response, err := nc.Commands[i].Run()

		if command.CmdName == "help" && Utils.StringSliceContains(command.CmdOpts, "-remote") {
			r, e := nc.runCommand(command)
			if e != nil {
				cr.CmdError = e.Error()
				return cr, e
			}

			response.CommandResult.CmdResult += fmt.Sprintf("\nREMOTE COMMANDS:\n%s", r.CmdResult)
		}

		return &response.CommandResult, err
	} else {
		// try to execute server command
		//return nc.ExecCommand(command)
		return nc.runCommand(command)
	}
}
