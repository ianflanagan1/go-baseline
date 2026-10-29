package analyzers

import (
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// PortBranch returns the analyzer that keeps decisions out of the shell. The
// shell may branch on what the core returns and on whether an error is nil,
// but not on a value read through a port: it passes that value to the core
// and acts on what the core returns.
//
// A value is read through a port when it comes from a method of an
// interface the core doesn't declare, or from a call to a function value.
// Anything computed from it outside the core is too, including a struct,
// slice or map it's stored in and what a call outside the core writes
// through a pointer passed with it. Ranging over one is allowed: doing the
// same thing for each element isn't a decision. A value passed to another
// shell function that branches on it is reported where it's passed, using
// facts for functions in other packages.
func PortBranch(p Project) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:      "portbranch",
		Doc:       "reports the shell branching on values read through ports",
		Requires:  []*analysis.Analyzer{buildssa.Analyzer},
		FactTypes: []analysis.Fact{new(shellSummary)},
		Run: func(pass *analysis.Pass) (any, error) {
			if role, own := p.role(pass.Pkg.Path()); !own || role != baseline.Shell {
				return nil, nil
			}
			var fns []*ssa.Function
			for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
				if !isTestFile(pass, fn.Pos()) {
					fns = append(fns, fn)
				}
			}
			s := shell{project: p, pass: pass, summaries: map[*ssa.Function]*shellSummary{}}
			s.summarise(fns)
			reported := map[token.Pos]bool{}
			for _, fn := range fns {
				s.check(fn, nil, reported)
			}
			s.exportFacts(fns)
			return nil, nil
		},
	}
}

// shellSummary is what a shell function's callers need to know about it:
// whether it returns a value read through a port, which parameters it
// branches on and which it passes on to its results.
type shellSummary struct {
	Returns bool
	Decides []bool
	Passes  []bool
}

func (*shellSummary) AFact() {}

func (s *shellSummary) String() string {
	var parts []string
	if s.Returns {
		parts = append(parts, "returns a port value")
	}
	for i, decides := range s.Decides {
		if decides {
			parts = append(parts, fmt.Sprintf("branches on parameter %d", i))
		}
	}
	for i, passes := range s.Passes {
		if passes {
			parts = append(parts, fmt.Sprintf("returns parameter %d", i))
		}
	}
	return strings.Join(parts, "; ")
}

func (s *shellSummary) empty() bool {
	return !s.Returns && !slices.Contains(s.Decides, true) && !slices.Contains(s.Passes, true)
}

type shell struct {
	project   Project
	pass      *analysis.Pass
	summaries map[*ssa.Function]*shellSummary
}

// decision is a place where the shell decides on a value read through a
// port.
type decision struct {
	pos     token.Pos
	message string
}

// summarise works out the package's summaries together, until they stop
// changing, since its functions may call each other.
func (s shell) summarise(fns []*ssa.Function) {
	for _, fn := range fns {
		s.summaries[fn] = &shellSummary{}
	}
	for changed := true; changed; {
		changed = false
		for _, fn := range fns {
			next := s.summary(fn)
			if next.String() != s.summaries[fn].String() {
				*s.summaries[fn] = next
				changed = true
			}
		}
	}
}

func (s shell) summary(fn *ssa.Function) shellSummary {
	sum := shellSummary{Returns: returnsTainted(fn, s.taint(fn, nil, true))}
	for _, param := range fn.Params {
		tainted := s.taint(fn, []ssa.Value{param}, false)
		sum.Decides = append(sum.Decides, len(s.decisions(fn, tainted)) > 0)
		sum.Passes = append(sum.Passes, returnsTainted(fn, tainted))
	}
	return sum
}

func (s shell) exportFacts(fns []*ssa.Function) {
	for _, fn := range fns {
		if obj := fn.Object(); obj != nil && obj.Pkg() == s.pass.Pkg && !s.summaries[fn].empty() {
			s.pass.ExportObjectFact(obj, s.summaries[fn])
		}
	}
}

// check reports fn's decisions. seeds are values already read through a
// port when fn starts: a closure's captured variables.
func (s shell) check(fn *ssa.Function, seeds []ssa.Value, reported map[token.Pos]bool) {
	tainted := s.taint(fn, seeds, true)
	for _, d := range s.decisions(fn, tainted) {
		if !reported[d.pos] {
			reported[d.pos] = true
			s.pass.Report(analysis.Diagnostic{Pos: d.pos, Message: d.message})
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			closure, ok := instr.(*ssa.MakeClosure)
			if !ok {
				continue
			}
			inner := closure.Fn.(*ssa.Function)
			var captured []ssa.Value
			for i, binding := range closure.Bindings {
				if tainted[binding] {
					captured = append(captured, inner.FreeVars[i])
				}
			}
			if len(captured) > 0 {
				s.check(inner, captured, reported)
			}
		}
	}
}

// decisions returns where fn decides on a tainted value.
func (s shell) decisions(fn *ssa.Function, tainted map[ssa.Value]bool) []decision {
	var found []decision
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr := instr.(type) {
			case *ssa.If:
				if tainted[instr.Cond] && !isRangeLoop(block) {
					found = append(found, decision{
						position(instr.Cond, fn),
						"the shell branches on a value read through a port; pass it to the core and act on what the core returns",
					})
				}
			case ssa.CallInstruction:
				found = append(found, s.callDecisions(instr, tainted)...)
			}
		}
	}
	return found
}

func (s shell) callDecisions(call ssa.CallInstruction, tainted map[ssa.Value]bool) []decision {
	common := call.Common()
	if common.IsInvoke() {
		return nil
	}
	callee := common.StaticCallee()
	if callee == nil {
		if _, builtin := common.Value.(*ssa.Builtin); !builtin && tainted[common.Value] {
			return []decision{{call.Pos(), "the shell calls a function chosen by a value read through a port; let the core choose"}}
		}
		return nil
	}
	sum := s.summaryOf(callee)
	if sum == nil {
		return nil
	}
	for i, arg := range common.Args {
		if i < len(sum.Decides) && sum.Decides[i] && tainted[arg] {
			return []decision{{call.Pos(), fmt.Sprintf("%s branches on a value read through a port, passed to it here; pass the value to the core instead", callee.Name())}}
		}
	}
	return nil
}

// summaryOf returns the summary of a shell function, or nil when callee
// isn't one.
func (s shell) summaryOf(callee *ssa.Function) *shellSummary {
	if sum, ok := s.summaries[callee]; ok {
		return sum
	}
	obj := callee.Object()
	if obj == nil || obj.Pkg() == nil {
		return nil
	}
	if role, own := s.project.role(obj.Pkg().Path()); !own || role != baseline.Shell {
		return nil
	}
	// A function with nothing to say has no fact.
	sum := new(shellSummary)
	s.pass.ImportObjectFact(obj, sum)
	return sum
}

// taint returns the values in fn that come from seeds, or, when ports is
// true, from a port.
func (s shell) taint(fn *ssa.Function, seeds []ssa.Value, ports bool) map[ssa.Value]bool {
	tainted := map[ssa.Value]bool{}
	for _, v := range seeds {
		tainted[v] = true
	}
	for changed := true; changed; {
		changed = false
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				for _, v := range s.taints(instr, tainted, ports) {
					if !tainted[v] {
						tainted[v] = true
						changed = true
					}
				}
			}
		}
	}
	return tainted
}

// taints returns the values that instr taints, given those already tainted.
func (s shell) taints(instr ssa.Instruction, tainted map[ssa.Value]bool, ports bool) []ssa.Value {
	switch instr := instr.(type) {
	case *ssa.Store:
		if tainted[instr.Val] {
			return referents(instr.Addr)
		}
		return nil
	case ssa.CallInstruction:
		return s.callTaints(instr, tainted, ports)
	case *ssa.Extract:
		if sel, isSelect := instr.Tuple.(*ssa.Select); isSelect {
			if selectTainted(sel, instr.Index, tainted) {
				return []ssa.Value{instr}
			}
			return nil
		}
	case *ssa.BinOp:
		if isErrorNilCheck(instr) {
			return nil
		}
	}
	if v, ok := instr.(ssa.Value); ok && anyTainted(instr.Operands(nil), tainted) {
		return []ssa.Value{v}
	}
	return nil
}

// callTaints returns what a call taints: its result, and what it may write
// through the pointers passed to it.
func (s shell) callTaints(call ssa.CallInstruction, tainted map[ssa.Value]bool, ports bool) []ssa.Value {
	common := call.Common()
	argsTainted := anyTainted(call.Operands(nil), tainted)
	var result, writes bool
	switch callee := common.StaticCallee(); {
	case common.IsInvoke() && common.Method.Pkg() == nil:
		// error's Error method.
		result = tainted[common.Value]
	case common.IsInvoke():
		// A port's results come from the port, whatever it was given.
		result = ports && !s.project.isCore(common.Method.Pkg())
		writes = result
	case callee == nil:
		if _, builtin := common.Value.(*ssa.Builtin); builtin {
			result = argsTainted
		} else {
			result = ports
		}
		writes = result
	case s.project.isCore(functionPackage(callee)):
		// The core's results are its decisions.
	default:
		result, writes = s.calleeTaints(callee, common.Args, tainted, ports, argsTainted)
	}
	var out []ssa.Value
	if v := call.Value(); v != nil && result {
		out = append(out, v)
	}
	if writes {
		for _, arg := range common.Args {
			out = append(out, referents(arg)...)
		}
	}
	return out
}

// calleeTaints is callTaints for a static call outside the core. A shell
// function's summary says what it returns; anything else is assumed to
// return, and write, what it's given.
func (s shell) calleeTaints(callee *ssa.Function, args []ssa.Value, tainted map[ssa.Value]bool, ports, argsTainted bool) (result, writes bool) {
	sum := s.summaryOf(callee)
	if sum == nil {
		return argsTainted, argsTainted
	}
	result = ports && sum.Returns
	for i, arg := range args {
		if i < len(sum.Passes) && sum.Passes[i] && tainted[arg] {
			result = true
		}
	}
	return result, false
}

// referents returns v and what it was taken from, when v refers to memory
// that a store or a call may write: a pointer, slice or map, or an
// interface holding one.
func referents(v ssa.Value) []ssa.Value {
	if mi, ok := v.(*ssa.MakeInterface); ok {
		v = mi.X
	}
	switch v.Type().Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map:
	default:
		return nil
	}
	out := []ssa.Value{v}
	for {
		switch addr := v.(type) {
		case *ssa.FieldAddr:
			v = addr.X
		case *ssa.IndexAddr:
			v = addr.X
		case *ssa.Slice:
			v = addr.X
		default:
			return out
		}
		out = append(out, v)
	}
}

// selectTainted reports whether the index'th result of sel is tainted. Which
// case ran isn't a value from a port; a received value is tainted when its
// channel is.
func selectTainted(sel *ssa.Select, index int, tainted map[ssa.Value]bool) bool {
	var received []ssa.Value
	for _, state := range sel.States {
		if state.Dir == types.RecvOnly {
			received = append(received, state.Chan)
		}
	}
	switch index {
	case 0:
		return false
	case 1:
		return slices.ContainsFunc(received, func(ch ssa.Value) bool { return tainted[ch] })
	default:
		return tainted[received[index-2]]
	}
}

func anyTainted(operands []*ssa.Value, tainted map[ssa.Value]bool) bool {
	return slices.ContainsFunc(operands, func(op *ssa.Value) bool { return op != nil && tainted[*op] })
}

// isErrorNilCheck reports whether op compares an error with nil, the one
// question about a port's result the shell may ask.
func isErrorNilCheck(op *ssa.BinOp) bool {
	if op.Op != token.EQL && op.Op != token.NEQ {
		return false
	}
	errorType := types.Universe.Lookup("error").Type()
	isNil := func(v ssa.Value) bool { c, ok := v.(*ssa.Const); return ok && c.IsNil() }
	return (isNil(op.X) && types.Identical(op.Y.Type(), errorType)) ||
		(isNil(op.Y) && types.Identical(op.X.Type(), errorType))
}

func returnsTainted(fn *ssa.Function, tainted map[ssa.Value]bool) bool {
	for _, block := range fn.Blocks {
		if ret, ok := block.Instrs[len(block.Instrs)-1].(*ssa.Return); ok && slices.ContainsFunc(ret.Results, func(v ssa.Value) bool { return tainted[v] }) {
			return true
		}
	}
	return false
}

// isRangeLoop reports whether block is the head of a range loop, which
// branches only on whether there's another element.
func isRangeLoop(block *ssa.BasicBlock) bool {
	return strings.HasPrefix(block.Comment, "range") && strings.HasSuffix(block.Comment, ".loop")
}

// position returns where v is written in the source. A value the compiler
// made, such as a φ-node, takes the position of one it was made from.
func position(v ssa.Value, fn *ssa.Function) token.Pos {
	seen := map[ssa.Value]bool{}
	var find func(v ssa.Value) token.Pos
	find = func(v ssa.Value) token.Pos {
		if seen[v] {
			return token.NoPos
		}
		seen[v] = true
		if v.Pos().IsValid() {
			return v.Pos()
		}
		switch v := v.(type) {
		case *ssa.Extract:
			return find(v.Tuple)
		case *ssa.Phi:
			for _, edge := range v.Edges {
				if pos := find(edge); pos.IsValid() {
					return pos
				}
			}
		}
		return token.NoPos
	}
	if pos := find(v); pos.IsValid() {
		return pos
	}
	return fn.Pos()
}
