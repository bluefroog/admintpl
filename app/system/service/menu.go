package service

import (
	"errors"
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/model/request"
	"strings"
	"time"
)

type Menu struct{}

// GetCount 获取菜单列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取菜单列表数据数量
//@param: searchParams *domain.MenuSearchRequest
//@return: err error, total int64
func (s Menu) GetCount(searchParams *domain.MenuSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Menu{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取菜单列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取菜单列表数据
//@param: searchParams *domain.MenuSearchRequest
//@return: err error, list interface{}, total int64
func (s Menu) GetList(searchParams *domain.MenuSearchRequest) (err error, list []domain.Menu, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Menu{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("menu_id asc")
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
//@description: 菜单列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.MenuSearchRequest
//@return: *gorm.DB
func (s Menu) parseFilter(db *gorm.DB, searchParams *domain.MenuSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 菜单ID
		db = db.Where("menu_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 菜单ID 数组
		db = db.Where("menu_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("menu_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.ParentID != nil { // 父菜单ID
		db = db.Where("parent_id = ?", searchParams.ParentID)
	}
	if len(searchParams.ParentIDIn) > 0 { // 父菜单ID 数组
		db = db.Where("parent_id IN (?)", searchParams.ParentIDIn)
	}
	if searchParams.Name != "" { // 菜单名称
		db = db.Where("menu_name LIKE ?", "%"+searchParams.Name+"%")
	}
	if searchParams.Path != "" { // 路由地址
		db = db.Where("path LIKE ?", "%"+searchParams.Path+"%")
	}
	if searchParams.Perms != "" { // 权限标识
		db = db.Where("perms LIKE ?", "%"+searchParams.Perms+"%")
	}
	if searchParams.MenuType != "" { // 菜单类型（M目录 C菜单 F按钮）
		db = db.Where("menu_type = ?", searchParams.MenuType)
	}
	if searchParams.Status != nil { // 菜单状态（0正常 1停用）
		db = db.Where("status = ?", searchParams.Status)
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
		db = db.Where("(`menu_name` LIKE ? OR `path` LIKE ? OR `perms` LIKE ? OR `menu_id` = ? )", k1, k1, k1, k)
	}
	return db
}

// Create 添加菜单数据
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加菜单数据
//@param: data *domain.MenuCreateRequest
//@return: error
func (s Menu) Create(data *domain.MenuCreateRequest) error {
	// 判断菜单名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的菜单名称")
	}
	menu := new(domain.Menu)
	menu.Name = data.Name
	menu.ParentID = data.ParentID
	menu.Sort = data.Sort
	menu.Path = data.Path
	menu.Component = data.Component
	menu.Query = data.Query
	menu.RouteName = data.RouteName
	menu.IsFrame = data.IsFrame
	menu.IsCache = data.IsCache
	menu.MenuType = data.MenuType
	menu.Visible = data.Visible
	menu.Status = data.Status
	menu.Perms = data.Perms
	menu.Icon = data.Icon

	menu.CreateBy = "" // todo 通过登录服务获取
	menu.CreateTime = time.Now()
	menu.UpdateBy = "" // todo 通过登录服务获取
	menu.UpdateTime = time.Now()

	return domain.DB.Create(&menu).Error
}

// Update 修改菜单
//@author: [bluefrog](https://github.com/freewu)
//@function: Update
//@description: 修改菜单
//@param: data *domain.MenuUpdateRequest
//@return: err error
func (s Menu) Update(data *domain.MenuUpdateRequest) (err error) {
	// 判断菜单名称是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的菜单名称")
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["ParentID"] = data.ParentID
	record["Sort"] = data.Sort
	record["Path"] = data.Path
	record["Component"] = data.Component
	record["Query"] = data.Query
	record["RouteName"] = data.RouteName
	record["IsFrame"] = data.IsFrame
	record["IsCache"] = data.IsCache
	record["MenuType"] = data.MenuType
	record["Visible"] = data.Visible
	record["Status"] = data.Status
	record["Perms"] = data.Perms
	record["Icon"] = data.Icon
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	return s.UpdateByMap(data.ID, record)
}

// UpdateByMap 更新指定ID的菜单信息
//@author: [bluefrog](https://github.com/freewu)
//@function: UpdateByMap
//@description: 更新指定ID的菜单信息
//@param: id uint
//@param: updateMap map[string]interface{}
//@return: err error
func (s Menu) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := domain.DB.Model(&domain.Menu{}).Where("menu_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除菜单
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除菜单
//@param: ids []uint
//@return: err error
func (s Menu) Delete(ids []uint) (err error) {
	// 如果有子菜单存在，禁止删除,需要先删除完子菜单
	_, _, total := s.GetList(&domain.MenuSearchRequest{
		ParentIDIn: ids,
		PageInfo:   request.PageInfo{PageSize: 1},
	})
	if total > 0 {
		return errors.New("存在子菜单，禁止删除")
	}
	if err = domain.DB.Where("menu_id in (?) ", ids).Unscoped().Delete(&domain.Menu{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID菜单详情
//@author: [bluefrog](https://github.com/freewu)
//@function: Detail
//@description: 获取指定ID菜单详情
//@param: id uint
//@return: *domain.Menu,err error
func (s Menu) Detail(id uint) (*domain.Menu, error) {
	var detail domain.Menu
	db := domain.DB.Where("menu_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断菜单名称是否存在
//@author: [bluefrog](https://github.com/freewu)
//@function: NameIsExist
//@description: 判断菜单名称是否存在
//@param: name string 菜单名称
//@param: excludeIds *[]uint 不包含的菜单ID
//@return: err error
func (s Menu) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.MenuSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}
