package website

import (
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"net/url"
	"os"
	"path/filepath"
)

func getWebContentDir() (string, error) {
	webContentDir := configs.WebsitePath
	// websiteLog.Debugf("Web content dir: %s", webContentDir)
	if _, err := os.Stat(webContentDir); os.IsNotExist(err) {
		err = os.MkdirAll(webContentDir, 0700)
		if err != nil {
			return "", err
		}
	}
	return webContentDir, nil
}

// GetContent - Get static content for a given path
func GetContent(name string, path string) (*clientpb.WebContent, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	// Use path without any query parameters
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	webContent, err := db.WebContentByIDAndPath(website.ID, u.Path, webContentDir, true)
	if err != nil {
		return nil, err
	}

	return webContent, err
}

// AddContent - Add website content for a path
func AddContent(name string, pbWebContent *clientpb.WebContent) error {
	// websiteName string, path string, contentType string, content []byte

	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	if pbWebContent.WebsiteID == "" {
		website, err := db.AddWebSite(name, webContentDir)
		if err != nil {
			return err
		}
		pbWebContent.WebsiteID = website.ID
	}

	webContent, err := db.AddContent(pbWebContent, webContentDir)
	if err != nil {
		return err
	}

	// Write content to disk
	webContentPath := filepath.Join(webContentDir, webContent.ID)
	return os.WriteFile(webContentPath, pbWebContent.Content, 0600)
}

// RemoveContent - Remove website content for a path
func RemoveContent(name string, path string) error {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return err
	}

	content, err := db.WebContentByIDAndPath(website.ID, path, webContentDir, true)
	if err != nil {
		return err
	}

	// Delete file
	webContentsDir, err := getWebContentDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(webContentsDir, content.ID))
	if err != nil {
		return err
	}

	// Delete row
	err = db.RemoveContent(content.ID)
	return err
}

// RemoveWebAllContent - Remove website content for website ID
func RemoveWebAllContent(ID string) error {
	webContentDir, err := getWebContentDir()
	// Delete file
	IDs, err := db.GetWebContentIDByWebsiteID(ID)
	if err != nil {
		return err
	}
	for _, contentID := range IDs {
		err = os.Remove(filepath.Join(webContentDir, contentID))
		if err != nil {
			return err
		}
	}
	// Delete row
	err = db.RemoveWebAllContent(ID)
	return err
}

// Names - List all websites
func Names() ([]string, error) {
	webContentsDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	websites, err := db.Websites(webContentsDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, website := range websites {
		names = append(names, website.Name)
	}
	return names, nil
}

// MapContent - List the content of a specific site, returns map of path->json(content-type/size)
func MapContent(name string, eager bool) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	if eager {
		eagerContents := map[string]*clientpb.WebContent{}
		for _, content := range website.Contents {
			eagerContent, err := db.WebContentByIDAndPath(website.ID, content.Path, webContentDir, true)
			if err != nil {
				continue
			}
			eagerContents[content.Path] = eagerContent
		}
		website.Contents = eagerContents
	}

	return website, nil
}

func AddWebsite(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	website, err := db.AddWebSite(name, webContentDir)
	if err != nil {
		return nil, err
	}
	return website, nil
}

func WebsiteByName(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	modelsWebsite, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}
	return modelsWebsite, nil
}
