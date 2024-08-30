package db

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
	"os"
	"path/filepath"
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
		sessions = append(sessions, session.ToRegisterProtobuf())
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
	return session.ToRegisterProtobuf(), nil
}

func FindAllSessions() (*clientpb.Sessions, error) {
	var sessions []models.Session
	result := Session().Order("group_name").Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	var pbSessions []*clientpb.Session
	for _, session := range sessions {
		pbSessions = append(pbSessions, session.ToClientProtobuf())
	}
	return &clientpb.Sessions{Sessions: pbSessions}, nil

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
	session.Time.LastCheckin = uint64(session.Last.Unix())
	result = Session().Save(&session)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func UpdateSessionStatus() error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var sessions []models.Session
		if err := Session().Find(&sessions).Error; err != nil {
			return err
		}
		for _, session := range sessions {
			currentTime := time.Now()
			timeDiff := currentTime.Sub(session.Last)
			isAlive := timeDiff <= time.Duration(session.Time.Interval)*time.Second
			if err := Session().Model(&session).Update("IsAlive", isAlive).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func UpdateSessionInfo(coreSession *core.Session) error {
	updateSession := models.ConvertToSessionDB(coreSession)
	result := Session().Save(updateSession)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Basic Session OP
func DeleteSession(sessionID string) error {
	result := Session().Where("session_id = ?", sessionID).Delete(&models.Session{})
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

func CreateOperator(name string) error {
	var operator models.Operator
	operator.Name = name
	err := Session().Save(&operator).Error
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

// AddCertificate add a certificate to the database
func AddCertificate(caType int, keyType string, commonName string, cert []byte, key []byte) error {
	certModel := &models.Certificate{
		CommonName:     commonName,
		CAType:         caType,
		KeyType:        keyType,
		CertificatePEM: string(cert),
		PrivateKeyPEM:  string(key),
	}
	err := Session().Save(certModel).Error
	if err != nil {
		return err
	}
	return nil
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
	tdString, err := td.ToJsonString()
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

// website
// WebsiteByName - Get website by name
func WebsiteByName(name string, webContentDir string) (*lispb.Website, error) {
	var website models.Website
	if err := Session().Preload("WebContents").Where("name = ?", name).First(&website).Error; err != nil {
		return nil, err
	}
	//err := Session().Where("name = ?", name).First(&website).Error
	//if err != nil {
	//	return nil, err
	//}
	return website.ToProtobuf(webContentDir), nil
}

// Websites - Return all websites
func Websites(webContentDir string) ([]*lispb.Website, error) {
	var websites []*models.Website
	err := Session().Where(&models.Website{}).Find(&websites).Error

	var pbWebsites []*lispb.Website
	for _, website := range websites {
		pbWebsites = append(pbWebsites, website.ToProtobuf(webContentDir))
	}

	return pbWebsites, err
}

// WebContent by ID and path
func WebContentByIDAndPath(id string, path string, webContentDir string, eager bool) (*lispb.WebContent, error) {
	uuidFromString, _ := uuid.FromString(id)
	content := models.WebContent{}
	err := Session().Where(&models.WebContent{
		WebsiteID: uuidFromString,
		Path:      path,
	}).First(&content).Error

	if err != nil {
		return nil, err
	}
	var data []byte
	if eager {
		data, err = os.ReadFile(filepath.Join(webContentDir, content.ID.String()))
	} else {
		data = []byte{}
	}
	return content.ToProtobuf(&data), err
}

// AddWebsite - Return website, create if it does not exist
func AddWebSite(webSiteName string, webContentDir string) (*lispb.Website, error) {
	pbWebSite, err := WebsiteByName(webSiteName, webContentDir)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = Session().Create(&models.Website{Name: webSiteName}).Error
		if err != nil {
			return nil, err
		}
		pbWebSite, err = WebsiteByName(webSiteName, webContentDir)
		if err != nil {
			return nil, err
		}
	}
	return pbWebSite, nil
}

// AddContent - Add content to website
func AddContent(pbWebContent *lispb.WebContent, webContentDir string) (*lispb.WebContent, error) {
	dbWebContent, err := WebContentByIDAndPath(pbWebContent.WebsiteID, pbWebContent.Path, webContentDir, false)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		dbModelWebContent := models.WebContentFromProtobuf(pbWebContent)
		err = Session().Create(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
		dbWebContent, err = WebContentByIDAndPath(pbWebContent.WebsiteID, pbWebContent.Path, webContentDir, false)
		if err != nil {
			return nil, err
		}
	} else {
		dbWebContent.ContentType = pbWebContent.ContentType
		dbWebContent.Size = pbWebContent.Size

		dbModelWebContent := models.WebContentFromProtobuf(dbWebContent)
		err = Session().Save(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
	}
	return dbWebContent, nil
}

func GetWebContentIDByWebsiteID(websiteID string) ([]string, error) {
	uuid, err := uuid.FromString(websiteID)
	if err != nil {
		return nil, err
	}

	var IDs []string

	if err := Session().Model(&models.WebContent{}).Select("ID").Where("website_id = ?", uuid).Pluck("ID", &IDs).Error; err != nil {
		return nil, err
	}

	return IDs, nil
}

func RemoveWebAllContent(id string) error {
	uuid, _ := uuid.FromString(id)
	if err := Session().Where("website_id = ?", uuid).Delete(&models.WebContent{}).Error; err != nil {
		return err
	}

	return nil
}

func RemoveContent(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(models.WebContent{}, uuid).Error
	return err
}

func RemoveWebSite(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(models.Website{}, uuid).Error
	return err
}
