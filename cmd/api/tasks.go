package main

import (
	"net/http"
	"sync"
)

type TaskMessage struct {
	Type    string        `json:"type"`    // "progress", "finished", or "error"
	Content string        `json:"content"` // The message or result
	Request *http.Request `json:"-"`       // Not serialized for output, used to extract IP
}

type TaskStore struct {
	sync.RWMutex
	channels map[string]chan TaskMessage
}

// Global task store
var taskStore = &TaskStore{
	channels: make(map[string]chan TaskMessage),
}

func (ts *TaskStore) Set(taskID string, ch chan TaskMessage) {
	ts.Lock()
	ts.channels[taskID] = ch
	ts.Unlock()
}

func (ts *TaskStore) Get(taskID string) (chan TaskMessage, bool) {
	ts.RLock()
	ch, exists := ts.channels[taskID]
	ts.RUnlock()
	return ch, exists
}

func (ts *TaskStore) Delete(taskID string) {
	ts.Lock()
	delete(ts.channels, taskID)
	ts.Unlock()
}
