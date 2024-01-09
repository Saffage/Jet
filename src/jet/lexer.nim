import std/strutils except Newlines
import std/strformat
import std/streams
import std/options

import pkg/questionable

import jet/token

import lib/utils
import lib/utils/line_info


type Lexer* = object
    buffer      : string   ## Content of the file to be scanned
    pos         : int = 0  ## Position in the buffer
    lineStart   : int = 0  ## Position of line start in the buffer
    linePos     : int = 1  ## Current line number

    # Language specific fields
    curr* : Token
    prev* : Token

const
    EOF      = '\0'                    ## End of File
    CR       = '\r'                    ## Carriage Return
    LF       = '\n'                    ## Line Feed
    TAB      = '\t'                    ## Tabulation (horisontal)
    SPACE    = ' '                     ## Just a whitespace char, nothing special

const
    NewLines = {CR, LF}                ## New line char
    EOL      = {EOF} + NewLines        ## End of Line
    Digits   = {'0' .. '9'}
    Letters  = {'a' .. 'z', 'A' .. 'Z'}

func peek(self: Lexer): char =
    ## Peek current character from the `buffer`.
    result = self.buffer[self.pos]

func at(self: Lexer; n: Natural): char =
    ## Peek character at position `n` from the `buffer`.
    result = self.buffer[n]

func eat(self: var Lexer) =
    ## Peek character from the `buffer` and increment `pos`.
    self.curr.value &= self.peek()
    self.pos += 1

func eatChar(self: var Lexer; c: char) =
    ## Peek character from the `buffer` and increment `pos`.
    ## Expected character is `c`.
    assert(self.peek() == c)

    self.curr.value &= c
    self.pos += 1

func line(self: Lexer): int =
    ## Returns line number.
    result = self.linePos

func column(self: Lexer): int =
    ## Returns column number in the current line.
    result = self.pos - self.lineStart

func handleNewLine(self: var Lexer) =
    ## Call this when **CR** or **LF** is reached.
    ## The `pos` field must be at a position of this char.
    assert(self.peek() in NewLines)

    let wasCR      = (self.buffer[self.pos] == CR)
    self.pos      += (1 + wasCR.ord)
    self.linePos  += 1
    self.lineStart = self.pos

func skipLine(self: var Lexer) =
    ## Skip whole line until a new line.
    while self.peek() notin EOL:
        self.pos += 1

func lineInfo(self: Lexer): LineInfo =
    result = LineInfo(line: self.line().uint32, column: self.column().uint32)

func getLine*(self: Lexer; lineNum: Positive): Option[string] =
    ## **Returns:** line at `line` in `buffer` (new line character excluded).
    result = none(string)
    var i = 1
    for line in self.buffer.splitLines():
        if i == lineNum:
            result = some(line)
            break
        i += 1

func getLines*(self: Lexer; lineNums: openArray[int] | Slice[int]): Option[seq[string]] =
    ## **Returns:** specified lines from the `buffer`.
    result = some(newSeq[string]())
    for line in lineNums:
        let returnedLine = self.getLine(line)
        if returnedLine.isNone():
            result = none(seq[string])
            break
        result.get() &= returnedLine.get()

const
    BinChars*         = {'0' .. '1'}
    OctChars*         = {'0' .. '7'}
    HexChars*         = {'0' .. '9', 'a' .. 'f', 'A' .. 'F'}
    IdChars*          = {'_'} + Letters + Digits
    IdStartChars*     = IdChars - Digits
    UnaryOpWhitelist* = {' ', ',', ';', '(', '[', '{'} + EOL

func escape*(self: string): string =
    result = self.multiReplace(
        ("\'", "\\'"),
        ("\"", "\\\""),
        ("\\", "\\\\"),
        ("\0", "\\0"),
        ("\t", "\\t"),
        ("\n", "\\n"),
        ("\r", "\\r"),
    )

func getOperatorChars(): set[char] {.compileTime.} =
    result = {}
    for kind in TokenKind:
        if kind.isOperator():
            for c in $kind: result.incl(c)

const operatorChars = getOperatorChars()

func isAnyComment(self: Lexer): bool =
    return self.peek() == '#'

func isComment(self: Lexer): bool =
    return self.peek() == '#' and self.at(self.pos + 1) notin {'#', '!'}

func isCommentStmt(self: Lexer): bool =
    return self.peek() == '#' and self.at(self.pos + 1) in {'/', '!'}

proc scanComment(self: var Lexer) =
    ## Scans the doc comment statement (multiline)
    assert(self.isAnyComment())

    self.curr.kind = case self.at(self.pos + 1):
        of '#':
            Comment
        of '!':
            TopLevelComment
        else:
            self.curr.kind = Invalid # is it realy needed?
            self.skipLine()
            return

    self.pos += 2

    let spacesBefore      = self.curr.spacesBefore() |? self.curr.indent().get()
    var firstLine         = true
    var firstSpaceSkipped = true

    # нужно в случае когда у первых строк комментария были пропущены первые пробелы
    var skipedSpaces = 0

    while true:
        if firstSpaceSkipped:
            if self.peek() == ' ':
                self.pos += 1
                skipedSpaces += 1
            else:
                firstSpaceSkipped = false
        if firstLine:
            firstLine = false
        else:
            self.curr.value.add('\n')

        while self.peek() notin NewLines:
            self.curr.value.add(self.peek())
            self.pos += 1
        self.handleNewLine()

        let lastPos = self.pos
        var indent = 0

        while self.peek() == ' ':
            self.pos += 1
            indent += 1

        # TODO: handle it in parser?
        if indent != spacesBefore:
            # another comment stmt
            self.pos = lastPos
            break

        if not self.isCommentStmt():
            break

        case self.at(self.pos + 1)
        of '#': (if self.curr.kind != Comment: break)
        of '!': (if self.curr.kind != TopLevelComment: break)
        else: unreachable()
        self.pos += 2

    if not firstSpaceSkipped:
        # string reallocation is all you need to be happy
        var i = 0
        while skipedSpaces > 0:
            dec(skipedSpaces)
            self.curr.value.insert(" ", i)
            i = self.curr.value.find('\n', i) + 1

proc skipSpaces(self: var Lexer) =
    ## Skip spaces and set spacing\indentation for a token
    self.curr.setSpacesBefore(0)

    while true:
        case self.peek()
        of TAB:
            self.pos += 1
            panic("tabs are not allowed")
        of SPACE:
            self.pos += 1
            if indent =? self.curr.indent():
                self.curr.setIndent(indent + 1)
            if spacesBefore =? self.curr.spacesBefore():
                self.curr.setSpacesBefore(spacesBefore + 1)
        of Newlines:
            self.handleNewLine()
            self.curr.setFirstInLine(true)

            while self.peek() == SPACE:
                self.pos += 1

            # space is the last special char in ASCII so
            # any char after it ord value is visible
            if self.peek() > SPACE and not self.isComment():
                self.curr.setIndent(self.column())
                break
            else:
                self.curr.setSpacesBefore(self.column())
        else:
            if self.isAnyComment():
                self.scanComment()
            else:
                break

func scanMatchChars(self: var Lexer; chars: set[char]; minCharsCount: Natural = 0, handleUnderscore: bool = false): string =
    result = ""
    var wasUnderscore = false
    var charsCount    = 0

    while true:
        if handleUnderscore and self.peek() == '_':
            if wasUnderscore: panic("doubled '_' are illegal here")
            wasUnderscore = true
        else:
            wasUnderscore = false

        if self.peek() notin chars:
            break

        result &= self.peek()
        charsCount += 1
        self.pos += 1

    if wasUnderscore:
        panic("trailing '_' are illegal here")

    if minCharsCount > 0 and charsCount < minCharsCount:
        panic(fmt"expected {minCharsCount} or more characters, got {charsCount}")

func scanIdOrKeyword(self: var Lexer) =
    assert(self.peek() in IdStartChars)

    self.curr.value = self.scanMatchChars(IdChars)
    self.curr.kind = fromString(self.curr.value) |? Id

func scanEscapedChar(self: var Lexer) =
    assert(self.peek() == '\\')

    self.pos += 1

    case self.peek()
    of '\'': self.eat()
    of '\"': self.eat()
    of '\\': self.eat()
    of '0': self.eatChar(EOF)
    of 't': self.eatChar(TAB)
    of 'n': self.eatChar(LF)
    of 'r': self.eatChar(CR)
    of 'x': unimplemented("\\x character escape")
    of 'u':
        if self.curr.kind == CharLit: panic("\\u is not allowed in character literals")
        unimplemented("\\u character escape")
    else: panic("invalid character escape")

func scanChar(self: var Lexer) =
    assert(self.peek() == '\'')

    self.pos += 1
    self.curr.kind = CharLit

    case self.peek()
    of '\0' .. pred(' '): panic("invalid character literal")
    of '\'': panic("empty character literal")
    of '\\': self.scanEscapedChar()
    else: self.eat()

    if self.peek() != '\'':
        panic("missing closing \'")

    self.pos += 1

func performMultilineStringLitIndentation(self: var Lexer; lineIndices: openArray[int]) =
    assert(self.peek() == '\"' and self.at(self.pos + 1) == '\"' and self.at(self.pos + 2) == '\"')
    assert(lineIndices.len() > 0)

    var indentStr = self.buffer[self.lineStart ..< self.pos]

    if not indentStr.isEmptyOrWhitespace():
        #|  foo""" or something like that
        return

    unimplemented("maybe later")

    # var newlinePos = 0
    # var lines      = self.token.value.splitLines()

    # while newlinePos != -1:
    #     block inner:
    #         for i in 0 ..< indentStr.len():
    #             if self.token.value[newlinePos + i] != indentStr[i]:
    #                 break inner

    #         self.token.value[newlinePos .. indentStr.len()] = ""
    #         newlinePos = self.token.value.find('\n', newlinePos) + 1

func scanString(self: var Lexer; raw = false) =
    assert(self.peek() == '\"')
    self.pos += 1

    let multiline   = self.peek() == '\"' and self.at(self.pos + 1) == '\"'
    var lineIndices = newSeq[int]()

    self.curr.kind =
        if multiline:
            self.pos += 2

            # skip trailing spaces\tabs
            var pos = self.pos
            while self.peek() in {SPACE, TAB}:
                pos += 1

            if self.at(pos) in NewLines:
                self.pos = pos
                self.handleNewLine()

            if raw: LongRawStringLit
            else: LongStringLit
        else:
            if raw: RawStringLit
            else: StringLit

    while true:
        case self.peek()
        of '\"':
            if multiline:
                if self.at(self.pos + 1) == '\"' and
                   self.at(self.pos + 2) == '\"' and
                   self.at(self.pos + 3) != '\"':
                    # compute indentation and remove it from string value
                    if lineIndices.len() > 0:
                        self.performMultilineStringLitIndentation(lineIndices)
                    self.pos += 3
                    break
                self.eat()
            else:
                self.pos += 1
                break
        of '\\':
            if raw:
                self.eat()
            else:
                self.scanEscapedChar()
        of EOF:
            if multiline:
                panic("missing closing \"\"\"; end of line reached")
            else:
                panic("missing closing \"; end of line reached")
        of NewLines:
            if multiline:
                lineIndices.add(self.line())
                self.handleNewLine()
                self.curr.value.add('\n')
            else:
                panic("missing closing \"")
        else:
            self.eat()

func getOperatorKinds(): seq[TokenKind]
    {.compileTime.} =
    result = @[]
    for kind in TokenKind:
        if kind.isOperator():
            result &= kind

func scanOperator(self: var Lexer) =
    const operatorKinds = getOperatorKinds()

    var kind = Invalid
    var pos  = self.pos

    for i, operatorKind in operatorKinds.pairs():
        pos = self.pos
        let operator = $operatorKind

        for j, c in operator:
            if self.at(pos) != c:
                break

            # check it was the last character in this operator
            if j == operator.high:
                kind = operatorKind

            pos += 1

        # if operator was found, check next character
        if kind != Invalid:
            if self.at(pos) notin operatorChars:
                break
            else:
                # drop value if this operator is longer that found operator
                kind = Invalid
    if kind == Invalid:
        var i  = self.pos
        var op = ""
        while self.buffer[i] in operatorChars:
            op &= self.buffer[i]
            i  += 1
        panic(fmt"unknown operator '{op}'")

    assert(kind.isOperator())
    self.curr.kind = kind
    self.pos = pos

func scanNumber(self: var Lexer) =
    assert(self.peek() in Digits + {'-', '+'})

    self.curr.kind  = IntLit
    self.curr.value = newStringOfCap(64) # IDK why 64

    block beforeSuffix:
        if self.peek() in {'-', '+'}:
            self.eat()
        if self.peek() == '0':
            self.eat()
            var chars: set[char] = {}

            case self.peek()
            of 'b'           : chars = BinChars
            of 'x'           : chars = HexChars
            of 'o'           : chars = OctChars
            of 'B', 'X', 'O' : panic("using uppercase letters for number literals are not allowed. Use lowercase letters instead")
            of Digits        : panic("'0' can't be a first digit in literals if there is no '.' after it")
            else             : break beforeSuffix

            if chars != {}:
                self.eat()
                self.curr.value &= self.scanMatchChars(chars, minCharsCount=1, handleUnderscore=true)
        else:
            self.curr.value &= self.scanMatchChars(Digits, minCharsCount=1, handleUnderscore=true)

            if self.peek() == '.':
                self.curr.kind = FloatLit
                self.eat()
                self.curr.value &= self.scanMatchChars(Digits, minCharsCount=1, handleUnderscore=true)
            if self.peek() in {'e', 'E'}:
                self.curr.kind = FloatLit
                self.eatChar('e')

                if self.peek() in {'+', '-'}:
                    self.eat()

                self.curr.value &= self.scanMatchChars(Digits, minCharsCount=1)

    # scan literal suffix
    if self.peek() in Letters:
        var suffix = newStringOfCap(3)
        suffix &= self.scanMatchChars(IdChars - {'_'}, minCharsCount=1)

        self.curr.kind = case self.curr.kind:
            of FloatLit:
                case suffix.toLowerAscii()
                of "f32" : F32Lit
                of "f64" : F64Lit
                of "f"   : panic("redundant float literal suffix")
                else     : panic(fmt"invalid suffix '{suffix}' for float literal")
            of IntLit:
                case suffix.toLowerAscii()
                of "f32" : F32Lit
                of "f64" : F64Lit
                of "f"   : FloatLit
                of "i8"  : I8Lit
                of "i16" : I16Lit
                of "i32" : I32Lit
                of "i64" : I64Lit
                of "u8"  : U8Lit
                of "u16" : U16Lit
                of "u32" : U32Lit
                of "u64" : U64Lit
                of "u"   : USizeLit
                of "i"   : ISizeLit
                else: panic(fmt"invalid suffix '{suffix}' for integer literal")
            else: unreachable()

        const UnsignedLitKinds = {USizeLit, U8Lit .. U64Lit}

        if self.curr.kind in UnsignedLitKinds and
           self.curr.value.startsWith('-'):
            panic("unsigned integers can't be negative")

proc nextToken*(self: var Lexer) =
    ## Get next token and store it in the `tok` field.
    self.curr = newToken(Invalid)

    if self.pos >= self.buffer.len():
        panic(fmt"EOF reached, no tokens to scan (bufferPos: '{self.pos}', max: '{self.buffer.len()}')")

    let posBeforeSkip = self.pos
    self.skipSpaces()

    if posBeforeSkip == 0 and spacesBefore =? self.curr.spacesBefore():
        # first token
        self.curr.setIndent(spacesBefore)

    self.curr.scannerPos = self.pos
    self.curr.info       = self.lineInfo()

    if self.curr.kind != Invalid:
        unreachable()

    case self.peek()
    of EOF:
        self.curr.kind = Last
    of Newlines, ' ':
        unreachable()
    of '`':
        unimplemented("scanAccentId")
    of '?', '&', '@', ',', ';', '(', ')', '{', '}', '[', ']':
        self.curr.kind = fromString($self.peek()).get()
        self.pos += 1
    of ':':
        if self.at(self.pos + 1) == ':':
            self.curr.kind = ColonColon
            self.pos += 2
        else:
            self.curr.kind = Colon
            self.pos += 1
    of '-', '+':
        if self.at(self.pos + 1) in Digits and self.at(self.pos - 1) in UnaryOpWhitelist:
            self.scanNumber()
        else:
            self.scanOperator()
    of '|':
        self.curr.kind = Bar
        self.pos += 1
    of '.':
        if self.at(self.pos + 1) == '.':
            if self.at(self.pos + 2) == '<':
                self.curr.kind = DotDotLess
                self.pos += 3
            elif self.at(self.pos + 2) == '.':
                self.curr.kind = DotDotDot
                self.pos += 3
            else:
                self.curr.kind = DotDot
                self.pos += 2
        else:
            self.curr.kind = Dot
            self.pos += 1
    of '#':
        if self.at(self.pos + 1) in {'#', '!'}:
            self.scanComment()
        else:
            unreachable("comment are not skiped by 'skipSpaces', fix pls")
    of '_':
        if self.at(self.pos + 1) in IdChars:
            self.scanIdOrKeyword()
        else:
            self.curr.kind = Underscore
            self.pos += 1
    of '\"':
        self.scanString()
    of '\'':
        self.scanChar()
    of operatorChars - {'/', '_', '.', '-', '+', ':'}:
        self.scanOperator()
    of Digits:
        self.scanNumber()
    of Letters:
        if self.peek() in {'r', 'R'} and
           self.at(self.pos + 1) == '\"':
            self.pos += 1
            self.scanString(raw=true)
        else:
            self.scanIdOrKeyword()
    else:
        self.curr.value = $self.peek()
        self.curr.kind  = Invalid

        panic(fmt"invalid token '{self.curr.value}'")

    self.curr.info.length = uint32(self.pos - self.curr.scannerPos)

proc getToken*(self: var Lexer): ?Token =
    if self.curr.kind == Invalid:
        if self.curr.value.len() > 0:
            panic("got invalid token, data: '$1'" % self.curr.value)
        else:
            panic("got invalid token")
        return none(Token)

    self.prev = self.curr

    if self.curr.kind != Last:
        self.nextToken()

    if self.curr.isFirstInLine() or self.curr.kind == Last:
        self.prev.setLastInLine(true)
    else:
        self.prev.setSpacesAfter(self.curr.spacesBefore().get())

    result = some(self.prev)

proc getAllTokens*(self: var Lexer): seq[Token] =
    result = @[]

    while token =? self.getToken():
        result &= token
        if token.kind == Last: break


# ----- [] ----- #
proc newLexer*(content: sink string): Lexer
    {.raises: [ValueError].} =
    # manualy insert an additional '\0' to be able to index at it
    content.setLen(content.len() + 1)
    content[content.high] = '\0'

    result = Lexer(
        buffer: content,
        curr: newToken(Invalid),
        prev: newToken(Invalid),
    )
    result.nextToken()

proc newLexerFromFile*(file: File): Lexer
    {.raises: [IOError, OSError, ValueError].} =
    assert(file != nil)

    let fs = newFileStream(file)
    result = newLexer(fs.readAll())

proc newLexerFromFileName*(fileName: string): Lexer
    {.raises: [IOError, OSError, ValueError].} =
    let file = open(fileName, fmRead)
    result = newLexerFromFile(file)
