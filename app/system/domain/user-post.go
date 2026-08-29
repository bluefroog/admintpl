package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
)

// UserPost 用户与岗位关联 model
type UserPost struct {
	UserID uint `json:"user_id" form:"user_id" gorm:"column:user_id;type:bigint(20);primary_key;not null;comment:用户ID"` // 用户ID
	PostID uint `json:"post_id" form:"post_id" gorm:"column:post_id;type:bigint(20);primary_key;not null;comment:岗位ID"` // 岗位ID
}

// TableName 指定表名
func (m *UserPost) TableName() string {
	return "sys_user_post"
}

// UserPostSearchRequest 用户与岗位关联搜索请求结构体
type UserPostSearchRequest struct {
	request.PageInfo
	request.SortInfo

	UserID *uint `json:"user_id" form:"user_id"` // 用户ID
	PostID *uint `json:"post_id" form:"post_id"` // 岗位ID

	UserIDIn []uint `json:"userIDIn"` // 用户ID 数组
	PostIDIn []uint `json:"postIDIn"` // 岗位ID 数组
}

// UserPostCreateRequest 用户与岗位关联添加请求结构体
type UserPostCreateRequest struct {
	UserID uint `json:"user_id" form:"user_id" validate:"required"` // 用户ID
	PostID uint `json:"post_id" form:"post_id" validate:"required"` // 岗位ID
}

// UserPostDeleteRequest 用户与岗位关联删除请求结构体
type UserPostDeleteRequest struct {
	UserID *uint `json:"user_id" form:"user_id"` // 用户ID
	PostID *uint `json:"post_id" form:"post_id"` // 岗位ID
}
