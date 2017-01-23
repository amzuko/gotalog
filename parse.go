package gotalog

import (
	"bufio"
	"fmt"
	"io"
)

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

var eof = rune(0)

func (s scanner) scanPredicateName() (string, error) {
	name := ""
	for {
		ch, _, err := s.r.ReadRune()
		if err != nil {
			return "", err
		}
		if !isLetter(ch) {
			err := s.r.UnreadRune()
			if err != nil {
				return "", err
			}
			break
		}
		name = name + string(ch)
	}
	if len(name) == 0 {
		return "", fmt.Errorf("expected predicate name")
	}
	return name, nil
}

func (s scanner) mustConsume(r rune) error {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return err
	}
	if ch != r {
		return fmt.Errorf("Expected %v, but got %v", r, ch)
	}
	return nil
}

func (s scanner) consumeWhitespace() {
	for {
		ch, _, err := s.r.ReadRune()
		if err != nil || !isWhitespace(ch) {
			s.r.UnreadRune()
			return
		}
	}
}

func (s scanner) scanTerm() (t term, err error) {
	s.consumeWhitespace()
	ch, _, err := s.r.ReadRune()
	if !isLetter(ch) {
		return t, fmt.Errorf("Expected a term composed of letters, but got %v", string(ch))
	}
	if isLowerCase(ch) {
		t.isConstant = true
	}
	t.value = t.value + string(ch)
	for {
		ch, _, err = s.r.ReadRune()

		if !isLetter(ch) {
			s.r.UnreadRune()
			return
		}
		t.value = t.value + string(ch)
	}
}

func (s scanner) scanLiteral() (lit makeLiteral, err error) {
	name, err := s.scanPredicateName()
	if err != nil {
		return
	}

	lit = makeLiteral{
		pName: name,
	}

	s.consumeWhitespace()

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
