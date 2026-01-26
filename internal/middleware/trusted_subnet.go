package middleware

import (
	"net"
	"net/http"
)

func IsIPInSubnet(ipStr, subnetStr string) bool {
	if subnetStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, subnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		return false
	}

	return subnet.Contains(ip)
}

func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(trustedSubnet) > 0 {
				clientIP := r.Header.Get("X-Real-IP")
				if clientIP == "" {
					http.Error(w, "X-Real-IP header is required", http.StatusForbidden)
					return
				}

				if !IsIPInSubnet(clientIP, trustedSubnet) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
