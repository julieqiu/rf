// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"rsc.io/rf/refactor"
)

func cmdKey(snap *refactor.Snapshot, argsText string) (more []string, exp bool) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		snap.ErrorAt(token.NoPos, "usage: key StructType...")
		return
	}

	fixing := make(map[types.Type]bool)
	for _, arg := range args {
		item := snap.Lookup(arg)
		if item == nil {
			snap.ErrorAt(token.NoPos, "cannot find %s", arg)
			continue
		}
		if item.Kind != refactor.ItemType {
			snap.ErrorAt(token.NoPos, "%s is not a type", arg)
			continue
		}
		typ := item.Obj.(*types.TypeName).Type().(*types.Named)
		if _, ok := typ.Underlying().(*types.Struct); !ok {
			snap.ErrorAt(token.NoPos, "%s is not a struct type", arg)
			continue
		}
		fixing[typ] = true
	}
	if snap.Errors() > 0 {
		return
	}

	snap.ForEachTargetFile(func(pkg *packages.Package, file *ast.File) {
		refactor.Walk(file, func(stack []ast.Node) {
			lit, ok := stack[0].(*ast.CompositeLit)
			if !ok || len(lit.Elts) == 0 || lit.Incomplete {
				return
			}
			if _, ok := lit.Elts[0].(*ast.KeyValueExpr); ok {
				// already keyed
				return
			}
			typ := pkg.TypesInfo.TypeOf(lit)
			if !fixing[typ] {
				return
			}
			struc := typ.Underlying().(*types.Struct)
			if struc.NumFields() != len(lit.Elts) {
				snap.ErrorAt(lit.Pos(), "wrong number of struct literal initializers")
				return
			}
			for i, e := range lit.Elts {
				f := struc.Field(i)
				snap.InsertAt(e.Pos(), f.Name()+":")
			}
		})
	})

	return nil, false
}
