package handler

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
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
	case errs.MaleficErrorPanic:
		err = fmt.Errorf("module Panic")
	case errs.MaleficErrorUnpackError:
		err = fmt.Errorf("module unpack error")
	case errs.MaleficErrorMissbody:
		err = fmt.Errorf("module miss body")
	case errs.MaleficErrorModuleError:
		err = fmt.Errorf("module error")
	case errs.MaleficErrorModuleNotFound:
		err = fmt.Errorf("module not found")
	case errs.MaleficErrorTaskError:
		return HandleTaskError(content.Status)
	case errs.MaleficErrorTaskNotFound:
		err = fmt.Errorf("task not found")
	case errs.MaleficErrorTaskOperatorNotFound:
		err = fmt.Errorf("task operator not found")
	case errs.MaleficErrorExtensionNotFound:
		err = fmt.Errorf("extension not found")
	case errs.MaleficErrorUnexceptBody:
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
	case errs.TaskErrorOperatorError:
		err = fmt.Errorf("task error: %s", status.Error)
	case errs.TaskErrorNotExpectBody:
		err = fmt.Errorf("task error: %s", status.Error)
	case errs.TaskErrorFieldRequired:
		err = fmt.Errorf("task error: %s", status.Error)
	case errs.TaskErrorFieldLengthMismatch:
		err = fmt.Errorf("task error: %s", status.Error)
	case errs.TaskErrorFieldInvalid:
		err = fmt.Errorf("task error: %s", status.Error)
	case errs.TaskError:
		err = fmt.Errorf("task error: %s", status.Error)
	default:
		err = fmt.Errorf("unknown error, %v", status)
	}
	return err
}

func AssertRequestName(req *implantpb.Request, expect types.MsgName) error {
	if req.Name != string(expect) {
		return fmt.Errorf("%w, assert request name failure, expect %s, got %s", ErrAssertFailure, expect, req.Name)
	}
	return nil
}

func AssertSpite(spite *implantpb.Spite, expect types.MsgName) error {
	body := spite.GetBody()
	if body == nil && expect != types.MsgNil {
		return ErrNilResponseBody
	}

	if expect != types.MessageType(spite) {
		return fmt.Errorf("%w, assert response type failure, expect %s, got %s", ErrAssertFailure, expect, types.MessageType(spite))
	}
	return nil
}

func AssertStatusAndSpite(spite *implantpb.Spite, expect types.MsgName) error {
	if err := HandleMaleficError(spite); err != nil {
		return err
	}
	return AssertSpite(spite, expect)
}
