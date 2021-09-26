package asyncsbsjob

// GlobalConfig ...
var GlobalConfig = new(Config)

// Config ...
type Config struct {
	LoadTaskPeriod int // 拉取task周期
}
