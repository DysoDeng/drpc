package register

import (
	"github.com/rcrowley/go-metrics"
)

// Register 服务注册器接口
type Register interface {

	// Init 初始化服务注册
	Init() error

	// Stop 停止服务注册
	Stop() error

	// Register 服务注册
	// name 服务名称
	// metadata 服务元数据
	Register(name string, metadata string) error

	// Unregister 服务注销
	// name 服务名称
	Unregister(name string) error

	// GetMetrics 获取Meter
	GetMetrics() metrics.Meter

	// IsShowMetricsLog 是否显示监控日志
	IsShowMetricsLog() bool
}
