package chatbot

type State int

const (
	StateSyncing State = iota
	StateSynced
)
