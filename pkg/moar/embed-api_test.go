package moar

import (
	"bytes"
	"fmt"
	"os"
)

// Inspired from here:
// https://github.com/Friends-Of-Noso/NosoData-Go/blob/82de894968e752d6d93d779ecf57db78b10c4acf/cmd/block.go#L145-L163
//
// This function is not meant to be called (because then it would start paging
// which is impractical during testing). It's just here to demonstrate how the
// API can be used, and to ensure the API compiles.
func _demoUsageShouldCompile() {
	blockNumber := 12_345
	buf := new(bytes.Buffer)

	err := PageFromStream(buf, Options{
		Title:         fmt.Sprintf("Block: %d", blockNumber),
		WrapLongLines: true,
	})
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
