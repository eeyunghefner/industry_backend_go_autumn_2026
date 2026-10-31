package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: make(map[string]Task)}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	id := strconv.Itoa(int(r.seq))

	newTask := Task{ID: id, Title: title, Done: false, UpdatedAt: r.clock.Now()}

	r.tasks[id] = newTask
	return newTask, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.tasks[id]
	return value, ok
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Task, 0)
	for _, val := range r.tasks {
		if val.Done == done {
			result = append(result, val)
		}
	}
	return result
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.tasks[id]
	if !ok {
		return value, ErrNotFound
	}
	if value.Done == done {
		return value, nil
	}
	value.Done = done
	value.UpdatedAt = r.clock.Now()
	r.tasks[id] = value
	return value, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/tasks/") {
		id := strings.TrimPrefix(path, "/tasks/")
		if id == "" || strings.Contains(id, "/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Title == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	task, err := h.repo.Create(*req.Title)
	if errors.Is(err, ErrInvalidTitle) {
		w.WriteHeader(http.StatusBadRequest)
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		writeJSON(w, http.StatusCreated, task)
	}
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()["done"]
	var done bool

	if len(vals) == 0 {
		done = false
	} else if len(vals) == 1 && (vals[0] == "true" || vals[0] == "false") {
		if vals[0] == "true" {
			done = true
		} else {
			done = false
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tasks := h.repo.List(done)
	if tasks == nil {
		tasks = []Task{}
	}
	slices.SortFunc(tasks, func(a, b Task) int {
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			if a.UpdatedAt.After(b.UpdatedAt) {
				return -1
			}
			return 1
		}
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Done == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	task, err := h.repo.SetDone(id, *req.Done)
	if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		writeJSON(w, http.StatusOK, task)
	}
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	err := decoder.Decode(&struct{}{})
	switch err {
	case io.EOF:
		return nil
	case nil:
		return errors.New("unexpected data after JSON object")
	default:
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
