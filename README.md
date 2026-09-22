# oteljobfx

[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/oteljobfx.svg)](https://pkg.go.dev/github.com/uchaloop/oteljobfx)
[![CI](https://github.com/uchaloop/oteljobfx/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/oteljobfx/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/uchaloop/oteljobfx?include_prereleases)](https://github.com/uchaloop/oteljobfx/releases)

OpenTelemetry metrics for `job` through Uber Fx. Requires Go 1.27 or later.

```sh
go get github.com/uchaloop/oteljobfx@v0.1.0
```

## Wiring

| Input from DI | Output to DI |
| --- | --- |
| `metric.MeterProvider` | `job.Handler` |

`Module()` constructs the handler on the `github.com/uchaloop/oteljob`
instrumentation scope. It does not execute work, configure exporters, or install
provider lifecycle hooks. Provide the interface `metric.MeterProvider`, rather
than only its concrete SDK type.

Application wiring (configuration, work and provider are application-owned):

```go
var runner *job.Runner
var observer job.Handler

app := fx.New(
    fx.Provide(provideMeterProvider),
    fx.Supply(jobConfig),
    fx.Provide(provideWork), // returns job.Func
    jobfx.Module(),
    oteljobfx.Module(),
    fx.Populate(&runner, &observer),
)
// After successful app.Start(startCtx):
result := runner.Run(workCtx)
observer.Handle(observationCtx, result)
// Then app.Stop with a fresh, bounded cleanup context.
```

`jobfx` constructs the runner without executing it. Observe each result exactly
once; Runner.Run does not call the metrics handler. Follow the
[jobfx lifecycle example](https://github.com/uchaloop/jobfx) for signals and exit handling.

## Environment configuration

Read telemetry identity from the standard OTel environment variables when the
application constructs its MeterProvider. The adapters consume that provider;
they do not read environment variables themselves. No separate configuration
library is needed for these values.

| Environment variable | Value |
| --- | --- |
| `OTEL_SERVICE_NAME` | Set to a stable service name, such as `daemon-efiro`. |
| `OTEL_RESOURCE_ATTRIBUTES` | Comma-separated `key=value` Resource attributes, as below. |

Recommended attributes inside `OTEL_RESOURCE_ATTRIBUTES`:

| Attribute | When to set it |
| --- | --- |
| `deployment.environment.name` | Local, staging or production environment. |
| `service.instance.id` | A unique identity for each concurrently running instance. |
| `k8s.namespace.name` | Kubernetes namespace; omit outside Kubernetes. |
| `k8s.cluster.name` | Kubernetes cluster name; omit outside Kubernetes. |

These are application identity conventions, not required configuration fields
of the handler. `OTEL_SERVICE_NAME` takes precedence over `service.name` inside
`OTEL_RESOURCE_ATTRIBUTES`. Set identity before constructing the provider;
changing the environment afterward does not refresh its Resource.

Local example (use a distinct instance ID for each concurrent process):

```sh
export OTEL_SERVICE_NAME=worker
export OTEL_RESOURCE_ATTRIBUTES='deployment.environment.name=local,service.instance.id=worker-local-1'
```

In Kubernetes, obtain the namespace (`metadata.namespace`) and Pod identity
(`metadata.uid`) through the
[Downward API](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/).
Supply the service name, deployment environment and cluster name through your
deployment configuration. To compose `OTEL_RESOURCE_ATTRIBUTES` from these values,
follow Kubernetes' official guide to
[dependent environment variables](https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/).

`resource.WithFromEnv()` reads these values. It does not configure an exporter.
Select a reader/exporter separately; exporter-specific environment variables
(such as an OTLP endpoint) only apply when that exporter is constructed.
Resource attributes also do not automatically become Prometheus labels: configure
selected Resource-to-label promotion in your exporter or Collector and preserve
distinct replica identity.

## Provider lifecycle

A minimal provider constructor with an application-configured reader:

```go
import (
    "context"

    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    "go.uber.org/fx"
)

func provideMeterProvider(lc fx.Lifecycle, reader sdkmetric.Reader) (metric.MeterProvider, error) {
    res, err := resource.New(context.Background(), resource.WithFromEnv())
    if err != nil {
        return nil, err
    }

    provider := sdkmetric.NewMeterProvider(
        sdkmetric.WithResource(res),
        sdkmetric.WithReader(reader),
    )
    lc.Append(fx.Hook{OnStop: provider.Shutdown})
    return provider, nil
}
```

Provide the reader as `sdkmetric.Reader`. Configure Resource, Views and the
exporter in your application. Start dependencies before work and stop the app
only after work and observation finish. Give shutdown a fresh, bounded context
so canceled work does not prevent export. A ManualReader is useful for tests;
it does not export metrics to a backend.

For short-lived jobs, choose a suitable delivery pipeline explicitly. This module
does not publish Pushgateway snapshots or change the underlying metric semantics.

If another provider already supplies the same Handler interface, combine handlers
in your application using named providers or MultiHandler instead of registering
two unnamed providers for that interface.

## Acknowledgements

Thanks to the authors and maintainers of
[Uber Fx](https://github.com/uber-go/fx) and
[OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go).
