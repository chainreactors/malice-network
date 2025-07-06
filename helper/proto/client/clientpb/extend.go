package clientpb

import "fmt"

func (pipe *Pipeline) Address() string {
	switch body := pipe.Body.(type) {
	case *Pipeline_Http:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Http.Port)
	case *Pipeline_Tcp:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Tcp.Port)
	default:
		return ""
	}
}

func (task *Task) Progress() string {
	if task.Total == -1 {
		return fmt.Sprintf("%d/∞", task.Cur)
	} else {
		return fmt.Sprintf("%d/%d", task.Cur, task.Total)
	}
}
