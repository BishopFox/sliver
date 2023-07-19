module github.com/bishopfox/sliver

go 1.20

replace github.com/rsteube/carapace v0.36.3 => github.com/reeflective/carapace v0.25.2-0.20230602202234-e8d757e458ca

require (
	filippo.io/age v1.1.1
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Binject/binjection v0.0.0-20200705191933-da1a50d7013d
	github.com/Binject/debug v0.0.0-20210312092933-6277045c2fdf
	github.com/Binject/go-donut v0.0.0-20210701074227-67a31e2d883e
	github.com/Binject/universal v0.0.0-20210304094126-daefaa886313
	github.com/Ne0nd0g/go-clr v1.0.3
	github.com/alecthomas/chroma v0.10.0
	github.com/cheggaaa/pb/v3 v3.1.2
	github.com/chromedp/cdproto v0.0.0-20230220211738-2b1ec77315c9
	github.com/chromedp/chromedp v0.9.1
	github.com/glebarez/sqlite v1.8.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/jedib0t/go-pretty/v6 v6.4.6
	github.com/kbinani/screenshot v0.0.0-20191211154542-3a185f1ce18f
	github.com/klauspost/compress v1.16.6
	github.com/lesnuages/go-winio v0.4.19
	github.com/lesnuages/snitch v0.6.0
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/miekg/dns v1.1.55
	github.com/moloch--/asciicast v0.1.0
	github.com/moloch--/memmod v0.0.0-20211120144554-8b37cc654945
	github.com/ncruces/go-sqlite3 v0.7.2
	github.com/reeflective/console v0.1.6
	github.com/reeflective/readline v1.0.8
	github.com/rsteube/carapace v0.36.3
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.2
	github.com/tetratelabs/wazero v1.2.1
	github.com/things-go/go-socks5 v0.0.3
	github.com/xlab/treeprint v1.2.0
	github.com/yiya1989/sshkrb5 v0.0.0-20201110125252-a1455b75a35e
	golang.org/x/crypto v0.10.0
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/net v0.11.0
	golang.org/x/sys v0.9.0
	golang.org/x/term v0.9.0
	golang.org/x/text v0.10.0
	golang.zx2c4.com/wireguard v0.0.0-20220316235147-5aff28b14c24
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20220208144051-fde48d68ee68
	google.golang.org/grpc v1.56.1
	google.golang.org/protobuf v1.31.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gorm.io/driver/mysql v1.5.1
	gorm.io/driver/postgres v1.5.2
	gorm.io/driver/sqlite v1.5.2
	gorm.io/gorm v1.25.2
	gvisor.dev/gvisor v0.0.0-20230504175454-7b0a1988a28f
	modernc.org/sqlite v1.23.1
	tailscale.com v1.44.0
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/Binject/shellcode v0.0.0-20191101084904-a8a90e7d4563 // indirect
	github.com/BurntSushi/xgb v0.0.0-20201008132610-5f9e7b3c49cd // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/VirusTotal/vt-go v0.0.0-20210528074736-45bbe34cc8ab // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/akutz/memconn v0.1.0 // indirect
	github.com/alexbrainman/sspi v0.0.0-20210105120005-909beea2cc74 // indirect
	github.com/awgh/cppgo v0.0.0-20210224085512-3d24bca8edc0 // indirect
	github.com/awgh/rawreader v0.0.0-20200626064944-56820a9c6da4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.22 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.33 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.10 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/demisto/goxforce v0.0.0-20160322194047-db8357535b1d // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/gen2brain/shm v0.0.0-20200228170931-49f9650110c5 // indirect
	github.com/glebarez/go-sqlite v1.21.1 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.1.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/illarion/gonotify v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/insomniacslk/dhcp v0.0.0-20230407062729-974c6f05fe16 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.3.1 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86 // indirect
	github.com/jsimonetti/rtnetlink v1.3.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kortschak/wol v0.0.0-20200729010619-da482cc4850a // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/sdnotify v1.0.0 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/ncruces/julianday v0.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/tailscale/certstore v0.1.1-0.20220316223106-78d6e1c49d8d // indirect
	github.com/tailscale/golang-x-crypto v0.0.0-20221115211329-17a3db2c30d2 // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/tailscale/wireguard-go v0.0.0-20230410165232-af172621b4dd // indirect
	github.com/tcnksm/go-httpstat v0.2.0 // indirect
	github.com/thedevsaddam/gojsonq/v2 v2.5.2 // indirect
	github.com/u-root/uio v0.0.0-20230305220412-3e8cd9d6bf63 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go4.org/mem v0.0.0-20220726221520-4f986261bf13 // indirect
	go4.org/netipx v0.0.0-20230303233057-f1b76eb4bb35 // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	inet.af/peercred v0.0.0-20210906144145-0893ea02156a // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
