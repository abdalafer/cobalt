package cobalt_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardanlabs/cobalt"
)

var r = map[int][]string{
	1: []string{"/", "Get"},
	2: []string{"/foo", "Get"},
	3: []string{"/", "Post"},
	4: []string{"/foo", "Post"},
	5: []string{"/", "Put"},
	6: []string{"/foo", "Put"}}

func NewRequest(method, path string, body io.Reader) *http.Request {
	r, _ := http.NewRequest(method, path, body)
	u, _ := url.Parse(path)
	r.URL = u
	r.RequestURI = path
	return r
}

// TestReqeust tests
func TestRequest(t *testing.T) {
	r := NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	const key = "KEY"
	const value = "DATA"
	const code = 200

	mw := func(h cobalt.Handler) cobalt.Handler {
		return func(ctx *cobalt.Context) {
			ctx.SetData(key, value)
			fmt.Println("Middleware Fired")
			h(ctx)
		}
	}

	h := func(ctx *cobalt.Context) {
		fmt.Println("Route Fired")
		v := ctx.GetData(key)
		if v != value {
			t.Errorf("expected %s got %s", value, v)
		}
		ctx.Response.Write([]byte(value))
	}
	c := cobalt.New(&JSONEncoder{})
	c.Get("/", h, mw)

	c.ServeHTTP(w, r)

	if w.Code != code {
		t.Errorf("expected status code to be 200 instead got %d", w.Code)
	}
	if w.Body.String() != value {
		t.Errorf("expected body to be %s instead got %s", value, w.Body.String())
	}
}

// TestMidwareExit tests a middleware exiting.
func TestMidwareExit(t *testing.T) {
	r := NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	const key = "KEY"
	const value = "DATA"
	const code = 400
	c := cobalt.New(&JSONEncoder{})

	mw := func(h cobalt.Handler) cobalt.Handler {
		return func(ctx *cobalt.Context) {
			fmt.Println("Middleware")
			ctx.Response.WriteHeader(http.StatusBadRequest)
			ctx.Response.Write([]byte(value))
		}
	}

	h := func(ctx *cobalt.Context) {
		fmt.Println("Route Fired")
		v := ctx.GetData(key)
		if v != value {
			t.Errorf("expected %s got %s", value, v)
		}
		ctx.Response.Write([]byte(value))
	}
	c.Get("/", h, mw)

	c.ServeHTTP(w, r)

	if w.Code != code {
		t.Errorf("expected status code to be %d instead got %d", code, w.Code)
	}
	if w.Body.String() != value {
		t.Errorf("expected body to be %s instead got %s", value, w.Body.String())
	}
}

// TestRoutes tests the routing of requests.
func TestRoutes(t *testing.T) {
	c := cobalt.New(&JSONEncoder{})

	// GET
	c.Get("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Get/"))
	})
	c.Get("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Get/foo"))
	})

	// POST
	c.Post("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Post/"))
	})
	c.Post("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Post/foo"))
	})

	// PUT
	c.Put("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Put/"))
	})
	c.Put("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Put/foo"))
	})

	// Delete
	c.Delete("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Delete/"))
	})
	c.Delete("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Delete/foo"))
	})

	// OPTIONS
	c.Options("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Options/"))
	})
	c.Options("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Options/foo"))
	})

	// HEAD
	c.Head("/", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Head/"))
	})
	c.Head("/foo", func(ctx *cobalt.Context) {
		ctx.Response.Write([]byte("Head/foo"))
	})

	for _, v := range r {
		AssertRoute(v[0], v[1], c, t)
	}
}

// AsserRoute is a helper method to tests routes
func AssertRoute(path, verb string, c *cobalt.Cobalt, t *testing.T) {
	r := NewRequest(strings.ToUpper(verb), path, nil)
	w := httptest.NewRecorder()

	c.ServeHTTP(w, r)
	if w.Body.String() != verb+path {
		t.Errorf("expected body to be %s instead got %s", verb+path, w.Body.String())
	}
}

// TestNotFoundHandler tests handler for 404.
func TestNotFoundHandler(t *testing.T) {
	//setup request
	r := NewRequest("GET", "/FOO", nil)
	w := httptest.NewRecorder()

	m := struct{ Message string }{"Not Found"}

	nf := func(c *cobalt.Context) {
		c.ServeWithStatus(m, http.StatusNotFound)
	}

	c := cobalt.New(&JSONEncoder{})
	c.NotFound(nf)

	c.Get("/",
		func(ctx *cobalt.Context) {
			panic("Panic Test")
		},
		nil)

	c.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status code to be 404 instead got %d", w.Code)
	}

	var msg struct{ Message string }
	json.Unmarshal([]byte(w.Body.String()), &msg)

	if msg.Message != m.Message {
		t.Errorf("expected body to be %s instead got %s", msg.Message, m.Message)
	}
}

// TestServerErrorHandler tests handler for 500.
func TestServerErrorHandler(t *testing.T) {
	//setup request
	r := NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	m := struct{ Message string }{"Internal Error"}

	se := func(c *cobalt.Context) {
		c.ServeWithStatus(m, http.StatusInternalServerError)
	}

	c := cobalt.New(&JSONEncoder{})
	c.ServerErr(se)

	c.Get("/",
		func(ctx *cobalt.Context) {
			panic("Panic Test")
		},
		nil)

	c.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status code to be 500 instead got %d", w.Code)
	}

	var msg struct{ Message string }
	json.Unmarshal([]byte(w.Body.String()), &msg)

	if msg.Message != m.Message {
		t.Errorf("expected body to be %s instead got %s", msg.Message, m.Message)
	}
}

type JSONEncoder struct{}

func (enc JSONEncoder) Encode(w io.Writer, val interface{}) error {
	return json.NewEncoder(w).Encode(val)
}

func (enc JSONEncoder) Decode(r io.Reader, val interface{}) error {
	return json.NewDecoder(r).Decode(val)
}

func (enc JSONEncoder) ContentType() string {
	return "application/json;charset=UTF-8"
}
