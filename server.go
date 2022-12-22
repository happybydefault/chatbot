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

func (s *Server) Close() error {
	s.client.Disconnect()
	s.logger.Debug("disconnected from WhatsApp")

	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	s.logger.Debug("closed database")

	return nil
}

// TODO: Refactor; this is copied/pasted.
func (s *Server) Serve(ctx context.Context) error {
	db := sqlstore.NewWithDB(
		s.db,
		"postgres",
		newWALogger(s.logger.Named("whatsmeow-container")),
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
	s.client.AddEventHandler(s.eventHandler)

	if s.client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := s.client.GetQRChannel(ctx)
		err = s.client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				s.logger.Info(
					"received login event",
					zap.String("login_event", evt.Event),
				)
			}
		}
	} else {
		// Already logged in, just connect
		err = s.client.Connect()
		if err != nil {
			panic(err)
		}
	}

	<-ctx.Done()
	s.logger.Info("shutting down")

	return nil
}

// TODO: Refactor; this is copied/pasted.
func (s *Server) eventHandler(event interface{}) {
	switch e := event.(type) {
	case *events.Message:
		s.logger.Info(
			"message received",
			zap.String("message", e.Message.GetConversation()),
		)
	default:
		s.logger.Debug(
			"unhandled event received",
			zap.String("event", fmt.Sprintf("%#v", event)),
		)
	}
}
