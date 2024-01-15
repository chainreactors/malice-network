package client

import (
	"errors"
	"fmt"
	"testing"
)

func TestRegister(t *testing.T) {

	aerr := errors.New("testttt")
	berr := fmt.Errorf("berr %w", aerr)
	println(errors.Is(berr, aerr))
	//implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	//implant.Register()
}
