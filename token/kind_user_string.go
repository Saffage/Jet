// Code generated by "stringer -type=Kind -output=kind_user_string.go -linecomment"; DO NOT EDIT.

package token

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Illegal-0]
	_ = x[EOF-1]
	_ = x[Comment-2]
	_ = x[Whitespace-3]
	_ = x[Tab-4]
	_ = x[NewLine-5]
	_ = x[Ident-6]
	_ = x[Int-7]
	_ = x[Float-8]
	_ = x[String-9]
	_ = x[LParen-10]
	_ = x[RParen-11]
	_ = x[LCurly-12]
	_ = x[RCurly-13]
	_ = x[LBracket-14]
	_ = x[RBracket-15]
	_ = x[Comma-16]
	_ = x[Colon-17]
	_ = x[Semicolon-18]
	_ = x[Eq-19]
	_ = x[EqOp-20]
	_ = x[Bang-21]
	_ = x[NeOp-22]
	_ = x[LtOp-23]
	_ = x[LeOp-24]
	_ = x[GtOp-25]
	_ = x[GeOp-26]
	_ = x[Plus-27]
	_ = x[PlusEq-28]
	_ = x[Minus-29]
	_ = x[MinusEq-30]
	_ = x[Asterisk-31]
	_ = x[AsteriskEq-32]
	_ = x[Slash-33]
	_ = x[SlashEq-34]
	_ = x[Percent-35]
	_ = x[PercentEq-36]
	_ = x[Amp-37]
	_ = x[Pipe-38]
	_ = x[Caret-39]
	_ = x[At-40]
	_ = x[QuestionMark-41]
	_ = x[QuestionMarkDot-42]
	_ = x[Arrow-43]
	_ = x[FatArrow-44]
	_ = x[Shl-45]
	_ = x[Shr-46]
	_ = x[Dot-47]
	_ = x[Dot2-48]
	_ = x[Dot2Less-49]
	_ = x[Ellipsis-50]
	_ = x[KwAnd-51]
	_ = x[KwOr-52]
	_ = x[KwModule-53]
	_ = x[KwImport-54]
	_ = x[KwAlias-55]
	_ = x[KwStruct-56]
	_ = x[KwEnum-57]
	_ = x[KwFunc-58]
	_ = x[KwVal-59]
	_ = x[KwVar-60]
	_ = x[KwConst-61]
	_ = x[KwOf-62]
	_ = x[KwIf-63]
	_ = x[KwElse-64]
	_ = x[KwWhile-65]
	_ = x[KwReturn-66]
	_ = x[KwBreak-67]
	_ = x[KwContinue-68]
}

const _Kind_user_name = "illegal characterend of filecommentwhitespacehorizontal tabulationnew lineidentifieruntyped intuntyped floatuntyped string'('')''{''}''['']'','':'';'operator '='operator '=='operator '!'operator '!='operator '<'operator '<='operator '>'operator '>='operator '+'operator '+='operator '-'operator '-='operator '*'operator '*='operator '/'operator '/='operator '%'operator '%='operator '&'operator '|'operator '^'operator '@'operator '?'operator '?.'operator '->'operator '=>'operator '<<'operator '>>'operator '.'operator '..'operator '..<'operator '...'keyword 'and'keyword 'or'keyword 'module'keyword 'import'keyword 'alias'keyword 'struct'keyword 'enum'keyword 'func'keyword 'val'keyword 'var'keyword 'const'keyword 'of'keyword 'if'keyword 'else'keyword 'while'keyword 'return'keyword 'break'keyword 'continue'"

var _Kind_user_index = [...]uint16{0, 17, 28, 35, 45, 66, 74, 84, 95, 108, 122, 125, 128, 131, 134, 137, 140, 143, 146, 149, 161, 174, 186, 199, 211, 224, 236, 249, 261, 274, 286, 299, 311, 324, 336, 349, 361, 374, 386, 398, 410, 422, 434, 447, 460, 473, 486, 499, 511, 524, 538, 552, 565, 577, 593, 609, 624, 640, 654, 668, 681, 694, 709, 721, 733, 747, 762, 778, 793, 811}

func (i Kind) UserString() string {
	if i >= Kind(len(_Kind_user_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_user_name[_Kind_user_index[i]:_Kind_user_index[i+1]]
}
