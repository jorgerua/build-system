package natsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// NATSClient define a interface para o cliente NATS
type NATSClient interface {
	Connect(url string) error
	Publish(subject string, data []byte) error
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error)
	Close()
	IsConnected() bool
}

// Client implementa a interface NATSClient
type Client struct {
	conn   *nats.Conn
	logger *zap.Logger
	url    string
}

// Config contém as configurações do cliente NATS
type Config struct {
	URL            string
	MaxReconnects  int
	ReconnectWait  time.Duration
	ConnectTimeout time.Duration
}

// DefaultConfig retorna uma configuração padrão
func DefaultConfig() *Config {
	return &Config{
		MaxReconnects:  10,
		ReconnectWait:  2 * time.Second,
		ConnectTimeout: 5 * time.Second,
	}
}

// NewClient cria uma nova instância do cliente NATS
func NewClient(logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		logger: logger,
	}
}

// NewClientWithConfig cria uma nova instância do cliente NATS com configuração customizada
func NewClientWithConfig(logger *zap.Logger, config *Config) (*Client, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	client := &Client{
		logger: logger,
		url:    config.URL,
	}
	
	if err := client.ConnectWithConfig(config); err != nil {
		return nil, err
	}
	
	return client, nil
}

// Connect estabelece conexão com o servidor NATS com retry automático
func (c *Client) Connect(url string) error {
	config := DefaultConfig()
	config.URL = url
	return c.ConnectWithConfig(config)
}

// ConnectWithConfig estabelece conexão com o servidor NATS usando configuração customizada
func (c *Client) ConnectWithConfig(config *Config) error {
	c.url = config.URL
	
	c.logger.Info("connecting to NATS server",
		zap.String("url", config.URL),
		zap.Int("max_reconnects", config.MaxReconnects),
		zap.Duration("reconnect_wait", config.ReconnectWait),
	)

	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.Timeout(config.ConnectTimeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				c.logger.Warn("disconnected from NATS server",
					zap.Error(err),
					zap.String("url", nc.ConnectedUrl()),
				)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.logger.Info("reconnected to NATS server",
				zap.String("url", nc.ConnectedUrl()),
			)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			c.logger.Info("connection to NATS server closed",
				zap.String("url", nc.ConnectedUrl()),
			)
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			c.logger.Error("NATS error",
				zap.Error(err),
				zap.String("subject", sub.Subject),
			)
		}),
	}

	conn, err := nats.Connect(config.URL, opts...)
	if err != nil {
		c.logger.Error("failed to connect to NATS server",
			zap.Error(err),
			zap.String("url", config.URL),
		)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	c.conn = conn
	c.logger.Info("successfully connected to NATS server",
		zap.String("url", conn.ConnectedUrl()),
		zap.String("server_id", conn.ConnectedServerId()),
	)

	return nil
}

// Publish publica uma mensagem em um subject
func (c *Client) Publish(subject string, data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to NATS server")
	}

	c.logger.Debug("publishing message",
		zap.String("subject", subject),
		zap.Int("size", len(data)),
	)

	if err := c.conn.Publish(subject, data); err != nil {
		c.logger.Error("failed to publish message",
			zap.Error(err),
			zap.String("subject", subject),
		)
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	c.logger.Debug("message published successfully",
		zap.String("subject", subject),
	)

	return nil
}

// Subscribe cria uma subscrição para um subject
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	c.logger.Info("subscribing to subject",
		zap.String("subject", subject),
	)

	// Wrapper do handler para adicionar logging
	wrappedHandler := func(msg *nats.Msg) {
		c.logger.Debug("received message",
			zap.String("subject", msg.Subject),
			zap.Int("size", len(msg.Data)),
		)
		handler(msg)
	}

	sub, err := c.conn.Subscribe(subject, wrappedHandler)
	if err != nil {
		c.logger.Error("failed to subscribe to subject",
			zap.Error(err),
			zap.String("subject", subject),
		)
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	c.logger.Info("successfully subscribed to subject",
		zap.String("subject", subject),
	)

	return sub, nil
}

// Request envia uma requisição e aguarda resposta
func (c *Client) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	c.logger.Debug("sending request",
		zap.String("subject", subject),
		zap.Int("size", len(data)),
		zap.Duration("timeout", timeout),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msg, err := c.conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		c.logger.Error("request failed",
			zap.Error(err),
			zap.String("subject", subject),
		)
		return nil, fmt.Errorf("request to %s failed: %w", subject, err)
	}

	c.logger.Debug("received response",
		zap.String("subject", subject),
		zap.Int("size", len(msg.Data)),
	)

	return msg, nil
}

// Close fecha a conexão com graceful shutdown
func (c *Client) Close() {
	if c.conn == nil {
		return
	}

	c.logger.Info("closing NATS connection",
		zap.String("url", c.url),
	)

	// Drain permite que mensagens pendentes sejam processadas antes de fechar
	if err := c.conn.Drain(); err != nil {
		c.logger.Warn("error draining connection",
			zap.Error(err),
		)
	}

	c.conn.Close()
	c.conn = nil

	c.logger.Info("NATS connection closed")
}

// IsConnected verifica se o cliente está conectado
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Stats retorna estatísticas da conexão
func (c *Client) Stats() nats.Statistics {
	if c.conn == nil {
		return nats.Statistics{}
	}
	return c.conn.Stats()
}

// Flush força o envio de mensagens pendentes
func (c *Client) Flush() error {
	if c.conn == nil {
		return fmt.Errorf("not connected to NATS server")
	}
	return c.conn.Flush()
}

// FlushTimeout força o envio de mensagens pendentes com timeout
func (c *Client) FlushTimeout(timeout time.Duration) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to NATS server")
	}
	return c.conn.FlushTimeout(timeout)
}
