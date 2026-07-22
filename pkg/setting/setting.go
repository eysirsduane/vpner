package setting

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
)

type App struct {
	Product      string
	RunMode      string
	HttpPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IpDbPath     string
	MappingFile  string
	JwtSecret    string
	JwtExpire    time.Duration
}

type Database struct {
	Type            string
	Host            string
	User            string
	Password        string
	DbName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Redis struct {
	Host     string
	Password string
}

const (
	AppModeDev  = "dev"
	AppModeProd = "prod"
)

var (
	Cfg            *ini.File
	AppConfig      = &App{}
	DatabaseConfig = &Database{}
	RedisConfig    = &Redis{}
	ChinaLocation  *time.Location
)

func init() {
	loadTimezone()

	var err error
	configPath := ""
	for _, path := range []string{"conf/app.ini", "../conf/app.ini", "../../conf/app.ini"} {
		Cfg, err = ini.Load(path)
		if err == nil {
			configPath = path
			break
		}
	}
	if err != nil {
		log.Fatalf("Fail to load 'conf/app.ini': %v", err)
	}
	loadLocalOverride(configPath)

	loadApp()
	loadDatabase()
	loadRedis()
}

func loadLocalOverride(configPath string) {
	localPath := filepath.Join(filepath.Dir(configPath), "app.local.ini")
	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("Fail to stat local config %q: %v", localPath, err)
	}

	localCfg, err := ini.Load(localPath)
	if err != nil {
		log.Fatalf("Fail to load local config %q: %v", localPath, err)
	}
	for _, localSec := range localCfg.Sections() {
		sec := Cfg.Section(localSec.Name())
		for _, key := range localSec.Keys() {
			sec.Key(key.Name()).SetValue(key.Value())
		}
	}
}

func loadTimezone() {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	ChinaLocation = location
	time.Local = location
}

func loadApp() {
	sec := Cfg.Section("app")
	AppConfig.Product = sec.Key("Product").MustString("just-vpn")
	AppConfig.RunMode = normalizeRunMode(sec.Key("RunMode").MustString(AppModeDev))
	AppConfig.HttpPort = sec.Key("HttpPort").MustInt(8080)
	AppConfig.ReadTimeout = time.Duration(sec.Key("ReadTimeout").MustInt(60)) * time.Second
	AppConfig.WriteTimeout = time.Duration(sec.Key("WriteTimeout").MustInt(60)) * time.Second
	AppConfig.IpDbPath = sec.Key("IpDbPath").MustString("conf/ip2region.db")
	AppConfig.MappingFile = sec.Key("MappingFile").MustString("conf/mapping.json")
	AppConfig.JwtSecret = sec.Key("JwtSecret").MustString("just-vpn-jwt-secret")
	AppConfig.JwtExpire = time.Duration(sec.Key("JwtExpireHours").MustInt(720)) * time.Hour
}

func normalizeRunMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case AppModeDev, AppModeProd:
		return mode
	default:
		log.Fatalf("Invalid app RunMode %q, only dev or prod are supported", mode)
		return AppModeDev
	}
}

func GinMode() string {
	if AppConfig.RunMode == AppModeProd {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

func loadDatabase() {
	sec := Cfg.Section("database")
	DatabaseConfig.Type = sec.Key("TYPE").MustString("mysql")
	DatabaseConfig.User = sec.Key("USER").MustString("just_vpn")
	DatabaseConfig.Password = sec.Key("PASSWORD").MustString("")
	DatabaseConfig.Host = sec.Key("HOST").MustString("127.0.0.1:3306")
	DatabaseConfig.DbName = sec.Key("NAME").MustString("just_vpn")
	DatabaseConfig.MaxOpenConns = sec.Key("MaxOpenConns").MustInt(100)
	DatabaseConfig.MaxIdleConns = sec.Key("MaxIdleConns").MustInt(20)
	DatabaseConfig.ConnMaxLifetime = time.Duration(sec.Key("ConnMaxLifetimeSeconds").MustInt(300)) * time.Second
}

func loadRedis() {
	sec := Cfg.Section("redis")
	RedisConfig.Host = sec.Key("HOST").MustString("127.0.0.1:6379")
	RedisConfig.Password = sec.Key("PASSWORD").MustString("")
}
