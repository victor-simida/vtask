package schema

// SbsJobInfo ...
type SbsJobInfo struct {
	JobName    string `json:"JobName"`
	Status     int32  `json:"Status"`
	CreateTime string `json:"CreateTime"`
	Info       string `json:"Info"`
	JobId      string `json:"JobId"`
	Step       int32  `json:"Step"`
	Id         int64  `json:"Id"`
	TotalStep  int32  `json:"TotalStep"`
	StepDesc   string `json:"StepDesc"`
}

// JobStatus ...
type JobStatus int

// StatusRunning ...
const (
	StatusRunning JobStatus = iota // doing
	StatusFail                     // 成功
	StatusSuccess                  // 失败
)

// InvokeAsyncJobReq ...
type InvokeAsyncJobReq struct {
	JobName   string `json:"JobName"`
	InputData string `json:"InputData"`
	Keyword   string `json:"Keyword"`
}

// InvokeAsyncJobResp ...
type InvokeAsyncJobResp struct {
	JobID  string `json:"JobID"`
	FlowID int64  `json:"FlowID"`
}

// GetAsyncJobsReq ...
type GetAsyncJobsReq struct {
	Keyword string `json:"Keyword"`
	Status  int32  `json:"Status"`
	JobId   string `json:"JobId"`
}

// GetAsyncJobsResp ...
type GetAsyncJobsResp struct {
	List []SbsJobInfo `json:"List"`
}

// RestartJobReq ...
type RestartJobReq struct {
	KeyWord string `json:"KeyWord"`
	Step    int32  `json:"step"`
}
