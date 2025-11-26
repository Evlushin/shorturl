package observers

import (
	"context"
	"encoding/json"
	"github.com/Evlushin/shorturl/internal/models"
	"log"
	"os"
	"sync"
)

type FileAuditLogger struct {
	filename string
	fileChan chan models.AuditEvent
	file     *os.File
	encoder  *json.Encoder
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	closed   bool
}

func NewFileAuditLogger(ctx context.Context, filename string) *FileAuditLogger {
	ctx, cancel := context.WithCancel(ctx)

	logger := &FileAuditLogger{
		filename: filename,
		fileChan: make(chan models.AuditEvent, 50),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Запускаем writer в отдельной горутине
	logger.wg.Add(1)
	go logger.startWriter()

	return logger
}

func (l *FileAuditLogger) OnAuditEvent(event models.AuditEvent) {
	select {
	case l.fileChan <- event:
		// Событие успешно отправлено
	case <-l.ctx.Done():
		// Контекст отменён — игнорируем событие
	}
}

func (l *FileAuditLogger) startWriter() {
	defer l.wg.Done()

	// Открываем файл
	file, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Ошибка открытия файла для аудита: %v", err)
		return
	}
	l.file = file
	l.encoder = json.NewEncoder(file)

	for {
		select {
		case event, ok := <-l.fileChan:
			if !ok {
				// Канал закрыт — завершаем работу
				l.closeFile()
				return
			}
			if err := l.encoder.Encode(event); err != nil {
				log.Printf("Ошибка записи в файл аудита: %v", err)
			}

		case <-l.ctx.Done():
			// Контекст отменён — закрываем канал и завершаем
			l.mu.Lock()
			if !l.closed {
				close(l.fileChan)
				l.closed = true
			}
			l.mu.Unlock()
			l.closeFile()
			return
		}
	}
}

// Закрытие файла с защитой от повторного вызова
func (l *FileAuditLogger) closeFile() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}

// Stop Метод для корректного завершения логгера
func (l *FileAuditLogger) Stop() {
	l.cancel()  // Отменяем контекст
	l.wg.Wait() // Ждём завершения горутины writer
}
