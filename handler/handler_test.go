package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/BluestNight/static-forms/errors"
)

func fakeBody() url.Values {
	body := url.Values{}
	body.Add("name", "Joe Smith")
	body.Add("email", "joe.smith@example.com")
	body.Add("favorite-nums", "1")
	body.Add("favorite-nums", "14")
	body.Add("favorite-nums", "19")
	return body
}

func fakeRequest() *http.Request {
	req := httptest.NewRequest(
		http.MethodPost, "/forms/test", strings.NewReader(fakeBody().Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestFormValuesFunc(t *testing.T) {
	body := fakeBody()
	req := fakeRequest()
	f := FormValuesFunc(req)
	s, err := f("name")
	if err != nil {
		t.Error(err)
	} else if len(s) != 1 {
		t.Errorf(
			"Received wrong number of values for \"name\": expected %#v, got %#v",
			body.Get("name"), s)
	} else if s[0] != string(body["name"][0]) {
		t.Errorf("Wrong value for \"name\": expected %s, got %s",
			string(body.Get("name")[0]), s[0])
	}

	s, err = f("favorite-nums")
	if err != nil {
		t.Error(err)
	} else if len(s) != 3 {
		t.Errorf(
			"Received wrong number of values for \"favorite-nums\": expected %#v, got %#v",
			body.Get("favorite_nums"), s)
	} else {
		for i := range s {
			if s[i] != string(body["favorite-nums"][i]) {
				t.Errorf("Wrong value for \"favorite-nums\": expected %s, got %s",
					string(body.Get("favorite-nums")[i]), s[i])
			}
		}
	}

	s, err = f("no_exist")
	if err != nil {
		t.Error(err)
	} else if len(s) != 0 {
		t.Errorf(
			"Received wrong number of values for \"no_exist\": expected %#v, got %#v",
			body.Get("no_exist"), s)
	}
}

func TestErrorfFunc(t *testing.T) {
	// Ensure zero values exist
	var err errors.HTTPError
	if err.Status() != 0 {
		t.Error("HTTPError initialized with non-zero status code")
	}
	if err.Error() != "" {
		t.Error("HTTPError initialized with non-empty error message")
	}

	errorf := ErrorfFunc(&err)
	// Ensure zero values were not changed by creating the function
	if err.Status() != 0 {
		t.Error("HTTPError has non-zero status code after generating errorf")
	}
	if err.Error() != "" {
		t.Error("HTTPError has non-empty error message after generating errorf")
	}

	// Ensure return values are correct and error pointer was modified
	v, e := errorf("An error occurred")
	if v != nil {
		t.Error("errorf should return nil as first return value")
	}
	if e.Error() != "An error occurred" {
		t.Error("errorf should not modify the error message")
	}
	if err.Error() != e.Error() {
		t.Errorf(
			"HTTPError message should be same as returned error: got %s, expected %s",
			err.Error(), e.Error())
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("errorf represents invalid form inputs (400), not %d", err.Status())
	}

	v, e = errorf("A %s error occurred", "different")
	if v != nil {
		t.Error("errorf should return nil as first return value")
	}
	if e.Error() != "A different error occurred" {
		t.Error("errorf should not modify the error message")
	}
	if err.Error() != e.Error() {
		t.Errorf(
			"HTTPError message should be same as returned error: got %s, expected %s",
			err.Error(), e.Error())
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("errorf represents invalid form inputs (400), not %d", err.Status())
	}
}
