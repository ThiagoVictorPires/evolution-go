package whatsmeow_service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Limites do pool do store do whatsmeow. Um unico pool atende TODAS as
// instancias; o limite protege o max_connections do Postgres (padrao 100).
const (
	storeMaxOpenConns    = 20
	storeMaxIdleConns    = 5
	storeConnMaxIdleTime = 1 * time.Minute
	storeConnMaxLifetime = 30 * time.Minute
)

// sharedStore guarda o unico sqlstore.Container (e seu *sql.DB) do processo.
//
// Antes, StartClient chamava sqlstore.New a cada (re)conexao: cada chamada abria
// um *sql.DB novo, nunca fechado, e as conexoes TCP ficavam abertas ate o
// Postgres recusar com "sorry, too many clients already". O whatsmeow suporta
// varios devices no mesmo container, entao um so basta.
type sharedStore struct {
	mu        sync.Mutex
	container *sqlstore.Container
	db        *sql.DB
}

// get devolve o container compartilhado, criando-o na primeira chamada com
// open(). Erro NAO e memorizado (mutex, nao sync.Once): se o banco estiver
// fora do ar uma vez, a proxima chamada tenta de novo.
func (s *sharedStore) get(open func() (*sqlstore.Container, *sql.DB, error)) (*sqlstore.Container, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.container != nil {
		return s.container, nil
	}

	container, db, err := open()
	if err != nil {
		return nil, err
	}

	s.container = container
	s.db = db
	return container, nil
}

// getContainer devolve o container do whatsmeow, compartilhado por todas as instancias.
func (w whatsmeowService) getContainer() (*sqlstore.Container, error) {
	return w.store.get(w.openStoreContainer)
}

func (w whatsmeowService) openStoreContainer() (*sqlstore.Container, *sql.DB, error) {
	var dbLog waLog.Logger
	if w.config.WaDebug != "" {
		dbLog = waLog.Stdout("Database", w.config.WaDebug, true)
	}

	dialect := "sqlite"
	dsn := fmt.Sprintf("file:%s/dbdata/main.db?_pragma=foreign_keys(1)&_busy_timeout=5000&cache=shared&mode=rwc&_journal_mode=WAL", w.exPath)
	if w.config.PostgresAuthDB != "" {
		dialect = "postgres"
		dsn = w.config.PostgresAuthDB
	}

	db, err := sql.Open(dialect, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(storeMaxOpenConns)
	db.SetMaxIdleConns(storeMaxIdleConns)
	db.SetConnMaxIdleTime(storeConnMaxIdleTime)
	db.SetConnMaxLifetime(storeConnMaxLifetime)

	container := sqlstore.NewWithDB(db, dialect, dbLog)
	if err := container.Upgrade(context.Background()); err != nil {
		// Fecha o pool para nao vazar conexoes quando o upgrade falha.
		_ = db.Close()
		return nil, nil, fmt.Errorf("failed to upgrade database: %w", err)
	}

	return container, db, nil
}

// shouldAutoRestart diz se a instancia pode ser reiniciada sozinha. Somente
// instancias pareadas (com device JID) reconectam automaticamente; uma instancia
// nunca pareada que terminou o ciclo de QR fica parada ate pedirem QR de novo.
func shouldAutoRestart(hasDeviceJID bool) bool {
	return hasDeviceJID
}
