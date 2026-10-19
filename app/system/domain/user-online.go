package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
)

/*
CREATE TABLE `user_online` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uuid` char(32) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL DEFAULT '' COMMENT '用户标识',
  `token` varchar(255) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL DEFAULT '' COMMENT '用户token',
  `create_time` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '登录时间',
  `user_name` varchar(255) NOT NULL COMMENT '用户名',
  `ip` varchar(120) NOT NULL DEFAULT '' COMMENT '登录ip',
  `explorer` varchar(30) NOT NULL DEFAULT '' COMMENT '浏览器',
  `os` varchar(30) NOT NULL DEFAULT '' COMMENT '操作系统',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uni_token` (`token`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COMMENT='用户在线状态表';
*/

// UserOnline 用户在线状态 model
type UserOnline struct {
	ID         uint   `json:"id" form:"id" gorm:"column:id;type:int(10) unsigned auto_increment;primary_key;not null;comment:主键ID"`              // 主键ID
	UUID       string `json:"uuid" form:"uuid" gorm:"column:uuid;type:char(32);not null;default:'';comment:用户标识"`                                // 用户标识
	Token      string `json:"token" form:"token" gorm:"column:token;type:varchar(255);not null;default:'';unique;comment:用户token"`               // 用户token
	CreateTime uint   `json:"create_time" form:"create_time" gorm:"column:create_time;type:bigint(20) unsigned;not null;default:0;comment:登录时间"` // 登录时间（13位时间戳）
	UserName   string `json:"user_name" form:"user_name" gorm:"column:user_name;type:varchar(255);not null;comment:用户名"`                         // 用户名
	IP         string `json:"ip" form:"ip" gorm:"column:ip;type:varchar(120);not null;default:'';comment:登录ip"`                                  // 登录ip
	Explorer   string `json:"explorer" form:"explorer" gorm:"column:explorer;type:varchar(30);not null;default:'';comment:浏览器"`                  // 浏览器
	OS         string `json:"os" form:"os" gorm:"column:os;type:varchar(30);not null;default:'';comment:操作系统"`                                   // 操作系统
}

// TableName 指定表名
func (m *UserOnline) TableName() string {
	return "user_online"
}

// UserOnlineSearchRequest 用户在线状态搜索请求结构体
type UserOnlineSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword  string `json:"keyword" form:"keyword"`     // 关键字
	UUID     string `json:"uuid" form:"uuid"`           // 用户标识
	Token    string `json:"token" form:"token"`         // 用户token
	UserName string `json:"user_name" form:"user_name"` // 用户名
	IP       string `json:"ip" form:"ip"`               // 登录ip
	Explorer string `json:"explorer" form:"explorer"`   // 浏览器
	OS       string `json:"os" form:"os"`               // 操作系统
	ID       *uint  `json:"id" form:"id"`               // 主键ID

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表
}

// UserOnlineCreateRequest 用户在线状态添加请求结构体
type UserOnlineCreateRequest struct {
	UUID     string `json:"uuid" form:"uuid"`                       // 用户标识
	Token    string `json:"token" form:"token" validate:"required"` // 用户token
	UserName string `json:"user_name" form:"user_name"`             // 用户名
	IP       string `json:"ip" form:"ip"`                           // 登录ip
	Explorer string `json:"explorer" form:"explorer"`               // 浏览器
	OS       string `json:"os" form:"os"`                           // 操作系统
}

// UserOnlineForceLogoutRequest 强退用户请求结构体
type UserOnlineForceLogoutRequest struct {
	Token string `json:"token" form:"token" validate:"required"` // 用户token
}
