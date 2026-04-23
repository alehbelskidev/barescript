
# Language Design Document

---

## Why This Exists

TypeScript tried to fix JavaScript by pretending it's Java.
It gave us classes over prototypes, type guards instead of real types,
`as` casts when the type system gave up, and interfaces that describe
object shapes instead of behavior.

We went a different direction.

**Problems we solved:**

- **Typing** — immutable by default, explicit `mut`, no surprise `let` vs `const`
- **Interfaces** — behavior only, Go-style duck typing, not object shape contracts
- **Hidden JS** — prototypes are first-class, no class syntax pretending otherwise
- **Type guards** — gone. Enums with data + match make them unnecessary
- **Dumb enums** — enums carry data, like Rust/Swift. A `Failure` without a message is impossible to construct
- **Readability** — `match` over `switch`, trailing lambdas, `if user = find(1)` unwrapping
- **Anonymous objects** — banned. Everything is named. Deserialization is typed and explicit
- **Nesting hell** — trailing lambda syntax, last expression as return, match as expression

This is Kotlin for JavaScript.

---

## Philosophy

- **Honest about JS** — no pretending it's Java. Prototypes are first-class citizens.
- **Immutable by default** — mutability is explicit.
- **Named everything** — no plain objects. Every shape has a name.
- **Clear boundaries** — the outside world is dirty (null, NaN, undefined). Inside your code, it's clean.
- **Last expression is return** — no `return` keyword needed.
- **No try/catch** — errors are values, handled through `Result` and `try` operator.
- **Everything is private by default** — explicit `export` to expose.

---

## Variables

```
// Immutable, inferred type — compiles to: const greeting = "Hello"
greeting = 'Hello'

// Mutable, inferred type — compiles to: let flag = true
mut flag = true

// Immutable typed without value — ERROR: must be initialized
name string       // ❌ compiler error

// Mutable typed without value — OK
mut name string   // ✅ compiles to: let name: string
```

**Rule:** `mut` = mutable (`let`). No `mut` = immutable (`const`). Type annotation optional if value is provided.

---

## Functions

```
// Global function — last expression is implicitly returned
fn sum(a number, b number) number {
    a + b
}

// Async function
async fn sumAsync(a number, b number) Promise<number> {
    Promise.resolve(a + b)
}

// Generic function
fn<T> identity(val T) T {
    val
}

// Async generic — async implies T is wrapped in Promise
async fn<T> wrap(val T) T {
    Promise.resolve(val)
}

// Optional parameter — T? means may be null
fn greet(name string, title string?) {
    if t = title {
        console.log('Hello #{t} #{name}')
    } else {
        console.log('Hello #{name}')
    }
}

// void return — omit return type, compiler infers void
fn log(msg string) {
    console.log(msg)
}

// Explicit void if you want
fn log(msg string) void {
    console.log(msg)
}

// Exported function
export fn printMe() {
    console.log('printme')
}

// Private by default — not exported, not accessible outside module
fn helper() {
    console.log('internal')
}
```

---

## Lambdas & Trailing Closures

```
// Inline lambda
anotherFn = () {
    console.log(this)
}

// Trailing lambda — last argument moved outside parens
nums.map { item, index in
    item * 2
}

nums.filter { item in
    item > 0
}

// No arguments
setTimeout(1000) {
    console.log('done')
}

// With initial accumulator
nums.reduce(0) { acc, n in
    acc + n
}

// Non-trailing — lambda is not the last argument, old syntax
nums.reduce((acc, n) { acc + n }, 0)
```

---

## Generators

```
// Generator function — fn*
fn* range(start number, end number) {
    mut i = start
    for i < end {
        yield i
        i = i + 1
    }
}

for n in range(0, 10) {
    console.log(n)  // 0 1 2 3 ... 9
}

// Lazy — values produced on demand, no array allocated
```

---

## Imports & Exports

```
// Everything is private by default
// Use export to expose

export fn publicFn() { ... }
export object PublicThing { ... }

fn privateFn() { ... }  // not accessible outside this file

// Imports — no quotes, paths are identifiers
import (
    * as z        from zod
    {v4 as uuid}  from uuid
    fs            from node:fs
    .{helper}     from ./utils/helper
)
```

---

## String Interpolation

```
name = 'World'
greeting = 'Hello #{name}'  // → "Hello World"
```

---

## Control Flow

```
// if — no parentheses
if counter > 0 {
    console.log('positive')
}

// for — no parentheses
for item in nums {
    console.log(item)
}
```

---

## Arrays & Spread

```
// Array type syntax
fn sum(nums []number) number {
    nums.reduce(0) { acc, n in acc + n }
}

tags []string = ['a', 'b', 'c']

// Spread — same as JS
merged = [...a, ...b]
fn log(...args) { console.log(args) }
```

---

## Destructuring

```
// Object destructuring — prefix with dot
.{name, age} = person
```

---

## Objects

> **No plain objects allowed.** Every shape must be declared as a named `object`.

```
object Human {
    age    number
    height number
    name   string

    // Static method — available on Human, not on instances
    // Compiles to: Human.describe = function() {}
    fn describe() {
        console.log('I am Human')
    }

    // Constructor
    fn init(age number, height number, name string) {
        this{age: age, height: height, name: name}
    }
}

// this{} — constructor-only syntax for field assignment
// Human() — no new keyword, compiler adds it
user = Human(25, 180, 'Alex')
```

### Generic Objects

```
object Response<T> {
    data   T
    status number

    fn init(data T, status number) {
        this{data: data, status: status}
    }
}

res = Response<User>(user, 200)
```

### Inheritance

```
object Programmer: Human {
    language string

    fn init(age number, height number, name string, language string) {
        super(age, height, name)
        this{language: language}
    }
}

dev = Programmer(28, 180, 'Alex', 'JavaScript')
```

---

## Prototypes

> Methods on the prototype — shared across all instances.
> Compiles to: `Human.prototype.sayHi = function() {}`

```
prototype Human: Greetable {
    fn sayHi() {
        console.log('Hi, i am #{this.name}')
    }

    fn sayBye() {
        console.log('Bye from #{this.name}')
    }
}
```

---

## Interfaces

> Behavior only — method signatures.
> Bound to `prototype` blocks. Go-style duck typing.

```
interface Greetable {
    sayHi()
    sayBye()
}

// Compiler error if any interface method is missing in prototype
prototype Human: Greetable {
    fn sayHi() {
        console.log('Hi, i am #{this.name}')
    }

    fn sayBye() {
        console.log('Bye from #{this.name}')
    }
}
```

---

## Union Types

```
fn stringify(val string | number) string {
    val.toString()
}
```

---

## Nullable Types

```
fn findUser(id number) User? {
    // returns User or null
}

// Swift-style if-let — unwraps nullable, User? becomes User inside block
if user = findUser(1) {
    console.log(user.name)  // ✅ safe
}

// Compiler prevents direct access without unwrapping
console.log(findUser(1).name)  // ❌ compiler error
```

**Rule:** `T` = never null. `T?` = explicitly nullable. No surprise nulls.

---

## Enums

```
// Simple enum
enum Direction {
    North
    South
    East
    West
}

// Enum with data — the variant carries its payload
// It's impossible to construct Failure without a message
enum RequestState {
    Idle
    Loading
    Success(data User)
    Failure(message string)
}

// vs the old way:
//   status = 'failure'
//   errorMessage = 'not found'  // nothing links these together
//
// now:
//   state = RequestState.Failure('not found')  // inseparable
```

---

## Match

> Replaces `switch/case`. Expression — returns a value.
> Exhaustive — compiler errors on missing variants.

```
mut state = RequestState.Loading
state = RequestState.Success(User(1, 'Alex'))

// Unwraps variant and payload in one step
match state {
    RequestState.Idle           -> console.log('waiting')
    RequestState.Loading        -> console.log('loading...')
    RequestState.Success(user)  -> console.log('got #{user.name}')
    RequestState.Failure(msg)   -> console.log('error: #{msg}')
}

// Match as expression
label = match state {
    RequestState.Idle     -> 'idle'
    RequestState.Loading  -> 'loading'
    RequestState.Success  -> 'done'
    RequestState.Failure  -> 'failed'
}

// _ for default when not all variants needed
match direction {
    Direction.North -> console.log('going north')
    _               -> console.log('some other direction')
}
```

---

## Option & Result

> Both are enums — no magic, just match.

```
// Option<T> — Some(value T) | None
// Result<T, E> — Ok(value T) | Err(error E)

fn findUser(id number) Option<User> { ... }
fn parseUser(raw string) Result<User, Error> { ... }

match findUser(1) {
    Some(user) -> console.log(user.name)
    None       -> console.log('not found')
}

match parseUser(raw) {
    Ok(user) -> console.log(user.name)
    Err(e)   -> console.log(e.message)
}
```

---

## try Operator

> Wraps any throwing call into `Result<T, Error>`.
> No try/catch blocks. `Err(e)` is the new `catch`.

```
// Assign
result = try JSON.parse(raw)

match result {
    Ok(data) -> console.log(data)
    Err(e)   -> console.log('failed: #{e.message}')
}

// Inline
match try JSON.parse(raw) {
    Ok(data) -> console.log(data)
    Err(e)   -> console.log('failed: #{e.message}')
}
```

---

## Serialization

> `toJSON` is free — JS drops methods on `JSON.stringify`.
> `fromJSON` is generated by `@serializable`.

```
@serializable
object Address {
    street string
    city   string
    fn init(street string, city string) { this{street: street, city: city} }
}

@serializable
object Profile {
    name    string
    address Address  // ✅ Address is @serializable

    fn init(name string, address Address) { this{name: name, address: address} }
}

// ❌ Compiler error — Human is not @serializable
@serializable
object Broken {
    human Human
}

// fromJSON returns Result — failure is explicit
match try Profile.fromJSON(responseBody) {
    Ok(profile) -> console.log(profile.address.city)
    Err(e)      -> console.log('bad payload: #{e.message}')
}
```

**Circular `@serializable` dependency** = compiler error + LSP highlight.

---

## Boundary Pattern

> Dirty outside, clean inside.
> External data enters only through `@serializable` or explicit `T?`.

```
raw = await fetchUserRaw(1)

match try User.fromJSON(raw) {
    Ok(user) -> {
        // clean zone:
        // - no surprise nulls
        // - no untyped objects
        // - errors are explicit values
        greetUser(user)
    }
    Err(e) -> console.log('invalid: #{e.message}')
}
```

---

## never & unknown

```
// unknown — you have a value but don't know its type yet
// must narrow before use (via match or if-let)
fn parse(raw unknown) User? { ... }

// never — a code path that should never be reached
// compiler verifies exhaustiveness using it internally
// surfaces as a compiler error, rarely written by hand
```

---

## Compile Targets

| Source | Compiles to |
|---|---|
| `object` | constructor function |
| `prototype` | `Object.prototype` assignments |
| `interface` | type-check only, erased at runtime |
| `enum` | frozen object with static fields |
| `match` | if/else chain |
| `try expr` | try/catch block returning `Ok`/`Err` |
| `@serializable` | generated `fromJSON` method |
| `mut` | `let` |
| no `mut` | `const` |
| `T?` | nullable with compiler null-check enforcement |
| `Type()` | `new Type()` |
| `fn*` / `yield` | JS generator function |
| no `export` | module-private |

---

## What This Is Not

- Not a new runtime — compiles to JS
- Not hiding JS — embracing it
- Not TypeScript — no `as`, no type guards, no `interface` as object shape
- Not Java — no classes pretending prototypes don't exist
