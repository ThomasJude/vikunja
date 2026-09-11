// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/log"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPLoggerRedactsInviteTokens(t *testing.T) {
	config.InitDefaultConfig()
	for _, key := range []config.Key{config.LogEnabled, config.LogHTTP, config.LogHTTPLevel, config.LogFormat} {
		old := key.Get()
		t.Cleanup(func() { key.Set(old) })
	}
	logDir := t.TempDir()
	log.ConfigureStandardLogger(false, "off", logDir, "INFO", "structured")
	config.LogEnabled.Set(true)
	config.LogHTTP.Set("file")
	config.LogHTTPLevel.Set("INFO")
	config.LogFormat.Set("structured")
	e := NewEcho()
	e.Any("/*", func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) })

	tests := []struct{ uri, want string }{
		{"/api/v2/invite-links/synthetic-secret", "/api/v2/invite-links/[redacted]"},
		{"/api/v2/invite-links/synthetic-secret/register?source=email", "/api/v2/invite-links/[redacted]/register?source=email"},
		{"/invite/synthetic-secret", "/invite/[redacted]"},
		{"/vikunja/invite/synthetic-secret", "/vikunja/invite/[redacted]"},
		{"/api/v2/invite-links/synthetic%2Dsecret/register", "/api/v2/invite-links/[redacted]/register"},
		{"/api/v2/invite-links/synthetic%2Fsecret/register", "/api/v2/invite-links/[redacted]/register"},
		{"/api/v2/invite%2Dlinks/synthetic-secret", "/api/v2/invite%2Dlinks/[redacted]"},
		{"/vikunja/%69nvite/synthetic-secret", "/vikunja/%69nvite/[redacted]"},
		{"/api/v2/admin/invite-links/42", "/api/v2/admin/invite-links/42"},
		{"/api/v2/tasks?filter=title%20%3D%20test", "/api/v2/tasks?filter=title%20%3D%20test"},
	}
	for _, tt := range tests {
		request := httptest.NewRequest(http.MethodGet, tt.uri, nil)
		request.Header.Set("User-Agent", "invite-logging-test")
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code)
		assert.Equal(t, tt.uri, request.RequestURI)
	}
	output, err := os.ReadFile(filepath.Join(logDir, "http.log"))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	require.Len(t, lines, len(tests))
	for i, line := range lines {
		var record struct {
			URI       string `json:"uri"`
			Method    string `json:"method"`
			Status    int    `json:"status"`
			UserAgent string `json:"user_agent"`
			RemoteIP  string `json:"remote_ip"`
			Latency   *int64 `json:"latency"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		assert.Equal(t, tests[i].want, record.URI)
		assert.Equal(t, "GET", record.Method)
		assert.Equal(t, http.StatusNoContent, record.Status)
		assert.Equal(t, "invite-logging-test", record.UserAgent)
		assert.NotEmpty(t, record.RemoteIP)
		assert.NotNil(t, record.Latency)
	}
}

func TestSentryMiddlewareRedactsInviteTokens(t *testing.T) {
	for _, uri := range []string{
		"/api/v2/invite-links/synthetic-secret/register",
		"/api/v2/invite-links/synthetic%2Fsecret/register",
		"/vikunja/%69nvite/synthetic-secret",
	} {
		scope := sentry.NewScope()
		hub := sentry.NewHub(nil, scope)
		e := echo.New()
		e.Use(SentryMiddleware(SentryOptions{}))
		e.Any("/*", func(c *echo.Context) error {
			assert.Equal(t, uri, c.Request().RequestURI)
			assert.Equal(t, "http://example.com/invite/synthetic-secret", c.Request().Header.Get("Referer"))
			assert.Equal(t, "http://example.com"+uri, c.Request().URL.String())
			return c.NoContent(http.StatusNoContent)
		})
		request := httptest.NewRequest(http.MethodGet, "http://example.com"+uri, nil)
		request.RequestURI = uri
		request.Header.Set("Referer", "http://example.com/invite/synthetic-secret")
		request = request.WithContext(sentry.SetHubOnContext(request.Context(), hub))
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code)
		event := scope.ApplyToEvent(sentry.NewEvent(), nil, nil)
		require.NotNil(t, event.Request)
		assert.NotContains(t, event.Request.URL, "synthetic")
		assert.Contains(t, event.Request.URL, "redacted")
		assert.Equal(t, "http://example.com/invite/[redacted]", event.Request.Headers["Referer"])
	}
}
