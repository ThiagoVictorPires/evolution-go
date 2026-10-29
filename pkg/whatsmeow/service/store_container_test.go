package whatsmeow_service

import (
	"database/sql"
	"errors"
	"sync"
	"testing"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

func TestSharedStoreCreatesOnce(t *testing.T) {
	s := &sharedStore{}
	calls := 0
	fake := &sqlstore.Container{}
	open := func() (*sqlstore.Container, *sql.DB, error) {
		calls++
		return fake, nil, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := s.get(open)
			if err != nil || c != fake {
				t.Errorf("get = %v, %v", c, err)
			}
		}()
	}
	wg.Wait()

	if calls != 1 {
		t.Fatalf("open chamado %d vezes, esperado 1", calls)
	}
}

func TestSharedStoreDoesNotMemoizeError(t *testing.T) {
	s := &sharedStore{}
	calls := 0
	fake := &sqlstore.Container{}
	open := func() (*sqlstore.Container, *sql.DB, error) {
		calls++
		if calls == 1 {
			return nil, nil, errors.New("too many clients already")
		}
		return fake, nil, nil
	}

	if _, err := s.get(open); err == nil {
		t.Fatal("esperava erro na primeira chamada")
	}
	c, err := s.get(open)
	if err != nil || c != fake {
		t.Fatalf("segunda chamada deveria recuperar: %v, %v", c, err)
	}
	if calls != 2 {
		t.Fatalf("open chamado %d vezes, esperado 2", calls)
	}
}

func TestShouldAutoRestart(t *testing.T) {
	if shouldAutoRestart(false) {
		t.Error("instancia sem device JID nao deve reiniciar sozinha")
	}
	if !shouldAutoRestart(true) {
		t.Error("instancia pareada deve reconectar sozinha")
	}
}
