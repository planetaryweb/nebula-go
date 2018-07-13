package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"text/template"
	"time"

	e "github.com/BluestNight/static-forms/errors"
	"github.com/BluestNight/static-forms/handler"
	"github.com/Shadow53/interparser/parse"
	"gopkg.in/gomail.v2"
)

var funcMap = template.FuncMap{
	"FormValue": func(name string) string {
		return ""
	}}

// Type tells the main configuration which are email handlers
const Type = "email"

// "Global" varibles to help keep track of things
var senders = make(map[string]Sender)
var senderMux sync.Mutex // RWMutex?

// Configuration labels to avoid mistypes
// As public variables for easy customization by programs using this as
// a library
var (
	// LabelEmailSenders is the label for the collection of configurations for
	// senders of emails
	LabelEmailSenders = "email"
	LabelSenderType   = "type"
	LabelSubject      = "subject"
	LabelBody         = "body"
	LabelTo           = "to"
	LabelCC           = "cc"
	LabelBCC          = "bcc"
	LabelFrom         = "from"
	LabelFiles        = "files"
	LabelSender       = "sender"
	LabelReplyTo      = "reply_to"
)

// Sender represents anything that can send an email - an SMTP server, or
// a server-local sendmail binary, or mutt, or something else.
type Sender interface {
	Send(ctx context.Context, msg *gomail.Message) error
}

// Handler represents a handler for a particular form where the expected
// behavior is to send an email to someone.
type Handler struct {
	sender        Sender
	subject       string
	body          string
	to            string
	cc            string
	bcc           string
	replyTo       string
	from          string
	files         []string
	allowedDomain string
}

// NewSender creates a Sender that can be referenced later using the given name
func NewSender(name string, d interface{}) error {
	data, err := parse.MapStringKeys(d)
	if err != nil {
		return fmt.Errorf(e.ErrBaseConfig, name, err)
	}

	senderType, err := parse.String(data[LabelSenderType])
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelSenderType, name, err)
	}

	switch senderType {
	default:
		return errors.New("invalid email sender type")
	case "sendmail":
		return errors.New("sendmail email sender has not been created yet")
	case "smtp":
		sender, err := NewSMTPSender(d)
		if err != nil {
			return fmt.Errorf(e.ErrBaseConfig, name, err)
		}
		// Got the SMTP sender, add to map
		senderMux.Lock()
		senders[name] = sender
		senderMux.Unlock()
		return nil
	}
}

// NewHandler returns a Handler that sends an email on a form submission
func NewHandler(d interface{}) (*Handler, error) {
	data, err := parse.MapStringKeys(d)
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, "handler", err)
	}

	h := &Handler{}

	// Parse sender id
	sender, err := parse.String(data[LabelSender])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelSender, err)
	}
	var ok bool
	if h.sender, ok = senders[sender]; !ok {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelSender,
			"no sender exists with name "+sender)
	}

	// Parse subject line template string
	h.subject, err = parse.String(data[LabelSubject])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelSubject, err)
	}

	// Parse body template string
	h.body, err = parse.String(data[LabelBody])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelBody, err)
	}

	// Parse "to" field
	h.to, err = parse.String(data[LabelTo])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelTo, err)
	}

	// Parse Reply-To field, if exists
	h.replyTo = parse.StringOrDefault(data[LabelReplyTo], "")

	// Parse CC field, if exists
	h.cc = parse.StringOrDefault(data[LabelCC], "")

	// Parse bcc field, if exists
	h.bcc = parse.StringOrDefault(data[LabelBCC], "")

	// Parse "from" field, if exists
	h.from = parse.StringOrDefault(data[LabelFrom], "")

	// Parse files slice
	files := parse.SliceOrNil(data[LabelFiles])

	for _, f := range files {
		file, err := parse.String(f)
		if err != nil {
			return nil, fmt.Errorf(e.ErrConfigItem, LabelFiles, err)
		}
		h.files = append(h.files, file)
	}

	// Parse allowed domain
	h.allowedDomain, err = parse.String(data[handler.LabelAllowedDomain])
	if err != nil {
		return nil, fmt.Errorf(
			e.ErrConfigItem, handler.LabelAllowedDomain, err)
	}

	return h, nil
}

// AllowedDomain returns the domain allowed to access this handler
func (h Handler) AllowedDomain() string {
	return h.allowedDomain
}

// Handle parses the form submission and sends the generated email
func (h Handler) Handle(req *http.Request, ch chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	// Create Buffer as io.Writer for calls to Template.Execute
	buf := &bytes.Buffer{}
	msg := gomail.NewMessage()

	// Define all templates - must be defined here because they use the
	// FormValue method from the current Request
	// First define the FuncMap
	funcMap := template.FuncMap{
		"FormValue": req.FormValue}

	// Parse subject line template
	sTemp, err := template.New("subject").Funcs(funcMap).Parse(h.subject)
	if err != nil {
		ch <- err
		return
	}

	// Execute template - nil data because nothing to pass
	err = sTemp.Execute(buf, nil)
	if err != nil {
		ch <- err
		return
	}

	// Set Subject header and reset buffer
	msg.SetHeader("Subject", buf.String())
	buf.Reset()

	// Parse email body template
	bTemp, err := template.New("body").Funcs(funcMap).Parse(h.body)
	if err != nil {
		ch <- err
		return
	}

	err = bTemp.Execute(buf, nil)
	if err != nil {
		ch <- err
		return
	}

	// TODO: Give users the option of an HTML message
	msg.SetBody("text/plain", buf.String())
	buf.Reset()

	// Parse email To field

	toTemp, err := template.New("to").Funcs(funcMap).Parse(h.to)
	if err != nil {
		ch <- err
		return
	}

	err = toTemp.Execute(buf, nil)
	if err != nil {
		ch <- err
		return
	}

	msg.SetHeader("To", buf.String())
	buf.Reset()

	// Parse email Reply-To field
	if h.replyTo != "" {
		rTemp, err := template.New("replyto").Funcs(funcMap).Parse(h.replyTo)
		if err != nil {
			ch <- err
			return
		}

		err = rTemp.Execute(buf, nil)
		if err != nil {
			ch <- err
			return
		}

		msg.SetHeader("Reply-To", buf.String())
		buf.Reset()
	}

	// Parse email CC field

	if h.cc != "" {
		ccTemp, err := template.New("cc").Funcs(funcMap).Parse(h.cc)
		if err != nil {
			ch <- err
			return
		}

		err = ccTemp.Execute(buf, nil)
		if err != nil {
			ch <- err
			return
		}

		msg.SetHeader("Cc", buf.String())
		buf.Reset()
	}

	if h.bcc != "" {
		bccTemp, err := template.New("bcc").Funcs(funcMap).Parse(h.bcc)
		if err != nil {
			ch <- err
			return
		}

		err = bccTemp.Execute(buf, nil)
		if err != nil {
			ch <- err
			return
		}

		msg.SetHeader("Bcc", buf.String())
		buf.Reset()
	}

	if h.from != "" {
		fromTemp, err := template.New("from").Funcs(funcMap).Parse(h.from)
		if err != nil {
			ch <- err
			return
		}

		err = fromTemp.Execute(buf, nil)
		if err != nil {
			ch <- err
			return
		}

		msg.SetHeader("From", buf.String())
		buf.Reset()
	}

	// Attach any files from the form
	// Won't run if files slice is nil
	for _, file := range h.files {
		f, fh, err := req.FormFile(file)
		// Have to check for error manually because an EOF is fine
		// but other errors are not, and io.EOF == "EOF", not the full
		// text checked below
		if err != nil {
			ch <- err
			return
		}

		// Using empty file name for file because name and contents are
		// modified by functions
		msg.Attach("", gomail.Rename(fh.Filename),
			gomail.SetCopyFunc(func(w io.Writer) error {
				buf, err := ioutil.ReadAll(f)
				if err != nil {
					return err
				}
				_, err = w.Write(buf)
				return err
			}))
	}

	// Send email
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	err = h.sender.Send(ctx, msg)
	cancel()
	if err != nil {
		ch <- err
		return
	}
}
