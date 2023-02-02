package http

import (
	"log"
	"net/http"
)

func ListenAndServe() {
	fs := http.FileServer(http.Dir("./static"))

	http.Handle("/", fs)

	log.Print("web server listen on :5555...")

	err := http.ListenAndServe(":5555", nil)
	if err != nil {
		log.Printf("web serve fail: %s", err.Error())
	}
}
