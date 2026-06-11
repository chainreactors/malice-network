package core

import (
	"sync"
	"sync/atomic"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

var (
	// Clients - Manages client active
	Clients = &clients{
		mutex:  &sync.Mutex{},
		active: map[int]*Client{},
	}

	clientID atomic.Uint32
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
			ID:     getClientID(),
			Name:   operatorName,
			Online: true,
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
		Client:    client.Client,
		Important: true,
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

// Remove removes a client and publishes a leave event so other operators are notified.
func (cc *clients) Remove(clientID int) {
	cc.mutex.Lock()
	client := cc.active[clientID]
	if client == nil {
		cc.mutex.Unlock()
		return
	}
	delete(cc.active, clientID)
	cc.mutex.Unlock()

	EventBroker.Publish(Event{
		EventType: consts.EventLeft,
		Client:    client.Client,
		Important: true,
	})
}

func (cc *clients) ActiveClients() []*Client {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	var cs []*Client
	for _, c := range cc.active {
		if c.Online {
			cs = append(cs, c)
		}
	}
	return cs
}

func GetCurrentID() uint32 {
	return clientID.Load()
}

func getClientID() uint32 {
	return clientID.Add(1)
}
