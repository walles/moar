package moar

import (
	"bytes"
	"fmt"
	"os"

	"github.com/alecthomas/chroma/v2/formatters"
)

// Almost verbatim copied from here, this must compile!
//
// https://github.com/Friends-Of-Noso/NosoData-Go/blob/82de894968e752d6d93d779ecf57db78b10c4acf/cmd/block.go#L145-L163
func _doNotCall_compileTestOnly() {
	blockNumber := 12_345
	buf := new(bytes.Buffer)
	options := ReaderOptions{}

	reader, err := NewReaderFromStream(
		fmt.Sprintf("Block: %d", blockNumber),
		buf,
		formatters.TTY,
		options,
	)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	pager := NewPager(reader)
	pager.WrapLongLines = true

	err = pager.Page()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
