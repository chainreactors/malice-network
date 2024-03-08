package models

import (
	"errors"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

// Certificate - Certificate database model
type Certificate struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	CommonName     string
	CAType         int
	KeyType        string
	CertificatePEM string
	PrivateKeyPEM  string
}

// BeforeCreate - GORM hook to automatically set values
func (c *Certificate) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

// DeleteAllCertificates
func DeleteAllCertificates(db *gorm.DB) error {
	result := db.Exec("DELETE FROM certificates")
	return result.Error
}

// DeleteCertificate
func DeleteCertificate(db *gorm.DB, name string) error {
	var cert Certificate
	result := db.Where("common_name = ?", name).First(&cert)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	result = db.Delete(&cert)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func isDuplicateCommonNameAndCAType(db *gorm.DB, commonName string, caType int) bool {
	var count int64
	db.Model(&Certificate{}).Where("common_name = ? AND ca_type = ?", commonName, caType).Count(&count)
	return count > 0
}

func SaveCertificate(db *gorm.DB, certificate *Certificate) error {
	if isDuplicateCommonNameAndCAType(db, certificate.CommonName, certificate.CAType) {
		return errors.New("duplicate CommonName and CAType")
	}
	if err := db.Create(certificate).Error; err != nil {
		return err
	}

	return nil
}
