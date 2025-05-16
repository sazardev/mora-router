package router

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMoraRouterBasic(t *testing.T) {
	// usando alias New con middlewares de logging y recuperación
	r := New(WithLogging(), WithRecovery(), WithCORS("*"))

	// test ruta con parámetro y JSON helper
	r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request, p Params) {
		if p["id"] != "42" {
			t.Errorf("expected id 42, got %s", p["id"])
		}
		JSON(w, http.StatusOK, map[string]string{"id": p["id"]})
	})

	req := httptest.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON Content-Type, got %s", ct)
	}
	body, _ := ioutil.ReadAll(res.Body)
	if !strings.Contains(string(body), `"id":"42"`) {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	r := NewMoraRouter(WithRecovery())
	r.Get("/panic", func(w http.ResponseWriter, req *http.Request, p Params) {
		panic("oops")
	})
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on panic, got %d", res.StatusCode)
	}
}

func TestCORSMiddleware(t *testing.T) {
	r := NewMoraRouter(WithCORS("https://example.com"))
	r.Get("/cors", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusOK)
	})
	// test preflight
	req := httptest.NewRequest("OPTIONS", "/cors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("CORS origin header missing or incorrect")
	}
}

func TestRouteGroupAndNotFound(t *testing.T) {
	r := NewMoraRouter()
	api := r.Group("/api")
	api.Get("/ping", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusTeapot)
	})
	// test group route
	req := httptest.NewRequest("GET", "/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusTeapot {
		t.Errorf("expected 418, got %d", w.Result().StatusCode)
	}
	// custom not found
	r.NotFound(func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(499)
	})
	req2 := httptest.NewRequest("GET", "/nope", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Result().StatusCode != 499 {
		t.Errorf("expected custom 499, got %d", w2.Result().StatusCode)
	}
}

func TestNamedRoutesAndURLReversal(t *testing.T) {
	r := NewMoraRouter()
	// register and name the route
	r.Get("/articles/:category/:id", func(w http.ResponseWriter, req *http.Request, p Params) {})
	r.Name("article_detail", "/articles/:category/:id")

	// valid URL generation
	url, err := r.URL("article_detail", "tech", "100")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if url != "/articles/tech/100" {
		t.Errorf("expected /articles/tech/100, got %s", url)
	}

	// missing parameter
	_, err = r.URL("article_detail", "only")
	if err == nil {
		t.Errorf("expected error for missing params")
	}

	// too many parameters
	_, err = r.URL("article_detail", "a", "b", "c")
	if err == nil {
		t.Errorf("expected error for too many params")
	}

	// unknown route name
	_, err = r.URL("nonexistent")
	if err == nil {
		t.Errorf("expected error for unknown route name")
	}
}
