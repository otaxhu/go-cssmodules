# CSS Modules in Go
With this library you can parse any CSS that you have and get some key-value pairs with your classes and the scoped classes

- #### Features:
[x] Class scoping per function call

[x] Global scoping trought the `:global` keyword

[x] Media query scoping support

[ ] Another `@` (at) declarations support:
`@import`, `@font-face`, `@keyframes`, etc.

[x] Your ID (`#`), element (`div`, `span`, etc.) and universal (`*`) selectors are global scoped whether they are outside or not of a `:global` block

- #### Installation:
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