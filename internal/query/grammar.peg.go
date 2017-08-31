package query

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleQuery
	ruleColumnExpr
	ruleGroupExpr
	ruleFilterExpr
	ruleOrderByExpr
	ruleLimitExpr
	rulePointSizeExpr
	ruleColumns
	ruleColumn
	ruleColumnAggregation
	ruleLogicExpr
	ruleOPERATOR
	ruleFilterKey
	ruleFilterCondition
	ruleFilterValue
	ruleValue
	ruleDescending
	ruleString
	ruleStringChar
	ruleEscape
	ruleSimpleEscape
	ruleOctalEscape
	ruleHexEscape
	ruleUniversalCharacter
	ruleHexQuad
	ruleHexDigit
	ruleUnsigned
	ruleSign
	ruleInteger
	ruleFloat
	ruleDuration
	ruleIdentifier
	ruleIdChar
	ruleKeyword
	rule_
	ruleLPAR
	ruleRPAR
	ruleCOMMA
	ruleAction0
	ruleAction1
	ruleAction2
	rulePegText
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
)

var rul3s = [...]string{
	"Unknown",
	"Query",
	"ColumnExpr",
	"GroupExpr",
	"FilterExpr",
	"OrderByExpr",
	"LimitExpr",
	"PointSizeExpr",
	"Columns",
	"Column",
	"ColumnAggregation",
	"LogicExpr",
	"OPERATOR",
	"FilterKey",
	"FilterCondition",
	"FilterValue",
	"Value",
	"Descending",
	"String",
	"StringChar",
	"Escape",
	"SimpleEscape",
	"OctalEscape",
	"HexEscape",
	"UniversalCharacter",
	"HexQuad",
	"HexDigit",
	"Unsigned",
	"Sign",
	"Integer",
	"Float",
	"Duration",
	"Identifier",
	"IdChar",
	"Keyword",
	"_",
	"LPAR",
	"RPAR",
	"COMMA",
	"Action0",
	"Action1",
	"Action2",
	"PegText",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type parser struct {
	expression

	Buffer string
	buffer []rune
	rules  [54]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *parser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *parser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *parser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *parser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *parser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.currentSection = "columns"
		case ruleAction1:
			p.currentSection = "group by"
		case ruleAction2:
			p.currentSection = "order by"
		case ruleAction3:
			p.SetLimit(text)
		case ruleAction4:
			p.SetPointSize(text)
		case ruleAction5:
			p.AddColumn()
		case ruleAction6:
			p.SetColumnName(text)
		case ruleAction7:
			p.SetColumnAggregate(text)
		case ruleAction8:
			p.SetColumnName(text)
		case ruleAction9:
			p.AddFilter()
		case ruleAction10:
			p.SetFilterColumn(text)
		case ruleAction11:
			p.SetFilterCondition(text)
		case ruleAction12:
			p.SetFilterValue(text)
		case ruleAction13:
			p.SetDescending()

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *parser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Query <- <(_ ColumnExpr? _ GroupExpr? _ FilterExpr? _ OrderByExpr? _ LimitExpr? _ PointSizeExpr? _ !.)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[rule_]() {
					goto l0
				}
				{
					position2, tokenIndex2 := position, tokenIndex
					if !_rules[ruleColumnExpr]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex = position2, tokenIndex2
				}
			l3:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position4, tokenIndex4 := position, tokenIndex
					if !_rules[ruleGroupExpr]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex = position4, tokenIndex4
				}
			l5:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position6, tokenIndex6 := position, tokenIndex
					if !_rules[ruleFilterExpr]() {
						goto l6
					}
					goto l7
				l6:
					position, tokenIndex = position6, tokenIndex6
				}
			l7:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position8, tokenIndex8 := position, tokenIndex
					if !_rules[ruleOrderByExpr]() {
						goto l8
					}
					goto l9
				l8:
					position, tokenIndex = position8, tokenIndex8
				}
			l9:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position10, tokenIndex10 := position, tokenIndex
					if !_rules[ruleLimitExpr]() {
						goto l10
					}
					goto l11
				l10:
					position, tokenIndex = position10, tokenIndex10
				}
			l11:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position12, tokenIndex12 := position, tokenIndex
					if !_rules[rulePointSizeExpr]() {
						goto l12
					}
					goto l13
				l12:
					position, tokenIndex = position12, tokenIndex12
				}
			l13:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position14, tokenIndex14 := position, tokenIndex
					if !matchDot() {
						goto l14
					}
					goto l0
				l14:
					position, tokenIndex = position14, tokenIndex14
				}
				add(ruleQuery, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 ColumnExpr <- <(('s' / 'S') ('e' / 'E') ('l' / 'L') ('e' / 'E') ('c' / 'C') ('t' / 'T') _ Action0 Columns)> */
		func() bool {
			position15, tokenIndex15 := position, tokenIndex
			{
				position16 := position
				{
					position17, tokenIndex17 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l18
					}
					position++
					goto l17
				l18:
					position, tokenIndex = position17, tokenIndex17
					if buffer[position] != rune('S') {
						goto l15
					}
					position++
				}
			l17:
				{
					position19, tokenIndex19 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l20
					}
					position++
					goto l19
				l20:
					position, tokenIndex = position19, tokenIndex19
					if buffer[position] != rune('E') {
						goto l15
					}
					position++
				}
			l19:
				{
					position21, tokenIndex21 := position, tokenIndex
					if buffer[position] != rune('l') {
						goto l22
					}
					position++
					goto l21
				l22:
					position, tokenIndex = position21, tokenIndex21
					if buffer[position] != rune('L') {
						goto l15
					}
					position++
				}
			l21:
				{
					position23, tokenIndex23 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l24
					}
					position++
					goto l23
				l24:
					position, tokenIndex = position23, tokenIndex23
					if buffer[position] != rune('E') {
						goto l15
					}
					position++
				}
			l23:
				{
					position25, tokenIndex25 := position, tokenIndex
					if buffer[position] != rune('c') {
						goto l26
					}
					position++
					goto l25
				l26:
					position, tokenIndex = position25, tokenIndex25
					if buffer[position] != rune('C') {
						goto l15
					}
					position++
				}
			l25:
				{
					position27, tokenIndex27 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l28
					}
					position++
					goto l27
				l28:
					position, tokenIndex = position27, tokenIndex27
					if buffer[position] != rune('T') {
						goto l15
					}
					position++
				}
			l27:
				if !_rules[rule_]() {
					goto l15
				}
				if !_rules[ruleAction0]() {
					goto l15
				}
				if !_rules[ruleColumns]() {
					goto l15
				}
				add(ruleColumnExpr, position16)
			}
			return true
		l15:
			position, tokenIndex = position15, tokenIndex15
			return false
		},
		/* 2 GroupExpr <- <(('g' / 'G') ('r' / 'R') ('o' / 'O') ('u' / 'U') ('p' / 'P') ' ' ('b' / 'B') ('y' / 'Y') _ Action1 Columns)> */
		func() bool {
			position29, tokenIndex29 := position, tokenIndex
			{
				position30 := position
				{
					position31, tokenIndex31 := position, tokenIndex
					if buffer[position] != rune('g') {
						goto l32
					}
					position++
					goto l31
				l32:
					position, tokenIndex = position31, tokenIndex31
					if buffer[position] != rune('G') {
						goto l29
					}
					position++
				}
			l31:
				{
					position33, tokenIndex33 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l34
					}
					position++
					goto l33
				l34:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('R') {
						goto l29
					}
					position++
				}
			l33:
				{
					position35, tokenIndex35 := position, tokenIndex
					if buffer[position] != rune('o') {
						goto l36
					}
					position++
					goto l35
				l36:
					position, tokenIndex = position35, tokenIndex35
					if buffer[position] != rune('O') {
						goto l29
					}
					position++
				}
			l35:
				{
					position37, tokenIndex37 := position, tokenIndex
					if buffer[position] != rune('u') {
						goto l38
					}
					position++
					goto l37
				l38:
					position, tokenIndex = position37, tokenIndex37
					if buffer[position] != rune('U') {
						goto l29
					}
					position++
				}
			l37:
				{
					position39, tokenIndex39 := position, tokenIndex
					if buffer[position] != rune('p') {
						goto l40
					}
					position++
					goto l39
				l40:
					position, tokenIndex = position39, tokenIndex39
					if buffer[position] != rune('P') {
						goto l29
					}
					position++
				}
			l39:
				if buffer[position] != rune(' ') {
					goto l29
				}
				position++
				{
					position41, tokenIndex41 := position, tokenIndex
					if buffer[position] != rune('b') {
						goto l42
					}
					position++
					goto l41
				l42:
					position, tokenIndex = position41, tokenIndex41
					if buffer[position] != rune('B') {
						goto l29
					}
					position++
				}
			l41:
				{
					position43, tokenIndex43 := position, tokenIndex
					if buffer[position] != rune('y') {
						goto l44
					}
					position++
					goto l43
				l44:
					position, tokenIndex = position43, tokenIndex43
					if buffer[position] != rune('Y') {
						goto l29
					}
					position++
				}
			l43:
				if !_rules[rule_]() {
					goto l29
				}
				if !_rules[ruleAction1]() {
					goto l29
				}
				if !_rules[ruleColumns]() {
					goto l29
				}
				add(ruleGroupExpr, position30)
			}
			return true
		l29:
			position, tokenIndex = position29, tokenIndex29
			return false
		},
		/* 3 FilterExpr <- <(('f' / 'F') ('i' / 'I') ('l' / 'L') ('t' / 'T') ('e' / 'E') ('r' / 'R') _ LogicExpr (_ COMMA? LogicExpr)*)> */
		func() bool {
			position45, tokenIndex45 := position, tokenIndex
			{
				position46 := position
				{
					position47, tokenIndex47 := position, tokenIndex
					if buffer[position] != rune('f') {
						goto l48
					}
					position++
					goto l47
				l48:
					position, tokenIndex = position47, tokenIndex47
					if buffer[position] != rune('F') {
						goto l45
					}
					position++
				}
			l47:
				{
					position49, tokenIndex49 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l50
					}
					position++
					goto l49
				l50:
					position, tokenIndex = position49, tokenIndex49
					if buffer[position] != rune('I') {
						goto l45
					}
					position++
				}
			l49:
				{
					position51, tokenIndex51 := position, tokenIndex
					if buffer[position] != rune('l') {
						goto l52
					}
					position++
					goto l51
				l52:
					position, tokenIndex = position51, tokenIndex51
					if buffer[position] != rune('L') {
						goto l45
					}
					position++
				}
			l51:
				{
					position53, tokenIndex53 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l54
					}
					position++
					goto l53
				l54:
					position, tokenIndex = position53, tokenIndex53
					if buffer[position] != rune('T') {
						goto l45
					}
					position++
				}
			l53:
				{
					position55, tokenIndex55 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l56
					}
					position++
					goto l55
				l56:
					position, tokenIndex = position55, tokenIndex55
					if buffer[position] != rune('E') {
						goto l45
					}
					position++
				}
			l55:
				{
					position57, tokenIndex57 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l58
					}
					position++
					goto l57
				l58:
					position, tokenIndex = position57, tokenIndex57
					if buffer[position] != rune('R') {
						goto l45
					}
					position++
				}
			l57:
				if !_rules[rule_]() {
					goto l45
				}
				if !_rules[ruleLogicExpr]() {
					goto l45
				}
			l59:
				{
					position60, tokenIndex60 := position, tokenIndex
					if !_rules[rule_]() {
						goto l60
					}
					{
						position61, tokenIndex61 := position, tokenIndex
						if !_rules[ruleCOMMA]() {
							goto l61
						}
						goto l62
					l61:
						position, tokenIndex = position61, tokenIndex61
					}
				l62:
					if !_rules[ruleLogicExpr]() {
						goto l60
					}
					goto l59
				l60:
					position, tokenIndex = position60, tokenIndex60
				}
				add(ruleFilterExpr, position46)
			}
			return true
		l45:
			position, tokenIndex = position45, tokenIndex45
			return false
		},
		/* 4 OrderByExpr <- <(('o' / 'O') ('r' / 'R') ('d' / 'D') ('e' / 'E') ('r' / 'R') ' ' ('b' / 'B') ('y' / 'Y') _ Action2 Columns Descending?)> */
		func() bool {
			position63, tokenIndex63 := position, tokenIndex
			{
				position64 := position
				{
					position65, tokenIndex65 := position, tokenIndex
					if buffer[position] != rune('o') {
						goto l66
					}
					position++
					goto l65
				l66:
					position, tokenIndex = position65, tokenIndex65
					if buffer[position] != rune('O') {
						goto l63
					}
					position++
				}
			l65:
				{
					position67, tokenIndex67 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l68
					}
					position++
					goto l67
				l68:
					position, tokenIndex = position67, tokenIndex67
					if buffer[position] != rune('R') {
						goto l63
					}
					position++
				}
			l67:
				{
					position69, tokenIndex69 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l70
					}
					position++
					goto l69
				l70:
					position, tokenIndex = position69, tokenIndex69
					if buffer[position] != rune('D') {
						goto l63
					}
					position++
				}
			l69:
				{
					position71, tokenIndex71 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l72
					}
					position++
					goto l71
				l72:
					position, tokenIndex = position71, tokenIndex71
					if buffer[position] != rune('E') {
						goto l63
					}
					position++
				}
			l71:
				{
					position73, tokenIndex73 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l74
					}
					position++
					goto l73
				l74:
					position, tokenIndex = position73, tokenIndex73
					if buffer[position] != rune('R') {
						goto l63
					}
					position++
				}
			l73:
				if buffer[position] != rune(' ') {
					goto l63
				}
				position++
				{
					position75, tokenIndex75 := position, tokenIndex
					if buffer[position] != rune('b') {
						goto l76
					}
					position++
					goto l75
				l76:
					position, tokenIndex = position75, tokenIndex75
					if buffer[position] != rune('B') {
						goto l63
					}
					position++
				}
			l75:
				{
					position77, tokenIndex77 := position, tokenIndex
					if buffer[position] != rune('y') {
						goto l78
					}
					position++
					goto l77
				l78:
					position, tokenIndex = position77, tokenIndex77
					if buffer[position] != rune('Y') {
						goto l63
					}
					position++
				}
			l77:
				if !_rules[rule_]() {
					goto l63
				}
				if !_rules[ruleAction2]() {
					goto l63
				}
				if !_rules[ruleColumns]() {
					goto l63
				}
				{
					position79, tokenIndex79 := position, tokenIndex
					if !_rules[ruleDescending]() {
						goto l79
					}
					goto l80
				l79:
					position, tokenIndex = position79, tokenIndex79
				}
			l80:
				add(ruleOrderByExpr, position64)
			}
			return true
		l63:
			position, tokenIndex = position63, tokenIndex63
			return false
		},
		/* 5 LimitExpr <- <(('l' / 'L') ('i' / 'I') ('m' / 'M') ('i' / 'I') ('t' / 'T') _ <Unsigned> Action3)> */
		func() bool {
			position81, tokenIndex81 := position, tokenIndex
			{
				position82 := position
				{
					position83, tokenIndex83 := position, tokenIndex
					if buffer[position] != rune('l') {
						goto l84
					}
					position++
					goto l83
				l84:
					position, tokenIndex = position83, tokenIndex83
					if buffer[position] != rune('L') {
						goto l81
					}
					position++
				}
			l83:
				{
					position85, tokenIndex85 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l86
					}
					position++
					goto l85
				l86:
					position, tokenIndex = position85, tokenIndex85
					if buffer[position] != rune('I') {
						goto l81
					}
					position++
				}
			l85:
				{
					position87, tokenIndex87 := position, tokenIndex
					if buffer[position] != rune('m') {
						goto l88
					}
					position++
					goto l87
				l88:
					position, tokenIndex = position87, tokenIndex87
					if buffer[position] != rune('M') {
						goto l81
					}
					position++
				}
			l87:
				{
					position89, tokenIndex89 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l90
					}
					position++
					goto l89
				l90:
					position, tokenIndex = position89, tokenIndex89
					if buffer[position] != rune('I') {
						goto l81
					}
					position++
				}
			l89:
				{
					position91, tokenIndex91 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l92
					}
					position++
					goto l91
				l92:
					position, tokenIndex = position91, tokenIndex91
					if buffer[position] != rune('T') {
						goto l81
					}
					position++
				}
			l91:
				if !_rules[rule_]() {
					goto l81
				}
				{
					position93 := position
					if !_rules[ruleUnsigned]() {
						goto l81
					}
					add(rulePegText, position93)
				}
				if !_rules[ruleAction3]() {
					goto l81
				}
				add(ruleLimitExpr, position82)
			}
			return true
		l81:
			position, tokenIndex = position81, tokenIndex81
			return false
		},
		/* 6 PointSizeExpr <- <(('p' / 'P') ('o' / 'O') ('i' / 'I') ('n' / 'N') ('t' / 'T') ' ' ('s' / 'S') ('i' / 'I') ('z' / 'Z') ('e' / 'E') _ <Duration> Action4)> */
		func() bool {
			position94, tokenIndex94 := position, tokenIndex
			{
				position95 := position
				{
					position96, tokenIndex96 := position, tokenIndex
					if buffer[position] != rune('p') {
						goto l97
					}
					position++
					goto l96
				l97:
					position, tokenIndex = position96, tokenIndex96
					if buffer[position] != rune('P') {
						goto l94
					}
					position++
				}
			l96:
				{
					position98, tokenIndex98 := position, tokenIndex
					if buffer[position] != rune('o') {
						goto l99
					}
					position++
					goto l98
				l99:
					position, tokenIndex = position98, tokenIndex98
					if buffer[position] != rune('O') {
						goto l94
					}
					position++
				}
			l98:
				{
					position100, tokenIndex100 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l101
					}
					position++
					goto l100
				l101:
					position, tokenIndex = position100, tokenIndex100
					if buffer[position] != rune('I') {
						goto l94
					}
					position++
				}
			l100:
				{
					position102, tokenIndex102 := position, tokenIndex
					if buffer[position] != rune('n') {
						goto l103
					}
					position++
					goto l102
				l103:
					position, tokenIndex = position102, tokenIndex102
					if buffer[position] != rune('N') {
						goto l94
					}
					position++
				}
			l102:
				{
					position104, tokenIndex104 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l105
					}
					position++
					goto l104
				l105:
					position, tokenIndex = position104, tokenIndex104
					if buffer[position] != rune('T') {
						goto l94
					}
					position++
				}
			l104:
				if buffer[position] != rune(' ') {
					goto l94
				}
				position++
				{
					position106, tokenIndex106 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l107
					}
					position++
					goto l106
				l107:
					position, tokenIndex = position106, tokenIndex106
					if buffer[position] != rune('S') {
						goto l94
					}
					position++
				}
			l106:
				{
					position108, tokenIndex108 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l109
					}
					position++
					goto l108
				l109:
					position, tokenIndex = position108, tokenIndex108
					if buffer[position] != rune('I') {
						goto l94
					}
					position++
				}
			l108:
				{
					position110, tokenIndex110 := position, tokenIndex
					if buffer[position] != rune('z') {
						goto l111
					}
					position++
					goto l110
				l111:
					position, tokenIndex = position110, tokenIndex110
					if buffer[position] != rune('Z') {
						goto l94
					}
					position++
				}
			l110:
				{
					position112, tokenIndex112 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l113
					}
					position++
					goto l112
				l113:
					position, tokenIndex = position112, tokenIndex112
					if buffer[position] != rune('E') {
						goto l94
					}
					position++
				}
			l112:
				if !_rules[rule_]() {
					goto l94
				}
				{
					position114 := position
					if !_rules[ruleDuration]() {
						goto l94
					}
					add(rulePegText, position114)
				}
				if !_rules[ruleAction4]() {
					goto l94
				}
				add(rulePointSizeExpr, position95)
			}
			return true
		l94:
			position, tokenIndex = position94, tokenIndex94
			return false
		},
		/* 7 Columns <- <(Column (COMMA Column)*)> */
		func() bool {
			position115, tokenIndex115 := position, tokenIndex
			{
				position116 := position
				if !_rules[ruleColumn]() {
					goto l115
				}
			l117:
				{
					position118, tokenIndex118 := position, tokenIndex
					if !_rules[ruleCOMMA]() {
						goto l118
					}
					if !_rules[ruleColumn]() {
						goto l118
					}
					goto l117
				l118:
					position, tokenIndex = position118, tokenIndex118
				}
				add(ruleColumns, position116)
			}
			return true
		l115:
			position, tokenIndex = position115, tokenIndex115
			return false
		},
		/* 8 Column <- <(Action5 (ColumnAggregation / (<Identifier> _ Action6)))> */
		func() bool {
			position119, tokenIndex119 := position, tokenIndex
			{
				position120 := position
				if !_rules[ruleAction5]() {
					goto l119
				}
				{
					position121, tokenIndex121 := position, tokenIndex
					if !_rules[ruleColumnAggregation]() {
						goto l122
					}
					goto l121
				l122:
					position, tokenIndex = position121, tokenIndex121
					{
						position123 := position
						if !_rules[ruleIdentifier]() {
							goto l119
						}
						add(rulePegText, position123)
					}
					if !_rules[rule_]() {
						goto l119
					}
					if !_rules[ruleAction6]() {
						goto l119
					}
				}
			l121:
				add(ruleColumn, position120)
			}
			return true
		l119:
			position, tokenIndex = position119, tokenIndex119
			return false
		},
		/* 9 ColumnAggregation <- <(<Identifier> Action7 LPAR <Identifier> RPAR Action8)> */
		func() bool {
			position124, tokenIndex124 := position, tokenIndex
			{
				position125 := position
				{
					position126 := position
					if !_rules[ruleIdentifier]() {
						goto l124
					}
					add(rulePegText, position126)
				}
				if !_rules[ruleAction7]() {
					goto l124
				}
				if !_rules[ruleLPAR]() {
					goto l124
				}
				{
					position127 := position
					if !_rules[ruleIdentifier]() {
						goto l124
					}
					add(rulePegText, position127)
				}
				if !_rules[ruleRPAR]() {
					goto l124
				}
				if !_rules[ruleAction8]() {
					goto l124
				}
				add(ruleColumnAggregation, position125)
			}
			return true
		l124:
			position, tokenIndex = position124, tokenIndex124
			return false
		},
		/* 10 LogicExpr <- <((LPAR LogicExpr RPAR) / (Action9 FilterKey _ FilterCondition _ FilterValue))> */
		func() bool {
			position128, tokenIndex128 := position, tokenIndex
			{
				position129 := position
				{
					position130, tokenIndex130 := position, tokenIndex
					if !_rules[ruleLPAR]() {
						goto l131
					}
					if !_rules[ruleLogicExpr]() {
						goto l131
					}
					if !_rules[ruleRPAR]() {
						goto l131
					}
					goto l130
				l131:
					position, tokenIndex = position130, tokenIndex130
					if !_rules[ruleAction9]() {
						goto l128
					}
					if !_rules[ruleFilterKey]() {
						goto l128
					}
					if !_rules[rule_]() {
						goto l128
					}
					if !_rules[ruleFilterCondition]() {
						goto l128
					}
					if !_rules[rule_]() {
						goto l128
					}
					if !_rules[ruleFilterValue]() {
						goto l128
					}
				}
			l130:
				add(ruleLogicExpr, position129)
			}
			return true
		l128:
			position, tokenIndex = position128, tokenIndex128
			return false
		},
		/* 11 OPERATOR <- <((('e' / 'E') ('q' / 'Q')) / (('n' / 'N') ('e' / 'E') ('q' / 'Q')))> */
		func() bool {
			position132, tokenIndex132 := position, tokenIndex
			{
				position133 := position
				{
					position134, tokenIndex134 := position, tokenIndex
					{
						position136, tokenIndex136 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l137
						}
						position++
						goto l136
					l137:
						position, tokenIndex = position136, tokenIndex136
						if buffer[position] != rune('E') {
							goto l135
						}
						position++
					}
				l136:
					{
						position138, tokenIndex138 := position, tokenIndex
						if buffer[position] != rune('q') {
							goto l139
						}
						position++
						goto l138
					l139:
						position, tokenIndex = position138, tokenIndex138
						if buffer[position] != rune('Q') {
							goto l135
						}
						position++
					}
				l138:
					goto l134
				l135:
					position, tokenIndex = position134, tokenIndex134
					{
						position140, tokenIndex140 := position, tokenIndex
						if buffer[position] != rune('n') {
							goto l141
						}
						position++
						goto l140
					l141:
						position, tokenIndex = position140, tokenIndex140
						if buffer[position] != rune('N') {
							goto l132
						}
						position++
					}
				l140:
					{
						position142, tokenIndex142 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l143
						}
						position++
						goto l142
					l143:
						position, tokenIndex = position142, tokenIndex142
						if buffer[position] != rune('E') {
							goto l132
						}
						position++
					}
				l142:
					{
						position144, tokenIndex144 := position, tokenIndex
						if buffer[position] != rune('q') {
							goto l145
						}
						position++
						goto l144
					l145:
						position, tokenIndex = position144, tokenIndex144
						if buffer[position] != rune('Q') {
							goto l132
						}
						position++
					}
				l144:
				}
			l134:
				add(ruleOPERATOR, position133)
			}
			return true
		l132:
			position, tokenIndex = position132, tokenIndex132
			return false
		},
		/* 12 FilterKey <- <(<Identifier> Action10)> */
		func() bool {
			position146, tokenIndex146 := position, tokenIndex
			{
				position147 := position
				{
					position148 := position
					if !_rules[ruleIdentifier]() {
						goto l146
					}
					add(rulePegText, position148)
				}
				if !_rules[ruleAction10]() {
					goto l146
				}
				add(ruleFilterKey, position147)
			}
			return true
		l146:
			position, tokenIndex = position146, tokenIndex146
			return false
		},
		/* 13 FilterCondition <- <(<OPERATOR> Action11)> */
		func() bool {
			position149, tokenIndex149 := position, tokenIndex
			{
				position150 := position
				{
					position151 := position
					if !_rules[ruleOPERATOR]() {
						goto l149
					}
					add(rulePegText, position151)
				}
				if !_rules[ruleAction11]() {
					goto l149
				}
				add(ruleFilterCondition, position150)
			}
			return true
		l149:
			position, tokenIndex = position149, tokenIndex149
			return false
		},
		/* 14 FilterValue <- <(<Value> Action12)> */
		func() bool {
			position152, tokenIndex152 := position, tokenIndex
			{
				position153 := position
				{
					position154 := position
					if !_rules[ruleValue]() {
						goto l152
					}
					add(rulePegText, position154)
				}
				if !_rules[ruleAction12]() {
					goto l152
				}
				add(ruleFilterValue, position153)
			}
			return true
		l152:
			position, tokenIndex = position152, tokenIndex152
			return false
		},
		/* 15 Value <- <(Float / Integer / String)> */
		func() bool {
			position155, tokenIndex155 := position, tokenIndex
			{
				position156 := position
				{
					position157, tokenIndex157 := position, tokenIndex
					if !_rules[ruleFloat]() {
						goto l158
					}
					goto l157
				l158:
					position, tokenIndex = position157, tokenIndex157
					if !_rules[ruleInteger]() {
						goto l159
					}
					goto l157
				l159:
					position, tokenIndex = position157, tokenIndex157
					if !_rules[ruleString]() {
						goto l155
					}
				}
			l157:
				add(ruleValue, position156)
			}
			return true
		l155:
			position, tokenIndex = position155, tokenIndex155
			return false
		},
		/* 16 Descending <- <(('d' / 'D') ('e' / 'E') ('s' / 'S') ('c' / 'C') Action13)> */
		func() bool {
			position160, tokenIndex160 := position, tokenIndex
			{
				position161 := position
				{
					position162, tokenIndex162 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l163
					}
					position++
					goto l162
				l163:
					position, tokenIndex = position162, tokenIndex162
					if buffer[position] != rune('D') {
						goto l160
					}
					position++
				}
			l162:
				{
					position164, tokenIndex164 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l165
					}
					position++
					goto l164
				l165:
					position, tokenIndex = position164, tokenIndex164
					if buffer[position] != rune('E') {
						goto l160
					}
					position++
				}
			l164:
				{
					position166, tokenIndex166 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l167
					}
					position++
					goto l166
				l167:
					position, tokenIndex = position166, tokenIndex166
					if buffer[position] != rune('S') {
						goto l160
					}
					position++
				}
			l166:
				{
					position168, tokenIndex168 := position, tokenIndex
					if buffer[position] != rune('c') {
						goto l169
					}
					position++
					goto l168
				l169:
					position, tokenIndex = position168, tokenIndex168
					if buffer[position] != rune('C') {
						goto l160
					}
					position++
				}
			l168:
				if !_rules[ruleAction13]() {
					goto l160
				}
				add(ruleDescending, position161)
			}
			return true
		l160:
			position, tokenIndex = position160, tokenIndex160
			return false
		},
		/* 17 String <- <('"' <StringChar*> '"')+> */
		func() bool {
			position170, tokenIndex170 := position, tokenIndex
			{
				position171 := position
				if buffer[position] != rune('"') {
					goto l170
				}
				position++
				{
					position174 := position
				l175:
					{
						position176, tokenIndex176 := position, tokenIndex
						if !_rules[ruleStringChar]() {
							goto l176
						}
						goto l175
					l176:
						position, tokenIndex = position176, tokenIndex176
					}
					add(rulePegText, position174)
				}
				if buffer[position] != rune('"') {
					goto l170
				}
				position++
			l172:
				{
					position173, tokenIndex173 := position, tokenIndex
					if buffer[position] != rune('"') {
						goto l173
					}
					position++
					{
						position177 := position
					l178:
						{
							position179, tokenIndex179 := position, tokenIndex
							if !_rules[ruleStringChar]() {
								goto l179
							}
							goto l178
						l179:
							position, tokenIndex = position179, tokenIndex179
						}
						add(rulePegText, position177)
					}
					if buffer[position] != rune('"') {
						goto l173
					}
					position++
					goto l172
				l173:
					position, tokenIndex = position173, tokenIndex173
				}
				add(ruleString, position171)
			}
			return true
		l170:
			position, tokenIndex = position170, tokenIndex170
			return false
		},
		/* 18 StringChar <- <(Escape / (!('"' / '\n' / '\\') .))> */
		func() bool {
			position180, tokenIndex180 := position, tokenIndex
			{
				position181 := position
				{
					position182, tokenIndex182 := position, tokenIndex
					if !_rules[ruleEscape]() {
						goto l183
					}
					goto l182
				l183:
					position, tokenIndex = position182, tokenIndex182
					{
						position184, tokenIndex184 := position, tokenIndex
						{
							position185, tokenIndex185 := position, tokenIndex
							if buffer[position] != rune('"') {
								goto l186
							}
							position++
							goto l185
						l186:
							position, tokenIndex = position185, tokenIndex185
							if buffer[position] != rune('\n') {
								goto l187
							}
							position++
							goto l185
						l187:
							position, tokenIndex = position185, tokenIndex185
							if buffer[position] != rune('\\') {
								goto l184
							}
							position++
						}
					l185:
						goto l180
					l184:
						position, tokenIndex = position184, tokenIndex184
					}
					if !matchDot() {
						goto l180
					}
				}
			l182:
				add(ruleStringChar, position181)
			}
			return true
		l180:
			position, tokenIndex = position180, tokenIndex180
			return false
		},
		/* 19 Escape <- <(SimpleEscape / OctalEscape / HexEscape / UniversalCharacter)> */
		func() bool {
			position188, tokenIndex188 := position, tokenIndex
			{
				position189 := position
				{
					position190, tokenIndex190 := position, tokenIndex
					if !_rules[ruleSimpleEscape]() {
						goto l191
					}
					goto l190
				l191:
					position, tokenIndex = position190, tokenIndex190
					if !_rules[ruleOctalEscape]() {
						goto l192
					}
					goto l190
				l192:
					position, tokenIndex = position190, tokenIndex190
					if !_rules[ruleHexEscape]() {
						goto l193
					}
					goto l190
				l193:
					position, tokenIndex = position190, tokenIndex190
					if !_rules[ruleUniversalCharacter]() {
						goto l188
					}
				}
			l190:
				add(ruleEscape, position189)
			}
			return true
		l188:
			position, tokenIndex = position188, tokenIndex188
			return false
		},
		/* 20 SimpleEscape <- <('\\' ('\'' / '"' / '?' / '\\' / 'a' / 'b' / 'f' / 'n' / 'r' / 't' / 'v'))> */
		func() bool {
			position194, tokenIndex194 := position, tokenIndex
			{
				position195 := position
				if buffer[position] != rune('\\') {
					goto l194
				}
				position++
				{
					position196, tokenIndex196 := position, tokenIndex
					if buffer[position] != rune('\'') {
						goto l197
					}
					position++
					goto l196
				l197:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('"') {
						goto l198
					}
					position++
					goto l196
				l198:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('?') {
						goto l199
					}
					position++
					goto l196
				l199:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('\\') {
						goto l200
					}
					position++
					goto l196
				l200:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('a') {
						goto l201
					}
					position++
					goto l196
				l201:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('b') {
						goto l202
					}
					position++
					goto l196
				l202:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('f') {
						goto l203
					}
					position++
					goto l196
				l203:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('n') {
						goto l204
					}
					position++
					goto l196
				l204:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('r') {
						goto l205
					}
					position++
					goto l196
				l205:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('t') {
						goto l206
					}
					position++
					goto l196
				l206:
					position, tokenIndex = position196, tokenIndex196
					if buffer[position] != rune('v') {
						goto l194
					}
					position++
				}
			l196:
				add(ruleSimpleEscape, position195)
			}
			return true
		l194:
			position, tokenIndex = position194, tokenIndex194
			return false
		},
		/* 21 OctalEscape <- <('\\' [0-7] [0-7]? [0-7]?)> */
		func() bool {
			position207, tokenIndex207 := position, tokenIndex
			{
				position208 := position
				if buffer[position] != rune('\\') {
					goto l207
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('7') {
					goto l207
				}
				position++
				{
					position209, tokenIndex209 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l209
					}
					position++
					goto l210
				l209:
					position, tokenIndex = position209, tokenIndex209
				}
			l210:
				{
					position211, tokenIndex211 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l211
					}
					position++
					goto l212
				l211:
					position, tokenIndex = position211, tokenIndex211
				}
			l212:
				add(ruleOctalEscape, position208)
			}
			return true
		l207:
			position, tokenIndex = position207, tokenIndex207
			return false
		},
		/* 22 HexEscape <- <('\\' 'x' HexDigit+)> */
		func() bool {
			position213, tokenIndex213 := position, tokenIndex
			{
				position214 := position
				if buffer[position] != rune('\\') {
					goto l213
				}
				position++
				if buffer[position] != rune('x') {
					goto l213
				}
				position++
				if !_rules[ruleHexDigit]() {
					goto l213
				}
			l215:
				{
					position216, tokenIndex216 := position, tokenIndex
					if !_rules[ruleHexDigit]() {
						goto l216
					}
					goto l215
				l216:
					position, tokenIndex = position216, tokenIndex216
				}
				add(ruleHexEscape, position214)
			}
			return true
		l213:
			position, tokenIndex = position213, tokenIndex213
			return false
		},
		/* 23 UniversalCharacter <- <(('\\' 'u' HexQuad) / ('\\' 'U' HexQuad HexQuad))> */
		func() bool {
			position217, tokenIndex217 := position, tokenIndex
			{
				position218 := position
				{
					position219, tokenIndex219 := position, tokenIndex
					if buffer[position] != rune('\\') {
						goto l220
					}
					position++
					if buffer[position] != rune('u') {
						goto l220
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l220
					}
					goto l219
				l220:
					position, tokenIndex = position219, tokenIndex219
					if buffer[position] != rune('\\') {
						goto l217
					}
					position++
					if buffer[position] != rune('U') {
						goto l217
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l217
					}
					if !_rules[ruleHexQuad]() {
						goto l217
					}
				}
			l219:
				add(ruleUniversalCharacter, position218)
			}
			return true
		l217:
			position, tokenIndex = position217, tokenIndex217
			return false
		},
		/* 24 HexQuad <- <(HexDigit HexDigit HexDigit HexDigit)> */
		func() bool {
			position221, tokenIndex221 := position, tokenIndex
			{
				position222 := position
				if !_rules[ruleHexDigit]() {
					goto l221
				}
				if !_rules[ruleHexDigit]() {
					goto l221
				}
				if !_rules[ruleHexDigit]() {
					goto l221
				}
				if !_rules[ruleHexDigit]() {
					goto l221
				}
				add(ruleHexQuad, position222)
			}
			return true
		l221:
			position, tokenIndex = position221, tokenIndex221
			return false
		},
		/* 25 HexDigit <- <([a-f] / [A-F] / [0-9])> */
		func() bool {
			position223, tokenIndex223 := position, tokenIndex
			{
				position224 := position
				{
					position225, tokenIndex225 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('f') {
						goto l226
					}
					position++
					goto l225
				l226:
					position, tokenIndex = position225, tokenIndex225
					if c := buffer[position]; c < rune('A') || c > rune('F') {
						goto l227
					}
					position++
					goto l225
				l227:
					position, tokenIndex = position225, tokenIndex225
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l223
					}
					position++
				}
			l225:
				add(ruleHexDigit, position224)
			}
			return true
		l223:
			position, tokenIndex = position223, tokenIndex223
			return false
		},
		/* 26 Unsigned <- <[0-9]+> */
		func() bool {
			position228, tokenIndex228 := position, tokenIndex
			{
				position229 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l228
				}
				position++
			l230:
				{
					position231, tokenIndex231 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l231
					}
					position++
					goto l230
				l231:
					position, tokenIndex = position231, tokenIndex231
				}
				add(ruleUnsigned, position229)
			}
			return true
		l228:
			position, tokenIndex = position228, tokenIndex228
			return false
		},
		/* 27 Sign <- <('-' / '+')> */
		func() bool {
			position232, tokenIndex232 := position, tokenIndex
			{
				position233 := position
				{
					position234, tokenIndex234 := position, tokenIndex
					if buffer[position] != rune('-') {
						goto l235
					}
					position++
					goto l234
				l235:
					position, tokenIndex = position234, tokenIndex234
					if buffer[position] != rune('+') {
						goto l232
					}
					position++
				}
			l234:
				add(ruleSign, position233)
			}
			return true
		l232:
			position, tokenIndex = position232, tokenIndex232
			return false
		},
		/* 28 Integer <- <<(Sign? Unsigned)>> */
		func() bool {
			position236, tokenIndex236 := position, tokenIndex
			{
				position237 := position
				{
					position238 := position
					{
						position239, tokenIndex239 := position, tokenIndex
						if !_rules[ruleSign]() {
							goto l239
						}
						goto l240
					l239:
						position, tokenIndex = position239, tokenIndex239
					}
				l240:
					if !_rules[ruleUnsigned]() {
						goto l236
					}
					add(rulePegText, position238)
				}
				add(ruleInteger, position237)
			}
			return true
		l236:
			position, tokenIndex = position236, tokenIndex236
			return false
		},
		/* 29 Float <- <(Integer ('.' Unsigned)? (('e' / 'E') Integer)?)> */
		func() bool {
			position241, tokenIndex241 := position, tokenIndex
			{
				position242 := position
				if !_rules[ruleInteger]() {
					goto l241
				}
				{
					position243, tokenIndex243 := position, tokenIndex
					if buffer[position] != rune('.') {
						goto l243
					}
					position++
					if !_rules[ruleUnsigned]() {
						goto l243
					}
					goto l244
				l243:
					position, tokenIndex = position243, tokenIndex243
				}
			l244:
				{
					position245, tokenIndex245 := position, tokenIndex
					{
						position247, tokenIndex247 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l248
						}
						position++
						goto l247
					l248:
						position, tokenIndex = position247, tokenIndex247
						if buffer[position] != rune('E') {
							goto l245
						}
						position++
					}
				l247:
					if !_rules[ruleInteger]() {
						goto l245
					}
					goto l246
				l245:
					position, tokenIndex = position245, tokenIndex245
				}
			l246:
				add(ruleFloat, position242)
			}
			return true
		l241:
			position, tokenIndex = position241, tokenIndex241
			return false
		},
		/* 30 Duration <- <(Integer ('.' Unsigned)? (('n' 's') / ('u' 's') / ('µ' 's') / ('m' 's') / 's' / 'm' / 'h'))> */
		func() bool {
			position249, tokenIndex249 := position, tokenIndex
			{
				position250 := position
				if !_rules[ruleInteger]() {
					goto l249
				}
				{
					position251, tokenIndex251 := position, tokenIndex
					if buffer[position] != rune('.') {
						goto l251
					}
					position++
					if !_rules[ruleUnsigned]() {
						goto l251
					}
					goto l252
				l251:
					position, tokenIndex = position251, tokenIndex251
				}
			l252:
				{
					position253, tokenIndex253 := position, tokenIndex
					if buffer[position] != rune('n') {
						goto l254
					}
					position++
					if buffer[position] != rune('s') {
						goto l254
					}
					position++
					goto l253
				l254:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('u') {
						goto l255
					}
					position++
					if buffer[position] != rune('s') {
						goto l255
					}
					position++
					goto l253
				l255:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('µ') {
						goto l256
					}
					position++
					if buffer[position] != rune('s') {
						goto l256
					}
					position++
					goto l253
				l256:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('m') {
						goto l257
					}
					position++
					if buffer[position] != rune('s') {
						goto l257
					}
					position++
					goto l253
				l257:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('s') {
						goto l258
					}
					position++
					goto l253
				l258:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('m') {
						goto l259
					}
					position++
					goto l253
				l259:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('h') {
						goto l249
					}
					position++
				}
			l253:
				add(ruleDuration, position250)
			}
			return true
		l249:
			position, tokenIndex = position249, tokenIndex249
			return false
		},
		/* 31 Identifier <- <(!Keyword <(([a-z] / [A-Z] / '_') IdChar*)>)> */
		func() bool {
			position260, tokenIndex260 := position, tokenIndex
			{
				position261 := position
				{
					position262, tokenIndex262 := position, tokenIndex
					if !_rules[ruleKeyword]() {
						goto l262
					}
					goto l260
				l262:
					position, tokenIndex = position262, tokenIndex262
				}
				{
					position263 := position
					{
						position264, tokenIndex264 := position, tokenIndex
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l265
						}
						position++
						goto l264
					l265:
						position, tokenIndex = position264, tokenIndex264
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l266
						}
						position++
						goto l264
					l266:
						position, tokenIndex = position264, tokenIndex264
						if buffer[position] != rune('_') {
							goto l260
						}
						position++
					}
				l264:
				l267:
					{
						position268, tokenIndex268 := position, tokenIndex
						if !_rules[ruleIdChar]() {
							goto l268
						}
						goto l267
					l268:
						position, tokenIndex = position268, tokenIndex268
					}
					add(rulePegText, position263)
				}
				add(ruleIdentifier, position261)
			}
			return true
		l260:
			position, tokenIndex = position260, tokenIndex260
			return false
		},
		/* 32 IdChar <- <([a-z] / [A-Z] / [0-9] / '_')> */
		func() bool {
			position269, tokenIndex269 := position, tokenIndex
			{
				position270 := position
				{
					position271, tokenIndex271 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l272
					}
					position++
					goto l271
				l272:
					position, tokenIndex = position271, tokenIndex271
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l273
					}
					position++
					goto l271
				l273:
					position, tokenIndex = position271, tokenIndex271
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l274
					}
					position++
					goto l271
				l274:
					position, tokenIndex = position271, tokenIndex271
					if buffer[position] != rune('_') {
						goto l269
					}
					position++
				}
			l271:
				add(ruleIdChar, position270)
			}
			return true
		l269:
			position, tokenIndex = position269, tokenIndex269
			return false
		},
		/* 33 Keyword <- <((('s' 'e' 'l' 'e' 'c' 't') / ('g' 'r' 'o' 'u' 'p' ' ' 'b' 'y') / ('f' 'i' 'l' 't' 'e' 'r' 's') / ('o' 'r' 'd' 'e' 'r' ' ' 'b' 'y') / ('d' 'e' 's' 'c') / ('l' 'i' 'm' 'i' 't') / ('e' 'q') / ('n' 'e' 'q')) !IdChar)> */
		func() bool {
			position275, tokenIndex275 := position, tokenIndex
			{
				position276 := position
				{
					position277, tokenIndex277 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l278
					}
					position++
					if buffer[position] != rune('e') {
						goto l278
					}
					position++
					if buffer[position] != rune('l') {
						goto l278
					}
					position++
					if buffer[position] != rune('e') {
						goto l278
					}
					position++
					if buffer[position] != rune('c') {
						goto l278
					}
					position++
					if buffer[position] != rune('t') {
						goto l278
					}
					position++
					goto l277
				l278:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('g') {
						goto l279
					}
					position++
					if buffer[position] != rune('r') {
						goto l279
					}
					position++
					if buffer[position] != rune('o') {
						goto l279
					}
					position++
					if buffer[position] != rune('u') {
						goto l279
					}
					position++
					if buffer[position] != rune('p') {
						goto l279
					}
					position++
					if buffer[position] != rune(' ') {
						goto l279
					}
					position++
					if buffer[position] != rune('b') {
						goto l279
					}
					position++
					if buffer[position] != rune('y') {
						goto l279
					}
					position++
					goto l277
				l279:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('f') {
						goto l280
					}
					position++
					if buffer[position] != rune('i') {
						goto l280
					}
					position++
					if buffer[position] != rune('l') {
						goto l280
					}
					position++
					if buffer[position] != rune('t') {
						goto l280
					}
					position++
					if buffer[position] != rune('e') {
						goto l280
					}
					position++
					if buffer[position] != rune('r') {
						goto l280
					}
					position++
					if buffer[position] != rune('s') {
						goto l280
					}
					position++
					goto l277
				l280:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('o') {
						goto l281
					}
					position++
					if buffer[position] != rune('r') {
						goto l281
					}
					position++
					if buffer[position] != rune('d') {
						goto l281
					}
					position++
					if buffer[position] != rune('e') {
						goto l281
					}
					position++
					if buffer[position] != rune('r') {
						goto l281
					}
					position++
					if buffer[position] != rune(' ') {
						goto l281
					}
					position++
					if buffer[position] != rune('b') {
						goto l281
					}
					position++
					if buffer[position] != rune('y') {
						goto l281
					}
					position++
					goto l277
				l281:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('d') {
						goto l282
					}
					position++
					if buffer[position] != rune('e') {
						goto l282
					}
					position++
					if buffer[position] != rune('s') {
						goto l282
					}
					position++
					if buffer[position] != rune('c') {
						goto l282
					}
					position++
					goto l277
				l282:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('l') {
						goto l283
					}
					position++
					if buffer[position] != rune('i') {
						goto l283
					}
					position++
					if buffer[position] != rune('m') {
						goto l283
					}
					position++
					if buffer[position] != rune('i') {
						goto l283
					}
					position++
					if buffer[position] != rune('t') {
						goto l283
					}
					position++
					goto l277
				l283:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('e') {
						goto l284
					}
					position++
					if buffer[position] != rune('q') {
						goto l284
					}
					position++
					goto l277
				l284:
					position, tokenIndex = position277, tokenIndex277
					if buffer[position] != rune('n') {
						goto l275
					}
					position++
					if buffer[position] != rune('e') {
						goto l275
					}
					position++
					if buffer[position] != rune('q') {
						goto l275
					}
					position++
				}
			l277:
				{
					position285, tokenIndex285 := position, tokenIndex
					if !_rules[ruleIdChar]() {
						goto l285
					}
					goto l275
				l285:
					position, tokenIndex = position285, tokenIndex285
				}
				add(ruleKeyword, position276)
			}
			return true
		l275:
			position, tokenIndex = position275, tokenIndex275
			return false
		},
		/* 34 _ <- <(' ' / '\t' / ('\r' '\n') / '\n' / '\r')*> */
		func() bool {
			{
				position287 := position
			l288:
				{
					position289, tokenIndex289 := position, tokenIndex
					{
						position290, tokenIndex290 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l291
						}
						position++
						goto l290
					l291:
						position, tokenIndex = position290, tokenIndex290
						if buffer[position] != rune('\t') {
							goto l292
						}
						position++
						goto l290
					l292:
						position, tokenIndex = position290, tokenIndex290
						if buffer[position] != rune('\r') {
							goto l293
						}
						position++
						if buffer[position] != rune('\n') {
							goto l293
						}
						position++
						goto l290
					l293:
						position, tokenIndex = position290, tokenIndex290
						if buffer[position] != rune('\n') {
							goto l294
						}
						position++
						goto l290
					l294:
						position, tokenIndex = position290, tokenIndex290
						if buffer[position] != rune('\r') {
							goto l289
						}
						position++
					}
				l290:
					goto l288
				l289:
					position, tokenIndex = position289, tokenIndex289
				}
				add(rule_, position287)
			}
			return true
		},
		/* 35 LPAR <- <(_ '(' _)> */
		func() bool {
			position295, tokenIndex295 := position, tokenIndex
			{
				position296 := position
				if !_rules[rule_]() {
					goto l295
				}
				if buffer[position] != rune('(') {
					goto l295
				}
				position++
				if !_rules[rule_]() {
					goto l295
				}
				add(ruleLPAR, position296)
			}
			return true
		l295:
			position, tokenIndex = position295, tokenIndex295
			return false
		},
		/* 36 RPAR <- <(_ ')' _)> */
		func() bool {
			position297, tokenIndex297 := position, tokenIndex
			{
				position298 := position
				if !_rules[rule_]() {
					goto l297
				}
				if buffer[position] != rune(')') {
					goto l297
				}
				position++
				if !_rules[rule_]() {
					goto l297
				}
				add(ruleRPAR, position298)
			}
			return true
		l297:
			position, tokenIndex = position297, tokenIndex297
			return false
		},
		/* 37 COMMA <- <(_ ',' _)> */
		func() bool {
			position299, tokenIndex299 := position, tokenIndex
			{
				position300 := position
				if !_rules[rule_]() {
					goto l299
				}
				if buffer[position] != rune(',') {
					goto l299
				}
				position++
				if !_rules[rule_]() {
					goto l299
				}
				add(ruleCOMMA, position300)
			}
			return true
		l299:
			position, tokenIndex = position299, tokenIndex299
			return false
		},
		/* 39 Action0 <- <{ p.currentSection = "columns" }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 40 Action1 <- <{ p.currentSection = "group by" }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 41 Action2 <- <{ p.currentSection = "order by" }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		nil,
		/* 43 Action3 <- <{ p.SetLimit(text) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 44 Action4 <- <{ p.SetPointSize(text) }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 45 Action5 <- <{ p.AddColumn() }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 46 Action6 <- <{ p.SetColumnName(text) }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 47 Action7 <- <{ p.SetColumnAggregate(text) }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 48 Action8 <- <{ p.SetColumnName(text)      }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 49 Action9 <- <{ p.AddFilter() }> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 50 Action10 <- <{ p.SetFilterColumn(text) }> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 51 Action11 <- <{ p.SetFilterCondition(text) }> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 52 Action12 <- <{ p.SetFilterValue(text) }> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 53 Action13 <- <{ p.SetDescending() }> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
	}
	p.rules = _rules
}
