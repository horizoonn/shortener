package request

import (
	"net"
	"net/http"
	"strings"
)

type IPResolver struct {
	trusted []*net.IPNet
}

func NewIPResolver(trustedCIDRs []string) (*IPResolver, error) {
	trusted, err := parseCIDRs(trustedCIDRs)
	if err != nil {
		return nil, err
	}

	return &IPResolver{trusted: trusted}, nil
}

func (r *IPResolver) Resolve(req *http.Request) string {
	remoteIP := extractRemoteIP(req.RemoteAddr)
	if remoteIP == "" {
		return ""
	}

	if !r.isTrusted(remoteIP) {
		return remoteIP
	}

	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := lastNonTrustedIP(xff, r.trusted); ip != "" {
			return ip
		}
	}

	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		if parsed := net.ParseIP(xri); parsed != nil {
			return xri
		}
	}

	return remoteIP
}

func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := firstValidIP(xff); ip != "" {
			return ip
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if net.ParseIP(host) == nil {
		return ""
	}

	return host
}

func (r *IPResolver) isTrusted(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, cidr := range r.trusted {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

func lastNonTrustedIP(xff string, trusted []*net.IPNet) string {
	parts := strings.Split(xff, ",")

	for i := len(parts) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(parts[i])
		ip := net.ParseIP(trimmed)
		if ip == nil {
			continue
		}

		if !isInCIDRs(ip, trusted) {
			return trimmed
		}
	}

	return firstValidIP(xff)
}

func firstValidIP(xff string) string {
	for _, part := range strings.Split(xff, ",") {
		trimmed := strings.TrimSpace(part)
		if net.ParseIP(trimmed) != nil {
			return trimmed
		}
	}

	return ""
}

func extractRemoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	if net.ParseIP(host) == nil {
		return ""
	}

	return host
}

func parseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(cidrs))

	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			ip := net.ParseIP(cidr)
			if ip == nil {
				return nil, &net.ParseError{Type: "CIDR address", Text: cidr}
			}

			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			network = &net.IPNet{IP: ip, Mask: mask}
		}

		networks = append(networks, network)
	}

	return networks, nil
}

func isInCIDRs(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}
