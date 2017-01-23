package gotalog

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"
)

// TODO: consider whether this should actually be the public API,
// rather than the assert/search api?
type makeLiteral struct {
	pName string
	terms []term
}

type datalogCommand struct {
	head makeLiteral
	body []makeLiteral
	// If isQuery is true, head is interpreted as
	isQuery bool
}

func buildLiteral(ml makeLiteral, db *database) literal {
	return literal{
		pred:  db.newPredicate(ml.pName, len(ml.terms)),
		terms: ml.terms,
	}
}

func apply(cmd datalogCommand, db *database) (*result, error) {
	head := buildLiteral(cmd.head, db)
	if cmd.isQuery {
		res, err := ask(head)
		return &res, err
	}
	body := make([]literal, len(cmd.body))
	for i, ml := range cmd.body {
		body[i] = buildLiteral(ml, db)
	}
	err := db.assert(clause{
		head: head,
		body: body,
	})
	return nil, err
}

func applyAll(cmds []datalogCommand, db *database) (results []result, err error) {
	for _, cmd := range cmds {
		res, err := apply(cmd, db)
		if err != nil {
			return results, err
		}
		if res != nil {
			results = append(results, *res)
		}
	}
	return
}

type scanner struct {
	r *bufio.Reader
}

func newScanner(input io.Reader) scanner {
	return scanner{bufio.NewReader(input)}
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isLowerCase(ch rune) bool {
	return (ch >= 'a' && ch <= 'z')
}

func isUpperCase(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z')
}

func isLetter(ch rune) bool {
	return isLowerCase(ch) || isUpperCase(ch)
}

func isAllowedBodyRune(ch rune) bool {
	return (ch >= '0' && ch <= '9') ||
		isLetter(ch) ||
		(ch == '_' || ch == '-')
}

var eof = rune(0)

func (s scanner) mustConsume(r rune) error {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return err
	}
	if ch != r {
		return fmt.Errorf("Expected %v, but got %v", string(r), string(ch))
	}
	return nil
}

func (s scanner) consumeRestOfLine() {
	for {
		ch, _, err := s.r.ReadRune()
		if err != nil || ch == '\n' {
			break
		}
	}
}

func (s scanner) consumeWhitespace() {
	for {
		ch, _, err := s.r.ReadRune()
		if err != nil || !isWhitespace(ch) {
			if ch == '%' {
				s.consumeRestOfLine()
			} else {
				s.r.UnreadRune()
				return
			}
		}
	}
}

func (s scanner) scanIdentifier() (str string, err error) {
	s.consumeWhitespace()
	ch, _, err := s.r.ReadRune()
	if !isLetter(ch) {
		return str, fmt.Errorf("Expected a term composed of letters and numbers, but got %v", string(ch))
	}
	str = str + string(ch)
	for {
		ch, _, err = s.r.ReadRune()

		if !isAllowedBodyRune(ch) {
			s.r.UnreadRune()
			return
		}
		str = str + string(ch)
	}
}

func (s scanner) scanTerm() (t term, err error) {

	t.value, err = s.scanIdentifier()
	if err != nil {
		return t, err
	}
	leading, _ := utf8.DecodeRuneInString(t.value)

	if isLowerCase(leading) {
		t.isConstant = true
	}
	return
}

func (s scanner) scanLiteral() (lit makeLiteral, err error) {
	name, err := s.scanIdentifier()
	if err != nil {
		return
	}

	lit = makeLiteral{
		pName: name,
	}

	s.consumeWhitespace()

	// We might have  a 0-arity literal, so check if we have a period, and return if so.

	ch, _, err := s.r.ReadRune()
	if err != nil {
		return lit, err
	}
	s.r.UnreadRune()
	if ch == '.' || ch == '?' {
		return
	}

	err = s.mustConsume('(')
	if err != nil {
		return
	}
	for {
		s.consumeWhitespace()

		t, err := s.scanTerm()
		if err != nil {
			return lit, err
		}
		lit.terms = append(lit.terms, t)

		s.consumeWhitespace()

		ch, _, err := s.r.ReadRune()
		if err != nil {
			return lit, err
		}
		// ')' closes the literal
		if ch == ')' {
			break
		}
		s.r.UnreadRune()
		s.mustConsume(',')
	}
	return
}

func (s scanner) scanCommand() (cmd datalogCommand, err error) {
	s.consumeWhitespace()
	cmd.head, err = s.scanLiteral()
	if err != nil {
		return
	}

	s.consumeWhitespace()
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return cmd, err
	}
	if ch == '?' {
		cmd.isQuery = true
		return
	}

	if ch == '.' {
		return
	}

	s.r.UnreadRune()
	err = s.mustConsume(':')
	if err != nil {
		return
	}
	err = s.mustConsume('-')
	if err != nil {
		return
	}

	for {
		var l makeLiteral
		s.consumeWhitespace()
		l, err = s.scanLiteral()
		if err != nil {
			return
		}
		cmd.body = append(cmd.body, l)

		s.consumeWhitespace()

		// Check for terminus
		ch, _, err = s.r.ReadRune()
		if err != nil {
			return
		}
		if ch == '.' {
			return
		}
		if ch == ',' {
			continue
		}
		err = fmt.Errorf("Expected '.' or ',', but got %v", string(ch))
		return
	}
}

func (s scanner) scanCommands() ([]datalogCommand, error) {
	commands := make([]datalogCommand, 0)
	for {
		c, err := s.scanCommand()
		if err != nil {
			return nil, err
		}
		commands = append(commands, c)

		s.consumeWhitespace()
		ch, _, err := s.r.ReadRune()

		if ch == eof || err != nil {
			return commands, nil
		}
		s.r.UnreadRune()
	}
}

func parse(input io.Reader) ([]datalogCommand, error) {
	s := newScanner(input)
	return s.scanCommands()
}
