package Utils

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

type Entitlement struct {
	UserMap  map[string]bool
	Filepath string
}

func NewEntitlement(path string) *Entitlement {
	return &Entitlement{UserMap: make(map[string]bool), Filepath: path}
}

func (e *Entitlement) IsEntitled(user string) bool {
	if ok, val := e.UserMap[strings.ToLower(user)]; ok {
		return val
	}

	return false
}

func (e *Entitlement) AddUser(user string, allowed bool) {
	e.UserMap[strings.ToLower(user)] = allowed
}

func (e *Entitlement) DelUser(user string) {
	if ok, _ := e.UserMap[strings.ToLower(user)]; ok {
		e.UserMap[user] = false
	}
}

func (e *Entitlement) Save() {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(e.UserMap)
	if err != nil {
		log.Printf("error encoding entitlement: %s\n", err)
		return
	}

	f, err := os.Create(e.Filepath)
	if err != nil {
		log.Printf("error creating file: %s\n", err)
		return
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	if err != nil {
		log.Printf("error writing to file: %s\n", err)
		return
	}

	err = f.Sync()
	if err != nil {
		log.Printf("error syncing to file: %s\n", err)
		return
	}
}

func (e *Entitlement) Load() {
	if _, err := os.Stat(e.Filepath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("no file found: %s\n", err)
			return
		}
	}

	f, err := os.Open(e.Filepath)
	if err != nil {
		log.Printf("error opening file: %s\n", err)
		return
	}

	reader := bufio.NewReader(f)
	decoder := gob.NewDecoder(reader)

	err = decoder.Decode(&e.UserMap)
	if err != nil {
		log.Printf("error decoding file: %s\n", err)
	}
}
