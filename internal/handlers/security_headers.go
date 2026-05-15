package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware sets a defensive set of HTTP response headers when
// SECURITY_ENABLED=true. In vulnerable mode no headers are added — the absence
// is itself the demo (an attacker can iframe the page, the browser will sniff
// content types, error pages may leak stack traces, etc.).
//
// Headers set in secure mode:
//
//   - Content-Security-Policy: limits script/style sources to self + the two
//     CDNs the lab actually uses (htmx and Google fonts). Inline event handlers
//     and eval are disallowed.
//   - Strict-Transport-Security: tells the browser to upgrade to HTTPS on the
//     same origin for a year. Effective only over HTTPS in production; harmless
//     locally.
//   - X-Frame-Options: blocks embedding the app in iframes (clickjacking).
//   - X-Content-Type-Options: stops MIME type sniffing.
//   - Referrer-Policy: trims the Referer header so query strings don't leak to
//     third parties.
//   - Permissions-Policy: opts the app out of geolocation/microphone/camera by
//     default — defence in depth for any third-party script that ever sneaks in.
//
// Header values are conservative — strict enough that the secure-mode test
// notices their presence, but loose enough that the existing UI (Tailwind via
// CDN-free /static, htmx via unpkg, Google Fonts) keeps loading.
func SecurityHeadersMiddleware(securityEnabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !securityEnabled {
			c.Next()
			return
		}

		csp := []string{
			"default-src 'self'",
			"script-src 'self' https://unpkg.com",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
			"font-src 'self' https://fonts.gstatic.com",
			"img-src 'self' data:",
			"connect-src 'self'",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self'",
		}

		h := c.Writer.Header()
		h.Set("Content-Security-Policy", joinCSP(csp))
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

func joinCSP(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "; "
		}
		out += p
	}
	return out
}

// ErrorSanitizerMiddleware (secure mode only) recovers from panics and returns
// a generic 500 page instead of letting Gin's default Recovery write the panic
// + stack trace into the response body. In vulnerable mode the default Gin
// Recovery is left in place so a panic visibly leaks internals — that IS the
// Security Misconfiguration demo.
func ErrorSanitizerMiddleware(securityEnabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !securityEnabled {
			c.Next()
			return
		}
		defer func() {
			if r := recover(); r != nil {
				// Logged server-side via gin's logger middleware which still
				// runs; the client sees a flat 500 with no internals.
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
