import std/strformat
import std/enumutils
import std/options

import lib/line_info


type TokenKind* = enum
    #[ Special ]#
    Empty         = "<empty>"
    Eof           = "<end-of-file>"
    Id            = "<identifier>"
    Comment       = "<comment>"
    CommentModule = "<comment-module>"
    HSpace        = "<horizontal-space>"
    VSpace        = "<vertical-space>"

    #[ Literals ]#
    IntLit           = "<int-literal>"
    FloatLit         = "<float-literal>"
    CharLit          = "<char-literal>"
    StringLit        = "<string-literal>"

    #[ Typed Literals ]#
    ISizeLit = "<isize-literal>"
    USizeLit = "<iusize-literal>"
    I8Lit    = "<i8-literal>"
    I16Lit   = "<i16-literal>"
    I32Lit   = "<i32-literal>"
    I64Lit   = "<i64-literal>"
    U8Lit    = "<u8-literal>"
    U16Lit   = "<u16-literal>"
    U32Lit   = "<u32-literal>"
    U64Lit   = "<u64-literal>"
    F32Lit   = "<f32-literal>"
    F64Lit   = "<f64-literal>"

    #[ Brackets ]#
    LeRound  = "("
    RiRound  = ")"
    LeCurly  = "{"
    RiCurly  = "}"
    LeSquare = "["
    RiSquare = "]"

    #[ Keywords ]#
    KwNil     = "nil"
    KwTrue    = "true"
    KwFalse   = "false"
    KwVar     = "var"
    KwVal     = "val"
    KwFunc    = "func"
    KwType    = "type"
    KwStruct  = "struct"
    KwEnum    = "enum"
    KwIf      = "if"
    KwElif    = "elif"
    KwElse    = "else"
    KwReturn  = "return"
    KwWhile   = "while"
    KwDo      = "do"

    #[ Word-like Operators ]#
    KwOr      = "or"
    KwAnd     = "and"
    KwNot     = "not"

    #[ Punctuation ]#
    Dot       = "."
    Comma     = ","
    Colon     = ":"
    Semicolon = ";"

    #[ Comparison Operators ]#
    EqOp = "=="
    NeOp = "!="
    LtOp = "<"
    GtOp = ">"
    LeOp = "<="
    GeOp = ">="

    #[ Other Operators ]#
    Plus       = "+"
    Minus      = "-"
    Asterisk   = "*"
    Slash      = "/"
    Percent    = "%"
    PlusPlus   = "++"
    Shl        = "<<"
    Shr        = ">>"
    Eq         = "="
    DotDot     = ".."
    DotDotDot  = "..."
    DotDotLess = "..<"
    FatArrow   = "=>"

    #[ Special symbols ]#
    Bar             = "|"
    ColonColon      = "::"
    Underscore      = "_"
    At              = "@"
    Dollar          = "$"
    ExclamationMark = "!"
    QuestionMark    = "?"
    Ampersand       = "&"

const
    UntypedLiteralKinds*   = {IntLit .. FloatLit}
    TypedLiteralKinds*     = {ISizeLit .. F64Lit}
    LiteralKinds*          = {IntLit .. F64Lit}
    KeywordKinds*          = {KwNil .. KwNot}
    OperatorKinds*         = {EqOp .. FatArrow} ## Word-like operators are not included, use `WordLikeOperatorKinds`
    WordLikeOperatorKinds* = {KwOr .. KwNot}
    StringableKinds*       = {LeRound .. Ampersand}

# func `$`*(self: TokenKind): string =
#     result = self.symbolName()

func toTokenKind*(s: string): Option[TokenKind] =
    result = none(TokenKind)

    for kind in StringableKinds:
        if $kind == s:
            result = some(kind)
            break

type Token* = object
    data* : string
    kind* : TokenKind
    info* : LineInfo

func `$`*(self: Token): string =
    result = self.kind.symbolName()

func human*(self: Token): string
    {.raises: [ValueError].} =
    result = fmt"at [{self.info}] {self}"

    if self.data.len() > 0:
        result.addQuoted(self.data)

func initToken*(kind: TokenKind; data = ""; info = LineInfo()): Token
    {.raises: [].} =
    result = Token(kind: kind, info: info, data: data)

const
    emptyToken* = initToken(Empty)
