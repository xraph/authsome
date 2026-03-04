package api

import (
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Device route registration
// ──────────────────────────────────────────────────

func (a *API) registerDeviceRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base, forge.WithGroupTags("devices"))

	if err := g.GET("/devices", a.handleListDevices,
		forge.WithSummary("List devices"),
		forge.WithDescription("Returns all tracked devices for the authenticated user."),
		forge.WithOperationID("listDevices"),
		forge.WithResponseSchema(http.StatusOK, "Device list", DeviceListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/devices/:deviceId", a.handleGetDevice,
		forge.WithSummary("Get device"),
		forge.WithDescription("Returns details of a specific tracked device."),
		forge.WithOperationID("getDevice"),
		forge.WithResponseSchema(http.StatusOK, "Device details", device.Device{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/devices/:deviceId", a.handleDeleteDevice,
		forge.WithSummary("Delete device"),
		forge.WithDescription("Removes a tracked device."),
		forge.WithOperationID("deleteDevice"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.PATCH("/devices/:deviceId/trust", a.handleTrustDevice,
		forge.WithSummary("Trust device"),
		forge.WithDescription("Marks a device as trusted."),
		forge.WithOperationID("trustDevice"),
		forge.WithResponseSchema(http.StatusOK, "Trusted device", device.Device{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Device handlers
// ──────────────────────────────────────────────────

func (a *API) handleListDevices(ctx forge.Context, _ *ListDevicesRequest) (*DeviceListResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	devices, err := a.engine.ListUserDevices(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	if devices == nil {
		devices = []*device.Device{}
	}
	resp := &DeviceListResponse{Devices: devices}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleGetDevice(ctx forge.Context, _ *GetDeviceRequest) (*device.Device, error) {
	deviceID, err := id.ParseDeviceID(ctx.Param("deviceId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid device id: %v", err))
	}

	d, err := a.engine.GetDevice(ctx.Context(), deviceID)
	if err != nil {
		return nil, mapError(err)
	}

	return d, ctx.JSON(http.StatusOK, d)
}

func (a *API) handleDeleteDevice(ctx forge.Context, _ *DeleteDeviceRequest) (*StatusResponse, error) {
	deviceID, err := id.ParseDeviceID(ctx.Param("deviceId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid device id: %v", err))
	}

	if err := a.engine.DeleteDevice(ctx.Context(), deviceID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleTrustDevice(ctx forge.Context, _ *TrustDeviceRequest) (*device.Device, error) {
	deviceID, err := id.ParseDeviceID(ctx.Param("deviceId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid device id: %v", err))
	}

	d, err := a.engine.TrustDevice(ctx.Context(), deviceID)
	if err != nil {
		return nil, mapError(err)
	}

	return d, ctx.JSON(http.StatusOK, d)
}
