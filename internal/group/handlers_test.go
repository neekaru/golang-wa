package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListGroupsRejectsInvalidUser(t *testing.T) {
	router := gin.New()
	router.POST("/groups/list", NewHandlers(nil).ListGroupsHandler)

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"user":`},
		{name: "missing user", body: `{}`},
		{name: "empty user", body: `{"user":""}`},
		{name: "blank user", body: `{"user":" \t "}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/groups/list", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if body["error"] == "" {
				t.Fatalf("missing error message: %s", response.Body.String())
			}
		})
	}
}
