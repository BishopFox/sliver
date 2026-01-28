module github.com/bishopfox/sliver

go 1.25.5

replace github.com/rsteube/carapace v0.36.3 => github.com/reeflective/carapace v0.46.3-0.20231214181515-27e49f3c3b69

replace github.com/reeflective/readline => github.com/moloch--/readline v0.0.0-20260128035102-9f373ad18989

replace github.com/reeflective/console => github.com/moloch--/console v0.0.0-20260128035037-680800942770

require (
	filippo.io/age v1.3.1
	github.com/Binject/binjection v0.0.0-20210701074423-605d46e35deb
	github.com/Binject/debug v0.0.0-20230508195519-26db73212a7a
	github.com/Binject/go-donut v0.0.0-20210701074227-67a31e2d883e
	github.com/Binject/universal v0.0.0-20220519011857-bea739e758c0
	github.com/Ne0nd0g/go-clr v1.0.3
	github.com/alecthomas/chroma v0.10.0
	github.com/charmbracelet/huh v0.8.0
	github.com/charmbracelet/huh/spinner v0.0.0-20251215014908-6f7d32faaff3
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/cheggaaa/pb/v3 v3.1.7
	github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d
	github.com/chromedp/chromedp v0.14.1
	github.com/fatih/color v1.18.0
	github.com/glebarez/sqlite v1.11.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/jedib0t/go-pretty/v6 v6.7.8
	github.com/kbinani/screenshot v0.0.0-20250624051815-089614a94018
	github.com/klauspost/compress v1.18.1
	github.com/lesnuages/go-winio v0.4.19
	github.com/lesnuages/snitch v0.6.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/miekg/dns v1.1.70
	github.com/moloch--/asciicast v0.1.1
	github.com/moloch--/memmod v0.0.0-20230225130813-fd77d905589e
	github.com/moloch--/sgn v0.0.4
	github.com/ncruces/go-sqlite3 v0.30.4
	github.com/reeflective/console v0.1.25
	github.com/reeflective/readline v1.1.4
	github.com/rsteube/carapace v0.50.2
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	github.com/tetratelabs/wazero v1.11.0
	github.com/things-go/go-socks5 v0.1.0
	github.com/ulikunitz/xz v0.5.15
	github.com/xlab/treeprint v1.2.0
	github.com/yiya1989/sshkrb5 v0.0.1
	golang.org/x/crypto v0.46.0
	golang.org/x/exp v0.0.0-20251002181428-27f1f14c8bb9
	golang.org/x/mod v0.31.0
	golang.org/x/net v0.48.0
	golang.org/x/sys v0.40.0
	golang.org/x/term v0.38.0
	golang.org/x/text v0.32.0
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.11
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.0
	gvisor.dev/gvisor v0.0.0-20250503011706-39ed1f5ac29c
	modernc.org/sqlite v1.39.0
	tailscale.com v1.92.5
)

require (
	filippo.io/hpke v0.4.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/creachadair/msync v0.7.1 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/pires/go-proxyproto v0.8.1 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Binject/shellcode v0.0.0-20191101084904-a8a90e7d4563 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/VirusTotal/vt-go v1.0.1 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/akutz/memconn v0.1.0 // indirect
	github.com/alexbrainman/sspi v0.0.0-20250919150558-7d374ff0d59e // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/awgh/cppgo v0.0.0-20210224085512-3d24bca8edc0 // indirect
	github.com/awgh/rawreader v0.0.0-20200626064944-56820a9c6da4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.2 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.65.1 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/carapace-sh/carapace v1.11.0
	github.com/carapace-sh/carapace-shlex v1.1.1 // indirect
	github.com/catppuccin/go v0.3.0 // indirect
	github.com/charmbracelet/bubbles v0.21.1-0.20250623103423-23b8fd6302d7 // indirect
	github.com/charmbracelet/bubbletea v1.3.10 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.10.1 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dblohm7/wingoes v0.0.0-20250822163801-6d8e6105c62d // indirect
	github.com/demisto/goxforce v0.0.0-20160322194047-db8357535b1d // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gaissmai/bart v0.25.0 // indirect
	github.com/gen2brain/shm v0.1.1 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20250910080747-cc2cfa0554c3 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20230522191255-76236955d466 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/nftables v0.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hdevalence/ed25519consensus v0.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jsimonetti/rtnetlink v1.4.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mark3labs/mcp-go v0.43.2
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mdlayher/netlink v1.8.0 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/moloch--/go-keystone v0.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-community/pro-bing v0.4.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rsteube/carapace-shlex v0.1.2 // indirect
	github.com/safchain/ethtool v0.6.2 // indirect
	github.com/tailscale/certstore v0.1.1-0.20231202035212-d3fa0460f47e // indirect
	github.com/tailscale/go-winio v0.0.0-20231025203758-c4f33415bf55 // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tailscale/hujson v0.0.0-20250605163823-992244df8c5a // indirect
	github.com/tailscale/peercred v0.0.0-20250107143737-35a0c7bd7edc // indirect
	github.com/tailscale/web-client-prebuilt v0.0.0-20250124233751-d4cd19a26976 // indirect
	github.com/tailscale/wireguard-go v0.0.0-20250716170648-1d0488a3d7da // indirect
	github.com/thedevsaddam/gojsonq/v2 v2.5.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go4.org/mem v0.0.0-20240501181205-ae6ca9944745 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	golang.org/x/tools v0.40.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.66.10 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	mvdan.cc/sh/v3 v3.12.0 // indirect
)
