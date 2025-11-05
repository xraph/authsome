package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/enterprise/geofence"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create geofence_rules table
		_, err := db.NewCreateTable().
			Model((*geofence.GeofenceRule)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for geofence_rules
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_rules_org_id ON geofence_rules(organization_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_rules_user_id ON geofence_rules(user_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_rules_enabled ON geofence_rules(enabled)")
		if err != nil {
			return err
		}

		// Create location_events table
		_, err = db.NewCreateTable().
			Model((*geofence.LocationEvent)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for location_events
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_location_events_user_id ON location_events(user_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_location_events_org_id ON location_events(organization_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_location_events_session_id ON location_events(session_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_location_events_timestamp ON location_events(timestamp DESC)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_location_events_ip ON location_events(ip_address)")
		if err != nil {
			return err
		}

		// Create travel_alerts table
		_, err = db.NewCreateTable().
			Model((*geofence.TravelAlert)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for travel_alerts
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_travel_alerts_user_id ON travel_alerts(user_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_travel_alerts_org_id ON travel_alerts(organization_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_travel_alerts_status ON travel_alerts(status)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_travel_alerts_requires_approval ON travel_alerts(requires_approval)")
		if err != nil {
			return err
		}

		// Create trusted_locations table
		_, err = db.NewCreateTable().
			Model((*geofence.TrustedLocation)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for trusted_locations
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_trusted_locations_user_id ON trusted_locations(user_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_trusted_locations_org_id ON trusted_locations(organization_id)")
		if err != nil {
			return err
		}

		// Create geofence_violations table
		_, err = db.NewCreateTable().
			Model((*geofence.GeofenceViolation)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for geofence_violations
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_violations_user_id ON geofence_violations(user_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_violations_org_id ON geofence_violations(organization_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_violations_rule_id ON geofence_violations(rule_id)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_violations_resolved ON geofence_violations(resolved)")
		if err != nil {
			return err
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geofence_violations_created_at ON geofence_violations(created_at DESC)")
		if err != nil {
			return err
		}

		// Create geo_cache table
		_, err = db.NewCreateTable().
			Model((*geofence.GeoCache)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create index for geo_cache
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_geo_cache_expires_at ON geo_cache(expires_at)")
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback migration - drop tables in reverse order
		_, err := db.NewDropTable().
			Model((*geofence.GeoCache)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*geofence.GeofenceViolation)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*geofence.TrustedLocation)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*geofence.TravelAlert)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*geofence.LocationEvent)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*geofence.GeofenceRule)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
