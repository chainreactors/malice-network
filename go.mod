module github.com/chainreactors/malice-network

go 1.21.4

require (
	filippo.io/age v1.1.1
	github.com/AlecAivazis/survey/v2 v2.0.5
	github.com/Binject/go-donut v0.0.0-20220908180326-fcdcc35d591c
	github.com/chainreactors/files v0.0.0-20231102192550-a652458cee26
	github.com/chainreactors/grumble v0.0.0-20240111103643-830a8bf7d70a
	github.com/chainreactors/logs v0.0.0-20240207121836-c946f072f81f
	github.com/chainreactors/tui v0.0.0-20240706070223-def1f112c52d
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.26.6
	github.com/charmbracelet/lipgloss v0.12.1
	github.com/fatih/color v1.17.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gookit/config/v2 v2.2.4
	github.com/jessevdk/go-flags v1.5.0
	github.com/klauspost/compress v1.17.0
	github.com/ncruces/go-sqlite3 v0.9.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pterm/pterm v0.12.69
	github.com/robfig/cron/v3 v3.0.0
	github.com/tetratelabs/wazero v1.5.0
	golang.org/x/crypto v0.15.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.4
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	github.com/Binject/debug v0.0.0-20210312092933-6277045c2fdf // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/charmbracelet/x/ansi v0.1.4 // indirect
	github.com/charmbracelet/x/input v0.1.0 // indirect
	github.com/charmbracelet/x/term v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.1.0 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/desertbit/closer/v3 v3.1.3 // indirect
	github.com/desertbit/columnize v2.1.0+incompatible // indirect
	github.com/desertbit/go-shlex v0.1.1 // indirect
	github.com/desertbit/readline v1.5.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/goccy/go-yaml v1.11.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gookit/goutil v0.6.14 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/ncruces/julianday v0.1.5 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1-0.20230530133925-c48e322e2a8f // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20231127185646-65229373498e // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
)

replace github.com/chainreactors/tui v0.0.0-20240706070223-def1f112c52d => ./external/tui

replace github.com/chainreactors/grumble v0.0.0-20240111103643-830a8bf7d70a => ./external/grumble
