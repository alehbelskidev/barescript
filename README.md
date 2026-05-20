# BareScript - Language Design Document

---

## Why This Exists

TypeScript tried to fix JavaScript by pretending it's Java or C#. 
It gave us classes over prototypes, type guards instead of real types, `any`/`as` escape hatches that break the type system, and interfaces that describe passive object shapes instead of actual behavior.

We went a different direction. BareScript brings system-level predictability to the JS ecosystem.

**Problems we solved:**

- **The Billion Dollar Mistake** — `null` and `undefined` are completely eradicated from the language's core.
- **Zero Values** — uninitialized variables default to their type's empty state (0, "", false).
- **Honest Pointers** — JavaScript passes objects/arrays by reference. BareScript makes this explicit with `*`. You always know when you're mutating a reference.
- **Interfaces** — behavior only. Go-style duck typing. An interface is a contract of what an object *does*, not what it *looks like*.
- **Hidden JS** — prototypes are first-class. No class syntax pretending otherwise. No static methods.
- **Boundary Safety** — imported JS code is marked as `Unsafe`. It must be explicitly checked and unwrapped.
- **Smart Enums** — enums carry data, like Rust/Swift. A `Failure` without a message is impossible to construct.

This is the strictness of Rust and Go, running in JavaScript.

---

## Philosophy

- **No `undefined` or `null`** — literally. If a value might not exist, it must be wrapped in `Option<T>`.
- **Zero Values** — declaring a variable guarantees allocation of a default value.
- **Immutable by default** — mutability requires the explicit `mut` keyword.
- **Pointers are visible** — reference types are explicitly marked (e.g., `*[]number`).
- **Data vs Behavior** — `object` for shape, `prototype` for behavior.
- **The Outside World is Dirty** — external JS imports are `Unsafe` and quarantined until explicitly unwrapped.

---

## Variables & Zero Values

Variables are immutable by default. BareScript uses the `:=` walrus operator for type inference, and `mut` for mutable declarations. 

**There is no `undefined`.** A typed variable without an assignment receives a Zero Value.

```
// Variables with inferred types (immutable by default)
anotherId := 1001
anotherName := "Aleh Belski"
anotherIsAdult := true
anotherBigNum := 100000000000000000000000000001n

// Typed, mutable declarations get Zero Values (0, "", false)
mut id number       // defaults to 0
mut name string     // defaults to ""
mut isAdult boolean // defaults to false

// Special syntax for Symbols
@id 

// Reference types also get Zero Values (empty structures)
// Note the implicit pointers — we don't hide how JS works!
ids []number        // initialized as *[]
idMap Map<K, V>     // initialized as *Map()
idSet Set<K, V>     // initialized as *Set()

// Arrow functions must be initialized immediately
someFn := fn () {}
```

## Functions & Pointers

```
fn sum(a number, b number) number {
    return a + b
}

// Reference types must be shown explicitly with *
fn sumArr(arr *[]number) number {
    return arr.reduce(0) { acc, value do
        acc += value
    }
}

// Closures and mutable arguments
// The returned function is a reference type, marked with *
fn retFn(mut num number) *(num2 number) number {
    num *= 2 // num argument can be modified
    return fn (num2 number) number {
        return num + num2
    }
}

// Async functions
async fn fetchUser(id number) Promise<User> {
    return Promise.resolve(id)
}

// Generic functions
fn<T> identity(val T) T {
    return val
}
```

## Interfaces (Contracts)

Interfaces in BareScript do not describe object shapes like in TypeScript. They are strictly behavioral contracts.

```
// Only methods are allowed
interface Stringified {
    fn toString() string
}

// Interfaces are implicitly pointers (they point to an instance implementing the behavior)
fn printStr(item Stringified) {
    console.log(item.toString())
}
```

## Objects & Prototypes

BareScript separates Data `(object)` from Behavior `(prototype)`. No classes. No static methods.

```
// Pure data DTO
object Address {
    zip     number
    city    string
    state   string
    street  string
    country string
}

// Behavior attached to the object
prototype Address: Stringified {
    fn toString() {
        return "{street}, {zip} {city}, {state}, {country}"
    }
}

// Partial Initialization
// Unmentioned fields automatically receive their Zero Values (e.g., city becomes "")
addr := Address{zip: 2020}

// Destructuring with unpacking
// Mark explicitly if the unpacked variable should be mutable
unpack {street, mut city} := addr
```

## Enums & Pattern Matching

Enums can carry payloads. `match` is an exhaustive expression that replaces `switch`.

```
enum RequestState {
    Idle
    Loading
    Success(data User)
    Failure(message string)
}

fn handleState(state RequestState) {
    match state {
        // Dot shorthand for enum variants
        .Idle             -> console.log('idle')
        .Loading          -> console.log('loading')
        .Success(data)    -> console.log(data.name)
        .Failure(message) -> console.log(message)
    }
}
```

## Option & Result

Since `null` and `undefined` do not exist, absence of value is explicit via `Option<T>`. Operations that can fail use `Result<T, E>`.

```
// Option: Some | None
fn findUser(id number) Option<Address> {
    if id == 1 {
        return Some(Address{zip: 2020})
    }
    return None
}

match findUser(1) {
    Some(addr) -> console.log(addr.zip)
    None       -> console.log("Address not found")
}

// if-let shorthand for unwrapping Option
if user = findUser(1) {
    console.log(user.zip)
}
```


## Serialization & Try Operator

Every object automatically receives a .fromJSON() method that returns a Result. The try operator wraps throwing calls into Result<T, Error>.

```
// match + try combination for safe unwrapping
match try Address.fromJSON('{"state": "Nevada"}') {
    Ok(addr) -> console.log(addr.state)
    Err(e)   -> console.log("Failed to parse:", e)
}
```

## The Boundary Pattern (Unsafe)

Whenever you import code from external JavaScript/TypeScript files or npm packages, BareScript treats it as `Unsafe`.

```
// Imported code is Unsafe by default
import { getRawData } from "some-js-lib"

val data := getRawData() // Type is automatically Unsafe<T>

// The compiler forbids direct access to `data`.
// It must be verified and brought into the safe zone:
if safeData = data {
    // safeData is now guaranteed to be free of null/undefined
    console.log(safeData)
}
```

## Generators & Control Flow

```
// Generator
fn* range(start number, end number) {
    mut i = start
    for i in nums {
        yield i
    }
}

// Infinite loop / boolean condition
for true {
    break
}

// Arrays and iteration
nums := [1, 2, 3, 4, 5]
for n in nums {
    console.log(n)
}

// Trailing closures
result := nums.map { n in
    return n * 2
}

nums.reduce(0) { acc, n in
    return acc + n
}

setTimeout(1000) {
    console.log('done')
}

// Branching
if result.length == 2 {
    // do something
} else {
    // do another
}
```

## What This Is Not

- Not a new runtime — compiles to JS.
- Not hiding JS — embracing its prototype nature.
- Not TypeScript — no as, no type guards, no interface as object shape.
- Not Java — no classes pretending prototypes don't exist.
