package context

import (
	"fmt"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func requireContextTask(sess *client.Session, task *clientpb.Task) error {
	if sess == nil || sess.Session == nil {
		return fmt.Errorf("session is required")
	}
	if task == nil {
		return fmt.Errorf("task is required")
	}
	return nil
}
