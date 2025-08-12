package paper

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// Package level error definitions
var (
	// 数据库记录不存在错误
	ErrRecordNotFound = pgx.ErrNoRows
	// 用户ID相关错误
	ErrInvalidUserID = errors.New("用户ID无效或未找到")
	// 角色ID相关错误
	ErrInvalidRoleID = errors.New("角色ID无效或未找到")
	// 权限相关错误
	ErrWithoutPermission = errors.New("当前用户没有操作权限")
	// 试卷ID数组为空错误
	ErrEmptyPaperIDs = errors.New("试卷ID数组为空")
	// 试卷ID无效错误
	ErrInvalidPaperID = errors.New("试卷ID无效或未找到")
	// 题组数组为空错误
	ErrEmptyGroupID = errors.New("题组数组为空")
)
