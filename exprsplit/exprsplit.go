package exprsplit

import (
	"fmt"
	"strings"
	"unicode"
)

type state int

const (
	expectOperand state = iota
	expectOperator
)

type scanner struct {
	r []rune
	n int
	i int
}

func newScanner(s string) *scanner { return &scanner{r: []rune(s), n: len([]rune(s)), i: 0} }

func (s *scanner) eof() bool { return s.i >= s.n }

func (s *scanner) cur() rune {
	if s.eof() {
		return 0
	}
	return s.r[s.i]
}

func (s *scanner) advance() { s.i++ }

func (s *scanner) skipSpaces() {
	for !s.eof() && unicode.IsSpace(s.r[s.i]) {
		s.i++
	}
}

func (s *scanner) peekNextNonSpace() (r rune, ok bool, idx int) {
	j := s.i + 1
	for j < s.n && unicode.IsSpace(s.r[j]) {
		j++
	}
	if j < s.n {
		return s.r[j], true, j
	}
	return 0, false, j
}

// сканирование числа (инты и флоты). Возвращает (начало, конец, ok).
func (s *scanner) scanNumber() (int, int, bool) {
	start := s.i
	seenDigit := false
	seenDot := false
	for !s.eof() {
		rc := s.r[s.i]
		if unicode.IsDigit(rc) {
			seenDigit = true
			s.i++
			continue
		}
		if rc == '.' && !seenDot {
			seenDot = true
			s.i++
			continue
		}
		break
	}
	if !seenDigit {
		s.i = start
		return 0, 0, false
	}
	return start, s.i - 1, true
}

// сканирование сбалансированных скобок. Возвращает (начало, конец, ok).
func (s *scanner) scanBalancedParen() (int, int, bool) {
	if s.cur() != '(' {
		return 0, 0, false
	}
	start := s.i
	depth := 1
	s.i++
	for !s.eof() && depth > 0 {
		switch s.cur() {
		case '(':
			depth++
		case ')':
			depth--
		}
		s.i++
	}
	if depth == 0 {
		return start, s.i - 1, true
	}
	// несбалансировано, тогда сбрасываем
	s.i = start
	return 0, 0, false
}

// проверяет, начинается ли с позиции операнд (число, скобки, точка, или с унарного знака)
func (s *scanner) nextStartsOperand(from int) bool {
	j := from
	for j < s.n && unicode.IsSpace(s.r[j]) {
		j++
	}
	if j >= s.n {
		return false
	}
	ch := s.r[j]
	if unicode.IsDigit(ch) || ch == '(' || ch == '.' {
		return true
	}
	if ch == '+' || ch == '-' {
		j++
		for j < s.n && unicode.IsSpace(s.r[j]) {
			j++
		}
		if j < s.n {
			ch2 := s.r[j]
			return unicode.IsDigit(ch2) || ch2 == '(' || ch2 == '.'
		}
	}
	return false
}

func SplitExprAndComment(s string) (string, string, error) {
	sc := newScanner(s)

	st := expectOperand
	paren := 0
	lastGoodRuneIdx := -1

	// фиксируем позицию, где выражение может закончиться
	markGood := func(i int) {
		if st == expectOperator && paren == 0 {
			lastGoodRuneIdx = i
		}
	}

	sc.skipSpaces()

	for !sc.eof() {
		r := sc.cur()

		if st == expectOperand {
			// унарный +/-
			if r == '+' || r == '-' {
				if sc.nextStartsOperand(sc.i+1) {
					sc.advance()
					sc.skipSpaces()
					continue
				}
				break
			}

			// скобочное подвыражение
			if r == '(' {
				_, _, ok := sc.scanBalancedParen()
				if !ok {
					break
				}
				st = expectOperator
				sc.skipSpaces()
				markGood(sc.i - 1)
				continue
			}

			// число
			if unicode.IsDigit(r) || r == '.' {
				_, _, ok := sc.scanNumber()
				if !ok {
					break
				}
				st = expectOperator
				sc.skipSpaces()
				markGood(sc.i - 1)
				continue
			}

			break
		} else { // ожидаем оператор
			if r == ')' {
				if paren == 0 {
					break
				}
				paren--
				sc.advance()
				sc.skipSpaces()
				markGood(sc.i - 1)
				continue
			}

			if r == '%' {
				// если после % идёт операнд то бинарный модуль
				if sc.nextStartsOperand(sc.i + 1) {
					sc.advance()
					st = expectOperand
					sc.skipSpaces()
					continue
				}
				// иначе постфиксный процент
				sc.advance()
				sc.skipSpaces()
				markGood(sc.i - 1)
				continue
			}

			// бинарные операторы
			if r == '+' || r == '-' || r == '*' || r == '/' || r == '^' {
				sc.advance()
				st = expectOperand
				sc.skipSpaces()
				continue
			}

			break
		}
	}

	markGood(sc.i - 1)

	if lastGoodRuneIdx < 0 {
		return "", "", fmt.Errorf("No valid math expr")
	}

	rawExpr := strings.TrimSpace(string(sc.r[:lastGoodRuneIdx+1]))
	comment := strings.TrimSpace(string(sc.r[lastGoodRuneIdx+1:]))

	// переписываем постфиксные %
	exprReady := rewritePostfixPercentWithScanner(rawExpr)

	return exprReady, comment, nil
}

//меняем X% на (X/100), оставляя бинарный % без изменений, т.к в expr постфиксный % не работает
func rewritePostfixPercentWithScanner(expr string) string {
	sc := newScanner(expr)
	st := expectOperand

	lastStart, lastEnd := -1, -1

	var out strings.Builder
	lastFlush := 0

	flush := func(idx int) {
		if idx > lastFlush {
			out.WriteString(string(sc.r[lastFlush:idx]))
			lastFlush = idx
		}
	}

	recordOperand := func(start, end int) {
		lastStart, lastEnd = start, end
	}

	sc.skipSpaces()
	for !sc.eof() {
		ch := sc.cur()

		if st == expectOperand {
			if ch == '+' || ch == '-' {
				if sc.nextStartsOperand(sc.i + 1) {
					sc.advance()
					sc.skipSpaces()
					continue
				}
			}
			if ch == '(' {
				start, end, ok := sc.scanBalancedParen()
				if !ok {
					break
				}
				recordOperand(start, end)
				st = expectOperator
				sc.skipSpaces()
				continue
			}
			if unicode.IsDigit(ch) || ch == '.' {
				start, end, ok := sc.scanNumber()
				if !ok {
					break
				}
				recordOperand(start, end)
				st = expectOperator
				sc.skipSpaces()
				continue
			}
			break
		} else { // ожидаем оператор
			if ch == '%' {
				if sc.nextStartsOperand(sc.i + 1) {
					// бинарный модуль
					sc.advance()
					st = expectOperand
					sc.skipSpaces()
					continue
				}
				// постфиксный процент
				if lastStart >= 0 && lastEnd >= lastStart {
					flush(lastStart)
					out.WriteByte('(')
					out.WriteString(string(sc.r[lastStart : lastEnd+1]))
					out.WriteString("/100)")
					lastFlush = sc.i + 1
					sc.advance()
					sc.skipSpaces()
					continue
				}
				sc.advance()
				sc.skipSpaces()
				continue
			}

			if ch == ')' {
				sc.advance()
				sc.skipSpaces()
				continue
			}

			if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' || ch == '%' {
				sc.advance()
				st = expectOperand
				sc.skipSpaces()
				continue
			}

			break
		}
	}

	flush(sc.n)
	return out.String()
}
