package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	dbmodels "github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var taskFileNamePattern = regexp.MustCompile(`^(\d+)_(\d+)$`)

type taskSpiteEntry struct {
	Index int
	Spite *implantpb.Spite
}

var bindWaitPingInterval = time.Second

const (
	defaultTaskQueryPageSize = 100
	maxTaskQueryPageSize     = 500
	maxExpandedTaskPageSize  = 10
)

func readTaskSpitesFromDisk(sessionID string, taskID uint32) ([]taskSpiteEntry, error) {
	taskDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sessionID, consts.TaskPath))
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	entries := make([]taskSpiteEntry, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := taskFileNamePattern.FindStringSubmatch(file.Name())
		if matches == nil {
			continue
		}

		fileTaskID, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil || uint32(fileTaskID) != taskID {
			continue
		}

		index, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		taskPath := filepath.Join(taskDir, file.Name())
		content, err := os.ReadFile(taskPath)
		if err != nil {
			logs.Log.Warnf("failed to read task file %s: %v", taskPath, err)
			continue
		}

		spite := &implantpb.Spite{}
		if err = proto.Unmarshal(content, spite); err != nil {
			logs.Log.Warnf("failed to unmarshal task file %s: %v", taskPath, err)
			continue
		}

		entries = append(entries, taskSpiteEntry{Index: index, Spite: spite})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Index < entries[j].Index
	})

	return entries, nil
}

func readTaskRequestFromDisk(sessionID string, taskID uint32) (*implantpb.Spite, int64, string, bool, error) {
	requestPath, err := taskRequestPathFor(sessionID, taskID)
	if err != nil {
		return nil, 0, "", false, err
	}
	content, err := os.ReadFile(requestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, "", false, nil
		}
		return nil, 0, "", false, err
	}
	spite := &implantpb.Spite{}
	if err := proto.Unmarshal(content, spite); err != nil {
		return nil, 0, "", false, err
	}
	sum := sha256.Sum256(content)
	return spite, int64(len(content)), hex.EncodeToString(sum[:]), true, nil
}

func taskQueryPageSize(req *clientpb.TaskQuery) (int, error) {
	expanded := req.GetIncludeRawRequest() || req.GetIncludeResults()
	if req.GetPageSize() == 0 {
		if expanded && len(req.GetTaskIds()) == 0 {
			return maxExpandedTaskPageSize, nil
		}
		return defaultTaskQueryPageSize, nil
	}

	pageSize := int(req.GetPageSize())
	if pageSize > maxTaskQueryPageSize {
		return 0, status.Errorf(codes.InvalidArgument, "page_size must be <= %d", maxTaskQueryPageSize)
	}
	if expanded && len(req.GetTaskIds()) == 0 && pageSize > maxExpandedTaskPageSize {
		return 0, status.Errorf(codes.InvalidArgument, "page_size must be <= %d when include_raw_request or include_results is set without task_ids", maxExpandedTaskPageSize)
	}
	return pageSize, nil
}

func taskQueryOffset(pageToken string) (int, error) {
	if pageToken == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(pageToken)
	if err != nil || offset < 0 {
		return 0, status.Error(codes.InvalidArgument, "invalid page_token")
	}
	return offset, nil
}

func buildTaskQuery(req *clientpb.TaskQuery) *db.TaskQuery {
	query := db.NewTaskQuery()
	if sessionID := req.GetSessionId(); sessionID != "" {
		query = query.WhereSessionID(sessionID)
	}
	if len(req.GetTaskIds()) > 0 {
		query = query.WhereSeqs(req.GetTaskIds())
	}
	return query
}

func queryTaskModels(req *clientpb.TaskQuery, pageSize, offset int) (db.Tasks, bool, error) {
	modelTasks, err := buildTaskQuery(req).
		OrderBy("created DESC").
		OrderBy("seq DESC").
		Limit(pageSize + 1).
		Offset(offset).
		Find()
	if err != nil {
		return nil, false, err
	}
	hasMore := len(modelTasks) > pageSize
	if hasMore {
		modelTasks = modelTasks[:pageSize]
	}
	return modelTasks, hasMore, nil
}

func buildTaskDetail(req *clientpb.TaskQuery, modelTask *dbmodels.Task) (*clientpb.TaskDetail, error) {
	task := modelTask.ToProtobuf()
	detail := &clientpb.TaskDetail{Task: task}

	var rawRequest *implantpb.Spite
	needRequest := req.GetIncludeRawRequest() || (req.GetIncludeRequestSummary() && task.GetRequestSummary() == "")
	if needRequest {
		spite, size, sha256Hex, hasRequest, err := readTaskRequestFromDisk(modelTask.SessionID, modelTask.Seq)
		if err != nil {
			return nil, err
		}
		if hasRequest {
			rawRequest = spite
			task.HasRequest = true
			task.RequestSize = size
			task.RequestSha256 = sha256Hex
			if req.GetIncludeRequestSummary() && task.RequestSummary == "" {
				summary, err := core.BuildTaskRequestSummaryJSON(spite)
				if err != nil {
					return nil, err
				}
				task.CommandSummary = core.BuildTaskCommandSummary(spite)
				task.RequestSummary = summary
			}
		}
	}

	if !req.GetIncludeRequestSummary() {
		task.RequestSummary = ""
	}
	if req.GetIncludeRawRequest() {
		detail.RawRequest = rawRequest
	}
	if req.GetIncludeResults() {
		entries, err := readTaskSpitesFromDisk(modelTask.SessionID, modelTask.Seq)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			detail.Results = append(detail.Results, entry.Spite)
		}
	}

	return detail, nil
}

func getTaskContextFromDisk(sess *core.Session, task *core.Task, index int32) (*clientpb.TaskContext, error) {
	entries, err := readTaskSpitesFromDisk(sess.ID, task.Id)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}

	var spite *implantpb.Spite
	if index == -1 {
		spite = entries[len(entries)-1].Spite
	} else {
		for _, entry := range entries {
			if entry.Index == int(index) {
				spite = entry.Spite
				break
			}
		}
	}

	if spite == nil {
		return nil, types.ErrNotFoundTaskContent
	}

	return &clientpb.TaskContext{
		Task:    task.ToProtobuf(),
		Session: sess.ToProtobufLite(),
		Spite:   spite,
	}, nil
}

func getAllTaskContextsFromDisk(sess *core.Session, task *core.Task) (*clientpb.TaskContexts, error) {
	entries, err := readTaskSpitesFromDisk(sess.ID, task.Id)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}

	spites := make([]*implantpb.Spite, 0, len(entries))
	for _, entry := range entries {
		spites = append(spites, entry.Spite)
	}

	return &clientpb.TaskContexts{
		Task:    task.ToProtobuf(),
		Session: sess.ToProtobufLite(),
		Spites:  spites,
	}, nil
}

// getTaskContextFromDB fetches task context from DB + disk when session is not in memory.
func getTaskContextFromDB(sessionID string, taskID uint32, index int32) (*clientpb.TaskContext, error) {
	dbSess, err := db.FindSession(sessionID)
	if err != nil || dbSess == nil {
		return nil, types.ErrNotFoundSession
	}
	dbTask, err := db.GetTaskBySessionAndSeq(sessionID, taskID)
	if err != nil {
		return nil, types.ErrNotFoundTask
	}
	entries, err := readTaskSpitesFromDisk(sessionID, dbTask.Seq)
	if err != nil || len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}
	var spite *implantpb.Spite
	if index == -1 {
		spite = entries[len(entries)-1].Spite
	} else {
		for _, e := range entries {
			if e.Index == int(index) {
				spite = e.Spite
				break
			}
		}
	}
	if spite == nil {
		return nil, types.ErrNotFoundTaskContent
	}
	return &clientpb.TaskContext{
		Task:    dbTask.ToProtobuf(),
		Session: dbSess.ToProtobuf(),
		Spite:   spite,
	}, nil
}

// getAllTaskContextFromDB fetches all task contexts from DB + disk when session is not in memory.
func getAllTaskContextFromDB(sessionID string, taskID uint32) (*clientpb.TaskContexts, error) {
	dbSess, err := db.FindSession(sessionID)
	if err != nil || dbSess == nil {
		return nil, types.ErrNotFoundSession
	}
	dbTask, err := db.GetTaskBySessionAndSeq(sessionID, taskID)
	if err != nil {
		return nil, types.ErrNotFoundTask
	}
	entries, err := readTaskSpitesFromDisk(sessionID, dbTask.Seq)
	if err != nil || len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}
	spites := make([]*implantpb.Spite, 0, len(entries))
	for _, e := range entries {
		spites = append(spites, e.Spite)
	}
	return &clientpb.TaskContexts{
		Task:    dbTask.ToProtobuf(),
		Session: dbSess.ToProtobuf(),
		Spites:  spites,
	}, nil
}

func getTaskContext(sess *core.Session, task *core.Task, index int32) (*clientpb.TaskContext, error) {
	var msg *implantpb.Spite
	var ok bool
	if index == -1 {
		msg, ok = sess.GetLastMessage(int(task.Id))
	} else {
		msg, ok = sess.GetMessage(int(task.Id), int(index))
	}

	if ok {
		return &clientpb.TaskContext{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobufLite(),
			Spite:   msg,
		}, nil
	}
	return nil, types.ErrNotFoundTaskContent
}

func (rpc *Server) GetTasks(ctx context.Context, req *clientpb.TaskRequest) (*clientpb.Tasks, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	if req.All {
		modelTasks, err := db.ListTasksBySession(req.SessionId)
		if err != nil {
			return nil, err
		}
		return modelTasks.ToProtobuf(), nil
	} else {
		sess, err := core.Sessions.Get(req.SessionId)
		if err != nil {
			// Fallback to DB when session is not in memory (e.g., dead session)
			modelTasks, dbErr := db.ListTasksBySession(req.SessionId)
			if dbErr != nil {
				return nil, types.ErrNotFoundSession
			}
			return modelTasks.ToProtobuf(), nil
		}
		return sess.Tasks.ToProtobuf(), nil
	}
}

func (rpc *Server) QueryTasks(ctx context.Context, req *clientpb.TaskQuery) (*clientpb.TaskDetails, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}

	pageSize, err := taskQueryPageSize(req)
	if err != nil {
		return nil, err
	}
	offset, err := taskQueryOffset(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	modelTasks, hasMore, err := queryTaskModels(req, pageSize, offset)
	if err != nil {
		return nil, err
	}

	resp := &clientpb.TaskDetails{
		Tasks: make([]*clientpb.TaskDetail, 0, len(modelTasks)),
	}
	for _, modelTask := range modelTasks {
		if modelTask == nil {
			continue
		}
		detail, err := buildTaskDetail(req, modelTask)
		if err != nil {
			return nil, err
		}
		resp.Tasks = append(resp.Tasks, detail)
	}
	if hasMore {
		resp.NextPageToken = strconv.Itoa(offset + len(resp.Tasks))
	}
	if req.GetIncludeTotalCount() {
		total, err := buildTaskQuery(req).Count()
		if err != nil {
			return nil, err
		}
		resp.TotalCount = total
	}
	return resp, nil
}

// tryGetContent tries to find task content from cache first, then from disk.
func tryGetContent(sess *core.Session, task *core.Task, index int32) (*clientpb.TaskContext, error) {
	content, err := getTaskContext(sess, task, index)
	if err == nil || !errors.Is(err, types.ErrNotFoundTaskContent) {
		return content, err
	}
	return getTaskContextFromDisk(sess, task, index)
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return getTaskContextFromDB(req.SessionId, req.TaskId, req.Need)
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	return tryGetContent(sess, task, req.Need)
}

func (rpc *Server) WaitTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		// Session not in memory (dead), try DB+disk directly
		return getTaskContextFromDB(req.SessionId, req.TaskId, req.Need)
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	_, total := task.Progress()
	if req.Need >= 0 && total >= 0 && int(req.Need) >= total {
		return nil, types.ErrTaskIndexExceed
	}

	for {
		if content, err := tryGetContent(sess, task, req.Need); err == nil {
			return content, nil
		} else if !errors.Is(err, types.ErrNotFoundTaskContent) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case _, ok := <-task.DoneCh:
			if ok {
				continue
			}
		case <-task.Ctx.Done():
		}

		// Final attempt after signal
		if content, err := tryGetContent(sess, task, req.Need); err == nil {
			return content, nil
		} else if !errors.Is(err, types.ErrNotFoundTaskContent) {
			return nil, err
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, types.ErrNotFoundTaskContent
	}
}

func (rpc *Server) WaitTaskFinish(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		// Session not in memory (dead), try DB+disk directly
		return getTaskContextFromDB(req.SessionId, req.TaskId, -1)
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	stopBindWaitPing := startBindWaitPing(ctx, sess, task)
	defer stopBindWaitPing()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-task.Ctx.Done():
		msg, ok := sess.GetLastMessage(int(task.Id))
		if ok {
			return &clientpb.TaskContext{
				Task:    task.ToProtobuf(),
				Session: sess.ToProtobufLite(),
				Spite:   msg,
			}, nil
		}

		content, err := getTaskContextFromDisk(sess, task, -1)
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, types.ErrNotFoundTaskContent) {
			return nil, err
		}
	}
	return nil, types.ErrNotFoundTaskContent
}

func startBindWaitPing(ctx context.Context, sess *core.Session, task *core.Task) context.CancelFunc {
	if sess == nil || task == nil || sess.Type != consts.BindPipeline {
		return func() {}
	}
	pingCtx, cancel := context.WithCancel(ctx)
	label := fmt.Sprintf("bind-wait-ping:%s:%d", sess.ID, task.Id)
	core.GoGuarded(label, func() error {
		ticker := time.NewTicker(bindWaitPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return nil
			case <-task.Ctx.Done():
				return nil
			case <-ticker.C:
				if bindPollingRunning(sess.ID) {
					continue
				}
				if err := sendBindPing(sess); err != nil {
					logs.Log.Debugf("bind wait ping failed for session %s task %d: %v", sess.ID, task.Id, err)
				}
			}
		}
	}, core.LogGuardedError(label))
	return cancel
}

func (rpc *Server) GetAllTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContexts, error) {
	if req == nil {
		return nil, types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return getAllTaskContextFromDB(req.SessionId, req.TaskId)
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}
	msgs, ok := sess.GetMessages(int(task.Id))
	if ok {
		return &clientpb.TaskContexts{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobufLite(),
			Spites:  msgs,
		}, nil
	}

	contexts, err := getAllTaskContextsFromDisk(sess, task)
	if err == nil {
		return contexts, nil
	}
	if errors.Is(err, types.ErrNotFoundTaskContent) {
		return nil, types.ErrNotFoundTask
	}
	return nil, fmt.Errorf("load task content from disk: %w", err)
}

func (rpc *Server) GetFiles(ctx context.Context, req *clientpb.Session) (*clientpb.Files, error) {
	files, err := db.GetDownloadFiles(req.SessionId)
	if err != nil {
		return nil, err
	}
	return &clientpb.Files{
		Files: files,
	}, nil
}

func (rpc *Server) CancelTask(ctx context.Context, req *implantpb.TaskCtrl) (*clientpb.Task, error) {
	sess, err := getSession(ctx)
	if err != nil {
		return nil, err
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty, func(spite *implantpb.Spite) {
		task.CancelTask(spite, "")
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ListTasks(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleListTask, types.MsgTasks)
}

func (rpc *Server) QueryTask(ctx context.Context, req *implantpb.TaskCtrl) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgTask)
	return greq.Task.ToProtobuf(), nil
}
