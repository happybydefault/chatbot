package chatbot

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func (s *Server) handleQR(ctx context.Context, qr *events.QR) error {
	s.logger.Info(
		"QR event received",
		zap.String("qr_codes", fmt.Sprintf("%q", qr.Codes)),
	)

	if len(qr.Codes) == 0 {
		return fmt.Errorf("received empty slice of QR codes")
	}

	// TODO: Handle multiple QR code instead of just using the first one.
	qrterminal.GenerateHalfBlock(qr.Codes[0], qrterminal.L, os.Stdout)

	return nil
}
