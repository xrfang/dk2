package ctrl

type (
	Config struct {
		MgmtPort  int               `yaml:"mgmt_port"`
		ServPort  int               `yaml:"serv_port"`
		MaxServes int               `yaml:"max_serves"`
		Handshake int               `yaml:"handshake"`
		KeepAlive int               `yaml:"keep_alive"`
		IdleClose int               `yaml:"idle_close"`
		AuthTime  int               `yaml:"auth_time"`
		OTPIssuer string            `yaml:"otp_issuer"`
		WebRoot   string            `yaml:"web_root"`
		Users     map[string]string `yaml:"users"`
		Auths     map[string]string `yaml:"auths"`
		Version   string            `yaml:"-"`
	}
)
