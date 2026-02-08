package exporters

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/syslog"
	"net"
	"time"

	"github.com/xraph/authsome/core/audit"
)

// =============================================================================
// SYSLOG EXPORTER - Exports audit events via Syslog (RFC 5424)
// =============================================================================

// SyslogExporter exports audit events via Syslog
type SyslogExporter struct {
	config *SyslogConfig
	writer *syslog.Writer
	conn   net.Conn
}

// SyslogConfig contains Syslog configuration
type SyslogConfig struct {
	Network   string        `json:"network"`  // tcp, udp, tcp+tls
	Address   string        `json:"address"`  // host:port
	Tag       string        `json:"tag"`      // Syslog tag
	Facility  string        `json:"facility"` // Syslog facility (e.g., local0)
	Severity  string        `json:"severity"` // Default severity (e.g., info)
	UseTLS    bool          `json:"useTls"`   // Use TLS for TCP
	TLSConfig *tls.Config   `json:"-"`        // TLS configuration
	Timeout   time.Duration `json:"timeout"`
	Format    string        `json:"format"` // rfc5424 or rfc3164
}

// DefaultSyslogConfig returns default Syslog configuration
func DefaultSyslogConfig() *SyslogConfig {
	return &SyslogConfig{
		Network:  "tcp",
		Tag:      "authsome",
		Facility: "local0",
		Severity: "info",
		UseTLS:   false,
		Timeout:  10 * time.Second,
		Format:   "rfc5424",
	}
}

// NewSyslogExporter creates a new Syslog exporter
func NewSyslogExporter(config *SyslogConfig) (*SyslogExporter, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("syslog address is required")
	}

	// Parse facility
	facility, err := parseFacility(config.Facility)
	if err != nil {
		return nil, err
	}

	// Parse severity
	severity, err := parseSeverity(config.Severity)
	if err != nil {
		return nil, err
	}

	// Create syslog writer
	priority := facility | severity
	writer, err := syslog.Dial(
		config.Network,
		config.Address,
		priority,
		config.Tag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to syslog: %w", err)
	}

	return &SyslogExporter{
		config: config,
		writer: writer,
	}, nil
}

// Name returns the exporter name
func (e *SyslogExporter) Name() string {
	return "syslog"
}

// Export exports a batch of events to Syslog
func (e *SyslogExporter) Export(ctx context.Context, events []*audit.Event) error {
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		msg := e.formatEvent(event)

		// Send to syslog based on severity
		if err := e.writer.Info(msg); err != nil {
			return fmt.Errorf("failed to send syslog message: %w", err)
		}
	}

	return nil
}

// formatEvent formats an audit event as Syslog message
func (e *SyslogExporter) formatEvent(event *audit.Event) string {
	if e.config.Format == "rfc5424" {
		return e.formatRFC5424(event)
	}
	return e.formatRFC3164(event)
}

// formatRFC5424 formats event in RFC 5424 format
func (e *SyslogExporter) formatRFC5424(event *audit.Event) string {
	// RFC 5424: <priority>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID STRUCTURED-DATA MSG
	return fmt.Sprintf(
		"[authsome@audit id=\"%s\" appId=\"%s\" userId=\"%s\" action=\"%s\" resource=\"%s\" ip=\"%s\"] %s performed %s on %s",
		event.ID.String(),
		event.AppID.String(),
		e.getUserID(event),
		event.Action,
		event.Resource,
		event.IPAddress,
		e.getUserID(event),
		event.Action,
		event.Resource,
	)
}

// formatRFC3164 formats event in RFC 3164 format (legacy BSD syslog)
func (e *SyslogExporter) formatRFC3164(event *audit.Event) string {
	// RFC 3164: <priority>TIMESTAMP HOSTNAME TAG: MSG
	return fmt.Sprintf(
		"[%s] %s performed %s on %s from %s",
		event.ID.String(),
		e.getUserID(event),
		event.Action,
		event.Resource,
		event.IPAddress,
	)
}

func (e *SyslogExporter) getUserID(event *audit.Event) string {
	if event.UserID != nil {
		return event.UserID.String()
	}
	return "system"
}

// HealthCheck checks if Syslog server is reachable
func (e *SyslogExporter) HealthCheck(ctx context.Context) error {
	// Try to send a test message
	err := e.writer.Info("health_check")
	if err != nil {
		return fmt.Errorf("syslog health check failed: %w", err)
	}
	return nil
}

// Close closes the Syslog connection
func (e *SyslogExporter) Close() error {
	return e.writer.Close()
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// parseFacility parses syslog facility from string
func parseFacility(facility string) (syslog.Priority, error) {
	facilities := map[string]syslog.Priority{
		"kern":     syslog.LOG_KERN,
		"user":     syslog.LOG_USER,
		"mail":     syslog.LOG_MAIL,
		"daemon":   syslog.LOG_DAEMON,
		"auth":     syslog.LOG_AUTH,
		"syslog":   syslog.LOG_SYSLOG,
		"lpr":      syslog.LOG_LPR,
		"news":     syslog.LOG_NEWS,
		"uucp":     syslog.LOG_UUCP,
		"cron":     syslog.LOG_CRON,
		"authpriv": syslog.LOG_AUTHPRIV,
		"ftp":      syslog.LOG_FTP,
		"local0":   syslog.LOG_LOCAL0,
		"local1":   syslog.LOG_LOCAL1,
		"local2":   syslog.LOG_LOCAL2,
		"local3":   syslog.LOG_LOCAL3,
		"local4":   syslog.LOG_LOCAL4,
		"local5":   syslog.LOG_LOCAL5,
		"local6":   syslog.LOG_LOCAL6,
		"local7":   syslog.LOG_LOCAL7,
	}

	f, ok := facilities[facility]
	if !ok {
		return 0, fmt.Errorf("unknown syslog facility: %s", facility)
	}

	return f, nil
}

// parseSeverity parses syslog severity from string
func parseSeverity(severity string) (syslog.Priority, error) {
	severities := map[string]syslog.Priority{
		"emerg":   syslog.LOG_EMERG,
		"alert":   syslog.LOG_ALERT,
		"crit":    syslog.LOG_CRIT,
		"err":     syslog.LOG_ERR,
		"warning": syslog.LOG_WARNING,
		"notice":  syslog.LOG_NOTICE,
		"info":    syslog.LOG_INFO,
		"debug":   syslog.LOG_DEBUG,
	}

	s, ok := severities[severity]
	if !ok {
		return 0, fmt.Errorf("unknown syslog severity: %s", severity)
	}

	return s, nil
}
