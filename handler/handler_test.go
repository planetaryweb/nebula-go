package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.shadow53.com/BluestNight/nebula-forms/errors"
)

func config() interface{} {
	return map[string]interface{} {
		LabelHoneypot: "pot",
		LabelAllowedDomain: "example.com",
		LabelHandleIf: map[string]interface{} {
			"name": true,
			"email": true,
			"favorite-nums": []interface{}{"1", "14", "19", "19"},
			"empty": []interface{}{""}}}
}

func fakeBody() url.Values {
	body := url.Values{}
	body.Add("name", "Joe Smith")
	body.Add("email", "joe.smith@example.com")
	body.Add("favorite-nums", "1")
	body.Add("favorite-nums", "14")
	body.Add("favorite-nums", "19")
	return body
}

func fakeRequest(body url.Values) *http.Request {
	if body == nil {
		body = fakeBody()
	}
	req := httptest.NewRequest(
		http.MethodPost, "https://example.com/forms/test", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "example.com")
	return req
}

func TestBase_Unmarshal(t *testing.T) {
	h := Base{}
	conf := config()
	err := h.Unmarshal(conf)
	if err != nil {
		t.Error(err)
	}

	// Honeypot must be a string
	conf.(map[string]interface{})[LabelHoneypot] = 12
	err = h.Unmarshal(conf)
	if err == nil {
		t.Errorf("Unmarshaling should fail if %s is not a string",
			LabelHoneypot)
	} else {
		t.Logf("Following error should be because %s was not a string: %s",
			LabelHoneypot, err)
	}

	// Honeypot is not necessary
	delete(conf.(map[string]interface{}), LabelHoneypot)
	err = h.Unmarshal(conf)
	if err != nil {
		t.Errorf("Unmarshal should succeed without %s, failed with error: %s",
			LabelHoneypot, err)
	}

	// Domain must be a string
	conf.(map[string]interface{})[LabelAllowedDomain] = 12
	err = h.Unmarshal(conf)
	if err == nil {
		t.Errorf("Unmarshaling should fail if %s is not a string",
			LabelAllowedDomain)
	} else {
		t.Logf("Following error should be because %s was not a string: %s",
			LabelAllowedDomain, err)
	}

	// Domain must be provided
	delete(conf.(map[string]interface{}), LabelAllowedDomain)
	err = h.Unmarshal(conf)
	if err == nil {
		t.Errorf("Unmarshaling should fail if %s is not present",
			LabelAllowedDomain)
	} else {
		t.Logf("Following error should be because %s was not present: %s",
			LabelAllowedDomain, err)
	}

	// Handler conditions should be correct
	if h.handleConditions["name"].AllowedValues != nil {
		t.Error("Handler condition set to a boolean value should have nil allowed values")
	}
	if !h.handleConditions["name"].MustBeNonEmpty {
		t.Error("Handler non-empty boolean was not set to true when it must")
	}
	if h.handleConditions["favorite-nums"].MustBeNonEmpty {
		t.Error("Handler condition with allowed values should not be set non-empty")
	}
	if len(h.handleConditions["favorite-nums"].AllowedValues) != 3 {
		t.Error("Input's allowed values must contain full list without repeats")
	}
}

func TestBase_ShouldHandle(t *testing.T) {
	h := Base{}
	h.domain = "*"
	h.honeypot = "pot"
	h.handleConditions = make(map[string]*handleCondition)
	h.handleConditions["name"] = &handleCondition{MustBeNonEmpty: true}
	h.handleConditions["email"] = &handleCondition{MustBeNonEmpty: true}
	h.handleConditions["empty"] = &handleCondition{AllowedValues:
		map[string]struct{}{"": {}}}
	h.handleConditions["favorite-nums"] = &handleCondition{
		AllowedValues:map[string]struct{}{
			"14": {},
			"1": {}}}

	body := fakeBody()
	req := fakeRequest(body)

	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Handler with fulfilled conditions failed to handle")
	}

	// Test with empty value for non-empty field
	body.Del("name")
	req = fakeRequest(body)

	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Handler with empty non-empty field should not handle")
	}

	// One of allowed values is found, should handle
	body.Add("name", "Joe Smith")
	body.Set("favorite-nums", "1")
	body.Add("favorite-nums", "19")
	req = fakeRequest(body)

	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Handler with some of allowed values failed to handle")
	}

	// None of allowed values is found, should not handle
	body.Set("favorite-nums", "19")
	req = fakeRequest(body)

	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Handler with none of allowed values should not handle")
	}

	// Value that must be empty is not empty, should not handle
	body.Add("favorite-nums", "1")
	body.Set("empty", "non-empty")
	req = fakeRequest(body)

	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Non-empty value when expecting only empty should not handle")
	}

	// Domain does not match, should not handle
	body.Del("empty")
	req = fakeRequest(body)
	h.domain = "baddomain.com"
	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Handler shouldn't handle when domains don't match")
	}

	// Domain matches, should handle
	h.domain = "example.com"
	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Handler should handle when domains match")
	}

	// Should not handle if honeypot has value
	body.Set("pot", "spamminess")
	req = fakeRequest(body)
	if ok, err := h.ShouldHandle(req); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Should not handle when honeypot has value")
	}

}

func TestFormValuesFunc(t *testing.T) {
	body := fakeBody()
	req := fakeRequest(nil)
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
