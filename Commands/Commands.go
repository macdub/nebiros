package Commands

import (
	"errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"nebiros"
	"reflect"
	"slices"
	"time"
)

type Command interface {
	Init([]string) error
	Run() (*Response, error)
	Name() string
	Usage() string
}

func InCommandList(list []ValidCommand, value string) (bool, int) {
	idx := slices.IndexFunc(list, func(c ValidCommand) bool {
		return c.Name == value
	})

	return idx >= 0, idx
}

func HandleCommandError(cr *nebiros.CommandResponse, err error, start time.Time) {
	cr.CmdError = err.Error()
	end := time.Now()
	cr.EndTime = timestamppb.New(end)
	cr.ExecTime = durationpb.New(end.Sub(start))
}

type ValidCommand struct {
	Name string
	Func interface{}
}

func (vc *ValidCommand) Call(params ...interface{}) (result Command, err error) {
	fn := reflect.ValueOf(vc.Func)
	if len(params) != fn.Type().NumIn() {
		err = errors.New("invalid number of parameters")
		return
	}

	inParams := make([]reflect.Value, len(params))
	for i, param := range params {
		inParams[i] = reflect.ValueOf(param)
	}

	res := fn.Call(inParams)
	result = res[0].Interface().(Command)
	return
}

type Response struct {
	CommandResult nebiros.CommandResponse
	Command       Command
}
