package web

import (
	"context"
	"embed"
	"encoding/base64"
	"html/template"
	"log/slog"
	"net/http"
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
	mux.HandleFunc("GET /session", s.sessionPartial)
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
	contacts, err := s.session.Contacts(r.Context())
	if err != nil {
		s.log.Warn("contacts", "err", err) // page still renders, datalist empty
	}
	s.render(w, "index", indexData{Messages: msgs, Contacts: contacts, Session: s.sessionData()})
}

func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	jid := strings.TrimSpace(r.PostForm.Get("recipient_jid"))
	name := strings.TrimSpace(r.PostForm.Get("recipient_name"))
	body := strings.TrimSpace(r.PostForm.Get("body"))
	if jid == "" || body == "" {
		http.Error(w, "recipient and message are required", http.StatusBadRequest)
		return
	}
	// datetime-local input has no zone. Interpret it in the machine's local zone, store UTC
	sendAt, err := time.ParseInLocation("2006-01-02T15:04", r.PostForm.Get("send_at"), time.Local)
	if err != nil {
		http.Error(w, "invalid send time", http.StatusBadRequest)
		return
	}
	if name == "" {
		name = jid
	}

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

func (s *Server) sessionPartial(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "session", s.sessionData())
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		s.log.Error("render", "template", name, "err", err)
	}
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	s.log.Error(what, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
