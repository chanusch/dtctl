package workflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dynatrace-oss/dtctl/pkg/client"
)

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		filters    WorkflowFilters
		wantOwner  string
		statusCode int
		response   WorkflowList
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "list all workflows",
			filters:    WorkflowFilters{},
			wantOwner:  "",
			statusCode: http.StatusOK,
			response: WorkflowList{
				Count: 2,
				Results: []Workflow{
					{ID: "wf1", Title: "Workflow 1", Owner: "user-123"},
					{ID: "wf2", Title: "Workflow 2", Owner: "user-456"},
				},
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:       "list with owner filter (--mine)",
			filters:    WorkflowFilters{Owner: "user-123"},
			wantOwner:  "user-123",
			statusCode: http.StatusOK,
			response: WorkflowList{
				Count: 1,
				Results: []Workflow{
					{ID: "wf1", Title: "Workflow 1", Owner: "user-123"},
				},
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:       "server error",
			filters:    WorkflowFilters{},
			statusCode: http.StatusInternalServerError,
			response:   WorkflowList{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request path
				if r.URL.Path != "/platform/automation/v1/workflows" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				// Verify the owner query parameter if expected
				gotOwner := r.URL.Query().Get("owner")
				if gotOwner != tt.wantOwner {
					t.Errorf("owner query param = %q, want %q", gotOwner, tt.wantOwner)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			c, err := client.New(server.URL, "test-token")
			if err != nil {
				t.Fatalf("client.New() error = %v", err)
			}
			c.HTTP().SetRetryCount(0)

			handler := NewHandler(c)
			list, err := handler.List(tt.filters)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if list == nil {
					t.Fatal("List() returned nil")
				}
				if len(list.Results) != tt.wantCount {
					t.Errorf("List() returned %d workflows, want %d", len(list.Results), tt.wantCount)
				}
			}
		})
	}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		statusCode int
		response   Workflow
		wantErr    bool
	}{
		{
			name:       "get existing workflow",
			id:         "wf-123",
			statusCode: http.StatusOK,
			response: Workflow{
				ID:          "wf-123",
				Title:       "Test Workflow",
				Owner:       "user-123",
				Description: "A test workflow",
			},
			wantErr: false,
		},
		{
			name:       "workflow not found",
			id:         "wf-nonexistent",
			statusCode: http.StatusNotFound,
			response:   Workflow{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/platform/automation/v1/workflows/" + tt.id
				if r.URL.Path != expectedPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			c, err := client.New(server.URL, "test-token")
			if err != nil {
				t.Fatalf("client.New() error = %v", err)
			}
			c.HTTP().SetRetryCount(0)

			handler := NewHandler(c)
			wf, err := handler.Get(tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if wf == nil {
					t.Fatal("Get() returned nil")
				}
				if wf.ID != tt.response.ID {
					t.Errorf("Get() ID = %v, want %v", wf.ID, tt.response.ID)
				}
				if wf.Title != tt.response.Title {
					t.Errorf("Get() Title = %v, want %v", wf.Title, tt.response.Title)
				}
			}
		})
	}
}

func TestWorkflowFilters(t *testing.T) {
	// Test that WorkflowFilters struct has the expected fields
	filters := WorkflowFilters{
		Owner: "user-123",
	}

	if filters.Owner != "user-123" {
		t.Errorf("WorkflowFilters.Owner = %v, want user-123", filters.Owner)
	}

	// Test empty filters
	emptyFilters := WorkflowFilters{}
	if emptyFilters.Owner != "" {
		t.Errorf("Empty WorkflowFilters.Owner should be empty, got %v", emptyFilters.Owner)
	}
}
