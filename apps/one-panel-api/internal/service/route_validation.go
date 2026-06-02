package service

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"
)

func (c *ControlPlane) validateMatchValue(matchType, matchValue string) link.MatchValueValidation {
	if !c.isValidEnum("match_type", matchType) {
		return link.MatchValueValidation{Valid: false, Format: matchType, Message: "Unknown match type"}
	}
	switch matchType {
	case domain.MatchTypeDomain:
		pattern := `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
		matched, _ := regexp.MatchString(pattern, matchValue)
		if matched {
			return link.MatchValueValidation{Valid: true, Format: "domain", Message: "Valid domain format"}
		}
		return link.MatchValueValidation{Valid: false, Format: "domain", Message: "Invalid domain format"}
	case domain.MatchTypeDomainSuffix:
		if !strings.HasPrefix(matchValue, ".") && !strings.HasPrefix(matchValue, "*.") {
			return link.MatchValueValidation{Valid: false, Format: "domain_suffix", Message: "Domain suffix must start with . or *."}
		}
		return link.MatchValueValidation{Valid: true, Format: "domain_suffix", Message: "Valid domain suffix"}
	case domain.MatchTypeIPCIDR:
		_, _, err := net.ParseCIDR(matchValue)
		if err != nil {
			return link.MatchValueValidation{Valid: false, Format: "ip_cidr", Message: "Invalid CIDR notation"}
		}
		return link.MatchValueValidation{Valid: true, Format: "ip_cidr", Message: "Valid CIDR notation"}
	case domain.MatchTypeIPRange:
		parts := strings.SplitN(matchValue, "-", 2)
		if len(parts) != 2 || net.ParseIP(strings.TrimSpace(parts[0])) == nil || net.ParseIP(strings.TrimSpace(parts[1])) == nil {
			return link.MatchValueValidation{Valid: false, Format: "ip_range", Message: "Invalid IP range format"}
		}
		return link.MatchValueValidation{Valid: true, Format: "ip_range", Message: "Valid IP range"}
	case domain.MatchTypePort:
		p, err := strconv.Atoi(matchValue)
		if err != nil || p < 1 || p > 65535 {
			return link.MatchValueValidation{Valid: false, Format: "port", Message: "Port must be between 1 and 65535"}
		}
		return link.MatchValueValidation{Valid: true, Format: "port", Message: "Valid port"}
	case domain.MatchTypeURLRegex:
		_, err := regexp.Compile(matchValue)
		if err != nil {
			return link.MatchValueValidation{Valid: false, Format: "url_regex", Message: fmt.Sprintf("Invalid regex: %s", err.Error())}
		}
		return link.MatchValueValidation{Valid: true, Format: "url_regex", Message: "Valid regex pattern"}
	case domain.MatchTypeDefault:
		return link.MatchValueValidation{Valid: true, Format: "default", Message: "Default match type"}
	}
	return link.MatchValueValidation{Valid: false, Format: matchType, Message: "Unknown match type"}
}

func (c *ControlPlane) validateRouteRule(actionType string, chainID string, destinationScope string, matchType string, matchValue string) error {
	if matchType == "" || matchValue == "" || actionType == "" {
		return invalidInput("invalid_route_rule_payload")
	}
	if !c.isValidEnum("action_type", actionType) {
		return invalidInput("invalid_route_rule_payload")
	}
	if !c.validateMatchValue(matchType, matchValue).Valid {
		return invalidInput("invalid_route_rule_payload")
	}
	switch actionType {
	case domain.ActionTypeChain:
		if chainID == "" {
			return invalidInput("invalid_route_rule_payload")
		}
		found := false
		for _, chain := range c.store.ListChains() {
			if chain.ID == chainID {
				found = true
				break
			}
		}
		if !found {
			return invalidInput("invalid_route_rule_payload")
		}
	case domain.ActionTypeDirect:
		if destinationScope == "" {
			return invalidInput("invalid_route_rule_payload")
		}
		if !c.scopeExists(destinationScope) {
			return invalidInput("invalid_route_rule_payload")
		}
	}
	return nil
}
