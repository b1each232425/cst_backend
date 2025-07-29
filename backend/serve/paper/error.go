package paper

import (
	"errors"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRecordNotFound                = pgx.ErrNoRows
	ErrPaperNotFound                 = errors.New("试卷没找到")
	ErrInvalidUserID                 = errors.New("用户ID不合规")
	ErrEmptyPaperIDs                 = errors.New("试卷ID数组为空")
	ErrPaperAlreadyDeletedOrUnNormal = errors.New("试卷已经被删除或异常")
	ErrInvalidPaperID                = errors.New("试卷ID不合规")
	ErrUserForbidden                 = errors.New("用户禁止访问当前模块")
	ErrPaperLocked                   = errors.New("试卷锁已被获取")
	ErrLockAcquire                   = errors.New("获取试卷锁失败")
	ErrLockReleased                  = errors.New("释放试卷锁失败")
	ErrInvalidPermissionLevel        = errors.New("无效权限等级")
	ErrInvalidPageSize               = errors.New("页大小不合规")
	ErrInvalidPageNumber             = errors.New("页数不合规")
	ErrInvalidCategory               = errors.New("试卷用途不合规")
	ErrInvalidName                   = errors.New("不合规名字")
	ErrInvalidTags                   = errors.New("不合规标签")
	ErrInvalidStatus                 = errors.New("不合规状态")
	ErrInvalidCreateTime             = errors.New("不合规创建时间")
	ErrInvalidUpdateTime             = errors.New("不合规更新时间")
	ErrEmptyGroupID                  = errors.New("题组数组为空")
	ErrEmptyQuestionIDs              = errors.New("题目数组为空")
	ErrNotPaperCreator               = errors.New("不是试卷创建者")
)
