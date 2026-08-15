package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// maxPersistedSessionIDLength bounds the persisted client session identifier to the
// usage_logs.session_id column width (VARCHAR(255)). Longer values are rejected so
// distinct identifiers can never alias through truncation.
const maxPersistedSessionIDLength = 255

// clientSessionIDHeaders extends the OpenAI-compatible sticky-session signals with
// native protocol identifiers that are safe to persist but must not alter OpenAI
// scheduling behavior.
var clientSessionIDHeaders = append(
	append([]string(nil), explicitOpenAIHeaderSessionNames...),
	claudeCodeSessionHeader,
)

type usageSessionIDContextKey struct{}

// attachUsageSessionIDToGin preserves a sanitized explicit client session ID on
// the request. OpenAI-compatible clients commonly send prompt_cache_key in the
// JSON body, which is available while scheduling but not when usage is persisted.
func attachUsageSessionIDToGin(c *gin.Context, raw string) {
	if c == nil || c.Request == nil {
		return
	}
	sessionID := sanitizeSessionID(raw)
	if sessionID == "" {
		return
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), usageSessionIDContextKey{}, sessionID))
}

func usageSessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	sessionID, _ := ctx.Value(usageSessionIDContextKey{}).(string)
	return sessionID
}

func clientSessionIDFromHeaders(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	for _, header := range clientSessionIDHeaders {
		if sessionID := sanitizeSessionID(c.GetHeader(header)); sessionID != "" {
			return sessionID
		}
	}
	if isGrokRequestContext(c) {
		if sessionID := sanitizeSessionID(c.GetHeader(grokConversationIDHeader)); sessionID != "" {
			return sessionID
		}
	}
	return ""
}

// ExtractClientSessionID resolves the explicit client-provided session identifier for
// usage-log correlation and returns it sanitized. Header values are read directly;
// body values discovered during scheduling are carried through request context. It is
// shared by every gateway handler so all supported protocols record session_id through
// one seam. Returns "" when no valid identifier is present.
//
// This value feeds only usage_logs.session_id persistence. It does NOT affect sticky
// routing, account selection, request_id semantics, or upstream prompt caching, which
// keep their own (intentionally broader) session-signal resolution.
func ExtractClientSessionID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if sessionID := usageSessionIDFromContext(c.Request.Context()); sessionID != "" {
		return sessionID
	}
	return clientSessionIDFromHeaders(c)
}

// sanitizeSessionID normalizes a raw client-supplied session identifier for safe
// persistence: it trims surrounding whitespace, rejects the value outright if it
// contains any control character (CR/LF/tab/NUL/…) so a log- or header-injection style
// payload cannot slip into stored correlation data, and rejects values longer than
// the DB column bound. Absent or invalid input yields "".
func sanitizeSessionID(raw string) string {
	if !utf8.ValidString(raw) {
		return ""
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	count := 0
	for _, r := range trimmed {
		if r < 0x20 || r == 0x7f {
			// An explicit correlation id never legitimately contains control
			// characters; drop the whole value rather than persist a mangled or
			// partially-injected identifier.
			return ""
		}
		count++
		if count > maxPersistedSessionIDLength {
			return ""
		}
	}
	return trimmed
}
