package jwtmiddleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

var (
	// ErrEmptyToken internal error when token it's empty
	ErrEmptyToken = errors.New("mdw: empty token")
	// ErrTokenMalformed internal error when the given token have an invalid format
	ErrTokenMalformed = errors.New("mdw: invalid token format")
)

type (
	// MiddlewareOption used to define `functional options` approach
	MiddlewareOption func(*options)
	// TokenExtractor function to extract token and used to define custom extractors
	TokenExtractor func(*http.Request) (string, error)
	errorHandler   func(http.ResponseWriter, *http.Request, error)
)

type options struct {
	errorHandler  errorHandler
	extractor     TokenExtractor
	signingMethod jwt.SigningMethod
	signKey       interface{}
	userProperty  interface{}
}

// Middleware entrypoint to validates incoming token
type Middleware struct {
	options *options
}

func onError(w http.ResponseWriter, r *http.Request, _ error) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("unauthorized"))
}

// BearerExtractor gets the jwt token from the `Authorization` header
func BearerExtractor(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", ErrEmptyToken
	}
	parts := strings.Split(token, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
		return "", ErrTokenMalformed
	}
	return parts[1], nil
}

// QueryStringExtractor gets the jwt token from the query string
// defined on the given query param
func QueryStringExtractor(param string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		if query := r.URL.Query().Get(param); query != "" {
			return query, nil
		}
		return "", errors.New("could not get query string value")
	}
}

// WithErrorHandler set a custom error handler when the given
// token is invalid.
func WithErrorHandler(handler errorHandler) MiddlewareOption {
	return func(opts *options) {
		opts.errorHandler = handler
	}
}

// WithExtractor set a custom way to extract the JWT token
func WithExtractor(extractor TokenExtractor) MiddlewareOption {
	return func(opts *options) {
		opts.extractor = extractor
	}
}

// WithSigningMethod set the expected jwt token signing method
func WithSigningMethod(method jwt.SigningMethod) MiddlewareOption {
	return func(opts *options) {
		opts.signingMethod = method
	}
}

// WithSignKey set a custom key to validate the incoming token
func WithSignKey(key interface{}) MiddlewareOption {
	return func(opts *options) {
		opts.signKey = key
	}
}

// WithUserProperty set a custom user property on the request context
func WithUserProperty(property interface{}) MiddlewareOption {
	return func(opts *options) {
		opts.userProperty = property
	}
}

// New creates an instance of the jwt middleware
func New(opts ...MiddlewareOption) *Middleware {
	defaults := &options{
		userProperty:  "user",
		signingMethod: jwt.SigningMethodHS256,
		errorHandler:  onError,
		extractor:     BearerExtractor,
	}
	for _, w := range opts {
		w(defaults)
	}
	return &Middleware{options: defaults}
}

func (m *Middleware) parseToken(r *http.Request) (*jwt.Token, error) {
	token, err := m.options.extractor(r)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, ErrEmptyToken
	}
	t, err := jwt.Parse(token, func(_ *jwt.Token) (interface{}, error) {
		return m.options.signKey, nil
	})
	if err != nil {
		return nil, err
	}
	if m.options.signingMethod != nil && t.Header["alg"] != m.options.signingMethod.Alg() {
		return nil, jwt.ErrSignatureInvalid
	}
	return t, nil
}

func setTokenToContext(r *http.Request, key interface{}, token *jwt.Token) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, token))
}

// Handler standard handler to check the incoming jwt token
func (m *Middleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.parseToken(r)
		if err != nil {
			m.options.errorHandler(w, r, err)
			return
		}
		h.ServeHTTP(w, setTokenToContext(r, m.options.userProperty, token))
	})
}

// HandlerNext implementation for `negroni` middleware
func (m *Middleware) HandlerNext(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	token, err := m.parseToken(r)
	if err != nil {
		m.options.errorHandler(w, r, err)
		return
	}
	next(w, setTokenToContext(r, m.options.userProperty, token))
}
