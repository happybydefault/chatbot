package chatbot

import (
	"fmt"
	"os"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow/types/events"
)

func (c *Client) handleQR(qr *events.QR) error {
	if len(qr.Codes) == 0 {
		return fmt.Errorf("received empty slice of QR codes")
	}

	// TODO: Handle multiple QR code instead of just using the first one.
	qrterminal.GenerateHalfBlock(qr.Codes[0], qrterminal.L, os.Stdout)

	return nil
}
