package store

import (
	"context"
	"time"

	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/envoy"
	"github.com/cortezaproject/corteza-server/pkg/envoy/resource"
	"github.com/cortezaproject/corteza-server/store"
)

type (
	composeNamespaceState struct {
		cfg *EncoderConfig

		res *resource.ComposeNamespace
		ns  *types.Namespace
	}
)

func NewComposeNamespaceState(res *resource.ComposeNamespace, cfg *EncoderConfig) resourceState {
	return &composeNamespaceState{
		cfg: cfg,

		res: res,
	}
}

func (n *composeNamespaceState) Prepare(ctx context.Context, s store.Storer, state *envoy.ResourceState) (err error) {
	// Initial values
	if n.res.Res.CreatedAt.IsZero() {
		n.res.Res.CreatedAt = time.Now()
	}

	// Try to get the original chart
	n.ns, err = findComposeNamespaceS(ctx, s, makeGenericFilter(n.res.Identifiers()))
	if err != nil {
		return err
	}

	if n.ns != nil {
		n.res.Res.ID = n.ns.ID
	}
	return nil
}

func (n *composeNamespaceState) Encode(ctx context.Context, s store.Storer, state *envoy.ResourceState) (err error) {
	res := n.res.Res
	exists := n.ns != nil && n.ns.ID > 0

	// Determine the ID
	if res.ID <= 0 && exists {
		res.ID = n.ns.ID
	}
	if res.ID <= 0 {
		res.ID = nextID()
	}

	// This is not possible, but let's do it anyway
	if state.Conflicting {
		return nil
	}

	// Create a fresh namespace
	if !exists {
		return store.CreateComposeNamespace(ctx, s, res)
	}

	// Update existing namespace
	switch n.cfg.OnExisting {
	case Skip:
		return nil

	case MergeLeft:
		res = mergeComposeNamespaces(n.ns, res)

	case MergeRight:
		res = mergeComposeNamespaces(res, n.ns)
	}

	err = store.UpdateComposeNamespace(ctx, s, res)
	if err != nil {
		return err
	}

	n.res.Res = res
	return nil
}

// mergeComposeNamespaces merges b into a, prioritising a
func mergeComposeNamespaces(a, b *types.Namespace) *types.Namespace {
	c := a.Clone()

	if c.Name == "" {
		c.Name = b.Name
	}
	if c.Slug == "" {
		c.Slug = b.Slug
	}

	// I'll just compare the entire struct for now
	if c.Meta == (types.NamespaceMeta{}) {
		c.Meta = b.Meta
	}

	return c
}

// findComposeNamespaceRS looks for the namespace in the resources & the store
//
// Provided resources are prioritized.
func findComposeNamespaceRS(ctx context.Context, s store.Storer, rr resource.InterfaceSet, ii resource.Identifiers) (ns *types.Namespace, err error) {
	ns = findComposeNamespaceR(rr, ii)
	if ns != nil {
		return ns, nil
	}

	return findComposeNamespaceS(ctx, s, makeGenericFilter(ii))
}

// findComposeNamespaceS looks for the namespace in the store
func findComposeNamespaceS(ctx context.Context, s store.Storer, gf genericFilter) (ns *types.Namespace, err error) {
	if gf.id > 0 {
		ns, err = store.LookupComposeNamespaceByID(ctx, s, gf.id)
		if err != nil && err != store.ErrNotFound {
			return nil, err
		}

		if ns != nil {
			return
		}
	}

	if gf.handle != "" {
		ns, err = store.LookupComposeNamespaceBySlug(ctx, s, gf.handle)
		if err != nil && err != store.ErrNotFound {
			return nil, err
		}

		if ns != nil {
			return
		}
	}

	return nil, nil
}

// findComposeNamespaceR looks for the namespace in the resources
func findComposeNamespaceR(rr resource.InterfaceSet, ii resource.Identifiers) (ns *types.Namespace) {
	var nsRes *resource.ComposeNamespace
	var ok bool

	rr.Walk(func(r resource.Interface) error {
		nsRes, ok = r.(*resource.ComposeNamespace)
		if !ok {
			return nil
		}

		if !nsRes.Identifiers().HasAny(r.Identifiers()) {
			nsRes = nil
		}
		return nil
	})

	// Found it
	if nsRes != nil {
		return nsRes.Res
	}

	return nil
}
