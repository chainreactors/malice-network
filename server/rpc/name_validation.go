package rpc

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func validateScopedResourceName(kind, name string) error {
	if name == "" {
		return nil
	}
	if strings.Contains(name, ":") {
		return fmt.Errorf("%s name %q cannot contain ':'", kind, name)
	}
	return nil
}

func validatePipelineIdentity(pipeline *clientpb.Pipeline) error {
	if pipeline == nil {
		return nil
	}
	if err := validateScopedResourceName("listener", pipeline.ListenerId); err != nil {
		return err
	}
	return validateScopedResourceName("pipeline", pipeline.Name)
}
