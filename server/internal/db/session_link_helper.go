package db

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
)

var (
	ErrSessionLinkSelf            = errors.New("session cannot be its own parent")
	ErrSessionLinkCycle           = errors.New("session link would create a cycle")
	ErrSessionLinkNotFound        = errors.New("session link not found")
	ErrSessionLinkSessionNotFound = errors.New("session not found")

	sessionLinkMu sync.Mutex
)

type SessionLinks []*models.SessionLink

func (links SessionLinks) ToProtobuf() *clientpb.SessionLinks {
	result := &clientpb.SessionLinks{Links: make([]*clientpb.SessionLink, 0, len(links))}
	for _, link := range links {
		if link != nil {
			result.Links = append(result.Links, link.ToProtobuf())
		}
	}
	return result
}

func SetSessionLink(parentSessionID, childSessionID, source string) (*models.SessionLink, error) {
	parentSessionID = strings.TrimSpace(parentSessionID)
	childSessionID = strings.TrimSpace(childSessionID)
	if parentSessionID == "" || childSessionID == "" {
		return nil, errors.New("parent and child session IDs are required")
	}
	if parentSessionID == childSessionID {
		return nil, ErrSessionLinkSelf
	}
	if source == "" {
		source = models.SessionLinkSourceManual
	}

	sessionLinkMu.Lock()
	defer sessionLinkMu.Unlock()

	var link *models.SessionLink
	err := Session().Transaction(func(tx *gorm.DB) error {
		if err := requireSessionLinkEndpoint(tx, parentSessionID); err != nil {
			return err
		}
		if err := requireSessionLinkEndpoint(tx, childSessionID); err != nil {
			return err
		}
		if err := rejectSessionLinkCycle(tx, parentSessionID, childSessionID); err != nil {
			return err
		}

		candidate := &models.SessionLink{ChildSessionID: childSessionID}
		if err := tx.Where("child_session_id = ?", childSessionID).
			Assign(models.SessionLink{ParentSessionID: parentSessionID, Source: source}).
			FirstOrCreate(candidate).Error; err != nil {
			return err
		}
		link = candidate
		return nil
	})
	return link, err
}

func ListSessionLinks(parentSessionID, childSessionID string) (SessionLinks, error) {
	query := Session().Model(&models.SessionLink{})
	if parentSessionID = strings.TrimSpace(parentSessionID); parentSessionID != "" {
		query = query.Where("parent_session_id = ?", parentSessionID)
	}
	if childSessionID = strings.TrimSpace(childSessionID); childSessionID != "" {
		query = query.Where("child_session_id = ?", childSessionID)
	}

	var links SessionLinks
	err := query.Order("parent_session_id ASC, child_session_id ASC").Find(&links).Error
	return links, err
}

func RemoveSessionLink(childSessionID string) error {
	childSessionID = strings.TrimSpace(childSessionID)
	if childSessionID == "" {
		return errors.New("child session ID is required")
	}

	sessionLinkMu.Lock()
	defer sessionLinkMu.Unlock()

	result := Session().Where("child_session_id = ?", childSessionID).Delete(&models.SessionLink{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSessionLinkNotFound
	}
	return nil
}

func removeSessionLinks(tx *gorm.DB, sessionID string) error {
	return tx.Where("parent_session_id = ? OR child_session_id = ?", sessionID, sessionID).
		Delete(&models.SessionLink{}).Error
}

func requireSessionLinkEndpoint(tx *gorm.DB, sessionID string) error {
	var count int64
	if err := tx.Model(&models.Session{}).
		Where("session_id = ? AND is_removed = ?", sessionID, false).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: %s", ErrSessionLinkSessionNotFound, sessionID)
	}
	return nil
}

func rejectSessionLinkCycle(tx *gorm.DB, parentSessionID, childSessionID string) error {
	current := parentSessionID
	visited := make(map[string]struct{})
	for current != "" {
		if current == childSessionID {
			return ErrSessionLinkCycle
		}
		if _, ok := visited[current]; ok {
			return ErrSessionLinkCycle
		}
		visited[current] = struct{}{}

		var link models.SessionLink
		err := tx.Where("child_session_id = ?", current).First(&link).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		current = link.ParentSessionID
	}
	return nil
}
