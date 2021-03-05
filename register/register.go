package register

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

	// 服务注销
	// name 服务名称
	Unregister(name string) error
}
