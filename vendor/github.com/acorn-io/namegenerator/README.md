# namegenerator

A random name generator for acorns (for acorns, or whatever) implementation in Golang.

## Install

```bash
go get github.com/acorn-io/namegenerator
```

## Example of use

```go
package main

import (
    "github.com/goombaio/namegenerator"
)

func main() {
    seed := time.Now().UTC().UnixNano()
    nameGenerator := namegenerator.NewNameGenerator(seed)

    name := nameGenerator.Generate()

    fmt.Println(name)
}
```

## License
Copyright (c) 2022 [Acorn Labs, Inc.](http://acorn.io)

Copyright (c) 2018 Goomba project Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.