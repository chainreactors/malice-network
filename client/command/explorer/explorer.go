package explorer

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"
)

type stack struct {
	Push   func(int)
	Pop    func() int
	Length func() int
}

func newStack() stack {
	slice := make([]int, 0)
	return stack{
		Push: func(i int) {
			slice = append(slice, i)
		},
		Pop: func() int {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Length: func() int { return len(slice) },
	}
}

func (e *ExplorerModel) pushView(selected, min, max int) {
	e.selectedStack.Push(selected)
	e.minStack.Push(min)
	e.maxStack.Push(max)
}

func (e *ExplorerModel) popView() (int, int, int) {

	return e.selectedStack.Pop(), e.minStack.Pop(), e.maxStack.Pop()
}

func NewExplorer(files []os.DirEntry, con *console.Console) *ExplorerModel {
	fp := &ExplorerModel{
		FilePicker:    filepicker.New(),
		Files:         files,
		selected:      0,
		min:           0,
		max:           0,
		maxStack:      newStack(),
		selectedStack: newStack(),
		minStack:      newStack(),
		con:           con,

		isProgress: false,
	}
	fp.FilePicker.CurrentDirectory, _ = os.UserHomeDir()
	return fp
}

type ExplorerModel struct {
	FilePicker filepicker.Model
	Files      []os.DirEntry

	selected      int
	min           int
	max           int
	maxStack      stack
	selectedStack stack
	minStack      stack
	quitting      bool
	err           error

	// progress model
	progress   *tui.BarModel
	isProgress bool

	con *console.Console
}

func (e *ExplorerModel) Init() tea.Cmd {
	return nil
}

func (e *ExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var keyMsg = tea.KeyMsg{}
	var windowsMsg = tea.WindowSizeMsg{}
	var isKeymsg = true
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			e.quitting = true
			keyMsg = msg
			return e, tea.Quit
		case "down", "j":
			e.selected++
			if e.selected >= len(e.Files) {
				e.selected = len(e.Files) - 1
			}
			if e.selected > e.max {
				e.min++
				e.max++
			}
			keyMsg = msg
		case "up", "k":
			e.selected--
			if e.selected < 0 {
				e.selected = 0
			}
			if e.selected < e.min {
				e.min--
				e.max--
			}
			keyMsg = msg
		case "n":
			e.selected += e.FilePicker.Height
			if e.selected >= len(e.Files) {
				e.selected = len(e.Files) - 1
			}
			e.min += e.FilePicker.Height
			e.max += e.FilePicker.Height

			if e.max >= len(e.Files) {
				e.max = len(e.Files) - 1
				e.min = e.max - e.FilePicker.Height
			}
		case "m":
			e.selected -= e.FilePicker.Height
			if e.selected < 0 {
				e.selected = 0
			}
			e.min -= e.FilePicker.Height
			e.max -= e.FilePicker.Height

			if e.min < 0 {
				e.min = 0
				e.max = e.min + e.FilePicker.Height
			}
		case "backspace", "left":
			e.FilePicker.CurrentDirectory = filepath.Dir(e.FilePicker.CurrentDirectory)
			if e.selectedStack.Length() > 0 {
				e.selected, e.min, e.max = e.popView()
				err := sendLsRequest(e)
				if err != nil {
					return nil, nil
				}
			} else {
				e.selected = 0
				e.min = 0
				e.max = e.FilePicker.Height - 1
			}

		case "enter", "right":
			if len(e.Files) == 0 {
				break
			}
			f := e.Files[e.selected]
			info, err := f.Info()
			if err != nil {
				break
			}
			isSymlink := info.Mode()&os.ModeSymlink != 0
			isDir := f.IsDir()

			if isSymlink {
				symlinkPath, _ := filepath.EvalSymlinks(filepath.Join(e.FilePicker.CurrentDirectory, f.Name()))
				info, err := os.Stat(symlinkPath)
				if err != nil {
					break
				}
				if info.IsDir() {
					isDir = true
				}
			}

			if (!isDir && e.FilePicker.FileAllowed) || (isDir && e.FilePicker.DirAllowed) {
				if key.Matches(msg, e.FilePicker.KeyMap.Select) {
					e.FilePicker.Path = filepath.Join(e.FilePicker.CurrentDirectory, f.Name())
				}
			}

			if !isDir {
				break
			}

			e.FilePicker.CurrentDirectory = filepath.Join(e.FilePicker.CurrentDirectory, f.Name())
			e.pushView(e.selected, e.min, e.max)
			e.selected = 0
			e.min = 0
			e.max = e.FilePicker.Height - 1
			err = sendLsRequest(e)
			e.max = max(e.max, e.FilePicker.Height-1)
			if err != nil {
				return nil, nil
			}
		case "d":
			err := downloadRequest(e)
			if err != nil {
				return nil, nil
			}
			e.isProgress = true
		}
	case tea.WindowSizeMsg:
		windowsMsg = msg
		isKeymsg = false
	}
	var cmd tea.Cmd
	if isKeymsg {
		e.FilePicker, cmd = e.FilePicker.Update(keyMsg)
	} else {
		e.FilePicker, cmd = e.FilePicker.Update(windowsMsg)
	}
	return e, cmd
}

func (e *ExplorerModel) View() string {
	if e.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if e.err != nil {
		s.WriteString(e.FilePicker.Styles.DisabledFile.Render(e.err.Error()))
	}
	s.WriteString("Current Directory: " + e.FilePicker.CurrentDirectory + "\n\n" + e.FilePicker.View() + "\n")
	if e.isProgress {
		s.WriteString(e.progress.View())
	}
	return s.String()
}

type refreshMsg struct {
}

// SetFiles
func SetFiles(model interface{}, files []os.DirEntry) error {
	v := reflect.ValueOf(model)

	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("model is not a pointer")
	}

	v = v.Elem()

	filesField := v.FieldByName("files")

	if !filesField.IsValid() {
		return fmt.Errorf("cannot find files field")
	}

	if !filesField.CanSet() {
		reflect.NewAt(filesField.Type(), unsafe.Pointer(filesField.UnsafeAddr())).Elem().Set(reflect.ValueOf(files))
		return nil
	}

	filesField.Set(reflect.ValueOf(files))
	return nil
}

func sendLsRequest(e *ExplorerModel) error {
	done := make(chan error, 1)

	ctx := e.con.ActiveTarget.Context()
	sid := e.con.ActiveTarget.GetInteractive().SessionId
	lsTask, err := e.con.Rpc.Ls(ctx, &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: e.FilePicker.CurrentDirectory,
	})
	if err != nil {
		e.con.SessionLog(sid).Errorf("load directory error: %v", err)
		return err
	}
	e.con.AddCallback(lsTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		var dirEntries []os.DirEntry
		for _, protoFile := range resp.GetFiles() {
			dirEntries = append(dirEntries, ProtobufDirEntry{FileInfo: protoFile})
		}

		err := SetFiles(&e.FilePicker, dirEntries)
		if err != nil {
			e.err = err
			done <- err
			return
		}
		e.Files = dirEntries
		done <- nil
	})

	return <-done
}

func downloadRequest(e *ExplorerModel) error {
	e.progress = tui.NewBar()
	ctx := e.con.ActiveTarget.Context()
	sid := e.con.ActiveTarget.GetInteractive().SessionId
	f := e.Files[e.selected]
	path := filepath.Join(e.FilePicker.CurrentDirectory, f.Name())
	downloadTask, err := e.con.Rpc.Download(ctx, &implantpb.DownloadRequest{
		Name: f.Name(),
		Path: path,
	})
	if err != nil {
		e.con.SessionLog(sid).Errorf("download error: %v", err)
		return err
	}
	total := downloadTask.Total
	e.con.AddCallback(downloadTask.TaskId, func(msg proto.Message) {
		block := msg.(*implantpb.Spite).GetBlock()
		e.progress.SetProgressPercent(float64(block.BlockId+1) / float64(total))
		e.progress.Update(tui.ViewMsg{})
		if block.BlockId+1 == uint32(total) {
			e.isProgress = false
		}
	})
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
