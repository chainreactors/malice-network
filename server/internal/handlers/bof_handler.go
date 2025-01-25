package handlers

import (
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CALLBACK_OUTPUT      = 0
	CALLBACK_FILE        = 0x02
	CALLBACK_FILE_WRITE  = 0x08
	CALLBACK_FILE_CLOSE  = 0x09
	CALLBACK_SCREENSHOT  = 0x03
	CALLBACK_ERROR       = 0x0d
	CALLBACK_OUTPUT_OEM  = 0x1e
	CALLBACK_OUTPUT_UTF8 = 0x20
)

type BOFResponse struct {
	CallbackType uint8
	OutputType   uint8
	Length       uint32
	Data         []byte
}

type BOFResponses []*BOFResponse

func (bofResps BOFResponses) Handler(session *clientpb.Session, taskpb *clientpb.Task) string {
	var err error
	var results strings.Builder
	sessionId := taskpb.SessionId
	fileMap := make(map[string]*os.File)

	for _, bofResp := range bofResps {
		var result string
		switch bofResp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			result = string(bofResp.Data)
		case CALLBACK_ERROR:
			result = fmt.Sprintf("Error occurred: %s", string(bofResp.Data))
		case CALLBACK_SCREENSHOT:
			result = func() string {
				if bofResp.Length-4 <= 0 {
					return fmt.Sprintf("Null screenshot data")
				}
				timestampMillis := time.Now().UnixNano() / int64(time.Millisecond)
				seconds := timestampMillis / 1000
				nanoseconds := (timestampMillis % 1000) * int64(time.Millisecond)
				t := time.Unix(seconds, nanoseconds)
				screenshotfilename := fmt.Sprintf("screenshot_%s.jpg", t.Format("2006-01-02_15-04-05"))
				sessionDir := filepath.Join(configs.ContextPath, sessionId, consts.ScreenShotPath)
				if !fileutils.Exist(sessionDir) {
					if err := os.MkdirAll(sessionDir, os.ModePerm); err != nil {
						logs.Log.Errorf("failed to create session directory: %s", err.Error())
					}
				}
				screenshotFullPath := filepath.Join(sessionDir, screenshotfilename)
				screenfile, err := os.Create(screenshotFullPath)
				if err != nil {
					return fmt.Sprintf("Failed to create screenshot file")
				}
				defer func() {
					err := screenfile.Close()
					if err != nil {
						return
					}
				}()
				data := bofResp.Data[4:]
				if _, err := screenfile.Write(data); err != nil {
					return fmt.Sprintf("Failed to write screenshot data: %s", err.Error())
				}
				checksum, _ := fileutils.CalculateSHA256Checksum(screenfile.Name())
				fileDescription := &models.FileDescription{
					Name:       screenshotfilename,
					Checksum:   checksum,
					SourcePath: "BOF SCREENSHOT",
					SavePath:   screenfile.Name(),
					Command:    "",
					Size:       int64(len(bofResp.Data[4:])),
				}
				fileJson, err := fileDescription.ToJsonString()
				err = db.SaveContext(&clientpb.Context{
					Task:    taskpb,
					Session: session,
					Type:    consts.ContextScreenShot,
					Value:   fileJson,
				})
				if err != nil {
					return fmt.Sprintf("Failed to save file: %s", err.Error())
				}
				return fmt.Sprintf("Screenshot saved to %s", screenfile.Name())
			}()
		case CALLBACK_FILE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				fileDir := filepath.Join(configs.ContextPath, sessionId, consts.DownloadPath)
				if !fileutils.Exist(fileDir) {
					if err := os.MkdirAll(fileDir, os.ModePerm); err != nil {
						logs.Log.Errorf("failed to create session directory: %s", err.Error())
					}
				}
				fileName := filepath.Base(string(bofResp.Data[8:]))
				fullPath := filepath.Join(fileDir, fileName)
				file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Sprintf("Could not open file '%s' (ID: %s): %s", filepath.Base(file.Name()), fileId, err)
				}
				fileMap[fileId] = file
				return fmt.Sprintf("File '%s' (ID: %s) opened successfully", filepath.Base(file.Name()), fileId)
			}()
		case CALLBACK_FILE_WRITE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				file := fileMap[fileId]
				if file == nil {
					return fmt.Sprintf("No open file to write to (ID: %s)", fileId)
				}
				_, err = file.Write(bofResp.Data[4:])
				if err != nil {
					return fmt.Sprintf("Error writing to file (ID: %s): %s", fileId, err)
				}
				return fmt.Sprintf("Data(Size: %d) written to file (ID: %s) successfully", bofResp.Length-4, fileId)
			}()
		case CALLBACK_FILE_CLOSE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				file := fileMap[fileId]
				fileName := file.Name()
				if file == nil {
					return fmt.Sprintf("No open file to close (ID: %s)", fileId)
				}
				checksum, _ := fileutils.CalculateSHA256Checksum(file.Name())
				fileDescription := &models.FileDescription{
					Name:       filepath.Base(fileName),
					Checksum:   checksum,
					SourcePath: "BOF FILE",
					SavePath:   file.Name(),
					Command:    "",
					Size:       int64(len(bofResp.Data[4:])),
				}
				fileJson, err := fileDescription.ToJsonString()
				err = db.SaveContext(&clientpb.Context{
					Task:    taskpb,
					Session: session,
					Type:    consts.ContextScreenShot,
					Value:   fileJson,
				})
				err = file.Close()
				if err != nil {
					return fmt.Sprintf("Error closing file (ID: %s): %s", fileId, err)
				}
				delete(fileMap, fileId)
				return fmt.Sprintf("File '%s' (ID: %s) closed successfully", filepath.Base(fileName), fileId)
			}()
		default:
			result = func() string {
				return fmt.Sprintf("Unimplemented callback type : %d", bofResp.CallbackType)
			}()
		}
		results.WriteString(result + "\n")
	}
	// Close any remaining open files
	for fileId, file := range fileMap {
		if file != nil {
			err := file.Close()
			if err != nil {
				results.WriteString(fmt.Sprintf("Error closing file (ID: %s): %s\n", fileId, err))
			} else {
				results.WriteString(fmt.Sprintf("File (ID: %s) closed automatically due to end of processing\n", fileId))
			}
			delete(fileMap, fileId)
		}
	}

	return results.String()
}
