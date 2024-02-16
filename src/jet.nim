import
  std/strformat,
  std/strutils,
  std/sequtils,
  std/os,
  std/options,
  std/json,
  std/parseopt,

  jet/ast,
  jet/astalgo,
  jet/token,
  jet/lexer,
  # jet/parser,
  jet/parser2,
  jet/symbol,
  jet/module,
  jet/sem,

  lib/utils,
  lib/lineinfo,
  lib/logging

# WHY???
proc `%`*(v: char): JsonNode =
  result = JsonNode(kind: JString, str: $v)

proc getLine(buf: openArray[char]; n: Natural): string =
  proc skipUntilEndl(buf: openArray[char]; idx: var int) =
    while idx < buf.high and buf[idx] notin {'\r', '\n', '\0'}:
      idx += 1

  proc skipEndl(buf: openArray[char]; idx: var int) =
    if buf[idx] == '\r':
      idx += 1

    if buf[idx] == '\n':
      idx += 1

  var i = 0
  var lineNum = 1
  while lineNum < n:
    buf.skipUntilEndl(i)
    buf.skipEndl(i)
    lineNum += 1

  let start = i
  buf.skipUntilEndl(i)

  result = buf.toOpenArray(start, i - 1).substr()

proc handleError(err: ref CatchableError; filePath: string; line: string) =
  let range =
    if err of ParserError:
      (ref ParserError)(err).range
    elif err of LexerError:
      some((ref LexerError)(err).range)
    elif err of SemanticError:
      (ref SemanticError)(err).range
    else:
      return

  let target =
    if range.isSome():
      some(HighlightTarget(range: range.get(), line: line))
    else:
      none(HighlightTarget)

  HighlightInfo(
    message: err.msg,
    target: target,
    filePath: filePath,
    kind: HighlightInfoKind.Error,
  ).highlightInfoInFile()

proc main() =
  logger.maxErrors = 3

  let libDir = getAppDir().parentDir() / "lib"

  if not dirExists(libDir):
    panic("can't find core library directory: \"$jet/lib\"")

  # Pipeline:
  #   - read file
  #   - lexical analysis
  #   - normalizing tokens
  #   - syntactic analysis
  #   - generate AST
  #   - (?) resolve macros
  #   - semantic analysis
  #   - generate typed AST
  #   - (?) resolve annotations
  #   - backend stage

  let params   = commandLineParams()
  var argument = ""
  var noSem    = false

  for kind, key, val in params.getopt():
    case kind
    of cmdLongOption, cmdShortOption:
      case key
      of "nosem":
        noSem = val == "" or val.parseBool()
      else:
        panic("unknown option: " & key)
    of cmdArgument:
      argument = key
    of cmdEnd: discard

  if argument == "":
    panic("expected path to Jet file")

  hint("file reading...")
  let file = open(argument, fmRead).readAll()

  hint("lexical analysis...")
  var tokens = try:
    var lexer = newLexer(file)
    lexer.getAllTokens()
  except LexerError as err:
    handleError(err, argument, file.getLine(err.range.a.line.int))
    raise
  let tmp1 = "  " & tokens.mapIt(it.human()).join("\n  ")
  debug(&"tokens: \n{tmp1}")

  hint("normalizing tokens...")
  tokens = normalizeTokens(tokens)
  let tmp2 = "  " & tokens.mapIt(it.human()).join("\n  ")
  debug(&"normalized tokens: \n{tmp2}")

  hint("syntactic analysis...")
  var parser = newParser(tokens, filename=argument)
  try:
    parser.parseAll()
  except ParserError as err:
    handleError(
      err,
      argument,
      if err.range.isSome():
        file.getLine(err.range.get().a.line.int)
      else:
        "")
    raise

  hint("done")

  if parser.getAst().kind != Empty:
    debug("generated AST")
    parser.getAst().printTree()
    writeFile("tests_local/ast.json", (%parser.getAst()).pretty())
  else:
    debug("AST is not generated")

  if not noSem:
    hint("semantic analysis...")
    var rootTree = parser.getAst()
    var mainModule = newModule(rootTree)
    try:
      mainModule.traverseSymbols()
      debug("Root scope symbols:\n    " & mainModule.scope.symbols.join("\n    "))
    except SemanticError, ModuleError, ValueError:
      let err = getCurrentException()
      if err of (ref SemanticError):
        let err = (ref SemanticError)(err)
        handleError(
          err,
          argument,
          if err.range.isSome():
            file.getLine(err.range.get().a.line.int)
          else:
            "")
      raise

  # hint("generating C code...")
  # let generatedCode = ast2c(rootTree)
  # writeFile("out.jet.c", generatedCode)

  # hint("done!")

  # hint("building...")
  # let flags = fmt"-I{libDir} -o out.exe"
  # debug("flags: " & flags)

  # let code = execShellCmd("gcc " & flags & " out.jet.c")
  # if code != 0:
  #   panic("CC error!")

when isMainModule:
  main()
