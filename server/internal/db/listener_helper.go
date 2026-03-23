package db

import (
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"mime"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// ============================================
// Pipeline Operations
// ============================================

func FindPipeline(name string) (*models.Pipeline, error) {
	return NewPipelineQuery().WhereName(name).WithCert().First()
}

func FindPipelineByListener(name, listenerID string) (*models.Pipeline, error) {
	return NewPipelineQuery().WhereName(name).WhereListenerID(listenerID).WithCert().First()
}

func UpdatePipelineCert(certName string, pipeline *models.Pipeline) (*models.Pipeline, error) {
	var cert *models.Certificate
	if certName != "" {
		var err error
		cert, err = FindCertificate(certName)
		if err != nil {
			return nil, err
		}
	}

	err := Session().Model(pipeline).Select("cert_name").Update("cert_name", certName).Error
	if err != nil {
		return nil, err
	}
	pipeline.Tls = implanttypes.FromTls(cert.ToProtobuf())
	return pipeline, err
}

func SavePipeline(pipeline *models.Pipeline) (*models.Pipeline, error) {
	if pipeline == nil {
		return nil, errors.New("pipeline cannot be nil")
	}
	existing, err := NewPipelineQuery().WhereName(pipeline.Name).WhereListenerID(pipeline.ListenerId).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if createErr := Session().Create(&pipeline).Error; createErr != nil {
				return nil, createErr
			}
			return pipeline, nil
		}
		return nil, err
	}
	pipeline.ID = existing.ID
	pipeline.CertName = existing.CertName
	if pipeline.IP == "" {
		pipeline.IP = existing.IP
	}
	saveErr := Session().Save(&pipeline).Error
	return pipeline, saveErr
}

func DeletePipeline(name string) error {
	return NewPipelineQuery().WhereName(name).Delete()
}

func DeletePipelineByListener(name, listenerID string) error {
	return NewPipelineQuery().WhereName(name).WhereListenerID(listenerID).Delete()
}

// setPipelineEnabled sets the Enable flag on a pipeline identified by name
// and optional listenerID. If listenerID is empty, matches by name only.
func setPipelineEnabled(pid, listenerID string, enabled bool) error {
	var pipeline *models.Pipeline
	var err error
	if listenerID != "" {
		pipeline, err = FindPipelineByListener(pid, listenerID)
	} else {
		pipeline, err = FindPipeline(pid)
	}
	if err != nil {
		return err
	}
	pipeline.Enable = enabled
	return Save(pipeline)
}

func EnablePipeline(pid string) error {
	return setPipelineEnabled(pid, "", true)
}

func EnablePipelineByListener(pid, listenerID string) error {
	return setPipelineEnabled(pid, listenerID, true)
}

func DisablePipeline(pid string) error {
	return setPipelineEnabled(pid, "", false)
}

func DisablePipelineByListener(pid, listenerID string) error {
	return setPipelineEnabled(pid, listenerID, false)
}

func FindPipelineCert(pipelineName, listenerID string) (*models.Certificate, error) {
	pipeline, err := NewPipelineQuery().WhereName(pipelineName).WhereListenerID(listenerID).First()
	if err != nil {
		return nil, err
	}
	if pipeline.CertName != "" {
		return FindCertificate(pipeline.CertName)
	}
	return nil, nil
}

// ============================================
// Certificate Operations
// ============================================

// DeleteCertificate
func DeleteCertificate(name string) error {
	_, err := FindCertificate(name)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return NewCertificateQuery().WhereName(name).Delete()
}

func FindCertificate(name string) (*models.Certificate, error) {
	return NewCertificateQuery().WhereName(name).First()
}

func GetAllCertificates() (Certificates, error) {
	return NewCertificateQuery().Find()
}

func UpdateCert(name, cert, key, ca string) error {
	return NewCertificateQuery().WhereName(name).UpdateFields(map[string]interface{}{
		"cert_pem":    cert,
		"key_pem":     key,
		"ca_cert_pem": ca,
	})
}

func isDuplicateCommonNameAndCAType(name string) bool {
	_, err := FindCertificate(name)
	return err == nil
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

func SaveCertFromTLS(tls *clientpb.TLS, pipelineName, listenerID string) (*models.Certificate, error) {
	certModel := &models.Certificate{
		CertPEM: tls.Cert.Cert,
		KeyPEM:  tls.Cert.Key,
	}
	if tls.Acme {
		certModel.Name = tls.Domain
		certModel.Domain = tls.Domain
		certModel.Type = certs.Acme
	} else if tls.Ca != nil && tls.Ca.Key != "" {
		certModel.Name = codenames.GetCodename()
		certModel.Type = certs.SelfSigned
		certModel.CACertPEM = tls.Ca.Cert
		certModel.CAKeyPEM = tls.Ca.Key
	} else {
		certModel.Name = codenames.GetCodename()
		certModel.Type = certs.Imported
		if tls.Ca != nil {
			certModel.CACertPEM = tls.Ca.Cert
		}
	}
	err := SaveCertificate(certModel)
	if err != nil {
		return certModel, err
	}
	if pipelineName != "" {
		var findPipeline *models.Pipeline
		var err error
		if listenerID != "" {
			findPipeline, err = FindPipelineByListener(pipelineName, listenerID)
		} else {
			findPipeline, err = FindPipeline(pipelineName)
		}
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

// ============================================
// Website Operations
// ============================================

func DeleteWebsite(name string) error {
	website, err := NewPipelineQuery().WhereName(name).First()
	if err != nil {
		return err
	}
	websiteDir, err := fileutils.SafeJoin(configs.WebsitePath, website.Name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(websiteDir); err != nil {
		return err
	}
	return Delete(website)
}

// FindWebsiteByName - Get website by name
func FindWebsiteByName(name string) (*models.Pipeline, error) {
	return NewPipelineQuery().WhereName(name).WhereType(consts.WebsitePipeline).WithCert().First()
}

// WebContent by ID and path
func FindWebContent(id string) (*models.WebsiteContent, error) {
	uuidFromString, err := uuid.FromString(id)
	if err != nil {
		return nil, err
	}
	return NewWebContentQuery().WhereID(uuidFromString).WithPipeline().First()
}

func FindWebContentsByWebsite(website string) ([]*models.WebsiteContent, error) {
	query := NewWebContentQuery().WithPipeline()
	if website != "" {
		query = query.WherePipelineID(website)
	}
	return query.Find()
}

// AddContent - Add content to website
func AddContent(content *clientpb.WebContent) (*models.WebsiteContent, error) {
	if content == nil {
		return nil, errors.New("content is nil")
	}
	if content.Size == 0 && len(content.Content) > 0 {
		content.Size = uint64(len(content.Content))
	}

	switch content.Type {
	case "", "raw", "default":
		content.Type = "raw"
		if content.ContentType == "" {
			content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
		}
	default:
		if content.ContentType == "" {
			content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
		}
	}

	var existingContent *models.WebsiteContent
	webModel := models.FromWebContentPb(content)
	err := Session().Preload("Pipeline").Where("pipeline_id = ? AND path = ?", content.WebsiteId, content.Path).First(&existingContent).Error
	if err == nil {
		webModel.ID = existingContent.ID
		query := Session()
		if webModel.PipelineID == "" {
			query = query.Omit("pipeline_id")
		}
		err = query.Save(&webModel).Error
		if err != nil {
			return nil, err
		}
		if err := Session().Preload("Pipeline").Where("id = ?", webModel.ID).First(webModel).Error; err != nil {
			return nil, err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		query := Session()
		if webModel.PipelineID == "" {
			query = query.Omit("pipeline_id")
		}
		err = query.Create(&webModel).Error
		if err != nil {
			return nil, err
		}
		if webModel.Pipeline == nil {
			err := Session().Model(webModel).Association("Pipeline").Find(&webModel.Pipeline)
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, err
	}
	if content.Type == "raw" {
		contentPath, err := fileutils.SafeJoin(configs.WebsitePath, filepath.Join(content.WebsiteId, webModel.ID.String()))
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(contentPath), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(contentPath, content.Content, 0o600); err != nil {
			return nil, err
		}
	}

	content.Id = webModel.ID.String()
	return webModel, nil
}

func AddAmountWebContent(artifactName, pipelineName string) (*clientpb.WebContent, error) {
	content := &clientpb.WebContent{
		WebsiteId: pipelineName,
		Path:      output.Encode(artifactName),
		Type:      consts.ArtifactWebcontent,
	}
	_, err := AddContent(content)
	if err != nil {
		return nil, err
	}
	content.Path = artifactName
	return content, nil
}

// RemoveContent - Remove content by ID
func RemoveContent(id string) error {
	content, err := FindWebContent(id)
	if err != nil {
		return err
	}

	if content.PipelineID != "" {
		contentPath, joinErr := fileutils.SafeJoin(configs.WebsitePath, filepath.Join(content.PipelineID, content.ID.String()))
		if joinErr != nil {
			return joinErr
		}
		if removeErr := os.Remove(contentPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return removeErr
		}
	}

	contentID, _ := uuid.FromString(id)
	return NewWebContentQuery().WhereID(contentID).Delete()
}

// FindEnabledWebsites - Get all enabled websites from database
func FindEnabledWebsites() (Pipelines, error) {
	return NewPipelineQuery().WhereType(consts.WebsitePipeline).WhereEnabled(true).WithCert().Find()
}
