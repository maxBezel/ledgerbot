package exprsplit

import (
	"fmt"
	"strings"
	"unicode"
)

/*
---------------------
SplitExprAndComment
---------------------
1. Сканирует строку слева направо, игнорируя пробелы.
2. Использует конечный автомат с двумя состояниями:
   - expectOperand ("ожидается операнд") — допустимы число, открывающая скобка,
     унарные + или -, либо начало дробного числа с точки.
   - expectOperator ("ожидается оператор") — допустимы знаки +, -, *, /, ^,
     либо закрывающая скобка. (Бинарного оператора % НЕТ.)
3. Ведёт счётчик скобок (paren) для баланса круглых скобок.
4. Корректный операнд переключает состояние в "ожидается оператор";
   корректный оператор — в "ожидается операнд".
5. Каждый раз, когда выражение корректно завершено (paren == 0 и состояние expectOperator),
   запоминается индекс конца (lastGood).
6. При встрече неподходящего символа разбор прекращается; всё после lastGood — это комментарий.
7. Если не найдено ни одного корректного выражения — возвращается ошибка (нечего эвалюировать).
8. Перед возвратом при наличии символа '%' вызывается rewritePostfixPercentChains
   (если '%' нет — переписывание не выполняется).

Ограничения и поддержка синтаксиса:
- Поддерживаются только числа, скобки и базовые арифметические операторы (+, -, *, /, ^).
  Бинарный modulo (%) не поддерживается.
- Поддерживается унарный плюс/минус.
- Нет поддержки переменных/функций (sin(x) и т.п.) — это идёт в комментарий.
- Числа: целые и с точкой; допустимы формы ".5" и "1.".
- Нет экспоненциальной записи (1e-3), нет разделителей разрядов (1_000).
- Несбалансированные скобки обрезают выражение до места ошибки.
- Если строка оканчивается оператором (например "1+"), берётся только последняя корректная часть.
*/


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

func newScanner(s string) *scanner { rr := []rune(s); return &scanner{r: rr, n: len(rr)} }
func (s *scanner) eof() bool       { return s.i >= s.n }
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
func (s *scanner) nextNonSpaceFrom(j int) (r rune, ok bool, idx int) {
	for j < s.n && unicode.IsSpace(s.r[j]) {
		j++
	}
	if j < s.n {
		return s.r[j], true, j
	}
	return 0, false, j
}
func (s *scanner) scanNumber() (start, end int, ok bool) {
	start = s.i
	seenDigit := false
	seenDot := false

	for !s.eof() {
		rc := s.r[s.i]
		switch {
		case unicode.IsDigit(rc):
			seenDigit = true
			s.i++
		case rc == '.':
			if seenDot {
				goto done
			}
			next := s.i + 1
			prev := s.i - 1
			hasPrevDigit := prev >= start && unicode.IsDigit(s.r[prev])
			hasNextDigit := next < s.n && unicode.IsDigit(s.r[next])
			if !hasPrevDigit && !hasNextDigit {
				s.i = start
				return 0, 0, false
			}
			seenDot = true
			s.i++
		default:
			goto done
		}
	}

done:
	if !seenDigit {
		s.i = start
		return 0, 0, false
	}
	return start, s.i - 1, true
}
func (s *scanner) nextStartsOperand(from int) bool {
	r, ok, idx := s.nextNonSpaceFrom(from)
	if !ok {
		return false
	}
	if unicode.IsDigit(r) || r == '(' || r == '.' {
		return true
	}

	if r == '-' {
		r2, ok2, _ := s.nextNonSpaceFrom(idx + 1)
		return ok2 && (unicode.IsDigit(r2) || r2 == '(' || r2 == '.')
	}
	return false
}

func SplitExprAndComment(orig string) (string, string, error) {
	s := orig
	if strings.ContainsRune(orig, ',') {
		s = strings.ReplaceAll(orig, ",", ".")
	}

	sc := newScanner(s)
	st := expectOperand
	paren := 0
	lastGood := -1

	markGood := func(i int) {
		if st == expectOperator && paren == 0 {
			lastGood = i
		}
	}

	sc.skipSpaces()

	for !sc.eof() {
		r := sc.cur()

		if st == expectOperand {
			if r == '+' || r == '-' {
				if sc.nextStartsOperand(sc.i + 1) {
					sc.advance()
					sc.skipSpaces()
					continue
				}
				break
			}

			if r == '(' {
				paren++
				sc.advance()
				sc.skipSpaces()
				continue
			}

			if unicode.IsDigit(r) || r == '.' {
				if _, _, ok := sc.scanNumber(); !ok {
					break
				}
				st = expectOperator
				sc.skipSpaces()
				markGood(sc.i - 1)
				if rn, ok2, _ := sc.nextNonSpaceFrom(sc.i); ok2 &&
					(unicode.IsDigit(rn) || rn == '.' || rn == '(') {
					break
				}
				continue
			}

			break
		} else {
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
				sc.advance()
				sc.skipSpaces()
				markGood(sc.i - 1)
				continue
			}

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

	if lastGood < 0 {
		return "", "", fmt.Errorf("no valid math expression found")
	}

	rawExpr := strings.TrimSpace(string([]rune(s)[:lastGood+1]))
	comment := strings.TrimSpace(string([]rune(orig)[lastGood+1:]))

	return rawExpr, comment, nil
}
