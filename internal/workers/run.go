package workers

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"leguiburger/internal/businesscare"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/operations"
	"log"
	"time"
)

// API and external worker use the same durable repositories and database leases.
// Disable the embedded worker when deploying cmd/worker independently.
func Run(ctx context.Context, db *gorm.DB) {
	auth.StartMailWorker(ctx, db)
	businesscare.StartWorker(ctx, db)
	mp := mercadopago.NewHandler(db, mercadopago.LoadConfig(), nil)
	mp.StartWorker(ctx)
	instance := uuid.NewString()
	db.Exec(`INSERT INTO legui_private.worker_heartbeats(instance_id) VALUES(?)`, instance)
	nextSecurityCleanup := time.Time{}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		work, cancel := context.WithTimeout(ctx, 12*time.Second)
		if time.Now().After(nextSecurityCleanup) {
			cleanup, stopCleanup := context.WithTimeout(ctx, 5*time.Second)
			cleanupErr := operations.CleanupCheckoutSecurity(db.WithContext(cleanup))
			stopCleanup()
			nextSecurityCleanup = time.Now().Add(time.Hour)
			if cleanupErr != nil {
				log.Printf("security retention retry pending (%T)", cleanupErr)
				nextSecurityCleanup = time.Now().Add(5 * time.Minute)
			}
		}
		err := operations.ProcessStorefrontWork(work, db)
		if err == nil {
			err = operations.GenerateNotifications(work, db)
		}
		if err != nil {
			log.Printf("worker notifications retry pending (%T)", err)
		}
		result := db.WithContext(work).Exec(`UPDATE legui_private.worker_heartbeats SET last_seen_at=clock_timestamp(),failed=?,last_success_at=CASE WHEN ? THEN last_success_at ELSE clock_timestamp() END WHERE instance_id=?`, err != nil, err != nil, instance)
		if result.Error != nil {
			log.Printf("worker heartbeat unavailable (%T)", result.Error)
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
