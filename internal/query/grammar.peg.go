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
		/* 11 OPERATOR <- <('=' / ('!' '=') / '<' / ('<' '=') / '>' / ('>' '=') / (('m' / 'M') ('a' / 'A') ('t' / 'T') ('c' / 'C') ('h' / 'H') ('e' / 'E') ('s' / 'S')))> */
		func() bool {
			position132, tokenIndex132 := position, tokenIndex
			{
				position133 := position
				{
					position134, tokenIndex134 := position, tokenIndex
					if buffer[position] != rune('=') {
						goto l135
					}
					position++
					goto l134
				l135:
					position, tokenIndex = position134, tokenIndex134
					if buffer[position] != rune('!') {
						goto l136
					}
					position++
					if buffer[position] != rune('=') {
						goto l136
					}
					position++
					goto l134
				l136:
					position, tokenIndex = position134, tokenIndex134
					if buffer[position] != rune('<') {
						goto l137
					}
					position++
					goto l134
				l137:
					position, tokenIndex = position134, tokenIndex134
					if buffer[position] != rune('<') {
						goto l138
					}
					position++
					if buffer[position] != rune('=') {
						goto l138
					}
					position++
					goto l134
				l138:
					position, tokenIndex = position134, tokenIndex134
					if buffer[position] != rune('>') {
						goto l139
					}
					position++
					goto l134
				l139:
					position, tokenIndex = position134, tokenIndex134
					if buffer[position] != rune('>') {
						goto l140
					}
					position++
					if buffer[position] != rune('=') {
						goto l140
					}
					position++
					goto l134
				l140:
					position, tokenIndex = position134, tokenIndex134
					{
						position141, tokenIndex141 := position, tokenIndex
						if buffer[position] != rune('m') {
							goto l142
						}
						position++
						goto l141
					l142:
						position, tokenIndex = position141, tokenIndex141
						if buffer[position] != rune('M') {
							goto l132
						}
						position++
					}
				l141:
					{
						position143, tokenIndex143 := position, tokenIndex
						if buffer[position] != rune('a') {
							goto l144
						}
						position++
						goto l143
					l144:
						position, tokenIndex = position143, tokenIndex143
						if buffer[position] != rune('A') {
							goto l132
						}
						position++
					}
				l143:
					{
						position145, tokenIndex145 := position, tokenIndex
						if buffer[position] != rune('t') {
							goto l146
						}
						position++
						goto l145
					l146:
						position, tokenIndex = position145, tokenIndex145
						if buffer[position] != rune('T') {
							goto l132
						}
						position++
					}
				l145:
					{
						position147, tokenIndex147 := position, tokenIndex
						if buffer[position] != rune('c') {
							goto l148
						}
						position++
						goto l147
					l148:
						position, tokenIndex = position147, tokenIndex147
						if buffer[position] != rune('C') {
							goto l132
						}
						position++
					}
				l147:
					{
						position149, tokenIndex149 := position, tokenIndex
						if buffer[position] != rune('h') {
							goto l150
						}
						position++
						goto l149
					l150:
						position, tokenIndex = position149, tokenIndex149
						if buffer[position] != rune('H') {
							goto l132
						}
						position++
					}
				l149:
					{
						position151, tokenIndex151 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l152
						}
						position++
						goto l151
					l152:
						position, tokenIndex = position151, tokenIndex151
						if buffer[position] != rune('E') {
							goto l132
						}
						position++
					}
				l151:
					{
						position153, tokenIndex153 := position, tokenIndex
						if buffer[position] != rune('s') {
							goto l154
						}
						position++
						goto l153
					l154:
						position, tokenIndex = position153, tokenIndex153
						if buffer[position] != rune('S') {
							goto l132
						}
						position++
					}
				l153:
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
			position155, tokenIndex155 := position, tokenIndex
			{
				position156 := position
				{
					position157 := position
					if !_rules[ruleIdentifier]() {
						goto l155
					}
					add(rulePegText, position157)
				}
				if !_rules[ruleAction10]() {
					goto l155
				}
				add(ruleFilterKey, position156)
			}
			return true
		l155:
			position, tokenIndex = position155, tokenIndex155
			return false
		},
		/* 13 FilterCondition <- <(<OPERATOR> Action11)> */
		func() bool {
			position158, tokenIndex158 := position, tokenIndex
			{
				position159 := position
				{
					position160 := position
					if !_rules[ruleOPERATOR]() {
						goto l158
					}
					add(rulePegText, position160)
				}
				if !_rules[ruleAction11]() {
					goto l158
				}
				add(ruleFilterCondition, position159)
			}
			return true
		l158:
			position, tokenIndex = position158, tokenIndex158
			return false
		},
		/* 14 FilterValue <- <(<Value> Action12)> */
		func() bool {
			position161, tokenIndex161 := position, tokenIndex
			{
				position162 := position
				{
					position163 := position
					if !_rules[ruleValue]() {
						goto l161
					}
					add(rulePegText, position163)
				}
				if !_rules[ruleAction12]() {
					goto l161
				}
				add(ruleFilterValue, position162)
			}
			return true
		l161:
			position, tokenIndex = position161, tokenIndex161
			return false
		},
		/* 15 Value <- <(Float / Integer / String)> */
		func() bool {
			position164, tokenIndex164 := position, tokenIndex
			{
				position165 := position
				{
					position166, tokenIndex166 := position, tokenIndex
					if !_rules[ruleFloat]() {
						goto l167
					}
					goto l166
				l167:
					position, tokenIndex = position166, tokenIndex166
					if !_rules[ruleInteger]() {
						goto l168
					}
					goto l166
				l168:
					position, tokenIndex = position166, tokenIndex166
					if !_rules[ruleString]() {
						goto l164
					}
				}
			l166:
				add(ruleValue, position165)
			}
			return true
		l164:
			position, tokenIndex = position164, tokenIndex164
			return false
		},
		/* 16 Descending <- <(('d' / 'D') ('e' / 'E') ('s' / 'S') ('c' / 'C') Action13)> */
		func() bool {
			position169, tokenIndex169 := position, tokenIndex
			{
				position170 := position
				{
					position171, tokenIndex171 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l172
					}
					position++
					goto l171
				l172:
					position, tokenIndex = position171, tokenIndex171
					if buffer[position] != rune('D') {
						goto l169
					}
					position++
				}
			l171:
				{
					position173, tokenIndex173 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l174
					}
					position++
					goto l173
				l174:
					position, tokenIndex = position173, tokenIndex173
					if buffer[position] != rune('E') {
						goto l169
					}
					position++
				}
			l173:
				{
					position175, tokenIndex175 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l176
					}
					position++
					goto l175
				l176:
					position, tokenIndex = position175, tokenIndex175
					if buffer[position] != rune('S') {
						goto l169
					}
					position++
				}
			l175:
				{
					position177, tokenIndex177 := position, tokenIndex
					if buffer[position] != rune('c') {
						goto l178
					}
					position++
					goto l177
				l178:
					position, tokenIndex = position177, tokenIndex177
					if buffer[position] != rune('C') {
						goto l169
					}
					position++
				}
			l177:
				if !_rules[ruleAction13]() {
					goto l169
				}
				add(ruleDescending, position170)
			}
			return true
		l169:
			position, tokenIndex = position169, tokenIndex169
			return false
		},
		/* 17 String <- <('"' <StringChar*> '"')+> */
		func() bool {
			position179, tokenIndex179 := position, tokenIndex
			{
				position180 := position
				if buffer[position] != rune('"') {
					goto l179
				}
				position++
				{
					position183 := position
				l184:
					{
						position185, tokenIndex185 := position, tokenIndex
						if !_rules[ruleStringChar]() {
							goto l185
						}
						goto l184
					l185:
						position, tokenIndex = position185, tokenIndex185
					}
					add(rulePegText, position183)
				}
				if buffer[position] != rune('"') {
					goto l179
				}
				position++
			l181:
				{
					position182, tokenIndex182 := position, tokenIndex
					if buffer[position] != rune('"') {
						goto l182
					}
					position++
					{
						position186 := position
					l187:
						{
							position188, tokenIndex188 := position, tokenIndex
							if !_rules[ruleStringChar]() {
								goto l188
							}
							goto l187
						l188:
							position, tokenIndex = position188, tokenIndex188
						}
						add(rulePegText, position186)
					}
					if buffer[position] != rune('"') {
						goto l182
					}
					position++
					goto l181
				l182:
					position, tokenIndex = position182, tokenIndex182
				}
				add(ruleString, position180)
			}
			return true
		l179:
			position, tokenIndex = position179, tokenIndex179
			return false
		},
		/* 18 StringChar <- <(Escape / (!('"' / '\n' / '\\') .))> */
		func() bool {
			position189, tokenIndex189 := position, tokenIndex
			{
				position190 := position
				{
					position191, tokenIndex191 := position, tokenIndex
					if !_rules[ruleEscape]() {
						goto l192
					}
					goto l191
				l192:
					position, tokenIndex = position191, tokenIndex191
					{
						position193, tokenIndex193 := position, tokenIndex
						{
							position194, tokenIndex194 := position, tokenIndex
							if buffer[position] != rune('"') {
								goto l195
							}
							position++
							goto l194
						l195:
							position, tokenIndex = position194, tokenIndex194
							if buffer[position] != rune('\n') {
								goto l196
							}
							position++
							goto l194
						l196:
							position, tokenIndex = position194, tokenIndex194
							if buffer[position] != rune('\\') {
								goto l193
							}
							position++
						}
					l194:
						goto l189
					l193:
						position, tokenIndex = position193, tokenIndex193
					}
					if !matchDot() {
						goto l189
					}
				}
			l191:
				add(ruleStringChar, position190)
			}
			return true
		l189:
			position, tokenIndex = position189, tokenIndex189
			return false
		},
		/* 19 Escape <- <(SimpleEscape / OctalEscape / HexEscape / UniversalCharacter)> */
		func() bool {
			position197, tokenIndex197 := position, tokenIndex
			{
				position198 := position
				{
					position199, tokenIndex199 := position, tokenIndex
					if !_rules[ruleSimpleEscape]() {
						goto l200
					}
					goto l199
				l200:
					position, tokenIndex = position199, tokenIndex199
					if !_rules[ruleOctalEscape]() {
						goto l201
					}
					goto l199
				l201:
					position, tokenIndex = position199, tokenIndex199
					if !_rules[ruleHexEscape]() {
						goto l202
					}
					goto l199
				l202:
					position, tokenIndex = position199, tokenIndex199
					if !_rules[ruleUniversalCharacter]() {
						goto l197
					}
				}
			l199:
				add(ruleEscape, position198)
			}
			return true
		l197:
			position, tokenIndex = position197, tokenIndex197
			return false
		},
		/* 20 SimpleEscape <- <('\\' ('\'' / '"' / '?' / '\\' / 'a' / 'b' / 'f' / 'n' / 'r' / 't' / 'v'))> */
		func() bool {
			position203, tokenIndex203 := position, tokenIndex
			{
				position204 := position
				if buffer[position] != rune('\\') {
					goto l203
				}
				position++
				{
					position205, tokenIndex205 := position, tokenIndex
					if buffer[position] != rune('\'') {
						goto l206
					}
					position++
					goto l205
				l206:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('"') {
						goto l207
					}
					position++
					goto l205
				l207:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('?') {
						goto l208
					}
					position++
					goto l205
				l208:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('\\') {
						goto l209
					}
					position++
					goto l205
				l209:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('a') {
						goto l210
					}
					position++
					goto l205
				l210:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('b') {
						goto l211
					}
					position++
					goto l205
				l211:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('f') {
						goto l212
					}
					position++
					goto l205
				l212:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('n') {
						goto l213
					}
					position++
					goto l205
				l213:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('r') {
						goto l214
					}
					position++
					goto l205
				l214:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('t') {
						goto l215
					}
					position++
					goto l205
				l215:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('v') {
						goto l203
					}
					position++
				}
			l205:
				add(ruleSimpleEscape, position204)
			}
			return true
		l203:
			position, tokenIndex = position203, tokenIndex203
			return false
		},
		/* 21 OctalEscape <- <('\\' [0-7] [0-7]? [0-7]?)> */
		func() bool {
			position216, tokenIndex216 := position, tokenIndex
			{
				position217 := position
				if buffer[position] != rune('\\') {
					goto l216
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('7') {
					goto l216
				}
				position++
				{
					position218, tokenIndex218 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l218
					}
					position++
					goto l219
				l218:
					position, tokenIndex = position218, tokenIndex218
				}
			l219:
				{
					position220, tokenIndex220 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l220
					}
					position++
					goto l221
				l220:
					position, tokenIndex = position220, tokenIndex220
				}
			l221:
				add(ruleOctalEscape, position217)
			}
			return true
		l216:
			position, tokenIndex = position216, tokenIndex216
			return false
		},
		/* 22 HexEscape <- <('\\' 'x' HexDigit+)> */
		func() bool {
			position222, tokenIndex222 := position, tokenIndex
			{
				position223 := position
				if buffer[position] != rune('\\') {
					goto l222
				}
				position++
				if buffer[position] != rune('x') {
					goto l222
				}
				position++
				if !_rules[ruleHexDigit]() {
					goto l222
				}
			l224:
				{
					position225, tokenIndex225 := position, tokenIndex
					if !_rules[ruleHexDigit]() {
						goto l225
					}
					goto l224
				l225:
					position, tokenIndex = position225, tokenIndex225
				}
				add(ruleHexEscape, position223)
			}
			return true
		l222:
			position, tokenIndex = position222, tokenIndex222
			return false
		},
		/* 23 UniversalCharacter <- <(('\\' 'u' HexQuad) / ('\\' 'U' HexQuad HexQuad))> */
		func() bool {
			position226, tokenIndex226 := position, tokenIndex
			{
				position227 := position
				{
					position228, tokenIndex228 := position, tokenIndex
					if buffer[position] != rune('\\') {
						goto l229
					}
					position++
					if buffer[position] != rune('u') {
						goto l229
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l229
					}
					goto l228
				l229:
					position, tokenIndex = position228, tokenIndex228
					if buffer[position] != rune('\\') {
						goto l226
					}
					position++
					if buffer[position] != rune('U') {
						goto l226
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l226
					}
					if !_rules[ruleHexQuad]() {
						goto l226
					}
				}
			l228:
				add(ruleUniversalCharacter, position227)
			}
			return true
		l226:
			position, tokenIndex = position226, tokenIndex226
			return false
		},
		/* 24 HexQuad <- <(HexDigit HexDigit HexDigit HexDigit)> */
		func() bool {
			position230, tokenIndex230 := position, tokenIndex
			{
				position231 := position
				if !_rules[ruleHexDigit]() {
					goto l230
				}
				if !_rules[ruleHexDigit]() {
					goto l230
				}
				if !_rules[ruleHexDigit]() {
					goto l230
				}
				if !_rules[ruleHexDigit]() {
					goto l230
				}
				add(ruleHexQuad, position231)
			}
			return true
		l230:
			position, tokenIndex = position230, tokenIndex230
			return false
		},
		/* 25 HexDigit <- <([a-f] / [A-F] / [0-9])> */
		func() bool {
			position232, tokenIndex232 := position, tokenIndex
			{
				position233 := position
				{
					position234, tokenIndex234 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('f') {
						goto l235
					}
					position++
					goto l234
				l235:
					position, tokenIndex = position234, tokenIndex234
					if c := buffer[position]; c < rune('A') || c > rune('F') {
						goto l236
					}
					position++
					goto l234
				l236:
					position, tokenIndex = position234, tokenIndex234
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l232
					}
					position++
				}
			l234:
				add(ruleHexDigit, position233)
			}
			return true
		l232:
			position, tokenIndex = position232, tokenIndex232
			return false
		},
		/* 26 Unsigned <- <[0-9]+> */
		func() bool {
			position237, tokenIndex237 := position, tokenIndex
			{
				position238 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l237
				}
				position++
			l239:
				{
					position240, tokenIndex240 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l240
					}
					position++
					goto l239
				l240:
					position, tokenIndex = position240, tokenIndex240
				}
				add(ruleUnsigned, position238)
			}
			return true
		l237:
			position, tokenIndex = position237, tokenIndex237
			return false
		},
		/* 27 Sign <- <('-' / '+')> */
		func() bool {
			position241, tokenIndex241 := position, tokenIndex
			{
				position242 := position
				{
					position243, tokenIndex243 := position, tokenIndex
					if buffer[position] != rune('-') {
						goto l244
					}
					position++
					goto l243
				l244:
					position, tokenIndex = position243, tokenIndex243
					if buffer[position] != rune('+') {
						goto l241
					}
					position++
				}
			l243:
				add(ruleSign, position242)
			}
			return true
		l241:
			position, tokenIndex = position241, tokenIndex241
			return false
		},
		/* 28 Integer <- <<(Sign? Unsigned)>> */
		func() bool {
			position245, tokenIndex245 := position, tokenIndex
			{
				position246 := position
				{
					position247 := position
					{
						position248, tokenIndex248 := position, tokenIndex
						if !_rules[ruleSign]() {
							goto l248
						}
						goto l249
					l248:
						position, tokenIndex = position248, tokenIndex248
					}
				l249:
					if !_rules[ruleUnsigned]() {
						goto l245
					}
					add(rulePegText, position247)
				}
				add(ruleInteger, position246)
			}
			return true
		l245:
			position, tokenIndex = position245, tokenIndex245
			return false
		},
		/* 29 Float <- <(Integer ('.' Unsigned)? (('e' / 'E') Integer)?)> */
		func() bool {
			position250, tokenIndex250 := position, tokenIndex
			{
				position251 := position
				if !_rules[ruleInteger]() {
					goto l250
				}
				{
					position252, tokenIndex252 := position, tokenIndex
					if buffer[position] != rune('.') {
						goto l252
					}
					position++
					if !_rules[ruleUnsigned]() {
						goto l252
					}
					goto l253
				l252:
					position, tokenIndex = position252, tokenIndex252
				}
			l253:
				{
					position254, tokenIndex254 := position, tokenIndex
					{
						position256, tokenIndex256 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l257
						}
						position++
						goto l256
					l257:
						position, tokenIndex = position256, tokenIndex256
						if buffer[position] != rune('E') {
							goto l254
						}
						position++
					}
				l256:
					if !_rules[ruleInteger]() {
						goto l254
					}
					goto l255
				l254:
					position, tokenIndex = position254, tokenIndex254
				}
			l255:
				add(ruleFloat, position251)
			}
			return true
		l250:
			position, tokenIndex = position250, tokenIndex250
			return false
		},
		/* 30 Duration <- <(Integer ('.' Unsigned)? (('n' 's') / ('u' 's') / ('µ' 's') / ('m' 's') / 's' / 'm' / 'h'))> */
		func() bool {
			position258, tokenIndex258 := position, tokenIndex
			{
				position259 := position
				if !_rules[ruleInteger]() {
					goto l258
				}
				{
					position260, tokenIndex260 := position, tokenIndex
					if buffer[position] != rune('.') {
						goto l260
					}
					position++
					if !_rules[ruleUnsigned]() {
						goto l260
					}
					goto l261
				l260:
					position, tokenIndex = position260, tokenIndex260
				}
			l261:
				{
					position262, tokenIndex262 := position, tokenIndex
					if buffer[position] != rune('n') {
						goto l263
					}
					position++
					if buffer[position] != rune('s') {
						goto l263
					}
					position++
					goto l262
				l263:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('u') {
						goto l264
					}
					position++
					if buffer[position] != rune('s') {
						goto l264
					}
					position++
					goto l262
				l264:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('µ') {
						goto l265
					}
					position++
					if buffer[position] != rune('s') {
						goto l265
					}
					position++
					goto l262
				l265:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('m') {
						goto l266
					}
					position++
					if buffer[position] != rune('s') {
						goto l266
					}
					position++
					goto l262
				l266:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('s') {
						goto l267
					}
					position++
					goto l262
				l267:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('m') {
						goto l268
					}
					position++
					goto l262
				l268:
					position, tokenIndex = position262, tokenIndex262
					if buffer[position] != rune('h') {
						goto l258
					}
					position++
				}
			l262:
				add(ruleDuration, position259)
			}
			return true
		l258:
			position, tokenIndex = position258, tokenIndex258
			return false
		},
		/* 31 Identifier <- <(!Keyword <(([a-z] / [A-Z] / '_') IdChar*)>)> */
		func() bool {
			position269, tokenIndex269 := position, tokenIndex
			{
				position270 := position
				{
					position271, tokenIndex271 := position, tokenIndex
					if !_rules[ruleKeyword]() {
						goto l271
					}
					goto l269
				l271:
					position, tokenIndex = position271, tokenIndex271
				}
				{
					position272 := position
					{
						position273, tokenIndex273 := position, tokenIndex
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l274
						}
						position++
						goto l273
					l274:
						position, tokenIndex = position273, tokenIndex273
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l275
						}
						position++
						goto l273
					l275:
						position, tokenIndex = position273, tokenIndex273
						if buffer[position] != rune('_') {
							goto l269
						}
						position++
					}
				l273:
				l276:
					{
						position277, tokenIndex277 := position, tokenIndex
						if !_rules[ruleIdChar]() {
							goto l277
						}
						goto l276
					l277:
						position, tokenIndex = position277, tokenIndex277
					}
					add(rulePegText, position272)
				}
				add(ruleIdentifier, position270)
			}
			return true
		l269:
			position, tokenIndex = position269, tokenIndex269
			return false
		},
		/* 32 IdChar <- <([a-z] / [A-Z] / [0-9] / '_')> */
		func() bool {
			position278, tokenIndex278 := position, tokenIndex
			{
				position279 := position
				{
					position280, tokenIndex280 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l281
					}
					position++
					goto l280
				l281:
					position, tokenIndex = position280, tokenIndex280
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l282
					}
					position++
					goto l280
				l282:
					position, tokenIndex = position280, tokenIndex280
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l283
					}
					position++
					goto l280
				l283:
					position, tokenIndex = position280, tokenIndex280
					if buffer[position] != rune('_') {
						goto l278
					}
					position++
				}
			l280:
				add(ruleIdChar, position279)
			}
			return true
		l278:
			position, tokenIndex = position278, tokenIndex278
			return false
		},
		/* 33 Keyword <- <((('s' 'e' 'l' 'e' 'c' 't') / ('g' 'r' 'o' 'u' 'p' ' ' 'b' 'y') / ('f' 'i' 'l' 't' 'e' 'r' 's') / ('o' 'r' 'd' 'e' 'r' ' ' 'b' 'y') / ('d' 'e' 's' 'c') / ('l' 'i' 'm' 'i' 't')) !IdChar)> */
		func() bool {
			position284, tokenIndex284 := position, tokenIndex
			{
				position285 := position
				{
					position286, tokenIndex286 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l287
					}
					position++
					if buffer[position] != rune('e') {
						goto l287
					}
					position++
					if buffer[position] != rune('l') {
						goto l287
					}
					position++
					if buffer[position] != rune('e') {
						goto l287
					}
					position++
					if buffer[position] != rune('c') {
						goto l287
					}
					position++
					if buffer[position] != rune('t') {
						goto l287
					}
					position++
					goto l286
				l287:
					position, tokenIndex = position286, tokenIndex286
					if buffer[position] != rune('g') {
						goto l288
					}
					position++
					if buffer[position] != rune('r') {
						goto l288
					}
					position++
					if buffer[position] != rune('o') {
						goto l288
					}
					position++
					if buffer[position] != rune('u') {
						goto l288
					}
					position++
					if buffer[position] != rune('p') {
						goto l288
					}
					position++
					if buffer[position] != rune(' ') {
						goto l288
					}
					position++
					if buffer[position] != rune('b') {
						goto l288
					}
					position++
					if buffer[position] != rune('y') {
						goto l288
					}
					position++
					goto l286
				l288:
					position, tokenIndex = position286, tokenIndex286
					if buffer[position] != rune('f') {
						goto l289
					}
					position++
					if buffer[position] != rune('i') {
						goto l289
					}
					position++
					if buffer[position] != rune('l') {
						goto l289
					}
					position++
					if buffer[position] != rune('t') {
						goto l289
					}
					position++
					if buffer[position] != rune('e') {
						goto l289
					}
					position++
					if buffer[position] != rune('r') {
						goto l289
					}
					position++
					if buffer[position] != rune('s') {
						goto l289
					}
					position++
					goto l286
				l289:
					position, tokenIndex = position286, tokenIndex286
					if buffer[position] != rune('o') {
						goto l290
					}
					position++
					if buffer[position] != rune('r') {
						goto l290
					}
					position++
					if buffer[position] != rune('d') {
						goto l290
					}
					position++
					if buffer[position] != rune('e') {
						goto l290
					}
					position++
					if buffer[position] != rune('r') {
						goto l290
					}
					position++
					if buffer[position] != rune(' ') {
						goto l290
					}
					position++
					if buffer[position] != rune('b') {
						goto l290
					}
					position++
					if buffer[position] != rune('y') {
						goto l290
					}
					position++
					goto l286
				l290:
					position, tokenIndex = position286, tokenIndex286
					if buffer[position] != rune('d') {
						goto l291
					}
					position++
					if buffer[position] != rune('e') {
						goto l291
					}
					position++
					if buffer[position] != rune('s') {
						goto l291
					}
					position++
					if buffer[position] != rune('c') {
						goto l291
					}
					position++
					goto l286
				l291:
					position, tokenIndex = position286, tokenIndex286
					if buffer[position] != rune('l') {
						goto l284
					}
					position++
					if buffer[position] != rune('i') {
						goto l284
					}
					position++
					if buffer[position] != rune('m') {
						goto l284
					}
					position++
					if buffer[position] != rune('i') {
						goto l284
					}
					position++
					if buffer[position] != rune('t') {
						goto l284
					}
					position++
				}
			l286:
				{
					position292, tokenIndex292 := position, tokenIndex
					if !_rules[ruleIdChar]() {
						goto l292
					}
					goto l284
				l292:
					position, tokenIndex = position292, tokenIndex292
				}
				add(ruleKeyword, position285)
			}
			return true
		l284:
			position, tokenIndex = position284, tokenIndex284
			return false
		},
		/* 34 _ <- <(' ' / '\t' / ('\r' '\n') / '\n' / '\r')*> */
		func() bool {
			{
				position294 := position
			l295:
				{
					position296, tokenIndex296 := position, tokenIndex
					{
						position297, tokenIndex297 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l298
						}
						position++
						goto l297
					l298:
						position, tokenIndex = position297, tokenIndex297
						if buffer[position] != rune('\t') {
							goto l299
						}
						position++
						goto l297
					l299:
						position, tokenIndex = position297, tokenIndex297
						if buffer[position] != rune('\r') {
							goto l300
						}
						position++
						if buffer[position] != rune('\n') {
							goto l300
						}
						position++
						goto l297
					l300:
						position, tokenIndex = position297, tokenIndex297
						if buffer[position] != rune('\n') {
							goto l301
						}
						position++
						goto l297
					l301:
						position, tokenIndex = position297, tokenIndex297
						if buffer[position] != rune('\r') {
							goto l296
						}
						position++
					}
				l297:
					goto l295
				l296:
					position, tokenIndex = position296, tokenIndex296
				}
				add(rule_, position294)
			}
			return true
		},
		/* 35 LPAR <- <(_ '(' _)> */
		func() bool {
			position302, tokenIndex302 := position, tokenIndex
			{
				position303 := position
				if !_rules[rule_]() {
					goto l302
				}
				if buffer[position] != rune('(') {
					goto l302
				}
				position++
				if !_rules[rule_]() {
					goto l302
				}
				add(ruleLPAR, position303)
			}
			return true
		l302:
			position, tokenIndex = position302, tokenIndex302
			return false
		},
		/* 36 RPAR <- <(_ ')' _)> */
		func() bool {
			position304, tokenIndex304 := position, tokenIndex
			{
				position305 := position
				if !_rules[rule_]() {
					goto l304
				}
				if buffer[position] != rune(')') {
					goto l304
				}
				position++
				if !_rules[rule_]() {
					goto l304
				}
				add(ruleRPAR, position305)
			}
			return true
		l304:
			position, tokenIndex = position304, tokenIndex304
			return false
		},
		/* 37 COMMA <- <(_ ',' _)> */
		func() bool {
			position306, tokenIndex306 := position, tokenIndex
			{
				position307 := position
				if !_rules[rule_]() {
					goto l306
				}
				if buffer[position] != rune(',') {
					goto l306
				}
				position++
				if !_rules[rule_]() {
					goto l306
				}
				add(ruleCOMMA, position307)
			}
			return true
		l306:
			position, tokenIndex = position306, tokenIndex306
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
