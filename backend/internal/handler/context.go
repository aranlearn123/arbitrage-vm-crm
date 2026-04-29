package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Context struct {
	w       http.ResponseWriter
	r       *http.Request
	status  int
	written bool
}

type HandlerFunc func(*Context) error

func Wrap(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &Context{
			w:      w,
			r:      r,
			status: http.StatusOK,
		}
		if err := fn(c); err != nil && !c.written {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (c *Context) UserContext() context.Context {
	return c.r.Context()
}

func (c *Context) Query(key string, fallback ...string) string {
	value := c.r.URL.Query().Get(key)
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

func (c *Context) Params(key string) string {
	return chi.URLParam(c.r, key)
}

func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

func (c *Context) JSON(value any) error {
	c.w.Header().Set("Content-Type", "application/json")
	if !c.written {
		c.w.WriteHeader(c.status)
		c.written = true
	}
	return json.NewEncoder(c.w).Encode(value)
}
