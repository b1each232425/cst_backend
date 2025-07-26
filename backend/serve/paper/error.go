package paper

import (
	"errors"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRecordNotFound                = pgx.ErrNoRows
	ErrPaperNotFound                 = errors.New("paper not found")
	ErrInvalidUserID                 = errors.New("invalid user ID")
	ErrEmptyPaperIDs                 = errors.New("paper IDs cannot be empty")
	ErrPaperAlreadyDeletedOrUnNormal = errors.New("paper already deleted or unnormal")
	ErrInvalidPaperID                = errors.New("试卷ID不合规")
	ErrUserForbidden                 = errors.New("user forbidden")
	ErrPaperLocked                   = errors.New("paper locked")
	ErrLockAcquire                   = errors.New("paper lock acquired failed")
	ErrLockReleased                  = errors.New("paper lock released failed")
	ErrInvalidPermissionLevel        = errors.New("invalid permission level")
	ErrInvalidPageSize               = errors.New("invalid page size")
	ErrInvalidPageNumber             = errors.New("invalid page number")
	ErrInvalidCategory               = errors.New("invalid category")
	ErrInvalidName                   = errors.New("invalid name")
	ErrInvalidTags                   = errors.New("invalid tags")
	ErrInvalidStatus                 = errors.New("invalid status")
	ErrInvalidCreateTime             = errors.New("invalid create time")
	ErrInvalidUpdateTime             = errors.New("invalid update time")
	ErrEmptyGroupID                  = errors.New("Invalid group ID")
	ErrEmptyQuestionIDs              = errors.New("Invalid question IDs")
	ErrNotPaperCreator               = errors.New("Not a paper creator")
)
