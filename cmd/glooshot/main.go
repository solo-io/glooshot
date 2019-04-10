package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("todo - apiserver runner")
	http.HandleFunc("/todo", handleTODO)
	dh := defaultHandler{}
	http.Handle("/", dh)
	http.ListenAndServe("localhost:8085", nil)
}

func handleTODO(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "TODO")
}

type defaultHandler struct{}

func (d defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from default")
}
