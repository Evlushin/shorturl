package config

import "time"

type TAudit struct {
	AuditFile string `env:"AUDIT_FILE" flag:"audit-file" default:"audit.txt" help:"path to the destination file where audit logs are saved"`
	AuditURL  string `env:"AUDIT_URL" flag:"audit-url" default:"" help:"the full URL of the remote receiving server where the audit logs are sent"`
}

type Config struct {
	ServerAddr   string `env:"SERVER_ADDRESS" flag:"a" default:"localhost:8080" help:"address of HTTP server"`
	BaseAddr     string `env:"BASE_URL" flag:"b" default:"http://localhost:8080" help:"base address of the resulting shortened URL"`
	SecretKey    string `env:"SECRET_KEY" flag:"s" default:"{{uuid}}" help:"secret key"`
	Audit        TAudit
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"5s" help:"HTTP server read timeout"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"10s" help:"HTTP server write timeout"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" default:"120s" help:"HTTP server idle timeout"`
}
