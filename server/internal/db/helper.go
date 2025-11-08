package db

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
)

// ============================================
// Plural Type Definitions
// ============================================

// Sessions is a slice of Session models
type Sessions []*models.Session

// ToProtobuf converts Sessions to protobuf
func (s Sessions) ToProtobuf() *clientpb.Sessions {
	pbSessions := &clientpb.Sessions{Sessions: make([]*clientpb.Session, 0, len(s))}
	for _, session := range s {
		if session != nil {
			pbSessions.Sessions = append(pbSessions.Sessions, session.ToProtobuf())
		}
	}
	return pbSessions
}

// Tasks is a slice of Task models
type Tasks []*models.Task

// ToProtobuf converts Tasks to protobuf
func (t Tasks) ToProtobuf() *clientpb.Tasks {
	pbTasks := &clientpb.Tasks{Tasks: make([]*clientpb.Task, 0, len(t))}
	for _, task := range t {
		if task != nil {
			pbTasks.Tasks = append(pbTasks.Tasks, task.ToProtobuf())
		}
	}
	return pbTasks
}

// Artifacts is a slice of Artifact models
type Artifacts []*models.Artifact

// ToProtobuf converts Artifacts to protobuf (without binary content)
func (a Artifacts) ToProtobuf() *clientpb.Artifacts {
	pbArtifacts := &clientpb.Artifacts{Artifacts: make([]*clientpb.Artifact, 0, len(a))}
	for _, artifact := range a {
		if artifact != nil {
			pbArtifacts.Artifacts = append(pbArtifacts.Artifacts, artifact.ToProtobuf([]byte{}))
		}
	}
	return pbArtifacts
}

// Pipelines is a slice of Pipeline models
type Pipelines []*models.Pipeline

// ToProtobuf converts Pipelines to protobuf
func (p Pipelines) ToProtobuf() *clientpb.Pipelines {
	pbPipelines := &clientpb.Pipelines{Pipelines: make([]*clientpb.Pipeline, 0, len(p))}
	for _, pipeline := range p {
		if pipeline != nil {
			pbPipelines.Pipelines = append(pbPipelines.Pipelines, pipeline.ToProtobuf())
		}
	}
	return pbPipelines
}

// Profiles is a slice of Profile models
type Profiles []*models.Profile

// ToProtobuf converts Profiles to protobuf
func (p Profiles) ToProtobuf() *clientpb.Profiles {
	pbProfiles := &clientpb.Profiles{Profiles: make([]*clientpb.Profile, 0, len(p))}
	for _, profile := range p {
		if profile != nil {
			pbProfiles.Profiles = append(pbProfiles.Profiles, profile.ToProtobuf())
		}
	}
	return pbProfiles
}

// Operators is a slice of Operator models
type Operators []*models.Operator

// ToProtobuf converts Operators to protobuf
func (o Operators) ToProtobuf() *clientpb.Clients {
	pbClients := &clientpb.Clients{Clients: make([]*clientpb.Client, 0, len(o))}
	for _, operator := range o {
		if operator != nil {
			pbClients.Clients = append(pbClients.Clients, operator.ToProtobuf())
		}
	}
	return pbClients
}

// Contexts is a slice of Context models
type Contexts []*models.Context

// ToProtobuf converts Contexts to protobuf
func (c Contexts) ToProtobuf() *clientpb.Contexts {
	pbContexts := &clientpb.Contexts{Contexts: make([]*clientpb.Context, 0, len(c))}
	for _, context := range c {
		if context != nil {
			pbContexts.Contexts = append(pbContexts.Contexts, context.ToProtobuf())
		}
	}
	return pbContexts
}

// Certificates is a slice of Certificate models
type Certificates []*models.Certificate

// ToProtobuf converts Certificates to protobuf TLS slice
func (c Certificates) ToProtobuf() []*clientpb.TLS {
	tlsList := make([]*clientpb.TLS, 0, len(c))
	for _, cert := range c {
		if cert != nil {
			tlsList = append(tlsList, cert.ToProtobuf())
		}
	}
	return tlsList
}

// ============================================
// Generic CRUD Operations
// ============================================

// Save creates or updates a model (generic save operation)
func Save(model interface{}) error {
	return Session().Save(model).Error
}

// Update updates a model (only updates non-zero fields)
func Update(model interface{}) error {
	return Session().Updates(model).Error
}

// UpdateFields updates specific fields of a model
func UpdateFields(model interface{}, fields map[string]interface{}) error {
	return Session().Model(model).Updates(fields).Error
}

// Delete soft-deletes a model
func Delete(model interface{}) error {
	return Session().Delete(model).Error
}

// ============================================
// SessionQuery Builder
// ============================================

type SessionQuery struct {
	db *gorm.DB
}

// NewSessionQuery creates a new session query builder
func NewSessionQuery() *SessionQuery {
	return &SessionQuery{
		db: Session(),
	}
}

// WhereID filters by session ID
func (q *SessionQuery) WhereID(id string) *SessionQuery {
	q.db = q.db.Where("session_id = ?", id)
	return q
}

// WhereAlive filters by alive status
func (q *SessionQuery) WhereAlive(alive bool) *SessionQuery {
	q.db = q.db.Where("is_alive = ?", alive)
	return q
}

// WhereGroup filters by group name
func (q *SessionQuery) WhereGroup(group string) *SessionQuery {
	q.db = q.db.Where("group_name = ?", group)
	return q
}

// WhereRemoved filters by removed status
func (q *SessionQuery) WhereRemoved(removed bool) *SessionQuery {
	q.db = q.db.Where("is_removed = ?", removed)
	return q
}

// WhereType filters by session type
func (q *SessionQuery) WhereType(typ string) *SessionQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// OrderBy orders results by field
func (q *SessionQuery) OrderBy(field string) *SessionQuery {
	q.db = q.db.Order(field)
	return q
}

// Limit limits the number of results
func (q *SessionQuery) Limit(limit int) *SessionQuery {
	q.db = q.db.Limit(limit)
	return q
}

// Offset sets the offset for results
func (q *SessionQuery) Offset(offset int) *SessionQuery {
	q.db = q.db.Offset(offset)
	return q
}

// Find executes the query and returns multiple sessions
func (q *SessionQuery) Find() (Sessions, error) {
	var sessions Sessions
	err := q.db.Find(&sessions).Error
	return sessions, err
}

// First executes the query and returns the first session
func (q *SessionQuery) First() (*models.Session, error) {
	var session models.Session
	err := q.db.First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Count counts the number of matching sessions
func (q *SessionQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.Session{}).Count(&count).Error
	return count, err
}

// ============================================
// TaskQuery Builder
// ============================================

type TaskQuery struct {
	db *gorm.DB
}

// NewTaskQuery creates a new task query builder
func NewTaskQuery() *TaskQuery {
	return &TaskQuery{
		db: Session(),
	}
}

// WhereID filters by task ID
func (q *TaskQuery) WhereID(id string) *TaskQuery {
	q.db = q.db.Where("id = ?", id)
	return q
}

// WhereSessionID filters by session ID
func (q *TaskQuery) WhereSessionID(sessionID string) *TaskQuery {
	q.db = q.db.Where("session_id = ?", sessionID)
	return q
}

// WhereType filters by task type
func (q *TaskQuery) WhereType(taskType string) *TaskQuery {
	q.db = q.db.Where("type = ?", taskType)
	return q
}

// WhereFinished filters by finished status
func (q *TaskQuery) WhereFinished(finished bool) *TaskQuery {
	q.db = q.db.Where("finished = ?", finished)
	return q
}

// OrderBySeq orders by sequence number
func (q *TaskQuery) OrderBySeq() *TaskQuery {
	q.db = q.db.Order("seq ASC")
	return q
}

// OrderBy orders by specified field
func (q *TaskQuery) OrderBy(field string) *TaskQuery {
	q.db = q.db.Order(field)
	return q
}

// Limit limits the number of results
func (q *TaskQuery) Limit(limit int) *TaskQuery {
	q.db = q.db.Limit(limit)
	return q
}

// Find executes the query and returns multiple tasks
func (q *TaskQuery) Find() (Tasks, error) {
	var tasks Tasks
	err := q.db.Find(&tasks).Error
	return tasks, err
}

// First executes the query and returns the first task
func (q *TaskQuery) First() (*models.Task, error) {
	var task models.Task
	err := q.db.First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Count counts the number of matching tasks
func (q *TaskQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.Task{}).Count(&count).Error
	return count, err
}

// ============================================
// PipelineQuery Builder
// ============================================

type PipelineQuery struct {
	db *gorm.DB
}

// NewPipelineQuery creates a new pipeline query builder
func NewPipelineQuery() *PipelineQuery {
	return &PipelineQuery{
		db: Session(),
	}
}

// WhereName filters by pipeline name
func (q *PipelineQuery) WhereName(name string) *PipelineQuery {
	q.db = q.db.Where("name = ?", name)
	return q
}

// WhereListenerID filters by listener ID
func (q *PipelineQuery) WhereListenerID(listenerID string) *PipelineQuery {
	q.db = q.db.Where("listener_id = ?", listenerID)
	return q
}

// WhereEnabled filters by enabled status
func (q *PipelineQuery) WhereEnabled(enabled bool) *PipelineQuery {
	q.db = q.db.Where("enable = ?", enabled)
	return q
}

// WhereType filters by pipeline type
func (q *PipelineQuery) WhereType(typ string) *PipelineQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// WhereNotType filters by NOT pipeline type
func (q *PipelineQuery) WhereNotType(typ string) *PipelineQuery {
	q.db = q.db.Where("type != ?", typ)
	return q
}

// Preload preloads associations
func (q *PipelineQuery) Preload(relation string) *PipelineQuery {
	q.db = q.db.Preload(relation)
	return q
}

// OrderBy orders by specified field
func (q *PipelineQuery) OrderBy(field string) *PipelineQuery {
	q.db = q.db.Order(field)
	return q
}

// Find executes the query and returns multiple pipelines
func (q *PipelineQuery) Find() (Pipelines, error) {
	var pipelines Pipelines
	err := q.db.Find(&pipelines).Error
	return pipelines, err
}

// First executes the query and returns the first pipeline
func (q *PipelineQuery) First() (*models.Pipeline, error) {
	var pipeline models.Pipeline
	err := q.db.First(&pipeline).Error
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// Count counts the number of matching pipelines
func (q *PipelineQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.Pipeline{}).Count(&count).Error
	return count, err
}

// ============================================
// Improved List Functions (returns models)
// ============================================

// ListSessions returns all sessions (non-removed)
func ListSessions() (Sessions, error) {
	return NewSessionQuery().
		WhereRemoved(false).
		OrderBy("group_name").
		Find()
}

// ListAliveSessions returns all alive sessions
func ListAliveSessions() (Sessions, error) {
	return FindAliveSessions()
}

// ListTasks returns all tasks
func ListTasks() (Tasks, error) {
	return NewTaskQuery().Find()
}

// ListTasksBySession returns tasks for a specific session
func ListTasksBySession(sessionID string) (Tasks, error) {
	return NewTaskQuery().
		WhereSessionID(sessionID).
		OrderBySeq().
		Find()
}

// ListArtifacts returns all artifacts
func ListArtifacts() (Artifacts, error) {
	var artifacts Artifacts
	result := Session().Preload("Profile").Preload("Profile.Pipeline").Find(&artifacts)
	return artifacts, result.Error
}

// ListPipelinesByListener returns pipelines for a listener (non-website)
func ListPipelinesByListener(listenerID string) (Pipelines, error) {
	query := NewPipelineQuery().WhereNotType(consts.WebsitePipeline)
	if listenerID != "" {
		query = query.WhereListenerID(listenerID)
	}
	return query.Find()
}

// ListWebsitesByListener returns website pipelines for a listener
func ListWebsitesByListener(listenerID string) (Pipelines, error) {
	query := NewPipelineQuery().WhereType(consts.WebsitePipeline)
	if listenerID != "" {
		query = query.WhereListenerID(listenerID)
	}
	return query.Find()
}
