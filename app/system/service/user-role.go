package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
)

type UserRole struct{}

// GetCount 获取用户角色关联列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取用户角色关联列表数据数量
//@param: searchParams *domain.UserRoleSearchRequest
//@return: err error, total int64
func (s UserRole) GetCount(searchParams *domain.UserRoleSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.UserRole{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取用户角色关联列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取用户角色关联列表数据
//@param: searchParams *domain.UserRoleSearchRequest
//@return: err error, list interface{}, total int64
func (s UserRole) GetList(searchParams *domain.UserRoleSearchRequest) (err error, list []domain.UserRole, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.UserRole{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("user_id asc")
		if searchParams.OrderBy != "" {
			db.Order(searchParams.OrderBy)
		}
		// 分页
		if searchParams.PageSize > 0 {
			limit := searchParams.PageSize
			offset := searchParams.PageSize * (searchParams.Page - 1)
			db.Limit(limit).Offset(offset)
		}
		err = db.Find(&list).Error
	}
	return err, list, total
}

//@author: [bluefrog](https://github.com/freewu)
//@function: parseFilter
//@description: 用户角色关联列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.UserRoleSearchRequest
//@return: *gorm.DB
func (s UserRole) parseFilter(db *gorm.DB, searchParams *domain.UserRoleSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.UserID != nil { // 用户ID
		db = db.Where("user_id = ?", searchParams.UserID)
	}
	if len(searchParams.UserIDIn) > 0 { // 用户ID 数组
		db = db.Where("user_id IN (?)", searchParams.UserIDIn)
	}
	if searchParams.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", searchParams.RoleID)
	}
	if len(searchParams.RoleIDIn) > 0 { // 角色ID 数组
		db = db.Where("role_id IN (?)", searchParams.RoleIDIn)
	}
	return db
}

// Create 添加用户角色关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加用户角色关联
//@param: data *domain.UserRoleCreateRequest
//@return: error
func (s UserRole) Create(data *domain.UserRoleCreateRequest) error {
	userRole := new(domain.UserRole)
	userRole.UserID = data.UserID
	userRole.RoleID = data.RoleID

	return domain.DB.Create(&userRole).Error
}

// Delete 删除用户角色关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除用户角色关联
//@param: data *domain.UserRoleDeleteRequest
//@return: err error
func (s UserRole) Delete(data *domain.UserRoleDeleteRequest) (err error) {
	db := domain.DB.Model(&domain.UserRole{})
	// 条件过滤
	if data.UserID != nil { // 用户ID
		db = db.Where("user_id = ?", data.UserID)
	}
	if data.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", data.RoleID)
	}
	if err = db.Unscoped().Delete(&domain.UserRole{}).Error; err != nil {
		return err
	}
	return nil
}
