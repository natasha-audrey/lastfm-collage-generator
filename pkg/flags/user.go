package flags

import (
	"flag"
)

var userOption = Option[string, string]{
	func() *string {
		return flag.String("u", "tashayasha", "The user to query")
	},
	func(user string) (string, error) {
		if user == "" {
			return "tashayasha", nil
		}
		return user, nil
	},
}
