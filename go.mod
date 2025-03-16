module github.com/chainreactors/malice-network

go 1.20

require (
	filippo.io/age v1.1.1
	github.com/carapace-sh/carapace v1.7.1
	github.com/chainreactors/logs v0.0.0-20250312104344-9f30fa69d3c9
	github.com/chainreactors/mals v0.0.0-20250312123103-4c3242132d76
	github.com/chainreactors/parsers v0.0.0-20250225073555-ab576124d61f
	github.com/chainreactors/rem v0.1.2-0.20250316181909-86daead65710
	github.com/chainreactors/tui v0.0.0-20250117083346-8eff1b67016e
	github.com/chainreactors/utils v0.0.0-20241209140746-65867d2f78b2
	github.com/charmbracelet/bubbletea v0.27.1
	github.com/charmbracelet/glamour v0.8.0
	github.com/charmbracelet/lipgloss v0.13.0
	github.com/dustin/go-humanize v1.0.1
	github.com/evertras/bubble-table v0.17.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/snappy v0.0.4
	github.com/gookit/config/v2 v2.2.5
	github.com/jessevdk/go-flags v1.6.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.17.0
	github.com/mattn/go-tty v0.0.7
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/termenv v0.15.3-0.20240618155329-98d742f6907a
	github.com/ncruces/go-sqlite3 v0.9.0
	github.com/nikoksr/notify v0.41.0
	github.com/pkg/errors v0.9.1
	github.com/reeflective/console v0.0.0-00010101000000-000000000000
	github.com/robfig/cron/v3 v3.0.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.6
	github.com/tetratelabs/wazero v1.5.0
	github.com/traefik/yaegi v0.14.3
	github.com/wabzsy/gonut v1.0.0
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/crypto v0.33.0
	golang.org/x/text v0.22.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.4
)

// compatibility
require (
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.9+incompatible
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63
	google.golang.org/grpc v1.57.2
	google.golang.org/protobuf v1.34.1
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	dario.cat/mergo v1.0.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Binject/debug v0.0.0-20230508195519-26db73212a7a // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/blinkbean/dingtalk v1.1.3 // indirect
	github.com/carapace-sh/carapace-shlex v1.0.1 // indirect
	github.com/cbroglie/mustache v1.4.0 // indirect
	github.com/chainreactors/files v0.0.0-20240716182835-7884ee1e77f0 // indirect
	github.com/chainreactors/fingers v0.0.0-20240702104653-a66e34aa41df // indirect
	github.com/chainreactors/go-metrics v0.0.0-20220926021830-24787b7a10f8 // indirect
	github.com/charmbracelet/bubbles v0.18.0 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/charmbracelet/x/ansi v0.1.4 // indirect
	github.com/charmbracelet/x/input v0.1.0 // indirect
	github.com/charmbracelet/x/term v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.1.0 // indirect
	github.com/cjoudrey/gluahttp v0.0.0-20201111170219-25003d9adfa9 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/facebookincubator/nvdtools v0.1.5 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/frankban/quicktest v1.14.6 // indirect
	github.com/go-dedup/megophone v0.0.0-20170830025436-f01be21026f5 // indirect
	github.com/go-dedup/simhash v0.0.0-20170904020510-9ecaca7b509c // indirect
	github.com/go-dedup/text v0.0.0-20170907015346-8bb1b95e3cb7 // indirect
	github.com/go-lark/lark v1.14.1 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible // indirect
	github.com/goccy/go-yaml v1.12.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gookit/goutil v0.6.15 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/ncruces/julianday v0.1.5 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/reeflective/readline v1.1.2 // indirect
	github.com/riobard/go-bloom v0.0.0-20200614022211-cdc8013cb5b3 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sahilm/fuzzy v0.1.1-0.20230530133925-c48e322e2a8f // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.5 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/tengattack/gluacrypto v0.0.0-20240324200146-54b58c95c255 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/wabzsy/compression v0.0.0-20230725232933-73109bacf457 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/gluamapper v0.0.0-20150323120927-d836955830e7 // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	github.com/yuin/goldmark-emoji v1.0.3 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230530153820-e85fd2cbaebc // indirect
	gotest.tools/v3 v3.5.1 // indirect
	layeh.com/gopher-luar v1.0.11 // indirect
	mvdan.cc/sh/v3 v3.7.0 // indirect
)

replace (
	github.com/reeflective/console => ./external/console
	github.com/reeflective/readline => ./external/readline
	github.com/wabzsy/gonut => ./external/gonut
)
