package chatbot

type State int

const (
	StatusSyncing State = iota
	StatusReady   State = iota
)
