package tasks

import "sync"

var BoxRunning sync.Map

func IsBoxRunning(id int64) bool {
	_, ok := BoxRunning.Load(id)
	return ok
}