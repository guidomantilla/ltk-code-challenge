package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type StopFn func(ctx context.Context)

func noopStop(ctx context.Context) {}

func Trace(ctx context.Context) (StopFn, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return noopStop, fmt.Errorf("failed to create the OTLP exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	)

	otel.SetTracerProvider(tp)

	stopFn := func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Str("stage", "shut down").Str("component", "otel tracer").Msg("error shutting down tracer")
		}
	}

	return stopFn, nil
}

func Profile(ctx context.Context) (StopFn, error) {
	err := runtimemetrics.Start(
		runtimemetrics.WithMinimumReadMemStatsInterval(1 * time.Second),
	)
	if err != nil {
		return noopStop, err
	}

	// No-op stop: runtime metrics no exponen Stop()
	return noopStop, nil
}

func Measure(ctx context.Context) (StopFn, error) {
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint("localhost:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return noopStop, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(5*time.Second)),
		),
	)
	otel.SetMeterProvider(mp)

	stopFn := func(ctx context.Context) {
		err := mp.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Str("stage", "shut down").Str("component", "otel meter").Msg("error shutting down meter")
		}
	}

	return stopFn, nil
}

func Logger(ctx context.Context) (StopFn, error) {
	exp, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint("localhost:4317"),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return noopStop, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exp)), // for dev
		// sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),  // for prod
	)

	global.SetLoggerProvider(lp)

	serviceName := "ltk-code-challenge" // same name as main TODO
	serviceVersion := "1.0"             // same version as main TODO

	// Bridge zerolog -> OTel Logs (still prints to stdout; additionally exports via OTLP to the collector)
	hook := &myHook{
		logger:         global.GetLoggerProvider().Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
	log.Logger = log.Logger.Hook(hook)
	stopFn := func(ctx context.Context) {
		err := lp.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Str("stage", "shut down").Str("component", "otel logger").Msg("error shutting down logger")
		}
	}

	return stopFn, nil
}

/*

 */

func CreateDatabaseConnectionPool(ctx context.Context) (*pgxpool.Pool, StopFn, error) {
	//nolint:nosprintfhostport
	cfg, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"), viper.GetString("DB_NAME")))
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to parse database connection string: %v", err))
		return nil, noopStop, fmt.Errorf("failed to parse database connection string: %w", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to connect to database: %v", err))
		return nil, noopStop, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Unable to ping to database: %v", err))
		return nil, noopStop, fmt.Errorf("failed to ping to database: %w", err)
	}

	stopFn := func(ctx context.Context) {
		pool.Close()
	}

	return pool, stopFn, nil
}

/*

 */

type myHook struct {
	logger         otelog.Logger
	serviceName    string
	serviceVersion string
}

func (h *myHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {

	b, ok := h.getBuffer(e)
	if !ok {
		return
	}

	var m map[string]any
	err := json.Unmarshal(b, &m)
	if err != nil {
		return
	}

	var rec otelog.Record
	ts := h.extractTimestamp(m)
	sev, sevText := h.zerologLevelToOTel(level)

	rec.SetTimestamp(ts)
	rec.SetSeverity(sev)
	rec.SetSeverityText(sevText)
	rec.SetBody(otelog.StringValue(msg))

	rec.AddAttributes(h.mapToAttrs(m)...)

	h.logger.Emit(e.GetCtx(), rec)
}

func (h *myHook) zerologLevelToOTel(level zerolog.Level) (otelog.Severity, string) {
	switch level {
	case zerolog.TraceLevel:
		return otelog.SeverityTrace, "TRACE"
	case zerolog.DebugLevel:
		return otelog.SeverityDebug, "DEBUG"
	case zerolog.InfoLevel:
		return otelog.SeverityInfo, "INFO"
	case zerolog.WarnLevel:
		return otelog.SeverityWarn, "WARN"
	case zerolog.ErrorLevel:
		return otelog.SeverityError, "ERROR"
	case zerolog.FatalLevel:
		return otelog.SeverityFatal, "FATAL"
	case zerolog.PanicLevel:
		return otelog.SeverityFatal4, "FATAL"
	default:
		return otelog.SeverityInfo, "INFO"
	}
}

func (h *myHook) getBuffer(e *zerolog.Event) ([]byte, bool) {
	if e == nil {
		return nil, false
	}

	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, false
	}

	ev := v.Elem()
	f := ev.FieldByName("buf")
	if !f.IsValid() || f.Kind() != reflect.Slice || f.Type().Elem().Kind() != reflect.Uint8 {
		return nil, false
	}

	b := append([]byte(nil), f.Bytes()...)
	if len(b) == 0 {
		return nil, false
	}

	if b[len(b)-1] != '}' {
		b = append(b, '}')
	}
	return b, true
}

func (h *myHook) mapToAttrs(m map[string]any) []otelog.KeyValue {
	kvs := make([]otelog.KeyValue, 0, len(m))
	for k, v := range m {
		switch x := v.(type) {
		case string:
			kvs = append(kvs, otelog.String(k, x))
		case bool:
			kvs = append(kvs, otelog.Bool(k, x))
		case float64: // json numbers
			if x == float64(int64(x)) {
				kvs = append(kvs, otelog.Int64(k, int64(x)))
			} else {
				kvs = append(kvs, otelog.Float64(k, x))
			}
		default:
			kvs = append(kvs, otelog.String(k, fmt.Sprintf("%v", x)))
		}
	}
	return kvs
}

func (h *myHook) extractTimestamp(m map[string]any) time.Time {
	v, ok := m["time"]
	if !ok {
		return time.Now()
	}

	s, ok := v.(string)
	if !ok {
		return time.Now()
	}

	ts, err := time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return ts
	}

	// Fallback RFC3339
	ts, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return ts
	}

	return time.Now()
}
