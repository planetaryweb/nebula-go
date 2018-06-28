package handler

import "http"

type Handler interface {
    Handle(r http.Request) error
}
