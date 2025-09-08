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

---------------------
rewritePostfixPercentChains
---------------------
Переписывает постфиксные проценты в деление на 100. Пример: "200%%" -> "((200/100)/100)".
(Только постфикс '%'; бинарного модуля нет. Унарный минус НЕ нормализуется:
"(-3)%" -> "((-3)/100)".)

1. Предобработка (prepass):
   Если внутри скобок встречается цепочка '%' непосредственно перед ')'
   (игнорируя пробелы), цепочка «поднимается» наружу — переносится сразу
   за соответствующую закрывающую скобку.
   Пример: "((123+2)%%)^-2" -> "((123+2))%%^-2".
   Это позволяет основному проходу увидеть '%' как оператор после всего
   скобочного выражения.

2. Основной проход:
   Разбор посимвольно в двух состояниях:
     - wantOperand ("ожидается операнд") — число или скобки;
     - wantOperator ("ожидается оператор") — знак операции после операнда.
   При встрече числа или скобочного выражения оно запоминается как «последний операнд».
   Если сразу после операнда идёт '%', считается длина цепочки ('%%…' с пробелами),
   и последний операнд переписывается как последовательность делений на 100:
     "200%"  -> "(200/100)"
     "200%%" -> "((200/100)/100)"
     и т.д.
   Всё остальное копируется без изменений.
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

	rewritten := rawExpr
	if strings.ContainsRune(rawExpr, '%') {
			rewritten = rewritePostfixPercentChains(rawExpr)
	}


	return rewritten, comment, nil
}

func rewritePostfixPercentChains(expr string) string {
	r := []rune(expr)
	n := len(r)

	isSpace := func(rr rune) bool { return unicode.IsSpace(rr) }
	skipSpacesFrom := func(i int) int {
		for i < n && isSpace(r[i]) {
			i++
		}
		return i
	}

	var outPre []rune
	depth := 0
	pending := []int{0}

	ensureDepth := func(d int) {
		for len(pending) <= d {
			pending = append(pending, 0)
		}
	}

	i := 0
	for i < n {
		ch := r[i]

		switch ch {
		case '(':
			outPre = append(outPre, ch)
			depth++
			ensureDepth(depth)
			i++

		case ')':
			outPre = append(outPre, ch)
			if depth >= 0 && depth < len(pending) && pending[depth] > 0 {
				for c := 0; c < pending[depth]; c++ {
					outPre = append(outPre, '%')
				}
				pending[depth] = 0
			}
			if depth > 0 {
				depth--
			}
			i++

		case '%':
			if depth > 0 {
				j := i
				count := 0
				for {
					j = skipSpacesFrom(j)
					if j < n && r[j] == '%' {
						count++
						j++
						continue
					}
					break
				}
				k := skipSpacesFrom(j)
				if count > 0 && k < n && r[k] == ')' {
					ensureDepth(depth)
					pending[depth] += count

					i = j
					continue
				}
			}

			outPre = append(outPre, ch)
			i++

		default:
			outPre = append(outPre, ch)
			i++
		}
	}

	expr = string(outPre)
	r = []rune(expr)
	n = len(r)

	var out strings.Builder
	lastFlush := 0

	lastStart, lastEnd := -1, -1

	isSpace2 := func(rr rune) bool { return unicode.IsSpace(rr) }
	skipSpacesFrom2 := func(i int) int {
		for i < n && isSpace2(r[i]) {
			i++
		}
		return i
	}

	type st int
	const (
		wantOperand st = iota
		wantOperator
	)
	state := wantOperand

	i = 0
	for i < n {
		ch := r[i]

		switch state {
		case wantOperand:
			if ch == '+' || ch == '-' {
				j := skipSpacesFrom2(i + 1)
				if j < n && (unicode.IsDigit(r[j]) || r[j] == '(' || r[j] == '.') {
					i++
					i = skipSpacesFrom2(i)
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
				i = skipSpacesFrom2(i)
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
				i = skipSpacesFrom2(i)
				continue
			}

			out.WriteString(string(r[lastFlush:]))
			return out.String()

		case wantOperator:
			if ch == '%' {
				j := i
				count := 1
				for {
						k := skipSpacesFrom2(j + 1)
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
				i = skipSpacesFrom2(i)
				lastFlush = i
				continue
			}

			if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' {
				i++
				state = wantOperand
				i = skipSpacesFrom2(i)
				continue
			}

			i++
			continue
		}
	}

	out.WriteString(string(r[lastFlush:]))
	return out.String()
}
