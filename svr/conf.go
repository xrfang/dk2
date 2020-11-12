package svr

type (
	Config struct {
		CtrlPort  int               `yaml:"ctrl_port"`
		ServPort  int               `yaml:"serv_port"`
		Handshake int               `yaml:"handshake"`
		IdleClose int               `yaml:"idle_close"`
		AuthTime  int               `yaml:"auth_time"`
		OTPIssuer string            `yaml:"otp_issuer"`
		Users     map[string]string `yaml:"users"`
		Auths     map[string]string `yaml:"auths"`
	}
)
