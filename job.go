package asyncsbsjob

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"

	"go.uber.org/zap"
)

// Logger ...
var Logger *zap.SugaredLogger

func init() {
	tmp, _ := zap.NewProduction()
	Logger = tmp.Sugar()
}

// AlwaysRetry ...
const (
	AlwaysRetry = -1 // 始终重试
	JustOnce    = 0  // 只重试一次

)

// CtxJobId context中key定义
const (
	CtxJobId  = "CtxJobId" // 上下文 JobId
	CtxLogger = "CtxLogger"
)

// StepStatus ...
type StepStatus int

// GlobalStorage ...
var GlobalStorage JobStorage = &storage{GlobalJobCenter}

// JobCenter ...
type JobCenter struct {
	ctx    context.Context
	cancel context.CancelFunc
	m      map[string]JobDescribe
	host   string // host 宿主唯一标识，使用ip+port的方式进行区分

	wg *sync.WaitGroup
}

// GlobalJobCenter ...
var GlobalJobCenter = NewJobCenter()

// 错误定义
var (
	NoSuchJob            = fmt.Errorf("no such job")         // NoSuchJob jd没有发布
	StopJobError         = fmt.Errorf("job is failed")       // StopJobError 停止任务error，返回此error即可不再运行task
	retryReachLimitError = fmt.Errorf("retry reach limit")   // retryReachLimitError 任务重试超限
	JobCenterNoWorkJob   = fmt.Errorf("job center not work") // 不再产生新的job
)

// NewJobCenter 人才招聘会，负责招募员工执行任务
func NewJobCenter() *JobCenter {
	var jc JobCenter
	jc.ctx, jc.cancel = context.WithCancel(context.Background())
	jc.m = make(map[string]JobDescribe)
	jc.wg = new(sync.WaitGroup)
	return &jc
}

// PostJobDescribe 发布jd
func (jc *JobCenter) PostJobDescribe(name string, input interface{}, steps []Step, succCB, failCB Callback) {
	// input必须为指针
	if reflect.ValueOf(input).Kind() != reflect.Ptr {
		panic("post_a_job_describe_input_must_be_pointer")
	}

	for _, v := range steps {
		if v.RetryPeriod == 0 && v.RetryTimes > 0 {
			panic("retry period cannot be zero")
		}
	}

	var temp JobDescribe
	temp.input = input
	temp.Steps = steps
	temp.JobName = name
	temp.SuccessCallback = succCB
	temp.FailCallback = failCB
	jc.m[name] = temp
}

// JobDescribe step by step job,有步骤的任务
type JobDescribe struct {
	JobName         string
	input           interface{}
	Steps           []Step
	SuccessCallback Callback
	FailCallback    Callback
}

// Step 步骤
type Step struct {
	H           StepHandler   // 步骤处理函数
	RetryTimes  int           // 重试次数
	RetryPeriod time.Duration // 重试周期
	StepDesc    string
}

// StepHandler 步骤函数
type StepHandler func(ctx context.Context, req interface{}) (interface{}, error)

// Callback ...
type Callback func(interface{})

// Start ...
func (jc *JobCenter) Start(host string) {
	jc.host = host
	// 等待一分钟才开始异步任务，避免同一个任务同时被两个node执行
	ticker := time.NewTimer(time.Minute)
	var from, to time.Time
	for {
		select {
		case <-jc.ctx.Done():
			return
		case <-ticker.C:
			to = time.Now()
			ticker.Reset(time.Duration(GlobalConfig.LoadTaskPeriod) * time.
				Second)
			list, err := GlobalStorage.GetNeedWorkEmployees(from, to)
			if err != nil {
				Logger.With(zap.Error(err)).Errorf("Start asyncjobs  error %s",
					err.Error())
				continue
			}
			from = to
			Logger.Infof("input employee %+v", list)
			for _, v := range list {
				go v.Do()
			}
		}
	}
}

// NewEmployee ...
func (jc *JobCenter) NewEmployee(keyword, name, input string) (*Employee, error) {
	select {
	case <-jc.ctx.Done():
		return nil, JobCenterNoWorkJob
	default:
	}
	var e Employee
	if keyword == "" {
		e.jobID = uuid.New().String()
	} else {
		e.jobID = keyword
	}

	jd, ok := jc.m[name]
	if !ok {
		return nil, NoSuchJob
	}
	e.job = &jd
	e.step = 1
	e.input = clone(jd.input)
	if input != "" {
		err := json.Unmarshal([]byte(input), e.input)
		if err != nil {
			return nil, err
		}
	}

	e.wg = jc.wg
	e.ctx, e.cancel = context.WithCancel(jc.ctx)
	e.requestLog = Logger.With(zap.String("jobId", e.jobID))

	e.ctx = context.WithValue(e.ctx, CtxJobId, e.id)
	e.ctx = context.WithValue(e.ctx, CtxLogger, e.requestLog)
	return &e, nil
}

// Destroy 实现热重启方法
func (jc *JobCenter) Destroy() {
	jc.cancel()
	jc.wg.Wait()
}
