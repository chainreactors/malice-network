package db

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
	"strconv"
	"strings"
	"time"
)

func FindAliveSessions() ([]*lispb.RegisterSession, error) {
	var activeSessions []models.Session
	result := Session().Raw(`
		SELECT * 
		FROM sessions 
		WHERE last > datetime('now', '-' || (interval * 2) || ' seconds')
		`).Scan(&activeSessions)
	if result.Error != nil {
		return nil, result.Error
	}
	var sessions []*lispb.RegisterSession
	for _, session := range activeSessions {
		sessions = append(sessions, session.ToProtobuf())
	}
	return sessions, nil
}

func FindSession(sessionID string) (*lispb.RegisterSession, error) {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	//if session.Last.Before(time.Now().Add(-time.Second * time.Duration(session.Time.Interval*2))) {
	//	return nil, errors.New("session is dead")
	//}
	return session.ToProtobuf(), nil
}

func FindTaskAndMaxTasksID(sessionID string) ([]*models.Task, int, error) {
	var maxTaskID int
	var tasks []*models.Task

	err := Session().Where("session_id = ?", sessionID).Find(&tasks).Error
	if err != nil {
		return tasks, 0, err
	}

	maxTemp := 0
	for _, task := range tasks {
		parts := strings.Split(task.ID, "-")
		if len(parts) != 2 {
			continue
		}
		taskID, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		if taskID > maxTemp {
			maxTemp = taskID
		}
	}

	maxTaskID = maxTemp
	return tasks, maxTaskID, nil
}

func UpdateLast(sessionID string) error {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	loc := time.Now().Location()
	if result.Error != nil {
		return result.Error
	}
	session.Last = time.Now().In(loc)
	result = Session().Save(&session)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func CreateOperator(name string) error {
	var operator models.Operator
	result := Session().Where("name = ?", name).Delete(&operator)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
	}
	operator.Name = name
	err := Session().Create(&operator).Error
	return err

}

func ListOperators() (*clientpb.Clients, error) {
	var operators []models.Operator
	err := Session().Find(&operators).Error
	if err != nil {
		return nil, err
	}

	var clients []*clientpb.Client
	for _, op := range operators {
		client := &clientpb.Client{
			Name: op.Name,
		}
		clients = append(clients, client)
	}
	pbClients := &clientpb.Clients{
		Clients: clients,
	}
	return pbClients, nil
}

func GetTaskDescriptionByID(taskID string) (*models.FileDescription, error) {
	var task models.Task
	if err := Session().Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}

	var td models.FileDescription
	if err := json.Unmarshal([]byte(task.Description), &td); err != nil {
		return nil, err
	}

	return &td, nil
}

// Task
func GetAllTasks(sessionID string) ([]models.Task, error) {
	var tasks []models.Task
	result := Session().Where("session_id = ?", sessionID).Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

func FindTasksWithNonOneCurTotal(session models.Session) ([]models.Task, error) {
	var tasks []models.Task
	result := Session().Where("session_id = ?", session.SessionID).Where("cur != total").Find(&tasks)
	if result.Error != nil {
		return tasks, result.Error
	}
	if len(tasks) == 0 {
		return tasks, gorm.ErrRecordNotFound
	}
	return tasks, nil
}

func CreateListener(name string) error {
	var listener models.Listener
	result := Session().Where("name = ?", name).Delete(&listener)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
	}
	listener.Name = name
	err := Session().Create(&listener).Error
	return err
}

func ListListeners() ([]models.Listener, error) {
	var listeners []models.Listener
	err := Session().Find(&listeners).Error
	return listeners, err
}

// DeleteAllCertificates
func DeleteAllCertificates() error {
	result := Session().Exec("DELETE FROM certificates")
	return result.Error
}

// DeleteCertificate
func DeleteCertificate(name string) error {
	var cert models.Certificate
	result := Session().Where("common_name = ?", name).First(&cert)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	result = Session().Delete(&cert)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func isDuplicateCommonNameAndCAType(commonName string, caType int) bool {
	var count int64
	Session().Model(&models.Certificate{}).Where("common_name = ? AND ca_type = ?", commonName, caType).Count(&count)
	return count > 0
}

func SaveCertificate(certificate *models.Certificate) error {
	if isDuplicateCommonNameAndCAType(certificate.CommonName, certificate.CAType) {
		return errors.New("duplicate CommonName and CAType")
	}
	if err := Session().Create(certificate).Error; err != nil {
		return err
	}

	return nil
}

func AddTask(typ string, task *core.Task, td *models.FileDescription) error {
	tdString, err := td.ToJson()
	if err != nil {
		return err
	}
	taskModel := &models.Task{
		ID:          task.SessionId + "-" + utils.ToString(task.Id),
		Type:        typ,
		SessionID:   task.SessionId,
		Cur:         task.Cur,
		Total:       task.Total,
		Description: tdString,
	}
	Session().Create(taskModel)
	return nil
}

func UpdateTask(task *core.Task, newCur int) error {
	taskModel := &models.Task{
		ID: task.SessionId + "-" + utils.ToString(task.Id),
	}
	return taskModel.UpdateCur(Session(), newCur)
}

func ToTask(task models.Task) (*core.Task, error) {
	parts := strings.Split(task.ID, "-")
	if len(parts) != 2 {
		return nil, errors.New("invalid task id")
	}
	taskID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &core.Task{
		Id:        uint32(taskID),
		Type:      task.Type,
		SessionId: task.SessionID,
		Cur:       task.Cur,
		Total:     task.Total,
	}, nil
}
