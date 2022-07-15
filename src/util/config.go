package util

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	log        = logrus.New()
	file       []byte
	configPath string
)

var config = &Config{
	Auth:            make(map[string]interface{}),
	RepositoryStore: make(map[string]*Repository),
}

type Config struct {
	Listen          string        `yaml:"listen" default:"localhost"`
	Port            string        `yaml:"port" default:"8880"`
	Context         string        `yaml:"context" default:"maven"`
	LocalRepository string        `yaml:"localRepository" default:"."`
	User            []*User       `yaml:"user" default:"[{\"Name\":\"user\",\"Password\":\"password\"}]"`
	Repository      []*Repository `yaml:"repository" default:"[{\"Id\":\"public\",\"Name\":\"mirror\",\"Mirror\":[\"https://repo1.maven.org/maven2\",\"https://maven.aliyun.com/nexus/content/repositories/public\"]}]"`
	Logging         *Logging      `yaml:"logging" default:"{\"Path\":\"\",\"Level\":\"debug\"}"`
	Auth            map[string]interface{}
	RepositoryStore map[string]*Repository
}

type User struct {
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
}

type Repository struct {
	Id     string   `yaml:"id"`
	Name   string   `yaml:"name"`
	Mode   int      `yaml:"mode" default:"4"`
	Target string   `yaml:"target"`
	Cache  bool     `yaml:"cache" default:"false"`
	Mirror []string `yaml:"mirror"`
}

type Logging struct {
	Path  string       `yaml:"path" default:""`
	Level logrus.Level `yaml:"level" default:"debug"`
}

func LoadConfig() *Config {
	return config
}

func init() {
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	var err error

	// 命令行参数解析
	flag.StringVar(&configPath, "c", "config.yaml", "配置文件路径")
	flag.Parse()

	log.Infof("configure file: %s", configPath)
	// 读取配置文件
	if file, err = ioutil.ReadFile(configPath); err != nil {
		log.Errorf("config.yaml read error %v", err)
	}
	// 解析yaml
	if err = yaml.Unmarshal(file, config); err != nil {
		log.Errorf("config.yaml unmarshal error %v", err)
	}
	// 添加默认值
	if err = defaults.Set(config); err != nil {
		log.Errorf("set defaults error %v", err)
	}
	// 预处理认证信息
	for _, user := range config.User {
		base := fmt.Sprintf("%s:%s", user.Name, user.Password)
		auth := base64.StdEncoding.EncodeToString([]byte(base))
		config.Auth[auth] = auth
	}
	// 预处理存储库
	for _, repository := range config.Repository {
		config.RepositoryStore[repository.Id] = repository
		if repository.Mode > 0 {
			log.Infof("repository: http://%s:%s/%s/%s", config.Listen, config.Port, config.Context, repository.Id)
		}
	}
}
