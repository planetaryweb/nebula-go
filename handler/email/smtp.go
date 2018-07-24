package email

import (
	"context"
	"fmt"
	"net/http"

	e "github.com/BluestNight/static-forms/errors"
	"github.com/Shadow53/interparser/parse"
	"gopkg.in/gomail.v2"
)

const defaultSMTPPort int = 587

var (
	LabelUsername = "username"
	LabelPassword = "password"
	LabelHost     = "host"
	LabelPort     = "port"
)

// SMTPSender provides a method of sending an email message via an SMTP server
type SMTPSender struct {
	d    *gomail.Dialer
	from string
}

// NewSMTPSender creates a new Sender with populated fields based on
// the contents of the configurations passed as arguments.
//
// Arguments are passed as interfaces because NewSMTPSender is expected to
// be called while parsing configuration options, which are interfaces when
// parsed using Unmarshal methods.
//
// The second argument allows for a "global" configuration, e.g. a default
// email configuration to use if one isn't specified for this handler.
// "global" configurations may have the following specified:
// - username
// - password
// - host
// - port
// - from
func NewSMTPSender(d interface{}) (Sender, error) {
	data, err := parse.MapStringKeys(d)
	if err != nil {
		return nil, err
	}

	// Parse username
	username, err := parse.String(data[LabelUsername])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelUsername, err)
	}

	// Parse password
	password, err := parse.String(data[LabelPassword])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelPassword, err)
	}

	// Parse host
	// TODO: Validate host? They'll get errors anyways if it's invalid
	host, err := parse.String(data[LabelHost])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelHost, err)
	}

	// Parse port
	port, err := parse.Int64(data[LabelPort])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelPort, err)
	}

	sender := SMTPSender{}

	// NewDialer doesn't return any errors
	sender.d = gomail.NewDialer(host, int(port), username, password)

	// Parse from address
	sender.from, err = parse.String(data[LabelFrom])
	if err != nil {
		return nil, fmt.Errorf(e.ErrConfigItem, LabelFrom, err)
	}

	return sender, nil
}

// Send sends an email message after first attaching the files at the path(s)
// listed.
func (s SMTPSender) Send(ctx context.Context, msg *gomail.Message) *e.HTTPError {
	// Ensure there is a "from" field
	// Checking length instead of nil in case the slice is empty but non-nil
	if from := msg.GetHeader("From"); len(from) == 0 {
		msg.SetHeader("From", s.from)
	}

	// Attempt to send the message
	err := s.d.DialAndSend(msg)
	return e.NewHTTPError(err.Error(), http.StatusInternalServerError)
}
