package db

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
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

func FindAliveSessions() ([]*lispb.RegisterSession, error) {
	var activeSessions []models.Session
	result := Session().Raw(`
		SELECT * 
		FROM sessions 
		WHERE last_checkin > strftime('%s', 'now') - (interval * 2)
		`).Scan(&activeSessions)
	if result.Error != nil {
		return nil, result.Error
	}
	var sessions []*lispb.RegisterSession
	for _, session := range activeSessions {
		if session.IsRemoved {
			continue
		}
		sessions = append(sessions, session.ToRegisterProtobuf())
	}
	return sessions, nil
}

func FindSession(sessionID string) (*lispb.RegisterSession, error) {
	var session models.Session
	result := Session().Where("session_id = ? AND is_removed = ?", sessionID, false).First(&session)
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
	var tasks []*models.Task

	err := Session().Where("session_id = ?", sessionID).Find(&tasks).Error
	if err != nil {
		return tasks, 0, err
	}

	maxTemp := 0
	for _, task := range tasks {
		parts := strings.Split(task.ID, "-")
		taskID, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		maxTemp = taskID
	}

	return tasks, maxTemp, nil
}

func UpdateLast(sessionID string) error {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	session.Time.LastCheckin = uint64(time.Now().Unix())
	result = Session().Save(&session)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func UpdateSessionStatus() error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var sessions []models.Session
		if err := Session().Find(&sessions).Error; err != nil {
			return err
		}
		for _, session := range sessions {
			lastCheckin := time.Unix(int64(session.Time.LastCheckin), 0)
			currentTime := time.Now()
			timeDiff := currentTime.Sub(lastCheckin)
			isAlive := timeDiff <= time.Duration(session.Time.Interval+session.Time.Jitter)*2*time.Second
			if err := Session().Model(&session).Update("IsAlive", isAlive).Error; err != nil {
				return err
			}
		}
		//for _, session := range core.Sessions.All() {
		//	currentTime := time.Now()
		//	timeDiff := currentTime.Sub(time.Unix(int64(session.Timer.LastCheckin), 0))
		//	isAlive := timeDiff <= time.Duration(session.Timer.Interval)*time.Second
		//	if !isAlive {
		//		err := core.Notifier.Send(&core.Event{
		//			EventType: consts.EventSession,
		//			Op:        consts.CtrlSessionStop,
		//			Message: fmt.Sprintf("session %s from %s at %s stop",
		//				session.ID, session.PipelineID, session.RemoteAddr),
		//		})
		//		if err != nil {
		//			return err
		//		}
		//	}
		//}
	}
	return nil
}

func UpdateSessionInfo(coreSession *core.Session) error {
	updateSession := models.ConvertToSessionDB(coreSession)
	updateSession.IsAlive = true
	result := Session().Save(updateSession)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Basic Session OP
func DeleteSession(sessionID string) error {
	result := Session().Where("session_id = ?", sessionID).Update("is_removed", true)
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

func CreateOperator(name string, typ string, remoteAddr string) error {
	var operator models.Operator
	operator.Name = name
	operator.Type = typ
	operator.Remote = remoteAddr
	err := Session().Save(&operator).Error
	return err

}

func ListClients() ([]models.Operator, error) {
	var operators []models.Operator
	err := Session().Find(&operators).Where("type = ?", mtls.Client).Error
	if err != nil {
		return nil, err
	}

	return operators, nil
}

func GetTaskDescriptionByID(taskID string) (*models.FileDescription, error) {
	var task models.File
	if err := Session().Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}

	var td models.FileDescription
	if err := json.Unmarshal([]byte(task.Description), &td); err != nil {
		return nil, err
	}

	return &td, nil
}

// File
func GetAllFiles(sessionID string) ([]models.File, error) {
	var files []models.File
	result := Session().Where("session_id = ?", sessionID).Find(&files)
	if result.Error != nil {
		return nil, result.Error
	}
	return files, nil
}

func FindFilesWithNonOneCurTotal(session models.Session) ([]models.File, error) {
	var files []models.File
	result := Session().Where("session_id = ?", session.SessionID).Where("cur != total").Find(&files)
	if result.Error != nil {
		return files, result.Error
	}
	if len(files) == 0 {
		return files, gorm.ErrRecordNotFound
	}
	return files, nil
}

func FindPipeline(name, listenerID string) (models.Pipeline, error) {
	var pipeline models.Pipeline
	result := Session().Where("name = ? AND listener_id = ?", name, listenerID).First(&pipeline)
	if result.Error != nil {
		return pipeline, result.Error
	}
	pipeline.Enable = true
	result = Session().Save(&pipeline)
	if result.Error != nil {
		return pipeline, result.Error
	}
	return pipeline, nil

}

func CreatePipeline(ppProto *lispb.Pipeline) error {
	pipeline := models.ProtoBufToDB(ppProto)
	newPipeline := models.Pipeline{}
	result := Session().Where("name = ? AND listener_id  = ?", pipeline.Name, pipeline.ListenerID).First(&newPipeline)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			err := Session().Create(&pipeline).Error
			if err != nil {
				return err
			}
			return nil
		}
		return result.Error
	}
	pipeline.ID = newPipeline.ID
	err := Session().Save(&pipeline).Error
	if err != nil {
		return err
	}
	return nil
}

func ListPipelines(listenerID string, pipelineType string) ([]models.Pipeline, error) {
	var pipelines []models.Pipeline
	err := Session().Where("listener_id = ? AND type = ?", listenerID, pipelineType).Find(&pipelines).Error
	return pipelines, err
}

func EnablePipeline(pipeline models.Pipeline) error {
	pipeline.Enable = true
	result := Session().Save(&pipeline)
	return result.Error
}

func UnEnablePipeline(pipeline models.Pipeline) error {
	pipeline.Enable = false
	result := Session().Save(&pipeline)
	return result.Error
}

func FindPipelineCert(pipelineName, listenerID string) (string, string, error) {
	var pipeline models.Pipeline
	result := Session().Where("name = ? AND listener_id = ?", pipelineName, listenerID).First(&pipeline)
	if result.Error != nil {
		return "", "", result.Error
	}
	return pipeline.Tls.Cert, pipeline.Tls.Key, nil
}

func ListListeners() ([]models.Operator, error) {
	var listeners []models.Operator
	err := Session().Find(&listeners).Where("type = ?", mtls.Listener).Error
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

func AddFile(typ string, task *core.Task, td *models.FileDescription) error {
	tdString, err := td.ToJsonString()
	if err != nil {
		return err
	}
	fileModel := &models.File{
		ID:          task.SessionId + "-" + utils.ToString(task.Id),
		Type:        typ,
		SessionID:   task.SessionId,
		Cur:         task.Cur,
		Total:       task.Total,
		Description: tdString,
	}
	Session().Create(fileModel)
	return nil
}

func UpdateFile(task *core.Task, newCur int) error {
	fileModel := &models.File{
		ID: task.SessionId + "-" + utils.ToString(task.Id),
	}
	return fileModel.UpdateCur(Session(), newCur)
}

func GetTaskPB(taskID string) (*clientpb.Task, error) {
	var task models.Task
	err := Session().Where("id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	taskProto := task.ToProtobuf()
	return taskProto, nil
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

func AddTask(task *core.Task) error {

	taskModel := &models.Task{
		ID:        task.SessionId + "-" + utils.ToString(task.Id),
		Type:      task.Type,
		SessionID: task.SessionId,
		Cur:       task.Cur,
		Total:     task.Total,
	}
	return Session().Create(taskModel).Error
}

func UpdateTask(task *core.Task) error {
	taskModel := &models.Task{
		ID: task.SessionId + "-" + utils.ToString(task.Id),
	}
	return taskModel.UpdateCur(Session(), task.Total)
}

func UpdateDownloadTotal(task *core.Task, total int) error {
	taskModel := &models.Task{
		ID: task.SessionId + "-" + utils.ToString(task.Id),
	}
	return taskModel.UpdateTotal(Session(), total)
}

func UpdateTaskDescription(taskID, Description string) error {
	return Session().Model(&models.Task{}).Where("id = ?", taskID).Update("description", Description).Error
}

// WebsiteByName - Get website by name
func WebsiteByName(name string, webContentDir string) (*lispb.Website, error) {
	var websiteContent models.WebsiteContent
	if err := Session().Where("name = ?", name).First(&websiteContent).Error; err != nil {
		return nil, err
	}
	return websiteContent.ToProtobuf(webContentDir), nil
}

// Websites - Return all websites
func Websites(webContentDir string) ([]*lispb.Website, error) {
	var websiteContents []*models.WebsiteContent
	err := Session().Find(&websiteContents).Error

	var pbWebsites []*lispb.Website
	for _, websiteContent := range websiteContents {
		pbWebsites = append(pbWebsites, websiteContent.ToProtobuf(webContentDir))
	}

	return pbWebsites, err
}

// WebContent by ID and path
func WebContentByIDAndPath(id string, path string, webContentDir string, eager bool) (*lispb.WebContent, error) {
	uuidFromString, _ := uuid.FromString(id)
	content := models.WebsiteContent{}
	err := Session().Where(&models.WebsiteContent{
		ID:   uuidFromString,
		Name: path,
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
	result := content.ToProtobuf(webContentDir).Contents[content.ID.String()]
	result.Content = data
	return result, err
}

// AddWebsite - Return website, create if it does not exist
func AddWebsite(webSiteName string, webContentDir string) (*lispb.Website, error) {
	pbWebSite, err := WebsiteByName(webSiteName, webContentDir)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = Session().Create(&models.WebsiteContent{Name: webSiteName}).Error
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
	dbModelWebContent := models.WebsiteContentFromProtobuf(pbWebContent)
	err := Session().Save(&dbModelWebContent).Error
	if err != nil {
		return nil, err
	}
	pbWebContent.ID = dbModelWebContent.ID.String()
	return pbWebContent, nil
}

// RemoveWebsiteContent - Remove all content of a website by ID
func RemoveWebsiteContent(id string) error {
	uuid, _ := uuid.FromString(id)
	if err := Session().Where("id = ?", uuid).Delete(&models.WebsiteContent{}).Error; err != nil {
		return err
	}
	return nil
}

// RemoveContent - Remove content by ID
func RemoveContent(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.WebsiteContent{}, uuid).Error
	return err
}

// RemoveWebsite - Remove website by ID
func RemoveWebsite(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.WebsiteContent{}, uuid).Error
	return err
}

// generator
func NewProfile(profile *clientpb.Profile) error {
	model := &models.Profile{
		Name:       profile.Name,
		Target:     profile.Target,
		Type:       profile.Type,
		Proxy:      profile.Proxy,
		Obfuscate:  profile.Obfuscate,
		Modules:    profile.Modules,
		CA:         profile.Ca,
		ParamsJson: profile.Params,
		ListenerID: profile.ListenerId,
		PipelineID: profile.PipelineId,
	}
	return Session().Create(model).Error
}

func GetProfile(name string) (models.Profile, error) {
	var profile models.Profile

	result := Session().Preload("Pipeline").Where("name = ?", name).First(&profile)
	if result.Error != nil {
		return profile, result.Error
	}
	return profile, nil
}

func GetProfiles() ([]models.Profile, error) {
	var profiles []models.Profile
	result := Session().Find(&profiles)
	return profiles, result.Error
}
