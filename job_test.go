package vtask

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"vtask/schema"

	"github.com/stretchr/testify/assert"
)

var ms *mockStorage

var succCallback = func(e interface{}) {
	fmt.Println("success callback")
}
var failCallback = func(e interface{}) {
	fmt.Println("failed callback")
}

type commonInfo struct {
	RequestId string `json:"RequestId"`
}

func init() {
	ms = new(mockStorage)
	ms.m = make(map[string]mockRow)
	GlobalStorage = ms

	GlobalJobCenter.PostJobDescribe("test_job", &commonInfo{}, []Step{
		{
			H: func(ctx context.Context, req interface{}) (interface{}, error) {
				input := req.(*commonInfo)
				input.RequestId += "1"
				fmt.Println(input)
				return input, nil
			},
			RetryTimes:  JustOnce,
			RetryPeriod: time.Second,
		},
		{
			H: func(ctx context.Context, req interface{}) (interface{}, error) {
				input := req.(*commonInfo)
				input.RequestId += "2"
				fmt.Println(input)
				return input, nil
			},
			RetryTimes:  JustOnce,
			RetryPeriod: time.Second,
		},
	}, succCallback, failCallback)

	GlobalJobCenter.PostJobDescribe("test_job_fail", &commonInfo{}, []Step{
		{
			H: func(ctx context.Context, req interface{}) (interface{}, error) {
				input := req.(*commonInfo)
				input.RequestId += "1"
				fmt.Println(input)
				return input, nil
			},
			RetryTimes:  JustOnce,
			RetryPeriod: time.Second,
		},
		{
			H: func(ctx context.Context, req interface{}) (interface{}, error) {
				input := req.(*commonInfo)
				input.RequestId += "2"
				fmt.Println("job failed")
				return input, StopJobError
			},
			RetryTimes:  JustOnce,
			RetryPeriod: time.Second,
		},
	}, succCallback, failCallback)
	GlobalJobCenter.PostJobDescribe("test_job_always_fail", &commonInfo{}, []Step{
		{
			H: func(ctx context.Context, req interface{}) (interface{}, error) {
				input := req.(*commonInfo)
				time.Sleep(time.Second)
				fmt.Println(input)
				return input, fmt.Errorf("failed")
			},
			RetryTimes:  AlwaysRetry,
			RetryPeriod: time.Second,
		},
	}, succCallback, failCallback)
}

type mockRow struct {
	uuid   string
	step   int
	status schema.JobStatus
	name   string
	input  string
	resp   string
	host   string
}

type mockStorage struct {
	m map[string]mockRow
}

func (store *mockStorage) Record(ctx context.Context, e *Employee) (int64,
	error) {
	store.m[e.jobID] = mockRow{
		uuid:   e.jobID,
		step:   e.step,
		status: e.status,
		name:   e.job.JobName,
		input:  e.inputString(),
		resp:   e.responseString(),
	}
	return 0, nil
}
func (store *mockStorage) Put(ctx context.Context, e *Employee) (int64, error) {
	store.m[e.jobID] = mockRow{
		uuid:   e.jobID,
		step:   e.step,
		status: e.status,
		name:   e.job.JobName,
		input:  e.inputString(),
		resp:   e.responseString(),
	}
	return 0, nil
}
func (store *mockStorage) GetByID(id int64) (*schema.SbsJobInfo, error) {
	return nil, nil
}
func (store *mockStorage) GetByIDs(ids []int64) ([]schema.SbsJobInfo, error) {
	return nil, nil
}
func (store *mockStorage) FetchEmployeeByKeyword(keyword string) (*Employee, error) {
	return nil, nil
}
func (store *mockStorage) UpdateRestartInfo(keyword string, step int32) error {
	return nil
}

func (store *mockStorage) GetNeedWorkEmployees(from, to time.Time) ([]*Employee, error) {
	var result []*Employee
	for k, v := range store.m {
		e, err := GlobalJobCenter.NewEmployee(k, v.name, "")
		if err != nil {
			return nil, err
		}
		if v.status != schema.StatusRunning {
			continue
		}

		e.step = v.step
		e.status = v.status
		err = json.Unmarshal([]byte(v.input), &e.input)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(v.resp), &e.resp)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}
func (store *mockStorage) Get(ctx context.Context, keyword string) (*schema.
	SbsJobInfo, error) {
	return nil, nil
}

func TestJob(t *testing.T) {
	e, err := GlobalJobCenter.NewEmployee("", "test_job", "")
	assert.Nil(t, err)
	e.Do()

	assert.Equal(t, schema.StatusSuccess, e.status)
	assert.Equal(t, 3, e.step)
	assert.Equal(t, nil, e.resp)
	fmt.Println(ms.m)

	list, err := GlobalStorage.GetNeedWorkEmployees(time.Now(), time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(list))
}

func TestJobFail(t *testing.T) {
	e, err := GlobalJobCenter.NewEmployee("", "test_job_fail", "")
	assert.Nil(t, err)
	e.Do()

	assert.Equal(t, schema.StatusFail, e.status)
	assert.Equal(t, 2, e.step)
	fmt.Println(ms.m)
	list, err := GlobalStorage.GetNeedWorkEmployees(time.Now(), time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(list))
	assert.Equal(t, &commonInfo{RequestId: "12"}, e.resp)
}

func TestJobDoing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	e, err := GlobalJobCenter.NewEmployee("", "test_job_always_fail", "")
	assert.Nil(t, err)
	GlobalStorage.Record(context.TODO(), e)
	go e.Do()

	select {
	case <-ctx.Done():
	}
	assert.Equal(t, schema.StatusRunning, e.status)
	assert.Equal(t, 1, e.step)
	fmt.Println(ms.m)
	list, err := GlobalStorage.GetNeedWorkEmployees(time.Now(), time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(list))
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx2, cancel2 := context.WithCancel(ctx)
	cancel2()

	select {
	case <-ctx.Done():
		panic("parent done")
	case <-ctx2.Done():
		fmt.Println("ok")
	default:
	}

	ctx, cancel = context.WithCancel(context.Background())
	ctx2, cancel2 = context.WithCancel(ctx)

	cancel() // cancel掉父context

	select {
	case <-ctx.Done(): // 父context已经done掉
		fmt.Println("ok")
	default:
		panic("parent done")
	}

	select {
	case <-ctx2.Done(): // 子context也会done掉
		fmt.Println("ok")
	default:
		panic("parent done")
	}

}

func TestContext2(t *testing.T) {
	ctx := context.WithValue(context.Background(), 1, 1)
	fmt.Println(ctx.Value(1)) // 1
	ctx2 := context.WithValue(ctx, 2, 2)
	fmt.Println(ctx2.Value(1)) // 1
	fmt.Println(ctx2.Value(2)) // 2
}
func TestContext3(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	context.TODO()
	ctx2, _ := context.WithTimeout(ctx, 2*time.Second)
	now := time.Now()
	select {
	case <-ctx2.Done():
		fmt.Println(time.Since(now)) // 1s
	}
}

func alwaysLock(rw *sync.RWMutex) {
	rw.Lock()
	go func() {
		time.Sleep(time.Second)
		rw.Unlock()
	}()

	time.Sleep(500 * time.Millisecond)
	go alwaysLock(rw)
}

func TestRwlock(t *testing.T) {
	rw := new(sync.RWMutex)
	now := time.Now()
	go func() {
		rw.Lock()
		time.Sleep(time.Second)
		rw.Unlock()
	}()

	go func() {
		time.Sleep(time.Millisecond * 500)
		rw.Lock()
		time.Sleep(time.Second)
		rw.Unlock()
	}()

	go func() {
		time.Sleep(time.Millisecond * 1500)
		rw.Lock()
		time.Sleep(time.Second)
		rw.Unlock()
	}()

	go func() {
		time.Sleep(800 * time.Millisecond)
		rw.RLock()
		fmt.Println(time.Since(now))
		rw.RUnlock()
	}()

	time.Sleep(10 * time.Second)

}
