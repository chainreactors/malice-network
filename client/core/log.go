package core

import (
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
)

var (
	LogLevel = logs.WarnLevel
	Stdout   = NewStdoutWrapper(os.Stdout)
	Log      = &Logger{Logger: NewLog(LogLevel)}
	MuteLog  = &Logger{Logger: NewLog(logs.ImportantLevel + 1)}
)

var (
	NewLine                    = "\x1b[1E"
	Debug           logs.Level = 10
	Warn            logs.Level = 20
	Info            logs.Level = 30
	Error           logs.Level = 40
	Important       logs.Level = 50
	GroupStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	NameStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultLogStyle            = map[logs.Level]string{
		Debug:     NewLine + tui.BlueBg.Bold(true).Render(tui.Rocket+"[+]") + " %s",
		Warn:      NewLine + tui.YellowBg.Bold(true).Render(tui.Zap+"[warn]") + " %s",
		Important: NewLine + tui.PurpleBg.Bold(true).Render(tui.Fire+"[*]") + " %s",
		Info:      NewLine + tui.GreenBg.Bold(true).Render(tui.HotSpring+"[i]") + " %s",
		Error:     NewLine + tui.RedBg.Bold(true).Render(tui.Monster+"[-]") + " %s",
	}
)

type LogEntry struct {
	Timestamp time.Time
	Data      []byte
}

type StdoutWrapper struct {
	stdout     io.Writer
	buffer     []LogEntry
	bufferSize int
	maxSize    int
	mu         sync.Mutex
}

func NewStdoutWrapper(stdout io.Writer) *StdoutWrapper {
	return &StdoutWrapper{
		stdout:     stdout,
		buffer:     make([]LogEntry, 0),
		bufferSize: 0,
		maxSize:    1000,
	}
}

// RemoveOldestEntries 删除指定数量的最早日志条目
func (w *StdoutWrapper) RemoveOldestEntries(count int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if count <= 0 {
		return
	}

	if count >= w.bufferSize {
		w.buffer = make([]LogEntry, 0)
		w.bufferSize = 0
		return
	}

	w.buffer = w.buffer[count:]
	w.bufferSize -= count
}

// Write 实现 io.Writer 接口
func (w *StdoutWrapper) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Data:      make([]byte, len(p)),
	}
	copy(entry.Data, p)

	if w.bufferSize >= w.maxSize {
		// 只删除一条最早的数据
		w.buffer = w.buffer[1:]
		w.bufferSize--
	}

	w.buffer = append(w.buffer, entry)
	w.bufferSize++

	return w.stdout.Write(p)
}

func (w *StdoutWrapper) GetBuffer() []LogEntry {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]LogEntry, len(w.buffer))
	copy(result, w.buffer)
	return result
}

func (w *StdoutWrapper) Range(start, end time.Time) string {
	w.mu.Lock()
	defer w.mu.Unlock()

	var result strings.Builder

	if start.After(end) {
		start, end = end, start
	}

	for _, entry := range w.buffer {
		if (entry.Timestamp.Equal(start) || entry.Timestamp.After(start)) &&
			(entry.Timestamp.Equal(end) || entry.Timestamp.Before(end)) {
			result.Write(entry.Data)
		}
	}

	return result.String()
}

func NewLog(level logs.Level) *logs.Logger {
	log := logs.NewLogger(level)
	log.SetFormatter(DefaultLogStyle)
	log.SetOutput(Stdout)
	return log
}

func NewLogger(filename string) *Logger {
	log := NewLog(LogLevel)
	if filename != "" {
		logFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			Log.Warnf("Failed to open log file: %v", err)
		}
		return &Logger{Logger: log, logFile: logFile}
	} else {
		return &Logger{Logger: log}
	}
}

type Logger struct {
	*logs.Logger
	logFile *os.File
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (l *Logger) FileLog(s string) {
	if l.logFile != nil {
		l.logFile.WriteString(ansi.ReplaceAllString(s, ""))
		l.logFile.Sync()
	}
}
