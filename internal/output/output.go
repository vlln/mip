package output

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func Lines(w io.Writer, lines ...string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}
