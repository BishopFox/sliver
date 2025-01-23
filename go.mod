module github.com/bishopfox/sliver

go 1.22.0

toolchain go1.22.2

replace github.com/rsteube/carapace v0.36.3 => github.com/reeflective/carapace v0.46.3-0.20231214181515-27e49f3c3b69

require (
	filippo.io/age v1.1.1
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Binject/binjection v0.0.0-20210701074423-605d46e35deb
	github.com/Binject/debug v0.0.0-20230508195519-26db73212a7a
	github.com/Binject/go-donut v0.0.0-20220908180326-fcdcc35d591c
	github.com/Binject/universal v0.0.0-20220519011857-bea739e758c0
	github.com/Ne0nd0g/go-clr v1.0.3
	github.com/alecthomas/chroma v0.10.0
	github.com/cheggaaa/pb/v3 v3.1.5
	github.com/chromedp/cdproto v0.0.0-20240426225625-909263490071
	github.com/chromedp/chromedp v0.9.5
	github.com/fatih/color v1.17.0
	github.com/glebarez/sqlite v1.11.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/jedib0t/go-pretty/v6 v6.5.9
	github.com/kbinani/screenshot v0.0.0-20191211154542-3a185f1ce18f
	github.com/klauspost/compress v1.17.9
	github.com/lesnuages/go-winio v0.4.19
	github.com/lesnuages/snitch v0.6.0
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/miekg/dns v1.1.59
	github.com/moloch--/asciicast v0.1.0
	github.com/moloch--/memmod v0.0.0-20230225130813-fd77d905589e
	github.com/ncruces/go-sqlite3 v0.20.3
	github.com/reeflective/console v0.1.15
	github.com/reeflective/readline v1.0.14
	github.com/rsteube/carapace v0.46.3-0.20231214181515-27e49f3c3b69
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/tetratelabs/wazero v1.8.2
	github.com/things-go/go-socks5 v0.0.5
	github.com/ulikunitz/xz v0.5.12
	github.com/xlab/treeprint v1.2.0
	github.com/yiya1989/sshkrb5 v0.0.1
	golang.org/x/crypto v0.32.0
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f
	golang.org/x/net v0.33.0
	golang.org/x/sys v0.29.0
	golang.org/x/term v0.28.0
	golang.org/x/text v0.21.0
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
	google.golang.org/grpc v1.64.1
	google.golang.org/protobuf v1.34.1
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gorm.io/driver/mysql v1.5.7
	gorm.io/driver/postgres v1.5.9
	gorm.io/driver/sqlite v1.5.5
	gorm.io/gorm v1.25.10
	gvisor.dev/gvisor v0.0.0-20240306221502-ee1e1f6070e3
	modernc.org/sqlite v1.30.1
	tailscale.com v1.66.4
)

require (
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Binject/shellcode v0.0.0-20191101084904-a8a90e7d4563 // indirect
	github.com/BurntSushi/xgb v0.0.0-20201008132610-5f9e7b3c49cd // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/VirusTotal/vt-go v0.0.0-20210528074736-45bbe34cc8ab // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/akutz/memconn v0.1.0 // indirect
	github.com/alexbrainman/sspi v0.0.0-20231016080023-1a75b4708caa // indirect
	github.com/awgh/cppgo v0.0.0-20210224085512-3d24bca8edc0 // indirect
	github.com/awgh/rawreader v0.0.0-20200626064944-56820a9c6da4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.5 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.44.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/coreos/go-iptables v0.7.1-0.20240112124308-65c67c9f46e6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dblohm7/wingoes v0.0.0-20240119213807-a09d6be7affa // indirect
	github.com/demisto/goxforce v0.0.0-20160322194047-db8357535b1d // indirect
	github.com/digitalocean/go-smbios v0.0.0-20180907143718-390a4f403a8e // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.5.0 // indirect
	github.com/gaissmai/bart v0.4.1 // indirect
	github.com/gen2brain/shm v0.0.0-20200228170931-49f9650110c5 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-json-experiment/json v0.0.0-20231102232822-2e55bd4e08b0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.3.2 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20230522191255-76236955d466 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/nftables v0.2.1-0.20240414091927-5e242ec57806 // indirect
	github.com/gorilla/csrf v1.7.2 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hdevalence/ed25519consensus v0.2.0 // indirect
	github.com/illarion/gonotify v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/insomniacslk/dhcp v0.0.0-20231206064809-8c70d406f6d2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86 // indirect
	github.com/jsimonetti/rtnetlink v1.4.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kortschak/wol v0.0.0-20200729010619-da482cc4850a // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/sdnotify v1.0.0 // indirect
	github.com/mdlayher/socket v0.5.0 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rsteube/carapace-shlex v0.1.1 // indirect
	github.com/safchain/ethtool v0.3.0 // indirect
	github.com/tailscale/certstore v0.1.1-0.20231202035212-d3fa0460f47e // indirect
	github.com/tailscale/go-winio v0.0.0-20231025203758-c4f33415bf55 // indirect
	github.com/tailscale/golang-x-crypto v0.0.0-20240108194725-7ce1f622c780 // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/tailscale/peercred v0.0.0-20240214030740-b535050b2aa4 // indirect
	github.com/tailscale/web-client-prebuilt v0.0.0-20240226180453-5db17b287bf1 // indirect
	github.com/tailscale/wireguard-go v0.0.0-20240429185444-03c5a0ccf754 // indirect
	github.com/tcnksm/go-httpstat v0.2.0 // indirect
	github.com/thedevsaddam/gojsonq/v2 v2.5.2 // indirect
	github.com/u-root/uio v0.0.0-20240118234441-a3c409a6018e // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go4.org/mem v0.0.0-20220726221520-4f986261bf13 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.52.1 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
	mvdan.cc/sh/v3 v3.7.0 // indirect
	nhooyr.io/websocket v1.8.10 // indirect
)
