package exprsplit

import (
	"strings"
	"testing"
	"unicode"

	"github.com/maxBezel/ledgerbot/exprsplit"
)

func canonWS(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}


func TestSeparateExprComment_Table(t *testing.T) {
	tests := []struct {
		in      string
		expr    string
		comment string
	}{
		{"42", "42", ""},
		{"  42  ", "42", ""},
		{"42*3.14", "42*3.14", ""},
		{"(1+2)*3", "(1+2)*3", ""},
		{"( 1 + 2 ) * 3", "( 1 + 2 ) * 3", ""},

		{"   7*8  \t  blah", "7*8", "blah"},
		{"7*8\t\n\r  # hash", "7*8", "# hash"},
		{"  (1+2)  текст", "(1+2)", "текст"},

		{"-3+4", "-3+4", ""},
		{"+3+4", "+3+4", ""},
		{"-(4+5)", "-(4+5)", ""},
		{"+(4+5) коммент", "+(4+5)", "коммент"},
		{"- 3 + 4 tail", "- 3 + 4", "tail"},

		{".5 + .25", ".5 + .25", ""},
		{"3. .x", "3.", ".x"},
		{"3..14 текст", "3.", ".14 текст"},

		{"50%", "(50/100)", ""},
		{"50%%", "((50/100)/100)", ""},
		{"(100*78.5)*50%/(10-5.3)", "(100*78.5)*(50/100)/(10-5.3)", ""},
		{"-(4+5)%", "-((4+5)/100)", ""},
		{"(.5 + .25)% * 200", "((.5 + .25)/100) * 200", ""},
		{"((2+3))%", "(((2+3))/100)", ""},
		{"(a+b)%", "", "(a+b)%"},

		{"10+5/(12 * 7)%4^7 blah blah", "10+5/((12*7)/100)", "4^7 blah blah"},
		{"((2+3)*5 - -7) // comment with + - *", "((2+3)*5 - -7)", "// comment with + - *"},
		{"10+5//not a delimiter, just text with /", "10+5", "//not a delimiter, just text with /"},

		{"5+ (6-2", "5", "+ (6-2"},
		{"(((1+2)", "", "(((1+2)"},
		{")1+2", "", ")1+2"},

		{"10%5", "(10/100)", "5"},
		{"10 % 5", "(10/100)", "5"},
		{"(10)%5", "((10)/100)", "5"},
		{"10%+5", "(10/100)+5", ""},
		{"10% - 5", "(10/100)-5", ""},
		{"10%-5", "(10/100)-5", ""},
		{"(10)% - 5", "((10)/100)-5", ""},
		{"((123 + 2)%%)^-2", "((((123+2))/100)/100)^-2", ""},
		{"2^-3%", "2^-(3/100)", ""},
		{"2^(-3)%", "2^((-3)/100)", ""},


		{"(1+2)^3*50%% + (4-1)%^2 tail", "(1+2)^3*( (50/100)/100 ) + ((4-1)/100)^2", "tail"},
	}

	for _, tc := range tests {
		expr, comment, err := exprsplit.SplitExprAndComment(tc.in)

		shouldErr := tc.expr == "" && tc.comment == tc.in
		if shouldErr {
			if err == nil {
				t.Fatalf("expected error for input %q but got none (expr=%q comment=%q)", tc.in, expr, comment)
			}
			continue
		}

		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.in, err)
		}
		if canonWS(expr) != canonWS(tc.expr) || comment != tc.comment {
			t.Fatalf("input %q:\n  got  expr=%q comment=%q\n  want expr=%q comment=%q",
				tc.in, canonWS(expr), comment, canonWS(tc.expr), tc.comment)
		}
	}
}

func TestSeparateExprComment_NoExprError(t *testing.T) {
	bad := []string{
		"foo bar",
		"abc + def",
		")",
		"((a))",
	}
	for _, s := range bad {
		_, _, err := exprsplit.SplitExprAndComment(s)
		if err == nil {
			t.Fatalf("expected error for: %q", s)
		}
	}
}

func afterRewriteHasNoPostfixPercent(expr string) bool {
	if strings.Contains(expr, ")%") || strings.Contains(expr, ".%") {
		return false
	}
	r := []rune(expr)
	n := len(r)

	skip := func(i int) int {
		for i < n && unicode.IsSpace(r[i]) {
			i++
		}
		return i
	}
	nextStartsOperand := func(i int) bool {
		i = skip(i)
		if i >= n {
			return false
		}
		ch := r[i]
		if unicode.IsDigit(ch) || ch == '(' || ch == '.' {
			return true
		}
		if ch == '-' {
			i++
			i = skip(i)
			return i < n && (unicode.IsDigit(r[i]) || r[i] == '(' || r[i] == '.')
		}
		return false
	}

	for i := 0; i+1 < n; i++ {
		if r[i+1] == '%' {
			if r[i] >= '0' && r[i] <= '9' {
				if nextStartsOperand(i + 2) {
					continue
				}
				return false
			}
		}
	}
	return true
}

func TestSeparateExprComment_NoPostfixPercentRemains(t *testing.T) {
	cases := []string{
		"50%",
		"50%%",
		"(1+2)% + 3",
		"-(4+5)%",
		"(.5 + .25)% * 200",
		"((2+3))%",
		"10% + 5",
		"10%+5",
	}
	for _, in := range cases {
		expr, _, err := exprsplit.SplitExprAndComment(in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", in, err)
		}
		if !afterRewriteHasNoPostfixPercent(expr) {
			t.Fatalf("postfix %% remained after rewrite: in=%q expr=%q", in, expr)
		}
	}
}

func FuzzSeparateExprComment(f *testing.F) {
	seeds := []string{
		"", " ", "1", "1+2", "((3))", "4+", "+5", "-(6)", "7*8  text",
		"3. .x", "10+5//hi", ")))", "(((1+2)", ".5 + .25 end",
		"(100 * 78.5)*50%/(10-5.3)",
		"50%%", "10%5", "10%-5", "10% - 5",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		expr, comment, err := exprsplit.SplitExprAndComment(s)
		if err == nil {
			if expr == "" {
				t.Fatalf("expr is empty but no error for input %q (comment=%q)", s, comment)
			}
			if !afterRewriteHasNoPostfixPercent(expr) {
				t.Fatalf("postfix %% remained after rewrite: in=%q expr=%q", s, expr)
			}
		}
	})
}

func BenchmarkSeparateExprComment_Short(b *testing.B) {
	in := "(100 * 78.5)*50%/(10-5.3) blah"
	for i := 0; i < b.N; i++ {
		_, _, _ = exprsplit.SplitExprAndComment(in)
	}
}

func BenchmarkSeparateExprComment_Long(b *testing.B) {
	in := strings.Repeat("(1+2*3-4/5)^2*50%% + ", 200) + "tail"
	for i := 0; i < b.N; i++ {
		_, _, _ = exprsplit.SplitExprAndComment(in)
	}
}
