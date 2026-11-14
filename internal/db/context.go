package db

import (
	"context"
	"gorm.io/gorm"
)

type transactionKey struct{}

func WithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

func ForContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return DB.WithContext(ctx)
}
