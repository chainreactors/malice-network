package cli

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui/mux"
	"github.com/spf13/cobra"
)

// startMux launches the terminal multiplexer after the user has already logged
// in on the real terminal. All child panes reuse the same auth config.
//
// The first pane (id=0, "index") gets full event output, MCP, and LocalRPC.
// Subsequent panes get --quiet for a clean, distraction-free experience.
func startMux(cmd *cobra.Command, con *core.Console) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	configPath := con.ConfigPath
	if configPath == "" {
		return fmt.Errorf("no config path recorded; use --auth to specify")
	}

	paneCounter := 0
	var mu sync.Mutex

	// Base args builder: auth + quiet for non-index panes.
	buildArgs := func(sessionID string) []string {
		mu.Lock()
		idx := paneCounter
		paneCounter++
		mu.Unlock()

		args := []string{"--mux-child", "--auth", configPath}
		if idx > 0 {
			args = append(args, "--quiet")
		}
		// Forward --mcp/--rpc only to the index pane.
		if idx == 0 {
			if mcp, _ := cmd.Flags().GetString("mcp"); mcp != "" {
				args = append(args, "--mcp", mcp)
			}
			if rpc, _ := cmd.Flags().GetString("rpc"); rpc != "" {
				args = append(args, "--rpc", rpc)
			}
			// Forward root-level --use to the index pane (e.g. --mux --use <sid>).
			if sessionID == "" {
				if rootUse, _ := cmd.Flags().GetString("use"); rootUse != "" {
					sessionID = rootUse
				}
			}
		}
		if sessionID != "" {
			args = append(args, "--use", sessionID)
		}
		return args
	}

	m := mux.New(
		mux.WithSidebarWidth(22),

		// Generic pane factory: creates a new console without a pre-selected session.
		mux.WithPaneFactory(func(id int, w, h int) (*mux.TermPane, error) {
			args := buildArgs("")
			name := fmt.Sprintf("console-%d", id)
			return mux.NewTermPane(id, name, exe, args, w, h)
		}),

		// Session pane factory: creates a pane that auto-uses a specific session.
		// Triggered when user types `use <session>` in the index pane (via OSC).
		mux.WithSessionPaneFactory(func(id int, sessionID string, w, h int) (*mux.TermPane, error) {
			args := buildArgs(sessionID)
			// Use a short session ID prefix as the pane name for readability.
			name := sessionID
			if len(name) > 8 {
				name = name[:8]
			}
			return mux.NewTermPane(id, name, exe, args, w, h)
		}),
	)

	// Background goroutine: update sidebar state from the mux process's own
	// gRPC connection (established during the login step above).
	go func() {
		for {
			if con.Server != nil {
				var alive int
				var sessions []mux.SessionInfo
				for _, s := range con.Sessions {
					if s.IsAlive {
						alive++
					}
					osShort := "?"
					if s.Os != nil {
						switch {
						case s.Os.Name == "windows":
							osShort = "win"
						case s.Os.Name == "linux":
							osShort = "lin"
						case s.Os.Name == "darwin":
							osShort = "mac"
						default:
							osShort = s.Os.Name
						}
						if len(osShort) > 3 {
							osShort = osShort[:3]
						}
					}
					sessions = append(sessions, mux.SessionInfo{
						ID:    s.SessionId,
						Name:  s.Note,
						OS:    osShort,
						Alive: s.IsAlive,
					})
				}
				m.SetSidebarState(mux.SidebarState{
					SessionAlive:  alive,
					SessionTotal:  len(con.Sessions),
					ListenerCount: len(con.Listeners),
					PipelineCount: len(con.Pipelines),
					Sessions:      sessions,
				})
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// Start the mux process's own event handler to keep sidebar state fresh.
	// Runs quietly — the mux process has no readline, so no console output.
	if con.Server != nil {
		go func() {
			for {
				if !con.Server.EventStatus {
					con.Server.Quiet = true
					con.EventHandler()
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	return m.Run()
}
