package flags

import (
	"flag"
	"fmt"
)

var sizeOption = Option[int, int]{
	func() *int {
		return flag.Int("s", 5, "Sets the size x size of the collage")
	},
	func(size int) (int, error) {
		if size <= 2 || size > 10 {
			return size, fmt.Errorf("size %v needs to be between 3 and 10", size)
		}
		return size, nil
	},
}
