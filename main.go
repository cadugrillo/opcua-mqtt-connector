package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"opcua-mqtt-connector/config"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

var (
	endpoint   *string
	ConfigFile config.Config
	nodes      []*string
	nodesGroup NodesGroup
	nodesList  NodesList
	PubConnOk  bool
	SubConnOk  bool
	payload    Payload
	v          interface{}
)

type NodesList struct {
	nodesGroup []NodesGroup
}

type NodesGroup struct {
	nodes []*ua.ReadValueID
}

type Payload struct {
	ClientName    string   `json:"clientName"`
	ServerAddress string   `json:"serverAddress"`
	Signals       []Signal `json:"signals"`
}

type Signal struct {
	Name  string        `json:"name"`
	Qc    ua.StatusCode `json:"qc"`
	Ts    time.Time     `json:"ts"`
	Value interface{}   `json:"value"`
}

func NewTLSConfig(rootCAPath string, clientKeyPath string, privateKeyPath string, insecureSkipVerify bool) *tls.Config {

	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(rootCAPath)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	cert, err := tls.LoadX509KeyPair(clientKeyPath, privateKeyPath)
	if err != nil {
		panic(err)
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		panic(err)
	}
	//fmt.Println(cert.Leaf)

	return &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: insecureSkipVerify,
		Certificates:       []tls.Certificate{cert},
	}
}

func main() {
	////////////////////OPCUA CONFIGURATION SECTION////////////////////////////
	ConfigFile = config.ReadConfig()
	endpoint = flag.String("endpoint", ConfigFile.OpcUaClient.ServerAddress, "OPC UA Endpoint URL")
	payload.ClientName = ConfigFile.OpcUaClient.ClientId
	payload.ServerAddress = ConfigFile.OpcUaClient.ServerAddress

	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	for i := 0; i < len(ConfigFile.NodesToRead.Nodes); i++ {
		node := []*string{flag.String(ConfigFile.NodesToRead.Nodes[i].Name, ConfigFile.NodesToRead.Nodes[i].NodeID, "NodeID to read")}
		nodes = append(nodes, node...)
	}

	j := 0
	k := 0
	for i := 0; i < len(nodes); i++ {

		id, err := ua.ParseNodeID(*nodes[i])
		if err != nil {
			log.Fatalf("invalid node id: %v", err)
		}
		r := []*ua.ReadValueID{{NodeID: id}}
		nodesGroup.nodes = append(nodesGroup.nodes, r...)
		j = j + 1

		if (j == ConfigFile.OpcUaClient.MaxSignalsPerRead) || (i == len(nodes)-1) {
			s := []NodesGroup{{nodes: nodesGroup.nodes}}
			nodesList.nodesGroup = append(nodesList.nodesGroup, s...)
			nodesGroup.nodes = nil
			j = 0
			log.Println("New Node Group: ", nodesList.nodesGroup[k].nodes)
			k = k + 1
		}

	}

	////////////////////END OF OPCUA CONFIGURATION SECTION/////////////////////

	////////////////////MQTT CONFIGURATION SECTION////////////////////////////
	//logs
	if ConfigFile.Logs.Error {
		mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	}
	if ConfigFile.Logs.Critical {
		mqtt.CRITICAL = log.New(os.Stdout, "[CRITICAL] ", 0)
	}
	if ConfigFile.Logs.Warning {
		mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	}
	if ConfigFile.Logs.Debug {
		mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)
	}

	/////opts for Pub Broker
	optsPub := mqtt.NewClientOptions()
	optsPub.AddBroker(ConfigFile.ClientPub.ServerAddress)

	switch ConfigFile.ClientPub.TlsConn {
	case true:
		tlsPub := NewTLSConfig(ConfigFile.ClientPub.RootCAPath, ConfigFile.ClientPub.ClientKeyPath, ConfigFile.ClientPub.PrivateKeyPath, ConfigFile.ClientPub.InsecureSkipVerify)
		optsPub.SetClientID(ConfigFile.ClientPub.ClientId).SetTLSConfig(tlsPub)
	case false:
		optsPub.SetClientID(ConfigFile.ClientPub.ClientId)
		optsPub.SetUsername(ConfigFile.ClientPub.UserName)
		optsPub.SetPassword(ConfigFile.ClientPub.Password)
	}

	optsPub.SetOrderMatters(ConfigFile.ClientPub.OrderMaters)                                      // Allow out of order messages (use this option unless in order delivery is essential)
	optsPub.ConnectTimeout = (time.Duration(ConfigFile.ClientPub.ConnectionTimeout) * time.Second) // Minimal delays on connect
	optsPub.WriteTimeout = (time.Duration(ConfigFile.ClientPub.WriteTimeout) * time.Second)        // Minimal delays on writes
	optsPub.KeepAlive = int64(ConfigFile.ClientPub.KeepAlive)                                      // Keepalive every 10 seconds so we quickly detect network outages
	optsPub.PingTimeout = (time.Duration(ConfigFile.ClientPub.PingTimeout) * time.Second)          // local broker so response should be quick
	optsPub.ConnectRetry = ConfigFile.ClientPub.ConnectRetry                                       // Automate connection management (will keep trying to connect and will reconnect if network drops)
	optsPub.AutoReconnect = ConfigFile.ClientPub.AutoConnect
	optsPub.DefaultPublishHandler = func(_ mqtt.Client, msg mqtt.Message) { fmt.Printf("PUB BROKER - UNEXPECTED : %s\n", msg) }

	optsPub.OnConnectionLost = func(cl mqtt.Client, err error) {
		fmt.Println("PUB BROKER - CONNECTION LOST")
		PubConnOk = false
	}

	optsPub.OnConnect = func(c mqtt.Client) {
		fmt.Println("PUB BROKER - CONNECTION STABLISHED")
		PubConnOk = true
	}

	optsPub.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) { fmt.Println("PUB BROKER - ATTEMPTING TO RECONNECT") }

	//connect to PUB broker
	//
	clientPub := mqtt.NewClient(optsPub)

	if tokenPub := clientPub.Connect(); tokenPub.Wait() && tokenPub.Error() != nil {
		panic(tokenPub.Error())
	}
	fmt.Println("PUB BROKER  - CONNECTION IS UP")
	////////////////////END OF MQTT CONFIGURATION SECTION////////////////////////////

	ctx := context.Background()

	c := opcua.NewClient(*endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.CloseWithContext(ctx)
	fmt.Println("OPC UA SERVER - CONNECTION STABLISHED")

	go func() {
		for {
			for j := 0; j < len(nodesList.nodesGroup); j++ {
				req := &ua.ReadRequest{
					MaxAge:             ConfigFile.OpcUaClient.MaxAge,
					NodesToRead:        nodesList.nodesGroup[j].nodes,
					TimestampsToReturn: ua.TimestampsToReturnBoth,
				}

				resp, err := c.ReadWithContext(ctx, req)
				if err != nil {
					log.Fatalf("Read failed: %s", err)
				}

				for i := 0; i < len(resp.Results); i++ {

					if resp.Results[i].Status != ua.StatusOK {
						log.Fatalf("Status not OK: %v", resp.Results[i].Status)
					}

					x := resp.Results[i].Value.Value()
					switch x.(type) {
					case nil:
						log.Println("node value is nil")
					case bool:
						v = x.(bool)
						log.Println("node value (bool): ", v)
					case int:
						v = x.(int)
						log.Println("node value (int): ", v)
					case float32:
						v = x.(float32)
						log.Println("node value (float32): ", v)
					}

					opcsignal := []Signal{{Name: ConfigFile.NodesToRead.Nodes[i].Name,
						Qc:    resp.Results[i].Status,
						Ts:    resp.Results[i].SourceTimestamp,
						Value: v,
					}}

					payload.Signals = append(payload.Signals, opcsignal...)
				}
				pl, err := json.Marshal(payload)
				if err != nil {
					log.Fatal(err)
				}
				clientPub.Publish(ConfigFile.TopicsPub.Topic[0], byte(ConfigFile.ClientPub.Qos), false, pl)
				payload.Signals = nil
				pl = nil
				time.Sleep(time.Duration(ConfigFile.OpcUaClient.MinTimeBetweenRead) * time.Millisecond)
			}
			time.Sleep(time.Duration(ConfigFile.OpcUaClient.PollInterval) * time.Second)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	fmt.Println("signal caught - exiting")
	c.Close()
	fmt.Println("shutdown complete")
}
