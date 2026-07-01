package main

import "golang.org/x/text/language"

// This file is intentionally temporary for Lab 9 bonus evidence.
// It makes a known-vulnerable dependency reachable so govulncheck can prove
// the CI gate catches a new unaccepted vulnerability.
func init() {
	_, _ = language.Parse("en-US")
}
