package project

import "testing"

func TestProjectSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantName string
	}{
		{
			name:     "name",
			input:    "engagement-a",
			wantName: "engagement-a",
		},
		{
			name:   "id",
			input:  "8b22b915-0f34-4d85-8da9-7182731bfb9b",
			wantID: "8b22b915-0f34-4d85-8da9-7182731bfb9b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := projectSelector(tt.input)
			if selector.Id != tt.wantID {
				t.Fatalf("Id = %q, want %q", selector.Id, tt.wantID)
			}
			if selector.Name != tt.wantName {
				t.Fatalf("Name = %q, want %q", selector.Name, tt.wantName)
			}
		})
	}
}
