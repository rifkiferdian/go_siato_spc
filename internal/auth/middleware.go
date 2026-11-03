package auth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const principalContextKey = "auth.principal"

func (s *Service) LoadPrincipal() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(SessionCookieName)
		if err == nil {
			principal, authErr := s.Authenticate(c.Request.Context(), cookie)
			if authErr == nil {
				c.Set(principalContextKey, principal)
			} else {
				s.ClearSessionCookie(c.Writer)
			}
		}
		c.Next()
	}
}

func RequireAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentPrincipal(c) == nil {
			returnURL := c.Request.URL.RequestURI()
			c.Redirect(http.StatusSeeOther, "/panel-admin-stieww?return_url="+url.QueryEscape(returnURL))
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := CurrentPrincipal(c)
		if principal == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !principal.HasPermission(permission) && !principal.HasRole("Super Admin") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func CurrentPrincipal(c *gin.Context) *Principal {
	value, ok := c.Get(principalContextKey)
	if !ok {
		return nil
	}
	principal, _ := value.(*Principal)
	return principal
}

func SafeReturnURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\r\n") {
		return "/dashboard"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/dashboard"
	}
	return value
}
