package Utils

import (
	"database/sql/driver"
	"fmt"
	go_ora "github.com/sijms/go-ora/v2"
	"io"
	"nebiros/Server/Config"
)

type OracleSource struct {
	connStr string
	dbh     *go_ora.Connection
}

func NewOracleSource(oracfg Config.OracleConfig) (*OracleSource, error) {
	os := &OracleSource{
		connStr: go_ora.BuildUrl(oracfg.Address, oracfg.Port, oracfg.Sid, oracfg.User, oracfg.Pass, nil),
	}

	conn, err := go_ora.NewConnection(os.connStr)
	if err != nil {
		fmt.Printf("error creating new connection to %s -- %s\n", oracfg.Address, err.Error())
		return nil, err
	}

	os.dbh = conn
	err = os.dbh.Open()
	if err != nil {
		fmt.Printf("error connecting to %s -- %s\n", oracfg.Address, err.Error())
		return nil, err
	}

	return os, nil
}

func (os *OracleSource) Close() {
	_ = os.dbh.Close()
}

func (os *OracleSource) SetDateTimeFormat(format string) error {
	_, err := os.dbh.Exec("ALTER SESSION SET time_zone = '0:0'")
	if err != nil {
		return fmt.Errorf("error setting session time_zone for %s -- %s\n", format, err.Error())
	}

	_, err = os.dbh.Exec(fmt.Sprintf("ALTER SESSION SET nls_date_format = '%s'", format))
	if err != nil {
		return fmt.Errorf("error setting date format for %s -- %s\n", format, err.Error())
	}

	_, err = os.dbh.Exec(fmt.Sprintf("ALTER SESSION SET nls_timestamp_format = '%s'", format))
	if err != nil {
		return fmt.Errorf("error setting timestamp format for %s -- %s\n", format, err.Error())
	}

	return nil
}

func (os *OracleSource) ExecQuery(query string) ([]map[string]string, error) {
	stmt, err := os.dbh.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(nil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := rows.Columns()
	values := make([]driver.Value, len(columns))

	var records []map[string]string
	for {
		err = rows.Next(values)
		if err != nil {
			break
		}

		Record(columns, values, &records)
	}

	if err != io.EOF {
		return nil, err
	}

	return records, nil
}

// convert the query values to a map column:value
func Record(columns []string, values []driver.Value, records *[]map[string]string) {
	rec := make(map[string]string)
	for i, col := range values {
		rec[columns[i]] = fmt.Sprintf("%v", col)
	}

	*records = append(*records, rec)
}
