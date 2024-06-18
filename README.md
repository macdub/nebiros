# Nebiros

## Description
Nebiros provides an interface to monitor cloud resources as well as to modify running state of said resources.

## Nebiros Server
The Nebiros Server operates as the command and control node between users and the cloud resources. All actions
that check or modify cloud resources are parsed through this. It also operates the audit logging functions.

### Configuration
There are two configuration files. One for the AKS clusters and the other is the base server configuration. There
should be a "Config" directory in the same place as the binary.

#### Config/server_config.json
This configuration sets listening ports, tls, and remote database information

```json
{
  "oracle": {
    "address": "",
    "port": "",
    "sid": "",
    "username": "",
    "password": ""
  },
  "tls": {
    "use_tls": false,
    "cert_file": "path/to/cert/file",
    "key_file": "path/to/key/file"
  },
  "hostname": "<not used currently>",
  "port": 50051,
  "watcher_sleep_seconds": 120
}
```

#### Config/aks_config.json
A list of AKS clusters to monitor.

```json
[
  {
    "name": "Example Cluster",
    "resourceGroup": "resource-group-name",
    "clusterName": "aks-cluster-name"
  }
]
```

## Nebiros Client
This facilitates communication between the user facing applications and the Nebiros Server backend. This is used
by the user facing applications, such as the CLI and web page.

### Nebiros Client CLI
Simple command line interface. Commands are provided in this form:
```bash
$ NebirosClientCLI <COMMAND> <OPTIONS>

# example: get status of single configured cluster
$ NebirosClientCLI aks-status --cluster <CLUSTER_NAME>

# example: get status of all configured clusters
$ NebirosClientCLI aks-status
```

### Nebiros Web
Is the web UI that provides at-a-glance status of cloud resources. Provides users the ability to modify state of
resources. As with the Nebiros Server, there should be a "Config" directory in the place as the binary.

#### Config/.nebirosrc
This is the configuration used by the NebirosClient.

#### Config/webconfig.json
```json
{
  "port": 8080,
  "use_tls": false,
  "tls_config": {
    "cert_file_path": "<PATH/TO/CERTFILE>",
    "key_file_path": "<PATH/TO/KEYFILE>"
  },
  "client_config_path": "Config/",
  "oracle": {
    "address": "<ADDRESS>",
    "port": "<PORT>",
    "sid": "<SID>",
    "username": "<USERNAME>",
    "password": "<PASSWORD>"
  },
  "oauth": {
    "client_id": "<CLIENT ID>",
    "tenant": "<TENANT ID>",
    "redirectUrl": "<REDIRECT URL>",
    "scopes": ["User.Read"]
  }
}
```

## Build
The Nebiros Server, Nebiros Client CLI, and Nebiros Web Server all are built individually.

```bash
# from the root of the nebiros project

# Build the Server
$ go build Server/NebirosServer.go

# Build the CLI
$ go build Client/CLI/NebirosClientCLI.go

# Build the Web Server
$ go build Web/WebServer.go
```

Depending upon where these are deployed, you may see an error message when executing the binary referencing the GLIBC
libraries not having expected versions. If you wish to avoid modifying the system GLIBC versions, build with the following
environment variable:

```bash
# Build the Server
$ CGO_ENABLED=0 go build Server/NebirosServer.go

# Build the CLI
$ CGO_ENABLED=0 go build Client/CLI/NebirosClientCLI.go

# Build the Web Server
$ CGO_ENABLED=0 go build Web/WebServer.go
```
