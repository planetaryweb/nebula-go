package errors

import (
	"errors"
	"http"
	"testing"
)

func TestNewHTTPError(t *testing.T) {
	err := NewHTTPError("error", http.StatusOK)
	if err.Error() != "error" {
		t.Errorf("Error string should be \"error\", not \"%s\"", err.Error())
	}

	if err.Status() != http.StatusOK {
		t.Errorf("Status code should be %d, not %d", http.StatusOK, err.Status())
	}
}

func TestHTTPErrorToChan(t *testing.T) {
	// Non-default status code
	httperr := NewHTTPError("error", http.StatusNoContent)
	err := errors.New("non-http error")
	// Very non-default status code
	def := http.StatusTeapot
	// Buffered so no goroutines are needed
	ch := make(chan HTTPError, 2)

	// Call for both errors
	HTTPErrorToChan(ch, httperr, def)
	HTTPErrorToChan(ch, err, def)

  for e := <-ch {
    switch (e.Status()) {
    case httperr.Status():
      if e.Error() != httperr.Error() {
        t.Errorf("Error \"%s\" does not match expected error \"%s\"",
          e.Error(), httperr.Error())
      }
    case def:
      if e.Error() != err.Error() {
        t.Errorf("Error \"%s\" does not match expected error \"%s\"",
          e.Error(), err.Error())
      }
    default:
      t.Errorf("Unexpected status code: %d", e.Status())
    }
  }
}
