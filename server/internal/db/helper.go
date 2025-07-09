package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func RemoveOperator(name string) error {
	err := Session().Where(&models.Operator{
		Name: name,
	}).Delete(&models.Operator{}).Error
	return err
}

func FindAliveSessions() ([]*models.Session, error) {
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

	var activeSessions []*models.Session
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

func FindAllSessions() (*clientpb.Sessions, error) {
	var sessions []*models.Session
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

	var max uint32
	for _, task := range tasks {
		if task.Seq > max {
			max = task.Seq
		}
	}

	return tasks, max, nil
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
	var session *models.Session
	result := Session().Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return result.Error
	}
	if interval != 0 {
		session.Data.Interval = interval
	}
	if jitter != 0 {
		session.Data.Jitter = jitter
	}
	result = Session().Save(&session)
	return result.Error
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
		result = Session().Where("session_id = ?", sid).Where("type = ?", consts.ContextDownload).Find(&files)
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
		res = append(res, &clientpb.File{
			Name:      download.Name,
			Local:     download.FilePath,
			Checksum:  download.Checksum,
			Remote:    download.TargetPath,
			TaskId:    file.Task.Seq,
			SessionId: file.SessionID,
		})
	}

	return res, nil
}

//func FindFilesWithNonOneCurTotal(session models.Session) ([]models.File, error) {
//	var files []models.File
//	result := Session().Where("session_id = ?", session.SessionID).Where("cur != total").Find(&files)
//	if result.Error != nil {
//		return files, result.Error
//	}
//	if len(files) == 0 {
//		return files, gorm.ErrRecordNotFound
//	}
//	return files, nil
//}

func FindPipeline(name string) (*models.Pipeline, error) {
	var pipeline *models.Pipeline
	result := Session().Where("name = ?", name).First(&pipeline)
	if result.Error != nil {
		return pipeline, result.Error
	}
	if pipeline.CertName != "" {
		certificate, err := FindCertificate(pipeline.CertName)
		if err != nil && !errors.Is(err, ErrRecordNotFound) {
			logs.Log.Errorf("failed to find cert %s", err)
		}
		if certificate != nil {
			pipeline.Tls = types.FromTls(certificate.ToProtobuf())
		}
	}
	return pipeline, nil
}

func UpdatePipelineCert(certName string, pipeline *models.Pipeline) (*models.Pipeline, error) {
	var cert *models.Certificate
	if certName != "" {
		err := Session().Where("name = ?", certName).First(&cert).Error
		if err != nil {
			return nil, err
		}
	}

	err := Session().Model(pipeline).Select("cert_name").Update("cert_name", certName).Error
	if err != nil {
		return nil, err
	}
	pipeline.Tls = types.FromTls(cert.ToProtobuf())
	return pipeline, err
}

func SavePipeline(pipeline *models.Pipeline) (*models.Pipeline, error) {
	newPipeline := &models.Pipeline{}
	result := Session().Where("name = ? AND listener_id  = ?", pipeline.Name, pipeline.ListenerId).First(&newPipeline)
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
	pipeline.CertName = newPipeline.CertName
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

func ListPipelines(listenerID string) ([]*models.Pipeline, error) {
	var pipelines []*models.Pipeline
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

func FindPipelineCert(pipelineName, listenerID string) (*models.Certificate, error) {
	var pipeline *models.Pipeline
	result := Session().Where("name = ? AND listener_id = ?", pipelineName, listenerID).First(&pipeline)
	if result.Error != nil {
		return nil, result.Error
	}
	if pipeline.CertName != "" {
		certificate, err := FindCertificate(pipeline.CertName)
		if err != nil {
			return nil, err
		}
		return certificate, nil
	}
	return nil, nil
}

func ListListeners() ([]*models.Operator, error) {
	var listeners []*models.Operator
	err := Session().Find(&listeners).Where("type = ?", mtls.Listener).Error
	return listeners, err
}

// DeleteAllCertificates
func DeleteAllCertificates() error {
	result := Session().Exec("DELETE FROM certificates")
	return result.Error
}

// DeleteCertificate
func DeleteCertificate(name string) error {
	var cert *models.Certificate
	result := Session().Where("name = ?", name).First(&cert)
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

func FindCertificate(name string) (*models.Certificate, error) {
	var cert *models.Certificate
	result := Session().Where("name = ?", name).First(&cert)
	if result.Error != nil {
		return nil, result.Error
	}
	return cert, nil
}

func GetAllCertificates() ([]*models.Certificate, error) {
	var certificates []*models.Certificate
	err := Session().Find(&certificates).Error
	return certificates, err
}

func UpdateCert(name, cert, key string) error {
	return Session().Model(&models.Certificate{}).
		Where("name = ?", name).
		Select("cert_pem", "key_pem").
		Updates(models.Certificate{
			CertPEM: cert,
			KeyPEM:  key,
		}).Error
}

func isDuplicateCommonNameAndCAType(name string) bool {
	var count int64
	Session().Model(&models.Certificate{}).Where("name = ?", name).Count(&count)
	return count > 0
}

func SaveCertificate(certificate *models.Certificate) error {
	if isDuplicateCommonNameAndCAType(certificate.Name) {
		return errors.New("duplicate CommonName")
	}
	if err := Session().Create(certificate).Error; err != nil {
		return err
	}

	return nil
}

func SaveCertFromTLS(tls *clientpb.TLS, pipeline string) (*models.Certificate, error) {
	certModel := &models.Certificate{
		CertPEM: tls.Cert.Cert,
		KeyPEM:  tls.Cert.Key,
	}
	if tls.Acme {
		certModel.Name = tls.Domain
		certModel.Domain = tls.Domain
		certModel.Type = certs.Acme
	} else if tls.Ca.Key != "" {
		certModel.Name = codenames.GetCodename()
		certModel.Type = certs.SelfSigned
		certModel.CACertPEM = tls.Ca.Cert
		certModel.CAKeyPEM = tls.Ca.Key
	} else {
		certModel.Name = codenames.GetCodename()
		certModel.Type = certs.Imported
		certModel.CACertPEM = tls.Ca.Cert
	}
	err := SaveCertificate(certModel)
	if err != nil {
		return certModel, err
	}
	if pipeline != "" {
		findPipeline, err := FindPipeline(pipeline)
		if err != nil {
			return nil, err
		}
		_, err = UpdatePipelineCert(certModel.Name, findPipeline)
		if err != nil {
			return nil, err
		}
	}

	return certModel, nil
}

//func AddFile(typ string, taskpb *clientpb.Task, td *types.FileDescriptor) error {
//	tdString, err := td.Marshal()
//	if err != nil {
//		return err
//	}
//	fileModel := &models.File{
//		ID:          taskpb.SessionId + "-" + utils.ToString(taskpb.TaskId),
//		Type:        typ,
//		SessionID:   taskpb.SessionId,
//		Cur:         int(taskpb.Total),
//		Total:       int(taskpb.Total),
//		Description: tdString,
//	}
//	Session().Create(fileModel)
//	return nil
//}

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
	switch content.Type {
	case "", "raw", "default":
		content.Type = "raw"
		content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
	default:
		content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
	}

	var existingContent *models.WebsiteContent
	webModel := models.FromWebContentPb(content)
	err := Session().Preload("Pipeline").Where("pipeline_id = ? AND path = ?", content.WebsiteId, content.Path).First(&existingContent).Error
	if err == nil {
		webModel.ID = existingContent.ID
		err = Session().Save(&webModel).Error
		if err != nil {
			return nil, err
		}
		webModel = existingContent
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		err = Session().Create(&webModel).Error
		if err != nil {
			return nil, err
		}
		if webModel.Pipeline == nil {
			err := Session().Model(webModel).Association("Pipeline").Find(&webModel.Pipeline)
			if err != nil {
				return nil, err
			}
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

// validateProfileName validates the profile name
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("profile name too long (max 100 characters)")
	}
	return nil
}

// generator
func NewProfile(profile *clientpb.Profile) error {
	// Validate input
	if err := validateProfileName(profile.Name); err != nil {
		return err
	}

	if profile.Content == nil {
		profile.Content = types.DefaultProfile
	}

	// Check if profile name already exists
	var existingProfile models.Profile
	result := Session().Where("name = ?", profile.Name).First(&existingProfile)
	if result.Error == nil {
		// Found existing profile with same name, return friendly error message
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// If it's not "record not found" error, it's another database error
		return result.Error
	}

	model := &models.Profile{
		Name:       profile.Name,
		ParamsData: profile.Params,
		PipelineID: profile.PipelineId,
		Raw:        profile.Content,
	}
	//if profile.PulsePipelineId != "" {
	//	pulsePipeline, err := FindPipeline(profile.PulsePipelineId)
	//	if err != nil {
	//		return err
	//	}
	//	if strings.ToUpper(pulsePipeline.Encryption.Type) != consts.CryptorXOR {
	//		return errs.ErrInvalidEncType
	//	}
	//}
	return Session().Create(model).Error
}

// GetProfile recovers profile from database
func GetProfile(name string) (*types.ProfileConfig, error) {
	var profileModel *models.Profile

	result := Session().Preload("Pipeline").Where("name = ?", name).First(&profileModel)
	if result.Error != nil {
		return nil, result.Error
	}
	if profileModel.PipelineID != "" && profileModel.Pipeline == nil {
		return nil, errs.ErrNotFoundPipeline
	}
	//if profileModel.PulsePipelineID != "" && profileModel.PulsePipeline == nil {
	//	return nil, errs.ErrNotFoundPipeline
	//}
	err := profileModel.DeserializeImplantConfig()
	if err != nil {
		return nil, err
	}
	profile, err := types.LoadProfile(profileModel.Raw)
	if err != nil {
		return nil, err
	}
	if profileModel.Name != "" {
		profile.Basic.Name = profileModel.Name
	}

	if profileModel.Pipeline != nil {
		profile.Basic.Targets = []string{profileModel.Pipeline.Address()}
		profile.Basic.Encryption = profileModel.Pipeline.Encryption.Choice().Type
		profile.Basic.Key = profileModel.Pipeline.Encryption.Choice().Key
		profile.Basic.Protocol = profileModel.Pipeline.Type
		profile.Basic.TLS.Enable = profileModel.Pipeline.Tls.Enable
	}
	if params := profileModel.Params; params != nil {
		profile.Basic.Interval = profileModel.Params.Interval
		profile.Basic.Jitter = profileModel.Params.Jitter
		if params.REMPipeline != "" {
			profile.Basic.Protocol = consts.RemPipeline
			pipeline, err := FindPipeline(params.REMPipeline)
			if err != nil {
				return nil, err
			}
			profile.Basic.REM = &types.REMProfile{
				Link: pipeline.PipelineParams.Link,
			}
		}

	}
	if profile.Pulse != nil && profileModel.Pipeline != nil {
		profile.Pulse.Target = profileModel.Pipeline.Address()
		profile.Pulse.Protocol = profileModel.Pipeline.Type
	}

	return profile, nil
}

func GetProfiles() ([]*models.Profile, error) {
	var profiles []*models.Profile
	result := Session().Preload("Pipeline").Order("created_at ASC").Find(&profiles)
	return profiles, result.Error
}

// FindBuildersByPipelineID 遍历所有 builder，找到 profile.pipelineID = pipelineID 的 builder
func FindBuildersByPipelineID(pipelineID string) ([]*models.Artifact, error) {
	var builders []*models.Artifact
	err := Session().Preload("Profile").Find(&builders).Error
	if err != nil {
		return nil, err
	}

	var validBuilders []*models.Artifact
	for _, b := range builders {
		if b.Profile.PipelineID == pipelineID {
			validBuilders = append(validBuilders, b)
		}
	}
	return validBuilders, nil
}

func DeleteProfileByName(profileName string) error {
	// Check if profile exists first
	var existingProfile models.Profile
	result := Session().Where("name = ?", profileName).First(&existingProfile)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("profile '%s' not found", profileName)
	} else if result.Error != nil {
		return result.Error
	}

	// Execute deletion
	err := Session().Where("name = ?", profileName).Delete(&models.Profile{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete profile '%s': %v", profileName, err)
	}
	return nil
}

func UpdateProfileRaw(profileName string, raw []byte) error {
	return Session().Model(&models.Profile{}).Where("name = ?", profileName).Update("raw", raw).Error
}

func SaveArtifactFromConfig(req *clientpb.BuildConfig, profileByte []byte) (*models.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	builder := models.Artifact{
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.Type,
		Source:      req.Source,
		Arch:        target.Arch,
		Os:          target.OS,
		ProfileByte: profileByte,
		ParamsData:  string(req.ParamsBytes),
	}

	if Session() == nil {
		return &builder, nil
	}
	if err := Session().Create(&builder).Error; err != nil {
		return nil, err
	}

	return &builder, nil
}

func SaveArtifactFromID(req *clientpb.BuildConfig, ID uint32, resource string, profileByte []byte) (*models.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	builder := models.Artifact{
		ID:          ID,
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.Type,
		Source:      resource,
		Arch:        target.Arch,
		Os:          target.OS,
		ProfileByte: profileByte,
		ParamsData:  string(req.ParamsBytes),
	}

	if err := Session().Create(&builder).Error; err != nil {
		return nil, err

	}

	return &builder, nil
}

func UpdateBuilderPath(builder *models.Artifact) error {
	if Session() == nil {
		return nil
	}
	return Session().Model(builder).
		Select("path").
		Updates(builder).
		Error
}

func UpdateBuilderSrdi(builder *models.Artifact) error {
	if Session() == nil {
		return nil
	}
	return Session().Model(builder).
		Select("is_srdi", "shellcode_path").
		Updates(builder).
		Error
}

func UpdatePulseRelink(pusleID, beanconID uint32) error {
	pulse, err := GetArtifactById(pusleID)
	if err != nil {
		return err
	}
	pulse.Params.RelinkBeaconID = beanconID
	err = Session().Model(pulse).
		Select("ParamsData").
		Updates(pulse).
		Error
	if err != nil {
		return err
	}
	originBeacon, err := GetArtifactById(pulse.Params.OriginBeaconID)
	if err != nil {
		return err
	}
	originBeacon.Params.RelinkBeaconID = beanconID
	err = Session().Model(originBeacon).
		Select("ParamsData").
		Updates(originBeacon).
		Error
	if err != nil {
		return err
	}
	return nil
}

func SaveArtifact(name, artifactType, platform, arch, source string) (*models.Artifact, error) {
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, err
	}

	artifact := &models.Artifact{
		Name:   name,
		Os:     platform,
		Arch:   arch,
		Type:   artifactType,
		Source: source,
	}

	artifact.Path = filepath.Join(absBuildOutputPath, encoders.UUID())

	if err := Session().Create(artifact).Error; err != nil {
		return nil, err
	}
	return artifact, nil
}

func GetArtifacts() (*clientpb.Artifacts, error) {
	var builders []*models.Artifact
	result := Session().Preload("Profile").Preload("Profile.Pipeline").Find(&builders)
	if result.Error != nil {
		return nil, result.Error
	}
	var pbBuilders = &clientpb.Artifacts{
		Artifacts: make([]*clientpb.Artifact, 0),
	}
	for _, artifact := range builders {
		pbBuilders.Artifacts = append(pbBuilders.GetArtifacts(), artifact.ToProtobuf([]byte{}))
	}
	return pbBuilders, nil
}

func GetValidArtifacts() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	result := Session().Preload("Profile").Preload("Profile.Pipeline").Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}

	var validArtifacts []*models.Artifact
	for _, artifact := range artifacts {
		if artifact.Path != "" {
			if _, err := os.Stat(artifact.Path); err == nil {
				validArtifacts = append(validArtifacts, artifact)
			}
		}
	}
	return validArtifacts, nil
}

// FindArtifact
func FindArtifact(target *clientpb.Artifact) (*clientpb.Artifact, error) {
	var artifact *models.Artifact
	var result *gorm.DB
	// 根据 ID 或名称查找构建器
	if target.Id != 0 {
		result = Session().Where("id = ?", target.Id).First(&artifact)
	} else if target.Name != "" {
		result = Session().Where("name = ?", target.Name).First(&artifact)
	} else if target.Profile != "" {
		result = Session().Where("profile_name = ?", target.Profile).First(&artifact)
	} else {
		var builders []*models.Artifact
		result = Session().Where("os = ? AND arch = ? AND type = ?", target.Platform, target.Arch, target.Type).
			Preload("Profile.Pipeline").
			Find(&builders)
		for _, v := range builders {
			if v.Type == consts.ImplantPulse && v.Profile.PipelineID == target.Pipeline {
				artifact = v
				break
			}
			if v.Profile.PipelineID == target.Pipeline {
				artifact = v
				break
			}
		}
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error finding artifact: %v, target: %+v", result.Error, target)
	}
	if artifact == nil {
		return nil, errs.ErrNotFoundArtifact
	}
	content, err := os.ReadFile(artifact.Path)
	if err != nil && artifact.Status == consts.BuildStatusFailure {
		return nil, fmt.Errorf("error reading file for artifact: %s, error: %v", artifact.Name, err)
	}

	return artifact.ToProtobuf(content), nil
}

func GetArtifact(req *clientpb.Artifact) (*models.Artifact, error) {
	if req.Id != 0 {
		return GetArtifactById(req.Id)
	} else if req.Name != "" {
		return GetArtifactByName(req.Name)
	} else {
		return nil, errs.ErrNotFoundArtifact
	}
}

func GetArtifactByName(name string) (*models.Artifact, error) {
	var artifact models.Artifact
	result := Session().Preload("Profile").Where("name = ?", name).First(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	return &artifact, nil
}

func GetArtifactById(id uint32) (*models.Artifact, error) {
	var artifact models.Artifact
	result := Session().Preload("Profile").Where("id = ?", id).First(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	return &artifact, nil
}

func GetArtifactWithSaas() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	result := Session().Where("source = ?", consts.ArtifactFromSaas).Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}
	return artifacts, nil
}

// GetBeaconBuilderByRelinkID 查找 type=beacon 且 RelinkBeaconID=指定id 的 builder
func GetBeaconBuilderByRelinkID(relinkID uint32) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	err := Session().Where("type = ?", "beacon").Find(&artifacts).Error
	if err != nil {
		return nil, err
	}

	var result []*models.Artifact
	for _, b := range artifacts {
		var params types.ProfileParams
		if b.ParamsData != "" {
			if err := json.Unmarshal([]byte(b.ParamsData), &params); err == nil {
				if params.RelinkBeaconID == relinkID {
					result = append(result, b)
				}
			}
		}
	}
	return result, nil
}

func DeleteArtifactByName(artifactName string) error {
	model := &models.Artifact{}
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
	err = Session().Delete(model).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateGeneratorConfig - Update the generator config
func UpdateGeneratorConfig(req *clientpb.BuildConfig, config *types.ProfileConfig) error {
	if config.Basic != nil {
		if req.BuildName != "" {
			config.Basic.Name = req.BuildName
		}

		if len(req.ParamsBytes) > 0 {
			params, err := types.UnmarshalProfileParams(req.ParamsBytes)
			if err != nil {
				return err
			}
			if params.Interval != -1 {
				config.Basic.Interval = params.Interval
			}

			if params.Jitter != -1 {
				config.Basic.Jitter = params.Jitter
			}
			if params.Proxy != "" {
				config.Basic.Proxy = params.Proxy
			}

			if params.Enable3RD {
				config.Implant.Extras["3rd_modules"] = strings.Split(params.Modules, ",")
				config.Implant.Extras["enable_3rd"] = true
				config.Implant.Modules = []string{}
			} else {
				config.Implant.Modules = strings.Split(params.Modules, ",")
			}
		}
	}
	if req.ArtifactId != 0 && config.Pulse.Flags.ArtifactID == 0 {
		config.Pulse.Flags.ArtifactID = req.ArtifactId
	}

	if req.Type == consts.CommandBuildBind {
		config.Implant.Mod = consts.CommandBuildBind
	}
	return nil
}

func UpdateBuilderLog(name string, logEntry string) {
	if Session() == nil {
		return
	}
	err := Session().Model(&models.Artifact{}).
		Where("name = ?", name).
		Update("log", gorm.Expr("ifnull(log, '') || ?", logEntry)).
		Error

	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact name %s: %v", name, err)
	}
}

func GetBuilderLogs(builderName string, limit int) (string, error) {
	var builder models.Artifact
	if err := Session().Where("name = ?", builderName).First(&builder).Error; err != nil {
		return "", err
	}

	split := strings.Split(builder.Log, "\n")

	if limit > 0 && len(split) > limit {
		split = split[len(split)-limit:]
	}
	result := strings.Join(split, "\n")

	return result, nil
}

func UpdateBuilderStatus(builderID uint32, status string) {
	if Session() == nil {
		return
	}
	err := Session().Model(&models.Artifact{}).
		Where("id = ?", builderID).
		Update("status", status).
		Error
	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact id %d: %v", builderID, err)
	}
	return
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
