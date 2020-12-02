package main

import (
	"fmt"
	"os"
	"path/filepath"

	res "github.com/xrfang/go-res"
)

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	PROJ_ROOT, _ := filepath.Abs(filepath.Dir(os.Args[0]) + "/..")
	assert(os.Chdir(PROJ_ROOT))
	root := filepath.Join(PROJ_ROOT, "resources")
	fmt.Printf("pack: processing... ")
	assert(res.Pack(root, "dk"))
	fmt.Println("done")
}
