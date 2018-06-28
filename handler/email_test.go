package handler

var testConfig = map[string]interface{} {
    emailLabels.Host: "smtp.mailtrap.io",
    emailLabels.Port: 465,
    emailLabels.Username: "302cf7c45f9d5f",
    emailLabels.Password: "05b7d0d170f3a8"
}

func TestNewEmailHandler(t *testing.T) {
    h, err := NewEmailHandler(interface{}(testConfig))
    if err != nil {
        t.Error(err)
    }

    if h.dialer.Host != testConfig[emailLabels.Host].(string) {
        t.Errorf("failed to parse host. expected: %s, received: %s",
            testConfig[emailLabels.Host].(string), h.dialer.Host)
    }

    if h.dialer.Port != testConfig[emailLabels.Port].(int) {
        t.Errorf("failed to parse port. expected: %s, received: %s",
            testConfig[emailLabels.Port].(int), h.dialer.Port)
    }

    if h.dialer.Username != testConfig[emailLabels.Username].(string) {
        t.Errorf("failed to parse username. expected: %s, received: %s",
            testConfig[emailLabels.Username].(string), h.dialer.Username)
    }

    if h.dialer.Password != testConfig[emailLabels.Password].(string) {
        t.Errorf("failed to parse password. expected: %s, received: %s",
            testConfig[emailLabels.Password].(string), h.dialer.Password)
    }
}
