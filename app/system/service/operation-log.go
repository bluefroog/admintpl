package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
	"strings"
	"time"
)

type OperationLog struct{}

// GetCount 获取操作日志列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取操作日志列表数据数量
//@param: searchParams *domain.OperationLogSearchRequest
//@return: err error, total int64
func (s OperationLog) GetCount(searchParams *domain.OperationLogSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.OperationLog{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取操作日志列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取操作日志列表数据
//@param: searchParams *domain.OperationLogSearchRequest
//@return: err error, list interface{}, total int64
func (s OperationLog) GetList(searchParams *domain.OperationLogSearchRequest) (err error, list []domain.OperationLog, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.OperationLog{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	if err != nil {
		return err, nil, 0
	}
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("oper_id desc")
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
//@description: 操作日志列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.OperationLogSearchRequest
//@return: *gorm.DB
func (s OperationLog) parseFilter(db *gorm.DB, searchParams *domain.OperationLogSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 日志主键
		db = db.Where("oper_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 日志主键 数组
		db = db.Where("oper_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("oper_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Title != "" { // 模块标题
		db = db.Where("title LIKE ?", "%"+searchParams.Title+"%")
	}
	if searchParams.BusinessType != nil { // 业务类型（0其它 1新增 2修改 3删除）
		db = db.Where("business_type = ?", searchParams.BusinessType)
	}
	if searchParams.RequestMethod != "" { // 请求方式
		db = db.Where("request_method = ?", searchParams.RequestMethod)
	}
	if searchParams.OperName != "" { // 操作人员
		db = db.Where("oper_name LIKE ?", "%"+searchParams.OperName+"%")
	}
	if searchParams.DeptName != "" { // 部门名称
		db = db.Where("dept_name LIKE ?", "%"+searchParams.DeptName+"%")
	}
	if searchParams.OperURL != "" { // 请求URL
		db = db.Where("oper_url LIKE ?", "%"+searchParams.OperURL+"%")
	}
	if searchParams.OperIP != "" { // 主机地址
		db = db.Where("oper_ip = ?", searchParams.OperIP)
	}
	if searchParams.Status != nil { // 操作状态（0正常 1异常）
		db = db.Where("status = ?", searchParams.Status)
	}
	if searchParams.Keyword != "" { // 关键词
		k := strings.Trim(searchParams.Keyword, " \t\r\n")
		k1 := "%" + k + "%"
		db = db.Where("(`title` LIKE ? OR `method` LIKE ? OR `oper_name` LIKE ? OR `dept_name` LIKE ? OR `oper_url` LIKE ? OR `oper_ip` LIKE ? OR `oper_id` = ? )", k1, k1, k1, k1, k1, k1, k)
	}
	return db
}

// Create 添加操作日志数据
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加操作日志数据
//@param: data *domain.OperationLogCreateRequest
//@return: error
func (s OperationLog) Create(data *domain.OperationLogCreateRequest) error {
	operationLog := new(domain.OperationLog)
	operationLog.Title = data.Title
	operationLog.BusinessType = data.BusinessType
	operationLog.Method = data.Method
	operationLog.RequestMethod = data.RequestMethod
	operationLog.OperatorType = data.OperatorType
	operationLog.OperName = data.OperName
	operationLog.DeptName = data.DeptName
	operationLog.OperURL = data.OperURL
	operationLog.OperIP = data.OperIP
	operationLog.OperLocation = data.OperLocation
	operationLog.OperParam = data.OperParam
	operationLog.JSONResult = data.JSONResult
	operationLog.Status = data.Status
	operationLog.ErrorMsg = data.ErrorMsg
	operationLog.OperTime = time.Now()
	operationLog.CostTime = data.CostTime

	return domain.DB.Create(&operationLog).Error
}

// Delete 删除操作日志
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除操作日志
//@param: ids []uint
//@return: err error
func (s OperationLog) Delete(ids []uint) (err error) {
	if err = domain.DB.Where("oper_id in (?) ", ids).Unscoped().Delete(&domain.OperationLog{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID操作日志详情
//@author: [bluefrog](https://github.com/freewu)
//@function: Detail
//@description: 获取指定ID操作日志详情
//@param: id uint
//@return: *domain.OperationLog,err error
func (s OperationLog) Detail(id uint) (*domain.OperationLog, error) {
	var detail domain.OperationLog
	db := domain.DB.Where("oper_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}
