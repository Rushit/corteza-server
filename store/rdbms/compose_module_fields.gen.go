package rdbms

// This file is an auto-generated file
//
// Template:    pkg/codegen/assets/store_rdbms.gen.go.tpl
// Definitions: store/compose_module_fields.yaml
//
// Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/store"
)

var _ = errors.Is

const (
	TriggerBeforeComposeModuleFieldCreate triggerKey = "composeModuleFieldBeforeCreate"
	TriggerBeforeComposeModuleFieldUpdate triggerKey = "composeModuleFieldBeforeUpdate"
	TriggerBeforeComposeModuleFieldUpsert triggerKey = "composeModuleFieldBeforeUpsert"
	TriggerBeforeComposeModuleFieldDelete triggerKey = "composeModuleFieldBeforeDelete"
)

// SearchComposeModuleFields returns all matching rows
//
// This function calls convertComposeModuleFieldFilter with the given
// types.ModuleFieldFilter and expects to receive a working squirrel.SelectBuilder
func (s Store) SearchComposeModuleFields(ctx context.Context, f types.ModuleFieldFilter) (types.ModuleFieldSet, types.ModuleFieldFilter, error) {
	var scap uint
	q, err := s.convertComposeModuleFieldFilter(f)
	if err != nil {
		return nil, f, err
	}

	if scap == 0 {
		scap = DefaultSliceCapacity
	}

	var (
		set = make([]*types.ModuleField, 0, scap)
		// Paging is disabled in definition yaml file
		// {search: {enablePaging:false}} and this allows
		// a much simpler row fetching logic
		fetch = func() error {
			var (
				res       *types.ModuleField
				rows, err = s.Query(ctx, q)
			)

			if err != nil {
				return err
			}

			for rows.Next() {
				if rows.Err() == nil {
					res, err = s.internalComposeModuleFieldRowScanner(rows)
				}

				if err != nil {
					if cerr := rows.Close(); cerr != nil {
						err = fmt.Errorf("could not close rows (%v) after scan error: %w", cerr, err)
					}

					return err
				}

				// If check function is set, call it and act accordingly
				set = append(set, res)
			}

			return rows.Close()
		}
	)

	return set, f, s.config.ErrorHandler(fetch())
}

// LookupComposeModuleFieldByModuleIDName searches for compose module field by name (case-insensitive)
func (s Store) LookupComposeModuleFieldByModuleIDName(ctx context.Context, module_id uint64, name string) (*types.ModuleField, error) {
	return s.execLookupComposeModuleField(ctx, squirrel.Eq{
		s.preprocessColumn("cmf.rel_module", ""): s.preprocessValue(module_id, ""),
		s.preprocessColumn("cmf.name", "lower"):  s.preprocessValue(name, "lower"),
	})
}

// CreateComposeModuleField creates one or more rows in compose_module_field table
func (s Store) CreateComposeModuleField(ctx context.Context, rr ...*types.ModuleField) (err error) {
	for _, res := range rr {
		err = s.checkComposeModuleFieldConstraints(ctx, res)
		if err != nil {
			return err
		}

		// err = s.composeModuleFieldHook(ctx, TriggerBeforeComposeModuleFieldCreate, res)
		// if err != nil {
		// 	return err
		// }

		err = s.execCreateComposeModuleFields(ctx, s.internalComposeModuleFieldEncoder(res))
		if err != nil {
			return err
		}
	}

	return
}

// UpdateComposeModuleField updates one or more existing rows in compose_module_field
func (s Store) UpdateComposeModuleField(ctx context.Context, rr ...*types.ModuleField) error {
	return s.config.ErrorHandler(s.PartialComposeModuleFieldUpdate(ctx, nil, rr...))
}

// PartialComposeModuleFieldUpdate updates one or more existing rows in compose_module_field
func (s Store) PartialComposeModuleFieldUpdate(ctx context.Context, onlyColumns []string, rr ...*types.ModuleField) (err error) {
	for _, res := range rr {
		err = s.checkComposeModuleFieldConstraints(ctx, res)
		if err != nil {
			return err
		}

		// err = s.composeModuleFieldHook(ctx, TriggerBeforeComposeModuleFieldUpdate, res)
		// if err != nil {
		// 	return err
		// }

		err = s.execUpdateComposeModuleFields(
			ctx,
			squirrel.Eq{
				s.preprocessColumn("cmf.id", ""): s.preprocessValue(res.ID, ""),
			},
			s.internalComposeModuleFieldEncoder(res).Skip("id").Only(onlyColumns...))
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return
}

// UpsertComposeModuleField updates one or more existing rows in compose_module_field
func (s Store) UpsertComposeModuleField(ctx context.Context, rr ...*types.ModuleField) (err error) {
	for _, res := range rr {
		err = s.checkComposeModuleFieldConstraints(ctx, res)
		if err != nil {
			return err
		}

		// err = s.composeModuleFieldHook(ctx, TriggerBeforeComposeModuleFieldUpsert, res)
		// if err != nil {
		// 	return err
		// }

		err = s.config.ErrorHandler(s.execUpsertComposeModuleFields(ctx, s.internalComposeModuleFieldEncoder(res)))
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteComposeModuleField Deletes one or more rows from compose_module_field table
func (s Store) DeleteComposeModuleField(ctx context.Context, rr ...*types.ModuleField) (err error) {
	for _, res := range rr {
		// err = s.composeModuleFieldHook(ctx, TriggerBeforeComposeModuleFieldDelete, res)
		// if err != nil {
		// 	return err
		// }

		err = s.execDeleteComposeModuleFields(ctx, squirrel.Eq{
			s.preprocessColumn("cmf.id", ""): s.preprocessValue(res.ID, ""),
		})
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return nil
}

// DeleteComposeModuleFieldByID Deletes row from the compose_module_field table
func (s Store) DeleteComposeModuleFieldByID(ctx context.Context, ID uint64) error {
	return s.execDeleteComposeModuleFields(ctx, squirrel.Eq{
		s.preprocessColumn("cmf.id", ""): s.preprocessValue(ID, ""),
	})
}

// TruncateComposeModuleFields Deletes all rows from the compose_module_field table
func (s Store) TruncateComposeModuleFields(ctx context.Context) error {
	return s.config.ErrorHandler(s.Truncate(ctx, s.composeModuleFieldTable()))
}

// execLookupComposeModuleField prepares ComposeModuleField query and executes it,
// returning types.ModuleField (or error)
func (s Store) execLookupComposeModuleField(ctx context.Context, cnd squirrel.Sqlizer) (res *types.ModuleField, err error) {
	var (
		row rowScanner
	)

	row, err = s.QueryRow(ctx, s.composeModuleFieldsSelectBuilder().Where(cnd))
	if err != nil {
		return
	}

	res, err = s.internalComposeModuleFieldRowScanner(row)
	if err != nil {
		return
	}

	return res, nil
}

// execCreateComposeModuleFields updates all matched (by cnd) rows in compose_module_field with given data
func (s Store) execCreateComposeModuleFields(ctx context.Context, payload store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.InsertBuilder(s.composeModuleFieldTable()).SetMap(payload)))
}

// execUpdateComposeModuleFields updates all matched (by cnd) rows in compose_module_field with given data
func (s Store) execUpdateComposeModuleFields(ctx context.Context, cnd squirrel.Sqlizer, set store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.UpdateBuilder(s.composeModuleFieldTable("cmf")).Where(cnd).SetMap(set)))
}

// execUpsertComposeModuleFields inserts new or updates matching (by-primary-key) rows in compose_module_field with given data
func (s Store) execUpsertComposeModuleFields(ctx context.Context, set store.Payload) error {
	upsert, err := s.config.UpsertBuilder(
		s.config,
		s.composeModuleFieldTable(),
		set,
		"id",
	)

	if err != nil {
		return err
	}

	return s.config.ErrorHandler(s.Exec(ctx, upsert))
}

// execDeleteComposeModuleFields Deletes all matched (by cnd) rows in compose_module_field with given data
func (s Store) execDeleteComposeModuleFields(ctx context.Context, cnd squirrel.Sqlizer) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.DeleteBuilder(s.composeModuleFieldTable("cmf")).Where(cnd)))
}

func (s Store) internalComposeModuleFieldRowScanner(row rowScanner) (res *types.ModuleField, err error) {
	res = &types.ModuleField{}

	if _, has := s.config.RowScanners["composeModuleField"]; has {
		scanner := s.config.RowScanners["composeModuleField"].(func(_ rowScanner, _ *types.ModuleField) error)
		err = scanner(row, res)
	} else {
		err = row.Scan(
			&res.ID,
			&res.Name,
			&res.ModuleID,
			&res.Place,
			&res.Kind,
			&res.Label,
			&res.Options,
			&res.Private,
			&res.Required,
			&res.Visible,
			&res.Multi,
			&res.DefaultValue,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.DeletedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("could not scan db row for ComposeModuleField: %w", err)
	} else {
		return res, nil
	}
}

// QueryComposeModuleFields returns squirrel.SelectBuilder with set table and all columns
func (s Store) composeModuleFieldsSelectBuilder() squirrel.SelectBuilder {
	return s.SelectBuilder(s.composeModuleFieldTable("cmf"), s.composeModuleFieldColumns("cmf")...)
}

// composeModuleFieldTable name of the db table
func (Store) composeModuleFieldTable(aa ...string) string {
	var alias string
	if len(aa) > 0 {
		alias = " AS " + aa[0]
	}

	return "compose_module_field" + alias
}

// ComposeModuleFieldColumns returns all defined table columns
//
// With optional string arg, all columns are returned aliased
func (Store) composeModuleFieldColumns(aa ...string) []string {
	var alias string
	if len(aa) > 0 {
		alias = aa[0] + "."
	}

	return []string{
		alias + "id",
		alias + "name",
		alias + "rel_module",
		alias + "place",
		alias + "kind",
		alias + "label",
		alias + "options",
		alias + "is_private",
		alias + "is_required",
		alias + "is_visible",
		alias + "is_multi",
		alias + "default_value",
		alias + "created_at",
		alias + "updated_at",
		alias + "deleted_at",
	}
}

// {true true false false false}

// internalComposeModuleFieldEncoder encodes fields from types.ModuleField to store.Payload (map)
//
// Encoding is done by using generic approach or by calling encodeComposeModuleField
// func when rdbms.customEncoder=true
func (s Store) internalComposeModuleFieldEncoder(res *types.ModuleField) store.Payload {
	return store.Payload{
		"id":            res.ID,
		"name":          res.Name,
		"rel_module":    res.ModuleID,
		"place":         res.Place,
		"kind":          res.Kind,
		"label":         res.Label,
		"options":       res.Options,
		"is_private":    res.Private,
		"is_required":   res.Required,
		"is_visible":    res.Visible,
		"is_multi":      res.Multi,
		"default_value": res.DefaultValue,
		"created_at":    res.CreatedAt,
		"updated_at":    res.UpdatedAt,
		"deleted_at":    res.DeletedAt,
	}
}

func (s *Store) checkComposeModuleFieldConstraints(ctx context.Context, res *types.ModuleField) error {

	{
		ex, err := s.LookupComposeModuleFieldByModuleIDName(ctx, res.ModuleID, res.Name)
		if err == nil && ex != nil && ex.ID != res.ID {
			return store.ErrNotUnique
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
	}

	return nil
}

// func (s *Store) composeModuleFieldHook(ctx context.Context, key triggerKey, res *types.ModuleField) error {
// 	if fn, has := s.config.TriggerHandlers[key]; has {
// 		return fn.(func (ctx context.Context, s *Store, res *types.ModuleField) error)(ctx, s, res)
// 	}
//
// 	return nil
// }
