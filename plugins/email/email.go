package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"text/template"
	"time"

	e "gitlab.com/BluestNight/nebula-forms/errors"
	"gitlab.com/BluestNight/nebula-forms/handler"
	pb "gitlab.com/BluestNight/nebula-forms/proto"
	"gitlab.com/Shadow53/interparser/parse"
	"gopkg.in/gomail.v2"
)

// Configuration labels to avoid mistypes
// As public variables for easy customization by programs using this as
// a library
var (
	// LabelEmailSenders is the label for the collection of configurations for
	// senders of emails
	LabelSubject = "subject"
	LabelBody    = "body"
	LabelTo      = "to"
	LabelCC      = "cc"
	LabelBCC     = "bcc"
	LabelFrom    = "from"
	LabelFiles   = "files"
	LabelSender  = "sender"
	LabelReplyTo = "reply_to"
	LabelMethod  = "method"
)

// Sender represents anything that can send an email - an SMTP server, or
// a server-local sendmail binary, or mutt, or something else.
type Sender interface {
	Send(ctx context.Context, msg *gomail.Message) *e.HTTPError
}

// Handler represents a handler for a particular form where the expected
// behavior is to send an email to someone.
type Handler struct {
	handler.Base
	sender  Sender
	subject string
	body    string
	to      string
	cc      string
	bcc     string
	replyTo string
	from    string
	files   []string
}

// NewSender creates a Sender that can be referenced later using the given name
//func NewSender(name string, d interface{}) error {
//	data, err := parse.MapStringKeys(d)
//	if err != nil {
//		return fmt.Errorf(e.ErrBaseConfig, name, err)
//	}
//
//	senderType, err := parse.String(data[LabelSenderType])
//	if err != nil {
//		return fmt.Errorf(e.ErrConfigItem,
//			fmt.Sprintf("%s (%s)", LabelSenderType, name), err)
//	}
//
//	switch senderType {
//	default:
//		return errors.New("invalid email sender type")
//	case "smtp":
//		sender, err := NewSMTPSender(d)
//		if err != nil {
//			return fmt.Errorf(e.ErrBaseConfig, name, err)
//		}
//		// Got the SMTP sender, add to map
//		senderMux.Lock()
//		senders[name] = sender
//		senderMux.Unlock()
//	}
//	return nil
//}
//
//func Configure(data interface{}) error {
//	conf, err := parse.MapStringKeys(data)
//	if err != nil {
//		return err
//	}
//
//	for name, d := range conf {
//		err = NewSender(name, d)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

// NewHandler returns a Handler that sends an email on a form submission
func NewHandler(d interface{}) (*Handler, error) {
	data, err := parse.MapStringKeys(d)
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, "handler", err)
	}

	h := &Handler{}
	err = h.Unmarshal(d)
	if err != nil {
		return nil, err
	}

	// Parse sender id
	method, err := parse.String(data[LabelMethod])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelMethod, err)
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
	h.replyTo, err = parse.StringOrDefault(data[LabelReplyTo], "")
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelReplyTo, err)
	}

	// Parse CC field, if exists
	h.cc, err = parse.StringOrDefault(data[LabelCC], "")
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelCC, err)
	}

	// Parse bcc field, if exists
	h.bcc, err = parse.StringOrDefault(data[LabelBCC], "")
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelBCC, err)
	}

	// Parse "from" field, if exists
	h.from, err = parse.StringOrDefault(data[LabelFrom], "")
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelFrom, err)
	}

	if method == "sendmail" {
		// Sendmail requires "from"
		if h.from == "" {
			return nil, errors.New(
				"sendmail requires handlers to provide \"From\" address")
		}

		h.sender, err = NewSendmailSender()
		if err != nil {
			return nil, err
		}
	} else if method == "smtp" {
		if data[LabelSender] != nil {
			sender, err := NewSMTPSender(data[LabelSender])
			if err != nil {
				return nil, fmt.Errorf(e.ErrConfigItem, LabelSender, err)
			}

			if sender.from == "" {
				return nil, fmt.Errorf(
					"\"from\" needs to be set on handler and/or SMTP sender")
			}

			h.sender = sender
		} else {
			return nil, fmt.Errorf("method \"smtp\" requires [sender] to be defined")
		}
	} else {
		return nil, errors.New("invalid email sending method")
	}

	// Parse files slice
	files, err := parse.SliceOrNil(data[LabelFiles])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelFiles, err)
	}

	for _, f := range files {
		file, err := parse.String(f)
		if err != nil {
			return nil, fmt.Errorf(e.ErrConfigItem, LabelFiles, err)
		}
		h.files = append(h.files, file)
	}

	return h, nil
}

// Handle parses the form submission and sends the generated email
func (h Handler) Handle(req *pb.HTTPRequest) *e.HTTPError {
	// Create Buffer as io.Writer for calls to Template.Execute
	buf := &bytes.Buffer{}
	msg := gomail.NewMessage()

	// Error pointer containing whatever HTTPError occurred while templating
	tErr := &e.HTTPError{}

	// Define all templates - must be defined here because they use the
	// FormValue method from the current Request
	// First define the FuncMap
	funcMap := template.FuncMap{
		"Errorf":     handler.ErrorfFunc(tErr),
		"FormValue":  req.Form,
		"FormValues": handler.FormValuesFunc(req),
		"Matches":    regexp.MatchString}

	// Parse subject line template
	sTemp, err := template.New("subject").Funcs(funcMap).Parse(h.subject)
	if err != nil {
		h.Logger.Error(err.Error())
		return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	// Execute template - nil data because nothing to pass
	err = sTemp.Execute(buf, handler.TemplateContext)
	if err != nil {
		h.Logger.Error(err.Error())
		if tErr.Status() != 0 {
			return tErr
		} else {
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}
	}

	// Set Subject header and reset buffer
	msg.SetHeader("Subject", buf.String())
	buf.Reset()

	// Parse email body template
	bTemp, err := template.New("body").Funcs(funcMap).Parse(h.body)
	if err != nil {
		h.Logger.Error(err.Error())
		return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	err = bTemp.Execute(buf, handler.TemplateContext)
	if err != nil {
		h.Logger.Error(err.Error())
		if tErr.Status() != 0 {
			return tErr
		} else {
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}
	}

	// TODO: Give users the option of an HTML message
	msg.SetBody("text/plain", buf.String())
	buf.Reset()

	// Parse email To field

	toTemp, err := template.New("to").Funcs(funcMap).Parse(h.to)
	if err != nil {
		h.Logger.Error(err.Error())
		return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	err = toTemp.Execute(buf, handler.TemplateContext)
	if err != nil {
		if tErr.Status() != 0 {
			return tErr
		} else {
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}
	}

	msg.SetHeader("To", buf.String())
	buf.Reset()

	// Parse email Reply-To field
	if h.replyTo != "" {
		rTemp, err := template.New("replyto").Funcs(funcMap).Parse(h.replyTo)
		if err != nil {
			h.Logger.Error(err.Error())
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}

		err = rTemp.Execute(buf, handler.TemplateContext)
		if err != nil {
			h.Logger.Error(err.Error())
			if tErr.Status() != 0 {
				return tErr
			} else {
				return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
			}
		}

		msg.SetHeader("Reply-To", buf.String())
		buf.Reset()
	}

	// Parse email CC field

	if h.cc != "" {
		ccTemp, err := template.New("cc").Funcs(funcMap).Parse(h.cc)
		if err != nil {
			h.Logger.Error(err.Error())
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}

		err = ccTemp.Execute(buf, handler.TemplateContext)
		if err != nil {
			h.Logger.Error(err.Error())
			if tErr.Status() != 0 {
				return tErr
			} else {
				return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
			}
		}

		msg.SetHeader("Cc", buf.String())
		buf.Reset()
	}

	if h.bcc != "" {
		bccTemp, err := template.New("bcc").Funcs(funcMap).Parse(h.bcc)
		if err != nil {
			h.Logger.Error(err.Error())
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}

		err = bccTemp.Execute(buf, handler.TemplateContext)
		if err != nil {
			h.Logger.Error(err.Error())
			if tErr.Status() != 0 {
				return tErr
			} else {
				return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
			}
		}

		msg.SetHeader("Bcc", buf.String())
		buf.Reset()
	}

	if h.from != "" {
		fromTemp, err := template.New("from").Funcs(funcMap).Parse(h.from)
		if err != nil {
			h.Logger.Error(err.Error())
			return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
		}

		err = fromTemp.Execute(buf, handler.TemplateContext)
		if err != nil {
			h.Logger.Error(err.Error())
			if tErr.Status() != 0 {
				return tErr
			} else {
				return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
			}
		}

		msg.SetHeader("From", buf.String())
		buf.Reset()
	}

	// Attach any files from the form
	// Won't run if files slice is nil
	for _, file := range h.files {
		for _, val := range req.Form[file].Values {
			f := val.GetFile()
			if f == nil {
				// TODO: What to do with invalid input?
				h.Logger.Error(err.Error())
				return e.NewHTTPError("Input field " + file + " is not a file type", http.StatusInternalServerError)
			}

			// Using empty file name for file because name and contents are
			// modified by functions
			msg.Attach("", gomail.Rename(f.FileName),
				gomail.SetCopyFunc(func(w io.Writer) error {
					_, err = w.Write(f.Contents)
					h.Logger.Error(err.Error())
					return err
				}))

		}
	}

	// Send email
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = h.sender.Send(ctx, msg)
	cancel()
	if err != nil {
		h.Logger.Error(err.Error())
		return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	return nil
}
