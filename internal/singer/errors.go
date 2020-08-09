package singer

import "errors"

var ErrDuplicate = errors.New("Singer already exist")
var ErrNotFound = errors.New("Singer not found")
var ErrBadValue = errors.New("Singer record has a bad value")
var ErrUnknown = errors.New("Internal persistence error")
