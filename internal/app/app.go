package app

import (
	"go.uber.org/fx"

	"github.com/Additional-Code/atlas/internal/cache"
	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/database"
	"github.com/Additional-Code/atlas/internal/logger"
	"github.com/Additional-Code/atlas/internal/messaging"
	"github.com/Additional-Code/atlas/internal/observability"
	repositoryorder "github.com/Additional-Code/atlas/internal/repository/order"
	httpserver "github.com/Additional-Code/atlas/internal/server/http"
	serviceorder "github.com/Additional-Code/atlas/internal/service/order"
	transporthttp "github.com/Additional-Code/atlas/internal/transport/http"
	"github.com/Additional-Code/atlas/internal/worker"
	workerorder "github.com/Additional-Code/atlas/internal/worker/order"
)

// Core provides the foundational modules shared across executables.
var Core = fx.Options(
	config.Module,
	cache.Module,
	database.Module,
	logger.Module,
	messaging.Module,
	observability.Module,
	repositoryorder.Module,
	serviceorder.Module,
)

// HTTP wires the HTTP transport on top of the core modules.
var HTTP = fx.Options(
	Core,
	httpserver.Module,
	transporthttp.Module,
)

// Worker exposes background worker processing.
var Worker = fx.Options(
	Core,
	worker.Module,
	workerorder.Module,
)

// Module is the default application wiring (HTTP only).
var Module = HTTP
