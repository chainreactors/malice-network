package core

type Event struct {
	Session *Session
	Job     *Job
	Client  *Client

	EventType string

	Data []byte
	Err  error
}
