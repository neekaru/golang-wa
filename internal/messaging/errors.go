package messaging

import (
	"fmt"
	"time"
)

// DuplicateMessageError indicates a duplicate message was blocked.
type DuplicateMessageError struct {
	RetryAfter time.Duration
}

func (e *DuplicateMessageError) Error() string {
	if e.RetryAfter > 0 {
		seconds := int(e.RetryAfter.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		return fmt.Sprintf("message cooldown active, retry after %d seconds", seconds)
	}
	return "message cooldown active"
}

func isDuplicateMessageError(err error) (*DuplicateMessageError, bool) {
	if err == nil {
		return nil, false
	}
	dupErr, ok := err.(*DuplicateMessageError)
	return dupErr, ok
}
