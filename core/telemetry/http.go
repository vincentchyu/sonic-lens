package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHTTPClient 为出站 HTTP client 挂上标准 OpenTelemetry transport，统一补齐外呼链路。
func WrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}

	cloned := *client
	cloned.Transport = otelhttp.NewTransport(
		resolveBaseRoundTripper(client.Transport),
		otelhttp.WithTracerProvider(GetTracerProvider()),
		otelhttp.WithMeterProvider(GetMeterProvider()),
	)
	return &cloned
}

func resolveBaseRoundTripper(transport http.RoundTripper) http.RoundTripper {
	if transport != nil {
		return transport
	}
	return http.DefaultTransport
}
