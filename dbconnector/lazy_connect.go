package dbconnector

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/rs/zerolog/log"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// ConnectRethinkDB establishes lazy rethinkdb connection
// It will make attempt to connect with first call and reconnect after every error
func ConnectRethinkDB(
	addresses []string,
	username, password string,
	tlsConfig *tls.Config,
	poolSize int,
) *LazyRethinkSession {
	const systemDatabase = "rethinkdb"

	return &LazyRethinkSession{
		opts: r.ConnectOpts{
			Addresses: addresses,
			Database:  systemDatabase,
			Username:  username,
			Password:  password,
			TLSConfig: tlsConfig,
			MaxOpen:   poolSize,
		},
	}
}

// LazyRethinkSession is a connection to the rethinkdb.
// It implements r.QueryExecutor interface.
// It will make attempt to connect with first call and reconnect after every error.
type LazyRethinkSession struct {
	*r.Session

	opts r.ConnectOpts
	m    sync.Mutex
}

// Close closes connections
func (l *LazyRethinkSession) Close() error {
	if l.Session != nil {
		return l.Session.Close()
	}
	return nil
}

// IsConnected returns true if session has a valid connection.
func (l *LazyRethinkSession) IsConnected() bool {
	if l.Session == nil {
		err := l.connect()
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to rethinkdb")
			return false
		}
	}

	is := l.Session.IsConnected()
	if !is {
		err := l.Session.Reconnect()
		if err != nil {
			return false
		}
		is = l.Session.IsConnected()
	}
	return is
}

// Query executes a ReQL query using the session to connect to the database
func (l *LazyRethinkSession) Query(ctx context.Context, q r.Query) (*r.Cursor, error) {
	if l.Session == nil {
		err := l.connect()
		if err != nil {
			return nil, err
		}
	}

	cur, err := l.Session.Query(ctx, q)
	if err == r.ErrConnectionClosed {
		err = l.Session.Reconnect()
		if err != nil {
			return nil, err
		}
		cur, err = l.Session.Query(ctx, q)
	}
	return cur, err
}

// Exec executes a ReQL query using the session to connect to the database
func (l *LazyRethinkSession) Exec(ctx context.Context, q r.Query) error {
	if l.Session == nil {
		err := l.connect()
		if err != nil {
			return err
		}
	}

	err := l.Session.Exec(ctx, q)
	if err == r.ErrConnectionClosed {
		err = l.Session.Reconnect()
		if err != nil {
			return err
		}
		err = l.Session.Exec(ctx, q)
	}
	return err
}

func (l *LazyRethinkSession) connect() error {
	l.m.Lock()
	defer l.m.Unlock()

	var err error
	if l.Session == nil {
		l.Session, err = r.Connect(l.opts)
		if err != nil {
			// to connect at next attempt
			l.Session = nil
		}
	}
	return err
}
