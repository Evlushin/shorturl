package subjects

import (
	"context"
	"sync"

	"github.com/Evlushin/shorturl/internal/models"
)

type AuditObserver interface {
	OnAuditEvent(event models.AuditEvent)
	Stop()
}

type AuditManager struct {
	observers []AuditObserver
	eventChan chan models.AuditEvent
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	closed    bool
}

func NewAuditManager(ctx context.Context) *AuditManager {
	ctx, cancel := context.WithCancel(ctx)

	manager := &AuditManager{
		observers: make([]AuditObserver, 0),
		eventChan: make(chan models.AuditEvent, 100),
		ctx:       ctx,
		cancel:    cancel,
	}

	manager.wg.Add(1)
	go manager.startDispatcher()
	return manager
}

func (am *AuditManager) RegisterObserver(observer AuditObserver) {
	am.mu.Lock()
	am.observers = append(am.observers, observer)
	am.mu.Unlock()
}

func (am *AuditManager) RemoveObserver(observer AuditObserver) {
	am.mu.Lock()
	for i, obs := range am.observers {
		if obs == observer {
			observer.Stop()
			am.observers = append(am.observers[:i], am.observers[i+1:]...)
			break
		}
	}
	am.mu.Unlock()
}

func (am *AuditManager) NotifyObservers(event models.AuditEvent) {
	select {
	case am.eventChan <- event:
	case <-am.ctx.Done():
	}
}

func (am *AuditManager) startDispatcher() {
	defer am.wg.Done()

	for {
		select {
		case event, ok := <-am.eventChan:
			if !ok {
				return
			}
			am.mu.Lock()
			observers := am.observers
			am.mu.Unlock()

			for _, observer := range observers {
				am.wg.Add(1)
				go func(obs AuditObserver, evt models.AuditEvent) {
					defer am.wg.Done()
					obs.OnAuditEvent(evt)
				}(observer, event)
			}

		case <-am.ctx.Done():
			am.mu.Lock()
			if !am.closed {
				close(am.eventChan)
				am.closed = true
			}
			am.mu.Unlock()
			return
		}
	}
}

func (am *AuditManager) Stop() {
	am.cancel() // Отменяем контекст — это запустит завершение диспетчера

	// Ждём завершения диспетчера (основной горутины)
	am.wg.Wait()

	// Теперь останавливаем всех наблюдателей
	am.mu.Lock()
	for _, observer := range am.observers {
		observer.Stop()
	}
	am.observers = nil // Очищаем список, чтобы избежать повторных вызовов
	am.mu.Unlock()
}
