package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/internal/repository/notification"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type NotificationInbox interface {
	List(context.Context, int32, string, int32) (notification.Page, error)
	MarkRead(context.Context, int32, string, int32) (bool, error)
}

type staffNotifications struct{ inbox NotificationInbox }

func NewStaffNotifications(inbox NotificationInbox) *staffNotifications {
	return &staffNotifications{inbox: inbox}
}

func notificationEmployee(session dto.Session) (int32, error) {
	if !policy.Has(session, policy.PermBookings) ||
		(session.Role != policy.RolePA && session.Role != policy.RoleProtocol) {
		return 0, failure.New(403, "forbidden")
	}
	return validate.ID(session.ID)
}

func (h *staffNotifications) List(c fiber.Ctx) error {
	session := restmiddleware.Session(c)
	employee, err := notificationEmployee(session)
	if err != nil {
		return err
	}
	query, err := Query(c, "before")
	if err != nil {
		return err
	}
	cursor, err := query.String("before")
	if err != nil {
		return err
	}
	var before int32
	if cursor != "" {
		before, err = validate.ID(cursor)
		if err != nil {
			return invalidQuery()
		}
	}
	page, err := h.inbox.List(c.Context(), employee, session.Role, before)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(page)
}

func (h *staffNotifications) MarkRead(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	session := restmiddleware.Session(c)
	employee, err := notificationEmployee(session)
	if err != nil {
		return err
	}
	event, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	updated, err := h.inbox.MarkRead(c.Context(), employee, session.Role, event)
	if err != nil {
		return err
	}
	if !updated {
		return failure.New(404, "not_found")
	}
	c.Set("Cache-Control", "no-store")
	return c.Status(204).Send(nil)
}
