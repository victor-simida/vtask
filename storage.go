package vtask

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vtask/schema"
	"vtask/storage/db"
)

// JobStorage 存储interface
type JobStorage interface {
	// GetNeedWorkEmployees 从数据库中拉取所有属于该节点的employee
	GetNeedWorkEmployees(time.Time, time.Time) ([]*Employee, error)

	// Record 记录当前employee的状态
	Record(context.Context, *Employee) (int64, error)

	// Put 插入employee
	Put(context.Context, *Employee) (int64, error)

	// Get 获取某个uuid下任务信息
	Get(context.Context, string) (*schema.SbsJobInfo, error)

	// GetByID 获取某个id下的任务信息
	GetByID(id int64) (*schema.SbsJobInfo, error)

	// GetByIDs 获取某些id对应的x任务信息
	GetByIDs(ids []int64) ([]schema.SbsJobInfo, error)

	// FetchEmployeeByKeyword 获取指定的employee
	FetchEmployeeByKeyword(keyword string) (*Employee, error)

	// UpdateRestartInfo 更新指定keyword对应任务的当前执行步骤，用于重启任务
	UpdateRestartInfo(keyword string, step int32) error
}

type storage struct {
	*JobCenter
}

// Record ...
func (store *storage) Record(ctx context.Context, e *Employee) (int64, error) {
	var j = db.JobRecord{
		UUid:      e.jobID,
		Name:      e.job.JobName,
		Step:      e.step,
		Status:    e.status,
		Responses: e.responseString(),
		Input:     e.inputString(),
	}

	return db.UpsertJobs(ctx, &j, store.host)
}

// Put ...
func (store *storage) Put(ctx context.Context, e *Employee) (int64, error) {
	var j = db.JobRecord{
		UUid:      e.jobID,
		Name:      e.job.JobName,
		Step:      e.step,
		Status:    e.status,
		Responses: e.responseString(),
		Input:     e.inputString(),
	}

	insertId, err := db.CreateJobs(ctx, &j, store.host)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			info, err := store.Get(ctx, e.jobID)
			if err != nil {
				return 0, err
			}

			return info.Id, nil

		}
	}

	return insertId, nil
}

// GetNeedWorkEmployees ...
func (store *storage) GetNeedWorkEmployees(from, to time.Time) ([]*Employee, error) {
	list, err := db.FindWorkingEmployee(store.host, from.Format("2006-01-02 15:04:05"),
		to.Format("2006-01-02 15:04:05"))
	if err != nil {
		Logger.Errorf("FindWorkingEmployee_err %s", err.Error())
		return nil, err
	}

	var resp []*Employee
	for _, v := range list {
		e, err := store.NewEmployee(v.UUid, v.Name, v.Input)
		if err != nil {
			Logger.Errorf("Init NewEmployee error %s", err.Error())
			continue
		}

		e.id = v.ID
		e.step = v.Step
		e.status = v.Status
		err = json.Unmarshal([]byte(v.Responses), &e.resp)
		if err != nil {
			Logger.Errorf("Init Unmarshal error %s", err.Error())
			continue
		}
		resp = append(resp, e)
	}
	return resp, nil
}

// Get ...
func (store *storage) Get(ctx context.Context, uuid string) (*schema.SbsJobInfo, error) {
	j, err := db.GetEmployeeByUUID(uuid)
	if err != nil {
		return nil, err
	}
	var temp schema.SbsJobInfo
	temp.Id = j.ID
	temp.JobId = j.UUid
	temp.JobName = j.Name
	temp.Step = int32(j.Step)
	temp.Status = int32(j.Status)

	if j, ok := store.m[j.Name]; ok {
		temp.TotalStep = int32(len(j.Steps))
	}
	return &temp, nil
}

// GetByID ...
func (store *storage) GetByID(id int64) (*schema.SbsJobInfo, error) {
	j, err := db.GetEmployeeByID(id)
	if err != nil {
		return nil, err
	}

	var temp schema.SbsJobInfo
	temp.Id = j.ID
	temp.JobId = j.UUid
	temp.JobName = j.Name
	temp.Step = int32(j.Step)
	temp.Status = int32(j.Status)
	temp.Info = j.Input

	// 获取总步骤数
	if j, ok := store.m[temp.JobName]; ok {
		temp.TotalStep = int32(len(j.Steps))
	}
	if temp.Step <= temp.TotalStep {
		temp.StepDesc = store.JobCenter.m[temp.JobName].Steps[temp.Step-1].StepDesc
	}

	return &temp, nil
}

// GetByIDs ...
func (store *storage) GetByIDs(ids []int64) ([]schema.SbsJobInfo, error) {
	j, err := db.GetEmployeesByIDs(ids)
	if err != nil {
		return nil, err
	}

	data := make([]schema.SbsJobInfo, len(j))
	for i := 0; i < len(j); i++ {
		var temp schema.SbsJobInfo
		temp.Id = j[i].ID
		temp.JobId = j[i].UUid
		temp.JobName = j[i].Name
		temp.Step = int32(j[i].Step)
		temp.Status = int32(j[i].Status)
		temp.Info = j[i].Input

		// 获取总步骤数
		if k, ok := store.m[temp.JobName]; ok {
			temp.TotalStep = int32(len(k.Steps))
		}
		if temp.Step <= temp.TotalStep {
			temp.StepDesc = store.JobCenter.m[temp.JobName].Steps[temp.Step-1].StepDesc
		}
		data[i] = temp
	}

	return data, nil
}

// FetchEmployeeByKeyword ...
func (store *storage) FetchEmployeeByKeyword(keyword string) (*Employee, error) {
	job, err := db.GetEmployeeByUUID(keyword)
	if err != nil {
		return nil, err
	}

	e, err := store.NewEmployee(job.UUid, job.Name, job.Input)
	if err != nil {
		Logger.Errorf("Init NewEmployee error %s", err.Error())
		return nil, err
	}

	e.id = job.ID
	e.step = job.Step
	e.status = job.Status
	err = json.Unmarshal([]byte(job.Responses), &e.resp)
	if err != nil {
		Logger.Errorf("Init Unmarshal error %s", err.Error())
		return nil, err
	}

	return e, nil
}

// UpdateRestartInfo ...
func (store *storage) UpdateRestartInfo(keyword string, step int32) error {
	err := db.UpdateEmployeeStep(keyword, store.host, int(step))
	if err != nil {
		return fmt.Errorf("update_err_%v", err)
	}
	return nil
}
