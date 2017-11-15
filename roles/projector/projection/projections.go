//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2017 Rui NI <ranqus@gmail.com>
//
//  This file is part of Crypto-Obscured Forwarder.
//
//  Crypto-Obscured Forwarder is free software: you can redistribute it
//  and/or modify it under the terms of the GNU General Public License
//  as published by the Free Software Foundation, either version 3 of
//  the License, or (at your option) any later version.
//
//  Crypto-Obscured Forwarder is distributed in the hope that it will be
//  useful, but WITHOUT ANY WARRANTY; without even the implied warranty
//  of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with Crypto-Obscured Forwarder. If not, see
//  <http://www.gnu.org/licenses/>.

package projection

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrProjectionAlreadyExisted = errors.New(
		"Projection already existed")

	ErrProjectionNotFound = errors.New(
		"Projection not found")
)

// ID Projection ID
type ID byte

// Consts
const (
	MaxID          = math.MaxUint8
	MaxProjections = MaxID + 1
)

// Projections represents a Projection manager
type Projections interface {
	Projection(id ID) (Projection, error)
	Handler(id ID) (network.Handler, error)
}

// projections implements Projections
type projections struct {
	cfg         Config
	projections [MaxProjections]*projection
	runner      worker.Runner
}

// New creates a new Projections
func New(
	runner worker.Runner,
	reqReceiveTick <-chan time.Time,
	cfg Config,
) Projections {
	p := &projections{
		cfg:         cfg,
		projections: [MaxProjections]*projection{},
		runner:      runner,
	}

	for pIdx := range cfg.Projects {
		p.projections[cfg.Projects[pIdx]] = &projection{
			id: cfg.Projects[pIdx],
			receivers: receivers{
				Head:      nil,
				Tail:      nil,
				Capcity:   make(chan struct{}, cfg.MaxReceivers),
				Capacitor: sync.Cond{L: &sync.Mutex{}},
			},
			receiveTimeoutTick: reqReceiveTick,
			requestTimeout:     cfg.RequestTimeout,
		}
	}

	return p
}

// Projection returns a Projection
func (p *projections) Projection(id ID) (Projection, error) {
	if id > MaxID || p.projections[id] == nil {
		return nil, ErrProjectionNotFound
	}

	return p.projections[id], nil
}

// Handler returns a new Projection Server Handler
func (p *projections) Handler(id ID) (network.Handler, error) {
	if id > MaxID || p.projections[id] == nil {
		return nil, ErrProjectionNotFound
	}

	return handler{
		projection: p.projections[id],
	}, nil
}
