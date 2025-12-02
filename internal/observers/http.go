package observers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Evlushin/shorturl/internal/models"
)

type HTTPAuditSender struct {
	endpoint string
	client   *http.Client
	reqChan  chan models.AuditEvent
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	closed   bool
}

func NewHTTPAuditSender(ctx context.Context, endpoint string) *HTTPAuditSender {
	// Создаем дочерний контекст с возможностью отмены
	ctx, cancel := context.WithCancel(ctx)

	sender := &HTTPAuditSender{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 5 * time.Second},
		reqChan:  make(chan models.AuditEvent, 50),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Запускаем обработчик в отдельной горутине
	sender.wg.Add(1)
	go sender.startSender()

	return sender
}

func (s *HTTPAuditSender) OnAuditEvent(event models.AuditEvent) {
	select {
	case s.reqChan <- event:
		// Событие успешно отправлено в канал
	case <-s.ctx.Done():
		// Контекст отменен — игнорируем событие
	}
}

func (s *HTTPAuditSender) startSender() {
	defer s.wg.Done()

	for {
		select {
		case event, ok := <-s.reqChan:
			if !ok {
				// Канал закрыт — завершаем работу
				return
			}

			// Обрабатываем событие в отдельной горутине (как в оригинале)
			go func(evt models.AuditEvent) {
				jsonData, err := json.Marshal(evt)
				if err != nil {
					log.Printf("Ошибка сериализации события аудита: %v", err)
					return
				}

				resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					log.Printf("Ошибка отправки аудита на %s: %v", s.endpoint, err)
					return
				}
				defer resp.Body.Close() // Обязательно закрываем тело ответа

				if resp.StatusCode != http.StatusOK {
					log.Printf("Сервер вернул код %d при отправке аудита", resp.StatusCode)
				}
			}(event)

		case <-s.ctx.Done():
			// Контекст отменен — закрываем канал и завершаем
			s.mu.Lock()
			if !s.closed {
				close(s.reqChan)
				s.closed = true
			}
			s.mu.Unlock()
			return
		}
	}
}

// Stop Метод для корректного завершения отправителя
func (s *HTTPAuditSender) Stop() {
	s.cancel()  // Отменяем контекст
	s.wg.Wait() // Ждем завершения горутины startSender
}
