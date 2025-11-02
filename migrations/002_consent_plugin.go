package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/enterprise/consent"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create consent_records table
		_, err := db.NewCreateTable().
			Model((*consent.ConsentRecord)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create consent_policies table
		_, err = db.NewCreateTable().
			Model((*consent.ConsentPolicy)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create data_processing_agreements table
		_, err = db.NewCreateTable().
			Model((*consent.DataProcessingAgreement)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create consent_audit_logs table
		_, err = db.NewCreateTable().
			Model((*consent.ConsentAuditLog)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create cookie_consents table
		_, err = db.NewCreateTable().
			Model((*consent.CookieConsent)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create data_export_requests table
		_, err = db.NewCreateTable().
			Model((*consent.DataExportRequest)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create data_deletion_requests table
		_, err = db.NewCreateTable().
			Model((*consent.DataDeletionRequest)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create privacy_settings table
		_, err = db.NewCreateTable().
			Model((*consent.PrivacySettings)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for consent_records
		_, err = db.NewCreateIndex().
			Model((*consent.ConsentRecord)(nil)).
			Index("idx_consent_records_user_org").
			Column("user_id", "organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.ConsentRecord)(nil)).
			Index("idx_consent_records_type").
			Column("consent_type").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.ConsentRecord)(nil)).
			Index("idx_consent_records_expires_at").
			Column("expires_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for consent_policies
		_, err = db.NewCreateIndex().
			Model((*consent.ConsentPolicy)(nil)).
			Index("idx_consent_policies_org_type").
			Column("organization_id", "consent_type").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.ConsentPolicy)(nil)).
			Index("idx_consent_policies_active").
			Column("active").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for consent_audit_logs
		_, err = db.NewCreateIndex().
			Model((*consent.ConsentAuditLog)(nil)).
			Index("idx_consent_audit_user_org").
			Column("user_id", "organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.ConsentAuditLog)(nil)).
			Index("idx_consent_audit_consent_id").
			Column("consent_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.ConsentAuditLog)(nil)).
			Index("idx_consent_audit_created_at").
			Column("created_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for cookie_consents
		_, err = db.NewCreateIndex().
			Model((*consent.CookieConsent)(nil)).
			Index("idx_cookie_consents_user_org").
			Column("user_id", "organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.CookieConsent)(nil)).
			Index("idx_cookie_consents_session").
			Column("session_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for data_export_requests
		_, err = db.NewCreateIndex().
			Model((*consent.DataExportRequest)(nil)).
			Index("idx_export_requests_user_org").
			Column("user_id", "organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.DataExportRequest)(nil)).
			Index("idx_export_requests_status").
			Column("status").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.DataExportRequest)(nil)).
			Index("idx_export_requests_expires_at").
			Column("expires_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for data_deletion_requests
		_, err = db.NewCreateIndex().
			Model((*consent.DataDeletionRequest)(nil)).
			Index("idx_deletion_requests_user_org").
			Column("user_id", "organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.DataDeletionRequest)(nil)).
			Index("idx_deletion_requests_status").
			Column("status").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create index for privacy_settings
		_, err = db.NewCreateIndex().
			Model((*consent.PrivacySettings)(nil)).
			Index("idx_privacy_settings_org").
			Column("organization_id").
			Unique().
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for data_processing_agreements
		_, err = db.NewCreateIndex().
			Model((*consent.DataProcessingAgreement)(nil)).
			Index("idx_dpa_org_type").
			Column("organization_id", "agreement_type").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*consent.DataProcessingAgreement)(nil)).
			Index("idx_dpa_status").
			Column("status").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop all consent plugin tables
		tables := []string{
			"privacy_settings",
			"data_deletion_requests",
			"data_export_requests",
			"cookie_consents",
			"consent_audit_logs",
			"data_processing_agreements",
			"consent_policies",
			"consent_records",
		}

		for _, table := range tables {
			_, err := db.NewDropTable().
				Table(table).
				IfExists().
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

