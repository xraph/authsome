package geofence

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
)

// getNotificationAdapter retrieves the notification adapter from service registry
func (s *Service) getNotificationAdapter() interface{} {
	if s.authInst == nil {
		return nil
	}

	// Type assert to access service registry
	authInst, ok := s.authInst.(interface {
		GetServiceRegistry() interface {
			Get(string) (interface{}, bool)
		}
	})
	if !ok {
		return nil
	}

	registry := authInst.GetServiceRegistry()
	if registry == nil {
		return nil
	}

	adapter, exists := registry.Get("notification.adapter")
	if !exists {
		return nil
	}

	return adapter
}

// getUserService retrieves user service from registry
func (s *Service) getUserService() interface{} {
	if s.authInst == nil {
		return nil
	}

	authInst, ok := s.authInst.(interface {
		GetServiceRegistry() interface {
			UserService() interface {
				FindByID(context.Context, xid.ID) (interface {
					GetID() xid.ID
					GetName() string
					GetEmail() string
				}, error)
			}
		}
	})
	if !ok {
		return nil
	}

	return authInst.GetServiceRegistry().UserService()
}

// notifyNewLocation sends new location login notification
func (s *Service) notifyNewLocation(ctx context.Context, userID xid.ID, appID xid.ID, newLoc *GeoData, oldLoc *GeoData, distance float64) error {
	// Type assert to notification adapter
	adapter := s.getNotificationAdapter()
	if adapter == nil {
		return nil
	}

	notifAdapter, ok := adapter.(interface {
		SendNewLocationLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, location, timestamp, ipAddress string) error
	})
	if !ok {
		return nil
	}

	// Get user details
	userSvc := s.getUserService()
	if userSvc == nil {
		return nil
	}

	userService, ok := userSvc.(interface {
		FindByID(context.Context, xid.ID) (interface {
			GetName() string
			GetEmail() string
		}, error)
	})
	if !ok {
		return nil
	}

	user, err := userService.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	userName := user.GetName()
	if userName == "" {
		userName = user.GetEmail()
	}

	// Build location string with context
	location := fmt.Sprintf("%s, %s", newLoc.City, newLoc.Country)
	if oldLoc != nil {
		location = fmt.Sprintf("%s, %s (%.0f km from previous location: %s, %s)",
			newLoc.City, newLoc.Country,
			distance,
			oldLoc.City, oldLoc.Country)
	}

	return notifAdapter.SendNewLocationLogin(
		ctx,
		appID,
		user.GetEmail(),
		userName,
		location,
		time.Now().Format(time.RFC3339),
		newLoc.IPAddress,
	)
}

// notifySuspiciousLogin sends suspicious login notification
func (s *Service) notifySuspiciousLogin(ctx context.Context, userID xid.ID, appID xid.ID, reason string, loc *GeoData) error {
	adapter := s.getNotificationAdapter()
	if adapter == nil {
		return nil
	}

	notifAdapter, ok := adapter.(interface {
		SendSuspiciousLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, reason, location, timestamp, ipAddress string) error
	})
	if !ok {
		return nil
	}

	userSvc := s.getUserService()
	if userSvc == nil {
		return nil
	}

	userService, ok := userSvc.(interface {
		FindByID(context.Context, xid.ID) (interface {
			GetName() string
			GetEmail() string
		}, error)
	})
	if !ok {
		return nil
	}

	user, err := userService.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	userName := user.GetName()
	if userName == "" {
		userName = user.GetEmail()
	}

	location := fmt.Sprintf("%s, %s", loc.City, loc.Country)

	return notifAdapter.SendSuspiciousLogin(
		ctx,
		appID,
		user.GetEmail(),
		userName,
		reason,
		location,
		time.Now().Format(time.RFC3339),
		loc.IPAddress,
	)
}
