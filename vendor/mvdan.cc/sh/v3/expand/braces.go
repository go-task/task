// Copyright (c) 2018, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package expand

import (
	"strconv"

	"mvdan.cc/sh/v3/syntax"
)

// Braces performs brace expansion on a word, given that it contains any
// syntax.BraceExp parts. For example, the word with a brace expansion
// "foo{bar,baz}" will return two literal words, "foobar" and "foobaz".
//
// Note that the resulting words may share word parts.
func Braces(word *syntax.Word) []*syntax.Word {
	var all []*syntax.Word
	var left []syntax.WordPart
	for i, wp := range word.Parts {
		br, ok := wp.(*syntax.BraceExp)
		if !ok {
			left = append(left, wp.(syntax.WordPart))
			continue
		}
		if br.Sequence {
			chars := false
			from, err1 := strconv.Atoi(br.Elems[0].Lit())
			to, err2 := strconv.Atoi(br.Elems[1].Lit())
			if err1 != nil || err2 != nil {
				chars = true
				from = int(br.Elems[0].Lit()[0])
				to = int(br.Elems[1].Lit()[0])
			}
			upward := from <= to
			incr := 1
			if !upward {
				incr = -1
			}
			if len(br.Elems) > 2 {
				n, _ := strconv.Atoi(br.Elems[2].Lit())
				if n != 0 && n > 0 == upward {
					incr = n
				}
			}
			n := from
			for {
				if upward && n > to {
					break
				}
				if !upward && n < to {
					break
				}
				next := *word
				next.Parts = next.Parts[i+1:]
				lit := &syntax.Lit{}
				if chars {
					lit.Value = string(n)
				} else {
					lit.Value = strconv.Itoa(n)
				}
				next.Parts = append([]syntax.WordPart{lit}, next.Parts...)
				exp := Braces(&next)
				for _, w := range exp {
					w.Parts = append(left, w.Parts...)
				}
				all = append(all, exp...)
				n += incr
			}
			return all
		}
		for _, elem := range br.Elems {
			next := *word
			next.Parts = next.Parts[i+1:]
			next.Parts = append(elem.Parts, next.Parts...)
			exp := Braces(&next)
			for _, w := range exp {
				w.Parts = append(left, w.Parts...)
			}
			all = append(all, exp...)
		}
		return all
	}
	return []*syntax.Word{{Parts: left}}
}
