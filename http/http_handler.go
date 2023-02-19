package http

import (
	"fmt"
	"net/http"
)

func ListenAndServe() {
	fs := http.FileServer(http.Dir("./static"))

	http.Handle("/", fs)

	fmt.Printf("web server listen on :5555...\n")

	err := http.ListenAndServe(":5555", nil)
	if err != nil {
		fmt.Printf("web serve fail: %s", err.Error())
	}
}
