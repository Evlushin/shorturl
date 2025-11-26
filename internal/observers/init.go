package observers

import (
	"context"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/observers/subjects"
)

func InitAuditObservers(ctx context.Context, cfg config.Config) *subjects.AuditManager {
	// Создаём менеджер аудита
	auditManager := subjects.NewAuditManager(ctx)

	// Регистрируем наблюдателей
	if cfg.Audit.AuditFile != "" {
		fileLogger := NewFileAuditLogger(ctx, cfg.Audit.AuditFile)
		auditManager.RegisterObserver(fileLogger)
	}

	if cfg.Audit.AuditURL != "" {
		httpSender := NewHTTPAuditSender(ctx, cfg.Audit.AuditURL)
		auditManager.RegisterObserver(httpSender)
	}

	return auditManager
}
