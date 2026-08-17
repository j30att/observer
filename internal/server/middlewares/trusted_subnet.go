package middlewares

import (
	"net/http"
	"net/netip"
	"strings"
)

const realIPHeader = "X-Real-IP"

// TrustedSubnet allows metric update requests only from the configured CIDR.
func TrustedSubnet(trustedSubnet string) func(http.Handler) http.Handler {
	return trustedSubnetMiddleware(trustedSubnet, func(*http.Request) bool {
		return true
	})
}

// TrustedSubnetForMetricUpdates allows metric update requests only from the configured CIDR.
func TrustedSubnetForMetricUpdates(trustedSubnet string) func(http.Handler) http.Handler {
	return trustedSubnetMiddleware(trustedSubnet, isMetricUpdateRequest)
}

func trustedSubnetMiddleware(trustedSubnet string, shouldCheck func(*http.Request) bool) func(http.Handler) http.Handler {
	trustedSubnet = strings.TrimSpace(trustedSubnet)
	if trustedSubnet == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	prefix, err := netip.ParsePrefix(trustedSubnet)
	if err != nil {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldCheck(r) {
				next.ServeHTTP(w, r)
				return
			}

			realIP, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get(realIPHeader)))
			if err != nil || !prefix.Contains(realIP) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isMetricUpdateRequest(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	return r.URL.Path == "/updates" || r.URL.Path == "/update" || strings.HasPrefix(r.URL.Path, "/update/")
}
