import
  std/strformat,
  std/strutils,
  std/sequtils,
  std/os,
  std/options,
  std/json,

  jet/astalgo,
  jet/token,
  jet/lexer,
  jet/parser,
  jet/symbol,
  jet/module,
  jet/sem,

  lib/utils,
  lib/line_info,

  pkg/results

# WHY???
proc `%`*(v: char): JsonNode =
  result = JsonNode(kind: JString, str: $v)

proc main() =
  logger.maxErrors = 3

  if not dirExists(getAppDir().parentDir() / "lib"):
    panic("can't find core library directory: \"$jet/lib\"")

  # Pipeline:
  #   - tokenize
  #   - parse AST
  #   - (?) annonations resolve
  #   - semantic checks
  #   - (?) typed AST
  #   - (?) deffered annonations resolve (typed annonations)
  #   - backend stage

  if paramCount() != 1:
    panic("expected path to Jet file as 1 argument")

  hint("file reading...")
  let argument = paramStr(1)
  let file     = open(argument, fmRead).readAll()

  hint("lexical analysis...")
  var tokens = try:
    var lexer = newLexer(file)
    lexer.getAllTokens()
  except LexerError as e:
    # TODO: file id
    stdout.write(argument & ":" & $e.info.noLength() & ": ")
    error(e.msg)
    raise
  let tmp1 = "  " & tokens.mapIt(it.human()).join("\n  ")
  debug(&"tokens: \n{tmp1}")

  hint("normalizing tokens...")
  tokens = normalizeTokens(tokens)
  let tmp2 = "  " & tokens.mapIt(it.human()).join("\n  ")
  debug(&"normalized tokens: \n{tmp2}")

  hint("syntactic analysis...")
  var parser = newParser(tokens)
  try:
    parser.parseAll()
  except ParserError as e:
    # TODO: file id
    stdout.write(argument & ":" & $e.info.noLength() & ": ")
    error(e.msg)
    raise

  hint("done")

  if parser.getAst().isSome():
    debug("generated AST")
    parser.getAst().get().printTree()
    writeFile("tests_local/ast.json", (%parser.getAst().get()).pretty())
  else:
    debug("AST is not generated")

  hint("semantic analysis...")
  var rootTree = parser.getAst().get()
  try:
    var mainModule = newModule(rootTree)
    mainModule.traverseSymbols()
    debug("Root scope symbols:\n    " & mainModule.rootScope.symbols.join("\n    "))
  except SemanticError as e:
    # TODO: file id
    stdout.write(argument & ":" & $e.info.noLength() & ": ")
    error(e.msg)
    raise

when isMainModule: main()
