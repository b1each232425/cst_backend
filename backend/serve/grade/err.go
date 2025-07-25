package grade

import "errors"

var (
	ErrNilID                  = errors.New("ID is nil")
	ErrNilExamInfo            = errors.New("exam info is nil")
	ErrExamIsNotOver          = errors.New("exam has not ended yet")
	ErrInvalidID              = errors.New("invalid ID, expected a positive integer, example: 20251122")
	ErrInvalidPage            = errors.New("invalid page, expected a positive integer, example: 1")
	ErrInvalidPageSize        = errors.New("invalid page size, expected a positive integer, example: 10")
	ErrNoSuchRow              = errors.New("no such rows error")
	ErrNoSuchQueryArg         = errors.New("no such query arg error")
	ErrInvalidScore           = errors.New("invalid score error")
	ErrInvalidRate            = errors.New("invalid rate error")
	ErrPermissionDenied       = errors.New("no operation permission error")
	ErrUnknownPermissionLevel = errors.New("unknown permission level error")
	ErrGetContextInfo         = errors.New("get context info error")
	ErrEmptyQuery             = errors.New("empty query error")
	ErrInvalidArg             = errors.New("invalid argument error")

	ErrNilDBConn      = errors.New("nil db conn error")
	ErrInvalidPaperID = errors.New("invalid paper id")
)
