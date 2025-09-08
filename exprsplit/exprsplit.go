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
   - expectOperand ("ожидается операнд") - на этом шаге допустимы число,
     открывающая скобка, унарные + или -, либо начало дробного числа с точки.
   - expectOperator ("ожидается оператор") - на этом шаге допустимы знаки
     +, -, *, /, ^, %, либо закрывающая скобка.
3. Ведёт счётчик скобок (paren), чтобы отслеживать баланс скобок (только круглых, на остальные похуй).
4. Когда встречается корректный операнд - переключается в состояние "ожидается оператор".
   Когда встречается корректный оператор - переключается в состояние "ожидается операнд".
5. В каждый момент, когда выражение завершено корректно (paren == 0, состояние expectOperator),
   запоминается индекс конца (lastGood).
6. При встрече символа, который не подходит ни под одно состояние, разбор прекращается.
   Всё, что идёт после lastGood, считается комментарием.
7. Если в строке не найдено ни одного корректного выражения - возвращается ошибка, т.к нечего эвалюировать.
8. Перед возвратом вызывается вспомогательная функция rewritePostfixPercentChains,


- Поддерживаются только числа, скобки и базовые арифметические операторы (+, -, *, /, ^, %).
- Учитывается унарный плюс/минус.
- Нет поддержки переменных, функций (например sin(x)), идентификаторов.
  Всё это попадает в "комментарий".
- Числа могут быть целыми или с точкой, форматы ".5" и "1." допустимы.
- Нет поддержки экспоненциальной формы (1e-3), нет разделителей разрядов (1_000).
- Несбалансированные скобки приводят к обрезке выражения до ошибки.
- Если строка оканчивается оператором (например "1+"), то выражение считается неполным,
  и в результат попадает только последняя корректная часть.

---------------------
rewritePostfixPercentChains
---------------------
отвечает за переписывание выражений с постфиксными процентами. Пример: "200%%" → "((200/100)/100)".
expr какает в штаны с этого :(

1. Строка разбирается посимвольно в двух состояниях:
   - wantOperand ("ожидается операнд") - число или скобки;
   - wantOperator ("ожидается оператор") - знак операции после операнда.
2. Когда встречается число или выражение в скобках, оно запоминается как последний операнд.
3. Когда после операнда идёт знак '%':
   - Если за ним следует новый операнд (например "50%3"), то '%' трактуется как обычный оператор "остаток от деления".
   - Если операнда нет → начинается цепочка постфиксных процентов. Считается количество подряд идущих '%' (с пробелами).
     Последний операнд переписывается как последовательность делений на 100:
       "200%"  → "(200/100)"
       "200%%" → "((200/100)/100)"
       и т.д.
4. Всё остальное копируется в результат без изменений.

- Если '%' стоит в неожиданном месте, строка может быть переписана только частично.
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

func SplitExprAndComment(s string) (string, string, error) {
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

			// number
			if unicode.IsDigit(r) || r == '.' {
				if _, _, ok := sc.scanNumber(); !ok {
					break
				}
				st = expectOperator
				sc.skipSpaces()
				markGood(sc.i - 1)
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
				rn, ok, _ := sc.nextNonSpaceFrom(sc.i + 1)
				if ok && (unicode.IsDigit(rn) || rn == '(' || rn == '.' || rn == '-') && sc.nextStartsOperand(sc.i+1) {
					sc.advance()
					st = expectOperand
					sc.skipSpaces()
					continue
				}

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

	rawExpr := strings.TrimSpace(string(sc.r[:lastGood+1]))
	comment := strings.TrimSpace(string(sc.r[lastGood+1:]))

	rewritten := rewritePostfixPercentChains(rawExpr)

	return rewritten, comment, nil
}

func rewritePostfixPercentChains(expr string) string {
	r := []rune(expr)
	n := len(r)

	var out strings.Builder
	lastFlush := 0

	lastStart, lastEnd := -1, -1

	isSpace := func(rr rune) bool { return unicode.IsSpace(rr) }
	skipSpacesFrom := func(i int) int {
		for i < n && isSpace(r[i]) {
			i++
		}
		return i
	}

	nextStartsOperand := func(from int) bool {
		j := skipSpacesFrom(from)
		if j >= n {
			return false
		}
		ch := r[j]
		if unicode.IsDigit(ch) || ch == '(' || ch == '.' {
			return true
		}
		if ch == '-' {
			j++
			j = skipSpacesFrom(j)
			if j < n {
				ch2 := r[j]
				return unicode.IsDigit(ch2) || ch2 == '(' || ch2 == '.'
			}
		}
		return false
	}

	type st int
	const (
		wantOperand st = iota
		wantOperator
	)
	state := wantOperand

	i := 0
	for i < n {
		ch := r[i]

		switch state {
		case wantOperand:
			if ch == '+' || ch == '-' {
				j := skipSpacesFrom(i + 1)
				if j < n && (unicode.IsDigit(r[j]) || r[j] == '(' || r[j] == '.') {
					i++
					i = skipSpacesFrom(i)
					continue
				}
			}
			if ch == '(' {
				start := i
				depth := 1
				i++
				for i < n && depth > 0 {
					if r[i] == '(' {
						depth++
					} else if r[i] == ')' {
						depth--
					}
					i++
				}
				if depth != 0 {
					out.WriteString(string(r[lastFlush:]))
					return out.String()
				}
				lastStart, lastEnd = start, i-1
				state = wantOperator
				i = skipSpacesFrom(i)
				continue
			}

			if unicode.IsDigit(ch) || ch == '.' {
				start := i
				seenDigit := false
				seenDot := false
				for i < n {
					switch r[i] {
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						seenDigit = true
						i++
					case '.':
						if seenDot {
							goto numDone
						}
						seenDot = true
						i++
					default:
						goto numDone
					}
				}
			numDone:
				if !seenDigit {
					out.WriteString(string(r[lastFlush:]))
					return out.String()
				}
				lastStart, lastEnd = start, i-1
				state = wantOperator
				i = skipSpacesFrom(i)
				continue
			}

			out.WriteString(string(r[lastFlush:]))
			return out.String()

		case wantOperator:
			if ch == '%' {
				if nextStartsOperand(i + 1) {
					i++
					state = wantOperand
					i = skipSpacesFrom(i)
					continue
				}

				j := i
				count := 1
				for {
					k := skipSpacesFrom(j + 1)
					if k < n && r[k] == '%' {
						count++
						j = k
						continue
					}
					break
				}

				out.WriteString(string(r[lastFlush:lastStart]))

				op := string(r[lastStart : lastEnd+1])
				for c := 0; c < count; c++ {
					op = "(" + op + "/100)"
				}
				out.WriteString(op)

				i = j + 1
				i = skipSpacesFrom(i)
				lastFlush = i

				continue
			}

			if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' || ch == '%' {
				i++
				state = wantOperand
				i = skipSpacesFrom(i)
				continue
			}

			i++
			continue
		}
	}

	out.WriteString(string(r[lastFlush:]))
	return out.String()
}
