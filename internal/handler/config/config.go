package config

import "time"

type TAudit struct {
	AuditFile string `env:"AUDIT_FILE" env-default:"audit.txt" env-description:"path to the destination file where audit logs are saved"`
	AuditURL  string `env:"AUDIT_URL" env-default:"" env-description:"the full URL of the remote receiving server where the audit logs are sent"`
}

type Config struct {
	ServerAddr   string `env:"SERVER_ADDRESS" env-default:"localhost:8080" env-description:"address of HTTP server"`
	BaseAddr     string `env:"BASE_URL" env-default:"http://localhost:8080" env-description:"base address of the resulting shortened URL"`
	SecretKey    string `env:"SECRET_KEY" env-default:"{{uuid}}" env-description:"secret key"`
	Audit        TAudit
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" env-default:"5s" env-description:"HTTP server read timeout"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" env-default:"10s" env-description:"HTTP server write timeout"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" env-default:"120s" env-description:"HTTP server idle timeout"`
}
