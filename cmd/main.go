package main

import (
	"fmt"
	"srmt-admin/internal/config"
)

func main() {
	cfg := config.MustLoad()

	fmt.Printf("%#v\n", cfg)
}
