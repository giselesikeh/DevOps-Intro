package main

import "golang.org/x/text/language"

// vulnerableLanguageParseDemo is intentionally temporary for Lab 9 bonus evidence.
// It creates a reachable call path for govulncheck, then will be reverted.
func vulnerableLanguageParseDemo(input string) error {
	_, err := language.Parse(input)
	return err
}
