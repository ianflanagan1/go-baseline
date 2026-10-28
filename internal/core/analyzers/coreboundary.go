package analyzers

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// CoreBoundary returns the analyzer that keeps I/O from reaching the core
// through the values passed to it. Wherever code outside the core passes a
// value to the core as an interface, a function or a type argument, or
// stores one in a core type's field, the value's concrete type must be
// pure: the core's own, a pure standard library type or a pure third-party
// type. A value whose concrete type can't be known at that point, such as a
// port, is reported too.
func CoreBoundary(p Project) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "coreboundary",
		Doc:      "reports values that may do I/O passed into the core",
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			if _, own := p.role(pass.Pkg.Path()); !own || p.isCore(pass.Pkg) {
				return nil, nil
			}
			b := boundary{p, pass}
			for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
				if !isTestFile(pass, fn.Pos()) {
					b.checkFunction(fn)
				}
			}
			return nil, nil
		},
	}
}

type boundary struct {
	project Project
	pass    *analysis.Pass
}

func (b boundary) checkFunction(fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr := instr.(type) {
			case ssa.CallInstruction:
				b.checkCall(instr)
			case *ssa.Store:
				b.checkStore(instr)
			}
		}
	}
}

// checkCall checks the arguments, and type arguments, of a call to the core.
func (b boundary) checkCall(call ssa.CallInstruction) {
	common := call.Common()
	var callee *types.Package
	if common.IsInvoke() {
		callee = common.Method.Pkg()
	} else if fn := common.StaticCallee(); fn != nil {
		callee = functionPackage(fn)
		if b.project.isCore(callee) {
			b.checkTypeArgs(fn, call.Pos())
		}
	}
	if !b.project.isCore(callee) {
		return
	}
	params := common.Signature().Params()
	// A static method call passes its receiver first; the receiver is the
	// core's own.
	offset := len(common.Args) - params.Len()
	for i := range params.Len() {
		arg, param := common.Args[offset+i], params.At(i).Type()
		if common.Signature().Variadic() && i == params.Len()-1 {
			b.checkVariadic(arg, param, call.Pos())
			continue
		}
		b.checkValue(arg, param, call.Pos())
	}
}

// checkVariadic checks the values passed as a variadic parameter. Go packs
// them into an array allocated at the call, so each value is stored to an
// element of it. A slice passed with ... is reported, because the values in
// it aren't visible here.
func (b boundary) checkVariadic(arg ssa.Value, param types.Type, pos token.Pos) {
	elem := param.Underlying().(*types.Slice).Elem()
	if !canHideIO(elem) {
		return
	}
	switch arg := arg.(type) {
	case *ssa.Const:
		return
	case *ssa.Slice:
		if array, ok := arg.X.(*ssa.Alloc); ok {
			for _, element := range *array.Referrers() {
				b.checkElementStores(element, elem, pos)
			}
			return
		}
	}
	b.pass.Reportf(pos, "a slice of %s made outside the core is passed to it; pass the values one by one, or values the shell has read", elem)
}

func (b boundary) checkElementStores(element ssa.Instruction, elem types.Type, pos token.Pos) {
	index, ok := element.(*ssa.IndexAddr)
	if !ok {
		return
	}
	for _, ref := range *index.Referrers() {
		if store, ok := ref.(*ssa.Store); ok && store.Addr == index {
			b.checkValue(store.Val, elem, pos)
		}
	}
}

func (b boundary) checkTypeArgs(fn *ssa.Function, pos token.Pos) {
	for _, arg := range fn.TypeArgs() {
		if hasMethods(arg) && !b.isPureType(arg) {
			b.pass.Reportf(pos, "type argument %s has methods outside the core, which may do I/O; the core takes only pure types", arg)
		}
	}
}

// checkStore checks values stored in the fields of the core's types.
func (b boundary) checkStore(store *ssa.Store) {
	field, ok := store.Addr.(*ssa.FieldAddr)
	if !ok {
		return
	}
	pointer, ok := field.X.Type().Underlying().(*types.Pointer)
	if !ok {
		return
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	if !ok || !b.project.isCore(named.Obj().Pkg()) {
		return
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return
	}
	b.checkValue(store.Val, structType.Field(field.Field).Type(), store.Pos())
}

// checkValue reports v when it reaches dest, an interface or function type
// in the core, and may do I/O.
func (b boundary) checkValue(v ssa.Value, dest types.Type, pos token.Pos) {
	if !canHideIO(dest) {
		return
	}
	if pos == token.NoPos {
		pos = v.Pos()
	}
	switch v := v.(type) {
	case *ssa.Const:
	case *ssa.MakeInterface:
		if _, isFunc := v.X.Type().Underlying().(*types.Signature); isFunc {
			b.checkValue(v.X, v.X.Type(), pos)
		} else if hasMethods(v.X.Type()) && !b.isPureType(v.X.Type()) {
			b.pass.Reportf(pos, "%s may do I/O and is passed to the core as %s; the core takes only pure types", v.X.Type(), dest)
		}
	case *ssa.Function:
		if !b.project.isCore(functionPackage(v)) {
			b.pass.Reportf(pos, "function %s is declared outside the core and passed to it; the core takes only its own functions", v.Name())
		}
	case *ssa.MakeClosure:
		b.pass.Reportf(pos, "a closure made outside the core is passed to it; pass the values it would read instead")
	default:
		b.pass.Reportf(pos, "a value of interface type %s, whose concrete type isn't known here, is passed to the core; pass values the shell has read instead", dest)
	}
}

// canHideIO reports whether a value of type t can run code that isn't
// visible where it's used: an interface other than error, or a function.
func canHideIO(t types.Type) bool {
	if types.Identical(t, types.Universe.Lookup("error").Type()) {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Interface, *types.Signature:
		return true
	default:
		return false
	}
}

// functionPackage returns the package that declares fn, or the function it
// was made from, or nil for a synthetic function.
func functionPackage(fn *ssa.Function) *types.Package {
	switch {
	case fn.Object() != nil:
		return fn.Object().Pkg()
	case fn.Parent() != nil:
		return functionPackage(fn.Parent())
	case fn.Origin() != nil:
		return functionPackage(fn.Origin())
	case fn.Pkg != nil:
		return fn.Pkg.Pkg
	default:
		return nil
	}
}

func hasMethods(t types.Type) bool {
	if _, isPointer := t.Underlying().(*types.Pointer); !isPointer {
		t = types.NewPointer(t)
	}
	return types.NewMethodSet(t).Len() > 0
}

// isPureType reports whether t is declared in a pure package, with any type
// arguments pure too.
func (b boundary) isPureType(t types.Type) bool {
	for {
		pointer, ok := t.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		t = pointer.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return true
	}
	if !b.project.isPurePackage(named.Obj().Pkg()) {
		return false
	}
	for arg := range named.TypeArgs().Types() {
		if hasMethods(arg) && !b.isPureType(arg) {
			return false
		}
	}
	return true
}
