package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
)

type RoleDept struct{}

// GetCount 获取角色部门关联列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取角色部门关联列表数据数量
//@param: searchParams *domain.RoleDeptSearchRequest
//@return: err error, total int64
func (s RoleDept) GetCount(searchParams *domain.RoleDeptSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.RoleDept{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取角色部门关联列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取角色部门关联列表数据
//@param: searchParams *domain.RoleDeptSearchRequest
//@return: err error, list interface{}, total int64
func (s RoleDept) GetList(searchParams *domain.RoleDeptSearchRequest) (err error, list []domain.RoleDept, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.RoleDept{})
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
//@description: 角色部门关联列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.RoleDeptSearchRequest
//@return: *gorm.DB
func (s RoleDept) parseFilter(db *gorm.DB, searchParams *domain.RoleDeptSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", searchParams.RoleID)
	}
	if len(searchParams.RoleIDIn) > 0 { // 角色ID 数组
		db = db.Where("role_id IN (?)", searchParams.RoleIDIn)
	}
	if searchParams.DeptID != nil { // 部门ID
		db = db.Where("dept_id = ?", searchParams.DeptID)
	}
	if len(searchParams.DeptIDIn) > 0 { // 部门ID 数组
		db = db.Where("dept_id IN (?)", searchParams.DeptIDIn)
	}
	return db
}

// Create 添加角色部门关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加角色部门关联
//@param: data *domain.RoleDeptCreateRequest
//@return: error
func (s RoleDept) Create(data *domain.RoleDeptCreateRequest) error {
	roleDept := new(domain.RoleDept)
	roleDept.RoleID = data.RoleID
	roleDept.DeptID = data.DeptID

	return domain.DB.Create(&roleDept).Error
}

// Delete 删除角色部门关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除角色部门关联
//@param: data *domain.RoleDeptDeleteRequest
//@return: err error
func (s RoleDept) Delete(data *domain.RoleDeptDeleteRequest) (err error) {
	db := domain.DB.Model(&domain.RoleDept{})
	// 条件过滤
	if data.RoleID != nil { // 角色ID
		db = db.Where("role_id = ?", data.RoleID)
	}
	if data.DeptID != nil { // 部门ID
		db = db.Where("dept_id = ?", data.DeptID)
	}
	if err = db.Unscoped().Delete(&domain.RoleDept{}).Error; err != nil {
		return err
	}
	return nil
}
