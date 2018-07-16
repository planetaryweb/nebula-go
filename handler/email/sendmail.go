package email

import (
	"context"
	"os"
	"os/exec"

	"gopkg.in/gomail.v2"
)

const sendmail = "/usr/sbin/sendmail"

// SendmailSender provides a method of sending emails via the system
// `sendmail` command
type SendmailSender struct {}

// NewSendmailSender tests to see if the sendmail program exists on the system
// and errors if not found, otherwise returning a SendmailSender
func NewSendmailSender() (*SendmailSender, error) {
	if _, err := os.Stat(sendmail); os.IsNotExist(err) {
		return nil, err
	}
	return &SendmailSender{}, nil
}

// Send sends an email using the system `sendmail` command, assumed to be
// found at /usr/sbin/sendmail. Some systems alias other MTAs like Postfix
// to /usr/sbin/sendmail in some way, so this makes this Sender compatible
// with those programs as well.
func (s *SendmailSender) Send(ctx context.Context, msg *gomail.Message) error {
	// Start a sendmail process
	cmd := exec.CommandContext(ctx, sendmail, "-t")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Get a writable pipe to stdin
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// Start the sendmail command
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Write message to sendmail's stdin
	var errs [3]error
	_, errs[0] = msg.WriteTo(in)
	errs[1] = in.Close()
	errs[2] = cmd.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
