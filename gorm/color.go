package gorm

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	gutils "github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
	"github.com/Laisky/zap/zapcore"
)

type gormLoggerItf interface {
	Debug(string, ...zap.Field)
	Info(string, ...zap.Field)
	Error(string, ...zap.Field)
}

// GormLogger colored logger for gorm
type GormLogger struct {
	logger    gormLoggerItf
	formatter func(...interface{}) []interface{}
}

// NewGormLogger new gorm sql logger
func NewGormLogger(formatter func(...interface{}) []interface{}, logger gormLoggerItf) *GormLogger {
	return &GormLogger{
		logger:    logger,
		formatter: formatter,
	}
}

// Print print sql logger
func (l *GormLogger) Print(vs ...interface{}) {
	fvs := l.formatter(vs...)
	var fields []zapcore.Field
	for i, v := range vs {
		switch i {
		case 0:
			fields = append(fields, zap.Any("type", v))
		case 1:
			fields = append(fields, zap.Any("caller", v))
		case 2:
			switch v := v.(type) {
			case time.Duration:
				fields = append(fields, zap.Int("ms", int(v/time.Millisecond)))
			}
		case 3:
			if len(fvs) < 4 {
				fields = append(fields, zap.Any("sql", v))
			}
		case 4:
			if len(fvs) < 4 {
				fields = append(fields, zap.Any("args", v))
			}
		case 5:
			fields = append(fields, zap.Any("affected", v))
		default:
			fields = append(fields, zap.Any(strconv.FormatInt(int64(i), 10), v))
		}
	}

	if len(fvs) < 4 {
		l.logger.Debug("", fields...)
		return
	}

	var msg string
	switch fvs[3].(type) {
	case string:
		msg = fvs[3].(string)
	case []byte:
		msg = string(fvs[3].([]byte))
	default:
		msg = fmt.Sprint(fvs[3])
	}

	// ignore some logs
	if strings.Contains(msg, "/*disable_log*/") {
		return
	}

	switch strings.TrimSpace(strings.ToLower(strings.SplitN(msg, " ", 2)[0])) {
	case "drop", "delete":
		l.logger.Info(gutils.Color(gutils.ANSIColorFgMagenta, msg), fields...)
	case "insert":
		l.logger.Info(gutils.Color(gutils.ANSIColorFgGreen, msg), fields...)
	case "update":
		l.logger.Info(gutils.Color(gutils.ANSIColorFgYellow, msg), fields...)
	case "select":
		l.logger.Debug(gutils.Color(gutils.ANSIColorFgCyan, msg), fields...)
	case "error":
		l.logger.Error(gutils.Color(gutils.ANSIColorFgHiRed, msg), fields...)
	default:
		l.logger.Info(gutils.Color(gutils.ANSIColorFgBlue, msg), fields...)
	}
}
