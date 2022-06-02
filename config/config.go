package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	OpcUaClient struct {
		ClientId      string  `yaml:"clientId"`
		ServerAddress string  `yaml:"serverAddress"`
		PollInterval  int     `yaml:"pollInterval"`
		MaxAge        float64 `yaml:"maxAge"`
	} `yaml:"opcUaClient"`
	ClientPub struct {
		ClientId           string `yaml:"clientId"`
		ServerAddress      string `yaml:"serverAddress"`
		Qos                int    `yaml:"qos"`
		ConnectionTimeout  int    `yaml:"connectionTimeout"`
		WriteTimeout       int    `yaml:"writeTimeout"`
		KeepAlive          int    `yaml:"keepAlive"`
		PingTimeout        int    `yaml:"pingTimeout"`
		ConnectRetry       bool   `yaml:"connectRetry"`
		AutoConnect        bool   `yaml:"autoConnect"`
		OrderMaters        bool   `yaml:"orderMaters"`
		UserName           string `yaml:"userName"`
		Password           string `yaml:"password"`
		TlsConn            bool   `yaml:"tlsConn"`
		RootCAPath         string `yaml:"rootCAPath"`
		ClientKeyPath      string `yaml:"clientKeyPath"`
		PrivateKeyPath     string `yaml:"privateKeyPath"`
		InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	} `yaml:"clientPub"`
	Logs struct {
		Debug    bool `yaml:"debug"`
		Warning  bool `yaml:"warning"`
		Error    bool `yaml:"error"`
		Critical bool `yaml:"critical"`
	} `yaml:"logs"`
	TopicsPub struct {
		Topic []string
	} `yaml:"topicsPub"`
	NodesToRead struct {
		Nodes []Node `yaml:"nodes"`
	} `yaml:"nodesToRead"`
}

type Node struct {
	Name   string `yaml:"name"`
	NodeID string `yaml:"nodeID"`
}

func ReadConfig() Config {
	f, err := os.Open("./config/config.yml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}
