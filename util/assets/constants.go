package assets

const (
	goVersion      = "1.25.7"
	garbleVersion  = "1.25.7"
	zigVersion     = "0.15.2"
	zigSourceParam = "source=sliver"

	zigMinisignPublicKey = "RWSGOq2NVecA2UPNdBUZykf1CCb147pkmdtYxgb3Ti+JO/wCYvhbAb/U"

	goTotal     = 6
	zigTotal    = 6
	garbleTotal = 6
)

var defaultZigMirrors = []string{
	"https://pkg.machengine.org/zig",
	"https://zigmirror.hryx.net/zig",
	"https://zig.linus.dev/zig",
	"https://zig.squirl.dev",
	"https://zig.florent.dev",
	"https://zig.mirror.mschae23.de/zig",
	"https://zigmirror.meox.dev",
}

var goBloatPaths = []string{
	"AUTHORS",
	"CONTRIBUTORS",
	"PATENTS",
	"VERSION",
	"favicon.ico",
	"robots.txt",
	"SECURITY.md",
	"CONTRIBUTING.md",
	"README.md",
	"doc",
	"test",
	"api",
	"misc",
}
