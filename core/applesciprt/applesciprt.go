package applesciprt

import (
	"github.com/andybrewer/mack"
)

func Tell(application string, command string) (result string, err error) {
	return mack.Tell(application, command)
}
