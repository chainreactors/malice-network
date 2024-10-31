package website

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"net/url"
	"os"
	"path/filepath"
)

func getWebContentDir() (string, error) {
	webContentDir := configs.WebsitePath
	if _, err := os.Stat(webContentDir); os.IsNotExist(err) {
		err = os.MkdirAll(webContentDir, 0700)
		if err != nil {
			return "", err
		}
	}
	return webContentDir, nil
}

// GetContent
func GetContent(name string, path string) (*clientpb.WebContent, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	websiteContent, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	return db.WebContentByIDAndPath(websiteContent.ID, u.Path, webContentDir, true)
}

// AddContent
func AddContent(name string, pbWebContent *clientpb.WebContent) (string, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return "", err
	}

	webContent, err := db.AddContent(pbWebContent, webContentDir)
	if err != nil {
		return "", err
	}

	//webContentPath := filepath.Join(webContentDir, webContent.ID)
	return webContent.ID, nil
}

// RemoveContent
func RemoveContent(name string, path string) error {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	websiteContent, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return err
	}

	content, err := db.WebContentByIDAndPath(websiteContent.ID, path, webContentDir, true)
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(webContentDir, content.ID))
	if err != nil {
		return err
	}

	return db.RemoveContent(content.ID)
}

// Name
func Names() ([]string, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	websites, err := db.Websites(webContentDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, website := range websites {
		names = append(names, website.ID)
	}
	return names, nil
}

// MapConten
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
		content, err := db.WebContentByIDAndPath(website.ID, website.RootPath, webContentDir, true)
		if err != nil {
			return nil, err
		}
		eagerContents[content.Path] = content
		website.Contents = eagerContents
	}

	return website, nil
}

// AddWebsite
func AddWebsite(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	return db.AddWebsite(name, webContentDir)
}

// WebsiteByName
func WebsiteByName(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	return db.WebsiteByName(name, webContentDir)
}
