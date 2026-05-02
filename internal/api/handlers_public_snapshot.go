package api

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"relay/internal/contracts"
	"relay/internal/lib"
)

//go:embed templates/*.gohtml
var publicTemplateFS embed.FS

var (
	publicSnapshotTmplOnce sync.Once
	publicSnapshotTmpl     *template.Template
	publicSnapshotTmplErr  error
)

const publicSnapshotPathPrefix = "/p/"

func loadPublicSnapshotTemplate() (*template.Template, error) {
	publicSnapshotTmplOnce.Do(func() {
		publicSnapshotTmpl, publicSnapshotTmplErr = template.ParseFS(publicTemplateFS, "templates/public_snapshot.gohtml")
	})
	return publicSnapshotTmpl, publicSnapshotTmplErr
}

type publicSnapshotPageData struct {
	ProjectName   string
	TaskSummary   string
	RenderedBody  string
	SchemaVersion string
	CreatedAt     string
	HTMLLang      string
	SnapshotLabel string
	TitleSuffix   string
	OGDescription string
	OGImageURL    string
	PublicURL     string
	FooterText    string
	FooterCTA     string
}

// handlePublicSnapshotPage handles GET /p/{token} (and GET /p/{token}/og.png
// via the same dispatcher). It is unauthenticated and must remain so.
func (h Handler) handlePublicSnapshotPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writePublicGoneHTML(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	token, sub, ok := parsePublicSnapshotPath(r.URL.Path)
	if !ok || token == "" {
		writePublicGoneHTML(w, http.StatusNotFound, "Snapshot not found")
		return
	}
	if sub == "og.png" {
		h.handlePublicSnapshotOGImage(w, r, token)
		return
	}
	if sub != "" {
		writePublicGoneHTML(w, http.StatusNotFound, "Snapshot not found")
		return
	}

	view, err := h.services.GetPublicSnapshot(r.Context(), token)
	if err != nil {
		if isPublicSnapshotMissing(err) {
			writePublicGoneHTML(w, http.StatusGone, "Snapshot is no longer public")
			return
		}
		writePublicGoneHTML(w, http.StatusInternalServerError, "Snapshot is temporarily unavailable")
		return
	}

	tmpl, err := loadPublicSnapshotTemplate()
	if err != nil {
		writePublicGoneHTML(w, http.StatusInternalServerError, "Snapshot is temporarily unavailable")
		return
	}

	publicURL := h.services.PublicURL("/p/" + token)
	ogImageURL := h.services.PublicURL("/p/" + token + "/og.png")
	messages := resolvePublicSnapshotMessages(r)
	data := publicSnapshotPageData{
		ProjectName:   view.Project.Name,
		TaskSummary:   firstNonEmptyString(view.Snapshot.TaskSummary, messages.DefaultTaskSummary),
		RenderedBody:  view.Snapshot.RenderedBody,
		SchemaVersion: view.Snapshot.SchemaVersion,
		CreatedAt:     view.Snapshot.CreatedAt.UTC().Format(time.RFC3339),
		HTMLLang:      messages.HTMLLang,
		SnapshotLabel: messages.SnapshotLabel,
		TitleSuffix:   messages.TitleSuffix,
		OGDescription: publicSnapshotOGDescription(view.Snapshot.RenderedBody, messages.OGDescription),
		OGImageURL:    ogImageURL,
		PublicURL:     publicURL,
		FooterText:    messages.FooterText,
		FooterCTA:     messages.FooterCTA,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		writePublicGoneHTML(w, http.StatusInternalServerError, "Snapshot is temporarily unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Vary", "Accept-Language, Cookie")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// handlePublicSnapshotOGImage handles GET /p/{token}/og.png.
func (h Handler) handlePublicSnapshotOGImage(w http.ResponseWriter, r *http.Request, token string) {
	png, err := h.services.PublicSnapshotOGImage(r.Context(), token)
	if err != nil {
		if isPublicSnapshotMissing(err) {
			writePublicGoneHTML(w, http.StatusGone, "Snapshot is no longer public")
			return
		}
		writePublicGoneHTML(w, http.StatusInternalServerError, "Snapshot is temporarily unavailable")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// handleSnapshotPublish handles POST /v1/snapshots/{id}/publish.
func (h Handler) handleSnapshotPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay snapshot publish", "METHOD_NOT_ALLOWED", "method not allowed", false))
		return
	}
	snapshotID, action, ok := parseSnapshotAdminPath(r.URL.Path)
	if !ok || action != "publish" || snapshotID == "" {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay snapshot publish", "NOT_FOUND", "unknown snapshot route", false, "path"))
		return
	}
	result, err := h.services.PublishPacketSnapshot(r.Context(), snapshotID)
	if err != nil {
		writeServiceError(w, "relay snapshot publish", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay snapshot publish", result))
}

// handleSnapshotRevoke handles POST /v1/snapshots/{id}/revoke.
func (h Handler) handleSnapshotRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay snapshot revoke", "METHOD_NOT_ALLOWED", "method not allowed", false))
		return
	}
	snapshotID, action, ok := parseSnapshotAdminPath(r.URL.Path)
	if !ok || action != "revoke" || snapshotID == "" {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay snapshot revoke", "NOT_FOUND", "unknown snapshot route", false, "path"))
		return
	}
	if err := h.services.RevokePacketSnapshotPublic(r.Context(), snapshotID); err != nil {
		writeServiceError(w, "relay snapshot revoke", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay snapshot revoke", map[string]string{"snapshot_id": snapshotID}))
}

// parsePublicSnapshotPath turns "/p/{token}" or "/p/{token}/{sub}" into
// (token, sub, true). Anything deeper is rejected so adversaries can't
// probe nested paths.
func parsePublicSnapshotPath(path string) (token string, sub string, ok bool) {
	trimmed := strings.TrimPrefix(path, publicSnapshotPathPrefix)
	if trimmed == path {
		return "", "", false
	}
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) > 2 {
		return "", "", false
	}
	token = parts[0]
	if len(parts) == 2 {
		sub = parts[1]
	}
	return token, sub, true
}

// parseSnapshotAdminPath parses /v1/snapshots/{id}/{action}.
func parseSnapshotAdminPath(path string) (snapshotID string, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/v1/snapshots/")
	if trimmed == path {
		return "", "", false
	}
	trimmed = strings.Trim(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func writePublicGoneHTML(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>" + html(message) + "</title><style>body{font-family:Nunito,sans-serif;background:#FAF8F3;color:#0E1A35;display:grid;place-items:center;min-height:100vh;margin:0;padding:24px;text-align:center}h1{font-family:Fraunces,Georgia,serif;font-size:36px;font-weight:600;letter-spacing:-0.02em}p{color:#4A5669;margin-top:12px}</style></head><body><div><h1>" + html(message) + "</h1><p>This snapshot is no longer available.</p></div></body></html>"))
}

func isPublicSnapshotMissing(err error) bool {
	var appErr lib.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "PUBLIC_SNAPSHOT_NOT_FOUND" || appErr.Code == "PACKET_SNAPSHOT_NOT_FOUND" || appErr.Code == "PROJECT_NOT_FOUND"
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func publicSnapshotOGDescription(renderedBody string, fallback string) string {
	if strings.TrimSpace(renderedBody) == "" {
		return fallback
	}
	return clipForOG(renderedBody, 160)
}

// clipForOG produces a single-line meta description from the rendered body.
func clipForOG(input string, limit int) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	runes := []rune(cleaned)
	if len(runes) <= limit {
		return cleaned
	}
	if limit <= 0 {
		return ""
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}

// html escapes for inline HTML output. We use html/template for the main
// page; this small helper is for the static 410 page string only.
func html(s string) string {
	return template.HTMLEscapeString(s)
}
