package core

import "sync"

var (
	// Clients - Manages client active
	Clients = &clients{
		active: map[int]*Client{},
		mutex:  &sync.Mutex{},
	}

	clientID = 0
)

// clients - Manage active clients
type clients struct {
	mutex  *sync.Mutex
	active map[int]*Client
}

// Client - Single client connection
type Client struct {
	ID int
	//Operator *clientpb.Operator
}
