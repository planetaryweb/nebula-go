package handler

import (
	"testing"
)

var body = url.Values{
  "name": []string{"Joe Smith"},
  "email": []string{"joe.smith@example.com"},
  "favorite-nums": []string{"1", "14", "19"}
}

func fakeRequest() *http.Request {
  return httptest.NewRequest(
    "POST", "/forms/test", strings.NewReader(body.Encode()))
}

func TestFormValuesFunc(t *testing.T) {
  f := FormValuesFunc(fakeRequest())
  s, err := f("name")
  if err != nil {
    t.Error(err)
  } else if len(s) != 1 {
    t.Errorf(
      "Received wrong number of values for \"name\": expected %#v, got %#v",
      body.Get("name"), s)
  } else if s[0] != body.Get("name")[0] {
    t.Errorf("Wrong value for \"name\": expected %s, got %s",
      body.Get("name")[0], s[0])
  }

  s, err := f("favorite_nums")
  if err != nil {
    t.Error(err)
  } else if len(s) != 3 {
    t.Errorf(
      "Received wrong number of values for \"favorite_nums\": expected %#v, got %#v",
      body.Get("favorite_nums"), s)
  } else {
    for i, _ := range s {
      if s[i] != body.Get("favorite_nums")[i] {
        t.Errorf("Wrong value for \"favorite_nums\": expected %s, got %s",
          body.Get("favorite_nums")[i], s[i])
      }
    }
  }

  s, err := f("no_exist")
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
  var err HTTPError
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

  v, err = errorf("A %s error occurred", "different")
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
