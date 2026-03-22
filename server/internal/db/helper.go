package db

import (
	"errors"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
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
			pb := pipeline.ToProtobuf()
			if pb != nil {
				pbPipelines.Pipelines = append(pbPipelines.Pipelines, pb)
			}
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
	db       *gorm.DB
	loadCert bool
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

// WithCert enables automatic certificate loading for query results.
// When enabled, Find() and First() will call loadPipelineCert on each pipeline.
func (q *PipelineQuery) WithCert() *PipelineQuery {
	q.loadCert = true
	return q
}

// Find executes the query and returns multiple pipelines
func (q *PipelineQuery) Find() (Pipelines, error) {
	var pipelines Pipelines
	err := q.db.Find(&pipelines).Error
	if err != nil {
		return nil, err
	}
	if q.loadCert {
		for _, p := range pipelines {
			loadPipelineCert(p)
		}
	}
	return pipelines, err
}

// First executes the query and returns the first pipeline
func (q *PipelineQuery) First() (*models.Pipeline, error) {
	var pipeline models.Pipeline
	err := q.db.First(&pipeline).Error
	if err != nil {
		return nil, err
	}
	if q.loadCert {
		loadPipelineCert(&pipeline)
	}
	return &pipeline, nil
}

// Count counts the number of matching pipelines
func (q *PipelineQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.Pipeline{}).Count(&count).Error
	return count, err
}

// Delete deletes matching pipelines
func (q *PipelineQuery) Delete() error {
	return q.db.Delete(&models.Pipeline{}).Error
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
	return NewArtifactQuery().WithProfilePipeline().Find()
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

// ============================================
// ContextQuery Builder
// ============================================

// ContextQuery 用于构建Context查询的结构体
type ContextQuery struct {
	db *gorm.DB
}

// NewContextQuery 创建新的Context查询构建器
func NewContextQuery() *ContextQuery {
	return &ContextQuery{
		db: Session().
			Preload("Session").
			Preload("Pipeline").
			Preload("Task"),
	}
}

// WhereID filters by context ID
func (q *ContextQuery) WhereID(id uuid.UUID) *ContextQuery {
	q.db = q.db.Where("id = ?", id)
	return q
}

// ByType 按类型查询
func (q *ContextQuery) ByType(typ string) *ContextQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// BySession 按会话ID查询
func (q *ContextQuery) BySession(sessionID string) *ContextQuery {
	q.db = q.db.Where("session_id = ?", sessionID)
	return q
}

// ByTask 按任务ID查询
func (q *ContextQuery) ByTask(taskID string) *ContextQuery {
	q.db = q.db.Where("task_id = ?", taskID)
	return q
}

// ByPipeline 按Pipeline ID查询
func (q *ContextQuery) ByPipeline(pipelineID string) *ContextQuery {
	q.db = q.db.Where("pipeline_id = ?", pipelineID)
	return q
}

// ByNonce 按Nonce查询
func (q *ContextQuery) ByNonce(nonce string) *ContextQuery {
	q.db = q.db.Where("nonce = ?", nonce)
	return q
}

// Find 执行查询并返回结果
func (q *ContextQuery) Find() ([]*models.Context, error) {
	var contexts []*models.Context
	err := q.db.Find(&contexts).Error
	return contexts, err
}

// First 查询单个结果
func (q *ContextQuery) First() (*models.Context, error) {
	var context models.Context
	err := q.db.First(&context).Error
	if err != nil {
		return nil, err
	}
	return &context, nil
}

// Delete deletes matching contexts
func (q *ContextQuery) Delete() error {
	return q.db.Delete(&models.Context{}).Error
}

// ============================================
// OperatorQuery Builder
// ============================================

type OperatorQuery struct {
	db *gorm.DB
}

// NewOperatorQuery creates a new operator query builder
func NewOperatorQuery() *OperatorQuery {
	return &OperatorQuery{db: Session()}
}

// WhereName filters by operator name
func (q *OperatorQuery) WhereName(name string) *OperatorQuery {
	q.db = q.db.Where("name = ?", name)
	return q
}

// WhereType filters by operator type
func (q *OperatorQuery) WhereType(typ string) *OperatorQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// WhereFingerprint filters by certificate fingerprint
func (q *OperatorQuery) WhereFingerprint(fp string) *OperatorQuery {
	q.db = q.db.Where("fingerprint = ?", fp)
	return q
}

// WhereFingerprintEmpty filters operators with empty or null fingerprints
func (q *OperatorQuery) WhereFingerprintEmpty() *OperatorQuery {
	q.db = q.db.Where("fingerprint = '' OR fingerprint IS NULL")
	return q
}

// Find returns all matching operators
func (q *OperatorQuery) Find() (Operators, error) {
	var operators Operators
	err := q.db.Find(&operators).Error
	return operators, err
}

// First returns the first matching operator
func (q *OperatorQuery) First() (*models.Operator, error) {
	var op models.Operator
	err := q.db.First(&op).Error
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// Count counts matching operators
func (q *OperatorQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.Operator{}).Count(&count).Error
	return count, err
}

// Delete deletes matching operators
func (q *OperatorQuery) Delete() error {
	return q.db.Delete(&models.Operator{}).Error
}

// Update updates a single field on matching operators
func (q *OperatorQuery) Update(column string, value interface{}) error {
	return q.db.Model(&models.Operator{}).Update(column, value).Error
}

// Updates updates multiple fields on matching operators
func (q *OperatorQuery) Updates(fields map[string]interface{}) error {
	return q.db.Model(&models.Operator{}).Updates(fields).Error
}

// ============================================
// ProfileQuery Builder
// ============================================

type ProfileQuery struct {
	db *gorm.DB
}

// NewProfileQuery creates a new profile query builder
func NewProfileQuery() *ProfileQuery {
	return &ProfileQuery{db: Session()}
}

// WhereName filters by profile name
func (q *ProfileQuery) WhereName(name string) *ProfileQuery {
	q.db = q.db.Where("name = ?", name)
	return q
}

// WherePipelineID filters by pipeline ID
func (q *ProfileQuery) WherePipelineID(pipelineID string) *ProfileQuery {
	q.db = q.db.Where("pipeline_id = ?", pipelineID)
	return q
}

// WithPipeline preloads the Pipeline association
func (q *ProfileQuery) WithPipeline() *ProfileQuery {
	q.db = q.db.Preload("Pipeline")
	return q
}

// OrderByCreated orders by created_at ASC
func (q *ProfileQuery) OrderByCreated() *ProfileQuery {
	q.db = q.db.Order("created_at ASC")
	return q
}

// Find returns all matching profiles
func (q *ProfileQuery) Find() (Profiles, error) {
	var profiles Profiles
	err := q.db.Find(&profiles).Error
	return profiles, err
}

// First returns the first matching profile
func (q *ProfileQuery) First() (*models.Profile, error) {
	var profile models.Profile
	err := q.db.First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Delete deletes matching profiles
func (q *ProfileQuery) Delete() error {
	return q.db.Delete(&models.Profile{}).Error
}

// ============================================
// ArtifactQuery Builder
// ============================================

type ArtifactQuery struct {
	db *gorm.DB
}

// NewArtifactQuery creates a new artifact query builder
func NewArtifactQuery() *ArtifactQuery {
	return &ArtifactQuery{db: Session()}
}

// WhereName filters by artifact name
func (q *ArtifactQuery) WhereName(name string) *ArtifactQuery {
	q.db = q.db.Where("name = ?", name)
	return q
}

// WhereID filters by artifact ID (uint32)
func (q *ArtifactQuery) WhereID(id uint32) *ArtifactQuery {
	q.db = q.db.Where("id = ?", id)
	return q
}

// WhereType filters by artifact type
func (q *ArtifactQuery) WhereType(typ string) *ArtifactQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// WhereSource filters by artifact source
func (q *ArtifactQuery) WhereSource(source string) *ArtifactQuery {
	q.db = q.db.Where("source = ?", source)
	return q
}

// WhereStatus filters by artifact status
func (q *ArtifactQuery) WhereStatus(status string) *ArtifactQuery {
	q.db = q.db.Where("status = ?", status)
	return q
}

// WhereProfileName filters by profile name
func (q *ArtifactQuery) WhereProfileName(profileName string) *ArtifactQuery {
	q.db = q.db.Where("profile_name = ?", profileName)
	return q
}

// WhereOs filters by OS
func (q *ArtifactQuery) WhereOs(os string) *ArtifactQuery {
	q.db = q.db.Where("os = ?", os)
	return q
}

// WhereArch filters by architecture
func (q *ArtifactQuery) WhereArch(arch string) *ArtifactQuery {
	q.db = q.db.Where("arch = ?", arch)
	return q
}

// WherePipelineID filters artifacts by their profile's pipeline ID using a JOIN.
func (q *ArtifactQuery) WherePipelineID(pipelineID string) *ArtifactQuery {
	q.db = q.db.Joins("JOIN profiles ON profiles.name = artifacts.profile_name").
		Where("profiles.pipeline_id = ?", pipelineID)
	return q
}

// WherePathNotEmpty filters out artifacts with empty or NULL paths.
func (q *ArtifactQuery) WherePathNotEmpty() *ArtifactQuery {
	q.db = q.db.Where("path != '' AND path IS NOT NULL")
	return q
}

// WithProfile preloads the Profile association
func (q *ArtifactQuery) WithProfile() *ArtifactQuery {
	q.db = q.db.Preload("Profile")
	return q
}

// WithProfilePipeline preloads Profile and its Pipeline association
func (q *ArtifactQuery) WithProfilePipeline() *ArtifactQuery {
	q.db = q.db.Preload("Profile").Preload("Profile.Pipeline")
	return q
}

// Find returns all matching artifacts
func (q *ArtifactQuery) Find() (Artifacts, error) {
	var artifacts Artifacts
	err := q.db.Find(&artifacts).Error
	return artifacts, err
}

// First returns the first matching artifact
func (q *ArtifactQuery) First() (*models.Artifact, error) {
	var artifact models.Artifact
	err := q.db.First(&artifact).Error
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}

// Last returns the last matching artifact
func (q *ArtifactQuery) Last() (*models.Artifact, error) {
	var artifact models.Artifact
	err := q.db.Last(&artifact).Error
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}

// Delete deletes matching artifacts
func (q *ArtifactQuery) Delete() error {
	return q.db.Delete(&models.Artifact{}).Error
}

// Update updates a single field on matching artifacts
func (q *ArtifactQuery) Update(column string, value interface{}) error {
	return q.db.Model(&models.Artifact{}).Update(column, value).Error
}

// ============================================
// CertificateQuery Builder
// ============================================

type CertificateQuery struct {
	db *gorm.DB
}

// NewCertificateQuery creates a new certificate query builder
func NewCertificateQuery() *CertificateQuery {
	return &CertificateQuery{db: Session()}
}

// WhereName filters by certificate name
func (q *CertificateQuery) WhereName(name string) *CertificateQuery {
	q.db = q.db.Where("name = ?", name)
	return q
}

// WhereType filters by certificate type
func (q *CertificateQuery) WhereType(typ string) *CertificateQuery {
	q.db = q.db.Where("type = ?", typ)
	return q
}

// Find returns all matching certificates
func (q *CertificateQuery) Find() (Certificates, error) {
	var certs Certificates
	err := q.db.Find(&certs).Error
	return certs, err
}

// First returns the first matching certificate
func (q *CertificateQuery) First() (*models.Certificate, error) {
	var cert models.Certificate
	err := q.db.First(&cert).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// Delete deletes matching certificates
func (q *CertificateQuery) Delete() error {
	return q.db.Delete(&models.Certificate{}).Error
}

// UpdateFields updates specific fields on matching certificates
func (q *CertificateQuery) UpdateFields(fields map[string]interface{}) error {
	return q.db.Model(&models.Certificate{}).Updates(fields).Error
}

// ============================================
// AuthzRuleQuery Builder
// ============================================

type AuthzRuleQuery struct {
	db *gorm.DB
}

// NewAuthzRuleQuery creates a new authz rule query builder
func NewAuthzRuleQuery() *AuthzRuleQuery {
	return &AuthzRuleQuery{db: Session()}
}

// WhereRole filters by role
func (q *AuthzRuleQuery) WhereRole(role string) *AuthzRuleQuery {
	q.db = q.db.Where("role = ?", role)
	return q
}

// WhereID filters by rule ID
func (q *AuthzRuleQuery) WhereID(id string) *AuthzRuleQuery {
	q.db = q.db.Where("id = ?", id)
	return q
}

// Find returns all matching rules
func (q *AuthzRuleQuery) Find() ([]*models.AuthzRule, error) {
	var rules []*models.AuthzRule
	err := q.db.Find(&rules).Error
	return rules, err
}

// Count counts matching rules
func (q *AuthzRuleQuery) Count() (int64, error) {
	var count int64
	err := q.db.Model(&models.AuthzRule{}).Count(&count).Error
	return count, err
}

// Delete deletes matching rules
func (q *AuthzRuleQuery) Delete() error {
	return q.db.Delete(&models.AuthzRule{}).Error
}

// ============================================
// WebContentQuery Builder
// ============================================

type WebContentQuery struct {
	db *gorm.DB
}

// NewWebContentQuery creates a new web content query builder
func NewWebContentQuery() *WebContentQuery {
	return &WebContentQuery{db: Session()}
}

// WhereID filters by content ID
func (q *WebContentQuery) WhereID(id uuid.UUID) *WebContentQuery {
	q.db = q.db.Where("id = ?", id)
	return q
}

// WherePipelineID filters by pipeline/website ID
func (q *WebContentQuery) WherePipelineID(pipelineID string) *WebContentQuery {
	q.db = q.db.Where("pipeline_id = ?", pipelineID)
	return q
}

// WherePath filters by content path
func (q *WebContentQuery) WherePath(path string) *WebContentQuery {
	q.db = q.db.Where("path = ?", path)
	return q
}

// WithPipeline preloads the Pipeline association
func (q *WebContentQuery) WithPipeline() *WebContentQuery {
	q.db = q.db.Preload("Pipeline")
	return q
}

// Find returns all matching web contents
func (q *WebContentQuery) Find() ([]*models.WebsiteContent, error) {
	var contents []*models.WebsiteContent
	err := q.db.Find(&contents).Error
	return contents, err
}

// First returns the first matching web content
func (q *WebContentQuery) First() (*models.WebsiteContent, error) {
	var content models.WebsiteContent
	err := q.db.First(&content).Error
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// Delete deletes matching web contents
func (q *WebContentQuery) Delete() error {
	return q.db.Delete(&models.WebsiteContent{}).Error
}

// ============================================
// Additional TaskQuery methods
// ============================================

// WhereSeq filters by task sequence number
func (q *TaskQuery) WhereSeq(seq uint32) *TaskQuery {
	q.db = q.db.Where("seq = ?", seq)
	return q
}

// Delete deletes matching tasks
func (q *TaskQuery) Delete() error {
	return q.db.Delete(&models.Task{}).Error
}

// Update updates a single field on matching tasks
func (q *TaskQuery) Update(column string, value interface{}) error {
	return q.db.Model(&models.Task{}).Update(column, value).Error
}

// ============================================
// Additional SessionQuery methods
// ============================================

// Update updates a single field on matching sessions
func (q *SessionQuery) Update(column string, value interface{}) error {
	return q.db.Model(&models.Session{}).Update(column, value).Error
}

// ============================================
// Internal Helpers
// ============================================

// saveWithOmitEmpty saves a model, omitting specified FK columns when their value is empty.
// fkChecks maps column name -> current value; empty values cause the column to be omitted.
func saveWithOmitEmpty(model interface{}, fkChecks map[string]string) error {
	query := Session()
	for column, value := range fkChecks {
		if value == "" {
			query = query.Omit(column)
		}
	}
	return query.Save(model).Error
}

// createWithOmitEmpty creates a model, omitting specified FK columns when their value is empty.
func createWithOmitEmpty(model interface{}, fkChecks map[string]string) error {
	query := Session()
	for column, value := range fkChecks {
		if value == "" {
			query = query.Omit(column)
		}
	}
	return query.Create(model).Error
}

// loadPipelineCert loads and attaches TLS cert to a pipeline model if CertName is set.
func loadPipelineCert(pipeline *models.Pipeline) {
	if pipeline == nil || pipeline.CertName == "" {
		return
	}
	certificate, err := FindCertificate(pipeline.CertName)
	if err != nil && !errors.Is(err, ErrRecordNotFound) {
		logs.Log.Errorf("failed to find cert %s", err)
		return
	}
	if certificate != nil {
		pipeline.Tls = implanttypes.FromTls(certificate.ToProtobuf())
	}
}
