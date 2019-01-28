/*
Package exp is a simple and extensible expression language, built on xelf types and literals.
It is meant to be used as a base for domain specific languages for a variety of tasks.

Language elements share the common interface El.
Atom elements are represented by a symbol, a literal or, for some types, a special type expression.
The atoms are:
    Type: a type definition as defined by package typ
    Lit:  a literal value as defined by package lit
    Ref:  a referenced and unresolved atom or spec

Expression elements are enclosed in parenthesis and usually start with a reference to a spec.
    Expr: an unresolved expression where the spec name is known
    Dyn:  a unresolved, dynamic expression where the spec name is yet unknown
    Tag:  a tagged argument group that applies to the parent expression
    Decl: a declaration group that applies to the parent expression

The parsing and resolution process is very abstract and uses following rules:
Literals and type symbols as well as type expressions are parsed normally.
Tag and declaration symbols always start an implicit sub expression.
    (eq (e :a 1 +b 2) (e (:a 1) (+b 2)))

All other symbols are parsed as references. Expressions not starting with a reference, are parsed
as a dynamic expression.

Implicit declarations associate to the elements to the right until the end, another declaration or
the special naked declaration symbol "-". The elements before the first and after the last
declaration and the elements inside the declarations are then parsed for tags.

Implicit tags associate to one element right of it unless it is the end, another tag or the special
naked tag symbol "::". The elements before the first tag are left alone, the ones after the last
tag are grouped in a tag expression with the special tag "::".

    (eq (e 1 :x 1 +a :y 2 3 +b 4 +c +d 5 - 6 : z 7 8 9)
        (e 1 (:x 1) (+a (:y 2) (:: 3)) (+b 4) (+c) (+d 5) 6 (:z 7) (:: 8 9)))

In the end a resolver implementation decides whether and how those sub expression are interpreted.

Dynamic expressions starting with a literal or type are resolved as the 'dyn' expression. Languages
built on this package can choose to use the built-in std resolver or use a custom implementation.

Dynamic expressions starting with a tag or declaration are invalid. The only other allowed
start elements are another unresolved or dynamic expression.

*/
package exp
