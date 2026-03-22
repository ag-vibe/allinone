package asynctask

import (
	"context"

	"github.com/cloudcarver/anclax/pkg/taskcore/worker"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/taskgen"
)

type Executor struct {
	model model.ModelInterface
}

func NewExecutor(model model.ModelInterface) taskgen.ExecutorInterface {
	return &Executor{
		model: model,
	}
}

func (e *Executor) ExecuteIncrementCounter(ctx context.Context, _ worker.Task, _ *taskgen.IncrementCounterParameters) error {
	return e.model.IncrementCounter(ctx)
}

func (e *Executor) ExecuteAutoIncrementCounter(ctx context.Context, _ worker.Task, _ *taskgen.AutoIncrementCounterParameters) error {
	return e.model.IncrementCounter(ctx)
}
