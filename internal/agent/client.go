package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/alltomatos/clawflow/internal/core"
)

// Client representa a conexo com o Gateway OpenClaw
type Client struct {
	conn   *websocket.Conn
	config *core.OpenClawConfig
}

// NewClient cria uma nova instncia do cliente agentico
func NewClient(cfg *core.OpenClawConfig) *Client {
	return &Client{
		config: cfg,
	}
}

// Connect inicia a conexo WebSocket e realiza o handshake
func (c *Client) Connect() error {
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("127.0.0.1:%d", c.config.Gateway.Port), Path: "/ws"}
	log.Printf("Conectando ao Gateway OpenClaw em %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("erro na discagem: %w", err)
	}
	c.conn = conn

	// Handshake inicial (Aguardando Challenge)
	go c.listen()

	return nil
}

func (c *Client) listen() {
	defer c.conn.Close()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Erro na leitura do WS:", err)
			return
		}
		
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Lgica de resposta ao challenge do protocolo v3
		if msg["event"] == "connect.challenge" {
			log.Println("Challenge recebido. Respondendo com Handshake...")
			c.sendHandshake()
		}
	}
}

func (c *Client) sendHandshake() {
	// Estrutura completa de conexo exigida pelo protocolo OpenClaw v3
	handshake := map[string]interface{}{
		"type": "req",
		"id":   fmt.Sprintf("clawflow-%d", time.Now().Unix()),
		"method": "connect",
		"params": map[string]interface{}{
			"minProtocol": 3,
			"maxProtocol": 3,
			"role":        "operator",
			"scopes":      []string{"operator.read", "operator.write"},
			"auth": map[string]string{
				"token": c.config.Gateway.Token,
			},
			"client": map[string]interface{}{
				"id":       "clawflow",
				"version":  "0.1.0",
				"platform": runtime.GOOS,
				"mode":     "operator",
			},
		},
	}

	data, _ := json.Marshal(handshake)
	c.conn.WriteMessage(websocket.TextMessage, data)
}
