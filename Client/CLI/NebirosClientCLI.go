package main

import (
	"flag"
	"fmt"
	"log"
	"nebiros/Utils"
	"os"
	"strings"

	// internal packages
	"nebiros"
	"nebiros/Client"
	"nebiros/Client/Config"
)

var version = "undefined"

func main() {
	showVersion := flag.Bool("V", false, "show cli version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Nebiros Client CLI version: %s\n", version)
		os.Exit(0)
	}

	cc, err := Config.GetClientConfig()
	if err != nil {
		log.Fatalf("failed to get client Config: %s", err.Error())
	}

	nc, err := Client.NewNebirosClient(cc)
	if err != nil {
		log.Fatalf("failed to create NebirosClient: %+v\n", err)
	}

	cmd := &nebiros.Command{
		UserID:  cc.User,
		CmdName: os.Args[1],
		CmdOpts: os.Args[2:],
	}

	// only connect if the command is a remote command OR if wanting usage for remote commands
	if (cmd.CmdName == "help" && Utils.StringSliceContains(cmd.CmdOpts, "-remote")) || !nc.IsLocalCommand(cmd) {
		_, err = nc.Connect()
		if err != nil {
			log.Fatalf("failed to connect to Nebiros: %+v\n", err)
		}
		defer nc.Connection.Close()
	}

	result, err := nc.DoCommand(cmd)
	if result != nil && err == nil {
		if len(result.CmdResult) > 0 {
			fmt.Printf("%s\n", strings.TrimRight(result.CmdResult, "\n"))
		}

		if len(result.CmdError) > 0 {
			fmt.Printf("%s\n", strings.TrimRight(result.CmdError, "\n"))
		}

		fmt.Printf("Command completed in %s\n", result.ExecTime.AsDuration())
	} else {
		fmt.Printf("Command error: %s", err.Error())
	}
}
