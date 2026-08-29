package service

import (
	"errors"
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
	"strings"
	"time"
)

type Job struct{}

// GetCount 获取定时任务列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取定时任务列表数据数量
//@param: searchParams *domain.JobSearchRequest
//@return: err error, total int64
func (s Job) GetCount(searchParams *domain.JobSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Job{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取定时任务列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取定时任务列表数据
//@param: searchParams *domain.JobSearchRequest
//@return: err error, list interface{}, total int64
func (s Job) GetList(searchParams *domain.JobSearchRequest) (err error, list []domain.Job, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Job{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("job_id desc")
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
//@description: 定时任务列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.JobSearchRequest
//@return: *gorm.DB
func (s Job) parseFilter(db *gorm.DB, searchParams *domain.JobSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 任务ID
		db = db.Where("job_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 任务ID 数组
		db = db.Where("job_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("job_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Name != "" { // 任务名称
		db = db.Where("job_name LIKE ?", "%"+searchParams.Name+"%")
	}
	if searchParams.Group != "" { // 任务组名
		db = db.Where("job_group = ?", searchParams.Group)
	}
	if searchParams.Executor != "" { // 任务执行器
		db = db.Where("job_executor = ?", searchParams.Executor)
	}
	if searchParams.InvokeTarget != "" { // 调用目标字符串
		db = db.Where("invoke_target LIKE ?", "%"+searchParams.InvokeTarget+"%")
	}
	if searchParams.Status != nil { // 状态（0正常 1暂停）
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
		db = db.Where("(`job_name` LIKE ? OR `job_group` LIKE ? OR `invoke_target` LIKE ? OR `job_id` = ? )", k1, k1, k1, k)
	}
	return db
}

// Create 添加定时任务
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加定时任务
//@param: data *domain.JobCreateRequest
//@return: error
func (s Job) Create(data *domain.JobCreateRequest) error {
	// 判断任务名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的任务名称")
	}
	job := new(domain.Job)
	job.Name = data.Name
	job.Group = data.Group
	job.Executor = data.Executor
	job.InvokeTarget = data.InvokeTarget
	job.JobArgs = data.JobArgs
	job.JobKwargs = data.JobKwargs
	job.CronExpression = data.CronExpression
	job.MisfirePolicy = data.MisfirePolicy
	job.Concurrent = data.Concurrent
	job.Status = data.Status

	job.CreateBy = "" // todo 通过登录服务获取
	job.CreateTime = time.Now()
	job.UpdateBy = "" // todo 通过登录服务获取
	job.UpdateTime = time.Now()

	return domain.DB.Create(&job).Error
}

// Update 修改定时任务
//@author: [bluefrog](https://github.com/freewu)
//@function: Update
//@description: 修改定时任务
//@param: data *domain.JobUpdateRequest
//@return: err error
func (s Job) Update(data *domain.JobUpdateRequest) (err error) {
	// 判断任务名称是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的任务名称")
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["Group"] = data.Group
	record["Executor"] = data.Executor
	record["InvokeTarget"] = data.InvokeTarget
	record["JobArgs"] = data.JobArgs
	record["JobKwargs"] = data.JobKwargs
	record["CronExpression"] = data.CronExpression
	record["MisfirePolicy"] = data.MisfirePolicy
	record["Concurrent"] = data.Concurrent
	record["Status"] = data.Status
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	return s.UpdateByMap(data.ID, record)
}

// UpdateByMap 更新指定ID的定时任务信息
//@author: [bluefrog](https://github.com/freewu)
//@function: UpdateByMap
//@description: 更新指定ID的定时任务信息
//@param: id uint
//@param: updateMap map[string]interface{}
//@return: err error
func (s Job) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := domain.DB.Model(&domain.Job{}).Where("job_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除定时任务
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除定时任务
//@param: ids []uint
//@return: err error
func (s Job) Delete(ids []uint) (err error) {
	if err = domain.DB.Where("job_id in (?) ", ids).Unscoped().Delete(&domain.Job{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID定时任务详情
//@author: [bluefrog](https://github.com/freewu)
//@function: Detail
//@description: 获取指定ID定时任务详情
//@param: id uint
//@return: *domain.Job,err error
func (s Job) Detail(id uint) (*domain.Job, error) {
	var detail domain.Job
	db := domain.DB.Where("job_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断任务名称是否存在
//@author: [bluefrog](https://github.com/freewu)
//@function: NameIsExist
//@description: 判断任务名称是否存在
//@param: name string 任务名称
//@param: excludeIds *[]uint 不包含的任务ID
//@return: err error
func (s Job) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.JobSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}
