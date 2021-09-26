package asyncsbsjob

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"vtask/schema"

	"go.uber.org/zap"
)

// Employee 负责处理job
type Employee struct {
	id     int64
	jobID  string // 对应数据库中uuid，全局唯一
	ctx    context.Context
	cancel context.CancelFunc
	job    *JobDescribe
	step   int
	status schema.JobStatus

	input interface{}
	resp  interface{}

	requestLog *zap.SugaredLogger

	wg *sync.WaitGroup

	err error
}

func clone(src interface{}) interface{} {
	in := reflect.ValueOf(src)
	if in.IsNil() {
		return src
	}
	out := reflect.New(in.Type().Elem())
	return out.Interface()
}

// Do ...
func (e *Employee) Do() {
	defer func() {
		if emsg := recover(); emsg != nil {
			e.requestLog.Errorf("panic:%v\n", emsg)
			e.status = schema.StatusFail
			e.Record(emsg)
		}
		e.requestLog.Info("Exit")
		e.cancel()
		e.wg.Done()
	}()

	e.wg.Add(1)
	var cannotRetry bool
	for i := e.step; i <= len(e.job.Steps); i++ {
		s := e.job.Steps[i-1]
		var retryTimes int
		for {
			select {
			case <-e.ctx.Done():
				return
			default:
			}
			resp, err := s.H(e.ctx, e.input)
			retryTimes++
			e.err = err

			e.requestLog.With(zap.Error(err)).Infof("do_returns_%v_%+v", i,
				resp)
			if e.err != nil && e.err != StopJobError {
				e.requestLog.With(zap.Error(err)).Infof("step_error_%v", i)
				// 重试
				if s.RetryTimes == AlwaysRetry || retryTimes < s.RetryTimes {
					time.Sleep(s.RetryPeriod)
					continue
				}

				// 否则为reach limit error
				cannotRetry = true
			}

			if cannotRetry || err == StopJobError {
				e.status = schema.StatusFail
				e.Record(resp)
				e.requestLog.With(zap.Error(err)).Error("failed")
				if e.job.FailCallback != nil {
					e.job.FailCallback(e.input)
				}
				return
			} else {
				e.step++
				e.Record(resp)
				break
			}
		}
	}
	e.status = schema.StatusSuccess
	e.Record(nil)

	if e.job.SuccessCallback != nil {
		e.job.SuccessCallback(e.input)
	}
	return
}

// Record ...
func (e *Employee) Record(resp interface{}) {
	select {
	case <-e.ctx.Done():
		e.requestLog.Infof("record_ctx_done")
		return
	default:

	}
	e.resp = resp
	_, err := GlobalStorage.Record(e.ctx, e)
	if err != nil {
		e.requestLog.Errorf("Record %v error %s", e.jobID, err.Error())
	}
}

func (e *Employee) responseString() string {
	b, _ := json.Marshal(e.resp)
	return string(b)
}

func (e *Employee) inputString() string {
	b, _ := json.Marshal(e.input)
	return string(b)
}
