package models

import (
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// Beacon - Represents a host machine
type Beacon struct {
	CreatedAt time.Time `gorm:"->;<-:create;"`

	ID                uuid.UUID `gorm:"type:uuid;"`
	Name              string
	Hostname          string
	UUID              uuid.UUID `gorm:"type:uuid;"` // Host UUID
	Username          string
	UID               string
	GID               string
	OS                string
	Arch              string
	Transport         string
	RemoteAddress     string
	PID               int32
	Filename          string
	LastCheckin       time.Time
	Version           string
	ReconnectInterval int64
	ActiveC2          string
	ProxyURL          string
	Locale            string

	ImplantBuildID uuid.UUID `gorm:"type:uuid;"`

	Interval    int64
	Jitter      int64
	NextCheckin int64

	Tasks []BeaconTask
}

// BeforeCreate - GORM hook
func (b *Beacon) BeforeCreate(tx *gorm.DB) (err error) {
	b.CreatedAt = time.Now()
	return nil
}

func (b *Beacon) ToProtobuf() *clientpb.Beacon {
	return &clientpb.Beacon{
		ID:                b.ID.String(),
		Name:              b.Name,
		Hostname:          b.Hostname,
		UUID:              b.UUID.String(),
		Username:          b.Username,
		UID:               b.UID,
		GID:               b.GID,
		OS:                b.OS,
		Arch:              b.Arch,
		Transport:         b.Transport,
		RemoteAddress:     b.RemoteAddress,
		PID:               b.PID,
		Filename:          b.Filename,
		LastCheckin:       b.LastCheckin.Unix(),
		Version:           b.Version,
		ActiveC2:          b.ActiveC2,
		ProxyURL:          b.ProxyURL,
		ReconnectInterval: b.ReconnectInterval,
		Interval:          b.Interval,
		Jitter:            b.Jitter,
		NextCheckin:       b.NextCheckin,
		Locale:            b.Locale,
		FirstContact:      b.CreatedAt.Unix(),
	}
}

func (b *Beacon) Task(envelope *sliverpb.Envelope) (*BeaconTask, error) {
	data, err := proto.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	task := &BeaconTask{
		BeaconID: b.ID,
		State:    PENDING,
		Request:  data,
	}
	return task, nil
}

// BeaconTask - Represents a host machine
const (
	PENDING   = "pending"
	SENT      = "sent"
	COMPLETED = "completed"
	CANCELED  = "canceled"
)

type BeaconTask struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	EnvelopeID  int64     `gorm:"uniqueIndex"`
	BeaconID    uuid.UUID `gorm:"type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	State       string
	SentAt      time.Time
	CompletedAt time.Time
	Description string
	Request     []byte // *sliverpb.Envelope
	Response    []byte // *sliverpb.Envelope
}

// BeforeCreate - GORM hook
func (b *BeaconTask) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	b.CreatedAt = time.Now()
	b.State = PENDING
	buf := make([]byte, 8)
	_, err = rand.Read(buf)
	if err != nil {
		panic(err)
	}
	b.EnvelopeID = int64(binary.LittleEndian.Uint64(buf))
	return nil
}

func (b *BeaconTask) ToProtobuf(content bool) *clientpb.BeaconTask {
	task := &clientpb.BeaconTask{
		ID:          b.ID.String(),
		BeaconID:    b.BeaconID.String(),
		CreatedAt:   int64(b.CreatedAt.UTC().Unix()),
		State:       b.State,
		SentAt:      int64(b.SentAt.UTC().Unix()),
		CompletedAt: int64(b.CompletedAt.UTC().Unix()),
		Description: b.Description,
	}
	if content {
		task.Request = b.Request
		task.Response = b.Response
	}
	return task
}
