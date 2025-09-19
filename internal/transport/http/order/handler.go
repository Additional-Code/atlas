package order

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Additional-Code/atlas/internal/dto"
	"github.com/Additional-Code/atlas/internal/entity"
	"github.com/Additional-Code/atlas/internal/presentation/http/response"
	service "github.com/Additional-Code/atlas/internal/service/order"
	"github.com/Additional-Code/atlas/pkg/errorbank"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var httpTracer = otel.Tracer("github.com/Additional-Code/atlas/transport/http/order")

// Handler exposes order endpoints over HTTP.
type Handler struct {
	svc *service.Service
}

// NewHandler constructs an order Handler.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// Register routes with provided Echo group.
func Register(e *echo.Echo, h *Handler) {
	g := e.Group("/orders")
	g.GET("/:id", h.getByID)
	g.POST("", h.create)
}

func (h *Handler) getByID(c echo.Context) error {
	b := response.New(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return b.WithError(errorbank.BadRequest("invalid id", errorbank.WithCause(err))).Build()
	}

	ctx, span := httpTracer.Start(c.Request().Context(), "orders.getByID", trace.WithAttributes(attribute.Int64("order.id", id)))
	defer span.End()

	order, err := h.svc.Get(ctx, id)
	if err != nil {
		return b.WithError(err).Build()
	}

	return b.WithData(toDTO(order)).Build()
}

func (h *Handler) create(c echo.Context) error {
	b := response.New(c)

	var payload struct {
		Number string `json:"number"`
		Status string `json:"status"`
	}
	if err := c.Bind(&payload); err != nil {
		return b.WithError(errorbank.BadRequest("invalid payload", errorbank.WithCause(err))).Build()
	}
	if payload.Number == "" || payload.Status == "" {
		return b.WithError(errorbank.BadRequest("number and status are required")).Build()
	}

	order := &entity.Order{
		Number: payload.Number,
		Status: payload.Status,
	}

	ctx, span := httpTracer.Start(c.Request().Context(), "orders.create")
	span.SetAttributes(
		attribute.String("order.number", order.Number),
	)
	defer span.End()

	if err := h.svc.Create(ctx, order); err != nil {
		return b.WithError(err).Build()
	}

	return b.WithStatus(http.StatusCreated).WithData(toDTO(order)).Build()
}

func toDTO(order *entity.Order) dto.OrderResponse {
	return dto.OrderResponse{
		ID:        order.ID,
		Number:    order.Number,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
		UpdatedAt: order.UpdatedAt,
	}
}
