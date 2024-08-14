package core

import (
	"google.golang.org/protobuf/proto"
)

type Website interface {
	ID() string
	Start() error
	Addr() string
	Close() error
	ToProtobuf() proto.Message
	ToTLSProtobuf() proto.Message
}

type Websites map[string]Website

func (web Websites) Add(w Website) {
	web[w.ID()] = w
}

func (web Websites) Get(id string) Website {
	return web[id]
}
