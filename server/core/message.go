package core

import "google.golang.org/protobuf/proto"

type Message struct {
	proto.Message
	RemoteAddr string
}
