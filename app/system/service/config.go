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

type Config struct{}

// 参数缓存：与若依一致，sys_config 全量缓存在进程内（config_key -> config_value），
// 新增/修改/删除后失效，启动时预热，支持手动刷新缓存
const (
	// ConfigCacheName 参数缓存名称（缓存监控页面按此名称展示）
	ConfigCacheName = "sys_config"
	// configCacheTTL 缓存条目有效期（进程内缓存，仅作兜底清理）
	configCacheTTL = 24 * time.Hour
)

// configCacheStore 参数缓存实例（注册到缓存监控）
var configCacheStore = cache.New(10 * time.Minute)

func init() {
	cache.Register(ConfigCacheName, configCacheStore)
}

// ConfigCacheEnabled 是否启用参数缓存（关闭后每次都查库，便于排查配置不生效问题）
// @return: bool
func ConfigCacheEnabled() bool {
	return !config.IsSet("system.cache.enabled") || config.Bool("system.cache.enabled")
}

// GetByKeyFromCache 仅从缓存读取参数键值（不查库）
// @param: key string 参数键名
// @return: string, bool
func (s Config) GetByKeyFromCache(key string) (string, bool) {
	value, exists := configCacheStore.Get(key)
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// EvictCache 清理指定参数键名的缓存
// @param: keys ...string 参数键名
func (s Config) EvictCache(keys ...string) {
	for _, key := range keys {
		if key != "" {
			configCacheStore.Delete(key)
		}
	}
}

// LoadCache 预热参数缓存（sys_config 全表写入缓存）
// @return: err error, total int
func (s Config) LoadCache() (err error, total int) {
	if !ConfigCacheEnabled() {
		return nil, 0
	}
	var list []domain.Config
	if err = database.DB.Find(&list).Error; err != nil {
		return err, 0
	}
	configCacheStore.Clear()
	for _, item := range list {
		configCacheStore.Set(item.Key, item.Value, configCacheTTL)
	}
	return nil, len(list)
}

// RefreshCache 刷新参数缓存（清空后重新加载）
// @return: err error, total int
func (s Config) RefreshCache() (err error, total int) {
	configCacheStore.Clear()
	return s.LoadCache()
}

// GetCount 获取参数列表数据数量
// @author: [bluefrog](https://github.com/freewu)
// @function: GetCount
// @description: 获取参数列表数据数量
// @param: searchParams *domain.ConfigSearchRequest
// @return: err error, total int64
func (s Config) GetCount(searchParams *domain.ConfigSearchRequest) (err error, total int64) {
	// 创建db
	db := database.DB.Model(&domain.Config{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取参数列表数据
// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取参数列表数据
// @param: searchParams *domain.ConfigSearchRequest
// @return: err error, list interface{}, total int64
func (s Config) GetList(searchParams *domain.ConfigSearchRequest) (err error, list []domain.Config, total int64) {
	// 创建db
	db := database.DB.Model(&domain.Config{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		// orderBy := "id desc"
		db.Order("config_id desc")
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
// @description: 参数列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.ConfigSearchRequest
// @return: *gorm.DB
func (s Config) parseFilter(db *gorm.DB, searchParams *domain.ConfigSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 参数编号
		db = db.Where("config_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 参数编号 数组
		db = db.Where("config_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("config_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Name != "" { // 参数名称
		db = db.Where("config_name = ?", searchParams.Name)
	}
	if searchParams.Key != "" { // 参数键名
		db = db.Where("config_key = ?", searchParams.Key)
	}
	if searchParams.Value != "" { // 参数键值
		db = db.Where("config_value LIKE ?", "%"+searchParams.Value+"%")
	}
	if searchParams.Type != "" { // 是否系统内置（Y是 | N否）
		db = db.Where("config_type = ?", searchParams.Type)
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
		db = db.Where("(`config_name` LIKE ? OR "+
			"`config_key` LIKE ? OR "+
			"`config_value` LIKE ? OR "+
			"`config_id` = ? )", k1, k1, k1, k)
	}
	return db
}

// Create 添加参数数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加参数数据
// @param: data *domain.ConfigAddRequest
// @return: error
func (s Config) Create(data *domain.ConfigCreateRequest) error {
	// 判断参数名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的参数名称")
	}
	// 判断参数键名是否唯一
	if s.KeyIsExist(data.Key, nil) {
		return errors.New("已存在相同的参数键名")
	}
	config := new(domain.Config)
	config.Key = data.Key
	config.Name = data.Name
	config.Type = data.Type
	config.Value = data.Value

	config.CreateBy = "" // todo 通过登录服务获取
	config.CreateTime = time.Now()
	config.UpdateBy = "" // todo 通过登录服务获取
	config.UpdateTime = time.Now()
	config.Remark = data.Remark

	if err := database.DB.Create(&config).Error; err != nil {
		return err
	}
	// 写库后失效缓存（下次读取时回源）
	s.EvictCache(config.Key)
	return nil
}

// Update 修改参数
// @author: [bluefrog](https://github.com/freewu)
// @function: Update
// @description: 修改参数
// @param: data *model.ConfigEditRequest
// @return: err error
func (s Config) Update(data *domain.ConfigUpdateRequest) (err error) {
	// 判断名字是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的参数名称")
	}
	// 判断参数键名是否唯一
	if s.KeyIsExist(data.Key, &[]uint{data.ID}) {
		return errors.New("已存在相同的参数键名")
	}
	// 修改前的键名（键名变更时需同时失效旧键）
	oldKey := ""
	if detail, detailErr := s.Detail(data.ID); detailErr == nil && detail != nil {
		oldKey = detail.Key
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["Key"] = data.Key
	record["Type"] = data.Type
	record["Value"] = data.Value
	record["Remark"] = data.Remark
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	if err = s.UpdateByMap(data.ID, record); err != nil {
		return err
	}
	s.EvictCache(oldKey, data.Key)
	return nil
}

// UpdateByMap 更新指定ID的参数信息
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateByMap
// @description: 更新指定ID的参数信息
// @param: id uint
// @param: updateMap map[string]interface{}
// @return: err error
func (s Config) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := database.DB.Model(&domain.Config{}).Where("config_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除参数
// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 删除参数
// @param: ids []uint
// @return: err error
func (s Config) Delete(ids []uint) (err error) {
	// 删除前取出键名，用于失效缓存
	keys := make([]string, 0, len(ids))
	var list []domain.Config
	if err = database.DB.Where("config_id in (?)", ids).Find(&list).Error; err != nil {
		return err
	}
	for _, item := range list {
		keys = append(keys, item.Key)
	}
	if err = database.DB.Where("config_id in (?) ", ids).Unscoped().Delete(&domain.Config{}).Error; err != nil {
		return err
	}
	s.EvictCache(keys...)
	return nil
}

// Detail 获取指定ID参数详情
// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID参数详情
// @param: id uint
// @return: *domain.Config,err error
func (s Config) Detail(id uint) (*domain.Config, error) {
	var detail domain.Config
	db := database.DB.Where("config_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断参数名称是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: NameIsExist
// @description: 判断参数名称是否存在
// @param: name string 参数名称
// @param: excludeIds *[]uint 不包含的参数ID
// @return: err error
func (s Config) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.ConfigSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// KeyIsExist 判断参数键名是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: KeyIsExist
// @description: 判断参数键名是否存在
// @param: key string 参数键名
// @param: excludeIds *[]uint 不包含的区域类型ID
// @return: err error
func (s Config) KeyIsExist(key string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.ConfigSearchRequest{
		Key: key,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// GetByKey 根据参数键名获取参数键值
// @author: [bluefrog](https://github.com/freewu)
// @function: GetByKey
// @description: 根据参数键名获取参数键值
// @param: key string 参数键名
// @return: err error, value string
func (s Config) GetByKey(key string) (err error, value string) {
	// 缓存优先（与若依 selectConfigByKey 一致）
	if ConfigCacheEnabled() {
		if value, exists := s.GetByKeyFromCache(key); exists {
			return nil, value
		}
	}
	var config domain.Config
	if err = database.DB.Where("config_key = ?", key).First(&config).Error; err != nil {
		return err, ""
	}
	if ConfigCacheEnabled() {
		configCacheStore.Set(config.Key, config.Value, configCacheTTL)
	}
	return nil, config.Value
}

// CheckKeyUnique 校验参数键名是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckKeyUnique
// @description: 校验参数键名是否唯一
// @param: req *domain.ConfigCheckKeyRequest
// @return: err error, unique bool
func (s Config) CheckKeyUnique(req *domain.ConfigCheckKeyRequest) (err error, unique bool) {
	if req.Key == "" {
		return nil, true
	}
	db := database.DB.Model(&domain.Config{}).Where("config_key = ?", req.Key)
	if req.ID != nil {
		db = db.Where("config_id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}
