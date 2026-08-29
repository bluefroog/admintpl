package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
)

type RoleMenu struct{}

// GetCount 获取角色菜单关联列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取角色菜单关联列表数据数量
//@param: searchParams *domain.RoleMenuSearchRequest
//@return: err error, total int64
func (s RoleMenu) GetCount(searchParams *domain.RoleMenuSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.RoleMenu{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取角色菜单关联列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取角色菜单关联列表数据
//@param: searchParams *domain.RoleMenuSearchRequest
//@return: err error, list interface{}, total int64
func (s RoleMenu) GetList(searchParams *domain.RoleMenuSearchRequest) (err error, list []domain.RoleMenu, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.RoleMenu{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("role_id asc")
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
//@description: 角色菜单关联列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.RoleMenuSearchRequest
//@return: *gorm.DB
func (s RoleMenu) parseFilter(db *gorm.DB, searchParams *domain.RoleMenuSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", searchParams.RoleID)
	}
	if len(searchParams.RoleIDIn) > 0 { // 角色ID 数组
		db = db.Where("role_id IN (?)", searchParams.RoleIDIn)
	}
	if searchParams.MenuID != nil { // 菜单ID
		db = db.Where("menu_id = ?", searchParams.MenuID)
	}
	if len(searchParams.MenuIDIn) > 0 { // 菜单ID 数组
		db = db.Where("menu_id IN (?)", searchParams.MenuIDIn)
	}
	return db
}

// Create 添加角色菜单关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加角色菜单关联
//@param: data *domain.RoleMenuCreateRequest
//@return: error
func (s RoleMenu) Create(data *domain.RoleMenuCreateRequest) error {
	roleMenu := new(domain.RoleMenu)
	roleMenu.RoleID = data.RoleID
	roleMenu.MenuID = data.MenuID

	return domain.DB.Create(&roleMenu).Error
}

// Delete 删除角色菜单关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除角色菜单关联
//@param: data *domain.RoleMenuDeleteRequest
//@return: err error
func (s RoleMenu) Delete(data *domain.RoleMenuDeleteRequest) (err error) {
	db := domain.DB.Model(&domain.RoleMenu{})
	// 条件过滤
	if data.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", data.RoleID)
	}
	if data.MenuID != nil { // 菜单ID
		db = db.Where("menu_id = ?", data.MenuID)
	}
	if err = db.Unscoped().Delete(&domain.RoleMenu{}).Error; err != nil {
		return err
	}
	return nil
}
