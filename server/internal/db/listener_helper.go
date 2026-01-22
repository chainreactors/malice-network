package db

import (
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"mime"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
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
			pipeline.Tls = implanttypes.FromTls(certificate.ToProtobuf())
		}
	}
	return pipeline, nil
}

func FindPipelineByListener(name, listenerID string) (*models.Pipeline, error) {
	var pipeline *models.Pipeline
	result := Session().Where("name = ? AND listener_id = ?", name, listenerID).First(&pipeline)
	if result.Error != nil {
		return pipeline, result.Error
	}
	if pipeline.CertName != "" {
		certificate, err := FindCertificate(pipeline.CertName)
		if err != nil && !errors.Is(err, ErrRecordNotFound) {
			logs.Log.Errorf("failed to find cert %s", err)
		}
		if certificate != nil {
			pipeline.Tls = implanttypes.FromTls(certificate.ToProtobuf())
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
	pipeline.Tls = implanttypes.FromTls(cert.ToProtobuf())
	return pipeline, err
}

func SavePipeline(pipeline *models.Pipeline) (*models.Pipeline, error) {
	if pipeline == nil {
		return nil, errors.New("pipeline cannot be nil")
	}
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

func DeletePipelineByListener(name, listenerID string) error {
	result := Session().Where("name = ? AND listener_id = ?", name, listenerID).Delete(&models.Pipeline{})
	return result.Error
}

func EnablePipeline(pid string) error {
	pipeline, err := FindPipeline(pid)
	if err != nil {
		return err
	}
	pipeline.Enable = true
	return Save(pipeline)
}

func EnablePipelineByListener(pid, listenerID string) error {
	pipeline, err := FindPipelineByListener(pid, listenerID)
	if err != nil {
		return err
	}
	pipeline.Enable = true
	return Save(pipeline)
}

func DisablePipeline(pid string) error {
	pipeline, err := FindPipeline(pid)
	if err != nil {
		return err
	}
	pipeline.Enable = false
	return Save(pipeline)
}

func DisablePipelineByListener(pid, listenerID string) error {
	pipeline, err := FindPipelineByListener(pid, listenerID)
	if err != nil {
		return err
	}
	pipeline.Enable = false
	return Save(pipeline)
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

// ============================================
// Certificate Operations
// ============================================

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

func UpdateCert(name, cert, key, ca string) error {
	return Session().Model(&models.Certificate{}).
		Where("name = ?", name).
		Select("cert_pem", "key_pem", "ca_cert_pem").
		Updates(models.Certificate{
			CertPEM:   cert,
			KeyPEM:    key,
			CACertPEM: ca,
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
			findPipeline = &models.Pipeline{}
			result := Session().Where("name = ? AND listener_id = ?", pipelineName, listenerID).First(&findPipeline)
			if result.Error != nil {
				return nil, result.Error
			}
		} else {
			findPipeline, err = FindPipeline(pipelineName)
			if err != nil {
				return nil, err
			}
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
	website := models.Pipeline{}
	result := Session().Where("name = ?", name).First(&website)
	if result.Error != nil {
		return result.Error
	}
	err := os.RemoveAll(filepath.Join(configs.WebsitePath, website.Name))
	if err != nil {
		return err
	}
	result = Session().Delete(&website)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// FindWebsiteByName - Get website by name
func FindWebsiteByName(name string) (*models.Pipeline, error) {
	var website *models.Pipeline
	if err := Session().Where("name = ? AND type = 'website'", name).First(&website).Error; err != nil {
		return nil, err
	}
	if website.CertName != "" {
		certificate, err := FindCertificate(website.CertName)
		if err != nil && !errors.Is(err, ErrRecordNotFound) {
			logs.Log.Errorf("failed to find cert %s", err)
		}
		if certificate != nil {
			website.Tls = implanttypes.FromTls(certificate.ToProtobuf())
		}
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
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.WebsiteContent{}, uuid).Error
	return err
}

// FindEnabledWebsites - Get all enabled websites from database
func FindEnabledWebsites() ([]*models.Pipeline, error) {
	var websites []*models.Pipeline
	err := Session().Where("type = ? AND enable = ?", consts.WebsitePipeline, true).Find(&websites).Error
	if err != nil {
		return nil, err
	}

	// Load certificates for each website
	for _, website := range websites {
		if website.CertName != "" {
			certificate, err := FindCertificate(website.CertName)
			if err != nil && !errors.Is(err, ErrRecordNotFound) {
				logs.Log.Errorf("failed to find cert %s for website %s: %s", website.CertName, website.Name, err)
				continue
			}
			if certificate != nil {
				website.Tls = implanttypes.FromTls(certificate.ToProtobuf())
			}
		}
	}

	return websites, nil
}
