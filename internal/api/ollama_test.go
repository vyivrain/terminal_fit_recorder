package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPromptAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{name: "sends bearer token when key is set", apiKey: "secret", want: "Bearer secret"},
		{name: "omits header when key is empty", apiKey: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Authorization")
				w.Write([]byte(`{"response":"ok","done":true}`))
			}))
			defer server.Close()

			client := NewClient(server.URL, "model", tt.apiKey, "")
			if _, err := client.SendPrompt(context.Background(), "hi"); err != nil {
				t.Fatalf("SendPrompt: %v", err)
			}
			if got != tt.want {
				t.Errorf("Authorization = %q, want %q", got, tt.want)
			}
		})
	}
}
