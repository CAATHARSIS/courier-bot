package assignment

// import (
// 	"context"
// 	"log/slog"
// 	"sync"
// 	"time"

// 	"github.com/CAATHARSIS/courier-bot/internal/models"
// )

// type AssignmentManager struct {
// 	service         *Service
// 	log             *slog.Logger
// 	activeTimers    map[int]*time.Timer
// 	assignmentLocks map[int]bool
// 	waitingOrders   map[int]*WaitingOrder
// 	mu              sync.RWMutex
// }

// type WaitingOrder struct {
// 	OrderID    int
// 	AssignedAt time.Time
// 	ExpiredAt  time.Time
// 	CourierID  int
// 	Status     models.CourierResponseStatus
// 	RetryCount int
// 	LastError  string
// }

// func NewAssignmentManager(service *Service, log *slog.Logger) *AssignmentManager {
// 	return &AssignmentManager{
// 		service:         service,
// 		log:             log,
// 		activeTimers:    make(map[int]*time.Timer),
// 		assignmentLocks: make(map[int]bool),
// 		waitingOrders:   make(map[int]*WaitingOrder),
// 	}
// }

// func (m *AssignmentManager) ProcessNewOrder(ctx context.Context, orderID int) error {
// 	m.log.Info("AssignmentManager: processing new order", "orderID", orderID)

// 	if m.isOrderLocked(orderID) {
// 		m.log.Debug("This order is processing (skip)", "orderID", orderID)
// 		return nil
// 	}

// 	m.lockOrder(orderID)
// 	defer m.unlockOrder(orderID)

// 	m.registryWaitingOrder(orderID)

// 	return m.service.ProcessNewOrder(ctx, orderID)
// }

// func (m *AssignmentManager) HandleAssignmentTimeout(ctx context.Context, orderID int) {
// 	m.log.Info("AssignmentManager: timeout for order", "orderID", orderID)

// 	m.mu.Lock()
// 	delete(m.activeTimers, orderID)

// 	if waiting, exists := m.waitingOrders[orderID]; exists {
// 		waiting.Status = models.ResponsseStatusExpired
// 		waiting.LastError = "Assignment timeout"
// 	}
// 	m.mu.Unlock()

// 	go m.retryAssignment(ctx, orderID)
// }

// func (m *AssignmentManager) HandleCourierResponse(ctx context.Context, chatID int64, orderID int, accepted bool) error {
// 	m.log.Info("AssignmentManager: handling courier response for order", "orderID", orderID, "accepted", accepted)

// 	m.cancelTimer(orderID)

// 	m.updateWaitingOrderStatus(orderID, accepted)

// 	err := m.service.HandleCourierResponse(ctx, chatID, orderID, accepted)
// 	if err != nil {
// 		m.log.Error("Failed to handle courier response", "error", err)
// 	}

// 	return err
// }

// func (m *AssignmentManager) RegisterTimer(ctx context.Context, orderID int, expiry time.Time) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if oldTimer, exists := m.activeTimers[orderID]; exists {
// 		oldTimer.Stop()
// 	}

// 	duration := time.Until(expiry)
// 	if duration <= 0 {
// 		go m.HandleAssignmentTimeout(ctx, orderID)
// 		return
// 	}

// 	timer := time.AfterFunc(duration, func() {
// 		m.HandleAssignmentTimeout(ctx, orderID)
// 	})

// 	m.activeTimers[orderID] = timer

// 	if waiting, exists := m.waitingOrders[orderID]; exists {
// 		waiting.ExpiredAt = expiry
// 	}
// }

// func (m *AssignmentManager) registerWaitingOrder(orderID int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.waitingOrders[orderID] = &WaitingOrder{
// 		OrderID:    orderID,
// 		AssignedAt: time.Now(),
// 		Status:     models.ResponseStatusWaiting,
// 	}
// }

// func (m *AssignmentManager) GetActiveAssignments(orderID int) map[int]*WaitingOrder {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	active := make(map[int]*WaitingOrder)
// 	for orderID, waiting := range m.waitingOrders {
// 		if waiting.Status == models.ResponseStatusWaiting {
// 			active[orderID] = waiting
// 		}
// 	}

// 	return active
// }

// func (m *AssignmentManager) GetAssignmentsStats() *AssignmentStats {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	stats := &AssignmentStats{
// 		ActiveTimers: len(m.activeTimers),
// 		LockedOrders: len(m.assignmentLocks),
// 		ByStatus:     make(map[models.CourierResponseStatus]int),
// 	}

// 	for _, waiting := range m.waitingOrders {
// 		stats.TotalWaiting++
// 		stats.ByStatus[waiting.Status]++

// 		if waiting.RetryCount > stats.MaxRetries {
// 			stats.MaxRetries = waiting.RetryCount
// 		}
// 	}

// 	return stats
// }

// type AssignmentStats struct {
// 	TotalWaiting int
// 	ActiveTimers int
// 	LockedOrders int
// 	MaxRetries   int
// 	ByStatus     map[models.CourierResponseStatus]int
// }

// func (m *AssignmentManager) CancelAssignment(orderID int) error {
// 	m.log.Info("AssignmentManager: canceling assignment for order", "orderID", orderID)

// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if timer, exists := m.activeTimers[orderID]; exists {
// 		timer.Stop()
// 		delete(m.activeTimers, orderID)
// 	}

// 	delete(m.waitingOrders, orderID)

// 	delete(m.assignmentLocks, orderID)

// 	return nil
// }

// func (m *AssignmentManager) isOrderLocked(orderID int) bool {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	return m.assignmentLocks[orderID]
// }

// func (m *AssignmentManager) lockOrder(orderID int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.assignmentLocks[orderID] = true
// }

// func (m *AssignmentManager) unlockOrder(orderID int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	delete(m.assignmentLocks, orderID)
// }

// func (m *AssignmentManager) registryWaitingOrder(orderID int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.waitingOrders[orderID] = &WaitingOrder{
// 		OrderID:    orderID,
// 		AssignedAt: time.Now(),
// 		Status:     models.ResponseStatusWaiting,
// 		RetryCount: 0,
// 	}
// }

// func (m *AssignmentManager) cancelTimer(orderID int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if timer, exists := m.activeTimers[orderID]; exists {
// 		timer.Stop()
// 		delete(m.activeTimers, orderID)
// 	}
// }

// func (m *AssignmentManager) retryAssignment(ctx context.Context, orderID int) {
// 	m.mu.Lock()

// 	if wating, exists := m.waitingOrders[orderID]; exists {
// 		wating.RetryCount++
// 		wating.Status = models.ResponseStatusWaiting
// 		wating.LastError = ""
// 	}

// 	m.mu.Unlock()

// 	m.log.Info("AssignmentManager: retrying assignment for order", "orderID", orderID)

// 	if m.getRetryCount(orderID) > 3 {
// 		m.log.Warn("Max retry count reached for order", "orderID", orderID)
// 		m.CancelAssignment(orderID)
// 		return
// 	}

// 	if err := m.ProcessNewOrder(ctx, orderID); err != nil {
// 		m.log.Error("Retry failed for order", "orderID", orderID, "error", err)
// 	}
// }

// func (m *AssignmentManager) getRetryCount(orderID int) int {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	if wating, exists := m.waitingOrders[orderID]; exists {
// 		return wating.RetryCount
// 	}

// 	return 0
// }

// func (m *AssignmentManager) updateWaitingOrderStatus(orderID int, accepted bool) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if waiting, exists := m.waitingOrders[orderID]; exists {
// 		if accepted {
// 			waiting.Status = models.ResponseStatusAccepted
// 		} else {
// 			waiting.Status = models.ResponseStatusRejected
// 		}

// 		waiting.LastError = ""
// 	}
// }

// func (m *AssignmentManager) updateWaitingOrderError(orderID int, errorMsg string) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if waiting, exists := m.waitingOrders[orderID]; exists {
// 		waiting.LastError = errorMsg
// 	}
// }

// func (m *AssignmentManager) CleanUpAssignments() {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	now := time.Now()
// 	lifeCycle := 24 * time.Hour

// 	for orderID, waiting := range m.waitingOrders {
// 		if now.Sub(waiting.AssignedAt) > lifeCycle {
// 			m.log.Info("Delete dead assignments", "orderID", orderID)

// 			if timer, exists := m.activeTimers[orderID]; exists {
// 				timer.Stop()
// 				delete(m.activeTimers, orderID)
// 			}

// 			delete(m.waitingOrders, orderID)
// 			delete(m.assignmentLocks, orderID)
// 		}
// 	}
// }

// func (m *AssignmentManager) GetWaitingOrderInfo(orderID int) *WaitingOrder {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	return m.waitingOrders[orderID]
// }

// func (m *AssignmentManager) StartCleanupWorker() {
// 	go func() {
// 		ticker := time.NewTicker(1 * time.Hour)
// 		defer ticker.Stop()

// 		for range ticker.C {
// 			m.CleanUpAssignments()
// 		}
// 	}()
// }
