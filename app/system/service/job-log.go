package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
	"strings"
	"time"
)

type JobLog struct{}

// GetCount 获取定时任务日志列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取定时任务日志列表数据数量
//@param: searchParams *domain.JobLogSearchRequest
//@return: err error, total int64
func (s JobLog) GetCount(searchParams *domain.JobLogSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.JobLog{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取定时任务日志列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取定时任务日志列表数据
//@param: searchParams *domain.JobLogSearchRequest
//@return: err error, list interface{}, total int64
func (s JobLog) GetList(searchParams *domain.JobLogSearchRequest) (err error, list []domain.JobLog, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.JobLog{})
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
		db.Order("job_log_id desc")
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
//@description: 定时任务日志列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.JobLogSearchRequest
//@return: *gorm.DB
func (s JobLog) parseFilter(db *gorm.DB, searchParams *domain.JobLogSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 任务日志ID
		db = db.Where("job_log_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 任务日志ID 数组
		db = db.Where("job_log_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("job_log_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.JobName != "" { // 任务名称
		db = db.Where("job_name LIKE ?", "%"+searchParams.JobName+"%")
	}
	if searchParams.JobGroup != "" { // 任务组名
		db = db.Where("job_group = ?", searchParams.JobGroup)
	}
	if searchParams.JobExecutor != "" { // 任务执行器
		db = db.Where("job_executor = ?", searchParams.JobExecutor)
	}
	if searchParams.InvokeTarget != "" { // 调用目标字符串
		db = db.Where("invoke_target LIKE ?", "%"+searchParams.InvokeTarget+"%")
	}
	if searchParams.JobTrigger != "" { // 任务触发器
		db = db.Where("job_trigger = ?", searchParams.JobTrigger)
	}
	if searchParams.Status != nil { // 执行状态（0正常 1失败）
		db = db.Where("status = ?", searchParams.Status)
	}
	if searchParams.Keyword != "" { // 关键词
		k := strings.Trim(searchParams.Keyword, " \t\r\n")
		k1 := "%" + k + "%"
		db = db.Where("(`job_name` LIKE ? OR `job_group` LIKE ? OR `job_executor` LIKE ? OR `invoke_target` LIKE ? OR `job_trigger` LIKE ? OR `job_message` LIKE ? OR `job_log_id` = ? )", k1, k1, k1, k1, k1, k1, k)
	}
	return db
}

// Create 添加定时任务日志数据
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加定时任务日志数据
//@param: data *domain.JobLogCreateRequest
//@return: error
func (s JobLog) Create(data *domain.JobLogCreateRequest) error {
	jobLog := new(domain.JobLog)
	jobLog.JobName = data.JobName
	jobLog.JobGroup = data.JobGroup
	jobLog.JobExecutor = data.JobExecutor
	jobLog.InvokeTarget = data.InvokeTarget
	jobLog.JobArgs = data.JobArgs
	jobLog.JobKwargs = data.JobKwargs
	jobLog.JobTrigger = data.JobTrigger
	jobLog.JobMessage = data.JobMessage
	jobLog.ExceptionInfo = data.ExceptionInfo
	jobLog.Status = data.Status
	jobLog.CreateTime = time.Now()

	return domain.DB.Create(&jobLog).Error
}

// Delete 删除定时任务日志
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除定时任务日志
//@param: ids []uint
//@return: err error
func (s JobLog) Delete(ids []uint) (err error) {
	if err = domain.DB.Where("job_log_id in (?) ", ids).Unscoped().Delete(&domain.JobLog{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID定时任务日志详情
//@author: [bluefrog](https://github.com/freewu)
//@function: Detail
//@description: 获取指定ID定时任务日志详情
//@param: id uint
//@return: *domain.JobLog,err error
func (s JobLog) Detail(id uint) (*domain.JobLog, error) {
	var detail domain.JobLog
	db := domain.DB.Where("job_log_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}
