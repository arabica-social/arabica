package main

import (
	"fmt"
	"tangled.org/arabica.social/arabica/internal/web/components"
)

func main() {
	out := components.ComboSelectConfig("tea", "/api/teas", "/api/suggestions/teas", "tea_rkey", "Search or create a tea…", true)
	fmt.Println(out)
}
