package service

import (
	"errors"
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
	"strings"
	"time"
)

type DictType struct{}

// GetCount 获取字典类型列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取字典类型列表数据数量
//@param: searchParams *domain.DictTypeSearchRequest
//@return: err error, total int64
func (s DictType) GetCount(searchParams *domain.DictTypeSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.DictType{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取字典类型列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取字典类型列表数据
//@param: searchParams *domain.DictTypeSearchRequest
//@return: err error, list interface{}, total int64
func (s DictType) GetList(searchParams *domain.DictTypeSearchRequest) (err error, list []domain.DictType, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.DictType{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("dict_id desc")
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
//@description: 字典类型列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.DictTypeSearchRequest
//@return: *gorm.DB
func (s DictType) parseFilter(db *gorm.DB, searchParams *domain.DictTypeSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 字典主键
		db = db.Where("dict_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 字典主键 数组
		db = db.Where("dict_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("dict_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Name != "" { // 字典名称
		db = db.Where("dict_name LIKE ?", "%"+searchParams.Name+"%")
	}
	if searchParams.Type != "" { // 字典类型
		db = db.Where("dict_type = ?", searchParams.Type)
	}
	if searchParams.Status != nil { // 状态（0正常 1停用）
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
		db = db.Where("(`dict_name` LIKE ? OR `dict_type` LIKE ? OR `dict_id` = ? )", k1, k1, k)
	}
	return db
}

// Create 添加字典类型数据
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加字典类型数据
//@param: data *domain.DictTypeCreateRequest
//@return: error
func (s DictType) Create(data *domain.DictTypeCreateRequest) error {
	// 判断字典名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的字典名称")
	}
	// 判断字典类型是否唯一
	if s.TypeIsExist(data.Type, nil) {
		return errors.New("已存在相同的字典类型")
	}
	dictType := new(domain.DictType)
	dictType.Name = data.Name
	dictType.Type = data.Type
	dictType.Status = data.Status

	dictType.CreateBy = "" // todo 通过登录服务获取
	dictType.CreateTime = time.Now()
	dictType.UpdateBy = "" // todo 通过登录服务获取
	dictType.UpdateTime = time.Now()

	return domain.DB.Create(&dictType).Error
}

// Update 修改字典类型
//@author: [bluefrog](https://github.com/freewu)
//@function: Update
//@description: 修改字典类型
//@param: data *domain.DictTypeUpdateRequest
//@return: err error
func (s DictType) Update(data *domain.DictTypeUpdateRequest) (err error) {
	// 判断名字是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的字典名称")
	}
	// 判断字典类型是否唯一
	if s.TypeIsExist(data.Type, &[]uint{data.ID}) {
		return errors.New("已存在相同的字典类型")
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["Type"] = data.Type
	record["Status"] = data.Status
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	return s.UpdateByMap(data.ID, record)
}

// UpdateByMap 更新指定ID的字典类型信息
//@author: [bluefrog](https://github.com/freewu)
//@function: UpdateByMap
//@description: 更新指定ID的字典类型信息
//@param: id uint
//@param: updateMap map[string]interface{}
//@return: err error
func (s DictType) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := domain.DB.Model(&domain.DictType{}).Where("dict_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除字典类型
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除字典类型
//@param: ids []uint
//@return: err error
func (s DictType) Delete(ids []uint) (err error) {
	if err = domain.DB.Where("dict_id in (?) ", ids).Unscoped().Delete(&domain.DictType{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID字典类型详情
//@author: [bluefrog](https://github.com/freewu)
//@function: Detail
//@description: 获取指定ID字典类型详情
//@param: id uint
//@return: *domain.DictType,err error
func (s DictType) Detail(id uint) (*domain.DictType, error) {
	var detail domain.DictType
	db := domain.DB.Where("dict_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断字典名称是否存在
//@author: [bluefrog](https://github.com/freewu)
//@function: NameIsExist
//@description: 判断字典名称是否存在
//@param: name string 字典名称
//@param: excludeIds *[]uint 不包含的字典类型ID
//@return: err error
func (s DictType) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.DictTypeSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// TypeIsExist 判断字典类型是否存在
//@author: [bluefrog](https://github.com/freewu)
//@function: TypeIsExist
//@description: 判断字典类型是否存在
//@param: dictType string 字典类型
//@param: excludeIds *[]uint 不包含的字典类型ID
//@return: err error
func (s DictType) TypeIsExist(dictType string, excludeIds *[]uint) bool {
	// 判断字典类型是否唯一
	var filter = &domain.DictTypeSearchRequest{
		Type: dictType,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}
