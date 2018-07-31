package email

import (
	"testing"
)

var SMTPConf = map[string]interface{}{
	LabelHost:     "smtp.mailtrap.io",
	LabelPort:     2525,
	LabelUsername: " 302cf7c45f9d5f",
	LabelPassword: "05b7d0d170f3a8"}

func TestNewSMTPSender(t *testing.T) {
	send, err := NewSMTPSender(smtpconf)
	if err != nil {
		t.Error(err)
	} else if send == nil {
		t.Errorf("No error occurred but the sender is nil")
	}

	s, ok := send.(*SMTPSender)
	if !ok {
		t.Error("NewSMTPSender didn't return an *SMTPSender")
	}

	if s.d.Username != smtpconf[LabelUsername].(string) {
		t.Errorf("Usernames don't match. Expected %s, got %s",
			smtpconf[LabelUsername], s.d.Username)
	}
	if s.d.Password != smtpconf[LabelPassword].(string) {
		t.Errorf("Passwords don't match. Expected %s, got %s",
			smtpconf[LabelPassword], s.d.Password)
	}
	if s.d.Host != smtpconf[LabelHost].(string) {
		t.Errorf("Hosts don't match. Expected %s, got %s",
			smtpconf[LabelHost], s.d.Host)
	}
	if s.d.Port != smtpconf[LabelPort].(int) {
		t.Errorf("Ports don't match. Expected %d, got %d",
			smtpconf[LabelPort], s.d.Port)
	}
	if s.from != smtpconf[LabelFrom].(string) {
		t.Errorf("From addresses don't match. Expected %s, got %s",
			smtpconf[LabelFrom], s.from)
	}
}

func TestSMTPSend(t *testing.T) {
	t.Error("Test not implemented")
}
