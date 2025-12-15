package config

type TAudit struct {
	AuditFile string
	AuditURL  string
}

type Config struct {
	ServerAddr string
	BaseAddr   string
	SecretKey  string
	Audit      TAudit
}
