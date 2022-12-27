package chatbot

type State int

const (
	StateReady   State = iota
	StateSyncing State = iota
)
