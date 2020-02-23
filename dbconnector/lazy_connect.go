package dbconnector

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/rs/zerolog/log"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func ConnectRethinkDB(
	addresses []string,
	username, password string,
	tlsConfig *tls.Config,
	poolSize int,
) *lazySession {
	const systemDatabase = "rethinkdb"

	return &lazySession{
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

type lazySession struct {
	*r.Session

	opts r.ConnectOpts
	m    sync.Mutex
}

func (l *lazySession) Close() error {
	if l.Session != nil {
		return l.Session.Close()
	}
	return nil
}

func (l *lazySession) IsConnected() bool {
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

func (l *lazySession) Query(ctx context.Context, q r.Query) (*r.Cursor, error) {
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

func (l *lazySession) Exec(ctx context.Context, q r.Query) error {
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

func (l *lazySession) connect() error {
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
