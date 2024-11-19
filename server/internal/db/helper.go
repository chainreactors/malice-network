package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
	"os"
	"path/filepath"
	"strconv"
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

func FindAliveSessions() ([]*clientpb.Session, error) {
	var activeSessions []models.Session
	result := Session().Raw(`
		SELECT * 
		FROM sessions 
		WHERE last_checkin > strftime('%s', 'now') - (interval * 2) AND is_removed = false
		`).Scan(&activeSessions)
	if result.Error != nil {
		return nil, result.Error
	}
	var sessions []*clientpb.Session
	for _, session := range activeSessions {
		sessions = append(sessions, session.ToProtobuf())
	}
	return sessions, nil
}

func FindSession(sessionID string) (*clientpb.Session, error) {
	var session models.Session
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
	return session.ToProtobuf(), nil
}

func FindAllSessions() (*clientpb.Sessions, error) {
	var sessions []models.Session
	result := Session().Order("group_name").Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	var pbSessions []*clientpb.Session
	for _, session := range sessions {
		if session.IsRemoved {
			continue
		}
		pbSessions = append(pbSessions, session.ToProtobuf())
	}
	return &clientpb.Sessions{Sessions: pbSessions}, nil
}

func FindTaskAndMaxTasksID(sessionID string) ([]*models.Task, uint32, error) {
	var tasks []*models.Task

	err := Session().Where("session_id = ?", sessionID).Find(&tasks).Error
	if err != nil {
		return tasks, 0, err
	}

	var max int
	for _, task := range tasks {
		if task.Seq > max {
			max = task.Seq
		}
	}

	return tasks, uint32(max), nil
}

func UpdateLast(sessionID string) error {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	session.LastCheckin = time.Now().Unix()
	result = Session().Save(&session)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Basic Session OP
func DeleteSession(sessionID string) error {
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

func FindPipeline(name string) (models.Pipeline, error) {
	var pipeline models.Pipeline
	result := Session().Where("name = ?", name).First(&pipeline)
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

func CreatePipeline(pipeline *models.Pipeline) error {
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

func ListPipelines(listenerID string) ([]models.Pipeline, error) {
	var pipelines []models.Pipeline
	var err error
	if listenerID == "" {
		err = Session().Where(" type != ?", consts.WebsitePipeline).Find(&pipelines).Error
	} else {
		err = Session().Where("listener_id = ? AND type != ?", listenerID, consts.WebsitePipeline).Find(&pipelines).Error
	}
	return pipelines, err
}

func ListWebsite(listenerID string) ([]models.Pipeline, error) {
	var pipelines []models.Pipeline
	//err := Session().Where("listener_id = ? AND type = ?", listenerID, consts.WebsitePipeline).Find(&pipelines).Error
	var err error
	if listenerID == "" {
		err = Session().Where(" type = ?", consts.WebsitePipeline).Find(&pipelines).Error
	} else {
		err = Session().Where("listener_id = ? AND type = ?", listenerID, consts.WebsitePipeline).Find(&pipelines).Error
	}
	return pipelines, err
}

func EnablePipeline(pipeline models.Pipeline) error {
	pipeline.Enable = true
	result := Session().Save(&pipeline)
	return result.Error
}

func DisablePipeline(pipeline models.Pipeline) error {
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

func AddFile(typ string, taskpb *clientpb.Task, td *models.FileDescription) error {
	tdString, err := td.ToJsonString()
	if err != nil {
		return err
	}
	fileModel := &models.File{
		ID:          taskpb.SessionId + "-" + utils.ToString(taskpb.TaskId),
		Type:        typ,
		SessionID:   taskpb.SessionId,
		Cur:         int(taskpb.Cur),
		Total:       int(taskpb.Total),
		Description: tdString,
	}
	Session().Create(fileModel)
	return nil
}

func UpdateFileByID(ID string, newCur int) error {
	fileModel := &models.File{
		ID: ID,
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

func GetAllTask() (*clientpb.Tasks, error) {
	var tasks []models.Task
	err := Session().Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	pbTasks := &clientpb.Tasks{}
	for _, task := range tasks {
		pbTasks.Tasks = append(pbTasks.Tasks, task.ToProtobuf())
	}
	return pbTasks, nil
}

func AddTask(task *clientpb.Task) error {
	taskModel := &models.Task{
		ID:         task.SessionId + "-" + utils.ToString(task.TaskId),
		Seq:        int(task.TaskId),
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

func UpdateDownloadTotal(task *clientpb.Task, total int) error {
	taskModel := &models.Task{
		ID: task.SessionId + "-" + utils.ToString(task.TaskId),
	}
	return taskModel.UpdateTotal(Session(), total)
}

func UpdateTaskDescription(taskID, Description string) error {
	return Session().Model(&models.Task{}).Where("id = ?", taskID).Update("description", Description).Error
}

// WebsiteByName - Get website by name
func WebsiteByName(name string, webContentDir string) (*clientpb.Website, error) {
	var websiteContent models.WebsiteContent
	if err := Session().Where("name = ?", name).First(&websiteContent).Error; err != nil {
		return nil, err
	}
	return websiteContent.ToProtobuf(webContentDir), nil
}

// Websites - Return all websites
func Websites(webContentDir string) ([]*clientpb.Website, error) {
	var websiteContents []*models.WebsiteContent
	err := Session().Find(&websiteContents).Error

	var pbWebsites []*clientpb.Website
	for _, websiteContent := range websiteContents {
		pbWebsites = append(pbWebsites, websiteContent.ToProtobuf(webContentDir))
	}

	return pbWebsites, err
}

func WebsitesAllByname(name, webContentDir string) ([]*clientpb.Website, error) {
	var websiteContent []models.WebsiteContent
	if err := Session().Where("name = ?", name).Find(&websiteContent).Error; err != nil {
		return nil, err
	}
	var pbWebsites []*clientpb.Website
	for _, website := range websiteContent {
		pbWebsites = append(pbWebsites, website.ToProtobuf(webContentDir))
	}
	return pbWebsites, nil
}

// WebContent by ID and path
func WebContentByIDAndPath(id string, path string, webContentDir string, eager bool) (*clientpb.WebContent, error) {
	uuidFromString, _ := uuid.FromString(id)
	content := models.WebsiteContent{}
	err := Session().Where(&models.WebsiteContent{
		ID:   uuidFromString,
		Path: path,
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
func AddWebsite(webSiteName string, webContentDir string) (*clientpb.Website, error) {
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
func AddContent(pbWebContent *clientpb.WebContent, webContentDir string) (*clientpb.WebContent, error) {
	var existingContent models.WebsiteContent
	dbModelWebContent := models.WebsiteContentFromProtobuf(pbWebContent)
	err := Session().Where("name = ? AND path = ?", pbWebContent.Name, pbWebContent.Path).First(&existingContent).Error
	if err == nil {
		dbModelWebContent.ID = existingContent.ID
		err = Session().Save(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		err = Session().Create(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
	}
	pbWebContent.Id = dbModelWebContent.ID.String()
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

func SaveArtifactFromGenerate(req *clientpb.Generate, realName, path string) (*models.Builder, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	builder := models.Builder{
		Name:        req.Name,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.Type,
		Stager:      req.Stager,
		CA:          req.Ca,
		Modules:     req.Feature,
		Path:        path,
		Arch:        target.Arch,
		Os:          target.OS,
	}

	paramsJson, err := json.Marshal(req.Params)
	if err != nil {
		return nil, err
	}
	builder.ParamsJson = string(paramsJson)

	if err := Session().Create(&builder).Error; err != nil {
		return nil, err

	}

	return &builder, nil
}

func SaveArtifact(name, artifactType, platform, arch, stage string) (*models.Builder, error) {
	absBuildOutputPath, err := filepath.Abs(configs.TempPath)
	if err != nil {
		return nil, err
	}
	builder := models.Builder{
		Name:   name,
		Os:     platform,
		Arch:   arch,
		Stager: stage,
		Type:   artifactType,
		Path:   filepath.Join(absBuildOutputPath, encoders.UUID()),
	}
	if err := Session().Create(&builder).Error; err != nil {
		return nil, err
	}
	return &builder, nil
}

func GetArtifacts() (*clientpb.Builders, error) {
	var builders []models.Builder
	result := Session().Preload("Profile").Find(&builders)
	if result.Error != nil {
		return nil, result.Error

	}
	var pbBuilders = &clientpb.Builders{
		Builders: make([]*clientpb.Builder, 0),
	}
	for _, builder := range builders {
		pbBuilders.Builders = append(pbBuilders.GetBuilders(), builder.ToProtobuf())
	}
	return pbBuilders, nil
}

func GetArtifactByName(name string) (*models.Builder, error) {
	var builder models.Builder
	result := Session().Where("name = ?", name).First(&builder)
	if result.Error != nil {
		return nil, result.Error
	}
	return &builder, nil
}

func GetArtifactById(id uint32) (*models.Builder, error) {
	var builder models.Builder
	result := Session().Where("id = ?", id).First(&builder)
	if result.Error != nil {
		return nil, result.Error
	}
	return &builder, nil
}

// UpdateGeneratorConfig - Update the generator config
func UpdateGeneratorConfig(req *clientpb.Generate, path string, profile models.Profile) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var config *configs.GeneratorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	if config.Basic != nil {
		if profile.Name != "" {
			config.Basic.Name = profile.Name
		}
		if req.Address != "" {
			config.Basic.Targets = []string{}
			config.Basic.Targets = append(config.Basic.Targets, req.Address)
		} else if profile.Name != "" {
			config.Basic.Targets = []string{}
			config.Basic.Targets = append(config.Basic.Targets,
				fmt.Sprintf("%s:%v", profile.Pipeline.Host, profile.Pipeline.Port))
		}
		var dbParams *models.Params
		err = profile.DeserializeImplantConfig(dbParams)
		if err != nil {
			return err
		}
		if val, ok := req.Params["interval"]; ok && val != "" {
			interval, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			config.Basic.Interval = interval
		} else if profile.Name != "" {
			dbInterval, err := strconv.Atoi(dbParams.Interval)
			if err != nil {
				return err
			}
			config.Basic.Interval = dbInterval
		}

		if val, ok := req.Params["jitter"]; ok && val != "" {
			jitter, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			config.Basic.Jitter = jitter
		} else if profile.Name != "" {
			dbJitter, err := strconv.ParseFloat(dbParams.Jitter, 64)
			if err != nil {
				return err
			}
			config.Basic.Jitter = dbJitter
		}

		if val, ok := req.Params["ca"]; ok {
			config.Basic.CA = val
		} else if profile.Pipeline.Tls.Enable {
			config.Basic.CA = profile.Pipeline.Tls.Cert
		}

		//var modules []string
		//if len(req.Modules) > 0 {
		//	modules = req.Modules
		//}else if profile.Name != ""{
		//	modules = strings.Split(profile.Modules, ",")
		//}
		//config.Basic.Modules = modules

	} else if config.Pulse != nil {
		if profile.Name != "" {
			config.Pulse.Target = req.Target
		}
	}

	newData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, newData, 0644)
}
