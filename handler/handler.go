package handler

import (
	"net/http"
	"sync"
)

var (
	// LabelHandlers is the label for the collection of form submission
	// handlers
	LabelHandlers = "handler"
	// LabelHandlerPath is the label for the path that a handler handles
	LabelHandlerPath = "path"
	// LabelAllowedDomains represents
	LabelAllowedDomain = "allowed_domain"
)

// Handler represents anything that can handle a form submission
type Handler interface {
	Handle(req *http.Request, ch chan error, wg *sync.WaitGroup)
	AllowedDomain() string
}
