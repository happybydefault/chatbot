package chatbot

import (
	"fmt"
	"sync"

	"github.com/PullRequestInc/go-gpt3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

type Client struct {
	logger *zap.Logger
	store  data.Store

	whatsmeowClient *whatsmeow.Client
	gpt3Client      gpt3.Client

	status       Status
	pendingChats map[string]struct{}

	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewClient(cfg Config) (*Client, error) {
	// TODO: Maybe refactor.
	whatsmeowLogger := cfg.Logger.Named("whatsmeow").WithOptions(
		zap.IncreaseLevel(zap.InfoLevel),
	)

	whatsmeowDB := sqlstore.NewWithDB(
		cfg.WhatsmeowDB,
		"postgres",
		newWALogger(whatsmeowLogger.Named("db")),
	)
	err := whatsmeowDB.Upgrade()
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade the whatsmeow database: %w", err)
	}

	device, err := whatsmeowDB.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get the first device: %w", err)
	}
	whatsmeowClient := whatsmeow.NewClient(
		device,
		newWALogger(whatsmeowLogger.Named("client")),
	)

	gpt3Client := gpt3.NewClient(cfg.OpenAIAPIKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

	return &Client{
		logger:          cfg.Logger,
		store:           cfg.Store,
		whatsmeowClient: whatsmeowClient,
		gpt3Client:      gpt3Client,
		pendingChats:    make(map[string]struct{}),
		stopChan:        make(chan struct{}),
	}, nil
}

func (c *Client) Start() error {
	c.whatsmeowClient.AddEventHandler(c.eventHandler)

	err := c.whatsmeowClient.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect the whatsmeow client to WhatsApp: %w", err)
	}

	<-c.stopChan

	return nil
}

func (c *Client) Stop() error {
	c.logger.Debug("stopping")

	close(c.stopChan)

	c.whatsmeowClient.RemoveEventHandlers()

	c.logger.Info("waiting for all event handlers and chats handlers to finish")
	c.wg.Wait()
	c.logger.Info("all event handlers and chats handlers finished")

	err := c.whatsmeowClient.SendPresence(types.PresenceUnavailable)
	if err != nil {
		err = fmt.Errorf("failed to send unavailable presence: %w", err)
	}

	c.whatsmeowClient.Disconnect()
	c.logger.Debug("whatsmeow client disconnected from WhatsApp")

	return err
}

func (c *Client) eventHandler(event interface{}) {
	c.wg.Add(1)
	defer c.wg.Done()

	logger := c.logger.With(
		zap.String("event_type", fmt.Sprintf("%T", event)),
		zap.Any("event", event),
	)

	logger.Debug("received event")

	var err error

	switch e := event.(type) {
	case *events.Connected:
		err = c.handleConnectedEvent()
	case *events.QR:
		err = c.handleQREvent(e)
	case *events.Message:
		err = c.handleMessageEvent(e)
	case *events.OfflineSyncCompleted:
		err = c.handleOfflineSyncCompletedEvent()
	case *events.LoggedOut:
		err = c.handleLoggedOutEvent(e)
	default:
		return
	}

	if err != nil {
		logger.Error("failed to handle event", zap.Error(err))
	}
}
