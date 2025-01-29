package core

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// 处理截图保存
func HandleScreenshot(data []byte, task *Task) (string, error) {
	t := time.Now()
	filename := fmt.Sprintf("%d.jpg", t.Unix())
	savePath := filepath.Join(configs.ContextPath, task.SessionId, consts.ScreenShotPath, filename)

	if err := os.MkdirAll(filepath.Dir(savePath), os.ModePerm); err != nil {
		return "", fmt.Errorf("create directory failed: %w", err)
	}

	if err := os.WriteFile(savePath, data[4:], 0644); err != nil {
		return "", fmt.Errorf("write file failed: %w", err)
	}

	checksum, _ := fileutils.CalculateSHA256Checksum(savePath)
	if err := saveContext(&models.FileDescription{
		Name:       filename,
		Checksum:   checksum,
		SourcePath: "BOF SCREENSHOT",
		SavePath:   savePath,
		Size:       int64(len(data[4:])),
	}, task, consts.ContextScreenShot); err != nil {
		return "", fmt.Errorf("save context failed: %w", err)
	}

	return fmt.Sprintf("Screenshot saved to %s", savePath), nil
}

// 获取文件扩展名的key
func getFileExtKey(fileId uint32) string {
	return fmt.Sprintf("file_ext_%d", fileId)
}

// 处理文件操作
func HandleFileOperations(op string, data []byte, task *Task) (string, error) {
	fileId := binary.LittleEndian.Uint32(data[:4])
	sess := task.Session
	dirPath := filepath.Join(configs.ContextPath, sess.ID, consts.DownloadPath)

	switch op {
	case "open":
		originalName := string(data[8:])
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return "", fmt.Errorf("create directory failed: %w", err)
		}

		savePath := filepath.Join(dirPath, originalName)
		file, err := os.OpenFile(savePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("open file failed: %w", err)
		}
		defer file.Close()

		sess.Any[getFileExtKey(fileId)] = savePath
		return fmt.Sprintf("File '%s' created", originalName), nil

	case "write":
		savePath, ok := sess.Any[getFileExtKey(fileId)].(string)
		if !ok {
			return "", fmt.Errorf("no file found for ID: %d", fileId)
		}

		file, err := os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("open file failed: %w", err)
		}
		defer file.Close()

		if _, err := file.Write(data[4:]); err != nil {
			return "", fmt.Errorf("write file failed: %w", err)
		}
		return fmt.Sprintf("Data written to %s", filepath.Base(savePath)), nil

	case "close":
		savePath, ok := sess.Any[getFileExtKey(fileId)].(string)
		if !ok {
			return "", fmt.Errorf("no file found for ID: %d", fileId)
		}

		checksum, _ := fileutils.CalculateSHA256Checksum(savePath)
		if err := saveContext(&models.FileDescription{
			Name:       filepath.Base(savePath),
			Checksum:   checksum,
			SourcePath: "BOF FILE",
			SavePath:   savePath,
			Size:       int64(len(data[4:])),
		}, task, consts.ContextDownload); err != nil {
			return "", fmt.Errorf("save context failed: %w", err)
		}

		delete(sess.Any, getFileExtKey(fileId))
		return fmt.Sprintf("File '%s' completed", filepath.Base(savePath)), nil
	}

	return "", fmt.Errorf("unknown operation: %s", op)
}

// 保存文件上下文
func saveContext(fileDesc *models.FileDescription, task *Task, contextType string) error {
	fileJson, err := fileDesc.ToJsonString()
	if err != nil {
		return err
	}

	return db.SaveContext(&clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    contextType,
		Value:   fileJson,
	})
}
