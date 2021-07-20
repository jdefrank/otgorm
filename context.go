package otgorm

import (
	"context"

	"github.com/jinzhu/gorm"
)

// WithContext sets the current context in the db instance for instrumentation.
func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.Set(contextScopeKey, ctx)
}

// WithContext sets the current context in the db instance for instrumentation.
func SetSpanToGorm(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.Set(contextScopeKey, ctx)
}
