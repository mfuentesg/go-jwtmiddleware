package jwtmiddleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
)

// token with HS256 algorithm
const jwtToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o"

func extractor(_ *http.Request) (string, error) {
	return "", nil
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func TestNew(t *testing.T) {
	m := New()

	if reflect.TypeOf(m) != reflect.TypeOf(new(Middleware)) {
		t.Errorf("expect an instance of Middleware struct, got %v", reflect.TypeOf(m))
	}

	if m.options.signKey != nil {
		t.Errorf("wrong default value, got %v", m.options.signKey)
	}

	if m.options.signingMethod != jwt.SigningMethodHS256 {
		t.Errorf("wrong default signingMethod got %v", m.options.signingMethod)
	}

	if reflect.TypeOf(m.options.userProperty) != reflect.TypeOf(UserProperty("")) {
		t.Errorf("expect an instance of jwtmiddleware.UserProperty type, got %v", reflect.TypeOf(m.options.userProperty))
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/fake", nil)
	m.options.errorHandler(w, r, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code expected 401, got %d", w.Code)
	}

	if w.Body.String() != "unauthorized" {
		t.Errorf("wrong response body expected `unauthorized` got `%s`", w.Body.String())
	}

	m = New(WithSignKey("secret"))
	if m.options.signKey.(string) != "secret" {
		t.Errorf("expected signKey `secret` got %v", m.options.signKey)
	}

	m = New(WithSigningMethod(jwt.SigningMethodES384))
	if m.options.signingMethod != jwt.SigningMethodES384 {
		t.Errorf("expected signKey `SigningMethodES384` got %v", m.options.signingMethod)
	}

	m = New(WithUserProperty("user"))
	if m.options.userProperty != "user" {
		t.Errorf("expected userProperty `user` got %v", m.options.userProperty)
	}

	m = New(WithErrorHandler(func(w http.ResponseWriter, r *http.Request, _ error) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/fake", nil)
	m.options.errorHandler(w, r, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("wrong status code expected 400, got %d", w.Code)
	}
}

func TestQueryStringExtractor(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake?jwt=token", nil)
	tests := map[string]string{
		"token": "",
		"":      "",
		"f":     "",
		"_":     "",
		"1":     "",
		":":     "",
		"jwt":   "token",
	}

	for test, expected := range tests {
		if token, _ := QueryStringExtractor(test)(r); token != expected {
			t.Errorf("expected token `%s` got `%s`", expected, token)
		}
	}
}

func TestBearerExtractor(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake", nil)

	if _, err := BearerExtractor(r); err == nil {
		t.Error("expected no header found error")
	}

	r.Header.Set("Authorization", "")
	if _, err := BearerExtractor(r); err == nil {
		t.Error("expected empty header error")
	}

	r.Header.Set("Authorization", "wrong")
	if _, err := BearerExtractor(r); err == nil {
		t.Error("expected token invalid format error")
	}

	// insensitive cases
	tests := [...]string{
		"authorization",
		"AUTHORIZATION",
		"AUthorizATION",
		"Authorization",
	}
	for _, test := range tests {
		r.Header.Set(test, "bearer token")
		if token, _ := BearerExtractor(r); token != "token" {
			t.Errorf("case %s: - invalid token expected `token` got `%s`", test, token)
		}
	}
}

func TestParseToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake", nil)
	r.Header.Set("authorization", "")
	if token, err := New(WithSignKey([]byte("secret"))).parseToken(r); err == nil {
		t.Errorf("expect extractor error due to empty token got %v", token)
	}

	if _, err := New(WithSignKey([]byte("secret")), WithExtractor(extractor)).parseToken(r); err == nil {
		t.Errorf("expect empty error due to wrong extractor implementation")
	}

	r.Header.Set("authorization", "bearer invalid.token.format")
	if _, err := New(WithSignKey([]byte("secret"))).parseToken(r); err == nil {
		t.Errorf("expect invalid token error")
	}
	r.Header.Set("authorization", fmt.Sprintf("bearer %s", jwtToken))
	if _, err := New(WithSigningMethod(jwt.SigningMethodES512), WithSignKey([]byte("secret"))).parseToken(r); err == nil {
		t.Errorf("expect token algorithm mistmatch")
	}

	if _, err := New(WithSignKey([]byte("secret"))).parseToken(r); err != nil {
		t.Errorf("expect valid token got %v", err)
	}
}

func TestHandler(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake", nil)
	r.Header.Set("authorization", fmt.Sprintf("bearer %s", jwtToken))
	tests := map[string]int{
		"wrong":  http.StatusUnauthorized,
		"secret": http.StatusOK,
	}

	for secret, expected := range tests {
		m := New(WithSignKey([]byte(secret)), WithUserProperty("user"))
		h := m.Handler(http.HandlerFunc(handlerFunc))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != expected {
			t.Errorf("wrong status code expected %d got %d", expected, w.Code)
		}
	}
}

func TestHandlerNext(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake", nil)
	r.Header.Set("authorization", fmt.Sprintf("bearer %s", jwtToken))
	tests := map[string]int{
		"wrong":  http.StatusUnauthorized,
		"secret": http.StatusOK,
	}

	for secret, expected := range tests {
		m := New(WithSignKey([]byte(secret)), WithUserProperty("user"))
		w := httptest.NewRecorder()
		m.HandlerNext(w, r, handlerFunc)
		if w.Code != expected {
			t.Errorf("wrong status code expected %d got %d", expected, w.Code)
		}
	}
}

func TestSetTokenToContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/fake", nil)
	token, _ := jwt.Parse(jwtToken, func(_ *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	r = setTokenToContext(r, "user", token)
	if r.Context().Value("user") == nil {
		t.Errorf("context doesn't contains the user key")
	}
	token = r.Context().Value("user").(*jwt.Token)
	if !token.Valid {
		t.Errorf("expected valid token got %v", token.Raw)
	}
	if token.Raw != jwtToken {
		t.Errorf("wrong token expected %s got %s", jwtToken, token.Raw)
	}
}
