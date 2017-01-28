package gotalog

import (
	"bufio"
	"fmt"
	"io"
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

func commandForTerminal(ch rune) commandType {
	switch ch {
	case '.':
		return assert
	case '?':
		return query
	case '~':
		return retract
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

func (s scanner) scanTerm() (t term, err error) {

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

func (s scanner) scanCommand() (cmd DatalogCommand, err error) {
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

	if isTerminal(ch) {
		cmd.commandType = commandForTerminal(ch)
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

func (s scanner) scanCommands() ([]DatalogCommand, error) {
	commands := make([]DatalogCommand, 0)
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

// Parse consumes a reader, producing a slice of datalogCommands.
// Interestingly, we should not actually need to read all of the commands
// into memory before we start executing them.
// TODO: consider modifying this to return a channel.
func Parse(input io.Reader) ([]DatalogCommand, error) {
	s := newScanner(input)
	return s.scanCommands()
}
