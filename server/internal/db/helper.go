package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
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
	updateResult := Session().Exec(`
        UPDATE sessions
        SET is_alive = false
        WHERE last_checkin < strftime('%s', 'now') - (interval * 2) 
        AND is_removed = false
    `)

	if updateResult.Error != nil {
		logs.Log.Infof("Failed to update inactive sessions: %v", updateResult.Error)
		return nil, updateResult.Error
	}

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

// Basic Session OP
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

func UpdateSessionTimer(sessionID string, interval uint64, jitter float64) error {
	var session models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	if interval != 0 {
		session.Interval = interval
	}
	if jitter != 0 {
		session.Jitter = jitter
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

func GetTaskDescriptionByID(taskID string) (*models.File, *models.FileDescription, error) {
	var task models.File
	if err := Session().Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, nil, err
	}

	var td models.FileDescription
	if err := json.Unmarshal([]byte(task.Description), &td); err != nil {
		return nil, nil, err
	}

	return &task, &td, nil
}

// File
func GetFilesBySessionID(sessionID string) ([]models.File, error) {
	var files []models.File
	result := Session().Where("session_id = ?", sessionID).Find(&files)
	if result.Error != nil {
		return nil, result.Error
	}
	return files, nil
}

func GetAllDownloadFiles() (files []models.File, err error) {
	result := Session().Where("type = ?", "download").Find(&files)
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

func FindPipeline(name string) (*models.Pipeline, error) {
	var pipeline *models.Pipeline
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

func SavePipeline(pipeline *models.Pipeline) (*models.Pipeline, error) {
	newPipeline := &models.Pipeline{}
	result := Session().Where("name = ? AND listener_id  = ?", pipeline.Name, pipeline.ListenerID).First(&newPipeline)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			err := Session().Create(&pipeline).Error
			if err != nil {
				return nil, err
			}
			return pipeline, nil
		}
		return nil, result.Error
	}
	pipeline.ID = newPipeline.ID
	if pipeline.IP == "" {
		pipeline.IP = newPipeline.IP
	}
	err := Session().Save(&pipeline).Error
	return pipeline, err
}

func DeletePipeline(name string) error {
	result := Session().Where("name = ?", name).Delete(&models.Pipeline{})
	return result.Error
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

func DeleteWebsite(name string) error {
	website := models.WebsiteContent{}
	result := Session().Where("pipeline_id = ?", name).First(&website)
	if result.Error != nil {
		return result.Error
	}
	err := os.Remove(filepath.Join(configs.WebsitePath, website.ID.String()))
	if err != nil {
		return err
	}
	result = Session().Delete(&website)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func ListWebsite(listenerID string) ([]*models.Pipeline, error) {
	var pipelines []*models.Pipeline
	//err := Session().Where("listener_id = ? AND type = ?", listenerID, consts.WebsitePipeline).Find(&pipelines).Error
	var err error
	if listenerID == "" {
		err = Session().Where(" type = ?", consts.WebsitePipeline).Find(&pipelines).Error
	} else {
		err = Session().Where("listener_id = ? AND type = ?", listenerID, consts.WebsitePipeline).Find(&pipelines).Error
	}
	return pipelines, err
}

func EnablePipeline(pid string) error {
	pipeline, err := FindPipeline(pid)
	if err != nil {
		return err
	}
	pipeline.Enable = true
	result := Session().Save(&pipeline)
	return result.Error
}

func DisablePipeline(pid string) error {
	pipeline, err := FindPipeline(pid)
	if err != nil {
		return err
	}
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
		Cur:         int(taskpb.Total),
		Total:       int(taskpb.Total),
		Description: tdString,
	}
	Session().Create(fileModel)
	return nil
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

func GetTasksByID(sessionID string) (*clientpb.Tasks, error) {
	var tasks []models.Task
	err := Session().Where("session_id = ?", sessionID).Find(&tasks).Error
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

func UpdateTaskCur(cur int, taskID string) error {
	taskModel := &models.Task{
		ID: taskID,
	}
	return taskModel.UpdateCur(Session(), cur)
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

// FindWebsiteByName - Get website by name
func FindWebsiteByName(name string) (*models.Pipeline, error) {
	var website *models.Pipeline
	if err := Session().Where("name = ? AND type = 'website'", name).First(&website).Error; err != nil {
		return nil, err
	}
	return website, nil
}

// WebContent by ID and path
func FindWebContent(id string) (*models.WebsiteContent, error) {
	uuidFromString, err := uuid.FromString(id)
	if err != nil {
		return nil, err
	}
	contents := &models.WebsiteContent{}
	err = Session().Where(&models.WebsiteContent{
		ID: uuidFromString,
	}).First(&contents).Error
	if err != nil {
		return nil, err
	}
	return contents, err
}

func FindWebContentsByWebsite(website string) ([]*models.WebsiteContent, error) {
	var contents []*models.WebsiteContent
	var err error
	if website == "" {
		err = Session().Preload("Pipeline").Find(&contents).Error
	} else {
		err = Session().Where(&models.WebsiteContent{
			PipelineID: website,
		}).Preload("Pipeline").Find(&contents).Error
	}
	if err != nil {
		return nil, err
	}

	return contents, err
}

// AddWebsite - Return website, create if it does not exist
//func AddWebsite(webSiteName string) (*clientpb.WebContent, error) {
//	pbWebSite, err := FindWebsiteByName(webSiteName)
//	if errors.Is(err, gorm.ErrRecordNotFound) {
//		err = Session().Create(&models.WebsiteContent{
//			File: webSiteName,
//		}).Error
//		if err != nil {
//			return nil, err
//		}
//		pbWebSite, err = FindWebsiteByName(webSiteName)
//		if err != nil {
//			return nil, err
//		}
//	}
//	return pbWebSite, nil
//}

// AddContent - Add content to website
func AddContent(content *clientpb.WebContent) (*models.WebsiteContent, error) {
	var existingContent models.WebsiteContent
	switch content.Type {
	case "", "raw":
		content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
	case consts.ImplantPulse:
		content.ContentType = "application/octet-stream"
	default:
		content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
	}

	webModel := models.FromWebContentPb(content)
	err := Session().Where("pipeline_id = ? AND path = ?", content.WebsiteId, content.Path).First(&existingContent).Error
	if err == nil {
		webModel.ID = existingContent.ID
		err = Session().Save(&webModel).Error
		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		err = Session().Create(&webModel).Error
		if err != nil {
			return nil, err
		}
	}
	if content.Type == "raw" {
		err = os.WriteFile(filepath.Join(configs.WebsitePath, content.WebsiteId, webModel.ID.String()), content.Content, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	content.Id = webModel.ID.String()
	return webModel, nil
}

// RemoveContent - Remove content by ID
func RemoveContent(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.WebsiteContent{}, uuid).Error
	return err
}

// generator
func NewProfile(profile *clientpb.Profile) error {
	if profile.Content == nil {
		profile.Content = types.DefaultProfile
	}
	model := &models.Profile{
		Name: profile.Name,
		Type: profile.Type,
		//Obfuscate:  profile.Obfuscate,
		Modules:         profile.Modules,
		CA:              profile.Ca,
		ParamsData:      profile.Params,
		PulsePipelineID: profile.PulsePipelineId,
		PipelineID:      profile.PipelineId,
		Raw:             profile.Content,
	}
	basicPipeline, err := FindPipeline(profile.PipelineId)
	if err != nil {
		return err
	}
	if strings.ToUpper(basicPipeline.Type) != consts.CryptorAES {
		return errs.ErrInvalidEncType
	}
	if profile.PulsePipelineId != "" {
		pulsePipeline, err := FindPipeline(profile.PulsePipelineId)
		if err != nil {
			return err
		}
		if strings.ToUpper(pulsePipeline.Encryption.Type) != consts.CryptorXOR {
			return errs.ErrInvalidEncType
		}
	}
	return Session().Create(model).Error
}

func GetProfile(name string) (*types.ProfileConfig, error) {
	var profileModel *models.Profile

	result := Session().Preload("Pipeline").Preload("PulsePipeline").Where("name = ?", name).First(&profileModel)
	if result.Error != nil {
		return nil, result.Error
	}
	if profileModel.PipelineID != "" && profileModel.Pipeline == nil {
		return nil, errs.ErrNotFoundPipeline
	}
	if profileModel.PulsePipelineID != "" && profileModel.PulsePipeline == nil {
		return nil, errs.ErrNotFoundPipeline
	}
	err := profileModel.DeserializeImplantConfig()
	if err != nil {
		return nil, err
	}
	profile, err := types.LoadProfile(profileModel.Raw)
	if err != nil {
		return nil, err
	}
	if profile.Basic != nil {
		if profileModel.CA != "" {
			profile.Basic.CA = profileModel.CA
		}
		if profileModel.Name != "" {
			profile.Basic.Name = profileModel.Name
		}
		if profileModel.Modules != "" {
			profile.Implant.Modules = strings.Split(profileModel.Modules, ",")
		}
		if profileModel.Params != nil {
			profile.Basic.Interval = profileModel.Params.Interval
			profile.Basic.Jitter = profileModel.Params.Jitter
		}
		if profileModel.Pipeline != nil {
			profile.Basic.Targets = []string{profileModel.Pipeline.Address()}
		}
	}
	if profile.Pulse != nil {
		if profileModel.PulsePipeline != nil {
			profile.Pulse.Target = profileModel.PulsePipeline.Address()
		}
	}

	return profile, nil
}

func GetProfiles() ([]models.Profile, error) {
	var profiles []models.Profile
	result := Session().Find(&profiles)
	return profiles, result.Error
}

func DeleteProfileByName(profileName string) error {
	err := Session().Where("name = ?", profileName).Delete(&models.Profile{}).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateProfileRaw(profileName string, raw []byte) error {
	return Session().Model(&models.Profile{}).Where("name = ?", profileName).Update("raw", raw).Error
}

func SaveBuilderFromAction(inputs map[string]string, req *clientpb.Generate) (*models.Builder, error) {
	target, ok := consts.GetBuildTarget(inputs["targets"])
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	builder := models.Builder{
		Name:        codenames.GetCodename(),
		ProfileName: req.ProfileName,
		Target:      target.Name,
		Type:        inputs["package"],
		Source:      consts.ArtifactFromAction,
		Arch:        target.Arch,
		IsSRDI:      req.Srdi,
		Modules:     strings.Join(req.Modules, ","),
		Os:          target.OS,
		CA:          req.Ca,
	}

	if err := Session().Create(&builder).Error; err != nil {
		return nil, err
	}

	return &builder, nil
}

func SaveArtifactFromGenerate(req *clientpb.Generate) (*models.Builder, error) {
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
		Source:      consts.ArtifactFromDocker,
		CA:          req.Ca,
		IsSRDI:      req.Srdi,
		Modules:     strings.Join(req.Modules, ""),
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

func SaveArtifactFromID(req *clientpb.Generate, ID uint32, resource string) (*models.Builder, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	builder := models.Builder{
		ID:          ID,
		Name:        req.Name,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.Type,
		Stager:      req.Stager,
		Source:      resource,
		IsSRDI:      req.Srdi,
		CA:          req.Ca,
		Modules:     req.Feature,
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

func UpdateBuilderPath(builder *models.Builder) error {
	return Session().Model(builder).
		Select("path").
		Updates(builder).
		Error
}

func UpdateBuilderSrdi(builder *models.Builder) error {
	return Session().Model(builder).
		Select("is_srdi", "shellcode_path").
		Updates(builder).
		Error
}

func SaveArtifact(name, artifactType, platform, arch, stage, source string) (*models.Builder, error) {
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, err
	}

	builder := models.Builder{
		Name:   name,
		Os:     platform,
		Arch:   arch,
		Stager: stage,
		Type:   artifactType,
		Source: source,
	}
	if artifactType == consts.CommandBuildShellCode {
		builder.IsSRDI = true
		builder.ShellcodePath = filepath.Join(absBuildOutputPath, encoders.UUID())
	} else {
		builder.Path = filepath.Join(absBuildOutputPath, encoders.UUID())
	}

	if err := Session().Create(&builder).Error; err != nil {
		return nil, err
	}
	return &builder, nil
}

func GetBuilders() (*clientpb.Builders, error) {
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

// FindArtifact
func FindArtifact(target *clientpb.Artifact) (*clientpb.Artifact, error) {
	var builder *models.Builder
	var result *gorm.DB
	// 根据 ID 或名称查找构建器
	if target.Id != 0 {
		result = Session().Where("id = ?", target.Id).First(&builder)
	} else if target.Name != "" {
		result = Session().Where("name = ?", target.Name).First(&builder)
	} else {
		var builders []*models.Builder
		result = Session().Where("os = ? AND arch = ? AND type = ?", target.Platform, target.Arch, target.Type).
			Preload("Profile.Pipeline").
			Preload("Profile.PulsePipeline").
			Find(&builders)
		for _, v := range builders {
			if v.Type == consts.ImplantPulse && v.Profile.PulsePipelineID == target.Pipeline {
				builder = v
				break
			}
			if v.Profile.PipelineID == target.Pipeline {
				builder = v
				break
			}
		}
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error finding artifact: %v, target: %+v", result.Error, target)
	}
	if builder == nil {
		return nil, errs.ErrNotFoundArtifact
	}

	var content []byte
	var err error
	if target.IsSrdi {
		if builder.ShellcodePath != "" {
			content, err = os.ReadFile(builder.ShellcodePath)
		}
	} else {
		content, err = os.ReadFile(builder.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading file for builder: %s, error: %v", builder.Name, err)
	}

	return builder.ToArtifact(content), nil
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

func DeleteArtifactByName(artifactName string) error {
	model := &models.Builder{}
	err := Session().Where("name = ?", artifactName).First(&model).Error
	if err != nil {
		return err
	}
	if model.Path != "" {
		err = os.Remove(model.Path)
		if err != nil {
			return err
		}
	}
	if model.ShellcodePath != "" {
		err = os.Remove(model.ShellcodePath)
		if err != nil {
			return err
		}
	}
	err = Session().Delete(model).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateGeneratorConfig - Update the generator config
func UpdateGeneratorConfig(req *clientpb.Generate, path string, config *types.ProfileConfig) (string, error) {
	if config.Basic != nil {
		if req.Name != "" {
			config.Basic.Name = req.Name
		}
		if req.Address != "" {
			config.Basic.Targets = []string{req.Address}
		}
		var params *types.ProfileParams
		if req.Params != "" {
			err := json.Unmarshal([]byte(req.Params), &params)
			if err != nil {
				return "", err
			}
			if params.Interval != -1 {
				config.Basic.Interval = params.Interval
			}

			if params.Jitter != -1 {
				config.Basic.Jitter = params.Jitter
			}
		}
		if req.Ca != "" {
			config.Basic.CA = req.Ca
		}

		if len(req.Modules) > 0 {
			config.Implant.Modules = req.Modules
		}

	} else if config.Pulse != nil {
		if req.Address != "" {
			config.Pulse.Target = req.Address
		}
	}
	if req.ArtifactId != 0 && config.Pulse.Extras["flags"].(map[string]interface{})["artifact_id"].(int) == 0 {
		config.Pulse.Extras["flags"].(map[string]interface{})["artifact_id"] = req.ArtifactId
	}

	if req.Type == consts.CommandBuildBind {
		config.Implant.Mod = consts.CommandBuildBind
	}
	newData, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(newData), os.WriteFile(path, newData, 0644)
}

func UpdateBuilderLog(name string, logEntry string) {
	err := Session().Model(&models.Builder{}).
		Where("name = ?", name).
		Update("log", gorm.Expr("ifnull(log, '') || ?", logEntry)).
		Error

	if err != nil {
		logs.Log.Errorf("Error updating log for Builder name %s: %v", name, err)
	}
}

func GetBuilderLogs(builderID uint32, limit int) (string, error) {
	var builder models.Builder
	if err := Session().Where("id = ?", builderID).First(&builder).Error; err != nil {
		return "", err
	}

	split := strings.Split(builder.Log, "\n")

	if limit > 0 && len(split) > limit {
		split = split[len(split)-limit:]
	}
	result := strings.Join(split, "\n")

	return result, nil
}

func GetBuilderByModules(target string, modules []string) (*models.Builder, error) {
	sort.Strings(modules)
	modulesStr := strings.Join(modules, ",")
	var builder models.Builder
	result := Session().Where("target = ? AND modules = ?", target, modulesStr).First(&builder)
	if result.Error != nil {
		return nil, result.Error
	}
	return &builder, nil
}

func GetBuilderByProfileName(profileName string) (*clientpb.Builders, error) {
	var builders []models.Builder
	result := Session().Where("profile_name = ?", profileName).Find(&builders)
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

// context
func GetAllContext() ([]*models.Context, error) {
	var contexts []*models.Context
	result := Session().Find(&contexts)
	if result.Error != nil {
		return nil, result.Error
	}
	return contexts, nil
}

func CreateContext(ctx *models.Context) error {
	return Session().Create(ctx).Error
}
