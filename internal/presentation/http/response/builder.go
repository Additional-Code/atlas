package response

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Additional-Code/atlas/pkg/errorbank"
)

// Builder helps construct consistent HTTP responses.
type Builder struct {
	ctx    echo.Context
	status int
	data   any
	err    error
	meta   map[string]any
}

// New instantiates a Builder for the provided request context.
func New(ctx echo.Context) *Builder {
	return &Builder{ctx: ctx, status: http.StatusOK}
}

// WithStatus overrides the response status code.
func (b *Builder) WithStatus(status int) *Builder {
	if status > 0 {
		b.status = status
	}
	return b
}

// WithData attaches a success payload.
func (b *Builder) WithData(data any) *Builder {
	b.data = data
	return b
}

// WithError records an error to be rendered.
func (b *Builder) WithError(err error) *Builder {
	b.err = err
	return b
}

// WithMeta appends auxiliary metadata to the response.
func (b *Builder) WithMeta(key string, value any) *Builder {
	if key == "" {
		return b
	}
	if b.meta == nil {
		b.meta = make(map[string]any)
	}
	b.meta[key] = value
	return b
}

// Build finalises and emits the HTTP response.
func (b *Builder) Build() error {
	if b.err != nil {
		return b.buildError()
	}
	return b.buildSuccess()
}

func (b *Builder) buildSuccess() error {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	payload := struct {
		Success bool           `json:"success"`
		Data    any            `json:"data,omitempty"`
		Meta    map[string]any `json:"meta,omitempty"`
	}{
		Success: true,
		Data:    b.data,
		Meta:    b.meta,
	}
	return b.ctx.JSON(b.status, payload)
}

func (b *Builder) buildError() error {
	appErr := errorbank.From(b.err)
	status := b.status
	if status < 400 {
		status = appErr.StatusCode()
	}
	payload := struct {
		Success bool `json:"success"`
		Error   struct {
			Kind    string         `json:"kind"`
			Message string         `json:"message"`
			Details map[string]any `json:"details,omitempty"`
		} `json:"error"`
		Meta map[string]any `json:"meta,omitempty"`
	}{
		Success: false,
		Meta:    b.meta,
	}
	payload.Error.Kind = string(appErr.Kind())
	payload.Error.Message = appErr.Message()
	payload.Error.Details = appErr.Details()

	return b.ctx.JSON(status, payload)
}
