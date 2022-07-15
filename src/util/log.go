package util

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"time"
)

var Log = logrus.New()

func init() {
	Log.SetLevel(config.Logging.Level)
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	if config.Logging.Path != "" {
		logFile := path.Join(config.Logging.Path, config.Logging.Level.String()+".log")
		if err := CreateFileIfNotExist(logFile); err != nil {
			Log.Errorf("create log file error, file is: %s, message: %v", logFile, err)
			return
		}
		src, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			Log.Errorf("open log file error, file is: %s, message: %v", logFile, err)
			return
		}
		// 同时写文件和屏幕
		fileAndStdoutWriter := io.MultiWriter(src, os.Stdout)
		log.SetOutput(fileAndStdoutWriter)
	}
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()
		// 处理请求
		c.Next()
		// 结束时间
		endTime := time.Now()
		// 执行时间
		latencyTime := endTime.Sub(startTime)
		// 请求方式
		reqMethod := c.Request.Method
		// 请求路由
		reqUri := c.Request.RequestURI
		// 状态码
		statusCode := c.Writer.Status()
		// 请求IP
		clientIP := c.ClientIP()
		// 日志格式
		Log.Infof("| %3d | %13v | %15s | %s | %s",
			statusCode,
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
		)
	}
}
