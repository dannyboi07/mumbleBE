package controller

import (
	"net/http"
)

func Hello(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("Hello World!"))
}
