package core

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func PushContextEvent(Op string, ctx *models.Context) {
	EventBroker.Publish(Event{
		EventType: consts.EventContext,
		Op:        Op,
		Task:      ctx.Task.ToProtobuf(),
		Important: true,
		Message:   ctx.Context.String(),
	})
}

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
	ctx := &output.ScreenShotContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       filename,
			Checksum:   checksum,
			TargetPath: "BOF SCREENSHOT",
			FilePath:   savePath,
			Size:       int64(len(data[4:])),
		},
	}
	ictx, err := SaveContext(ctx, task)
	if err != nil {
		return err
	}

	PushContextEvent(consts.CtrlContextScreenShot, ictx)
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
		EventBroker.Publish(Event{
			EventType: consts.EventContext,
			Op:        consts.CtrlContextFileCreate,
			Task:      task.ToProtobuf(),
			Message:   fmt.Sprintf("file created: %s", originalName),
		})
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
		//EventBroker.Publish(Event{
		//	EventType: consts.EventContext,
		//	Op:        consts.CtrlContextFileWrite,
		//	Task:      task.ToProtobuf(),
		//	Message:   fmt.Sprintf("file write: %s %d", savePath, len(data[4:])),
		//})
		return nil

	case "close":
		savePath, ok := sess.Any[getFileExtKey(fileId)].(string)
		if !ok {
			return fmt.Errorf("no file found for ID: %d", fileId)
		}
		checksum, _ := fileutils.CalculateSHA256Checksum(savePath)
		_, err := SaveContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
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
		EventBroker.Publish(Event{
			EventType: consts.EventContext,
			Op:        consts.CtrlContextFileClose,
			Task:      task.ToProtobuf(),
			Message:   fmt.Sprintf("file_saved_on_server: %s", savePath),
		})
		return nil
	}

	return fmt.Errorf("unknown operation: %s", op)
}

func SaveContext(ctx output.Context, task *Task) (*models.Context, error) {
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

func LoadContext(ctx output.Context) (output.Context, error) {
	switch c := ctx.(type) {
	case *output.ScreenShotContext:
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return nil, err
		}
		c.Content = data
		return c, nil
	case *output.DownloadContext:
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return nil, err
		}
		c.Content = data
		return c, nil
	case *output.KeyLoggerContext:
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return nil, err
		}
		c.Content = data
		return c, nil
	case *output.UploadContext:
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return nil, err
		}
		c.Content = data
		return c, nil
	case *output.MediaContext:
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return nil, err
		}
		c.Content = data
		return c, nil
	}

	return ctx, nil
}

func ReadFileForContext(ctx output.Context) ([]byte, error) {
	var filePath string
	switch c := ctx.(type) {
	case *output.ScreenShotContext:
		filePath = c.FilePath
	case *output.DownloadContext:
		filePath = c.FilePath
	case *output.KeyLoggerContext:
		filePath = c.FilePath
	case *output.UploadContext:
		filePath = c.FilePath
	case *output.MediaContext:
		filePath = c.FilePath
	default:
		return nil, errors.New("unsupported context type")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func sanitizeContextFragment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	value = filepath.Base(value)
	value = strings.ReplaceAll(value, "..", "")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, string(filepath.Separator), "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func sanitizeFileName(name string, fallback string) string {
	cleaned := sanitizeContextFragment(name)
	if cleaned == "" {
		cleaned = fallback
	}
	if cleaned == "" {
		cleaned = fmt.Sprintf("media-%d.bin", time.Now().UnixNano())
	}
	return cleaned
}

func deterministicContextID(sessionID, identifier, nonce, contextType string) string {
	base := fmt.Sprintf("%s:%s:%s:%s", sessionID, identifier, nonce, contextType)
	return uuid.NewV5(uuid.NamespaceOID, base).String()
}

func HandleKeylogger(data []byte, task *Task, identifier string, filename string, nonce string) error {
	if len(data) == 0 {
		return nil
	}

	name := sanitizeContextFragment(filename)
	identifier = sanitizeContextFragment(identifier)
	if name == "" {
		if identifier == "" {
			identifier = sanitizeContextFragment(nonce)
		}
		if identifier == "" {
			identifier = time.Now().Format("2006_01_02_15_04_05")
		}
		name = fmt.Sprintf("%s.log", identifier)
	}

	dir := filepath.Join(configs.ContextPath, task.SessionId, consts.KeyLoggerPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	savePath := filepath.Join(dir, name)

	file, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	info, err := os.Stat(savePath)
	if err != nil {
		return err
	}
	checksum, _ := fileutils.CalculateSHA256Checksum(savePath)

	ctx := &output.KeyLoggerContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       name,
			Checksum:   checksum,
			TargetPath: "KeyLogger",
			FilePath:   savePath,
			Size:       info.Size(),
		},
	}

	value := output.MarshalContext(ctx)
	if value == nil {
		return errors.New("failed to marshal keylogger context")
	}

	contextPB := &clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    consts.ContextKeyLogger,
		Value:   value,
		Nonce:   nonce,
	}
	contextPB.Id = deterministicContextID(task.SessionId, name, nonce, consts.ContextKeyLogger)

	model, err := db.SaveContext(contextPB)
	if err != nil {
		return err
	}

	EventBroker.Publish(Event{
		EventType: consts.EventContext,
		Op:        consts.ContextKeyLogger,
		Task:      task.ToProtobuf(),
		Message:   fmt.Sprintf("keylogger context %s updated (%s)", model.ID.String(), name),
	})
	return nil
}

func HandleMediaChunk(task *Task, nonce, identifier, filename, mediaKind string, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	sanitizedID := sanitizeContextFragment(identifier)
	if sanitizedID == "" {
		sanitizedID = sanitizeContextFragment(nonce)
	}
	if sanitizedID == "" {
		sanitizedID = fmt.Sprintf("%s-%d", task.SessionId, time.Now().UnixNano())
	}

	saveName := sanitizeFileName(filename, sanitizedID+".bin")
	dir := filepath.Join(configs.ContextPath, task.SessionId, consts.MediaPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	savePath := filepath.Join(dir, saveName)

	file, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	mediaCtx := &output.MediaContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       saveName,
			TargetPath: mediaKind,
			FilePath:   savePath,
			Size:       info.Size(),
		},
		Identifier: sanitizedID,
		MediaKind:  mediaKind,
	}

	value := output.MarshalContext(mediaCtx)
	if value == nil {
		return errors.New("failed to marshal media context")
	}

	contextPB := &clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    consts.ContextMedia,
		Value:   value,
		Nonce:   nonce,
	}
	if sanitizedID != "" || nonce != "" {
		contextPB.Id = deterministicContextID(task.SessionId, sanitizedID, nonce, consts.ContextMedia)
	}

	model, err := db.SaveContext(contextPB)
	if err != nil {
		return err
	}

	EventBroker.Publish(Event{
		EventType: consts.EventContext,
		Op:        consts.ContextMedia,
		Task:      task.ToProtobuf(),
		Message:   fmt.Sprintf("media context %s updated (%s)", model.ID.String(), saveName),
	})
	return nil
}
