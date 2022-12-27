package chatbot

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

type Server struct {
	logger    *zap.Logger
	store     data.Store
	db        *sql.DB
	whatsmeow *whatsmeow.Client
	gpt3      gpt3.Client

	wg sync.WaitGroup

	mu           sync.RWMutex
	state        State
	pendingChats map[types.JID]struct{}
}

func NewServer(cfg Config) (*Server, error) {
	connConfig, err := pgx.ParseConfig(cfg.PostgresConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Postgres connection string: %w", err)
	}
	db := stdlib.OpenDB(*connConfig)

	whatsmeowLogger := cfg.Logger.Named("whatsmeow").WithOptions(
		zap.IncreaseLevel(zap.InfoLevel),
	)

	whatsappDB := sqlstore.NewWithDB(
		db,
		"postgres",
		newWALogger(whatsmeowLogger.Named("db")),
	)
	err = whatsappDB.Upgrade()
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade the whatsmeow database: %w", err)
	}

	device, err := whatsappDB.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get the first device: %w", err)
	}

	whatsappClient := whatsmeow.NewClient(
		device,
		newWALogger(whatsmeowLogger.Named("client")),
	)

	gpt3Client := gpt3.NewClient(cfg.OpenAIAPIKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

	return &Server{
		logger:       cfg.Logger,
		store:        cfg.Store,
		db:           db,
		whatsmeow:    whatsappClient,
		gpt3:         gpt3Client,
		pendingChats: make(map[types.JID]struct{}),
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	s.whatsmeow.AddEventHandler(s.eventHandler(ctx))

	err := s.whatsmeow.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect the whatsmeow client to WhatsApp: %w", err)
	}

	<-ctx.Done()
	s.whatsmeow.RemoveEventHandlers()

	s.logger.Info("waiting for all event handlers to finish before shutting down")
	s.wg.Wait()
	s.logger.Info("shutting down")

	err = s.whatsmeow.SendPresence(types.PresenceUnavailable)
	if err != nil {
		return fmt.Errorf("failed to send unavailable presence: %w", err)
	}

	err = s.close()
	if err != nil {
		return fmt.Errorf("failed to close the server: %w", err)
	}

	return nil
}

func (s *Server) eventHandler(ctx context.Context) func(event interface{}) {
	return func(event interface{}) {
		s.wg.Add(1)
		defer s.wg.Done()

		switch e := event.(type) {
		case *events.Connected:
			err := s.handleConnected(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle connected event", zap.Error(err))
				return
			}
		case *events.Message:
			s.handleMessage(ctx, e)
		case *events.QR:
			err := s.handleQR(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle QR event", zap.Error(err))
				return
			}
		case *events.LoggedOut:
			err := s.handleLoggedOut(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle LoggedOut event", zap.Error(err))
				return
			}
		case *events.OfflineSyncPreview:
			err := s.handleOfflineSyncPreview(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle OfflineSyncPreview event", zap.Error(err))
				return
			}
		case *events.OfflineSyncCompleted:
			err := s.handleOfflineSyncCompleted(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle OfflineSyncCompleted event", zap.Error(err))
				return
			}
		default:
			s.logger.Debug(
				"unhandled event received",
				zap.String("event", fmt.Sprintf("%#v", e)),
			)
		}
	}
}

func (s *Server) close() error {
	s.whatsmeow.Disconnect()
	s.logger.Debug("whatsmeow client disconnected from WhatsApp")

	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	s.logger.Debug("closed database")

	return nil
}
