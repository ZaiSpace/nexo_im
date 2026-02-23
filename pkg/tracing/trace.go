package tracing

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var Init = sync.OnceFunc(func() {
	//ctx := context.Background()
	//traceClient := otlptracegrpc.NewClient()
	//traceExp, _ := otlptrace.New(ctx, traceClient)
	//bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	//res, _ := resource.New(ctx)

	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		//sdktrace.WithSampler(sdktrace.AlwaysSample()),
		//sdktrace.WithResource(res),
		//sdktrace.WithSpanProcessor(bsp),
	))
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			//b3.New(),
			//ot.OT{},
			//jaeger.Jaeger{},
			//opencensus.Binary{},
			propagation.Baggage{},
			propagation.TraceContext{},
		),
	)
})
