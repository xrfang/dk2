package serv

type Config struct {
	Name    string   `yaml:"name"`
	SvrHost string   `yaml:"svr_host"`
	SvrPort int      `yaml:"svr_port"`
	Auth    string   `yaml:"auth"`
	LanNets []string `yaml:"lan_nets"`
	MacScan int      `yaml:"mac_scan"`
}
