package handler

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
)

var (
	ErrNilStatus       = errors.New("nil status or unknown error")
	ErrAssertFailure   = errors.New("assert spite type failure")
	ErrNilResponseBody = errors.New("must return spite body")
)

func HandleMaleficError(content *implantpb.Spite) error {
	if content == nil {
		return fmt.Errorf("nil spite")
	}
	var err error
	switch content.Error {
	case 0:
		return nil
	case consts.MaleficErrorPanic:
		err = fmt.Errorf("module Panic")
	case consts.MaleficErrorUnpackError:
		err = fmt.Errorf("module unpack error")
	case consts.MaleficErrorMissbody:
		err = fmt.Errorf("module miss body")
	case consts.MaleficErrorModuleError:
		err = fmt.Errorf("module error")
	case consts.MaleficErrorModuleNotFound:
		err = fmt.Errorf("module not found")
	case consts.MaleficErrorTaskError:
		return HandleTaskError(content.Status)
	case consts.MaleficErrorTaskNotFound:
		err = fmt.Errorf("task not found")
	case consts.MaleficErrorTaskOperatorNotFound:
		err = fmt.Errorf("task operator not found")
	case consts.MaleficErrorExtensionNotFound:
		err = fmt.Errorf("extension not found")
	case consts.MaleficErrorUnexceptBody:
		err = fmt.Errorf("unexcept body")
	default:
		err = fmt.Errorf("unknown Malefic error, %d", content.Error)
	}
	return err
}

func HandleTaskError(status *implantpb.Status) error {
	var err error
	switch status.Status {
	case 0:
		return nil
	case consts.TaskErrorOperatorError:
		err = fmt.Errorf("task error: %s", status.Error)
	case consts.TaskErrorNotExpectBody:
		err = fmt.Errorf("task error: %s", status.Error)
	case consts.TaskErrorFieldRequired:
		err = fmt.Errorf("task error: %s", status.Error)
	case consts.TaskErrorFieldLengthMismatch:
		err = fmt.Errorf("task error: %s", status.Error)
	case consts.TaskErrorFieldInvalid:
		err = fmt.Errorf("task error: %s", status.Error)
	case consts.TaskError:
		err = fmt.Errorf("task error: %s", status.Error)
	default:
		err = fmt.Errorf("unknown error, %v", status)
	}
	return err
}

func AssertRequestName(req *implantpb.Request, expect types.MsgName) error {
	if req.Name != string(expect) {
		return ErrAssertFailure
	}
	return nil
}

func AssertResponse(spite *implantpb.Spite, expect types.MsgName) error {
	body := spite.GetBody()
	if body == nil && expect != types.MsgNil {
		return ErrNilResponseBody
	}

	if expect != types.MessageType(spite) {
		return ErrAssertFailure
	}
	return nil
}

func AssertStatusAndResponse(spite *implantpb.Spite, expect types.MsgName) error {
	if err := HandleMaleficError(spite); err != nil {
		return err
	}
	return AssertResponse(spite, expect)
}
