package utils
import(
	"os"
	"github.com/sirupsen/logrus"
)
var Logger *logrus.Logger

func Init(logLevel string){
	Logger=logrus.New()

	Logger.SetOutput(os.Stdout)

	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2005-12-30 22:04:05",
	})
	level, err:=logrus.ParseLevel(logLevel)
	if err !=nil{
		Logger.SetLevel(logrus.InfoLevel)
		Logger.Warn("Invalid log level, defaulting to info")
	}else{
		Logger.SetLevel(level)
	}
}
func GetLogger() *logrus.Logger {
	if Logger == nil {
		Init("info")
	}
	return Logger
}
// Convenience functions
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}