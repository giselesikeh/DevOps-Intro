package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	method := os.Getenv("REQUEST_METHOD")
	path := os.Getenv("PATH_INFO")

	if method == "" {
		method = "GET"
	}

	if path == "" {
		path = "/time"
	}

	if method != "GET" {
		fmt.Fprintln(os.Stderr, "method not allowed")
		os.Exit(1)
	}

	if path != "/time" {
		fmt.Fprintln(os.Stderr, "not found")
		os.Exit(1)
	}

	nowUTC := time.Now().UTC()
	moscow := nowUTC.Add(3 * time.Hour)

	iso := moscow.Format("2006-01-02T15:04:05") + "+03:00"
	hourMinute := moscow.Format("15:04")

	fmt.Fprintf(
		os.Stdout,
		"{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q,\"timezone\":\"Europe/Moscow\",\"utc_offset\":\"+03:00\"}\n",
		nowUTC.Unix(),
		iso,
		hourMinute,
	)
}
