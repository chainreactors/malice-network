package db

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/output"
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
	var count int64
	err := Session().Model(&models.Operator{}).Where("type = ?", typ).Count(&count).Error
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	return true, nil
}

func RemoveOperator(name string) error {
	err := Session().Where(&models.Operator{
		Name: name,
	}).Delete(&models.Operator{}).Error
	return err
}

func CreateOperator(client *models.Operator) error {
	err := Session().Save(client).Error
	return err

}

func ListClients() ([]*models.Operator, error) {
	var operators []*models.Operator
	err := Session().Find(&operators).Where("type = ?", mtls.Client).Error
	if err != nil {
		return nil, err
	}

	return operators, nil
}

func ListListeners() ([]*models.Operator, error) {
	var listeners []*models.Operator
	err := Session().Find(&listeners).Where("type = ?", mtls.Listener).Error
	return listeners, err
}

// ============================================
// Session Operations
// ============================================

func FindAliveSessions() (Sessions, error) {
	updateResult := Session().Exec(`
        UPDATE sessions
        SET is_alive = false
        WHERE last_checkin < strftime('%s', 'now') - (
            CAST(COALESCE(
                JSON_EXTRACT(data, '$.interval'),
                '30'  -- default value if interval doesn't exist
            ) AS INTEGER) * 2
        )
        AND is_removed = false
    `)

	if updateResult.Error != nil {
		logs.Log.Infof("Failed to update inactive sessions: %v", updateResult.Error)
		return nil, updateResult.Error
	}

	var activeSessions Sessions
	result := Session().Raw(`
        SELECT *
        FROM sessions
        WHERE last_checkin > strftime('%s', 'now') - (
            CAST(COALESCE(
                JSON_EXTRACT(data, '$.interval'),
                '30'  -- default value if interval doesn't exist
            ) AS INTEGER) * 2
        )
        AND is_removed = false
    `).Scan(&activeSessions)

	if result.Error != nil {
		return nil, result.Error
	}

	return activeSessions, nil
}

func FindSession(sessionID string) (*models.Session, error) {
	var session *models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	if session.IsRemoved {
		return nil, nil
	}
	//if session.Last.Before(time.Now().Add(-time.Second * time.Duration(session.Time.Interval*2))) {
	//	return nil, errors.New("session is dead")
	//}
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
	return Session().Create(session).Error
}

func FindTaskAndMaxTasksID(sessionID string) ([]*models.Task, uint32, error) {
	var tasks []*models.Task

	err := Session().Where("session_id = ?", sessionID).Find(&tasks).Error
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
	return result.Error
}

func UpdateSession(sessionID, note, group string) error {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	if group != "" {
		session.GroupName = group
	}
	if note != "" {
		session.Note = note
	}
	result = Session().Save(&session)
	return result.Error
}

func UpdateSessionTimer(sessionID string, expression string, jitter float64) error {
	var session *models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	session.Data.Expression = expression
	if jitter != 0 {
		session.Data.Jitter = jitter
	}
	result = Session().Save(&session)
	return result.Error
}

// ============================================
// Task Operations
// ============================================

func GetTask(taskID string) (*models.Task, error) {
	var task *models.Task
	err := Session().Where("id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
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
	return Session().Create(taskModel).Error
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
	var taskModel models.Task
	if err := Session().First(&taskModel, "id = ?", taskID).Error; err != nil {
		return err
	}
	return taskModel.UpdateFinish(Session())
}

func UpdateDownloadTotal(taskID uint32, sessionID string, total int) error {
	taskModel := &models.Task{
		ID: sessionID + "-" + utils.ToString(taskID),
	}
	return taskModel.UpdateTotal(Session(), total)
}

func UpdateTaskDescription(taskID, Description string) error {
	return Session().Model(&models.Task{}).Where("id = ?", taskID).Update("description", Description).Error
}

// ============================================
// Context Operations
// ============================================

func FindContext(taskID string) (*models.Context, error) {
	var task *models.Context
	if err := Session().Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func GetContextFilesBySessionID(sessionID string, fileTypes []string) ([]*models.Context, error) {
	var files []*models.Context
	query := Session().Model(&models.Context{}).Where("session_id = ?", sessionID)

	if len(fileTypes) > 0 {
		query = query.Where("type IN (?)", fileTypes)
	}

	result := query.Find(&files)
	if result.Error != nil {
		return nil, result.Error
	}
	return files, nil
}

func GetContextByTask(taskID string) (*models.Context, error) {
	var task *models.Context
	result := Session().Model(&models.Context{}).Where("task_id = ?", taskID).First(&task)
	if result.Error != nil {
		return task, result.Error
	}
	return task, nil
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

	return contextDB, Session().Session(&gorm.Session{
		FullSaveAssociations: false,
	}).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(contextDB).Error
}

func DeleteContext(contextID string) error {
	return Session().Where("id = ?", contextID).Delete(&models.Context{}).Error
}

// UpdateContext updates an existing context
func UpdateContext(context *models.Context) error {
	return Session().Save(context).Error
}

// FindContextBySessionAndTypeAndDate finds a context by session ID, type and date
// date format should be "2006-01-02"
func FindContextBySessionAndTypeAndDate(sessionID, contextType, date string) (*models.Context, error) {
	var context models.Context
	result := Session().Model(&models.Context{}).
		Joins("JOIN sessions ON contexts.session_id = sessions.session_id").
		Where("sessions.session_id = ? AND contexts.type = ? AND DATE(contexts.created_at) = ?",
			sessionID, contextType, date).
		First(&context)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &context, nil
}

// FindTodayKeyloggerContext finds today's keylogger context for a session
func FindTodayKeyloggerContext(sessionID string) (*models.Context, error) {
	today := time.Now().Format("2006-01-02")
	return FindContextBySessionAndTypeAndDate(sessionID, consts.ContextKeyLogger, today)
}

// SaveSessionModel saves a session model to database
func SaveSessionModel(session *models.Session) error {
	return Session().Save(session).Error
}
