// Package oteljobfx provides an OpenTelemetry job.Handler through Uber Fx.
// The application owns the metric.MeterProvider, exporter and provider shutdown.
package oteljobfx

import (
	"github.com/uchaloop/job"
	"github.com/uchaloop/oteljob"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/fx"
)

// Module provides a job.Handler using the container's metric.MeterProvider.
// It does not run work or configure the SDK, Resource, exporter or shutdown.
func Module() fx.Option {
	return fx.Module("oteljobfx", fx.Provide(
		func(provider metric.MeterProvider) (job.Handler, error) {
			return oteljob.MakeHandler(provider.Meter("github.com/uchaloop/oteljob"))
		},
	))
}
