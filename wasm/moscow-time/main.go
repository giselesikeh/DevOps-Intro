package main

import (
	"fmt"
	"net/http"
	"time"

	spinhttp "github.com/spinframework/spin-go-sdk/v2/http"
)

func init() {
	spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `{"error":"method not allowed"}`)
			return
		}

		nowUTC := time.Now().UTC()
		moscow := nowUTC.Add(3 * time.Hour)

		iso := moscow.Format("2006-01-02T15:04:05") + "+03:00"
		hourMinute := moscow.Format("15:04")

		fmt.Fprintf(
			w,
			"{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q,\"timezone\":\"Europe/Moscow\",\"utc_offset\":\"+03:00\"}",
			nowUTC.Unix(),
			iso,
			hourMinute,
		)
	})
}
