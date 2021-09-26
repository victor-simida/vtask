# vtask

一个golang基础的分布式异步任务框架

# install

```
    go get github.com/victor-simida/vtask
```

# usage

### 使用Register方法来注册异步任务

```
    type input struct {
        Input string
    }
    
    vtask.GlobalJobCenter.Register("test_job", &input{},
    []vtask.Step{
        {
            // 在这里定义执行该步骤执行的函数
            H: func(ctx context.Context, req interface{}) (interface{}, error) {
                input := req.(*input)   // 上下文在这里进行保存
                input.Input = "hello"
                return input, nil
            },
            RetryTimes:  vtask.JustOnce, // 该步骤最多重试次数
            RetryPeriod: time.Second,    // 重试周期
        },
        {
            H: func(ctx context.Context, req interface{}) (interface{}, error) {
                input := req.(*input)   // 此时读取的input是上一步骤中改写的
                input.Input = fmt.Sprintf("%s world", input.Input)
                return input, nil
            },
            RetryTimes:  vtask.JustOnce,
            RetryPeriod: time.Second,
        },
    }, nil, nil)
```

### 使用InvokeAsyncJob方法来启动异步任务

```
	resp, err := GlobalJobCenter.InvokeAsyncJob(context.TODO(),
        &schema.InvokeAsyncJobReq{
            JobName:   "test_job",
            InputData: `{"RequestId":"test"}`,
            Keyword:   "uuid1",
        })
    assert.Nil(t, err)
    assert.Equal(t, "uuid1", resp.JobID)    
```

如果指定keyword，则返回中的jobId等于这个keyword。

同一个keyword必须全局唯一，任务框架通过这个keyword保证幂等，如果传入一个非空的已经存在的keyword，则不会启动新的异步任务

### 使用Start方法来启动异步任务服务

```go
    GlobalJobCenter.Start("127.0.0.1:1234")
```

Start方法的这个入参需要传入一个host值，用于标识节点身份。在分布式系统中，每个节点的这个值应当唯一且不变，建议使用ip+port做host

### 持久化
目前只支持使用mysql来进行持久化

异步任务会缓存在mysql数据库中，每执行一个步骤都会记录更新其上下文信息，保证服务挂掉后能够重新拉起
通过SetDataBase来设置全局mysql
```mysql
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
) 
```
您也可以自己实现自己的持久化interface:
```go
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
```

### 服务重启
通过Destroy方法，可以重启服务，服务会等待当前所有doing状态的异步任务运行完当前步骤
```go
    GlobalJobCenter.Destroy()
```
