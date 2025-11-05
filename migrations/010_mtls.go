package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/enterprise/mtls"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create mtls_certificates table
		_, err := db.NewCreateTable().
			Model((*mtls.Certificate)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create mtls_trust_anchors table
		_, err = db.NewCreateTable().
			Model((*mtls.TrustAnchor)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create mtls_crls table
		_, err = db.NewCreateTable().
			Model((*mtls.CertificateRevocationList)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create mtls_ocsp_responses table
		_, err = db.NewCreateTable().
			Model((*mtls.OCSPResponse)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create mtls_auth_events table
		_, err = db.NewCreateTable().
			Model((*mtls.CertificateAuthEvent)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create mtls_policies table
		_, err = db.NewCreateTable().
			Model((*mtls.CertificatePolicy)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for performance

		// Certificates indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_fingerprint").
			Column("fingerprint").
			Unique().
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_serial").
			Column("serial_number").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_org_id").
			Column("organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_user_id").
			Column("user_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_device_id").
			Column("device_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_status").
			Column("status").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.Certificate)(nil)).
			Index("idx_mtls_certs_not_after").
			Column("not_after").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Trust anchors indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.TrustAnchor)(nil)).
			Index("idx_mtls_anchors_fingerprint").
			Column("fingerprint").
			Unique().
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.TrustAnchor)(nil)).
			Index("idx_mtls_anchors_org_id").
			Column("organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// CRL indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.CertificateRevocationList)(nil)).
			Index("idx_mtls_crls_issuer").
			Column("issuer").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.CertificateRevocationList)(nil)).
			Index("idx_mtls_crls_trust_anchor").
			Column("trust_anchor_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// OCSP indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.OCSPResponse)(nil)).
			Index("idx_mtls_ocsp_cert_id").
			Column("certificate_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.OCSPResponse)(nil)).
			Index("idx_mtls_ocsp_expires").
			Column("expires_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Auth events indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.CertificateAuthEvent)(nil)).
			Index("idx_mtls_events_cert_id").
			Column("certificate_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.CertificateAuthEvent)(nil)).
			Index("idx_mtls_events_org_id").
			Column("organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.CertificateAuthEvent)(nil)).
			Index("idx_mtls_events_created").
			Column("created_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Policies indexes
		_, err = db.NewCreateIndex().
			Model((*mtls.CertificatePolicy)(nil)).
			Index("idx_mtls_policies_org_id").
			Column("organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*mtls.CertificatePolicy)(nil)).
			Index("idx_mtls_policies_default").
			Column("organization_id", "is_default").
			Where("is_default = true").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop all mTLS tables
		tables := []string{
			"mtls_policies",
			"mtls_auth_events",
			"mtls_ocsp_responses",
			"mtls_crls",
			"mtls_trust_anchors",
			"mtls_certificates",
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
