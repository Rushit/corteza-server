package rdbms

// This file is an auto-generated file
//
// Template:    pkg/codegen/assets/store_rdbms.gen.go.tpl
// Definitions: store/compose_records.yaml
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
	"github.com/cortezaproject/corteza-server/pkg/filter"
	"github.com/cortezaproject/corteza-server/store"
)

var _ = errors.Is

// searchComposeRecords returns all matching rows
//
// This function calls convertComposeRecordFilter with the given
// types.RecordFilter and expects to receive a working squirrel.SelectBuilder
func (s Store) searchComposeRecords(ctx context.Context, _mod *types.Module, f types.RecordFilter) (types.RecordSet, types.RecordFilter, error) {
	var (
		err error
		set []*types.Record
		q   squirrel.SelectBuilder
	)
	q, err = s.convertComposeRecordFilter(_mod, f)
	if err != nil {
		return nil, f, err
	}

	// Cleanup anything we've accidentally received...
	f.PrevPage, f.NextPage = nil, nil

	// When cursor for a previous page is used it's marked as reversed
	// This tells us to flip the descending flag on all used sort keys
	reversedCursor := f.PageCursor != nil && f.PageCursor.Reverse

	// If paging with reverse cursor, change the sorting
	// direction for all columns we're sorting by
	curSort := f.Sort.Clone()
	if reversedCursor {
		curSort.Reverse()
	}

	return set, f, s.config.ErrorHandler(func() error {
		set, err = s.fetchFullPageOfComposeRecords(ctx, _mod, q, curSort, f.PageCursor, f.Limit, f.Check)

		if err != nil {
			return err
		}

		if f.Limit > 0 && len(set) > 0 {
			if f.PageCursor != nil && (!f.PageCursor.Reverse || uint(len(set)) == f.Limit) {
				f.PrevPage = s.collectComposeRecordCursorValues(set[0], curSort.Columns()...)
				f.PrevPage.Reverse = true
			}

			// Less items fetched then requested by page-limit
			// not very likely there's another page
			f.NextPage = s.collectComposeRecordCursorValues(set[len(set)-1], curSort.Columns()...)
		}

		f.PageCursor = nil
		return nil
	}())
}

// fetchFullPageOfComposeRecords collects all requested results.
//
// Function applies:
//  - cursor conditions (where ...)
//  - sorting rules (order by ...)
//  - limit
//
// Main responsibility of this function is to perform additional sequential queries in case when not enough results
// are collected due to failed check on a specific row (by check fn). Function then moves cursor to the last item fetched
func (s Store) fetchFullPageOfComposeRecords(
	ctx context.Context, _mod *types.Module,
	q squirrel.SelectBuilder,
	sort filter.SortExprSet,
	cursor *filter.PagingCursor,
	limit uint,
	check func(*types.Record) (bool, error),
) ([]*types.Record, error) {
	var (
		set  = make([]*types.Record, 0, DefaultSliceCapacity)
		aux  []*types.Record
		last *types.Record

		// When cursor for a previous page is used it's marked as reversed
		// This tells us to flip the descending flag on all used sort keys
		reversedCursor = cursor != nil && cursor.Reverse

		// copy of the select builder
		tryQuery squirrel.SelectBuilder

		fetched uint
		err     error
	)

	// Make sure we always end our sort by primary keys
	if sort.Get("id") == nil {
		sort = append(sort, &filter.SortExpr{Column: "id"})
	}

	if q, err = s.composeRecordsSorter(_mod, q, sort); err != nil {
		return nil, err
	}

	for try := 0; try < MaxRefetches; try++ {
		tryQuery = setCursorCond(q, cursor)
		if limit > 0 {
			tryQuery = tryQuery.Limit(uint64(limit))
		}

		if aux, fetched, last, err = s.QueryComposeRecords(ctx, _mod, tryQuery, check); err != nil {
			return nil, err
		}

		if limit > 0 && uint(len(aux)) >= limit {
			// we should use only as much as requested
			set = append(set, aux[0:limit]...)
			break
		} else {
			set = append(set, aux...)
		}

		// if limit is not set or we've already collected enough items
		// we can break the loop right away
		if limit == 0 || fetched == 0 || fetched < limit {
			break
		}

		// In case limit is set very low and we've missed records in the first fetch,
		// make sure next fetch limit is a bit higher
		if limit < MinEnsureFetchLimit {
			limit = MinEnsureFetchLimit
		}

		// @todo improve strategy for collecting next page with lower limit

		// Point cursor to the last fetched element
		if cursor = s.collectComposeRecordCursorValues(last, sort.Columns()...); cursor == nil {
			break
		}
	}

	if reversedCursor {
		// Cursor for previous page was used
		// Fetched set needs to be reverseCursor because we've forced a descending order to
		// get the previous page
		for i, j := 0, len(set)-1; i < j; i, j = i+1, j-1 {
			set[i], set[j] = set[j], set[i]
		}
	}

	return set, nil
}

// QueryComposeRecords queries the database, converts and checks each row and
// returns collected set
//
// Fn also returns total number of fetched items and last fetched item so that the caller can construct cursor
// for next page of results
func (s Store) QueryComposeRecords(
	ctx context.Context, _mod *types.Module,
	q squirrel.Sqlizer,
	check func(*types.Record) (bool, error),
) ([]*types.Record, uint, *types.Record, error) {
	var (
		set = make([]*types.Record, 0, DefaultSliceCapacity)
		res *types.Record

		// Query rows with
		rows, err = s.Query(ctx, q)

		fetched uint
	)

	if err != nil {
		return nil, 0, nil, err
	}

	defer rows.Close()
	for rows.Next() {
		fetched++
		if err = rows.Err(); err == nil {
			res, err = s.internalComposeRecordRowScanner(_mod, rows)
		}

		if err != nil {
			return nil, 0, nil, err
		}

		// If check function is set, call it and act accordingly
		if check != nil {
			if chk, err := check(res); err != nil {
				return nil, 0, nil, err
			} else if !chk {
				// did not pass the check
				// go with the next row
				continue
			}
		}

		set = append(set, res)
	}

	if err = s.composeRecordPostLoadProcessor(ctx, _mod, set...); err != nil {
		return nil, 0, nil, err
	}

	return set, fetched, res, rows.Err()
}

// lookupComposeRecordByID searches for compose record by ID
// It returns compose record even if deleted
func (s Store) lookupComposeRecordByID(ctx context.Context, _mod *types.Module, id uint64) (*types.Record, error) {
	return s.execLookupComposeRecord(ctx, _mod, squirrel.Eq{
		s.preprocessColumn("crd.id", ""): s.preprocessValue(id, ""),
	})
}

// createComposeRecord creates one or more rows in compose_record table
func (s Store) createComposeRecord(ctx context.Context, _mod *types.Module, rr ...*types.Record) (err error) {
	for _, res := range rr {
		err = s.checkComposeRecordConstraints(ctx, _mod, res)
		if err != nil {
			return err
		}

		err = s.execCreateComposeRecords(ctx, s.internalComposeRecordEncoder(res))
		if err != nil {
			return err
		}
	}

	return
}

// updateComposeRecord updates one or more existing rows in compose_record
func (s Store) updateComposeRecord(ctx context.Context, _mod *types.Module, rr ...*types.Record) error {
	return s.config.ErrorHandler(s.partialComposeRecordUpdate(ctx, _mod, nil, rr...))
}

// partialComposeRecordUpdate updates one or more existing rows in compose_record
func (s Store) partialComposeRecordUpdate(ctx context.Context, _mod *types.Module, onlyColumns []string, rr ...*types.Record) (err error) {
	for _, res := range rr {
		err = s.checkComposeRecordConstraints(ctx, _mod, res)
		if err != nil {
			return err
		}

		err = s.execUpdateComposeRecords(
			ctx,
			squirrel.Eq{
				s.preprocessColumn("crd.id", ""): s.preprocessValue(res.ID, ""),
			},
			s.internalComposeRecordEncoder(res).Skip("id").Only(onlyColumns...))
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return
}

// upsertComposeRecord updates one or more existing rows in compose_record
func (s Store) upsertComposeRecord(ctx context.Context, _mod *types.Module, rr ...*types.Record) (err error) {
	for _, res := range rr {
		err = s.checkComposeRecordConstraints(ctx, _mod, res)
		if err != nil {
			return err
		}

		err = s.config.ErrorHandler(s.execUpsertComposeRecords(ctx, s.internalComposeRecordEncoder(res)))
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteComposeRecord Deletes one or more rows from compose_record table
func (s Store) deleteComposeRecord(ctx context.Context, _mod *types.Module, rr ...*types.Record) (err error) {
	for _, res := range rr {

		err = s.execDeleteComposeRecords(ctx, squirrel.Eq{
			s.preprocessColumn("crd.id", ""): s.preprocessValue(res.ID, ""),
		})
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return nil
}

// deleteComposeRecordByID Deletes row from the compose_record table
func (s Store) deleteComposeRecordByID(ctx context.Context, _mod *types.Module, ID uint64) error {
	return s.execDeleteComposeRecords(ctx, squirrel.Eq{
		s.preprocessColumn("crd.id", ""): s.preprocessValue(ID, ""),
	})
}

// truncateComposeRecords Deletes all rows from the compose_record table
func (s Store) truncateComposeRecords(ctx context.Context, _mod *types.Module) error {
	return s.config.ErrorHandler(s.Truncate(ctx, s.composeRecordTable()))
}

// execLookupComposeRecord prepares ComposeRecord query and executes it,
// returning types.Record (or error)
func (s Store) execLookupComposeRecord(ctx context.Context, _mod *types.Module, cnd squirrel.Sqlizer) (res *types.Record, err error) {
	var (
		row rowScanner
	)

	row, err = s.QueryRow(ctx, s.composeRecordsSelectBuilder().Where(cnd))
	if err != nil {
		return
	}

	res, err = s.internalComposeRecordRowScanner(_mod, row)
	if err != nil {
		return
	}

	if err = s.composeRecordPostLoadProcessor(ctx, _mod, res); err != nil {
		return nil, err
	}

	return res, nil
}

// execCreateComposeRecords updates all matched (by cnd) rows in compose_record with given data
func (s Store) execCreateComposeRecords(ctx context.Context, payload store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.InsertBuilder(s.composeRecordTable()).SetMap(payload)))
}

// execUpdateComposeRecords updates all matched (by cnd) rows in compose_record with given data
func (s Store) execUpdateComposeRecords(ctx context.Context, cnd squirrel.Sqlizer, set store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.UpdateBuilder(s.composeRecordTable("crd")).Where(cnd).SetMap(set)))
}

// execUpsertComposeRecords inserts new or updates matching (by-primary-key) rows in compose_record with given data
func (s Store) execUpsertComposeRecords(ctx context.Context, set store.Payload) error {
	upsert, err := s.config.UpsertBuilder(
		s.config,
		s.composeRecordTable(),
		set,
		"id",
	)

	if err != nil {
		return err
	}

	return s.config.ErrorHandler(s.Exec(ctx, upsert))
}

// execDeleteComposeRecords Deletes all matched (by cnd) rows in compose_record with given data
func (s Store) execDeleteComposeRecords(ctx context.Context, cnd squirrel.Sqlizer) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.DeleteBuilder(s.composeRecordTable("crd")).Where(cnd)))
}

func (s Store) internalComposeRecordRowScanner(_mod *types.Module, row rowScanner) (res *types.Record, err error) {
	res = &types.Record{}

	if _, has := s.config.RowScanners["composeRecord"]; has {
		scanner := s.config.RowScanners["composeRecord"].(func(_mod *types.Module, _ rowScanner, _ *types.Record) error)
		err = scanner(_mod, row, res)
	} else {
		err = row.Scan(
			&res.ID,
			&res.ModuleID,
			&res.NamespaceID,
			&res.OwnedBy,
			&res.CreatedBy,
			&res.UpdatedBy,
			&res.DeletedBy,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.DeletedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("could not scan db row for ComposeRecord: %w", err)
	} else {
		return res, nil
	}
}

// QueryComposeRecords returns squirrel.SelectBuilder with set table and all columns
func (s Store) composeRecordsSelectBuilder() squirrel.SelectBuilder {
	return s.SelectBuilder(s.composeRecordTable("crd"), s.composeRecordColumns("crd")...)
}

// composeRecordTable name of the db table
func (Store) composeRecordTable(aa ...string) string {
	var alias string
	if len(aa) > 0 {
		alias = " AS " + aa[0]
	}

	return "compose_record" + alias
}

// ComposeRecordColumns returns all defined table columns
//
// With optional string arg, all columns are returned aliased
func (Store) composeRecordColumns(aa ...string) []string {
	var alias string
	if len(aa) > 0 {
		alias = aa[0] + "."
	}

	return []string{
		alias + "id",
		alias + "module_id",
		alias + "rel_namespace",
		alias + "owned_by",
		alias + "created_by",
		alias + "updated_by",
		alias + "deleted_by",
		alias + "created_at",
		alias + "updated_at",
		alias + "deleted_at",
	}
}

// {true false true true true}

// sortableComposeRecordColumns returns all ComposeRecord columns flagged as sortable
//
// With optional string arg, all columns are returned aliased
func (Store) sortableComposeRecordColumns() []string {
	return []string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
	}
}

// internalComposeRecordEncoder encodes fields from types.Record to store.Payload (map)
//
// Encoding is done by using generic approach or by calling encodeComposeRecord
// func when rdbms.customEncoder=true
func (s Store) internalComposeRecordEncoder(res *types.Record) store.Payload {
	return store.Payload{
		"id":            res.ID,
		"module_id":     res.ModuleID,
		"rel_namespace": res.NamespaceID,
		"owned_by":      res.OwnedBy,
		"created_by":    res.CreatedBy,
		"updated_by":    res.UpdatedBy,
		"deleted_by":    res.DeletedBy,
		"created_at":    res.CreatedAt,
		"updated_at":    res.UpdatedAt,
		"deleted_at":    res.DeletedAt,
	}
}

// checkComposeRecordConstraints performs lookups (on valid) resource to check if any of the values on unique fields
// already exists in the store
//
// Using built-in constraint checking would be more performant but unfortunately we can not rely
// on the full support (MySQL does not support conditional indexes)
func (s *Store) checkComposeRecordConstraints(ctx context.Context, _mod *types.Module, res *types.Record) error {
	// Consider resource valid when all fields in unique constraint check lookups
	// have valid (non-empty) value
	//
	// Only string and uint64 are supported for now
	// feel free to add additional types if needed
	var valid = true

	if !valid {
		return nil
	}

	return nil
}
