// Copyright 2018 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package sqlbuilder

import (
	"fmt"
	"strconv"
)

const (
	updateMarkerInit injectionMarker = iota
	updateMarkerAfterWith
	updateMarkerAfterUpdate
	updateMarkerAfterSet
	updateMarkerAfterWhere
	updateMarkerAfterOrderBy
	updateMarkerAfterLimit
)

// NewUpdateBuilder creates a new UPDATE builder.
func NewUpdateBuilder() *UpdateBuilder {
	return DefaultFlavor.NewUpdateBuilder()
}

func newUpdateBuilder() *UpdateBuilder {
	args := &Args{}
	proxy := &whereClauseProxy{}
	return &UpdateBuilder{
		whereClauseProxy: proxy,
		whereClauseExpr:  args.Add(proxy),

		Cond: Cond{
			Args: args,
		},
		limit:     -1,
		args:      args,
		injection: newInjection(),
	}
}

// UpdateBuilder is a builder to build UPDATE.
type UpdateBuilder struct {
	*WhereClause
	Cond

	whereClauseProxy *whereClauseProxy
	whereClauseExpr  string

	cteBuilder  string
	table       string
	assignments []string
	orderByCols []string
	order       string
	limit       int

	args *Args

	injection *injection
	marker    injectionMarker
}

var _ Builder = new(UpdateBuilder)

// Update sets table name in UPDATE.
func Update(table string) *UpdateBuilder {
	return DefaultFlavor.NewUpdateBuilder().Update(table)
}

// With sets WITH clause (the Common Table Expression) before UPDATE.
func (ub *UpdateBuilder) With(builder *CTEBuilder) *UpdateBuilder {
	ub.marker = updateMarkerAfterWith
	ub.cteBuilder = ub.Var(builder)
	return ub
}

// Update sets table name in UPDATE.
func (ub *UpdateBuilder) Update(table string) *UpdateBuilder {
	ub.table = Escape(table)
	ub.marker = updateMarkerAfterUpdate
	return ub
}

// Set sets the assignments in SET.
func (ub *UpdateBuilder) Set(assignment ...string) *UpdateBuilder {
	ub.assignments = assignment
	ub.marker = updateMarkerAfterSet
	return ub
}

// SetMore appends the assignments in SET.
func (ub *UpdateBuilder) SetMore(assignment ...string) *UpdateBuilder {
	ub.assignments = append(ub.assignments, assignment...)
	ub.marker = updateMarkerAfterSet
	return ub
}

// Where sets expressions of WHERE in UPDATE.
func (ub *UpdateBuilder) Where(andExpr ...string) *UpdateBuilder {
	if len(andExpr) == 0 || estimateStringsBytes(andExpr) == 0 {
		return ub
	}

	if ub.WhereClause == nil {
		ub.WhereClause = NewWhereClause()
	}

	ub.WhereClause.AddWhereExpr(ub.args, andExpr...)
	ub.marker = updateMarkerAfterWhere
	return ub
}

// AddWhereClause adds all clauses in the whereClause to SELECT.
func (ub *UpdateBuilder) AddWhereClause(whereClause *WhereClause) *UpdateBuilder {
	if ub.WhereClause == nil {
		ub.WhereClause = NewWhereClause()
	}

	ub.WhereClause.AddWhereClause(whereClause)
	return ub
}

// Assign represents SET "field = value" in UPDATE.
func (ub *UpdateBuilder) Assign(field string, value interface{}) string {
	return fmt.Sprintf("%s = %s", Escape(field), ub.args.Add(value))
}

// Incr represents SET "field = field + 1" in UPDATE.
func (ub *UpdateBuilder) Incr(field string) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s + 1", f, f)
}

// Decr represents SET "field = field - 1" in UPDATE.
func (ub *UpdateBuilder) Decr(field string) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s - 1", f, f)
}

// Add represents SET "field = field + value" in UPDATE.
func (ub *UpdateBuilder) Add(field string, value interface{}) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s + %s", f, f, ub.args.Add(value))
}

// Sub represents SET "field = field - value" in UPDATE.
func (ub *UpdateBuilder) Sub(field string, value interface{}) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s - %s", f, f, ub.args.Add(value))
}

// Mul represents SET "field = field * value" in UPDATE.
func (ub *UpdateBuilder) Mul(field string, value interface{}) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s * %s", f, f, ub.args.Add(value))
}

// Div represents SET "field = field / value" in UPDATE.
func (ub *UpdateBuilder) Div(field string, value interface{}) string {
	f := Escape(field)
	return fmt.Sprintf("%s = %s / %s", f, f, ub.args.Add(value))
}

// OrderBy sets columns of ORDER BY in UPDATE.
func (ub *UpdateBuilder) OrderBy(col ...string) *UpdateBuilder {
	ub.orderByCols = col
	ub.marker = updateMarkerAfterOrderBy
	return ub
}

// Asc sets order of ORDER BY to ASC.
func (ub *UpdateBuilder) Asc() *UpdateBuilder {
	ub.order = "ASC"
	ub.marker = updateMarkerAfterOrderBy
	return ub
}

// Desc sets order of ORDER BY to DESC.
func (ub *UpdateBuilder) Desc() *UpdateBuilder {
	ub.order = "DESC"
	ub.marker = updateMarkerAfterOrderBy
	return ub
}

// Limit sets the LIMIT in UPDATE.
func (ub *UpdateBuilder) Limit(limit int) *UpdateBuilder {
	ub.limit = limit
	ub.marker = updateMarkerAfterLimit
	return ub
}

// NumAssignment returns the number of assignments to update.
func (ub *UpdateBuilder) NumAssignment() int {
	return len(ub.assignments)
}

// String returns the compiled UPDATE string.
func (ub *UpdateBuilder) String() string {
	s, _ := ub.Build()
	return s
}

// Build returns compiled UPDATE string and args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (ub *UpdateBuilder) Build() (sql string, args []interface{}) {
	return ub.BuildWithFlavor(ub.args.Flavor)
}

// BuildWithFlavor returns compiled UPDATE string and args with flavor and initial args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (ub *UpdateBuilder) BuildWithFlavor(flavor Flavor, initialArg ...interface{}) (sql string, args []interface{}) {
	buf := newStringBuilder()
	ub.injection.WriteTo(buf, updateMarkerInit)

	if ub.cteBuilder != "" {
		buf.WriteLeadingString(ub.cteBuilder)
		ub.injection.WriteTo(buf, updateMarkerAfterWith)
	}

	if len(ub.table) > 0 {
		buf.WriteLeadingString("UPDATE ")
		buf.WriteString(ub.table)
	}

	ub.injection.WriteTo(buf, updateMarkerAfterUpdate)

	if len(ub.assignments) > 0 {
		buf.WriteLeadingString("SET ")
		buf.WriteStrings(ub.assignments, ", ")
	}

	ub.injection.WriteTo(buf, updateMarkerAfterSet)

	if ub.WhereClause != nil {
		ub.whereClauseProxy.WhereClause = ub.WhereClause
		defer func() {
			ub.whereClauseProxy.WhereClause = nil
		}()

		buf.WriteLeadingString(ub.whereClauseExpr)
		ub.injection.WriteTo(buf, updateMarkerAfterWhere)
	}

	if len(ub.orderByCols) > 0 {
		buf.WriteLeadingString("ORDER BY ")
		buf.WriteStrings(ub.orderByCols, ", ")

		if ub.order != "" {
			buf.WriteLeadingString(ub.order)
		}

		ub.injection.WriteTo(buf, updateMarkerAfterOrderBy)
	}

	if ub.limit >= 0 {
		buf.WriteLeadingString("LIMIT ")
		buf.WriteString(strconv.Itoa(ub.limit))

		ub.injection.WriteTo(buf, updateMarkerAfterLimit)
	}

	return ub.args.CompileWithFlavor(buf.String(), flavor, initialArg...)
}

// SetFlavor sets the flavor of compiled sql.
func (ub *UpdateBuilder) SetFlavor(flavor Flavor) (old Flavor) {
	old = ub.args.Flavor
	ub.args.Flavor = flavor
	return
}

// Flavor returns flavor of builder
func (ub *UpdateBuilder) Flavor() Flavor {
	return ub.args.Flavor
}

// SQL adds an arbitrary sql to current position.
func (ub *UpdateBuilder) SQL(sql string) *UpdateBuilder {
	ub.injection.SQL(ub.marker, sql)
	return ub
}
