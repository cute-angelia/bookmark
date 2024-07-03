package db

import (
	"gorm.io/gorm"
	"math"
)

// Paginate 分页函数,传入db为主DB实例,page为页码(从1开始),pageSize为每页大小
// 返回[]T为分页后的数据,int64为总记录数,error为错误信息
func Paginate[T any](db *gorm.DB, page, pageSize int) ([]T, int64, error) {
	var count int64
	var offset int
	var err error
	var results []T

	// 获取总记录数
	err = db.Model(new(T)).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(count) / float64(pageSize)))

	// 处理边界情况
	if page > totalPages {
		page = totalPages
	}
	if page <= 0 {
		page = 1
	}

	// 计算偏移量
	offset = (page - 1) * pageSize

	// 构建分页查询并执行
	err = db.Offset(offset).Limit(pageSize).Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	return results, count, nil
}
