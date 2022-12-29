package chatbot

type State int

const (
	StateSyncing State = iota
	StateReady   State = iota
)
