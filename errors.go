package rm

import (
	"fmt"
)

type notFound struct {
	message string
}

func (n notFound) Error() string {
	return n.message
}

func asNotFound(e error) error {
	return notFound{fmt.Sprintf("Not found: %v", e)}
}

func IsNotFound(err error) bool {
	_, ok := err.(notFound)
	return ok
}
