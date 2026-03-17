module github.com/chainreactors/malice-network

go 1.25.0

require (
	filippo.io/age v1.2.1
	github.com/carapace-sh/carapace v1.10.3
	github.com/chainreactors/IoM-go v0.0.0-20251104165543-b0edd05b982e
	github.com/chainreactors/files v0.0.0-20240716182835-7884ee1e77f0
	github.com/chainreactors/logs v0.0.0-20250312104344-9f30fa69d3c9
	github.com/chainreactors/mals v0.0.0-20250717185731-227f71a931fa
	github.com/chainreactors/parsers v0.0.0-20250225073555-ab576124d61f
	github.com/chainreactors/rem v0.2.4
	github.com/chainreactors/tui v0.1.1
	github.com/chainreactors/utils v0.0.0-20241209140746-65867d2f78b2
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/glamour v0.8.0
	github.com/charmbracelet/huh v0.8.0
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/corpix/uarand v0.2.0
	github.com/dustin/go-humanize v1.0.1
	github.com/evertras/bubble-table v0.19.2
	github.com/go-acme/lego/v4 v4.32.0
	github.com/go-lark/lark v1.14.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/snappy v1.0.0
	github.com/gookit/config/v2 v2.2.5
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/h2non/filetype v1.1.3
	github.com/jessevdk/go-flags v1.6.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.17.8
	github.com/mark3labs/mcp-go v0.25.0
	github.com/mattn/go-tty v0.0.7
	github.com/muesli/termenv v0.16.0
	github.com/ncruces/go-sqlite3 v0.9.0
	github.com/nikoksr/notify v0.41.0
	github.com/reeflective/console v0.0.0-00010101000000-000000000000
	github.com/reeflective/readline v1.1.3
	github.com/robfig/cron/v3 v3.0.0
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d
	github.com/samber/lo v1.49.1
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.9
	github.com/tetratelabs/wazero v1.5.0
	github.com/traefik/yaegi v0.14.3
	github.com/wabzsy/gonut v1.0.0
	github.com/yuin/gopher-lua v1.1.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.25.10
	layeh.com/gopher-luar v1.0.11
)

// compatibility
require (
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.9+incompatible
	golang.org/x/crypto v0.48.0
	golang.org/x/exp v0.0.0-20241210194714-1829a127f884
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.40.0
	golang.org/x/text v0.34.0
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	dario.cat/mergo v1.0.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.15 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/tea v1.4.0 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/aliyun/credentials-go v1.4.7 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.8 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/blinkbean/dingtalk v1.1.3 // indirect
	github.com/carapace-sh/carapace-shlex v1.1.1 // indirect
	github.com/catppuccin/go v0.3.0 // indirect
	github.com/cbroglie/mustache v1.4.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/chainreactors/fingers v0.0.0-20240702104653-a66e34aa41df // indirect
	github.com/chainreactors/go-metrics v0.0.0-20220926021830-24787b7a10f8 // indirect
	github.com/chainreactors/proxyclient v1.0.2 // indirect
	github.com/charmbracelet/bubbles v0.21.1-0.20250623103423-23b8fd6302d7 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260316091819-b93f6a3b8502 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/conpty v0.1.1 // indirect
	github.com/charmbracelet/x/errors v0.0.0-20240508181413-e8d8b6e2de86 // indirect
	github.com/charmbracelet/x/exp/ordered v0.1.0 // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/vt v0.0.0-20260316093931-f2fb44ab3145 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/charmbracelet/x/xpty v0.1.3 // indirect
	github.com/cjoudrey/gluahttp v0.0.0-20201111170219-25003d9adfa9 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/creack/pty v1.1.24 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/facebookincubator/nvdtools v0.1.5 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/go-acme/alidns-20150109/v4 v4.7.0 // indirect
	github.com/go-dedup/megophone v0.0.0-20170830025436-f01be21026f5 // indirect
	github.com/go-dedup/simhash v0.0.0-20170904020510-9ecaca7b509c // indirect
	github.com/go-dedup/text v0.0.0-20170907015346-8bb1b95e3cb7 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible // indirect
	github.com/goccy/go-yaml v1.12.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gookit/goutil v0.6.15 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.13-0.20220915233716-71ac16282d12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/ncruces/julianday v0.1.5 // indirect
	github.com/nrdcg/dnspod-go v0.4.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/riobard/go-bloom v0.0.0-20200614022211-cdc8013cb5b3 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/saferwall/pe v1.5.6 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/secDre4mer/pkcs7 v0.0.0-20240322103146-665324a4461d // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.5 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tengattack/gluacrypto v0.0.0-20240324200146-54b58c95c255 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/wabzsy/compression v0.0.0-20230725232933-73109bacf457 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yuin/gluamapper v0.0.0-20150323120927-d836955830e7 // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	github.com/yuin/goldmark-emoji v1.0.3 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	gopkg.in/ini.v1 v1.67.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
	mvdan.cc/sh/v3 v3.7.0 // indirect
)

replace (
	github.com/imdario/mergo => dario.cat/mergo v1.0.0
	github.com/miekg/dns => github.com/miekg/dns v1.1.58
	golang.org/x/crypto => golang.org/x/crypto v0.48.0
	golang.org/x/mod => golang.org/x/mod v0.17.0
	golang.org/x/net => golang.org/x/net v0.50.0
	golang.org/x/sync => golang.org/x/sync v0.19.0
	golang.org/x/sys => golang.org/x/sys v0.42.0
	golang.org/x/tools => golang.org/x/tools v0.21.0
)

replace (
	github.com/chainreactors/IoM-go => ./external/IoM-go
	github.com/chainreactors/proxyclient => github.com/chainreactors/proxyclient v1.0.3
	//github.com/chainreactors/rem => github.com/chainreactors/rem-community v0.2.4
	github.com/chainreactors/tui => ./external/tui
	github.com/mark3labs/mcp-go => ./external/mcp-go
	github.com/reeflective/console => ./external/console
	github.com/reeflective/readline => ./external/readline
	github.com/wabzsy/gonut => ./external/gonut
)
