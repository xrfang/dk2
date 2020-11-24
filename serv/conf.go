package serv

type Config struct {
	Name     string   `yaml:"name"`
	ConnWait int      `yaml:"conn_wait"`
	CtrlHost string   `yaml:"ctrl_host"`
	CtrlPort int      `yaml:"ctrl_port"`
	Auth     string   `yaml:"auth"`
	LanNets  []string `yaml:"lan_nets"`
	MacScan  int      `yaml:"mac_scan"`
}
