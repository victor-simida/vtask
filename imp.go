package asyncsbsjob

import (
	"context"
	"fmt"

	"vtask/schema"

	"github.com/google/uuid"
)

// InvokeAsyncJob ...
func (jc *JobCenter) InvokeAsyncJob(ctx context.Context,
	req *schema.InvokeAsyncJobReq) (*schema.InvokeAsyncJobResp, error) {

	keyword := req.Keyword
	if keyword == "" {
		keyword = uuid.New().String()
	}
	e, err := jc.NewEmployee(keyword, req.JobName, req.InputData)
	if err != nil {
		return nil, err
	}

	flowId, err := GlobalStorage.Put(ctx, e)
	if err != nil {
		return nil, err
	}

	resp := new(schema.InvokeAsyncJobResp)
	resp.JobID = e.jobID
	resp.FlowID = flowId
	return resp, nil
}

// GetAsyncJobs ...
func (jc *JobCenter) GetAsyncJobs(ctx context.Context,
	req *schema.GetAsyncJobsReq) (*schema.GetAsyncJobsResp, error) {
	if req.Keyword == "" && req.JobId == "" {
		return nil, fmt.Errorf("wrong parameter")
	}

	e, err := GlobalStorage.Get(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	var resp = new(schema.GetAsyncJobsResp)
	if e != nil {
		resp.List = []schema.SbsJobInfo{*e}
	}
	return resp, nil
}

// QueryJobStatus ...
func (jc *JobCenter) QueryJobStatus(ctx context.Context,
	flowID int64) (*schema.SbsJobInfo, error) {
	var resp schema.SbsJobInfo
	ret, err := GlobalStorage.GetByID(flowID)
	if err != nil {
		return nil, err
	}

	resp.JobName = ret.JobName
	resp.JobId = ret.JobId
	resp.Step = ret.Step
	resp.CreateTime = ret.CreateTime
	resp.Info = ret.Info
	resp.Status = ret.Status
	resp.TotalStep = ret.TotalStep
	resp.StepDesc = ret.StepDesc
	return &resp, nil
}

// RestartJob ...
func (jc *JobCenter) RestartJob(ctx context.Context, req *schema.RestartJobReq) error {
	keyword := req.KeyWord
	step := req.Step

	// check
	if step <= 0 {
		return fmt.Errorf("step num should be positive")
	}
	if keyword == "" {
		return fmt.Errorf("keyword is nil")
	}

	// 更新任务信息
	err := GlobalStorage.UpdateRestartInfo(keyword, step)
	if err != nil {
		Logger.Infof("update restartinfo  error %s", err.Error())
		return err
	}
	// 重启任务
	restartJob, err := GlobalStorage.FetchEmployeeByKeyword(keyword)
	if err != nil {
		Logger.Infof("Start asyncjobs  error %s", err.Error())
		return err
	}

	go restartJob.Do()

	return nil

}
