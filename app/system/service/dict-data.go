package service

import (
	"errors"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/cache"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/database"
	"gorm.io/gorm"
	"strings"
	"time"
)

type DictData struct{}

// 字典缓存：按字典类型缓存字典数据列表（dict_type -> []DictData），
// 字典数据/字典类型变更后失效，启动时预热，支持手动刷新缓存
const (
	// DictCacheName 字典缓存名称（缓存监控页面按此名称展示）
	DictCacheName = "sys_dict"
	// dictCacheTTL 缓存条目有效期（进程内缓存，仅作兜底清理）
	dictCacheTTL = 24 * time.Hour
)

// dictCacheStore 字典缓存实例（注册到缓存监控）
var dictCacheStore = cache.New(10 * time.Minute)

func init() {
	cache.Register(DictCacheName, dictCacheStore)
	// 字典缓存值为 []domain.DictData，注册后才能从外部缓存后端（本地文件/memcache/redis）还原
	cache.RegisterType([]domain.DictData{})
}

// DictCacheEnabled 是否启用字典缓存（关闭后每次都查库，便于排查字典不生效问题）
// @return: bool
func DictCacheEnabled() bool {
	return !config.IsSet("cache.enabled") || config.Bool("cache.enabled")
}

// GetByTypeFromCache 仅从缓存读取字典数据列表（不查库）
// @param: dictType string 字典类型
// @return: []domain.DictData, bool
func (s DictData) GetByTypeFromCache(dictType string) ([]domain.DictData, bool) {
	value, exists := dictCacheStore.Get(dictType)
	if !exists {
		return nil, false
	}
	list, ok := value.([]domain.DictData)
	return list, ok
}

// EvictCache 清理指定字典类型的缓存
// @param: dictTypes ...string 字典类型
func (s DictData) EvictCache(dictTypes ...string) {
	for _, dictType := range dictTypes {
		if dictType != "" {
			dictCacheStore.Delete(dictType)
		}
	}
}

// ClearCache 清空全部字典缓存
func (s DictData) ClearCache() {
	dictCacheStore.Clear()
}

// LoadCache 预热字典缓存（按字典类型分组写入缓存）
// @return: err error, total int
func (s DictData) LoadCache() (err error, total int) {
	if !DictCacheEnabled() {
		return nil, 0
	}
	var list []domain.DictData
	if err = database.DB.Order("dict_sort asc").Find(&list).Error; err != nil {
		return err, 0
	}
	grouped := make(map[string][]domain.DictData)
	for _, item := range list {
		grouped[item.Type] = append(grouped[item.Type], item)
	}
	dictCacheStore.Clear()
	for dictType, items := range grouped {
		dictCacheStore.Set(dictType, items, dictCacheTTL)
	}
	return nil, len(grouped)
}

// RefreshCache 刷新字典缓存（清空后重新加载）
// @return: err error, total int
func (s DictData) RefreshCache() (err error, total int) {
	dictCacheStore.Clear()
	return s.LoadCache()
}

// GetCount 获取字典数据列表数据数量
// @author: [bluefrog](https://github.com/freewu)
// @function: GetCount
// @description: 获取字典数据列表数据数量
// @param: searchParams *domain.DictDataSearchRequest
// @return: err error, total int64
func (s DictData) GetCount(searchParams *domain.DictDataSearchRequest) (err error, total int64) {
	// 创建db
	db := database.DB.Model(&domain.DictData{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取字典数据列表数据
// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取字典数据列表数据
// @param: searchParams *domain.DictDataSearchRequest
// @return: err error, list interface{}, total int64
func (s DictData) GetList(searchParams *domain.DictDataSearchRequest) (err error, list []domain.DictData, total int64) {
	// 创建db
	db := database.DB.Model(&domain.DictData{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("dict_code desc")
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
// @description: 字典数据列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.DictDataSearchRequest
// @return: *gorm.DB
func (s DictData) parseFilter(db *gorm.DB, searchParams *domain.DictDataSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 字典编码
		db = db.Where("dict_code = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 字典编码 数组
		db = db.Where("dict_code IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("dict_code NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Type != "" { // 字典类型
		db = db.Where("dict_type = ?", searchParams.Type)
	}
	if searchParams.Label != "" { // 字典标签
		db = db.Where("dict_label LIKE ?", "%"+searchParams.Label+"%")
	}
	if searchParams.Value != "" { // 字典键值
		db = db.Where("dict_value = ?", searchParams.Value)
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
		db = db.Where("(`dict_label` LIKE ? OR `dict_value` LIKE ? OR `dict_type` LIKE ? OR `dict_code` = ? )", k1, k1, k1, k)
	}
	return db
}

// Create 添加字典数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加字典数据
// @param: data *domain.DictDataCreateRequest
// @return: error
func (s DictData) Create(data *domain.DictDataCreateRequest) error {
	// 判断字典标签是否唯一
	if s.LabelIsExist(data.Type, data.Label, nil) {
		return errors.New("已存在相同的字典标签")
	}
	dictData := new(domain.DictData)
	dictData.Sort = data.Sort
	dictData.Label = data.Label
	dictData.Value = data.Value
	dictData.Type = data.Type
	dictData.CssClass = data.CssClass
	dictData.ListClass = data.ListClass
	dictData.IsDefault = data.IsDefault
	dictData.Status = data.Status

	dictData.Remark = data.Remark
	dictData.CreateBy = "" // todo 通过登录服务获取
	dictData.CreateTime = time.Now()
	dictData.UpdateBy = "" // todo 通过登录服务获取
	dictData.UpdateTime = time.Now()

	if err := database.DB.Create(&dictData).Error; err != nil {
		return err
	}
	s.EvictCache(dictData.Type)
	return nil
}

// Update 修改字典数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Update
// @description: 修改字典数据
// @param: data *domain.DictDataUpdateRequest
// @return: err error
func (s DictData) Update(data *domain.DictDataUpdateRequest) (err error) {
	// 判断字典标签是否唯一
	if s.LabelIsExist(data.Type, data.Label, &[]uint{data.ID}) {
		return errors.New("已存在相同的字典标签")
	}
	// 修改前的字典类型（类型变更时需同时失效旧类型）
	oldType := ""
	if detail, detailErr := s.Detail(data.ID); detailErr == nil && detail != nil {
		oldType = detail.Type
	}
	record := make(map[string]interface{})
	record["Sort"] = data.Sort
	record["Label"] = data.Label
	record["Value"] = data.Value
	record["Type"] = data.Type
	record["CssClass"] = data.CssClass
	record["ListClass"] = data.ListClass
	record["IsDefault"] = data.IsDefault
	record["Status"] = data.Status
	record["Remark"] = data.Remark
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	if err = s.UpdateByMap(data.ID, record); err != nil {
		return err
	}
	s.EvictCache(oldType, data.Type)
	return nil
}

// UpdateByMap 更新指定ID的字典数据信息
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateByMap
// @description: 更新指定ID的字典数据信息
// @param: id uint
// @param: updateMap map[string]interface{}
// @return: err error
func (s DictData) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := database.DB.Model(&domain.DictData{}).Where("dict_code = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除字典数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 删除字典数据
// @param: ids []uint
// @return: err error
func (s DictData) Delete(ids []uint) (err error) {
	// 删除前取出字典类型，用于失效缓存
	var list []domain.DictData
	if err = database.DB.Where("dict_code in (?)", ids).Find(&list).Error; err != nil {
		return err
	}
	if err = database.DB.Where("dict_code in (?) ", ids).Unscoped().Delete(&domain.DictData{}).Error; err != nil {
		return err
	}
	for _, item := range list {
		s.EvictCache(item.Type)
	}
	return nil
}

// Detail 获取指定ID字典数据详情
// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID字典数据详情
// @param: id uint
// @return: *domain.DictData,err error
func (s DictData) Detail(id uint) (*domain.DictData, error) {
	var detail domain.DictData
	db := database.DB.Where("dict_code = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// LabelIsExist 判断字典标签是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: LabelIsExist
// @description: 判断指定字典类型下字典标签是否存在
// @param: dictType string 字典类型
// @param: label string 字典标签
// @param: excludeIds *[]uint 不包含的字典数据ID
// @return: err error
func (s DictData) LabelIsExist(dictType string, label string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.DictDataSearchRequest{
		Type:  dictType,
		Label: label,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// GetByType 根据字典类型查询字典数据列表
// @author: [bluefrog](https://github.com/freewu)
// @function: GetByType
// @description: 根据字典类型查询字典数据列表（前端下拉渲染使用）
// @param: req *domain.DictDataByTypeRequest
// @return: err error, list []domain.DictData
func (s DictData) GetByType(req *domain.DictDataByTypeRequest) (err error, list []domain.DictData) {
	// 缓存优先
	if DictCacheEnabled() {
		if cached, exists := s.GetByTypeFromCache(req.Type); exists {
			return nil, cached
		}
	}
	if err = database.DB.Where("dict_type = ?", req.Type).Order("dict_sort asc").Find(&list).Error; err != nil {
		return err, nil
	}
	if DictCacheEnabled() {
		dictCacheStore.Set(req.Type, list, dictCacheTTL)
	}
	return nil, list
}

// CheckUniqueLabel 校验同一字典类型下字典标签是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckUniqueLabel
// @description: 校验同一字典类型下字典标签是否唯一
// @param: dictType string 字典类型
// @param: label string 字典标签
// @param: excludeIds *[]uint 排除的字典编码
// @return: err error, unique bool
func (s DictData) CheckUniqueLabel(dictType string, label string, excludeIds *[]uint) (err error, unique bool) {
	db := database.DB.Model(&domain.DictData{}).
		Where("dict_type = ?", dictType).
		Where("dict_label = ?", label)
	if excludeIds != nil && len(*excludeIds) > 0 {
		db = db.Where("dict_code NOT IN (?)", *excludeIds)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}
