package db

import (
	"database/sql"
	"sync"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

var (
	dbStatsRegistrations sync.Map
)

// RegisterDBStatsMetrics 注册 database/sql 连接池指标，避免只看 trace 看不到池子压力。
func RegisterDBStatsMetrics(db *sql.DB, dbSystem string) error {
	if db == nil || !telemetry.HasMeterProvider() {
		return nil
	}
	if _, exists := dbStatsRegistrations.Load(db); exists {
		return nil
	}

	reg, err := otelsql.RegisterDBStatsMetrics(
		db,
		otelsql.WithMeterProvider(telemetry.GetMeterProvider()),
		otelsql.WithAttributes(attribute.String("db.system", dbSystem)),
	)
	if err != nil {
		return err
	}

	dbStatsRegistrations.Store(db, reg)
	telemetry.RegisterMetricRegistration(reg)
	return nil
}

func unregisterDBStatsMetrics(db *sql.DB) {
	if db == nil {
		return
	}
	reg, ok := dbStatsRegistrations.LoadAndDelete(db)
	if !ok {
		return
	}
	registration, ok := reg.(metric.Registration)
	if !ok {
		return
	}
	_ = registration.Unregister()
}
