package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32 // 256 bits of entropy
)

// CSRFProtect validates the Double Submit Cookie pattern for state-changing
// requests (POST, PUT, PATCH, DELETE). It expects the client to:
//  1. Read the "csrf_token" cookie (non-HttpOnly, set by the server).
//  2. Echo it back in the "X-CSRF-Token" request header.
//
// The middleware aborts with 403 Forbidden if the header is absent, empty,
// or does not match the cookie value.
//
// GET, HEAD, and OPTIONS requests are allowed through without validation
// because they must be safe and idempotent by convention (RFC 7231).
//
// The secure parameter controls whether the CSRF cookie itself has the
// Secure flag (should be true in production with HTTPS).
func CSRFProtect(secure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}

		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
			return
		}

		c.Next()
	}
}

// NewCSRFToken generates a cryptographically random CSRF token.
func NewCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SetCSRFCookie writes a new CSRF token cookie on c and returns the token.
// The cookie is intentionally NOT HttpOnly so JavaScript can read and echo it
// in the X-CSRF-Token header (Double Submit Cookie pattern).
func SetCSRFCookie(c *gin.Context, secure bool) (string, error) {
	token, err := NewCSRFToken()
	if err != nil {
		return "", err
	}

	// maxAge = 0 makes the cookie session-scoped (cleared on tab/browser close).
	// We refresh it on every token refresh so it stays in sync with the session.
	c.SetCookie(csrfCookieName, token, 0, "/", "", secure, false /* not HttpOnly */)
	return token, nil
}
