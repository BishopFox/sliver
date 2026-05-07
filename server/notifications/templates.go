package notifications

import (
	"bytes"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db/models"
)

const (
	templateTypeText = "text"
	templateTypeHTML = "html"
)

type templateSpec struct {
	name string
	typ  string
}

type templateRenderer struct {
	baseDir       string
	textTemplates map[string]*texttemplate.Template
	htmlTemplates map[string]*htmltemplate.Template
	mutex         sync.RWMutex
}

type templateData struct {
	EventType      string
	Session        *core.Session
	Beacon         *models.Beacon
	Job            *core.Job
	Client         *core.Client
	Error          string
	DefaultSubject string
	DefaultMessage string
	Timestamp      time.Time
}

func templatesDir() string {
	return filepath.Join(assets.GetRootAppDir(), "notifications", "templates")
}

func ensureTemplatesDir() (string, error) {
	dir := templatesDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func newTemplateRenderer(baseDir string) *templateRenderer {
	return &templateRenderer{
		baseDir:       baseDir,
		textTemplates: map[string]*texttemplate.Template{},
		htmlTemplates: map[string]*htmltemplate.Template{},
	}
}

func (r *templateRenderer) render(spec templateSpec, data templateData) (rendered string, err error) {
	if spec.name == "" {
		return "", errors.New("template name is empty")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			notificationsLog.Errorf("Template render panic for %q: %v", spec.name, recovered)
			err = fmt.Errorf("template render panic: %v", recovered)
		}
	}()
	exec, err := r.getTemplate(spec)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := exec.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type templateExecutor interface {
	Execute(io.Writer, any) error
}

func (r *templateRenderer) getTemplate(spec templateSpec) (templateExecutor, error) {
	switch spec.typ {
	case templateTypeHTML:
		return r.getHTMLTemplate(spec.name)
	default:
		return r.getTextTemplate(spec.name)
	}
}

func (r *templateRenderer) getTextTemplate(name string) (templateExecutor, error) {
	r.mutex.RLock()
	if tmpl, ok := r.textTemplates[name]; ok {
		r.mutex.RUnlock()
		return tmpl, nil
	}
	r.mutex.RUnlock()

	path, err := resolveTemplatePath(r.baseDir, name)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tmpl, err := texttemplate.New(name).Parse(string(content))
	if err != nil {
		return nil, err
	}

	r.mutex.Lock()
	r.textTemplates[name] = tmpl
	r.mutex.Unlock()

	return tmpl, nil
}

func (r *templateRenderer) getHTMLTemplate(name string) (templateExecutor, error) {
	r.mutex.RLock()
	if tmpl, ok := r.htmlTemplates[name]; ok {
		r.mutex.RUnlock()
		return tmpl, nil
	}
	r.mutex.RUnlock()

	path, err := resolveTemplatePath(r.baseDir, name)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tmpl, err := htmltemplate.New(name).Parse(string(content))
	if err != nil {
		return nil, err
	}

	r.mutex.Lock()
	r.htmlTemplates[name] = tmpl
	r.mutex.Unlock()

	return tmpl, nil
}

func resolveTemplatePath(baseDir, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("template name is empty")
	}
	if strings.Contains(trimmed, "..") {
		return "", errors.New("template name contains invalid path segments")
	}
	if strings.ContainsAny(trimmed, `/\\`) {
		return "", errors.New("template name must not contain path separators")
	}
	cleanBase := filepath.Clean(baseDir)
	path := filepath.Join(cleanBase, filepath.Base(trimmed))
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) && cleanPath != cleanBase {
		return "", errors.New("template path escapes template directory")
	}
	return cleanPath, nil
}

func buildTemplateData(event core.Event, subject, message string) templateData {
	var errText string
	if event.Err != nil {
		errText = event.Err.Error()
	}
	return templateData{
		EventType:      event.EventType,
		Session:        event.Session,
		Beacon:         event.Beacon,
		Job:            event.Job,
		Client:         event.Client,
		Error:          errText,
		DefaultSubject: subject,
		DefaultMessage: message,
		Timestamp:      time.Now().UTC(),
	}
}

func parseTemplateType(value string) (string, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return templateTypeText, true
	}
	switch trimmed {
	case "text", "plain":
		return templateTypeText, true
	case "html":
		return templateTypeHTML, true
	default:
		return templateTypeText, false
	}
}
