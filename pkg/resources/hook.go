package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog"
	otelog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type ZerologHook struct {
	logger         otelog.Logger
	serviceName    string
	serviceVersion string
}

func NewZerologHook(serviceName string, serviceVersion string) *ZerologHook {
	return &ZerologHook{
		logger:         global.GetLoggerProvider().Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

func (h *ZerologHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
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

func (h *ZerologHook) zerologLevelToOTel(level zerolog.Level) (otelog.Severity, string) {
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

func (h *ZerologHook) getBuffer(e *zerolog.Event) ([]byte, bool) {
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

func (h *ZerologHook) mapToAttrs(m map[string]any) []otelog.KeyValue {
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

func (h *ZerologHook) extractTimestamp(m map[string]any) time.Time {
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
