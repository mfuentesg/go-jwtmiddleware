# JWT Middleware

[![GoDoc](https://godoc.org/github.com/mfuentesg/go-jwtmiddleware?status.svg)](https://godoc.org/github.com/mfuentesg/go-jwtmiddleware)
[![Build Status](https://travis-ci.org/mfuentesg/go-jwtmiddleware.svg?branch=master)](https://travis-ci.org/mfuentesg/go-jwtmiddleware)
[![codecov](https://codecov.io/gh/mfuentesg/go-jwtmiddleware/branch/master/graph/badge.svg)](https://codecov.io/gh/mfuentesg/go-jwtmiddleware)

<a href="https://www.buymeacoffee.com/mfuentesg" target="_blank">
   <img height="41" src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" />
</a>

<br />
<br />

This package has as purpose will help you to check the validness of any token based on [JWT spec](https://tools.ietf.org/html/rfc7519).

## Getting started

### Installation

```bash
$ go get github.com/mfuentesg/go-jwtmiddleware
```

### Using it

You can use it with the `net/http` package or even with a middleware-focused library like [Negroni](https://github.com/urfave/negroni).


#### net/http

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mfuentesg/go-jwtmiddleware"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello world!"))
}

func main() {
	m := jwtmiddleware.New(jwtmiddleware.WithSignKey([]byte("secret")))
	http.Handle("/", m.Handler(http.HandlerFunc(helloWorld)))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```


#### negroni

```go
package main

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/mfuentesg/go-jwtmiddleware"
)

func main() {
	m := jwtmiddleware.New(jwtmiddleware.WithSignKey("user"))
	n := negroni.Classic()
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world!"))
	})
	n.UseFunc(m.HandlerNext)
	n.UseHandler(mux)

	http.ListenAndServe(":3000", n)
}
```

### Options

This project is based on `functional options` concept, to set the initial settings of the middleware. These options must be passed on the `jwtmiddleware.New()` function which options are of the type `MiddlewareOption func(*options)`.

#### errorHandler

This property is a `callback` to control the ocurred errors on the validation process.
Unlike other middlewares, this package returns [jwt-go](https://github.com/dgrijalva/jwt-go) native errors, given you the posibility to check errors with the defined contants, for example `jwt.ErrSignatureInvalid`.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	mdw "github.com/mfuentesg/go-jwtmiddleware"
)

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(fmt.Sprintf("unauthorized : %v", err)))
}

func main() {
	m := mdw.New(mdw.WithSignKey([]byte("secret")), mdw.WithErrorHandler(errorHandler))
	http.Handle("/", m.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world!"))
	})))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

```

#### extractor

This property allows you to get token value from any place, by default the token will be extracted from the `Authorization` header on the incominng request.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	mdw "github.com/mfuentesg/go-jwtmiddleware"
)

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(fmt.Sprintf("unauthorized : %v", err)))
}

func main() {
	m := mdw.New()
	http.Handle("/", m.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("user").(*jwt.Token).Claims.(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("hello world %s!", token["name"])))
	})))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

```bash
$ curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o" localhost:3000

hello world John Doe!
```

#### signingMethod

This property indicates the used algorithm to encrypt the token, i.e. `jwt.SigningMethodES256`.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	mdw "github.com/mfuentesg/go-jwtmiddleware"
)

func main() {
	m := mdw.New(mdw.WithSigningMethod())
	http.Handle("/", m.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("user").(*jwt.Token).Claims.(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("hello world %s!", token["name"])))
	})))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

#### signKey
It used to set the secret key to encrypt/decrypt the token.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	mdw "github.com/mfuentesg/go-jwtmiddleware"
)

func main() {
	m := mdw.New(mdw.WithSignKey([]byte("my secret key")))
	http.Handle("/", m.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("user").(*jwt.Token).Claims.(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("hello world %s!", token["name"])))
	})))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

```bash
# token created with the secret `secret`
$ curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o" localhost:3000

unauthorized
```

#### userProperty
It defines the name of the property in the request where the token will be stored.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	mdw "github.com/mfuentesg/go-jwtmiddleware"
)

type userKey string

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(fmt.Sprintf("unauthorized : %v", err)))
}

func main() {
	m := mdw.New(mdw.WithSignKey([]byte("secret")), mdw.WithUserProperty(userKey("user")))
	http.Handle("/", m.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value(userKey("user")).(*jwt.Token).Claims.(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("hello world %s!", token["name"])))
	})))
	fmt.Println("listening on port :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

```
$ curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o" localhost:3000

hello world John Doe!
```
