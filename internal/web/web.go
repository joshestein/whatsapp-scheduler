package web

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/message"
	"github.com/joshestein/whatsapp-scheduler/internal/session"
	"github.com/joshestein/whatsapp-scheduler/internal/store"
	"github.com/skip2/go-qrcode"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	store   *store.Store
	session *session.Session
	log     *slog.Logger
	tmpl    *template.Template
}

func New(st *store.Store, sess *session.Session, log *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{store: st, session: sess, log: log, tmpl: tmpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(staticFS))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /messages", s.createMessage)
	mux.HandleFunc("DELETE /messages/{id}", s.deleteMessage)
	mux.HandleFunc("POST /messages/{id}/ack", s.ackMessage)
	mux.HandleFunc("GET /session", s.sessionPartial)
	mux.HandleFunc("GET /healthz", s.healthz)
	return mux
}

type listData struct {
	Messages []message.Message
}

type indexData struct {
	Messages []message.Message
	Contacts []session.Contact
	Session  sessionData
}

type sessionData struct {
	State session.State
	QR    template.URL
}

func (s *Server) sessionData() sessionData {
	d := sessionData{State: s.session.State()}
	if code := s.session.QR(); code != "" {
		png, err := qrcode.Encode(code, qrcode.Medium, 256)
		if err != nil {
			s.log.Error("qr encode", "err", err)
		} else {
			d.QR = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
		}
	}
	return d
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.store.List(r.Context())
	if err != nil {
		s.fail(w, "list messages", err)
		return
	}
	s.render(w, "index", indexData{
		Messages: msgs,
		Contacts: s.session.Contacts(r.Context()),
		Session:  s.sessionData(),
	})
}

func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.formError(w, "bad form")
		return
	}
	jid, err := normaliseRecipient(r.PostForm.Get("recipient_jid"))
	if err != nil {
		s.formError(w, "invalid recipient: enter a phone number with country code, or pick a contact")
		return
	}
	body := strings.TrimSpace(r.PostForm.Get("body"))
	if jid == "" || body == "" {
		s.formError(w, "recipient and message are required")
		return
	}
	// datetime-local input has no zone. Interpret it in the machine's local zone, store UTC
	sendAt, err := time.ParseInLocation("2006-01-02T15:04", r.PostForm.Get("send_at"), time.Local)
	if err != nil {
		s.formError(w, "invalid send time")
		return
	}
	now := time.Now()
	if sendAt.Before(now.Truncate(time.Minute)) {
		s.formError(w, "send time is in the past")
		return
	}
	name := s.contactName(r.Context(), jid)

	_, err = s.store.Create(r.Context(), message.Message{
		RecipientJID:  jid,
		RecipientName: name,
		Body:          body,
		SendAt:        sendAt.UTC(),
	}, time.Now())
	if err != nil {
		s.fail(w, "create message", err)
		return
	}

	msgs, err := s.store.List(r.Context())
	if err != nil {
		s.fail(w, "list messages", err)
		return
	}
	s.render(w, "list", listData{Messages: msgs})
}

func (s *Server) renderList(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.store.List(r.Context())
	if err != nil {
		s.fail(w, "list messages", err)
		return
	}
	s.render(w, "list", listData{Messages: msgs})
}

// An unparseable id names nothing, so there is nothing to do. Both handlers
// fall through to the list, which shows the current truth either way.

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if id, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err == nil {
		if err := s.store.Delete(r.Context(), id); err != nil {
			s.fail(w, "delete message", err)
			return
		}
	}
	s.renderList(w, r)
}

func (s *Server) ackMessage(w http.ResponseWriter, r *http.Request) {
	if id, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err == nil {
		if err := s.store.Acknowledge(r.Context(), id, time.Now()); err != nil {
			s.fail(w, "acknowledge message", err)
			return
		}
	}
	s.renderList(w, r)
}

// contactName returns the known name for jid, or jid itself when unknown.
func (s *Server) contactName(ctx context.Context, jid string) string {
	for _, c := range s.session.Contacts(ctx) {
		if c.JID == jid {
			return c.Name
		}
	}
	return jid
}

func (s *Server) sessionPartial(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "session", s.sessionData())
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"session":%q}`, s.session.State())
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		s.log.Error("render", "template", name, "err", err)
	}
}

// formError answers a POST with a validation message. htmx is configured in
// index.html to swap 422 responses; the headers steer the text into #form-error
// instead of the form's normal target, so the list is left untouched.
func (s *Server) formError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Retarget", "#form-error")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.WriteHeader(http.StatusUnprocessableEntity)
	template.HTMLEscape(w, []byte(msg))
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	s.log.Error(what, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
