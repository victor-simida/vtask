package db

import (
	"context"

	"github.com/jinzhu/gorm"

	"vtask/schema"
)

/* 对应数据库表定义如下:
CREATE TABLE `task` (
`id` int(11) unsigned NOT NULL AUTO_INCREMENT,
`uuid` varchar(64) NOT NULL DEFAULT '',
`name` varchar(64) NOT NULL DEFAULT '',
`step` int(11) NOT NULL,
`status` int(11) NOT NULL DEFAULT '0',
`responses` text NOT NULL,
`input` text NOT NULL,
`host` varchar(64) NOT NULL DEFAULT '',
`create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
`update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uidx_uuid` (`uuid`),
KEY `idx_host_status` (`host`,`status`)
*/
var db *gorm.DB

// SetDataBase ...
func SetDataBase(input *gorm.DB) {
	db = input
}

// JobRecord 异步任务实体信息
type JobRecord struct {
	ID        int64            `json:"id,omitempty" gorm:"column:id"`
	UUid      string           `json:"uuid,omitempty" gorm:"column:uuid"`
	Name      string           `json:"name,omitempty" gorm:"column:name"`
	Step      int              `json:"step,omitempty" gorm:"column:step"`
	Status    schema.JobStatus `json:"status,omitempty" gorm:"column:status"`
	Responses string           `json:"responses,omitempty" gorm:"column:responses"`
	Input     string           `json:"input,omitempty" gorm:"column:input"`
	Host      string           `json:"host,omitempty" gorm:"column:host"`
}

// TableName ...
func (p *JobRecord) TableName() string {
	return "task"
}

// GetEmployeeByUUID ...
func GetEmployeeByUUID(uuid string) (*JobRecord, error) {
	var j JobRecord
	r := db.First(&j, "uuid = ?", uuid)
	return &j, r.Error
}

// GetEmployeeByID ...
func GetEmployeeByID(id int64) (*JobRecord, error) {
	var j JobRecord
	r := db.First(&j, "id = ?", id)
	return &j, r.Error
}

// GetEmployeesByIDs ...
func GetEmployeesByIDs(ids []int64) ([]JobRecord, error) {
	var j []JobRecord
	r := db.Find(&j, "id in (?)", ids)
	return j, r.Error
}

// UpdateEmployeeStep ...
func UpdateEmployeeStep(uuid, host string, step int) error {
	r := db.Exec("update task set step=?,host = ?,status = ? "+
		"where uuid = ? and status != ?", step, host, schema.StatusRunning,
		uuid, schema.StatusRunning)

	if r.Error != nil {
		return r.Error
	}
	return nil
}

// FindWorkingEmployee ...
func FindWorkingEmployee(host, startTime, endTime string) ([]JobRecord, error) {
	var list []JobRecord

	r := db.Find(&list,
		"host= ? and status = ? and create_time >= ? and create_time< ?",
		host, schema.StatusRunning, startTime, endTime)

	return list, r.Error
}

// CreateJobs ...
func CreateJobs(ctx context.Context, j *JobRecord, host string) (int64, error) {
	ret, err := db.DB().Exec(
		"insert into task (uuid, name, step, "+
			"status, responses, input, host) values (?,?,?,?,?,?,?) ", j.UUid,
		j.Name, j.Step, j.Status, j.Responses, j.Input, host)
	if err != nil {
		return 0, err
	}

	return ret.LastInsertId()
}

// UpsertJobs ...
func UpsertJobs(ctx context.Context, j *JobRecord, host string) (int64, error) {
	ret, err := db.DB().Exec(
		"insert into task (uuid, name, step, status, "+
			"responses, input, host) values (?,?,?,?,?,?,?) on duplicate key update step = ?, status = ?, responses = ?, "+
			"input = ? ", j.UUid, j.Name, j.Step, j.Status, j.Responses,
		j.Input, host, j.Step, j.Status, j.Responses, j.Input)
	if err != nil {
		return 0, err
	}

	return ret.LastInsertId()
}
