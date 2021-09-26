package example

import (
	"context"
	"fmt"
	"time"

	"vtask"
)

type input struct {
	Input string
}

func init() {
	vtask.GlobalJobCenter.Register("test_job", &input{},
		[]vtask.Step{
			{
				// 在这里定义执行该步骤执行的函数
				H: func(ctx context.Context, req interface{}) (interface{}, error) {
					input := req.(*input)
					input.Input = "hello"
					return input, nil
				},
				RetryTimes:  vtask.JustOnce, // 该步骤最多重试次数
				RetryPeriod: time.Second,    // 重试周期
			},
			{
				H: func(ctx context.Context, req interface{}) (interface{}, error) {
					input := req.(*input)
					input.Input = fmt.Sprintf("%s world", input.Input)
					return input, nil
				},
				RetryTimes:  vtask.JustOnce,
				RetryPeriod: time.Second,
			},
		}, nil, nil)
}
