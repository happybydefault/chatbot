package chatbot

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.Logger
	db     *sql.DB
	client *whatsmeow.Client
}

func NewServer(config Config) (*Server, error) {
	connConfig, err := pgx.ParseConfig(config.PostgresConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Postgres connection string: %w", err)
	}
	db := stdlib.OpenDB(*connConfig)

	return &Server{
		logger: config.Logger,
		db:     db,
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

	s.client = whatsmeow.NewClient(
		device,
		newWALogger(s.logger.Named("whatsmeow-client")),
	)
	s.client.AddEventHandler(s.eventHandler(ctx))

	// TODO: Refactor; this is copied/pasted.
	if s.client.Store.ID == nil {
		ch, _ := s.client.GetQRChannel(ctx)
		err = s.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect the client to WhatsApp: %w", err)
		}
		for event := range ch {
			if event.Event == "code" {
				qrterminal.GenerateHalfBlock(event.Code, qrterminal.L, os.Stdout)
			} else {
				s.logger.Info(
					"received unhandled login event",
					zap.String("login_event", event.Event),
				)
			}
		}
	} else {
		err = s.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect the client to WhatsApp: %w", err)
		}
	}

	<-ctx.Done()
	s.logger.Info("shutting down")
	s.close()

	return nil
}

func (s *Server) eventHandler(ctx context.Context) func(event interface{}) {
	return func(event interface{}) {
		switch e := event.(type) {
		case *events.Message:
			err := s.handleMessage(ctx, e)
			if err != nil {
				s.logger.Error("failed to handle message", zap.Error(err))
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
	s.client.Disconnect()
	s.logger.Debug("disconnected from WhatsApp")

	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	s.logger.Debug("closed database")

	return nil
}
