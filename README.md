# CSS Modules in Go
This library follows almost all of the features and characteristics from the [CSS Modules non-standard specification](https://github.com/css-modules/css-modules)

With this library you can parse any CSS that you have and get some key-value pairs with your classes and the scoped classes

- ### Features:
- [x] Class scoping per function call
- [x] Global scoping trought the `:global` keyword
- [x] Media query scoping support
- [x] Another `@` (at) declarations support:
`@import`, `@font-face`, `@keyframes`, etc.
- [x] Your ID (`#`), element (`div`, `span`, etc.) and universal (`*`) selectors are global scoped whether they are outside or not of a `:global` block
- [ ] Scoping of animations (`@keyframes` declarations)
- [ ] `composes` keyword support

- ### Quick usage:
```go
package main

import (
    "log"
    "os"
    "strings"

    "github.com/otaxhu/go-cssmodules"
)

func main() {
    myCSS := strings.NewReader(
`.my-class {color: red;}`,
)
    // Now you have your cssScoped and you can access your classes throught 
    // the scopedClasses variable
    cssProcessed, scopedClasses, err := cssmodules.ProcessCSSModules(myCSS)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("your css processed: %s\nmy-class generated name: %s", cssProcessed, scopedClasses["my-class"])
    // Output in Stdout will be something like this:
    //
    // your css processed: .RANDOMCLASS{color: red;}
    // my-class generated name: RANDOMCLASS
}
```

## CSS Modules in HTML templates:
Also you can use the `ProcessHTMLWithCSSModules` function for processing HTML tags that has a `css-module` attribute.

Little example of a template with CSS Modules:

```html
<div css-module="class-foo">
    <p css-module="class-foo class-bar">Lorem ipsum</p>
</div>
```

Pass it with this `map[string]string` or `json`-like object with values of strings to the `ProcessHTMLWithCSSModules` function:
```js
// To obtain this object you have to first
// generate it with the function ProcessCSSModules
{
    "class-foo": "_class-foo_RANIDFOOBARPIZZA",
    "class-bar": "_class-bar_RANIDBARFOOAREPA"
}
```

Your template will become something like this after processed:

```html
<div class="_class-foo_RANIDFOOBARPIZZA">
    <p class="_class-foo_RANIDFOOBARPIZZA _class-bar_RANIDBARFOOAREPA">Lorem ipsum</p>
</div>
```

### Installation:
1. Create a new directory and initialize a go project with the following commands:
```sh
$ mkdir my-directory
$ cd my-directory
$ go mod init my-directory
go: creating new go.mod: module my-directory
```

2. Execute this command in your terminal and you are ready to go with CSS Modules:
```sh
$ go get github.com/otaxhu/go-cssmodules
```
