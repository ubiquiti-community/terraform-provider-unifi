package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

// v2FakeController serves v2 collections (qos-rules, content-filtering, ...)
// for site "default" from memory: GET lists, POST creates (on the collection
// or on <collection>/create), PUT replaces and DELETE removes by id, answering
// 404 for an unknown id. New qos-rules get an index, as the controller assigns
// one.
type v2FakeController struct {
	mu     sync.Mutex
	items  map[string][]map[string]any
	nextID int
	// fail makes every request with this method answer HTTP 400.
	fail map[string]bool
	// bodies records the last request body per method.
	bodies map[string]map[string]any
}

func newV2FakeController(t *testing.T) (*Client, *v2FakeController) {
	t.Helper()
	f := &v2FakeController{
		items:  map[string][]map[string]any{},
		fail:   map[string]bool{},
		bodies: map[string]map[string]any{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "unifises", Value: "fake-session", Path: "/"})
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	})
	mux.HandleFunc("/v2/api/site/default/", f.serve(t))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL: srv.URL, Username: "admin", Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	return &Client{ApiClient: apiClient, Site: "default"}, f
}

func (f *v2FakeController) serve(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.fail[r.Method] {
			http.Error(w, `{"code":"api.err.Invalid"}`, http.StatusBadRequest)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v2/api/site/default/"), "/")
		coll := parts[0]
		var body map[string]any
		if b, _ := io.ReadAll(r.Body); len(b) > 0 && string(b) != "{}" {
			if err := json.Unmarshal(b, &body); err != nil {
				t.Errorf("%s body is not a JSON object: %v", r.Method, err)
			}
			f.bodies[r.Method] = body
		}

		switch {
		case r.Method == http.MethodGet && len(parts) == 1:
			list := f.items[coll]
			if list == nil {
				list = []map[string]any{}
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost && (len(parts) == 1 || parts[1] == "create"):
			f.nextID++
			body["_id"] = fmt.Sprintf("fake%020d", f.nextID)
			if coll == "qos-rules" {
				body["index"] = 10000 + f.nextID
			}
			f.items[coll] = append(f.items[coll], body)
			_ = json.NewEncoder(w).Encode(body)
		case (r.Method == http.MethodPut || r.Method == http.MethodDelete) && len(parts) == 2:
			for i, it := range f.items[coll] {
				if it["_id"] != parts[1] {
					continue
				}
				if r.Method == http.MethodDelete {
					f.items[coll] = append(f.items[coll][:i], f.items[coll][i+1:]...)
					w.WriteHeader(http.StatusOK)
					return
				}
				body["_id"] = parts[1]
				if idx, ok := it["index"]; ok {
					body["index"] = idx
				}
				f.items[coll][i] = body
				_ = json.NewEncoder(w).Encode(body)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

// remove deletes an item behind the provider's back.
func (f *v2FakeController) remove(coll, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, it := range f.items[coll] {
		if it["_id"] == id {
			f.items[coll] = append(f.items[coll][:i], f.items[coll][i+1:]...)
			return
		}
	}
}

func (f *v2FakeController) count(coll string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items[coll])
}
