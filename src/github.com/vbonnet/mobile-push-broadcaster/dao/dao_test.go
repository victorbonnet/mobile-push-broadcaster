package dao

import (
	"strconv"
	"testing"
	// "time"
	"sync"
)

func TestGCMApi(t *testing.T) {
	app := "App1"
	tokens1 := GetGCMTokens(app)
	if len(tokens1) != 0 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens1), 0)
	}

	AddGCMToken(app, "123")
	AddGCMToken(app, "456")
	AddGCMToken(app, "789")
	AddGCMToken(app, "100")
	tokens2 := GetGCMTokens(app)
	if len(tokens2) != 4 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens2), 4)
	}

	AddGCMToken(app, "123")
	tokens21 := GetGCMTokens(app)
	if len(tokens21) != 4 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens21), 4)
	}

	RemoveGCMToken(app, "100")
	tokens3 := GetGCMTokens(app)
	if len(tokens3) != 3 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens3), 3)
	}

	RemoveGCMToken(app, "111")
	tokens4 := GetGCMTokens(app)
	if len(tokens4) != 3 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens4), 3)
	}
}

func add(n int) {
	AddGCMToken("App2", strconv.Itoa(n))
}

func remove(n int) {
	RemoveGCMToken("App2", strconv.Itoa(n))
}

func TestThreadsafe(t *testing.T) {
	MAX := 30000
	var w sync.WaitGroup
	w.Add(MAX)

	for i := 1; i <= MAX; i++ {
		go func(val int) {
			add(val)
			w.Done()
		}(i)

		// go remove(MAX - i)
	}
	// time.Sleep(30 * time.Second)
	w.Wait()

	tokens := GetGCMTokens("App2")
	if len(tokens) != 30000 {
		t.Errorf("len(GetGCMTokens() = %v, want %v", len(tokens), 30000)
	}

	// for i := 1; i <= MAX; i++ {
	// 	go remove(MAX - i)
	// }
}
