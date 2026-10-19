package service

import (
	"errors"
	"github.com/bluefroog/admintpl/app/system/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

type Role struct{}

// GetCount 获取角色列表数据数量
// @author: [bluefrog](https://github.com/freewu)
// @function: GetCount
// @description: 获取角色列表数据数量
// @param: searchParams *domain.RoleSearchRequest
// @return: err error, total int64
func (s Role) GetCount(searchParams *domain.RoleSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Role{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取角色列表数据
// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取角色列表数据
// @param: searchParams *domain.RoleSearchRequest
// @return: err error, list interface{}, total int64
func (s Role) GetList(searchParams *domain.RoleSearchRequest) (err error, list []domain.Role, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Role{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		// orderBy := "id desc"
		db.Order("role_id desc")
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

// @author: [bluefrog](https://github.com/freewu)
// @function: parseFilter
// @description: 角色列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.RoleSearchRequest
// @return: *gorm.DB
func (s Role) parseFilter(db *gorm.DB, searchParams *domain.RoleSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 角色编号
		db = db.Where("role_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 角色编号 数组
		db = db.Where("role_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("role_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Name != "" { // 角色名称
		db = db.Where("role_name = ?", searchParams.Name)
	}
	if searchParams.Key != "" { // 角色权限字符串
		db = db.Where("role_key = ?", searchParams.Key)
	}
	if searchParams.Status != nil { // 角色状态（0正常 1停用）
		db = db.Where("status = ?", searchParams.Status)
	}
	if searchParams.MenuCheckStrictly != nil { // 菜单树选择项是否关联显示（0否 1是）
		db = db.Where("menu_check_strictly = ?", searchParams.MenuCheckStrictly)
	}
	if searchParams.DepartmentCheckStrictly != nil { // 部门树选择项是否关联显示（0否 1是）
		db = db.Where("dept_check_strictly = ?", searchParams.Status)
	}
	if searchParams.DataScope != nil { // 数据范围（1：全部数据权限 2：自定数据权限 3：本部门数据权限 4：本部门及以下数据权限）
		db = db.Where("data_scope = ?", searchParams.Status)
	}
	if searchParams.DeleteFlag != nil { // 删除标志（0代表存在 2代表删除）
		db = db.Where("del_flag = ?", searchParams.Status)
	}
	if searchParams.CreateBy != "" { // 创建者
		db = db.Where("create_by LIKE ?", "%"+searchParams.CreateBy+"%")
	}
	if searchParams.UpdateBy != "" { // 更新者
		db = db.Where("update_by LIKE ?", "%"+searchParams.UpdateBy+"%")
	}
	if searchParams.Keyword != "" { // 关键词
		k := strings.Trim(searchParams.Keyword, " \t\r\n")
		k1 := "%" + k + "%"
		db = db.Where("(`role_name` LIKE ? OR `role_key` LIKE ? OR `role_id` = ? )", k1, k1, k)
	}
	return db
}

// Create 添加角色数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加角色数据
// @param: data *domain.RoleAddRequest
// @return: error
func (s Role) Create(data *domain.RoleCreateRequest) error {
	// 判断角色名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的角色名称")
	}
	// 判断角色权限字符串是否唯一
	if s.KeyIsExist(data.Key, nil) {
		return errors.New("已存在相同的角色权限字符串")
	}
	role := new(domain.Role)
	role.Key = data.Key
	role.Name = data.Name
	role.Status = data.Status
	role.Sort = data.Sort
	role.MenuCheckStrictly = data.MenuCheckStrictly
	role.DepartmentCheckStrictly = data.DepartmentCheckStrictly
	role.DataScope = data.DataScope

	role.Remark = data.Remark
	role.CreateBy = "" // todo 通过登录服务获取
	role.CreateTime = time.Now()
	role.UpdateBy = "" // todo 通过登录服务获取
	role.UpdateTime = time.Now()

	return domain.DB.Create(&role).Error
}

// Update 修改角色
// @author: [bluefrog](https://github.com/freewu)
// @function: Update
// @description: 修改角色
// @param: data *model.RoleEditRequest
// @return: err error
func (s Role) Update(data *domain.RoleUpdateRequest) (err error) {
	// 判断名字是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的角色名称")
	}
	// 判断角色权限字符串是否唯一
	if s.KeyIsExist(data.Key, &[]uint{data.ID}) {
		return errors.New("已存在相同的角色权限字符串")
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["Key"] = data.Key
	record["MenuCheckStrictly"] = data.MenuCheckStrictly
	record["DepartmentCheckStrictly"] = data.DepartmentCheckStrictly
	record["DataScope"] = data.DataScope
	record["DeleteFlag"] = data.DeleteFlag
	record["Status"] = data.Status
	record["Sort"] = data.Sort
	record["Remark"] = data.Remark
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	return s.UpdateByMap(data.ID, record)
}

// UpdateByMap 更新指定ID的角色信息
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateByMap
// @description: 更新指定ID的角色信息
// @param: id uint
// @param: updateMap map[string]interface{}
// @return: err error
func (s Role) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := domain.DB.Model(&domain.Role{}).Where("role_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除角色
// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 删除角色
// @param: ids []uint
// @return: err error
func (s Role) Delete(ids []uint) (err error) {
	if err = domain.DB.Where("role_id in (?) ", ids).Unscoped().Delete(&domain.Role{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID角色详情
// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID角色详情
// @param: id uint
// @return: *domain.Role,err error
func (s Role) Detail(id uint) (*domain.Role, error) {
	var detail domain.Role
	db := domain.DB.Where("role_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断角色名称是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: NameIsExist
// @description: 判断角色名称是否存在
// @param: name string 角色名称
// @param: excludeIds *[]uint 不包含的角色ID
// @return: err error
func (s Role) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.RoleSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// KeyIsExist 判断角色编号是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: KeyIsExist
// @description: 判断角色编号是否存在
// @param: code string 角色编号
// @param: excludeIds *[]uint 不包含的区域类型ID
// @return: err error
func (s Role) KeyIsExist(key string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.RoleSearchRequest{
		Key: key,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// AssignMenus 给角色分配菜单权限（先删除原有关联，再批量新增）
// @author: [bluefrog](https://github.com/freewu)
// @function: AssignMenus
// @description: 角色菜单授权
// @param: req *domain.RoleMenuAssignRequest
// @return: err error
func (s Role) AssignMenus(req *domain.RoleMenuAssignRequest) (err error) {
	return domain.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", req.ID).Delete(&domain.RoleMenu{}).Error; err != nil {
			return err
		}
		if len(req.MenuIDs) == 0 {
			return nil
		}
		roleMenus := make([]domain.RoleMenu, 0, len(req.MenuIDs))
		for _, menuId := range req.MenuIDs {
			if menuId == 0 {
				continue
			}
			roleMenus = append(roleMenus, domain.RoleMenu{RoleID: req.ID, MenuID: menuId})
		}
		if len(roleMenus) == 0 {
			return nil
		}
		return tx.Create(&roleMenus).Error
	})
}

// AuthDataScope 修改角色数据权限（数据范围 + 自定义部门）
// @author: [bluefrog](https://github.com/freewu)
// @function: AuthDataScope
// @description: 修改角色数据权限
// @param: req *domain.RoleDataScopeRequest
// @return: err error
func (s Role) AuthDataScope(req *domain.RoleDataScopeRequest) (err error) {
	return domain.DB.Transaction(func(tx *gorm.DB) error {
		record := make(map[string]interface{})
		record["DataScope"] = req.DataScope
		record["UpdateBy"] = "" // todo 通过登录服务获取
		record["UpdateTime"] = time.Now()
		if err := tx.Model(&domain.Role{}).Where("role_id = ?", req.ID).Updates(record).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", req.ID).Delete(&domain.RoleDept{}).Error; err != nil {
			return err
		}
		if len(req.DeptIDs) == 0 {
			return nil
		}
		roleDepts := make([]domain.RoleDept, 0, len(req.DeptIDs))
		for _, deptId := range req.DeptIDs {
			if deptId == 0 {
				continue
			}
			roleDepts = append(roleDepts, domain.RoleDept{RoleID: req.ID, DeptID: deptId})
		}
		if len(roleDepts) == 0 {
			return nil
		}
		return tx.Create(&roleDepts).Error
	})
}

// AllocatedList 查询角色已分配的用户列表
// @author: [bluefrog](https://github.com/freewu)
// @function: AllocatedList
// @description: 查询角色已分配的用户列表
// @param: searchParams *domain.RoleUserListRequest
// @return: err error, list []domain.User, total int64
func (s Role) AllocatedList(searchParams *domain.RoleUserListRequest) (err error, list []domain.User, total int64) {
	sub := domain.DB.Model(&domain.UserRole{}).Select("user_id").Where("role_id = ?", searchParams.RoleID)
	db := domain.DB.Model(&domain.User{}).Where("id IN (?)", sub)
	if searchParams.Username != "" {
		db = db.Where("username LIKE ?", "%"+searchParams.Username+"%")
	}
	if searchParams.Mobile != "" {
		db = db.Where("mobile LIKE ?", "%"+searchParams.Mobile+"%")
	}
	if err = db.Count(&total).Error; err != nil {
		return err, nil, 0
	}
	if total > 0 {
		db.Order("id desc")
		if searchParams.PageSize > 0 {
			limit := searchParams.PageSize
			offset := searchParams.PageSize * (searchParams.Page - 1)
			db.Limit(limit).Offset(offset)
		}
		err = db.Find(&list).Error
	}
	return err, list, total
}

// UnallocatedList 查询角色未分配的用户列表
// @author: [bluefrog](https://github.com/freewu)
// @function: UnallocatedList
// @description: 查询角色未分配的用户列表
// @param: searchParams *domain.RoleUserListRequest
// @return: err error, list []domain.User, total int64
func (s Role) UnallocatedList(searchParams *domain.RoleUserListRequest) (err error, list []domain.User, total int64) {
	sub := domain.DB.Model(&domain.UserRole{}).Select("user_id").Where("role_id = ?", searchParams.RoleID)
	db := domain.DB.Model(&domain.User{}).Where("id NOT IN (?)", sub)
	if searchParams.Username != "" {
		db = db.Where("username LIKE ?", "%"+searchParams.Username+"%")
	}
	if searchParams.Mobile != "" {
		db = db.Where("mobile LIKE ?", "%"+searchParams.Mobile+"%")
	}
	if err = db.Count(&total).Error; err != nil {
		return err, nil, 0
	}
	if total > 0 {
		db.Order("id desc")
		if searchParams.PageSize > 0 {
			limit := searchParams.PageSize
			offset := searchParams.PageSize * (searchParams.Page - 1)
			db.Limit(limit).Offset(offset)
		}
		err = db.Find(&list).Error
	}
	return err, list, total
}

// SelectAllUsers 批量选择用户授权
// @author: [bluefrog](https://github.com/freewu)
// @function: SelectAllUsers
// @description: 批量选择用户授予角色
// @param: req *domain.RoleUserAssignRequest
// @return: err error
func (s Role) SelectAllUsers(req *domain.RoleUserAssignRequest) (err error) {
	if len(req.UserIDs) == 0 {
		return nil
	}
	userRoles := make([]domain.UserRole, 0, len(req.UserIDs))
	for _, userId := range req.UserIDs {
		if userId == 0 {
			continue
		}
		userRoles = append(userRoles, domain.UserRole{UserID: userId, RoleID: req.RoleID})
	}
	if len(userRoles) == 0 {
		return nil
	}
	return domain.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&userRoles).Error
}

// CancelUser 取消单个用户的角色授权
// @author: [bluefrog](https://github.com/freewu)
// @function: CancelUser
// @description: 取消单个用户的角色授权
// @param: req *domain.RoleUserCancelRequest
// @return: err error
func (s Role) CancelUser(req *domain.RoleUserCancelRequest) (err error) {
	return domain.DB.Where("role_id = ? AND user_id = ?", req.RoleID, req.UserID).Unscoped().Delete(&domain.UserRole{}).Error
}

// CancelAllUsers 批量取消用户的角色授权
// @author: [bluefrog](https://github.com/freewu)
// @function: CancelAllUsers
// @description: 批量取消用户的角色授权
// @param: req *domain.RoleUserCancelAllRequest
// @return: err error
func (s Role) CancelAllUsers(req *domain.RoleUserCancelAllRequest) (err error) {
	if len(req.UserIDs) == 0 {
		return nil
	}
	return domain.DB.Where("role_id = ? AND user_id IN (?)", req.RoleID, req.UserIDs).Unscoped().Delete(&domain.UserRole{}).Error
}

// CheckNameUnique 校验角色名称是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckNameUnique
// @description: 校验角色名称是否唯一
// @param: req *domain.RoleCheckNameRequest
// @return: err error, unique bool
func (s Role) CheckNameUnique(req *domain.RoleCheckNameRequest) (err error, unique bool) {
	db := domain.DB.Model(&domain.Role{}).Where("role_name = ?", req.Name)
	if req.ID != nil {
		db = db.Where("role_id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}

// CheckKeyUnique 校验角色权限字符串是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckKeyUnique
// @description: 校验角色权限字符串是否唯一
// @param: req *domain.RoleCheckKeyRequest
// @return: err error, unique bool
func (s Role) CheckKeyUnique(req *domain.RoleCheckKeyRequest) (err error, unique bool) {
	db := domain.DB.Model(&domain.Role{}).Where("role_key = ?", req.Key)
	if req.ID != nil {
		db = db.Where("role_id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}

// IsAdmin 判断是否为超级管理员角色
// @author: [bluefrog](https://github.com/freewu)
// @function: IsAdmin
// @description: ID 为 1 的角色为超级管理员
// @param: id uint
// @return: bool
func (s Role) IsAdmin(id uint) bool {
	return id == 1
}

// CheckRoleAllowed 校验角色是否允许操作
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckRoleAllowed
// @description: 超级管理员角色不允许操作
// @param: id uint
// @return: bool
func (s Role) CheckRoleAllowed(id uint) bool {
	return !s.IsAdmin(id)
}
