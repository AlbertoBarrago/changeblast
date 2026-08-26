package main

import (
	"fmt"

	"example.com/simplego/internal/auth"
)

func main() {
	fmt.Println(auth.SignToken("req"))
}
