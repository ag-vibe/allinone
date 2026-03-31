//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/wibus-wee/allinone/pkg"
	"github.com/wibus-wee/allinone/pkg/asynctask"
	"github.com/wibus-wee/allinone/pkg/config"
	"github.com/wibus-wee/allinone/pkg/deviceauth"
	"github.com/wibus-wee/allinone/pkg/handler"
	"github.com/wibus-wee/allinone/pkg/zcore/app"
	"github.com/wibus-wee/allinone/pkg/zcore/injection"
	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/taskgen"

	"github.com/google/wire"
)

func InitApp() (*app.App, error) {
	wire.Build(
		injection.InjectAuth,
		injection.InjectService,
		injection.InjectTaskStore,
		deviceauth.NewService,
		handler.NewHandler,
		handler.NewValidator,
		taskgen.NewTaskHandler,
		taskgen.NewTaskRunner,
		asynctask.NewExecutor,
		model.NewModel,
		config.NewConfig,
		pkg.Init,
		pkg.InitAnclaxApplication,
		app.NewPlugin,
	)
	return nil, nil
}
