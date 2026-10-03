package main

import (
	"fmt"
	"net/http"
)

func main() {
	response, err := http.Get("http://example.com")
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	fmt.Println(response.Status)
}
