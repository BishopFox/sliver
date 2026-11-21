package magebuild

// GoConfig is the compiler configuration accepted by the built-in Sliver
// Magefile. Its field names intentionally match gogo.GoConfig JSON output.
type GoConfig struct {
	ProjectDir  string
	GOOS        string
	GOARCH      string
	GOROOT      string
	GOCACHE     string
	GOMODCACHE  string
	GOPROXY     string
	CGO         string
	CC          string
	CXX         string
	HTTPPROXY   string
	HTTPSPROXY  string
	Obfuscation bool
	GOGARBLE    string
}

// GoManifest is the v1 input consumed by the built-in Sliver Magefile target.
// SourceDir and Destination must point into one per-build workspace. The
// generic executor and output acceptance layer validate these paths.
type GoManifest struct {
	Version      int      `json:"version"`
	GoConfig     GoConfig `json:"goConfig"`
	MaterialPath string   `json:"materialPath,omitempty"`
	SourceDir    string   `json:"sourceDir"`
	Destination  string   `json:"destination"`
	BuildMode    string   `json:"buildMode,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	LDFlags      []string `json:"ldFlags,omitempty"`
	GCFlags      string   `json:"gcFlags,omitempty"`
	ASMFlags     string   `json:"asmFlags,omitempty"`
	ResultPath   string   `json:"resultPath"`
}

// Result is a location hint, never an authority for artifact type or hash.
// Primary is a basename relative to the result file's directory.
type Result struct {
	Primary     string `json:"primary"`
	DisplayName string `json:"displayName"`
}
