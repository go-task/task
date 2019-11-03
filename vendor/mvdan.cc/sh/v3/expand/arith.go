// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package expand

import (
	"fmt"
	"strconv"

	"mvdan.cc/sh/v3/syntax"
)

func Arithm(cfg *Config, expr syntax.ArithmExpr) (int, error) {
	switch x := expr.(type) {
	case *syntax.Word:
		str, err := Literal(cfg, x)
		if err != nil {
			return 0, err
		}
		// recursively fetch vars
		i := 0
		for syntax.ValidName(str) {
			val := cfg.envGet(str)
			if val == "" {
				break
			}
			if i++; i >= maxNameRefDepth {
				break
			}
			str = val
		}
		// default to 0
		return atoi(str), nil
	case *syntax.ParenArithm:
		return Arithm(cfg, x.X)
	case *syntax.UnaryArithm:
		switch x.Op {
		case syntax.Inc, syntax.Dec:
			name := x.X.(*syntax.Word).Lit()
			old := atoi(cfg.envGet(name))
			val := old
			if x.Op == syntax.Inc {
				val++
			} else {
				val--
			}
			if err := cfg.envSet(name, strconv.Itoa(val)); err != nil {
				return 0, err
			}
			if x.Post {
				return old, nil
			}
			return val, nil
		}
		val, err := Arithm(cfg, x.X)
		if err != nil {
			return 0, err
		}
		switch x.Op {
		case syntax.Not:
			return oneIf(val == 0), nil
		case syntax.BitNegation:
			return ^val, nil
		case syntax.Plus:
			return val, nil
		default: // syntax.Minus
			return -val, nil
		}
	case *syntax.BinaryArithm:
		switch x.Op {
		case syntax.Assgn, syntax.AddAssgn, syntax.SubAssgn,
			syntax.MulAssgn, syntax.QuoAssgn, syntax.RemAssgn,
			syntax.AndAssgn, syntax.OrAssgn, syntax.XorAssgn,
			syntax.ShlAssgn, syntax.ShrAssgn:
			return cfg.assgnArit(x)
		case syntax.TernQuest: // TernColon can't happen here
			cond, err := Arithm(cfg, x.X)
			if err != nil {
				return 0, err
			}
			b2 := x.Y.(*syntax.BinaryArithm) // must have Op==TernColon
			if cond == 1 {
				return Arithm(cfg, b2.X)
			}
			return Arithm(cfg, b2.Y)
		}
		left, err := Arithm(cfg, x.X)
		if err != nil {
			return 0, err
		}
		right, err := Arithm(cfg, x.Y)
		if err != nil {
			return 0, err
		}
		return binArit(x.Op, left, right), nil
	default:
		panic(fmt.Sprintf("unexpected arithm expr: %T", x))
	}
}

func oneIf(b bool) int {
	if b {
		return 1
	}
	return 0
}

// atoi is just a shorthand for strconv.Atoi that ignores the error,
// just like shells do.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func (cfg *Config) assgnArit(b *syntax.BinaryArithm) (int, error) {
	name := b.X.(*syntax.Word).Lit()
	val := atoi(cfg.envGet(name))
	arg, err := Arithm(cfg, b.Y)
	if err != nil {
		return 0, err
	}
	switch b.Op {
	case syntax.Assgn:
		val = arg
	case syntax.AddAssgn:
		val += arg
	case syntax.SubAssgn:
		val -= arg
	case syntax.MulAssgn:
		val *= arg
	case syntax.QuoAssgn:
		val /= arg
	case syntax.RemAssgn:
		val %= arg
	case syntax.AndAssgn:
		val &= arg
	case syntax.OrAssgn:
		val |= arg
	case syntax.XorAssgn:
		val ^= arg
	case syntax.ShlAssgn:
		val <<= uint(arg)
	case syntax.ShrAssgn:
		val >>= uint(arg)
	}
	if err := cfg.envSet(name, strconv.Itoa(val)); err != nil {
		return 0, err
	}
	return val, nil
}

func intPow(a, b int) int {
	p := 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

func binArit(op syntax.BinAritOperator, x, y int) int {
	switch op {
	case syntax.Add:
		return x + y
	case syntax.Sub:
		return x - y
	case syntax.Mul:
		return x * y
	case syntax.Quo:
		return x / y
	case syntax.Rem:
		return x % y
	case syntax.Pow:
		return intPow(x, y)
	case syntax.Eql:
		return oneIf(x == y)
	case syntax.Gtr:
		return oneIf(x > y)
	case syntax.Lss:
		return oneIf(x < y)
	case syntax.Neq:
		return oneIf(x != y)
	case syntax.Leq:
		return oneIf(x <= y)
	case syntax.Geq:
		return oneIf(x >= y)
	case syntax.And:
		return x & y
	case syntax.Or:
		return x | y
	case syntax.Xor:
		return x ^ y
	case syntax.Shr:
		return x >> uint(y)
	case syntax.Shl:
		return x << uint(y)
	case syntax.AndArit:
		return oneIf(x != 0 && y != 0)
	case syntax.OrArit:
		return oneIf(x != 0 || y != 0)
	default: // syntax.Comma
		// x is executed but its result discarded
		return y
	}
}
