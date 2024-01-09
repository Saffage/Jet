import std/strformat
import std/sequtils

import jet/ast/nodetypes
import jet/literal
import jet/token

import lib/utils


export nodetypes


template `[]`*(self: Node; i: Natural): Node = self.children[i]
template `[]`*(self: Node; i: BackwardsIndex): Node = self.children[i]
template `[]`*[U; V](self: Node; i: HSlice[U, V]): seq[Node] = self.children[i]
template `[]=`*(self: Node; i: Natural; node: Node) = self.children[i] = node
template `[]=`*(self: Node; i: BackwardsIndex; node: Node) = self.children[i] = node
template `&=`*(self: Node; node: Node) = self.children.add(node)

template len*(self: Node): int = self.children.len()
template add*(self: Node; node: Node) = self.children.add(node)
template add*(self: Node; node: openArray[Node]) = self.children.add(node)
template insert*(self: Node; node: Node; i: Natural = 0) = self.children.insert(node, i)

proc expectKind*(self: Node; kind: NodeKind) =
    if self != nil and self.kind != kind:
        panic(fmt"expected {kind}, got {self.kind} instead")

proc expectKind*(self: Node; kinds: set[NodeKind]) =
    if self != nil and self.kind notin kinds:
        panic(fmt"expected one of {kinds}, got {self.kind} instead")

proc canHavePragma*(self: Node): bool =
    return self.kind in {nkFunc, nkType}

proc annotation*(self: Node): Node =
    result = case self.kind:
        of nkFunc: self[4]
        of nkType: self[2]
        else: panic(fmt"node of kind {self.kind} can't have an annotation")

    assert(result != nil)

proc `annotation=`*(self: Node; node: Node) =
    assert(node != nil)

    case self.kind:
        of nkFunc: self[4] = node
        of nkType: self[2] = node
        else: panic(fmt"node of kind {self.kind} can't have an annotation")

proc newNode*(kind: NodeKind): Node =
    result = Node(kind: kind)

proc newEmptyNode*(): Node =
    result = Node(kind: nkEmpty)

proc newIdNode*(id: string): Node =
    result = Node(kind: nkId, id: id)

proc newGenericId*(id: string): Node =
    result = Node(kind: nkGenericId, id: id)

proc newLitNode*(lit: TypedLiteral): Node =
    result = Node(kind: nkLit, lit: lit)

proc newLitNode*(lit: Literal): Node =
    result = newLitNode(lit.toTypedLit())

proc id*(token: Token): Node =
    assert(token.kind in {TokenKind.Id, TokenKind.Underscore})
    newIdNode(token.value)

proc id*(identifier: string): Node =
    newIdNode(identifier)

proc traverseTree(tree: Node; buffer: var string; indent: string; last: bool) =
    when defined(jetAstAsciiRepr):
        const connector = "|  "
        const leaf      = "|- "
        const lastLeaf  = "'- "
        const space     = "   "
    else:
        const connector = "│  "
        const leaf      = "├─╴"
        const lastLeaf  = "└─╴"
        const space     = "   "

    buffer &= indent & (if last: lastLeaf else: leaf)

    if tree != nil:
        buffer &= $tree & "\n"

        if not tree.isLeaf():
            let indent = indent & (if last: space else: connector)

            for i, node in tree.children:
                node.traverseTree(buffer, indent, i == tree.children.high)
    else:
        buffer &= "null\n"

proc treeRepr*(node: Node): string =
    result = ""
    node.traverseTree(result, "", true)
