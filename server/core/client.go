package core

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"sync"
)

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

// NewClient - Create a new client object
func NewClient(operatorName string) *Client {
	return &Client{
		Client: &clientpb.Client{
			ID:   uint32(getClientID()),
			Name: operatorName,
		},
	}
}

// Client - Single client connection
type Client struct {
	*clientpb.Client
}

func (c *Client) ToProtobuf() *clientpb.Client {
	return c.Client
}

func (cc *clients) Add(client *Client) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.active[int(client.ID)] = client
	EventBroker.Publish(Event{
		EventType: consts.EventJoin,
		Client:    client,
	})
}

// AddClient - Add a client struct atomically
func (cc *clients) ActiveOperators() []string {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	operators := []string{}
	for _, client := range cc.active {
		operators = append(operators, client.Name)
	}
	return operators
}

// RemoveClient - Remove a client struct atomically
func (cc *clients) Remove(clientID int) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	client := cc.active[clientID]
	delete(cc.active, clientID)
	EventBroker.Publish(Event{
		EventType: consts.EventLeft,
		Client:    client,
	})
}

func getClientID() int {
	newID := clientID + 1
	clientID++
	return newID
}
