package KiroAuth

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var awsRegionPattern = regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)

func normalizeRegion(region string) (string, error) {
	region = strings.TrimSpace(strings.ToLower(region))
	if region == "" {
		region = "us-east-1"
	}
	if !awsRegionPattern.MatchString(region) {
		return "", fmt.Errorf("invalid region")
	}
	return region, nil
}

func validateStartURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid startUrl")
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", fmt.Errorf("startUrl must be https")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || (!strings.HasSuffix(host, ".awsapps.com") && host != "view.awsapps.com") {
		return "", fmt.Errorf("invalid startUrl host")
	}
	return u.String(), nil
}
