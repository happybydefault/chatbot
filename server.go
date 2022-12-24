package chatbot

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

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
	logger   *zap.Logger
	store    data.Store
	db       *sql.DB
	whatsapp *whatsmeow.Client
	gpt3     gpt3.Client

	wg sync.WaitGroup
}

func NewServer(config Config) (*Server, error) {
	connConfig, err := pgx.ParseConfig(config.PostgresConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Postgres connection string: %w", err)
	}
	db := stdlib.OpenDB(*connConfig)

	gpt3Client := gpt3.NewClient(config.OpenAIAPIKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

	return &Server{
		logger: config.Logger,
		store:  config.Store,
		db:     db,
		gpt3:   gpt3Client,
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	db := sqlstore.NewWithDB(
		s.db,
		"postgres",
		newWALogger(s.logger.Named("whatsmeow-db")),
	)
	err := db.Upgrade()
	if err != nil {
		return fmt.Errorf("failed to upgrade the whatsmeow database: %w", err)
	}

	device, err := db.GetFirstDevice()
	if err != nil {
		return fmt.Errorf("failed to get the first device: %w", err)
	}

	s.whatsapp = whatsmeow.NewClient(
		device,
		newWALogger(s.logger.Named("whatsmeow-client")),
	)
	s.whatsapp.AddEventHandler(s.eventHandler(ctx))

	err = s.whatsapp.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect the client to WhatsApp: %w", err)
	}

	<-ctx.Done()

	s.logger.Info("waiting for all event handlers to finish before shutting down")
	s.wg.Wait()
	s.logger.Info("shutting down")

	err = s.whatsapp.SendPresence(types.PresenceUnavailable)
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
			// Starting from a background context,
			// so the message continues being handled when the server is shutting down.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := s.handleMessage(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle Message event", zap.Error(err))
				return
			}
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
		default:
			s.logger.Debug(
				"unhandled event received",
				zap.String("event", fmt.Sprintf("%#v", e)),
			)
		}
	}
}

func (s *Server) close() error {
	s.whatsapp.Disconnect()
	s.logger.Debug("disconnected from WhatsApp")

	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	s.logger.Debug("closed database")

	return nil
}
