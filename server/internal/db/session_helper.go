package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

// ============================================
// Operator Operations
// ============================================

func HasOperator(typ string) (bool, error) {
	count, err := NewOperatorQuery().WhereType(typ).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func RemoveOperator(name string) error {
	return NewOperatorQuery().WhereName(name).Delete()
}

func CreateOperator(client *models.Operator) error {
	return Save(client)
}

func ListClients() (Operators, error) {
	return NewOperatorQuery().WhereType(mtls.Client).Find()
}

func ListListeners() (Operators, error) {
	return NewOperatorQuery().WhereType(mtls.Listener).Find()
}

// FindOperatorByFingerprint looks up an active operator by certificate SHA-256 fingerprint.
func FindOperatorByFingerprint(fingerprint string) (*models.Operator, error) {
	return NewOperatorQuery().WhereFingerprint(fingerprint).First()
}

// FindOperatorByName looks up an operator by name.
func FindOperatorByName(name string) (*models.Operator, error) {
	return NewOperatorQuery().WhereName(name).First()
}

// RevokeOperator sets the revoked flag on an operator.
func RevokeOperator(name string) error {
	return NewOperatorQuery().WhereName(name).Update("revoked", true)
}

// BackfillOperatorFingerprints computes and stores fingerprints for operators
// that were created before the fingerprint column existed.
func BackfillOperatorFingerprints() error {
	operators, err := NewOperatorQuery().WhereFingerprintEmpty().Find()
	if err != nil {
		return err
	}
	for _, op := range operators {
		if op.CertificatePEM == "" {
			continue
		}
		fp, err := certutils.CertFingerprint([]byte(op.CertificatePEM))
		if err != nil {
			logs.Log.Warnf("failed to compute fingerprint for operator %s: %v", op.Name, err)
			continue
		}

		// Infer role from Type if not set
		role := op.Role
		if role == "" {
			switch op.Type {
			case mtls.Listener:
				role = models.RoleListener
			default:
				role = models.RoleOperator
			}
		}

		NewOperatorQuery().WhereName(op.Name).Updates(map[string]interface{}{
			"fingerprint": fp,
			"role":        role,
		})
		logs.Log.Infof("backfilled fingerprint for operator %s (role=%s)", op.Name, role)
	}
	return nil
}

// ============================================
// Session Operations
// ============================================

func FindAliveSessions() (Sessions, error) {
	updateResult := Session().Exec(Adapter.FindAliveSessionsUpdateSQL())

	if updateResult.Error != nil {
		logs.Log.Infof("Failed to update inactive sessions: %v", updateResult.Error)
		return nil, updateResult.Error
	}

	var activeSessions Sessions
	result := Session().Raw(Adapter.FindAliveSessionsSelectSQL()).Scan(&activeSessions)

	if result.Error != nil {
		return nil, result.Error
	}

	return activeSessions, nil
}

func FindSession(sessionID string) (*models.Session, error) {
	session, err := NewSessionQuery().WhereID(sessionID).First()
	if err != nil {
		return nil, err
	}
	if session.IsRemoved {
		return nil, nil
	}
	return session, nil
}

// CreateOrRecoverSession creates a new session or recovers a soft-deleted one
// If a soft-deleted session with the same ID exists, it will be deleted and recreated
func CreateOrRecoverSession(session *models.Session) error {
	// Check if there's an existing session (including soft-deleted ones)
	var existingSession models.Session
	result := Session().Unscoped().Where("session_id = ?", session.SessionID).First(&existingSession)

	if result.Error == nil {
		// Session exists (might be soft-deleted), delete it first
		if err := Session().Unscoped().Delete(&existingSession).Error; err != nil {
			return err
		}
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// If it's not "record not found" error, return the error
		return result.Error
	}

	// Now create the new session
	return createWithOmitEmpty(session, map[string]string{"profile_name": session.ProfileName})
}

func FindTaskAndMaxTasksID(sessionID string) (Tasks, uint32, error) {
	tasks, err := NewTaskQuery().WhereSessionID(sessionID).Find()
	if err != nil {
		return tasks, 0, err
	}

	var max uint32
	for _, task := range tasks {
		if task.Seq > max {
			max = task.Seq
		}
	}

	return tasks, max, nil
}

func RemoveSession(sessionID string) error {
	result := Session().Model(&models.Session{}).Where("session_id = ?", sessionID).Update("is_removed", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// RecoverRemovedSession finds a soft-deleted session and resets is_removed flag.
// Returns nil,nil if no removed session found.
func RecoverRemovedSession(sessionID string) (*models.Session, error) {
	session, err := NewSessionQuery().WhereID(sessionID).WhereRemoved(true).First()
	if err != nil {
		return nil, err
	}
	session.IsRemoved = false
	if err := saveWithOmitEmpty(session, map[string]string{"profile_name": session.ProfileName}); err != nil {
		return nil, err
	}
	return session, nil
}

func UpdateSession(sessionID, note, group string) error {
	session, err := NewSessionQuery().WhereID(sessionID).First()
	if err != nil {
		return err
	}
	if group != "" {
		session.GroupName = group
	}
	if note != "" {
		session.Note = note
	}
	return saveWithOmitEmpty(session, map[string]string{"profile_name": session.ProfileName})
}

func UpdateSessionTimer(sessionID string, expression string, jitter float64) error {
	session, err := NewSessionQuery().WhereID(sessionID).First()
	if err != nil {
		return err
	}
	session.Data.Expression = expression
	if jitter != 0 {
		session.Data.Jitter = jitter
	}
	return saveWithOmitEmpty(session, map[string]string{"profile_name": session.ProfileName})
}

// ============================================
// Task Operations
// ============================================

func GetTask(taskID string) (*models.Task, error) {
	return NewTaskQuery().WhereID(taskID).First()
}

func GetTaskBySessionAndSeq(sessionID string, seq uint32) (*models.Task, error) {
	return NewTaskQuery().WhereSessionID(sessionID).WhereSeq(seq).First()
}

func AddTask(task *clientpb.Task) error {
	taskModel := &models.Task{
		ID:         task.SessionId + "-" + utils.ToString(task.TaskId),
		Seq:        task.TaskId,
		Type:       task.Type,
		SessionID:  task.SessionId,
		Cur:        int(task.Cur),
		Total:      int(task.Total),
		ClientName: task.Callby,
	}
	return Session().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"seq", "type", "session_id", "cur", "total", "client_name"}),
	}).Create(taskModel).Error
}

func UpdateTask(task *clientpb.Task) error {
	taskModel := &models.Task{
		ID: task.SessionId + "-" + utils.ToString(task.TaskId),
	}
	return taskModel.UpdateCur(Session(), int(task.Total))
}

func UpdateTaskCur(taskID string, cur int) error {
	taskModel := &models.Task{
		ID: taskID,
	}
	return taskModel.UpdateCur(Session(), cur)
}

func UpdateTaskFinish(taskID string) error {
	task, err := NewTaskQuery().WhereID(taskID).First()
	if err != nil {
		return err
	}
	return task.UpdateFinish(Session())
}

func UpdateTaskDescription(taskID, Description string) error {
	return NewTaskQuery().WhereID(taskID).Update("description", Description)
}

// ============================================
// Context Operations
// ============================================

func FindContext(contextID string) (*models.Context, error) {
	var ctx *models.Context
	// full UUID exact match
	if len(contextID) == 36 {
		if err := Session().Where("id = ?", contextID).First(&ctx).Error; err != nil {
			return nil, err
		}
		return ctx, nil
	}

	// prefix match for short IDs
	var contexts []*models.Context
	if err := Session().Where(Adapter.CastIDAsText("id"), contextID+"%").Find(&contexts).Error; err != nil {
		return nil, err
	}
	switch len(contexts) {
	case 0:
		return nil, fmt.Errorf("context not found with prefix: %s", contextID)
	case 1:
		return contexts[0], nil
	default:
		return nil, fmt.Errorf("ambiguous context prefix '%s', matched %d records", contextID, len(contexts))
	}
}

func GetContextByTask(taskID string) (*models.Context, error) {
	return NewContextQuery().ByTask(taskID).First()
}

func GetDownloadFiles(sid string) ([]*clientpb.File, error) {
	var files []*models.Context
	var result *gorm.DB
	if sid == "" {
		result = Session().Where("type = ?", consts.ContextDownload).Find(&files)
	} else {
		result = Session().Where("session_id = ?", sid).Preload("Session").Where("type = ?", consts.ContextDownload).Find(&files)
	}
	if result.Error != nil {
		return nil, result.Error
	}
	var res []*clientpb.File
	for _, file := range files {
		download, err := output.AsContext[*output.DownloadContext](file.Context)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(file.TaskID, "-")
		taskID := parts[len(parts)-1]
		taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
		res = append(res, &clientpb.File{
			Name:      download.Name,
			Local:     download.FilePath,
			Checksum:  download.Checksum,
			Remote:    download.TargetPath,
			TaskId:    uint32(taskIDUint),
			SessionId: file.SessionID,
		})
	}

	return res, nil
}

func SaveContext(ctx *clientpb.Context) (*models.Context, error) {
	contextDB, err := models.FromContextProtobuf(ctx)
	if err != nil {
		return nil, err
	}

	if ctx.Id != "" {
		id, err := uuid.FromString(ctx.Id)
		if err != nil {
			return nil, err
		}
		contextDB.ID = id
	}

	var omitFields []string
	if contextDB.SessionID == "" {
		omitFields = append(omitFields, "session_id")
	}
	if contextDB.PipelineID == "" {
		omitFields = append(omitFields, "pipeline_id")
	}
	if contextDB.TaskID == "" {
		omitFields = append(omitFields, "task_id")
	}

	query := Session().Session(&gorm.Session{
		FullSaveAssociations: false,
	}).Clauses(clause.OnConflict{
		UpdateAll: true,
	})
	if len(omitFields) > 0 {
		query = query.Omit(omitFields...)
	}
	return contextDB, query.Create(contextDB).Error
}

func DeleteContext(contextID string) error {
	id, err := uuid.FromString(contextID)
	if err != nil {
		return err
	}
	return NewContextQuery().WhereID(id).Delete()
}

// SaveSessionModel saves a session model to database
func SaveSessionModel(session *models.Session) error {
	return saveWithOmitEmpty(session, map[string]string{"profile_name": session.ProfileName})
}
