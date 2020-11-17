package store

import (
	"context"
	"errors"
	"sync"

	"github.com/cortezaproject/corteza-server/pkg/envoy"
	"github.com/cortezaproject/corteza-server/pkg/envoy/resource"
	"github.com/cortezaproject/corteza-server/store"
	"github.com/davecgh/go-spew/spew"
)

const (
	// Skip skips the existing resource
	Skip mergeAlg = iota
	// Replace replaces the existing resource
	Replace
	// MergeLeft updates the existing resource, giving priority to the existing data
	MergeLeft
	// MergeRight updates the existing resource, giving priority to the new data
	MergeRight
)

type (
	storeEncoder struct {
		s   store.Storer
		cfg *EncoderConfig

		// Each resource should define its own state that is used when encoding the resource.
		// Such approach removes the need for a janky generic global state.
		// This also simplifies any slight deviations between resource complexities.
		state map[resource.Interface]resourceState
	}

	mergeAlg int

	// EncoderConfig allows us to configure the resource encoding process
	EncoderConfig struct {
		OnExisting mergeAlg
	}

	// resourceState allows each conforming struct to be initialized and encoded
	// by the store encoder
	resourceState interface {
		Prepare(ctx context.Context, s store.Storer, state *envoy.ResourceState) (err error)
		Encode(ctx context.Context, s store.Storer, state *envoy.ResourceState) (err error)
	}
)

// NewStoreEncoder initializes a fresh store encoder
//
// If no config is provided, it uses Skip as the default merge alg.
func NewStoreEncoder(s store.Storer, cfg *EncoderConfig) envoy.PrepareEncoder {
	if cfg == nil {
		cfg = &EncoderConfig{
			OnExisting: Skip,
		}
	}

	return &storeEncoder{
		s:   s,
		cfg: cfg,

		state: make(map[resource.Interface]resourceState),
	}
}

// Prepare prepares the encoder for the given set of resources
//
// It initializes and prepares the resource state for each provided resource
func (se *storeEncoder) Prepare(ctx context.Context, ee ...*envoy.ResourceState) (err error) {
	f := func(rs resourceState, es *envoy.ResourceState) error {
		err = rs.Prepare(ctx, se.s, es)
		if err != nil {
			return err
		}

		se.state[es.Res] = rs
		return nil
	}

	for _, e := range ee {
		switch res := e.Res.(type) {
		// Compose things
		case *resource.ComposeNamespace:
			err = f(NewComposeNamespaceState(res, se.cfg), e)
		case *resource.ComposeModule:
			err = f(NewComposeModuleState(res, se.cfg), e)
		case *resource.ComposeRecord:
			err = f(NewComposeRecordState(res, se.cfg), e)
		case *resource.ComposeChart:
			err = f(NewComposeChartState(res, se.cfg), e)
		case *resource.ComposePage:
			err = f(NewComposePageState(res, se.cfg), e)

			// System things
		case *resource.User:
			err = f(NewUserState(res, se.cfg), e)
		case *resource.Role:
			err = f(NewRole(res, se.cfg), e)
		case *resource.Application:
			err = f(NewApplicationState(res, se.cfg), e)
		case *resource.Settings:
			err = f(NewSettingsState(res, se.cfg), e)
		case *resource.RbacRule:
			err = f(NewRbacRuleState(res, se.cfg), e)

		default:
			return errors.New("[encoder] unknown resource; @todo error")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes available resource states using the given store encoder
func (se *storeEncoder) Encode(ctx context.Context, wg *sync.WaitGroup, rc envoy.Rc, ec envoy.Ec) {
	defer wg.Done()

	var e *envoy.ResourceState
	err := store.Tx(ctx, se.s, func(ctx context.Context, s store.Storer) (err error) {
		for {
			e = <-rc
			if e == nil {
				return nil
			}

			state := se.state[e.Res]
			if state == nil {
				return errors.New("Resource state not defined; @todo error")
			}

			err = state.Encode(ctx, se.s, e)
			if err != nil {
				return err
			}
		}
	})

	if err != nil {
		// ec <- err
		spew.Dump(err)
	}
}
