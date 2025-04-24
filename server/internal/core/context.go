package core

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/malice-network/helper/types"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func HandleScreenshot(data []byte, task *Task) error {
	t := time.Now()
	filename := fmt.Sprintf("%d.jpg", t.Unix())
	savePath := filepath.Join(configs.ContextPath, task.SessionId, consts.ScreenShotPath, filename)

	if err := os.MkdirAll(filepath.Dir(savePath), os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(savePath, data[4:], 0644); err != nil {
		return err
	}

	checksum, _ := fileutils.CalculateSHA256Checksum(savePath)
	ctx := &types.ScreenShotContext{
		FileDescriptor: &types.FileDescriptor{
			Name:       filename,
			Checksum:   checksum,
			TargetPath: "BOF SCREENSHOT",
			FilePath:   savePath,
			Size:       int64(len(data[4:])),
		},
	}
	ictx, err := saveFileContext(ctx, task)
	if err != nil {
		return err
	}

	EventBroker.Publish(Event{
		EventType: consts.EventContext,
		Op:        consts.CtrlContextScreenShot,
		Task:      task.ToProtobuf(),
		Message:   fmt.Sprintf("Screenshot saved to %s, id: %s", savePath, ictx.ID),
	})
	return nil
}

func getFileExtKey(fileId uint32) string {
	return fmt.Sprintf("file_ext_%d", fileId)
}

func HandleFileOperations(op string, data []byte, task *Task) error {
	fileId := binary.LittleEndian.Uint32(data[:4])
	sess := task.Session
	dirPath := filepath.Join(configs.ContextPath, sess.ID, consts.DownloadPath)

	switch op {
	case "open":
		originalName := string(data[8:])
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}

		savePath := filepath.Join(dirPath, originalName)
		file, err := os.OpenFile(savePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open file failed: %w", err)
		}
		defer file.Close()

		sess.Any[getFileExtKey(fileId)] = savePath
		return nil

	case "write":
		savePath, ok := sess.Any[getFileExtKey(fileId)].(string)
		if !ok {
			return fmt.Errorf("no file found for ID: %d", fileId)
		}

		file, err := os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open file failed: %w", err)
		}
		defer file.Close()

		if _, err := file.Write(data[4:]); err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}
		return nil

	case "close":
		savePath, ok := sess.Any[getFileExtKey(fileId)].(string)
		if !ok {
			return fmt.Errorf("no file found for ID: %d", fileId)
		}

		checksum, _ := fileutils.CalculateSHA256Checksum(savePath)
		_, err := saveFileContext(&types.DownloadContext{
			FileDescriptor: &types.FileDescriptor{
				Name:       filepath.Base(savePath),
				Checksum:   checksum,
				TargetPath: "BOF DOWNLOAD",
				FilePath:   savePath,
				//Size:       fileutils.GetFileSize(savePath),
			},
		}, task)
		if err != nil {
			return err
		}
		delete(sess.Any, getFileExtKey(fileId))
		return nil
	}

	return fmt.Errorf("unknown operation: %s", op)
}

func saveFileContext(ctx types.Context, task *Task) (*models.Context, error) {
	value, err := json.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	return db.SaveContext(&clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    ctx.Type(),
		Value:   value,
	})
}
