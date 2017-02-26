package gotalog

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

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

func isNumber(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}

func isLetter(ch rune) bool {
	return isLowerCase(ch) || isUpperCase(ch)
}

func isTerminal(ch rune) bool {
	return ch == '?' || ch == '.' || ch == '~'
}

func commandForTerminal(ch rune) CommandType {
	switch ch {
	case '.':
		return Assert
	case '?':
		return Query
	case '~':
		return Retract
	default:
		panic("invalid terminal rune.")
	}
}

func isAllowedBodyRune(ch rune) bool {
	return isLetter(ch) ||
		isNumber(ch) ||
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
	if !isLetter(ch) && !isNumber(ch) {
		return str, fmt.Errorf("Expected a term startign with a letter or number, but got %v", string(ch))
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

func (s scanner) scanTerm() (t Term, err error) {

	t.value, err = s.scanIdentifier()
	if err != nil {
		return t, err
	}
	leading, _ := utf8.DecodeRuneInString(t.value)

	if !isUpperCase(leading) {
		t.isConstant = true
	}
	return
}

func (s scanner) scanLiteral() (lit LiteralDefinition, err error) {
	name, err := s.scanIdentifier()
	if err != nil {
		return
	}

	lit = LiteralDefinition{
		PredicateName: name,
	}

	s.consumeWhitespace()

	// We might have  a 0-arity literal, so check if we have a period, and return if so.

	ch, _, err := s.r.ReadRune()
	if err != nil {
		return lit, err
	}
	s.r.UnreadRune()
	if isTerminal(ch) {
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
		lit.Terms = append(lit.Terms, t)

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

func (s scanner) scanCommand() (cmd DatalogCommand, err error) {
	s.consumeWhitespace()
	cmd.Head, err = s.scanLiteral()
	if err != nil {
		return
	}

	s.consumeWhitespace()
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return cmd, err
	}

	if isTerminal(ch) {
		cmd.CommandType = commandForTerminal(ch)
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
		var l LiteralDefinition
		s.consumeWhitespace()
		l, err = s.scanLiteral()
		if err != nil {
			return
		}
		cmd.Body = append(cmd.Body, l)

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

func (s scanner) scanOneCommand() (DatalogCommand, bool, error) {
	s.consumeWhitespace()
	ch, _, err := s.r.ReadRune()

	if ch == eof || err != nil {
		return DatalogCommand{}, true, nil
	}
	s.r.UnreadRune()

	c, err := s.scanCommand()
	return c, false, err
}

func buildLiteral(ml LiteralDefinition, db Database) literal {
	return literal{
		pred:  db.newPredicate(ml.PredicateName, len(ml.Terms)),
		terms: ml.Terms,
	}
}

// Should we instead write commands back to disk,
// and focus on providing utility methods to convert clauses back to commands?
// TODO:consider this.
func writeLiteral(w io.Writer, l *literal) error {
	_, err := io.WriteString(w, l.pred.Name)
	if err != nil {
		return err
	}
	if l.pred.Arity > 0 {
		_, err := io.WriteString(w, "(")
		if err != nil {
			return err
		}
		strs := make([]string, len(l.terms))
		for i, t := range l.terms {
			strs[i] = t.value
		}
		_, err = io.WriteString(w, strings.Join(strs, ", "))
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, ")")
		if err != nil {
			return err
		}
	}
	return nil
}

func writeClause(w io.Writer, c *clause, t CommandType) error {
	err := writeLiteral(w, &c.head)
	if err != nil {
		return err
	}
	if len(c.body) > 0 {
		_, err := io.WriteString(w, " :- ")
		for i, l := range c.body {
			if i > 0 {
				_, err := io.WriteString(w, ", ")
				if err != nil {
					return err
				}
			}
			err = writeLiteral(w, &l)
			if err != nil {
				return err
			}
		}
	}
	switch t {
	case Assert:
		_, err = io.WriteString(w, ".\n")
	case Query:
		_, err = io.WriteString(w, "?\n")
	case Retract:
		_, err = io.WriteString(w, "~\n")
	}
	return err
}
