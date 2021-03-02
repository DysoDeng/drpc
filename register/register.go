package register

// Register 服务注册插件接口
type Register interface {
	Start() error
	Stop() error
	Register(name string, metadata string) error
	Unregister(name string) error
}
