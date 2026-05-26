// handlers_devices.go: Phase C.6 — Devices dashboard.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type DeviceSummary struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	Browser    string `json:"browser,omitempty"`
	OS         string `json:"os,omitempty"`
	IPAddress  string `json:"ipAddress,omitempty"`
	Trusted    bool   `json:"trusted"`
	LastSeenAt string `json:"lastSeenAt"`
	CreatedAt  string `json:"createdAt"`
}

// DeviceListInput optionally narrows the list to a single user; this is
// how the rich user-detail page's embedded devices table binds
// (params: { userId: { from: route.id } }). When unset the handler
// returns the global recent-devices window.
type DeviceListInput struct {
	UserID string `json:"userId,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type DeviceListResponse struct {
	Devices []DeviceSummary `json:"devices"`
}
type GetDeviceInput struct {
	ID string `json:"id"`
}
type TrustDeviceInput struct {
	ID string `json:"id"`
}
type DeleteDeviceInput struct {
	ID string `json:"id"`
}

func devicesListHandler(deps Deps) func(ctx context.Context, in DeviceListInput, _ contract.Principal) (DeviceListResponse, error) {
	return func(ctx context.Context, in DeviceListInput, _ contract.Principal) (DeviceListResponse, error) {
		if deps.Engine == nil {
			return DeviceListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		var (
			list []*device.Device
			err  error
		)
		if uidStr := strings.TrimSpace(in.UserID); uidStr != "" {
			uid, pErr := parseUserID(uidStr)
			if pErr != nil {
				return DeviceListResponse{}, pErr
			}
			list, err = deps.Engine.ListUserDevices(ctx, uid)
		} else {
			limit := in.Limit
			if limit <= 0 {
				limit = 100
			}
			list, err = deps.Engine.ListAllDevices(ctx, limit)
		}
		if err != nil {
			return DeviceListResponse{}, mapEngineError(err)
		}
		out := DeviceListResponse{Devices: make([]DeviceSummary, 0, len(list))}
		for _, d := range list {
			out.Devices = append(out.Devices, projectDevice(d))
		}
		return out, nil
	}
}

// devicesDetailHandler reads a single device via the engine's combined
// store. engine.Store() returns a Store interface that embeds
// device.Store, so GetDevice is directly callable — no engine wrapper
// needed.
func devicesDetailHandler(deps Deps) func(ctx context.Context, in GetDeviceInput, _ contract.Principal) (DeviceSummary, error) {
	return func(ctx context.Context, in GetDeviceInput, _ contract.Principal) (DeviceSummary, error) {
		if deps.Engine == nil {
			return DeviceSummary{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		did, err := parseDeviceID(in.ID)
		if err != nil {
			return DeviceSummary{}, err
		}
		d, err := deps.Engine.Store().GetDevice(ctx, did)
		if err != nil {
			return DeviceSummary{}, mapEngineError(err)
		}
		return projectDevice(d), nil
	}
}

func devicesTrustHandler(deps Deps) func(ctx context.Context, in TrustDeviceInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in TrustDeviceInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		did, err := parseDeviceID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if _, err := deps.Engine.TrustDevice(ctx, did); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: did.String()}, nil
	}
}

func devicesDeleteHandler(deps Deps) func(ctx context.Context, in DeleteDeviceInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteDeviceInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		did, err := parseDeviceID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.DeleteDevice(ctx, did); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: did.String()}, nil
	}
}

func projectDevice(d *device.Device) DeviceSummary {
	if d == nil {
		return DeviceSummary{}
	}
	return DeviceSummary{
		ID: d.ID.String(), UserID: d.UserID.String(),
		Name: d.Name, Type: d.Type, Browser: d.Browser, OS: d.OS,
		IPAddress: d.IPAddress, Trusted: d.Trusted,
		LastSeenAt: d.LastSeenAt.UTC().Format(time.RFC3339),
		CreatedAt:  d.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func parseDeviceID(s string) (id.DeviceID, error) {
	if strings.TrimSpace(s) == "" {
		return id.DeviceID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	did, err := id.ParseDeviceID(s)
	if err != nil {
		return id.DeviceID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid device id: " + err.Error()}
	}
	return did, nil
}
