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

// Attention is the number of missed or failed messages not yet dismissed.
func (d listData) Attention() int {
	n := 0
	for _, m := range d.Messages {
		if m.NeedsAck() {
			n++
		}
	}
	return n
}

// indexData embeds listData so index.html can render the "list" template with
// the same dot: Messages and Attention resolve through the embedded struct.
type indexData struct {
	listData
	Contacts []session.Contact
	Session  sessionData
}

type sessionData struct {
	State  session.State
	QR     template.URL
	Paired bool // pairing happened in this process; the page's contact list predates it
}

func (s *Server) sessionData() sessionData {
	d := sessionData{State: s.session.State(), Paired: s.session.Paired()}
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
		listData: listData{Messages: msgs},
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
	// datetime-local input has no zone. The browser posts its IANA zone in
	// send_zone; interpret the wall-clock time in that zone and store UTC. Fall
	// back to the machine's local zone when the browser sends nothing usable.
	zone := r.PostForm.Get("send_zone")
	loc := time.Local
	if zone != "" {
		if l, err := time.LoadLocation(zone); err == nil {
			loc = l
		} else {
			zone = "" // unknown name; fall back and do not store junk
		}
	}
	sendAt, err := time.ParseInLocation("2006-01-02T15:04", r.PostForm.Get("send_at"), loc)
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
		SendZone:      zone,
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

// mutate parses the path id and applies op to it, then renders the list. An
// unparseable id names nothing, so there is nothing to do: both handlers fall
// through to the list, which shows the current truth either way.
func (s *Server) mutate(w http.ResponseWriter, r *http.Request, what string, op func(context.Context, int64) error) {
	if id, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err == nil {
		if err := op(r.Context(), id); err != nil {
			s.fail(w, what, err)
			return
		}
	}
	s.renderList(w, r)
}

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	s.mutate(w, r, "delete message", s.store.Delete)
}

func (s *Server) ackMessage(w http.ResponseWriter, r *http.Request) {
	s.mutate(w, r, "acknowledge message", func(ctx context.Context, id int64) error {
		return s.store.Acknowledge(ctx, id, time.Now())
	})
}

// contactName returns the known primary name for jid, or jid itself when unknown.
func (s *Server) contactName(ctx context.Context, jid string) string {
	for _, c := range s.session.Contacts(ctx) {
		if c.JID == jid {
			return c.Primary
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
