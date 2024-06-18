package Data

import (
	"database/sql"
	"fmt"
	go_ora "github.com/sijms/go-ora/v2"
	"nebiros"
	"nebiros/Server/Config"
	"nebiros/Utils"
	"strings"
	"time"
)

type RecMessage struct {
	Rec Record
}

type Record interface {
	SetTimestamp(time.Time)
	GetCommandType() Utils.CommandType
}

type AuditRecord struct {
	User    sql.NullString `db:"name=username"`
	Cmd     sql.NullString `db:"name=command"`
	Opts    sql.NullString `db:"name=command_options"`
	Result  sql.NullString `db:"name=command_result"`
	ErrStr  sql.NullString `db:"name=command_error"`
	Start   time.Time      `db:"name=start_time"`
	End     time.Time      `db:"name=end_time"`
	Tstp    time.Time      `db:"name=tstp"`
	CmdType Utils.CommandType
}

func (r *AuditRecord) SetTimestamp(timestamp time.Time) {
	r.Tstp = timestamp
}

func (r *AuditRecord) GetCommandType() Utils.CommandType { return Utils.COMMAND }

type KubeStatusRecord struct {
	Username    sql.NullString `db:"name=username"`
	ClusterName sql.NullString `db:"name=clustername"`
	Status      sql.NullString `db:"name=status"`
	Tstp        time.Time      `db:"name=tstp"`
	CmdType     Utils.CommandType
}

func (r *KubeStatusRecord) SetTimestamp(timestamp time.Time) {
	r.Tstp = timestamp
}

func (r *KubeStatusRecord) GetCommandType() Utils.CommandType { return Utils.KUBERNETES }

func NewAuditRecord(command *nebiros.Command, response *nebiros.CommandResponse) *AuditRecord {
	ar := &AuditRecord{
		User:    GetNullString(command.UserID),
		Cmd:     GetNullString(command.CmdName),
		Opts:    GetNullString(strings.Join(command.CmdOpts, " ")),
		Result:  GetNullString(response.CmdResult),
		ErrStr:  GetNullString(response.CmdError),
		Start:   response.StartTime.AsTime(),
		End:     response.EndTime.AsTime(),
		CmdType: Utils.COMMAND,
	}

	return ar
}

func NewKubeStatusRecord(clustername string, userid string, status string) *KubeStatusRecord {
	ksr := &KubeStatusRecord{
		Username:    GetNullString(userid),
		ClusterName: GetNullString(clustername),
		Status:      GetNullString(status),
		Tstp:        time.Now().UTC(),
		CmdType:     Utils.KUBERNETES,
	}

	return ksr
}

func GetNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{String: s, Valid: true}
}

type Registrar struct {
	connStr         string
	dbh             *go_ora.Connection
	insertCommand   string
	insertK8sStatus string
	sleepTime       int
	DoShutdown      bool
	NoExec          bool
}

func NewRegistrar(oracfg *Config.OracleConfig, sleepTime int, noexec bool) (*Registrar, error) {
	registrar := &Registrar{
		connStr:    go_ora.BuildUrl(oracfg.Address, oracfg.Port, oracfg.Sid, oracfg.User, oracfg.Pass, nil),
		sleepTime:  sleepTime,
		DoShutdown: false,
		NoExec:     noexec,
	}

	registrar.insertCommand =
		`INSERT INTO nebiroslogs
			(username, command, command_options, command_result, start_time, end_time, tstp)
		VALUES
			(:username, :command, :command_options, :command_result, :start_time, :end_time, :tstp)`

	registrar.insertK8sStatus =
		`INSERT INTO k8sstatus
			(username, cluster_name, cluster_status, tstp)
		VALUES
			(:username, :clustername, :status, :tstp)`

	conn, err := go_ora.NewConnection(registrar.connStr)
	if err != nil {
		fmt.Printf("Error creating new connection to %s -- %s\n", oracfg.Address, err.Error())
		return nil, err
	}

	registrar.dbh = conn
	err = registrar.dbh.Open()
	if err != nil {
		fmt.Printf("Error connecting to %s -- %s\n", oracfg.Address, err.Error())
		return nil, err
	}

	return registrar, nil
}

func (r *Registrar) Shutdown() error {
	r.DoShutdown = true
	err := r.dbh.Close()
	if err != nil {
		return err
	}
	return nil
}

func (r *Registrar) ExecInsert(record Record, cmdType Utils.CommandType) error {
	var query string
	switch cmdType {
	case Utils.COMMAND:
		record = record.(*AuditRecord)
		query = r.insertCommand
	case Utils.KUBERNETES:
		record = record.(*KubeStatusRecord)
		query = r.insertK8sStatus
	default:
		return fmt.Errorf("unknown command type: %d\n", cmdType)
	}

	record.SetTimestamp(time.Now().UTC())
	_, err := r.dbh.Exec(query, record)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registrar) Watch(auditChannel <-chan *RecMessage) {
	// wait for stuff to come into the channel
	// then do the insert for each message that comes in
	for {
		if r.DoShutdown {
			break
		}

		select {
		case cr := <-auditChannel:
			err := r.ExecInsert(cr.Rec, cr.Rec.GetCommandType())
			if err != nil {
				fmt.Printf("error inserting AuditRecord: %s", err.Error())
			}
		default:
			// sleep for 3 seconds
			time.Sleep(time.Second * 3)
		}
	}
}
