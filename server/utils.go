package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

func prettyPrintJSON(in string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(in), "", "\t")
	if err != nil {
		return in
	}
	return out.String()
}

func jsonCodeBlock(in string) string {
	return fmt.Sprintf("``` json\n%s\n```", in)
}

func codeBlock(in string) string {
	return fmt.Sprintf("```\n%s\n```", in)
}

func inlineCode(in string) string {
	return fmt.Sprintf("`%s`", in)
}

func standardizeName(name string) string {
	return strings.ToLower(name)
}

func validLicenseOption(license string) bool {
	return license == licenseOptionEnterprise ||
		license == licenseOptionProfessional ||
		license == licenseOptionE20 ||
		license == licenseOptionE10 ||
		license == licenseOptionTE
}

func validVersionOption(version string) error {
	v, err := semver.Parse(version)
	if err != nil {
		// in case using a non version like latest or any other tag
		return nil
	}

	expectedRange, err := semver.ParseRange("< 5.12.0")
	if err != nil {
		return errors.Wrapf(err, "failed to parse the version range for %s", version)
	}
	if expectedRange(v) {
		return errors.Errorf("invalid Version option %s, must be greater than 5.12.0", version)
	}

	return nil
}

func validInstallationName(name string) bool {
	return installationNameMatcher.MatchString(name)
}

// Contains finds if needle is inside haystack
func Contains[T comparable](haystack []T, needle T) bool {
	for i := range haystack {
		if haystack[i] == needle {
			return true
		}
	}
	return false
}

// NewBool returns a pointer to a given bool.
func NewBool(b bool) *bool { return &b }

// NewInt returns a pointer to a given int.
func NewInt(n int) *int { return &n }

// NewInt32 returns a pointer to a given int32.
func NewInt32(n int32) *int32 { return &n }

// NewInt64 returns a pointer to a given int64.
func NewInt64(n int64) *int64 { return &n }

// NewString returns a pointer to a given string.
func NewString(s string) *string { return &s }
