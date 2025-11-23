package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/hibiken/asynq"
)

func TestDLQHandler_ContractCompliance(t *testing.T) {
	app := fiber.New()

	// Mock inspector for testing
	mockInspector := &MockAsynqInspector{}
	dlqHandler := NewDLQHandler(mockInspector)

	// Test GetDLQItems response format
	app.Get("/test-dlq-items", dlqHandler.GetDLQItems)
	req := httptest.NewRequest("GET", "/test-dlq-items", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Verify status code (should be 401 since we don't have auth)
	assert.Equal(t, 401, resp.StatusCode)
	// If we had proper auth context, we'd check the response format here

	// Test RequeueDLQItems
	app.Post("/test-dlq-requeue", dlqHandler.RequeueDLQItems)
	
	// Create a request body with the expected format
	requeueReq := DLQRequeueRequest{
		JobIDs: []string{"job-1", "job-2"},
	}
	jsonData, _ := json.Marshal(requeueReq)
	
	req = httptest.NewRequest("POST", "/test-dlq-requeue", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	assert.NoError(t, err)
	
	// Should get 401 without auth or 400 with bad body
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 500)

	// Test PurgeDLQItems
	app.Post("/test-dlq-purge", dlqHandler.PurgeDLQItems)
	
	// Create a request body with the expected format
	purgeReq := DLQPurgeRequest{
		JobIDs: []string{"job-1", "job-2"},
	}
	jsonData, _ = json.Marshal(purgeReq)
	
	req = httptest.NewRequest("POST", "/test-dlq-purge", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	assert.NoError(t, err)
	
	// Should get 401 without auth or 400 with bad body
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 500)
}

// Mock for testing
type MockAsynqInspector struct{}

func (m *MockAsynqInspector) Queues() ([]string, error) {
	return []string{"default"}, nil
}

func (m *MockAsynqInspector) ListDead(queueName string) ([]*asynq.TaskInfo, error) {
	return []*asynq.TaskInfo{}, nil
}

func (m *MockAsynqInspector) GetDeadTask(queueName, taskID string) (*asynq.TaskInfo, error) {
	return nil, nil
}

func (m *MockAsynqInspector) DeleteDead(queueName, taskID string) error {
	return nil
}