package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot"
)

func main() {
	var statusCode int
	defer func() {
		os.Exit(statusCode)
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := run(ctx)
	if err != nil {
		log.Println(err)
		statusCode = 1
	}
}

func run(ctx context.Context) error {
	fmt.Println("hello")

	connConfig, err := pgx.ParseConfig("postgres://postgres:password@localhost:5432/postgres")
	if err != nil {
		return fmt.Errorf("failed to parse Postgres connection string: %w", err)
	}
	db := stdlib.OpenDB(*connConfig)
	defer func() {
		err := db.Close()
		if err != nil {
			log.Printf("failed to close database connection: %s", err)
		}
	}()

	zapDBLogger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create zap DB logger: %w", err)
	}
	defer zapDBLogger.Sync()
	dbLogger := chatbot.NewLogger(zapDBLogger)

	zapClientLogger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create zap client logger: %w", err)
	}
	defer zapClientLogger.Sync()
	clientLogger := chatbot.NewLogger(zapClientLogger)

	deviceStore := sqlstore.NewWithDB(db, "postgres", dbLogger)
	err = deviceStore.Upgrade()
	if err != nil {
		return fmt.Errorf("failed to upgrade the database: %w", err)
	}
	device, err := deviceStore.GetFirstDevice()
	if err != nil {
		return fmt.Errorf("failed to get the first device: %w", err)
	}

	client := whatsmeow.NewClient(device, clientLogger)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	<-ctx.Done()
	log.Println("shutting down")

	client.Disconnect()

	return nil
}

// TODO: Refactor; this is copied/pasted.
func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}
