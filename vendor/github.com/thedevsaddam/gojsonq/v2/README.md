![gojsonq-logo](gojsonq.png)

[![Build Status](https://travis-ci.org/thedevsaddam/gojsonq.svg?branch=master)](https://travis-ci.org/thedevsaddam/gojsonq)
[![Project status](https://img.shields.io/badge/version-v2-green.svg)](https://github.com/thedevsaddam/gojsonq/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/thedevsaddam/gojsonq)](https://goreportcard.com/report/github.com/thedevsaddam/gojsonq)
[![Coverage Status](https://coveralls.io/repos/github/thedevsaddam/gojsonq/badge.svg?branch=master)](https://coveralls.io/github/thedevsaddam/gojsonq)
[![GoDoc](https://godoc.org/github.com/thedevsaddam/gojsonq?status.svg)](https://pkg.go.dev/github.com/thedevsaddam/gojsonq/v2)
[![License](https://img.shields.io/dub/l/vibe-d.svg)](LICENSE.md)

A simple Go package to Query over JSON Data. It provides [simple](https://github.com/thedevsaddam/gojsonq/wiki/Queries#jsonstringjson), [elegant](https://github.com/thedevsaddam/gojsonq/wiki/Queries#selectproperties) and [fast](https://github.com/thedevsaddam/gojsonq/wiki/Benchmark) [ODM](https://github.com/thedevsaddam/gojsonq/wiki/Queries#frompath) like API to access, query JSON document

### Installation

Install the package using
```go
$ go get github.com/thedevsaddam/gojsonq/v2
```

### Usage

To use the package import it in your `*.go` code
```go
import "github.com/thedevsaddam/gojsonq/v2"
```

Let's see a quick example:

[See in playground](https://play.golang.org/p/UiqyllP2vkn)

```go
package main

import gojsonq "github.com/thedevsaddam/gojsonq/v2"

func main() {
	const json = `{"name":{"first":"Tom","last":"Hanks"},"age":61}`
	name := gojsonq.New().FromString(json).Find("name.first")
	println(name.(string)) // Tom
}
```

Another example:

[See in playground](https://play.golang.org/p/QLVxpi6nVbi)

```go
package main

import (
	"fmt"

	gojsonq "github.com/thedevsaddam/gojsonq/v2"
)

func main() {
	const json = `{"city":"dhaka","type":"weekly","temperatures":[30,39.9,35.4,33.5,31.6,33.2,30.7]}`
	avg := gojsonq.New().FromString(json).From("temperatures").Avg()
	fmt.Printf("Average temperature: %.2f", avg) // 33.471428571428575
}
```

You can query your document using the various query methods such as **[Find](https://github.com/thedevsaddam/gojsonq/wiki/Queries#findpath)**, **[First](https://github.com/thedevsaddam/gojsonq/wiki/Queries#first)**, **[Nth](https://github.com/thedevsaddam/gojsonq/wiki/Queries#nthindex)**, **[Pluck](https://github.com/thedevsaddam/gojsonq/wiki/Queries#pluckproperty)**,  **[Where](https://github.com/thedevsaddam/gojsonq/wiki/Queries#wherekey-op-val)**, **[OrWhere](https://github.com/thedevsaddam/gojsonq/wiki/Queries#orwherekey-op-val)**, **[WhereIn](https://github.com/thedevsaddam/gojsonq/wiki/Queries#whereinkey-val)**, **[WhereStartsWith](https://github.com/thedevsaddam/gojsonq/wiki/Queries#wherestartswithkey-val)**, **[WhereEndsWith](https://github.com/thedevsaddam/gojsonq/wiki/Queries#whereendswithkey-val)**, **[WhereContains](https://github.com/thedevsaddam/gojsonq/wiki/Queries#wherecontainskey-val)**, **[Sort](https://github.com/thedevsaddam/gojsonq/wiki/Queries#sortorder)**,  **[GroupBy](https://github.com/thedevsaddam/gojsonq/wiki/Queries#groupbyproperty)**,  **[SortBy](https://github.com/thedevsaddam/gojsonq/wiki/Queries#sortbyproperty-order)** and so on. Also you can aggregate data after query using **[Avg](https://github.com/thedevsaddam/gojsonq/wiki/Queries#avgproperty)**,  **[Count](https://github.com/thedevsaddam/gojsonq/wiki/Queries#count)**, **[Max](https://github.com/thedevsaddam/gojsonq/wiki/Queries#maxproperty)**, **[Min](https://github.com/thedevsaddam/gojsonq/wiki/Queries#minproperty)**, **[Sum](https://github.com/thedevsaddam/gojsonq/wiki/Queries#sumproperty)** etc.

## Find more query API in [Wiki page](https://github.com/thedevsaddam/gojsonq/wiki/Queries)

## Bugs and Issues

If you encounter any bugs or issues, feel free to [open an issue at
github](https://github.com/thedevsaddam/gojsonq/issues).

Also, you can shoot me an email to
<mailto:thedevsaddam@gmail.com> for hugs or bugs.

## Credit

Special thanks to [Nahid Bin Azhar](https://github.com/nahid) for the inspiration and guidance for the package. Thanks to [Ahmed Shamim Hasan Shaon](https://github.com/me-shaon) for his support from the very beginning.

## Contributors
* [Lenin Hasda](https://github.com/leninhasda)
* [Sadlil Rhythom](https://github.com/sadlil)
* [See contributors list here](https://github.com/thedevsaddam/gojsonq/graphs/contributors)

## Contribution
If you are interested to make the package better please send pull requests or create an issue so that others can fix.
[Read the contribution guide here](CONTRIBUTING.md)

## Special Thanks
<a href="https://www.jetbrains.com/?from=gojsonq"><img src="jetbrains-grayscale.png" height="100" width="100" ></a>

## License
The **gojsonq** is an open-source software licensed under the [MIT License](LICENSE.md).
